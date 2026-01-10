// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package transactioncmd

import (
	"github.com/luxfi/cli/pkg/chain"
	keychainpkg "github.com/luxfi/cli/pkg/keychain"
	"github.com/luxfi/cli/pkg/txutils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/ids"
	"github.com/luxfi/node/vms/secp256k1fx"
	"github.com/spf13/cobra"
)

// lux transaction commit
func newTransactionCommitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "commit [chainName]",
		Short:        "commit a transaction",
		Long:         "The transaction commit command commits a transaction by submitting it to the P-Chain.",
		RunE:         commitTx,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&inputTxPath, inputTxPathFlag, "", "Path to the transaction signed by all signatories")
	return cmd
}

func commitTx(_ *cobra.Command, args []string) error {
	var err error
	if inputTxPath == "" {
		inputTxPath, err = app.Prompt.CaptureExistingFilepath("What is the path to the signed transactions file?")
		if err != nil {
			return err
		}
	}
	tx, err := txutils.LoadFromDisk(inputTxPath)
	if err != nil {
		return err
	}

	network, err := txutils.GetNetwork(tx)
	if err != nil {
		return err
	}

	chainName := args[0]
	sc, err := app.LoadSidecar(chainName)
	if err != nil {
		return err
	}
	chainID := sc.Networks[network.String()].ChainID
	if chainID == ids.Empty {
		return errNoChainID
	}

	_, controlKeys, _, err := txutils.GetOwners(network, chainID)
	if err != nil {
		return err
	}
	chainAuthKeys, remainingChainAuthKeys, err := txutils.GetRemainingSigners(tx, controlKeys)
	if err != nil {
		return err
	}

	if len(remainingChainAuthKeys) != 0 {
		signedCount := len(chainAuthKeys) - len(remainingChainAuthKeys)
		ux.Logger.PrintToUser("%d of %d required signatures have been signed.", signedCount, len(chainAuthKeys))
		ux.Logger.PrintToUser("Remaining signers for %s:", chainName)
		for _, addr := range remainingChainAuthKeys {
			ux.Logger.PrintToUser("  - %s", addr)
		}
		ux.Logger.PrintToUser("Transaction file: %s", inputTxPath)
		return nil
	}

	// get kc with some random address, to pass wallet creation checks
	secpKC := secp256k1fx.NewKeychain()
	_, err = secpKC.New()
	if err != nil {
		return err
	}
	// Wrap the secp256k1fx keychain to implement node keychain interface
	kc := keychainpkg.WrapSecp256k1fxKeychain(secpKC)

	deployer := chain.NewPublicDeployer(app, false, kc, network)
	txID, err := deployer.Commit(tx)
	if err != nil {
		return err
	}

	if txutils.IsCreateChainTx(tx) {
		ux.Logger.PrintToUser("Blockchain %s deployed successfully", chainName)
		ux.Logger.PrintToUser("Chain ID: %s", chainID)
		ux.Logger.PrintToUser("Blockchain ID: %s", txID)
		return app.UpdateSidecarNetworks(&sc, network, chainID, txID)
	}
	ux.Logger.PrintToUser("Transaction successful, transaction ID: %s", txID)

	return nil
}
