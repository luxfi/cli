// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package keycmd

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/contract"
	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/cli/pkg/networkoptions"
	"github.com/luxfi/sdk/prompts"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/cli/pkg/vm"
	"github.com/luxfi/cli/pkg/warp"
	"github.com/luxfi/sdk/evm"
	eth_crypto "github.com/luxfi/crypto"
	goethereumcommon "github.com/luxfi/geth/common"
	"github.com/luxfi/ids"
	luxdconstants "github.com/luxfi/node/utils/constants"
	cryptokeychain "github.com/luxfi/node/utils/crypto/keychain"
	ledger "github.com/luxfi/node/utils/crypto/ledger"
	walletkeychain "github.com/luxfi/node/wallet/keychain"
	"github.com/luxfi/node/utils/formatting/address"
	"github.com/luxfi/node/utils/units"
	"github.com/luxfi/node/vms/components/lux"
	"github.com/luxfi/node/vms/platformvm/txs"
	"github.com/luxfi/node/vms/secp256k1fx"
	xvmtxs "github.com/luxfi/node/vms/xvm/txs"
	"github.com/luxfi/node/wallet/chain/c"
	"github.com/luxfi/node/wallet/chain/p/builder"
	"github.com/luxfi/node/wallet/net/primary"
	"github.com/luxfi/node/wallet/net/primary/common"
	"github.com/spf13/cobra"
)

const (
	keyNameFlag         = "key"
	ledgerIndexFlag     = "ledger"
	amountFlag          = "amount"
	destinationAddrFlag = "destination-addr"
	wrongLedgerIndexVal = 32768
)

var (
	keyName            string
	ledgerIndex        uint32
	destinationAddrStr string
	amountFlt          float64
	// token transferrer experimental
	originSubnet                  string
	destinationSubnet             string
	originTransferrerAddress      string
	destinationTransferrerAddress string
	destinationKeyName            string
	//
	senderChainFlags   contract.ChainSpec
	receiverChainFlags contract.ChainSpec
)

func newTransferCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer [options]",
		Short: "Fund a ledger address or stored key from another one",
		Long:  `The key transfer command allows to transfer funds between stored keys or ledger addresses.`,
		RunE:  transferF,
		Args:  cobrautils.ExactArgs(0),
	}
	networkoptions.AddNetworkFlagsToCmd(cmd, &globalNetworkFlags, false, networkoptions.DefaultSupportedNetworkOptions)
	cmd.Flags().StringVarP(
		&keyName,
		keyNameFlag,
		"k",
		"",
		"key associated to the sender or receiver address",
	)
	cmd.Flags().Uint32VarP(
		&ledgerIndex,
		ledgerIndexFlag,
		"i",
		wrongLedgerIndexVal,
		"ledger index associated to the sender or receiver address",
	)
	cmd.Flags().StringVarP(
		&destinationAddrStr,
		destinationAddrFlag,
		"a",
		"",
		"destination address",
	)
	cmd.Flags().StringVar(
		&destinationKeyName,
		"destination-key",
		"",
		"key associated to a destination address",
	)
	cmd.Flags().Float64VarP(
		&amountFlt,
		amountFlag,
		"o",
		0,
		"amount to send or receive (LUX or TOKEN units)",
	)
	cmd.Flags().StringVar(
		&originSubnet,
		"origin-subnet",
		"",
		"subnet where the funds belong (token transferrer experimental)",
	)
	cmd.Flags().StringVar(
		&destinationSubnet,
		"destination-subnet",
		"",
		"subnet where the funds will be sent (token transferrer experimental)",
	)
	cmd.Flags().StringVar(
		&originTransferrerAddress,
		"origin-transferrer-address",
		"",
		"token transferrer address at the origin subnet (token transferrer experimental)",
	)
	cmd.Flags().StringVar(
		&destinationTransferrerAddress,
		"destination-transferrer-address",
		"",
		"token transferrer address at the destination subnet (token transferrer experimental)",
	)
	senderChainFlags.SetFlagNames(
		"sender-blockchain",
		"c-chain-sender",
		"p-chain-sender",
		"x-chain-sender",
		"sender-blockchain-id",
	)
	senderChainFlags.AddToCmd(cmd, "send from %s")
	receiverChainFlags.SetFlagNames(
		"receiver-blockchain",
		"c-chain-receiver",
		"p-chain-receiver",
		"x-chain-receiver",
		"receiver-blockchain-id",
	)
	receiverChainFlags.AddToCmd(cmd, "receive at %s")
	return cmd
}

func transferF(*cobra.Command, []string) error {
	if keyName != "" && ledgerIndex != wrongLedgerIndexVal {
		return fmt.Errorf("only one between a keyname or a ledger index must be given")
	}

	network, err := networkoptions.GetNetworkFromCmdLineFlags(
		app,
		"On what Network do you want to execute the transfer?",
		globalNetworkFlags,
		true,
		false,
		networkoptions.DefaultSupportedNetworkOptions,
		"",
	)
	if err != nil {
		return err
	}

	if !senderChainFlags.Defined() {
		prompt := "Where are the funds to transfer?"
		if cancel, err := contract.PromptChain(
			app,
			network,
			prompt,
			"",
			&senderChainFlags,
		); err != nil {
			return err
		} else if cancel {
			return nil
		}
	}

	if !receiverChainFlags.Defined() {
		prompt := "Where are the funds going to?"
		if cancel, err := contract.PromptChain(
			app,
			network,
			prompt,
			"",
			&receiverChainFlags,
		); err != nil {
			return err
		} else if cancel {
			return nil
		}
	}

	if (senderChainFlags.CChain && receiverChainFlags.CChain) ||
		(senderChainFlags.BlockchainName != "" && senderChainFlags.BlockchainName == receiverChainFlags.BlockchainName) {
		return intraEvmSend(network, senderChainFlags)
	}

	if !senderChainFlags.PChain && !senderChainFlags.XChain && !receiverChainFlags.PChain && !receiverChainFlags.XChain {
		return interEvmSend(network, senderChainFlags, receiverChainFlags)
	}

	senderDesc, err := contract.GetBlockchainDesc(senderChainFlags)
	if err != nil {
		return err
	}
	receiverDesc, err := contract.GetBlockchainDesc(receiverChainFlags)
	if err != nil {
		return err
	}
	if senderChainFlags.BlockchainName != "" || receiverChainFlags.BlockchainName != "" || senderChainFlags.XChain {
		return fmt.Errorf("transfer from %s to %s is not supported", senderDesc, receiverDesc)
	}

	if keyName == "" && ledgerIndex == wrongLedgerIndexVal {
		var useLedger bool
		goalStr := "as the sender address"
		if receiverChainFlags.XChain {
			ux.Logger.PrintToUser("P->X transfer is an intra-account operation.")
			ux.Logger.PrintToUser("Tokens will be transferred to the same account address on the other chain")
			goalStr = "specify the sender/receiver address"
		}
		if senderChainFlags.CChain && receiverChainFlags.PChain {
			ux.Logger.PrintToUser("C->P transfer is an intra-account operation.")
			ux.Logger.PrintToUser("Tokens will be transferred to the same account address on the other chain")
			goalStr = "as the sender/receiver address"
		}
		useLedger, keyName, err = prompts.GetKeyOrLedger(app.Prompt, goalStr, app.GetKeyDir(), true)
		if err != nil {
			return err
		}
		if useLedger {
			ledgerIndexStr, err := app.Prompt.CaptureString("Ledger index to use")
			if err != nil {
				return err
			}
			ledgerIndexUint64, err := strconv.ParseUint(ledgerIndexStr, 10, 32)
			if err != nil {
				return fmt.Errorf("invalid ledger index: %w", err)
			}
			ledgerIndex = uint32(ledgerIndexUint64)
		}
	}

	var kc walletkeychain.Keychain
	var sk *key.SoftKey
	if keyName != "" {
		keyPath := app.GetKeyPath(keyName)
		sk, err = key.LoadSoft(network.ID(), keyPath)
		if err != nil {
			return err
		}
		kc = sk.KeyChain()
	} else {
		ledgerDevice, err := ledger.NewLedger()
		if err != nil {
			return err
		}
		ledgerIndices := []uint32{ledgerIndex}
		kc, err = NewLedgerKeychain(ledgerDevice, ledgerIndices)
		if err != nil {
			return err
		}
	}
	usingLedger := ledgerIndex != wrongLedgerIndexVal

	if amountFlt == 0 {
		amountFlt, err = captureAmount("LUX units")
		if err != nil {
			return err
		}
	}
	amount := uint64(amountFlt * float64(units.Lux))

	if destinationAddrStr == "" && senderChainFlags.PChain && (receiverChainFlags.PChain || receiverChainFlags.CChain) {
		if destinationKeyName != "" {
			keyPath := app.GetKeyPath(destinationKeyName)
			networkID, err := network.NetworkID()
			if err != nil {
				return err
			}
			k, err := key.LoadSoft(networkID, keyPath)
			if err != nil {
				return err
			}
			if receiverChainFlags.CChain {
				destinationAddrStr = k.C()
			}
			if receiverChainFlags.PChain {
				addrs := k.P()
				if len(addrs) == 0 {
					return fmt.Errorf("unexpected null number of P-Chain addresses for key")
				}
				destinationAddrStr = addrs[0]
			}
		} else {
			// format could be used for validation in the future
			// format := prompts.EVMFormat
			// if receiverChainFlags.PChain {
			// 	format = prompts.PChainFormat
			// }
			destinationAddrStr, err = prompts.PromptAddress(
				app.Prompt,
				"destination address",
			)
			if err != nil {
				return err
			}
		}
	}

	if senderChainFlags.PChain && receiverChainFlags.PChain {
		return pToPSend(
			network,
			kc,
			usingLedger,
			destinationAddrStr,
			amount,
		)
	}

	if senderChainFlags.PChain && receiverChainFlags.CChain {
		return pToCSend(
			network,
			kc,
			usingLedger,
			destinationAddrStr,
			amount,
		)
	}
	if senderChainFlags.CChain && receiverChainFlags.PChain {
		return cToPSend(
			network,
			kc,
			sk,
			usingLedger,
			amount,
		)
	}
	if senderChainFlags.PChain && receiverChainFlags.XChain {
		return pToXSend(
			network,
			kc,
			usingLedger,
			amount,
		)
	}

	return nil
}

func captureAmount(tokenDesc string) (float64, error) {
	promptStr := fmt.Sprintf("Amount to send (%s)", tokenDesc)
	amountFlt, err := app.Prompt.CaptureFloat(promptStr)
	if err != nil {
		return 0, err
	}
	if amountFlt <= 0 {
		return 0, fmt.Errorf("value %f must be greater than zero", amountFlt)
	}
	return amountFlt, nil
}

func intraEvmSend(
	network models.Network,
	senderChain contract.ChainSpec,
) error {
	var (
		err        error
		privateKey string
	)
	if keyName != "" {
		keyPath := app.GetKeyPath(keyName)
		k, err := key.LoadSoft(network.ID(), keyPath)
		if err != nil {
			return err
		}
		privateKey = k.PrivateKeyRaw()
	} else {
		privateKey, err = app.Prompt.CaptureString("sender private key")
		if err != nil {
			return err
		}
	}
	if destinationKeyName != "" {
		keyPath := app.GetKeyPath(destinationKeyName)
		k, err := key.LoadSoft(network.ID(), keyPath)
		if err != nil {
			return err
		}
		destinationAddrStr = k.C()
	}
	if destinationAddrStr == "" {
		destinationAddrStr, err = prompts.PromptAddress(
			app.Prompt,
			"destination address",
		)
		if err != nil {
			return err
		}
	}
	if amountFlt == 0 {
		amountFlt, err = app.Prompt.CaptureFloat("Amount to transfer")
		if err != nil {
			return err
		}
	} else if amountFlt < 0 {
		return fmt.Errorf("amount must be positive")
	}
	amountBigFlt := new(big.Float).SetFloat64(amountFlt)
	amountBigFlt = amountBigFlt.Mul(amountBigFlt, new(big.Float).SetInt(vm.OneLux))
	amount, _ := amountBigFlt.Int(nil)
	senderURL, _, err := contract.GetBlockchainEndpoints(
		app,
		network,
		senderChain,
		true,
		false,
	)
	if err != nil {
		return err
	}
	client, err := evm.GetClient(senderURL)
	if err != nil {
		return err
	}

	receipt, err := client.FundAddress(privateKey, destinationAddrStr, amount)
	if err != nil {
		return err
	}
	chainName, err := contract.GetBlockchainDesc(senderChain)
	if err != nil {
		return err
	}
	ux.Logger.PrintToUser("%s Paid fee: %.9f LUX",
		chainName,
		evm.CalculateEvmFeeInLux(receipt.GasUsed, receipt.EffectiveGasPrice))
	return err
}

func interEvmSend(
	network models.Network,
	senderChain contract.ChainSpec,
	receiverChain contract.ChainSpec,
) error {
	senderURL, _, err := contract.GetBlockchainEndpoints(
		app,
		network,
		senderChain,
		true,
		false,
	)
	if err != nil {
		return err
	}
	receiverBlockchainID, err := contract.GetBlockchainID(
		app,
		network,
		receiverChain,
	)
	if err != nil {
		return err
	}
	senderDesc, err := contract.GetBlockchainDesc(senderChainFlags)
	if err != nil {
		return err
	}
	receiverDesc, err := contract.GetBlockchainDesc(receiverChainFlags)
	if err != nil {
		return err
	}
	if originTransferrerAddress == "" {
		addr, err := app.Prompt.CaptureAddress(
			fmt.Sprintf("Enter the address of the Token Transferrer on %s", senderDesc),
		)
		if err != nil {
			return err
		}
		originTransferrerAddress = addr.Hex()
	} else {
		if err := prompts.ValidateAddress(originTransferrerAddress); err != nil {
			return err
		}
	}
	if destinationTransferrerAddress == "" {
		addr, err := app.Prompt.CaptureAddress(
			fmt.Sprintf("Enter the address of the Token Transferrer on %s", receiverDesc),
		)
		if err != nil {
			return err
		}
		destinationTransferrerAddress = addr.Hex()
	} else {
		if err := prompts.ValidateAddress(destinationTransferrerAddress); err != nil {
			return err
		}
	}
	if keyName == "" {
		keyName, err = app.Prompt.CaptureString("Enter the key name to fund the transfer")
		if err != nil {
			return err
		}
	}
	keyPath := app.GetKeyPath(keyName)
	originK, err := key.LoadSoft(network.ID(), keyPath)
	if err != nil {
		return err
	}
	privateKey := originK.PrivateKeyRaw()
	var destinationAddr goethereumcommon.Address
	if destinationAddrStr == "" && destinationKeyName == "" {
		option, err := app.Prompt.CaptureList(
			"Do you want to choose a stored key for the destination, or input a destination address?",
			[]string{"Key", "Address"},
		)
		if err != nil {
			return err
		}
		switch option {
		case "Key":
			destinationKeyName, err = app.Prompt.CaptureString("Enter the key name to receive the transfer")
			if err != nil {
				return err
			}
		case "Address":
			addr, err := app.Prompt.CaptureAddress(
				"Enter the destination address",
			)
			if err != nil {
				return err
			}
			destinationAddrStr = addr.Hex()
		}
	}
	switch {
	case destinationAddrStr != "":
		if err := prompts.ValidateAddress(destinationAddrStr); err != nil {
			return err
		}
		destinationAddr = goethereumcommon.HexToAddress(destinationAddrStr)
	case destinationKeyName != "":
		destKeyPath := app.GetKeyPath(destinationKeyName)
		destinationK, err := key.LoadSoft(network.ID(), destKeyPath)
		if err != nil {
			return err
		}
		destinationAddrStr = destinationK.C()
		destinationAddr = goethereumcommon.HexToAddress(destinationAddrStr)
	default:
		return fmt.Errorf("you should set the destination address or destination key")
	}
	if amountFlt == 0 {
		amountFlt, err = captureAmount("TOKEN units")
		if err != nil {
			return err
		}
	}
	amount := new(big.Float).SetFloat64(amountFlt)
	amount = amount.Mul(amount, new(big.Float).SetFloat64(float64(units.Lux)))
	amount = amount.Mul(amount, new(big.Float).SetFloat64(float64(units.Lux)))
	amountInt, _ := amount.Int(nil)
	// Import crypto for Address type
	originAddr := goethereumcommon.HexToAddress(originTransferrerAddress)
	destTransferrerAddr := goethereumcommon.HexToAddress(destinationTransferrerAddress)
	
	// Convert to crypto.Address by converting to hex and back
	cryptoOriginAddr := eth_crypto.HexToAddress(originAddr.Hex())
	cryptoDestTransferrerAddr := eth_crypto.HexToAddress(destTransferrerAddr.Hex())
	cryptoDestAddr := eth_crypto.HexToAddress(destinationAddr.Hex())
	
	receipt, receipt2, err := warp.Send(
		senderURL,
		cryptoOriginAddr,
		privateKey,
		receiverBlockchainID,
		cryptoDestTransferrerAddr,
		cryptoDestAddr,
		amountInt,
	)
	if err != nil {
		return err
	}

	chainName, err := contract.GetBlockchainDesc(senderChain)
	if err != nil {
		return err
	}
	ux.Logger.PrintToUser("%s Paid fee: %.9f LUX",
		chainName,
		evm.CalculateEvmFeeInLux(receipt.GasUsed, receipt.EffectiveGasPrice))

	if receipt2 != nil {
		chainName, err := contract.GetBlockchainDesc(receiverChain)
		if err != nil {
			return err
		}
		ux.Logger.PrintToUser("%s Paid fee: %.9f LUX",
			chainName,
			evm.CalculateEvmFeeInLux(receipt2.GasUsed, receipt2.EffectiveGasPrice))
	}

	return nil
}

func pToPSend(
	network models.Network,
	kc walletkeychain.Keychain,
	usingLedger bool,
	destinationAddrStr string,
	amount uint64,
) error {
	ethKeychain := secp256k1fx.NewKeychain()
	walletConfig := &primary.WalletConfig{
		URI:          network.Endpoint(),
		LUXKeychain:  kc,
		EthKeychain:  ethKeychain,
	}
	wallet, err := primary.MakeWallet(
		context.Background(),
		walletConfig,
	)
	if err != nil {
		return err
	}
	destinationAddr, err := address.ParseToID(destinationAddrStr)
	if err != nil {
		return err
	}
	to := secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs:     []ids.ShortID{destinationAddr},
	}
	output := &lux.TransferableOutput{
		Asset: lux.Asset{ID: getBuilderContext(wallet).LUXAssetID},
		Out: &secp256k1fx.TransferOutput{
			Amt:          amount,
			OutputOwners: to,
		},
	}
	outputs := []*lux.TransferableOutput{output}
	ux.Logger.PrintToUser("Issuing BaseTx P -> P")
	if usingLedger {
		ux.Logger.PrintToUser("*** Please sign 'Export Tx / P to X Chain' transaction on the ledger device *** ")
	}
	unsignedTx, err := wallet.P().Builder().NewBaseTx(
		outputs,
	)
	if err != nil {
		return fmt.Errorf("error building tx: %w", err)
	}
	tx := txs.Tx{Unsigned: unsignedTx}
	if err := wallet.P().Signer().Sign(context.Background(), &tx); err != nil {
		return fmt.Errorf("error signing tx: %w", err)
	}
	ctx, cancel := utils.GetAPIContext()
	defer cancel()
	err = wallet.P().IssueTx(
		&tx,
		common.WithContext(ctx),
	)
	if err != nil {
		if ctx.Err() != nil {
			err = fmt.Errorf("timeout issuing/verifying tx with ID %s: %w", tx.ID, err)
		} else {
			err = fmt.Errorf("error issuing tx with ID %s: %w", tx.ID, err)
		}
		return err
	}
	// Calculate fee - use default for now
	// TODO: Use proper fee calculation when API is available
	txFee := uint64(1000000) // Default 0.001 LUX
	ux.Logger.PrintToUser("P-Chain Paid fee: %.9f LUX", float64(txFee)/float64(units.Lux))
	ux.Logger.PrintToUser("Transaction successful")
	return nil
}

func pToXSend(
	network models.Network,
	kc walletkeychain.Keychain,
	usingLedger bool,
	amount uint64,
) error {
	ethKeychain := secp256k1fx.NewKeychain()
	walletConfig := &primary.WalletConfig{
		URI:          network.Endpoint(),
		LUXKeychain:  kc,
		EthKeychain:  ethKeychain,
	}
	wallet, err := primary.MakeWallet(
		context.Background(),
		walletConfig,
	)
	if err != nil {
		return err
	}
	to := secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs:     kc.Addresses(),
	}
	if err := exportFromP(
		amount,
		wallet,
		wallet.X().Builder().Context().BlockchainID,
		"X",
		to,
		usingLedger,
	); err != nil {
		return err
	}
	time.Sleep(5 * time.Second)
	return importIntoX(
		wallet,
		luxdconstants.PlatformChainID,
		"P",
		to,
		usingLedger,
	)
}

func exportFromP(
	amount uint64,
	wallet primary.Wallet,
	blockchainID ids.ID,
	blockchainAlias string,
	to secp256k1fx.OutputOwners,
	usingLedger bool,
) error {
	output := &lux.TransferableOutput{
		Asset: lux.Asset{ID: getBuilderContext(wallet).LUXAssetID},
		Out: &secp256k1fx.TransferOutput{
			Amt:          amount,
			OutputOwners: to,
		},
	}
	outputs := []*lux.TransferableOutput{output}
	ux.Logger.PrintToUser("Issuing ExportTx P -> %s", blockchainAlias)
	if usingLedger {
		ux.Logger.PrintToUser("*** Please sign 'Export Tx / P to %s Chain' transaction on the ledger device *** ", blockchainAlias)
	}
	unsignedTx, err := wallet.P().Builder().NewExportTx(
		blockchainID,
		outputs,
	)
	if err != nil {
		return fmt.Errorf("error building tx: %w", err)
	}
	tx := txs.Tx{Unsigned: unsignedTx}
	if err := wallet.P().Signer().Sign(context.Background(), &tx); err != nil {
		return fmt.Errorf("error signing tx: %w", err)
	}
	ctx, cancel := utils.GetAPIContext()
	defer cancel()
	err = wallet.P().IssueTx(
		&tx,
		common.WithContext(ctx),
	)
	if err != nil {
		if ctx.Err() != nil {
			err = fmt.Errorf("timeout issuing/verifying tx with ID %s: %w", tx.ID, err)
		} else {
			err = fmt.Errorf("error issuing tx with ID %s: %w", tx.ID, err)
		}
		return err
	}
	// Calculate fee - use default for now
	// TODO: Use proper fee calculation when API is available
	txFee := uint64(1000000) // Default 0.001 LUX
	ux.Logger.PrintToUser("P-Chain Paid fee: %.9f LUX", float64(txFee)/float64(units.Lux))
	ux.Logger.PrintToUser("Transaction successful")
	return nil
}

func importIntoX(
	wallet primary.Wallet,
	blockchainID ids.ID,
	blockchainAlias string,
	to secp256k1fx.OutputOwners,
	usingLedger bool,
) error {
	ux.Logger.PrintToUser("Issuing ImportTx %s -> X", blockchainAlias)
	if usingLedger {
		ux.Logger.PrintToUser("*** Please sign ImportTx transaction on the ledger device *** ")
	}
	unsignedTx, err := wallet.X().Builder().NewImportTx(
		blockchainID,
		&to,
	)
	if err != nil {
		return fmt.Errorf("error building tx: %w", err)
	}
	tx := xvmtxs.Tx{Unsigned: unsignedTx}
	if err := wallet.X().Signer().Sign(context.Background(), &tx); err != nil {
		return fmt.Errorf("error signing tx: %w", err)
	}
	ctx, cancel := utils.GetAPIContext()
	defer cancel()
	err = wallet.X().IssueTx(
		&tx,
		common.WithContext(ctx),
	)
	if err != nil {
		if ctx.Err() != nil {
			err = fmt.Errorf("timeout issuing/verifying tx with ID %s: %w", tx.ID, err)
		} else {
			err = fmt.Errorf("error issuing tx with ID %s: %w", tx.ID, err)
		}
		return err
	}
	ux.Logger.PrintToUser("X-Chain Paid fee: %.9f LUX", float64(wallet.X().Builder().Context().BaseTxFee)/float64(units.Lux))
	return nil
}

func pToCSend(
	network models.Network,
	kc walletkeychain.Keychain,
	usingLedger bool,
	destinationAddrStr string,
	amount uint64,
) error {
	ethKeychain := secp256k1fx.NewKeychain()
	walletConfig := &primary.WalletConfig{
		URI:          network.Endpoint(),
		LUXKeychain:  kc,
		EthKeychain:  ethKeychain,
	}
	wallet, err := primary.MakeWallet(
		context.Background(),
		walletConfig,
	)
	if err != nil {
		return err
	}
	to := secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs:     kc.Addresses(),
	}
	if err := exportFromP(
		amount,
		wallet,
		wallet.C().Builder().Context().BlockchainID,
		"C",
		to,
		usingLedger,
	); err != nil {
		return err
	}
	time.Sleep(5 * time.Second)
	if err != nil {
		return err
	}
	return importIntoC(
		network,
		wallet,
		luxdconstants.PlatformChainID,
		"P",
		destinationAddrStr,
		usingLedger,
	)
}

func importIntoC(
	network models.Network,
	wallet primary.Wallet,
	blockchainID ids.ID,
	blockchainAlias string,
	destinationAddrStr string,
	usingLedger bool,
) error {
	ux.Logger.PrintToUser("Issuing ImportTx %s -> C", blockchainAlias)
	if usingLedger {
		ux.Logger.PrintToUser("*** Please sign ImportTx transaction on the ledger device *** ")
	}
	amt, err := wallet.C().Builder().GetImportableBalance(blockchainID)
	if err != nil {
		return fmt.Errorf("error getting importable balance: %w", err)
	}
	// Construct C-Chain endpoint
	cChainEndpoint := network.Endpoint() + "/ext/bc/C/rpc"
	client, err := evm.GetClient(cChainEndpoint)
	if err != nil {
		return err
	}
	baseFee, err := client.EstimateBaseFee()
	if err != nil {
		return err
	}
	unsignedTx, err := wallet.C().Builder().NewImportTx(
		blockchainID,
		goethereumcommon.HexToAddress(destinationAddrStr),
		baseFee,
	)
	if err != nil {
		return fmt.Errorf("error building tx: %w", err)
	}
	tx := c.Tx{UnsignedAtomicTx: unsignedTx}
	if err := wallet.C().Signer().SignAtomic(context.Background(), &tx); err != nil {
		return fmt.Errorf("error signing tx: %w", err)
	}
	ctx, cancel := utils.GetAPIContext()
	defer cancel()
	err = wallet.C().IssueAtomicTx(
		&tx,
		common.WithContext(ctx),
	)
	if err != nil {
		if ctx.Err() != nil {
			err = fmt.Errorf("timeout issuing/verifying tx with ID %s: %w", tx.ID, err)
		} else {
			err = fmt.Errorf("error issuing tx with ID %s: %w", tx.ID, err)
		}
		return err
	}

	if len(unsignedTx.Outs) == 0 {
		return fmt.Errorf("no outputs for C-Chain transaction")
	}
	ux.Logger.PrintToUser("C-Chain Paid fee: %.9f LUX", float64(amt-unsignedTx.Outs[0].Amount)/float64(units.Lux))
	return nil
}

func cToPSend(
	network models.Network,
	kc walletkeychain.Keychain,
	sk *key.SoftKey,
	usingLedger bool,
	amount uint64,
) error {
	ethKeychain := sk.KeyChain()
	walletConfig := &primary.WalletConfig{
		URI:          network.Endpoint(),
		LUXKeychain:  kc,
		EthKeychain:  ethKeychain,
	}
	wallet, err := primary.MakeWallet(
		context.Background(),
		walletConfig,
	)
	if err != nil {
		return err
	}
	to := secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs:     kc.Addresses(),
	}
	if err := exportFromC(
		network,
		amount,
		wallet,
		luxdconstants.PlatformChainID,
		"P",
		to,
		usingLedger,
	); err != nil {
		return err
	}
	time.Sleep(5 * time.Second)
	wallet, err = primary.MakeWallet(
		context.Background(),
		walletConfig,
	)
	if err != nil {
		return err
	}
	return importIntoP(
		wallet,
		wallet.C().Builder().Context().BlockchainID,
		"C",
		to,
		usingLedger,
	)
}

func exportFromC(
	network models.Network,
	amount uint64,
	wallet primary.Wallet,
	blockchainID ids.ID,
	blockchainAlias string,
	to secp256k1fx.OutputOwners,
	usingLedger bool,
) error {
	ux.Logger.PrintToUser("Issuing ExportTx C -> %s", blockchainAlias)
	if usingLedger {
		ux.Logger.PrintToUser("*** Please sign ExportTx transaction on the ledger device *** ")
	}
	// Construct C-Chain endpoint
	cChainEndpoint := network.Endpoint() + "/ext/bc/C/rpc"
	client, err := evm.GetClient(cChainEndpoint)
	if err != nil {
		return err
	}
	baseFee, err := client.EstimateBaseFee()
	if err != nil {
		return err
	}
	outputs := []*secp256k1fx.TransferOutput{
		{
			Amt:          amount,
			OutputOwners: to,
		},
	}
	unsignedTx, err := wallet.C().Builder().NewExportTx(
		blockchainID,
		outputs,
		baseFee,
	)
	if err != nil {
		return fmt.Errorf("error building tx: %w", err)
	}
	tx := c.Tx{UnsignedAtomicTx: unsignedTx}
	if err := wallet.C().Signer().SignAtomic(context.Background(), &tx); err != nil {
		return fmt.Errorf("error signing tx: %w", err)
	}
	ctx, cancel := utils.GetAPIContext()
	defer cancel()
	err = wallet.C().IssueAtomicTx(
		&tx,
		common.WithContext(ctx),
	)
	if err != nil {
		if ctx.Err() != nil {
			err = fmt.Errorf("timeout issuing/verifying tx with ID %s: %w", tx.ID, err)
		} else {
			err = fmt.Errorf("error issuing tx with ID %s: %w", tx.ID, err)
		}
		return err
	}
	if len(unsignedTx.Ins) == 0 {
		return fmt.Errorf("no inputs for C-Chain transaction")
	}
	ux.Logger.PrintToUser("C-Chain Paid fee: %.9f LUX", float64(unsignedTx.Ins[0].Amount-amount)/float64(units.Lux))

	return nil
}

func importIntoP(
	wallet primary.Wallet,
	blockchainID ids.ID,
	blockchainAlias string,
	to secp256k1fx.OutputOwners,
	usingLedger bool,
) error {
	ux.Logger.PrintToUser("Issuing ImportTx %s -> P", blockchainAlias)
	if usingLedger {
		ux.Logger.PrintToUser("*** Please sign ImportTx transaction on the ledger device *** ")
	}
	unsignedTx, err := wallet.P().Builder().NewImportTx(
		blockchainID,
		&to,
	)
	if err != nil {
		return fmt.Errorf("error building tx: %w", err)
	}
	tx := txs.Tx{Unsigned: unsignedTx}
	if err := wallet.P().Signer().Sign(context.Background(), &tx); err != nil {
		return fmt.Errorf("error signing tx: %w", err)
	}
	ctx, cancel := utils.GetAPIContext()
	defer cancel()
	err = wallet.P().IssueTx(
		&tx,
		common.WithContext(ctx),
	)
	if err != nil {
		if ctx.Err() != nil {
			err = fmt.Errorf("timeout issuing/verifying tx with ID %s: %w", tx.ID, err)
		} else {
			err = fmt.Errorf("error issuing tx with ID %s: %w", tx.ID, err)
		}
		return err
	}
	// Calculate fee - use default for now
	// TODO: Use proper fee calculation when API is available
	txFee := uint64(1000000) // Default 0.001 LUX
	ux.Logger.PrintToUser("P-Chain Paid fee: %.9f LUX", float64(txFee)/float64(units.Lux))
	ux.Logger.PrintToUser("Transaction successful")

	return nil
}

func getBuilderContext(wallet primary.Wallet) *builder.Context {
	if wallet == nil {
		return nil
	}
	return wallet.P().Builder().Context()
}

// ledgerKeychain wraps a ledger device to implement wallet keychain interface
type ledgerKeychain struct {
	ledger    cryptokeychain.Ledger
	indices   []uint32
	addresses []ids.ShortID
}

// NewLedgerKeychain creates a new ledger keychain
func NewLedgerKeychain(ledgerDevice cryptokeychain.Ledger, indices []uint32) (walletkeychain.Keychain, error) {
	addresses, err := ledgerDevice.GetAddresses(indices)
	if err != nil {
		return nil, err
	}
	return &ledgerKeychain{
		ledger:    ledgerDevice,
		indices:   indices,
		addresses: addresses,
	}, nil
}

// Addresses returns the list of addresses
func (lk *ledgerKeychain) Addresses() []ids.ShortID {
	return lk.addresses
}

// Get returns a signer for the given address
func (lk *ledgerKeychain) Get(addr ids.ShortID) (walletkeychain.Signer, bool) {
	for i, a := range lk.addresses {
		if a == addr {
			return &ledgerSigner{
				ledger: lk.ledger,
				index:  lk.indices[i],
				addr:   addr,
			}, true
		}
	}
	return nil, false
}

// ledgerSigner implements the Signer interface for ledger
type ledgerSigner struct {
	ledger cryptokeychain.Ledger
	index  uint32
	addr   ids.ShortID
}

// SignHash signs a hash
func (ls *ledgerSigner) SignHash(hash []byte) ([]byte, error) {
	return ls.ledger.SignHash(hash, ls.index)
}

// Sign signs data
func (ls *ledgerSigner) Sign(data []byte) ([]byte, error) {
	return ls.ledger.Sign(data, ls.index)
}

// Address returns the address
func (ls *ledgerSigner) Address() ids.ShortID {
	return ls.addr
}
