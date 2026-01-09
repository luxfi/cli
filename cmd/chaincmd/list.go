// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chaincmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luxfi/constantsants"
	"github.com/luxfi/sdk/models"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configured blockchains",
		Long: `List all configured blockchains with their details.

OVERVIEW:

  Displays a table of all blockchain configurations stored in ~/.lux/chains/.
  Shows configuration details and deployment status across networks.

OUTPUT COLUMNS:

  Name        Blockchain configuration name
  Type        Chain type (L1, L2, L3)
  Chain ID    EVM chain ID
  VM          Virtual machine type (EVM, CustomVM)
  Sequencer   Sequencer type (lux, ethereum, op)
  Deployed    Whether chain is deployed to any network

EXAMPLES:

  # List all configured chains
  lux chain list

TYPICAL OUTPUT:

  +----------+------+----------+-----+-----------+----------+
  | NAME     | TYPE | CHAIN ID | VM  | SEQUENCER | DEPLOYED |
  +----------+------+----------+-----+-----------+----------+
  | mychain  | L2   | 200200   | EVM | lux       | Yes      |
  | testnet  | L1   | 36911    | EVM | lux       | No       |
  +----------+------+----------+-----+-----------+----------+

NOTES:

  - Only shows chains with valid configurations
  - "Deployed: Yes" means chain is deployed to at least one network
  - Use 'lux chain describe <name>' for detailed chain information
  - Use 'lux network status' to see endpoints of deployed chains`,
		RunE: listChains,
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
		data, err := os.ReadFile(sidecarPath) //nolint:gosec // G304: Reading from app's data directory
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
		if len(sc.Networks) > 0 {
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
