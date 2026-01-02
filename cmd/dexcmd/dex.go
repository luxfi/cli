// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package dexcmd

import (
	"github.com/luxfi/cli/pkg/application"
	"github.com/spf13/cobra"
)

// NewCmd creates a new dex command
func NewCmd(_ *application.Lux) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dex",
		Short: "Manage decentralized exchange operations",
		Long: `Commands for interacting with Lux DEX - a high-performance
decentralized exchange with spot trading, AMM pools, and perpetual futures.

Features:
  - Central Limit Order Book (CLOB) for spot trading
  - AMM pools (Constant Product, StableSwap, Concentrated Liquidity)
  - Perpetual futures with up to 100x leverage
  - Cross-chain swaps via Warp messaging
  - 1ms block times for ultra-low latency HFT

Example usage:
  lux dex market list              # List all markets
  lux dex order place              # Place an order
  lux dex pool create              # Create liquidity pool
  lux dex perp open                # Open perpetual position`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(newMarketCmd())
	cmd.AddCommand(newOrderCmd())
	cmd.AddCommand(newPoolCmd())
	cmd.AddCommand(newPerpCmd())
	cmd.AddCommand(newAccountCmd())
	cmd.AddCommand(newStatusCmd())

	return cmd
}

// newMarketCmd creates the market subcommand
func newMarketCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "market",
		Short: "Manage trading markets",
		Long:  "Commands for listing, creating, and managing trading markets",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all available markets",
		Long:  "Display all spot and perpetual markets with current prices and volume",
		RunE:  marketListCmd,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "info [symbol]",
		Short: "Get detailed market information",
		Long:  "Display detailed information about a specific market including orderbook depth, recent trades, and statistics",
		Args:  cobra.ExactArgs(1),
		RunE:  marketInfoCmd,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "create",
		Short: "Create a new market",
		Long:  "Create a new spot or perpetual market with specified parameters",
		RunE:  marketCreateCmd,
	})

	return cmd
}

// newOrderCmd creates the order subcommand
func newOrderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "order",
		Short: "Manage orders",
		Long:  "Commands for placing, cancelling, and viewing orders",
	}

	placeCmd := &cobra.Command{
		Use:   "place",
		Short: "Place a new order",
		Long: `Place a limit or market order on a trading pair.

Examples:
  lux dex order place --market LUX/USDT --side buy --type limit --price 10.50 --amount 100
  lux dex order place --market BTC/USDT --side sell --type market --amount 0.5`,
		RunE: orderPlaceCmd,
	}
	placeCmd.Flags().String("market", "", "Trading pair symbol (e.g., LUX/USDT)")
	placeCmd.Flags().String("side", "", "Order side: buy or sell")
	placeCmd.Flags().String("type", "limit", "Order type: limit or market")
	placeCmd.Flags().Float64("price", 0, "Limit price (required for limit orders)")
	placeCmd.Flags().Float64("amount", 0, "Order amount")
	placeCmd.Flags().String("tif", "gtc", "Time in force: gtc, ioc, fok")
	cmd.AddCommand(placeCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "cancel [order-id]",
		Short: "Cancel an order",
		Args:  cobra.ExactArgs(1),
		RunE:  orderCancelCmd,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List open orders",
		RunE:  orderListCmd,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "history",
		Short: "View order history",
		RunE:  orderHistoryCmd,
	})

	return cmd
}

// newPoolCmd creates the pool subcommand
func newPoolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pool",
		Short: "Manage liquidity pools",
		Long:  "Commands for creating, managing, and interacting with AMM liquidity pools",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all liquidity pools",
		RunE:  poolListCmd,
	})

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new liquidity pool",
		Long: `Create a new AMM liquidity pool.

Pool types:
  - constant-product: Standard x*y=k AMM (like Uniswap V2)
  - stableswap: Optimized for stable pairs (like Curve)
  - concentrated: Concentrated liquidity (like Uniswap V3)

Examples:
  lux dex pool create --token0 LUX --token1 USDT --amount0 1000 --amount1 10000 --type constant-product --fee 30`,
		RunE: poolCreateCmd,
	}
	createCmd.Flags().String("token0", "", "First token symbol")
	createCmd.Flags().String("token1", "", "Second token symbol")
	createCmd.Flags().Float64("amount0", 0, "Initial amount of token0")
	createCmd.Flags().Float64("amount1", 0, "Initial amount of token1")
	createCmd.Flags().String("type", "constant-product", "Pool type: constant-product, stableswap, concentrated")
	createCmd.Flags().Uint16("fee", 30, "Fee in basis points (30 = 0.3%)")
	cmd.AddCommand(createCmd)

	addCmd := &cobra.Command{
		Use:   "add [pool-id]",
		Short: "Add liquidity to a pool",
		Args:  cobra.ExactArgs(1),
		RunE:  poolAddLiquidityCmd,
	}
	addCmd.Flags().Float64("amount0", 0, "Amount of token0 to add")
	addCmd.Flags().Float64("amount1", 0, "Amount of token1 to add")
	cmd.AddCommand(addCmd)

	removeCmd := &cobra.Command{
		Use:   "remove [pool-id]",
		Short: "Remove liquidity from a pool",
		Args:  cobra.ExactArgs(1),
		RunE:  poolRemoveLiquidityCmd,
	}
	removeCmd.Flags().Float64("percent", 0, "Percentage of liquidity to remove (0-100)")
	cmd.AddCommand(removeCmd)

	swapCmd := &cobra.Command{
		Use:   "swap",
		Short: "Swap tokens using AMM pools",
		Long: `Swap tokens using the best available route through AMM pools.

Examples:
  lux dex pool swap --from LUX --to USDT --amount 100 --slippage 0.5`,
		RunE: poolSwapCmd,
	}
	swapCmd.Flags().String("from", "", "Token to swap from")
	swapCmd.Flags().String("to", "", "Token to swap to")
	swapCmd.Flags().Float64("amount", 0, "Amount to swap")
	swapCmd.Flags().Float64("slippage", 0.5, "Maximum slippage tolerance (%)")
	cmd.AddCommand(swapCmd)

	return cmd
}

// newPerpCmd creates the perpetuals subcommand
func newPerpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "perp",
		Aliases: []string{"perpetual", "futures"},
		Short:   "Manage perpetual futures positions",
		Long: `Commands for trading perpetual futures contracts.

Features:
  - Up to 100x leverage
  - Cross and isolated margin modes
  - Automatic liquidation protection
  - 8-hour funding rate intervals

Similar to Hyperliquid and GMX perpetual trading.`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "markets",
		Short: "List perpetual markets",
		RunE:  perpMarketsCmd,
	})

	openCmd := &cobra.Command{
		Use:   "open",
		Short: "Open a perpetual position",
		Long: `Open a new perpetual futures position.

Examples:
  lux dex perp open --market BTC-PERP --side long --size 0.1 --leverage 10
  lux dex perp open --market ETH-PERP --side short --size 1 --leverage 5 --margin isolated`,
		RunE: perpOpenCmd,
	}
	openCmd.Flags().String("market", "", "Perpetual market symbol (e.g., BTC-PERP)")
	openCmd.Flags().String("side", "", "Position side: long or short")
	openCmd.Flags().Float64("size", 0, "Position size in base units")
	openCmd.Flags().Uint16("leverage", 10, "Leverage multiplier (1-100)")
	openCmd.Flags().String("margin", "cross", "Margin mode: cross or isolated")
	cmd.AddCommand(openCmd)

	closeCmd := &cobra.Command{
		Use:   "close [market]",
		Short: "Close a perpetual position",
		Args:  cobra.ExactArgs(1),
		RunE:  perpCloseCmd,
	}
	closeCmd.Flags().Float64("percent", 100, "Percentage of position to close (0-100)")
	cmd.AddCommand(closeCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "positions",
		Short: "List open positions",
		RunE:  perpPositionsCmd,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "pnl",
		Short: "View profit/loss summary",
		RunE:  perpPnLCmd,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "funding",
		Short: "View funding rate information",
		RunE:  perpFundingCmd,
	})

	return cmd
}

// newAccountCmd creates the account subcommand
func newAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "Manage trading account",
		Long:  "Commands for managing your DEX trading account, deposits, and withdrawals",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "balance",
		Short: "View account balances",
		RunE:  accountBalanceCmd,
	})

	depositCmd := &cobra.Command{
		Use:   "deposit",
		Short: "Deposit funds to trading account",
		RunE:  accountDepositCmd,
	}
	depositCmd.Flags().String("token", "", "Token to deposit")
	depositCmd.Flags().Float64("amount", 0, "Amount to deposit")
	cmd.AddCommand(depositCmd)

	withdrawCmd := &cobra.Command{
		Use:   "withdraw",
		Short: "Withdraw funds from trading account",
		RunE:  accountWithdrawCmd,
	}
	withdrawCmd.Flags().String("token", "", "Token to withdraw")
	withdrawCmd.Flags().Float64("amount", 0, "Amount to withdraw")
	cmd.AddCommand(withdrawCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "history",
		Short: "View transaction history",
		RunE:  accountHistoryCmd,
	})

	return cmd
}

// newStatusCmd creates the status subcommand
func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show DEX status and statistics",
		Long: `Display DEX network status including:
  - Connected nodes
  - Market statistics
  - Recent trades
  - Network health`,
		RunE: statusCmd,
	}
}
