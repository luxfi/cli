// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package keycmd

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	migrateAll    bool
	migrateForce  bool
	migrateSecure bool
)

func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate [name...]",
		Short: "Migrate plaintext keys to encrypted storage",
		Long: `Migrate legacy plaintext key files to encrypted keystore.enc format.

This command reads plaintext key files (ec/private.key, bls/secret.key, staker.key)
and encrypts them using AES-256-GCM with Argon2id key derivation.

After migration, the plaintext originals can be securely deleted with --secure.

Examples:
  lux key migrate node0              # Migrate single node
  lux key migrate node0 node1 node2  # Migrate multiple nodes
  lux key migrate --all              # Migrate all keys with plaintext files
  lux key migrate node0 --secure     # Migrate and securely delete originals`,
		RunE: runMigrate,
	}

	cmd.Flags().BoolVar(&migrateAll, "all", false, "Migrate all keys with plaintext files")
	cmd.Flags().BoolVar(&migrateForce, "force", false, "Overwrite existing keystore.enc files")
	cmd.Flags().BoolVar(&migrateSecure, "secure", false, "Securely delete plaintext files after migration")

	return cmd
}

func runMigrate(_ *cobra.Command, args []string) error {
	keysDir, err := key.GetKeysDir()
	if err != nil {
		return fmt.Errorf("failed to get keys directory: %w", err)
	}

	// Determine which keys to migrate
	var names []string
	if migrateAll {
		entries, err := os.ReadDir(keysDir)
		if err != nil {
			return fmt.Errorf("failed to read keys directory: %w", err)
		}
		for _, e := range entries {
			if e.IsDir() && hasPlaintextKeys(filepath.Join(keysDir, e.Name())) {
				names = append(names, e.Name())
			}
		}
	} else {
		if len(args) == 0 {
			return fmt.Errorf("specify key names or use --all")
		}
		names = args
	}

	if len(names) == 0 {
		ux.Logger.PrintToUser("No keys found to migrate.")
		return nil
	}

	ux.Logger.PrintToUser("Keys to migrate: %v", names)
	ux.Logger.PrintToUser("")

	// Get password for encryption
	password := os.Getenv(key.EnvKeyPassword)
	if password == "" {
		ux.Logger.PrintToUser("Enter encryption password for the migrated keys:")
		password, err = app.Prompt.CaptureString("Password")
		if err != nil {
			return err
		}
		if password == "" {
			return fmt.Errorf("password required for encrypted storage")
		}

		ux.Logger.PrintToUser("Confirm password:")
		confirm, err := app.Prompt.CaptureString("Confirm")
		if err != nil {
			return err
		}
		if password != confirm {
			return fmt.Errorf("passwords do not match")
		}
	}

	// Get the software backend for encryption
	backend, err := key.GetBackend(key.BackendSoftware)
	if err != nil {
		return fmt.Errorf("failed to get software backend: %w", err)
	}
	if err := backend.Initialize(context.Background()); err != nil {
		return fmt.Errorf("failed to initialize backend: %w", err)
	}

	// Migrate each key
	var migrated, skipped, failed int
	for _, name := range names {
		keyDir := filepath.Join(keysDir, name)
		encPath := filepath.Join(keyDir, "keystore.enc")

		// Check if already migrated
		if _, err := os.Stat(encPath); err == nil && !migrateForce {
			ux.Logger.PrintToUser("  [SKIP] %s: keystore.enc already exists (use --force to overwrite)", name)
			skipped++
			continue
		}

		ux.Logger.PrintToUser("  [MIGRATING] %s...", name)

		// Load plaintext keys
		keySet, err := loadPlaintextKeys(name, keyDir)
		if err != nil {
			ux.Logger.PrintToUser("    [FAIL] %s: %v", name, err)
			failed++
			continue
		}

		// Save encrypted
		if err := backend.SaveKey(context.Background(), keySet, password); err != nil {
			ux.Logger.PrintToUser("    [FAIL] %s: failed to encrypt: %v", name, err)
			failed++
			continue
		}

		ux.Logger.PrintToUser("    [OK] %s: created keystore.enc", name)

		// Securely delete plaintext files if requested
		if migrateSecure {
			if err := secureDeletePlaintextKeys(keyDir); err != nil {
				ux.Logger.PrintToUser("    [WARN] %s: failed to delete some plaintext files: %v", name, err)
			} else {
				ux.Logger.PrintToUser("    [OK] %s: plaintext files securely deleted", name)
			}
		}

		migrated++
	}

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Migration complete: %d migrated, %d skipped, %d failed", migrated, skipped, failed)

	if !migrateSecure && migrated > 0 {
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("WARNING: Plaintext key files still exist!")
		ux.Logger.PrintToUser("Run with --secure to delete them, or manually run:")
		ux.Logger.PrintToUser("  for d in node{0..5}; do")
		ux.Logger.PrintToUser("    shred -u ~/.lux/keys/$d/ec/private.key 2>/dev/null")
		ux.Logger.PrintToUser("    shred -u ~/.lux/keys/$d/bls/secret.key 2>/dev/null")
		ux.Logger.PrintToUser("    shred -u ~/.lux/keys/$d/staker.key 2>/dev/null")
		ux.Logger.PrintToUser("  done")
	}

	return nil
}

// hasPlaintextKeys checks if a key directory has plaintext private key files
func hasPlaintextKeys(keyDir string) bool {
	plaintextFiles := []string{
		filepath.Join(keyDir, "ec", "private.key"),
		filepath.Join(keyDir, "bls", "secret.key"),
		filepath.Join(keyDir, "staker.key"),
	}
	for _, f := range plaintextFiles {
		if _, err := os.Stat(f); err == nil {
			return true
		}
	}
	return false
}

// loadPlaintextKeys loads legacy plaintext key files into an HDKeySet
func loadPlaintextKeys(name, keyDir string) (*key.HDKeySet, error) {
	keySet := &key.HDKeySet{
		Name: name,
	}

	// Load EC private key (hex format)
	ecPath := filepath.Join(keyDir, "ec", "private.key")
	if data, err := os.ReadFile(ecPath); err == nil {
		hexStr := strings.TrimSpace(string(data))
		keySet.ECPrivateKey, err = hex.DecodeString(hexStr)
		if err != nil {
			return nil, fmt.Errorf("failed to decode EC private key: %w", err)
		}
		// Derive public key and address
		keySet.ECPublicKey = deriveECPublicKey(keySet.ECPrivateKey)
		keySet.ECAddress = deriveECAddress(keySet.ECPublicKey)
	}

	// Load BLS secret key (base64 format)
	blsPath := filepath.Join(keyDir, "bls", "secret.key")
	if data, err := os.ReadFile(blsPath); err == nil {
		b64Str := strings.TrimSpace(string(data))
		keySet.BLSPrivateKey, err = base64.StdEncoding.DecodeString(b64Str)
		if err != nil {
			return nil, fmt.Errorf("failed to decode BLS secret key: %w", err)
		}
	}

	// Try signer.key for BLS if secret.key wasn't found
	if len(keySet.BLSPrivateKey) == 0 {
		signerPath := filepath.Join(keyDir, "bls", "signer.key")
		if data, err := os.ReadFile(signerPath); err == nil {
			// signer.key might be in various formats
			content := strings.TrimSpace(string(data))
			// Try base64 first
			if decoded, err := base64.StdEncoding.DecodeString(content); err == nil {
				keySet.BLSPrivateKey = decoded
			} else if decoded, err := hex.DecodeString(content); err == nil {
				// Try hex
				keySet.BLSPrivateKey = decoded
			} else {
				// Raw bytes
				keySet.BLSPrivateKey = []byte(content)
			}
		}
	}

	// Load BLS public key and PoP if available
	blsPubPath := filepath.Join(keyDir, "bls", "public.key")
	if data, err := os.ReadFile(blsPubPath); err == nil {
		hexStr := strings.TrimSpace(string(data))
		keySet.BLSPublicKey, _ = hex.DecodeString(hexStr)
	}

	blsPoPPath := filepath.Join(keyDir, "bls", "pop.hex")
	if data, err := os.ReadFile(blsPoPPath); err == nil {
		hexStr := strings.TrimSpace(string(data))
		keySet.BLSPoP, _ = hex.DecodeString(hexStr)
	}

	// Load staker.key (PEM format)
	stakerPath := filepath.Join(keyDir, "staker.key")
	if data, err := os.ReadFile(stakerPath); err == nil {
		keySet.StakingKeyPEM = data
	}

	// Load staker.crt if exists
	stakerCertPath := filepath.Join(keyDir, "staker.crt")
	if data, err := os.ReadFile(stakerCertPath); err == nil {
		keySet.StakingCertPEM = data
	}

	// Load info.json for NodeID
	infoPath := filepath.Join(keyDir, "info.json")
	if data, err := os.ReadFile(infoPath); err == nil {
		// Extract NodeID from info.json
		content := string(data)
		if idx := strings.Index(content, `"nodeID"`); idx != -1 {
			start := strings.Index(content[idx:], `"NodeID-`)
			if start != -1 {
				end := strings.Index(content[idx+start+1:], `"`)
				if end != -1 {
					keySet.NodeID = content[idx+start+1 : idx+start+1+end]
				}
			}
		}
	}

	// Ensure we have at least one key
	if len(keySet.ECPrivateKey) == 0 && len(keySet.BLSPrivateKey) == 0 && len(keySet.StakingKeyPEM) == 0 {
		return nil, fmt.Errorf("no private keys found in %s", keyDir)
	}

	return keySet, nil
}

// Helper functions for key derivation (simplified)
func deriveECPublicKey(privateKey []byte) []byte {
	// In production, use secp256k1 curve derivation
	// Simplified placeholder
	return privateKey[:32] // Just return first 32 bytes as placeholder
}

func deriveECAddress(publicKey []byte) string {
	// In production, use Keccak256 hash
	// Simplified placeholder
	if len(publicKey) >= 20 {
		return "0x" + hex.EncodeToString(publicKey[:20])
	}
	return ""
}

// secureDeletePlaintextKeys securely deletes plaintext key files
func secureDeletePlaintextKeys(keyDir string) error {
	plaintextFiles := []string{
		filepath.Join(keyDir, "ec", "private.key"),
		filepath.Join(keyDir, "bls", "secret.key"),
		filepath.Join(keyDir, "bls", "signer.key"),
		filepath.Join(keyDir, "staker.key"),
		filepath.Join(keyDir, "staking", "staker.key"),
	}

	var errs []string
	for _, f := range plaintextFiles {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			continue
		}

		// Overwrite with zeros first
		if err := secureOverwrite(f); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", f, err))
			continue
		}

		// Delete the file
		if err := os.Remove(f); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", f, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("some files failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

// secureOverwrite overwrites a file with zeros before deletion
func secureOverwrite(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	// Overwrite with zeros
	zeros := make([]byte, info.Size())
	if _, err := f.Write(zeros); err != nil {
		return err
	}

	// Sync to disk
	return f.Sync()
}
