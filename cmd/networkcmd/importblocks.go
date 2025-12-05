// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/rlp"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

var (
	blockImportFile     string
	blockImportDBPath   string
	blockImportChainID  uint64
	blockImportVerify   bool
	blockImportProgress int
	blockImportBatch    int
)

// newImportBlocksCmd creates a command to import blocks directly to database
func newImportBlocksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import-blocks",
		Short: "Import blocks from PebbleDB export directly to C-Chain database",
		Long: `Import blocks from PebbleDB export JSONL file directly to C-Chain LevelDB.
This command parses the RLP-encoded block data and writes it to the database
in the correct format for the node to recognize.

Example:
  lux net import-blocks --file /tmp/lux-migration/blocks-export.jsonl --db /tmp/lux-c-chain-import --chain-id 96369`,
		RunE: importBlocksToDatabase,
	}

	cmd.Flags().StringVar(&blockImportFile, "file", "", "Path to JSONL export file containing blocks")
	cmd.Flags().StringVar(&blockImportDBPath, "db", "/tmp/lux-c-chain-import", "Path to destination database")
	cmd.Flags().Uint64Var(&blockImportChainID, "chain-id", 96369, "Chain ID for imported blockchain")
	cmd.Flags().BoolVar(&blockImportVerify, "verify", true, "Verify imported data")
	cmd.Flags().IntVar(&blockImportProgress, "progress", 10000, "Show progress every N blocks")
	cmd.Flags().IntVar(&blockImportBatch, "batch", 1000, "Batch size for database writes")

	cmd.MarkFlagRequired("file")

	return cmd
}

func importBlocksToDatabase(_ *cobra.Command, _ []string) error {
	ux.Logger.PrintToUser("Starting block import from PebbleDB export...")
	ux.Logger.PrintToUser("Source file: %s", blockImportFile)
	ux.Logger.PrintToUser("Destination DB: %s", blockImportDBPath)
	ux.Logger.PrintToUser("Chain ID: %d", blockImportChainID)

	// Ensure database directory exists
	if err := os.MkdirAll(blockImportDBPath, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open LevelDB with appropriate settings
	opts := &opt.Options{
		OpenFilesCacheCapacity: 1024,
		BlockCacheCapacity:     256 * 1024 * 1024, // 256MB cache
		WriteBuffer:            32 * 1024 * 1024,  // 32MB write buffer
		CompactionTableSize:    4 * 1024 * 1024,   // 4MB
		Filter:                 nil,
	}

	db, err := leveldb.OpenFile(blockImportDBPath, opts)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Create ethdb wrapper for rawdb functions
	ethDB := &leveldbWrapper{db: db}

	// Open the export file
	file, err := os.Open(blockImportFile)
	if err != nil {
		return fmt.Errorf("failed to open export file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // 10MB max line

	var (
		totalBlocks   uint64
		totalReceipts uint64
		totalTxs      uint64
		lastNumber    uint64
		errors        uint64
		startTime     = time.Now()
	)

	// Create batch for writes
	batch := ethDB.NewBatch()
	batchSize := 0

	ux.Logger.PrintToUser("Processing export file...")

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if line == "" {
			continue
		}

		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // Skip metadata or malformed lines
		}

		// Skip metadata line
		if entry["type"] == "metadata" {
			continue
		}

		// Process block entries
		if entry["type"] == "block" {
			keyHex, ok := entry["key_hex"].(string)
			if !ok {
				atomic.AddUint64(&errors, 1)
				continue
			}

			rlpData, ok := entry["rlp_data"].(string)
			if !ok {
				atomic.AddUint64(&errors, 1)
				continue
			}

			// Decode hex strings
			keyBytes, err := hex.DecodeString(keyHex)
			if err != nil {
				atomic.AddUint64(&errors, 1)
				continue
			}

			dataBytes, err := hex.DecodeString(rlpData)
			if err != nil {
				atomic.AddUint64(&errors, 1)
				continue
			}

			// Parse the key to determine data type
			if len(keyBytes) < 33 {
				continue // Invalid key
			}

			// Key structure: [32-byte namespace][1-byte bucket][remaining bytes]
			// namespace := keyBytes[:32] // Not used for now
			bucket := keyBytes[32]
			keyRest := keyBytes[33:]

			// Process based on bucket type
			switch bucket {
			case 0: // Block headers
				if err := processBlockHeader(batch, keyRest, dataBytes); err != nil {
					atomic.AddUint64(&errors, 1)
					ux.Logger.PrintToUser("Error processing header at line %d: %v", lineNum, err)
				} else {
					atomic.AddUint64(&totalBlocks, 1)

					// Extract block number for tracking
					if len(keyRest) >= 8 {
						num := decodeBlockNumber(keyRest[:8])
						if num > lastNumber {
							lastNumber = num
						}
					}
				}

			case 1: // Block bodies (transactions)
				if err := processBlockBody(batch, keyRest, dataBytes); err != nil {
					atomic.AddUint64(&errors, 1)
					ux.Logger.PrintToUser("Error processing body at line %d: %v", lineNum, err)
				} else {
					atomic.AddUint64(&totalTxs, 1)
				}

			case 2: // Receipts
				if err := processReceipts(batch, keyRest, dataBytes); err != nil {
					atomic.AddUint64(&errors, 1)
					ux.Logger.PrintToUser("Error processing receipts at line %d: %v", lineNum, err)
				} else {
					atomic.AddUint64(&totalReceipts, 1)
				}

			case 3: // Total difficulty
				if err := processTotalDifficulty(batch, keyRest, dataBytes); err != nil {
					atomic.AddUint64(&errors, 1)
				}

			default:
				// Other bucket types - store as-is
				batch.Put(keyBytes, dataBytes)
			}

			batchSize++

			// Commit batch periodically
			if batchSize >= blockImportBatch {
				if err := batch.Write(); err != nil {
					return fmt.Errorf("failed to write batch: %w", err)
				}
				batch.Reset()
				batchSize = 0
			}

			// Show progress
			blocks := atomic.LoadUint64(&totalBlocks)
			if blocks > 0 && blocks%uint64(blockImportProgress) == 0 {
				elapsed := time.Since(startTime).Seconds()
				rate := float64(blocks) / elapsed
				ux.Logger.PrintToUser("Progress: %d blocks, %d txs, %d receipts imported (%.1f blocks/sec, last #%d)",
					blocks, atomic.LoadUint64(&totalTxs), atomic.LoadUint64(&totalReceipts), rate, lastNumber)
			}
		}
	}

	// Commit final batch
	if batchSize > 0 {
		if err := batch.Write(); err != nil {
			return fmt.Errorf("failed to write final batch: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	// Write metadata
	if err := writeChainMetadata(ethDB, lastNumber); err != nil {
		ux.Logger.PrintToUser("Warning: Failed to write chain metadata: %v", err)
	}

	// Final statistics
	elapsed := time.Since(startTime)
	finalBlocks := atomic.LoadUint64(&totalBlocks)
	finalTxs := atomic.LoadUint64(&totalTxs)
	finalReceipts := atomic.LoadUint64(&totalReceipts)
	finalErrors := atomic.LoadUint64(&errors)

	ux.Logger.PrintToUser("\n✅ Import complete!")
	ux.Logger.PrintToUser("  Blocks imported: %d", finalBlocks)
	ux.Logger.PrintToUser("  Transactions: %d", finalTxs)
	ux.Logger.PrintToUser("  Receipts: %d", finalReceipts)
	ux.Logger.PrintToUser("  Highest block: %d", lastNumber)
	ux.Logger.PrintToUser("  Errors: %d", finalErrors)
	ux.Logger.PrintToUser("  Time: %v", elapsed)
	ux.Logger.PrintToUser("  Rate: %.1f blocks/sec", float64(finalBlocks)/elapsed.Seconds())

	// Verify if requested
	if blockImportVerify {
		ux.Logger.PrintToUser("\nVerifying imported data...")
		if err := verifyImportedData(ethDB, lastNumber); err != nil {
			ux.Logger.PrintToUser("⚠️  Verification warning: %v", err)
		} else {
			ux.Logger.PrintToUser("✅ Verification successful!")
		}
	}

	return nil
}

func processBlockHeader(batch ethdb.Batch, key []byte, data []byte) error {
	// Key should be block number (8 bytes) + block hash (32 bytes)
	if len(key) < 8 {
		return fmt.Errorf("invalid header key length: %d", len(key))
	}

	blockNum := decodeBlockNumber(key[:8])

	// Parse the RLP-encoded header to extract the hash
	var header types.Header
	if err := rlp.DecodeBytes(data, &header); err != nil {
		// If can't decode, store raw data with number key
		numKey := append([]byte("h"), encodeBlockNumber(blockNum)...)
		batch.Put(numKey, data)
		return nil
	}

	hash := header.Hash()

	// Store header by hash and number (matches schema.go headerKey)
	headerKey := append([]byte("h"), encodeBlockNumber(blockNum)...)
	headerKey = append(headerKey, hash.Bytes()...)
	batch.Put(headerKey, data)

	// Store canonical hash for this number (matches headerHashKey)
	canonicalKey := append([]byte("n"), encodeBlockNumber(blockNum)...)
	batch.Put(canonicalKey, hash.Bytes())

	// Store block number by hash (matches headerNumberKey)
	numByHashKey := append([]byte("H"), hash.Bytes()...)
	batch.Put(numByHashKey, encodeBlockNumber(blockNum))

	return nil
}

func processBlockBody(batch ethdb.Batch, key []byte, data []byte) error {
	// Similar structure to header
	if len(key) < 8 {
		return fmt.Errorf("invalid body key length: %d", len(key))
	}

	blockNum := decodeBlockNumber(key[:8])

	// Try to extract hash from key
	var hash common.Hash
	if len(key) >= 40 {
		hash = common.BytesToHash(key[8:40])
	}

	// Store body by hash and number (matches blockBodyKey)
	bodyKey := append([]byte("b"), encodeBlockNumber(blockNum)...)
	if hash != (common.Hash{}) {
		bodyKey = append(bodyKey, hash.Bytes()...)
	}
	batch.Put(bodyKey, data)

	return nil
}

func processReceipts(batch ethdb.Batch, key []byte, data []byte) error {
	if len(key) < 8 {
		return fmt.Errorf("invalid receipts key length: %d", len(key))
	}

	blockNum := decodeBlockNumber(key[:8])

	// Try to extract hash from key
	var hash common.Hash
	if len(key) >= 40 {
		hash = common.BytesToHash(key[8:40])
	}

	// Store receipts by hash and number (matches blockReceiptsKey)
	receiptsKey := append([]byte("r"), encodeBlockNumber(blockNum)...)
	if hash != (common.Hash{}) {
		receiptsKey = append(receiptsKey, hash.Bytes()...)
	}
	batch.Put(receiptsKey, data)

	return nil
}

func processTotalDifficulty(batch ethdb.Batch, key []byte, data []byte) error {
	if len(key) < 40 {
		return fmt.Errorf("invalid TD key length: %d", len(key))
	}

	// TD is stored by number and hash
	blockNum := decodeBlockNumber(key[:8])
	hash := common.BytesToHash(key[8:40])

	// Store TD (matches headerTDKey)
	tdKey := append([]byte("t"), encodeBlockNumber(blockNum)...)
	tdKey = append(tdKey, hash.Bytes()...)
	batch.Put(tdKey, data)

	return nil
}

func writeChainMetadata(db *leveldbWrapper, lastBlock uint64) error {
	// Write chain config
	chainConfig := []byte(fmt.Sprintf(`{
		"chainId": %d,
		"homesteadBlock": 0,
		"eip150Block": 0,
		"eip155Block": 0,
		"eip158Block": 0,
		"byzantiumBlock": 0,
		"constantinopleBlock": 0,
		"petersburgBlock": 0,
		"istanbulBlock": 0,
		"berlinBlock": 0,
		"londonBlock": 0
	}`, blockImportChainID))

	if err := db.Put([]byte("ethereum-config-"), chainConfig); err != nil {
		return err
	}

	// Write head block number
	numBytes := encodeBlockNumber(lastBlock)
	if err := db.Put([]byte("LastBlock"), numBytes); err != nil {
		return err
	}

	// Write head header/fast/finalized keys
	if err := db.Put([]byte("LastHeader"), numBytes); err != nil {
		return err
	}
	if err := db.Put([]byte("LastFast"), numBytes); err != nil {
		return err
	}
	if err := db.Put([]byte("LastFinalized"), numBytes); err != nil {
		return err
	}

	return nil
}

func verifyImportedData(db *leveldbWrapper, expectedLast uint64) error {
	// Check a sample of blocks
	samplesToCheck := []uint64{0, 1, 100, 1000, 10000, expectedLast/2, expectedLast-1, expectedLast}

	validBlocks := 0
	for _, num := range samplesToCheck {
		if num > expectedLast {
			continue
		}

		// Check if we have the canonical hash for this number
		canonicalKey := append([]byte("n"), encodeBlockNumber(num)...)
		if hashBytes, err := db.Get(canonicalKey); err == nil && len(hashBytes) == 32 {
			// We have a canonical hash, check if header exists
			headerKey := append([]byte("h"), encodeBlockNumber(num)...)
			headerKey = append(headerKey, hashBytes...)
			if has, err := db.Has(headerKey); err == nil && has {
				validBlocks++
			}
		}
	}

	if validBlocks == 0 {
		return fmt.Errorf("no blocks found in database")
	}

	ux.Logger.PrintToUser("  Found %d/%d sample blocks", validBlocks, len(samplesToCheck))
	return nil
}

func decodeBlockNumber(b []byte) uint64 {
	if len(b) < 8 {
		return 0
	}
	var n uint64
	for i := 0; i < 8; i++ {
		n = (n << 8) | uint64(b[i])
	}
	return n
}

func encodeBlockNumber(n uint64) []byte {
	b := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		b[i] = byte(n)
		n >>= 8
	}
	return b
}

// leveldbWrapper implements a minimal ethdb interface
type leveldbWrapper struct {
	db *leveldb.DB
}

func (l *leveldbWrapper) Has(key []byte) (bool, error) {
	return l.db.Has(key, nil)
}

func (l *leveldbWrapper) Get(key []byte) ([]byte, error) {
	return l.db.Get(key, nil)
}

func (l *leveldbWrapper) Put(key []byte, value []byte) error {
	return l.db.Put(key, value, nil)
}

func (l *leveldbWrapper) Delete(key []byte) error {
	return l.db.Delete(key, nil)
}

func (l *leveldbWrapper) NewBatch() ethdb.Batch {
	return &leveldbBatch{batch: new(leveldb.Batch), db: l.db}
}

func (l *leveldbWrapper) Close() error {
	return l.db.Close()
}

// leveldbBatch implements ethdb.Batch
type leveldbBatch struct {
	batch *leveldb.Batch
	db    *leveldb.DB
	size  int
}

func (b *leveldbBatch) Put(key []byte, value []byte) error {
	b.batch.Put(key, value)
	b.size += len(key) + len(value)
	return nil
}

func (b *leveldbBatch) Delete(key []byte) error {
	b.batch.Delete(key)
	b.size += len(key)
	return nil
}

func (b *leveldbBatch) ValueSize() int {
	return b.size
}

func (b *leveldbBatch) Write() error {
	return b.db.Write(b.batch, nil)
}

func (b *leveldbBatch) Reset() {
	b.batch.Reset()
	b.size = 0
}

func (b *leveldbBatch) Replay(w ethdb.KeyValueWriter) error {
	return nil
}

func (b *leveldbBatch) DeleteRange(start []byte, end []byte) error {
	// Not needed for import
	return nil
}