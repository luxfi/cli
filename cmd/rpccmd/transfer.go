// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rpccmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/luxfi/address"
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/constants"
	"github.com/luxfi/evm/plugin/evm/atomic"
	"github.com/luxfi/formatting"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/ethclient"
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/api/info"
	"github.com/luxfi/sdk/api/platformvm"
	"github.com/luxfi/sdk/wallet/chain/c"
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
	wait      bool
}

func newTransferCmd(app *application.Lux) *cobra.Command {
	flags := &transferFlags{}
	cmd := &cobra.Command{
		Use:   "transfer",
		Short: "Transfer LUX across chains (P/X <-> C)",
		Long: `Transfer LUX across chains using atomic export/import.

Supported:
  - P -> C
  - X -> C
  - C -> P
  - C -> X

Example:
  lux rpc transfer --from-chain P --to-chain C --to 0x9011... --amount 10
`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runTransfer(app, flags)
		},
	}

	cmd.Flags().StringVar(&flags.rpcURL, "rpc-url", "", "Base RPC URL (default: LUX_RPC_URL or running network endpoint)")
	cmd.Flags().StringVar(&flags.from, "from", "", "Key name to use (default: LUX_MNEMONIC account 0)")
	cmd.Flags().StringVar(&flags.fromChain, "from-chain", "P", "Source chain: P, X, or C")
	cmd.Flags().StringVar(&flags.toChain, "to-chain", "C", "Destination chain: P, X, or C")
	cmd.Flags().StringVar(&flags.to, "to", "", "Destination address (C-Chain hex for C, bech32 for P/X)")
	cmd.Flags().Float64Var(&flags.amount, "amount", 0, "Amount to transfer in LUX")
	cmd.Flags().BoolVar(&flags.wait, "wait", true, "Wait for export acceptance before import")

	_ = cmd.MarkFlagRequired("amount")
	_ = cmd.MarkFlagRequired("to")

	return cmd
}

func runTransfer(app *application.Lux, flags *transferFlags) error {
	if flags.amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	fromChain := strings.ToUpper(flags.fromChain)
	toChain := strings.ToUpper(flags.toChain)
	if fromChain == toChain {
		return fmt.Errorf("from-chain and to-chain must differ")
	}

	baseURL, err := resolveRPCBaseURL(app, flags.rpcURL)
	if err != nil {
		return err
	}
	networkID, err := resolveNetworkID(baseURL)
	if err != nil {
		return err
	}

	softKey, err := loadSoftKeyForTransfer(networkID, flags.from)
	if err != nil {
		return err
	}

	switch fromChain {
	case "P", "X":
		if toChain != "C" {
			return fmt.Errorf("only P/X -> C is supported from %s", fromChain)
		}
		return transferPXToC(baseURL, networkID, softKey, fromChain, flags.to, flags.amount, flags.wait)
	case "C":
		if toChain != "P" && toChain != "X" {
			return fmt.Errorf("only C -> P/X is supported")
		}
		return transferCToPX(baseURL, networkID, softKey, toChain, flags.to, flags.amount, flags.wait)
	default:
		return fmt.Errorf("unsupported from-chain: %s", fromChain)
	}
}

func resolveRPCBaseURL(app *application.Lux, override string) (string, error) {
	if override != "" {
		return strings.TrimSuffix(override, "/")
	}
	if env := os.Getenv("LUX_RPC_URL"); env != "" {
		return strings.TrimSuffix(env, "/")
	}
	if app != nil {
		if endpoint := app.GetRunningNetworkEndpoint(); endpoint != "" {
			return strings.TrimSuffix(endpoint, "/")
		}
	}
	return "", fmt.Errorf("rpc base URL not set (use --rpc-url or LUX_RPC_URL)")
}

func resolveNetworkID(baseURL string) (uint32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	infoClient := info.NewClient(baseURL)
	networkID, err := infoClient.GetNetworkID(ctx)
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
		return nil, fmt.Errorf("no key specified and LUX_MNEMONIC not set")
	}
	return key.NewSoftFromMnemonic(networkID, mnemonic)
}

func transferPXToC(baseURL string, networkID uint32, sk *key.SoftKey, source string, toAddr string, amount float64, wait bool) error {
	if !common.IsHexAddress(toAddr) {
		return fmt.Errorf("invalid C-Chain address: %s", toAddr)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pWallet, kcAdapter, err := makePrimaryWallet(ctx, baseURL, sk)
	if err != nil {
		return err
	}

	amountNLUX, err := luxToNLUX(amount)
	if err != nil {
		return err
	}

	outputOwner, err := outputOwnerFromKey(sk)
	if err != nil {
		return err
	}

	// Export from P/X to C (UTXO in shared memory for C)
	if source == "P" {
		_, err = pWallet.P().IssueExportTx(constants.CChainID, []*secp256k1fx.TransferOutput{{
			Amt:          amountNLUX,
			OutputOwners: *outputOwner,
		}})
		if err != nil {
			return fmt.Errorf("P->C export failed: %w", err)
		}
	} else {
		_, err = pWallet.X().IssueExportTx(constants.CChainID, []*utxo.TransferableOutput{{
			Asset: utxo.Asset{ID: pWallet.X().Builder().Context().XAssetID},
			Out: &secp256k1fx.TransferOutput{
				Amt:          amountNLUX,
				OutputOwners: *outputOwner,
			},
		}})
		if err != nil {
			return fmt.Errorf("X->C export failed: %w", err)
		}
	}

	if wait {
		time.Sleep(2 * time.Second)
	}

	// Import on C
	cCtx, cBackend, ethClient, err := newCChainRPCBackend(ctx, baseURL, networkID, sk, kcAdapter)
	if err != nil {
		return err
	}
	baseFee, err := getBaseFee(ctx, ethClient)
	if err != nil {
		return err
	}

	builder := c.NewBuilder(kcAdapter.Addresses(), kcAdapter.EthAddresses(), cCtx, cBackend)
	importTx, err := builder.NewImportTx(chainIDFromAlias(source), common.HexToAddress(toAddr), baseFee)
	if err != nil {
		return fmt.Errorf("C import tx build failed: %w", err)
	}
	signer := c.NewSigner(kcAdapter, kcAdapter, cBackend)
	signed, err := c.SignUnsignedAtomic(ctx, signer, importTx)
	if err != nil {
		return fmt.Errorf("C import tx sign failed: %w", err)
	}

	txID, err := issueCChainAtomicTx(ctx, baseURL, signed)
	if err != nil {
		return err
	}

	ux.Logger.PrintToUser("P/X -> C transfer submitted")
	ux.Logger.PrintToUser("  Import TxID: %s", txID)
	return nil
}

func transferCToPX(baseURL string, networkID uint32, sk *key.SoftKey, dest string, toAddr string, amount float64, wait bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pWallet, kcAdapter, err := makePrimaryWallet(ctx, baseURL, sk)
	if err != nil {
		return err
	}

	// Build C export tx
	cCtx, cBackend, ethClient, err := newCChainRPCBackend(ctx, baseURL, networkID, sk, kcAdapter)
	if err != nil {
		return err
	}
	baseFee, err := getBaseFee(ctx, ethClient)
	if err != nil {
		return err
	}

	outputOwner, err := outputOwnerFromBech32Address(toAddr)
	if err != nil {
		return err
	}
	amountNLUX, err := luxToNLUX(amount)
	if err != nil {
		return err
	}

	destChainID := chainIDFromAlias(dest)
	builder := c.NewBuilder(kcAdapter.Addresses(), kcAdapter.EthAddresses(), cCtx, cBackend)
	exportTx, err := builder.NewExportTx(destChainID, []*secp256k1fx.TransferOutput{{
		Amt:          amountNLUX,
		OutputOwners: *outputOwner,
	}}, baseFee)
	if err != nil {
		return fmt.Errorf("C export tx build failed: %w", err)
	}
	signer := c.NewSigner(kcAdapter, kcAdapter, cBackend)
	signed, err := c.SignUnsignedAtomic(ctx, signer, exportTx)
	if err != nil {
		return fmt.Errorf("C export tx sign failed: %w", err)
	}
	exportTxID, err := issueCChainAtomicTx(ctx, baseURL, signed)
	if err != nil {
		return err
	}

	if wait {
		if err := waitAtomicAccepted(ctx, baseURL, exportTxID); err != nil {
			return err
		}
	}

	// Import on P/X
	if dest == "P" {
		_, err = pWallet.P().IssueImportTx(constants.CChainID, outputOwner)
		if err != nil {
			return fmt.Errorf("C->P import failed: %w", err)
		}
	} else {
		_, err = pWallet.X().IssueImportTx(constants.CChainID, outputOwner)
		if err != nil {
			return fmt.Errorf("C->X import failed: %w", err)
		}
	}

	ux.Logger.PrintToUser("C -> P/X transfer submitted")
	ux.Logger.PrintToUser("  Export TxID: %s", exportTxID)
	return nil
}

func makePrimaryWallet(ctx context.Context, baseURL string, sk *key.SoftKey) (primary.Wallet, *primary.KeychainAdapter, error) {
	kc := secp256k1fx.NewKeychain(sk.Key())
	adapter := primary.NewKeychainAdapter(kc)
	wallet, err := primary.MakeWallet(ctx, &primary.WalletConfig{
		URI:         baseURL,
		LUXKeychain: adapter,
		EthKeychain: adapter,
	})
	if err != nil {
		return nil, nil, err
	}
	return wallet, adapter, nil
}

func newCChainRPCBackend(
	ctx context.Context,
	baseURL string,
	networkID uint32,
	sk *key.SoftKey,
	adapter *primary.KeychainAdapter,
) (*c.Context, *rpcCBackend, *ethclient.Client, error) {
	infoClient := info.NewClient(baseURL)
	platformClient := platformvm.NewClient(baseURL)
	luxAssetID, err := platformClient.GetStakingAssetID(ctx, networkID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get LUX asset ID: %w", err)
	}
	cCtx, err := c.NewContextFromClients(ctx, infoClient, luxAssetID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create C context: %w", err)
	}
	rpcURL := fmt.Sprintf("%s/ext/bc/C/rpc", baseURL)
	ethClient, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, nil, nil, err
	}
	backend := newRPCCBackend(baseURL, sk)
	if err := backend.RefreshUTXOs(ctx); err != nil {
		return nil, nil, nil, err
	}
	return cCtx, backend, ethClient, nil
}

type rpcCBackend struct {
	baseURL string
	utxos   map[ids.ID]*utxo.UTXO
	pAddrs  []string
	xAddrs  []string
}

func newRPCCBackend(baseURL string, sk *key.SoftKey) *rpcCBackend {
	return &rpcCBackend{
		baseURL: baseURL,
		utxos:   make(map[ids.ID]*utxo.UTXO),
		pAddrs:  sk.P(),
		xAddrs:  sk.X(),
	}
}

func (b *rpcCBackend) RefreshUTXOs(ctx context.Context) error {
	b.utxos = make(map[ids.ID]*utxo.UTXO)
	for _, source := range []string{"P", "X"} {
		addrs := b.pAddrs
		if source == "X" {
			addrs = b.xAddrs
		}
		utxos, err := fetchCChainUTXOs(ctx, b.baseURL, source, addrs)
		if err != nil {
			return err
		}
		for _, u := range utxos {
			b.utxos[u.InputID()] = u
		}
	}
	return nil
}

func (b *rpcCBackend) AddUTXO(_ context.Context, _ ids.ID, utxo *utxo.UTXO) error {
	b.utxos[utxo.InputID()] = utxo
	return nil
}

func (b *rpcCBackend) RemoveUTXO(_ context.Context, _ ids.ID, utxoID ids.ID) error {
	delete(b.utxos, utxoID)
	return nil
}

func (b *rpcCBackend) UTXOs(_ context.Context, _ ids.ID) ([]*utxo.UTXO, error) {
	out := make([]*utxo.UTXO, 0, len(b.utxos))
	for _, u := range b.utxos {
		out = append(out, u)
	}
	return out, nil
}

func (b *rpcCBackend) GetUTXO(_ context.Context, _ ids.ID, utxoID ids.ID) (*utxo.UTXO, error) {
	u, ok := b.utxos[utxoID]
	if !ok {
		return nil, fmt.Errorf("utxo not found")
	}
	return u, nil
}

func (b *rpcCBackend) Balance(ctx context.Context, addr common.Address) (*big.Int, error) {
	ethClient, err := ethclient.DialContext(ctx, fmt.Sprintf("%s/ext/bc/C/rpc", b.baseURL))
	if err != nil {
		return nil, err
	}
	return ethClient.BalanceAt(ctx, addr, nil)
}

func (b *rpcCBackend) Nonce(ctx context.Context, addr common.Address) (uint64, error) {
	ethClient, err := ethclient.DialContext(ctx, fmt.Sprintf("%s/ext/bc/C/rpc", b.baseURL))
	if err != nil {
		return 0, err
	}
	return ethClient.PendingNonceAt(ctx, addr)
}

func fetchCChainUTXOs(ctx context.Context, baseURL, sourceChain string, addrs []string) ([]*utxo.UTXO, error) {
	endpoint := fmt.Sprintf("%s/ext/bc/C/lux", baseURL)
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "lux.getUTXOs",
		"params": map[string]interface{}{
			"addresses":   addrs,
			"sourceChain": sourceChain,
			"limit":       1024,
			"encoding":    "hex",
		},
	}
	data, _ := json.Marshal(req)
	httpClient := &http.Client{Timeout: 15 * time.Second}
	resp, err := httpClient.Post(endpoint, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		Result struct {
			UTXOs    []string `json:"utxos"`
			Encoding string   `json:"encoding"`
		} `json:"result"`
		Error interface{} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, fmt.Errorf("lux.getUTXOs error: %v", result.Error)
	}

	utxos := make([]*utxo.UTXO, 0, len(result.Result.UTXOs))
	for _, u := range result.Result.UTXOs {
		raw, err := formatting.Decode(result.Result.Encoding, u)
		if err != nil {
			return nil, err
		}
		utxoObj := &utxo.UTXO{}
		if _, err := atomic.Codec.Unmarshal(raw, utxoObj); err != nil {
			return nil, err
		}
		utxos = append(utxos, utxoObj)
	}
	return utxos, nil
}

func issueCChainAtomicTx(ctx context.Context, baseURL string, tx *c.Tx) (string, error) {
	endpoint := fmt.Sprintf("%s/ext/bc/C/lux", baseURL)
	encoded, err := formatting.Encode("hex", tx.SignedBytes())
	if err != nil {
		return "", err
	}
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "lux.issueTx",
		"params": map[string]interface{}{
			"tx":       encoded,
			"encoding": "hex",
		},
	}
	data, _ := json.Marshal(req)
	httpClient := &http.Client{Timeout: 15 * time.Second}
	resp, err := httpClient.Post(endpoint, "application/json", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		Result struct {
			TxID string `json:"txID"`
		} `json:"result"`
		Error interface{} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Error != nil {
		return "", fmt.Errorf("lux.issueTx error: %v", result.Error)
	}
	return result.Result.TxID, nil
}

func waitAtomicAccepted(ctx context.Context, baseURL, txID string) error {
	endpoint := fmt.Sprintf("%s/ext/bc/C/lux", baseURL)
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "lux.getAtomicTxStatus",
		"params": map[string]interface{}{
			"txID": txID,
		},
	}

	httpClient := &http.Client{Timeout: 10 * time.Second}
	for {
		data, _ := json.Marshal(req)
		resp, err := httpClient.Post(endpoint, "application/json", bytes.NewReader(data))
		if err != nil {
			return err
		}
		var result struct {
			Result struct {
				Status string `json:"status"`
			} `json:"result"`
			Error interface{} `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			_ = resp.Body.Close()
			return err
		}
		_ = resp.Body.Close()
		if result.Error != nil {
			return fmt.Errorf("lux.getAtomicTxStatus error: %v", result.Error)
		}
		if result.Result.Status == "Accepted" {
			return nil
		}
		select {
		case <-time.After(2 * time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func chainIDFromAlias(alias string) ids.ID {
	switch strings.ToUpper(alias) {
	case "P":
		return constants.PlatformChainID
	case "X":
		return constants.XChainID
	case "C":
		return constants.CChainID
	default:
		return ids.Empty
	}
}

func outputOwnerFromKey(sk *key.SoftKey) (*secp256k1fx.OutputOwners, error) {
	addrs := sk.Addresses()
	if len(addrs) == 0 {
		return nil, fmt.Errorf("no key addresses available")
	}
	return &secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs:     []ids.ShortID{addrs[0]},
	}, nil
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

func getBaseFee(ctx context.Context, client *ethclient.Client) (*big.Int, error) {
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}
	if header.BaseFee != nil {
		return header.BaseFee, nil
	}
	return client.SuggestGasPrice(ctx)
}

func luxToNLUX(amount float64) (uint64, error) {
	if amount <= 0 {
		return 0, fmt.Errorf("amount must be positive")
	}
	value := new(big.Float).Mul(new(big.Float).SetFloat64(amount), big.NewFloat(1e9))
	nlux := new(big.Int)
	value.Int(nlux)
	if !nlux.IsUint64() {
		return 0, fmt.Errorf("amount too large")
	}
	return nlux.Uint64(), nil
}
