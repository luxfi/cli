// Copyright (C) 2022, Lux Partners Limited, All rights reserved.
// See the file LICENSE for licensing terms.
package keycmd

import (
	"context"
	"fmt"
	"time"

	"github.com/luxdefi/cli/pkg/key"
	"github.com/luxdefi/cli/pkg/models"
	"github.com/luxdefi/cli/pkg/prompts"
	"github.com/luxdefi/cli/pkg/subnet"
	"github.com/luxdefi/cli/pkg/utils"
	"github.com/luxdefi/cli/pkg/ux"
	"github.com/luxdefi/node/ids"
	avagoconstants "github.com/luxdefi/node/utils/constants"
	"github.com/luxdefi/node/utils/crypto/keychain"
	ledger "github.com/luxdefi/node/utils/crypto/ledger"
	"github.com/luxdefi/node/utils/formatting/address"
	"github.com/luxdefi/node/utils/logging"
	"github.com/luxdefi/node/utils/units"
	avmtxs "github.com/luxdefi/node/vms/avm/txs"
	"github.com/luxdefi/node/vms/components/lux"
	"github.com/luxdefi/node/vms/platformvm/txs"
	"github.com/luxdefi/node/vms/secp256k1fx"
	"github.com/luxdefi/node/wallet/subnet/primary"
	"github.com/luxdefi/node/wallet/subnet/primary/common"
	"github.com/spf13/cobra"
)

const (
	sendFlag                = "send"
	receiveFlag             = "receive"
	keyNameFlag             = "key"
	ledgerIndexFlag         = "ledger"
	receiverAddrFlag        = "target-addr"
	amountFlag              = "amount"
	wrongLedgerIndexVal     = 32768
	receiveRecoveryStepFlag = "receive-recovery-step"
)

var (
	send                bool
	receive             bool
	keyName             string
	ledgerIndex         uint32
	force               bool
	receiverAddrStr     string
	amountFlt           float64
	receiveRecoveryStep uint64
)

func newTransferCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "transfer [options]",
		Short:        "Fund a ledger address or stored key from another one",
		Long:         `The key transfer command allows to transfer funds between stored keys or ledger addresses.`,
		RunE:         transferF,
		Args:         cobra.ExactArgs(0),
		SilenceUsage: true,
	}
	cmd.Flags().BoolVarP(
		&force,
		forceFlag,
		"f",
		false,
		"avoid transfer confirmation",
	)
	cmd.Flags().BoolVarP(
		&local,
		localFlag,
		"l",
		false,
		"transfer between local network addresses",
	)
	cmd.Flags().BoolVarP(
		&testnet,
		fujiFlag,
		"u",
		false,
		"transfer between testnet (fuji) addresses",
	)
	cmd.Flags().BoolVarP(
		&testnet,
		testnetFlag,
		"t",
		false,
		"transfer between testnet (fuji) addresses",
	)
	cmd.Flags().BoolVarP(
		&mainnet,
		mainnetFlag,
		"m",
		false,
		"transfer between mainnet addresses",
	)
	cmd.Flags().BoolVarP(
		&send,
		sendFlag,
		"s",
		false,
		"send the transfer",
	)
	cmd.Flags().BoolVarP(
		&receive,
		receiveFlag,
		"g",
		false,
		"receive the transfer",
	)
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
	cmd.Flags().Uint64VarP(
		&receiveRecoveryStep,
		receiveRecoveryStepFlag,
		"r",
		0,
		"receive step to use for multiple step transaction recovery",
	)
	cmd.Flags().StringVarP(
		&receiverAddrStr,
		receiverAddrFlag,
		"a",
		"",
		"receiver address",
	)
	cmd.Flags().Float64VarP(
		&amountFlt,
		amountFlag,
		"o",
		0,
		"amount to send or receive (LUX units)",
	)
	return cmd
}

func transferF(*cobra.Command, []string) error {
	if send && receive {
		return fmt.Errorf("only one of %s, %s flags should be selected", sendFlag, receiveFlag)
	}

	if keyName != "" && ledgerIndex != wrongLedgerIndexVal {
		return fmt.Errorf("only one between a keyname or a ledger index must be given")
	}

	var network models.Network
	switch {
	case local:
		network = models.LocalNetwork
	case testnet:
		network = models.FujiNetwork
	case mainnet:
		network = models.MainnetNetwork
	default:
		networkStr, err := app.Prompt.CaptureList(
			"Network to use",
			[]string{models.Mainnet.String(), models.Fuji.String(), models.Local.String()},
		)
		if err != nil {
			return err
		}
		network = models.NetworkFromString(networkStr)
	}

	var err error

	if !send && !receive {
		option, err := app.Prompt.CaptureList(
			"Step of the transfer",
			[]string{"Send", "Receive"},
		)
		if err != nil {
			return err
		}
		if option == "Send" {
			send = true
		} else {
			receive = true
		}
	}

	if keyName == "" && ledgerIndex == wrongLedgerIndexVal {
		var useLedger bool
		goalStr := ""
		if send {
			goalStr = " for the sender address"
		} else {
			goalStr = " for the receiver address"
		}
		useLedger, keyName, err = prompts.GetFujiKeyOrLedger(app.Prompt, goalStr, app.GetKeyDir())
		if err != nil {
			return err
		}
		if useLedger {
			ledgerIndex, err = app.Prompt.CaptureUint32("Ledger index to use")
			if err != nil {
				return err
			}
		}
	}

	if amountFlt == 0 {
		var promptStr string
		if send {
			promptStr = "Amount to send (LUX units)"
		} else {
			promptStr = "Amount to receive (LUX units)"
		}
		amountFlt, err = app.Prompt.CaptureFloat(promptStr, func(v float64) error {
			if v <= 0 {
				return fmt.Errorf("value %f must be greater than zero", v)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	amount := uint64(amountFlt * float64(units.Lux))

	fee := network.GenesisParams().TxFee

	var kc keychain.Keychain
	if keyName != "" {
		keyPath := app.GetKeyPath(keyName)
		sk, err := key.LoadSoft(network.ID, keyPath)
		if err != nil {
			return err
		}
		kc = sk.KeyChain()
	} else {
		ledgerDevice, err := ledger.New()
		if err != nil {
			return err
		}
		ledgerIndices := []uint32{ledgerIndex}
		kc, err = keychain.NewLedgerKeychainFromIndices(ledgerDevice, ledgerIndices)
		if err != nil {
			return err
		}
	}

	var receiverAddr ids.ShortID
	if send {
		if receiverAddrStr == "" {
			receiverAddrStr, err = app.Prompt.CapturePChainAddress("Receiver address", network)
			if err != nil {
				return err
			}
		}
		receiverAddr, err = address.ParseToID(receiverAddrStr)
		if err != nil {
			return err
		}
	} else {
		receiverAddr = kc.Addresses().List()[0]
		receiverAddrStr, err = address.Format("P", key.GetHRP(network.ID), receiverAddr[:])
		if err != nil {
			return err
		}
	}

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("this operation is going to:")
	if send {
		addr := kc.Addresses().List()[0]
		addrStr, err := address.Format("P", key.GetHRP(network.ID), addr[:])
		if err != nil {
			return err
		}
		if addr == receiverAddr {
			return fmt.Errorf("sender addr is the same as receiver addr")
		}
		ux.Logger.PrintToUser("- send %.9f LUX from %s to target address %s", float64(amount)/float64(units.Lux), addrStr, receiverAddrStr)
		ux.Logger.PrintToUser("- take a fee of %.9f LUX from source address %s", float64(4*fee)/float64(units.Lux), addrStr)
	} else {
		ux.Logger.PrintToUser("- receive %.9f LUX at target address %s", float64(amount)/float64(units.Lux), receiverAddrStr)
	}
	ux.Logger.PrintToUser("")

	if !force {
		confStr := "Confirm transfer"
		conf, err := app.Prompt.CaptureNoYes(confStr)
		if err != nil {
			return err
		}
		if !conf {
			ux.Logger.PrintToUser("Cancelled")
			return nil
		}
	}

	to := secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs:     []ids.ShortID{receiverAddr},
	}

	if send {
		wallet, err := primary.MakeWallet(
			context.Background(),
			&primary.WalletConfig{
				URI:          network.Endpoint,
				LUXKeychain: kc,
				EthKeychain:  secp256k1fx.NewKeychain(),
			},
		)
		if err != nil {
			return err
		}
		output := &lux.TransferableOutput{
			Asset: lux.Asset{ID: wallet.P().LUXAssetID()},
			Out: &secp256k1fx.TransferOutput{
				Amt:          amount + fee*3,
				OutputOwners: to,
			},
		}
		outputs := []*lux.TransferableOutput{output}
		ux.Logger.PrintToUser("Issuing ExportTx P -> X")

		if ledgerIndex != wrongLedgerIndexVal {
			ux.Logger.PrintToUser("*** Please sign 'Export Tx / P to X Chain' transaction on the ledger device *** ")
		}
		unsignedTx, err := wallet.P().Builder().NewExportTx(
			wallet.X().BlockchainID(),
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
				err = fmt.Errorf("timeout issuing/verifying tx with ID %s: %w", tx.ID(), err)
			} else {
				err = fmt.Errorf("error issuing tx with ID %s: %w", tx.ID(), err)
			}
			return err
		}
	} else {
		if receiveRecoveryStep == 0 {
			wallet, err := primary.MakeWallet(
				context.Background(),
				&primary.WalletConfig{
					URI:          network.Endpoint,
					LUXKeychain: kc,
					EthKeychain:  secp256k1fx.NewKeychain(),
				},
			)
			if err != nil {
				ux.Logger.PrintToUser(logging.LightRed.Wrap("ERROR: restart from this step by using the same command"))
				return err
			}
			ux.Logger.PrintToUser("Issuing ImportTx P -> X")
			if ledgerIndex != wrongLedgerIndexVal {
				ux.Logger.PrintToUser("*** Please sign ImportTx transaction on the ledger device *** ")
			}
			unsignedTx, err := wallet.X().Builder().NewImportTx(
				avagoconstants.PlatformChainID,
				&to,
			)
			if err != nil {
				ux.Logger.PrintToUser(logging.LightRed.Wrap("ERROR: restart from this step by using the same command"))
				return fmt.Errorf("error building tx: %w", err)
			}
			tx := avmtxs.Tx{Unsigned: unsignedTx}
			if err := wallet.X().Signer().Sign(context.Background(), &tx); err != nil {
				ux.Logger.PrintToUser(logging.LightRed.Wrap("ERROR: restart from this step by using the same command"))
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
					err = fmt.Errorf("timeout issuing/verifying tx with ID %s: %w", tx.ID(), err)
				} else {
					err = fmt.Errorf("error issuing tx with ID %s: %w", tx.ID(), err)
				}
				ux.Logger.PrintToUser(logging.LightRed.Wrap("ERROR: restart from this step by using the same command"))
				return err
			}

			time.Sleep(2 * time.Second)
			receiveRecoveryStep++
		}
		if receiveRecoveryStep == 1 {
			wallet, err := primary.MakeWallet(
				context.Background(),
				&primary.WalletConfig{
					URI:          network.Endpoint,
					LUXKeychain: kc,
					EthKeychain:  secp256k1fx.NewKeychain(),
				},
			)
			if err != nil {
				ux.Logger.PrintToUser(logging.LightRed.Wrap(fmt.Sprintf("ERROR: restart from this step by using the same command with extra arguments: --%s %d", receiveRecoveryStepFlag, receiveRecoveryStep)))
				return err
			}
			ux.Logger.PrintToUser("Issuing ExportTx X -> P")
			_, err = subnet.IssueXToPExportTx(
				wallet,
				ledgerIndex != wrongLedgerIndexVal,
				true,
				wallet.P().LUXAssetID(),
				amount+fee*1,
				&to,
			)
			if err != nil {
				ux.Logger.PrintToUser(logging.LightRed.Wrap(fmt.Sprintf("ERROR: restart from this step by using the same command with extra arguments: --%s %d", receiveRecoveryStepFlag, receiveRecoveryStep)))
				return err
			}
			time.Sleep(2 * time.Second)
			receiveRecoveryStep++
		}
		if receiveRecoveryStep == 2 {
			wallet, err := primary.MakeWallet(
				context.Background(),
				&primary.WalletConfig{
					URI:          network.Endpoint,
					LUXKeychain: kc,
					EthKeychain:  secp256k1fx.NewKeychain(),
				},
			)
			if err != nil {
				ux.Logger.PrintToUser(logging.LightRed.Wrap(fmt.Sprintf("ERROR: restart from this step by using the same command with extra arguments: --%s %d", receiveRecoveryStepFlag, receiveRecoveryStep)))
				return err
			}
			ux.Logger.PrintToUser("Issuing ImportTx X -> P")
			_, err = subnet.IssuePFromXImportTx(
				wallet,
				ledgerIndex != wrongLedgerIndexVal,
				true,
				&to,
			)
			if err != nil {
				ux.Logger.PrintToUser(logging.LightRed.Wrap(fmt.Sprintf("ERROR: restart from this step by using the same command with extra arguments: --%s %d", receiveRecoveryStepFlag, receiveRecoveryStep)))
				return err
			}
		}
	}

	return nil
}
