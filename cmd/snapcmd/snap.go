// Package snapcmd implements `lux snap <network>/<env>` — tar.zst the
// data-dir of a stopped node. Resolves the snapshot path and name from
// chain.yaml runtime block; never asked for as flags.
package snapcmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/luxfi/cli/pkg/brand"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	tag       string
	allowLive bool
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snap <network>/<env>",
		Short: "Compress the data-dir into <snapshotDir>/<name>.tar.zst",
		Long: `Snapshots the resolved data-dir to <snapshotDir>/<snapshotName>.tar.zst.

The node should be stopped first (` + "`lux down`" + `) — snapshotting a
live database produces corrupt archives. Use --live to override the
safety check.

Examples:
  lux snap zoo/devnet              # → ~/work/lux/snapshots/200202-zoo-devnet-with-contracts.tar.zst
  lux snap lux/mainnet --tag pre-merge`,
		Args:         cobra.ExactArgs(1),
		RunE:         func(_ *cobra.Command, args []string) error { return snap(args[0]) },
		SilenceUsage: true,
	}
	cmd.Flags().StringVar(&tag, "tag", "", "override the trailing token in the snapshot name (default: with-contracts)")
	cmd.Flags().BoolVar(&allowLive, "live", false, "allow snapshot while node is still up (unsafe)")
	return cmd
}

func snap(ref string) error {
	prof, err := brand.Resolve(ref)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if matches, _, probeErr := prof.Probe(ctx); probeErr == nil && matches && !allowLive {
		return fmt.Errorf("%s is up on port %d — `lux down %s` first (or pass --live)",
			prof, prof.HTTPPort, prof)
	}

	name := prof.SnapshotName
	if tag != "" {
		i := strings.LastIndex(name, "-")
		if i > 0 {
			name = name[:i] + "-" + tag
		} else {
			name = name + "-" + tag
		}
	}
	outPath := filepath.Join(prof.SnapshotDir, name+".tar.zst")
	if err := os.MkdirAll(prof.SnapshotDir, 0o750); err != nil {
		return err
	}

	ux.Logger.PrintToUser("snap %s → %s", prof, outPath)
	if _, err := os.Stat(prof.DataDir); err != nil {
		return fmt.Errorf("data-dir %s missing: %w", prof.DataDir, err)
	}

	cmd := exec.Command(
		"tar", "--use-compress-program=zstd",
		"-cf", outPath,
		"-C", filepath.Dir(prof.DataDir),
		filepath.Base(prof.DataDir),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tar: %w", err)
	}

	sum, err := fileSHA256(outPath)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outPath+".sha256",
		[]byte(sum+"  "+filepath.Base(outPath)+"\n"), 0o600); err != nil {
		return err
	}
	st, _ := os.Stat(outPath)
	ux.Logger.PrintToUser("  %s (%d MB) sha256=%s",
		filepath.Base(outPath), st.Size()/(1024*1024), sum[:12]+"...")
	return nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path) //nolint:gosec
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
