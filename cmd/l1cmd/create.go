// Copyright (C) 2024, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package l1cmd

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	"github.com/luxfi/cli/v2/v2/pkg/constants"
	"github.com/luxfi/cli/v2/v2/pkg/models"
	"github.com/luxfi/cli/v2/v2/pkg/ux"
	"github.com/luxfi/cli/v2/v2/pkg/vm"
	"github.com/spf13/cobra"
)

var (
	createFlags struct {
		usePoA              bool
		usePoS              bool
		evmChainID          uint64
		tokenName           string
		tokenSymbol         string
		validatorManagement string
		force               bool
	}
)

func newCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [l1Name]",
		Short: "Create a new L1 blockchain configuration",
		Long: `Create a new L1 blockchain configuration with custom validator management.

This command creates a sovereign L1 blockchain that can use either:
- Proof of Authority (PoA): Validators managed by an owner address
- Proof of Stake (PoS): Validators stake tokens to participate

The L1 will have its own token, consensus rules, and validator set.`,
		Args: cobra.ExactArgs(1),
		RunE: createL1,
	}

	cmd.Flags().BoolVar(&createFlags.usePoA, "proof-of-authority", false, "Use Proof of Authority validator management")
	cmd.Flags().BoolVar(&createFlags.usePoS, "proof-of-stake", false, "Use Proof of Stake validator management")
	cmd.Flags().Uint64Var(&createFlags.evmChainID, "evm-chain-id", 0, "EVM chain ID for the L1")
	cmd.Flags().StringVar(&createFlags.tokenName, "token-name", "", "Native token name")
	cmd.Flags().StringVar(&createFlags.tokenSymbol, "token-symbol", "", "Native token symbol")
	cmd.Flags().BoolVarP(&createFlags.force, "force", "f", false, "Overwrite existing configuration")

	return cmd
}

func createL1(cmd *cobra.Command, args []string) error {
	l1Name := args[0]

	// Check if L1 already exists
	if _, err := app.LoadSidecar(l1Name); err == nil && !createFlags.force {
		return fmt.Errorf("L1 %s already exists. Use --force to overwrite", l1Name)
	}

	ux.Logger.PrintToUser("Creating new L1 blockchain: %s", l1Name)

	// Determine validator management type
	validatorManagement := ""
	if createFlags.usePoA && createFlags.usePoS {
		return fmt.Errorf("cannot use both PoA and PoS. Choose one")
	} else if createFlags.usePoA {
		validatorManagement = "proof-of-authority"
	} else if createFlags.usePoS {
		validatorManagement = "proof-of-stake"
	} else {
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
			validatorManagement = "proof-of-authority"
		} else {
			validatorManagement = "proof-of-stake"
		}
	}

	// Get chain ID
	chainID := createFlags.evmChainID
	if chainID == 0 {
		chainIDStr, err := app.Prompt.CaptureString("Enter EVM chain ID")
		if err != nil {
			return err
		}
		chainID, err = strconv.ParseUint(chainIDStr, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid chain ID: %w", err)
		}
	}

	// Get token info
	tokenName := createFlags.tokenName
	if tokenName == "" {
		tokenName, _ = app.Prompt.CaptureString("Enter native token name")
	}

	tokenSymbol := createFlags.tokenSymbol
	if tokenSymbol == "" {
		tokenSymbol, _ = app.Prompt.CaptureString("Enter native token symbol")
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
		big.NewInt(int64(chainID)),
		nil, // allocations will be added later
		nil, // timestamps
	)

	// Add validator manager configuration based on type
	if validatorManagement == "proof-of-authority" {
		// PoA configuration
		genesis["contractConfig"] = map[string]interface{}{
			"poaValidatorManager": map[string]interface{}{
				"enabled":              true,
				"churnPeriodSeconds":   3600,  // 1 hour
				"maximumChurnPercentage": 20,  // 20% max churn
			},
		}
		ux.Logger.PrintToUser("Configured Proof of Authority validator management")
	} else {
		// PoS configuration
		genesis["contractConfig"] = map[string]interface{}{
			"nativeTokenStakingManager": map[string]interface{}{
				"enabled":                true,
				"minimumStakeAmount":     "1000000000000000000", // 1 token
				"maximumStakeAmount":     "1000000000000000000000000", // 1M tokens
				"minimumStakeDuration":   86400, // 1 day
				"minimumDelegationFeeBips": 100, // 1%
				"maximumStakeMultiplier": 10,
				"weightToValueFactor":    1,
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