// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package relayercmd

import (
	"fmt"

	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/interchain/relayer"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/cli/pkg/networkoptions"
	"github.com/luxfi/cli/pkg/node"
	"github.com/luxfi/cli/pkg/ssh"
	"github.com/luxfi/cli/pkg/ux"

	"github.com/spf13/cobra"
)

var stopNetworkOptions = []networkoptions.NetworkOption{
	networkoptions.Local,
	networkoptions.Cluster,
	networkoptions.Testnet,
}

type StopFlags struct {
	Network networkoptions.NetworkFlags
}

var stopFlags StopFlags

// lux interchain relayer stop
func newStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "stops AWM relayer",
		Long:  `Stops AWM relayer on the specified network (Currently only for local network, cluster).`,
		RunE:  stop,
		Args:  cobrautils.ExactArgs(0),
	}
	// Network flags handled globally to avoid conflicts
	return cmd
}

func stop(_ *cobra.Command, args []string) error {
	return CallStop(args, stopFlags, models.UndefinedNetwork)
}

func CallStop(_ []string, flags StopFlags, network models.Network) error {
	var err error
	if network == models.UndefinedNetwork {
		network, err = networkoptions.GetNetworkFromCmdLineFlags(
			app,
			"",
			flags.Network,
			false,
			false,
			stopNetworkOptions,
			"",
		)
		if err != nil {
			return err
		}
	}
	switch {
	case network.ClusterName() != "":
		host, err := node.GetWarpRelayerHost(app, network.ClusterName())
		if err != nil {
			return err
		}
		if err := ssh.RunSSHStopWarpRelayerService(host); err != nil {
			return err
		}
		ux.Logger.GreenCheckmarkToUser("Remote AWM Relayer on %s successfully stopped", host.GetCloudID())
	default:
		b, _, _, err := relayer.RelayerIsUp(
			app.GetLocalRelayerRunPath(network.Kind()),
		)
		if err != nil {
			return err
		}
		if !b {
			return fmt.Errorf("there is no CLI-managed local AWM relayer running for %s", network.Kind())
		}
		if err := relayer.RelayerCleanup(
			app.GetLocalRelayerRunPath(network.Kind()),
			app.GetLocalRelayerLogPath(network.Kind()),
			app.GetLocalRelayerStorageDir(network.Kind()),
		); err != nil {
			return err
		}
		ux.Logger.GreenCheckmarkToUser("Local AWM Relayer successfully stopped for %s", network.Kind())
	}
	return nil
}
