// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package ammcmd

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/luxfi/crypto"
	"github.com/luxfi/geth/accounts/abi"
	"github.com/luxfi/geth/accounts/abi/bind"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/ethclient"
	"github.com/luxfi/go-bip39"
)

// ABI strings for contract interactions
const (
	// ERC20 ABI
	ERC20ABI = `[
		{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"type":"function"},
		{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"type":"function"},
		{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"type":"function"},
		{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"type":"function"},
		{"constant":true,"inputs":[{"name":"owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"type":"function"},
		{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"type":"function"},
		{"constant":true,"inputs":[{"name":"owner","type":"address"},{"name":"spender","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"type":"function"},
		{"constant":false,"inputs":[{"name":"to","type":"address"},{"name":"amount","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"type":"function"}
	]`

	// Uniswap V2 Router ABI (minimal)
	V2RouterABI = `[
		{"inputs":[{"internalType":"uint256","name":"amountIn","type":"uint256"},{"internalType":"uint256","name":"amountOutMin","type":"uint256"},{"internalType":"address[]","name":"path","type":"address[]"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"deadline","type":"uint256"}],"name":"swapExactTokensForTokens","outputs":[{"internalType":"uint256[]","name":"amounts","type":"uint256[]"}],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"uint256","name":"amountOutMin","type":"uint256"},{"internalType":"address[]","name":"path","type":"address[]"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"deadline","type":"uint256"}],"name":"swapExactETHForTokens","outputs":[{"internalType":"uint256[]","name":"amounts","type":"uint256[]"}],"stateMutability":"payable","type":"function"},
		{"inputs":[{"internalType":"uint256","name":"amountIn","type":"uint256"},{"internalType":"uint256","name":"amountOutMin","type":"uint256"},{"internalType":"address[]","name":"path","type":"address[]"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"deadline","type":"uint256"}],"name":"swapExactTokensForETH","outputs":[{"internalType":"uint256[]","name":"amounts","type":"uint256[]"}],"stateMutability":"nonpayable","type":"function"},
		{"inputs":[{"internalType":"uint256","name":"amountIn","type":"uint256"},{"internalType":"address[]","name":"path","type":"address[]"}],"name":"getAmountsOut","outputs":[{"internalType":"uint256[]","name":"amounts","type":"uint256[]"}],"stateMutability":"view","type":"function"},
		{"inputs":[],"name":"WETH","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
		{"inputs":[],"name":"factory","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}
	]`

	// Uniswap V2 Factory ABI (minimal)
	V2FactoryABI = `[
		{"inputs":[{"internalType":"address","name":"tokenA","type":"address"},{"internalType":"address","name":"tokenB","type":"address"}],"name":"getPair","outputs":[{"internalType":"address","name":"pair","type":"address"}],"stateMutability":"view","type":"function"},
		{"inputs":[],"name":"allPairsLength","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"uint256","name":"","type":"uint256"}],"name":"allPairs","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}
	]`

	// Uniswap V2 Pair ABI (minimal)
	V2PairABI = `[
		{"constant":true,"inputs":[],"name":"token0","outputs":[{"name":"","type":"address"}],"type":"function"},
		{"constant":true,"inputs":[],"name":"token1","outputs":[{"name":"","type":"address"}],"type":"function"},
		{"constant":true,"inputs":[],"name":"getReserves","outputs":[{"name":"reserve0","type":"uint112"},{"name":"reserve1","type":"uint112"},{"name":"blockTimestampLast","type":"uint32"}],"type":"function"},
		{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"type":"function"}
	]`

	// Uniswap V3 SwapRouter ABI (minimal)
	V3RouterABI = `[
		{"inputs":[{"components":[{"internalType":"address","name":"tokenIn","type":"address"},{"internalType":"address","name":"tokenOut","type":"address"},{"internalType":"uint24","name":"fee","type":"uint24"},{"internalType":"address","name":"recipient","type":"address"},{"internalType":"uint256","name":"deadline","type":"uint256"},{"internalType":"uint256","name":"amountIn","type":"uint256"},{"internalType":"uint256","name":"amountOutMinimum","type":"uint256"},{"internalType":"uint160","name":"sqrtPriceLimitX96","type":"uint160"}],"internalType":"struct ISwapRouter.ExactInputSingleParams","name":"params","type":"tuple"}],"name":"exactInputSingle","outputs":[{"internalType":"uint256","name":"amountOut","type":"uint256"}],"stateMutability":"payable","type":"function"},
		{"inputs":[],"name":"WETH9","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}
	]`

	// Quoter ABI (minimal)
	QuoterABI = `[
		{"inputs":[{"internalType":"address","name":"tokenIn","type":"address"},{"internalType":"address","name":"tokenOut","type":"address"},{"internalType":"uint24","name":"fee","type":"uint24"},{"internalType":"uint256","name":"amountIn","type":"uint256"},{"internalType":"uint160","name":"sqrtPriceLimitX96","type":"uint160"}],"name":"quoteExactInputSingle","outputs":[{"internalType":"uint256","name":"amountOut","type":"uint256"}],"stateMutability":"nonpayable","type":"function"}
	]`

	// Uniswap V3 Pool ABI (minimal)
	V3PoolABI = `[
		{"constant":true,"inputs":[],"name":"token0","outputs":[{"name":"","type":"address"}],"type":"function"},
		{"constant":true,"inputs":[],"name":"token1","outputs":[{"name":"","type":"address"}],"type":"function"},
		{"constant":true,"inputs":[],"name":"fee","outputs":[{"name":"","type":"uint24"}],"type":"function"},
		{"constant":true,"inputs":[],"name":"liquidity","outputs":[{"name":"","type":"uint128"}],"type":"function"},
		{"constant":true,"inputs":[],"name":"slot0","outputs":[{"name":"sqrtPriceX96","type":"uint160"},{"name":"tick","type":"int24"},{"name":"observationIndex","type":"uint16"},{"name":"observationCardinality","type":"uint16"},{"name":"observationCardinalityNext","type":"uint16"},{"name":"feeProtocol","type":"uint8"},{"name":"unlocked","type":"bool"}],"type":"function"}
	]`

	// Uniswap V3 Factory ABI (minimal)
	V3FactoryABI = `[
		{"inputs":[{"internalType":"address","name":"tokenA","type":"address"},{"internalType":"address","name":"tokenB","type":"address"},{"internalType":"uint24","name":"fee","type":"uint24"}],"name":"getPool","outputs":[{"internalType":"address","name":"pool","type":"address"}],"stateMutability":"view","type":"function"}
	]`
)

// AMM represents an AMM client for interacting with Uniswap-style DEX
type AMM struct {
	config       *NetworkConfig
	client       *ethclient.Client
	auth         *bind.TransactOpts
	address      common.Address
	chainID      *big.Int
	v2Router     *bind.BoundContract
	v2Factory    *bind.BoundContract
	v3Router     *bind.BoundContract
	v3Factory    *bind.BoundContract
	quoter       *bind.BoundContract
	erc20ABI     abi.ABI
	v2RouterABI  abi.ABI
	v3RouterABI  abi.ABI
	v3FactoryABI abi.ABI
	quoterABI    abi.ABI
}

// TokenInfo holds ERC20 token information
type TokenInfo struct {
	Address  common.Address
	Name     string
	Symbol   string
	Decimals uint8
	Balance  *big.Int
}

// PoolInfo holds liquidity pool information
type PoolInfo struct {
	Address  common.Address
	Token0   common.Address
	Token1   common.Address
	Reserve0 *big.Int
	Reserve1 *big.Int
	TVL      *big.Float
}

// NewAMM creates a new AMM client for the specified network
func NewAMM(config *NetworkConfig) (*AMM, error) {
	// Connect to RPC
	client, err := ethclient.Dial(config.RPC)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", config.RPC, err)
	}

	// Verify chain ID
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	if chainID.Int64() != config.ChainID {
		return nil, fmt.Errorf("chain ID mismatch: expected %d, got %d", config.ChainID, chainID.Int64())
	}

	// Parse ABIs
	erc20ABI, err := abi.JSON(strings.NewReader(ERC20ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ERC20 ABI: %w", err)
	}

	v2RouterABI, err := abi.JSON(strings.NewReader(V2RouterABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse V2Router ABI: %w", err)
	}

	v2FactoryABI, err := abi.JSON(strings.NewReader(V2FactoryABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse V2Factory ABI: %w", err)
	}

	v3RouterABI, err := abi.JSON(strings.NewReader(V3RouterABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse V3Router ABI: %w", err)
	}

	v3FactoryABI, err := abi.JSON(strings.NewReader(V3FactoryABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse V3Factory ABI: %w", err)
	}

	quoterABI, err := abi.JSON(strings.NewReader(QuoterABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse Quoter ABI: %w", err)
	}

	// Bind contracts
	v2Router := bind.NewBoundContract(config.V2Router, v2RouterABI, client, client, client)
	v2Factory := bind.NewBoundContract(config.V2Factory, v2FactoryABI, client, client, client)
	v3Router := bind.NewBoundContract(config.V3Router, v3RouterABI, client, client, client)
	v3Factory := bind.NewBoundContract(config.V3Factory, v3FactoryABI, client, client, client)
	quoter := bind.NewBoundContract(config.Quoter, quoterABI, client, client, client)

	return &AMM{
		config:       config,
		client:       client,
		chainID:      chainID,
		v2Router:     v2Router,
		v2Factory:    v2Factory,
		v3Router:     v3Router,
		v3Factory:    v3Factory,
		quoter:       quoter,
		erc20ABI:     erc20ABI,
		v2RouterABI:  v2RouterABI,
		v3RouterABI:  v3RouterABI,
		v3FactoryABI: v3FactoryABI,
		quoterABI:    quoterABI,
	}, nil
}

// LoadWallet loads wallet from private key or mnemonic
// Priority: privateKey param > LUX_PRIVATE_KEY env > LUX_MNEMONIC env
func (a *AMM) LoadWallet() error {
	return a.LoadWalletWithKey("")
}

// LoadWalletWithKey loads wallet with optional private key parameter
func (a *AMM) LoadWalletWithKey(privateKey string) error {
	var key *ecdsa.PrivateKey
	var err error

	// Priority 1: passed private key
	if privateKey != "" {
		key, err = crypto.HexToECDSA(strings.TrimPrefix(privateKey, "0x"))
		if err != nil {
			return fmt.Errorf("invalid private key: %w", err)
		}
	}

	// Priority 2: LUX_PRIVATE_KEY environment variable
	if key == nil {
		if envKey := os.Getenv("LUX_PRIVATE_KEY"); envKey != "" {
			key, err = crypto.HexToECDSA(strings.TrimPrefix(envKey, "0x"))
			if err != nil {
				return fmt.Errorf("invalid LUX_PRIVATE_KEY: %w", err)
			}
		}
	}

	// Priority 3: LUX_MNEMONIC environment variable
	if key == nil {
		mnemonic := os.Getenv("LUX_MNEMONIC")
		if mnemonic == "" {
			return fmt.Errorf("no wallet credentials provided: use --private-key, LUX_PRIVATE_KEY, or LUX_MNEMONIC")
		}

		// Derive key from mnemonic using BIP39/BIP44
		seed := bip39.NewSeed(mnemonic, "")

		// Derive m/44'/60'/0'/0/0 (standard Ethereum path)
		key, err = deriveKey(seed, "m/44'/60'/0'/0/0")
		if err != nil {
			return fmt.Errorf("failed to derive key: %w", err)
		}
	}

	// Create transactor
	auth, err := bind.NewKeyedTransactorWithChainID(key, a.chainID)
	if err != nil {
		return fmt.Errorf("failed to create transactor: %w", err)
	}

	a.auth = auth
	a.address = auth.From

	return nil
}

// GetAddress returns the wallet address
func (a *AMM) GetAddress() common.Address {
	return a.address
}

// GetBalance returns the native token balance
func (a *AMM) GetBalance(ctx context.Context) (*big.Int, error) {
	return a.client.BalanceAt(ctx, a.address, nil)
}

// GetTokenInfo returns information about an ERC20 token
func (a *AMM) GetTokenInfo(ctx context.Context, tokenAddr common.Address) (*TokenInfo, error) {
	token := bind.NewBoundContract(tokenAddr, a.erc20ABI, a.client, a.client, a.client)

	var name, symbol string
	var decimals uint8
	var balance *big.Int

	// Get name
	var nameResult []interface{}
	if err := token.Call(&bind.CallOpts{Context: ctx}, &nameResult, "name"); err == nil && len(nameResult) > 0 {
		name = nameResult[0].(string)
	}

	// Get symbol
	var symbolResult []interface{}
	if err := token.Call(&bind.CallOpts{Context: ctx}, &symbolResult, "symbol"); err == nil && len(symbolResult) > 0 {
		symbol = symbolResult[0].(string)
	}

	// Get decimals
	var decimalsResult []interface{}
	if err := token.Call(&bind.CallOpts{Context: ctx}, &decimalsResult, "decimals"); err == nil && len(decimalsResult) > 0 {
		decimals = decimalsResult[0].(uint8)
	}

	// Get balance
	var balanceResult []interface{}
	if err := token.Call(&bind.CallOpts{Context: ctx}, &balanceResult, "balanceOf", a.address); err == nil && len(balanceResult) > 0 {
		balance = balanceResult[0].(*big.Int)
	}

	return &TokenInfo{
		Address:  tokenAddr,
		Name:     name,
		Symbol:   symbol,
		Decimals: decimals,
		Balance:  balance,
	}, nil
}

// GetPair returns the pair address for two tokens
func (a *AMM) GetPair(ctx context.Context, token0, token1 common.Address) (common.Address, error) {
	var result []interface{}
	if err := a.v2Factory.Call(&bind.CallOpts{Context: ctx}, &result, "getPair", token0, token1); err != nil {
		return common.Address{}, err
	}
	if len(result) == 0 {
		return common.Address{}, fmt.Errorf("no pair found")
	}
	return result[0].(common.Address), nil
}

// GetPoolCount returns the total number of pools
func (a *AMM) GetPoolCount(ctx context.Context) (uint64, error) {
	var result []interface{}
	if err := a.v2Factory.Call(&bind.CallOpts{Context: ctx}, &result, "allPairsLength"); err != nil {
		return 0, err
	}
	if len(result) == 0 {
		return 0, nil
	}
	return result[0].(*big.Int).Uint64(), nil
}

// GetAmountsOut returns expected output amounts for a swap path
func (a *AMM) GetAmountsOut(ctx context.Context, amountIn *big.Int, path []common.Address) ([]*big.Int, error) {
	var result []interface{}
	if err := a.v2Router.Call(&bind.CallOpts{Context: ctx}, &result, "getAmountsOut", amountIn, path); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no amounts returned")
	}
	return result[0].([]*big.Int), nil
}

// ApproveToken approves a token for spending by the router
func (a *AMM) ApproveToken(ctx context.Context, tokenAddr common.Address, amount *big.Int) (*types.Transaction, error) {
	if a.auth == nil {
		return nil, fmt.Errorf("wallet not loaded")
	}

	token := bind.NewBoundContract(tokenAddr, a.erc20ABI, a.client, a.client, a.client)

	// Set gas price
	gasPrice, err := a.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}
	a.auth.GasPrice = gasPrice

	tx, err := token.Transact(a.auth, "approve", a.config.V2Router, amount)
	if err != nil {
		return nil, fmt.Errorf("failed to approve: %w", err)
	}

	return tx, nil
}

// SwapExactTokensForTokens executes a token-to-token swap
func (a *AMM) SwapExactTokensForTokens(ctx context.Context, amountIn, amountOutMin *big.Int, path []common.Address, deadline time.Time) (*types.Transaction, error) {
	if a.auth == nil {
		return nil, fmt.Errorf("wallet not loaded")
	}

	// Set gas price
	gasPrice, err := a.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}
	a.auth.GasPrice = gasPrice
	a.auth.GasLimit = 300000 // Set reasonable gas limit for swap

	tx, err := a.v2Router.Transact(a.auth, "swapExactTokensForTokens",
		amountIn,
		amountOutMin,
		path,
		a.address,
		big.NewInt(deadline.Unix()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to swap: %w", err)
	}

	return tx, nil
}

// WaitForTx waits for a transaction to be mined
func (a *AMM) WaitForTx(ctx context.Context, tx *types.Transaction) (*types.Receipt, error) {
	return bind.WaitMined(ctx, a.client, tx)
}

// GetAllowance returns the allowance of a token for the router
func (a *AMM) GetAllowance(ctx context.Context, tokenAddr common.Address) (*big.Int, error) {
	token := bind.NewBoundContract(tokenAddr, a.erc20ABI, a.client, a.client, a.client)

	var result []interface{}
	if err := token.Call(&bind.CallOpts{Context: ctx}, &result, "allowance", a.address, a.config.V2Router); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return big.NewInt(0), nil
	}
	return result[0].(*big.Int), nil
}

// Close closes the client connection
func (a *AMM) Close() {
	if a.client != nil {
		a.client.Close()
	}
}

// Common V3 fee tiers
var V3FeeTiers = []uint32{100, 500, 3000, 10000}

// GetV3Pool returns the V3 pool address for a token pair and fee tier
func (a *AMM) GetV3Pool(ctx context.Context, token0, token1 common.Address, fee uint32) (common.Address, error) {
	var result []interface{}
	if err := a.v3Factory.Call(&bind.CallOpts{Context: ctx}, &result, "getPool", token0, token1, big.NewInt(int64(fee))); err != nil {
		return common.Address{}, err
	}
	if len(result) == 0 {
		return common.Address{}, fmt.Errorf("no pool found")
	}
	return result[0].(common.Address), nil
}

// GetV3Quote returns expected output for a V3 swap
func (a *AMM) GetV3Quote(ctx context.Context, tokenIn, tokenOut common.Address, fee uint32, amountIn *big.Int) (*big.Int, error) {
	var result []interface{}
	if err := a.quoter.Call(&bind.CallOpts{Context: ctx}, &result, "quoteExactInputSingle",
		tokenIn, tokenOut, big.NewInt(int64(fee)), amountIn, big.NewInt(0)); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no quote returned")
	}
	return result[0].(*big.Int), nil
}

// FindBestV3Pool finds the best V3 pool (highest liquidity) for a token pair
func (a *AMM) FindBestV3Pool(ctx context.Context, tokenIn, tokenOut common.Address, amountIn *big.Int) (uint32, *big.Int, error) {
	var bestFee uint32
	var bestAmount *big.Int

	for _, fee := range V3FeeTiers {
		pool, err := a.GetV3Pool(ctx, tokenIn, tokenOut, fee)
		if err != nil || pool == (common.Address{}) {
			continue
		}

		amount, err := a.GetV3Quote(ctx, tokenIn, tokenOut, fee, amountIn)
		if err != nil {
			continue
		}

		if bestAmount == nil || amount.Cmp(bestAmount) > 0 {
			bestFee = fee
			bestAmount = amount
		}
	}

	if bestAmount == nil {
		return 0, nil, fmt.Errorf("no V3 pool found")
	}

	return bestFee, bestAmount, nil
}

// ApproveTokenForV3 approves a token for V3 router
func (a *AMM) ApproveTokenForV3(ctx context.Context, tokenAddr common.Address, amount *big.Int) (*types.Transaction, error) {
	if a.auth == nil {
		return nil, fmt.Errorf("wallet not loaded")
	}

	token := bind.NewBoundContract(tokenAddr, a.erc20ABI, a.client, a.client, a.client)

	// Set gas price
	gasPrice, err := a.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}
	a.auth.GasPrice = gasPrice

	tx, err := token.Transact(a.auth, "approve", a.config.V3Router, amount)
	if err != nil {
		return nil, fmt.Errorf("failed to approve: %w", err)
	}

	return tx, nil
}

// GetV3Allowance returns the allowance of a token for the V3 router
func (a *AMM) GetV3Allowance(ctx context.Context, tokenAddr common.Address) (*big.Int, error) {
	token := bind.NewBoundContract(tokenAddr, a.erc20ABI, a.client, a.client, a.client)

	var result []interface{}
	if err := token.Call(&bind.CallOpts{Context: ctx}, &result, "allowance", a.address, a.config.V3Router); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return big.NewInt(0), nil
	}
	return result[0].(*big.Int), nil
}

// SwapExactInputSingleV3 executes a V3 single-hop swap
func (a *AMM) SwapExactInputSingleV3(ctx context.Context, tokenIn, tokenOut common.Address, fee uint32, amountIn, amountOutMin *big.Int, deadline time.Time) (*types.Transaction, error) {
	if a.auth == nil {
		return nil, fmt.Errorf("wallet not loaded")
	}

	// Set gas price
	gasPrice, err := a.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}
	a.auth.GasPrice = gasPrice
	a.auth.GasLimit = 500000 // V3 swaps need more gas

	// Build ExactInputSingleParams struct
	// struct ExactInputSingleParams {
	//     address tokenIn;
	//     address tokenOut;
	//     uint24 fee;
	//     address recipient;
	//     uint256 deadline;
	//     uint256 amountIn;
	//     uint256 amountOutMinimum;
	//     uint160 sqrtPriceLimitX96;
	// }
	params := struct {
		TokenIn           common.Address
		TokenOut          common.Address
		Fee               *big.Int
		Recipient         common.Address
		Deadline          *big.Int
		AmountIn          *big.Int
		AmountOutMinimum  *big.Int
		SqrtPriceLimitX96 *big.Int
	}{
		TokenIn:           tokenIn,
		TokenOut:          tokenOut,
		Fee:               big.NewInt(int64(fee)),
		Recipient:         a.address,
		Deadline:          big.NewInt(deadline.Unix()),
		AmountIn:          amountIn,
		AmountOutMinimum:  amountOutMin,
		SqrtPriceLimitX96: big.NewInt(0),
	}

	tx, err := a.v3Router.Transact(a.auth, "exactInputSingle", params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute V3 swap: %w", err)
	}

	return tx, nil
}

// deriveKey derives a private key from seed using BIP44 path m/44'/60'/0'/0/0
func deriveKey(seed []byte, _ string) (*ecdsa.PrivateKey, error) {
	// Create master key from seed using btcsuite hdkeychain
	masterKey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create master key: %w", err)
	}

	// BIP44 path: m/44'/60'/0'/0/0 for Ethereum
	// 44' = purpose (BIP44)
	// 60' = coin type (Ethereum)
	// 0' = account
	// 0 = change (external)
	// 0 = address index

	// Derive m/44'
	purpose, err := masterKey.Derive(hdkeychain.HardenedKeyStart + 44)
	if err != nil {
		return nil, fmt.Errorf("failed to derive purpose: %w", err)
	}

	// Derive m/44'/60'
	coinType, err := purpose.Derive(hdkeychain.HardenedKeyStart + 60)
	if err != nil {
		return nil, fmt.Errorf("failed to derive coin type: %w", err)
	}

	// Derive m/44'/60'/0'
	account, err := coinType.Derive(hdkeychain.HardenedKeyStart + 0)
	if err != nil {
		return nil, fmt.Errorf("failed to derive account: %w", err)
	}

	// Derive m/44'/60'/0'/0
	change, err := account.Derive(0)
	if err != nil {
		return nil, fmt.Errorf("failed to derive change: %w", err)
	}

	// Derive m/44'/60'/0'/0/0
	addressKey, err := change.Derive(0)
	if err != nil {
		return nil, fmt.Errorf("failed to derive address key: %w", err)
	}

	// Get the EC private key
	ecPrivKey, err := addressKey.ECPrivKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get EC private key: %w", err)
	}

	// Convert to ECDSA private key
	return ecPrivKey.ToECDSA(), nil
}
