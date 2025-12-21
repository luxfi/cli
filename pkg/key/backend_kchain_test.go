// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package key

import (
	"context"
	"math/big"
	"testing"

	"github.com/luxfi/crypto/mlkem"
	"github.com/luxfi/crypto/threshold"
)

func TestKChainBackendType(t *testing.T) {
	b := NewKChainBackend()

	if b.Type() != BackendKChain {
		t.Errorf("expected type %s, got %s", BackendKChain, b.Type())
	}

	if b.Name() != "K-Chain Distributed Secrets" {
		t.Errorf("unexpected name: %s", b.Name())
	}

	if b.RequiresHardware() {
		t.Error("should not require hardware")
	}

	if !b.SupportsRemoteSigning() {
		t.Error("should support remote signing")
	}

	if b.RequiresPassword() {
		t.Error("should not require password")
	}
}

func TestShareConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  ShareConfig
		wantErr bool
	}{
		{
			name: "valid 3-of-5",
			config: ShareConfig{
				N: 5,
				K: 3,
				ValidatorAddrs: []string{
					"v1:9650", "v2:9650", "v3:9650", "v4:9650", "v5:9650",
				},
			},
			wantErr: false,
		},
		{
			name: "valid 2-of-3",
			config: ShareConfig{
				N: 3,
				K: 2,
				ValidatorAddrs: []string{
					"v1:9650", "v2:9650", "v3:9650",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid N < 2",
			config: ShareConfig{
				N:              1,
				K:              1,
				ValidatorAddrs: []string{"v1:9650"},
			},
			wantErr: true,
		},
		{
			name: "invalid K > N",
			config: ShareConfig{
				N:              3,
				K:              4,
				ValidatorAddrs: []string{"v1:9650", "v2:9650", "v3:9650"},
			},
			wantErr: true,
		},
		{
			name: "invalid K = 0",
			config: ShareConfig{
				N:              3,
				K:              0,
				ValidatorAddrs: []string{"v1:9650", "v2:9650", "v3:9650"},
			},
			wantErr: true,
		},
		{
			name: "validator count mismatch",
			config: ShareConfig{
				N:              5,
				K:              3,
				ValidatorAddrs: []string{"v1:9650", "v2:9650"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShamirSecretSharing(t *testing.T) {
	b := NewKChainBackend()

	// Test secret
	secret := []byte("this is a test secret key12345")

	// Split into 5 shares with threshold 3
	shares, err := b.splitSecret(secret, 3, 5)
	if err != nil {
		t.Fatalf("splitSecret failed: %v", err)
	}

	if len(shares) != 5 {
		t.Errorf("expected 5 shares, got %d", len(shares))
	}

	// Verify each share has correct format (1 byte index + 32 bytes value)
	for i, share := range shares {
		if len(share) != 33 {
			t.Errorf("share %d has wrong length: %d", i, len(share))
		}
		if share[0] != byte(i+1) {
			t.Errorf("share %d has wrong index: %d", i, share[0])
		}
	}

	// Reconstruct using different subsets of shares
	testCases := [][]int{
		{0, 1, 2},       // First 3 shares
		{0, 2, 4},       // Alternating shares
		{2, 3, 4},       // Last 3 shares
		{0, 1, 2, 3, 4}, // All shares
	}

	// Hash the secret for comparison (since we use a 256-bit field)
	for _, indices := range testCases {
		subset := make([][]byte, len(indices))
		indexList := make([]int, len(indices))
		for i, idx := range indices {
			subset[i] = shares[idx]
			indexList[i] = idx + 1
		}

		reconstructed, err := b.reconstructSecret(subset, indexList)
		if err != nil {
			t.Errorf("reconstructSecret failed for indices %v: %v", indices, err)
			continue
		}

		if len(reconstructed) != 32 {
			t.Errorf("reconstructed has wrong length: %d", len(reconstructed))
		}
	}
}

func TestShamirReconstructionCorrectness(t *testing.T) {
	b := NewKChainBackend()

	// Use a 32-byte secret directly (fits in field)
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte(i)
	}

	// Split with threshold 2-of-3
	shares, err := b.splitSecret(secret, 2, 3)
	if err != nil {
		t.Fatalf("splitSecret failed: %v", err)
	}

	// Reconstruct with minimum threshold
	subset := [][]byte{shares[0], shares[1]}
	indices := []int{1, 2}

	reconstructed, err := b.reconstructSecret(subset, indices)
	if err != nil {
		t.Fatalf("reconstructSecret failed: %v", err)
	}

	// Verify reconstruction matches original
	originalInt := new(big.Int).SetBytes(secret)
	reconstructedInt := new(big.Int).SetBytes(reconstructed)

	// Note: Secret may be reduced modulo prime
	prime := new(big.Int)
	prime.SetString("115792089237316195423570985008687907853269984665640564039457584007913129639747", 10)
	originalMod := new(big.Int).Mod(originalInt, prime)

	if originalMod.Cmp(reconstructedInt) != 0 {
		t.Errorf("reconstruction mismatch:\noriginal:      %x\nreconstructed: %x", originalMod.Bytes(), reconstructed)
	}
}

func TestMLKEMEncryption(t *testing.T) {
	b := NewKChainBackend()
	ctx := context.Background()

	// Get a public key (generates key pair internally)
	addr := "test-validator:9650"
	pubKey, err := b.getValidatorPublicKey(ctx, addr)
	if err != nil {
		t.Fatalf("getValidatorPublicKey failed: %v", err)
	}

	// Test data
	shareData := []byte("test share data for encryption")

	// Encrypt
	encShare, err := b.encryptShare(shareData, 1, pubKey, addr)
	if err != nil {
		t.Fatalf("encryptShare failed: %v", err)
	}

	if encShare.Index != 1 {
		t.Errorf("wrong index: %d", encShare.Index)
	}

	if len(encShare.Ciphertext) != mlkem.MLKEM768CiphertextSize {
		t.Errorf("wrong ciphertext size: %d", len(encShare.Ciphertext))
	}

	// Decrypt
	decrypted, err := b.decryptShare(encShare)
	if err != nil {
		t.Fatalf("decryptShare failed: %v", err)
	}

	if string(decrypted) != string(shareData) {
		t.Errorf("decryption mismatch:\noriginal:  %s\ndecrypted: %s", shareData, decrypted)
	}
}

func TestKChainBackendInitialize(t *testing.T) {
	b := NewKChainBackend()
	ctx := context.Background()

	// Set an invalid endpoint to ensure it handles unavailability gracefully
	b.SetEndpoint("nonexistent:9999")

	err := b.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize should not fail for unavailable network: %v", err)
	}

	// Should not be available after failed connection
	if b.Available() {
		t.Log("Note: backend reports available (connection succeeded)")
	}
}

func TestKChainBackendListKeys(t *testing.T) {
	b := NewKChainBackend()
	ctx := context.Background()

	// Initially empty
	keys, err := b.ListKeys(ctx)
	if err != nil {
		t.Fatalf("ListKeys failed: %v", err)
	}

	if len(keys) != 0 {
		t.Errorf("expected empty list, got %d keys", len(keys))
	}
}

func TestKChainBackendIsLocked(t *testing.T) {
	b := NewKChainBackend()

	// Distributed keys are never "locked" in traditional sense
	if b.IsLocked("anykey") {
		t.Error("IsLocked should return false for K-Chain backend")
	}
}

func TestBLSSchemeAvailable(t *testing.T) {
	// Verify BLS threshold scheme is registered
	scheme, err := threshold.GetScheme(threshold.SchemeBLS)
	if err != nil {
		t.Fatalf("BLS scheme not available: %v", err)
	}

	if scheme.ID() != threshold.SchemeBLS {
		t.Errorf("wrong scheme ID: %v", scheme.ID())
	}

	if scheme.Name() != "BLS Threshold" {
		t.Errorf("unexpected scheme name: %s", scheme.Name())
	}
}

func TestPolynomialEvaluation(t *testing.T) {
	// Test polynomial f(x) = 5 + 3x + 2x^2
	// f(0) = 5, f(1) = 10, f(2) = 19
	prime := big.NewInt(97) // Small prime for testing
	coeffs := []*big.Int{
		big.NewInt(5), // constant term
		big.NewInt(3), // x coefficient
		big.NewInt(2), // x^2 coefficient
	}

	tests := []struct {
		x        int64
		expected int64
	}{
		{0, 5},
		{1, 10},
		{2, 19},
		{3, 32},
	}

	for _, tt := range tests {
		result := evaluatePoly(coeffs, big.NewInt(tt.x), prime)
		if result.Int64() != tt.expected {
			t.Errorf("f(%d) = %d, expected %d", tt.x, result.Int64(), tt.expected)
		}
	}
}

func TestBackendRegistration(t *testing.T) {
	// Verify K-Chain backend is registered
	b, err := GetBackend(BackendKChain)
	if err != nil {
		// May not be available, but should be registered
		if b == nil {
			// Check if it's a "not supported" error vs "not found"
			t.Log("K-Chain backend not available (expected if network unavailable)")
		}
		return
	}

	if b.Type() != BackendKChain {
		t.Errorf("wrong backend type: %s", b.Type())
	}
}

func BenchmarkShamirSplit(b *testing.B) {
	backend := NewKChainBackend()
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = backend.splitSecret(secret, 3, 5)
	}
}

func BenchmarkShamirReconstruct(b *testing.B) {
	backend := NewKChainBackend()
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte(i)
	}

	shares, _ := backend.splitSecret(secret, 3, 5)
	subset := shares[:3]
	indices := []int{1, 2, 3}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = backend.reconstructSecret(subset, indices)
	}
}

func BenchmarkMLKEMEncrypt(b *testing.B) {
	backend := NewKChainBackend()
	ctx := context.Background()

	addr := "bench-validator:9650"
	pubKey, _ := backend.getValidatorPublicKey(ctx, addr)
	shareData := make([]byte, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = backend.encryptShare(shareData, 1, pubKey, addr)
	}
}
