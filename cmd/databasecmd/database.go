// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package databasecmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luxfi/cli/v2/pkg/ux"
	"github.com/luxfi/database"
	"github.com/spf13/cobra"
)

// NewCmd returns the database command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "database",
		Short: "Direct database operations and migrations",
		Long: `The database command provides direct access to blockchain databases for
migrations, conversions, and analysis. Supports BadgerDB, LevelDB, PebbleDB, and MemDB.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		Args: cobra.NoArgs,
	}

	// Add subcommands
	cmd.AddCommand(newMigrateCmd())
	cmd.AddCommand(newInspectCmd())
	cmd.AddCommand(newCompactCmd())
	cmd.AddCommand(newStatsCmd())

	return cmd
}

// newMigrateCmd creates the migrate subcommand
func newMigrateCmd() *cobra.Command {
	var (
		sourcePath   string
		sourceType   string
		targetPath   string
		targetType   string
		batchSize    int
		skipVerify   bool
	)

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate database from one type to another",
		Long: `Migrate blockchain data between different database backends.
Commonly used to convert from PebbleDB to BadgerDB for better performance.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return migrateDatabase(sourcePath, sourceType, targetPath, targetType, batchSize, skipVerify)
		},
		Args: cobra.NoArgs,
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&sourcePath, "source", "", "Source database path")
	cmd.Flags().StringVar(&sourceType, "source-type", "pebbledb", "Source database type (badgerdb, leveldb, pebbledb)")
	cmd.Flags().StringVar(&targetPath, "target", "", "Target database path")
	cmd.Flags().StringVar(&targetType, "target-type", "badgerdb", "Target database type (badgerdb, leveldb, pebbledb)")
	cmd.Flags().IntVar(&batchSize, "batch-size", 10000, "Batch size for migration")
	cmd.Flags().BoolVar(&skipVerify, "skip-verify", false, "Skip verification after migration")

	cmd.MarkFlagRequired("source")
	cmd.MarkFlagRequired("target")

	return cmd
}

// newInspectCmd creates the inspect subcommand
func newInspectCmd() *cobra.Command {
	var (
		dbPath  string
		dbType  string
		prefix  string
		limit   int
		showKeys bool
	)

	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect database contents",
		Long:  "Examine the contents of a blockchain database, including keys and values",
		RunE: func(cmd *cobra.Command, args []string) error {
			return inspectDatabase(dbPath, dbType, prefix, limit, showKeys)
		},
		Args: cobra.NoArgs,
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&dbPath, "path", "", "Database path")
	cmd.Flags().StringVar(&dbType, "type", "badgerdb", "Database type (badgerdb, leveldb, pebbledb)")
	cmd.Flags().StringVar(&prefix, "prefix", "", "Key prefix to filter (hex)")
	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum number of entries to show")
	cmd.Flags().BoolVar(&showKeys, "show-keys", false, "Show full key contents")

	cmd.MarkFlagRequired("path")

	return cmd
}

// newCompactCmd creates the compact subcommand
func newCompactCmd() *cobra.Command {
	var (
		dbPath string
		dbType string
	)

	cmd := &cobra.Command{
		Use:   "compact",
		Short: "Compact database to reclaim space",
		Long:  "Run compaction on the database to reclaim disk space and improve performance",
		RunE: func(cmd *cobra.Command, args []string) error {
			return compactDatabase(dbPath, dbType)
		},
		Args: cobra.NoArgs,
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&dbPath, "path", "", "Database path")
	cmd.Flags().StringVar(&dbType, "type", "badgerdb", "Database type (badgerdb, leveldb, pebbledb)")

	cmd.MarkFlagRequired("path")

	return cmd
}

// newStatsCmd creates the stats subcommand
func newStatsCmd() *cobra.Command {
	var (
		dbPath string
		dbType string
	)

	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show database statistics",
		Long:  "Display statistics about the database including size, key count, and performance metrics",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showDatabaseStats(dbPath, dbType)
		},
		Args: cobra.NoArgs,
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&dbPath, "path", "", "Database path")
	cmd.Flags().StringVar(&dbType, "type", "badgerdb", "Database type (badgerdb, leveldb, pebbledb)")

	cmd.MarkFlagRequired("path")

	return cmd
}

// migrateDatabase migrates data from one database type to another
func migrateDatabase(sourcePath, sourceType, targetPath, targetType string, batchSize int, skipVerify bool) error {
	ux.Logger.PrintToUser("🔄 Database Migration")
	ux.Logger.PrintToUser("=" + strings.Repeat("=", 50))
	ux.Logger.PrintToUser("Source: %s (%s)", sourcePath, sourceType)
	ux.Logger.PrintToUser("Target: %s (%s)", targetPath, targetType)
	ux.Logger.PrintToUser("Batch Size: %d", batchSize)

	// Verify source exists
	if _, err := os.Stat(sourcePath); err != nil {
		return fmt.Errorf("source database not found: %w", err)
	}

	// Create target directory
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Open source database
	sourceConfig := []database.Option{
		database.WithPath(sourcePath),
	}
	
	var sourceDB database.Database
	var err error
	
	switch sourceType {
	case "badgerdb":
		sourceDB, err = database.NewBadgerDB(sourceConfig...)
	case "leveldb":
		sourceDB, err = database.NewLevelDB(sourceConfig...)
	case "pebbledb":
		sourceDB, err = database.NewPebbleDB(sourceConfig...)
	default:
		return fmt.Errorf("unsupported source database type: %s", sourceType)
	}
	
	if err != nil {
		return fmt.Errorf("failed to open source database: %w", err)
	}
	defer sourceDB.Close()

	// Open target database
	targetConfig := []database.Option{
		database.WithPath(targetPath),
	}
	
	var targetDB database.Database
	
	switch targetType {
	case "badgerdb":
		targetDB, err = database.NewBadgerDB(targetConfig...)
	case "leveldb":
		targetDB, err = database.NewLevelDB(targetConfig...)
	case "pebbledb":
		targetDB, err = database.NewPebbleDB(targetConfig...)
	default:
		return fmt.Errorf("unsupported target database type: %s", targetType)
	}
	
	if err != nil {
		return fmt.Errorf("failed to open target database: %w", err)
	}
	defer targetDB.Close()

	// Perform migration
	ux.Logger.PrintToUser("\n⏳ Migrating data...")
	
	iterator := sourceDB.NewIterator()
	defer iterator.Release()

	batch := targetDB.NewBatch()
	count := 0
	totalCount := 0

	for iterator.Next() {
		key := iterator.Key()
		value := iterator.Value()
		
		if err := batch.Put(key, value); err != nil {
			return fmt.Errorf("failed to put key: %w", err)
		}
		
		count++
		totalCount++
		
		if count >= batchSize {
			if err := batch.Write(); err != nil {
				return fmt.Errorf("failed to write batch: %w", err)
			}
			batch.Reset()
			count = 0
			
			ux.Logger.PrintToUser("   Migrated %d entries...", totalCount)
		}
	}

	// Write final batch
	if count > 0 {
		if err := batch.Write(); err != nil {
			return fmt.Errorf("failed to write final batch: %w", err)
		}
	}

	if err := iterator.Error(); err != nil {
		return fmt.Errorf("iterator error: %w", err)
	}

	ux.Logger.PrintToUser("\n✅ Migration completed!")
	ux.Logger.PrintToUser("   Total entries: %d", totalCount)

	// Verify migration if requested
	if !skipVerify {
		ux.Logger.PrintToUser("\n🔍 Verifying migration...")
		if err := verifyMigration(sourceDB, targetDB); err != nil {
			return fmt.Errorf("verification failed: %w", err)
		}
		ux.Logger.PrintToUser("✅ Verification passed!")
	}

	return nil
}

// verifyMigration verifies that all data was migrated correctly
func verifyMigration(sourceDB, targetDB database.Database) error {
	sourceIter := sourceDB.NewIterator()
	defer sourceIter.Release()

	mismatches := 0
	checked := 0

	for sourceIter.Next() {
		key := sourceIter.Key()
		sourceValue := sourceIter.Value()

		targetValue, err := targetDB.Get(key)
		if err != nil {
			return fmt.Errorf("key not found in target: %x", key)
		}

		if !bytesEqual(sourceValue, targetValue) {
			mismatches++
			if mismatches <= 5 {
				ux.Logger.PrintToUser("   Mismatch at key %x", key)
			}
		}

		checked++
		if checked%10000 == 0 {
			ux.Logger.PrintToUser("   Verified %d entries...", checked)
		}
	}

	if mismatches > 0 {
		return fmt.Errorf("found %d mismatches", mismatches)
	}

	return sourceIter.Error()
}

// bytesEqual compares two byte slices
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// inspectDatabase examines database contents
func inspectDatabase(dbPath, dbType, prefix string, limit int, showKeys bool) error {
	ux.Logger.PrintToUser("🔍 Database Inspection")
	ux.Logger.PrintToUser("=" + strings.Repeat("=", 50))
	ux.Logger.PrintToUser("Path: %s", dbPath)
	ux.Logger.PrintToUser("Type: %s", dbType)

	// Open database
	config := []database.Option{
		database.WithPath(dbPath),
		database.WithReadOnly(true),
	}
	
	var db database.Database
	var err error
	
	switch dbType {
	case "badgerdb":
		db, err = database.NewBadgerDB(config...)
	case "leveldb":
		db, err = database.NewLevelDB(config...)
	case "pebbledb":
		db, err = database.NewPebbleDB(config...)
	default:
		return fmt.Errorf("unsupported database type: %s", dbType)
	}
	
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Iterate through database
	iterator := db.NewIterator()
	defer iterator.Release()

	if prefix != "" {
		iterator = db.NewIteratorWithPrefix([]byte(prefix))
	}

	count := 0
	for iterator.Next() && count < limit {
		key := iterator.Key()
		value := iterator.Value()

		if showKeys {
			ux.Logger.PrintToUser("\nKey[%d]: %x", count, key)
			ux.Logger.PrintToUser("Value: %x", value)
			ux.Logger.PrintToUser("Size: %d bytes", len(value))
		} else {
			ux.Logger.PrintToUser("Entry[%d]: key=%d bytes, value=%d bytes", count, len(key), len(value))
		}

		count++
	}

	if iterator.Error() != nil {
		return fmt.Errorf("iterator error: %w", iterator.Error())
	}

	ux.Logger.PrintToUser("\nTotal entries shown: %d", count)

	return nil
}

// compactDatabase runs compaction on the database
func compactDatabase(dbPath, dbType string) error {
	ux.Logger.PrintToUser("🗜️  Database Compaction")
	ux.Logger.PrintToUser("=" + strings.Repeat("=", 50))
	ux.Logger.PrintToUser("Path: %s", dbPath)
	ux.Logger.PrintToUser("Type: %s", dbType)

	// Get initial size
	initialSize, err := getDirSize(dbPath)
	if err != nil {
		return fmt.Errorf("failed to get initial size: %w", err)
	}

	ux.Logger.PrintToUser("Initial size: %s", formatBytes(initialSize))

	// Open database
	config := []database.Option{
		database.WithPath(dbPath),
	}
	
	var db database.Database
	
	switch dbType {
	case "badgerdb":
		db, err = database.NewBadgerDB(config...)
	case "leveldb":
		db, err = database.NewLevelDB(config...)
	case "pebbledb":
		db, err = database.NewPebbleDB(config...)
	default:
		return fmt.Errorf("unsupported database type: %s", dbType)
	}
	
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	ux.Logger.PrintToUser("\n⏳ Running compaction...")
	
	// Run compaction
	if compactor, ok := db.(database.Compactor); ok {
		if err := compactor.Compact(nil, nil); err != nil {
			db.Close()
			return fmt.Errorf("compaction failed: %w", err)
		}
	} else {
		db.Close()
		return fmt.Errorf("database type %s does not support compaction", dbType)
	}

	db.Close()

	// Get final size
	finalSize, err := getDirSize(dbPath)
	if err != nil {
		return fmt.Errorf("failed to get final size: %w", err)
	}

	ux.Logger.PrintToUser("\n✅ Compaction completed!")
	ux.Logger.PrintToUser("Final size: %s", formatBytes(finalSize))
	ux.Logger.PrintToUser("Space saved: %s (%.1f%%)", 
		formatBytes(initialSize-finalSize),
		float64(initialSize-finalSize)/float64(initialSize)*100)

	return nil
}

// showDatabaseStats displays database statistics
func showDatabaseStats(dbPath, dbType string) error {
	ux.Logger.PrintToUser("📊 Database Statistics")
	ux.Logger.PrintToUser("=" + strings.Repeat("=", 50))
	ux.Logger.PrintToUser("Path: %s", dbPath)
	ux.Logger.PrintToUser("Type: %s", dbType)

	// Get directory size
	size, err := getDirSize(dbPath)
	if err != nil {
		return fmt.Errorf("failed to get size: %w", err)
	}

	ux.Logger.PrintToUser("\n📁 Storage:")
	ux.Logger.PrintToUser("   Total size: %s", formatBytes(size))

	// Open database for more stats
	config := []database.Option{
		database.WithPath(dbPath),
		database.WithReadOnly(true),
	}
	
	var db database.Database
	
	switch dbType {
	case "badgerdb":
		db, err = database.NewBadgerDB(config...)
	case "leveldb":
		db, err = database.NewLevelDB(config...)
	case "pebbledb":
		db, err = database.NewPebbleDB(config...)
	default:
		return fmt.Errorf("unsupported database type: %s", dbType)
	}
	
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Count entries
	ux.Logger.PrintToUser("\n📊 Counting entries...")
	
	iterator := db.NewIterator()
	defer iterator.Release()

	count := 0
	totalKeySize := int64(0)
	totalValueSize := int64(0)

	for iterator.Next() {
		count++
		totalKeySize += int64(len(iterator.Key()))
		totalValueSize += int64(len(iterator.Value()))

		if count%100000 == 0 {
			ux.Logger.PrintToUser("   Processed %d entries...", count)
		}
	}

	if iterator.Error() != nil {
		return fmt.Errorf("iterator error: %w", iterator.Error())
	}

	ux.Logger.PrintToUser("\n📈 Statistics:")
	ux.Logger.PrintToUser("   Total entries: %s", formatNumber(int64(count)))
	ux.Logger.PrintToUser("   Average key size: %d bytes", totalKeySize/int64(count))
	ux.Logger.PrintToUser("   Average value size: %d bytes", totalValueSize/int64(count))
	ux.Logger.PrintToUser("   Total key size: %s", formatBytes(totalKeySize))
	ux.Logger.PrintToUser("   Total value size: %s", formatBytes(totalValueSize))

	return nil
}

// getDirSize calculates the total size of a directory
func getDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// formatBytes formats bytes in human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatNumber formats a number with thousands separators
func formatNumber(n int64) string {
	str := fmt.Sprintf("%d", n)
	var result strings.Builder
	
	for i, digit := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result.WriteString(",")
		}
		result.WriteRune(digit)
	}
	
	return result.String()
}