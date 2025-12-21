// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package chaincmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/sdk/models"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configured blockchains",
		Long:  "Display a table of all configured blockchains with their type, chain ID, and status.",
		RunE:  listChains,
	}
}

func listChains(cmd *cobra.Command, args []string) error {
	subnetDir := app.GetChainsDir()
	entries, err := os.ReadDir(subnetDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No chains configured")
			return nil
		}
		return fmt.Errorf("failed to read chains directory: %w", err)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Name", "Type", "Chain ID", "VM", "Sequencer", "Deployed")

	rowCount := 0
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		sidecarPath := filepath.Join(subnetDir, entry.Name(), constants.SidecarFileName)
		data, err := os.ReadFile(sidecarPath)
		if err != nil {
			continue
		}

		var sc models.Sidecar
		if err := json.Unmarshal(data, &sc); err != nil {
			continue
		}

		// Determine chain type
		chainType := "L2"
		if sc.Sovereign {
			chainType = "L1"
		}

		// Determine deployment status
		deployed := "No"
		if sc.Networks != nil && len(sc.Networks) > 0 {
			deployed = "Yes"
		}

		// Get sequencer
		sequencer := sc.SequencerType
		if sequencer == "" {
			sequencer = "lux"
		}

		_ = table.Append([]string{
			sc.Name,
			chainType,
			sc.ChainID,
			string(sc.VM),
			sequencer,
			deployed,
		})
		rowCount++
	}

	if rowCount == 0 {
		fmt.Println("No chains configured. Create one with: lux chain create <name>")
		return nil
	}

	_ = table.Render()
	return nil
}
