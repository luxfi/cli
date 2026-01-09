// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package networkcmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/luxfi/cli/pkg/status"
	"github.com/spf13/cobra"
)

var (
	monitorInterval int
	monitorFormat   string
	monitorCompact  bool
	monitorOutput   string
)

// NewMonitorCmd returns the monitor command
func NewMonitorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "monitor",
		Short: "Monitor network status in real-time",
		Long: `The monitor command shows real-time network status updates.

OVERVIEW:

  Continuously monitors network health, validator nodes, endpoints, and custom chains.
  Updates display every second by default, showing live statistics.

OPTIONS:

  --interval, -i   Update interval in seconds (default: 1)
  --format         Output format (full, summary, chains, nodes)
  --compact        Use compact output format

EXAMPLES:

  # Monitor with default 1-second updates
  lux network monitor

  # Monitor with 5-second updates
  lux network monitor --interval 5

  # Monitor with compact format
  lux network monitor --compact

  # Monitor only chain status
  lux network monitor --format chains`,

		RunE:         runMonitor,
		Args:         cobra.ExactArgs(0),
		SilenceUsage: true,
	}

	cmd.Flags().IntVarP(&monitorInterval, "interval", "i", 1, "update interval in seconds")
	cmd.Flags().StringVar(&monitorFormat, "format", "full", "output format (full, summary, chains, nodes)")
	cmd.Flags().BoolVar(&monitorCompact, "compact", false, "use compact output format")
	cmd.Flags().StringVarP(&monitorOutput, "output", "o", "text", "output format (text, json)")

	return cmd
}

func runMonitor(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Channel to handle OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create status service
	service := status.NewStatusService()

	// Create formatter
	formatter := status.NewStatusFormatter(os.Stdout)

	firstRun := true
	ticker := time.NewTicker(time.Duration(monitorInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Clear screen for subsequent updates (except first)
			if !firstRun {
				fmt.Print("\033[2J\033[H") // ANSI escape codes to clear screen and move cursor to top-left
			}

			// Print timestamp
			fmt.Printf("LUX Network Monitor (refresh: %ds) - %s\n", monitorInterval, time.Now().Format("2006-01-02 15:04:05"))
			fmt.Println("============================================================")

			// Get status
			result, err := service.GetStatus(ctx)
			if err != nil {
				return fmt.Errorf("failed to get status: %w", err)
			}

			// Format based on requested output format
			switch monitorOutput {
			case "json":
				if err := formatter.FormatJSON(result); err != nil {
					return fmt.Errorf("failed to format JSON: %w", err)
				}
			case "text":
				fallthrough
			default:
				// Format based on requested display format
				switch monitorFormat {
				case "summary":
					formatter.FormatStatusSummary(result)
				case "chains":
					formatter.FormatChainStatus(result)
				case "nodes":
					formatter.FormatNodeStatus(result)
				case "full":
					fallthrough
				default:
					if monitorCompact {
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

			firstRun = false

		case <-sigChan:
			fmt.Println("\nMonitor stopped by user.")
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
