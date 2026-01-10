// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package transactioncmd

import (
	"errors"

	"github.com/luxfi/cli/pkg/chain"
	keychainpkg "github.com/luxfi/cli/pkg/keychain"
	"github.com/luxfi/cli/pkg/txutils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/sdk/prompts"
	"github.com/spf13/cobra"
)

const inputTxPathFlag = "input-tx-filepath"

var (
	inputTxPath     string
	keyName         string
	useLedger       bool
	ledgerAddresses []string

	errNoChainID                  = errors.New("failed to find the chain ID for this chain, has it been deployed/created on this network?")
	errMutuallyExclusiveKeyLedger = errors.New("--key and --ledger/--ledger-addrs are mutually exclusive")
	errStoredKeyOnMainnet         = errors.New("--key is not available for mainnet operations")
)

// lux transaction sign
func newTransactionSignCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "sign [chainName]",
		Short:        "sign a transaction",
		Long:         "The transaction sign command signs a multisig transaction.",
		RunE:         signTx,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&inputTxPath, inputTxPathFlag, "", "Path to the transaction file for signing")
	cmd.Flags().StringVarP(&keyName, "key", "k", "", "select the key to use [testnet only]")
	cmd.Flags().BoolVarP(&useLedger, "ledger", "g", false, "use ledger instead of key (always true on mainnet, defaults to false on testnet)")
	cmd.Flags().StringSliceVar(&ledgerAddresses, "ledger-addrs", []string{}, "use the given ledger addresses")
	return cmd
}

func signTx(_ *cobra.Command, args []string) error {
	var err error
	if inputTxPath == "" {
		inputTxPath, err = app.Prompt.CaptureExistingFilepath("What is the path to the transactions file which needs signing?")
		if err != nil {
			return err
		}
	}
	tx, err := txutils.LoadFromDisk(inputTxPath)
	if err != nil {
		return err
	}

	if len(ledgerAddresses) > 0 {
		useLedger = true
	}

	if useLedger && keyName != "" {
		return errMutuallyExclusiveKeyLedger
	}

	// we need network to decide if ledger is forced (mainnet)
	network, err := txutils.GetNetwork(tx)
	if err != nil {
		return err
	}
	switch network {
	case models.Testnet, models.Local:
		if !useLedger && keyName == "" {
			useLedger, keyName, err = prompts.GetTestnetKeyOrLedger(app.Prompt, "sign transaction", app.GetKeyDir())
			if err != nil {
				return err
			}
		}
	case models.Mainnet:
		useLedger = true
		if keyName != "" {
			return errStoredKeyOnMainnet
		}
	default:
		return errors.New("unsupported network")
	}

	// we need chain wallet signing validation + process
	chainName := args[0]
	sc, err := app.LoadSidecar(chainName)
	if err != nil {
		return err
	}
	chainID := sc.Networks[network.String()].SubnetID
	if chainID == ids.Empty {
		return errNoChainID
	}

	_, controlKeys, _, err := txutils.GetOwners(network, chainID)
	if err != nil {
		return err
	}

	// get the remaining tx signers so as to check that the wallet does contain an expected signer
	chainAuthKeys, remainingChainAuthKeys, err := txutils.GetRemainingSigners(tx, controlKeys)
	if err != nil {
		return err
	}

	if len(remainingChainAuthKeys) == 0 {
		ux.Logger.PrintToUser("Transaction for %s is ready to commit", chainName)
		ux.Logger.PrintToUser("Run: lux transaction commit %s --input-tx-filepath %s", chainName, inputTxPath)
		return nil
	}

	// get keychain accessor
	kc, err := keychainpkg.GetKeychain(app, keyName != "", useLedger, ledgerAddresses, keyName, network, 0)
	if err != nil {
		return err
	}

	deployer := chain.NewPublicDeployer(app, useLedger, kc.Keychain, network)
	if err := deployer.Sign(tx, remainingChainAuthKeys, chainID); err != nil {
		if errors.Is(err, chain.ErrNoChainAuthKeysInWallet) {
			ux.Logger.PrintToUser("There are no required chain auth keys present in the wallet")
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("Expected one of:")
			for _, addr := range remainingChainAuthKeys {
				ux.Logger.PrintToUser("  %s", addr)
			}
			return nil
		}
		return err
	}

	// update the remaining tx signers after the signature has been done
	_, remainingChainAuthKeys, err = txutils.GetRemainingSigners(tx, controlKeys)
	if err != nil {
		return err
	}

	// Save the transaction to disk
	if err := txutils.SaveToDisk(tx, inputTxPath, true); err != nil {
		return err
	}

	signedCount := len(chainAuthKeys) - len(remainingChainAuthKeys)
	ux.Logger.PrintToUser("%d of %d required signatures have been signed.", signedCount, len(chainAuthKeys))
	if len(remainingChainAuthKeys) > 0 {
		ux.Logger.PrintToUser("Remaining signers:")
		for _, addr := range remainingChainAuthKeys {
			ux.Logger.PrintToUser("  - %s", addr)
		}
	} else {
		ux.Logger.PrintToUser("Transaction is fully signed and ready to commit.")
	}

	return nil
}
