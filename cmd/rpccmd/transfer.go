// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rpccmd

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/luxfi/address"
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/constants"
	"github.com/luxfi/ids"
	sdkinfo "github.com/luxfi/sdk/info"
	"github.com/luxfi/sdk/wallet/primary"
	"github.com/luxfi/utxo"
	"github.com/luxfi/utxo/secp256k1fx"
	"github.com/spf13/cobra"
)

type transferFlags struct {
	rpcURL    string
	from      string
	fromChain string
	toChain   string
	to        string
	amount    float64
}

func newTransferCmd(app *application.Lux) *cobra.Command {
	flags := &transferFlags{}
	cmd := &cobra.Command{
		Use:   "transfer",
		Short: "Transfer LUX between the P and X chains",
		Long: `Transfer LUX between the P and X chains by atomic export/import.

The C-Chain is an EVM: move value on it with an ordinary EVM transaction
against /v1/bc/C/rpc, not with this command.

Example:
  lux rpc transfer --from-chain P --to-chain X --to X-lux1... --amount 10
`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runTransfer(app, flags)
		},
	}

	cmd.Flags().StringVar(&flags.rpcURL, "rpc-url", "", "Base RPC URL (default: RPC_URL or running network endpoint)")
	cmd.Flags().StringVar(&flags.from, "from", "", "Key name to use (default: MNEMONIC account 0)")
	cmd.Flags().StringVar(&flags.fromChain, "from-chain", "P", "Source chain: P or X")
	cmd.Flags().StringVar(&flags.toChain, "to-chain", "X", "Destination chain: P or X")
	cmd.Flags().StringVar(&flags.to, "to", "", "Destination bech32 address")
	cmd.Flags().Float64Var(&flags.amount, "amount", 0, "Amount to transfer in LUX")

	_ = cmd.MarkFlagRequired("amount")
	_ = cmd.MarkFlagRequired("to")

	return cmd
}

func runTransfer(app *application.Lux, flags *transferFlags) error {
	source, err := chainIDFromAlias(flags.fromChain)
	if err != nil {
		return err
	}
	destination, err := chainIDFromAlias(flags.toChain)
	if err != nil {
		return err
	}
	if source == destination {
		return fmt.Errorf("from-chain and to-chain must differ")
	}

	amount, err := luxToMicroLux(flags.amount)
	if err != nil {
		return err
	}
	owner, err := outputOwnerFromBech32Address(flags.to)
	if err != nil {
		return err
	}

	baseURL, err := resolveRPCBaseURL(app, flags.rpcURL)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	networkID, err := resolveNetworkID(ctx, baseURL)
	if err != nil {
		return err
	}
	softKey, err := loadSoftKeyForTransfer(networkID, flags.from)
	if err != nil {
		return err
	}
	wallet, err := makePrimaryWallet(ctx, baseURL, softKey)
	if err != nil {
		return err
	}

	var exportID, importID ids.ID
	if source == constants.PlatformChainID {
		exported := []*utxo.TransferableOutput{transferOut(wallet.P().Builder().Context().UTXOAssetID, amount, owner)}
		exportTx, err := wallet.P().IssueExportTx(destination, exported)
		if err != nil {
			return fmt.Errorf("P export failed: %w", err)
		}
		importTx, err := wallet.X().IssueImportTx(source, owner)
		if err != nil {
			return fmt.Errorf("X import failed: %w", err)
		}
		exportID, importID = exportTx.ID(), importTx.ID()
	} else {
		exported := []*utxo.TransferableOutput{transferOut(wallet.X().Builder().Context().UTXOAssetID, amount, owner)}
		exportTx, err := wallet.X().IssueExportTx(destination, exported)
		if err != nil {
			return fmt.Errorf("X export failed: %w", err)
		}
		importTx, err := wallet.P().IssueImportTx(source, owner)
		if err != nil {
			return fmt.Errorf("P import failed: %w", err)
		}
		exportID, importID = exportTx.ID(), importTx.ID()
	}

	ux.Logger.PrintToUser("%s -> %s transfer submitted", flags.fromChain, flags.toChain)
	ux.Logger.PrintToUser("  Export TxID: %s", exportID)
	ux.Logger.PrintToUser("  Import TxID: %s", importID)
	return nil
}

func transferOut(assetID ids.ID, amount uint64, owner *secp256k1fx.OutputOwners) *utxo.TransferableOutput {
	return &utxo.TransferableOutput{
		Asset: utxo.Asset{ID: assetID},
		Out: &secp256k1fx.TransferOutput{
			Amt:          amount,
			OutputOwners: *owner,
		},
	}
}

func resolveRPCBaseURL(app *application.Lux, override string) (string, error) {
	if override != "" {
		return strings.TrimSuffix(override, "/"), nil
	}
	if env := os.Getenv("RPC_URL"); env != "" {
		return strings.TrimSuffix(env, "/"), nil
	}
	if app != nil {
		if endpoint := app.GetRunningNetworkEndpoint(); endpoint != "" {
			return strings.TrimSuffix(endpoint, "/"), nil
		}
	}
	return "", fmt.Errorf("rpc base URL not set (use --rpc-url or RPC_URL)")
}

func resolveNetworkID(ctx context.Context, baseURL string) (uint32, error) {
	networkID, err := sdkinfo.NewClient(baseURL).GetNetworkID(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get network ID: %w", err)
	}
	return networkID, nil
}

func loadSoftKeyForTransfer(networkID uint32, from string) (*key.SoftKey, error) {
	if from != "" {
		keySet, err := key.LoadKeySet(from)
		if err != nil {
			return nil, fmt.Errorf("failed to load key '%s': %w", from, err)
		}
		if len(keySet.ECPrivateKey) == 0 {
			return nil, fmt.Errorf("key '%s' has no EC private key", from)
		}
		return key.NewSoftFromBytes(networkID, keySet.ECPrivateKey)
	}
	mnemonic := key.GetMnemonicFromEnv()
	if mnemonic == "" {
		return nil, fmt.Errorf("no key specified and MNEMONIC not set")
	}
	return key.NewSoftFromMnemonic(networkID, mnemonic)
}

func makePrimaryWallet(ctx context.Context, baseURL string, sk *key.SoftKey) (primary.Wallet, error) {
	return primary.MakeWallet(ctx, &primary.WalletConfig{
		URI:         baseURL,
		LUXKeychain: primary.NewKeychainAdapter(secp256k1fx.NewKeychain(sk.Key())),
	})
}

func chainIDFromAlias(alias string) (ids.ID, error) {
	switch strings.ToUpper(alias) {
	case "P":
		return constants.PlatformChainID, nil
	case "X":
		return constants.XChainID, nil
	default:
		return ids.Empty, fmt.Errorf("unsupported chain %q: transfer moves value between P and X", alias)
	}
}

func outputOwnerFromBech32Address(addr string) (*secp256k1fx.OutputOwners, error) {
	bech32Addr := addr
	if parts := strings.SplitN(addr, "-", 2); len(parts) == 2 {
		bech32Addr = parts[1]
	}
	_, addrBytes, err := address.ParseBech32(bech32Addr)
	if err != nil {
		return nil, fmt.Errorf("invalid bech32 address: %w", err)
	}
	shortID, err := ids.ToShortID(addrBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid bech32 address bytes: %w", err)
	}
	return &secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs:     []ids.ShortID{shortID},
	}, nil
}

// luxToMicroLux converts whole LUX to the P/X-Chain base unit (microLUX, 6
// decimals — NOT the legacy 9-decimal nanoLUX). 1 LUX = 1e6 microLUX.
func luxToMicroLux(amount float64) (uint64, error) {
	if amount <= 0 {
		return 0, fmt.Errorf("amount must be positive")
	}
	value := new(big.Float).Mul(new(big.Float).SetFloat64(amount), big.NewFloat(1e6))
	microlux := new(big.Int)
	value.Int(microlux)
	if !microlux.IsUint64() {
		return 0, fmt.Errorf("amount too large")
	}
	return microlux.Uint64(), nil
}
