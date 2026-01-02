// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package devcmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	cliconstants "github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	port       int    // HTTP port (default 8545, Anvil-compatible)
	automine   string // Automine delay (empty = instant, "1s" = 1 block per second, etc.)
	nodePath   string // Path to custom luxd binary
	logLevel   string // Log level (info, debug, warn, error)
	cleanState bool   // Clean state before starting
)

const nodeBinaryName = "luxd"

func newStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start local dev node",
		Long: `Start a single-node Lux development network.

The dev node uses K=1 consensus for instant block finality without
validator sampling. All chains are enabled with full validator signing:
  • C-Chain: EVM-compatible smart contracts
  • P-Chain: Platform staking and validation
  • X-Chain: UTXO-based asset exchange
  • T-Chain: Threshold FHE operations

Default port is 8545 (Anvil-compatible) so it works seamlessly with
Hardhat, Foundry, and other Ethereum tooling.

FHE Support:
  The T-Chain provides threshold homomorphic encryption for confidential
  smart contracts. Use FHE precompiles at 0x0200...0080 or the @luxfi/fhe SDK.

Examples:
  lux dev start                    # Start on default port 8545
  lux dev start --port 9650        # Start on custom port
  lux dev start --automine 1s      # Mine blocks every 1 second
  lux dev start --automine 500ms   # Mine blocks every 500ms`,
		RunE:         startDevNode,
		Args:         cobra.ExactArgs(0),
		SilenceUsage: true,
	}

	cmd.Flags().IntVar(&port, "port", 8545, "HTTP port for RPC (Anvil-compatible default)")
	cmd.Flags().StringVar(&automine, "automine", "", "auto-mine interval (e.g., '1s', '500ms'); empty = mine as blocks arrive")
	cmd.Flags().StringVar(&nodePath, "node-path", "", "path to luxd binary (auto-detected if not set)")
	cmd.Flags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	cmd.Flags().BoolVar(&cleanState, "clean", false, "clean state before starting (fresh genesis)")

	return cmd
}

// findNodeBinary locates the luxd binary
func findNodeBinary() (string, error) {
	// Priority 1: User-provided path
	if nodePath != "" {
		if _, err := os.Stat(nodePath); os.IsNotExist(err) {
			return "", fmt.Errorf("%s not found at: %s", nodeBinaryName, nodePath)
		}
		return nodePath, nil
	}

	// Priority 2: Environment/config
	if configPath := viper.GetString(cliconstants.ConfigNodePath); configPath != "" {
		if strings.HasPrefix(configPath, "~") {
			home, _ := os.UserHomeDir()
			configPath = filepath.Join(home, configPath[1:])
		}
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
	}

	// Priority 3: PATH
	if binaryPath, err := exec.LookPath(nodeBinaryName); err == nil {
		return binaryPath, nil
	}

	// Priority 4: Relative to CLI
	if execPath, err := os.Executable(); err == nil {
		if execPath, err = filepath.EvalSymlinks(execPath); err == nil {
			cliDir := filepath.Dir(filepath.Dir(execPath))
			relativePath := filepath.Join(cliDir, "..", "node", "build", nodeBinaryName)
			if absPath, err := filepath.Abs(relativePath); err == nil {
				if _, err := os.Stat(absPath); err == nil {
					return absPath, nil
				}
			}
		}
	}

	return "", fmt.Errorf("%s not found. Set --node-path or add to PATH", nodeBinaryName)
}

func startDevNode(*cobra.Command, []string) error {
	ux.Logger.PrintToUser("Starting Lux dev node (K=1 consensus)...")

	localNodePath, err := findNodeBinary()
	if err != nil {
		return err
	}

	// Data directories - use constants for consistent paths
	baseDir := filepath.Join(os.Getenv("HOME"), cliconstants.BaseDirName)
	dataDir := filepath.Join(baseDir, cliconstants.DevDir)
	dbDir := filepath.Join(dataDir, "db")
	logDir := filepath.Join(dataDir, "logs")

	// Clean state if requested or if db doesn't exist
	if cleanState {
		ux.Logger.PrintToUser("Cleaning dev state...")
		if err := os.RemoveAll(dbDir); err != nil {
			ux.Logger.PrintToUser("Warning: failed to clean database: %v", err)
		}
	}

	// Ensure directories exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	stakingPort := port + 1

	ux.Logger.PrintToUser("Binary: %s", localNodePath)
	ux.Logger.PrintToUser("Port: %d (staking: %d)", port, stakingPort)

	// Build luxd command with --dev flag
	// The --dev flag configures K=1 consensus automatically:
	// - consensus-sample-size=1, consensus-quorum-size=1
	// - skip-bootstrap=true
	// - sybil-protection-enabled=false
	// - ephemeral staking certs
	// Chain config dir - luxd's --chain-config-dir points here
	// Uses ~/.lux/chains/ for all chain configs (genesis, config.json, etc.)
	chainConfigDir := filepath.Join(baseDir, cliconstants.ChainsDir)
	args := []string{
		"--dev",
		fmt.Sprintf("--network-id=%d", 1337),
		fmt.Sprintf("--http-host=%s", "0.0.0.0"),
		fmt.Sprintf("--http-port=%d", port),
		fmt.Sprintf("--staking-port=%d", stakingPort),
		fmt.Sprintf("--data-dir=%s", dataDir),
		fmt.Sprintf("--log-dir=%s", logDir),
		fmt.Sprintf("--log-level=%s", logLevel),
		fmt.Sprintf("--chain-config-dir=%s", chainConfigDir), // Read chain configs (dexConfig, etc.)
		"--api-admin-enabled=true",
		"--api-keystore-enabled=true",
		"--index-enabled=true",
		"--track-all-chains=true", // Enable ALL chains: A,B,C,D,G,K,P,Q,T,X,Z
	}

	// Add automine configuration if specified
	if automine != "" {
		// Parse to validate the duration format
		duration, err := time.ParseDuration(automine)
		if err != nil {
			return fmt.Errorf("invalid --automine value '%s': %w", automine, err)
		}
		// luxd expects milliseconds for automining interval
		args = append(args, fmt.Sprintf("--automine-interval=%d", duration.Milliseconds()))
		ux.Logger.PrintToUser("Automine: %s interval", automine)
	} else {
		ux.Logger.PrintToUser("Automine: instant (as blocks arrive)")
	}

	cmd := exec.Command(localNodePath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start luxd: %w", err)
	}

	// Save PID file for later use by 'lux dev stop' and network detection
	pidFile := filepath.Join(dataDir, "luxd.pid")
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644); err != nil {
		ux.Logger.PrintToUser("Warning: failed to save PID file: %v", err)
	}

	ux.Logger.PrintToUser("luxd started (PID: %d)", cmd.Process.Pid)

	// Wait for health with explicit timeout (60 seconds for all chains to bootstrap)
	healthURL := fmt.Sprintf("http://localhost:%d/ext/health", port)
	healthTimeout := 60 * time.Second
	healthCtx, healthCancel := context.WithTimeout(context.Background(), healthTimeout)
	defer healthCancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-healthCtx.Done():
			return fmt.Errorf("timeout waiting for node to become healthy after %s: %w", healthTimeout, healthCtx.Err())
		case <-ticker.C:
			resp, err := http.Get(healthURL)
			if err != nil {
				continue // Network not ready yet
			}
			resp.Body.Close()
			if resp.StatusCode != 200 {
				continue
			}
			// Additional check: verify C-Chain is responding
			cchainURL := fmt.Sprintf("http://localhost:%d/ext/bc/C/rpc", port)
			cResp, cErr := http.Post(cchainURL, "application/json",
				strings.NewReader(`{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}`))
			if cErr != nil {
				continue
			}
			cResp.Body.Close()
			if cResp.StatusCode == 200 {
				goto healthy
			}
		}
	}
healthy:

	// Print success info
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Dev node ready!")
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Endpoints:")
	ux.Logger.PrintToUser("  C-Chain RPC:  http://localhost:%d/ext/bc/C/rpc", port)
	ux.Logger.PrintToUser("  C-Chain WS:   ws://localhost:%d/ext/bc/C/ws", port)
	ux.Logger.PrintToUser("  P-Chain:      http://localhost:%d/ext/bc/P", port)
	ux.Logger.PrintToUser("  X-Chain:      http://localhost:%d/ext/bc/X", port)
	ux.Logger.PrintToUser("  T-Chain:      http://localhost:%d/ext/bc/T", port)
	ux.Logger.PrintToUser("  Health:       http://localhost:%d/ext/health", port)
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Features:")
	ux.Logger.PrintToUser("  • K=1 consensus (instant finality)")
	ux.Logger.PrintToUser("  • Full validator signing")
	ux.Logger.PrintToUser("  • All chains: C/P/X/T enabled")
	ux.Logger.PrintToUser("  • Chain ID: 1337")
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("FHE Precompiles (C-Chain):")
	ux.Logger.PrintToUser("  • FHEOS:    0x0200000000000000000000000000000000000080")
	ux.Logger.PrintToUser("  • ACL:      0x0200000000000000000000000000000000000081")
	ux.Logger.PrintToUser("  • Verifier: 0x0200000000000000000000000000000000000082")
	ux.Logger.PrintToUser("  • Gateway:  0x0200000000000000000000000000000000000083")
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Data: %s", dataDir)
	ux.Logger.PrintToUser("Logs: %s", logDir)
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Stop with: lux dev stop")

	// Wait for process
	return cmd.Wait()
}
