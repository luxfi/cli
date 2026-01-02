// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package keycmd

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

// Post-quantum indicator suffix
const pqSuffix = " [PQ]"

var (
	kchainEndpoint    string
	kchainThreshold   int
	kchainTotalShares int
	kchainValidators  []string
	kchainAlgorithm   string
	kchainFormat      string
	kchainSecureWipe  bool
)

func newKChainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kchain",
		Short: "K-Chain distributed key management",
		Long: `K-Chain provides distributed key management using threshold cryptography.

Keys are split across multiple validators using Shamir Secret Sharing,
requiring a threshold of shares to reconstruct or sign.

Features:
  - Distributed key storage across validators
  - Threshold signing without key reconstruction
  - Proactive secret resharing
  - ML-KEM post-quantum encryption
  - ML-DSA post-quantum signatures

Default port range: 963N (9630-9639)

Examples:
  lux key kchain status                    # Check K-Chain service status
  lux key kchain distribute mykey          # Distribute key to validators
  lux key kchain sign mykey "data"         # Threshold sign data
  lux key kchain encrypt mykey "plaintext" # Encrypt with ML-KEM
  lux key kchain algorithms                # List supported algorithms`,
		RunE: cobrautils.CommandSuiteUsage,
	}

	// Add persistent flags for endpoint
	cmd.PersistentFlags().StringVar(&kchainEndpoint, "endpoint", "http://localhost:9630", "K-Chain RPC endpoint")

	// Add subcommands
	cmd.AddCommand(newKChainStatusCmd())
	cmd.AddCommand(newKChainDistributeCmd())
	cmd.AddCommand(newKChainGatherCmd())
	cmd.AddCommand(newKChainSignCmd())
	cmd.AddCommand(newKChainVerifyCmd())
	cmd.AddCommand(newKChainEncryptCmd())
	cmd.AddCommand(newKChainDecryptCmd())
	cmd.AddCommand(newKChainReshareCmd())
	cmd.AddCommand(newKChainAlgorithmsCmd())
	cmd.AddCommand(newKChainListCmd())
	cmd.AddCommand(newKChainShowCmd())
	cmd.AddCommand(newKChainCreateCmd())
	cmd.AddCommand(newKChainDeleteCmd())

	return cmd
}

func newKChainStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check K-Chain service status",
		Long:  `Check the health and status of the K-Chain distributed key management service.`,
		Args:  cobra.NoArgs,
		RunE:  runKChainStatus,
	}
}

func runKChainStatus(_ *cobra.Command, _ []string) error {
	client := key.NewKChainRPCClient(kchainEndpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health, err := client.Health(ctx)
	if err != nil {
		ux.Logger.PrintToUser("K-Chain service: UNAVAILABLE")
		ux.Logger.PrintToUser("  Endpoint: %s", kchainEndpoint)
		ux.Logger.PrintToUser("  Error: %v", err)
		return nil
	}

	statusIcon := "✓"
	status := "healthy"
	if !health.Healthy {
		statusIcon = "✗"
		status = "unhealthy"
	}

	ux.Logger.PrintToUser("K-Chain Service Status")
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("  %s Status:     %s", statusIcon, status)
	ux.Logger.PrintToUser("  Endpoint:    %s", kchainEndpoint)
	ux.Logger.PrintToUser("  Version:     %s", health.Version)
	ux.Logger.PrintToUser("  Uptime:      %ds", health.Uptime)
	ux.Logger.PrintToUser("  Validators:  %d", len(health.Validators))

	if len(health.Validators) > 0 {
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("  Validator Status:")
		for v, healthy := range health.Validators {
			vStatus := "✓"
			if !healthy {
				vStatus = "✗"
			}
			latency := ""
			if l, ok := health.Latency[v]; ok {
				latency = fmt.Sprintf(" (%dms)", l)
			}
			ux.Logger.PrintToUser("    %s %s%s", vStatus, v, latency)
		}
	}
	ux.Logger.PrintToUser("")

	return nil
}

func newKChainDistributeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "distribute <key-name>",
		Short: "Distribute key to validators",
		Long: `Distribute a key across K-Chain validators using Shamir Secret Sharing.

The key is split into shares, each stored on a different validator.
A threshold number of shares is required to reconstruct or sign.

Examples:
  lux key kchain distribute mykey                    # Use defaults (3-of-5)
  lux key kchain distribute mykey -t 2 -n 3          # 2-of-3 threshold
  lux key kchain distribute mykey --validators v1:9630,v2:9631,v3:9632`,
		Args: cobra.ExactArgs(1),
		RunE: runKChainDistribute,
	}

	cmd.Flags().IntVarP(&kchainThreshold, "threshold", "t", 3, "Number of shares required to reconstruct")
	cmd.Flags().IntVarP(&kchainTotalShares, "shares", "n", 5, "Total number of shares to create")
	cmd.Flags().StringSliceVar(&kchainValidators, "validators", nil, "Validator endpoints (host:port)")

	return cmd
}

func runKChainDistribute(_ *cobra.Command, args []string) error {
	keyName := args[0]

	client := key.NewKChainRPCClient(kchainEndpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// First check if key exists
	keyMeta, err := client.GetKeyByName(ctx, keyName)
	if err != nil {
		return fmt.Errorf("key '%s' not found: %w", keyName, err)
	}

	params := key.DistributeKeyParams{
		KeyID:      keyMeta.ID,
		Threshold:  kchainThreshold,
		TotalParts: kchainTotalShares,
		Validators: kchainValidators,
	}

	ux.Logger.PrintToUser("Distributing key '%s' to validators...", keyName)
	ux.Logger.PrintToUser("  Threshold: %d-of-%d", kchainThreshold, kchainTotalShares)

	result, err := client.DistributeKey(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to distribute key: %w", err)
	}

	ux.Logger.PrintToUser("")
	if result.Success {
		ux.Logger.PrintToUser("Key distributed successfully!")
		ux.Logger.PrintToUser("  Shares:      %d", len(result.ShareIDs))
		if result.GroupPublicKey != "" {
			ux.Logger.PrintToUser("  Group Key:   %s...", result.GroupPublicKey[:32])
		}
	} else {
		ux.Logger.PrintToUser("Key distribution failed.")
	}
	ux.Logger.PrintToUser("")

	return nil
}

func newKChainGatherCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gather <key-name>",
		Short: "Gather shares from validators",
		Long: `Gather threshold shares from validators to reconstruct a key.

This command contacts validators to retrieve shares and reconstructs
the original key material. Requires threshold number of responsive validators.

Examples:
  lux key kchain gather mykey`,
		Args: cobra.ExactArgs(1),
		RunE: runKChainGather,
	}

	return cmd
}

func runKChainGather(_ *cobra.Command, args []string) error {
	keyName := args[0]

	client := key.NewKChainRPCClient(kchainEndpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Get key metadata first
	keyMeta, err := client.GetKeyByName(ctx, keyName)
	if err != nil {
		return fmt.Errorf("key '%s' not found: %w", keyName, err)
	}

	params := key.GatherSharesParams{
		KeyID: keyMeta.ID,
	}

	ux.Logger.PrintToUser("Gathering shares for key '%s'...", keyName)

	result, err := client.GatherShares(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to gather shares: %w", err)
	}

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Share Status:")
	ux.Logger.PrintToUser("  Available:  %d", result.Available)
	ux.Logger.PrintToUser("  Required:   %d", result.Required)
	ux.Logger.PrintToUser("  Ready:      %t", result.Ready)
	if len(result.ShareIDs) > 0 {
		ux.Logger.PrintToUser("  Share IDs:  %s", strings.Join(result.ShareIDs, ", "))
	}
	ux.Logger.PrintToUser("")

	return nil
}

func newKChainSignCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sign <key-name> <data>",
		Short: "Threshold sign data",
		Long: `Sign data using threshold signatures without reconstructing the key.

Each validator computes a partial signature using their share.
Partial signatures are combined to produce the final signature.

Examples:
  lux key kchain sign mykey "message to sign"
  lux key kchain sign mykey --hex 48656c6c6f
  lux key kchain sign mykey --algorithm bls-threshold "data"`,
		Args: cobra.RangeArgs(1, 2),
		RunE: runKChainSign,
	}

	cmd.Flags().StringVarP(&kchainAlgorithm, "algorithm", "a", "bls-threshold", "Signing algorithm")
	cmd.Flags().Bool("hex", false, "Interpret data as hex-encoded")

	return cmd
}

func runKChainSign(cmd *cobra.Command, args []string) error {
	keyName := args[0]

	var data []byte
	if len(args) > 1 {
		isHex, _ := cmd.Flags().GetBool("hex")
		if isHex {
			var err error
			data, err = hex.DecodeString(args[1])
			if err != nil {
				return fmt.Errorf("invalid hex data: %w", err)
			}
		} else {
			data = []byte(args[1])
		}
	} else {
		// Read from stdin
		var err error
		data, err = os.ReadFile("/dev/stdin")
		if err != nil {
			return fmt.Errorf("failed to read data from stdin: %w", err)
		}
	}

	client := key.NewKChainRPCClient(kchainEndpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Get key metadata
	keyMeta, err := client.GetKeyByName(ctx, keyName)
	if err != nil {
		return fmt.Errorf("key '%s' not found: %w", keyName, err)
	}

	params := key.ThresholdSignParams{
		KeyID:     keyMeta.ID,
		Message:   base64.StdEncoding.EncodeToString(data),
		Algorithm: kchainAlgorithm,
	}

	ux.Logger.PrintToUser("Requesting threshold signature for '%s'...", keyName)

	result, err := client.ThresholdSign(ctx, params)
	if err != nil {
		return fmt.Errorf("threshold signing failed: %w", err)
	}

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Signature:    %s", result.Signature)
	ux.Logger.PrintToUser("Participants: %d", len(result.ParticipantIDs))
	if result.GroupPublicKey != "" {
		ux.Logger.PrintToUser("Group Key:    %s...", result.GroupPublicKey[:32])
	}
	ux.Logger.PrintToUser("")

	return nil
}

func newKChainVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify <key-name> <data> <signature>",
		Short: "Verify a signature",
		Long: `Verify a signature against the key's public key.

Examples:
  lux key kchain verify mykey "message" <signature>`,
		Args: cobra.ExactArgs(3),
		RunE: runKChainVerify,
	}

	cmd.Flags().StringVarP(&kchainAlgorithm, "algorithm", "a", "bls-threshold", "Signature algorithm")

	return cmd
}

func runKChainVerify(_ *cobra.Command, args []string) error {
	keyName := args[0]
	data := []byte(args[1])
	signature := args[2]

	client := key.NewKChainRPCClient(kchainEndpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	keyMeta, err := client.GetKeyByName(ctx, keyName)
	if err != nil {
		return fmt.Errorf("key '%s' not found: %w", keyName, err)
	}

	params := key.VerifyParams{
		KeyID:     keyMeta.ID,
		Message:   base64.StdEncoding.EncodeToString(data),
		Signature: signature,
		Algorithm: kchainAlgorithm,
	}

	result, err := client.Verify(ctx, params)
	if err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	if result.Valid {
		ux.Logger.PrintToUser("✓ Signature is VALID")
	} else {
		ux.Logger.PrintToUser("✗ Signature is INVALID")
		if result.Message != "" {
			ux.Logger.PrintToUser("  Reason: %s", result.Message)
		}
	}

	return nil
}

func newKChainEncryptCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "encrypt <key-name> <plaintext>",
		Short: "Encrypt data with ML-KEM",
		Long: `Encrypt data using the key's ML-KEM public key.

ML-KEM (Module-Lattice Key Encapsulation Mechanism) provides
post-quantum secure encryption.

Examples:
  lux key kchain encrypt mykey "secret message"
  lux key kchain encrypt mykey --algorithm ml-kem-768 "data"`,
		Args: cobra.ExactArgs(2),
		RunE: runKChainEncrypt,
	}

	cmd.Flags().StringVarP(&kchainAlgorithm, "algorithm", "a", "ml-kem-768", "Encryption algorithm")

	return cmd
}

func runKChainEncrypt(_ *cobra.Command, args []string) error {
	keyName := args[0]
	plaintext := args[1]

	client := key.NewKChainRPCClient(kchainEndpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	keyMeta, err := client.GetKeyByName(ctx, keyName)
	if err != nil {
		return fmt.Errorf("key '%s' not found: %w", keyName, err)
	}

	params := key.EncryptParams{
		KeyID:     keyMeta.ID,
		Plaintext: base64.StdEncoding.EncodeToString([]byte(plaintext)),
	}

	result, err := client.Encrypt(ctx, params)
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}

	ux.Logger.PrintToUser("Ciphertext: %s", result.Ciphertext)
	if result.Nonce != "" {
		ux.Logger.PrintToUser("Nonce:      %s", result.Nonce)
	}
	if result.Tag != "" {
		ux.Logger.PrintToUser("Tag:        %s", result.Tag)
	}

	return nil
}

func newKChainDecryptCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "decrypt <key-name> <ciphertext>",
		Short: "Decrypt data with threshold reconstruction",
		Long: `Decrypt data using threshold key reconstruction.

Requires gathering shares from validators to reconstruct the
decryption key. The key is immediately cleared after decryption.

Examples:
  lux key kchain decrypt mykey <ciphertext>`,
		Args: cobra.ExactArgs(2),
		RunE: runKChainDecrypt,
	}

	return cmd
}

func runKChainDecrypt(_ *cobra.Command, args []string) error {
	keyName := args[0]
	ciphertext := args[1]

	client := key.NewKChainRPCClient(kchainEndpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	keyMeta, err := client.GetKeyByName(ctx, keyName)
	if err != nil {
		return fmt.Errorf("key '%s' not found: %w", keyName, err)
	}

	params := key.DecryptParams{
		KeyID:      keyMeta.ID,
		Ciphertext: ciphertext,
	}

	ux.Logger.PrintToUser("Decrypting with threshold reconstruction...")

	result, err := client.Decrypt(ctx, params)
	if err != nil {
		return fmt.Errorf("decryption failed: %w", err)
	}

	// Decode plaintext
	plaintext, err := base64.StdEncoding.DecodeString(result.Plaintext)
	if err != nil {
		return fmt.Errorf("failed to decode plaintext: %w", err)
	}

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Plaintext: %s", string(plaintext))

	return nil
}

func newKChainReshareCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reshare <key-name>",
		Short: "Proactive secret resharing",
		Long: `Perform proactive secret resharing to rotate key shares.

This creates new shares without changing the underlying key,
limiting the window of exposure if any shares are compromised.

Examples:
  lux key kchain reshare mykey
  lux key kchain reshare mykey -t 4 -n 7   # Change to 4-of-7`,
		Args: cobra.ExactArgs(1),
		RunE: runKChainReshare,
	}

	cmd.Flags().IntVarP(&kchainThreshold, "threshold", "t", 0, "New threshold (0 = keep current)")
	cmd.Flags().IntVarP(&kchainTotalShares, "shares", "n", 0, "New total shares (0 = keep current)")
	cmd.Flags().StringSliceVar(&kchainValidators, "validators", nil, "New validator set")

	return cmd
}

func runKChainReshare(_ *cobra.Command, args []string) error {
	keyName := args[0]

	client := key.NewKChainRPCClient(kchainEndpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	keyMeta, err := client.GetKeyByName(ctx, keyName)
	if err != nil {
		return fmt.Errorf("key '%s' not found: %w", keyName, err)
	}

	params := key.ReshareKeyParams{
		KeyID:         keyMeta.ID,
		NewThreshold:  kchainThreshold,
		NewTotalParts: kchainTotalShares,
		NewValidators: kchainValidators,
	}

	ux.Logger.PrintToUser("Performing proactive resharing for '%s'...", keyName)

	result, err := client.ReshareKey(ctx, params)
	if err != nil {
		return fmt.Errorf("resharing failed: %w", err)
	}

	ux.Logger.PrintToUser("")
	if result.Success {
		ux.Logger.PrintToUser("Resharing complete!")
		ux.Logger.PrintToUser("  New shares: %d", len(result.NewShareIDs))
	} else {
		ux.Logger.PrintToUser("Resharing failed.")
	}
	ux.Logger.PrintToUser("")

	return nil
}

func newKChainAlgorithmsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "algorithms",
		Short: "List supported cryptographic algorithms",
		Long:  `List all cryptographic algorithms supported by K-Chain.`,
		Args:  cobra.NoArgs,
		RunE:  runKChainAlgorithms,
	}
}

func runKChainAlgorithms(_ *cobra.Command, _ []string) error {
	client := key.NewKChainRPCClient(kchainEndpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := client.ListAlgorithms(ctx)
	if err != nil {
		return fmt.Errorf("failed to list algorithms: %w", err)
	}

	ux.Logger.PrintToUser("Supported Cryptographic Algorithms")
	ux.Logger.PrintToUser("")

	// Group by type
	signing := []key.AlgorithmInfo{}
	encryption := []key.AlgorithmInfo{}
	keyExchange := []key.AlgorithmInfo{}

	for _, alg := range result.Algorithms {
		switch alg.Type {
		case "signing":
			signing = append(signing, alg)
		case "encryption":
			encryption = append(encryption, alg)
		case "key-exchange":
			keyExchange = append(keyExchange, alg)
		}
	}

	if len(signing) > 0 {
		ux.Logger.PrintToUser("Signing:")
		for _, alg := range signing {
			pq := ""
			if alg.PostQuantum {
				pq = pqSuffix
			}
			th := ""
			if alg.ThresholdSupport {
				th = " [threshold]"
			}
			ux.Logger.PrintToUser("  - %-20s %s%s%s", alg.Name, alg.Description, pq, th)
		}
		ux.Logger.PrintToUser("")
	}

	if len(encryption) > 0 {
		ux.Logger.PrintToUser("Encryption:")
		for _, alg := range encryption {
			pq := ""
			if alg.PostQuantum {
				pq = pqSuffix
			}
			ux.Logger.PrintToUser("  - %-20s %s%s", alg.Name, alg.Description, pq)
		}
		ux.Logger.PrintToUser("")
	}

	if len(keyExchange) > 0 {
		ux.Logger.PrintToUser("Key Encapsulation:")
		for _, alg := range keyExchange {
			pq := ""
			if alg.PostQuantum {
				pq = pqSuffix
			}
			ux.Logger.PrintToUser("  - %-20s %s%s", alg.Name, alg.Description, pq)
		}
		ux.Logger.PrintToUser("")
	}

	return nil
}

func newKChainListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List distributed keys",
		Long:  `List all keys stored in K-Chain.`,
		Args:  cobra.NoArgs,
		RunE:  runKChainList,
	}

	cmd.Flags().StringVarP(&kchainAlgorithm, "algorithm", "a", "", "Filter by algorithm")

	return cmd
}

func runKChainList(_ *cobra.Command, _ []string) error {
	client := key.NewKChainRPCClient(kchainEndpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	params := key.ListKeysParams{
		Algorithm: kchainAlgorithm,
		Limit:     100,
	}

	result, err := client.ListKeys(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to list keys: %w", err)
	}

	if len(result.Keys) == 0 {
		ux.Logger.PrintToUser("No keys found in K-Chain.")
		return nil
	}

	ux.Logger.PrintToUser("K-Chain Distributed Keys")
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("  %-20s  %-16s  %-10s  %s", "NAME", "ALGORITHM", "THRESHOLD", "STATUS")
	ux.Logger.PrintToUser("  %-20s  %-16s  %-10s  %s", "----", "---------", "---------", "------")

	for _, k := range result.Keys {
		threshold := fmt.Sprintf("%d-of-%d", k.Threshold, k.TotalShares)
		if k.Threshold == 0 {
			threshold = "N/A"
		}
		ux.Logger.PrintToUser("  %-20s  %-16s  %-10s  %s", k.Name, k.Algorithm, threshold, k.Status)
	}

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Total: %d keys", result.Total)
	ux.Logger.PrintToUser("")

	return nil
}

func newKChainShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <key-name>",
		Short: "Show key details",
		Long:  `Show detailed information about a distributed key.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runKChainShow,
	}

	cmd.Flags().StringVarP(&kchainFormat, "format", "f", "pem", "Public key format (pem, der, raw)")

	return cmd
}

func runKChainShow(_ *cobra.Command, args []string) error {
	keyName := args[0]

	client := key.NewKChainRPCClient(kchainEndpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	keyMeta, err := client.GetKeyByName(ctx, keyName)
	if err != nil {
		return fmt.Errorf("key '%s' not found: %w", keyName, err)
	}

	ux.Logger.PrintToUser("Key: %s", keyMeta.Name)
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("  ID:          %s", keyMeta.ID)
	ux.Logger.PrintToUser("  Algorithm:   %s", keyMeta.Algorithm)
	ux.Logger.PrintToUser("  Key Type:    %s", keyMeta.KeyType)
	if keyMeta.Threshold > 0 {
		ux.Logger.PrintToUser("  Threshold:   %d-of-%d", keyMeta.Threshold, keyMeta.TotalShares)
	}
	ux.Logger.PrintToUser("  Distributed: %t", keyMeta.Distributed)
	ux.Logger.PrintToUser("  Status:      %s", keyMeta.Status)
	ux.Logger.PrintToUser("  Created:     %s", keyMeta.CreatedAt.Format(time.RFC3339))
	if len(keyMeta.Tags) > 0 {
		ux.Logger.PrintToUser("  Tags:        %s", strings.Join(keyMeta.Tags, ", "))
	}

	// Get public key
	pubParams := key.GetPublicKeyParams{
		KeyID:  keyMeta.ID,
		Format: kchainFormat,
	}

	pubResult, err := client.GetPublicKey(ctx, pubParams)
	if err == nil && pubResult.PublicKey != "" {
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("  Public Key (%s):", pubResult.Format)
		// Show truncated key if too long
		pubKey := pubResult.PublicKey
		if len(pubKey) > 80 {
			ux.Logger.PrintToUser("  %s...", pubKey[:80])
		} else {
			ux.Logger.PrintToUser("  %s", pubKey)
		}
	}

	ux.Logger.PrintToUser("")

	return nil
}

func newKChainCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <key-name>",
		Short: "Create a new distributed key",
		Long: `Create a new key and distribute it across K-Chain validators.

Examples:
  lux key kchain create mykey
  lux key kchain create mykey -a ml-kem-768 -t 3 -n 5`,
		Args: cobra.ExactArgs(1),
		RunE: runKChainCreate,
	}

	cmd.Flags().StringVarP(&kchainAlgorithm, "algorithm", "a", "ml-kem-768", "Key algorithm")
	cmd.Flags().IntVarP(&kchainThreshold, "threshold", "t", 3, "Threshold for reconstruction")
	cmd.Flags().IntVarP(&kchainTotalShares, "shares", "n", 5, "Total shares")

	return cmd
}

func runKChainCreate(_ *cobra.Command, args []string) error {
	keyName := args[0]

	client := key.NewKChainRPCClient(kchainEndpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	params := key.CreateKeyParams{
		Name:        keyName,
		Algorithm:   kchainAlgorithm,
		Threshold:   kchainThreshold,
		TotalShares: kchainTotalShares,
	}

	ux.Logger.PrintToUser("Creating distributed key '%s'...", keyName)
	ux.Logger.PrintToUser("  Algorithm: %s", kchainAlgorithm)
	ux.Logger.PrintToUser("  Threshold: %d-of-%d", kchainThreshold, kchainTotalShares)

	result, err := client.CreateKey(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to create key: %w", err)
	}

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Key created successfully!")
	ux.Logger.PrintToUser("  ID:        %s", result.Key.ID)
	ux.Logger.PrintToUser("  Status:    %s", result.Key.Status)
	if len(result.PublicKey) > 64 {
		ux.Logger.PrintToUser("  Public Key: %s...", result.PublicKey[:64])
	} else {
		ux.Logger.PrintToUser("  Public Key: %s", result.PublicKey)
	}
	if len(result.ShareIDs) > 0 {
		ux.Logger.PrintToUser("  Shares:    %d", len(result.ShareIDs))
	}
	ux.Logger.PrintToUser("")

	return nil
}

func newKChainDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <key-name>",
		Short: "Delete a distributed key",
		Long: `Delete a key and securely wipe all shares from validators.

Examples:
  lux key kchain delete mykey
  lux key kchain delete mykey --force`,
		Args: cobra.ExactArgs(1),
		RunE: runKChainDelete,
	}

	cmd.Flags().BoolVar(&kchainSecureWipe, "force", false, "Force deletion even if shares exist")

	return cmd
}

func runKChainDelete(_ *cobra.Command, args []string) error {
	keyName := args[0]

	client := key.NewKChainRPCClient(kchainEndpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	keyMeta, err := client.GetKeyByName(ctx, keyName)
	if err != nil {
		return fmt.Errorf("key '%s' not found: %w", keyName, err)
	}

	params := key.DeleteKeyParams{
		ID:    keyMeta.ID,
		Force: kchainSecureWipe,
	}

	ux.Logger.PrintToUser("Deleting key '%s'...", keyName)

	result, err := client.DeleteKey(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to delete key: %w", err)
	}

	ux.Logger.PrintToUser("")
	if result.Success {
		ux.Logger.PrintToUser("Key deleted successfully.")
		if len(result.DeletedShares) > 0 {
			ux.Logger.PrintToUser("  Deleted shares: %d", len(result.DeletedShares))
		}
	} else {
		ux.Logger.PrintToUser("Failed to delete key.")
	}
	ux.Logger.PrintToUser("")

	return nil
}
