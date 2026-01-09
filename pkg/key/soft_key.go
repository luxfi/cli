// Copyright (C) 2019-2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package key

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/luxfi/address"
	"github.com/luxfi/constantsants"
	"github.com/luxfi/crypto/cb58"
	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/go-bip32"
	"github.com/luxfi/go-bip39"
	"github.com/luxfi/ids"
	"github.com/luxfi/vm/vms/components/lux"
	"github.com/luxfi/vm/vms/platformvm/txs"
	"github.com/luxfi/vm/vms/secp256k1fx"

	eth_crypto "github.com/luxfi/crypto"
	"go.uber.org/zap"
)

var (
	ErrInvalidPrivateKey         = errors.New("invalid private key")
	ErrInvalidPrivateKeyLen      = errors.New("invalid private key length (expect 64 bytes in hex)")
	ErrInvalidPrivateKeyEnding   = errors.New("invalid private key ending")
	ErrInvalidPrivateKeyEncoding = errors.New("invalid private key encoding")
)

// LUXCoinType is the BIP-44 coin type for LUX (9000')
const LUXCoinType = 9000

// deriveMnemonicKey derives a private key from a BIP-39 mnemonic using BIP-44 path.
// Path: m/44'/9000'/0'/0/{accountIndex}
func deriveMnemonicKey(mnemonic string, accountIndex uint32) ([]byte, error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, fmt.Errorf("invalid mnemonic phrase")
	}
	seed := bip39.NewSeed(mnemonic, "")

	// Create master key from seed
	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		return nil, fmt.Errorf("failed to create master key: %w", err)
	}

	// BIP-44 path: m/44'/9000'/0'/0/{accountIndex}
	// m/44' (purpose)
	key, err := masterKey.NewChildKey(bip32.FirstHardenedChild + 44)
	if err != nil {
		return nil, fmt.Errorf("failed to derive purpose: %w", err)
	}

	// m/44'/9000' (coin type for LUX)
	key, err = key.NewChildKey(bip32.FirstHardenedChild + LUXCoinType)
	if err != nil {
		return nil, fmt.Errorf("failed to derive coin type: %w", err)
	}

	// m/44'/9000'/0' (account)
	key, err = key.NewChildKey(bip32.FirstHardenedChild + 0)
	if err != nil {
		return nil, fmt.Errorf("failed to derive account: %w", err)
	}

	// m/44'/9000'/0'/0 (change)
	key, err = key.NewChildKey(0)
	if err != nil {
		return nil, fmt.Errorf("failed to derive change: %w", err)
	}

	// m/44'/9000'/0'/0/{accountIndex} (address index)
	key, err = key.NewChildKey(accountIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to derive address index: %w", err)
	}

	return key.Key, nil
}

var _ Key = &SoftKey{}

type SoftKey struct {
	privKey        *secp256k1.PrivateKey
	privKeyRaw     []byte
	privKeyEncoded string

	pAddr string

	keyChain *secp256k1fx.Keychain
}

const (
	privKeyEncPfx = "PrivateKey-"
	privKeySize   = 64

	// LocalKeyName is the name of the local development key file
	LocalKeyName = "local-key"
	// LocalKeyPath is the path where the local key is stored
	LocalKeyPath = "~/.lux/keys/" + LocalKeyName + ".pk"

	// Environment variables for key configuration are defined in backend_env.go:
	// - EnvMnemonic (LUX_MNEMONIC) - BIP39 mnemonic phrase for deterministic key generation
	// - EnvPrivateKey (LUX_PRIVATE_KEY) - CB58 encoded private key (PrivateKey-xxx format)
)

// GetMnemonicFromEnv returns the mnemonic from LUX_MNEMONIC environment variable.
// Returns empty string if not set or invalid.
func GetMnemonicFromEnv() string {
	mnemonic := os.Getenv(EnvMnemonic)
	if mnemonic == "" {
		return ""
	}
	// Validate the mnemonic
	if !bip39.IsMnemonicValid(mnemonic) {
		return ""
	}
	return mnemonic
}

type SOp struct {
	privKey        *secp256k1.PrivateKey
	privKeyEncoded string
}

type SOpOption func(*SOp)

func (sop *SOp) applyOpts(opts []SOpOption) {
	for _, opt := range opts {
		opt(sop)
	}
}

// To create a new key SoftKey with a pre-loaded private key.
func WithPrivateKey(privKey *secp256k1.PrivateKey) SOpOption {
	return func(sop *SOp) {
		sop.privKey = privKey
	}
}

// To create a new key SoftKey with a pre-defined private key.
func WithPrivateKeyEncoded(privKey string) SOpOption {
	return func(sop *SOp) {
		sop.privKeyEncoded = privKey
	}
}

func NewSoft(networkID uint32, opts ...SOpOption) (*SoftKey, error) {
	ret := &SOp{}
	ret.applyOpts(opts)

	// set via "WithPrivateKeyEncoded"
	if len(ret.privKeyEncoded) > 0 {
		privKey, err := decodePrivateKey(ret.privKeyEncoded)
		if err != nil {
			return nil, err
		}
		// to not overwrite
		if ret.privKey != nil &&
			!bytes.Equal(ret.privKey.Bytes(), privKey.Bytes()) {
			return nil, ErrInvalidPrivateKey
		}
		ret.privKey = privKey
	}

	// generate a new one
	if ret.privKey == nil {
		var err error
		ret.privKey, err = secp256k1.NewPrivateKey()
		if err != nil {
			return nil, err
		}
	}

	privKey := ret.privKey
	privKeyEncoded, err := encodePrivateKey(ret.privKey)
	if err != nil {
		return nil, err
	}

	// double-check encoding is consistent
	if ret.privKeyEncoded != "" &&
		ret.privKeyEncoded != privKeyEncoded {
		return nil, ErrInvalidPrivateKeyEncoding
	}

	keyChain := secp256k1fx.NewKeychain()
	keyChain.Add(privKey)

	m := &SoftKey{
		privKey:        privKey,
		privKeyRaw:     privKey.Bytes(),
		privKeyEncoded: privKeyEncoded,

		keyChain: keyChain,
	}

	// Parse HRP to create valid address
	hrp := GetHRP(networkID)
	m.pAddr, err = address.Format("P", hrp, m.privKey.PublicKey().Address().Bytes())
	if err != nil {
		return nil, err
	}

	return m, nil
}

// LoadSoft loads the private key from disk and creates the corresponding SoftKey.
func LoadSoft(networkID uint32, keyPath string) (*SoftKey, error) {
	kb, err := os.ReadFile(keyPath) //nolint:gosec // G304: Reading user-specified key file
	if err != nil {
		return nil, err
	}

	// in case, it's already encoded
	k, err := NewSoft(networkID, WithPrivateKeyEncoded(string(kb)))
	if err == nil {
		return k, nil
	}

	r := bufio.NewReader(bytes.NewBuffer(kb))
	buf := make([]byte, privKeySize)
	n, err := readASCII(buf, r)
	if err != nil {
		return nil, err
	}
	if n != len(buf) {
		return nil, ErrInvalidPrivateKeyLen
	}
	if err := checkKeyFileEnd(r); err != nil {
		return nil, err
	}

	skBytes, err := hex.DecodeString(string(buf))
	if err != nil {
		return nil, err
	}
	privKey, err := secp256k1.ToPrivateKey(skBytes)
	if err != nil {
		return nil, err
	}

	return NewSoft(networkID, WithPrivateKey(privKey))
}

// readASCII reads into 'buf', stopping when the buffer is full or
// when a non-printable control character is encountered.
func readASCII(buf []byte, r io.ByteReader) (n int, err error) {
	for ; n < len(buf); n++ {
		buf[n], err = r.ReadByte()
		switch {
		case errors.Is(err, io.EOF) || buf[n] < '!':
			return n, nil
		case err != nil:
			return n, err
		}
	}
	return n, nil
}

const fileEndLimit = 1

// checkKeyFileEnd skips over additional newlines at the end of a key file.
func checkKeyFileEnd(r io.ByteReader) error {
	for idx := 0; ; idx++ {
		b, err := r.ReadByte()
		switch {
		case errors.Is(err, io.EOF):
			return nil
		case err != nil:
			return err
		case b != '\n' && b != '\r':
			return ErrInvalidPrivateKeyEnding
		case idx > fileEndLimit:
			return ErrInvalidPrivateKeyLen
		}
	}
}

func encodePrivateKey(pk *secp256k1.PrivateKey) (string, error) {
	privKeyRaw := pk.Bytes()
	enc, err := cb58.Encode(privKeyRaw)
	if err != nil {
		return "", err
	}
	return privKeyEncPfx + enc, nil
}

func decodePrivateKey(enc string) (*secp256k1.PrivateKey, error) {
	rawPk := strings.Replace(enc, privKeyEncPfx, "", 1)
	skBytes, err := cb58.Decode(rawPk)
	if err != nil {
		return nil, err
	}
	privKey, err := secp256k1.ToPrivateKey(skBytes)
	if err != nil {
		return nil, err
	}
	return privKey, nil
}

func (m *SoftKey) C() string {
	// Convert private key bytes to ECDSA format
	privKeyBytes := m.privKey.Bytes()
	ecdsaPrv, err := eth_crypto.ToECDSA(privKeyBytes)
	if err != nil {
		return ""
	}
	pub := ecdsaPrv.PublicKey

	addr := eth_crypto.PubkeyToAddress(pub)
	return addr.String()
}

// X returns the X-Chain addresses (as a slice for compatibility)
func (m *SoftKey) X() []string {
	// Parse HRP for X-Chain
	hrp := GetHRP(1) // Use network ID 1 for mainnet by default
	xAddr, err := address.Format("X", hrp, m.privKey.PublicKey().Address().Bytes())
	if err != nil {
		return []string{}
	}
	return []string{xAddr}
}

// Returns the KeyChain
func (m *SoftKey) KeyChain() *secp256k1fx.Keychain {
	return m.keyChain
}

// PrivateKeyRaw returns the private key in hex format
func (m *SoftKey) PrivateKeyRaw() string {
	return hex.EncodeToString(m.privKeyRaw)
}

// Returns the private key.
func (m *SoftKey) Key() *secp256k1.PrivateKey {
	return m.privKey
}

// Returns the private key in raw bytes.
func (m *SoftKey) Raw() []byte {
	return m.privKeyRaw
}

// Returns the private key encoded in CB58 and "PrivateKey-" prefix.
func (m *SoftKey) Encode() string {
	return m.privKeyEncoded
}

func (m *SoftKey) PrivKeyHex() string {
	return hex.EncodeToString(m.privKeyRaw)
}

// Saves the private key to disk with hex encoding.
func (m *SoftKey) Save(p string) error {
	k := hex.EncodeToString(m.privKeyRaw)
	return os.WriteFile(p, []byte(k), fsModeWrite)
}

func (m *SoftKey) P() []string {
	return []string{m.pAddr}
}

func (m *SoftKey) Spends(outputs []*lux.UTXO, opts ...OpOption) (
	totalBalanceToSpend uint64,
	inputs []*lux.TransferableInput,
	signers [][]ids.ShortID,
) {
	ret := &Op{}
	ret.applyOpts(opts)

	for _, out := range outputs {
		input, psigners, err := m.spend(out, ret.time)
		if err != nil {
			zap.L().Warn("cannot spend with current key", zap.Error(err))
			continue
		}
		totalBalanceToSpend += input.Amount()
		inputs = append(inputs, &lux.TransferableInput{
			UTXOID: out.UTXOID,
			Asset:  out.Asset,
			In:     input,
		})
		// Convert to ids.ShortID to adhere with interface
		pksigners := make([]ids.ShortID, len(psigners))
		for i, psigner := range psigners {
			addr := psigner.PublicKey().Address()
			copy(pksigners[i][:], addr[:])
		}
		signers = append(signers, pksigners)
		if ret.targetAmount > 0 &&
			totalBalanceToSpend > ret.targetAmount+ret.feeDeduct {
			break
		}
	}
	SortTransferableInputsWithSigners(inputs, signers)
	return totalBalanceToSpend, inputs, signers
}

func (m *SoftKey) spend(output *lux.UTXO, time uint64) (
	input lux.TransferableIn,
	signers []*secp256k1.PrivateKey,
	err error,
) {
	// "time" is used to check whether the key owner
	// is still within the lock time (thus can't spend).
	inputf, psigners, err := m.keyChain.Spend(output.Out, time)
	if err != nil {
		return nil, nil, err
	}
	var ok bool
	input, ok = inputf.(lux.TransferableIn)
	if !ok {
		return nil, nil, ErrInvalidType
	}
	return input, psigners, nil
}

const fsModeWrite = 0o600

func (m *SoftKey) Addresses() []ids.ShortID {
	addr := m.privKey.PublicKey().Address()
	shortID := ids.ShortID{}
	copy(shortID[:], addr[:])
	return []ids.ShortID{shortID}
}

func (m *SoftKey) Sign(pTx *txs.Tx, signers [][]ids.ShortID) error {
	privsigners := make([][]*secp256k1.PrivateKey, len(signers))
	for i, inputSigners := range signers {
		privsigners[i] = make([]*secp256k1.PrivateKey, len(inputSigners))
		for j, signer := range inputSigners {
			addr := m.privKey.PublicKey().Address()
			// Compare the underlying bytes
			if !bytes.Equal(signer[:], addr[:]) {
				// Should never happen
				return ErrCantSpend
			}
			privsigners[i][j] = m.privKey
		}
	}

	return pTx.Sign(txs.Codec, privsigners)
}

func (m *SoftKey) Match(owners *secp256k1fx.OutputOwners, time uint64) ([]uint32, []ids.ShortID, bool) {
	indices, privs, ok := m.keyChain.Match(owners, time)
	pks := make([]ids.ShortID, len(privs))
	for i, priv := range privs {
		addr := priv.PublicKey().Address()
		copy(pks[i][:], addr[:])
	}
	return indices, pks, ok
}

// GetLocalKeyPath returns the expanded path to the local key file
func GetLocalKeyPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".lux", "keys", LocalKeyName+".pk")
}

// GetOrCreateLocalKey loads a key with the following priority:
// 1. LUX_PRIVATE_KEY environment variable (CB58 encoded)
// 2. LUX_MNEMONIC environment variable (BIP39 mnemonic)
// 3. Local key file at ~/.lux/keys/local-key.pk (generated if not exists)
// This ensures no hardcoded keys - all keys are either from environment or generated locally.
func GetOrCreateLocalKey(networkID uint32) (*SoftKey, error) {
	// Priority 1: Check for LUX_PRIVATE_KEY environment variable
	if privKeyEnc := os.Getenv(EnvPrivateKey); privKeyEnc != "" {
		return NewSoft(networkID, WithPrivateKeyEncoded(privKeyEnc))
	}

	// Priority 2: Check for LUX_MNEMONIC environment variable
	if mnemonic := os.Getenv(EnvMnemonic); mnemonic != "" {
		return NewSoftFromMnemonic(networkID, mnemonic)
	}

	// Priority 3: Use local key file (generate if not exists)
	keyPath := GetLocalKeyPath()
	if keyPath == "" {
		return nil, errors.New("could not determine home directory")
	}

	// Create the keys directory if it doesn't exist
	keyDir := filepath.Dir(keyPath)
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create key directory: %w", err)
	}

	// Try to load existing key
	if _, err := os.Stat(keyPath); err == nil {
		return LoadSoft(networkID, keyPath)
	}

	// Generate a new key
	newKey, err := NewSoft(networkID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate local key: %w", err)
	}

	// Save the new key
	if err := newKey.Save(keyPath); err != nil {
		return nil, fmt.Errorf("failed to save local key: %w", err)
	}

	return newKey, nil
}

// NewSoftFromMnemonic creates a SoftKey from a BIP39 mnemonic phrase.
// Uses standard BIP44 derivation path: m/44'/9000'/0'/0/0
func NewSoftFromMnemonic(networkID uint32, mnemonic string) (*SoftKey, error) {
	return NewSoftFromMnemonicWithAccount(networkID, mnemonic, 0)
}

// NewSoftFromBytes creates a SoftKey from raw private key bytes.
func NewSoftFromBytes(networkID uint32, privKeyBytes []byte) (*SoftKey, error) {
	privKey, err := secp256k1.ToPrivateKey(privKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create private key from bytes: %w", err)
	}
	return NewSoft(networkID, WithPrivateKey(privKey))
}

// NewSoftFromMnemonicWithAccount creates a SoftKey from a BIP39 mnemonic with specific account index.
// Uses standard BIP44 derivation path: m/44'/9000'/0'/0/{accountIndex}
func NewSoftFromMnemonicWithAccount(networkID uint32, mnemonic string, accountIndex uint32) (*SoftKey, error) {
	keyBytes, err := deriveMnemonicKey(mnemonic, accountIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key from mnemonic: %w", err)
	}

	privKey, err := secp256k1.ToPrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create private key: %w", err)
	}

	return NewSoft(networkID, WithPrivateKey(privKey))
}

// GetLocalPrivateKey returns the secp256k1 private key for local development.
// It loads from ~/.lux/keys/local-key.pk, generating a new key if needed.
func GetLocalPrivateKey() (*secp256k1.PrivateKey, error) {
	// Use local network ID (1337) as default for key loading
	softKey, err := GetOrCreateLocalKey(constants.LocalNetworkID)
	if err != nil {
		return nil, err
	}
	return softKey.Key(), nil
}
