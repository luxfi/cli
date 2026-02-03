// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package networkcmd

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/luxfi/cli/pkg/ux"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// K8sNetworkConfig holds configuration for K8s network deployment
type K8sNetworkConfig struct {
	NetworkName  string
	NetworkID    uint32
	ChainID      uint64
	Namespace    string
	HTTPPort     int32
	StakingPort  int32
	Replicas     int32
	Image        string
	StorageClass string
	GenesisJSON  string
}

// StartK8sNetwork deploys a Lux network to Kubernetes
func StartK8sNetwork(cfg K8sNetworkConfig) error {
	ctx := context.Background()

	// Load kubeconfig
	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	// Use specific context if provided
	if k8sCluster != "" {
		rules := clientcmd.NewDefaultClientConfigLoadingRules()
		overrides := &clientcmd.ConfigOverrides{CurrentContext: k8sCluster}
		config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
		if err != nil {
			return fmt.Errorf("failed to load kubeconfig for context %s: %w", k8sCluster, err)
		}
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create k8s client: %w", err)
	}

	ux.Logger.PrintToUser("Deploying %s to Kubernetes cluster...", cfg.NetworkName)
	ux.Logger.PrintToUser("  Namespace: %s", cfg.Namespace)
	ux.Logger.PrintToUser("  Replicas: %d", cfg.Replicas)
	ux.Logger.PrintToUser("  Image: %s", cfg.Image)

	// Create namespace
	if err := createNamespace(ctx, client, cfg.Namespace); err != nil {
		return err
	}

	// Create genesis ConfigMap
	if err := createGenesisConfigMap(ctx, client, cfg); err != nil {
		return err
	}

	// Create headless service
	if err := createHeadlessService(ctx, client, cfg); err != nil {
		return err
	}

	// Create LoadBalancer service
	if err := createLoadBalancerService(ctx, client, cfg); err != nil {
		return err
	}

	// Create StatefulSet
	if err := createStatefulSet(ctx, client, cfg); err != nil {
		return err
	}

	ux.Logger.PrintToUser("Waiting for pods to be ready...")
	if err := waitForReady(ctx, client, cfg.Namespace, cfg.Replicas); err != nil {
		return err
	}

	// Get external IP
	externalIP, err := getExternalIP(ctx, client, cfg.Namespace)
	if err != nil {
		ux.Logger.PrintToUser("Warning: could not get external IP: %v", err)
	} else {
		ux.Logger.PrintToUser("\n%s deployed successfully!", cfg.NetworkName)
		ux.Logger.PrintToUser("  RPC: http://%s:%d/ext/bc/C/rpc", externalIP, cfg.HTTPPort)
		ux.Logger.PrintToUser("  Staking: %s:%d", externalIP, cfg.StakingPort)
	}

	return nil
}

func createNamespace(ctx context.Context, client *kubernetes.Clientset, namespace string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Labels: map[string]string{
				"app": "luxd",
			},
		},
	}
	_, err := client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func createGenesisConfigMap(ctx context.Context, client *kubernetes.Clientset, cfg K8sNetworkConfig) error {
	// Try to read genesis from user's genesis repo
	home, _ := os.UserHomeDir()
	genesisPath := filepath.Join(home, "work", "lux", "genesis", "configs", cfg.NetworkName, "genesis.json")

	genesisJSON, err := os.ReadFile(genesisPath)
	if err != nil {
		// Fall back to minimal genesis if file not found
		ux.Logger.PrintToUser("Warning: genesis file not found at %s, using minimal genesis", genesisPath)
		genesisJSON = []byte(fmt.Sprintf(`{"networkID": %d}`, cfg.NetworkID))
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "luxd-genesis",
			Namespace: cfg.Namespace,
		},
		Data: map[string]string{
			"genesis.json": string(genesisJSON),
		},
	}

	_, err = client.CoreV1().ConfigMaps(cfg.Namespace).Create(ctx, cm, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		_, err = client.CoreV1().ConfigMaps(cfg.Namespace).Update(ctx, cm, metav1.UpdateOptions{})
	}
	return err
}

func createHeadlessService(ctx context.Context, client *kubernetes.Clientset, cfg K8sNetworkConfig) error {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "luxd-headless",
			Namespace: cfg.Namespace,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Selector: map[string]string{
				"app": "luxd",
			},
			Ports: []corev1.ServicePort{
				{Name: "http", Port: cfg.HTTPPort, TargetPort: intstr.FromInt32(cfg.HTTPPort)},
				{Name: "staking", Port: cfg.StakingPort, TargetPort: intstr.FromInt32(cfg.StakingPort)},
			},
		},
	}

	_, err := client.CoreV1().Services(cfg.Namespace).Create(ctx, svc, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func createLoadBalancerService(ctx context.Context, client *kubernetes.Clientset, cfg K8sNetworkConfig) error {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "luxd",
			Namespace: cfg.Namespace,
			Annotations: map[string]string{
				"service.beta.kubernetes.io/do-loadbalancer-name": fmt.Sprintf("luxd-%s", cfg.NetworkName),
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeLoadBalancer,
			Selector: map[string]string{
				"app": "luxd",
			},
			Ports: []corev1.ServicePort{
				{Name: "http", Port: cfg.HTTPPort, TargetPort: intstr.FromInt32(cfg.HTTPPort)},
				{Name: "staking", Port: cfg.StakingPort, TargetPort: intstr.FromInt32(cfg.StakingPort)},
			},
		},
	}

	_, err := client.CoreV1().Services(cfg.Namespace).Create(ctx, svc, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func createStatefulSet(ctx context.Context, client *kubernetes.Clientset, cfg K8sNetworkConfig) error {
	storageClass := cfg.StorageClass
	if storageClass == "" {
		storageClass = "do-block-storage"
	}

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "luxd",
			Namespace: cfg.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: "luxd-headless",
			Replicas:    &cfg.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "luxd"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":         "luxd",
						"lux-network": cfg.NetworkName,
					},
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup:    int64Ptr(1000),
						RunAsUser:  int64Ptr(1000),
						RunAsGroup: int64Ptr(1000),
					},
					InitContainers: []corev1.Container{{
						Name:  "init-plugins",
						Image: "curlimages/curl:latest",
						SecurityContext: &corev1.SecurityContext{
							RunAsUser:  int64Ptr(0),
							RunAsGroup: int64Ptr(0),
						},
						Command: []string{"/bin/sh", "-c"},
						Args: []string{
							"mkdir -p /data/plugins /data/staking && " +
								"curl -sL https://github.com/luxfi/evm/releases/download/v0.8.35/evm-plugin-linux-amd64 -o /data/plugins/mgj786NP7uDwBCcq6YwThhaN8FLyybkCa4zBWTQbNgmK6k9A6 && " +
								"chmod +x /data/plugins/* && " +
								"chown -R 1000:1000 /data && " +
								"echo 'Plugins installed successfully'",
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "data", MountPath: "/data"},
						},
					}},
					Containers: []corev1.Container{{
						Name:            "luxd",
						Image:           cfg.Image,
						ImagePullPolicy: corev1.PullAlways,
						Ports: []corev1.ContainerPort{
							{Name: "http", ContainerPort: cfg.HTTPPort},
							{Name: "staking", ContainerPort: cfg.StakingPort},
						},
						Env: []corev1.EnvVar{
							{Name: "HOME", Value: "/data"},
							{Name: "NETWORK_ID", Value: fmt.Sprintf("%d", cfg.NetworkID)},
							{Name: "HTTP_HOST", Value: "0.0.0.0"},
							{Name: "HTTP_PORT", Value: fmt.Sprintf("%d", cfg.HTTPPort)},
							{Name: "STAKING_PORT", Value: fmt.Sprintf("%d", cfg.StakingPort)},
							{Name: "LOG_LEVEL", Value: "info"},
							{Name: "DB_TYPE", Value: "pebbledb"},
							{Name: "INDEX_ENABLED", Value: "true"},
							{Name: "API_ADMIN_ENABLED", Value: "true"},
						},
						Command: []string{"/bin/sh", "-c"},
						Args: []string{
							fmt.Sprintf("mkdir -p /data/plugins && exec /luxd/build/luxd "+
								"--network-id=%d "+
								"--http-host=0.0.0.0 "+
								"--http-port=%d "+
								"--staking-port=%d "+
								"--data-dir=/data "+
								"--genesis-file=/genesis/genesis.json "+
								"--db-type=badgerdb "+
								"--index-enabled=true "+
								"--api-admin-enabled=true "+
								"--log-level=info "+
								"--plugin-dir=/data/plugins",
								cfg.NetworkID, cfg.HTTPPort, cfg.StakingPort),
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "data", MountPath: "/data"},
							{Name: "genesis", MountPath: "/genesis", ReadOnly: true},
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse("1Gi"),
								corev1.ResourceCPU:    resource.MustParse("500m"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse("4Gi"),
								corev1.ResourceCPU:    resource.MustParse("2"),
							},
						},
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								TCPSocket: &corev1.TCPSocketAction{
									Port: intstr.FromInt32(cfg.HTTPPort),
								},
							},
							InitialDelaySeconds: 30,
							PeriodSeconds:       10,
						},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								TCPSocket: &corev1.TCPSocketAction{
									Port: intstr.FromInt32(cfg.HTTPPort),
								},
							},
							InitialDelaySeconds: 10,
							PeriodSeconds:       5,
						},
					}},
					Volumes: []corev1.Volume{
						{
							Name: "genesis",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{Name: "luxd-genesis"},
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{
				ObjectMeta: metav1.ObjectMeta{Name: "data"},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					StorageClassName: &storageClass,
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("100Gi"),
						},
					},
				},
			}},
		},
	}

	_, err := client.AppsV1().StatefulSets(cfg.Namespace).Create(ctx, sts, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		_, err = client.AppsV1().StatefulSets(cfg.Namespace).Update(ctx, sts, metav1.UpdateOptions{})
	}
	return err
}

func waitForReady(ctx context.Context, client *kubernetes.Clientset, namespace string, replicas int32) error {
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for pods to be ready")
		case <-ticker.C:
			sts, err := client.AppsV1().StatefulSets(namespace).Get(ctx, "luxd", metav1.GetOptions{})
			if err != nil {
				continue
			}
			if sts.Status.ReadyReplicas == replicas {
				return nil
			}
			ux.Logger.PrintToUser("  Ready: %d/%d", sts.Status.ReadyReplicas, replicas)
		}
	}
}

func getExternalIP(ctx context.Context, client *kubernetes.Clientset, namespace string) (string, error) {
	// Wait for LoadBalancer IP assignment
	timeout := time.After(2 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return "", fmt.Errorf("timeout waiting for external IP")
		case <-ticker.C:
			svc, err := client.CoreV1().Services(namespace).Get(ctx, "luxd", metav1.GetOptions{})
			if err != nil {
				continue
			}
			for _, ingress := range svc.Status.LoadBalancer.Ingress {
				if ingress.IP != "" {
					return ingress.IP, nil
				}
			}
		}
	}
}

func int64Ptr(i int64) *int64 {
	return &i
}

// StartK8sMainnet deploys mainnet to Kubernetes
func StartK8sMainnet() error {
	return StartK8sNetwork(K8sNetworkConfig{
		NetworkName:  "mainnet",
		NetworkID:    1,
		ChainID:      96369,
		Namespace:    "lux-mainnet",
		HTTPPort:     9630,
		StakingPort:  9631,
		Replicas:     5,
		Image:        k8sImage,
		StorageClass: "do-block-storage",
	})
}

// StartK8sTestnet deploys testnet to Kubernetes
func StartK8sTestnet() error {
	return StartK8sNetwork(K8sNetworkConfig{
		NetworkName:  "testnet",
		NetworkID:    2,
		ChainID:      96368,
		Namespace:    "lux-testnet",
		HTTPPort:     9640,
		StakingPort:  9641,
		Replicas:     5,
		Image:        k8sImage,
		StorageClass: "do-block-storage",
	})
}

// StartK8sDevnet deploys devnet to Kubernetes
func StartK8sDevnet() error {
	return StartK8sNetwork(K8sNetworkConfig{
		NetworkName:  "devnet",
		NetworkID:    3,
		ChainID:      96370,
		Namespace:    "lux-devnet",
		HTTPPort:     9650,
		StakingPort:  9651,
		Replicas:     5,
		Image:        k8sImage,
		StorageClass: "do-block-storage",
	})
}
