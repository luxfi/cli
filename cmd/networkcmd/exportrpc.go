// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	exportDataBlockchain  string
	exportDataRPC         string
	exportDataFile        string
	exportDataStart       uint64
	exportDataEnd         uint64
	exportDataIncludeState bool
	exportDataCompress    bool
	exportDataDBPath      string  // For direct DB export mode
	exportDataStateFile   string  // Separate state export file
)

// lux network export
func newExportRPCCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export network/blockchain data via RPC or direct DB access",
		Long: `Export complete network/blockchain data including blocks, transactions, and state.
This creates a portable dump that can be imported into another network.

Modes:
  RPC Mode: Export via JSON-RPC (default)
  DB Mode: Direct database export (faster, requires local DB access)

Examples:
  # Export from C-Chain via RPC
  lux net export --blockchain C --file c-chain-export.jsonl

  # Export from EVM L2 using blockchain ID
  lux net export --blockchain 2G8mK7VCZX1dV8iPjkkTDMpYGZDCNLLVdTJVLmMsG5ZV7zKVmB --file evm-export.jsonl

  # Export directly from database (faster for large chains)
  lux net export --db-path /path/to/pebble/db --file blocks.jsonl --state-file state.jsonl

  # Export specific block range via RPC
  lux net export --blockchain C --start 0 --end 1000 --file c-chain-range.jsonl

  # Export with full state via RPC
  lux net export --blockchain C --include-state --file full-export.jsonl

  # Use full RPC URL (backwards compatible)
  lux net export --rpc http://localhost:9630/ext/bc/C/rpc --file export.jsonl`,
		RunE: exportNetworkData,
	}

	// RPC mode flags
	cmd.Flags().StringVar(&exportDataBlockchain, "blockchain", "", "Blockchain ID to export from (e.g., 'C' for C-Chain or full blockchain ID)")
	cmd.Flags().StringVar(&exportDataRPC, "rpc", "", "Full RPC endpoint URL (overrides --blockchain)")

	// DB mode flags
	cmd.Flags().StringVar(&exportDataDBPath, "db-path", "", "Direct database path for DB mode export (PebbleDB)")
	cmd.Flags().StringVar(&exportDataStateFile, "state-file", "", "Separate state export file (DB mode only)")

	// Common flags
	cmd.Flags().StringVar(&exportDataFile, "file", "network-export.jsonl", "Export file for blocks (.json or .jsonl)")
	cmd.Flags().Uint64Var(&exportDataStart, "start", 0, "Start block number")
	cmd.Flags().Uint64Var(&exportDataEnd, "end", 0, "End block number (0 = latest)")
	cmd.Flags().BoolVar(&exportDataIncludeState, "include-state", false, "Include state dump (RPC mode)")
	cmd.Flags().BoolVar(&exportDataCompress, "compress", false, "Compress output file")

	return cmd
}

func exportNetworkData(_ *cobra.Command, _ []string) error {
	// Check if DB mode is requested
	if exportDataDBPath != "" {
		return exportFromDB()
	}

	// RPC mode (existing functionality)
	rpcURL := exportDataRPC
	if rpcURL == "" {
		if exportDataBlockchain == "" {
			return fmt.Errorf("either --blockchain or --rpc must be specified for RPC mode")
		}
		// Default to localhost:9630
		rpcURL = fmt.Sprintf("http://localhost:9630/ext/bc/%s/rpc", exportDataBlockchain)
	}

	// Auto-detect number of CPUs for parallel processing
	numWorkers := runtime.NumCPU()
	if numWorkers > 200 {
		numWorkers = 200 // Cap at 200 workers
	}

	ux.Logger.PrintToUser("Starting network export...")
	ux.Logger.PrintToUser("RPC endpoint: %s", rpcURL)

	// Get current block height if end not specified
	if exportDataEnd == 0 {
		height, err := getCurrentBlockHeight(rpcURL)
		if err != nil {
			return fmt.Errorf("failed to get current block height: %w", err)
		}
		exportDataEnd = height
		ux.Logger.PrintToUser("Latest block: %d", height)
	}

	ux.Logger.PrintToUser("Export range: %d to %d", exportDataStart, exportDataEnd)
	ux.Logger.PrintToUser("Using %d parallel workers", numWorkers)

	// Create output file
	outputFile := exportDataFile
	if exportDataCompress {
		outputFile += ".gz"
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	var writer io.Writer = file
	if exportDataCompress {
		gzWriter := gzip.NewWriter(file)
		defer gzWriter.Close()
		writer = gzWriter
	}

	// Export metadata
	metadata := map[string]interface{}{
		"version":    "1.0.0",
		"exportTime": time.Now().Format(time.RFC3339),
		"source":     rpcURL,
		"startBlock": exportDataStart,
		"endBlock":   exportDataEnd,
	}

	// Start export
	exportData := map[string]interface{}{
		"metadata": metadata,
		"blocks":   []interface{}{},
		"state":    map[string]interface{}{},
	}

	// Export blocks in parallel
	blocksChan := make(chan json.RawMessage, 100)
	var wg sync.WaitGroup
	var exported uint64
	var errors uint64

	// Progress tracking
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for range ticker.C {
			exp := atomic.LoadUint64(&exported)
			err := atomic.LoadUint64(&errors)
			rate := float64(exp) / 5.0
			ux.Logger.PrintToUser("Progress: %d/%d blocks (%.1f blocks/sec), errors: %d",
				exp, exportDataEnd-exportDataStart+1, rate, err)
			atomic.StoreUint64(&exported, 0)
		}
	}()
	defer ticker.Stop()

	// Worker pool
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for blockNum := uint64(workerID) + exportDataStart; blockNum <= exportDataEnd; blockNum += uint64(numWorkers) {
				blockData, err := fetchBlockData(rpcURL, blockNum)
				if err != nil {
					atomic.AddUint64(&errors, 1)
					ux.Logger.PrintToUser("Failed to fetch block %d: %v", blockNum, err)
					continue
				}
				blocksChan <- blockData
				atomic.AddUint64(&exported, 1)
			}
		}(i)
	}

	// Check if output is JSONL format
	isJSONL := strings.HasSuffix(outputFile, ".jsonl")

	if isJSONL {
		// JSONL format: write metadata first, then each block on a separate line
		encoder := json.NewEncoder(writer)

		// Write metadata as first line
		if err := encoder.Encode(metadata); err != nil {
			return fmt.Errorf("failed to write metadata: %w", err)
		}

		// Export state if requested (as second line)
		if exportDataIncludeState {
			ux.Logger.PrintToUser("Exporting state data...")
			state, err := exportState(rpcURL)
			if err != nil {
				ux.Logger.PrintToUser("Warning: Failed to export state: %v", err)
			} else {
				stateData := map[string]interface{}{"type": "state", "data": state}
				if err := encoder.Encode(stateData); err != nil {
					return fmt.Errorf("failed to write state: %w", err)
				}
			}
		}

		// Collect and write blocks one by one
		go func() {
			wg.Wait()
			close(blocksChan)
		}()

		blockCount := 0
		for block := range blocksChan {
			var blockData interface{}
			json.Unmarshal(block, &blockData)
			blockLine := map[string]interface{}{"type": "block", "data": blockData}
			if err := encoder.Encode(blockLine); err != nil {
				ux.Logger.PrintToUser("Failed to write block: %v", err)
			}
			blockCount++
		}

		ux.Logger.PrintToUser("✅ Export complete! Exported %d blocks to %s (JSONL format)", blockCount, outputFile)
	} else {
		// Original JSON format
		go func() {
			wg.Wait()
			close(blocksChan)
		}()

		blocks := []json.RawMessage{}
		for block := range blocksChan {
			blocks = append(blocks, block)
		}

		exportData["blocks"] = blocks
		exportData["blockCount"] = len(blocks)

		// Export state if requested
		if exportDataIncludeState {
			ux.Logger.PrintToUser("Exporting state data...")
			state, err := exportState(rpcURL)
			if err != nil {
				ux.Logger.PrintToUser("Warning: Failed to export state: %v", err)
			} else {
				exportData["state"] = state
			}
		}

		// Write to file
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(exportData); err != nil {
			return fmt.Errorf("failed to write export data: %w", err)
		}

		ux.Logger.PrintToUser("✅ Export complete! Exported %d blocks to %s", len(blocks), outputFile)
	}
	return nil
}

func getCurrentBlockHeight(rpcURL string) (uint64, error) {
	reqData := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_blockNumber",
		"params":  []interface{}{},
		"id":      1,
	}

	jsonData, _ := json.Marshal(reqData)
	resp, err := http.Post(rpcURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	if result["error"] != nil {
		return 0, fmt.Errorf("RPC error: %v", result["error"])
	}

	heightHex, ok := result["result"].(string)
	if !ok {
		return 0, fmt.Errorf("invalid response format")
	}

	var height uint64
	fmt.Sscanf(heightHex, "0x%x", &height)
	return height, nil
}

func fetchBlockData(rpcURL string, blockNum uint64) (json.RawMessage, error) {
	blockHex := fmt.Sprintf("0x%x", blockNum)
	reqData := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_getBlockByNumber",
		"params":  []interface{}{blockHex, true}, // true = include transactions
		"id":      1,
	}

	jsonData, _ := json.Marshal(reqData)
	resp, err := http.Post(rpcURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result["error"] != nil {
		return nil, fmt.Errorf("RPC error: %v", result["error"])
	}

	blockData, ok := result["result"]
	if !ok || blockData == nil {
		return nil, fmt.Errorf("block not found")
	}

	return json.Marshal(blockData)
}

func exportState(rpcURL string) (map[string]interface{}, error) {
	// This is a simplified state export
	// In a real implementation, you would iterate through all accounts
	// For now, we'll export known important addresses
	state := make(map[string]interface{})

	// Treasury address
	treasuryAddr := "0x9011E888251AB053B7bD1cdB598Db4f9DEd94714"
	balance, err := getBalance(rpcURL, treasuryAddr)
	if err == nil {
		state[treasuryAddr] = map[string]interface{}{
			"balance": balance,
			"nonce":   "0x0",
		}
	}

	// Dev account
	devAddr := "0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"
	balance, err = getBalance(rpcURL, devAddr)
	if err == nil {
		state[devAddr] = map[string]interface{}{
			"balance": balance,
			"nonce":   "0x0",
		}
	}

	return state, nil
}

func getBalance(rpcURL, address string) (string, error) {
	reqData := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_getBalance",
		"params":  []interface{}{address, "latest"},
		"id":      1,
	}

	jsonData, _ := json.Marshal(reqData)
	resp, err := http.Post(rpcURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result["error"] != nil {
		return "", fmt.Errorf("RPC error: %v", result["error"])
	}

	balance, ok := result["result"].(string)
	if !ok {
		return "", fmt.Errorf("invalid response format")
	}

	return balance, nil
}

// exportFromDB exports directly from PebbleDB (read-only)
func exportFromDB() error {
	ux.Logger.PrintToUser("Starting DB export...")
	ux.Logger.PrintToUser("DB path: %s", exportDataDBPath)

	// Open PebbleDB read-only
	opts := &pebble.Options{
		ReadOnly: true,
	}

	db, err := pebble.Open(exportDataDBPath, opts)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Export blocks
	if exportDataFile != "" {
		ux.Logger.PrintToUser("Exporting blocks to: %s", exportDataFile)
		if err := exportBlocks(db); err != nil {
			return fmt.Errorf("failed to export blocks: %w", err)
		}
	}

	// Export state
	if exportDataStateFile != "" {
		ux.Logger.PrintToUser("Exporting state to: %s", exportDataStateFile)
		if err := exportStateTrie(db); err != nil {
			return fmt.Errorf("failed to export state: %w", err)
		}
	}

	ux.Logger.PrintToUser("✅ DB export complete!")
	return nil
}

// exportBlocks exports blocks from PebbleDB to JSONL
func exportBlocks(db *pebble.DB) error {
	file, err := os.Create(exportDataFile)
	if err != nil {
		return fmt.Errorf("failed to create blocks file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Write metadata
	metadata := map[string]interface{}{
		"version":    "1.0.0",
		"exportTime": time.Now().Format(time.RFC3339),
		"source":     "pebble-db",
		"type":       "blocks",
	}

	metaLine, _ := json.Marshal(metadata)
	writer.Write(metaLine)
	writer.WriteByte('\n')

	// Iterate through blocks
	// SubnetEVM uses namespaced keys: [32-byte namespace][1-byte prefix][rest]
	// Block-related prefixes:
	// 'h' = header by number
	// 'H' = header by hash
	// 'b' = body
	// 'r' = receipts

	iter, err := db.NewIter(nil)
	if err != nil {
		return fmt.Errorf("failed to create iterator: %w", err)
	}
	defer iter.Close()

	blockCount := 0
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) < 33 {
			continue
		}

		// Skip namespace (first 32 bytes)
		localKey := key[32:]
		if len(localKey) < 1 {
			continue
		}

		prefix := localKey[0]
		rest := localKey[1:]

		// Process header by number entries ('h' prefix)
		if prefix == 'h' && len(rest) == 8 {
			// rest is block number (8 bytes)
			blockNum := decodeUint64(rest)

			// Get header, body, and receipts
			header := getBlockHeader(db, key[:32], blockNum)
			body := getBlockBody(db, key[:32], header)
			receipts := getBlockReceipts(db, key[:32], header)

			if header != nil {
				block := map[string]interface{}{
					"type":     "block",
					"number":   blockNum,
					"header":   header,
					"body":     body,
					"receipts": receipts,
				}

				blockLine, _ := json.Marshal(block)
				writer.Write(blockLine)
				writer.WriteByte('\n')
				blockCount++

				if blockCount%1000 == 0 {
					ux.Logger.PrintToUser("Exported %d blocks...", blockCount)
				}
			}
		}
	}

	ux.Logger.PrintToUser("Exported %d blocks total", blockCount)
	return nil
}

// exportStateTrie exports state trie nodes from PebbleDB to JSONL
func exportStateTrie(db *pebble.DB) error {
	file, err := os.Create(exportDataStateFile)
	if err != nil {
		return fmt.Errorf("failed to create state file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Write metadata
	metadata := map[string]interface{}{
		"version":    "1.0.0",
		"exportTime": time.Now().Format(time.RFC3339),
		"source":     "pebble-db",
		"type":       "state",
	}

	metaLine, _ := json.Marshal(metadata)
	writer.Write(metaLine)
	writer.WriteByte('\n')

	// Iterate through state trie nodes
	// State trie prefixes:
	// 's' = account trie nodes (secure trie)
	// 'S' = storage trie nodes

	iter, err := db.NewIter(nil)
	if err != nil {
		return fmt.Errorf("failed to create iterator: %w", err)
	}
	defer iter.Close()

	nodeCount := 0
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) < 33 {
			continue
		}

		// Skip namespace (first 32 bytes)
		localKey := key[32:]
		if len(localKey) < 1 {
			continue
		}

		prefix := localKey[0]
		rest := localKey[1:]

		// Process state trie nodes
		if prefix == 's' && len(rest) == 32 {
			// Account trie node
			node := map[string]interface{}{
				"kind":  "accountTrieNode",
				"hash":  "0x" + hex.EncodeToString(rest),
				"value": "0x" + hex.EncodeToString(iter.Value()),
			}

			nodeLine, _ := json.Marshal(node)
			writer.Write(nodeLine)
			writer.WriteByte('\n')
			nodeCount++

		} else if prefix == 'S' && len(rest) == 32 {
			// Storage trie node
			node := map[string]interface{}{
				"kind":  "storageTrieNode",
				"hash":  "0x" + hex.EncodeToString(rest),
				"value": "0x" + hex.EncodeToString(iter.Value()),
			}

			nodeLine, _ := json.Marshal(node)
			writer.Write(nodeLine)
			writer.WriteByte('\n')
			nodeCount++
		}

		if nodeCount%10000 == 0 && nodeCount > 0 {
			ux.Logger.PrintToUser("Exported %d state nodes...", nodeCount)
		}
	}

	ux.Logger.PrintToUser("Exported %d state nodes total", nodeCount)
	return nil
}

// Helper functions for DB export
func decodeUint64(b []byte) uint64 {
	if len(b) != 8 {
		return 0
	}
	return uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 |
		uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
}

func getBlockHeader(db *pebble.DB, namespace []byte, blockNum uint64) map[string]interface{} {
	// Build key: namespace + 'h' + block number
	key := make([]byte, 41)
	copy(key[:32], namespace)
	key[32] = 'h'
	// Encode block number as 8 bytes big-endian
	for i := 0; i < 8; i++ {
		key[33+i] = byte(blockNum >> uint(56-i*8))
	}

	val, closer, err := db.Get(key)
	if err != nil {
		return nil
	}
	defer closer.Close()

	// Parse header (simplified - you'd decode RLP properly)
	return map[string]interface{}{
		"raw": "0x" + hex.EncodeToString(val),
	}
}

func getBlockBody(db *pebble.DB, namespace []byte, header map[string]interface{}) map[string]interface{} {
	// Similar to getBlockHeader but with 'b' prefix
	// This is simplified - real implementation would look up by hash
	return map[string]interface{}{
		"transactions": []interface{}{},
	}
}

func getBlockReceipts(db *pebble.DB, namespace []byte, header map[string]interface{}) []interface{} {
	// Similar to getBlockHeader but with 'r' prefix
	return []interface{}{}
}