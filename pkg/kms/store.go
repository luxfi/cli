// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package kms provides a unified Key Management Service with support for
// both embedded (BadgerDB) and distributed (PostgreSQL) storage backends.
package kms

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
)

// StorageBackend defines the storage interface for KMS operations.
type StorageBackend interface {
	// Key operations
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte) error
	SetWithTTL(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)

	// Iteration
	List(ctx context.Context, prefix string) ([]string, error)
	Scan(ctx context.Context, prefix string, fn func(key string, value []byte) error) error

	// Transaction support
	BeginTx(ctx context.Context) (Transaction, error)

	// Lifecycle
	Close() error
}

// Transaction represents a storage transaction.
type Transaction interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	Delete(key string) error
	Commit() error
	Rollback() error
}

// BadgerStore implements StorageBackend using BadgerDB for embedded storage.
type BadgerStore struct {
	db     *badger.DB
	mu     sync.RWMutex
	closed bool
}

// BadgerConfig holds BadgerDB configuration options.
type BadgerConfig struct {
	Dir           string
	InMemory      bool
	SyncWrites    bool
	Compression   bool
	EncryptionKey []byte // 16, 24, or 32 bytes for AES-128, AES-192, AES-256
}

// NewBadgerStore creates a new BadgerDB-backed storage.
func NewBadgerStore(cfg *BadgerConfig) (*BadgerStore, error) {
	opts := badger.DefaultOptions(cfg.Dir)

	if cfg.InMemory {
		opts = opts.WithInMemory(true)
	}

	opts = opts.WithSyncWrites(cfg.SyncWrites)

	if cfg.Compression {
		opts = opts.WithCompression(options.Snappy)
	}

	if len(cfg.EncryptionKey) > 0 {
		opts = opts.WithEncryptionKey(cfg.EncryptionKey)
	}

	// Performance tuning for KMS workloads
	opts = opts.WithNumVersionsToKeep(3)
	opts = opts.WithNumLevelZeroTables(5)
	opts = opts.WithNumLevelZeroTablesStall(15)
	opts = opts.WithValueLogFileSize(64 << 20) // 64MB

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open badger db: %w", err)
	}

	store := &BadgerStore{db: db}

	// Start background GC
	go store.runGC()

	return store, nil
}

// runGC periodically runs garbage collection on the value log.
func (s *BadgerStore) runGC() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.RLock()
		if s.closed {
			s.mu.RUnlock()
			return
		}
		s.mu.RUnlock()

		// Run GC until no more garbage to collect
		for {
			err := s.db.RunValueLogGC(0.5)
			if err != nil {
				break
			}
		}
	}
}

// Get retrieves a value by key.
func (s *BadgerStore) Get(ctx context.Context, key string) ([]byte, error) {
	var value []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		value, err = item.ValueCopy(nil)
		return err
	})
	if err == badger.ErrKeyNotFound {
		return nil, ErrKeyNotFound
	}
	return value, err
}

// Set stores a value at the given key.
func (s *BadgerStore) Set(ctx context.Context, key string, value []byte) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), value)
	})
}

// SetWithTTL stores a value with a time-to-live.
func (s *BadgerStore) SetWithTTL(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return s.db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte(key), value).WithTTL(ttl)
		return txn.SetEntry(e)
	})
}

// Delete removes a key from storage.
func (s *BadgerStore) Delete(ctx context.Context, key string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

// Exists checks if a key exists.
func (s *BadgerStore) Exists(ctx context.Context, key string) (bool, error) {
	err := s.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(key))
		return err
	})
	if err == badger.ErrKeyNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// List returns all keys with the given prefix.
func (s *BadgerStore) List(ctx context.Context, prefix string) ([]string, error) {
	var keys []string
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		opts.Prefix = []byte(prefix)

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek([]byte(prefix)); it.ValidForPrefix([]byte(prefix)); it.Next() {
			keys = append(keys, string(it.Item().Key()))
		}
		return nil
	})
	return keys, err
}

// Scan iterates over all key-value pairs with the given prefix.
func (s *BadgerStore) Scan(ctx context.Context, prefix string, fn func(key string, value []byte) error) error {
	return s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(prefix)

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek([]byte(prefix)); it.ValidForPrefix([]byte(prefix)); it.Next() {
			item := it.Item()
			key := string(item.Key())

			err := item.Value(func(val []byte) error {
				return fn(key, val)
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// badgerTx implements Transaction for BadgerDB.
type badgerTx struct {
	txn *badger.Txn
}

// BeginTx starts a new transaction.
func (s *BadgerStore) BeginTx(ctx context.Context) (Transaction, error) {
	return &badgerTx{txn: s.db.NewTransaction(true)}, nil
}

func (t *badgerTx) Get(key string) ([]byte, error) {
	item, err := t.txn.Get([]byte(key))
	if err == badger.ErrKeyNotFound {
		return nil, ErrKeyNotFound
	}
	if err != nil {
		return nil, err
	}
	return item.ValueCopy(nil)
}

func (t *badgerTx) Set(key string, value []byte) error {
	return t.txn.Set([]byte(key), value)
}

func (t *badgerTx) Delete(key string) error {
	return t.txn.Delete([]byte(key))
}

func (t *badgerTx) Commit() error {
	return t.txn.Commit()
}

func (t *badgerTx) Rollback() error {
	t.txn.Discard()
	return nil
}

// Close closes the BadgerDB store.
func (s *BadgerStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return s.db.Close()
}

// Helper functions for JSON storage

// GetJSON retrieves and unmarshals a JSON value.
func GetJSON[T any](ctx context.Context, store StorageBackend, key string) (*T, error) {
	data, err := store.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}
	return &v, nil
}

// SetJSON marshals and stores a JSON value.
func SetJSON(ctx context.Context, store StorageBackend, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal: %w", err)
	}
	return store.Set(ctx, key, data)
}

// Common errors
var (
	ErrKeyNotFound = fmt.Errorf("key not found")
	ErrInvalidKey  = fmt.Errorf("invalid key")
)
