package brand

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const lockFileName = "chain.lock"

// LockPath is <DataDir>/chain.lock.
func (p *RuntimeProfile) LockPath() string {
	return filepath.Join(p.DataDir, lockFileName)
}

// ReadLock loads the lock if present. Returns (nil, nil) when absent.
func (p *RuntimeProfile) ReadLock() (*ChainLock, error) {
	b, err := os.ReadFile(p.LockPath()) //nolint:gosec
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var lk ChainLock
	if err := json.Unmarshal(b, &lk); err != nil {
		return nil, fmt.Errorf("parse %s: %w", p.LockPath(), err)
	}
	return &lk, nil
}

// WriteLock writes a fresh manifest. Caller computes genesisHash.
func (p *RuntimeProfile) WriteLock(genesisHash string) error {
	if err := os.MkdirAll(p.DataDir, 0o750); err != nil {
		return err
	}
	lk := ChainLock{
		Version:     SchemaVersion,
		Brand:       p.Brand,
		Env:         p.Env,
		NetworkID:   p.NetworkID,
		HTTPPort:    p.HTTPPort,
		StakingPort: p.StakingPort,
		GenesisHash: genesisHash,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	b, err := json.MarshalIndent(lk, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p.LockPath(), b, 0o600)
}

// VerifyOrCreate inspects DataDir. If a lock exists it must match the
// profile (networkID + genesisHash). If absent and DataDir is empty, we
// write a fresh lock. If absent but DataDir has chain data, the dir is
// orphaned — refuse to mount so we don't corrupt unknown state.
func (p *RuntimeProfile) VerifyOrCreate(genesisHash string) error {
	lk, err := p.ReadLock()
	if err != nil {
		return err
	}
	if lk != nil {
		if lk.NetworkID != p.NetworkID {
			return fmt.Errorf(
				"%s networkID mismatch: lock=%d, profile=%d (refusing to mount foreign state)",
				p.DataDir, lk.NetworkID, p.NetworkID,
			)
		}
		if lk.GenesisHash != "" && genesisHash != "" && lk.GenesisHash != genesisHash {
			return fmt.Errorf(
				"%s genesisHash mismatch: lock=%s, computed=%s",
				p.DataDir, lk.GenesisHash, genesisHash,
			)
		}
		if lk.HTTPPort != 0 && lk.HTTPPort != p.HTTPPort {
			return fmt.Errorf(
				"%s httpPort mismatch: lock=%d, profile=%d",
				p.DataDir, lk.HTTPPort, p.HTTPPort,
			)
		}
		return nil
	}
	// No lock. Are we mounting an unknown state dir?
	if entries, _ := os.ReadDir(p.DataDir); len(entries) > 0 {
		return fmt.Errorf(
			"%s has data but no %s — refusing to mount unknown state. delete dir or write %s by hand",
			p.DataDir, lockFileName, lockFileName,
		)
	}
	return p.WriteLock(genesisHash)
}
