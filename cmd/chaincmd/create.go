// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package chaincmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/ux"
	luxconstants "github.com/luxfi/constants"
	"github.com/luxfi/evm/core"
	"github.com/luxfi/sdk/models"
	"github.com/spf13/cobra"
)

var (
	chainType        string // l1, l2, l3
	sequencerType    string // lux, ethereum, op, external
	forceCreate      bool
	genesisFile      string
	customVMBin      string
	useEVM           bool
	useCustomVM      bool
	vmVersion        string
	useLatestVM      bool
	enablePreconfirm bool
)

func newCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [chainName]",
		Short: "Create a new blockchain configuration",
		Long: `Create a new blockchain configuration for deployment.

Chain Types:
  l1    Sovereign L1 with independent validation
  l2    Layer 2 rollup/subnet (default)
  l3    App-specific L3 chain

Sequencer Options (for L2):
  lux       Lux-based rollup, 100ms blocks (default)
  ethereum  Ethereum-based rollup, 12s blocks
  op        OP Stack compatible
  external  External/custom sequencer

Examples:
  # Create L2 with Lux sequencing
  lux chain create mychain

  # Create sovereign L1
  lux chain create mychain --type=l1

  # Create L3 on existing L2
  lux chain create myapp --type=l3 --l2=mychain

  # Create with custom genesis
  lux chain create mychain --genesis=/path/to/genesis.json`,
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		RunE:         createChain,
	}

	cmd.Flags().StringVar(&chainType, "type", "l2", "Chain type: l1, l2, l3")
	cmd.Flags().StringVar(&sequencerType, "sequencer", "lux", "Sequencer: lux, ethereum, op, external")
	cmd.Flags().BoolVarP(&forceCreate, "force", "f", false, "Overwrite existing configuration")
	cmd.Flags().StringVar(&genesisFile, "genesis", "", "Path to custom genesis file")
	cmd.Flags().StringVar(&customVMBin, "vm", "", "Path to custom VM binary")
	cmd.Flags().BoolVar(&useEVM, "evm", false, "Use Lux EVM")
	cmd.Flags().BoolVar(&useCustomVM, "custom", false, "Use custom VM")
	cmd.Flags().StringVar(&vmVersion, "vm-version", "", "VM version to use")
	cmd.Flags().BoolVar(&useLatestVM, "latest", false, "Use latest VM version")
	cmd.Flags().BoolVar(&enablePreconfirm, "enable-preconfirm", false, "Enable pre-confirmations")

	return cmd
}

func createChain(cmd *cobra.Command, args []string) error {
	chainName := args[0]

	// Validate chain name
	if err := validateChainName(chainName); err != nil {
		return err
	}

	// Check if configuration already exists
	if app.ChainConfigExists(chainName) && !forceCreate {
		return fmt.Errorf("chain %s already exists. Use --force to overwrite", chainName)
	}

	// Determine VM type
	var vmType models.VMType
	switch {
	case useEVM:
		vmType = models.EVM
	case useCustomVM:
		vmType = models.CustomVM
	default:
		// Default to EVM for l2/l3, prompt for l1
		if chainType == "l1" {
			vmType = models.EVM // Default to EVM for now
		} else {
			vmType = models.EVM
		}
	}

	// Handle genesis
	var chainGenesis []byte
	var err error
	if genesisFile != "" {
		chainGenesis, err = os.ReadFile(genesisFile)
		if err != nil {
			return fmt.Errorf("failed to read genesis file: %w", err)
		}
		ux.Logger.PrintToUser("Importing genesis")
	} else {
		// Generate default genesis
		chainGenesis, err = generateDefaultGenesis(chainName, chainType)
		if err != nil {
			return fmt.Errorf("failed to generate genesis: %w", err)
		}
	}

	// Validate genesis
	if vmType == models.EVM {
		var genesis core.Genesis
		if err := json.Unmarshal(chainGenesis, &genesis); err != nil {
			return fmt.Errorf("invalid genesis format: %w", err)
		}
	}

	// Create sidecar configuration
	sc := models.Sidecar{
		Name:              chainName,
		VM:                vmType,
		Net:               chainName, // Network name (not subnet)
		TokenName:         "TOKEN",
		ChainID:           "",
		Version:           "1.4.0",
		BasedRollup:       chainType == "l2",
		Sovereign:         chainType == "l1",
		SequencerType:     sequencerType,
		PreconfirmEnabled: enablePreconfirm,
		ChainLayer:        getChainLayer(chainType),
	}

	// Set L1 block time based on sequencer
	switch sequencerType {
	case "lux":
		sc.L1BlockTime = 100 // 100ms
	case "ethereum":
		sc.L1BlockTime = 12000 // 12s
	default:
		sc.L1BlockTime = 2000 // 2s default
	}

	// Handle custom VM
	if useCustomVM && customVMBin != "" {
		// TODO: Implement custom VM copy
		ux.Logger.PrintToUser("Custom VM support coming soon")
	}

	// Get VM version and RPC version if using EVM
	if vmType == models.EVM {
		if vmVersion != "" {
			sc.VMVersion = vmVersion
		} else {
			sc.VMVersion = luxconstants.DefaultEVMVersion // Default EVM version
		}
		// Set correct RPC version for Lux EVM
		// This must match the running node's EVM RPC version
		sc.RPCVersion = luxconstants.DefaultEVMRPCVersion
	}

	// Create chain directory
	chainDir := filepath.Join(app.GetChainsDir(), chainName)
	if err := os.MkdirAll(chainDir, constants.DefaultPerms755); err != nil {
		return fmt.Errorf("failed to create chain directory: %w", err)
	}

	// Write genesis
	genesisPath := filepath.Join(chainDir, constants.GenesisFileName)
	if err := os.WriteFile(genesisPath, chainGenesis, constants.WriteReadReadPerms); err != nil {
		return fmt.Errorf("failed to write genesis: %w", err)
	}

	// Write sidecar
	if err := app.CreateSidecar(&sc); err != nil {
		return fmt.Errorf("failed to create sidecar: %w", err)
	}

	// Success message
	ux.Logger.PrintToUser("creating %s chain %s", chainType, chainName)
	ux.Logger.PrintToUser("🔧 Chain Configuration:")
	ux.Logger.PrintToUser("   Type: %s", strings.ToUpper(chainType))
	if chainType == "l2" {
		ux.Logger.PrintToUser("   Sequencer: %s", sequencerType)
		ux.Logger.PrintToUser("   Block Time: %dms", sc.L1BlockTime)
	}
	ux.Logger.PrintToUser("Successfully created chain configuration")

	return nil
}

func validateChainName(name string) error {
	if name == "" {
		return errors.New("chain name cannot be empty")
	}
	if len(name) > 32 {
		return errors.New("chain name must be 32 characters or less")
	}
	// Check for reserved names
	reserved := []string{"c", "p", "x", "primary", "platform"}
	for _, r := range reserved {
		if strings.EqualFold(name, r) {
			return fmt.Errorf("%s is a reserved chain name", name)
		}
	}
	return nil
}

// getChainLayer returns the chain layer (1=L1, 2=L2, 3=L3)
func getChainLayer(chainType string) int {
	switch chainType {
	case "l1":
		return 1
	case "l2":
		return 2
	case "l3":
		return 3
	default:
		return 2 // Default to L2
	}
}

func generateDefaultGenesis(chainName, chainType string) ([]byte, error) {
	// Default genesis for EVM-compatible chains
	genesis := map[string]interface{}{
		"config": map[string]interface{}{
			"chainId":             200200,
			"homesteadBlock":      0,
			"eip150Block":         0,
			"eip155Block":         0,
			"eip158Block":         0,
			"byzantiumBlock":      0,
			"constantinopleBlock": 0,
			"petersburgBlock":     0,
			"istanbulBlock":       0,
			"muirGlacierBlock":    0,
			"evmTimestamp":        0,
			"feeConfig": map[string]interface{}{
				"gasLimit":                 8000000,
				"targetBlockRate":          2,
				"minBaseFee":               25000000000,
				"targetGas":                15000000,
				"baseFeeChangeDenominator": 36,
				"minBlockGasCost":          0,
				"maxBlockGasCost":          1000000,
				"blockGasCostStep":         200000,
			},
			"allowFeeRecipients": true,
		},
		"alloc": map[string]interface{}{
			// Default funded address from LUX_MNEMONIC
			"9011E888251AB053B7bD1cdB598Db4f9DEd94714": map[string]interface{}{
				"balance": "0x193e5939a08ce9dbd480000000",
			},
		},
		"nonce":      "0x0",
		"timestamp":  "0x6727e9c3",
		"extraData":  "0x",
		"gasLimit":   "0x7a1200",
		"difficulty": "0x0",
		"mixHash":    "0x0000000000000000000000000000000000000000000000000000000000000000",
		"coinbase":   "0x0000000000000000000000000000000000000000",
	}

	return json.MarshalIndent(genesis, "", "  ")
}
