package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/cockroachdb/pebble"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

const (
	// Database namespaces for PebbleDB
	blockBodyPrefix     = 0x62 // 'b' - block body
	blockReceiptsPrefix = 0x72 // 'r' - block receipts
	headerPrefix        = 0x68 // 'h' - headers
	headerHashPrefix    = 0x48 // 'H' - header hash by number
	txLookupPrefix      = 0x6c // 'l' - transaction lookup

	// Account state prefixes
	accountPrefix = 0x61 // 'a' - accounts
	storagePrefix = 0x73 // 's' - storage
	codePrefix    = 0x63 // 'c' - code
)

type MigrationConfig struct {
	SourceDB       string
	TargetDB       string
	NetworkID      uint32
	ValidatorCount int
	OutputDir      string
}

func main() {
	var config MigrationConfig

	var networkID uint64

	flag.StringVar(&config.SourceDB, "source", "", "Path to subnet PebbleDB")
	flag.StringVar(&config.TargetDB, "target", "", "Path to output C-Chain LevelDB")
	flag.Uint64Var(&networkID, "network-id", 96369, "Network ID")
	flag.IntVar(&config.ValidatorCount, "validators", 5, "Number of validators")
	flag.StringVar(&config.OutputDir, "output", "./migration-output", "Output directory")
	flag.Parse()

	config.NetworkID = uint32(networkID)

	if config.SourceDB == "" || config.TargetDB == "" {
		log.Fatal("Source and target databases must be specified")
	}

	if err := migrateDatabase(config); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	fmt.Println("Migration completed successfully!")
}

func migrateDatabase(config MigrationConfig) error {
	log.Printf("Starting migration from %s to %s", config.SourceDB, config.TargetDB)

	// Open PebbleDB source
	pdb, err := openPebbleDB(config.SourceDB)
	if err != nil {
		return fmt.Errorf("failed to open PebbleDB: %w", err)
	}
	defer pdb.Close()

	// Create LevelDB target
	ldb, err := createLevelDB(config.TargetDB)
	if err != nil {
		return fmt.Errorf("failed to create LevelDB: %w", err)
	}
	defer ldb.Close()

	// Migrate blockchain data
	log.Println("Migrating blockchain data...")
	if err := migrateBlockchainData(pdb, ldb); err != nil {
		return fmt.Errorf("failed to migrate blockchain data: %w", err)
	}

	// Migrate state data
	log.Println("Migrating state data...")
	if err := migrateStateData(pdb, ldb); err != nil {
		return fmt.Errorf("failed to migrate state data: %w", err)
	}

	// Create genesis configuration
	log.Println("Creating genesis configuration...")
	if err := createGenesisConfig(config); err != nil {
		return fmt.Errorf("failed to create genesis config: %w", err)
	}

	// Create validator configurations
	log.Println("Creating validator configurations...")
	if err := createValidatorConfigs(config); err != nil {
		return fmt.Errorf("failed to create validator configs: %w", err)
	}

	return nil
}

func openPebbleDB(path string) (*pebble.DB, error) {
	opts := &pebble.Options{
		ReadOnly: true,
	}
	return pebble.Open(path, opts)
}

func createLevelDB(path string) (*leveldb.DB, error) {
	opts := &opt.Options{
		Compression: opt.SnappyCompression,
		WriteBuffer: 256 * 1024 * 1024, // 256MB
		BlockSize:   32 * 1024,         // 32KB
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(path, 0o755); err != nil {
		return nil, err
	}

	return leveldb.OpenFile(path, opts)
}

func migrateBlockchainData(src *pebble.DB, dst *leveldb.DB) error {
	iter, err := src.NewIter(nil)
	if err != nil {
		return fmt.Errorf("failed to create iterator: %w", err)
	}
	defer iter.Close()

	batch := new(leveldb.Batch)
	count := 0

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) == 0 {
			continue
		}

		// Check key prefix for blockchain data
		prefix := key[0]
		switch prefix {
		case blockBodyPrefix, blockReceiptsPrefix, headerPrefix,
			headerHashPrefix, txLookupPrefix:
			// Copy blockchain data - make copies of the byte slices
			keyCopy := make([]byte, len(key))
			copy(keyCopy, key)
			valueCopy := make([]byte, len(iter.Value()))
			copy(valueCopy, iter.Value())

			batch.Put(keyCopy, valueCopy)
			count++

			// Write batch every 10000 entries
			if count%10000 == 0 {
				if err := dst.Write(batch, nil); err != nil {
					return err
				}
				batch.Reset()
				log.Printf("Migrated %d blockchain entries", count)
			}
		}
	}

	// Write remaining batch
	if batch.Len() > 0 {
		if err := dst.Write(batch, nil); err != nil {
			return err
		}
	}

	log.Printf("Total blockchain entries migrated: %d", count)
	return iter.Error()
}

func migrateStateData(src *pebble.DB, dst *leveldb.DB) error {
	iter, err := src.NewIter(nil)
	if err != nil {
		return fmt.Errorf("failed to create iterator: %w", err)
	}
	defer iter.Close()

	batch := new(leveldb.Batch)
	count := 0

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) == 0 {
			continue
		}

		// Check key prefix for state data
		prefix := key[0]
		switch prefix {
		case accountPrefix, storagePrefix, codePrefix:
			// Copy state data - make copies of the byte slices
			keyCopy := make([]byte, len(key))
			copy(keyCopy, key)
			valueCopy := make([]byte, len(iter.Value()))
			copy(valueCopy, iter.Value())

			batch.Put(keyCopy, valueCopy)
			count++

			// Write batch every 10000 entries
			if count%10000 == 0 {
				if err := dst.Write(batch, nil); err != nil {
					return err
				}
				batch.Reset()
				log.Printf("Migrated %d state entries", count)
			}
		}
	}

	// Write remaining batch
	if batch.Len() > 0 {
		if err := dst.Write(batch, nil); err != nil {
			return err
		}
	}

	log.Printf("Total state entries migrated: %d", count)
	return iter.Error()
}

func createGenesisConfig(config MigrationConfig) error {
	genesis := map[string]interface{}{
		"networkId":         config.NetworkID,
		"chainId":           config.NetworkID,
		"validators":        config.ValidatorCount,
		"consensusProtocol": "snowman",
		"minBlockTime":      2000000000, // 2 seconds
	}

	genesisPath := filepath.Join(config.OutputDir, "genesis.json")
	file, err := os.Create(genesisPath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(genesis)
}

func createValidatorConfigs(config MigrationConfig) error {
	for i := 1; i <= config.ValidatorCount; i++ {
		validatorDir := filepath.Join(config.OutputDir, fmt.Sprintf("validator%d", i))
		if err := os.MkdirAll(validatorDir, 0o755); err != nil {
			return err
		}

		// Create basic node config
		nodeConfig := map[string]interface{}{
			"network-id":               config.NetworkID,
			"http-port":                9630 + (i-1)*10,
			"staking-port":             9631 + (i-1)*10,
			"db-dir":                   filepath.Join(validatorDir, "db"),
			"log-dir":                  filepath.Join(validatorDir, "logs"),
			"staking-enabled":          false,
			"sybil-protection-enabled": false,
			"snow-sample-size":         1,
			"snow-quorum-size":         1,
		}

		configPath := filepath.Join(validatorDir, "node.json")
		file, err := os.Create(configPath)
		if err != nil {
			return err
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(nodeConfig); err != nil {
			return err
		}
	}

	return nil
}
