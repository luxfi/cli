// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package backendcmd

import (
	"context"
	"fmt"
	"os"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/spf13/cobra"
)

var app *application.Lux

// NewCmd creates the lux-server command (formerly cli-backend).
// This is the gRPC server that manages local network nodes.
// The network type is determined by LUX_NETWORK_TYPE environment variable.
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	app = injectedApp

	// Create base command
	baseCmd := &cobra.Command{
		Use:    constants.LuxServerCmd,
		Short:  "Run the Lux gRPC server",
		Long: `The Lux gRPC server manages local network nodes.

This command is normally invoked automatically by 'lux network start'.
Each network type (mainnet, testnet, local) runs its own server on a dedicated port:
  - mainnet: 8097
  - testnet: 8098
  - local:   8099

The network type is determined by the LUX_NETWORK_TYPE environment variable.`,
		RunE:   startBackend,
		Args:   cobra.ExactArgs(0),
		Hidden: true,
	}

	return baseCmd
}

// NewNetworkCmd creates network-specific gRPC server commands.
// These commands allow easy identification of running network servers.
func NewNetworkCmd(injectedApp *application.Lux, networkType string) *cobra.Command {
	app = injectedApp
	cmdName := constants.GetServerCmdForNetwork(networkType)

	return &cobra.Command{
		Use:    cmdName,
		Short:  fmt.Sprintf("Run the Lux gRPC server for %s", networkType),
		Long:   fmt.Sprintf("The Lux gRPC server for %s network. Invoked automatically by 'lux network start'.", networkType),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Override the environment variable with the command's network type
			os.Setenv("LUX_NETWORK_TYPE", networkType)
			return startBackend(cmd, args)
		},
		Args:   cobra.ExactArgs(0),
		Hidden: true,
	}
}

// NewAllNetworkCmds creates all network-specific gRPC server commands.
// Call this from root.go to register all network commands.
func NewAllNetworkCmds(injectedApp *application.Lux) []*cobra.Command {
	networks := []string{"mainnet", "testnet", "devnet", "local"}
	cmds := make([]*cobra.Command, len(networks))
	for i, network := range networks {
		cmds[i] = NewNetworkCmd(injectedApp, network)
	}
	return cmds
}

func startBackend(_ *cobra.Command, _ []string) error {
	// Get network type from environment variable (set by StartServerProcessForNetwork)
	// Defaults to "mainnet" for backward compatibility
	networkType := os.Getenv("LUX_NETWORK_TYPE")
	if networkType == "" {
		networkType = "mainnet"
	}

	s, err := binutils.NewGRPCServerForNetwork(app.GetSnapshotsDir(), networkType)
	if err != nil {
		return err
	}

	serverCtx, serverCancel := context.WithCancel(context.Background())
	errc := make(chan error)
	ports := binutils.GetGRPCPorts(networkType)
	fmt.Printf("starting server for %s network on port %d\n", networkType, ports.Server)
	go binutils.WatchServerProcess(serverCancel, errc, app.Log)
	errc <- s.Run(serverCtx)

	return nil
}
