// Package upcmd implements `lux up <network>/<env>` — the one boot verb.
//
// Boots a sovereign-L1 luxd node identified by (name, env). Reads
// chain.yaml under $LUX_NETWORK_PATH, applies the runtime template,
// verifies/writes a chain.lock manifest in the data-dir, then exec's
// luxd in K=1 PoA mode bound to the resolved httpPort/stakingPort.
//
// Foreground only. Background with shell `&` or tmux; the CLI does not
// supervise. Identity for `lux down` is the (port + networkID)
// handshake, not a PID file.
package upcmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/luxfi/cli/pkg/network"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	cleanState bool
	nodePath   string
	automine   string
)

// NewCmd returns `lux up`.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up <network>/<env>",
		Short: "Boot a sovereign-L1 network instance (foreground)",
		Long: `Boots the luxd node for <network>/<env> in K=1 PoA mode.

The (network, env) tuple identifies one L1 instance globally. Every
parameter — port, networkID, dataDir, genesisFile — is derived from that
network's chain.yaml under $LUX_NETWORK_PATH. Foreground: ` + "`lux up`" + `
blocks until the node exits or Ctrl-C is pressed.

Examples:
  lux up zoo/localnet                 # K=1 dev node for Zoo, networkID 200203
  lux up lux/devnet                   # K=1 dev node for Lux, networkID 3
  lux up hanzo/testnet --clean        # wipe state then boot

Stop with: lux down <network>/<env>`,
		Args:         cobra.ExactArgs(1),
		RunE:         runUp,
		SilenceUsage: true,
	}
	cmd.Flags().BoolVar(&cleanState, "clean", false, "remove data-dir before boot (fresh genesis)")
	cmd.Flags().StringVar(&nodePath, "node-path", "", "path to luxd binary (auto-detected if empty)")
	cmd.Flags().StringVar(&automine, "automine", "", "auto-mine interval (e.g., '1s'); empty = instant")
	return cmd
}

func runUp(cmd *cobra.Command, args []string) error {
	prof, err := network.Resolve(args[0])
	if err != nil {
		return err
	}
	ux.Logger.PrintToUser("up %s — networkID=%d port=%d", prof, prof.NetworkID, prof.HTTPPort)
	ux.Logger.PrintToUser("  data-dir: %s", prof.DataDir)
	if prof.GenesisFile != "" {
		ux.Logger.PrintToUser("  genesis : %s", prof.GenesisFile)
	}

	if cleanState {
		ux.Logger.PrintToUser("  clean: removing %s", prof.DataDir)
		if err := os.RemoveAll(prof.DataDir); err != nil {
			return err
		}
	}

	// Write/verify chain.lock BEFORE any other dir entries so the
	// "orphan dataDir" check sees an empty dir on first boot.
	var genesisHash string
	if prof.GenesisFile != "" {
		if data, err := os.ReadFile(prof.GenesisFile); err == nil { //nolint:gosec
			genesisHash = network.HashGenesis(data)
		}
	}
	if err := prof.VerifyOrCreate(genesisHash); err != nil {
		return err
	}

	if err := os.MkdirAll(prof.LogDir, 0o750); err != nil {
		return err
	}

	luxd, err := findLuxd(nodePath)
	if err != nil {
		return err
	}

	args2 := []string{
		"--automine",
		"--consensus-sample-size=1",
		"--consensus-quorum-size=1",
		"--sybil-protection-enabled=false",
		"--skip-bootstrap=true",
		"--network-id=" + strconv.FormatUint(uint64(prof.NetworkID), 10),
		"--http-host=0.0.0.0",
		"--http-port=" + strconv.Itoa(prof.HTTPPort),
		"--staking-port=" + strconv.Itoa(prof.StakingPort),
		"--data-dir=" + prof.DataDir,
		"--log-dir=" + prof.LogDir,
		"--log-level=" + prof.LogLevel,
		"--api-admin-enabled=true",
		"--api-keystore-enabled=true",
		"--index-enabled=true",
		"--track-all-chains=true",
	}
	if prof.GenesisFile != "" {
		args2 = append(args2, "--genesis-file="+prof.GenesisFile)
	}
	if automine != "" {
		d, err := time.ParseDuration(automine)
		if err != nil {
			return fmt.Errorf("--automine %q: %w", automine, err)
		}
		args2 = append(args2, fmt.Sprintf("--automine-interval=%d", d.Milliseconds()))
	}

	c := exec.Command(luxd, args2...) //nolint:gosec
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Start(); err != nil {
		return fmt.Errorf("start luxd: %w", err)
	}

	// Best-effort PID file as a stop fallback. Identity-by-port-probe
	// remains primary.
	pidFile := prof.PIDFilePath()
	_ = os.WriteFile(pidFile, []byte(strconv.Itoa(c.Process.Pid)), 0o600)

	ux.Logger.PrintToUser("  luxd PID=%d", c.Process.Pid)
	ux.Logger.PrintToUser("  RPC: %s", prof.LocalRPCUrl)

	// Forward SIGINT/SIGTERM to luxd so Ctrl-C does a graceful stop.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		_ = c.Process.Signal(syscall.SIGTERM)
	}()

	// Block until luxd exits.
	waitErr := c.Wait()
	_ = os.Remove(pidFile)
	return waitErr
}

// findLuxd resolves the luxd binary path. Priority: explicit --node-path,
// $PATH, then ~/work/lux/node/build/luxd.
func findLuxd(explicit string) (string, error) {
	if explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return "", fmt.Errorf("luxd not found at %s", explicit)
		}
		return explicit, nil
	}
	if p, err := exec.LookPath("luxd"); err == nil {
		return p, nil
	}
	if home, err := os.UserHomeDir(); err == nil {
		fallback := filepath.Join(home, "work", "lux", "node", "build", "luxd")
		if _, err := os.Stat(fallback); err == nil {
			return fallback, nil
		}
	}
	return "", fmt.Errorf("luxd not in PATH; pass --node-path or build at ~/work/lux/node/build/luxd")
}

// Profile resolves name/env once, used by sibling commands that share
// a parent (cycle, snap-after-up). Not currently used externally — kept
// exported for that case.
func Profile(ctx context.Context, ref string) (*network.Profile, error) {
	_ = ctx
	return network.Resolve(ref)
}
