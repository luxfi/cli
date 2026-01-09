// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package networkcmd

import (
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/spf13/cobra"
)

var app *application.Lux

// NewCmd creates the network command for managing local network runtime.
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	app = injectedApp
	cmd := &cobra.Command{
		Use:   "network",
		Short: "Manage local network runtime",
		Long: `The network command manages local network runtime operations.

OVERVIEW:

  The network command suite controls the lifecycle of local Lux networks
  used for development and testing. It manages the node processes and runtime
  state, but does NOT manage blockchain configurations (use 'lux chain' for that).

COMMANDS:

  start     Start a local network (mainnet/testnet/devnet/dev mode)
  stop      Stop the running network and save a snapshot
  status    Show network status and endpoints
  clean     Stop network and delete runtime data (preserves chains)
  snapshot  Manage network snapshots

NETWORK TYPES:

  mainnet   Production network (5 validators, port 9630)
  testnet   Test network (5 validators, port 9640)
  devnet    Development network (5 validators, port 9650)
  dev       Single-node dev mode with K=1 consensus

TYPICAL WORKFLOW:

  # Start a development network
  lux network start --devnet

  # Check it's running
  lux network status

  # Deploy a chain (see 'lux chain --help')
  lux chain deploy mychain

  # Stop and save state
  lux network stop

  # Clean everything (preserves chain configs)
  lux network clean

NOTES:

  - Only one network type can run at a time
  - Chain configurations are managed separately via 'lux chain'
  - Runtime data is stored in ~/.lux/networks/<type>
  - Use 'lux network clean' to wipe runtime data but keep chain configs`,
		RunE: cobrautils.CommandSuiteUsage,
	}

	// Local network runtime operations only
	cmd.AddCommand(newStartCmd())
	cmd.AddCommand(newStopCmd())
	cmd.AddCommand(newCleanCmd())
	cmd.AddCommand(NewStatusCmd())  // New improved status command
	cmd.AddCommand(NewMonitorCmd()) // Real-time network monitor
	cmd.AddCommand(newSnapshotCmd())
	cmd.AddCommand(newBootstrapCmd())

	return cmd
}
