// Package infocmd implements `lux info <name>/<env>` — full Profile
// dump for one resolved sovereign-L1 network instance. Useful when you
// need the EVM chain ID for a wallet, the data-dir for inspection, or
// the snapshot name for backup tooling.
package infocmd

import (
	"context"
	"fmt"
	"text/tabwriter"
	"time"

	"os"

	"github.com/luxfi/cli/pkg/network"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <name>/<env>",
		Short: "Print the full runtime profile for one network instance",
		Long: `Resolves <name>/<env> against the registry and prints every field
of the resulting profile: distinct identifiers (network ID + EVM chain
ID), ports, paths, snapshot name, service label, RPC endpoints, and
live state.

The wire identifiers are deliberately distinct:
  network ID       uint32, validator wire (--network-id, info.getNetworkID)
  EVM chain ID     uint64, EIP-155 (eth_chainId, MetaMask)

Lux brand keeps them separate (NID 1 / EVM 96369). Sovereign-L1 brand
forks (Zoo, Hanzo, Pars, Osage, Liquidity) collapse them by convention.

Examples:
  lux info zoo/mainnet
  lux info lux/devnet`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			p, err := network.Resolve(args[0])
			if err != nil {
				return err
			}
			state := "down"
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if matches, found, perr := p.Probe(ctx); perr == nil {
				if matches {
					state = "up"
				} else {
					state = fmt.Sprintf("FOREIGN(nid=%d)", found)
				}
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			row := func(k, v string) { fmt.Fprintf(w, "%s\t%s\n", k, v) }
			row("network", p.String())
			row("name", p.Name)
			row("env", p.Env)
			row("network ID (uint32)", fmt.Sprintf("%d", p.NetworkID))
			if p.PrimaryEvmChainID != 0 {
				row("EVM chain ID (uint64)", fmt.Sprintf("%d", p.PrimaryEvmChainID))
			} else {
				row("EVM chain ID (uint64)", "(not declared)")
			}
			row("http port", fmt.Sprintf("%d", p.HTTPPort))
			row("staking port", fmt.Sprintf("%d", p.StakingPort))
			row("state", state)
			row("data-dir", p.DataDir)
			row("log-dir", p.LogDir)
			if p.GenesisFile != "" {
				row("genesis file", p.GenesisFile)
			}
			row("local RPC", p.LocalRPCUrl)
			if p.RPCUrl != "" {
				row("remote RPC", p.RPCUrl)
			}
			row("snapshot dir", p.SnapshotDir)
			row("snapshot name", p.SnapshotName)
			row("service label", p.ServiceLabel)
			row("log level", p.LogLevel)
			return w.Flush()
		},
		SilenceUsage: true,
	}
}
