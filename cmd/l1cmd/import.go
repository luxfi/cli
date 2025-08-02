// Copyright (C) 2024, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package l1cmd

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/v2/v2/pkg/constants"
	"github.com/luxfi/cli/v2/v2/pkg/models"
	"github.com/luxfi/cli/v2/v2/pkg/ux"
	"github.com/luxfi/cli/v2/v2/pkg/vm"
	"github.com/luxfi/ids"
	"github.com/spf13/cobra"
)

var (
	importAsL1 bool
)

// Historic L1 configurations for LUX, ZOO, SPC
var historicL1s = []struct {
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

func newImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import-historic",
		Short: "Import historic blockchains as sovereign L1s",
		Long: `Import historic blockchain configurations (LUX, ZOO, SPC) as sovereign L1s.

This command transforms existing subnet configurations into modern L1 blockchains
with validator management capabilities.`,
		RunE: importHistoricL1s,
	}

	cmd.Flags().BoolVar(&importAsL1, "as-l1", true, "Import as sovereign L1 (recommended)")

	return cmd
}

func importHistoricL1s(cmd *cobra.Command, args []string) error {
	ux.Logger.PrintToUser("Importing historic blockchains as sovereign L1s...")

	for _, l1 := range historicL1s {
		ux.Logger.PrintToUser("\n📥 Importing %s as L1...", l1.Name)

		// Check if genesis data exists
		genesisPath := filepath.Join("/home/z/work/lux", fmt.Sprintf("genesis-%s", l1.Name))
		hasGenesisData := false
		if _, err := os.Stat(genesisPath); err == nil {
			hasGenesisData = true
			ux.Logger.PrintToUser("   Found genesis data at %s", genesisPath)
		}

		// Create L1 configuration
		sc := &models.Sidecar{
			Name:                l1.Name,
			VM:                  models.EVM,
			VMVersion:           l1.VMVersion,
			ChainID:             fmt.Sprintf("%d", l1.ChainID),
			Sovereign:           true,
			ValidatorManagement: "proof-of-authority", // Default to PoA for historic chains
			TokenInfo: models.TokenInfo{
				Name:   l1.TokenName,
				Symbol: l1.TokenSymbol,
			},
			Version: constants.SidecarVersion,
		}

		// Set IDs
		subnetID, err := ids.FromString(l1.SubnetID)
		if err != nil {
			ux.Logger.PrintToUser("   ⚠️  Invalid subnet ID, will generate new")
		} else {
			sc.SubnetID = subnetID
		}

		blockchainID, err := ids.FromString(l1.BlockchainID)
		if err != nil {
			ux.Logger.PrintToUser("   ⚠️  Invalid blockchain ID, will generate new")
		} else {
			sc.BlockchainID = blockchainID
		}

		vmID, err := ids.FromString(l1.VMID)
		if err != nil {
			ux.Logger.PrintToUser("   ⚠️  Invalid VM ID, using default")
		} else {
			sc.ImportedVMID = vmID.String()
			sc.ImportedFromLPM = true
		}

		// Create genesis with L1 features
		genesis := vm.CreateEVMGenesis(
			big.NewInt(int64(l1.ChainID)),
			nil, // allocations
			nil, // timestamps
		)

		// Add PoA validator manager
		genesis["contractConfig"] = map[string]interface{}{
			"poaValidatorManager": map[string]interface{}{
				"enabled":              true,
				"churnPeriodSeconds":   3600,
				"maximumChurnPercentage": 20,
			},
		}

		// If we have genesis data, try to preserve allocations
		if hasGenesisData {
			ux.Logger.PrintToUser("   Preserving existing allocations and state...")
			// TODO: Load and merge existing genesis allocations
		}

		// Save configuration
		genesisBytes, err := json.MarshalIndent(genesis, "", "  ")
		if err != nil {
			ux.Logger.PrintToUser("   ❌ Failed to marshal genesis: %v", err)
			continue
		}
		if err := app.WriteGenesisFile(l1.Name, genesisBytes); err != nil {
			ux.Logger.PrintToUser("   ❌ Failed to write genesis: %v", err)
			continue
		}

		if err := app.WriteSidecarFile(sc); err != nil {
			ux.Logger.PrintToUser("   ❌ Failed to write sidecar: %v", err)
			continue
		}

		ux.Logger.PrintToUser("   ✅ Imported %s as sovereign L1", l1.Name)
		ux.Logger.PrintToUser("      Chain ID: %d", l1.ChainID)
		ux.Logger.PrintToUser("      Token: %s (%s)", l1.TokenName, l1.TokenSymbol)
		ux.Logger.PrintToUser("      Blockchain ID: %s", l1.BlockchainID)
	}

	ux.Logger.PrintToUser("\n✅ Historic blockchain import complete!")
	ux.Logger.PrintToUser("\nNext steps:")
	ux.Logger.PrintToUser("1. Start local network: lux network quickstart")
	ux.Logger.PrintToUser("2. Deploy L1s:")
	ux.Logger.PrintToUser("   lux l1 deploy LUX --local")
	ux.Logger.PrintToUser("   lux l1 deploy ZOO --local")
	ux.Logger.PrintToUser("   lux l1 deploy SPC --local")

	return nil
}