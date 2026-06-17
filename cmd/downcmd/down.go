// Package downcmd implements `lux down <network>/<env>` — stop a node by
// service identity, never by stored path.
//
// Stop probes the resolved httpPort with info.getNetworkID and verifies
// the responder is OUR (brand,env). Only then does it locate the
// listening PID and SIGTERM it. If the verifier fails we refuse —
// killing the wrong process is the worst outcome.
package downcmd

import (
	"context"
	"time"

	"github.com/luxfi/cli/pkg/brand"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down <network>/<env>",
		Short: "Stop a sovereign-L1 node by identity (not by PID file)",
		Long: `Stops the luxd node for <network>/<env> by:

  1. Probing http://127.0.0.1:<httpPort>/ext/info → info.getNetworkID
  2. Verifying the response equals the expected networkID from chain.yaml
  3. Looking up the PID listening on httpPort via lsof
  4. SIGTERM → wait drain → SIGKILL if still up

The (network, env) tuple alone determines what gets stopped. No --data-dir,
no --pid-file. Refusal-by-default: if the responder's networkID does not
match, we leave it alone.

Examples:
  lux down zoo/localnet
  lux down lux/devnet`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			prof, err := brand.Resolve(args[0])
			if err != nil {
				return err
			}
			ux.Logger.PrintToUser("down %s — networkID=%d port=%d", prof, prof.NetworkID, prof.HTTPPort)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := prof.Stop(ctx, 15*time.Second); err != nil {
				return err
			}
			ux.Logger.PrintToUser("  stopped")
			return nil
		},
		SilenceUsage: true,
	}
}
