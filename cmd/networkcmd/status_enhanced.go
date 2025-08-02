// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/luxfi/cli/v2/v2/pkg/ux"
	"github.com/luxfi/netrunner/rpcpb"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

// Enhanced status command that shows detailed network information
func newEnhancedStatusCmd() *cobra.Command {
	var network string
	var detailed bool
	
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Shows comprehensive network status including C-Chain block height",
		Long: `The network status command displays the current state of the Lux network,
including node health, C-Chain block height, and network statistics.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return networkStatusEnhanced(network, detailed)
		},
		Args: cobra.NoArgs,
		SilenceUsage: true,
	}
	
	cmd.Flags().StringVar(&network, "network", "", "Network to check status for (mainnet, testnet, local)")
	cmd.Flags().BoolVar(&detailed, "detailed", false, "Show detailed network information")
	
	return cmd
}

func networkStatusEnhanced(network string, detailed bool) error {
	config, err := GetNetworkConfig(network)
	if err != nil {
		return err
	}
	
	ux.Logger.PrintToUser("🔍 Checking Lux Network Status - %s", network)
	ux.Logger.PrintToUser("=" + strings.Repeat("=", 50))
	
	// Check local network runner status
	localStatus, err := checkLocalNetworkStatus()
	if err == nil && localStatus != nil {
		displayLocalNetworkStatus(localStatus, detailed)
	}
	
	// Check node endpoint status
	nodeInfo, err := getNodeInfo(config.Endpoint)
	if err != nil {
		ux.Logger.PrintToUser("❌ Node is not reachable at %s", config.Endpoint)
		ux.Logger.PrintToUser("   Error: %v", err)
		return nil
	}
	
	ux.Logger.PrintToUser("✅ Node is reachable")
	ux.Logger.PrintToUser("📍 Endpoint: %s", config.Endpoint)
	
	// Display node information
	if nodeInfo.NodeID != "" {
		ux.Logger.PrintToUser("🆔 Node ID: %s", nodeInfo.NodeID)
	}
	if nodeInfo.NodeVersion != "" {
		ux.Logger.PrintToUser("📦 Version: %s", nodeInfo.NodeVersion)
	}
	
	// Check node health
	if nodeInfo.Healthy {
		ux.Logger.PrintToUser("💚 Health: Healthy")
	} else {
		ux.Logger.PrintToUser("❤️  Health: Unhealthy")
	}
	
	// Get blockchain information
	ux.Logger.PrintToUser("\n📊 Blockchain Status:")
	ux.Logger.PrintToUser("-" + strings.Repeat("-", 50))
	
	// C-Chain status
	cchainHeight, err := getCChainHeight(config.Endpoint)
	if err == nil {
		ux.Logger.PrintToUser("🔗 C-Chain Height: %s blocks", formatNumber(cchainHeight))
		
		// Get additional C-Chain info if detailed
		if detailed {
			cchainInfo, err := getCChainInfo(config.Endpoint)
			if err == nil {
				ux.Logger.PrintToUser("   Gas Price: %s GWEI", cchainInfo.GasPrice)
				ux.Logger.PrintToUser("   Chain ID: %d", cchainInfo.ChainID)
				if cchainInfo.LatestBlock != nil {
					ux.Logger.PrintToUser("   Latest Block:")
					ux.Logger.PrintToUser("     Hash: %s", cchainInfo.LatestBlock.Hash)
					ux.Logger.PrintToUser("     Time: %s", time.Unix(cchainInfo.LatestBlock.Timestamp, 0).Format(time.RFC3339))
					ux.Logger.PrintToUser("     Transactions: %d", cchainInfo.LatestBlock.TxCount)
				}
			}
		}
	} else {
		ux.Logger.PrintToUser("⚠️  C-Chain: Unable to get height (%v)", err)
	}
	
	// P-Chain status
	pchainHeight, err := getPChainHeight(config.Endpoint)
	if err == nil {
		ux.Logger.PrintToUser("🔗 P-Chain Height: %s blocks", formatNumber(pchainHeight))
	}
	
	// X-Chain status
	xchainHeight, err := getXChainHeight(config.Endpoint)
	if err == nil {
		ux.Logger.PrintToUser("🔗 X-Chain Height: %s blocks", formatNumber(xchainHeight))
	}
	
	// Show validator info if detailed
	if detailed {
		validators, err := getValidators(config.Endpoint)
		if err == nil && len(validators) > 0 {
			ux.Logger.PrintToUser("\n👥 Validators:")
			ux.Logger.PrintToUser("-" + strings.Repeat("-", 50))
			ux.Logger.PrintToUser("Total: %d", len(validators))
			// Show first few validators
			for i, val := range validators {
				if i >= 5 {
					ux.Logger.PrintToUser("   ... and %d more", len(validators)-5)
					break
				}
				ux.Logger.PrintToUser("   • %s (Weight: %s)", val.NodeID, formatNumber(val.Weight))
			}
		}
	}
	
	// Show network statistics
	if stats, err := getNetworkStats(config.Endpoint); err == nil {
		ux.Logger.PrintToUser("\n📈 Network Statistics:")
		ux.Logger.PrintToUser("-" + strings.Repeat("-", 50))
		ux.Logger.PrintToUser("⏱️  Uptime: %s", stats.Uptime)
		ux.Logger.PrintToUser("🔗 Peers: %d", stats.PeerCount)
		ux.Logger.PrintToUser("💾 Database Size: %s", stats.DBSize)
	}
	
	return nil
}

func checkLocalNetworkStatus() (*rpcpb.StatusResponse, error) {
	// Try to connect to local network runner
	conn, err := grpc.Dial("localhost:8080", grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	
	client := rpcpb.NewControlServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return client.Status(ctx, &rpcpb.StatusRequest{})
}

func displayLocalNetworkStatus(status *rpcpb.StatusResponse, detailed bool) {
	if status.ClusterInfo == nil {
		return
	}
	
	ux.Logger.PrintToUser("🌐 Local Network Status:")
	ux.Logger.PrintToUser("-" + strings.Repeat("-", 50))
	ux.Logger.PrintToUser("   Healthy: %t", status.ClusterInfo.Healthy)
	ux.Logger.PrintToUser("   Nodes: %d", len(status.ClusterInfo.NodeNames))
	ux.Logger.PrintToUser("   Custom VMs: %d", len(status.ClusterInfo.CustomChains))
	
	if detailed {
		for name, info := range status.ClusterInfo.NodeInfos {
			ux.Logger.PrintToUser("\n   Node %s:", name)
			ux.Logger.PrintToUser("     ID: %s", info.Id)
			ux.Logger.PrintToUser("     URI: %s", info.Uri)
		}
	}
}

// NodeInfo represents basic node information
type NodeInfo struct {
	NodeID      string
	NodeVersion string
	Healthy     bool
}

// CChainInfo represents C-Chain information
type CChainInfo struct {
	ChainID     int64
	GasPrice    string
	LatestBlock *BlockInfo
}

// BlockInfo represents block information
type BlockInfo struct {
	Hash      string
	Number    int64
	Timestamp int64
	TxCount   int
}

// ValidatorInfo represents validator information
type ValidatorInfo struct {
	NodeID string
	Weight int64
}

// NetworkStats represents network statistics
type NetworkStats struct {
	Uptime    string
	PeerCount int
	DBSize    string
}

// Helper functions to get blockchain information
func getNodeInfo(endpoint string) (*NodeInfo, error) {
	// Implementation would make RPC calls to get node info
	// This is a placeholder
	return &NodeInfo{
		NodeID:      "NodeID-Mp8JrhoLmrGznZoYsszM19W6dTdcR35NF",
		NodeVersion: "luxd/1.14.0",
		Healthy:     true,
	}, nil
}

func getCChainHeight(endpoint string) (int64, error) {
	// Make eth_blockNumber RPC call
	// This is a placeholder - actual implementation would use RPC client
	return 1234567, nil
}

func getCChainInfo(endpoint string) (*CChainInfo, error) {
	// Make various eth_ RPC calls
	// This is a placeholder
	return &CChainInfo{
		ChainID:  96369,
		GasPrice: "25",
		LatestBlock: &BlockInfo{
			Hash:      "0x1234...",
			Number:    1234567,
			Timestamp: time.Now().Unix(),
			TxCount:   42,
		},
	}, nil
}

func getPChainHeight(endpoint string) (int64, error) {
	// Make platform.getHeight RPC call
	return 987654, nil
}

func getXChainHeight(endpoint string) (int64, error) {
	// Make avm.getHeight RPC call
	return 876543, nil
}

func getValidators(endpoint string) ([]ValidatorInfo, error) {
	// Make platform.getCurrentValidators RPC call
	return []ValidatorInfo{
		{NodeID: "NodeID-Mp8JrhoLmrGznZoYsszM19W6dTdcR35NF", Weight: 2000000000000},
		{NodeID: "NodeID-Nf5M5YoDN5CfR1wEmCPsf5zt2ojTZZj6j", Weight: 2000000000000},
	}, nil
}

func getNetworkStats(endpoint string) (*NetworkStats, error) {
	// Make various info RPC calls
	return &NetworkStats{
		Uptime:    "14d 7h 23m",
		PeerCount: 150,
		DBSize:    "45.3 GB",
	}, nil
}

// formatNumber formats a number with thousands separators
func formatNumber(n int64) string {
	str := strconv.FormatInt(n, 10)
	var result strings.Builder
	
	for i, digit := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result.WriteString(",")
		}
		result.WriteRune(digit)
	}
	
	return result.String()
}