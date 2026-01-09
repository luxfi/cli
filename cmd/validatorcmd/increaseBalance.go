// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package validatorcmd

import (
	"fmt"
	"time"

	"github.com/luxfi/cli/pkg/blockchain"
	"github.com/luxfi/cli/pkg/chain"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/keychain"
	"github.com/luxfi/cli/pkg/networkoptions"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/constants"
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/sdk/validator"
	"github.com/spf13/cobra"
)

var (
	keyName         string
	useLedger       bool
	useLocalKey     bool
	ledgerAddresses []string
	balanceLUX      float64
)

func NewIncreaseBalanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "increaseBalance",
		Short: "Increases current balance of validator on P-Chain",
		Long:  `This command increases the validator P-Chain balance`,
		RunE:  increaseBalance,
		Args:  cobrautils.ExactArgs(0),
	}

	// Network flags handled at higher level to avoid conflicts
	cmd.Flags().StringVarP(&keyName, "key", "k", "", "select the key to use [testnet/devnet deploy only]")
	cmd.Flags().StringVar(&l1, "l1", "", "name of L1 (to increase balance of bootstrap validators only)")
	cmd.Flags().StringVar(&validationIDStr, "validation-id", "", "validationIDStr of the validator")
	cmd.Flags().StringVar(&nodeIDStr, "node-id", "", "node ID of the validator")
	cmd.Flags().Float64Var(&balanceLUX, "balance", 0, "amount of LUX to increase validator's balance by")
	return cmd
}

func increaseBalance(_ *cobra.Command, _ []string) error {
	network, err := networkoptions.GetNetworkFromCmdLineFlags(
		app,
		"",
		globalNetworkFlags,
		true,
		false,
		networkoptions.DefaultSupportedNetworkOptions,
		"",
	)
	if err != nil {
		return err
	}

	validationID, cancel, err := getNodeValidationID(network, l1, nodeIDStr, validationIDStr)
	if err != nil {
		return err
	}
	if cancel {
		return nil
	}
	if validationID == ids.Empty {
		return fmt.Errorf("the specified node is not a L1 validator")
	}

	// Estimate fee based on network and transaction complexity
	fee := estimateIncreaseBalanceFee(network)
	kc, err := keychain.GetKeychainFromCmdLineFlags(
		app,
		constants.PayTxsFeesMsg,
		network,
		keyName,
		useLocalKey,
		useLedger,
		ledgerAddresses,
		fee,
	)
	if err != nil {
		return err
	}

	var balance uint64
	if balanceLUX == 0 {
		// Get the first address from the list since GetNetworkBalance expects a single address
		addresses := kc.Addresses().List()
		if len(addresses) == 0 {
			return fmt.Errorf("no addresses available in keychain")
		}
		availableBalance, err := utils.GetNetworkBalance(addresses[0], network)
		if err != nil {
			return err
		}
		prompt := "How many LUX do you want to increase the balance of this validator by?"
		balanceLUX, err = blockchain.PromptValidatorBalance(app, float64(availableBalance)/float64(constants.Lux), prompt)
		if err != nil {
			return err
		}
	}
	balance = uint64(balanceLUX * float64(constants.Lux))

	// Create deployer and increase validator balance
	deployer := chain.NewPublicDeployer(app, useLedger, kc.Keychain, network)
	if err := deployer.IncreaseValidatorPChainBalance(validationID, balance); err != nil {
		return fmt.Errorf("failed to increase validator balance: %w", err)
	}

	// add a delay to safely retrieve updated balance (to avoid issues when connecting to a different API node)
	time.Sleep(5 * time.Second)

	balance, err = validator.GetValidatorBalance(network, validationID)
	if err != nil {
		return err
	}
	ux.Logger.PrintToUser("  New Validator Balance: %.5f LUX", float64(balance)/float64(constants.Lux))

	return nil
}

// estimateIncreaseBalanceFee estimates the transaction fee for increasing validator balance
func estimateIncreaseBalanceFee(network models.Network) uint64 {
	// Base fee in nLUX (1 LUX = 1e9 nLUX)
	const baseFee = 1_000_000 // 0.001 LUX base fee

	// Adjust fee based on network
	switch network {
	case models.Mainnet:
		return baseFee * 2 // Higher fee for mainnet
	case models.Testnet:
		return baseFee // Standard fee for testnet
	case models.Local:
		return 0 // No fee for local network
	default:
		return baseFee
	}
}
