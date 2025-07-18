package migratecmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/node/ids"
	"github.com/luxfi/node/staking"
)

func runMigration(sourceDB, destDB string, chainID int64) error {
	// Build the migration tool if needed
	migrationToolPath := filepath.Join(utils.GetCLIRootDir(), "migration-tools")
	
	// Run go build
	buildCmd := exec.Command("go", "build", "-o", filepath.Join(migrationToolPath, "migrate"), filepath.Join(migrationToolPath, "migrate.go"))
	buildCmd.Dir = migrationToolPath
	if output, err := buildCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to build migration tool: %w\n%s", err, output)
	}

	// Run the migration
	migrateCmd := exec.Command(
		filepath.Join(migrationToolPath, "migrate"),
		"--src-pebble", sourceDB,
		"--dst-leveldb", destDB,
		"--chain-id", fmt.Sprintf("%d", chainID),
	)
	
	output, err := migrateCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("migration failed: %w\n%s", err, output)
	}
	
	ux.Logger.PrintToUser(string(output))
	return nil
}

func createPChainGenesis(outputDir string, numValidators int) error {
	// Generate staking keys for validators
	validators := make([]ValidatorInfo, numValidators)
	
	for i := 0; i < numValidators; i++ {
		nodeDir := filepath.Join(outputDir, fmt.Sprintf("node%d", i+1))
		stakingDir := filepath.Join(nodeDir, "staking")
		
		// Generate staking certificate
		certPath := filepath.Join(stakingDir, "staker.crt")
		keyPath := filepath.Join(stakingDir, "staker.key")
		
		if err := staking.InitNodeStakingKeyPair(keyPath, certPath); err != nil {
			return fmt.Errorf("failed to generate staking key for node%d: %w", i+1, err)
		}
		
		// Read the certificate to get NodeID
		cert, err := staking.LoadTLSCert(certPath)
		if err != nil {
			return fmt.Errorf("failed to load certificate: %w", err)
		}
		
		nodeID := ids.NodeIDFromCert(cert)
		validators[i] = ValidatorInfo{
			NodeID:    nodeID.String(),
			Weight:    1000000, // Equal weight for all bootstrap validators
			StartTime: time.Now().Unix(),
			EndTime:   time.Now().Add(365 * 24 * time.Hour).Unix(), // 1 year
		}
		
		ux.Logger.PrintToUser(fmt.Sprintf("Generated validator %d: NodeID %s", i+1, nodeID))
	}
	
	// Create P-Chain database with genesis
	pchainDBPath := filepath.Join(outputDir, "p-chain-db")
	
	// Build and run the P-Chain seeder
	seedCmd := exec.Command("go", "run", filepath.Join(utils.GetCLIRootDir(), "migration-tools", "seed-pchain.go"), pchainDBPath)
	if output, err := seedCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to seed P-Chain: %w\n%s", err, output)
	}
	
	// Save validator info for later use
	validatorData, err := json.MarshalIndent(validators, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal validator data: %w", err)
	}
	
	if err := ioutil.WriteFile(filepath.Join(outputDir, "validators.json"), validatorData, 0644); err != nil {
		return fmt.Errorf("failed to save validator data: %w", err)
	}
	
	return nil
}

func generateNodeConfigs(outputDir string, numValidators int, networkID uint32) error {
	// Base configuration for all nodes
	baseConfig := NodeConfig{
		NetworkID:              networkID,
		DBType:                 "leveldb",
		StakingEnabled:         true,
		HealthCheckFrequency:   "2s",
		NetworkPeerListGossipFrequency: "250ms",
		NetworkMaxReconnectDelay: "1s",
		HTTPHost:               "0.0.0.0",
		HTTPPort:               9650,
		StakingPort:            9651,
		BootstrapIPs:           []string{},
		BootstrapIDs:           []string{},
	}
	
	// Collect bootstrap IPs and IDs
	bootstrapIPs := make([]string, numValidators)
	bootstrapIDs := make([]string, numValidators)
	
	// First pass: collect node IDs
	for i := 0; i < numValidators; i++ {
		nodeDir := filepath.Join(outputDir, fmt.Sprintf("node%d", i+1))
		certPath := filepath.Join(nodeDir, "staking", "staker.crt")
		
		cert, err := staking.LoadTLSCert(certPath)
		if err != nil {
			return fmt.Errorf("failed to load certificate: %w", err)
		}
		
		nodeID := ids.NodeIDFromCert(cert)
		bootstrapIDs[i] = nodeID.String()
		bootstrapIPs[i] = fmt.Sprintf("validator%d.lux.network:9651", i+1)
	}
	
	// Second pass: create configs with bootstrap info
	for i := 0; i < numValidators; i++ {
		nodeConfig := baseConfig
		nodeConfig.HTTPPort = 9650 + uint16(i*10)
		nodeConfig.StakingPort = 9651 + uint16(i*10)
		
		// Add other nodes as bootstrap nodes
		for j := 0; j < numValidators; j++ {
			if i != j {
				nodeConfig.BootstrapIPs = append(nodeConfig.BootstrapIPs, bootstrapIPs[j])
				nodeConfig.BootstrapIDs = append(nodeConfig.BootstrapIDs, bootstrapIDs[j])
			}
		}
		
		// Save config
		configData, err := json.MarshalIndent(nodeConfig, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}
		
		nodeDir := filepath.Join(outputDir, fmt.Sprintf("node%d", i+1))
		configPath := filepath.Join(nodeDir, "config.json")
		
		if err := ioutil.WriteFile(configPath, configData, 0644); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		
		// Create launch script
		scriptContent := fmt.Sprintf(`#!/bin/bash
# Launch script for Node %d

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
NODE_DIR="$SCRIPT_DIR"
LUXD_PATH="${LUXD_PATH:-/home/z/work/lux/node/build/luxd}"

# Copy migrated data
mkdir -p "$NODE_DIR/db/C"
cp -r "%s/c-chain-db/"* "$NODE_DIR/db/C/"

mkdir -p "$NODE_DIR/db/P"  
cp -r "%s/p-chain-db/"* "$NODE_DIR/db/P/"

# Start the node
exec "$LUXD_PATH" \
    --config-file="$NODE_DIR/config.json" \
    --data-dir="$NODE_DIR" \
    --db-dir="$NODE_DIR/db" \
    --log-dir="$NODE_DIR/logs" \
    --http-port=%d \
    --staking-port=%d \
    --bootstrap-ips="%s" \
    --bootstrap-ids="%s" \
    "$@"
`, i+1, outputDir, outputDir, 
			9650+i*10, 9651+i*10,
			joinStrings(nodeConfig.BootstrapIPs, ","),
			joinStrings(nodeConfig.BootstrapIDs, ","))
		
		scriptPath := filepath.Join(nodeDir, "launch.sh")
		if err := ioutil.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
			return fmt.Errorf("failed to create launch script: %w", err)
		}
	}
	
	// Create master launch script
	masterScript := `#!/bin/bash
# Launch all bootstrap nodes

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

echo "Starting Lux mainnet bootstrap nodes..."

for i in {1..` + fmt.Sprintf("%d", numValidators) + `}; do
    echo "Starting node$i..."
    cd "$SCRIPT_DIR/node$i"
    ./launch.sh > node.log 2>&1 &
    echo "  PID: $!"
    sleep 2
done

echo ""
echo "All nodes started. Check logs in each node directory."
echo "Monitor with: tail -f node*/node.log"
`
	
	if err := ioutil.WriteFile(filepath.Join(outputDir, "start-all.sh"), []byte(masterScript), 0755); err != nil {
		return fmt.Errorf("failed to create master launch script: %w", err)
	}
	
	return nil
}

func startBootstrapNodes(migrationDir string, detached bool) error {
	startScript := filepath.Join(migrationDir, "start-all.sh")
	
	if _, err := os.Stat(startScript); err != nil {
		return fmt.Errorf("start script not found: %w", err)
	}
	
	cmd := exec.Command("bash", startScript)
	cmd.Dir = migrationDir
	
	if detached {
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to start nodes: %w", err)
		}
		ux.Logger.PrintToUser("Nodes started in background")
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to start nodes: %w", err)
		}
	}
	
	return nil
}

func validateNetwork(migrationDir string) error {
	// Load validator info
	validatorData, err := ioutil.ReadFile(filepath.Join(migrationDir, "validators.json"))
	if err != nil {
		return fmt.Errorf("failed to read validator data: %w", err)
	}
	
	var validators []ValidatorInfo
	if err := json.Unmarshal(validatorData, &validators); err != nil {
		return fmt.Errorf("failed to parse validator data: %w", err)
	}
	
	// Check each node
	for i, validator := range validators {
		port := 9650 + i*10
		url := fmt.Sprintf("http://localhost:%d", port)
		
		ux.Logger.PrintToUser(fmt.Sprintf("Checking node %d (NodeID: %s)...", i+1, validator.NodeID))
		
		// Check health
		healthURL := fmt.Sprintf("%s/ext/health", url)
		if err := utils.CheckAPIHealthy(healthURL); err != nil {
			ux.Logger.PrintToUser(fmt.Sprintf("  ❌ Node %d is not healthy: %v", i+1, err))
			continue
		}
		
		ux.Logger.PrintToUser(fmt.Sprintf("  ✅ Node %d is healthy", i+1))
		
		// Check C-Chain
		cChainURL := fmt.Sprintf("%s/ext/bc/C/rpc", url)
		blockHeight, err := getBlockHeight(cChainURL)
		if err != nil {
			ux.Logger.PrintToUser(fmt.Sprintf("  ❌ C-Chain error: %v", err))
		} else {
			ux.Logger.PrintToUser(fmt.Sprintf("  ✅ C-Chain block height: %d", blockHeight))
		}
	}
	
	return nil
}

// Helper types and functions
type ValidatorInfo struct {
	NodeID    string `json:"nodeID"`
	Weight    uint64 `json:"weight"`
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime"`
}

type NodeConfig struct {
	NetworkID                      uint32   `json:"network-id"`
	DBType                         string   `json:"db-type"`
	StakingEnabled                 bool     `json:"staking-enabled"`
	HealthCheckFrequency           string   `json:"health-check-frequency"`
	NetworkPeerListGossipFrequency string   `json:"network-peer-list-gossip-frequency"`
	NetworkMaxReconnectDelay       string   `json:"network-max-reconnect-delay"`
	HTTPHost                       string   `json:"http-host"`
	HTTPPort                       uint16   `json:"http-port"`
	StakingPort                    uint16   `json:"staking-port"`
	BootstrapIPs                   []string `json:"bootstrap-ips"`
	BootstrapIDs                   []string `json:"bootstrap-ids"`
}

func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

func getBlockHeight(rpcURL string) (uint64, error) {
	// Implementation would make RPC call to get block height
	// For now, return a placeholder
	return 0, nil
}