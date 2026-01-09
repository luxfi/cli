// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package l1cmd

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/cli/pkg/vm"
	"github.com/luxfi/constants"
	"github.com/luxfi/sdk/models"
	"github.com/spf13/cobra"
)

// Validator management types
const (
	ValidatorManagementPoA = "proof-of-authority"
	ValidatorManagementPoS = "proof-of-stake"
)

// Network types
const (
	NetworkLocal   = "local"
	NetworkTestnet = "testnet"
	NetworkMainnet = "mainnet"
)

var createFlags struct {
	usePoA              bool
	usePoS              bool
	evmChainID          uint64
	tokenName           string
	tokenSymbol         string
	validatorManagement string
	force               bool
	nonInteractive      bool
}

func newCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [l1Name]",
		Short: "Create a new L1 blockchain configuration",
		Long: `Create a new L1 blockchain configuration with custom validator management.

This command creates a sovereign L1 blockchain that can use either:
- Proof of Authority (PoA): Validators managed by an owner address
- Proof of Stake (PoS): Validators stake tokens to participate

The L1 will have its own token, consensus rules, and validator set.

NON-INTERACTIVE MODE:

  Use --non-interactive to skip all prompts and use provided flags or defaults.
  If a required value cannot be determined, the command fails with a clear error
  showing which flag to use.

  Required for non-interactive mode (if not provided, uses defaults):
    --proof-of-authority OR --proof-of-stake  (default: proof-of-authority)
    --evm-chain-id                            (default: 200200)
    --token-name                              (default: TOKEN)
    --token-symbol                            (default: TKN)

EXAMPLES:

  # Interactive mode (prompts for missing values)
  lux l1 create mychain

  # Fully non-interactive with defaults
  lux l1 create mychain --non-interactive

  # Non-interactive with custom values
  lux l1 create mychain --non-interactive --proof-of-stake --evm-chain-id=12345 --token-name=MYTOKEN --token-symbol=MTK`,
		Args: cobra.ExactArgs(1),
		RunE: createL1,
	}

	cmd.Flags().BoolVar(&createFlags.usePoA, "proof-of-authority", false, "Use Proof of Authority validator management")
	cmd.Flags().BoolVar(&createFlags.usePoS, "proof-of-stake", false, "Use Proof of Stake validator management")
	cmd.Flags().Uint64Var(&createFlags.evmChainID, "evm-chain-id", 0, "EVM chain ID for the L1 (default: 200200)")
	cmd.Flags().StringVar(&createFlags.tokenName, "token-name", "", "Native token name (default: TOKEN)")
	cmd.Flags().StringVar(&createFlags.tokenSymbol, "token-symbol", "", "Native token symbol (default: TKN)")
	cmd.Flags().BoolVarP(&createFlags.force, "force", "f", false, "Overwrite existing configuration")
	cmd.Flags().BoolVar(&createFlags.nonInteractive, "non-interactive", false, "Skip all prompts, use flags or defaults")

	return cmd
}

func createL1(_ *cobra.Command, args []string) error {
	l1Name := args[0]

	// Check if L1 already exists
	if _, err := app.LoadSidecar(l1Name); err == nil && !createFlags.force {
		return fmt.Errorf("L1 %s already exists. Use --force to overwrite", l1Name)
	}

	ux.Logger.PrintToUser("Creating new L1 blockchain: %s", l1Name)

	// Determine validator management type
	validatorManagement := ""
	switch {
	case createFlags.usePoA && createFlags.usePoS:
		return fmt.Errorf("cannot use both PoA and PoS. Choose one")
	case createFlags.usePoA:
		validatorManagement = ValidatorManagementPoA
	case createFlags.usePoS:
		validatorManagement = ValidatorManagementPoS
	case createFlags.nonInteractive:
		// Default to PoA in non-interactive mode
		validatorManagement = ValidatorManagementPoA
		ux.Logger.PrintToUser("Using default: proof-of-authority (use --proof-of-stake to change)")
	default:
		// Interactive prompt
		validatorManagementOptions := []string{"Proof of Authority", "Proof of Stake"}
		validatorManagementChoice, err := app.Prompt.CaptureList(
			"Choose validator management type",
			validatorManagementOptions,
		)
		if err != nil {
			return err
		}
		if validatorManagementChoice == "Proof of Authority" {
			validatorManagement = ValidatorManagementPoA
		} else {
			validatorManagement = ValidatorManagementPoS
		}
	}

	// Get chain ID
	chainID := createFlags.evmChainID
	if chainID == 0 {
		if createFlags.nonInteractive {
			// Default chain ID in non-interactive mode
			chainID = 200200
			ux.Logger.PrintToUser("Using default chain ID: %d (use --evm-chain-id to change)", chainID)
		} else {
			chainIDStr, err := app.Prompt.CaptureString("Enter EVM chain ID")
			if err != nil {
				return err
			}
			chainID, err = strconv.ParseUint(chainIDStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid chain ID: %w", err)
			}
		}
	}

	// Get token info
	tokenName := createFlags.tokenName
	if tokenName == "" {
		if createFlags.nonInteractive {
			tokenName = "TOKEN"
		} else {
			tokenName, _ = app.Prompt.CaptureString("Enter native token name")
		}
	}

	tokenSymbol := createFlags.tokenSymbol
	if tokenSymbol == "" {
		if createFlags.nonInteractive {
			tokenSymbol = "TKN"
		} else {
			tokenSymbol, _ = app.Prompt.CaptureString("Enter native token symbol")
		}
	}

	// Create L1 configuration
	sc := &models.Sidecar{
		Name:                l1Name,
		VM:                  models.EVM,
		VMVersion:           constants.LatestEVMVersion,
		ChainID:             fmt.Sprintf("%d", chainID),
		Sovereign:           true,
		ValidatorManagement: validatorManagement,
		TokenInfo: models.TokenInfo{
			Name:   tokenName,
			Symbol: tokenSymbol,
		},
		Version: constants.SidecarVersion,
	}

	// Create genesis configuration
	genesis := vm.CreateEVMGenesis(
		big.NewInt(int64(chainID)), //nolint:gosec // G115: Chain ID is within int64 range
		nil,                        // allocations will be added later
		nil,                        // timestamps
	)

	// Add validator manager configuration based on type
	if validatorManagement == ValidatorManagementPoA {
		// PoA configuration
		genesis["contractConfig"] = map[string]interface{}{
			"poaValidatorManager": map[string]interface{}{
				"enabled":                true,
				"churnPeriodSeconds":     3600, // 1 hour
				"maximumChurnPercentage": 20,   // 20% max churn
			},
		}
		ux.Logger.PrintToUser("Configured Proof of Authority validator management")
	} else {
		// PoS configuration
		genesis["contractConfig"] = map[string]interface{}{
			"nativeTokenStakingManager": map[string]interface{}{
				"enabled":                  true,
				"minimumStakeAmount":       "1000000000000000000",       // 1 token
				"maximumStakeAmount":       "1000000000000000000000000", // 1M tokens
				"minimumStakeDuration":     86400,                       // 1 day
				"minimumDelegationFeeBips": 100,                         // 1%
				"maximumStakeMultiplier":   10,
				"weightToValueFactor":      1,
			},
		}
		ux.Logger.PrintToUser("Configured Proof of Stake validator management")
	}

	// Save configuration
	genesisBytes, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal genesis: %w", err)
	}
	if err := app.WriteGenesisFile(l1Name, genesisBytes); err != nil {
		return fmt.Errorf("failed to write genesis: %w", err)
	}

	if err := app.WriteSidecarFile(sc); err != nil {
		return fmt.Errorf("failed to write sidecar: %w", err)
	}

	ux.Logger.PrintToUser("✅ Created L1 configuration: %s", l1Name)
	ux.Logger.PrintToUser("   Chain ID: %d", chainID)
	ux.Logger.PrintToUser("   Token: %s (%s)", tokenName, tokenSymbol)
	ux.Logger.PrintToUser("   Validator Management: %s", validatorManagement)
	ux.Logger.PrintToUser("\nNext steps:")
	ux.Logger.PrintToUser("   Deploy locally: lux l1 deploy %s --local", l1Name)
	ux.Logger.PrintToUser("   Deploy to testnet: lux l1 deploy %s --testnet", l1Name)

	return nil
}
