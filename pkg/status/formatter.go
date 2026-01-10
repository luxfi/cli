// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package status

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// StatusFormatter handles formatting of status output
type StatusFormatter struct {
	writer io.Writer
}

// NewStatusFormatter creates a new formatter
func NewStatusFormatter(writer io.Writer) *StatusFormatter {
	return &StatusFormatter{
		writer: writer,
	}
}

// getChainTypeName returns the human-readable name for a chain type
func getChainTypeName(chainAlias string) string {
	switch chainAlias {
	case "p":
		return "platform"
	case "x":
		return "exchange"
	case "c":
		return "coreth" // C-Chain is Coreth (EVM-compatible)
	case "a":
		return "ai"
	case "b":
		return "bridge"
	case "d":
		return "dex"
	case "g":
		return "graph"
	case "k":
		return "kms"
	case "q":
		return "quantum"
	case "t":
		return "threshold"
	case "z":
		return "zk"
	case "zoo":
		return "zoo" // Zoo L2
	case "hanzo":
		return "hanzo" // Hanzo L2
	case "spc":
		return "spc" // SPC L2
	default:
		return "custom"
	}
}

// FormatNetworkStatus formats network status in the requested clean format
func (f *StatusFormatter) FormatNetworkStatus(result *StatusResult) {
	// Format network summary
	for _, network := range result.Networks {
		status := "stopped"
		if network.Metadata.Status == "up" {
			status = "up"
		}

		fmt.Fprintf(f.writer, "status  %-8s %-8s  grpc=%d  nodes=%d  vms=%d  controller=%s\n",
			network.Name,
			status,
			network.Metadata.GRPCPort,
			network.Metadata.NodesCount,
			network.Metadata.VMsCount,
			network.Metadata.Controller)
	}

	// Format node details for each network
	for _, network := range result.Networks {
		if len(network.Nodes) > 0 {
			fmt.Fprintf(f.writer, "\n%s nodes\n", network.Name)
			fmt.Fprintf(f.writer, "node            node_id                                  http                         version       peers  uptime     gpu        ok\n")

			for _, node := range network.Nodes {
				okStr := "no"
				if node.OK {
					okStr = "✓ yes"
				}

				nodeID := "-"
				if node.NodeID != "" {
					nodeID = node.NodeID
				}

				// Create a more descriptive node identifier
				nodeIdentifier := fmt.Sprintf("%s-%s-%s", network.Name, node.ID, (func() string {
					if len(nodeID) > 8 {
						return nodeID[:8]
					}
					return nodeID
				}()))

				version := strings.TrimPrefix(node.Version, "luxd/")

				// GPU status
				gpuStatus := "-"
				if node.GPUAccelerated {
					if node.GPUDevice != "" {
						gpuStatus = node.GPUDevice
						if len(gpuStatus) > 10 {
							gpuStatus = gpuStatus[:10]
						}
					} else {
						gpuStatus = "yes"
					}
				}

				fmt.Fprintf(f.writer, "%-12s  %-30s  %-32s %-12s  %-5d  %-8s  %-10s %s\n",
					nodeIdentifier,
					nodeID,
					node.HTTPURL,
					version,
					node.PeerCount,
					node.Uptime,
					gpuStatus,
					okStr)
			}
		}
	}

	// Format node addresses for each network
	for _, network := range result.Networks {
		if len(network.Nodes) > 0 {
			fmt.Fprintf(f.writer, "\n%s node addresses\n", network.Name)
			fmt.Fprintf(f.writer, "node   x-chain address                          c-chain address\n")

			for _, node := range network.Nodes {
				xAddress := "-"
				if node.XChainAddress != "" {
					xAddress = node.XChainAddress
				}

				cAddress := "-"
				if node.CChainAddress != "" {
					cAddress = node.CChainAddress
				}

				// Format addresses with P/X-lux prefix if they look like Lux addresses
				displayX := xAddress
				if strings.HasPrefix(xAddress, "X-lux") {
					displayX = strings.Replace(xAddress, "X-lux", "X-lux", 1) // Ensure consistency
				} else if strings.HasPrefix(xAddress, "lux") {
					displayX = "X-" + xAddress
				}

				fmt.Fprintf(f.writer, "%-5s  %-40s  %s\n",
					node.ID,
					displayX,
					cAddress)
			}
		}
	}

	// Format chain status for each network
	for _, network := range result.Networks {
		if len(network.Chains) > 0 {
			fmt.Fprintf(f.writer, "\n%s chains (heights)\n", network.Name)
			fmt.Fprintf(f.writer, "chain  type       height     block_time           rpc_ok  latency  chain_id  rpc_endpoint\n")

			for _, chain := range network.Chains {
				rpcOK := "no"
				if chain.RPC_OK {
					rpcOK = "yes"
				}

				blockTime := "-"
				if chain.BlockTime != nil {
					blockTime = chain.BlockTime.Format("2006-01-02 15:04:05")
				}

				chainType := getChainTypeName(chain.Alias)
				chainID := "-"
				if chain.ChainID != "" {
					chainID = chain.ChainID
				}

				// Get RPC endpoint for this chain from actual network nodes
				rpcEndpoint := "-"
				baseURL := "http://127.0.0.1:9650"
				if len(network.Nodes) > 0 {
					baseURL = network.Nodes[0].HTTPURL
				}

				// P-chain and X-chain don't use /rpc suffix, EVM chains do
				switch chain.Alias {
				case "p":
					rpcEndpoint = fmt.Sprintf("%s/ext/bc/P", baseURL)
				case "x":
					rpcEndpoint = fmt.Sprintf("%s/ext/bc/X", baseURL)
				case "c", "a", "b", "d", "g", "k", "q", "t", "z":
					rpcEndpoint = fmt.Sprintf("%s/ext/bc/%s/rpc", baseURL, strings.ToUpper(chain.Alias))
				default:
					rpcEndpoint = fmt.Sprintf("%s/ext/bc/%s/rpc", baseURL, chain.Alias)
				}

				fmt.Fprintf(f.writer, "%-5s  %-10s  %-10d %-20s  %-6s  %dms      %-8s  %s\n",
					chain.Alias,
					chainType,
					chain.Height,
					blockTime,
					rpcOK,
					chain.LatencyMS,
					chainID,
					rpcEndpoint)
			}
		}
	}

	// Format endpoints by chain for each network
	for _, network := range result.Networks {
		if len(network.Endpoints) > 0 {
			fmt.Fprintf(f.writer, "\n%s endpoints (by chain)\n", network.Name)
			fmt.Fprintf(f.writer, "chain  endpoints\n")

			for _, endpoint := range network.Endpoints {
				fmt.Fprintf(f.writer, "%-5s  %s (x%d)\n",
					endpoint.ChainAlias,
					endpoint.URL,
					1) // Placeholder for count
			}
		}
	}

	// Format L1 EVM chains (Zoo, Hanzo, SPC)
	if len(result.TrackedEVMs) > 0 {
		fmt.Fprintf(f.writer, "\nl1 chains (zoo, hanzo, spc)\n")
		fmt.Fprintf(f.writer, "chain    network   chain_id  height     rpc_ok  client_version               rpc_endpoint\n")

		for _, evm := range result.TrackedEVMs {
			rpcOK := "no"
			rpcEndpoint := "-"
			if len(evm.Endpoints) > 0 {
				if evm.Endpoints[0].OK {
					rpcOK = "yes"
				}
				rpcEndpoint = evm.Endpoints[0].URL
			}

			chainID := "-"
			if evm.ChainID > 0 {
				chainID = fmt.Sprintf("%d", evm.ChainID)
			}

			clientVersion := "-"
			if evm.ClientVersion != "" {
				if len(evm.ClientVersion) > 28 {
					clientVersion = evm.ClientVersion[:28]
				} else {
					clientVersion = evm.ClientVersion
				}
			}

			fmt.Fprintf(f.writer, "%-8s %-9s %-9s  %-10d %-6s  %-28s  %s\n",
				evm.Name,
				evm.Network,
				chainID,
				evm.Height,
				rpcOK,
				clientVersion,
				rpcEndpoint)
		}
	}

	// Format validator accounts with balances for each network
	for _, network := range result.Networks {
		if len(network.Validators) > 0 {
			fmt.Fprintf(f.writer, "\n%s validators\n", network.Name)
			fmt.Fprintf(f.writer, "#  node_id                                    p-chain                                  x-chain                                  c-chain                                    active\n")

			for _, v := range network.Validators {
				activeStr := " "
				if v.IsActive {
					activeStr = "*"
				}

				// Truncate node_id for display
				nodeID := v.NodeID
				if len(nodeID) > 40 {
					nodeID = nodeID[:40]
				}

				// Truncate addresses for display
				pAddr := v.PChainAddress
				if len(pAddr) > 38 {
					pAddr = pAddr[:38]
				}
				xAddr := v.XChainAddress
				if len(xAddr) > 38 {
					xAddr = xAddr[:38]
				}

				fmt.Fprintf(f.writer, "%-2d %-42s %-40s %-40s %-42s %s\n",
					v.Index,
					nodeID,
					pAddr,
					xAddr,
					v.CChainAddress,
					activeStr)
			}
		}

		// Show validator balances
		if len(network.Validators) > 0 {
			fmt.Fprintf(f.writer, "\n%s validator balances\n", network.Name)
			fmt.Fprintf(f.writer, "#  p-chain balance      x-chain balance      c-chain balance\n")

			for _, v := range network.Validators {
				pBalance := FormatNLUXToLUX(v.PChainBalance)
				xBalance := FormatNLUXToLUX(v.XChainBalance)
				cBalance := v.CChainBalanceLUX
				if cBalance == "" {
					cBalance = "0 LUX"
				}

				fmt.Fprintf(f.writer, "%-2d %-20s %-20s %s\n",
					v.Index,
					pBalance,
					xBalance,
					cBalance)
			}
		}

		// Show active account summary
		if network.ActiveAccount != nil {
			fmt.Fprintf(f.writer, "\n%s active account\n", network.Name)
			fmt.Fprintf(f.writer, "  validator #%d\n", network.ActiveAccount.Index)
			fmt.Fprintf(f.writer, "  P-Chain: %s\n", network.ActiveAccount.PChainAddress)
			fmt.Fprintf(f.writer, "  X-Chain: %s\n", network.ActiveAccount.XChainAddress)
			fmt.Fprintf(f.writer, "  C-Chain: %s\n", network.ActiveAccount.CChainAddress)
		}
	}
}

// FormatStatusSummary provides a compact summary format
func (f *StatusFormatter) FormatStatusSummary(result *StatusResult) {
	for _, network := range result.Networks {
		status := "stopped"
		if network.Metadata.Status == "up" {
			status = "up"
		}

		fmt.Fprintf(f.writer, "status  %-8s %-8s  grpc=%d  nodes=%d  vms=%d  controller=%s\n",
			network.Name,
			status,
			network.Metadata.GRPCPort,
			network.Metadata.NodesCount,
			network.Metadata.VMsCount,
			network.Metadata.Controller)
	}
}

// FormatChainStatus provides a compact chain status format
func (f *StatusFormatter) FormatChainStatus(result *StatusResult) {
	for _, network := range result.Networks {
		if len(network.Chains) > 0 {
			fmt.Fprintf(f.writer, "\n%s chains\n", network.Name)
			fmt.Fprintf(f.writer, "chain  kind  height  rpc_ok  latency\n")

			for _, chain := range network.Chains {
				rpcOK := "no"
				if chain.RPC_OK {
					rpcOK = "yes"
				}

				fmt.Fprintf(f.writer, "%-5s  %-4s  %-6d  %-6s  %dms\n",
					chain.Alias,
					chain.Kind,
					chain.Height,
					rpcOK,
					chain.LatencyMS)
			}
		}
	}
}

// FormatNodeStatus provides a compact node status format
func (f *StatusFormatter) FormatNodeStatus(result *StatusResult) {
	for _, network := range result.Networks {
		if len(network.Nodes) > 0 {
			fmt.Fprintf(f.writer, "\n%s nodes\n", network.Name)
			fmt.Fprintf(f.writer, "node  http            version  peers  ok\n")

			for _, node := range network.Nodes {
				okStr := "no"
				if node.OK {
					okStr = "yes"
				}

				fmt.Fprintf(f.writer, "%-4s  %-15s  %-7s  %-5d  %s\n",
					node.ID,
					node.HTTPURL,
					strings.TrimPrefix(node.Version, "luxd/"),
					node.PeerCount,
					okStr)
			}
		}
	}
}

// FormatJSON outputs the status as JSON
func (f *StatusFormatter) FormatJSON(result *StatusResult) error {
	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// FormatYAML outputs the status as YAML
func (f *StatusFormatter) FormatYAML(result *StatusResult) error {
	encoder := yaml.NewEncoder(f.writer)
	encoder.SetIndent(2)
	return encoder.Encode(result)
}
