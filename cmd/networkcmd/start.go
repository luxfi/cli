// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"fmt"
	"path"

	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/models"
	"github.com/luxfi/cli/pkg/subnet"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/cli/pkg/vm"
	"github.com/luxfi/netrunner/client"
	"github.com/luxfi/netrunner/server"
	"github.com/luxfi/netrunner/utils"
	"github.com/spf13/cobra"
)

var (
	userProvidedLuxVersion string
	snapshotName             string
)

const latest = "latest"

func newStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Starts a local network",
		Long: `The network start command starts a local, multi-node Lux network on your machine.

By default, the command loads the default snapshot. If you provide the --snapshot-name
flag, the network loads that snapshot instead. The command fails if the local network is
already running.`,

		RunE:         StartNetwork,
		Args:         cobra.ExactArgs(0),
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&userProvidedLuxVersion, "node-version", latest, "use this version of node (ex: v1.17.12)")
	cmd.Flags().StringVar(&snapshotName, "snapshot-name", constants.DefaultSnapshotName, "name of snapshot to use to start the network from")

	return cmd
}

func StartNetwork(*cobra.Command, []string) error {
	luxVersion, err := determineLuxVersion(userProvidedLuxVersion)
	if err != nil {
		return err
	}

	sd := subnet.NewLocalDeployer(app, luxVersion, "")

	if err := sd.StartServer(); err != nil {
		return err
	}

	nodeBinPath, err := sd.SetupLocalEnv()
	if err != nil {
		return err
	}

	cli, err := binutils.NewGRPCClient()
	if err != nil {
		return err
	}

	var startMsg string
	if snapshotName == constants.DefaultSnapshotName {
		startMsg = "Starting previously deployed and stopped snapshot"
	} else {
		startMsg = fmt.Sprintf("Starting previously deployed and stopped snapshot %s...", snapshotName)
	}
	ux.Logger.PrintToUser("%s", startMsg)

	outputDirPrefix := path.Join(app.GetRunDir(), "restart")
	outputDir, err := utils.MkDirWithTimestamp(outputDirPrefix)
	if err != nil {
		return err
	}

	pluginDir := app.GetPluginsDir()

	loadSnapshotOpts := []client.OpOption{
		client.WithExecPath(nodeBinPath),
		client.WithRootDataDir(outputDir),
		client.WithReassignPortsIfUsed(true),
		client.WithPluginDir(pluginDir),
	}

	// load global node configs if they exist
	configStr, err := app.Conf.LoadNodeConfig()
	if err != nil {
		return err
	}
	if configStr != "" {
		loadSnapshotOpts = append(loadSnapshotOpts, client.WithGlobalNodeConfig(configStr))
	}

	ctx := binutils.GetAsyncContext()

	pp, err := cli.LoadSnapshot(
		ctx,
		snapshotName,
		loadSnapshotOpts...,
	)

	if err != nil {
		if !server.IsServerError(err, server.ErrAlreadyBootstrapped) {
			return fmt.Errorf("failed to start network with the persisted snapshot: %w", err)
		}
		ux.Logger.PrintToUser("Network has already been booted. Wait until healthy...")
	} else {
		ux.Logger.PrintToUser("Booting Network. Wait until healthy...")
		ux.Logger.PrintToUser("Node log path: %s/node<i>/logs", pp.ClusterInfo.RootDataDir)
	}

	clusterInfo, err := subnet.WaitForHealthy(ctx, cli)
	if err != nil {
		return fmt.Errorf("failed waiting for network to become healthy: %w", err)
	}

	fmt.Println()
	if subnet.HasEndpoints(clusterInfo) {
		ux.Logger.PrintToUser("Network ready to use. Local network node endpoints:")
		ux.PrintTableEndpoints(clusterInfo)
	}

	return nil
}

func determineLuxVersion(userProvidedLuxVersion string) (string, error) {
	// a specific user provided version should override this calculation, so just return
	if userProvidedLuxVersion != latest {
		return userProvidedLuxVersion, nil
	}

	// Need to determine which subnets have been deployed
	locallyDeployedSubnets, err := subnet.GetLocallyDeployedSubnetsFromFile(app)
	if err != nil {
		return "", err
	}

	// if no subnets have been deployed, use latest
	if len(locallyDeployedSubnets) == 0 {
		return latest, nil
	}

	currentRPCVersion := -1

	// For each deployed subnet, check RPC versions
	for _, deployedSubnet := range locallyDeployedSubnets {
		sc, err := app.LoadSidecar(deployedSubnet)
		if err != nil {
			return "", err
		}

		// if you have a custom vm, you must provide the version explicitly
		// if you upgrade from evm to a custom vm, the RPC version will be 0
		if sc.VM == models.CustomVM || sc.Networks[models.Local.String()].RPCVersion == 0 {
			continue
		}

		if currentRPCVersion == -1 {
			currentRPCVersion = sc.Networks[models.Local.String()].RPCVersion
		}

		if sc.Networks[models.Local.String()].RPCVersion != currentRPCVersion {
			return "", fmt.Errorf(
				"RPC version mismatch. Expected %d, got %d for Subnet %s. Upgrade all subnets to the same RPC version to launch the network",
				currentRPCVersion,
				sc.RPCVersion,
				sc.Name,
			)
		}
	}

	// If currentRPCVersion == -1, then only custom subnets have been deployed, the user must provide the version explicitly if not latest
	if currentRPCVersion == -1 {
		ux.Logger.PrintToUser("No Subnet RPC version found. Using latest Lux version")
		return latest, nil
	}

	return vm.GetLatestLuxByProtocolVersion(
		app,
		currentRPCVersion,
		constants.LuxCompatibilityURL,
	)
}
