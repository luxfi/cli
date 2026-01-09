// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package networkcmd

import (
	"context"
	"fmt"
	"os"

	"github.com/luxfi/cli/pkg/status"
	"github.com/spf13/cobra"
)

var (
	statusFormat  string
	statusCompact bool
	statusOutput  string
	statusVerbose bool
)

// NewStatusCmd returns the improved status command.
func NewStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "status",
		Aliases: []string{"stat"},
		Short:   "Show network status with improved formatting",
		Long: `The improved network status command shows detailed information about running networks.

OVERVIEW:

  Displays network health, validator nodes, endpoints, and custom chains.
  Uses clean, structured output suitable for scripting and human reading.

FORMAT OPTIONS:

  --format full     Show full detailed status (default)
  --format summary  Show only network summary
  --format chains   Show only chain status
  --format nodes    Show only node status
  --compact         Use compact output format

EXAMPLES:

  # Show full status
  lux network status-new

  # Show only chain status
  lux network status-new --format chains

  # Show compact summary
  lux network status-new --compact

OUTPUT FORMAT:

  status  mainnet  up   grpc=8369  nodes=5  vms=1  controller=on
  status  testnet  up   grpc=8368  nodes=5  vms=1  controller=on
  
  mainnet nodes
  node   http                         version       peers  uptime     ok
  1      http://127.0.0.1:9630        luxd/1.22.75   12     01:22:10   yes
  
  mainnet chains (heights)
  chain  kind  height     block_time           rpc_ok  latency
  p      p     12345      2026-01-06 14:27:03   yes     18ms
  c      evm   218        2026-01-06 14:27:01   yes     16ms`,

		RunE:         runStatusNew,
		Args:         cobra.ExactArgs(0),
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&statusFormat, "format", "full", "output format (full, summary, chains, nodes)")
	cmd.Flags().BoolVar(&statusCompact, "compact", false, "use compact output format")
	cmd.Flags().StringVarP(&statusOutput, "output", "o", "text", "output format (text, json, yaml, wide)")
	cmd.Flags().BoolVar(&statusVerbose, "verbose", false, "show verbose progress information")

	return cmd
}

func runStatusNew(cmd *cobra.Command, args []string) error {
	// Create progress tracker
	progress := status.NewProgressTracker(os.Stderr)

	// Create status service with progress callback if verbose
	var service *status.StatusService
	if statusVerbose {
		service = status.NewStatusServiceWithProgress(func(step string, current int, total int, message string) {
			if step == "networks" {
				progress.UpdateStep(fmt.Sprintf("Checking networks: %d/%d - %s", current, total, message))
			} else if step == "complete" {
				progress.CompleteStep("Network status checks")
			}
		})
	} else {
		service = status.NewStatusService()
	}

	// Start progress if verbose
	if statusVerbose {
		progress.StartStep("Checking network status")
	}

	// Get status
	ctx := context.Background()
	result, err := service.GetStatus(ctx)
	if err != nil {
		if statusVerbose {
			progress.FailStep("Network status check", err)
		}
		return fmt.Errorf("failed to get status: %w", err)
	}

	// Create formatter
	formatter := status.NewStatusFormatter(os.Stdout)

	// Format based on requested output format
	switch statusOutput {
	case "json":
		return formatter.FormatJSON(result)
	case "yaml":
		return formatter.FormatYAML(result)
	case "wide":
		// Wide format - currently maps to full network status
		formatter.FormatNetworkStatus(result)
	case "text":
		fallthrough
	default:
		// Format based on requested display format
		switch statusFormat {
		case "summary":
			formatter.FormatStatusSummary(result)
		case "chains":
			formatter.FormatChainStatus(result)
		case "nodes":
			formatter.FormatNodeStatus(result)
		case "full":
			fallthrough
		default:
			if statusCompact {
				// Compact full format
				formatter.FormatStatusSummary(result)
				formatter.FormatChainStatus(result)
				formatter.FormatNodeStatus(result)
			} else {
				// Full detailed format
				formatter.FormatNetworkStatus(result)
			}
		}
	}

	return nil
}
