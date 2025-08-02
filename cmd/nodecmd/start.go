// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package nodecmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	
	"github.com/luxfi/cli/v2/pkg/ux"
	"github.com/spf13/cobra"
)

type startFlags struct {
	networkID          uint32
	dataDir            string
	httpPort           int
	stakingPort        int
	skipBootstrap      bool
	enableAutomining   bool
	stakingEnabled     bool
	sybilProtection    bool
	consensusSampleSize     int
	consensusQuorumSize     int
	publicIP           string
	logLevel           string
	chainConfigDir     string
	genesisFile        string
	existingDataDir    string
	importMode         bool
}

func newStartCmd() *cobra.Command {
	flags := &startFlags{}
	
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start Lux node with custom configuration",
		Long: `Start a Lux node with custom configuration options.
This command provides fine-grained control over node startup parameters.`,
		Example: `  # Start mainnet node with skip-bootstrap
  lux node start --network-id 96369 --skip-bootstrap

  # Start with existing data
  lux node start --existing-data /path/to/data

  # Start with custom ports
  lux node start --http-port 8545 --staking-port 8546`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart(flags)
		},
	}

	// Network configuration
	cmd.Flags().Uint32Var(&flags.networkID, "network-id", 96369, "Network ID")
	cmd.Flags().StringVar(&flags.dataDir, "data-dir", "", "Data directory (default: ~/.luxd)")
	cmd.Flags().IntVar(&flags.httpPort, "http-port", 9630, "HTTP API port")
	cmd.Flags().IntVar(&flags.stakingPort, "staking-port", 9631, "Staking port")
	
	// Bootstrap and consensus
	cmd.Flags().BoolVar(&flags.skipBootstrap, "skip-bootstrap", false, "Skip bootstrapping phase")
	cmd.Flags().BoolVar(&flags.enableAutomining, "enable-automining", false, "Enable automining in POA mode")
	cmd.Flags().BoolVar(&flags.stakingEnabled, "staking-enabled", true, "Enable staking")
	cmd.Flags().BoolVar(&flags.sybilProtection, "sybil-protection", true, "Enable sybil protection")
	cmd.Flags().IntVar(&flags.consensusSampleSize, "consensus-sample-size", 20, "Consensus sample size")
	cmd.Flags().IntVar(&flags.consensusQuorumSize, "consensus-quorum-size", 14, "Consensus quorum size")
	
	// Advanced configuration
	cmd.Flags().StringVar(&flags.publicIP, "public-ip", "", "Public IP address")
	cmd.Flags().StringVar(&flags.logLevel, "log-level", "info", "Log level")
	cmd.Flags().StringVar(&flags.chainConfigDir, "chain-config-dir", "", "Chain config directory")
	cmd.Flags().StringVar(&flags.genesisFile, "genesis-file", "", "Custom genesis file")
	cmd.Flags().StringVar(&flags.existingDataDir, "existing-data", "", "Use existing data directory")
	cmd.Flags().BoolVar(&flags.importMode, "import-mode", false, "Enable import mode for one-time blockchain data import with pruning disabled")
	
	return cmd
}

func runStart(flags *startFlags) error {
	ux.Logger.PrintToUser("Starting Lux node...")
	
	// Find luxd binary - check multiple locations
	var luxdPath string
	possiblePaths := []string{
		filepath.Join(app.GetBaseDir(), "bin", "luxd"),
		filepath.Join(app.GetBaseDir(), "..", "node", "build", "luxd"),
		filepath.Join(app.GetBaseDir(), "..", "..", "node", "build", "luxd"),
		"/home/z/work/lux/node/build/luxd",
	}
	
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			luxdPath = path
			break
		}
	}
	
	if luxdPath == "" {
		return fmt.Errorf("luxd binary not found. Please install with 'lux node version install' or build from source")
	}
	
	// Determine data directory
	dataDir := flags.dataDir
	if dataDir == "" {
		if flags.existingDataDir != "" {
			dataDir = flags.existingDataDir
		} else {
			home, _ := os.UserHomeDir()
			dataDir = filepath.Join(home, ".luxd")
		}
	}
	
	// Build command arguments
	args := []string{
		"--network-id", fmt.Sprintf("%d", flags.networkID),
		"--http-port", fmt.Sprintf("%d", flags.httpPort),
		"--staking-port", fmt.Sprintf("%d", flags.stakingPort),
		"--log-level", flags.logLevel,
	}
	
	if flags.dataDir != "" {
		args = append(args, "--data-dir", flags.dataDir)
	}
	
	if flags.skipBootstrap {
		args = append(args, "--skip-bootstrap")
	}
	
	if flags.enableAutomining {
		args = append(args, "--enable-automining")
	}
	
	if !flags.stakingEnabled {
		args = append(args, "--staking-enabled=false")
	}
	
	if !flags.sybilProtection {
		args = append(args, "--sybil-protection-enabled=false")
	}
	
	if flags.consensusSampleSize != 20 {
		args = append(args, "--consensus-sample-size", fmt.Sprintf("%d", flags.consensusSampleSize))
	}
	
	if flags.consensusQuorumSize != 14 {
		args = append(args, "--consensus-quorum-size", fmt.Sprintf("%d", flags.consensusQuorumSize))
	}
	
	if flags.publicIP != "" {
		args = append(args, "--public-ip", flags.publicIP)
	}
	
	if flags.chainConfigDir != "" {
		args = append(args, "--chain-config-dir", flags.chainConfigDir)
	}
	
	if flags.genesisFile != "" {
		args = append(args, "--genesis-file", flags.genesisFile)
	}
	
	if flags.importMode {
		args = append(args, "--import-mode")
	}
	
	// Always enable useful APIs
	args = append(args,
		"--api-admin-enabled=true",
		"--api-keystore-enabled=true",
		"--api-metrics-enabled=true",
		"--index-enabled=false",
		"--index-allow-incomplete=true",
		"--http-host=0.0.0.0",
	)
	
	// Create and start the command
	cmd := exec.Command(luxdPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	ux.Logger.PrintToUser("Configuration:")
	ux.Logger.PrintToUser("- Network ID: %d", flags.networkID)
	ux.Logger.PrintToUser("- HTTP Port: %d", flags.httpPort)
	ux.Logger.PrintToUser("- Staking Port: %d", flags.stakingPort)
	ux.Logger.PrintToUser("- Data Directory: %s", dataDir)
	ux.Logger.PrintToUser("- Skip Bootstrap: %v", flags.skipBootstrap)
	ux.Logger.PrintToUser("- Enable Automining: %v", flags.enableAutomining)
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Command: %s %v", luxdPath, args)
	ux.Logger.PrintToUser("")
	
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start luxd: %w", err)
	}
	
	ux.Logger.PrintToUser("Node started with PID: %d", cmd.Process.Pid)
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("RPC Endpoints:")
	ux.Logger.PrintToUser("- Info: http://localhost:%d/ext/info", flags.httpPort)
	ux.Logger.PrintToUser("- C-Chain: http://localhost:%d/ext/bc/C/rpc", flags.httpPort)
	ux.Logger.PrintToUser("- X-Chain: http://localhost:%d/ext/bc/X", flags.httpPort)
	ux.Logger.PrintToUser("- P-Chain: http://localhost:%d/ext/bc/P", flags.httpPort)
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("To stop the node, press Ctrl+C")
	
	return cmd.Wait()
}