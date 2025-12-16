// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package ammcmd

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/geth/common"
	"github.com/spf13/cobra"
)

var app *application.Lux

// Global flags
var (
	networkFlag   string
	rpcFlag       string
	privateKeyFlag string
)

// NewCmd creates a new amm command
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	app = injectedApp
	cmd := &cobra.Command{
		Use:   "amm",
		Short: "Trade on Lux/Zoo AMM (Uniswap V2/V3)",
		Long: `Commands for trading on Lux Exchange AMM pools.

Supported networks:
  - lux (Lux Mainnet C-Chain, chain ID 96369)
  - zoo (Zoo Mainnet, chain ID 200200)
  - lux-testnet (Lux Testnet, chain ID 96368)

Wallet access via:
  - LUX_MNEMONIC environment variable (BIP39 mnemonic)
  - LUX_PRIVATE_KEY environment variable (hex private key)
  - --private-key flag (hex private key)

Example usage:
  lux amm balance --network zoo
  lux amm swap --network zoo --from LUX --to USDT --amount 100
  lux amm pools --network zoo
  lux amm quote --network zoo --from LUX --to USDT --amount 100
  lux amm balance --network zoo --private-key 0x...`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Global flags
	cmd.PersistentFlags().StringVar(&networkFlag, "network", "zoo", "Network: lux, zoo, or lux-testnet")
	cmd.PersistentFlags().StringVar(&rpcFlag, "rpc", "", "Custom RPC endpoint (overrides network default)")
	cmd.PersistentFlags().StringVar(&privateKeyFlag, "private-key", "", "Private key (hex) for wallet access")

	// Add subcommands
	cmd.AddCommand(newBalanceCmd())
	cmd.AddCommand(newSwapCmd())
	cmd.AddCommand(newQuoteCmd())
	cmd.AddCommand(newPoolsCmd())
	cmd.AddCommand(newTokensCmd())
	cmd.AddCommand(newStatusCmd())

	return cmd
}

// getAMM creates an AMM client based on flags
func getAMM() (*AMM, error) {
	config := GetNetwork(networkFlag)
	if config == nil {
		return nil, fmt.Errorf("unknown network: %s", networkFlag)
	}

	// Override RPC if specified
	if rpcFlag != "" {
		config.RPC = rpcFlag
	}

	return NewAMM(config)
}

// newBalanceCmd creates the balance subcommand
func newBalanceCmd() *cobra.Command {
	var tokenAddr string

	cmd := &cobra.Command{
		Use:   "balance",
		Short: "Show wallet balance",
		Long: `Display native token and ERC20 token balances.

Examples:
  lux amm balance --network zoo
  lux amm balance --network zoo --token 0x...`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			amm, err := getAMM()
			if err != nil {
				return err
			}
			defer amm.Close()

			if err := amm.LoadWalletWithKey(privateKeyFlag); err != nil {
				return err
			}

			ux.Logger.PrintToUser("Wallet: %s", amm.GetAddress().Hex())
			ux.Logger.PrintToUser("Network: %s (Chain ID: %d)", amm.config.Name, amm.config.ChainID)
			ux.Logger.PrintToUser("")

			// Get native balance
			balance, err := amm.GetBalance(ctx)
			if err != nil {
				return fmt.Errorf("failed to get balance: %w", err)
			}

			// Convert to LUX (18 decimals)
			luxBalance := new(big.Float).SetInt(balance)
			luxBalance.Quo(luxBalance, big.NewFloat(1e18))
			ux.Logger.PrintToUser("Native Balance: %s LUX", luxBalance.Text('f', 6))

			// Get token balance if specified
			if tokenAddr != "" {
				addr := common.HexToAddress(tokenAddr)
				info, err := amm.GetTokenInfo(ctx, addr)
				if err != nil {
					return fmt.Errorf("failed to get token info: %w", err)
				}

				if info.Balance != nil {
					divisor := new(big.Float).SetFloat64(1)
					for i := uint8(0); i < info.Decimals; i++ {
						divisor.Mul(divisor, big.NewFloat(10))
					}
					tokenBal := new(big.Float).SetInt(info.Balance)
					tokenBal.Quo(tokenBal, divisor)
					ux.Logger.PrintToUser("%s Balance: %s %s", info.Name, tokenBal.Text('f', 6), info.Symbol)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&tokenAddr, "token", "", "ERC20 token address to check")

	return cmd
}

// newSwapCmd creates the swap subcommand
func newSwapCmd() *cobra.Command {
	var (
		fromToken string
		toToken   string
		amount    float64
		slippage  float64
		dryRun    bool
		useV3     bool
	)

	cmd := &cobra.Command{
		Use:   "swap",
		Short: "Swap tokens on AMM",
		Long: `Swap tokens using Uniswap V2/V3 style AMM.
Tries V2 pools first, then V3 if no V2 pool exists.

Examples:
  lux amm swap --network zoo --from 0x... --to 0x... --amount 100
  lux amm swap --network zoo --from 0x... --to 0x... --amount 100 --slippage 1.0
  lux amm swap --network zoo --from 0x... --to 0x... --amount 100 --v3
  lux amm swap --network zoo --from 0x... --to 0x... --amount 100 --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if fromToken == "" || toToken == "" || amount == 0 {
				return fmt.Errorf("required flags: --from, --to, --amount")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			amm, err := getAMM()
			if err != nil {
				return err
			}
			defer amm.Close()

			if err := amm.LoadWalletWithKey(privateKeyFlag); err != nil {
				return err
			}

			fromAddr := common.HexToAddress(fromToken)
			toAddr := common.HexToAddress(toToken)

			// Get token info
			fromInfo, err := amm.GetTokenInfo(ctx, fromAddr)
			if err != nil {
				return fmt.Errorf("failed to get from token info: %w", err)
			}

			toInfo, err := amm.GetTokenInfo(ctx, toAddr)
			if err != nil {
				return fmt.Errorf("failed to get to token info: %w", err)
			}

			// Convert amount to wei
			amountWei := new(big.Float).SetFloat64(amount)
			multiplier := new(big.Float).SetFloat64(1)
			for i := uint8(0); i < fromInfo.Decimals; i++ {
				multiplier.Mul(multiplier, big.NewFloat(10))
			}
			amountWei.Mul(amountWei, multiplier)
			amountIn, _ := amountWei.Int(nil)

			ux.Logger.PrintToUser("Swap Details:")
			ux.Logger.PrintToUser("  From: %s (%s)", fromInfo.Symbol, fromAddr.Hex())
			ux.Logger.PrintToUser("  To: %s (%s)", toInfo.Symbol, toAddr.Hex())
			ux.Logger.PrintToUser("  Amount: %f %s", amount, fromInfo.Symbol)
			ux.Logger.PrintToUser("  Slippage: %.2f%%", slippage)
			ux.Logger.PrintToUser("")

			var amountOut *big.Int
			var isV3 bool
			var feeTier uint32

			// Try V2 first unless --v3 flag is set
			if !useV3 {
				path := []common.Address{fromAddr, toAddr}
				amounts, err := amm.GetAmountsOut(ctx, amountIn, path)
				if err == nil && len(amounts) >= 2 {
					amountOut = amounts[len(amounts)-1]
					ux.Logger.PrintToUser("Using V2 Pool")
				}
			}

			// Try V3 if V2 failed or --v3 flag is set
			if amountOut == nil {
				feeTier, amountOut, err = amm.FindBestV3Pool(ctx, fromAddr, toAddr, amountIn)
				if err != nil {
					return fmt.Errorf("no pool found for pair: %w", err)
				}
				isV3 = true
				ux.Logger.PrintToUser("Using V3 Pool (%.2f%% fee)", float64(feeTier)/10000)
			}

			divisor := new(big.Float).SetFloat64(1)
			for i := uint8(0); i < toInfo.Decimals; i++ {
				divisor.Mul(divisor, big.NewFloat(10))
			}
			outFloat := new(big.Float).SetInt(amountOut)
			outFloat.Quo(outFloat, divisor)

			ux.Logger.PrintToUser("Expected Output: %s %s", outFloat.Text('f', 6), toInfo.Symbol)

			if dryRun {
				ux.Logger.PrintToUser("")
				ux.Logger.PrintToUser("(dry-run mode - no transaction sent)")
				return nil
			}

			// Calculate minimum output with slippage
			slippageMul := 1.0 - (slippage / 100.0)
			amountOutMinFloat := new(big.Float).SetInt(amountOut)
			amountOutMinFloat.Mul(amountOutMinFloat, big.NewFloat(slippageMul))
			amountOutMin, _ := amountOutMinFloat.Int(nil)

			ux.Logger.PrintToUser("Min Output (with %.2f%% slippage): %s %s", slippage,
				new(big.Float).Quo(new(big.Float).SetInt(amountOutMin), divisor).Text('f', 6), toInfo.Symbol)
			ux.Logger.PrintToUser("")

			if isV3 {
				// V3 swap flow
				allowance, err := amm.GetV3Allowance(ctx, fromAddr)
				if err != nil {
					return fmt.Errorf("failed to get V3 allowance: %w", err)
				}

				if allowance.Cmp(amountIn) < 0 {
					ux.Logger.PrintToUser("Approving %s for V3 router...", fromInfo.Symbol)
					maxUint256 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
					approveTx, err := amm.ApproveTokenForV3(ctx, fromAddr, maxUint256)
					if err != nil {
						return fmt.Errorf("failed to approve for V3: %w", err)
					}
					ux.Logger.PrintToUser("Approval tx: %s", approveTx.Hash().Hex())

					receipt, err := amm.WaitForTx(ctx, approveTx)
					if err != nil {
						return fmt.Errorf("failed to wait for approval: %w", err)
					}
					if receipt.Status != 1 {
						return fmt.Errorf("approval transaction failed")
					}
					ux.Logger.PrintToUser("Approval confirmed!")
					ux.Logger.PrintToUser("")
				}

				ux.Logger.PrintToUser("Executing V3 swap...")
				deadline := time.Now().Add(20 * time.Minute)
				swapTx, err := amm.SwapExactInputSingleV3(ctx, fromAddr, toAddr, feeTier, amountIn, amountOutMin, deadline)
				if err != nil {
					return fmt.Errorf("failed to execute V3 swap: %w", err)
				}
				ux.Logger.PrintToUser("Swap tx: %s", swapTx.Hash().Hex())

				receipt, err := amm.WaitForTx(ctx, swapTx)
				if err != nil {
					return fmt.Errorf("failed to wait for swap: %w", err)
				}
				if receipt.Status != 1 {
					return fmt.Errorf("V3 swap transaction failed")
				}
				ux.Logger.PrintToUser("V3 Swap confirmed in block %d!", receipt.BlockNumber.Uint64())
				ux.Logger.PrintToUser("Gas used: %d", receipt.GasUsed)
			} else {
				// V2 swap flow
				allowance, err := amm.GetAllowance(ctx, fromAddr)
				if err != nil {
					return fmt.Errorf("failed to get allowance: %w", err)
				}

				if allowance.Cmp(amountIn) < 0 {
					ux.Logger.PrintToUser("Approving %s for router...", fromInfo.Symbol)
					maxUint256 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
					approveTx, err := amm.ApproveToken(ctx, fromAddr, maxUint256)
					if err != nil {
						return fmt.Errorf("failed to approve: %w", err)
					}
					ux.Logger.PrintToUser("Approval tx: %s", approveTx.Hash().Hex())

					receipt, err := amm.WaitForTx(ctx, approveTx)
					if err != nil {
						return fmt.Errorf("failed to wait for approval: %w", err)
					}
					if receipt.Status != 1 {
						return fmt.Errorf("approval transaction failed")
					}
					ux.Logger.PrintToUser("Approval confirmed!")
					ux.Logger.PrintToUser("")
				}

				ux.Logger.PrintToUser("Executing swap...")
				deadline := time.Now().Add(20 * time.Minute)
				path := []common.Address{fromAddr, toAddr}
				swapTx, err := amm.SwapExactTokensForTokens(ctx, amountIn, amountOutMin, path, deadline)
				if err != nil {
					return fmt.Errorf("failed to execute swap: %w", err)
				}
				ux.Logger.PrintToUser("Swap tx: %s", swapTx.Hash().Hex())

				receipt, err := amm.WaitForTx(ctx, swapTx)
				if err != nil {
					return fmt.Errorf("failed to wait for swap: %w", err)
				}
				if receipt.Status != 1 {
					return fmt.Errorf("swap transaction failed")
				}
				ux.Logger.PrintToUser("Swap confirmed in block %d!", receipt.BlockNumber.Uint64())
				ux.Logger.PrintToUser("Gas used: %d", receipt.GasUsed)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&fromToken, "from", "", "Token address to swap from")
	cmd.Flags().StringVar(&toToken, "to", "", "Token address to swap to")
	cmd.Flags().Float64Var(&amount, "amount", 0, "Amount to swap")
	cmd.Flags().Float64Var(&slippage, "slippage", 0.5, "Max slippage tolerance (%)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Only show quote, don't execute")
	cmd.Flags().BoolVar(&useV3, "v3", false, "Force V3 pool")

	return cmd
}

// newQuoteCmd creates the quote subcommand
func newQuoteCmd() *cobra.Command {
	var (
		fromToken string
		toToken   string
		amount    float64
		useV3     bool
	)

	cmd := &cobra.Command{
		Use:   "quote",
		Short: "Get swap quote",
		Long: `Get a quote for swapping tokens without executing.
Tries V2 pools first, then V3 if no V2 pool exists.

Examples:
  lux amm quote --network zoo --from 0x... --to 0x... --amount 100
  lux amm quote --network zoo --from 0x... --to 0x... --amount 100 --v3`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if fromToken == "" || toToken == "" || amount == 0 {
				return fmt.Errorf("required flags: --from, --to, --amount")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			amm, err := getAMM()
			if err != nil {
				return err
			}
			defer amm.Close()

			fromAddr := common.HexToAddress(fromToken)
			toAddr := common.HexToAddress(toToken)

			// Get token info
			fromInfo, err := amm.GetTokenInfo(ctx, fromAddr)
			if err != nil {
				fromInfo = &TokenInfo{Symbol: "TOKEN", Decimals: 18}
			}

			toInfo, err := amm.GetTokenInfo(ctx, toAddr)
			if err != nil {
				toInfo = &TokenInfo{Symbol: "TOKEN", Decimals: 18}
			}

			// Convert amount to wei
			amountWei := new(big.Float).SetFloat64(amount)
			multiplier := new(big.Float).SetFloat64(1)
			for i := uint8(0); i < fromInfo.Decimals; i++ {
				multiplier.Mul(multiplier, big.NewFloat(10))
			}
			amountWei.Mul(amountWei, multiplier)
			amountIn, _ := amountWei.Int(nil)

			var amountOut *big.Int
			var poolType string
			var feeTier uint32

			// Try V2 first unless --v3 flag is set
			if !useV3 {
				path := []common.Address{fromAddr, toAddr}
				amounts, err := amm.GetAmountsOut(ctx, amountIn, path)
				if err == nil && len(amounts) >= 2 {
					amountOut = amounts[len(amounts)-1]
					poolType = "V2"
				}
			}

			// Try V3 if V2 failed or --v3 flag is set
			if amountOut == nil {
				feeTier, amountOut, err = amm.FindBestV3Pool(ctx, fromAddr, toAddr, amountIn)
				if err != nil {
					return fmt.Errorf("no pool found for pair: %w", err)
				}
				poolType = fmt.Sprintf("V3 (%.2f%% fee)", float64(feeTier)/10000)
			}

			divisor := new(big.Float).SetFloat64(1)
			for i := uint8(0); i < toInfo.Decimals; i++ {
				divisor.Mul(divisor, big.NewFloat(10))
			}
			outFloat := new(big.Float).SetInt(amountOut)
			outFloat.Quo(outFloat, divisor)

			// Calculate price
			inFloat := new(big.Float).SetFloat64(amount)
			price := new(big.Float).Quo(outFloat, inFloat)

			ux.Logger.PrintToUser("Quote (%s):", poolType)
			ux.Logger.PrintToUser("  Input: %f %s", amount, fromInfo.Symbol)
			ux.Logger.PrintToUser("  Output: %s %s", outFloat.Text('f', 6), toInfo.Symbol)
			ux.Logger.PrintToUser("  Price: 1 %s = %s %s", fromInfo.Symbol, price.Text('f', 6), toInfo.Symbol)

			return nil
		},
	}

	cmd.Flags().StringVar(&fromToken, "from", "", "Token address to swap from")
	cmd.Flags().StringVar(&toToken, "to", "", "Token address to swap to")
	cmd.Flags().Float64Var(&amount, "amount", 0, "Amount to quote")
	cmd.Flags().BoolVar(&useV3, "v3", false, "Force V3 pool")

	return cmd
}

// newPoolsCmd creates the pools subcommand
func newPoolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pools",
		Short: "List liquidity pools",
		Long: `List all liquidity pools on the AMM.

Examples:
  lux amm pools --network zoo`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			amm, err := getAMM()
			if err != nil {
				return err
			}
			defer amm.Close()

			ux.Logger.PrintToUser("Network: %s (Chain ID: %d)", amm.config.Name, amm.config.ChainID)
			ux.Logger.PrintToUser("")

			count, err := amm.GetPoolCount(ctx)
			if err != nil {
				return fmt.Errorf("failed to get pool count: %w", err)
			}

			ux.Logger.PrintToUser("Total Pools: %d", count)
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("V2 Factory: %s", amm.config.V2Factory.Hex())
			ux.Logger.PrintToUser("V3 Factory: %s", amm.config.V3Factory.Hex())

			return nil
		},
	}

	return cmd
}

// newTokensCmd creates the tokens subcommand
func newTokensCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tokens [address...]",
		Short: "Get token information",
		Long: `Get information about ERC20 tokens.

Examples:
  lux amm tokens --network zoo 0x...`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("specify at least one token address")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			amm, err := getAMM()
			if err != nil {
				return err
			}
			defer amm.Close()

			ux.Logger.PrintToUser("Token Information:")
			ux.Logger.PrintToUser("")

			for _, addr := range args {
				tokenAddr := common.HexToAddress(addr)
				info, err := amm.GetTokenInfo(ctx, tokenAddr)
				if err != nil {
					ux.Logger.PrintToUser("  %s: error - %v", addr, err)
					continue
				}

				ux.Logger.PrintToUser("  %s (%s)", info.Name, info.Symbol)
				ux.Logger.PrintToUser("    Address: %s", info.Address.Hex())
				ux.Logger.PrintToUser("    Decimals: %d", info.Decimals)
				ux.Logger.PrintToUser("")
			}

			return nil
		},
	}

	return cmd
}

// newStatusCmd creates the status subcommand
func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show AMM status",
		Long: `Show AMM contract status and network info.

Examples:
  lux amm status --network zoo`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			amm, err := getAMM()
			if err != nil {
				return err
			}
			defer amm.Close()

			blockNum, err := amm.client.BlockNumber(ctx)
			if err != nil {
				return fmt.Errorf("failed to get block number: %w", err)
			}

			ux.Logger.PrintToUser("AMM Status:")
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("Network:")
			ux.Logger.PrintToUser("  Name: %s", amm.config.Name)
			ux.Logger.PrintToUser("  Chain ID: %d", amm.config.ChainID)
			ux.Logger.PrintToUser("  RPC: %s", amm.config.RPC)
			ux.Logger.PrintToUser("  Block: %d", blockNum)
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("Contracts:")
			ux.Logger.PrintToUser("  V2 Factory: %s", amm.config.V2Factory.Hex())
			ux.Logger.PrintToUser("  V2 Router: %s", amm.config.V2Router.Hex())
			ux.Logger.PrintToUser("  V3 Factory: %s", amm.config.V3Factory.Hex())
			ux.Logger.PrintToUser("  V3 Router: %s", amm.config.V3Router.Hex())
			ux.Logger.PrintToUser("  Multicall: %s", amm.config.Multicall.Hex())
			ux.Logger.PrintToUser("  Quoter: %s", amm.config.Quoter.Hex())

			// Get pool count
			count, err := amm.GetPoolCount(ctx)
			if err == nil {
				ux.Logger.PrintToUser("")
				ux.Logger.PrintToUser("Statistics:")
				ux.Logger.PrintToUser("  V2 Pools: %d", count)
			}

			return nil
		},
	}

	return cmd
}
