// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package networkcmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/netrunner/server"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	verbose       bool
	statusMainnet bool
	statusTestnet bool
	statusDevnet  bool
	statusAll     bool
)

// NewStatusCmdOld returns the old status command.
// Deprecated: Use the new status command instead.
func NewStatusCmdOld() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show network status and endpoints",
		Long: `The network status command shows detailed information about running networks.

OVERVIEW:

  Displays network health, validator nodes, endpoints, and custom chains.
  Checks status of all locally managed networks (mainnet, testnet, devnet, custom).

NETWORK FLAGS:

  --mainnet, -m    Check mainnet status (port 9630, gRPC 8369)
  --testnet, -t    Check testnet status (port 9640, gRPC 8368)
  --devnet, -d     Check devnet status (port 9650, gRPC 8370)
  --all            Check all network types (default behavior)

OPTIONS:

  --verbose, -v    Show detailed cluster info including raw protobuf response

OUTPUT INCLUDES:

  - Network health status
  - Number of validator nodes
  - Node endpoints (RPC, staking)
  - C-Chain block height
  - Custom chain endpoints (deployed chains)
  - Node version and VM info
  - gRPC server information

EXAMPLES:

  # Check all networks
  lux network status

  # Check specific network type
  lux network status --devnet
  lux network status -d

  # Verbose output with full cluster details
  lux network status --verbose

TYPICAL OUTPUT:

  Devnet Network is Up (gRPC port: 8370)
  ============================================
  Healthy: true
  Number of nodes: 5
  Number of custom VMs: 1
  -------- Node information --------
  node1 has ID NodeID-xxx and endpoint http://127.0.0.1:9650
  Version: lux/1.0.0...
  C-Chain Height: 1234
  ...

NOTES:

  - Only running networks will show full status
  - Stopped networks will be listed as Stopped
  - Use after 'lux network start' to verify successful startup`,

		RunE:          networkStatus,
		Args:          cobra.ExactArgs(0),
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "show detailed cluster info including raw protobuf response")
	cmd.Flags().BoolVarP(&statusMainnet, "mainnet", "m", false, "check mainnet network status")
	cmd.Flags().BoolVarP(&statusTestnet, "testnet", "t", false, "check testnet network status")
	cmd.Flags().BoolVarP(&statusDevnet, "devnet", "d", false, "check devnet network status")
	cmd.Flags().BoolVar(&statusAll, "all", false, "check status of all networks")

	return cmd
}

func networkStatus(cmd *cobra.Command, args []string) error {
	// Determine which networks to check
	networksToCheck := []string{}
	if statusAll || (!statusMainnet && !statusTestnet && !statusDevnet) {
		networksToCheck = []string{"mainnet", "testnet", "devnet", "custom"}
	} else {
		if statusMainnet {
			networksToCheck = append(networksToCheck, "mainnet")
		}
		if statusTestnet {
			networksToCheck = append(networksToCheck, "testnet")
		}
		if statusDevnet {
			networksToCheck = append(networksToCheck, "devnet")
		}
	}

	var wg sync.WaitGroup
	results := make([]string, len(networksToCheck))
	errors := make([]error, len(networksToCheck))

	// Check networks in parallel
	for i, netType := range networksToCheck {
		wg.Add(1)
		go func(index int, nt string) {
			defer wg.Done()

			// Check if process is running first to avoid timeout
			running, err := binutils.IsServerProcessRunningForNetwork(app, nt)
			if err != nil {
				// Don't error out completely, just record it
				// But IsServerProcessRunningForNetwork returns error if PID file checks fail in a bad way
				// Use debug log?
				errors[index] = fmt.Errorf("failed to check process status: %w", err)
				return
			}
			if !running {
				results[index] = fmt.Sprintf("%s: Stopped", strings.Title(nt))
				return
			}

			// Get detailed status
			out, err := getNetworkStatusOutput(nt)
			if err != nil {
				errors[index] = err
				// If error is timeout or not connected, say so
				if strings.Contains(err.Error(), "timed out") || strings.Contains(err.Error(), "connection refused") {
					results[index] = fmt.Sprintf("%s: Not reachable (process running but unresponsive)", strings.Title(nt))
				} else {
					results[index] = fmt.Sprintf("%s: Error - %v", strings.Title(nt), err)
				}
			} else {
				results[index] = out
			}
		}(i, netType)
	}
	wg.Wait()

	// Print results in order
	anyRunning := false
	for i, res := range results {
		if errors[i] != nil {
			// Only print error if it's not just "not exist" or similar
			ux.Logger.RedXToUser("%s: %v", networksToCheck[i], errors[i])
		} else if res != "" {
			if !strings.Contains(res, "Stopped") && !strings.Contains(res, "Not reachable") {
				anyRunning = true
			}
			ux.Logger.PrintToUser("%s", res)
		}
	}

	if !anyRunning && len(networksToCheck) == 4 && !statusAll {
		ux.Logger.PrintToUser("\nNo networks are currently running.")
	}

	return nil
}

func getNetworkStatusOutput(networkType string) (string, error) {
	var buf bytes.Buffer

	cli, err := binutils.NewGRPCClient(binutils.WithNetworkType(networkType))
	if err != nil {
		return "", err
	}
	defer func() { _ = cli.Close() }()

	ctx := binutils.GetAsyncContext()
	status, err := cli.Status(ctx)
	if err != nil {
		if server.IsServerError(err, server.ErrNotBootstrapped) {
			return fmt.Sprintf("%s: Not running (not bootstrapped)", networkType), nil
		}
		return "", err
	}

	// Use adaptive layout for different screen sizes
	const maxWidth = 100
	width := getTerminalWidth()
	if width > maxWidth {
		width = maxWidth
	}
	separator := strings.Repeat("=", width)
	nodeSeparator := strings.Repeat("-", width/2)

	if status == nil || status.ClusterInfo == nil {
		return "", fmt.Errorf("no %s network running", networkType)
	}

	// Get port info from gRPC ports config
	grpcPorts := binutils.GetGRPCPorts(networkType)

	fmt.Fprintf(&buf, "\n%s Network is Up (gRPC port: %d)\n", strings.ToUpper(networkType[:1])+networkType[1:], grpcPorts.Server)
	fmt.Fprintf(&buf, "%s\n", separator)
	fmt.Fprintf(&buf, "Healthy: %t\n", status.ClusterInfo.Healthy)
	fmt.Fprintf(&buf, "Custom VMs healthy: %t\n", status.ClusterInfo.CustomChainsHealthy)
	fmt.Fprintf(&buf, "Number of nodes: %d\n", len(status.ClusterInfo.NodeNames))
	fmt.Fprintf(&buf, "Number of custom VMs: %d\n", len(status.ClusterInfo.CustomChains))
	fmt.Fprintf(&buf, "Backend Controller: Enabled\n")

	fmt.Fprintf(&buf, "%s Node information %s\n", nodeSeparator, nodeSeparator)

	for n, nodeInfo := range status.ClusterInfo.NodeInfos {
		fmt.Fprintf(&buf, "%s has ID %s and endpoint %s \n", n, nodeInfo.Id, nodeInfo.Uri)

		// Query node info
		version, vmVersions, err := getNodeVersion(nodeInfo.Uri)
		if err == nil {
			fmt.Fprintf(&buf, "  Version: %s\n", version)
			if len(vmVersions) > 0 {
				fmt.Fprintf(&buf, "  VM Versions: %v\n", vmVersions)
			}
		} else {
			// If failed to get version, debug log?
			// fmt.Fprintf(&buf, "  Version check failed: %v\n", err)
		}

		// Query C-Chain height
		height, err := getCChainHeight(nodeInfo.Uri)
		if err == nil {
			fmt.Fprintf(&buf, "  C-Chain Height: %s\n", height)
		} else {
			fmt.Fprintf(&buf, "  C-Chain Height: Unknown\n")
		}
	}

	if len(status.ClusterInfo.CustomChains) > 0 {
		fmt.Fprintf(&buf, "%s Custom VM information %s\n", nodeSeparator, nodeSeparator)
		for _, nodeInfo := range status.ClusterInfo.NodeInfos {
			for blockchainID := range status.ClusterInfo.CustomChains {
				fmt.Fprintf(&buf, "Endpoint at %s for blockchain %q: %s/ext/bc/%s/rpc\n", nodeInfo.Name, blockchainID, nodeInfo.GetUri(), blockchainID)
			}
		}
	}

	if verbose {
		fmt.Fprintf(&buf, "\nVerbose output:\n%s\n", status.String())
	}

	return buf.String(), nil
}

func getNodeVersion(uri string) (string, map[string]string, error) {
	// uri is http://ip:port
	url := fmt.Sprintf("%s/ext/info", uri)
	reqBody := []byte(`{"jsonrpc":"2.0", "id":1, "method":"info.getNodeVersion", "params":{}}`)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	// Short timeout for local info check
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}

	var r map[string]interface{}
	if err := json.Unmarshal(body, &r); err != nil {
		return "", nil, err
	}

	if result, ok := r["result"].(map[string]interface{}); ok {
		version, _ := result["version"].(string)

		vmVersions := make(map[string]string)
		if vms, ok := result["vmVersions"].(map[string]interface{}); ok {
			for k, v := range vms {
				if s, ok := v.(string); ok {
					vmVersions[k] = s
				}
			}
		}
		return version, vmVersions, nil
	}
	return "", nil, fmt.Errorf("invalid response")
}

func getCChainHeight(uri string) (string, error) {
	// uri is http://ip:port
	url := fmt.Sprintf("%s/ext/bc/C/rpc", uri)
	reqBody := []byte(`{"jsonrpc":"2.0", "id":1, "method":"eth_blockNumber", "params":[]}`)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	// Short timeout for local check
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var r map[string]interface{}
	if err := json.Unmarshal(body, &r); err != nil {
		return "", err
	}

	if result, ok := r["result"].(string); ok {
		// Result is hex string (e.g., "0x1b4") - convert to decimal
		if strings.HasPrefix(result, "0x") {
			// Convert hex to decimal
			decimalValue, err := strconv.ParseUint(result[2:], 16, 64)
			if err == nil {
				return fmt.Sprintf("%d", decimalValue), nil
			}
		}
		return result, nil // Fallback to original if conversion fails
	}
	return "", fmt.Errorf("invalid response")
}

// getTerminalWidth returns the current terminal width, or a default if unable to determine
func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80 // Default width
	}
	return width
}

// minInt returns the minimum of two integers.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
