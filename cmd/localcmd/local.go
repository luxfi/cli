// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package localcmd

import (
	"fmt"

	"github.com/luxfi/cli/v2/pkg/application"
	"github.com/luxfi/cli/v2/pkg/binutils"
	"github.com/luxfi/cli/v2/pkg/subnet"
	"github.com/luxfi/cli/v2/pkg/ux"
	"github.com/luxfi/netrunner/client"
	"github.com/luxfi/netrunner/server"
	"github.com/spf13/cobra"
)

var app *application.Lux

func NewCmd(injectedApp *application.Lux) *cobra.Command {
	app = injectedApp
	cmd := &cobra.Command{
		Use:   "local",
		Short: "Commands for running a local development network",
		Long:  `The local command suite provides a collection of tools for managing a local, single-node, Proof-of-Authority network.`,
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				fmt.Println(err)
			}
		},
		Args: cobra.ExactArgs(0),
	}
	// local start
	cmd.AddCommand(newStartCmd())
	return cmd
}

func newStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Starts a local PoA network",
		Long:  `The local start command starts a local, single-node, Proof-of-Authority network on your machine.`,
		RunE:  startLocalNetwork,
		Args:  cobra.ExactArgs(0),
	}
	return cmd
}

func startLocalNetwork(*cobra.Command, []string) error {
	sd := subnet.NewLocalDeployer(app, "latest", "")

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

	ux.Logger.PrintToUser("Starting local PoA network...")

	startOpts := []client.OpOption{
		client.WithExecPath(nodeBinPath),
		client.WithRootDataDir(app.GetRunDir()),
		client.WithReassignPortsIfUsed(true),
		client.WithPluginDir(app.GetPluginsDir()),
	}

	ctx := binutils.GetAsyncContext()

	pp, err := cli.Start(
		ctx,
		"",
		startOpts...,
	)

	if err != nil {
		if !server.IsServerError(err, server.ErrAlreadyBootstrapped) {
			return fmt.Errorf("failed to start network: %w", err)
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
