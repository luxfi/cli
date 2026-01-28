// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package kms

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// MPC key types for threshold signing
type MPCKeyType string

const (
	MPCKeyTypeECDSA   MPCKeyType = "ecdsa"   // Ethereum, Bitcoin
	MPCKeyTypeEdDSA   MPCKeyType = "eddsa"   // Solana
	MPCKeyTypeTaproot MPCKeyType = "taproot" // Bitcoin Taproot
)

// MPCChain represents a supported blockchain.
type MPCChain string

const (
	MPCChainEthereum  MPCChain = "ethereum"
	MPCChainPolygon   MPCChain = "polygon"
	MPCChainArbitrum  MPCChain = "arbitrum"
	MPCChainOptimism  MPCChain = "optimism"
	MPCChainBase      MPCChain = "base"
	MPCChainAvalanche MPCChain = "avalanche"
	MPCChainBNB       MPCChain = "bnb"
	MPCChainBitcoin   MPCChain = "bitcoin"
	MPCChainSolana    MPCChain = "solana"
	MPCChainLux       MPCChain = "lux"
)

// MPCWallet represents a multi-party computation wallet.
type MPCWallet struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	KeyType        MPCKeyType          `json:"keyType"`
	Threshold      int                 `json:"threshold"`    // t in t-of-n
	TotalParties   int                 `json:"totalParties"` // n in t-of-n
	ParticipantIDs []string            `json:"participantIds"`
	PublicKey      []byte              `json:"publicKey"`
	ChainAddresses map[MPCChain]string `json:"chainAddresses"`
	Status         KeyStatus           `json:"status"`
	OrgID          string              `json:"orgId,omitempty"`
	ProjectID      string              `json:"projectId,omitempty"`
	Metadata       map[string]string   `json:"metadata,omitempty"`
	Created        time.Time           `json:"created"`
	Updated        time.Time           `json:"updated"`
}

// MPCNode represents a participant node in MPC operations.
type MPCNode struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Endpoint  string            `json:"endpoint"`
	Port      int               `json:"port"`
	PublicKey []byte            `json:"publicKey"`
	Status    string            `json:"status"`
	OrgID     string            `json:"orgId,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Created   time.Time         `json:"created"`
	LastSeen  time.Time         `json:"lastSeen"`
}

// MPCSigningRequest represents a request to sign data.
type MPCSigningRequest struct {
	ID             string            `json:"id"`
	WalletID       string            `json:"walletId"`
	Chain          MPCChain          `json:"chain"`
	RawTransaction []byte            `json:"rawTransaction"`
	Message        []byte            `json:"message,omitempty"` // For message signing
	Status         SigningStatus     `json:"status"`
	Signatures     map[string][]byte `json:"signatures"` // nodeID -> partial signature
	FinalSignature []byte            `json:"finalSignature,omitempty"`
	RequiredSigs   int               `json:"requiredSigs"`
	CollectedSigs  int               `json:"collectedSigs"`
	Created        time.Time         `json:"created"`
	ExpiresAt      time.Time         `json:"expiresAt"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// SigningStatus represents the status of a signing request.
type SigningStatus string

const (
	SigningStatusPending    SigningStatus = "pending"
	SigningStatusCollecting SigningStatus = "collecting"
	SigningStatusComplete   SigningStatus = "complete"
	SigningStatusFailed     SigningStatus = "failed"
	SigningStatusExpired    SigningStatus = "expired"
)

// Key prefixes for MPC storage
const (
	mpcWalletPrefix  = "kms/mpc/wallet/"
	mpcNodePrefix    = "kms/mpc/node/"
	mpcSigningPrefix = "kms/mpc/signing/"
	mpcSharePrefix   = "kms/mpc/share/"
)

// MPCManager handles MPC operations integrated with KMS.
type MPCManager struct {
	kms   *KMS
	store StorageBackend
}

// NewMPCManager creates a new MPC manager.
func NewMPCManager(kms *KMS) *MPCManager {
	return &MPCManager{
		kms:   kms,
		store: kms.store,
	}
}

// RegisterNode registers a new MPC node.
func (m *MPCManager) RegisterNode(ctx context.Context, name, endpoint string, port int, publicKey []byte, opts *NodeOptions) (*MPCNode, error) {
	nodeID := generateID(16)
	now := time.Now()

	node := &MPCNode{
		ID:        nodeID,
		Name:      name,
		Endpoint:  endpoint,
		Port:      port,
		PublicKey: publicKey,
		Status:    "active",
		Created:   now,
		LastSeen:  now,
	}

	if opts != nil {
		node.OrgID = opts.OrgID
		node.Metadata = opts.Metadata
	}

	if err := SetJSON(ctx, m.store, mpcNodePrefix+nodeID, node); err != nil {
		return nil, fmt.Errorf("failed to save node: %w", err)
	}

	return node, nil
}

// NodeOptions holds options for node registration.
type NodeOptions struct {
	OrgID    string
	Metadata map[string]string
}

// GetNode retrieves a node by ID.
func (m *MPCManager) GetNode(ctx context.Context, nodeID string) (*MPCNode, error) {
	return GetJSON[MPCNode](ctx, m.store, mpcNodePrefix+nodeID)
}

// ListNodes lists all registered MPC nodes.
func (m *MPCManager) ListNodes(ctx context.Context) ([]*MPCNode, error) {
	var nodes []*MPCNode
	err := m.store.Scan(ctx, mpcNodePrefix, func(key string, value []byte) error {
		var node MPCNode
		if err := json.Unmarshal(value, &node); err != nil {
			return nil
		}
		nodes = append(nodes, &node)
		return nil
	})
	return nodes, err
}

// UpdateNodeStatus updates a node's status and last seen time.
func (m *MPCManager) UpdateNodeStatus(ctx context.Context, nodeID, status string) error {
	node, err := m.GetNode(ctx, nodeID)
	if err != nil {
		return err
	}

	node.Status = status
	node.LastSeen = time.Now()

	return SetJSON(ctx, m.store, mpcNodePrefix+nodeID, node)
}

// CreateWallet creates a new MPC wallet.
func (m *MPCManager) CreateWallet(ctx context.Context, name string, keyType MPCKeyType, threshold, totalParties int, participantIDs []string, opts *WalletOptions) (*MPCWallet, error) {
	if threshold < 1 || threshold > totalParties {
		return nil, fmt.Errorf("invalid threshold: must be between 1 and %d", totalParties)
	}

	if len(participantIDs) != totalParties {
		return nil, fmt.Errorf("participant count mismatch: expected %d, got %d", totalParties, len(participantIDs))
	}

	walletID := generateID(16)
	now := time.Now()

	wallet := &MPCWallet{
		ID:             walletID,
		Name:           name,
		KeyType:        keyType,
		Threshold:      threshold,
		TotalParties:   totalParties,
		ParticipantIDs: participantIDs,
		ChainAddresses: make(map[MPCChain]string),
		Status:         KeyStatusPending, // Key generation not yet complete
		Created:        now,
		Updated:        now,
	}

	if opts != nil {
		wallet.OrgID = opts.OrgID
		wallet.ProjectID = opts.ProjectID
		wallet.Metadata = opts.Metadata
	}

	// Create corresponding KMS key reference
	kmsKey := &Key{
		ID:           walletID,
		Name:         fmt.Sprintf("mpc-%s", name),
		Type:         KeyType(keyType),
		Usage:        KeyUsageMPC,
		Status:       KeyStatusPending,
		Version:      1,
		Threshold:    threshold,
		TotalShares:  totalParties,
		ShareHolders: participantIDs,
		Created:      now,
		Updated:      now,
	}

	if opts != nil {
		kmsKey.OrgID = opts.OrgID
		kmsKey.ProjectID = opts.ProjectID
	}

	// Save wallet
	if err := SetJSON(ctx, m.store, mpcWalletPrefix+walletID, wallet); err != nil {
		return nil, fmt.Errorf("failed to save wallet: %w", err)
	}

	// Save KMS key reference
	if err := SetJSON(ctx, m.store, keyPrefix+walletID, kmsKey); err != nil {
		return nil, fmt.Errorf("failed to save key reference: %w", err)
	}

	return wallet, nil
}

// WalletOptions holds options for wallet creation.
type WalletOptions struct {
	OrgID     string
	ProjectID string
	Metadata  map[string]string
}

// GetWallet retrieves a wallet by ID.
func (m *MPCManager) GetWallet(ctx context.Context, walletID string) (*MPCWallet, error) {
	return GetJSON[MPCWallet](ctx, m.store, mpcWalletPrefix+walletID)
}

// ListWallets lists all MPC wallets.
func (m *MPCManager) ListWallets(ctx context.Context) ([]*MPCWallet, error) {
	var wallets []*MPCWallet
	err := m.store.Scan(ctx, mpcWalletPrefix, func(key string, value []byte) error {
		var wallet MPCWallet
		if err := json.Unmarshal(value, &wallet); err != nil {
			return nil
		}
		wallets = append(wallets, &wallet)
		return nil
	})
	return wallets, err
}

// SetWalletPublicKey sets the public key after MPC key generation completes.
func (m *MPCManager) SetWalletPublicKey(ctx context.Context, walletID string, publicKey []byte, chainAddresses map[MPCChain]string) error {
	wallet, err := m.GetWallet(ctx, walletID)
	if err != nil {
		return err
	}

	wallet.PublicKey = publicKey
	wallet.ChainAddresses = chainAddresses
	wallet.Status = KeyStatusActive
	wallet.Updated = time.Now()

	if err := SetJSON(ctx, m.store, mpcWalletPrefix+walletID, wallet); err != nil {
		return err
	}

	// Update KMS key status
	key, err := m.kms.GetKey(ctx, walletID)
	if err != nil {
		return err
	}

	key.Status = KeyStatusActive
	key.Updated = time.Now()

	return SetJSON(ctx, m.store, keyPrefix+walletID, key)
}

// CreateSigningRequest creates a new signing request.
func (m *MPCManager) CreateSigningRequest(ctx context.Context, walletID string, chain MPCChain, rawTransaction []byte, opts *SigningOptions) (*MPCSigningRequest, error) {
	wallet, err := m.GetWallet(ctx, walletID)
	if err != nil {
		return nil, err
	}

	if wallet.Status != KeyStatusActive {
		return nil, fmt.Errorf("wallet %s is not active", walletID)
	}

	requestID := generateID(16)
	now := time.Now()
	expiresAt := now.Add(5 * time.Minute) // Default 5 minute expiry

	if opts != nil && opts.ExpiresIn > 0 {
		expiresAt = now.Add(opts.ExpiresIn)
	}

	request := &MPCSigningRequest{
		ID:             requestID,
		WalletID:       walletID,
		Chain:          chain,
		RawTransaction: rawTransaction,
		Status:         SigningStatusPending,
		Signatures:     make(map[string][]byte),
		RequiredSigs:   wallet.Threshold,
		CollectedSigs:  0,
		Created:        now,
		ExpiresAt:      expiresAt,
	}

	if opts != nil {
		request.Message = opts.Message
		request.Metadata = opts.Metadata
	}

	if err := SetJSON(ctx, m.store, mpcSigningPrefix+requestID, request); err != nil {
		return nil, fmt.Errorf("failed to save signing request: %w", err)
	}

	return request, nil
}

// SigningOptions holds options for signing requests.
type SigningOptions struct {
	Message   []byte
	ExpiresIn time.Duration
	Metadata  map[string]string
}

// GetSigningRequest retrieves a signing request.
func (m *MPCManager) GetSigningRequest(ctx context.Context, requestID string) (*MPCSigningRequest, error) {
	return GetJSON[MPCSigningRequest](ctx, m.store, mpcSigningPrefix+requestID)
}

// SubmitPartialSignature submits a partial signature from a node.
func (m *MPCManager) SubmitPartialSignature(ctx context.Context, requestID, nodeID string, partialSig []byte) (*MPCSigningRequest, error) {
	request, err := m.GetSigningRequest(ctx, requestID)
	if err != nil {
		return nil, err
	}

	if time.Now().After(request.ExpiresAt) {
		request.Status = SigningStatusExpired
		SetJSON(ctx, m.store, mpcSigningPrefix+requestID, request)
		return nil, fmt.Errorf("signing request has expired")
	}

	if request.Status == SigningStatusComplete {
		return nil, fmt.Errorf("signing request already complete")
	}

	// Check if node is a participant
	wallet, err := m.GetWallet(ctx, request.WalletID)
	if err != nil {
		return nil, err
	}

	isParticipant := false
	for _, pid := range wallet.ParticipantIDs {
		if pid == nodeID {
			isParticipant = true
			break
		}
	}
	if !isParticipant {
		return nil, fmt.Errorf("node %s is not a participant in wallet %s", nodeID, request.WalletID)
	}

	// Store partial signature
	if request.Signatures[nodeID] != nil {
		return nil, fmt.Errorf("node %s has already submitted a signature", nodeID)
	}

	request.Signatures[nodeID] = partialSig
	request.CollectedSigs++
	request.Status = SigningStatusCollecting

	if request.CollectedSigs >= request.RequiredSigs {
		// In a real implementation, we would combine the partial signatures here
		// For now, we just mark it as complete
		request.Status = SigningStatusComplete
	}

	if err := SetJSON(ctx, m.store, mpcSigningPrefix+requestID, request); err != nil {
		return nil, err
	}

	return request, nil
}

// SetFinalSignature sets the combined final signature.
func (m *MPCManager) SetFinalSignature(ctx context.Context, requestID string, finalSig []byte) error {
	request, err := m.GetSigningRequest(ctx, requestID)
	if err != nil {
		return err
	}

	request.FinalSignature = finalSig
	request.Status = SigningStatusComplete

	return SetJSON(ctx, m.store, mpcSigningPrefix+requestID, request)
}

// StoreKeyShare stores an encrypted key share for a node.
func (m *MPCManager) StoreKeyShare(ctx context.Context, walletID, nodeID string, encryptedShare []byte) error {
	key := fmt.Sprintf("%s%s/%s", mpcSharePrefix, walletID, nodeID)
	return m.store.Set(ctx, key, encryptedShare)
}

// GetKeyShare retrieves an encrypted key share.
func (m *MPCManager) GetKeyShare(ctx context.Context, walletID, nodeID string) ([]byte, error) {
	key := fmt.Sprintf("%s%s/%s", mpcSharePrefix, walletID, nodeID)
	return m.store.Get(ctx, key)
}

// ListPendingSigningRequests lists all pending signing requests for a wallet.
func (m *MPCManager) ListPendingSigningRequests(ctx context.Context, walletID string) ([]*MPCSigningRequest, error) {
	var requests []*MPCSigningRequest
	err := m.store.Scan(ctx, mpcSigningPrefix, func(key string, value []byte) error {
		var request MPCSigningRequest
		if err := json.Unmarshal(value, &request); err != nil {
			return nil
		}
		if request.WalletID == walletID && (request.Status == SigningStatusPending || request.Status == SigningStatusCollecting) {
			requests = append(requests, &request)
		}
		return nil
	})
	return requests, err
}
