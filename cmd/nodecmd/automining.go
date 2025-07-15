// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package nodecmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

type autominingFlags struct {
	rpcURL     string
	account    string
	privateKey string
	threads    int
	monitor    bool
}

func newAutominingCmd() *cobra.Command {
	flags := &autominingFlags{}
	
	cmd := &cobra.Command{
		Use:   "automine",
		Short: "Control automining on a running node",
		Long: `Control automining functionality on a running Lux node.
This command allows you to start, stop, and monitor automining.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	
	cmd.AddCommand(newAutomineStartCmd(flags))
	cmd.AddCommand(newAutomineStopCmd(flags))
	cmd.AddCommand(newAutomineStatusCmd(flags))
	
	return cmd
}

func newAutomineStartCmd(flags *autominingFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start automining",
		Long:  `Start automining on a running Lux node`,
		Example: `  # Start automining with default account
  lux node automine start

  # Start automining with specific account
  lux node automine start --account 0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC

  # Start automining and monitor blocks
  lux node automine start --monitor`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return startAutomining(flags)
		},
	}
	
	cmd.Flags().StringVar(&flags.rpcURL, "rpc-url", "http://localhost:9650/ext/bc/C/rpc", "RPC URL of the node")
	cmd.Flags().StringVar(&flags.account, "account", "0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC", "Mining account address")
	cmd.Flags().StringVar(&flags.privateKey, "private-key", "56289e99c94b6912bfc12adc093c9b51124f0dc54ac7a766b2bc5ccf558d8027", "Private key for the mining account")
	cmd.Flags().IntVar(&flags.threads, "threads", 1, "Number of mining threads")
	cmd.Flags().BoolVar(&flags.monitor, "monitor", false, "Monitor block production")
	
	return cmd
}

func newAutomineStopCmd(flags *autominingFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop automining",
		Long:  `Stop automining on a running Lux node`,
		Example: `  # Stop automining
  lux node automine stop`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return stopAutomining(flags)
		},
	}
	
	cmd.Flags().StringVar(&flags.rpcURL, "rpc-url", "http://localhost:9650/ext/bc/C/rpc", "RPC URL of the node")
	
	return cmd
}

func newAutomineStatusCmd(flags *autominingFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check automining status",
		Long:  `Check the current automining status on a running Lux node`,
		Example: `  # Check automining status
  lux node automine status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return checkAutominingStatus(flags)
		},
	}
	
	cmd.Flags().StringVar(&flags.rpcURL, "rpc-url", "http://localhost:9650/ext/bc/C/rpc", "RPC URL of the node")
	
	return cmd
}

func startAutomining(flags *autominingFlags) error {
	ux.Logger.PrintToUser("Starting automining...")
	
	// First, import the private key
	if err := importPrivateKey(flags.rpcURL, flags.privateKey); err != nil {
		return fmt.Errorf("failed to import private key: %w", err)
	}
	
	// Set the coinbase
	if err := setCoinbase(flags.rpcURL, flags.account); err != nil {
		return fmt.Errorf("failed to set coinbase: %w", err)
	}
	
	// Start mining
	result, err := rpcCall(flags.rpcURL, "miner_start", []interface{}{flags.threads})
	if err != nil {
		return fmt.Errorf("failed to start mining: %w", err)
	}
	
	ux.Logger.PrintToUser("Automining started: %v", result)
	
	if flags.monitor {
		return monitorBlocks(flags.rpcURL)
	}
	
	return nil
}

func stopAutomining(flags *autominingFlags) error {
	ux.Logger.PrintToUser("Stopping automining...")
	
	result, err := rpcCall(flags.rpcURL, "miner_stop", []interface{}{})
	if err != nil {
		return fmt.Errorf("failed to stop mining: %w", err)
	}
	
	ux.Logger.PrintToUser("Automining stopped: %v", result)
	return nil
}

func checkAutominingStatus(flags *autominingFlags) error {
	// Check if mining
	mining, err := rpcCall(flags.rpcURL, "eth_mining", []interface{}{})
	if err != nil {
		return fmt.Errorf("failed to check mining status: %w", err)
	}
	
	ux.Logger.PrintToUser("Mining: %v", mining)
	
	// Get current block
	blockNum, err := rpcCall(flags.rpcURL, "eth_blockNumber", []interface{}{})
	if err != nil {
		return fmt.Errorf("failed to get block number: %w", err)
	}
	
	ux.Logger.PrintToUser("Current block: %v", blockNum)
	
	// Get coinbase
	coinbase, err := rpcCall(flags.rpcURL, "eth_coinbase", []interface{}{})
	if err != nil {
		ux.Logger.PrintToUser("Coinbase: not set")
	} else {
		ux.Logger.PrintToUser("Coinbase: %v", coinbase)
	}
	
	// Get hashrate
	hashrate, err := rpcCall(flags.rpcURL, "eth_hashrate", []interface{}{})
	if err == nil {
		ux.Logger.PrintToUser("Hashrate: %v", hashrate)
	}
	
	return nil
}

func importPrivateKey(rpcURL, privateKey string) error {
	_, err := rpcCall(rpcURL, "personal_importRawKey", []interface{}{privateKey, ""})
	return err
}

func setCoinbase(rpcURL, account string) error {
	_, err := rpcCall(rpcURL, "miner_setEtherbase", []interface{}{account})
	return err
}

func monitorBlocks(rpcURL string) error {
	ux.Logger.PrintToUser("Monitoring blocks (press Ctrl+C to stop)...")
	
	prevBlock := uint64(0)
	for {
		blockHex, err := rpcCall(rpcURL, "eth_blockNumber", []interface{}{})
		if err != nil {
			ux.Logger.PrintToUser("Error getting block number: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}
		
		var blockNum uint64
		if hexStr, ok := blockHex.(string); ok {
			fmt.Sscanf(hexStr, "0x%x", &blockNum)
		}
		
		if blockNum > prevBlock {
			ux.Logger.PrintToUser("[%s] New block mined: #%d", time.Now().Format("15:04:05"), blockNum)
			prevBlock = blockNum
		}
		
		time.Sleep(1 * time.Second)
	}
}

func rpcCall(url, method string, params interface{}) (interface{}, error) {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      1,
	}
	
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	
	if errField, ok := result["error"]; ok {
		return nil, fmt.Errorf("RPC error: %v", errField)
	}
	
	return result["result"], nil
}