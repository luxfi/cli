// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/luxfi/netrunner/client"
	"github.com/luxfi/cli/pkg/binutils"
)

func main() {
	subnetID := "Y1irJDjzPA69zqMgg2wZ6r1ufWcSpSpWoLoGchRugYsp7bvRz"
	blockchainID := "2NmYKwKZ4Ewvki6rZQBeMhXeMxDmfKox14fyqnz1KFXVJnXX2e"

	fmt.Printf("Tracking subnet:\n  SubnetID: %s\n  BlockchainID: %s\n", subnetID, blockchainID)

	cli, err := binutils.NewGRPCClient()
	if err != nil {
		log.Fatalf("failed to create gRPC client: %v", err)
	}

	rootCtx := context.Background()
	ctx, cancel := context.WithTimeout(rootCtx, 30*time.Second)
	defer cancel()

	resp, err := cli.Status(ctx)
	if err != nil {
		log.Fatalf("failed to get status: %v", err)
	}

	fmt.Printf("Cluster has %d nodes\n", len(resp.ClusterInfo.NodeNames))

	// First SAVE the current snapshot (to persist ZOO blockchain P-Chain state)
	ctx3, cancel3 := context.WithTimeout(rootCtx, 60*time.Second)
	defer cancel3()
	fmt.Printf("\nSaving current snapshot with ZOO blockchain state...\n")
	_, err = cli.SaveSnapshot(ctx3, "zoo_snapshot")
	if err != nil {
		fmt.Printf("Note: SaveSnapshot returned: %v\n", err)
	} else {
		fmt.Println("Snapshot saved successfully")
	}

	// Load the saved snapshot to restart the network
	ctx4, cancel4 := context.WithTimeout(rootCtx, 60*time.Second)
	defer cancel4()
	fmt.Printf("\nLoading saved snapshot...\n")
	_, err = cli.LoadSnapshot(ctx4, "zoo_snapshot")
	if err != nil {
		fmt.Printf("Note: LoadSnapshot returned: %v\n", err)
	} else {
		fmt.Println("Snapshot loaded successfully")
	}

	// Wait for network to be healthy before restarting nodes with tracking
	time.Sleep(5 * time.Second)

	// Restart each node with the whitelisted subnet
	for _, nodeName := range resp.ClusterInfo.NodeNames {
		ctx, cancel := context.WithTimeout(rootCtx, 120*time.Second)
		fmt.Printf("Restarting node %s with subnet %s...\n", nodeName, subnetID)
		_, err := cli.RestartNode(ctx, nodeName,
			client.WithWhitelistedSubnets(subnetID),
			client.WithChainConfigs(map[string]string{
				blockchainID: `{}`,
			}),
		)
		cancel()
		if err != nil {
			log.Printf("Warning: failed to restart node %s: %v", nodeName, err)
		} else {
			fmt.Printf("  Node %s restarted successfully\n", nodeName)
		}
	}

	fmt.Println("\nAll nodes restarted. Waiting for network health check...")
	time.Sleep(15 * time.Second)

	ctx2, cancel2 := context.WithTimeout(rootCtx, 30*time.Second)
	defer cancel2()
	healthResp, err := cli.Health(ctx2)
	if err != nil {
		log.Printf("Health check failed: %v", err)
	} else {
		fmt.Printf("Network healthy: %v\n", healthResp.ClusterInfo.Healthy)
	}
}
