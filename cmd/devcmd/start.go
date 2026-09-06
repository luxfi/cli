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

	"github.com/luxfi/cli/pkg/route"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/constants"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	port        int
	nodePath    string
	logLevel    string
	cleanState  bool
	networkID   uint32
	genesisFile string
	dataDir     string
	buildTags   string
	pluginDir   string
)

const nodeBinaryName = "luxd"

// devDataDir is where the dev node keeps its state — the caller's --data-dir,
// or the anvil-compat default. start and stop resolve it the same way, so stop
// looks where start wrote.
func devDataDir() string {
	if dataDir != "" {
		return dataDir
	}
	return filepath.Join(os.Getenv("HOME"), constants.BaseDirName, constants.DevDir)
}

// devPIDFile is the node's address for `lux dev stop`.
func devPIDFile() string {
	return filepath.Join(devDataDir(), "luxd.pid")
}

// dchainBuildTag is the build tag that selects a luxd built WITH the D-Chain
// (dexvm) linked in. The public default luxd is built without it; passing
// --build-tags dchain tells `lux dev start` to launch the dchain-tagged binary
// (which bakes D-Chain on localnet 1337) and to resolve the plugin dir
// explicitly so the EVM plugin subprocess is found deterministically.
const dchainBuildTag = "dchain"

// hasBuildTag reports whether the comma-separated --build-tags value contains
// tag (space-insensitive on each element).
func hasBuildTag(tags, tag string) bool {
	for _, t := range strings.Split(tags, ",") {
		if strings.TrimSpace(t) == tag {
			return true
		}
	}
	return false
}

// resolvePluginDir picks the VM plugin directory to pass to luxd. An explicit
// --plugin-dir (explicit) always wins. For a dchain launch with no explicit
// value it defaults to <baseDir>/plugins/current (luxd's own default) so the EVM
// plugin subprocess is found deterministically. For a non-dchain launch with no
// explicit value it returns "" — luxd then uses its own default unchanged.
func resolvePluginDir(explicit, baseDir string, dchain bool) string {
	if explicit != "" {
		return explicit
	}
	if dchain {
		return filepath.Join(baseDir, constants.PluginsDir, "current")
	}
	return ""
}

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

DEX / D-Chain:
  The public default luxd does NOT include the DEX D-Chain. To launch the
  dchain-tagged luxd (which bakes D-Chain on localnet 1337) pass:
    lux dev start --build-tags dchain
  This resolves the dchain-tagged binary and points --plugin-dir at the plugin
  directory so the EVM plugin subprocess is found deterministically.

Examples:
  lux dev start                    # Start on default port 8545
  lux dev start --port 9650        # Start on custom port
  lux dev start --build-tags dchain # Start the D-Chain-enabled (dexvm) node`,
		RunE:         startDevNode,
		Args:         cobra.ExactArgs(0),
		SilenceUsage: true,
	}

	cmd.Flags().IntVar(&port, "port", 8545, "HTTP port for RPC (Anvil-compatible default)")
	cmd.Flags().StringVar(&nodePath, "node-path", "", "path to luxd binary (auto-detected if not set)")
	cmd.Flags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	cmd.Flags().BoolVar(&cleanState, "clean", false, "clean state before starting (fresh genesis)")
	cmd.Flags().Uint32Var(&networkID, "network-id", 1337, "sovereign-L1 networkID (override 1337 default)")
	cmd.Flags().StringVar(&genesisFile, "genesis-file", "", "genesis file path (uses luxd embedded if empty)")
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "luxd data-dir (default ~/.lux/devnet)")
	cmd.Flags().StringVar(&buildTags, "build-tags", "", "luxd build-tag selector; 'dchain' launches the D-Chain-enabled (dexvm) node")
	cmd.Flags().StringVar(&pluginDir, "plugin-dir", "", "VM plugin directory passed to luxd (default: luxd's own ~/.lux/plugins/current)")

	return cmd
}

// findNodeBinary locates the luxd binary. An explicit --node-path always wins.
// When dchain is true, auto-detection prefers the dchain build output
// (node/build/luxd) over a luxd on PATH, because the public PATH luxd is built
// without the D-Chain — silently launching it would yield a node with no
// D-Chain. The default (dchain=false) search order is unchanged.
func findNodeBinary(dchain bool) (string, error) {
	// Priority 1: User-provided path (explicit choice always wins).
	if nodePath != "" {
		if _, err := os.Stat(nodePath); os.IsNotExist(err) {
			return "", fmt.Errorf("%s not found at: %s", nodeBinaryName, nodePath)
		}
		return nodePath, nil
	}

	// nodeBuildDir is the standard dchain build output: <repo>/node/build/luxd,
	// resolved relative to this CLI binary. The dchain-tagged luxd is built here.
	nodeBuildBinary := func() (string, bool) {
		execPath, err := os.Executable()
		if err != nil {
			return "", false
		}
		if execPath, err = filepath.EvalSymlinks(execPath); err != nil {
			return "", false
		}
		cliDir := filepath.Dir(filepath.Dir(execPath))
		absPath, err := filepath.Abs(filepath.Join(cliDir, "..", "node", "build", nodeBinaryName))
		if err != nil {
			return "", false
		}
		if _, err := os.Stat(absPath); err != nil {
			return "", false
		}
		return absPath, true
	}

	// For a dchain launch, prefer the dchain build output BEFORE PATH/config so a
	// public PATH luxd (no D-Chain) is never picked silently.
	if dchain {
		if p, ok := nodeBuildBinary(); ok {
			return p, nil
		}
	}

	// Priority 2: Environment/config
	if configPath := viper.GetString(constants.ConfigNodePath); configPath != "" {
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

	// Priority 4: Relative to CLI (node/build/luxd).
	if p, ok := nodeBuildBinary(); ok {
		return p, nil
	}

	if dchain {
		return "", fmt.Errorf("dchain-tagged %s not found. Build it (cd node && go build -tags dchain -o build/luxd ./main) or set --node-path", nodeBinaryName)
	}
	return "", fmt.Errorf("%s not found. Set --node-path or add to PATH", nodeBinaryName)
}

func startDevNode(*cobra.Command, []string) error {
	ux.Logger.PrintToUser("Starting Lux dev node (K=1 consensus)...")

	dchain := hasBuildTag(buildTags, dchainBuildTag)

	localNodePath, err := findNodeBinary(dchain)
	if err != nil {
		return err
	}

	// Data directories - use constants for consistent paths
	baseDir := filepath.Join(os.Getenv("HOME"), constants.BaseDirName)
	dataDir = devDataDir()
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
	if err := os.MkdirAll(logDir, 0o750); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	stakingPort := port + 1

	// Resolve the VM plugin directory (the capstone's missing --plugin-dir
	// blocker): explicit wins; a dchain launch defaults to ~/.lux/plugins/current
	// so the EVM plugin subprocess is found; a non-dchain launch stays empty and
	// lets luxd use its own default unchanged.
	effPluginDir := resolvePluginDir(pluginDir, baseDir, dchain)

	ux.Logger.PrintToUser("Binary: %s", localNodePath)
	if dchain {
		ux.Logger.PrintToUser("Build tags: dchain (D-Chain / dexvm enabled)")
	}
	ux.Logger.PrintToUser("Port: %d (staking: %d)", port, stakingPort)

	// Build luxd command. luxd has no `--dev` shortcut, so we spell out the
	// K=1, no-bootstrap, no-sybil-protection profile explicitly. --automine
	// supplies single-validator-quorum consensus and instant finality. Block
	// cadence is not set here: it is the C-Chain's own enable-automining, read
	// by the block builder from the chain config dir.
	// Chain config dir - luxd's --chain-config-dir points here.
	// Uses ~/.lux/chains/ for all chain configs (genesis, config.json, etc.)
	chainConfigDir := filepath.Join(baseDir, constants.ChainsDir)
	args := []string{
		"--automine",
		"--consensus-sample-size=1",
		"--consensus-quorum-size=1",
		"--sybil-protection-enabled=false",
		"--skip-bootstrap=true",
		fmt.Sprintf("--network-id=%d", networkID),
		fmt.Sprintf("--http-host=%s", "0.0.0.0"),
		fmt.Sprintf("--http-port=%d", port),
		fmt.Sprintf("--staking-port=%d", stakingPort),
		fmt.Sprintf("--data-dir=%s", dataDir),
		fmt.Sprintf("--log-dir=%s", logDir),
		fmt.Sprintf("--log-level=%s", logLevel),
		fmt.Sprintf("--chain-config-dir=%s", chainConfigDir),
		"--api-admin-enabled=true",
		"--index-enabled=true",
		"--track-all-chains=true",
	}
	if genesisFile != "" {
		args = append(args, fmt.Sprintf("--genesis-file=%s", genesisFile))
	}
	if effPluginDir != "" {
		args = append(args, fmt.Sprintf("--plugin-dir=%s", effPluginDir))
		ux.Logger.PrintToUser("Plugin dir: %s", effPluginDir)
	}

	cmd := exec.Command(localNodePath, args...) //nolint:gosec // G204: Running our own node binary
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start luxd: %w", err)
	}

	// Save PID file for later use by 'lux dev stop' and network detection
	pidFile := devPIDFile()
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0o644); err != nil { //nolint:gosec // G306: PID file needs to be readable
		ux.Logger.PrintToUser("Warning: failed to save PID file: %v", err)
	}

	ux.Logger.PrintToUser("luxd started (PID: %d)", cmd.Process.Pid)

	// Wait for health with explicit timeout (60 seconds for all chains to bootstrap)
	nodeURL := fmt.Sprintf("http://localhost:%d", port)
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
			resp, err := http.Get(route.Readiness(nodeURL))
			if err != nil {
				continue // Network not ready yet
			}
			_ = resp.Body.Close()
			if resp.StatusCode != 200 {
				continue
			}
			// Additional check: verify C-Chain is responding
			cchainURL := route.Chain(fmt.Sprintf("http://localhost:%d", port), "C") + "/rpc"
			cResp, cErr := http.Post(cchainURL, "application/json",
				strings.NewReader(`{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}`))
			if cErr != nil {
				continue
			}
			_ = cResp.Body.Close()
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
	uri := fmt.Sprintf("http://localhost:%d", port)
	ws := fmt.Sprintf("ws://localhost:%d", port)
	ux.Logger.PrintToUser("  C-Chain RPC:  %s", route.Chain(uri, "C")+"/rpc")
	ux.Logger.PrintToUser("  C-Chain WS:   %s", route.Chain(ws, "C")+"/ws")
	ux.Logger.PrintToUser("  P-Chain:      %s", route.Chain(uri, "P"))
	ux.Logger.PrintToUser("  X-Chain:      %s", route.Chain(uri, "X"))
	ux.Logger.PrintToUser("  T-Chain:      %s", route.Chain(uri, "T"))
	if dchain {
		ux.Logger.PrintToUser("  D-Chain:      %s", route.Chain(uri, "D"))
	}
	ux.Logger.PrintToUser("  Health:       %s", route.Health(nodeURL))
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

	// Wait for process (foreground mode — user backgrounds with shell `&` if needed)
	return cmd.Wait()
}
