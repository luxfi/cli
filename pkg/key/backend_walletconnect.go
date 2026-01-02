// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package key

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/skip2/go-qrcode"
)

// WalletConnect v2 protocol constants
const (
	// WalletConnectName is the display name for the WalletConnect backend
	WalletConnectName = "WalletConnect (Mobile Signing)"

	wcRelayURL      = "wss://relay.walletconnect.com"
	wcProtocolID    = "wc"
	wcVersion       = "2"
	wcSessionExpiry = 7 * 24 * time.Hour // 7 days

	// Request timeouts
	wcPairingTimeout  = 5 * time.Minute
	wcSigningTimeout  = 2 * time.Minute
	wcConnectTimeout  = 30 * time.Second
	wcHeartbeatPeriod = 30 * time.Second

	// Methods
	wcMethodPersonalSign = "personal_sign"        // EIP-191
	wcMethodSignTypedV4  = "eth_signTypedData_v4" // EIP-712
	wcMethodSendTx       = "eth_sendTransaction"
	wcMethodSignTx       = "eth_signTransaction"
)

var (
	ErrWCNotPaired       = errors.New("walletconnect: not paired, scan QR code first")
	ErrWCSessionExpired  = errors.New("walletconnect: session expired")
	ErrWCUserRejected    = errors.New("walletconnect: user rejected request")
	ErrWCTimeout         = errors.New("walletconnect: request timed out")
	ErrWCDisconnected    = errors.New("walletconnect: disconnected from relay")
	ErrWCNoProjectID     = errors.New("walletconnect: project ID required (set LUX_WC_PROJECT_ID)")
	ErrWCInvalidResponse = errors.New("walletconnect: invalid response from wallet")
)

// WalletConnectBackend implements remote signing via WalletConnect v2
type WalletConnectBackend struct {
	dataDir   string
	projectID string

	mu       sync.RWMutex
	sessions map[string]*wcSession
	conn     *websocket.Conn
	done     chan struct{}
}

// wcSession represents an active WalletConnect pairing session
type wcSession struct {
	Topic      string    `json:"topic"`
	SymKey     []byte    `json:"sym_key"`
	PeerPubKey string    `json:"peer_pub_key"`
	ChainID    int       `json:"chain_id"`
	Address    string    `json:"address"`
	PairedAt   time.Time `json:"paired_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	PeerName   string    `json:"peer_name"` // e.g. "MetaMask", "Rainbow"
	PeerIcon   string    `json:"peer_icon"`
}

// wcRequest represents a JSON-RPC request to the wallet
type wcRequest struct {
	ID      int64       `json:"id"`
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

// wcResponse represents a JSON-RPC response from the wallet
type wcResponse struct {
	ID      int64           `json:"id"`
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *wcError        `json:"error,omitempty"`
}

// wcError represents a JSON-RPC error
type wcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewWalletConnectBackend creates a new WalletConnect backend
func NewWalletConnectBackend() *WalletConnectBackend {
	return &WalletConnectBackend{
		sessions: make(map[string]*wcSession),
	}
}

func (b *WalletConnectBackend) Type() BackendType {
	return BackendWalletConnect
}

func (b *WalletConnectBackend) Name() string {
	return WalletConnectName
}

func (b *WalletConnectBackend) Available() bool {
	return true // Always available as a signing option
}

func (b *WalletConnectBackend) RequiresPassword() bool {
	return false
}

func (b *WalletConnectBackend) RequiresHardware() bool {
	return false
}

func (b *WalletConnectBackend) SupportsRemoteSigning() bool {
	return true
}

func (b *WalletConnectBackend) Initialize(ctx context.Context) error {
	// Get project ID from environment
	b.projectID = os.Getenv("LUX_WC_PROJECT_ID")
	if b.projectID == "" {
		// WalletConnect Cloud project ID is optional but recommended
		// Public fallback for development
		b.projectID = "3f44137a4b2e8e5f0c4e8f9a1b2c3d4e" // Placeholder - users should set their own
	}

	// Set up data directory
	if b.dataDir == "" {
		keysDir, err := GetKeysDir()
		if err != nil {
			return err
		}
		b.dataDir = filepath.Join(keysDir, ".walletconnect")
	}

	if err := os.MkdirAll(b.dataDir, 0o700); err != nil {
		return fmt.Errorf("failed to create walletconnect directory: %w", err)
	}

	// Load existing sessions
	return b.loadSessions()
}

func (b *WalletConnectBackend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.done != nil {
		close(b.done)
	}

	if b.conn != nil {
		_ = b.conn.Close()
	}

	// Zero out session keys
	for _, s := range b.sessions {
		for i := range s.SymKey {
			s.SymKey[i] = 0
		}
	}

	return nil
}

// CreateKey is not supported - WalletConnect uses external wallets
func (b *WalletConnectBackend) CreateKey(ctx context.Context, name string, opts CreateKeyOptions) (*HDKeySet, error) {
	return nil, errors.New("walletconnect: key creation not supported, use Pair() to connect mobile wallet")
}

// LoadKey loads session info for a paired wallet
func (b *WalletConnectBackend) LoadKey(ctx context.Context, name, password string) (*HDKeySet, error) {
	b.mu.RLock()
	session, ok := b.sessions[name]
	b.mu.RUnlock()

	if !ok {
		return nil, ErrKeyNotFound
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, ErrWCSessionExpired
	}

	// Return minimal key set with address (no private keys - signing is remote)
	return &HDKeySet{
		Name:      name,
		ECAddress: session.Address,
	}, nil
}

// SaveKey saves session info
func (b *WalletConnectBackend) SaveKey(ctx context.Context, keySet *HDKeySet, password string) error {
	return b.saveSessions()
}

// DeleteKey removes a pairing session
func (b *WalletConnectBackend) DeleteKey(ctx context.Context, name string) error {
	b.mu.Lock()
	if s, ok := b.sessions[name]; ok {
		for i := range s.SymKey {
			s.SymKey[i] = 0
		}
		delete(b.sessions, name)
	}
	b.mu.Unlock()

	return b.saveSessions()
}

// ListKeys returns all paired wallets
func (b *WalletConnectBackend) ListKeys(ctx context.Context) ([]KeyInfo, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	keys := make([]KeyInfo, 0, len(b.sessions))
	for name, session := range b.sessions {
		keys = append(keys, KeyInfo{
			Name:      name,
			Address:   session.Address,
			Encrypted: false,
			Locked:    time.Now().After(session.ExpiresAt),
			CreatedAt: session.PairedAt,
		})
	}
	return keys, nil
}

func (b *WalletConnectBackend) Lock(ctx context.Context, name string) error {
	// No-op for WalletConnect - sessions managed externally
	return nil
}

func (b *WalletConnectBackend) Unlock(ctx context.Context, name, password string) error {
	// Check if session exists and is valid
	b.mu.RLock()
	session, ok := b.sessions[name]
	b.mu.RUnlock()

	if !ok {
		return ErrWCNotPaired
	}

	if time.Now().After(session.ExpiresAt) {
		return ErrWCSessionExpired
	}

	return nil
}

func (b *WalletConnectBackend) IsLocked(name string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	session, ok := b.sessions[name]
	if !ok {
		return true
	}
	return time.Now().After(session.ExpiresAt)
}

// Sign sends a signing request to the connected wallet
func (b *WalletConnectBackend) Sign(ctx context.Context, name string, request SignRequest) (*SignResponse, error) {
	b.mu.RLock()
	session, ok := b.sessions[name]
	b.mu.RUnlock()

	if !ok {
		return nil, ErrWCNotPaired
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, ErrWCSessionExpired
	}

	// Display signing request info
	fmt.Printf("\n=== WalletConnect Signing Request ===\n")
	fmt.Printf("Wallet: %s (%s)\n", name, session.PeerName)
	fmt.Printf("Address: %s\n", session.Address)
	fmt.Printf("Type: %s\n", request.Type)
	fmt.Printf("Chain ID: %d\n", request.ChainID)
	if request.Description != "" {
		fmt.Printf("Description: %s\n", request.Description)
	}
	fmt.Printf("Data Hash: 0x%s\n", hex.EncodeToString(request.DataHash[:]))
	fmt.Printf("\nPlease approve the request in your mobile wallet...\n")
	fmt.Printf("=====================================\n\n")

	// Create JSON-RPC request
	var method string
	var params interface{}

	switch request.Type {
	case "message", "auth":
		// EIP-191 personal_sign: params = [message, address]
		method = wcMethodPersonalSign
		// Message should be hex-encoded with 0x prefix
		msgHex := "0x" + hex.EncodeToString(request.Data)
		params = []string{msgHex, session.Address}

	case "typed_data":
		// EIP-712 eth_signTypedData_v4: params = [address, typedData]
		method = wcMethodSignTypedV4
		params = []interface{}{session.Address, string(request.Data)}

	case "transaction":
		// eth_signTransaction: params = [txObject]
		method = wcMethodSignTx
		params = []json.RawMessage{request.Data}

	default:
		// Default to personal_sign
		method = wcMethodPersonalSign
		msgHex := "0x" + hex.EncodeToString(request.Data)
		params = []string{msgHex, session.Address}
	}

	// Send request and wait for response
	sig, err := b.sendRequest(ctx, session, method, params)
	if err != nil {
		return nil, err
	}

	return &SignResponse{
		Signature: sig,
		Address:   session.Address,
	}, nil
}

// Pair initiates a new WalletConnect pairing session
// Returns the pairing URI that should be displayed as QR code
func (b *WalletConnectBackend) Pair(ctx context.Context, name string, chainID int) (string, error) {
	// Generate random topic and symmetric key
	topic := make([]byte, 32)
	if _, err := rand.Read(topic); err != nil {
		return "", fmt.Errorf("failed to generate topic: %w", err)
	}

	symKey := make([]byte, 32)
	if _, err := rand.Read(symKey); err != nil {
		return "", fmt.Errorf("failed to generate symmetric key: %w", err)
	}

	topicHex := hex.EncodeToString(topic)
	symKeyHex := hex.EncodeToString(symKey)

	// Create pairing URI
	// Format: wc:{topic}@{version}?relay-protocol=irn&symKey={symKey}
	uri := fmt.Sprintf("%s:%s@%s?relay-protocol=irn&symKey=%s",
		wcProtocolID, topicHex, wcVersion, symKeyHex)

	// Create session placeholder
	session := &wcSession{
		Topic:     topicHex,
		SymKey:    symKey,
		ChainID:   chainID,
		PairedAt:  time.Now(),
		ExpiresAt: time.Now().Add(wcSessionExpiry),
	}

	b.mu.Lock()
	b.sessions[name] = session
	b.mu.Unlock()

	return uri, nil
}

// DisplayQR generates and displays a QR code in the terminal
func (b *WalletConnectBackend) DisplayQR(uri string) error {
	// Generate QR code
	qr, err := qrcode.New(uri, qrcode.Medium)
	if err != nil {
		return fmt.Errorf("failed to generate QR code: %w", err)
	}

	// Print QR code to terminal
	fmt.Println("\n=== Scan with your mobile wallet ===")
	fmt.Println(qr.ToSmallString(false))
	fmt.Println("====================================")
	fmt.Printf("\nURI: %s\n\n", uri)

	return nil
}

// WaitForPairing waits for a wallet to connect
func (b *WalletConnectBackend) WaitForPairing(ctx context.Context, name string) (*wcSession, error) {
	b.mu.RLock()
	session, ok := b.sessions[name]
	b.mu.RUnlock()

	if !ok {
		return nil, ErrWCNotPaired
	}

	// Connect to relay
	if err := b.connectRelay(ctx, session.Topic); err != nil {
		return nil, err
	}

	// Wait for session proposal from wallet
	ctx, cancel := context.WithTimeout(ctx, wcPairingTimeout)
	defer cancel()

	fmt.Println("Waiting for wallet to connect...")

	for {
		select {
		case <-ctx.Done():
			return nil, ErrWCTimeout

		default:
			// Read message from relay
			_, message, err := b.conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					return nil, ErrWCDisconnected
				}
				continue
			}

			// Parse session response
			var resp struct {
				Topic   string `json:"topic"`
				Type    string `json:"type"`
				Payload struct {
					Params struct {
						Accounts []string `json:"accounts"`
						PeerMeta struct {
							Name string `json:"name"`
							Icon string `json:"icon"`
						} `json:"peerMeta"`
					} `json:"params"`
				} `json:"payload"`
			}

			if err := json.Unmarshal(message, &resp); err != nil {
				continue
			}

			if resp.Type == "session_proposal" || resp.Type == "session_approval" {
				// Extract address from accounts (format: "eip155:1:0x...")
				if len(resp.Payload.Params.Accounts) > 0 {
					parts := strings.Split(resp.Payload.Params.Accounts[0], ":")
					if len(parts) >= 3 {
						session.Address = parts[2]
					} else {
						session.Address = resp.Payload.Params.Accounts[0]
					}
				}
				session.PeerName = resp.Payload.Params.PeerMeta.Name
				session.PeerIcon = resp.Payload.Params.PeerMeta.Icon

				// Update session
				b.mu.Lock()
				b.sessions[name] = session
				b.mu.Unlock()

				// Save session to disk
				if err := b.saveSessions(); err != nil {
					return nil, err
				}

				fmt.Printf("\nConnected to %s\n", session.PeerName)
				fmt.Printf("Address: %s\n", session.Address)

				return session, nil
			}
		}
	}
}

// connectRelay establishes WebSocket connection to WalletConnect relay
func (b *WalletConnectBackend) connectRelay(ctx context.Context, topic string) error {
	if b.conn != nil {
		return nil // Already connected
	}

	// Build relay URL with project ID
	relayURL := fmt.Sprintf("%s/?projectId=%s", wcRelayURL, b.projectID)

	// Set up WebSocket dialer with timeout
	dialer := websocket.Dialer{
		HandshakeTimeout: wcConnectTimeout,
	}

	// Connect
	conn, resp, err := dialer.DialContext(ctx, relayURL, http.Header{
		"Origin": []string{"https://lux.network"},
	})
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			return fmt.Errorf("relay connection failed: %s: %w", string(body), err)
		}
		return fmt.Errorf("relay connection failed: %w", err)
	}

	b.conn = conn
	b.done = make(chan struct{})

	// Subscribe to topic
	subscribeMsg := map[string]interface{}{
		"id":      time.Now().UnixNano(),
		"jsonrpc": "2.0",
		"method":  "irn_subscribe",
		"params": map[string]string{
			"topic": topic,
		},
	}

	if err := conn.WriteJSON(subscribeMsg); err != nil {
		_ = conn.Close()
		b.conn = nil
		return fmt.Errorf("failed to subscribe to topic: %w", err)
	}

	// Start heartbeat
	go b.heartbeat()

	return nil
}

// heartbeat sends periodic pings to keep connection alive
func (b *WalletConnectBackend) heartbeat() {
	ticker := time.NewTicker(wcHeartbeatPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-b.done:
			return
		case <-ticker.C:
			b.mu.RLock()
			conn := b.conn
			b.mu.RUnlock()

			if conn != nil {
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}
}

// sendRequest sends a JSON-RPC request to the wallet via relay
func (b *WalletConnectBackend) sendRequest(ctx context.Context, session *wcSession, method string, params interface{}) ([]byte, error) {
	// Connect to relay if not connected
	if err := b.connectRelay(ctx, session.Topic); err != nil {
		return nil, err
	}

	// Create request
	reqID := time.Now().UnixNano()
	req := wcRequest{
		ID:      reqID,
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	// Encode request
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	// Send via relay
	publishMsg := map[string]interface{}{
		"id":      time.Now().UnixNano(),
		"jsonrpc": "2.0",
		"method":  "irn_publish",
		"params": map[string]interface{}{
			"topic":   session.Topic,
			"message": hex.EncodeToString(reqData),
			"ttl":     300,  // 5 minutes
			"tag":     1100, // session request tag
		},
	}

	if err := b.conn.WriteJSON(publishMsg); err != nil {
		return nil, fmt.Errorf("failed to publish request: %w", err)
	}

	// Wait for response
	ctx, cancel := context.WithTimeout(ctx, wcSigningTimeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil, ErrWCTimeout

		default:
			// Read response
			_, message, err := b.conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					return nil, ErrWCDisconnected
				}
				continue
			}

			// Parse relay message
			var relayMsg struct {
				ID      int64  `json:"id"`
				JSONRPC string `json:"jsonrpc"`
				Method  string `json:"method"`
				Params  struct {
					Topic   string `json:"topic"`
					Message string `json:"message"`
				} `json:"params"`
			}

			if err := json.Unmarshal(message, &relayMsg); err != nil {
				continue
			}

			if relayMsg.Method != "irn_subscription" {
				continue
			}

			// Decode inner message
			msgData, err := hex.DecodeString(relayMsg.Params.Message)
			if err != nil {
				continue
			}

			var resp wcResponse
			if err := json.Unmarshal(msgData, &resp); err != nil {
				continue
			}

			// Check if this is our response
			if resp.ID != reqID {
				continue
			}

			// Check for error
			if resp.Error != nil {
				if resp.Error.Code == 4001 {
					return nil, ErrWCUserRejected
				}
				return nil, fmt.Errorf("wallet error: %s (code %d)", resp.Error.Message, resp.Error.Code)
			}

			// Parse signature result
			var sigHex string
			if err := json.Unmarshal(resp.Result, &sigHex); err != nil {
				return nil, ErrWCInvalidResponse
			}

			// Decode hex signature
			sigHex = strings.TrimPrefix(sigHex, "0x")
			sig, err := hex.DecodeString(sigHex)
			if err != nil {
				return nil, fmt.Errorf("failed to decode signature: %w", err)
			}

			return sig, nil
		}
	}
}

// loadSessions loads saved sessions from disk
func (b *WalletConnectBackend) loadSessions() error {
	sessionsFile := filepath.Join(b.dataDir, "sessions.json")

	data, err := os.ReadFile(sessionsFile) //nolint:gosec // G304: Reading from app's data directory
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var sessions map[string]*wcSession
	if err := json.Unmarshal(data, &sessions); err != nil {
		return err
	}

	b.mu.Lock()
	b.sessions = sessions
	b.mu.Unlock()

	// Remove expired sessions
	b.mu.Lock()
	for name, session := range b.sessions {
		if time.Now().After(session.ExpiresAt) {
			for i := range session.SymKey {
				session.SymKey[i] = 0
			}
			delete(b.sessions, name)
		}
	}
	b.mu.Unlock()

	return nil
}

// saveSessions saves sessions to disk
func (b *WalletConnectBackend) saveSessions() error {
	b.mu.RLock()
	data, err := json.MarshalIndent(b.sessions, "", "  ")
	b.mu.RUnlock()

	if err != nil {
		return err
	}

	sessionsFile := filepath.Join(b.dataDir, "sessions.json")
	return os.WriteFile(sessionsFile, data, 0o600)
}

// GetSessionChecksum returns a checksum for session verification
func (b *WalletConnectBackend) GetSessionChecksum(name string) (string, error) {
	b.mu.RLock()
	session, ok := b.sessions[name]
	b.mu.RUnlock()

	if !ok {
		return "", ErrWCNotPaired
	}

	h := sha256.Sum256([]byte(session.Topic + session.Address))
	return hex.EncodeToString(h[:8]), nil
}

// SignPersonal signs a message using EIP-191 personal_sign
func (b *WalletConnectBackend) SignPersonal(ctx context.Context, name string, message []byte) ([]byte, error) {
	request := SignRequest{
		Type: "message",
		Data: message,
	}
	copy(request.DataHash[:], sha256Sum(message))

	resp, err := b.Sign(ctx, name, request)
	if err != nil {
		return nil, err
	}
	return resp.Signature, nil
}

// SignTypedData signs typed data using EIP-712
func (b *WalletConnectBackend) SignTypedData(ctx context.Context, name string, typedData []byte) ([]byte, error) {
	request := SignRequest{
		Type: "typed_data",
		Data: typedData,
	}
	copy(request.DataHash[:], sha256Sum(typedData))

	resp, err := b.Sign(ctx, name, request)
	if err != nil {
		return nil, err
	}
	return resp.Signature, nil
}

func sha256Sum(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

func init() {
	RegisterBackend(NewWalletConnectBackend())
}
