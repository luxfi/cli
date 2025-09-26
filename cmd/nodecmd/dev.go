// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package nodecmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

type devFlags struct {
	instanceID  int
	httpPort    int
	stakingPort int
	dataDir     string
	chainID     uint32
	automine    bool
	blockTime   int
	accounts    []string
	balance     string
}

func newDevCmd() *cobra.Command {
	flags := &devFlags{}

	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Start Lux node in development mode",
		Long: `Starts a Lux node in development mode with single-node operation,
similar to 'geth --dev'. This mode includes:
- No bootstrapping required
- Single validator setup
- Optional automining
- Pre-funded test accounts
- Instant finality`,
		Example: `  # Start dev mode with default settings
  lux node dev

  # Start with automining enabled
  lux node dev --automine

  # Start on custom ports
  lux node dev --http-port 8545 --staking-port 8546

  # Start multiple instances
  lux node dev --instance 2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDev(flags)
		},
	}

	cmd.Flags().IntVar(&flags.instanceID, "instance", 1, "Instance ID for running multiple nodes")
	cmd.Flags().IntVar(&flags.httpPort, "http-port", 9630, "HTTP API port")
	cmd.Flags().IntVar(&flags.stakingPort, "staking-port", 9631, "Staking port")
	cmd.Flags().StringVar(&flags.dataDir, "data-dir", "", "Data directory (default: temp directory)")
	cmd.Flags().Uint32Var(&flags.chainID, "chain-id", 96369, "Chain ID for C-Chain")
	cmd.Flags().BoolVar(&flags.automine, "automine", true, "Enable automining")
	cmd.Flags().IntVar(&flags.blockTime, "block-time", 1, "Block time in seconds (for automining)")
	cmd.Flags().StringSliceVar(&flags.accounts, "accounts", []string{"0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"}, "Pre-funded accounts")
	cmd.Flags().StringVar(&flags.balance, "balance", "1000000", "Initial balance in LUX for accounts")

	return cmd
}

func runDev(flags *devFlags) error {
	ux.Logger.PrintToUser("Starting Lux node in development mode...")

	// Adjust ports based on instance ID
	if flags.instanceID > 1 {
		flags.httpPort += (flags.instanceID - 1) * 10
		flags.stakingPort += (flags.instanceID - 1) * 10
	}

	// Create data directory
	if flags.dataDir == "" {
		flags.dataDir = filepath.Join(os.TempDir(), fmt.Sprintf("lux-dev-%d", flags.instanceID))
	}

	// Ensure directories exist
	dirs := []string{
		filepath.Join(flags.dataDir, "staking"),
		filepath.Join(flags.dataDir, "configs", "chains", "C"),
		filepath.Join(flags.dataDir, "chainData"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Generate ephemeral staking credentials
	if err := generateStakingCredentials(flags.dataDir); err != nil {
		return fmt.Errorf("failed to generate staking credentials: %w", err)
	}

	// Create C-Chain config with automining settings
	if err := createCChainConfig(flags); err != nil {
		return fmt.Errorf("failed to create C-Chain config: %w", err)
	}

	// Create genesis with pre-funded accounts
	if err := createDevGenesis(flags); err != nil {
		return fmt.Errorf("failed to create genesis: %w", err)
	}

	// Build luxd command
	luxdPath := filepath.Join(app.GetBaseDir(), "bin", "luxd")
	if _, err := os.Stat(luxdPath); os.IsNotExist(err) {
		// Try the node build directory
		luxdPath = filepath.Join(app.GetBaseDir(), "..", "..", "node", "build", "luxd")
		if _, err := os.Stat(luxdPath); os.IsNotExist(err) {
			return fmt.Errorf("luxd binary not found. Please build it first with './scripts/build.sh'")
		}
	}

	args := []string{
		"--network-id", fmt.Sprintf("%d", flags.chainID),
		"--data-dir", flags.dataDir,
		"--db-dir", filepath.Join(flags.dataDir, "db"),
		"--staking-tls-key-file", filepath.Join(flags.dataDir, "staking", "staker.key"),
		"--staking-tls-cert-file", filepath.Join(flags.dataDir, "staking", "staker.crt"),
		"--chain-config-dir", filepath.Join(flags.dataDir, "configs", "chains"),
		"--genesis-file", filepath.Join(flags.dataDir, "genesis.json"),
		"--skip-bootstrap",
		"--staking-enabled=false",
		"--sybil-protection-enabled=false",
		"--consensus-sample-size=1",
		"--consensus-quorum-size=1",
		"--public-ip=127.0.0.1",
		"--http-host=0.0.0.0",
		"--http-port", fmt.Sprintf("%d", flags.httpPort),
		"--staking-port", fmt.Sprintf("%d", flags.stakingPort),
		"--api-admin-enabled=true",
		"--api-keystore-enabled=true",
		"--api-metrics-enabled=true",
		"--index-enabled=true",
		"--log-level=info",
	}

	cmd := exec.Command(luxdPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	ux.Logger.PrintToUser("Dev Mode Configuration:")
	ux.Logger.PrintToUser("- Instance ID: %d", flags.instanceID)
	ux.Logger.PrintToUser("- HTTP Port: %d", flags.httpPort)
	ux.Logger.PrintToUser("- Staking Port: %d", flags.stakingPort)
	ux.Logger.PrintToUser("- Chain ID: %d", flags.chainID)
	ux.Logger.PrintToUser("- Data Directory: %s", flags.dataDir)
	ux.Logger.PrintToUser("- Automining: %v", flags.automine)
	if flags.automine {
		ux.Logger.PrintToUser("- Block Time: %d seconds", flags.blockTime)
	}
	ux.Logger.PrintToUser("- Pre-funded Accounts: %v", flags.accounts)
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Starting luxd...")

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start luxd: %w", err)
	}

	ux.Logger.PrintToUser("Node started with PID: %d", cmd.Process.Pid)

	// Wait for node to initialize
	time.Sleep(10 * time.Second)

	// Display connection information
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Connection Information:")
	ux.Logger.PrintToUser("- RPC URL: http://localhost:%d/ext/bc/C/rpc", flags.httpPort)
	ux.Logger.PrintToUser("- WebSocket URL: ws://localhost:%d/ext/bc/C/ws", flags.httpPort)
	ux.Logger.PrintToUser("- Chain ID: %d", flags.chainID)
	ux.Logger.PrintToUser("- Dev Account: %s", flags.accounts[0])
	ux.Logger.PrintToUser("- Private Key: 56289e99c94b6912bfc12adc093c9b51124f0dc54ac7a766b2bc5ccf558d8027")
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("To stop the node, press Ctrl+C")

	// Wait for the process
	return cmd.Wait()
}

func generateStakingCredentials(dataDir string) error {
	// For dev mode, we use fixed ephemeral credentials
	stakingDir := filepath.Join(dataDir, "staking")

	keyPath := filepath.Join(stakingDir, "staker.key")
	certPath := filepath.Join(stakingDir, "staker.crt")

	// Dev mode staking key
	key := `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgWRQr2aIqVmXJIqSK
oLmJLqv1HqP4h1XuJopnYdT9KROhRANCAAQKRdbyne7H1M7nz2hEoMqjfFRXLaVl
qcr7sLvSk/bPLOYdmKR5s5B9fS3TCoNEL9fEp2xz0UbpVxK3z7T2tLWj
-----END PRIVATE KEY-----`

	// Dev mode staking cert
	cert := `-----BEGIN CERTIFICATE-----
MIIBwzCCAWqgAwIBAgIJAJmtmKQYj0GsMAoGCCqGSM49BAMCMDwxFDASBgNVBAMM
C2F2YWxhbmNoZWdvMQ0wCwYDVQQKDARhdmFsMQwwCgYDVQQHDANhdmExCzAJBgNV
BAYTAlVTMB4XDTIwMDcxNTIxMTAyNloXDTMwMDcxMzIxMTAyNlowPDEUMBIGA1UE
AwwLYXZhbGFuY2hlZ28xDTALBgNVBAoMBGF2YWwxDDAKBgNVBAcMA2F2YTELMAkG
A1UEBhMCVVMwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAQKRdbyne7H1M7nz2hE
oMqjfFRXLaVlqcr7sLvSk/bPLOYdmKR5s5B9fS3TCoNEL9fEp2xz0UbpVxK3z7T2
tLWjozUwMzAMBgNVHRMBAf8EAjAAMCMGA1UdEQQcMBqCCWxvY2FsaG9zdIcEfwAA
AYcECgAAAYcEwKgAATAKBggqhkjOPQQDAgNHADBEAiB5NLOtpWn6xnYAaLKQNqaZ
jIx4eNBEerJtA2hMqGEQvAIgVDD+NYn6K/B7gNqBi7efvBg0OYdmf0Ij3yPWGWdX
Cqc=
-----END CERTIFICATE-----`

	if err := os.WriteFile(keyPath, []byte(key), 0600); err != nil {
		return err
	}

	return os.WriteFile(certPath, []byte(cert), 0644)
}

func createCChainConfig(flags *devFlags) error {
	config := map[string]interface{}{
		"linear-api-enabled":        false,
		"geth-admin-api-enabled":    true,
		"eth-apis":                  []string{"eth", "eth-filter", "net", "web3", "admin", "debug", "personal", "txpool", "miner"},
		"local-txs-enabled":         true,
		"allow-unfinalized-queries": true,
		"allow-unprotected-txs":     true,
		"log-level":                 "info",
		"pruning-enabled":           false,
		"metrics-enabled":           true,
		"tx-lookup-limit":           0,
	}

	if flags.automine {
		config["dev-mode"] = true
		config["dev-etherbase"] = flags.accounts[0]
		config["dev-gas-limit"] = 99999999
		config["dev-period"] = flags.blockTime
	}

	configPath := filepath.Join(flags.dataDir, "configs", "chains", "C", "config.json")
	return writeJSON(configPath, config)
}

func createDevGenesis(flags *devFlags) error {
	// Create allocations for pre-funded accounts
	alloc := make(map[string]interface{})
	for _, account := range flags.accounts {
		// Remove 0x prefix if present
		addr := account
		if len(addr) > 2 && addr[:2] == "0x" {
			addr = addr[2:]
		}
		alloc[addr] = map[string]string{
			"balance": "0x33b2e3c9fd0803ce8000000", // 1000000 ETH in wei
		}
	}

	genesis := map[string]interface{}{
		"networkID": flags.chainID,
		"allocations": []map[string]interface{}{
			{
				"ethAddr":        flags.accounts[0],
				"luxAddr":        "X-lux1npswupzlgs3kng2q965as2la8rw4787hcn9p7q",
				"initialAmount":  "1000000000000000000000000000",
				"unlockSchedule": []interface{}{},
			},
		},
		"startTime":                  1625072400,
		"initialStakeDuration":       31536000,
		"initialStakeDurationOffset": 5400,
		"initialStakedFunds":         []interface{}{},
		"initialStakers":             []interface{}{},
		"cChainGenesis": map[string]interface{}{
			"config": map[string]interface{}{
				"chainId":                     flags.chainID,
				"homesteadBlock":              0,
				"eip150Block":                 0,
				"eip155Block":                 0,
				"eip158Block":                 0,
				"byzantiumBlock":              0,
				"constantinopleBlock":         0,
				"petersburgBlock":             0,
				"istanbulBlock":               0,
				"muirGlacierBlock":            0,
				"apricotPhase1BlockTimestamp": 0,
				"apricotPhase2BlockTimestamp": 0,
				"apricotPhase3BlockTimestamp": 0,
				"apricotPhase4BlockTimestamp": 0,
				"apricotPhase5BlockTimestamp": 0,
			},
			"nonce":      "0x0",
			"timestamp":  "0x0",
			"extraData":  "0x00",
			"gasLimit":   "0x5f5e100",
			"difficulty": "0x0",
			"mixHash":    "0x0000000000000000000000000000000000000000000000000000000000000000",
			"coinbase":   "0x0000000000000000000000000000000000000000",
			"alloc":      alloc,
			"number":     "0x0",
			"gasUsed":    "0x0",
			"parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
		},
		"message": "Lux Dev Mode",
	}

	genesisPath := filepath.Join(flags.dataDir, "genesis.json")
	return writeJSON(genesisPath, genesis)
}

func writeJSON(path string, data interface{}) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}
