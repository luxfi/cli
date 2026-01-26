// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package mpccmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luxfi/cli/pkg/mpc"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	// Node flags
	nodeThreshold  int
	nodeTotalNodes int
	nodeNetwork    string
	nodeForce      bool
)

// NewCmd creates the mpc command.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mpc",
		Short: "Manage MPC nodes and wallets",
		Long: `Multi-Party Computation (MPC) management commands.

MPC enables threshold signing for blockchain wallets, where multiple
parties must cooperate to sign transactions without any single party
having access to the complete private key.

Each MPC node holds exactly one key shard. For a t-of-n threshold scheme,
at least t nodes must cooperate to produce a valid signature.

NETWORK TYPES:

  --mainnet   Production MPC network (ports 9700-9799)
  --testnet   Test MPC network (ports 9710-9809)
  --devnet    Development MPC network (ports 9720-9819)

QUICK START:

  # Initialize a 3-of-5 MPC network
  lux mpc node init --threshold 3 --nodes 5 --devnet

  # Start all MPC nodes
  lux mpc node start

  # Check status
  lux mpc node status

  # Create a wallet
  lux mpc wallet create --name "Treasury"

  # Stop the network
  lux mpc node stop

SECURITY:

  Key shards are stored encrypted in ~/.lux/keys/mpc/
  Backups are stored in ~/.lux/mpc/backups/ by default

CLOUD DEPLOYMENT:

  Deploy MPC nodes to cloud providers:
  lux mpc deploy create mpc-devnet-xxx --provider aws --region us-east-1
  lux mpc deploy status mpc-devnet-xxx
  lux mpc deploy ssh mpc-devnet-xxx mpc-node-1

Available subcommands:
  node     - Manage MPC nodes (local)
  deploy   - Deploy MPC nodes to cloud
  backup   - Backup and restore node data
  wallet   - Manage MPC wallets
  sign     - Threshold signing operations`,
	}

	cmd.AddCommand(newNodeCmd())
	cmd.AddCommand(newDeployCmd())
	cmd.AddCommand(newBackupCmd())
	cmd.AddCommand(newWalletCmd())
	cmd.AddCommand(newSignCmd())

	return cmd
}

// newNodeCmd creates the node management command group.
func newNodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Manage MPC nodes",
		Long: `Commands for managing MPC node lifecycle.

MPC nodes form a threshold signing network. Each node holds one key shard
and participates in distributed signing operations.

Examples:
  # Initialize a new 3-of-5 MPC network
  lux mpc node init --threshold 3 --nodes 5 --devnet

  # Start all nodes in the network
  lux mpc node start

  # Check status of all nodes
  lux mpc node status

  # Stop all nodes
  lux mpc node stop

  # Clean up (stop and remove data)
  lux mpc node clean`,
	}

	cmd.AddCommand(newNodeInitCmd())
	cmd.AddCommand(newNodeStartCmd())
	cmd.AddCommand(newNodeStopCmd())
	cmd.AddCommand(newNodeStatusCmd())
	cmd.AddCommand(newNodeCleanCmd())
	cmd.AddCommand(newNodeListCmd())

	return cmd
}

func newNodeInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new MPC network",
		Long: `Initialize a new MPC network with the specified threshold configuration.

This creates the network directory structure, generates node configurations,
and prepares encrypted key storage directories.

Examples:
  # Create a 2-of-3 devnet MPC network
  lux mpc node init --threshold 2 --nodes 3 --devnet

  # Create a 3-of-5 mainnet MPC network
  lux mpc node init --threshold 3 --nodes 5 --mainnet`,
		RunE: runNodeInit,
	}

	cmd.Flags().IntVarP(&nodeThreshold, "threshold", "t", 2, "Signing threshold (t in t-of-n)")
	cmd.Flags().IntVarP(&nodeTotalNodes, "nodes", "n", 3, "Total number of nodes")
	cmd.Flags().BoolVar(&nodeMainnet, "mainnet", false, "Initialize mainnet MPC network")
	cmd.Flags().BoolVar(&nodeTestnet, "testnet", false, "Initialize testnet MPC network")
	cmd.Flags().BoolVar(&nodeDevnet, "devnet", false, "Initialize devnet MPC network (default)")

	return cmd
}

var (
	nodeMainnet bool
	nodeTestnet bool
	nodeDevnet  bool
)

func newNodeStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start [network-name]",
		Short: "Start MPC nodes",
		Long: `Start all MPC nodes in a network.

If no network name is specified, starts the most recently created network.

Examples:
  # Start all nodes in the default network
  lux mpc node start

  # Start a specific network
  lux mpc node start mpc-devnet-abc123`,
		RunE: runNodeStart,
	}

	cmd.Flags().BoolVar(&nodeMainnet, "mainnet", false, "Start mainnet MPC network")
	cmd.Flags().BoolVar(&nodeTestnet, "testnet", false, "Start testnet MPC network")
	cmd.Flags().BoolVar(&nodeDevnet, "devnet", false, "Start devnet MPC network")

	return cmd
}

func newNodeStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop [network-name]",
		Short: "Stop MPC nodes",
		Long: `Stop all MPC nodes in a network.

This gracefully shuts down nodes and saves state for later restart.

Examples:
  # Stop the default network
  lux mpc node stop

  # Stop a specific network
  lux mpc node stop mpc-devnet-abc123`,
		RunE: runNodeStop,
	}

	return cmd
}

func newNodeStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [network-name]",
		Short: "Check MPC node status",
		Long: `Display status of all MPC nodes in a network.

Shows running status, uptime, endpoints, and health information.

Examples:
  # Show status of default network
  lux mpc node status

  # Show status of specific network
  lux mpc node status mpc-devnet-abc123`,
		RunE: runNodeStatus,
	}

	return cmd
}

func newNodeCleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean [network-name]",
		Short: "Stop and remove MPC network",
		Long: `Stop all nodes and remove network data.

WARNING: This will delete all node data including key shards!
Make sure you have backups before running this command.

Examples:
  # Clean the default network
  lux mpc node clean

  # Force clean without confirmation
  lux mpc node clean --force`,
		RunE: runNodeClean,
	}

	cmd.Flags().BoolVarP(&nodeForce, "force", "f", false, "Skip confirmation")

	return cmd
}

func newNodeListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List MPC networks",
		Long:  `List all initialized MPC networks.`,
		RunE:  runNodeList,
	}
}

// Command implementations

func runNodeInit(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Determine network type
	networkType := "devnet"
	if nodeMainnet {
		networkType = "mainnet"
	} else if nodeTestnet {
		networkType = "testnet"
	}

	// Validate threshold
	if nodeThreshold < 1 || nodeThreshold > nodeTotalNodes {
		return fmt.Errorf("threshold must be between 1 and %d", nodeTotalNodes)
	}

	mgr := getNodeManager()

	ux.Logger.PrintToUser("Initializing %d-of-%d MPC network (%s)...", nodeThreshold, nodeTotalNodes, networkType)

	networkCfg, err := mgr.InitNetwork(ctx, networkType, nodeThreshold, nodeTotalNodes)
	if err != nil {
		return fmt.Errorf("failed to initialize network: %w", err)
	}

	ux.Logger.PrintToUser("\nMPC network initialized successfully!")
	ux.Logger.PrintToUser("  Network:   %s", networkCfg.NetworkName)
	ux.Logger.PrintToUser("  Type:      %s", networkCfg.NetworkType)
	ux.Logger.PrintToUser("  Threshold: %d-of-%d", networkCfg.Threshold, networkCfg.TotalNodes)
	ux.Logger.PrintToUser("  Nodes:     %d", len(networkCfg.Nodes))
	ux.Logger.PrintToUser("  Data:      %s", networkCfg.BaseDir)
	ux.Logger.PrintToUser("\nTo start the network, run:")
	ux.Logger.PrintToUser("  lux mpc node start %s", networkCfg.NetworkName)

	return nil
}

func runNodeStart(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	mgr := getNodeManager()

	// Determine which network to start
	networkName := ""
	if len(args) > 0 {
		networkName = args[0]
	} else {
		// Find network by type or use most recent
		networkName = findNetworkByType(mgr)
	}

	if networkName == "" {
		return fmt.Errorf("no MPC network found. Run 'lux mpc node init' first")
	}

	ux.Logger.PrintToUser("Starting MPC network %s...", networkName)

	if err := mgr.StartNetwork(ctx, networkName); err != nil {
		return err
	}

	// Show status
	infos, _ := mgr.GetNetworkStatus(networkName)
	ux.Logger.PrintToUser("\nMPC network started!")
	ux.Logger.PrintToUser("\n%-15s  %-10s  %-25s  %-10s", "NODE", "STATUS", "ENDPOINT", "PID")
	ux.Logger.PrintToUser("%s", strings.Repeat("-", 65))
	for _, info := range infos {
		status := string(info.Status)
		if info.Status == mpc.NodeStatusRunning {
			status = "✓ running"
		}
		ux.Logger.PrintToUser("%-15s  %-10s  %-25s  %-10d",
			info.Config.NodeName, status, info.Endpoint, info.PID)
	}

	return nil
}

func runNodeStop(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	mgr := getNodeManager()

	networkName := ""
	if len(args) > 0 {
		networkName = args[0]
	} else {
		networkName = findNetworkByType(mgr)
	}

	if networkName == "" {
		return fmt.Errorf("no MPC network found")
	}

	ux.Logger.PrintToUser("Stopping MPC network %s...", networkName)

	if err := mgr.StopNetwork(ctx, networkName); err != nil {
		return err
	}

	ux.Logger.PrintToUser("MPC network stopped")
	return nil
}

func runNodeStatus(cmd *cobra.Command, args []string) error {
	mgr := getNodeManager()

	networkName := ""
	if len(args) > 0 {
		networkName = args[0]
	} else {
		networkName = findNetworkByType(mgr)
	}

	if networkName == "" {
		return fmt.Errorf("no MPC network found")
	}

	networkCfg, err := mgr.LoadNetworkConfig(networkName)
	if err != nil {
		return err
	}

	infos, err := mgr.GetNetworkStatus(networkName)
	if err != nil {
		return err
	}

	runningCount := 0
	for _, info := range infos {
		if info.Status == mpc.NodeStatusRunning {
			runningCount++
		}
	}

	ux.Logger.PrintToUser("MPC Network: %s", networkName)
	ux.Logger.PrintToUser("Type:        %s", networkCfg.NetworkType)
	ux.Logger.PrintToUser("Threshold:   %d-of-%d", networkCfg.Threshold, networkCfg.TotalNodes)
	ux.Logger.PrintToUser("Nodes:       %d/%d running", runningCount, len(infos))
	ux.Logger.PrintToUser("")

	ux.Logger.PrintToUser("%-15s  %-10s  %-25s  %-8s  %-15s", "NODE", "STATUS", "ENDPOINT", "PID", "UPTIME")
	ux.Logger.PrintToUser("%s", strings.Repeat("-", 80))

	for _, info := range infos {
		status := string(info.Status)
		if info.Status == mpc.NodeStatusRunning {
			status = "✓ running"
		} else if info.Status == mpc.NodeStatusStopped {
			status = "○ stopped"
		} else if info.Status == mpc.NodeStatusError {
			status = "✗ error"
		}

		pid := ""
		if info.PID > 0 {
			pid = fmt.Sprintf("%d", info.PID)
		}

		ux.Logger.PrintToUser("%-15s  %-10s  %-25s  %-8s  %-15s",
			info.Config.NodeName, status, info.Endpoint, pid, info.Uptime)
	}

	return nil
}

func runNodeClean(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	mgr := getNodeManager()

	networkName := ""
	if len(args) > 0 {
		networkName = args[0]
	} else {
		networkName = findNetworkByType(mgr)
	}

	if networkName == "" {
		return fmt.Errorf("no MPC network found")
	}

	if !nodeForce {
		ux.Logger.PrintToUser("WARNING: This will delete all data for network %s", networkName)
		ux.Logger.PrintToUser("Use --force to skip this confirmation")
		return nil
	}

	ux.Logger.PrintToUser("Cleaning up MPC network %s...", networkName)

	if err := mgr.DeleteNetwork(ctx, networkName, true); err != nil {
		return err
	}

	ux.Logger.PrintToUser("MPC network removed")
	return nil
}

func runNodeList(cmd *cobra.Command, args []string) error {
	mgr := getNodeManager()

	networks, err := mgr.ListNetworks()
	if err != nil {
		return err
	}

	if len(networks) == 0 {
		ux.Logger.PrintToUser("No MPC networks found")
		ux.Logger.PrintToUser("\nTo create one, run:")
		ux.Logger.PrintToUser("  lux mpc node init --threshold 2 --nodes 3 --devnet")
		return nil
	}

	ux.Logger.PrintToUser("%-25s  %-10s  %-12s  %-8s  %-20s", "NETWORK", "TYPE", "THRESHOLD", "NODES", "CREATED")
	ux.Logger.PrintToUser("%s", strings.Repeat("-", 80))

	for _, net := range networks {
		ux.Logger.PrintToUser("%-25s  %-10s  %-12s  %-8d  %-20s",
			net.NetworkName,
			net.NetworkType,
			fmt.Sprintf("%d-of-%d", net.Threshold, net.TotalNodes),
			len(net.Nodes),
			net.Created.Format("2006-01-02 15:04"),
		)
	}

	return nil
}

// newWalletCmd creates the wallet management command group.
func newWalletCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wallet",
		Short: "Manage MPC wallets",
		Long: `Commands for managing MPC wallets and their key shares.

Examples:
  # List wallets
  lux mpc wallet list

  # Create a new wallet
  lux mpc wallet create --name "Treasury" --threshold 2 --parties 3

  # Show wallet details
  lux mpc wallet show <wallet-id>`,
	}

	cmd.AddCommand(newWalletListCmd())
	cmd.AddCommand(newWalletCreateCmd())
	cmd.AddCommand(newWalletShowCmd())
	cmd.AddCommand(newWalletExportCmd())

	return cmd
}

func newWalletListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List wallets",
		RunE: func(cmd *cobra.Command, args []string) error {
			ux.Logger.PrintToUser("No wallets found. Create one with 'lux mpc wallet create'")
			return nil
		},
	}
}

func newWalletCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "Create a new wallet",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement wallet creation with DKG
			ux.Logger.PrintToUser("Wallet creation requires running MPC nodes")
			ux.Logger.PrintToUser("Start nodes first with 'lux mpc node start'")
			return nil
		},
	}
}

func newWalletShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <wallet-id>",
		Short: "Show wallet details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("wallet not found: %s", args[0])
		},
	}
}

func newWalletExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export <wallet-id>",
		Short: "Export wallet public key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("wallet not found: %s", args[0])
		},
	}
}

// newSignCmd creates the signing command group.
func newSignCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sign",
		Short: "Threshold signing operations",
		Long: `Commands for threshold signing operations.

Examples:
  # Initiate a signing request
  lux mpc sign request --wallet <wallet-id> --message "0x..."

  # Approve a signing request
  lux mpc sign approve <request-id>

  # Check signing status
  lux mpc sign status <request-id>`,
	}

	cmd.AddCommand(newSignRequestCmd())
	cmd.AddCommand(newSignApproveCmd())
	cmd.AddCommand(newSignStatusCmd())

	return cmd
}

func newSignRequestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "request",
		Short: "Initiate a signing request",
		RunE: func(cmd *cobra.Command, args []string) error {
			ux.Logger.PrintToUser("Signing requires running MPC nodes and an existing wallet")
			return nil
		},
	}
}

func newSignApproveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "approve <request-id>",
		Short: "Approve a signing request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("signing request not found: %s", args[0])
		},
	}
}

func newSignStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <request-id>",
		Short: "Check signing status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("signing request not found: %s", args[0])
		},
	}
}

// Helper functions

func getNodeManager() *mpc.NodeManager {
	homeDir, _ := os.UserHomeDir()
	baseDir := filepath.Join(homeDir, ".lux", "mpc")
	os.MkdirAll(baseDir, 0750)
	return mpc.NewNodeManager(baseDir)
}

func findNetworkByType(mgr *mpc.NodeManager) string {
	networks, err := mgr.ListNetworks()
	if err != nil || len(networks) == 0 {
		return ""
	}

	// Find by type if flags set
	targetType := ""
	if nodeMainnet {
		targetType = "mainnet"
	} else if nodeTestnet {
		targetType = "testnet"
	} else if nodeDevnet {
		targetType = "devnet"
	}

	if targetType != "" {
		for _, net := range networks {
			if net.NetworkType == targetType {
				return net.NetworkName
			}
		}
	}

	// Return most recent
	return networks[len(networks)-1].NetworkName
}
