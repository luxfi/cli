// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package contractcmd

import (
	"math/big"

	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/networkoptions"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/crypto"
	"github.com/luxfi/geth/common"
	luxlog "github.com/luxfi/log"
	"github.com/luxfi/sdk/contract"
	"github.com/luxfi/sdk/prompts"

	"github.com/spf13/cobra"
)

type DeployERC20Flags struct {
	Network         networkoptions.NetworkFlags
	PrivateKeyFlags contract.PrivateKeyFlags
	chainFlags      contract.ChainSpec
	symbol          string
	funded          string
	supply          uint64
	rpcEndpoint     string
}

var deployERC20Flags DeployERC20Flags

// lux contract deploy erc20
func newDeployERC20Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "erc20",
		Short: "Deploy an ERC20 token into a given Network and Blockchain",
		Long:  "Deploy an ERC20 token into a given Network and Blockchain",
		RunE:  deployERC20,
		Args:  cobrautils.ExactArgs(0),
	}
	// Network flags handled globally to avoid conflicts
	deployERC20Flags.PrivateKeyFlags.AddToCmd(cmd, "as contract deployer")
	// enabling blockchain names, C-Chain and blockchain IDs
	deployERC20Flags.chainFlags.SetEnabled(true, true, false, false, true)
	deployERC20Flags.chainFlags.AddToCmd(cmd, "deploy the ERC20 contract into %s")
	cmd.Flags().StringVar(&deployERC20Flags.symbol, "symbol", "", "set the token symbol")
	cmd.Flags().Uint64Var(&deployERC20Flags.supply, "supply", 0, "set the token supply")
	cmd.Flags().StringVar(&deployERC20Flags.funded, "funded", "", "set the funded address")
	cmd.Flags().StringVar(&deployERC20Flags.rpcEndpoint, "rpc", "", "deploy the contract into the given rpc endpoint")
	return cmd
}

func deployERC20(_ *cobra.Command, _ []string) error {
	network, err := networkoptions.GetNetworkFromCmdLineFlags(
		app,
		"",
		deployERC20Flags.Network,
		true,
		false,
		networkoptions.DefaultSupportedNetworkOptions,
		"",
	)
	if err != nil {
		return err
	}
	if err := deployERC20Flags.chainFlags.CheckMutuallyExclusiveFields(); err != nil {
		return err
	}
	if !deployERC20Flags.chainFlags.Defined() {
		prompt := "Where do you want to Deploy the ERC-20 Token?"
		if cancel, err := contract.PromptChain(
			app.GetSDKApp(),
			network,
			prompt,
			"",
			&deployERC20Flags.chainFlags,
		); cancel || err != nil {
			return err
		}
	}
	if deployERC20Flags.rpcEndpoint == "" {
		deployERC20Flags.rpcEndpoint, _, err = contract.GetBlockchainEndpoints(
			app.GetSDKApp(),
			network,
			deployERC20Flags.chainFlags,
			true,
			false,
		)
		if err != nil {
			return err
		}
		ux.Logger.PrintToUser("%s", luxlog.Yellow.Wrap("RPC Endpoint: %s"), deployERC20Flags.rpcEndpoint)
	}
	_, genesisPrivateKey, err := contract.GetEVMSubnetPrefundedKey(
		app.GetSDKApp(),
		network,
		deployERC20Flags.chainFlags,
	)
	if err != nil {
		return err
	}
	privateKey, err := deployERC20Flags.PrivateKeyFlags.GetPrivateKey(app.GetSDKApp(), genesisPrivateKey)
	if err != nil {
		return err
	}
	if privateKey == "" {
		ux.Logger.PrintToUser("A private key is needed to pay for the contract deploy fees.")
		ux.Logger.PrintToUser("It will also be considered the owner address of the contract, beign able to call")
		ux.Logger.PrintToUser("the contract methods only available to owners.")
		privateKey, err = prompts.PromptPrivateKey(
			app.Prompt,
			"deploy the contract",
		)
		if err != nil {
			return err
		}
	}
	if deployERC20Flags.symbol == "" {
		ux.Logger.PrintToUser("Which is the token symbol?")
		deployERC20Flags.symbol, err = app.Prompt.CaptureString("Token symbol")
		if err != nil {
			return err
		}
	}
	supply := new(big.Int).SetUint64(deployERC20Flags.supply)
	if deployERC20Flags.supply == 0 {
		ux.Logger.PrintToUser("Which is the total token supply?")
		supply, err = app.Prompt.CapturePositiveBigInt("Token supply")
		if err != nil {
			return err
		}
	}
	if deployERC20Flags.funded == "" {
		ux.Logger.PrintToUser("Which address should receive the supply?")
		deployERC20Flags.funded, err = prompts.PromptAddress(
			app.Prompt,
			"receive the total token supply",
		)
		if err != nil {
			return err
		}
	}
	address, err := contract.DeployERC20(
		deployERC20Flags.rpcEndpoint,
		privateKey,
		deployERC20Flags.symbol,
		crypto.Address(common.HexToAddress(deployERC20Flags.funded).Bytes()),
		supply,
	)
	if err != nil {
		return err
	}
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Token Address: %s", address.Hex())
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("ERC20 Contract Successfully Deployed!")
	return nil
}
