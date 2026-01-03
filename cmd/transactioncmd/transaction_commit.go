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
		Use:          "commit [subnetName]",
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

	subnetName := args[0]
	sc, err := app.LoadSidecar(subnetName)
	if err != nil {
		return err
	}
	subnetID := sc.Networks[network.String()].SubnetID
	if subnetID == ids.Empty {
		return errNoSubnetID
	}

	_, controlKeys, _, err := txutils.GetOwners(network, subnetID)
	if err != nil {
		return err
	}
	subnetAuthKeys, remainingSubnetAuthKeys, err := txutils.GetRemainingSigners(tx, controlKeys)
	if err != nil {
		return err
	}

	if len(remainingSubnetAuthKeys) != 0 {
		signedCount := len(subnetAuthKeys) - len(remainingSubnetAuthKeys)
		ux.Logger.PrintToUser("%d of %d required signatures have been signed.", signedCount, len(subnetAuthKeys))
		ux.Logger.PrintToUser("Remaining signers for %s:", subnetName)
		for _, addr := range remainingSubnetAuthKeys {
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
		ux.Logger.PrintToUser("Blockchain %s deployed successfully", subnetName)
		ux.Logger.PrintToUser("Chain ID: %s", subnetID)
		ux.Logger.PrintToUser("Blockchain ID: %s", txID)
		return app.UpdateSidecarNetworks(&sc, network, subnetID, txID)
	}
	ux.Logger.PrintToUser("Transaction successful, transaction ID: %s", txID)

	return nil
}
