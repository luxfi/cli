// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package kms

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Server provides the HTTP API for KMS operations.
// API is compatible with the kms-go SDK client at github.com/luxfi/kms-go
type Server struct {
	kms    *KMS
	mpc    *MPCManager
	addr   string
	server *http.Server
	config *ServerConfig
}

// ServerConfig holds server configuration.
type ServerConfig struct {
	Addr           string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	MaxHeaderBytes int
	CORSOrigins    []string
	APIKey         string // Simple API key authentication
	EnableMPC      bool
	EnableSecrets  bool
}

// DefaultServerConfig returns default server configuration.
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Addr:           ":8200",
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
		CORSOrigins:    []string{"*"},
		EnableMPC:      true,
		EnableSecrets:  true,
	}
}

// NewServer creates a new KMS HTTP server.
func NewServer(kms *KMS, cfg *ServerConfig) *Server {
	if cfg == nil {
		cfg = DefaultServerConfig()
	}

	s := &Server{
		kms:    kms,
		mpc:    NewMPCManager(kms),
		addr:   cfg.Addr,
		config: cfg,
	}

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/v1/health", s.handleHealth)

	// KMS Key management - compatible with kms-go SDK
	mux.HandleFunc("/v1/kms/keys", s.handleKmsKeys)
	mux.HandleFunc("/v1/kms/keys/", s.handleKmsKey)

	// Legacy endpoints (for backwards compatibility)
	mux.HandleFunc("/v1/keys", s.handleKmsKeys)
	mux.HandleFunc("/v1/keys/", s.handleKmsKey)

	// Legacy encryption/signing endpoints
	mux.HandleFunc("/v1/encrypt", s.handleLegacyEncrypt)
	mux.HandleFunc("/v1/decrypt", s.handleLegacyDecrypt)
	mux.HandleFunc("/v1/sign", s.handleLegacySign)
	mux.HandleFunc("/v1/verify", s.handleLegacyVerify)

	// Secrets - v3 API compatible with kms-go SDK
	if cfg.EnableSecrets {
		mux.HandleFunc("/v3/secrets/raw", s.handleSecretsV3)
		mux.HandleFunc("/v3/secrets/raw/", s.handleSecretV3)
		mux.HandleFunc("/v3/secrets/batch/raw", s.handleSecretsBatchV3)
		// Legacy v1 endpoints
		mux.HandleFunc("/v1/secrets", s.handleSecretsV3)
		mux.HandleFunc("/v1/secrets/", s.handleSecretV3)
	}

	// MPC endpoints (if enabled)
	if cfg.EnableMPC {
		mux.HandleFunc("/v1/mpc/nodes", s.handleMPCNodes)
		mux.HandleFunc("/v1/mpc/nodes/", s.handleMPCNode)
		mux.HandleFunc("/v1/mpc/wallets", s.handleMPCWallets)
		mux.HandleFunc("/v1/mpc/wallets/", s.handleMPCWallet)
		mux.HandleFunc("/v1/mpc/sign", s.handleMPCSign)
		mux.HandleFunc("/v1/mpc/signing/", s.handleMPCSigning)
	}

	s.server = &http.Server{
		Addr:           cfg.Addr,
		Handler:        s.middleware(mux),
		ReadTimeout:    cfg.ReadTimeout,
		WriteTimeout:   cfg.WriteTimeout,
		MaxHeaderBytes: cfg.MaxHeaderBytes,
	}

	return s
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// middleware adds common middleware to all requests.
func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS
		origin := r.Header.Get("Origin")
		if origin != "" {
			for _, allowed := range s.config.CORSOrigins {
				if allowed == "*" || allowed == origin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
					break
				}
			}
		}

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// API Key authentication (if configured)
		if s.config.APIKey != "" {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				apiKey = r.Header.Get("Authorization")
				if strings.HasPrefix(apiKey, "Bearer ") {
					apiKey = strings.TrimPrefix(apiKey, "Bearer ")
				}
			}
			if apiKey != s.config.APIKey {
				s.writeError(w, http.StatusUnauthorized, "invalid API key")
				return
			}
		}

		// Content-Type
		w.Header().Set("Content-Type", "application/json")

		next.ServeHTTP(w, r)
	})
}

// Response types matching kms-go SDK expectations

type errorResponse struct {
	Error      string `json:"error"`
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message,omitempty"`
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, errorResponse{
		Error:      message,
		StatusCode: status,
		Message:    message,
	})
}

// Health check handler
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// KmsKey matches kms-go SDK KmsKey struct
type KmsKey struct {
	ID                  string `json:"id"`
	Description         string `json:"description"`
	IsDisabled          bool   `json:"isDisabled"`
	OrgID               string `json:"orgId"`
	Name                string `json:"name"`
	ProjectID           string `json:"projectId"`
	KeyUsage            string `json:"keyUsage"`            // "sign-verify" or "encrypt-decrypt"
	Version             int    `json:"version"`
	EncryptionAlgorithm string `json:"encryptionAlgorithm"` // "rsa-4096", "ecc-nist-p256", "aes-256-gcm", "aes-128-gcm"
}

// Convert internal Key to SDK-compatible KmsKey
func keyToKmsKey(key *Key) KmsKey {
	keyUsage := "encrypt-decrypt"
	if key.Usage == KeyUsageSignVerify {
		keyUsage = "sign-verify"
	}

	encAlg := string(key.Type)
	// Map internal types to SDK-expected format
	switch key.Type {
	case KeyTypeAES256:
		encAlg = "aes-256-gcm"
	case KeyTypeRSA3072, KeyTypeRSA4096:
		encAlg = "rsa-4096"
	case KeyTypeECDSAP256, KeyTypeECDSAP384:
		encAlg = "ecc-nist-p256"
	case KeyTypeEdDSA:
		encAlg = "ed25519"
	}

	return KmsKey{
		ID:                  key.ID,
		Description:         key.Description,
		IsDisabled:          key.Status != KeyStatusActive,
		OrgID:               key.OrgID,
		Name:                key.Name,
		ProjectID:           key.ProjectID,
		KeyUsage:            keyUsage,
		Version:             key.Version,
		EncryptionAlgorithm: encAlg,
	}
}

// KMS Key management handlers - compatible with kms-go SDK

func (s *Server) handleKmsKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case "GET":
		keys, err := s.kms.ListKeys(ctx, "")
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		// Convert to SDK format
		kmsKeys := make([]KmsKey, len(keys))
		for i, key := range keys {
			kmsKeys[i] = keyToKmsKey(key)
		}
		s.writeJSON(w, http.StatusOK, map[string]interface{}{"keys": kmsKeys})

	case "POST":
		// KmsCreateKeyV1Request format
		var req struct {
			KeyUsage            string `json:"keyUsage"`            // "sign-verify" or "encrypt-decrypt"
			Description         string `json:"description"`
			Name                string `json:"name"`
			EncryptionAlgorithm string `json:"encryptionAlgorithm"` // "rsa-4096", "ecc-nist-p256", "aes-256-gcm", "aes-128-gcm"
			ProjectID           string `json:"projectId"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		// Map SDK algorithm names to internal types
		var keyType KeyType
		switch req.EncryptionAlgorithm {
		case "aes-256-gcm":
			keyType = KeyTypeAES256
		case "aes-128-gcm":
			keyType = KeyTypeAES256 // Use AES-256 for both
		case "rsa-4096":
			keyType = KeyTypeRSA4096
		case "ecc-nist-p256":
			keyType = KeyTypeECDSAP256
		default:
			keyType = KeyTypeAES256
		}

		var keyUsage KeyUsage
		if req.KeyUsage == "sign-verify" {
			keyUsage = KeyUsageSignVerify
		} else {
			keyUsage = KeyUsageEncryptDecrypt
		}

		opts := &KeyOptions{
			Description: req.Description,
			ProjectID:   req.ProjectID,
		}

		key, err := s.kms.GenerateKey(ctx, req.Name, keyType, keyUsage, opts)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Return in KmsCreateKeyV1Response format
		s.writeJSON(w, http.StatusCreated, map[string]interface{}{
			"key": keyToKmsKey(key),
		})

	default:
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleKmsKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse path: /v1/kms/keys/{keyId} or /v1/kms/keys/{keyId}/{action}
	// or /v1/kms/keys/name/{keyName}/project/{projectId}
	path := r.URL.Path
	var keyID string

	// Handle different path patterns
	if strings.Contains(path, "/v1/kms/keys/") {
		path = strings.TrimPrefix(path, "/v1/kms/keys/")
	} else if strings.Contains(path, "/v1/keys/") {
		path = strings.TrimPrefix(path, "/v1/keys/")
	}

	parts := strings.Split(path, "/")

	// Check for /name/{keyName}/project/{projectId} pattern
	if len(parts) >= 4 && parts[0] == "name" {
		keyName := parts[1]
		projectID := ""
		if len(parts) >= 4 && parts[2] == "project" {
			projectID = parts[3]
		}
		s.handleGetKeyByName(w, r, keyName, projectID)
		return
	}

	keyID = parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	// Route to appropriate handler based on action
	switch action {
	case "encrypt":
		s.handleKeyEncrypt(w, r, keyID)
	case "decrypt":
		s.handleKeyDecrypt(w, r, keyID)
	case "sign":
		s.handleKeySign(w, r, keyID)
	case "verify":
		s.handleKeyVerify(w, r, keyID)
	case "public-key":
		s.handleKeyPublicKey(w, r, keyID)
	case "signing-algorithms":
		s.handleKeySigningAlgorithms(w, r, keyID)
	case "":
		// Direct key operations
		switch r.Method {
		case "GET":
			key, err := s.kms.GetKey(ctx, keyID)
			if err == ErrKeyNotFound {
				s.writeError(w, http.StatusNotFound, "key not found")
				return
			}
			if err != nil {
				s.writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			s.writeJSON(w, http.StatusOK, map[string]interface{}{
				"key": keyToKmsKey(key),
			})

		case "DELETE":
			key, err := s.kms.GetKey(ctx, keyID)
			if err == ErrKeyNotFound {
				s.writeError(w, http.StatusNotFound, "key not found")
				return
			}
			if err != nil {
				s.writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			kmsKey := keyToKmsKey(key)

			if err := s.kms.DeleteKey(ctx, keyID); err != nil {
				s.writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			// KmsDeleteKeyV1Response format
			s.writeJSON(w, http.StatusOK, map[string]interface{}{
				"key": kmsKey,
			})

		default:
			s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	default:
		s.writeError(w, http.StatusNotFound, "unknown action")
	}
}

func (s *Server) handleGetKeyByName(w http.ResponseWriter, r *http.Request, keyName, projectID string) {
	if r.Method != "GET" {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()

	// List keys and find by name
	keys, err := s.kms.ListKeys(ctx, "")
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	for _, key := range keys {
		if key.Name == keyName && (projectID == "" || key.ProjectID == projectID) {
			s.writeJSON(w, http.StatusOK, map[string]interface{}{
				"key": keyToKmsKey(key),
			})
			return
		}
	}

	s.writeError(w, http.StatusNotFound, "key not found")
}

func (s *Server) handleKeyEncrypt(w http.ResponseWriter, r *http.Request, keyID string) {
	if r.Method != "POST" {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()

	// KmsEncryptDataV1Request format
	var req struct {
		Plaintext string `json:"plaintext"` // Base64 encoded
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	plaintext, err := DecodeBase64(req.Plaintext)
	if err != nil {
		// Try as raw string
		plaintext = []byte(req.Plaintext)
	}

	ciphertext, err := s.kms.Encrypt(ctx, keyID, plaintext)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// KmsEncryptDataV1Response format
	s.writeJSON(w, http.StatusOK, map[string]string{
		"ciphertext": EncodeBase64(ciphertext),
	})
}

func (s *Server) handleKeyDecrypt(w http.ResponseWriter, r *http.Request, keyID string) {
	if r.Method != "POST" {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()

	// KmsDecryptDataV1Request format
	var req struct {
		Ciphertext string `json:"ciphertext"` // Base64 encoded
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ciphertext, err := DecodeBase64(req.Ciphertext)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid ciphertext encoding")
		return
	}

	plaintext, err := s.kms.Decrypt(ctx, ciphertext)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// KmsDecryptDataV1Response format
	s.writeJSON(w, http.StatusOK, map[string]string{
		"plaintext": EncodeBase64(plaintext),
	})
}

func (s *Server) handleKeySign(w http.ResponseWriter, r *http.Request, keyID string) {
	if r.Method != "POST" {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()

	// KmsSignDataV1Request format
	var req struct {
		Data             string `json:"data"`             // Base64 encoded
		SigningAlgorithm string `json:"signingAlgorithm"` // e.g., "RSASSA_PKCS1_V1_5_SHA_256"
		IsDigest         bool   `json:"isDigest"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	data, err := DecodeBase64(req.Data)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid data encoding")
		return
	}

	signature, err := s.kms.Sign(ctx, keyID, data)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// KmsSignDataV1Response format
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"signature":        EncodeBase64(signature),
		"keyId":            keyID,
		"signingAlgorithm": req.SigningAlgorithm,
	})
}

func (s *Server) handleKeyVerify(w http.ResponseWriter, r *http.Request, keyID string) {
	if r.Method != "POST" {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()

	// KmsVerifyDataV1Request format
	var req struct {
		Data             string `json:"data"`             // Base64 encoded
		Signature        string `json:"signature"`        // Base64 encoded
		SigningAlgorithm string `json:"signingAlgorithm"`
		IsDigest         bool   `json:"isDigest"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	data, err := DecodeBase64(req.Data)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid data encoding")
		return
	}

	signature, err := DecodeBase64(req.Signature)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid signature encoding")
		return
	}

	valid, err := s.kms.Verify(ctx, keyID, data, signature)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// KmsVerifyDataV1Response format
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"signatureValid":   valid,
		"keyId":            keyID,
		"signingAlgorithm": req.SigningAlgorithm,
	})
}

func (s *Server) handleKeyPublicKey(w http.ResponseWriter, r *http.Request, keyID string) {
	if r.Method != "GET" {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()

	// Use GetPublicKey which properly retrieves from KeyMaterial
	publicKey, err := s.kms.GetPublicKey(ctx, keyID)
	if err == ErrKeyNotFound {
		s.writeError(w, http.StatusNotFound, "key not found")
		return
	}
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// KmsGetPublicKeyV1Response format
	s.writeJSON(w, http.StatusOK, map[string]string{
		"publicKey": EncodeBase64(publicKey),
	})
}

func (s *Server) handleKeySigningAlgorithms(w http.ResponseWriter, r *http.Request, keyID string) {
	if r.Method != "GET" {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()

	key, err := s.kms.GetKey(ctx, keyID)
	if err == ErrKeyNotFound {
		s.writeError(w, http.StatusNotFound, "key not found")
		return
	}
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return algorithms based on key type
	var algorithms []string
	switch key.Type {
	case KeyTypeRSA3072, KeyTypeRSA4096:
		algorithms = []string{
			"RSASSA_PKCS1_V1_5_SHA_256",
			"RSASSA_PKCS1_V1_5_SHA_384",
			"RSASSA_PKCS1_V1_5_SHA_512",
			"RSASSA_PSS_SHA_256",
			"RSASSA_PSS_SHA_384",
			"RSASSA_PSS_SHA_512",
		}
	case KeyTypeECDSAP256:
		algorithms = []string{"ECDSA_SHA_256"}
	case KeyTypeECDSAP384:
		algorithms = []string{"ECDSA_SHA_384"}
	case KeyTypeEdDSA:
		algorithms = []string{"EDDSA"}
	default:
		algorithms = []string{}
	}

	// KmsListSigningAlgorithmsV1Response format
	s.writeJSON(w, http.StatusOK, map[string][]string{
		"signingAlgorithms": algorithms,
	})
}

// Legacy encryption handlers (backwards compatibility)

func (s *Server) handleLegacyEncrypt(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		KeyID     string `json:"keyId"`
		Plaintext string `json:"plaintext"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	s.handleKeyEncrypt(w, r, req.KeyID)
}

func (s *Server) handleLegacyDecrypt(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		KeyID      string `json:"keyId"`
		Ciphertext string `json:"ciphertext"`
	}

	body, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(body, &req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Re-create request with just ciphertext for the handler
	ctx := r.Context()
	ciphertext, err := DecodeBase64(req.Ciphertext)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid ciphertext encoding")
		return
	}

	plaintext, err := s.kms.Decrypt(ctx, ciphertext)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{
		"plaintext": EncodeBase64(plaintext),
	})
}

func (s *Server) handleLegacySign(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		KeyID string `json:"keyId"`
		Data  string `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	s.handleKeySign(w, r, req.KeyID)
}

func (s *Server) handleLegacyVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		KeyID     string `json:"keyId"`
		Data      string `json:"data"`
		Signature string `json:"signature"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	s.handleKeyVerify(w, r, req.KeyID)
}

// Secret response matching kms-go SDK Secret model
type SecretResponse struct {
	ID          string `json:"id"`
	SecretKey   string `json:"secretKey"`
	SecretValue string `json:"secretValue,omitempty"`
	Version     int    `json:"version"`
	Type        string `json:"type"`
	Environment string `json:"environment"`
	SecretPath  string `json:"secretPath"`
}

func secretToResponse(sec *Secret, includeValue bool, decryptedValue []byte) SecretResponse {
	resp := SecretResponse{
		ID:          sec.ID,
		SecretKey:   sec.Name,
		Version:     sec.Version,
		Type:        "shared", // Default type
		Environment: sec.Environment,
		SecretPath:  sec.Path,
	}
	if includeValue && decryptedValue != nil {
		resp.SecretValue = string(decryptedValue)
	}
	return resp
}

// Secrets V3 handlers - compatible with kms-go SDK

func (s *Server) handleSecretsV3(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case "GET":
		// ListSecretsV3RawRequest query params
		projectID := r.URL.Query().Get("workspaceId")
		if projectID == "" {
			projectID = r.URL.Query().Get("workspaceSlug")
		}
		environment := r.URL.Query().Get("environment")
		secretPath := r.URL.Query().Get("secretPath")
		if secretPath == "" {
			secretPath = "/"
		}

		secrets, err := s.kms.ListSecrets(ctx, environment, secretPath)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Convert to SDK response format
		secretResponses := make([]SecretResponse, len(secrets))
		for i, sec := range secrets {
			// Filter by project if specified
			if projectID != "" && sec.ProjectID != projectID {
				continue
			}
			secretResponses[i] = secretToResponse(sec, false, nil)
		}

		// ListSecretsV3RawResponse format
		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"secrets": secretResponses,
			"imports": []interface{}{},
		})

	default:
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleSecretV3(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse secret key from path
	path := r.URL.Path
	var secretKey string
	if strings.Contains(path, "/v3/secrets/raw/") {
		secretKey = strings.TrimPrefix(path, "/v3/secrets/raw/")
	} else if strings.Contains(path, "/v1/secrets/") {
		secretKey = strings.TrimPrefix(path, "/v1/secrets/")
	}

	// Handle /value suffix for getting decrypted value
	getValue := false
	if strings.HasSuffix(secretKey, "/value") {
		secretKey = strings.TrimSuffix(secretKey, "/value")
		getValue = true
	}

	switch r.Method {
	case "GET":
		// RetrieveSecretV3RawRequest query params
		projectID := r.URL.Query().Get("workspaceId")
		if projectID == "" {
			projectID = r.URL.Query().Get("workspaceSlug")
		}
		environment := r.URL.Query().Get("environment")
		secretPath := r.URL.Query().Get("secretPath")

		// Find secret by name
		secrets, err := s.kms.ListSecrets(ctx, environment, secretPath)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		for _, sec := range secrets {
			if sec.Name == secretKey && (projectID == "" || sec.ProjectID == projectID) {
				var value []byte
				if getValue {
					value, err = s.kms.GetSecretValue(ctx, sec.ID)
					if err != nil {
						s.writeError(w, http.StatusInternalServerError, err.Error())
						return
					}
				}
				// RetrieveSecretV3RawResponse format
				s.writeJSON(w, http.StatusOK, map[string]interface{}{
					"secret": secretToResponse(sec, getValue, value),
				})
				return
			}
		}

		s.writeError(w, http.StatusNotFound, "secret not found")

	case "POST":
		// CreateSecretV3RawRequest format
		var req struct {
			ProjectID             string `json:"workspaceId"`
			Environment           string `json:"environment"`
			SecretPath            string `json:"secretPath"`
			Type                  string `json:"type"`
			SecretComment         string `json:"secretComment"`
			SkipMultiLineEncoding bool   `json:"skipMultilineEncoding"`
			SecretValue           string `json:"secretValue"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		opts := &SecretOptions{
			Environment: req.Environment,
			Path:        req.SecretPath,
			ProjectID:   req.ProjectID,
		}

		secret, err := s.kms.CreateSecret(ctx, secretKey, []byte(req.SecretValue), opts)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// CreateSecretV3RawResponse format
		s.writeJSON(w, http.StatusCreated, map[string]interface{}{
			"secret": secretToResponse(secret, false, nil),
		})

	case "PATCH":
		// UpdateSecretV3RawRequest format
		var req struct {
			ProjectID                string `json:"workspaceId"`
			Environment              string `json:"environment"`
			SecretPath               string `json:"secretPath"`
			Type                     string `json:"type"`
			NewSecretValue           string `json:"secretValue"`
			NewSkipMultilineEncoding bool   `json:"skipMultilineEncoding"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		// Find and update secret
		secrets, err := s.kms.ListSecrets(ctx, req.Environment, req.SecretPath)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		for _, sec := range secrets {
			if sec.Name == secretKey && (req.ProjectID == "" || sec.ProjectID == req.ProjectID) {
				updatedSecret, err := s.kms.UpdateSecret(ctx, sec.ID, []byte(req.NewSecretValue))
				if err != nil {
					s.writeError(w, http.StatusInternalServerError, err.Error())
					return
				}
				// UpdateSecretV3RawResponse format
				s.writeJSON(w, http.StatusOK, map[string]interface{}{
					"secret": secretToResponse(updatedSecret, false, nil),
				})
				return
			}
		}

		s.writeError(w, http.StatusNotFound, "secret not found")

	case "DELETE":
		// DeleteSecretV3RawRequest query/body params
		projectID := r.URL.Query().Get("workspaceId")
		environment := r.URL.Query().Get("environment")
		secretPath := r.URL.Query().Get("secretPath")

		// Find and delete secret
		secrets, err := s.kms.ListSecrets(ctx, environment, secretPath)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		for _, sec := range secrets {
			if sec.Name == secretKey && (projectID == "" || sec.ProjectID == projectID) {
				resp := secretToResponse(sec, false, nil)
				if err := s.kms.DeleteSecret(ctx, sec.ID); err != nil {
					s.writeError(w, http.StatusInternalServerError, err.Error())
					return
				}
				// DeleteSecretV3RawResponse format
				s.writeJSON(w, http.StatusOK, map[string]interface{}{
					"secret": resp,
				})
				return
			}
		}

		s.writeError(w, http.StatusNotFound, "secret not found")

	default:
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleSecretsBatchV3(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()

	// BatchCreateSecretsV3RawRequest format
	var req struct {
		Environment string `json:"environment"`
		ProjectID   string `json:"workspaceId"`
		SecretPath  string `json:"secretPath"`
		Secrets     []struct {
			SecretKey             string `json:"secretKey"`
			SecretValue           string `json:"secretValue"`
			SecretComment         string `json:"secretComment"`
			SkipMultiLineEncoding bool   `json:"skipMultilineEncoding"`
		} `json:"secrets"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	createdSecrets := make([]SecretResponse, 0, len(req.Secrets))
	for _, secReq := range req.Secrets {
		opts := &SecretOptions{
			Environment: req.Environment,
			Path:        req.SecretPath,
			ProjectID:   req.ProjectID,
		}

		secret, err := s.kms.CreateSecret(ctx, secReq.SecretKey, []byte(secReq.SecretValue), opts)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create secret %s: %s", secReq.SecretKey, err.Error()))
			return
		}
		createdSecrets = append(createdSecrets, secretToResponse(secret, false, nil))
	}

	// BatchCreateSecretsV3RawResponse format
	s.writeJSON(w, http.StatusCreated, map[string]interface{}{
		"secrets": createdSecrets,
	})
}

// MPC handlers

func (s *Server) handleMPCNodes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case "GET":
		nodes, err := s.mpc.ListNodes(ctx)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.writeJSON(w, http.StatusOK, map[string]interface{}{"nodes": nodes})

	case "POST":
		var req struct {
			Name      string            `json:"name"`
			Endpoint  string            `json:"endpoint"`
			Port      int               `json:"port"`
			PublicKey string            `json:"publicKey"` // Base64
			OrgID     string            `json:"orgId,omitempty"`
			Metadata  map[string]string `json:"metadata,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		publicKey, err := DecodeBase64(req.PublicKey)
		if err != nil {
			publicKey = []byte(req.PublicKey)
		}

		opts := &NodeOptions{
			OrgID:    req.OrgID,
			Metadata: req.Metadata,
		}

		node, err := s.mpc.RegisterNode(ctx, req.Name, req.Endpoint, req.Port, publicKey, opts)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.writeJSON(w, http.StatusCreated, map[string]interface{}{"node": node})

	default:
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleMPCNode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	nodeID := strings.TrimPrefix(r.URL.Path, "/v1/mpc/nodes/")

	switch r.Method {
	case "GET":
		node, err := s.mpc.GetNode(ctx, nodeID)
		if err == ErrKeyNotFound {
			s.writeError(w, http.StatusNotFound, "node not found")
			return
		}
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.writeJSON(w, http.StatusOK, map[string]interface{}{"node": node})

	default:
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleMPCWallets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case "GET":
		wallets, err := s.mpc.ListWallets(ctx)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.writeJSON(w, http.StatusOK, map[string]interface{}{"wallets": wallets})

	case "POST":
		var req struct {
			Name           string            `json:"name"`
			KeyType        MPCKeyType        `json:"keyType"`
			Threshold      int               `json:"threshold"`
			TotalParties   int               `json:"totalParties"`
			ParticipantIDs []string          `json:"participantIds"`
			OrgID          string            `json:"orgId,omitempty"`
			ProjectID      string            `json:"projectId,omitempty"`
			Metadata       map[string]string `json:"metadata,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		opts := &WalletOptions{
			OrgID:     req.OrgID,
			ProjectID: req.ProjectID,
			Metadata:  req.Metadata,
		}

		wallet, err := s.mpc.CreateWallet(ctx, req.Name, req.KeyType, req.Threshold, req.TotalParties, req.ParticipantIDs, opts)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.writeJSON(w, http.StatusCreated, map[string]interface{}{"wallet": wallet})

	default:
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleMPCWallet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	walletID := strings.TrimPrefix(r.URL.Path, "/v1/mpc/wallets/")

	switch r.Method {
	case "GET":
		wallet, err := s.mpc.GetWallet(ctx, walletID)
		if err == ErrKeyNotFound {
			s.writeError(w, http.StatusNotFound, "wallet not found")
			return
		}
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.writeJSON(w, http.StatusOK, map[string]interface{}{"wallet": wallet})

	default:
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleMPCSign(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()

	var req struct {
		WalletID       string            `json:"walletId"`
		Chain          MPCChain          `json:"chain"`
		RawTransaction string            `json:"rawTransaction"` // Base64
		Message        string            `json:"message,omitempty"`
		Metadata       map[string]string `json:"metadata,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rawTx, err := DecodeBase64(req.RawTransaction)
	if err != nil {
		rawTx = []byte(req.RawTransaction)
	}

	opts := &SigningOptions{
		Metadata: req.Metadata,
	}

	if req.Message != "" {
		opts.Message, _ = DecodeBase64(req.Message)
	}

	sigReq, err := s.mpc.CreateSigningRequest(ctx, req.WalletID, req.Chain, rawTx, opts)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.writeJSON(w, http.StatusCreated, map[string]interface{}{"signingRequest": sigReq})
}

func (s *Server) handleMPCSigning(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	path := strings.TrimPrefix(r.URL.Path, "/v1/mpc/signing/")

	// Check for /signature suffix
	if strings.Contains(path, "/signature") {
		requestID := strings.Split(path, "/")[0]

		if r.Method == "POST" {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				s.writeError(w, http.StatusBadRequest, "failed to read body")
				return
			}

			var req struct {
				NodeID           string `json:"nodeId"`
				PartialSignature string `json:"partialSignature"` // Base64
			}

			if err := json.Unmarshal(body, &req); err != nil {
				s.writeError(w, http.StatusBadRequest, "invalid request body")
				return
			}

			sig, err := DecodeBase64(req.PartialSignature)
			if err != nil {
				s.writeError(w, http.StatusBadRequest, "invalid signature encoding")
				return
			}

			sigReq, err := s.mpc.SubmitPartialSignature(ctx, requestID, req.NodeID, sig)
			if err != nil {
				s.writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			s.writeJSON(w, http.StatusOK, map[string]interface{}{"signingRequest": sigReq})
			return
		}
	}

	requestID := path

	switch r.Method {
	case "GET":
		sigReq, err := s.mpc.GetSigningRequest(ctx, requestID)
		if err == ErrKeyNotFound {
			s.writeError(w, http.StatusNotFound, "signing request not found")
			return
		}
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.writeJSON(w, http.StatusOK, map[string]interface{}{"signingRequest": sigReq})

	default:
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
