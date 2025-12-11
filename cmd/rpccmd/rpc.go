// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rpccmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

// NewCmd returns the RPC command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rpc",
		Short: "Make RPC calls to Lux node",
		Long: `Make JSON-RPC calls to a Lux node.

Examples:
  # Get P-Chain height
  lux rpc call --method platform.getHeight --endpoint http://localhost:9630/ext/bc/P

  # Get blockchains with params
  lux rpc call --method platform.getBlockchains --params '{}' --endpoint http://localhost:9630/ext/bc/P

  # Create blockchain
  lux rpc call --method platform.createBlockchain \
    --params '{"vmID":"...", "name":"mychain", "genesis":"..."}' \
    --endpoint http://localhost:9630/ext/bc/P
`,
		RunE: nil,
	}

	cmd.AddCommand(newCallCmd())
	return cmd
}

func newCallCmd() *cobra.Command {
	var (
		method   string
		params   string
		endpoint string
		timeout  int
	)

	cmd := &cobra.Command{
		Use:   "call",
		Short: "Make a JSON-RPC call",
		Long:  "Make a JSON-RPC call to the specified endpoint with the given method and parameters",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse params if provided
			var paramsObj interface{}
			if params != "" {
				if err := json.Unmarshal([]byte(params), &paramsObj); err != nil {
					return fmt.Errorf("failed to parse params: %w", err)
				}
			}

			// Create RPC request
			request := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"method":  method,
			}
			if paramsObj != nil {
				request["params"] = paramsObj
			} else {
				request["params"] = map[string]interface{}{}
			}

			// Marshal request
			requestBytes, err := json.Marshal(request)
			if err != nil {
				return fmt.Errorf("failed to marshal request: %w", err)
			}

			// Make HTTP request
			client := &http.Client{
				Timeout: time.Duration(timeout) * time.Second,
			}

			resp, err := client.Post(
				endpoint,
				"application/json",
				bytes.NewReader(requestBytes),
			)
			if err != nil {
				return fmt.Errorf("failed to make request: %w", err)
			}
			defer resp.Body.Close()

			// Read response
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read response: %w", err)
			}

			// Pretty print JSON response
			var result interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				// If not valid JSON, just print raw
				fmt.Println(string(body))
				return nil
			}

			prettyJSON, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				fmt.Println(string(body))
				return nil
			}

			fmt.Println(string(prettyJSON))
			return nil
		},
	}

	cmd.Flags().StringVar(&method, "method", "", "RPC method to call (required)")
	cmd.Flags().StringVar(&params, "params", "", "JSON params object (optional)")
	cmd.Flags().StringVar(&endpoint, "endpoint", "http://localhost:9630/ext/bc/P", "RPC endpoint URL")
	cmd.Flags().IntVar(&timeout, "timeout", 30, "Request timeout in seconds")

	_ = cmd.MarkFlagRequired("method")

	return cmd
}
