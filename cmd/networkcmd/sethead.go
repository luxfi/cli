// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	setHeadChainID   uint64
	setHeadDBPath    string
	setHeadHeight    uint64
	setHeadHash      string
	setHeadVMDataDir string
	setHeadRPC       string
	setHeadAuto      bool
)

// lux network set-head
func newSetHeadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-head",
		Short: "Set VM metadata and chain head after import",
		Long: `Set VM metadata to establish chain head after importing blocks and state.
This command configures the VM to start from the imported tip instead of re-genesis.

The command will:
1. Set the canonical chain head in the database (LastBlock, LastHeader, LastFast)
2. Create/update VM metadata files (lastAccepted, lastAcceptedHeight, initialized)
3. Ensure the VM starts at the imported tip

Examples:
  # Set head manually with specific height and hash
  lux net set-head --chain-id 96369 --height 1082780 --hash 0x32dede...461f0

  # Auto-detect head from database
  lux net set-head --chain-id 96369 --db-path /path/to/badger/db --auto

  # Set head from RPC (queries current head)
  lux net set-head --chain-id 96369 --rpc http://localhost:9630/ext/bc/C/rpc --auto

  # Specify custom VM data directory
  lux net set-head --chain-id 96369 --height 1082780 --hash 0x32dede...461f0 \
    --vm-dir ~/.luxd/chainData/C`,
		RunE: setChainHead,
	}

	cmd.Flags().Uint64Var(&setHeadChainID, "chain-id", 96369, "Chain ID")
	cmd.Flags().StringVar(&setHeadDBPath, "db-path", "", "Database path (BadgerDB)")
	cmd.Flags().Uint64Var(&setHeadHeight, "height", 0, "Block height to set as head")
	cmd.Flags().StringVar(&setHeadHash, "hash", "", "Block hash to set as head")
	cmd.Flags().StringVar(&setHeadVMDataDir, "vm-dir", "", "VM data directory (default: ~/.luxd/chainData/C)")
	cmd.Flags().StringVar(&setHeadRPC, "rpc", "", "RPC endpoint for auto-detection")
	cmd.Flags().BoolVar(&setHeadAuto, "auto", false, "Auto-detect head from DB or RPC")

	return cmd
}

func setChainHead(_ *cobra.Command, _ []string) error {
	// Auto-detect head if requested
	if setHeadAuto {
		if setHeadRPC != "" {
			if err := detectHeadFromRPC(); err != nil {
				return fmt.Errorf("failed to detect head from RPC: %w", err)
			}
		} else if setHeadDBPath != "" {
			if err := detectHeadFromDB(); err != nil {
				return fmt.Errorf("failed to detect head from DB: %w", err)
			}
		} else {
			return fmt.Errorf("--rpc or --db-path required for auto-detection")
		}
	}

	// Validate inputs
	if setHeadHeight == 0 || setHeadHash == "" {
		return fmt.Errorf("--height and --hash required (or use --auto)")
	}

	// Clean up hash (remove 0x prefix if present)
	cleanHash := strings.TrimPrefix(setHeadHash, "0x")
	hashBytes, err := hex.DecodeString(cleanHash)
	if err != nil || len(hashBytes) != 32 {
		return fmt.Errorf("invalid hash format (must be 32 bytes hex)")
	}

	ux.Logger.PrintToUser("Setting chain head:")
	ux.Logger.PrintToUser("  Chain ID: %d", setHeadChainID)
	ux.Logger.PrintToUser("  Height: %d", setHeadHeight)
	ux.Logger.PrintToUser("  Hash: 0x%s", cleanHash)

	// Set database head if DB path provided
	if setHeadDBPath != "" {
		if err := setDatabaseHead(hashBytes); err != nil {
			return fmt.Errorf("failed to set database head: %w", err)
		}
	}

	// Set VM metadata
	if err := setVMMetadata(hashBytes); err != nil {
		return fmt.Errorf("failed to set VM metadata: %w", err)
	}

	ux.Logger.PrintToUser("✅ Chain head set successfully!")
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("The VM will now start from:")
	ux.Logger.PrintToUser("  Block: %d", setHeadHeight)
	ux.Logger.PrintToUser("  Hash: 0x%s", cleanHash)
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("You can now start luxd with:")
	ux.Logger.PrintToUser("  luxd --network-id=%d", setHeadChainID)

	return nil
}

func detectHeadFromRPC() error {
	rpcURL := setHeadRPC
	if rpcURL == "" {
		rpcURL = "http://localhost:9630/ext/bc/C/rpc"
	}

	ux.Logger.PrintToUser("Detecting head from RPC: %s", rpcURL)

	// Get current block number
	heightReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_blockNumber",
		"params":  []interface{}{},
		"id":      1,
	}

	heightData, _ := json.Marshal(heightReq)
	resp, err := http.Post(rpcURL, "application/json", bytes.NewBuffer(heightData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var heightResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&heightResult); err != nil {
		return err
	}

	heightHex, ok := heightResult["result"].(string)
	if !ok {
		return fmt.Errorf("invalid height response")
	}

	fmt.Sscanf(heightHex, "0x%x", &setHeadHeight)

	// Get block by number to get hash
	blockReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_getBlockByNumber",
		"params":  []interface{}{heightHex, false},
		"id":      1,
	}

	blockData, _ := json.Marshal(blockReq)
	resp2, err := http.Post(rpcURL, "application/json", bytes.NewBuffer(blockData))
	if err != nil {
		return err
	}
	defer resp2.Body.Close()

	var blockResult map[string]interface{}
	if err := json.NewDecoder(resp2.Body).Decode(&blockResult); err != nil {
		return err
	}

	if block, ok := blockResult["result"].(map[string]interface{}); ok {
		if hash, ok := block["hash"].(string); ok {
			setHeadHash = hash
			ux.Logger.PrintToUser("Detected head: height=%d hash=%s", setHeadHeight, setHeadHash)
			return nil
		}
	}

	return fmt.Errorf("failed to get block hash")
}

func detectHeadFromDB() error {
	ux.Logger.PrintToUser("Detecting head from database: %s", setHeadDBPath)

	// Open BadgerDB read-only
	opts := badger.DefaultOptions(setHeadDBPath)
	opts.ReadOnly = true
	opts.Logger = nil

	db, err := badger.Open(opts)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Look for LastHeader key
	// In Coreth/geth, this is typically: []byte("LastHeader")
	var lastHash []byte
	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("LastHeader"))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			lastHash = make([]byte, len(val))
			copy(lastHash, val)
			return nil
		})
	})

	if err != nil {
		return fmt.Errorf("failed to read LastHeader: %w", err)
	}

	// Get block header by hash to get height
	// This would need proper RLP decoding in production
	// For now, we'll require manual height input
	setHeadHash = "0x" + hex.EncodeToString(lastHash)
	ux.Logger.PrintToUser("Found last header hash: %s", setHeadHash)
	ux.Logger.PrintToUser("Please provide height with --height flag")

	return nil
}

func setDatabaseHead(hashBytes []byte) error {
	ux.Logger.PrintToUser("Setting database head...")

	// Open BadgerDB
	opts := badger.DefaultOptions(setHeadDBPath)
	opts.Logger = nil

	db, err := badger.Open(opts)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Set canonical head markers
	// These keys are used by geth/coreth to track the chain head
	headKeys := []string{
		"LastBlock",  // Last fully processed block
		"LastHeader", // Last known header
		"LastFast",   // Last fast-synced block
	}

	return db.Update(func(txn *badger.Txn) error {
		for _, key := range headKeys {
			if err := txn.Set([]byte(key), hashBytes); err != nil {
				return fmt.Errorf("failed to set %s: %w", key, err)
			}
		}
		ux.Logger.PrintToUser("  Set database head markers")
		return nil
	})
}

func setVMMetadata(hashBytes []byte) error {
	// Determine VM data directory
	vmDir := setHeadVMDataDir
	if vmDir == "" {
		home := os.Getenv("HOME")
		vmDir = filepath.Join(home, ".luxd", "chainData", "C")
	}

	ux.Logger.PrintToUser("Setting VM metadata in: %s", vmDir)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(vmDir, 0755); err != nil {
		return fmt.Errorf("failed to create VM directory: %w", err)
	}

	// Write vm/lastAccepted (block hash)
	lastAcceptedPath := filepath.Join(vmDir, "vm", "lastAccepted")
	if err := os.MkdirAll(filepath.Dir(lastAcceptedPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(lastAcceptedPath, hashBytes, 0644); err != nil {
		return fmt.Errorf("failed to write lastAccepted: %w", err)
	}
	ux.Logger.PrintToUser("  Wrote vm/lastAccepted")

	// Write vm/lastAcceptedHeight (8 bytes big-endian)
	lastHeightPath := filepath.Join(vmDir, "vm", "lastAcceptedHeight")
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, setHeadHeight)
	if err := os.WriteFile(lastHeightPath, heightBytes, 0644); err != nil {
		return fmt.Errorf("failed to write lastAcceptedHeight: %w", err)
	}
	ux.Logger.PrintToUser("  Wrote vm/lastAcceptedHeight")

	// Write vm/initialized (single byte 0x01)
	initializedPath := filepath.Join(vmDir, "vm", "initialized")
	if err := os.WriteFile(initializedPath, []byte{0x01}, 0644); err != nil {
		return fmt.Errorf("failed to write initialized: %w", err)
	}
	ux.Logger.PrintToUser("  Wrote vm/initialized")

	return nil
}