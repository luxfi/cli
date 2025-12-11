// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package keycmd

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	generateCount  int
	generatePrefix string
	generateStart  int
)

func newGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate",
		Aliases: []string{"gen", "batch"},
		Short:   "Generate multiple key sets quickly",
		Long: `Generate multiple key sets with indexed names.

Creates keys with names like: prefix-0, prefix-1, prefix-2, etc.
Each key set contains EC, BLS, Ringtail, and ML-DSA keys.

Examples:
  lux key generate -n 5                    # Creates key-0 through key-4
  lux key generate -n 10 --prefix validator # Creates validator-0 through validator-9
  lux key generate -n 3 --start 5          # Creates key-5, key-6, key-7`,
		RunE: runGenerate,
	}

	cmd.Flags().IntVarP(&generateCount, "count", "n", 1, "Number of key sets to generate")
	cmd.Flags().StringVarP(&generatePrefix, "prefix", "p", "key", "Prefix for key names")
	cmd.Flags().IntVarP(&generateStart, "start", "s", 0, "Starting index number")

	return cmd
}

func runGenerate(_ *cobra.Command, _ []string) error {
	if generateCount <= 0 {
		return fmt.Errorf("count must be positive")
	}

	// Get existing keys to check for conflicts
	existing, err := key.ListKeySets()
	if err != nil {
		return fmt.Errorf("failed to list existing keys: %w", err)
	}
	existingMap := make(map[string]bool)
	for _, k := range existing {
		existingMap[k] = true
	}

	// Prepare list of names to generate
	var names []string
	for i := 0; i < generateCount; i++ {
		name := fmt.Sprintf("%s-%d", generatePrefix, generateStart+i)
		if existingMap[name] {
			return fmt.Errorf("key '%s' already exists, use --start to change index", name)
		}
		names = append(names, name)
	}

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Generating %d key sets...", generateCount)
	ux.Logger.PrintToUser("")

	// Progress tracking
	var mu sync.Mutex
	completed := 0
	failed := 0
	results := make([]*key.HDKeySet, len(names))
	errors := make([]error, len(names))

	// Show progress bar header
	progressBar := strings.Repeat("░", 50)
	ux.Logger.PrintToUser("[%s] 0%%", progressBar)

	startTime := time.Now()

	// Generate keys (could parallelize for very large batches, but sequential for now to show progress)
	for i, name := range names {
		// Generate mnemonic
		mnemonic, err := key.GenerateMnemonic()
		if err != nil {
			mu.Lock()
			errors[i] = err
			failed++
			mu.Unlock()
			continue
		}

		// Derive all keys
		keySet, err := key.DeriveAllKeys(name, mnemonic)
		if err != nil {
			mu.Lock()
			errors[i] = err
			failed++
			mu.Unlock()
			continue
		}

		// Save keys
		if err := key.SaveKeySet(keySet); err != nil {
			mu.Lock()
			errors[i] = err
			failed++
			mu.Unlock()
			continue
		}

		mu.Lock()
		results[i] = keySet
		completed++
		progress := float64(completed+failed) / float64(generateCount) * 100
		filled := int(progress / 2)
		progressBar = strings.Repeat("█", filled) + strings.Repeat("░", 50-filled)
		ux.Logger.PrintToUser("\r[%s] %.0f%% - %s", progressBar, progress, name)
		mu.Unlock()
	}

	elapsed := time.Since(startTime)

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Generation complete in %v", elapsed.Round(time.Millisecond))
	ux.Logger.PrintToUser("  Created: %d", completed)
	if failed > 0 {
		ux.Logger.PrintToUser("  Failed:  %d", failed)
	}
	ux.Logger.PrintToUser("")

	// Show created keys summary
	if completed > 0 {
		ux.Logger.PrintToUser("Created keys:")
		ux.Logger.PrintToUser("%-20s  %-44s", "NAME", "EC ADDRESS")
		ux.Logger.PrintToUser("%s  %s", strings.Repeat("-", 20), strings.Repeat("-", 44))
		for i, ks := range results {
			if ks != nil {
				ux.Logger.PrintToUser("%-20s  %s", ks.Name, ks.ECAddress)
			} else if errors[i] != nil {
				ux.Logger.PrintToUser("%-20s  ERROR: %v", names[i], errors[i])
			}
		}
	}

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Use 'lux key ls' to see all keys")
	ux.Logger.PrintToUser("Use 'lux key show <name>' to view key details")

	return nil
}
