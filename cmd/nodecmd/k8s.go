// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package nodecmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// k8s flags shared across subcommands
var (
	flagContext   string // --context (k8s context override)
	flagNamespace string // --namespace (override auto-detection)
	flagMainnet   bool   // --mainnet
	flagTestnet   bool   // --testnet
	flagDevnet    bool   // --devnet
)

const (
	statefulSetName = "luxd"
	containerName   = "luxd"
	healthPath      = "/ext/health"
	defaultHTTPPort = int32(9630)
)

// resolveNamespace returns the k8s namespace from flags.
func resolveNamespace() (string, error) {
	if flagNamespace != "" {
		return flagNamespace, nil
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
		return "lux-mainnet", nil
	case flagTestnet:
		return "lux-testnet", nil
	case flagDevnet:
		return "lux-devnet", nil
	default:
		return "", fmt.Errorf("specify --mainnet, --testnet, --devnet, or --namespace")
	}
}

// newK8sClient creates a kubernetes clientset using kubeconfig with optional context override.
func newK8sClient() (*kubernetes.Clientset, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot determine home directory: %w", err)
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	rules.ExplicitPath = kubeconfig
	overrides := &clientcmd.ConfigOverrides{}
	if flagContext != "" {
		overrides.CurrentContext = flagContext
	}

	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}
	return client, nil
}

// podReady checks if a pod's readiness conditions indicate it is ready.
func podReady(ctx context.Context, client *kubernetes.Clientset, namespace, podName string) (bool, error) {
	pod, err := client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
			return true, nil
		}
	}
	return false, nil
}

// podImage returns the current image of the luxd container in a pod.
func podImage(ctx context.Context, client *kubernetes.Clientset, namespace, podName string) (string, error) {
	pod, err := client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	for _, c := range pod.Status.ContainerStatuses {
		if c.Name == containerName {
			return c.Image, nil
		}
	}
	return "", fmt.Errorf("container %s not found in pod %s", containerName, podName)
}

// int32Ptr returns a pointer to an int32 value.
func int32Ptr(i int32) *int32 {
	return &i
}
