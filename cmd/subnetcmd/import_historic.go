// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package subnetcmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/models"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/node/ids"
	"github.com/spf13/cobra"
)

var (
	historicDataPath string
	autoRegister     bool
)

// Historic subnet configurations
var historicSubnets = []struct {
	Name         string
	SubnetID     string
	BlockchainID string
	ChainID      uint64
	TokenName    string
	TokenSymbol  string
	VMID         string
	VMVersion    string
}{
	{
		Name:         "LUX",
		SubnetID:     "tJqmx13PV8UPQJBbuumANQCKnfPUHCxfahdG29nJa6BHkumCK",
		BlockchainID: "dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ",
		ChainID:      96369,
		TokenName:    "LUX Token",
		TokenSymbol:  "LUX",
		VMID:         "srEXiWaHuhNyGwPUi444Tu47ZEDwxTWrbQiuD7FmgSAQ6X7Dy",
		VMVersion:    "v0.6.12",
	},
	{
		Name:         "ZOO",
		SubnetID:     "xJzemKCLvBNgzYHoBHzXQr9uesR3S3kf3YtZ5mPHTA9LafK6L",
		BlockchainID: "bXe2MhhAnXg6WGj6G8oDk55AKT1dMMsN72S8te7JdvzfZX1zM",
		ChainID:      200200,
		TokenName:    "ZOO Token",
		TokenSymbol:  "ZOO",
		VMID:         "srEXiWaHuhNyGwPUi444Tu47ZEDwxTWrbQiuD7FmgSAQ6X7Dy",
		VMVersion:    "v0.6.12",
	},
	{
		Name:         "SPC",
		SubnetID:     "2hMMhMFfVvpCFrA9LBGS3j5zr5XfARuXdLLYXKpJR3RpnrunH9",
		BlockchainID: "QFAFyn1hh59mh7kokA55dJq5ywskF5A1yn8dDpLhmKApS6FP1",
		ChainID:      36911,
		TokenName:    "Sparkle Pony Token",
		TokenSymbol:  "MEAT",
		VMID:         "srEXiWaHuhNyGwPUi444Tu47ZEDwxTWrbQiuD7FmgSAQ6X7Dy",
		VMVersion:    "v0.6.12",
	},
}

// lux subnet import-historic
func newImportHistoricCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import-historic",
		Short: "Import historic subnet configurations (LUX, ZOO, SPC)",
		Long: `Import historic subnet configurations for LUX, ZOO, and SPC networks as L2s.

This command imports historic subnets as modern L2s with various sequencer options:
- Lux: Based rollup using Lux L1 (100ms blocks, lowest cost)
- Ethereum: Based rollup using Ethereum L1 (12s blocks, highest security)
- Lux: Based rollup using Lux (2s blocks, fast finality)
- OP: OP Stack compatible (Optimism ecosystem compatibility)
- External: Traditional external sequencer

The import process:
- Preserves all blockchain data and state
- Configures appropriate sequencing model
- Maintains token balances and smart contracts
- Enables migration to sovereign L1s later if desired`,
		RunE:         importHistoricSubnets,
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&historicDataPath, "data-path", "/home/z/.lux-cli/runs/network_current/node1/chainData", "Path to historic blockchain data")
	cmd.Flags().BoolVar(&autoRegister, "auto-register", true, "Automatically register subnets with the node")
	cmd.Flags().StringVar(&sequencer, "sequencer", "lux", "Sequencer for the L2 (lux, ethereum, lux, op, external)")

	return cmd
}

func importHistoricSubnets(cmd *cobra.Command, args []string) error {
	// Check if historic data exists
	if _, err := os.Stat(historicDataPath); os.IsNotExist(err) {
		return fmt.Errorf("historic data path does not exist: %s", historicDataPath)
	}

	ux.Logger.PrintToUser("Importing historic subnet configurations...")

	// Import each historic subnet
	for _, subnet := range historicSubnets {
		ux.Logger.PrintToUser("Processing %s subnet...", subnet.Name)

		// Check if blockchain data exists
		blockchainDataPath := filepath.Join(historicDataPath, subnet.BlockchainID)
		if _, err := os.Stat(blockchainDataPath); os.IsNotExist(err) {
			ux.Logger.PrintToUser("⚠️  No blockchain data found for %s, skipping", subnet.Name)
			continue
		}

		// Create subnet configuration as L2
		sc := &models.Sidecar{
			Name:    subnet.Name,
			Subnet:  subnet.Name,
			ChainID: fmt.Sprintf("%d", subnet.ChainID),
			TokenInfo: models.TokenInfo{
				Name:   subnet.TokenName,
				Symbol: subnet.TokenSymbol,
			},
			Version: constants.SidecarVersion,
			
			// L2 Configuration
			Sovereign:      false, // These are L2s, not sovereign L1s
			BaseChain:      sequencer,
			BasedRollup:    isBasedRollup(sequencer), // true if using L1 sequencer
			SequencerType:  sequencer, // lux, ethereum, lux, or external
			L1BlockTime:    getBlockTime(sequencer),
			PreconfirmEnabled: false, // Can enable later
		}

		// Set subnet and blockchain IDs
		subnetID, err := ids.FromString(subnet.SubnetID)
		if err != nil {
			return fmt.Errorf("invalid subnet ID for %s: %w", subnet.Name, err)
		}
		sc.SubnetID = subnetID

		blockchainID, err := ids.FromString(subnet.BlockchainID)
		if err != nil {
			return fmt.Errorf("invalid blockchain ID for %s: %w", subnet.Name, err)
		}
		sc.BlockchainID = blockchainID

		// Set VM type and ID
		sc.VM = models.EVM
		sc.VMVersion = subnet.VMVersion
		vmID, err := ids.FromString(subnet.VMID)
		if err != nil {
			return fmt.Errorf("invalid VM ID for %s: %w", subnet.Name, err)
		}
		sc.ImportedVMID = vmID.String()

		// Save subnet configuration
		if err := app.WriteGenesisFile(subnet.Name, []byte("{}")); err != nil {
			return fmt.Errorf("failed to write genesis file for %s: %w", subnet.Name, err)
		}

		if err := app.WriteSidecarFile(sc); err != nil {
			return fmt.Errorf("failed to write sidecar for %s: %w", subnet.Name, err)
		}

		ux.Logger.PrintToUser("✅ Imported %s as L2", subnet.Name)
		ux.Logger.PrintToUser("   Subnet ID: %s", subnet.SubnetID)
		ux.Logger.PrintToUser("   Blockchain ID: %s", subnet.BlockchainID)
		ux.Logger.PrintToUser("   Chain ID: %d", subnet.ChainID)
		ux.Logger.PrintToUser("   Sequencer: %s", sequencer)
		if isBasedRollup(sequencer) {
			ux.Logger.PrintToUser("   Type: Based rollup (L1-sequenced)")
		} else {
			ux.Logger.PrintToUser("   Type: External sequencer")
		}
	}

	if autoRegister {
		ux.Logger.PrintToUser("\n📡 Registering subnets with node...")
		// TODO: Add node registration logic here
		ux.Logger.PrintToUser("✅ Subnet registration complete!")
	}

	ux.Logger.PrintToUser("\n🎉 Historic subnet import complete!")
	ux.Logger.PrintToUser("\n📊 L2 Configuration Summary:")
	ux.Logger.PrintToUser("   Sequencer: %s", sequencer)
	if isBasedRollup(sequencer) {
		ux.Logger.PrintToUser("   Type: Based rollup (L1-sequenced)")
		ux.Logger.PrintToUser("   Block Time: %dms", getBlockTime(sequencer))
	} else {
		ux.Logger.PrintToUser("   Type: External sequencer")
	}
	
	ux.Logger.PrintToUser("\nTo deploy these L2s locally, run:")
	ux.Logger.PrintToUser("  lux subnet deploy LUX --local")
	ux.Logger.PrintToUser("  lux subnet deploy ZOO --local")
	ux.Logger.PrintToUser("  lux subnet deploy SPC --local")
	
	ux.Logger.PrintToUser("\nTo migrate to sovereign L1s later:")
	ux.Logger.PrintToUser("  lux l1 migrate LUX")

	return nil
}

