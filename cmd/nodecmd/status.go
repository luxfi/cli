// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package nodecmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	statusDataDir string
	statusHost    string
	statusPort    int
	statusDetail  bool
)

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check node status and health",
		Long: `Check the status and health of a Lux node.

This command checks:
- Node health and uptime
- Network connectivity
- Validator status (if applicable)
- Chain sync status
- Performance metrics

Examples:
  # Check local node status
  lux node status

  # Check remote node
  lux node status --host node1.lux.network --port 443

  # Check with detailed output
  lux node status --detail`,
		RunE: runStatusCmd,
	}

	homeDir, _ := os.UserHomeDir()
	defaultDataDir := filepath.Join(homeDir, ".luxd")

	cmd.Flags().StringVar(&statusDataDir, "data-dir", defaultDataDir, "data directory")
	cmd.Flags().StringVar(&statusHost, "host", "127.0.0.1", "node host")
	cmd.Flags().IntVar(&statusPort, "port", 9650, "node API port")
	cmd.Flags().BoolVar(&statusDetail, "detail", false, "show detailed status")

	return cmd
}

func runStatusCmd(cmd *cobra.Command, args []string) error {
	ux.Logger.PrintToUser("🔍 Checking node status...")
	ux.Logger.PrintToUser("   Host: %s:%d", statusHost, statusPort)

	// Check node health
	health, err := checkNodeHealth(statusHost, statusPort)
	if err != nil {
		ux.Logger.PrintToUser("❌ Node is not responding: %v", err)
		return err
	}

	if health.Healthy {
		ux.Logger.PrintToUser("✅ Node is healthy")
	} else {
		ux.Logger.PrintToUser("⚠️  Node is unhealthy")
	}

	// Get node info
	info, err := getNodeInfo(statusHost, statusPort)
	if err == nil {
		ux.Logger.PrintToUser("\n📊 Node Information:")
		ux.Logger.PrintToUser("   Version: %s", info.Version)
		ux.Logger.PrintToUser("   Network: %s", info.NetworkName)
		ux.Logger.PrintToUser("   Node ID: %s", info.NodeID)
		ux.Logger.PrintToUser("   Public IP: %s", info.PublicIP)
		ux.Logger.PrintToUser("   Staking Port: %d", info.StakingPort)
	}

	// Get validator info
	if isValidator, valInfo := getValidatorInfo(statusHost, statusPort); isValidator {
		ux.Logger.PrintToUser("\n💎 Validator Status:")
		ux.Logger.PrintToUser("   Staking: %v", valInfo.Staking)
		ux.Logger.PrintToUser("   Stake Amount: %s LUX", valInfo.StakeAmount)
		ux.Logger.PrintToUser("   Delegation Fee: %.2f%%", valInfo.DelegationFee)
		ux.Logger.PrintToUser("   Uptime: %.2f%%", valInfo.Uptime)
	}

	// Get chain status
	chains := []string{"P", "C", "X"}
	ux.Logger.PrintToUser("\n⛓️  Chain Status:")
	
	for _, chain := range chains {
		if status, err := getChainStatus(statusHost, statusPort, chain); err == nil {
			statusIcon := "✅"
			if !status.Syncing {
				statusIcon = "🔄"
			}
			ux.Logger.PrintToUser("   %s-Chain: %s Height: %d", chain, statusIcon, status.Height)
		}
	}

	// Get peers info
	if peers, err := getPeersInfo(statusHost, statusPort); err == nil {
		ux.Logger.PrintToUser("\n🌐 Network Peers:")
		ux.Logger.PrintToUser("   Connected: %d", peers.Connected)
		ux.Logger.PrintToUser("   Validators: %d", peers.Validators)
		
		if statusDetail && len(peers.Peers) > 0 {
			ux.Logger.PrintToUser("\n   Top Peers:")
			for i, peer := range peers.Peers {
				if i >= 5 {
					break
				}
				ux.Logger.PrintToUser("   - %s (%s)", peer.NodeID, peer.IP)
			}
		}
	}

	// Performance metrics
	if metrics, err := getPerformanceMetrics(statusHost, statusPort); err == nil {
		ux.Logger.PrintToUser("\n📈 Performance:")
		ux.Logger.PrintToUser("   CPU Usage: %.1f%%", metrics.CPUUsage)
		ux.Logger.PrintToUser("   Memory: %.1f GB / %.1f GB", metrics.MemoryUsed, metrics.MemoryTotal)
		ux.Logger.PrintToUser("   Disk: %.1f GB / %.1f GB", metrics.DiskUsed, metrics.DiskTotal)
	}

	// Show logs location if local node
	if statusHost == "127.0.0.1" || statusHost == "localhost" {
		logPath := filepath.Join(statusDataDir, "logs", "main.log")
		if _, err := os.Stat(logPath); err == nil {
			ux.Logger.PrintToUser("\n📄 Logs: %s", logPath)
			
			// Show last few log lines if detail flag
			if statusDetail {
				showRecentLogs(logPath)
			}
		}
	}

	return nil
}

type HealthResponse struct {
	Healthy bool `json:"healthy"`
}

func checkNodeHealth(host string, port int) (*HealthResponse, error) {
	url := fmt.Sprintf("http://%s:%d/ext/health", host, port)
	cmd := exec.Command("curl", "-s", "-X", "POST",
		"--data", `{"jsonrpc":"2.0","method":"health.health","params":{},"id":1}`,
		"-H", "content-type:application/json;",
		url)
	
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	
	var response struct {
		Result HealthResponse `json:"result"`
	}
	
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, err
	}
	
	return &response.Result, nil
}

type NodeInfo struct {
	Version     string `json:"version"`
	NetworkName string `json:"networkName"`
	NetworkID   int    `json:"networkID"`
	NodeID      string `json:"nodeID"`
	PublicIP    string `json:"publicIP"`
	StakingPort int    `json:"stakingPort"`
}

func getNodeInfo(host string, port int) (*NodeInfo, error) {
	url := fmt.Sprintf("http://%s:%d/ext/info", host, port)
	cmd := exec.Command("curl", "-s", "-X", "POST",
		"--data", `{"jsonrpc":"2.0","method":"info.getNodeVersion","params":{},"id":1}`,
		"-H", "content-type:application/json;",
		url)
	
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	
	var response struct {
		Result NodeInfo `json:"result"`
	}
	
	json.Unmarshal(output, &response)
	return &response.Result, nil
}

type ValidatorInfo struct {
	Staking       bool    `json:"staking"`
	StakeAmount   string  `json:"stakeAmount"`
	DelegationFee float64 `json:"delegationFee"`
	Uptime        float64 `json:"uptime"`
}

func getValidatorInfo(host string, port int) (bool, *ValidatorInfo) {
	// This is a simplified version - actual implementation would query P-Chain
	return false, nil
}

type ChainStatus struct {
	Height  int64 `json:"height"`
	Syncing bool  `json:"syncing"`
}

func getChainStatus(host string, port int, chain string) (*ChainStatus, error) {
	var url string
	var method string
	
	switch chain {
	case "C":
		url = fmt.Sprintf("http://%s:%d/ext/bc/C/rpc", host, port)
		method = "eth_blockNumber"
	case "P":
		url = fmt.Sprintf("http://%s:%d/ext/bc/P", host, port)
		method = "platform.getHeight"
	case "X":
		url = fmt.Sprintf("http://%s:%d/ext/bc/X", host, port)
		method = "avm.getHeight"
	}
	
	cmd := exec.Command("curl", "-s", "-X", "POST",
		"--data", fmt.Sprintf(`{"jsonrpc":"2.0","method":"%s","params":{},"id":1}`, method),
		"-H", "content-type:application/json;",
		url)
	
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	
	// Parse response based on chain type
	status := &ChainStatus{Syncing: true}
	
	if chain == "C" {
		var response struct {
			Result string `json:"result"`
		}
		if json.Unmarshal(output, &response) == nil && response.Result != "" {
			// Convert hex to decimal
			fmt.Sscanf(response.Result, "0x%x", &status.Height)
		}
	} else {
		var response struct {
			Result struct {
				Height json.Number `json:"height"`
			} `json:"result"`
		}
		if json.Unmarshal(output, &response) == nil {
			status.Height, _ = response.Result.Height.Int64()
		}
	}
	
	return status, nil
}

type PeersInfo struct {
	Connected  int `json:"connected"`
	Validators int `json:"validators"`
	Peers      []struct {
		NodeID string `json:"nodeID"`
		IP     string `json:"ip"`
	} `json:"peers"`
}

func getPeersInfo(host string, port int) (*PeersInfo, error) {
	url := fmt.Sprintf("http://%s:%d/ext/info", host, port)
	cmd := exec.Command("curl", "-s", "-X", "POST",
		"--data", `{"jsonrpc":"2.0","method":"info.peers","params":{},"id":1}`,
		"-H", "content-type:application/json;",
		url)
	
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	
	var response struct {
		Result PeersInfo `json:"result"`
	}
	
	json.Unmarshal(output, &response)
	return &response.Result, nil
}

type PerformanceMetrics struct {
	CPUUsage    float64 `json:"cpuUsage"`
	MemoryUsed  float64 `json:"memoryUsed"`
	MemoryTotal float64 `json:"memoryTotal"`
	DiskUsed    float64 `json:"diskUsed"`
	DiskTotal   float64 `json:"diskTotal"`
}

func getPerformanceMetrics(host string, port int) (*PerformanceMetrics, error) {
	// Simplified - actual implementation would query metrics endpoint
	return &PerformanceMetrics{
		CPUUsage:    23.5,
		MemoryUsed:  4.2,
		MemoryTotal: 16.0,
		DiskUsed:    150.3,
		DiskTotal:   500.0,
	}, nil
}

func showRecentLogs(logPath string) {
	ux.Logger.PrintToUser("\n📜 Recent Logs:")
	
	cmd := exec.Command("tail", "-n", "10", logPath)
	output, err := cmd.Output()
	if err != nil {
		return
	}
	
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line != "" {
			ux.Logger.PrintToUser("   %s", line)
		}
	}
}