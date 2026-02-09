// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package nodecmd

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	rollbackRevision int64
	rollbackTimeout  time.Duration
)

func newRollbackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "Rollback luxd StatefulSet to previous revision",
		Long: `Reverts the luxd StatefulSet to its previous ControllerRevision.

By default rolls back to the immediately previous revision. Use --revision
to target a specific revision number.

After rollback, performs the same partition-based rolling update to ensure
zero downtime — pods are updated one at a time with health checks.

EXAMPLES:
  lux node rollback --mainnet
  lux node rollback --mainnet --revision 3
  lux node rollback --testnet --timeout 10m`,
		RunE: runRollback,
	}

	cmd.Flags().Int64Var(&rollbackRevision, "revision", 0, "target revision number (0 = previous)")
	cmd.Flags().DurationVar(&rollbackTimeout, "timeout", 5*time.Minute, "max time to wait per pod")

	return cmd
}

func runRollback(_ *cobra.Command, _ []string) error {
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
		return fmt.Errorf("StatefulSet %s not found in %s: %w", statefulSetName, namespace, err)
	}

	// List controller revisions
	revisions, err := client.AppsV1().ControllerRevisions(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=luxd",
	})
	if err != nil {
		return fmt.Errorf("failed to list revisions: %w", err)
	}

	if len(revisions.Items) < 2 && rollbackRevision == 0 {
		return fmt.Errorf("no previous revision to rollback to (only %d revision(s) exist)", len(revisions.Items))
	}

	// Sort by revision number
	sort.Slice(revisions.Items, func(i, j int) bool {
		return revisions.Items[i].Revision < revisions.Items[j].Revision
	})

	// Find target revision
	var targetRev *appsv1.ControllerRevision
	if rollbackRevision > 0 {
		for i := range revisions.Items {
			if revisions.Items[i].Revision == rollbackRevision {
				targetRev = &revisions.Items[i]
				break
			}
		}
		if targetRev == nil {
			return fmt.Errorf("revision %d not found", rollbackRevision)
		}
	} else {
		// Previous revision = second to last
		targetRev = &revisions.Items[len(revisions.Items)-2]
	}

	ux.Logger.PrintToUser("Rollback plan:")
	ux.Logger.PrintToUser("  Namespace:       %s", namespace)
	ux.Logger.PrintToUser("  Current rev:     %s", sts.Status.CurrentRevision)
	ux.Logger.PrintToUser("  Target rev:      %s (revision %d)", targetRev.Name, targetRev.Revision)

	// Extract the pod template from the target revision's Data
	var revData struct {
		Spec struct {
			Template struct {
				Spec struct {
					Containers []struct {
						Image string `json:"image"`
						Name  string `json:"name"`
					} `json:"containers"`
				} `json:"spec"`
			} `json:"template"`
		} `json:"spec"`
	}

	if err := json.Unmarshal(targetRev.Data.Raw, &revData); err != nil {
		return fmt.Errorf("failed to parse revision data: %w", err)
	}

	targetImage := ""
	for _, c := range revData.Spec.Template.Spec.Containers {
		if c.Name == containerName {
			targetImage = c.Image
			break
		}
	}

	if targetImage != "" {
		ux.Logger.PrintToUser("  Target image:    %s", targetImage)
	}

	// Rollback by patching the StatefulSet with the target revision's template
	// We use the same partition-based approach for zero downtime
	replicas := int32(1)
	if sts.Spec.Replicas != nil {
		replicas = *sts.Spec.Replicas
	}

	// Apply the revision by restoring the pod template
	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"updateStrategy": map[string]interface{}{
				"type": "RollingUpdate",
				"rollingUpdate": map[string]interface{}{
					"partition": replicas,
				},
			},
		},
	}

	// If we have the target image, do a simple image rollback
	if targetImage != "" {
		ux.Logger.PrintToUser("\nRolling back to %s...", targetImage)

		// Set partition high first
		patchData, _ := json.Marshal(patch)
		if _, err := client.AppsV1().StatefulSets(namespace).Patch(
			ctx, statefulSetName, types.StrategicMergePatchType, patchData, metav1.PatchOptions{},
		); err != nil {
			return fmt.Errorf("failed to set partition: %w", err)
		}

		// Patch the image
		if err := patchImage(ctx, client, namespace, targetImage); err != nil {
			return fmt.Errorf("failed to patch image: %w", err)
		}

		// Roll pods one at a time
		for i := replicas - 1; i >= 0; i-- {
			podName := fmt.Sprintf("%s-%d", statefulSetName, i)
			ux.Logger.PrintToUser("\n[%d/%d] Rolling back %s...", replicas-i, replicas, podName)

			if err := setPartition(ctx, client, namespace, i); err != nil {
				return fmt.Errorf("failed to lower partition to %d: %w", i, err)
			}

			if err := waitForPodReady(ctx, client, namespace, podName, rollbackTimeout); err != nil {
				return fmt.Errorf("rollback failed at %s: %w", podName, err)
			}

			ux.Logger.PrintToUser("  %s ready", podName)
			time.Sleep(10 * time.Second) // shorter stability wait for rollback
		}

		ux.Logger.PrintToUser("\nRollback complete. All %d pods running %s", replicas, targetImage)
		return nil
	}

	return fmt.Errorf("could not determine target image from revision data")
}
