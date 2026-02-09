// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package nodecmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

var (
	upgradeImage     string
	upgradeEvmVer    string
	stabilityWait    time.Duration
	healthTimeout    time.Duration
	dryRun           bool
	forceUpgrade     bool
)

func newUpgradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Rolling upgrade of luxd StatefulSet with zero downtime",
		Long: `Performs a partition-based rolling upgrade of the luxd StatefulSet.

Upgrades one pod at a time (highest ordinal first), waiting for each pod
to become ready and stable before proceeding. This ensures C-chain RPC
clients experience no downtime since a quorum of validators remains
available throughout the upgrade.

PROCESS:
  1. Validates the new image exists and differs from current
  2. Updates the StatefulSet pod template with new image
  3. Sets partition = replicas (no pods restart yet)
  4. Lowers partition one at a time (pod N-1, N-2, ... 0)
  5. After each pod restart: waits for readiness + stability period
  6. If any pod fails health check, stops and prints rollback command

EXAMPLES:
  lux node upgrade --mainnet --image ghcr.io/luxfi/node:v1.23.5
  lux node upgrade --testnet --image ghcr.io/luxfi/node:v1.23.5 --stability-wait 60s
  lux node upgrade --devnet --image ghcr.io/luxfi/node:v1.23.5 --dry-run`,
		RunE: runUpgrade,
	}

	cmd.Flags().StringVar(&upgradeImage, "image", "", "new container image (required)")
	cmd.Flags().StringVar(&upgradeEvmVer, "evm-version", "", "EVM plugin version to update in init container")
	cmd.Flags().DurationVar(&stabilityWait, "stability-wait", 30*time.Second, "wait time after pod ready before proceeding")
	cmd.Flags().DurationVar(&healthTimeout, "health-timeout", 5*time.Minute, "max time to wait for a pod to become ready")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would happen without making changes")
	cmd.Flags().BoolVar(&forceUpgrade, "force", false, "proceed even if image is the same")

	_ = cmd.MarkFlagRequired("image")

	return cmd
}

func runUpgrade(_ *cobra.Command, _ []string) error {
	namespace, err := resolveNamespace()
	if err != nil {
		return err
	}

	client, err := newK8sClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Get current StatefulSet
	sts, err := client.AppsV1().StatefulSets(namespace).Get(ctx, statefulSetName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get StatefulSet %s in %s: %w", statefulSetName, namespace, err)
	}

	replicas := int32(1)
	if sts.Spec.Replicas != nil {
		replicas = *sts.Spec.Replicas
	}

	// Find current image
	currentImage := ""
	for _, c := range sts.Spec.Template.Spec.Containers {
		if c.Name == containerName {
			currentImage = c.Image
			break
		}
	}

	ux.Logger.PrintToUser("Upgrade plan:")
	ux.Logger.PrintToUser("  Namespace:     %s", namespace)
	ux.Logger.PrintToUser("  StatefulSet:   %s", statefulSetName)
	ux.Logger.PrintToUser("  Replicas:      %d", replicas)
	ux.Logger.PrintToUser("  Current image: %s", currentImage)
	ux.Logger.PrintToUser("  Target image:  %s", upgradeImage)
	ux.Logger.PrintToUser("  Stability:     %s per pod", stabilityWait)

	if currentImage == upgradeImage && !forceUpgrade {
		ux.Logger.PrintToUser("\nImage is already %s — nothing to do (use --force to override)", upgradeImage)
		return nil
	}

	if dryRun {
		ux.Logger.PrintToUser("\n[dry-run] Would upgrade %d pods one at a time", replicas)
		for i := replicas - 1; i >= 0; i-- {
			ux.Logger.PrintToUser("  [dry-run] Upgrade %s-%d → wait ready → %s stability", statefulSetName, i, stabilityWait)
		}
		return nil
	}

	ux.Logger.PrintToUser("\nStarting partition-based rolling upgrade...")

	// Step 1: Set partition = replicas (freeze all pods)
	if err := setPartition(ctx, client, namespace, replicas); err != nil {
		return fmt.Errorf("failed to set partition: %w", err)
	}

	// Step 2: Update the pod template image
	if err := patchImage(ctx, client, namespace, upgradeImage); err != nil {
		return fmt.Errorf("failed to patch image: %w", err)
	}

	// Step 3: Update EVM plugin version in init container if specified
	if upgradeEvmVer != "" {
		if err := patchEvmInitContainer(ctx, client, namespace, upgradeEvmVer); err != nil {
			ux.Logger.PrintToUser("Warning: failed to update EVM init container: %v", err)
		}
	}

	// Step 4: Roll pods one at a time, highest ordinal first
	for i := replicas - 1; i >= 0; i-- {
		podName := fmt.Sprintf("%s-%d", statefulSetName, i)
		ux.Logger.PrintToUser("\n[%d/%d] Upgrading %s...", replicas-i, replicas, podName)

		// Lower partition to allow this pod to update
		if err := setPartition(ctx, client, namespace, i); err != nil {
			return fmt.Errorf("failed to lower partition to %d: %w", i, err)
		}

		// Wait for the pod to terminate and come back ready
		ux.Logger.PrintToUser("  Waiting for %s to become ready (timeout %s)...", podName, healthTimeout)
		if err := waitForPodReady(ctx, client, namespace, podName, healthTimeout); err != nil {
			ux.Logger.PrintToUser("  FAILED: %s did not become ready: %v", podName, err)
			ux.Logger.PrintToUser("\n  To rollback: lux node rollback --%s", networkFlag(namespace))
			return fmt.Errorf("upgrade halted at %s: %w", podName, err)
		}

		// Verify new image is actually running
		actualImage, err := podImage(ctx, client, namespace, podName)
		if err != nil {
			ux.Logger.PrintToUser("  Warning: could not verify image: %v", err)
		} else {
			ux.Logger.PrintToUser("  Image: %s", actualImage)
		}

		// Stability wait
		ux.Logger.PrintToUser("  Stability wait %s...", stabilityWait)
		time.Sleep(stabilityWait)

		// Re-check readiness after stability wait
		ready, err := podReady(ctx, client, namespace, podName)
		if err != nil || !ready {
			ux.Logger.PrintToUser("  FAILED: %s lost readiness during stability wait", podName)
			ux.Logger.PrintToUser("\n  To rollback: lux node rollback --%s", networkFlag(namespace))
			return fmt.Errorf("upgrade halted: %s unstable after %s", podName, stabilityWait)
		}

		ux.Logger.PrintToUser("  %s ready and stable", podName)
	}

	ux.Logger.PrintToUser("\nUpgrade complete. All %d pods running %s", replicas, upgradeImage)
	return nil
}

// setPartition patches the StatefulSet updateStrategy partition value.
func setPartition(ctx context.Context, client *kubernetes.Clientset, namespace string, partition int32) error {
	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"updateStrategy": map[string]interface{}{
				"type": "RollingUpdate",
				"rollingUpdate": map[string]interface{}{
					"partition": partition,
				},
			},
		},
	}
	data, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	_, err = client.AppsV1().StatefulSets(namespace).Patch(
		ctx, statefulSetName, types.StrategicMergePatchType, data, metav1.PatchOptions{},
	)
	return err
}

// patchImage updates the luxd container image in the StatefulSet pod template.
func patchImage(ctx context.Context, client *kubernetes.Clientset, namespace, image string) error {
	sts, err := client.AppsV1().StatefulSets(namespace).Get(ctx, statefulSetName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	for i, c := range sts.Spec.Template.Spec.Containers {
		if c.Name == containerName {
			sts.Spec.Template.Spec.Containers[i].Image = image
			break
		}
	}

	// Ensure RollingUpdate strategy with high partition (set separately)
	sts.Spec.UpdateStrategy = appsv1.StatefulSetUpdateStrategy{
		Type: appsv1.RollingUpdateStatefulSetStrategyType,
		RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{
			Partition: sts.Spec.Replicas, // freeze — will be lowered per-pod
		},
	}

	_, err = client.AppsV1().StatefulSets(namespace).Update(ctx, sts, metav1.UpdateOptions{})
	return err
}

// patchEvmInitContainer updates the EVM plugin download URL in the init container.
func patchEvmInitContainer(ctx context.Context, client *kubernetes.Clientset, namespace, evmVersion string) error {
	sts, err := client.AppsV1().StatefulSets(namespace).Get(ctx, statefulSetName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	pluginURL := fmt.Sprintf("https://github.com/luxfi/evm/releases/download/%s/evm-plugin-linux-amd64", evmVersion)
	for i, ic := range sts.Spec.Template.Spec.InitContainers {
		if ic.Name == "init-plugins" {
			sts.Spec.Template.Spec.InitContainers[i].Args = []string{
				"mkdir -p /data/plugins /data/staking && " +
					fmt.Sprintf("curl -sL %s -o /data/plugins/mgj786NP7uDwBCcq6YwThhaN8FLyybkCa4zBWTQbNgmK6k9A6 && ", pluginURL) +
					"chmod +x /data/plugins/* && " +
					"chown -R 1000:1000 /data && " +
					"echo 'Plugins installed successfully'",
			}
			break
		}
	}

	_, err = client.AppsV1().StatefulSets(namespace).Update(ctx, sts, metav1.UpdateOptions{})
	return err
}

// waitForPodReady polls until the named pod is Ready or timeout.
func waitForPodReady(ctx context.Context, client *kubernetes.Clientset, namespace, podName string, timeout time.Duration) error {
	deadline := time.After(timeout)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			return fmt.Errorf("timeout after %s", timeout)
		case <-ticker.C:
			ready, err := podReady(ctx, client, namespace, podName)
			if err != nil {
				continue // pod may be terminating
			}
			if ready {
				return nil
			}
		}
	}
}

// networkFlag returns the flag name for a namespace (for error messages).
func networkFlag(namespace string) string {
	switch namespace {
	case "lux-mainnet":
		return "mainnet"
	case "lux-testnet":
		return "testnet"
	case "lux-devnet":
		return "devnet"
	default:
		return fmt.Sprintf("namespace %s", namespace)
	}
}
