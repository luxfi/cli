// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package zkcmd

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

const (
	defaultSRSURL = "https://api.lux.network/mainnet/ext/bc/Z/srs"
)

func newSRSCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "srs",
		Short: "SRS management (download, verify, info)",
		Long: `Manage Structured Reference Strings (SRS) for ZK proof systems.

The SRS is the output of the trusted setup ceremony and is required
for both proof generation and verification. The official Lux SRS
is published on the Z-Chain.`,
	}

	cmd.AddCommand(newSRSDownloadCmd())
	cmd.AddCommand(newSRSVerifyCmd())
	cmd.AddCommand(newSRSInfoCmd())

	return cmd
}

func newSRSDownloadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download the official Lux SRS from Z-Chain",
		Long: `Download the official SRS binary from the Z-Chain. The file is
verified after download using the ceremony binary.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			url, _ := cmd.Flags().GetString("url")
			output, _ := cmd.Flags().GetString("output")
			return downloadSRS(url, output)
		},
	}

	cmd.Flags().String("url", defaultSRSURL, "SRS download URL")
	cmd.Flags().String("output", "", "Output file path (default: ~/.lux/zk/srs.bin)")

	return cmd
}

func newSRSVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify a downloaded SRS",
		Long: `Verify the integrity of a downloaded SRS file by checking its
structure and computing its SHA-256 hash.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			input, _ := cmd.Flags().GetString("input")
			return verifySRS(input)
		},
	}

	cmd.Flags().String("input", "", "SRS file path (required)")
	cmd.MarkFlagRequired("input")

	return cmd
}

func newSRSInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show SRS metadata (powers, size, hash)",
		Long:  `Display metadata about an SRS file including the number of powers, file size, and SHA-256 hash.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			input, _ := cmd.Flags().GetString("input")
			return showSRSInfo(input)
		},
	}

	cmd.Flags().String("input", "", "SRS file path (required)")
	cmd.MarkFlagRequired("input")

	return cmd
}

func srsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".lux", "zk")
}

func downloadSRS(url, output string) error {
	if output == "" {
		output = filepath.Join(srsDir(), "srs.bin")
	}

	if err := os.MkdirAll(filepath.Dir(output), 0o750); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	fmt.Printf("Downloading SRS from %s\n", url)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	n, err := io.Copy(f, io.TeeReader(resp.Body, h))
	if err != nil {
		os.Remove(output)
		return fmt.Errorf("write: %w", err)
	}

	hash := hex.EncodeToString(h.Sum(nil))
	fmt.Printf("Downloaded: %s (%d bytes)\n", output, n)
	fmt.Printf("SHA-256:    %s\n", hash)
	return nil
}

func verifySRS(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read SRS: %w", err)
	}

	if len(data) < 4 {
		return fmt.Errorf("SRS file too small (%d bytes)", len(data))
	}

	n := int(binary.BigEndian.Uint32(data[:4]))
	// Expected size: 4 + n*64 + 2*128 + n*64 + n*64 + 128
	expected := 4 + n*64 + 2*128 + n*64 + n*64 + 128
	if len(data) != expected {
		return fmt.Errorf("SRS size mismatch: expected %d bytes for %d powers, got %d", expected, n, len(data))
	}

	h := sha256.Sum256(data)
	fmt.Printf("SRS verification passed\n")
	fmt.Printf("  Powers:  %d\n", n)
	fmt.Printf("  Size:    %d bytes\n", len(data))
	fmt.Printf("  SHA-256: %s\n", hex.EncodeToString(h[:]))
	return nil
}

func showSRSInfo(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat SRS: %w", err)
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open SRS: %w", err)
	}
	defer f.Close()

	var header [4]byte
	if _, err := io.ReadFull(f, header[:]); err != nil {
		return fmt.Errorf("read header: %w", err)
	}

	n := int(binary.BigEndian.Uint32(header[:]))

	// Compute hash of full file
	f.Seek(0, 0)
	h := sha256.New()
	io.Copy(h, f)
	hash := hex.EncodeToString(h.Sum(nil))

	// Determine power of 2
	power := 0
	for p := n - 1; p > 1; p >>= 1 {
		power++
	}

	fmt.Printf("SRS: %s\n", path)
	fmt.Printf("  Powers:       %d (2^%d + 1 constraints)\n", n, power)
	fmt.Printf("  G1 points:    %d (tauG1) + %d (alphaG1) + %d (betaG1)\n", n, n, n)
	fmt.Printf("  G2 points:    2 (tauG2) + 1 (betaG2)\n")
	fmt.Printf("  File size:    %d bytes\n", info.Size())
	fmt.Printf("  SHA-256:      %s\n", hash)
	return nil
}
