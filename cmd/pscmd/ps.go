// Package pscmd implements `lux ps` — for every brand/env in the
// registry, probe the local port and report whether a matching node is
// up.
package pscmd

import (
	"context"
	"fmt"
	"text/tabwriter"
	"time"

	"os"

	"github.com/luxfi/cli/pkg/brand"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ps",
		Short: "List running sovereign-L1 nodes on this host",
		Long: `Probes the resolved httpPort for every (network, env) tuple in the
registry and reports up/down + networkID match.`,
		Args: cobra.ExactArgs(0),
		RunE: func(*cobra.Command, []string) error {
			reg, err := brand.Discover()
			if err != nil {
				return err
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NETWORK\tNETID\tPORT\tSTATE\tPID")
			for _, ref := range reg.Refs() {
				prof, err := brand.Resolve(ref)
				if err != nil {
					continue
				}
				state, pid := probe(prof)
				fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\n",
					ref, prof.NetworkID, prof.HTTPPort, state, pid)
			}
			return w.Flush()
		},
		SilenceUsage: true,
	}
}

func probe(p *brand.RuntimeProfile) (state string, pid int) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	matches, found, err := p.Probe(ctx)
	if err != nil {
		return "down", 0
	}
	if !matches {
		return fmt.Sprintf("FOREIGN(nid=%d)", found), 0
	}
	pid, _ = brand.PIDOnPort(p.HTTPPort)
	return "up", pid
}
