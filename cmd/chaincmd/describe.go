// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chaincmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

func newDescribeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "describe [chainName]",
		Short: "Show detailed information about a blockchain",
		Args:  cobra.ExactArgs(1),
		RunE:  describeChain,
	}
}

func describeChain(cmd *cobra.Command, args []string) error {
	chainName := args[0]

	sc, err := app.LoadSidecar(chainName)
	if err != nil {
		return fmt.Errorf("chain %s not found", chainName)
	}

	ux.Logger.PrintToUser("Chain: %s", sc.Name)
	ux.Logger.PrintToUser("VM: %s", sc.VM)
	if sc.VMVersion != "" {
		ux.Logger.PrintToUser("VM Version: %s", sc.VMVersion)
	}

	if sc.Sovereign {
		ux.Logger.PrintToUser("Type: Sovereign L1")
	} else if sc.BasedRollup {
		ux.Logger.PrintToUser("Type: Based Rollup (L2)")
		ux.Logger.PrintToUser("Sequencer: %s", sc.SequencerType)
		ux.Logger.PrintToUser("Block Time: %dms", sc.L1BlockTime)
	}

	if sc.PreconfirmEnabled {
		ux.Logger.PrintToUser("Pre-confirmations: Enabled")
	}

	// Show deployment info
	if len(sc.Networks) > 0 {
		ux.Logger.PrintToUser("\nDeployments:")
		for network, data := range sc.Networks {
			ux.Logger.PrintToUser("  %s:", network)
			ux.Logger.PrintToUser("    Chain ID: %s", data.ChainID)
			ux.Logger.PrintToUser("    Blockchain ID: %s", data.BlockchainID)
		}
	}

	// Print genesis
	ux.Logger.PrintToUser("\nGenesis:")
	genesis, err := app.LoadRawGenesis(chainName)
	if err == nil {
		var prettyGenesis map[string]interface{}
		if err := json.Unmarshal(genesis, &prettyGenesis); err == nil {
			prettyBytes, _ := json.MarshalIndent(prettyGenesis, "", "  ")
			_, _ = fmt.Fprintln(os.Stdout, string(prettyBytes))
		}
	}

	return nil
}
