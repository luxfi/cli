// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package nodecmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

var (
	deployImage      string
	deployReplicas   int32
	deployEvmVersion string
	storageClass     string
	storageSize      string
	httpPort         int32
	stakingPort      int32
	networkID        uint32
	dbType           string
)

func newDeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy luxd StatefulSet to Kubernetes",
		Long: `Creates the full luxd k8s stack: namespace, headless service,
LoadBalancer service, genesis ConfigMap, and StatefulSet with PVCs.

If the resources already exist, they are updated in place.

EXAMPLES:
  lux node deploy --mainnet --replicas 5 --image ghcr.io/luxfi/node:v1.23.5
  lux node deploy --testnet --replicas 3
  lux node deploy --devnet --replicas 3 --db-type zapdb`,
		RunE: runDeploy,
	}

	cmd.Flags().StringVar(&deployImage, "image", "ghcr.io/luxfi/node:latest", "container image")
	cmd.Flags().Int32Var(&deployReplicas, "replicas", 5, "number of validator replicas")
	cmd.Flags().StringVar(&deployEvmVersion, "evm-version", "v0.8.35", "EVM plugin release version")
	cmd.Flags().StringVar(&storageClass, "storage-class", "do-block-storage", "k8s storage class for PVCs")
	cmd.Flags().StringVar(&storageSize, "storage-size", "100Gi", "PVC size per node")
	cmd.Flags().Int32Var(&httpPort, "http-port", 9630, "HTTP API port")
	cmd.Flags().Int32Var(&stakingPort, "staking-port", 9631, "staking port")
	cmd.Flags().Uint32Var(&networkID, "network-id", 0, "network ID (auto-detected from network flag)")
	cmd.Flags().StringVar(&dbType, "db-type", "zapdb", "database engine (zapdb, pebbledb, badgerdb)")

	return cmd
}

func runDeploy(_ *cobra.Command, _ []string) error {
	namespace, err := resolveNamespace()
	if err != nil {
		return err
	}

	// Auto-detect network ID from namespace
	if networkID == 0 {
		switch namespace {
		case "lux-mainnet":
			networkID = 96369
		case "lux-testnet":
			networkID = 96368
		case "lux-devnet":
			networkID = 96370
		default:
			return fmt.Errorf("--network-id required for custom namespace")
		}
	}

	client, err := newK8sClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	ux.Logger.PrintToUser("Deploying luxd to Kubernetes:")
	ux.Logger.PrintToUser("  Namespace:    %s", namespace)
	ux.Logger.PrintToUser("  Replicas:     %d", deployReplicas)
	ux.Logger.PrintToUser("  Image:        %s", deployImage)
	ux.Logger.PrintToUser("  EVM version:  %s", deployEvmVersion)
	ux.Logger.PrintToUser("  Network ID:   %d", networkID)
	ux.Logger.PrintToUser("  DB type:      %s", dbType)
	ux.Logger.PrintToUser("  Storage:      %s (%s)", storageSize, storageClass)

	// 1. Namespace
	ux.Logger.PrintToUser("\nCreating namespace...")
	if err := ensureNamespace(ctx, client, namespace); err != nil {
		return err
	}

	// 2. Genesis ConfigMap
	ux.Logger.PrintToUser("Creating genesis ConfigMap...")
	if err := ensureGenesisConfigMap(ctx, client, namespace); err != nil {
		return err
	}

	// 3. Headless service
	ux.Logger.PrintToUser("Creating headless service...")
	if err := ensureHeadlessService(ctx, client, namespace); err != nil {
		return err
	}

	// 4. LoadBalancer service
	ux.Logger.PrintToUser("Creating LoadBalancer service...")
	if err := ensureLBService(ctx, client, namespace); err != nil {
		return err
	}

	// 5. StatefulSet
	ux.Logger.PrintToUser("Creating StatefulSet...")
	if err := ensureStatefulSet(ctx, client, namespace); err != nil {
		return err
	}

	// Wait for pods
	ux.Logger.PrintToUser("Waiting for pods to be ready...")
	if err := waitForAllReady(ctx, client, namespace, deployReplicas, 10*time.Minute); err != nil {
		return err
	}

	// Report external IP
	svc, err := client.CoreV1().Services(namespace).Get(ctx, "luxd", metav1.GetOptions{})
	if err == nil {
		for _, ingress := range svc.Status.LoadBalancer.Ingress {
			if ingress.IP != "" {
				ux.Logger.PrintToUser("\nDeployed successfully!")
				ux.Logger.PrintToUser("  RPC: http://%s:%d/ext/bc/C/rpc", ingress.IP, httpPort)
				return nil
			}
		}
	}

	ux.Logger.PrintToUser("\nDeployed successfully! (LoadBalancer IP pending)")
	return nil
}

func ensureNamespace(ctx context.Context, client *kubernetes.Clientset, namespace string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   namespace,
			Labels: map[string]string{"app": "luxd"},
		},
	}
	_, err := client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func ensureGenesisConfigMap(ctx context.Context, client *kubernetes.Clientset, namespace string) error {
	// Determine network name from namespace
	netName := "mainnet"
	switch namespace {
	case "lux-testnet":
		netName = "testnet"
	case "lux-devnet":
		netName = "devnet"
	}

	home, _ := os.UserHomeDir()
	genesisPath := filepath.Join(home, "work", "lux", "genesis", "configs", netName, "genesis.json")

	genesisJSON, err := os.ReadFile(genesisPath)
	if err != nil {
		ux.Logger.PrintToUser("  Warning: genesis not found at %s, using minimal", genesisPath)
		genesisJSON = []byte(fmt.Sprintf(`{"networkID": %d}`, networkID))
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "luxd-genesis",
			Namespace: namespace,
		},
		Data: map[string]string{"genesis.json": string(genesisJSON)},
	}

	_, err = client.CoreV1().ConfigMaps(namespace).Create(ctx, cm, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		_, err = client.CoreV1().ConfigMaps(namespace).Update(ctx, cm, metav1.UpdateOptions{})
	}
	return err
}

func ensureHeadlessService(ctx context.Context, client *kubernetes.Clientset, namespace string) error {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "luxd-headless",
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Selector:  map[string]string{"app": "luxd"},
			Ports: []corev1.ServicePort{
				{Name: "http", Port: httpPort, TargetPort: intstr.FromInt32(httpPort)},
				{Name: "staking", Port: stakingPort, TargetPort: intstr.FromInt32(stakingPort)},
			},
		},
	}
	_, err := client.CoreV1().Services(namespace).Create(ctx, svc, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func ensureLBService(ctx context.Context, client *kubernetes.Clientset, namespace string) error {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "luxd",
			Namespace: namespace,
			Annotations: map[string]string{
				"service.beta.kubernetes.io/do-loadbalancer-name": fmt.Sprintf("luxd-%s", namespace),
			},
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeLoadBalancer,
			Selector: map[string]string{"app": "luxd"},
			Ports: []corev1.ServicePort{
				{Name: "http", Port: httpPort, TargetPort: intstr.FromInt32(httpPort)},
				{Name: "staking", Port: stakingPort, TargetPort: intstr.FromInt32(stakingPort)},
			},
		},
	}
	_, err := client.CoreV1().Services(namespace).Create(ctx, svc, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func ensureStatefulSet(ctx context.Context, client *kubernetes.Clientset, namespace string) error {
	evmPluginURL := fmt.Sprintf(
		"https://github.com/luxfi/evm/releases/download/%s/evm-plugin-linux-amd64",
		deployEvmVersion,
	)

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      statefulSetName,
			Namespace: namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: "luxd-headless",
			Replicas:    &deployReplicas,
			Selector:    &metav1.LabelSelector{MatchLabels: map[string]string{"app": "luxd"}},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":         "luxd",
						"lux-network": namespace,
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
								fmt.Sprintf("curl -sL %s -o /data/plugins/mgj786NP7uDwBCcq6YwThhaN8FLyybkCa4zBWTQbNgmK6k9A6 && ", evmPluginURL) +
								"chmod +x /data/plugins/* && " +
								"chown -R 1000:1000 /data && " +
								"echo 'Plugins installed successfully'",
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "data", MountPath: "/data"},
						},
					}},
					Containers: []corev1.Container{{
						Name:            containerName,
						Image:           deployImage,
						ImagePullPolicy: corev1.PullAlways,
						Ports: []corev1.ContainerPort{
							{Name: "http", ContainerPort: httpPort},
							{Name: "staking", ContainerPort: stakingPort},
						},
						Command: []string{"/app/luxd"},
						Args: []string{
							fmt.Sprintf("--network-id=%d", networkID),
							"--http-host=0.0.0.0",
							fmt.Sprintf("--http-port=%d", httpPort),
							"--http-allowed-hosts=*",
							fmt.Sprintf("--staking-port=%d", stakingPort),
							"--data-dir=/data",
							"--genesis-file=/genesis/genesis.json",
							fmt.Sprintf("--db-type=%s", dbType),
							"--index-enabled=true",
							"--api-admin-enabled=true",
							"--log-level=info",
							"--plugin-dir=/data/plugins",
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "data", MountPath: "/data"},
							{Name: "genesis", MountPath: "/genesis", ReadOnly: true},
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("1Gi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("2"),
								corev1.ResourceMemory: resource.MustParse("4Gi"),
							},
						},
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: healthPath,
									Port: intstr.FromInt32(httpPort),
								},
							},
							InitialDelaySeconds: 60,
							PeriodSeconds:       30,
							FailureThreshold:    5,
						},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: healthPath,
									Port: intstr.FromInt32(httpPort),
								},
							},
							InitialDelaySeconds: 10,
							PeriodSeconds:       10,
							FailureThreshold:    3,
						},
					}},
					Volumes: []corev1.Volume{{
						Name: "genesis",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: "luxd-genesis"},
							},
						},
					}},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{
				ObjectMeta: metav1.ObjectMeta{Name: "data"},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					StorageClassName: &storageClass,
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse(storageSize),
						},
					},
				},
			}},
		},
	}

	_, err := client.AppsV1().StatefulSets(namespace).Create(ctx, sts, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		// For existing StatefulSet, update only the template (not VolumeClaimTemplates)
		existing, getErr := client.AppsV1().StatefulSets(namespace).Get(ctx, statefulSetName, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		existing.Spec.Template = sts.Spec.Template
		existing.Spec.Replicas = sts.Spec.Replicas
		_, err = client.AppsV1().StatefulSets(namespace).Update(ctx, existing, metav1.UpdateOptions{})
	}
	return err
}

func waitForAllReady(ctx context.Context, client *kubernetes.Clientset, namespace string, replicas int32, timeout time.Duration) error {
	deadline := time.After(timeout)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			return fmt.Errorf("timeout waiting for %d replicas to be ready", replicas)
		case <-ticker.C:
			sts, err := client.AppsV1().StatefulSets(namespace).Get(ctx, statefulSetName, metav1.GetOptions{})
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

func int64Ptr(i int64) *int64 {
	return &i
}
