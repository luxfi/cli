// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package contractcmd

import (
	"math/big"

	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/networkoptions"
	cliprompts "github.com/luxfi/cli/pkg/prompts"
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
		Long: `Deploy an ERC20 token into a given Network and Blockchain.

The command deploys a standard ERC20 token contract with the specified
symbol, initial supply, and recipient address for the minted tokens.

Examples:
  # Interactive mode (prompts for missing values)
  lux contract deploy erc20

  # Non-interactive mode (all flags required)
  lux contract deploy erc20 --symbol USDC --supply 1000000 \
    --funded 0x1234...abcd --private-key-file ./key.txt \
    --c-chain --mainnet

  # Deploy to a specific blockchain
  lux contract deploy erc20 --symbol LUX --supply 100000000 \
    --funded 0xYourAddress --blockchain-id <ID> --testnet`,
		RunE: deployERC20,
		Args: cobrautils.ExactArgs(0),
	}
	// Network flags handled globally to avoid conflicts
	deployERC20Flags.PrivateKeyFlags.AddToCmd(cmd, "as contract deployer")
	// enabling blockchain names, C-Chain and blockchain IDs
	deployERC20Flags.chainFlags.SetEnabled(true, true, false, false, true)
	deployERC20Flags.chainFlags.AddToCmd(cmd, "deploy the ERC20 contract into %s")
	cmd.Flags().StringVar(&deployERC20Flags.symbol, "symbol", "", "token symbol (e.g., USDC, LUX)")
	cmd.Flags().Uint64Var(&deployERC20Flags.supply, "supply", 0, "total token supply to mint")
	cmd.Flags().StringVar(&deployERC20Flags.funded, "funded", "", "address to receive the initial token supply (0x...)")
	cmd.Flags().StringVar(&deployERC20Flags.rpcEndpoint, "rpc", "", "RPC endpoint URL (auto-detected if not specified)")
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
		ux.Logger.PrintToUser(luxlog.Yellow.Wrap("RPC Endpoint: %s"), deployERC20Flags.rpcEndpoint)
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
	// Collect all missing required options
	var missing []cliprompts.MissingOpt
	if deployERC20Flags.symbol == "" {
		missing = append(missing, cliprompts.MissingOpt{
			Flag:   "--symbol",
			Prompt: "Token symbol",
			Note:   "e.g., USDC, LUX",
		})
	}
	if deployERC20Flags.supply == 0 {
		missing = append(missing, cliprompts.MissingOpt{
			Flag:   "--supply",
			Prompt: "Token supply",
			Note:   "total tokens to mint",
		})
	}
	if deployERC20Flags.funded == "" {
		missing = append(missing, cliprompts.MissingOpt{
			Flag:   "--funded",
			Prompt: "Funded address",
			Note:   "address to receive initial supply (0x...)",
		})
	}

	// In non-interactive mode, fail with all missing options listed
	if len(missing) > 0 && !cliprompts.IsInteractive() {
		return cliprompts.MissingError("lux contract deploy erc20", missing)
	}

	// Interactive mode: prompt for missing values
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
