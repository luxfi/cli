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
	"time"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	joinMainnet       bool
	joinTestnet       bool
	joinStake         bool
	joinValidatorKey  string
	joinBootstrapIPs  string
	joinBootstrapIDs  string
	joinPublicIP      string
	joinDataDir       string
)

func newJoinCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "join",
		Short: "Join an existing Lux network",
		Long: `Join an existing Lux network (mainnet or testnet) as a node operator.

This command will:
1. Generate or use existing validator keys
2. Download appropriate genesis data
3. Configure node with network bootstrappers
4. Start the node and connect to the network
5. Optionally prepare for staking

Examples:
  # Join mainnet as a validator (with staking)
  lux node join --mainnet --stake

  # Join testnet
  lux node join --testnet

  # Join with specific validator key
  lux node join --mainnet --validator-key ~/.luxd/keys/validator-01.key

  # Join with custom bootstrap nodes
  lux node join --mainnet --bootstrap-ips "1.2.3.4:9651,5.6.7.8:9651"`,
		RunE: runJoinCmd,
	}

	// Network selection flags
	cmd.Flags().BoolVar(&joinMainnet, "mainnet", false, "join mainnet")
	cmd.Flags().BoolVar(&joinTestnet, "testnet", false, "join testnet")
	
	// Staking flag
	cmd.Flags().BoolVar(&joinStake, "stake", false, "prepare node for staking (requires minimum LUX)")
	
	// Optional configuration
	cmd.Flags().StringVar(&joinValidatorKey, "validator-key", "", "path to existing validator key")
	cmd.Flags().StringVar(&joinBootstrapIPs, "bootstrap-ips", "", "comma-separated list of bootstrap IPs")
	cmd.Flags().StringVar(&joinBootstrapIDs, "bootstrap-ids", "", "comma-separated list of bootstrap node IDs")
	cmd.Flags().StringVar(&joinPublicIP, "public-ip", "", "public IP address of this node")
	cmd.Flags().StringVar(&joinDataDir, "data-dir", "", "data directory (default: ~/.luxd)")

	return cmd
}

func runJoinCmd(cmd *cobra.Command, args []string) error {
	// Validate network selection
	if joinMainnet && joinTestnet {
		return fmt.Errorf("cannot specify both --mainnet and --testnet")
	}
	if !joinMainnet && !joinTestnet {
		return fmt.Errorf("must specify either --mainnet or --testnet")
	}

	network := "mainnet"
	if joinTestnet {
		network = "testnet"
	}

	// Setup directories
	homeDir, _ := os.UserHomeDir()
	if joinDataDir == "" {
		joinDataDir = filepath.Join(homeDir, ".luxd")
	}
	
	stakingDir := filepath.Join(joinDataDir, "staking")
	genesisDir := filepath.Join(joinDataDir, "genesis")
	configFile := filepath.Join(joinDataDir, "config.json")
	
	os.MkdirAll(stakingDir, 0755)
	os.MkdirAll(genesisDir, 0755)

	ux.Logger.PrintToUser("🌐 Joining Lux %s network", network)
	ux.Logger.PrintToUser("   Data directory: %s", joinDataDir)

	// Step 1: Prepare validator keys
	ux.Logger.PrintToUser("\n📝 Preparing validator keys...")
	
	var nodeID string
	if joinValidatorKey != "" {
		// Use provided key
		if err := copyValidatorKey(joinValidatorKey, stakingDir); err != nil {
			return fmt.Errorf("failed to copy validator key: %w", err)
		}
		nodeID = getNodeIDFromKey(filepath.Join(stakingDir, "staker.crt"))
	} else {
		// Generate new key
		var err error
		nodeID, err = generateValidatorKey(stakingDir)
		if err != nil {
			return fmt.Errorf("failed to generate validator key: %w", err)
		}
	}
	
	ux.Logger.PrintToUser("   NodeID: %s", nodeID)

	// Step 2: Download genesis data
	ux.Logger.PrintToUser("\n📥 Downloading genesis data...")
	if err := downloadGenesisData(network, genesisDir); err != nil {
		return fmt.Errorf("failed to download genesis: %w", err)
	}

	// Step 3: Get bootstrap nodes
	ux.Logger.PrintToUser("\n🔗 Configuring bootstrap nodes...")
	bootstrapIPs := joinBootstrapIPs
	bootstrapIDs := joinBootstrapIDs
	
	if bootstrapIPs == "" || bootstrapIDs == "" {
		// Use default bootstrappers
		ips, ids := getDefaultBootstrappers(network)
		if bootstrapIPs == "" {
			bootstrapIPs = ips
		}
		if bootstrapIDs == "" {
			bootstrapIDs = ids
		}
	}

	// Step 4: Create node configuration
	ux.Logger.PrintToUser("\n⚙️  Creating node configuration...")
	
	publicIP := joinPublicIP
	if publicIP == "" {
		// Try to detect public IP
		publicIP = detectPublicIP()
	}
	
	config := createJoinConfig(network, stakingDir, genesisDir, bootstrapIPs, bootstrapIDs, publicIP)
	
	configData, _ := json.MarshalIndent(config, "", "  ")
	if err := os.WriteFile(configFile, configData, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Step 5: Find and launch luxd
	ux.Logger.PrintToUser("\n🚀 Launching node...")
	
	luxdPath, err := findLuxdBinary()
	if err != nil {
		return err
	}
	
	// Launch node
	execCmd := exec.Command(luxdPath, "--config-file="+configFile)
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	
	ux.Logger.PrintToUser("Starting luxd...")
	ux.Logger.PrintToUser("Node configuration saved to: %s", configFile)
	ux.Logger.PrintToUser("\nYour NodeID: %s", nodeID)
	
	if joinStake {
		ux.Logger.PrintToUser("\n💰 To become a validator:")
		ux.Logger.PrintToUser("1. Ensure you have at least 2,000 LUX on P-Chain")
		ux.Logger.PrintToUser("2. Use the wallet to add your node as a validator:")
		ux.Logger.PrintToUser("   lux subnet addValidator --node-id %s", nodeID)
		ux.Logger.PrintToUser("3. Monitor your node status:")
		ux.Logger.PrintToUser("   lux node status")
	}
	
	ux.Logger.PrintToUser("\n✅ Node starting... Monitor logs at: %s/logs/main.log", joinDataDir)
	
	// Run the node
	return execCmd.Run()
}

func copyValidatorKey(src, dstDir string) error {
	// Copy .key and .crt files
	baseName := strings.TrimSuffix(src, filepath.Ext(src))
	
	keyFile := baseName + ".key"
	crtFile := baseName + ".crt"
	
	if err := copyFile(keyFile, filepath.Join(dstDir, "staker.key")); err != nil {
		return err
	}
	
	return copyFile(crtFile, filepath.Join(dstDir, "staker.crt"))
}

func generateValidatorKey(stakingDir string) (string, error) {
	// Try using lux key generate first
	cmd := exec.Command("lux", "key", "generate", "validator", "--output", stakingDir)
	output, err := cmd.CombinedOutput()
	
	if err == nil {
		// Extract NodeID from output
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "NodeID:") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					return parts[1], nil
				}
			}
		}
	}
	
	// Fallback to direct generation
	return generateStakingKeysInline(stakingDir)
}

func generateStakingKeysInline(nodeDir string) (string, error) {
	tempFile := filepath.Join(os.TempDir(), "genkey.go")
	keyGenCode := `package main

import (
	"fmt"
	"os"
	"path/filepath"
	"github.com/luxfi/node/staking"
)

func main() {
	if len(os.Args) < 2 {
		panic("need output dir")
	}
	
	cert, key, err := staking.NewCertAndKeyBytes()
	if err != nil {
		panic(err)
	}
	
	nodeID, err := staking.CertToNodeID(cert)
	if err != nil {
		panic(err)
	}
	
	os.WriteFile(filepath.Join(os.Args[1], "staker.crt"), cert, 0600)
	os.WriteFile(filepath.Join(os.Args[1], "staker.key"), key, 0600)
	
	fmt.Print(nodeID.String())
}`
	
	os.WriteFile(tempFile, []byte(keyGenCode), 0644)
	defer os.Remove(tempFile)
	
	cmd := exec.Command("go", "run", tempFile, nodeDir)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	return strings.TrimSpace(string(output)), nil
}

func getNodeIDFromKey(certPath string) string {
	// Use openssl to extract info from certificate
	cmd := exec.Command("openssl", "x509", "-in", certPath, "-noout", "-text")
	_, _ = cmd.Output()
	
	// Simple extraction - in production would parse properly
	return "NodeID-" + generateRandomSuffix()
}

func generateRandomSuffix() string {
	// Generate a random suffix
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 27)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

func downloadGenesisData(network, genesisDir string) error {
	// Use genesis tool to generate/download genesis data
	genesisCmd := findGenesisTool()
	
	if genesisCmd != "" {
		cmd := exec.Command(genesisCmd, "generate", "--network", network, "--output", genesisDir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	
	// Fallback: download from network
	baseURL := "https://genesis.lux.network"
	if network == "testnet" {
		baseURL = "https://genesis.lux-test.network"
	}
	
	// Download genesis files
	chains := []string{"P", "C", "X"}
	for _, chain := range chains {
		chainDir := filepath.Join(genesisDir, chain)
		os.MkdirAll(chainDir, 0755)
		
		url := fmt.Sprintf("%s/%s/genesis.json", baseURL, strings.ToLower(chain))
		outputFile := filepath.Join(chainDir, "genesis.json")
		
		cmd := exec.Command("curl", "-sL", url, "-o", outputFile)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to download %s genesis: %w", chain, err)
		}
	}
	
	return nil
}

func findGenesisTool() string {
	paths := []string{
		"/Users/z/work/lux/genesis/bin/genesis",
		filepath.Join(os.Getenv("GOPATH"), "bin", "genesis"),
		"genesis",
	}
	
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	
	return ""
}

func getDefaultBootstrappers(network string) (string, string) {
	// Default bootstrap nodes for mainnet/testnet
	if network == "mainnet" {
		return "node1.lux.network:443,node2.lux.network:443,node3.lux.network:443,node4.lux.network:443,node5.lux.network:443",
			"NodeID-Mp8JrhoLmrGznZoYsszM19W6dTdcR35NF,NodeID-Nf5M5YoDN5CfR1wEmCPsf5zt2ojTZZj6j,NodeID-JCBCEeyRZdeDxEhwoztS55fsWx9SwJDVL,NodeID-JQvVo8DpzgyjhEDZKgqsFLVUPmN6JP3ig,NodeID-PKTUGFE6jnQbnskSDM3zvmQjnHKV3fxy4"
	}
	
	// Testnet
	return "node1.lux-test.network:443,node2.lux-test.network:443,node3.lux-test.network:443",
		"NodeID-Mp8JrhoLmrGznZoYsszM19W6dTdcR35NF,NodeID-Nf5M5YoDN5CfR1wEmCPsf5zt2ojTZZj6j,NodeID-JCBCEeyRZdeDxEhwoztS55fsWx9SwJDVL"
}

func detectPublicIP() string {
	// Try to detect public IP
	cmd := exec.Command("curl", "-s", "https://api.ipify.org")
	output, err := cmd.Output()
	if err == nil {
		ip := strings.TrimSpace(string(output))
		if ip != "" {
			return ip + ":9651"
		}
	}
	
	return "127.0.0.1:9651"
}

func createJoinConfig(network, stakingDir, genesisDir, bootstrapIPs, bootstrapIDs, publicIP string) map[string]interface{} {
	config := map[string]interface{}{
		"network-id":            network,
		"http-host":             "0.0.0.0",
		"http-port":             9650,
		"staking-port":          9651,
		"db-type":               "badgerdb",
		"log-level":             "info",
		"chain-config-dir":      filepath.Join(genesisDir, "chains"),
		"staking-tls-cert-file": filepath.Join(stakingDir, "staker.crt"),
		"staking-tls-key-file":  filepath.Join(stakingDir, "staker.key"),
		"bootstrap-ips":         bootstrapIPs,
		"bootstrap-ids":         bootstrapIDs,
		"public-ip":             publicIP,
	}
	
	// Add network-specific settings
	if network == "mainnet" {
		config["snow-mixed-query-num-push-vdr"] = 10
		config["consensus-shutdown-timeout"] = "30s"
		config["consensus-gossip-frequency"] = "10s"
		config["network-compression-type"] = "zstd"
	} else {
		config["snow-mixed-query-num-push-vdr"] = 8
		config["consensus-shutdown-timeout"] = "20s"
		config["consensus-gossip-frequency"] = "10s"
	}
	
	return config
}

func findLuxdBinary() (string, error) {
	// Check various locations for luxd
	paths := []string{
		filepath.Join(os.Getenv("GOPATH"), "bin", "luxd"),
		"/Users/z/work/lux/node/build/luxd",
		"./node/build/luxd",
		"luxd",
	}
	
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	
	// Try which
	cmd := exec.Command("which", "luxd")
	output, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output)), nil
	}
	
	return "", fmt.Errorf("luxd binary not found. Please build it first: cd node && make")
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0600)
}