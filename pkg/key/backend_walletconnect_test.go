// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package key

import (
	"context"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWalletConnectBackend_Interface(t *testing.T) {
	// Verify WalletConnectBackend implements KeyBackend interface
	var _ KeyBackend = (*WalletConnectBackend)(nil)
}

func TestWalletConnectBackend_Type(t *testing.T) {
	b := NewWalletConnectBackend()
	if b.Type() != BackendWalletConnect {
		t.Errorf("Type() = %v, want %v", b.Type(), BackendWalletConnect)
	}
}

func TestWalletConnectBackend_Name(t *testing.T) {
	b := NewWalletConnectBackend()
	name := b.Name()
	if name != "WalletConnect (Mobile Signing)" {
		t.Errorf("Name() = %v, want WalletConnect (Mobile Signing)", name)
	}
}

func TestWalletConnectBackend_Available(t *testing.T) {
	b := NewWalletConnectBackend()
	if !b.Available() {
		t.Error("Available() should always return true")
	}
}

func TestWalletConnectBackend_Properties(t *testing.T) {
	b := NewWalletConnectBackend()

	if b.RequiresPassword() {
		t.Error("RequiresPassword() should be false")
	}

	if b.RequiresHardware() {
		t.Error("RequiresHardware() should be false")
	}

	if !b.SupportsRemoteSigning() {
		t.Error("SupportsRemoteSigning() should be true")
	}
}

func TestWalletConnectBackend_Initialize(t *testing.T) {
	tmpDir := t.TempDir()

	b := NewWalletConnectBackend()
	b.dataDir = filepath.Join(tmpDir, ".walletconnect")

	ctx := context.Background()
	if err := b.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Check directory was created
	if _, err := os.Stat(b.dataDir); os.IsNotExist(err) {
		t.Error("Initialize() did not create data directory")
	}

	// Close should not fail
	if err := b.Close(); err != nil {
		t.Errorf("Close() failed: %v", err)
	}
}

func TestWalletConnectBackend_CreateKeyNotSupported(t *testing.T) {
	b := NewWalletConnectBackend()
	b.dataDir = t.TempDir()

	ctx := context.Background()
	_, err := b.CreateKey(ctx, "test", CreateKeyOptions{})

	if err == nil {
		t.Error("CreateKey() should return error for WalletConnect backend")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("CreateKey() error should mention 'not supported', got: %v", err)
	}
}

func TestWalletConnectBackend_PairGeneratesValidURI(t *testing.T) {
	b := NewWalletConnectBackend()
	b.dataDir = t.TempDir()
	b.projectID = "test-project-id"

	ctx := context.Background()
	uri, err := b.Pair(ctx, "test-wallet", 1)
	if err != nil {
		t.Fatalf("Pair() failed: %v", err)
	}

	// Validate URI format: wc:{topic}@2?relay-protocol=irn&symKey={symKey}
	if !strings.HasPrefix(uri, "wc:") {
		t.Errorf("Pair() URI should start with 'wc:', got: %s", uri)
	}
	if !strings.Contains(uri, "@2") {
		t.Errorf("Pair() URI should contain '@2' for version 2, got: %s", uri)
	}
	if !strings.Contains(uri, "relay-protocol=irn") {
		t.Errorf("Pair() URI should contain relay-protocol=irn, got: %s", uri)
	}
	if !strings.Contains(uri, "symKey=") {
		t.Errorf("Pair() URI should contain symKey=, got: %s", uri)
	}

	// Check session was created
	b.mu.RLock()
	session, ok := b.sessions["test-wallet"]
	b.mu.RUnlock()

	if !ok {
		t.Error("Pair() did not create session")
	}
	if session.ChainID != 1 {
		t.Errorf("Session ChainID = %d, want 1", session.ChainID)
	}
	if len(session.SymKey) != 32 {
		t.Errorf("Session SymKey length = %d, want 32", len(session.SymKey))
	}
	if len(session.Topic) != 64 { // 32 bytes hex-encoded
		t.Errorf("Session Topic length = %d, want 64", len(session.Topic))
	}
}

func TestWalletConnectBackend_LoadKeyNotPaired(t *testing.T) {
	b := NewWalletConnectBackend()
	b.dataDir = t.TempDir()

	ctx := context.Background()
	_, err := b.LoadKey(ctx, "nonexistent", "")

	if err != ErrKeyNotFound {
		t.Errorf("LoadKey() error = %v, want %v", err, ErrKeyNotFound)
	}
}

func TestWalletConnectBackend_LoadKeyExpiredSession(t *testing.T) {
	b := NewWalletConnectBackend()
	b.dataDir = t.TempDir()

	// Create expired session
	b.mu.Lock()
	b.sessions["expired"] = &wcSession{
		Topic:     "test-topic",
		Address:   "0x1234567890123456789012345678901234567890",
		ExpiresAt: time.Now().Add(-time.Hour), // Expired
	}
	b.mu.Unlock()

	ctx := context.Background()
	_, err := b.LoadKey(ctx, "expired", "")

	if err != ErrWCSessionExpired {
		t.Errorf("LoadKey() error = %v, want %v", err, ErrWCSessionExpired)
	}
}

func TestWalletConnectBackend_LoadKeyValidSession(t *testing.T) {
	b := NewWalletConnectBackend()
	b.dataDir = t.TempDir()

	expectedAddr := "0x1234567890123456789012345678901234567890"

	// Create valid session
	b.mu.Lock()
	b.sessions["valid"] = &wcSession{
		Topic:     "test-topic",
		Address:   expectedAddr,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	b.mu.Unlock()

	ctx := context.Background()
	keySet, err := b.LoadKey(ctx, "valid", "")

	if err != nil {
		t.Fatalf("LoadKey() failed: %v", err)
	}
	if keySet.Name != "valid" {
		t.Errorf("LoadKey() keySet.Name = %s, want valid", keySet.Name)
	}
	if keySet.ECAddress != expectedAddr {
		t.Errorf("LoadKey() keySet.ECAddress = %s, want %s", keySet.ECAddress, expectedAddr)
	}
}

func TestWalletConnectBackend_DeleteKey(t *testing.T) {
	b := NewWalletConnectBackend()
	b.dataDir = t.TempDir()

	// Create session
	b.mu.Lock()
	b.sessions["to-delete"] = &wcSession{
		Topic:   "test-topic",
		SymKey:  []byte("12345678901234567890123456789012"),
		Address: "0x1234",
	}
	b.mu.Unlock()

	ctx := context.Background()
	if err := b.DeleteKey(ctx, "to-delete"); err != nil {
		t.Fatalf("DeleteKey() failed: %v", err)
	}

	// Verify session removed
	b.mu.RLock()
	_, ok := b.sessions["to-delete"]
	b.mu.RUnlock()

	if ok {
		t.Error("DeleteKey() did not remove session")
	}
}

func TestWalletConnectBackend_ListKeys(t *testing.T) {
	b := NewWalletConnectBackend()
	b.dataDir = t.TempDir()

	// Create some sessions
	now := time.Now()
	b.mu.Lock()
	b.sessions["wallet1"] = &wcSession{
		Address:   "0x1111111111111111111111111111111111111111",
		PairedAt:  now,
		ExpiresAt: now.Add(time.Hour),
	}
	b.sessions["wallet2"] = &wcSession{
		Address:   "0x2222222222222222222222222222222222222222",
		PairedAt:  now,
		ExpiresAt: now.Add(-time.Hour), // Expired
	}
	b.mu.Unlock()

	ctx := context.Background()
	keys, err := b.ListKeys(ctx)
	if err != nil {
		t.Fatalf("ListKeys() failed: %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("ListKeys() returned %d keys, want 2", len(keys))
	}

	// Find wallet1 and check properties
	var found1, found2 bool
	for _, k := range keys {
		if k.Name == "wallet1" {
			found1 = true
			if k.Locked {
				t.Error("wallet1 should not be locked")
			}
		}
		if k.Name == "wallet2" {
			found2 = true
			if !k.Locked {
				t.Error("wallet2 should be locked (expired)")
			}
		}
	}

	if !found1 || !found2 {
		t.Error("ListKeys() missing expected wallets")
	}
}

func TestWalletConnectBackend_IsLocked(t *testing.T) {
	b := NewWalletConnectBackend()
	b.dataDir = t.TempDir()

	// Non-existent should be locked
	if !b.IsLocked("nonexistent") {
		t.Error("IsLocked() should return true for non-existent session")
	}

	// Valid session should not be locked
	b.mu.Lock()
	b.sessions["valid"] = &wcSession{
		ExpiresAt: time.Now().Add(time.Hour),
	}
	b.mu.Unlock()

	if b.IsLocked("valid") {
		t.Error("IsLocked() should return false for valid session")
	}

	// Expired session should be locked
	b.mu.Lock()
	b.sessions["expired"] = &wcSession{
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	b.mu.Unlock()

	if !b.IsLocked("expired") {
		t.Error("IsLocked() should return true for expired session")
	}
}

func TestWalletConnectBackend_UnlockNotPaired(t *testing.T) {
	b := NewWalletConnectBackend()
	b.dataDir = t.TempDir()

	ctx := context.Background()
	err := b.Unlock(ctx, "nonexistent", "")

	if err != ErrWCNotPaired {
		t.Errorf("Unlock() error = %v, want %v", err, ErrWCNotPaired)
	}
}

func TestWalletConnectBackend_SessionPersistence(t *testing.T) {
	tmpDir := t.TempDir()

	// Create backend and add session
	b1 := NewWalletConnectBackend()
	b1.dataDir = tmpDir

	b1.mu.Lock()
	b1.sessions["persist-test"] = &wcSession{
		Topic:     "test-topic-hex",
		SymKey:    []byte("12345678901234567890123456789012"),
		Address:   "0xtest",
		ChainID:   1,
		PairedAt:  time.Now(),
		ExpiresAt: time.Now().Add(wcSessionExpiry),
		PeerName:  "TestWallet",
	}
	b1.mu.Unlock()

	// Save sessions
	if err := b1.saveSessions(); err != nil {
		t.Fatalf("saveSessions() failed: %v", err)
	}

	// Create new backend and load sessions
	b2 := NewWalletConnectBackend()
	b2.dataDir = tmpDir

	if err := b2.loadSessions(); err != nil {
		t.Fatalf("loadSessions() failed: %v", err)
	}

	b2.mu.RLock()
	session, ok := b2.sessions["persist-test"]
	b2.mu.RUnlock()

	if !ok {
		t.Fatal("loadSessions() did not restore session")
	}
	if session.Topic != "test-topic-hex" {
		t.Errorf("Restored session Topic = %s, want test-topic-hex", session.Topic)
	}
	if session.Address != "0xtest" {
		t.Errorf("Restored session Address = %s, want 0xtest", session.Address)
	}
	if session.PeerName != "TestWallet" {
		t.Errorf("Restored session PeerName = %s, want TestWallet", session.PeerName)
	}
}

func TestWalletConnectBackend_GetSessionChecksum(t *testing.T) {
	b := NewWalletConnectBackend()
	b.dataDir = t.TempDir()

	// Not paired should fail
	_, err := b.GetSessionChecksum("nonexistent")
	if err != ErrWCNotPaired {
		t.Errorf("GetSessionChecksum() error = %v, want %v", err, ErrWCNotPaired)
	}

	// Add session
	b.mu.Lock()
	b.sessions["checksum-test"] = &wcSession{
		Topic:   "test-topic",
		Address: "0x1234",
	}
	b.mu.Unlock()

	checksum, err := b.GetSessionChecksum("checksum-test")
	if err != nil {
		t.Fatalf("GetSessionChecksum() failed: %v", err)
	}

	// Checksum should be 16 hex chars (8 bytes)
	if len(checksum) != 16 {
		t.Errorf("GetSessionChecksum() length = %d, want 16", len(checksum))
	}

	// Should be valid hex
	if _, err := hex.DecodeString(checksum); err != nil {
		t.Errorf("GetSessionChecksum() not valid hex: %v", err)
	}
}

func TestWalletConnectBackend_Close(t *testing.T) {
	b := NewWalletConnectBackend()
	b.dataDir = t.TempDir()

	// Add session with key material
	symKey := make([]byte, 32)
	copy(symKey, "12345678901234567890123456789012")

	b.mu.Lock()
	b.sessions["close-test"] = &wcSession{
		SymKey: symKey,
	}
	b.mu.Unlock()

	// Close should zero out keys
	if err := b.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	// Check key was zeroed (session should still exist but key zeroed)
	b.mu.RLock()
	session := b.sessions["close-test"]
	b.mu.RUnlock()

	if session != nil {
		for i, v := range session.SymKey {
			if v != 0 {
				t.Errorf("Close() did not zero SymKey at index %d", i)
				break
			}
		}
	}
}

func TestWalletConnectBackend_Registration(t *testing.T) {
	// Register backend for this test (registry may have been reset by other tests)
	b := NewWalletConnectBackend()
	RegisterBackend(b)

	backend, err := GetBackend(BackendWalletConnect)
	if err != nil {
		t.Fatalf("WalletConnect backend not registered: %v", err)
	}

	if backend.Type() != BackendWalletConnect {
		t.Errorf("GetBackend() returned wrong type: %v", backend.Type())
	}

	if backend.Name() != "WalletConnect (Mobile Signing)" {
		t.Errorf("GetBackend() returned wrong name: %v", backend.Name())
	}
}

func TestWCErrors(t *testing.T) {
	// Verify all errors are properly defined
	errors := []error{
		ErrWCNotPaired,
		ErrWCSessionExpired,
		ErrWCUserRejected,
		ErrWCTimeout,
		ErrWCDisconnected,
		ErrWCNoProjectID,
		ErrWCInvalidResponse,
	}

	for _, err := range errors {
		if err == nil {
			t.Error("WC error should not be nil")
		}
		if err.Error() == "" {
			t.Error("WC error message should not be empty")
		}
	}
}
