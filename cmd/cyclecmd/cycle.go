// Package cyclecmd implements `lux cycle <network>/<env>` — the atomic
// up→deploy→down→snap pipeline. Spawns luxd in the background within
// the same process, waits for health, runs contract deploy against the
// resolved RPC, signals shutdown, waits for drain, snapshots.
package cyclecmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/luxfi/cli/pkg/network"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	skipDeploy bool
	skipSnap   bool
	nodePath   string
	mnemonic   string
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cycle <network>/<env>",
		Short: "Boot → deploy → stop → snapshot in one pipeline",
		Long: `Atomically boots the network's L1, deploys the standard contract
suite, stops the node cleanly, and writes a tar.zst snapshot.

Pipeline:
  1. lux up    <ref>           (background within this process)
  2. wait healthy (networkID match + C-chain responding)
  3. lux deploy <ref>          (forge against the local RPC)
  4. lux down  <ref>           (SIGTERM + wait drain)
  5. lux snap  <ref>           (tar.zst the data-dir)

Examples:
  lux cycle zoo/localnet
  lux cycle lux/devnet --skip-deploy   # boot, stop, snap only
  lux cycle hanzo/testnet --skip-snap`,
		Args:         cobra.ExactArgs(1),
		RunE:         func(_ *cobra.Command, args []string) error { return cycle(args[0]) },
		SilenceUsage: true,
	}
	cmd.Flags().BoolVar(&skipDeploy, "skip-deploy", false, "do not run contract deploy stage")
	cmd.Flags().BoolVar(&skipSnap, "skip-snap", false, "do not run snapshot stage")
	cmd.Flags().StringVar(&nodePath, "node-path", "", "path to luxd binary")
	cmd.Flags().StringVar(&mnemonic, "mnemonic", "", "BIP44 mnemonic for the deployer (defaults to $LUX_MNEMONIC)")
	return cmd
}

func cycle(ref string) error {
	prof, err := network.Resolve(ref)
	if err != nil {
		return err
	}
	ux.Logger.PrintToUser("cycle %s — networkID=%d port=%d", prof, prof.NetworkID, prof.HTTPPort)

	if err := os.MkdirAll(prof.LogDir, 0o750); err != nil {
		return err
	}
	var genesisHash string
	if prof.GenesisFile != "" {
		if data, err := os.ReadFile(prof.GenesisFile); err == nil { //nolint:gosec
			genesisHash = network.HashGenesis(data)
		}
	}
	if err := prof.VerifyOrCreate(genesisHash); err != nil {
		return err
	}

	luxd, err := findLuxd(nodePath)
	if err != nil {
		return err
	}

	// (1) boot
	args := []string{
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
		args = append(args, "--genesis-file="+prof.GenesisFile)
	}

	cmd := exec.Command(luxd, args...) //nolint:gosec
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start luxd: %w", err)
	}
	ux.Logger.PrintToUser("  luxd PID=%d", cmd.Process.Pid)

	// (2) wait healthy
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	if err := prof.WaitHealthy(ctx, 60*time.Second); err != nil {
		cancel()
		_ = cmd.Process.Signal(syscall.SIGTERM)
		_, _ = cmd.Process.Wait()
		return fmt.Errorf("health check: %w", err)
	}
	cancel()
	ux.Logger.PrintToUser("  healthy")

	// (3) deploy
	if !skipDeploy {
		if err := runDeploy(prof); err != nil {
			_ = cmd.Process.Signal(syscall.SIGTERM)
			_, _ = cmd.Process.Wait()
			return fmt.Errorf("deploy: %w", err)
		}
	}

	// (4) shutdown
	ux.Logger.PrintToUser("  stopping luxd")
	_ = cmd.Process.Signal(syscall.SIGTERM)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		ux.Logger.PrintToUser("  graceful shutdown timed out — SIGKILL")
		_ = cmd.Process.Signal(syscall.SIGKILL)
		<-done
	}

	// (5) snapshot
	if !skipSnap {
		if err := runTarSnap(prof); err != nil {
			return fmt.Errorf("snap: %w", err)
		}
	}

	ux.Logger.PrintToUser("✓ cycle %s done", prof)
	return nil
}

func runDeploy(p *network.Profile) error {
	// Compose: invoke the standard deploy script in lux/standard via
	// forge, using the local RPC. Reuses lux/standard's Deploy.s.sol —
	// no script duplication here.
	standardRoot := envOr("LUX_STANDARD_ROOT", filepath.Join(os.Getenv("HOME"), "work", "lux", "standard"))
	if _, err := os.Stat(standardRoot); err != nil {
		return fmt.Errorf("lux/standard not found at %s (set $LUX_STANDARD_ROOT)", standardRoot)
	}
	pk, err := derivePrivateKey()
	if err != nil {
		return err
	}
	forge, err := exec.LookPath("forge")
	if err != nil {
		return fmt.Errorf("forge not in PATH")
	}
	cmd := exec.Command(forge,
		"script", "script/Deploy.s.sol", "--tc", "Deploy",
		"--rpc-url", p.LocalRPCUrl,
		"--private-key", pk,
		"--broadcast", "--legacy",
		"--gas-price", "30000000000",
		"--disable-code-size-limit", "--slow", "--skip-simulation",
	)
	cmd.Dir = standardRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "FOUNDRY_DISABLE_NIGHTLY_WARNING=1")
	return cmd.Run()
}

func runTarSnap(p *network.Profile) error {
	if err := os.MkdirAll(p.SnapshotDir, 0o750); err != nil {
		return err
	}
	out := filepath.Join(p.SnapshotDir, p.SnapshotName+".tar.zst")
	cmd := exec.Command(
		"tar", "--use-compress-program=zstd",
		"-cf", out,
		"-C", filepath.Dir(p.DataDir),
		filepath.Base(p.DataDir),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	ux.Logger.PrintToUser("  snapshot: %s", out)
	return nil
}

func findLuxd(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if p, err := exec.LookPath("luxd"); err == nil {
		return p, nil
	}
	home, _ := os.UserHomeDir()
	fallback := filepath.Join(home, "work", "lux", "node", "build", "luxd")
	if _, err := os.Stat(fallback); err == nil {
		return fallback, nil
	}
	return "", fmt.Errorf("luxd not found")
}

func derivePrivateKey() (string, error) {
	if pk := os.Getenv("PRIVATE_KEY"); pk != "" {
		return pk, nil
	}
	m := mnemonic
	if m == "" {
		m = os.Getenv("LUX_MNEMONIC")
	}
	if m == "" {
		// Fall back to Keychain lookup on darwin.
		out, err := exec.Command("security", "find-generic-password", "-a", "LUX_MNEMONIC", "-w").Output()
		if err == nil {
			m = string(out)
		}
	}
	if m == "" {
		return "", fmt.Errorf("no LUX_MNEMONIC, PRIVATE_KEY, or Keychain entry")
	}
	cast, err := exec.LookPath("cast")
	if err != nil {
		return "", fmt.Errorf("cast not in PATH")
	}
	out, err := exec.Command(cast, "wallet", "derive-private-key", strings.TrimSpace(m), "0").Output()
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	return strings.TrimSpace(lines[len(lines)-1]), nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
