// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package key

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// KChainRPCClient implements the K-Chain Key Management API.
type KChainRPCClient struct {
	endpoint   string
	httpClient *http.Client
	apiKey     string
}

// NewKChainRPCClient creates a new K-Chain RPC client.
func NewKChainRPCClient(endpoint string) *KChainRPCClient {
	return &KChainRPCClient{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetAPIKey sets the API key for authenticated requests.
func (c *KChainRPCClient) SetAPIKey(apiKey string) {
	c.apiKey = apiKey
}

// RPCRequest represents a JSON-RPC 2.0 request.
type RPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// RPCResponse represents a JSON-RPC 2.0 response.
type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	if e.Data != "" {
		return fmt.Sprintf("RPC error %d: %s (%s)", e.Code, e.Message, e.Data)
	}
	return fmt.Sprintf("RPC error %d: %s", e.Code, e.Message)
}

// call makes an RPC call to the K-Chain endpoint.
func (c *KChainRPCClient) call(ctx context.Context, method string, params interface{}, result interface{}) error {
	req := RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/ext/kchain/rpc", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("RPC call failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var rpcResp RPCResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if rpcResp.Error != nil {
		return rpcResp.Error
	}

	if result != nil && len(rpcResp.Result) > 0 {
		if err := json.Unmarshal(rpcResp.Result, result); err != nil {
			return fmt.Errorf("failed to unmarshal result: %w", err)
		}
	}

	return nil
}

// ======== Key Management API ========

// KeyMetadata represents key information returned by the API.
type KeyMetadata struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Algorithm   string    `json:"algorithm"`
	KeyType     string    `json:"keyType"`
	PublicKey   string    `json:"publicKey,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Distributed bool      `json:"distributed"`
	Threshold   int       `json:"threshold,omitempty"`
	TotalShares int       `json:"totalShares,omitempty"`
	Status      string    `json:"status"`
	Tags        []string  `json:"tags,omitempty"`
}

// ListKeysParams contains parameters for listing keys.
type ListKeysParams struct {
	Offset    int      `json:"offset,omitempty"`
	Limit     int      `json:"limit,omitempty"`
	Algorithm string   `json:"algorithm,omitempty"`
	Status    string   `json:"status,omitempty"`
	Tags      []string `json:"tags,omitempty"`
}

// ListKeysResult contains the result of listing keys.
type ListKeysResult struct {
	Keys  []KeyMetadata `json:"keys"`
	Total int           `json:"total"`
}

// ListKeys retrieves all keys with optional filtering.
// GET /keys
func (c *KChainRPCClient) ListKeys(ctx context.Context, params ListKeysParams) (*ListKeysResult, error) {
	var result ListKeysResult
	if err := c.call(ctx, "kchain.listKeys", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetKeyByIDParams contains parameters for getting a key by ID.
type GetKeyByIDParams struct {
	ID string `json:"id"`
}

// GetKeyByID retrieves a key by its unique ID.
// GET /keys/{id}
func (c *KChainRPCClient) GetKeyByID(ctx context.Context, id string) (*KeyMetadata, error) {
	var result KeyMetadata
	if err := c.call(ctx, "kchain.getKeyByID", GetKeyByIDParams{ID: id}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetKeyByNameParams contains parameters for getting a key by name.
type GetKeyByNameParams struct {
	Name string `json:"name"`
}

// GetKeyByName retrieves a key by its name.
// GET /keys/name/{name}
func (c *KChainRPCClient) GetKeyByName(ctx context.Context, name string) (*KeyMetadata, error) {
	var result KeyMetadata
	if err := c.call(ctx, "kchain.getKeyByName", GetKeyByNameParams{Name: name}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateKeyParams contains parameters for creating a key.
type CreateKeyParams struct {
	Name        string   `json:"name"`
	Algorithm   string   `json:"algorithm"`           // "bls", "ecdsa-secp256k1", "eddsa-ed25519", "ml-dsa-65"
	KeyType     string   `json:"keyType,omitempty"`   // "signing", "encryption", "both"
	Threshold   int      `json:"threshold,omitempty"` // For distributed keys
	TotalShares int      `json:"totalShares,omitempty"`
	Validators  []string `json:"validators,omitempty"` // Validator addresses for distribution
	Tags        []string `json:"tags,omitempty"`
	Metadata    string   `json:"metadata,omitempty"` // Custom metadata JSON
}

// CreateKeyResult contains the result of creating a key.
type CreateKeyResult struct {
	Key       KeyMetadata `json:"key"`
	PublicKey string      `json:"publicKey"`
	ShareIDs  []string    `json:"shareIds,omitempty"` // For distributed keys
}

// CreateKey creates a new key.
// POST /keys
func (c *KChainRPCClient) CreateKey(ctx context.Context, params CreateKeyParams) (*CreateKeyResult, error) {
	var result CreateKeyResult
	if err := c.call(ctx, "kchain.createKey", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateKeyParams contains parameters for updating a key.
type UpdateKeyParams struct {
	ID       string   `json:"id"`
	Name     string   `json:"name,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Metadata string   `json:"metadata,omitempty"`
	Status   string   `json:"status,omitempty"` // "active", "disabled", "compromised"
}

// UpdateKey updates key metadata.
// PATCH /keys/{id}
func (c *KChainRPCClient) UpdateKey(ctx context.Context, params UpdateKeyParams) (*KeyMetadata, error) {
	var result KeyMetadata
	if err := c.call(ctx, "kchain.updateKey", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteKeyParams contains parameters for deleting a key.
type DeleteKeyParams struct {
	ID    string `json:"id"`
	Force bool   `json:"force,omitempty"` // Force deletion even if shares exist
}

// DeleteKeyResult contains the result of deleting a key.
type DeleteKeyResult struct {
	Success       bool     `json:"success"`
	DeletedShares []string `json:"deletedShares,omitempty"`
}

// DeleteKey removes a key and its distributed shares.
// DELETE /keys/{id}
func (c *KChainRPCClient) DeleteKey(ctx context.Context, params DeleteKeyParams) (*DeleteKeyResult, error) {
	var result DeleteKeyResult
	if err := c.call(ctx, "kchain.deleteKey", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ======== Cryptographic Operations ========

// EncryptParams contains parameters for encryption.
type EncryptParams struct {
	KeyID     string `json:"keyId"`
	Plaintext string `json:"plaintext"`     // Base64-encoded
	AAD       string `json:"aad,omitempty"` // Additional authenticated data
}

// EncryptResult contains the result of encryption.
type EncryptResult struct {
	Ciphertext string `json:"ciphertext"` // Base64-encoded
	Nonce      string `json:"nonce,omitempty"`
	Tag        string `json:"tag,omitempty"` // For AEAD
}

// Encrypt encrypts data using the specified key.
// POST /keys/{id}/encrypt
func (c *KChainRPCClient) Encrypt(ctx context.Context, params EncryptParams) (*EncryptResult, error) {
	var result EncryptResult
	if err := c.call(ctx, "kchain.encrypt", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DecryptParams contains parameters for decryption.
type DecryptParams struct {
	KeyID      string `json:"keyId"`
	Ciphertext string `json:"ciphertext"` // Base64-encoded
	Nonce      string `json:"nonce,omitempty"`
	Tag        string `json:"tag,omitempty"`
	AAD        string `json:"aad,omitempty"`
}

// DecryptResult contains the result of decryption.
type DecryptResult struct {
	Plaintext string `json:"plaintext"` // Base64-encoded
}

// Decrypt decrypts data using the specified key.
// POST /keys/{id}/decrypt
func (c *KChainRPCClient) Decrypt(ctx context.Context, params DecryptParams) (*DecryptResult, error) {
	var result DecryptResult
	if err := c.call(ctx, "kchain.decrypt", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SignParams contains parameters for signing.
type SignParams struct {
	KeyID     string `json:"keyId"`
	Message   string `json:"message"`             // Base64-encoded message or hash
	Algorithm string `json:"algorithm"`           // "bls-sig", "ecdsa", "eddsa", "ml-dsa"
	Prehashed bool   `json:"prehashed,omitempty"` // True if message is already hashed
}

// SignResult contains the result of signing.
type SignResult struct {
	Signature   string   `json:"signature"` // Base64-encoded
	PublicKey   string   `json:"publicKey,omitempty"`
	ShareProofs []string `json:"shareProofs,omitempty"` // For threshold signatures
}

// Sign creates a signature using the specified key.
// POST /keys/{id}/sign
func (c *KChainRPCClient) Sign(ctx context.Context, params SignParams) (*SignResult, error) {
	var result SignResult
	if err := c.call(ctx, "kchain.sign", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// VerifyParams contains parameters for signature verification.
type VerifyParams struct {
	KeyID     string `json:"keyId,omitempty"`     // Optional if publicKey provided
	PublicKey string `json:"publicKey,omitempty"` // Optional if keyId provided
	Message   string `json:"message"`             // Base64-encoded
	Signature string `json:"signature"`           // Base64-encoded
	Algorithm string `json:"algorithm"`
	Prehashed bool   `json:"prehashed,omitempty"`
}

// VerifyResult contains the result of signature verification.
type VerifyResult struct {
	Valid   bool   `json:"valid"`
	KeyID   string `json:"keyId,omitempty"`
	Message string `json:"message,omitempty"` // Error message if invalid
}

// Verify verifies a signature.
// POST /keys/{id}/verify or POST /verify
func (c *KChainRPCClient) Verify(ctx context.Context, params VerifyParams) (*VerifyResult, error) {
	var result VerifyResult
	if err := c.call(ctx, "kchain.verify", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPublicKeyParams contains parameters for retrieving a public key.
type GetPublicKeyParams struct {
	KeyID  string `json:"keyId"`
	Format string `json:"format,omitempty"` // "raw", "pem", "der", "jwk"
}

// GetPublicKeyResult contains the public key.
type GetPublicKeyResult struct {
	PublicKey string `json:"publicKey"`
	Algorithm string `json:"algorithm"`
	Format    string `json:"format"`
}

// GetPublicKey retrieves the public key for a key ID.
// GET /keys/{id}/publicKey
func (c *KChainRPCClient) GetPublicKey(ctx context.Context, params GetPublicKeyParams) (*GetPublicKeyResult, error) {
	var result GetPublicKeyResult
	if err := c.call(ctx, "kchain.getPublicKey", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ======== Algorithm Information ========

// AlgorithmInfo describes a supported signing algorithm.
type AlgorithmInfo struct {
	Name             string   `json:"name"`
	Type             string   `json:"type"`          // "signing", "encryption", "key-exchange"
	SecurityLevel    int      `json:"securityLevel"` // bits
	KeySize          int      `json:"keySize,omitempty"`
	SignatureSize    int      `json:"signatureSize,omitempty"`
	PostQuantum      bool     `json:"postQuantum"`
	ThresholdSupport bool     `json:"thresholdSupport"`
	Description      string   `json:"description"`
	Standards        []string `json:"standards,omitempty"` // NIST, IETF, etc.
}

// ListAlgorithmsResult contains supported algorithms.
type ListAlgorithmsResult struct {
	Algorithms []AlgorithmInfo `json:"algorithms"`
}

// ListAlgorithms lists all supported signing algorithms.
// GET /algorithms
func (c *KChainRPCClient) ListAlgorithms(ctx context.Context) (*ListAlgorithmsResult, error) {
	var result ListAlgorithmsResult
	if err := c.call(ctx, "kchain.listAlgorithms", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ======== Threshold Operations ========

// DistributeKeyParams contains parameters for key distribution.
type DistributeKeyParams struct {
	KeyID      string   `json:"keyId"`
	Threshold  int      `json:"threshold"`
	TotalParts int      `json:"totalParts"`
	Validators []string `json:"validators"`
}

// DistributeKeyResult contains the result of key distribution.
type DistributeKeyResult struct {
	Success        bool     `json:"success"`
	ShareIDs       []string `json:"shareIds"`
	GroupPublicKey string   `json:"groupPublicKey,omitempty"`
}

// DistributeKey distributes a key to validators using threshold sharing.
func (c *KChainRPCClient) DistributeKey(ctx context.Context, params DistributeKeyParams) (*DistributeKeyResult, error) {
	var result DistributeKeyResult
	if err := c.call(ctx, "kchain.distributeKey", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GatherSharesParams contains parameters for gathering shares.
type GatherSharesParams struct {
	KeyID     string   `json:"keyId"`
	ShareIDs  []string `json:"shareIds,omitempty"` // Optional: specific shares to use
	MinShares int      `json:"minShares,omitempty"`
}

// GatherSharesResult contains gathered share information.
type GatherSharesResult struct {
	Available int      `json:"available"`
	Required  int      `json:"required"`
	ShareIDs  []string `json:"shareIds"`
	Ready     bool     `json:"ready"`
}

// GatherShares checks availability of key shares.
func (c *KChainRPCClient) GatherShares(ctx context.Context, params GatherSharesParams) (*GatherSharesResult, error) {
	var result GatherSharesResult
	if err := c.call(ctx, "kchain.gatherShares", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ThresholdSignParams contains parameters for threshold signing.
type ThresholdSignParams struct {
	KeyID     string   `json:"keyId"`
	Message   string   `json:"message"`
	ShareIDs  []string `json:"shareIds,omitempty"` // Optional: specific shares to use
	Algorithm string   `json:"algorithm"`
}

// ThresholdSignResult contains the threshold signature.
type ThresholdSignResult struct {
	Signature      string   `json:"signature"`
	GroupPublicKey string   `json:"groupPublicKey"`
	ParticipantIDs []string `json:"participantIds"`
	Proofs         []string `json:"proofs,omitempty"`
}

// ThresholdSign performs a threshold signature using distributed shares.
func (c *KChainRPCClient) ThresholdSign(ctx context.Context, params ThresholdSignParams) (*ThresholdSignResult, error) {
	var result ThresholdSignResult
	if err := c.call(ctx, "kchain.thresholdSign", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ReshareKeyParams contains parameters for key resharing.
type ReshareKeyParams struct {
	KeyID         string   `json:"keyId"`
	NewThreshold  int      `json:"newThreshold"`
	NewTotalParts int      `json:"newTotalParts"`
	NewValidators []string `json:"newValidators"`
}

// ReshareKeyResult contains the result of key resharing.
type ReshareKeyResult struct {
	Success     bool     `json:"success"`
	NewShareIDs []string `json:"newShareIds"`
}

// ReshareKey reshares a distributed key with new parameters.
func (c *KChainRPCClient) ReshareKey(ctx context.Context, params ReshareKeyParams) (*ReshareKeyResult, error) {
	var result ReshareKeyResult
	if err := c.call(ctx, "kchain.reshareKey", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ======== Share Management ========

// StoreShareParams contains parameters for storing a share.
type StoreShareParams struct {
	KeyID       string `json:"keyId"`
	ShareIndex  int    `json:"shareIndex"`
	ShareData   string `json:"shareData"` // Encrypted share data
	ValidatorID string `json:"validatorId"`
}

// StoreShareResult contains the result of storing a share.
type StoreShareResult struct {
	ShareID   string `json:"shareId"`
	Stored    bool   `json:"stored"`
	Timestamp int64  `json:"timestamp"`
}

// StoreShare stores an encrypted share on a validator.
func (c *KChainRPCClient) StoreShare(ctx context.Context, params StoreShareParams) (*StoreShareResult, error) {
	var result StoreShareResult
	if err := c.call(ctx, "kchain.storeShare", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// RetrieveShareParams contains parameters for retrieving a share.
type RetrieveShareParams struct {
	KeyID       string `json:"keyId"`
	ShareID     string `json:"shareId,omitempty"`
	ValidatorID string `json:"validatorId,omitempty"`
}

// RetrieveShareResult contains the retrieved share.
type RetrieveShareResult struct {
	ShareID     string `json:"shareId"`
	ShareIndex  int    `json:"shareIndex"`
	ShareData   string `json:"shareData"` // Encrypted
	ValidatorID string `json:"validatorId"`
	Timestamp   int64  `json:"timestamp"`
}

// RetrieveShare retrieves an encrypted share from a validator.
func (c *KChainRPCClient) RetrieveShare(ctx context.Context, params RetrieveShareParams) (*RetrieveShareResult, error) {
	var result RetrieveShareResult
	if err := c.call(ctx, "kchain.retrieveShare", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteShareParams contains parameters for deleting a share.
type DeleteShareParams struct {
	KeyID       string `json:"keyId"`
	ShareID     string `json:"shareId,omitempty"`
	ValidatorID string `json:"validatorId,omitempty"`
}

// DeleteShareResult contains the result of share deletion.
type DeleteShareResult struct {
	Deleted bool   `json:"deleted"`
	Message string `json:"message,omitempty"`
}

// DeleteShare deletes a share from a validator.
func (c *KChainRPCClient) DeleteShare(ctx context.Context, params DeleteShareParams) (*DeleteShareResult, error) {
	var result DeleteShareResult
	if err := c.call(ctx, "kchain.deleteShare", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// RequestSignatureShareParams contains parameters for requesting a signature share.
type RequestSignatureShareParams struct {
	KeyID       string `json:"keyId"`
	Message     string `json:"message"`
	ValidatorID string `json:"validatorId"`
	Algorithm   string `json:"algorithm"`
}

// RequestSignatureShareResult contains the signature share.
type RequestSignatureShareResult struct {
	ShareID   string `json:"shareId"`
	ShareData string `json:"shareData"` // Signature share
	Proof     string `json:"proof,omitempty"`
}

// RequestSignatureShare requests a signature share from a validator.
func (c *KChainRPCClient) RequestSignatureShare(ctx context.Context, params RequestSignatureShareParams) (*RequestSignatureShareResult, error) {
	var result RequestSignatureShareResult
	if err := c.call(ctx, "kchain.requestSignatureShare", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ======== Health and Status ========

// HealthResult contains service health information.
type HealthResult struct {
	Healthy    bool             `json:"healthy"`
	Version    string           `json:"version"`
	Uptime     int64            `json:"uptime"` // seconds
	Validators map[string]bool  `json:"validators"`
	Latency    map[string]int64 `json:"latency"` // ms
}

// Health checks service health.
func (c *KChainRPCClient) Health(ctx context.Context) (*HealthResult, error) {
	var result HealthResult
	if err := c.call(ctx, "kchain.health", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
