// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package networkcmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

func newBootstrapCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Bootstrap a network from a remote snapshot",
		Long: `The bootstrap command downloads and installs a network snapshot from a remote source.
It supports downloading split archives (parts) in parallel and reassembling them.

This is useful for quickly syncing a new node by starting from a recent snapshot
instead of syncing from genesis.`,
		RunE: bootstrapNetwork,
	}

	cmd.Flags().StringVar(&snapshotNetworkType, "network-type", "mainnet", "network type to bootstrap (mainnet, testnet)")
	cmd.Flags().String("url", "", "base URL for the snapshot parts (optional, defaults to official repo)")
	cmd.Flags().String("snapshot-name", "", "specific snapshot name to download (optional)")

	return cmd
}

func bootstrapNetwork(cmd *cobra.Command, args []string) error {
	networkType := determineNetworkType()

	// Default URL pointing to the official repo's state/snapshots directory
	// In a real scenario, this might point to a release asset or raw content URL
	// For now, we'll assume a structure similar to what we saw in state/snapshots
	baseURL, _ := cmd.Flags().GetString("url")
	if baseURL == "" {
		// Use raw.githubusercontent.com for direct file access if it were a public repo
		// Or a configured S3 bucket / CDN
		// Placeholder for now
		baseURL = "https://raw.githubusercontent.com/luxfi/state/main/snapshots"
	}

	snapshotName, _ := cmd.Flags().GetString("snapshot-name")
	if snapshotName == "" {
		// Auto-detect latest based on convention if possible, or use hardcoded latest for now
		if networkType == "mainnet" {
			snapshotName = "mainnet_complete_20251225_083932"
		} else if networkType == "testnet" {
			snapshotName = "testnet_complete_20251225_035418"
		} else {
			return fmt.Errorf("unknown network type for auto-bootstrap: %s", networkType)
		}
	}

	ux.Logger.PrintToUser("Bootstrapping %s from snapshot: %s", networkType, snapshotName)

	// Create temp directory for download
	tmpDir, err := os.MkdirTemp("", "lux-bootstrap-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Identify parts
	// In a real implementation, we might fetch a manifest.
	// Here, we'll try to detect parts by probing or assume a pattern if we knew the count.
	// Since we don't know the exact count without a manifest, we'll implement a probe or
	// expect the user to provide it.
	// For this task, let's assume we implement a "Smart Download" that tries .part.aa, .ab, etc.

	parts, err := downloadParts(baseURL, snapshotName, tmpDir)
	if err != nil {
		return fmt.Errorf("failed to download snapshot parts: %w", err)
	}

	if len(parts) == 0 {
		return fmt.Errorf("no snapshot parts found at %s for %s", baseURL, snapshotName)
	}

	// Reassemble
	archivePath := filepath.Join(tmpDir, "full_snapshot.tar.gz")
	ux.Logger.PrintToUser("Reassembling snapshot from %d parts...", len(parts))
	if err := reassembleParts(parts, archivePath); err != nil {
		return fmt.Errorf("failed to reassemble snapshot: %w", err)
	}

	// Stop running network if any
	state, err := app.LoadNetworkStateForType(networkType)
	if err == nil && state != nil && state.Running {
		ux.Logger.PrintToUser("Stopping running network...")
		if err := StopNetwork(nil, nil); err != nil {
			return fmt.Errorf("failed to stop network: %w", err)
		}
		time.Sleep(2 * time.Second)
	}

	// Extract to run dir
	runDir := app.GetRunDir()
	destDir := filepath.Join(runDir, networkType)

	// Backup existing
	if _, err := os.Stat(destDir); err == nil {
		backupDir := destDir + ".backup." + time.Now().Format("20060102-150405")
		ux.Logger.PrintToUser("Backing up existing data to: %s", backupDir)
		if err := os.Rename(destDir, backupDir); err != nil {
			return fmt.Errorf("failed to backup existing data: %w", err)
		}
	}

	ux.Logger.PrintToUser("Extracting snapshot to %s...", destDir)
	if err := extractArchive(archivePath, destDir); err != nil {
		return fmt.Errorf("failed to extract snapshot: %w", err)
	}

	ux.Logger.PrintToUser("✓ Network bootstrapped successfully!")
	ux.Logger.PrintToUser("Run 'lux network start --%s' to start the node.", networkType)

	return nil
}

func downloadParts(baseURL, baseName, destDir string) ([]string, error) {
	var parts []string
	var partsMutex sync.Mutex
	var wg sync.WaitGroup

	// Suffixes aa, ab, ac, ...
	// We'll generate a reasonable range. 'az' is 26 parts. 'zz' is 676.
	// Should be enough for probe.
	suffixes := generateSuffixes()

	// Semaphore for concurrency limit
	sem := make(chan struct{}, 5)
	errChan := make(chan error, len(suffixes))

	// We need to know when to stop probing.
	// Strategy: Try downloading in parallel. If we get 404s, we assume we reached the end.
	// But parallel probing needs care.
	// Simpler approach: Download .tar.gz first (single file). If fail, try parts.

	// Try single file first
	singleUrl := fmt.Sprintf("%s/%s.tar.gz", baseURL, baseName)
	singleDest := filepath.Join(destDir, baseName+".tar.gz")
	ux.Logger.PrintToUser("Checking for single archive file...")
	if err := downloadFile(singleUrl, singleDest); err == nil {
		return []string{singleDest}, nil
	}

	ux.Logger.PrintToUser("Single file not found, checking for split archive parts...")

	// Download parts.
	// We'll iterate until we hit a 404 sequentially for the *existence* check?
	// Or just fire off requests and see what sticks?
	// Better: Sequentially check HEAD requests to determine count, then parallel download.

	var validSuffixes []string
	for _, suffix := range suffixes {
		url := fmt.Sprintf("%s/%s.tar.gz.part.%s", baseURL, baseName, suffix)
		resp, err := http.Head(url)
		if err == nil && resp.StatusCode == 200 {
			validSuffixes = append(validSuffixes, suffix)
			resp.Body.Close()
		} else {
			// Assume contiguous parts. If we miss 'aa', we stop?
			// If we miss 'aa', maybe it's not split.
			// If we found 'aa' but miss 'ab', that's the end.
			if len(validSuffixes) > 0 {
				break
			}
			if suffix == "aa" {
				// If first part missing, assume no split archive
				return nil, fmt.Errorf("snapshot not found")
			}
		}
	}

	ux.Logger.PrintToUser("Found %d parts. Downloading...", len(validSuffixes))

	for _, suffix := range validSuffixes {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			filename := fmt.Sprintf("%s.tar.gz.part.%s", baseName, s)
			url := fmt.Sprintf("%s/%s", baseURL, filename)
			dest := filepath.Join(destDir, filename)

			if err := downloadFile(url, dest); err != nil {
				errChan <- err
				return
			}

			partsMutex.Lock()
			parts = append(parts, dest)
			partsMutex.Unlock()
		}(suffix)
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		return nil, <-errChan
	}

	// Sort parts to ensure correct order
	// strings.Sort(parts) - actually verify they are sorted by suffix
	// Since we appended in random order, we must sort.
	// But we constructed filenames with suffixes, so standard sort works.
	// (We should implement sort)

	return sortParts(parts), nil
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("status code %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func reassembleParts(parts []string, dest string) error {
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	for _, part := range parts {
		in, err := os.Open(part)
		if err != nil {
			return err
		}

		if _, err := io.Copy(out, in); err != nil {
			in.Close()
			return err
		}
		in.Close()
	}
	return nil
}

func generateSuffixes() []string {
	var suffixes []string
	chars := "abcdefghijklmnopqrstuvwxyz"
	for i := 0; i < len(chars); i++ {
		for j := 0; j < len(chars); j++ {
			suffixes = append(suffixes, string(chars[i])+string(chars[j]))
		}
	}
	return suffixes
}

func sortParts(parts []string) []string {
	// Simple bubble sort or use sort package.
	// Since slice is small, bubble sort is fine, or just import sort.
	// networkcmd package context... let's check imports.
	// I'll add "sort" to imports.

	// Placeholder manual sort to avoid import churn for now if easy:
	for i := 0; i < len(parts); i++ {
		for j := i + 1; j < len(parts); j++ {
			if parts[i] > parts[j] {
				parts[i], parts[j] = parts[j], parts[i]
			}
		}
	}
	return parts
}
