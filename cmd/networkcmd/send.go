// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package networkcmd

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/cli/pkg/localnet"
	"github.com/luxfi/cli/pkg/ux"
	ethcrypto "github.com/luxfi/crypto"
	ethcommon "github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/ethclient"
	"github.com/spf13/cobra"
)

var (
	sendAmount      float64
	sendTo          string
	sendFromKey     string
	sendSourceChain string
	sendDestChain   string
)

func newSendCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send funds on the C-Chain of the running local network",
		Long: `Send funds on the C-Chain of the running local network.

This command uses a local key (LUX_MNEMONIC or --from) to sign a C-Chain
transfer and submit it to the running network's C-Chain RPC.

Examples:
  # Send 100 LUX to a C-Chain address (uses LUX_MNEMONIC account 0)
  lux network send --amount 100 --to 0x9011E888251AB053B7bD1cdB598Db4f9DEd94714

  # Send with a specific stored key
  lux network send --amount 25 --to 0x... --from node1

Notes:
  - Amount is in LUX (converted to wei)
  - Requires a running local network (mainnet/testnet/devnet)
  - Source/dest flags are accepted but only C->C is supported right now`,
		RunE: runSend,
	}

	cmd.Flags().Float64Var(&sendAmount, "amount", 0, "Amount to send in LUX (required)")
	cmd.Flags().StringVar(&sendTo, "to", "", "Destination address (C-Chain hex address)")
	cmd.Flags().StringVar(&sendFromKey, "from", "", "Key name to use for signing (default: LUX_MNEMONIC account 0)")
	cmd.Flags().StringVar(&sendSourceChain, "source", "C", "Source chain (only C supported)")
	cmd.Flags().StringVar(&sendDestChain, "dest", "C", "Destination chain (only C supported)")

	_ = cmd.MarkFlagRequired("amount")
	_ = cmd.MarkFlagRequired("to")

	return cmd
}

func runSend(_ *cobra.Command, _ []string) error {
	if sendAmount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	if sendTo == "" {
		return fmt.Errorf("destination address required (--to)")
	}
	if !ethcommon.IsHexAddress(sendTo) {
		return fmt.Errorf("invalid C-Chain address: %s", sendTo)
	}
	if strings.ToUpper(sendSourceChain) != "C" || strings.ToUpper(sendDestChain) != "C" {
		return fmt.Errorf("only C->C transfers are supported right now")
	}

	running, err := localnet.LocalNetworkIsRunning(app)
	if err != nil {
		return fmt.Errorf("failed to check network status: %w", err)
	}
	if !running {
		return fmt.Errorf("no local network running, start one with 'lux network start'")
	}

	state, err := findRunningNetworkState(app)
	if err != nil {
		return err
	}
	endpoint := state.APIEndpoint
	if endpoint == "" {
		endpoint = app.GetRunningNetworkEndpoint()
	}
	if endpoint == "" {
		return fmt.Errorf("could not determine network endpoint")
	}

	networkID := state.NetworkID
	if networkID == 0 {
		networkID = networkIDFromType(state.NetworkType)
	}

	softKey, err := loadSoftKey(networkID)
	if err != nil {
		return err
	}

	toAddr := ethcommon.HexToAddress(sendTo)
	valueWei, err := luxToWei(sendAmount)
	if err != nil {
		return err
	}

	rpcURL := fmt.Sprintf("%s/ext/bc/C/rpc", endpoint)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return fmt.Errorf("failed to connect to C-Chain RPC (%s): %w", rpcURL, err)
	}

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get chain ID: %w", err)
	}

	privKey, err := ethcrypto.ToECDSA(softKey.Raw())
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}
	fromAddr := ethcommon.Address(ethcrypto.PubkeyToAddress(privKey.PublicKey))

	nonce, err := client.PendingNonceAt(ctx, fromAddr)
	if err != nil {
		return fmt.Errorf("failed to get nonce: %w", err)
	}

	tx, err := buildSignedTx(ctx, client, chainID, nonce, toAddr, valueWei, privKey)
	if err != nil {
		return err
	}

	if err := client.SendTransaction(ctx, tx); err != nil {
		return fmt.Errorf("failed to send transaction: %w", err)
	}

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("C-Chain transfer submitted")
	ux.Logger.PrintToUser("  From:   %s", fromAddr.Hex())
	ux.Logger.PrintToUser("  To:     %s", toAddr.Hex())
	ux.Logger.PrintToUser("  Amount: %.6f LUX", sendAmount)
	ux.Logger.PrintToUser("  TxID:   %s", tx.Hash().Hex())
	ux.Logger.PrintToUser("")

	return nil
}

func loadSoftKey(networkID uint32) (*key.SoftKey, error) {
	if sendFromKey != "" {
		keySet, err := key.LoadKeySet(sendFromKey)
		if err != nil {
			return nil, fmt.Errorf("failed to load key '%s': %w", sendFromKey, err)
		}
		if len(keySet.ECPrivateKey) == 0 {
			return nil, fmt.Errorf("key '%s' has no EC private key", sendFromKey)
		}
		softKey, err := key.NewSoftFromBytes(networkID, keySet.ECPrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create soft key: %w", err)
		}
		return softKey, nil
	}

	mnemonic := key.GetMnemonicFromEnv()
	if mnemonic == "" {
		return nil, fmt.Errorf("no key specified and LUX_MNEMONIC not set")
	}
	softKey, err := key.NewSoftFromMnemonic(networkID, mnemonic)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key from mnemonic: %w", err)
	}
	return softKey, nil
}

func findRunningNetworkState(app *application.Lux) (*application.NetworkState, error) {
	if state, err := app.LoadNetworkState(); err == nil && state != nil {
		if state.Running || state.APIEndpoint != "" {
			return state, nil
		}
	}
	for _, netType := range []string{"mainnet", "testnet", "devnet", "custom"} {
		state, err := app.LoadNetworkStateForType(netType)
		if err != nil || state == nil {
			continue
		}
		if state.Running || state.APIEndpoint != "" {
			return state, nil
		}
	}
	return nil, fmt.Errorf("no running network state found")
}

func networkIDFromType(netType string) uint32 {
	switch netType {
	case "mainnet":
		return 1
	case "testnet":
		return 2
	case "devnet":
		return 5
	default:
		return 0
	}
}

func luxToWei(amount float64) (*big.Int, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}
	amountFloat := new(big.Float).SetPrec(256).SetFloat64(amount)
	if amountFloat.Sign() <= 0 {
		return nil, fmt.Errorf("invalid amount")
	}
	weiPerLux := new(big.Float).SetPrec(256).SetInt(big.NewInt(0).Exp(big.NewInt(10), big.NewInt(18), nil))
	amountFloat.Mul(amountFloat, weiPerLux)
	wei := new(big.Int)
	amountFloat.Int(wei)
	if wei.Sign() <= 0 {
		return nil, fmt.Errorf("amount too small (rounded to 0 wei)")
	}
	return wei, nil
}

func buildSignedTx(
	ctx context.Context,
	client *ethclient.Client,
	chainID *big.Int,
	nonce uint64,
	to ethcommon.Address,
	value *big.Int,
	privKey *ecdsa.PrivateKey,
) (*types.Transaction, error) {
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest header: %w", err)
	}

	if header.BaseFee == nil {
		gasPrice, err := client.SuggestGasPrice(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to suggest gas price: %w", err)
		}
		tx := types.NewTx(&types.LegacyTx{
			Nonce:    nonce,
			To:       &to,
			Value:    value,
			Gas:      21000,
			GasPrice: gasPrice,
		})
		signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), privKey)
		if err != nil {
			return nil, fmt.Errorf("failed to sign legacy tx: %w", err)
		}
		return signedTx, nil
	}

	tipCap, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to suggest gas tip cap: %w", err)
	}
	feeCap := new(big.Int).Add(new(big.Int).Mul(header.BaseFee, big.NewInt(2)), tipCap)

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		To:        &to,
		Value:     value,
		Gas:       21000,
		GasTipCap: tipCap,
		GasFeeCap: feeCap,
	})
	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign dynamic fee tx: %w", err)
	}
	return signedTx, nil
}
