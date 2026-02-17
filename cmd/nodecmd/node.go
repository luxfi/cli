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

KUBERNETES COMMANDS (via Helm chart):
  deploy      Deploy/update luxd via Helm (single source of truth)
  upgrade     Rolling upgrade with zero downtime (partition-based)
  status      Show pod status, images, and health
  logs        Stream logs from a luxd pod
  rollback    Revert to previous StatefulSet revision

The deploy command uses the canonical Helm chart at ~/work/lux/devops/charts/lux/
(configurable via --chart-path or $LUX_CHART_PATH). All other k8s commands use
the Kubernetes API directly for fast read/write operations.

All k8s commands require one of --mainnet, --testnet, --devnet, or --namespace.
Use --context to target a specific kubeconfig context.

EXAMPLES:
  # Local
  lux node link --auto

  # Deploy via Helm (uses canonical chart + values-{network}.yaml)
  lux node deploy --mainnet
  lux node deploy --testnet --set image.tag=luxd-v1.23.15

  # Zero-downtime upgrade (partition-based, per-pod health checks)
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
