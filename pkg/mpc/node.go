// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package mpc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

// NodeStatus represents the current state of an MPC node.
type NodeStatus string

const (
	NodeStatusStopped  NodeStatus = "stopped"
	NodeStatusStarting NodeStatus = "starting"
	NodeStatusRunning  NodeStatus = "running"
	NodeStatusError    NodeStatus = "error"
)

// NodeConfig holds configuration for an MPC node.
type NodeConfig struct {
	NodeID      string   `json:"nodeId"`
	NodeName    string   `json:"nodeName"`
	NodeIndex   int      `json:"nodeIndex"`   // 0-based index in the MPC network
	Threshold   int      `json:"threshold"`   // t in t-of-n threshold signing
	TotalNodes  int      `json:"totalNodes"`  // n in t-of-n
	Network     string   `json:"network"`     // mainnet, testnet, devnet
	ListenAddr  string   `json:"listenAddr"`  // gRPC listen address
	P2PPort     int      `json:"p2pPort"`     // P2P communication port
	APIPort     int      `json:"apiPort"`     // API/gRPC port
	Peers       []string `json:"peers"`       // Other MPC node addresses
	DataDir     string   `json:"dataDir"`     // Data directory
	KeysDir     string   `json:"keysDir"`     // Encrypted keys directory
	LogLevel    string   `json:"logLevel"`
	Created     time.Time `json:"created"`
}

// NodeInfo contains runtime information about an MPC node.
type NodeInfo struct {
	Config    *NodeConfig `json:"config"`
	Status    NodeStatus  `json:"status"`
	PID       int         `json:"pid,omitempty"`
	Uptime    string      `json:"uptime,omitempty"`
	StartTime time.Time   `json:"startTime,omitempty"`
	Endpoint  string      `json:"endpoint,omitempty"`
	Error     string      `json:"error,omitempty"`
}

// NetworkConfig holds configuration for an MPC network.
type NetworkConfig struct {
	NetworkID   string        `json:"networkId"`
	NetworkName string        `json:"networkName"`
	NetworkType string        `json:"networkType"` // mainnet, testnet, devnet
	Threshold   int           `json:"threshold"`   // t in t-of-n
	TotalNodes  int           `json:"totalNodes"`  // n in t-of-n
	Nodes       []*NodeConfig `json:"nodes"`
	Created     time.Time     `json:"created"`
	BaseDir     string        `json:"baseDir"`
}

// NodeManager manages MPC node lifecycle.
type NodeManager struct {
	baseDir string
}

// NewNodeManager creates a new node manager.
func NewNodeManager(baseDir string) *NodeManager {
	return &NodeManager{baseDir: baseDir}
}

// BaseDir returns the base directory for MPC data.
func (m *NodeManager) BaseDir() string {
	return m.baseDir
}

// InitNetwork initializes a new MPC network with the specified configuration.
func (m *NodeManager) InitNetwork(ctx context.Context, networkType string, threshold, totalNodes int) (*NetworkConfig, error) {
	if threshold < 1 || threshold > totalNodes {
		return nil, fmt.Errorf("invalid threshold: must be between 1 and %d", totalNodes)
	}
	if totalNodes < 2 {
		return nil, fmt.Errorf("MPC network requires at least 2 nodes")
	}

	// Generate network ID
	networkID := generateID(8)
	networkName := fmt.Sprintf("mpc-%s-%s", networkType, networkID[:8])

	// Create network directory
	networkDir := filepath.Join(m.baseDir, networkName)
	if err := os.MkdirAll(networkDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create network directory: %w", err)
	}

	// Create keys directory (encrypted keys only)
	keysDir := filepath.Join(m.baseDir, "..", "keys", "mpc", networkName)
	if err := os.MkdirAll(keysDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create keys directory: %w", err)
	}

	// Base ports by network type
	baseP2PPort := 9700
	baseAPIPort := 9800
	switch networkType {
	case "testnet":
		baseP2PPort = 9710
		baseAPIPort = 9810
	case "devnet":
		baseP2PPort = 9720
		baseAPIPort = 9820
	}

	// Create node configurations
	nodes := make([]*NodeConfig, totalNodes)
	peers := make([]string, totalNodes)

	// First pass: create peer list
	for i := 0; i < totalNodes; i++ {
		peers[i] = fmt.Sprintf("127.0.0.1:%d", baseP2PPort+i)
	}

	// Second pass: create node configs
	for i := 0; i < totalNodes; i++ {
		nodeID := generateID(16)
		nodeName := fmt.Sprintf("mpc-node-%d", i+1)
		nodeDir := filepath.Join(networkDir, nodeName)

		if err := os.MkdirAll(nodeDir, 0750); err != nil {
			return nil, fmt.Errorf("failed to create node directory: %w", err)
		}

		// Create node subdirectories
		for _, subdir := range []string{"db", "logs"} {
			if err := os.MkdirAll(filepath.Join(nodeDir, subdir), 0750); err != nil {
				return nil, fmt.Errorf("failed to create %s directory: %w", subdir, err)
			}
		}

		// Remove self from peers list for this node
		nodePeers := make([]string, 0, totalNodes-1)
		for j, peer := range peers {
			if j != i {
				nodePeers = append(nodePeers, peer)
			}
		}

		nodes[i] = &NodeConfig{
			NodeID:     nodeID,
			NodeName:   nodeName,
			NodeIndex:  i,
			Threshold:  threshold,
			TotalNodes: totalNodes,
			Network:    networkType,
			ListenAddr: fmt.Sprintf("127.0.0.1:%d", baseAPIPort+i),
			P2PPort:    baseP2PPort + i,
			APIPort:    baseAPIPort + i,
			Peers:      nodePeers,
			DataDir:    nodeDir,
			KeysDir:    filepath.Join(keysDir, nodeName),
			LogLevel:   "info",
			Created:    time.Now(),
		}

		// Create node keys directory
		if err := os.MkdirAll(nodes[i].KeysDir, 0700); err != nil {
			return nil, fmt.Errorf("failed to create node keys directory: %w", err)
		}

		// Save node config
		if err := m.saveNodeConfig(nodes[i]); err != nil {
			return nil, fmt.Errorf("failed to save node config: %w", err)
		}
	}

	networkCfg := &NetworkConfig{
		NetworkID:   networkID,
		NetworkName: networkName,
		NetworkType: networkType,
		Threshold:   threshold,
		TotalNodes:  totalNodes,
		Nodes:       nodes,
		Created:     time.Now(),
		BaseDir:     networkDir,
	}

	// Save network config
	if err := m.saveNetworkConfig(networkCfg); err != nil {
		return nil, fmt.Errorf("failed to save network config: %w", err)
	}

	return networkCfg, nil
}

// StartNetwork starts all nodes in an MPC network.
func (m *NodeManager) StartNetwork(ctx context.Context, networkName string) error {
	networkCfg, err := m.LoadNetworkConfig(networkName)
	if err != nil {
		return fmt.Errorf("failed to load network config: %w", err)
	}

	for _, nodeCfg := range networkCfg.Nodes {
		if err := m.StartNode(ctx, nodeCfg); err != nil {
			return fmt.Errorf("failed to start node %s: %w", nodeCfg.NodeName, err)
		}
	}

	return nil
}

// StopNetwork stops all nodes in an MPC network.
func (m *NodeManager) StopNetwork(ctx context.Context, networkName string) error {
	networkCfg, err := m.LoadNetworkConfig(networkName)
	if err != nil {
		return fmt.Errorf("failed to load network config: %w", err)
	}

	for _, nodeCfg := range networkCfg.Nodes {
		if err := m.StopNode(ctx, nodeCfg.NodeName); err != nil {
			// Log but continue stopping other nodes
			fmt.Printf("Warning: failed to stop node %s: %v\n", nodeCfg.NodeName, err)
		}
	}

	return nil
}

// StartNode starts a single MPC node.
func (m *NodeManager) StartNode(ctx context.Context, cfg *NodeConfig) error {
	// Check if already running
	info, _ := m.GetNodeStatus(cfg.NodeName)
	if info != nil && info.Status == NodeStatusRunning {
		return fmt.Errorf("node %s is already running", cfg.NodeName)
	}

	logFile := filepath.Join(cfg.DataDir, "logs", "node.log")
	pidFile := filepath.Join(cfg.DataDir, "node.pid")

	// Build the peer address map for consensus transport
	peerMap := make(map[string]string)
	peerMap[cfg.NodeID] = fmt.Sprintf("127.0.0.1:%d", cfg.P2PPort)
	for i, peer := range cfg.Peers {
		peerID := fmt.Sprintf("peer-%d", i)
		peerMap[peerID] = peer
	}

	// Find lux-mpc binary
	mpcBinary := findMPCBinary()
	if mpcBinary == "" {
		// Fallback to placeholder mode if binary not found
		return m.startPlaceholderNode(cfg, logFile, pidFile)
	}

	// Build command args for consensus-embedded transport mode
	args := []string{
		"start",
		"--node-id", cfg.NodeID,
		"--listen", fmt.Sprintf(":%d", cfg.P2PPort),
		"--api", fmt.Sprintf(":%d", cfg.APIPort),
		"--data", cfg.DataDir,
		"--keys", cfg.KeysDir,
		"--threshold", strconv.Itoa(cfg.Threshold),
		"--log-level", cfg.LogLevel,
		"--mode", "consensus", // Use consensus-embedded transport (not NATS/Consul)
	}

	// Add peers
	for _, peer := range cfg.Peers {
		args = append(args, "--peer", peer)
	}

	// Create log file
	logF, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}

	cmd := exec.Command(mpcBinary, args...)
	cmd.Stdout = logF
	cmd.Stderr = logF
	setSysProcAttr(cmd)

	if err := cmd.Start(); err != nil {
		logF.Close()
		return fmt.Errorf("failed to start node process: %w", err)
	}

	// Save PID
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644); err != nil {
		cmd.Process.Kill()
		logF.Close()
		return fmt.Errorf("failed to save PID: %w", err)
	}

	// Save start time
	startTimeFile := filepath.Join(cfg.DataDir, "start_time")
	if err := os.WriteFile(startTimeFile, []byte(time.Now().Format(time.RFC3339)), 0644); err != nil {
		return fmt.Errorf("failed to save start time: %w", err)
	}

	// Don't close log file - keep it for the daemon
	return nil
}

// startPlaceholderNode starts a placeholder process when the MPC binary isn't available
func (m *NodeManager) startPlaceholderNode(cfg *NodeConfig, logFile, pidFile string) error {
	// Fallback placeholder for testing without actual MPC daemon
	cmd := exec.Command("sh", "-c", fmt.Sprintf(
		"echo 'MPC Node %s (placeholder) started at %s' >> %s && while true; do sleep 3600; done",
		cfg.NodeName, time.Now().Format(time.RFC3339), logFile,
	))

	setSysProcAttr(cmd)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start placeholder process: %w", err)
	}

	// Save PID
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644); err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("failed to save PID: %w", err)
	}

	// Save start time
	startTimeFile := filepath.Join(cfg.DataDir, "start_time")
	if err := os.WriteFile(startTimeFile, []byte(time.Now().Format(time.RFC3339)), 0644); err != nil {
		return fmt.Errorf("failed to save start time: %w", err)
	}

	return nil
}

// findMPCBinary searches for the mpcd binary in common locations
func findMPCBinary() string {
	// Check common locations
	locations := []string{
		"/usr/local/bin/mpcd",
		"/usr/bin/mpcd",
	}

	// Check GOPATH/bin
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		locations = append(locations, filepath.Join(gopath, "bin", "mpcd"))
	}

	// Check HOME/go/bin
	if home, err := os.UserHomeDir(); err == nil {
		locations = append(locations, filepath.Join(home, "go", "bin", "mpcd"))
	}

	// Check PATH
	if path, err := exec.LookPath("mpcd"); err == nil {
		return path
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}

	return ""
}

// StopNode stops a single MPC node.
func (m *NodeManager) StopNode(ctx context.Context, nodeName string) error {
	// Find node config
	cfg, err := m.findNodeConfig(nodeName)
	if err != nil {
		return err
	}

	pidFile := filepath.Join(cfg.DataDir, "node.pid")
	pidData, err := os.ReadFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Node not running
		}
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	pid, err := strconv.Atoi(string(pidData))
	if err != nil {
		return fmt.Errorf("invalid PID: %w", err)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Send termination signal for graceful shutdown
	if err := signalTerm(process); err != nil {
		// Process might already be dead
		if err.Error() != "os: process already finished" {
			return fmt.Errorf("failed to stop process: %w", err)
		}
	}

	// Remove PID file
	os.Remove(pidFile)
	os.Remove(filepath.Join(cfg.DataDir, "start_time"))

	return nil
}

// GetNodeStatus returns the status of a single node.
func (m *NodeManager) GetNodeStatus(nodeName string) (*NodeInfo, error) {
	cfg, err := m.findNodeConfig(nodeName)
	if err != nil {
		return nil, err
	}

	info := &NodeInfo{
		Config:   cfg,
		Status:   NodeStatusStopped,
		Endpoint: fmt.Sprintf("http://%s", cfg.ListenAddr),
	}

	// Check if running
	pidFile := filepath.Join(cfg.DataDir, "node.pid")
	pidData, err := os.ReadFile(pidFile)
	if err == nil {
		pid, err := strconv.Atoi(string(pidData))
		if err == nil {
			process, err := os.FindProcess(pid)
			if err == nil {
				// Check if process is still alive
				err = checkProcessAlive(process)
				if err == nil {
					info.Status = NodeStatusRunning
					info.PID = pid

					// Get uptime
					startTimeFile := filepath.Join(cfg.DataDir, "start_time")
					if startTimeData, err := os.ReadFile(startTimeFile); err == nil {
						if startTime, err := time.Parse(time.RFC3339, string(startTimeData)); err == nil {
							info.StartTime = startTime
							info.Uptime = time.Since(startTime).Round(time.Second).String()
						}
					}
				}
			}
		}
	}

	return info, nil
}

// GetNetworkStatus returns the status of all nodes in a network.
func (m *NodeManager) GetNetworkStatus(networkName string) ([]*NodeInfo, error) {
	networkCfg, err := m.LoadNetworkConfig(networkName)
	if err != nil {
		return nil, err
	}

	infos := make([]*NodeInfo, len(networkCfg.Nodes))
	for i, nodeCfg := range networkCfg.Nodes {
		info, err := m.GetNodeStatus(nodeCfg.NodeName)
		if err != nil {
			infos[i] = &NodeInfo{
				Config: nodeCfg,
				Status: NodeStatusError,
				Error:  err.Error(),
			}
		} else {
			infos[i] = info
		}
	}

	return infos, nil
}

// ListNetworks returns all MPC networks.
func (m *NodeManager) ListNetworks() ([]*NetworkConfig, error) {
	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var networks []*NetworkConfig
	for _, entry := range entries {
		if !entry.IsDir() || !isNetworkDir(entry.Name()) {
			continue
		}

		cfg, err := m.LoadNetworkConfig(entry.Name())
		if err != nil {
			continue // Skip invalid networks
		}
		networks = append(networks, cfg)
	}

	return networks, nil
}

// LoadNetworkConfig loads a network configuration from disk.
func (m *NodeManager) LoadNetworkConfig(networkName string) (*NetworkConfig, error) {
	configPath := filepath.Join(m.baseDir, networkName, "network.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read network config: %w", err)
	}

	var cfg NetworkConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse network config: %w", err)
	}

	return &cfg, nil
}

// DeleteNetwork removes an MPC network and all its data.
func (m *NodeManager) DeleteNetwork(ctx context.Context, networkName string, force bool) error {
	networkCfg, err := m.LoadNetworkConfig(networkName)
	if err != nil {
		return err
	}

	// Stop all nodes first
	if err := m.StopNetwork(ctx, networkName); err != nil && !force {
		return fmt.Errorf("failed to stop network: %w", err)
	}

	// Remove network directory
	if err := os.RemoveAll(networkCfg.BaseDir); err != nil {
		return fmt.Errorf("failed to remove network directory: %w", err)
	}

	// Remove keys directory
	keysDir := filepath.Join(m.baseDir, "..", "keys", "mpc", networkName)
	if err := os.RemoveAll(keysDir); err != nil {
		// Log but don't fail
		fmt.Printf("Warning: failed to remove keys directory: %v\n", err)
	}

	return nil
}

// Helper functions

func (m *NodeManager) saveNodeConfig(cfg *NodeConfig) error {
	configPath := filepath.Join(cfg.DataDir, "config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0640)
}

func (m *NodeManager) saveNetworkConfig(cfg *NetworkConfig) error {
	configPath := filepath.Join(cfg.BaseDir, "network.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0640)
}

func (m *NodeManager) findNodeConfig(nodeName string) (*NodeConfig, error) {
	// Search all networks for the node
	networks, err := m.ListNetworks()
	if err != nil {
		return nil, err
	}

	for _, network := range networks {
		for _, node := range network.Nodes {
			if node.NodeName == nodeName {
				return node, nil
			}
		}
	}

	return nil, fmt.Errorf("node not found: %s", nodeName)
}

func generateID(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func isNetworkDir(name string) bool {
	// MPC network directories start with "mpc-"
	return len(name) > 4 && name[:4] == "mpc-"
}
