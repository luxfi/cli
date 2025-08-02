// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package blockchaincmd

import (
	"fmt"

	"github.com/luxfi/cli/v2/v2/pkg/cobrautils"
	"github.com/luxfi/cli/v2/v2/pkg/keychain"
	"github.com/luxfi/cli/v2/v2/pkg/networkoptions"
	"github.com/luxfi/cli/v2/v2/pkg/prompts"
	"github.com/luxfi/cli/v2/v2/pkg/subnet"
	"github.com/luxfi/cli/v2/v2/pkg/txutils"
	"github.com/luxfi/cli/v2/v2/pkg/utils"
	"github.com/luxfi/cli/v2/v2/pkg/ux"
	"github.com/luxfi/ids"

	"github.com/spf13/cobra"
)

// lux blockchain changeOwner
func newChangeOwnerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "changeOwner [blockchainName]",
		Short: "Change owner of the blockchain",
		Long:  `The blockchain changeOwner changes the owner of the deployed Blockchain.`,
		RunE:  changeOwner,
		Args:  cobrautils.ExactArgs(1),
	}
	networkoptions.AddNetworkFlagsToCmd(cmd, &globalNetworkFlags, true, networkoptions.DefaultSupportedNetworkOptions)
	cmd.Flags().BoolVarP(&useLedger, "ledger", "g", false, "use ledger instead of key (always true on mainnet, defaults to false on testnet/devnet)")
	cmd.Flags().StringSliceVar(&ledgerAddresses, "ledger-addrs", []string{}, "use the given ledger addresses")
	cmd.Flags().StringVarP(&keyName, "key", "k", "", "select the key to use [testnet/devnet]")
	cmd.Flags().BoolVarP(&useEwoq, "ewoq", "e", false, "use ewoq key [testnet/devnet]")
	cmd.Flags().StringSliceVar(&subnetAuthKeys, "auth-keys", nil, "control keys that will be used to authenticate transfer blockchain ownership tx")
	cmd.Flags().BoolVarP(&sameControlKey, "same-control-key", "s", false, "use the fee-paying key as control key")
	cmd.Flags().StringSliceVar(&controlKeys, "control-keys", nil, "addresses that may make blockchain changes")
	cmd.Flags().Uint32Var(&threshold, "threshold", 0, "required number of control key signatures to make blockchain changes")
	cmd.Flags().StringVar(&outputTxPath, "output-tx-path", "", "file path of the transfer blockchain ownership tx")
	return cmd
}

func changeOwner(_ *cobra.Command, args []string) error {
	blockchainName := args[0]

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

	// TODO: will estimate fee in subsecuent PR
	fee := uint64(0)
	kc, err := keychain.GetKeychainFromCmdLineFlags(
		app,
		"pay fees",
		network,
		keyName,
		useEwoq,
		useLedger,
		ledgerAddresses,
		fee,
	)
	if err != nil {
		return err
	}

	network.HandlePublicNetworkSimulation()

	if outputTxPath != "" {
		if utils.FileExists(outputTxPath) {
			return fmt.Errorf("outputTxPath %q already exists", outputTxPath)
		}
	}

	_, err = ValidateSubnetNameAndGetChains([]string{blockchainName})
	if err != nil {
		return err
	}

	sc, err := app.LoadSidecar(blockchainName)
	if err != nil {
		return err
	}

	subnetID := sc.Networks[network.Name()].SubnetID
	if subnetID == ids.Empty {
		return errNoSubnetID
	}

	_, currentControlKeys, currentThreshold, err := txutils.GetOwners(network, subnetID)
	if err != nil {
		return err
	}

	// add control keys to the keychain whenever possible
	if err := kc.AddAddresses(currentControlKeys); err != nil {
		return err
	}

	kcKeys, err := kc.PChainFormattedStrAddresses()
	if err != nil {
		return err
	}

	// get keys for add validator tx signing
	if subnetAuthKeys != nil {
		if err := prompts.CheckSubnetAuthKeys(kcKeys, subnetAuthKeys, currentControlKeys, currentThreshold); err != nil {
			return err
		}
	} else {
		subnetAuthKeys, err = prompts.GetSubnetAuthKeys(app.Prompt, kcKeys, currentControlKeys, currentThreshold)
		if err != nil {
			return err
		}
	}
	ux.Logger.PrintToUser("Your auth keys for add validator tx creation: %s", subnetAuthKeys)

	controlKeys, threshold, err = promptOwners(
		kc,
		controlKeys,
		sameControlKey,
		threshold,
		nil,
		false,
	)
	if err != nil {
		return err
	}

	deployer := subnet.NewPublicDeployer(app, kc, network)
	isFullySigned, tx, remainingSubnetAuthKeys, err := deployer.TransferSubnetOwnership(
		currentControlKeys,
		subnetAuthKeys,
		subnetID,
		controlKeys,
		threshold,
	)
	if err != nil {
		return err
	}
	if !isFullySigned {
		if err := SaveNotFullySignedTx(
			"Transfer Blockchain Ownership",
			tx,
			blockchainName,
			subnetAuthKeys,
			remainingSubnetAuthKeys,
			outputTxPath,
			false,
		); err != nil {
			return err
		}
	}
	return nil
}
