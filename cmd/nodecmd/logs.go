// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package nodecmd

import (
	"bufio"
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

var (
	follow    bool
	tailLines int64
)

func newLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs [pod-name]",
		Short: "Stream logs from a luxd pod",
		Long: `Streams logs from a specific luxd pod in the StatefulSet.

If no pod name is given, defaults to luxd-0.

EXAMPLES:
  lux node logs --mainnet
  lux node logs --mainnet luxd-2
  lux node logs --mainnet luxd-0 -f
  lux node logs --testnet --tail 100`,
		Args: cobra.MaximumNArgs(1),
		RunE: runLogs,
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "follow log output")
	cmd.Flags().Int64Var(&tailLines, "tail", 200, "number of recent lines to show")

	return cmd
}

func runLogs(_ *cobra.Command, args []string) error {
	namespace, err := resolveNamespace()
	if err != nil {
		return err
	}

	podName := fmt.Sprintf("%s-0", statefulSetName)
	if len(args) > 0 {
		podName = args[0]
	}

	client, err := newK8sClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	opts := &corev1.PodLogOptions{
		Container: containerName,
		Follow:    follow,
		TailLines: &tailLines,
	}

	req := client.CoreV1().Pods(namespace).GetLogs(podName, opts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return fmt.Errorf("failed to stream logs from %s/%s: %w", namespace, podName, err)
	}
	defer stream.Close()

	scanner := bufio.NewScanner(stream)
	// Increase buffer for long log lines
	scanner.Buffer(make([]byte, 0, 64*1024), 256*1024)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
	if err := scanner.Err(); err != nil && err != io.EOF {
		return err
	}
	return nil
}
