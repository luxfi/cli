// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

//go:build linux

package key

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/luxfi/crypto/secp256k1"
)

// SecretServiceBackend uses Linux Secret Service API (GNOME Keyring, KWallet)
type SecretServiceBackend struct {
	dataDir   string
	sessions  map[string]*keySession
	sessionMu sync.RWMutex
	tool      string // "secret-tool" or "kwallet-query"
}

// NewSecretServiceBackend creates a Linux Secret Service backend
func NewSecretServiceBackend() *SecretServiceBackend {
	return &SecretServiceBackend{
		sessions: make(map[string]*keySession),
	}
}

func (b *SecretServiceBackend) Type() BackendType {
	return BackendSecretService
}

func (b *SecretServiceBackend) Name() string {
	return "Linux Secret Service (GNOME Keyring/KWallet)"
}

func (b *SecretServiceBackend) Available() bool {
	// Check for secret-tool (GNOME) or kwallet-query (KDE)
	if _, err := exec.LookPath("secret-tool"); err == nil {
		b.tool = "secret-tool"
		return true
	}
	if _, err := exec.LookPath("kwallet-query"); err == nil {
		b.tool = "kwallet-query"
		return true
	}
	return false
}

func (b *SecretServiceBackend) RequiresPassword() bool {
	return false // Uses system keyring
}

func (b *SecretServiceBackend) RequiresHardware() bool {
	return false
}

func (b *SecretServiceBackend) SupportsRemoteSigning() bool {
	return false
}

func (b *SecretServiceBackend) Initialize(ctx context.Context) error {
	if b.dataDir == "" {
		keysDir, err := GetKeysDir()
		if err != nil {
			return err
		}
		b.dataDir = keysDir
	}
	return os.MkdirAll(b.dataDir, 0o700)
}

func (b *SecretServiceBackend) Close() error {
	b.sessionMu.Lock()
	defer b.sessionMu.Unlock()

	for _, s := range b.sessions {
		for i := range s.key {
			s.key[i] = 0
		}
	}
	b.sessions = make(map[string]*keySession)
	return nil
}

func (b *SecretServiceBackend) CreateKey(ctx context.Context, name string, opts CreateKeyOptions) (*HDKeySet, error) {
	keyDir := filepath.Join(b.dataDir, name)

	if _, err := os.Stat(keyDir); err == nil {
		return nil, ErrKeyExists
	}

	var mnemonic string
	if opts.Mnemonic != "" {
		if !ValidateMnemonic(opts.Mnemonic) {
			return nil, errors.New("invalid mnemonic phrase")
		}
		mnemonic = opts.Mnemonic
	} else {
		var err error
		mnemonic, err = GenerateMnemonic()
		if err != nil {
			return nil, fmt.Errorf("failed to generate mnemonic: %w", err)
		}
	}

	keySet, err := DeriveAllKeys(name, mnemonic)
	if err != nil {
		return nil, fmt.Errorf("failed to derive keys: %w", err)
	}

	if err := b.SaveKey(ctx, keySet, ""); err != nil {
		return nil, err
	}

	return keySet, nil
}

func (b *SecretServiceBackend) LoadKey(ctx context.Context, name, password string) (*HDKeySet, error) {
	// Check session cache
	b.sessionMu.RLock()
	if s, ok := b.sessions[name]; ok && time.Now().Before(s.expiresAt) {
		b.sessionMu.RUnlock()
		return parseKeySetJSON(append([]byte{}, s.key...))
	}
	b.sessionMu.RUnlock()

	// Read from secret service
	data, err := b.readFromSecretService(name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "No matching") {
			return nil, ErrKeyNotFound
		}
		return nil, fmt.Errorf("secret service read failed: %w", err)
	}

	keySet, err := parseKeySetJSON(data)
	if err != nil {
		return nil, err
	}

	// Cache in session
	b.sessionMu.Lock()
	b.sessions[name] = &keySession{
		name:       name,
		key:        data,
		unlockedAt: time.Now(),
		expiresAt:  time.Now().Add(sessionTimeout),
	}
	b.sessionMu.Unlock()

	return keySet, nil
}

func (b *SecretServiceBackend) SaveKey(ctx context.Context, keySet *HDKeySet, password string) error {
	keyDir := filepath.Join(b.dataDir, keySet.Name)
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	data, err := serializeKeySet(keySet)
	if err != nil {
		return fmt.Errorf("failed to serialize keys: %w", err)
	}

	if err := b.writeToSecretService(keySet.Name, data); err != nil {
		return fmt.Errorf("secret service write failed: %w", err)
	}

	// Write public info
	pubInfo := map[string]interface{}{
		"name":       keySet.Name,
		"ec_address": keySet.ECAddress,
		"node_id":    keySet.NodeID,
		"created_at": time.Now().Format(time.RFC3339),
		"backend":    string(BackendSecretService),
	}
	pubData, _ := json.MarshalIndent(pubInfo, "", "  ")
	_ = os.WriteFile(filepath.Join(keyDir, "info.json"), pubData, 0o644)

	return nil
}

func (b *SecretServiceBackend) DeleteKey(ctx context.Context, name string) error {
	if err := b.deleteFromSecretService(name); err != nil {
		if !strings.Contains(err.Error(), "not found") {
			return err
		}
	}

	b.sessionMu.Lock()
	if s, ok := b.sessions[name]; ok {
		for i := range s.key {
			s.key[i] = 0
		}
		delete(b.sessions, name)
	}
	b.sessionMu.Unlock()

	keyDir := filepath.Join(b.dataDir, name)
	return os.RemoveAll(keyDir)
}

func (b *SecretServiceBackend) ListKeys(ctx context.Context) ([]KeyInfo, error) {
	entries, err := os.ReadDir(b.dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []KeyInfo{}, nil
		}
		return nil, err
	}

	var keys []KeyInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		keyDir := filepath.Join(b.dataDir, name)

		info := KeyInfo{
			Name:      name,
			Encrypted: true,
			Locked:    b.IsLocked(name),
		}

		pubPath := filepath.Join(keyDir, "info.json")
		if data, err := os.ReadFile(pubPath); err == nil {
			var pubInfo struct {
				ECAddress string `json:"ec_address"`
				NodeID    string `json:"node_id"`
				CreatedAt string `json:"created_at"`
				Backend   string `json:"backend"`
			}
			if json.Unmarshal(data, &pubInfo) == nil {
				if pubInfo.Backend != string(BackendSecretService) {
					continue
				}
				info.Address = pubInfo.ECAddress
				info.NodeID = pubInfo.NodeID
				if t, err := time.Parse(time.RFC3339, pubInfo.CreatedAt); err == nil {
					info.CreatedAt = t
				}
			}
		}

		keys = append(keys, info)
	}

	return keys, nil
}

func (b *SecretServiceBackend) Lock(ctx context.Context, name string) error {
	b.sessionMu.Lock()
	defer b.sessionMu.Unlock()

	if s, ok := b.sessions[name]; ok {
		for i := range s.key {
			s.key[i] = 0
		}
		delete(b.sessions, name)
	}
	return nil
}

func (b *SecretServiceBackend) Unlock(ctx context.Context, name, password string) error {
	_, err := b.LoadKey(ctx, name, password)
	return err
}

func (b *SecretServiceBackend) IsLocked(name string) bool {
	b.sessionMu.RLock()
	defer b.sessionMu.RUnlock()

	s, ok := b.sessions[name]
	if !ok {
		return true
	}
	return time.Now().After(s.expiresAt)
}

func (b *SecretServiceBackend) Sign(ctx context.Context, name string, request SignRequest) (*SignResponse, error) {
	keySet, err := b.LoadKey(ctx, name, "")
	if err != nil {
		return nil, err
	}

	privKey, err := secp256k1.ToPrivateKey(keySet.ECPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	sig, err := privKey.Sign(request.DataHash[:])
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}

	return &SignResponse{
		Signature: sig,
		PublicKey: privKey.PublicKey().Bytes(),
		Address:   keySet.ECAddress,
	}, nil
}

// Secret Service operations

func (b *SecretServiceBackend) writeToSecretService(name string, data []byte) error {
	if b.tool == "secret-tool" {
		return b.writeWithSecretTool(name, data)
	}
	return b.writeWithKWallet(name, data)
}

func (b *SecretServiceBackend) readFromSecretService(name string) ([]byte, error) {
	if b.tool == "secret-tool" {
		return b.readWithSecretTool(name)
	}
	return b.readWithKWallet(name)
}

func (b *SecretServiceBackend) deleteFromSecretService(name string) error {
	if b.tool == "secret-tool" {
		return b.deleteWithSecretTool(name)
	}
	return b.deleteWithKWallet(name)
}

// GNOME secret-tool implementation

func (b *SecretServiceBackend) writeWithSecretTool(name string, data []byte) error {
	cmd := exec.Command("secret-tool", "store",
		"--label", fmt.Sprintf("Lux Key: %s", name),
		"application", "lux-cli",
		"key", name,
	)
	cmd.Stdin = strings.NewReader(hex.EncodeToString(data))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("secret-tool store failed: %s: %w", string(output), err)
	}
	return nil
}

func (b *SecretServiceBackend) readWithSecretTool(name string) ([]byte, error) {
	cmd := exec.Command("secret-tool", "lookup",
		"application", "lux-cli",
		"key", name,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("secret-tool lookup failed: %w", err)
	}

	return hex.DecodeString(strings.TrimSpace(string(output)))
}

func (b *SecretServiceBackend) deleteWithSecretTool(name string) error {
	cmd := exec.Command("secret-tool", "clear",
		"application", "lux-cli",
		"key", name,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("secret-tool clear failed: %s: %w", string(output), err)
	}
	return nil
}

// KDE KWallet implementation

func (b *SecretServiceBackend) writeWithKWallet(name string, data []byte) error {
	// KWallet uses kwalletcli or qdbus
	cmd := exec.Command("kwalletcli", "-f", "lux-cli", "-e", name, "-P")
	cmd.Stdin = strings.NewReader(hex.EncodeToString(data))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kwalletcli write failed: %s: %w", string(output), err)
	}
	return nil
}

func (b *SecretServiceBackend) readWithKWallet(name string) ([]byte, error) {
	cmd := exec.Command("kwalletcli", "-f", "lux-cli", "-e", name)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("kwalletcli read failed: %w", err)
	}

	return hex.DecodeString(strings.TrimSpace(string(output)))
}

func (b *SecretServiceBackend) deleteWithKWallet(name string) error {
	cmd := exec.Command("kwalletcli", "-f", "lux-cli", "-e", name, "-d")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kwalletcli delete failed: %s: %w", string(output), err)
	}
	return nil
}

func (b *SecretServiceBackend) GetKeyChecksum(name string) (string, error) {
	ks, err := b.LoadKey(context.Background(), name, "")
	if err != nil {
		return "", err
	}

	h := sha256.New()
	h.Write(ks.ECPrivateKey)
	h.Write(ks.BLSPrivateKey)
	return hex.EncodeToString(h.Sum(nil)[:8]), nil
}

func init() {
	RegisterBackend(NewSecretServiceBackend())
}
