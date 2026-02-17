// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package nodecmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	deployImage    string
	deployReplicas int32
	chartPath      string
	helmDryRun     bool
	helmSet        []string
)

// defaultChartPath returns the default Helm chart path.
// Searches: 1) $LUX_CHART_PATH, 2) ~/work/lux/devops/charts/lux
func defaultChartPath() string {
	if p := os.Getenv("LUX_CHART_PATH"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "work", "lux", "devops", "charts", "lux")
}

func newDeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy luxd to Kubernetes via Helm",
		Long: `Deploys the luxd Helm chart to Kubernetes using helm upgrade --install.

Uses the canonical Helm chart from ~/work/lux/devops/charts/lux/ as the
single source of truth. This ensures the CLI creates identical deployments
to the deploy-all.sh script — same startup.sh, staking keys, bootstrap
nodes, upgrade-file-content, chain configs, and per-pod services.

CHART DISCOVERY (in order):
  1. --chart-path flag
  2. $LUX_CHART_PATH environment variable
  3. ~/work/lux/devops/charts/lux/

EXAMPLES:
  lux node deploy --mainnet
  lux node deploy --testnet --set image.tag=luxd-v1.23.15
  lux node deploy --devnet --replicas 3
  lux node deploy --mainnet --chart-path /path/to/chart
  lux node deploy --mainnet --dry-run`,
		RunE: runDeploy,
	}

	cmd.Flags().StringVar(&deployImage, "image", "", "override image tag (shorthand for --set image.tag=TAG)")
	cmd.Flags().Int32Var(&deployReplicas, "replicas", 0, "override replica count (0 = use chart default)")
	cmd.Flags().StringVar(&chartPath, "chart-path", "", "path to Helm chart (default: auto-detect)")
	cmd.Flags().BoolVar(&helmDryRun, "dry-run", false, "helm dry-run mode (template only, no apply)")
	cmd.Flags().StringArrayVar(&helmSet, "set", nil, "additional Helm --set overrides (repeatable)")

	return cmd
}

func runDeploy(_ *cobra.Command, _ []string) error {
	network, err := resolveNetwork()
	if err != nil {
		return err
	}
	namespace := "lux-" + network

	// Resolve chart path
	chart := chartPath
	if chart == "" {
		chart = defaultChartPath()
	}

	// Validate chart exists
	if _, err := os.Stat(filepath.Join(chart, "Chart.yaml")); err != nil {
		return fmt.Errorf("Helm chart not found at %s (set --chart-path or $LUX_CHART_PATH)", chart)
	}

	// Validate values file exists
	valuesFile := filepath.Join(chart, fmt.Sprintf("values-%s.yaml", network))
	if _, err := os.Stat(valuesFile); err != nil {
		return fmt.Errorf("values file not found: %s", valuesFile)
	}

	// Check helm binary
	helmBin, err := exec.LookPath("helm")
	if err != nil {
		return fmt.Errorf("helm not found in PATH — install from https://helm.sh/docs/intro/install/")
	}

	// Build helm command
	releaseName := "luxd-" + network
	args := []string{
		"upgrade", "--install", releaseName, chart,
		"--namespace", namespace,
		"--create-namespace",
		"-f", valuesFile,
	}

	// K8s context override
	ctx := flagContext
	if ctx == "" {
		ctx = os.Getenv("KUBECONTEXT")
	}
	if ctx != "" {
		args = append(args, "--kube-context", ctx)
	}

	// Image override
	if deployImage != "" {
		args = append(args, "--set", "image.tag="+deployImage)
	}

	// Replicas override
	if deployReplicas > 0 {
		args = append(args, "--set", fmt.Sprintf("replicas=%d", deployReplicas))
	}

	// Additional --set flags
	for _, s := range helmSet {
		args = append(args, "--set", s)
	}

	// Dry-run
	if helmDryRun {
		args = append(args, "--dry-run")
	}

	ux.Logger.PrintToUser("Deploying %s via Helm:", network)
	ux.Logger.PrintToUser("  Release:    %s", releaseName)
	ux.Logger.PrintToUser("  Namespace:  %s", namespace)
	ux.Logger.PrintToUser("  Chart:      %s", chart)
	ux.Logger.PrintToUser("  Values:     %s", valuesFile)
	if deployImage != "" {
		ux.Logger.PrintToUser("  Image:      %s", deployImage)
	}
	if deployReplicas > 0 {
		ux.Logger.PrintToUser("  Replicas:   %d", deployReplicas)
	}
	ux.Logger.PrintToUser("")

	// Run helm
	helmCmd := exec.Command(helmBin, args...)
	helmCmd.Stdout = os.Stdout
	helmCmd.Stderr = os.Stderr
	helmCmd.Env = os.Environ()

	if err := helmCmd.Run(); err != nil {
		return fmt.Errorf("helm upgrade --install failed: %w", err)
	}

	if helmDryRun {
		ux.Logger.PrintToUser("\n[dry-run] No changes applied.")
		return nil
	}

	// Wait for pods to be ready
	ux.Logger.PrintToUser("\nWaiting for pods to be ready...")
	if err := waitForDeployReady(namespace, 10*time.Minute); err != nil {
		ux.Logger.PrintToUser("Warning: %v", err)
		ux.Logger.PrintToUser("Check status with: lux node status --%s", network)
		return nil
	}

	ux.Logger.PrintToUser("\nDeployed successfully!")
	ux.Logger.PrintToUser("  Status:  lux node status --%s", network)
	ux.Logger.PrintToUser("  Logs:    lux node logs --%s", network)
	ux.Logger.PrintToUser("  Upgrade: lux node upgrade --%s --image TAG", network)
	return nil
}

// resolveNetwork returns the network name from flags.
func resolveNetwork() (string, error) {
	if flagNamespace != "" {
		// Extract network from namespace like "lux-mainnet"
		switch flagNamespace {
		case "lux-mainnet":
			return "mainnet", nil
		case "lux-testnet":
			return "testnet", nil
		case "lux-devnet":
			return "devnet", nil
		default:
			return "", fmt.Errorf("custom namespace %q not supported for Helm deploy — use --mainnet, --testnet, or --devnet", flagNamespace)
		}
	}
	set := 0
	if flagMainnet {
		set++
	}
	if flagTestnet {
		set++
	}
	if flagDevnet {
		set++
	}
	if set > 1 {
		return "", fmt.Errorf("specify exactly one of --mainnet, --testnet, or --devnet")
	}
	switch {
	case flagMainnet:
		return "mainnet", nil
	case flagTestnet:
		return "testnet", nil
	case flagDevnet:
		return "devnet", nil
	default:
		return "", fmt.Errorf("specify --mainnet, --testnet, or --devnet")
	}
}

// waitForDeployReady waits for the StatefulSet to reach full readiness.
func waitForDeployReady(namespace string, timeout time.Duration) error {
	client, err := newK8sClient()
	if err != nil {
		return fmt.Errorf("cannot connect to k8s: %w", err)
	}

	ctx := context.Background()
	deadline := time.After(timeout)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			return fmt.Errorf("timeout waiting for pods to be ready")
		case <-ticker.C:
			sts, err := client.AppsV1().StatefulSets(namespace).Get(ctx, statefulSetName, metav1.GetOptions{})
			if err != nil {
				continue
			}
			replicas := int32(5)
			if sts.Spec.Replicas != nil {
				replicas = *sts.Spec.Replicas
			}
			if sts.Status.ReadyReplicas == replicas {
				ux.Logger.PrintToUser("  All %d pods ready", replicas)
				return nil
			}
			ux.Logger.PrintToUser("  Ready: %d/%d", sts.Status.ReadyReplicas, replicas)
		}
	}
}
