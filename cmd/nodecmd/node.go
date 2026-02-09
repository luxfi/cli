// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package nodecmd

import (
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/spf13/cobra"
)

var app *application.Lux

// NewCmd creates the node command suite.
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	app = injectedApp
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Manage luxd nodes (local binary and Kubernetes deployments)",
		Long: `Commands for managing luxd nodes — locally and on Kubernetes.

LOCAL COMMANDS:
  link        Symlink a luxd binary to ~/.lux/bin/luxd

KUBERNETES COMMANDS:
  deploy      Deploy luxd StatefulSet to a k8s cluster
  upgrade     Rolling upgrade with zero downtime
  status      Show pod status, images, and health
  logs        Stream logs from a luxd pod
  rollback    Revert to previous StatefulSet revision

All k8s commands require one of --mainnet, --testnet, --devnet, or --namespace.
Use --context to target a specific kubeconfig context.

EXAMPLES:
  # Local
  lux node link --auto

  # Deploy to k8s
  lux node deploy --mainnet --replicas 5 --image ghcr.io/luxfi/node:v1.23.5

  # Zero-downtime upgrade
  lux node upgrade --mainnet --image ghcr.io/luxfi/node:v1.23.6

  # Check status
  lux node status --mainnet

  # Stream logs
  lux node logs --mainnet luxd-0 -f

  # Rollback
  lux node rollback --mainnet`,
		RunE: cobrautils.CommandSuiteUsage,
	}

	// Local commands
	cmd.AddCommand(newLinkCmd())

	// K8s commands
	deployCmdObj := newDeployCmd()
	upgradeCmdObj := newUpgradeCmd()
	statusCmdObj := newStatusCmd()
	logsCmdObj := newLogsCmd()
	rollbackCmdObj := newRollbackCmd()

	// Add shared k8s flags to all k8s subcommands
	for _, sub := range []*cobra.Command{deployCmdObj, upgradeCmdObj, statusCmdObj, logsCmdObj, rollbackCmdObj} {
		sub.Flags().StringVar(&flagContext, "context", "", "kubeconfig context to use")
		sub.Flags().StringVar(&flagNamespace, "namespace", "", "k8s namespace (overrides network flags)")
		sub.Flags().BoolVar(&flagMainnet, "mainnet", false, "target lux-mainnet namespace")
		sub.Flags().BoolVar(&flagTestnet, "testnet", false, "target lux-testnet namespace")
		sub.Flags().BoolVar(&flagDevnet, "devnet", false, "target lux-devnet namespace")
	}

	cmd.AddCommand(deployCmdObj)
	cmd.AddCommand(upgradeCmdObj)
	cmd.AddCommand(statusCmdObj)
	cmd.AddCommand(logsCmdObj)
	cmd.AddCommand(rollbackCmdObj)

	return cmd
}
