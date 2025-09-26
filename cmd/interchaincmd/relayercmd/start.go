// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package relayercmd

import (
	"fmt"
	"path/filepath"

	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/interchain/relayer"
	"github.com/luxfi/cli/pkg/localnet"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/cli/pkg/networkoptions"
	"github.com/luxfi/cli/pkg/node"
	"github.com/luxfi/cli/pkg/ssh"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"

	"github.com/spf13/cobra"
)

var startNetworkOptions = []networkoptions.NetworkOption{
	networkoptions.Local,
	networkoptions.Cluster,
	networkoptions.Testnet,
}

type StartFlags struct {
	Network networkoptions.NetworkFlags
	BinPath string
	Version string
}

var startFlags StartFlags

// lux interchain relayer start
func newStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "starts AWM relayer",
		Long:  `Starts AWM relayer on the specified network (Currently only for local network).`,
		RunE:  start,
		Args:  cobrautils.ExactArgs(0),
	}
	networkoptions.AddNetworkFlagsToCmd(cmd, &startFlags.Network, true, startNetworkOptions)
	cmd.Flags().StringVar(&startFlags.BinPath, "bin-path", "", "use the given relayer binary")
	cmd.Flags().StringVar(
		&startFlags.Version,
		"version",
		constants.DefaultRelayerVersion,
		"version to use",
	)
	return cmd
}

func start(_ *cobra.Command, args []string) error {
	return CallStart(args, startFlags, models.UndefinedNetwork)
}

func CallStart(_ []string, flags StartFlags, network models.Network) error {
	var err error
	if network == models.UndefinedNetwork {
		network, err = networkoptions.GetNetworkFromCmdLineFlags(
			app,
			"",
			startFlags.Network,
			false,
			false,
			startNetworkOptions,
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
		if err := ssh.RunSSHStartWarpRelayerService(host); err != nil {
			return err
		}
		ux.Logger.GreenCheckmarkToUser("Remote AWM Relayer on %s successfully started", host.GetCloudID())
	default:
		if relayerIsUp, _, _, err := relayer.RelayerIsUp(
			app.GetLocalRelayerRunPath(network.Kind()),
		); err != nil {
			return err
		} else if relayerIsUp {
			return fmt.Errorf("local AWM relayer is already running for %s", network.Kind())
		}
		// localNetworkRootDir := ""
		if network.Kind() == models.Local {
			_, err = localnet.GetLocalNetworkDir(app)
			if err != nil {
				return err
			}
		}
		relayerConfigPath := app.GetLocalRelayerConfigPath()
		if network.Kind() == models.Local && flags.BinPath == "" && flags.Version == constants.DefaultRelayerVersion {
			if b, extraLocalNetworkData, err := localnet.GetExtraLocalNetworkData(app, ""); err != nil {
				return err
			} else if b {
				flags.BinPath = extraLocalNetworkData.RelayerPath
			}
		}
		if !utils.FileExists(relayerConfigPath) {
			return fmt.Errorf("there is no relayer configuration available")
		} else if binPath, err := relayer.DeployRelayer(
			flags.Version,
			flags.BinPath,
			filepath.Join(app.GetBaseDir(), "bin", "warp-relayer"),
			relayerConfigPath,
			"", // config string parameter
			app.GetLocalRelayerLogPath(network.Kind()),
			app.GetLocalRelayerRunPath(network.Kind()),
			app.GetLocalRelayerStorageDir(network.Kind()),
		); err != nil {
			return err
		} else if network.Kind() == models.Local {
			if err := localnet.WriteExtraLocalNetworkData(app, "", binPath, "", ""); err != nil {
				return err
			}
		}
		ux.Logger.GreenCheckmarkToUser("Local AWM Relayer successfully started for %s", network.Kind())
		ux.Logger.PrintToUser("Logs can be found at %s", app.GetLocalRelayerLogPath(network.Kind()))
	}
	return nil
}
