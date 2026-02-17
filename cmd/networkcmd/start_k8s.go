// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package networkcmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/luxfi/cli/pkg/ux"
)

// K8sNetworkConfig holds configuration for K8s network deployment via Helm.
type K8sNetworkConfig struct {
	NetworkName string
	Namespace   string
	Image       string
}

// StartK8sNetwork deploys a Lux network to Kubernetes using the canonical Helm chart.
// This delegates to `helm upgrade --install` to ensure a single source of truth.
func StartK8sNetwork(cfg K8sNetworkConfig) error {
	// Resolve chart path
	chartPath := os.Getenv("LUX_CHART_PATH")
	if chartPath == "" {
		home, _ := os.UserHomeDir()
		chartPath = filepath.Join(home, "work", "lux", "devops", "charts", "lux")
	}

	// Validate chart exists
	if _, err := os.Stat(filepath.Join(chartPath, "Chart.yaml")); err != nil {
		return fmt.Errorf("Helm chart not found at %s (set $LUX_CHART_PATH)", chartPath)
	}

	// Validate values file
	valuesFile := filepath.Join(chartPath, fmt.Sprintf("values-%s.yaml", cfg.NetworkName))
	if _, err := os.Stat(valuesFile); err != nil {
		return fmt.Errorf("values file not found: %s", valuesFile)
	}

	// Check helm binary
	helmBin, err := exec.LookPath("helm")
	if err != nil {
		return fmt.Errorf("helm not found in PATH — install from https://helm.sh/docs/intro/install/")
	}

	releaseName := "luxd-" + cfg.NetworkName
	args := []string{
		"upgrade", "--install", releaseName, chartPath,
		"--namespace", cfg.Namespace,
		"--create-namespace",
		"-f", valuesFile,
	}

	// K8s context override
	if k8sCluster != "" {
		args = append(args, "--kube-context", k8sCluster)
	} else if ctx := os.Getenv("KUBECONTEXT"); ctx != "" {
		args = append(args, "--kube-context", ctx)
	}

	// Image override
	if cfg.Image != "" && cfg.Image != "ghcr.io/luxfi/node:latest" {
		args = append(args, "--set", "image.tag="+cfg.Image)
	}

	ux.Logger.PrintToUser("Deploying %s via Helm:", cfg.NetworkName)
	ux.Logger.PrintToUser("  Release:   %s", releaseName)
	ux.Logger.PrintToUser("  Namespace: %s", cfg.Namespace)
	ux.Logger.PrintToUser("  Chart:     %s", chartPath)
	ux.Logger.PrintToUser("  Values:    %s", valuesFile)
	ux.Logger.PrintToUser("")

	helmCmd := exec.Command(helmBin, args...)
	helmCmd.Stdout = os.Stdout
	helmCmd.Stderr = os.Stderr
	helmCmd.Env = os.Environ()

	if err := helmCmd.Run(); err != nil {
		return fmt.Errorf("helm upgrade --install failed: %w", err)
	}

	ux.Logger.PrintToUser("\n%s deployed. Check status with: lux node status --%s", cfg.NetworkName, cfg.NetworkName)
	return nil
}

// StartK8sMainnet deploys mainnet to Kubernetes via Helm.
func StartK8sMainnet() error {
	return StartK8sNetwork(K8sNetworkConfig{
		NetworkName: "mainnet",
		Namespace:   "lux-mainnet",
		Image:       k8sImage,
	})
}

// StartK8sTestnet deploys testnet to Kubernetes via Helm.
func StartK8sTestnet() error {
	return StartK8sNetwork(K8sNetworkConfig{
		NetworkName: "testnet",
		Namespace:   "lux-testnet",
		Image:       k8sImage,
	})
}

// StartK8sDevnet deploys devnet to Kubernetes via Helm.
func StartK8sDevnet() error {
	return StartK8sNetwork(K8sNetworkConfig{
		NetworkName: "devnet",
		Namespace:   "lux-devnet",
		Image:       k8sImage,
	})
}
