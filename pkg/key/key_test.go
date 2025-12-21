// Copyright (C) 2019-2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package key

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/node/utils/cb58"
)

const (
	// Test private key - NOT for production use
	testPrivateKey    = "PrivateKey-2kqWNDaqUKQyE4ZsV5GLCGeizE6sHAJVyjnfjXoXrtcZpK9M67"
	testRawPrivateKey = "2kqWNDaqUKQyE4ZsV5GLCGeizE6sHAJVyjnfjXoXrtcZpK9M67"
	testPChainAddr    = "P-custom18jma8ppw3nhx5r4ap8clazz0dps7rv5u9xde7p"
	fallbackNetworkID = 999999 // unaffiliated networkID should trigger HRP Fallback
)

func TestNewKeyGenerated(t *testing.T) {
	t.Parallel()

	// Generate a new key for testing
	m, err := NewSoft(fallbackNetworkID)
	if err != nil {
		t.Fatal(err)
	}

	// Should have at least one P-Chain address
	if len(m.P()) == 0 {
		t.Fatal("expected at least one P-Chain address")
	}

	keyPath := filepath.Join(t.TempDir(), "key.pk")
	if err := m.Save(keyPath); err != nil {
		t.Fatal(err)
	}

	m2, err := LoadSoft(fallbackNetworkID, keyPath)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(m.Raw(), m2.Raw()) {
		t.Fatalf("loaded key unexpected %v, expected %v", m2.Raw(), m.Raw())
	}
}

func TestNewKeyWithOptions(t *testing.T) {
	t.Parallel()

	// Generate first key
	privKey1, err := secp256k1.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}

	privKey2, err := secp256k1.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}

	// Encode privKey1 to cb58
	encoded, err := cb58.Encode(privKey1.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	encodedWithPrefix := "PrivateKey-" + encoded

	tt := []struct {
		name   string
		opts   []SOpOption
		expErr error
	}{
		{
			name:   "test no opts",
			opts:   nil,
			expErr: nil,
		},
		{
			name: "with WithPrivateKey",
			opts: []SOpOption{
				WithPrivateKey(privKey1),
			},
			expErr: nil,
		},
		{
			name: "with WithPrivateKeyEncoded",
			opts: []SOpOption{
				WithPrivateKeyEncoded(encodedWithPrefix),
			},
			expErr: nil,
		},
		{
			name: "with WithPrivateKey and WithPrivateKeyEncoded matching",
			opts: []SOpOption{
				WithPrivateKey(privKey1),
				WithPrivateKeyEncoded(encodedWithPrefix),
			},
			expErr: nil,
		},
		{
			name: "with invalid mismatched keys",
			opts: []SOpOption{
				WithPrivateKey(privKey2),
				WithPrivateKeyEncoded(encodedWithPrefix),
			},
			expErr: ErrInvalidPrivateKey,
		},
	}
	for i, tv := range tt {
		_, err := NewSoft(fallbackNetworkID, tv.opts...)
		if !errors.Is(err, tv.expErr) {
			t.Fatalf("#%d(%s): unexpected error %v, expected %v", i, tv.name, err, tv.expErr)
		}
	}
}

func TestPrivateKeyEncoding(t *testing.T) {
	t.Parallel()

	// Generate a new private key
	privKey, err := secp256k1.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}

	// Encode to cb58
	encoded, err := cb58.Encode(privKey.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	// Decode back
	decoded, err := cb58.Decode(encoded)
	if err != nil {
		t.Fatal(err)
	}

	// Convert back to private key
	recoveredKey, err := secp256k1.ToPrivateKey(decoded)
	if err != nil {
		t.Fatal(err)
	}

	// Verify keys match
	if !bytes.Equal(privKey.Bytes(), recoveredKey.Bytes()) {
		t.Fatal("recovered key does not match original")
	}
}
