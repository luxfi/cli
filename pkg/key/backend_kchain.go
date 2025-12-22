// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package key

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"sync"
	"time"

	"github.com/luxfi/crypto/mlkem"
	"github.com/luxfi/crypto/threshold"
	_ "github.com/luxfi/crypto/threshold/bls" // register BLS scheme
)

// BackendKChain is the K-Chain distributed secrets backend type.
const BackendKChain BackendType = "kchain"

// K-Chain errors.
var (
	ErrKChainUnavailable      = errors.New("kchain: network unavailable")
	ErrInvalidShareConfig     = errors.New("kchain: invalid share configuration")
	ErrInsufficientShares     = errors.New("kchain: insufficient shares for reconstruction")
	ErrValidatorUnreachable   = errors.New("kchain: validator unreachable")
	ErrShareStoreFailed       = errors.New("kchain: failed to store share")
	ErrShareRetrieveFailed    = errors.New("kchain: failed to retrieve share")
	ErrThresholdSigningFailed = errors.New("kchain: threshold signing failed")
	ErrKeyNotDistributed      = errors.New("kchain: key not distributed to validators")
)

// ShareConfig configures threshold secret sharing parameters.
type ShareConfig struct {
	N              int      // Total number of shares
	K              int      // Threshold required to reconstruct
	ValidatorAddrs []string // Validator network addresses
}

// Validate checks if the share configuration is valid.
func (c *ShareConfig) Validate() error {
	if c.N < 2 {
		return fmt.Errorf("%w: N must be >= 2", ErrInvalidShareConfig)
	}
	if c.K < 1 || c.K > c.N {
		return fmt.Errorf("%w: K must be 1 <= K <= N", ErrInvalidShareConfig)
	}
	if len(c.ValidatorAddrs) != c.N {
		return fmt.Errorf("%w: validator count must equal N", ErrInvalidShareConfig)
	}
	return nil
}

// EncryptedShare holds an ML-KEM encrypted key share.
type EncryptedShare struct {
	Index        int    // Share index (1 to N)
	Ciphertext   []byte // ML-KEM ciphertext
	EncryptedKey []byte // AES-GCM encrypted share data
	Nonce        []byte // AES-GCM nonce
	ValidatorID  string // Target validator identifier
}

// DistributedKeyInfo holds metadata about a distributed key.
type DistributedKeyInfo struct {
	Name           string      `json:"name"`
	GroupPublicKey []byte      `json:"group_public_key"`
	ShareConfig    ShareConfig `json:"share_config"`
	CreatedAt      int64       `json:"created_at"`
	KeyType        string      `json:"key_type"` // "bls", "ec"
}

// KChainBackend implements distributed key storage using threshold cryptography.
type KChainBackend struct {
	mu              sync.RWMutex
	endpoint        string
	connected       bool
	timeout         time.Duration
	distributedKeys map[string]*DistributedKeyInfo
	mlkemKeys       map[string]*mlkem.PrivateKey // validator ML-KEM keys
	rpcClient       *KChainRPCClient             // RPC client for K-Chain API
}

// NewKChainBackend creates a new K-Chain distributed secrets backend.
func NewKChainBackend() *KChainBackend {
	return &KChainBackend{
		timeout:         30 * time.Second,
		distributedKeys: make(map[string]*DistributedKeyInfo),
		mlkemKeys:       make(map[string]*mlkem.PrivateKey),
	}
}

// Type returns the backend type identifier.
func (b *KChainBackend) Type() BackendType {
	return BackendKChain
}

// Name returns a human-readable name.
func (b *KChainBackend) Name() string {
	return "K-Chain Distributed Secrets"
}

// Available checks if this backend is available (connected to K-Chain).
func (b *KChainBackend) Available() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.connected
}

// RequiresPassword returns false; keys are protected by threshold distribution.
func (b *KChainBackend) RequiresPassword() bool {
	return false
}

// RequiresHardware returns false; uses network validators.
func (b *KChainBackend) RequiresHardware() bool {
	return false
}

// SupportsRemoteSigning returns true; signing happens on validators.
func (b *KChainBackend) SupportsRemoteSigning() bool {
	return true
}

// Initialize sets up the backend and attempts K-Chain connection.
func (b *KChainBackend) Initialize(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Default K-Chain endpoint (963N port range)
	if b.endpoint == "" {
		b.endpoint = "http://localhost:9630"
	}

	// Create RPC client
	b.rpcClient = NewKChainRPCClient(b.endpoint)

	// Check K-Chain connectivity via health endpoint
	health, err := b.rpcClient.Health(ctx)
	if err != nil {
		// Try direct TCP connection as fallback
		host := b.endpoint
		if len(host) > 7 && host[:7] == "http://" {
			host = host[7:]
		} else if len(host) > 8 && host[:8] == "https://" {
			host = host[8:]
		}
		conn, err := net.DialTimeout("tcp", host, 5*time.Second)
		if err != nil {
			b.connected = false
			return nil // Not available, but not an error
		}
		conn.Close()
		b.connected = true
		return nil
	}

	b.connected = health.Healthy
	return nil
}

// SetEndpoint configures the K-Chain endpoint.
func (b *KChainBackend) SetEndpoint(endpoint string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.endpoint = endpoint
}

// Close cleans up resources.
func (b *KChainBackend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Clear sensitive data
	for k := range b.mlkemKeys {
		b.mlkemKeys[k] = nil
		delete(b.mlkemKeys, k)
	}
	b.connected = false
	return nil
}

// CreateKey creates a new distributed key set.
func (b *KChainBackend) CreateKey(ctx context.Context, name string, opts CreateKeyOptions) (*HDKeySet, error) {
	if !b.Available() {
		return nil, ErrKChainUnavailable
	}

	// Generate key set locally first
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

	// Note: The key set is created but not yet distributed.
	// Call DistributeKey() to split and distribute to validators.

	return keySet, nil
}

// DistributeKey splits a key into shares and distributes to validators.
func (b *KChainBackend) DistributeKey(ctx context.Context, name string, keyData []byte, config ShareConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}

	// Split secret using Shamir Secret Sharing
	shares, err := b.splitSecret(keyData, config.K, config.N)
	if err != nil {
		return fmt.Errorf("failed to split secret: %w", err)
	}

	// Generate ML-KEM key pairs for each validator if not cached
	validatorPubKeys := make([]*mlkem.PublicKey, config.N)
	for i, addr := range config.ValidatorAddrs {
		pubKey, err := b.getValidatorPublicKey(ctx, addr)
		if err != nil {
			return fmt.Errorf("failed to get validator %s public key: %w", addr, err)
		}
		validatorPubKeys[i] = pubKey
	}

	// Encrypt and distribute shares
	for i, share := range shares {
		encShare, err := b.encryptShare(share, i+1, validatorPubKeys[i], config.ValidatorAddrs[i])
		if err != nil {
			return fmt.Errorf("failed to encrypt share %d: %w", i+1, err)
		}

		if err := b.storeShareOnValidator(ctx, config.ValidatorAddrs[i], name, encShare); err != nil {
			return fmt.Errorf("failed to store share on validator %s: %w", config.ValidatorAddrs[i], err)
		}
	}

	// Store distributed key metadata
	b.mu.Lock()
	b.distributedKeys[name] = &DistributedKeyInfo{
		Name:        name,
		ShareConfig: config,
		CreatedAt:   time.Now().Unix(),
		KeyType:     "generic",
	}
	b.mu.Unlock()

	return nil
}

// DistributeBLSKey distributes a BLS key using threshold BLS scheme.
func (b *KChainBackend) DistributeBLSKey(ctx context.Context, name string, config ShareConfig) (threshold.PublicKey, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Get the BLS threshold scheme
	scheme, err := threshold.GetScheme(threshold.SchemeBLS)
	if err != nil {
		return nil, fmt.Errorf("BLS scheme not available: %w", err)
	}

	// Create trusted dealer for key generation
	dealer, err := scheme.NewTrustedDealer(threshold.DealerConfig{
		Threshold:    config.K,
		TotalParties: config.N,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create dealer: %w", err)
	}

	// Generate key shares
	shares, groupKey, err := dealer.GenerateShares(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate shares: %w", err)
	}

	// Distribute shares to validators (encrypted with ML-KEM)
	for i, share := range shares {
		shareBytes := share.Bytes()

		pubKey, err := b.getValidatorPublicKey(ctx, config.ValidatorAddrs[i])
		if err != nil {
			return nil, fmt.Errorf("failed to get validator public key: %w", err)
		}

		encShare, err := b.encryptShare(shareBytes, i+1, pubKey, config.ValidatorAddrs[i])
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt share: %w", err)
		}

		if err := b.storeShareOnValidator(ctx, config.ValidatorAddrs[i], name, encShare); err != nil {
			return nil, fmt.Errorf("failed to store share: %w", err)
		}
	}

	// Store metadata
	b.mu.Lock()
	b.distributedKeys[name] = &DistributedKeyInfo{
		Name:           name,
		GroupPublicKey: groupKey.Bytes(),
		ShareConfig:    config,
		CreatedAt:      time.Now().Unix(),
		KeyType:        "bls",
	}
	b.mu.Unlock()

	return groupKey, nil
}

// ReconstructKey gathers K shares and reconstructs the secret.
func (b *KChainBackend) ReconstructKey(ctx context.Context, name string) ([]byte, error) {
	b.mu.RLock()
	keyInfo, exists := b.distributedKeys[name]
	b.mu.RUnlock()

	if !exists {
		return nil, ErrKeyNotDistributed
	}

	config := keyInfo.ShareConfig

	// Gather shares from validators
	var shares [][]byte
	var indices []int

	for i, addr := range config.ValidatorAddrs {
		encShare, err := b.retrieveShareFromValidator(ctx, addr, name)
		if err != nil {
			continue // Try other validators
		}

		// Decrypt share using our ML-KEM private key
		shareData, err := b.decryptShare(encShare)
		if err != nil {
			continue
		}

		shares = append(shares, shareData)
		indices = append(indices, i+1) // Shamir uses 1-indexed

		if len(shares) >= config.K {
			break
		}
	}

	if len(shares) < config.K {
		return nil, fmt.Errorf("%w: got %d, need %d", ErrInsufficientShares, len(shares), config.K)
	}

	// Reconstruct secret using Lagrange interpolation
	return b.reconstructSecret(shares, indices)
}

// LoadKey loads a distributed key by reconstructing from shares.
func (b *KChainBackend) LoadKey(ctx context.Context, name, password string) (*HDKeySet, error) {
	// For K-Chain, password is not used - security comes from threshold distribution
	b.mu.RLock()
	keyInfo, exists := b.distributedKeys[name]
	b.mu.RUnlock()

	if !exists {
		return nil, ErrKeyNotFound
	}

	if keyInfo.KeyType == "bls" {
		// BLS keys are not fully reconstructed locally for security
		// Return a stub with group public key
		return &HDKeySet{
			Name:         name,
			BLSPublicKey: keyInfo.GroupPublicKey,
		}, nil
	}

	// Reconstruct generic key
	keyData, err := b.ReconstructKey(ctx, name)
	if err != nil {
		return nil, err
	}

	// Parse reconstructed key data
	var keySet HDKeySet
	if err := json.Unmarshal(keyData, &keySet); err != nil {
		return nil, fmt.Errorf("failed to parse key data: %w", err)
	}

	return &keySet, nil
}

// SaveKey distributes a key set to validators.
func (b *KChainBackend) SaveKey(ctx context.Context, keySet *HDKeySet, password string) error {
	// Serialize key set
	keyData, err := json.Marshal(keySet)
	if err != nil {
		return fmt.Errorf("failed to serialize key set: %w", err)
	}

	// Use default share config if not set
	config := ShareConfig{
		N:              5,
		K:              3,
		ValidatorAddrs: b.getDefaultValidators(),
	}

	return b.DistributeKey(ctx, keySet.Name, keyData, config)
}

// DeleteKey removes distributed shares from validators.
func (b *KChainBackend) DeleteKey(ctx context.Context, name string) error {
	b.mu.Lock()
	keyInfo, exists := b.distributedKeys[name]
	if exists {
		delete(b.distributedKeys, name)
	}
	b.mu.Unlock()

	if !exists {
		return nil
	}

	// Request deletion from validators
	for _, addr := range keyInfo.ShareConfig.ValidatorAddrs {
		_ = b.deleteShareFromValidator(ctx, addr, name) // Best effort
	}

	return nil
}

// ListKeys returns all distributed keys.
func (b *KChainBackend) ListKeys(ctx context.Context) ([]KeyInfo, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var keys []KeyInfo
	for name, info := range b.distributedKeys {
		keys = append(keys, KeyInfo{
			Name:      name,
			Encrypted: true, // Shares are encrypted
			Locked:    false,
			CreatedAt: time.Unix(info.CreatedAt, 0),
		})
	}
	return keys, nil
}

// Lock is a no-op for distributed keys (always protected by threshold).
func (b *KChainBackend) Lock(ctx context.Context, name string) error {
	return nil
}

// Unlock is a no-op for distributed keys.
func (b *KChainBackend) Unlock(ctx context.Context, name, password string) error {
	return nil
}

// IsLocked returns false; distributed keys are not locked in traditional sense.
func (b *KChainBackend) IsLocked(name string) bool {
	return false
}

// Sign performs threshold BLS signing using validators.
func (b *KChainBackend) Sign(ctx context.Context, name string, request SignRequest) (*SignResponse, error) {
	b.mu.RLock()
	keyInfo, exists := b.distributedKeys[name]
	b.mu.RUnlock()

	if !exists {
		return nil, ErrKeyNotDistributed
	}

	if keyInfo.KeyType != "bls" {
		return nil, fmt.Errorf("threshold signing only supported for BLS keys")
	}

	config := keyInfo.ShareConfig

	// Request signature shares from validators
	var sigShares []threshold.SignatureShare
	scheme, err := threshold.GetScheme(threshold.SchemeBLS)
	if err != nil {
		return nil, err
	}

	for _, addr := range config.ValidatorAddrs {
		shareData, err := b.requestSignatureShare(ctx, addr, name, request.Data)
		if err != nil {
			continue
		}

		sigShare, err := scheme.ParseSignatureShare(shareData)
		if err != nil {
			continue
		}

		sigShares = append(sigShares, sigShare)
		if len(sigShares) >= config.K {
			break
		}
	}

	if len(sigShares) < config.K {
		return nil, fmt.Errorf("%w: insufficient signature shares", ErrThresholdSigningFailed)
	}

	// Parse group public key
	groupKey, err := scheme.ParsePublicKey(keyInfo.GroupPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse group key: %w", err)
	}

	// Aggregate signature shares
	aggregator, err := scheme.NewAggregator(groupKey)
	if err != nil {
		return nil, err
	}

	sig, err := aggregator.Aggregate(ctx, request.Data, sigShares, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate signatures: %w", err)
	}

	return &SignResponse{
		Signature: sig.Bytes(),
		PublicKey: keyInfo.GroupPublicKey,
	}, nil
}

// splitSecret splits a secret using Shamir Secret Sharing.
// Uses GF(2^256) arithmetic for splitting arbitrary byte data.
func (b *KChainBackend) splitSecret(secret []byte, k, n int) ([][]byte, error) {
	if k < 1 || k > n || n > 255 {
		return nil, ErrInvalidShareConfig
	}

	// Prime for finite field arithmetic (256-bit)
	prime := new(big.Int)
	prime.SetString("115792089237316195423570985008687907853269984665640564039457584007913129639747", 10)

	// Convert secret to big.Int
	secretInt := new(big.Int).SetBytes(secret)
	if secretInt.Cmp(prime) >= 0 {
		// Secret too large - hash it
		h := sha256.Sum256(secret)
		secretInt = new(big.Int).SetBytes(h[:])
	}

	// Generate random polynomial coefficients
	coeffs := make([]*big.Int, k)
	coeffs[0] = secretInt
	for i := 1; i < k; i++ {
		coeff, err := rand.Int(rand.Reader, prime)
		if err != nil {
			return nil, err
		}
		coeffs[i] = coeff
	}

	// Evaluate polynomial at points 1, 2, ..., n
	shares := make([][]byte, n)
	for i := 0; i < n; i++ {
		x := big.NewInt(int64(i + 1))
		y := evaluatePoly(coeffs, x, prime)

		// Encode share: index (1 byte) + y value (32 bytes)
		share := make([]byte, 33)
		share[0] = byte(i + 1)
		yBytes := y.Bytes()
		copy(share[33-len(yBytes):], yBytes)
		shares[i] = share
	}

	return shares, nil
}

// reconstructSecret reconstructs secret using Lagrange interpolation.
func (b *KChainBackend) reconstructSecret(shares [][]byte, indices []int) ([]byte, error) {
	prime := new(big.Int)
	prime.SetString("115792089237316195423570985008687907853269984665640564039457584007913129639747", 10)

	// Parse shares
	points := make(map[int]*big.Int)
	for i, share := range shares {
		if len(share) < 33 {
			continue
		}
		y := new(big.Int).SetBytes(share[1:33])
		points[indices[i]] = y
	}

	// Lagrange interpolation at x=0
	secret := big.NewInt(0)
	for xi, yi := range points {
		// Compute Lagrange basis polynomial at x=0
		numerator := big.NewInt(1)
		denominator := big.NewInt(1)

		for xj := range points {
			if xi == xj {
				continue
			}
			// numerator *= -xj = 0 - xj
			neg := new(big.Int).Neg(big.NewInt(int64(xj)))
			neg.Mod(neg, prime)
			numerator.Mul(numerator, neg)
			numerator.Mod(numerator, prime)

			// denominator *= (xi - xj)
			diff := big.NewInt(int64(xi - xj))
			diff.Mod(diff, prime)
			denominator.Mul(denominator, diff)
			denominator.Mod(denominator, prime)
		}

		// basis = numerator / denominator
		denomInv := new(big.Int).ModInverse(denominator, prime)
		basis := new(big.Int).Mul(numerator, denomInv)
		basis.Mod(basis, prime)

		// secret += yi * basis
		term := new(big.Int).Mul(yi, basis)
		term.Mod(term, prime)
		secret.Add(secret, term)
		secret.Mod(secret, prime)
	}

	// Convert back to bytes
	result := make([]byte, 32)
	secretBytes := secret.Bytes()
	copy(result[32-len(secretBytes):], secretBytes)
	return result, nil
}

// evaluatePoly evaluates polynomial at point x in finite field.
func evaluatePoly(coeffs []*big.Int, x, prime *big.Int) *big.Int {
	result := new(big.Int).Set(coeffs[len(coeffs)-1])
	for i := len(coeffs) - 2; i >= 0; i-- {
		result.Mul(result, x)
		result.Add(result, coeffs[i])
		result.Mod(result, prime)
	}
	return result
}

// encryptShare encrypts a share using ML-KEM hybrid encryption.
func (b *KChainBackend) encryptShare(shareData []byte, index int, pubKey *mlkem.PublicKey, validatorID string) (*EncryptedShare, error) {
	// ML-KEM encapsulation
	ciphertext, sharedSecret, err := pubKey.Encapsulate()
	if err != nil {
		return nil, fmt.Errorf("ML-KEM encapsulation failed: %w", err)
	}

	// Derive AES key from shared secret
	aesKey := sha256.Sum256(sharedSecret)

	// AES-GCM encryption of share data
	block, err := aes.NewCipher(aesKey[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	encryptedData := gcm.Seal(nil, nonce, shareData, nil)

	return &EncryptedShare{
		Index:        index,
		Ciphertext:   ciphertext,
		EncryptedKey: encryptedData,
		Nonce:        nonce,
		ValidatorID:  validatorID,
	}, nil
}

// decryptShare decrypts an encrypted share using local ML-KEM private key.
func (b *KChainBackend) decryptShare(encShare *EncryptedShare) ([]byte, error) {
	b.mu.RLock()
	privKey, exists := b.mlkemKeys[encShare.ValidatorID]
	b.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no private key for validator %s", encShare.ValidatorID)
	}

	// ML-KEM decapsulation
	sharedSecret, err := privKey.Decapsulate(encShare.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("ML-KEM decapsulation failed: %w", err)
	}

	// Derive AES key
	aesKey := sha256.Sum256(sharedSecret)

	// AES-GCM decryption
	block, err := aes.NewCipher(aesKey[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return gcm.Open(nil, encShare.Nonce, encShare.EncryptedKey, nil)
}

// getValidatorPublicKey retrieves or generates ML-KEM public key for a validator.
func (b *KChainBackend) getValidatorPublicKey(ctx context.Context, addr string) (*mlkem.PublicKey, error) {
	// In production, this would fetch the validator's public key from the network.
	// For now, generate deterministically for testing.
	b.mu.Lock()
	defer b.mu.Unlock()

	if privKey, exists := b.mlkemKeys[addr]; exists {
		return privKey.PublicKey(), nil
	}

	// Generate new ML-KEM key pair (ML-KEM-768 for 192-bit security)
	pubKey, privKey, err := mlkem.GenerateKey(mlkem.MLKEM768)
	if err != nil {
		return nil, err
	}

	b.mlkemKeys[addr] = privKey
	return pubKey, nil
}

// storeShareOnValidator sends an encrypted share to a validator.
func (b *KChainBackend) storeShareOnValidator(ctx context.Context, addr, keyName string, share *EncryptedShare) error {
	if b.rpcClient == nil {
		return ErrKChainUnavailable
	}

	// Encode share data
	shareData, err := json.Marshal(share)
	if err != nil {
		return fmt.Errorf("failed to encode share: %w", err)
	}

	result, err := b.rpcClient.StoreShare(ctx, StoreShareParams{
		KeyID:       keyName,
		ShareIndex:  share.Index,
		ShareData:   string(shareData),
		ValidatorID: addr,
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrShareStoreFailed, err)
	}

	if !result.Stored {
		return ErrShareStoreFailed
	}

	return nil
}

// retrieveShareFromValidator retrieves an encrypted share from a validator.
func (b *KChainBackend) retrieveShareFromValidator(ctx context.Context, addr, keyName string) (*EncryptedShare, error) {
	if b.rpcClient == nil {
		return nil, ErrKChainUnavailable
	}

	result, err := b.rpcClient.RetrieveShare(ctx, RetrieveShareParams{
		KeyID:       keyName,
		ValidatorID: addr,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrShareRetrieveFailed, err)
	}

	var share EncryptedShare
	if err := json.Unmarshal([]byte(result.ShareData), &share); err != nil {
		return nil, fmt.Errorf("failed to decode share: %w", err)
	}

	return &share, nil
}

// deleteShareFromValidator requests share deletion from a validator.
func (b *KChainBackend) deleteShareFromValidator(ctx context.Context, addr, keyName string) error {
	if b.rpcClient == nil {
		return ErrKChainUnavailable
	}

	result, err := b.rpcClient.DeleteShare(ctx, DeleteShareParams{
		KeyID:       keyName,
		ValidatorID: addr,
	})
	if err != nil {
		return err
	}

	if !result.Deleted {
		return fmt.Errorf("failed to delete share: %s", result.Message)
	}

	return nil
}

// requestSignatureShare requests a signature share from a validator.
func (b *KChainBackend) requestSignatureShare(ctx context.Context, addr, keyName string, message []byte) ([]byte, error) {
	if b.rpcClient == nil {
		return nil, ErrKChainUnavailable
	}

	// Get key info to determine algorithm
	b.mu.RLock()
	keyInfo, exists := b.distributedKeys[keyName]
	b.mu.RUnlock()

	algorithm := "bls-sig"
	if exists && keyInfo.KeyType != "" {
		algorithm = keyInfo.KeyType
	}

	result, err := b.rpcClient.RequestSignatureShare(ctx, RequestSignatureShareParams{
		KeyID:       keyName,
		Message:     string(message),
		ValidatorID: addr,
		Algorithm:   algorithm,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrValidatorUnreachable, err)
	}

	return []byte(result.ShareData), nil
}

// getDefaultValidators returns default validator addresses.
func (b *KChainBackend) getDefaultValidators() []string {
	return []string{
		"validator-1.kchain.lux.network:9630",
		"validator-2.kchain.lux.network:9631",
		"validator-3.kchain.lux.network:9632",
		"validator-4.kchain.lux.network:9633",
		"validator-5.kchain.lux.network:9634",
	}
}

func init() {
	RegisterBackend(NewKChainBackend())
}
