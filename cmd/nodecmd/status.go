// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package nodecmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show luxd StatefulSet status, pod images, and health",
		Long: `Displays the current state of the luxd Kubernetes deployment.

Shows:
  - StatefulSet metadata (replicas, revision, update strategy)
  - Per-pod status (ready, image, restarts, age)
  - LoadBalancer external IP
  - Revision history for rollback

EXAMPLES:
  lux node status --mainnet
  lux node status --testnet
  lux node status --namespace my-custom-ns`,
		RunE: runStatus,
	}
	return cmd
}

func runStatus(_ *cobra.Command, _ []string) error {
	namespace, err := resolveNamespace()
	if err != nil {
		return err
	}

	client, err := newK8sClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Get StatefulSet
	sts, err := client.AppsV1().StatefulSets(namespace).Get(ctx, statefulSetName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("StatefulSet %s not found in %s: %w", statefulSetName, namespace, err)
	}

	replicas := int32(0)
	if sts.Spec.Replicas != nil {
		replicas = *sts.Spec.Replicas
	}

	// StatefulSet summary
	ux.Logger.PrintToUser("StatefulSet: %s/%s", namespace, statefulSetName)
	ux.Logger.PrintToUser("  Replicas:     %d desired, %d ready, %d current",
		replicas, sts.Status.ReadyReplicas, sts.Status.CurrentReplicas)
	ux.Logger.PrintToUser("  Revision:     %s", sts.Status.CurrentRevision)
	if sts.Status.UpdateRevision != sts.Status.CurrentRevision {
		ux.Logger.PrintToUser("  Update rev:   %s (update in progress)", sts.Status.UpdateRevision)
	}
	if sts.Spec.UpdateStrategy.RollingUpdate != nil && sts.Spec.UpdateStrategy.RollingUpdate.Partition != nil {
		ux.Logger.PrintToUser("  Partition:    %d", *sts.Spec.UpdateStrategy.RollingUpdate.Partition)
	}

	// Template image
	for _, c := range sts.Spec.Template.Spec.Containers {
		if c.Name == containerName {
			ux.Logger.PrintToUser("  Template img: %s", c.Image)
			break
		}
	}

	// Pod table
	ux.Logger.PrintToUser("")
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=luxd",
	})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Pod", "Status", "Ready", "Image", "Restarts", "Age")

	for _, pod := range pods.Items {
		status := string(pod.Status.Phase)
		ready := "false"
		image := ""
		restarts := 0

		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Name == containerName {
				image = cs.Image
				restarts = int(cs.RestartCount)
				if cs.Ready {
					ready = "true"
				}
				break
			}
		}

		age := time.Since(pod.CreationTimestamp.Time).Truncate(time.Second)
		_ = table.Append([]string{
			pod.Name,
			status,
			ready,
			image,
			fmt.Sprintf("%d", restarts),
			formatDuration(age),
		})
	}
	_ = table.Render()

	// LoadBalancer IP
	svc, err := client.CoreV1().Services(namespace).Get(ctx, "luxd", metav1.GetOptions{})
	if err == nil {
		for _, ingress := range svc.Status.LoadBalancer.Ingress {
			if ingress.IP != "" {
				ux.Logger.PrintToUser("\nLoadBalancer: %s", ingress.IP)
				ux.Logger.PrintToUser("  RPC:     http://%s:9630/ext/bc/C/rpc", ingress.IP)
				ux.Logger.PrintToUser("  Health:  http://%s:9630/ext/health", ingress.IP)
			}
		}
	}

	// Controller revisions (for rollback info)
	revisions, err := client.AppsV1().ControllerRevisions(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=luxd",
	})
	if err == nil && len(revisions.Items) > 1 {
		ux.Logger.PrintToUser("\nRevision history (latest %d):", min(len(revisions.Items), 5))
		for i := len(revisions.Items) - 1; i >= 0 && i >= len(revisions.Items)-5; i-- {
			rev := revisions.Items[i]
			marker := ""
			if rev.Name == sts.Status.CurrentRevision {
				marker = " (current)"
			}
			ux.Logger.PrintToUser("  %s  revision=%d%s", rev.Name, rev.Revision, marker)
		}
	}

	return nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}
