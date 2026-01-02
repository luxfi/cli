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
	"github.com/luxfi/crypto/ring"
	"github.com/spf13/cobra"
)

// Ring signature scheme names
const (
	schemeLSAG        = "lsag"
	schemeLattice     = "lattice"
	schemeLatticePQ   = "pq"           // alias for lattice-lsag
	schemeLatticeFull = "lattice-lsag" // full name for lattice scheme
)

var (
	ringScheme     string
	ringSize       int
	ringOutputFile string
	ringInputFile  string
	ringRingKeys   []string
)

func newRingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ring",
		Short: "Ring signature operations for anonymous signing",
		Long: `Ring signatures allow signing messages such that the signature can be
verified as coming from someone in a group (the "ring"), without revealing
which member actually signed. This provides strong anonymity guarantees.

Features:
  - LSAG (Linkable Spontaneous Anonymous Group) signatures using secp256k1
  - Lattice-based ring signatures for post-quantum security
  - Key images for linkability (double-spend detection)

The ring signature uses your Ringtail key (secp256k1) from ~/.lux/keys/<name>/rt/

Examples:
  lux key ring sign mykey "message" --ring key1,key2,key3
  lux key ring verify "message" --signature <sig> --ring key1,key2,key3
  lux key ring keyimage mykey
  lux key ring schemes`,
		RunE: cobrautils.CommandSuiteUsage,
	}

	cmd.AddCommand(newRingSignCmd())
	cmd.AddCommand(newRingVerifyCmd())
	cmd.AddCommand(newRingKeyImageCmd())
	cmd.AddCommand(newRingSchemesCmd())
	cmd.AddCommand(newRingGenerateRingCmd())

	return cmd
}

func newRingSignCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sign <key-name> <message>",
		Short: "Create a ring signature",
		Long: `Create a ring signature for a message using your key and a ring of public keys.

Your key must be one of the keys in the ring. The signature proves you're a member
of the ring without revealing which member you are.

Examples:
  lux key ring sign mykey "message to sign" --ring key1,key2,key3
  lux key ring sign mykey --file message.txt --ring key1,key2,key3,key4
  lux key ring sign mykey "data" --ring key1,key2,key3 --scheme lattice`,
		Args: cobra.RangeArgs(1, 2),
		RunE: runRingSign,
	}

	cmd.Flags().StringSliceVar(&ringRingKeys, "ring", nil, "Ring member key names (comma-separated)")
	cmd.Flags().StringVarP(&ringScheme, "scheme", "s", schemeLSAG, "Signature scheme (lsag, lattice)")
	cmd.Flags().StringVarP(&ringInputFile, "file", "f", "", "Read message from file")
	cmd.Flags().StringVarP(&ringOutputFile, "output", "o", "", "Write signature to file")
	_ = cmd.MarkFlagRequired("ring")

	return cmd
}

func runRingSign(_ *cobra.Command, args []string) error {
	keyName := args[0]

	// Get message
	var message []byte
	switch {
	case ringInputFile != "":
		var err error
		message, err = os.ReadFile(ringInputFile)
		if err != nil {
			return fmt.Errorf("failed to read message file: %w", err)
		}
	case len(args) > 1:
		message = []byte(args[1])
	default:
		return fmt.Errorf("message required: provide as argument or use --file")
	}

	// Determine scheme
	var scheme ring.Scheme
	switch strings.ToLower(ringScheme) {
	case schemeLSAG, "":
		scheme = ring.LSAG
	case schemeLattice, schemeLatticeFull, schemeLatticePQ:
		scheme = ring.LatticeLSAG
	default:
		return fmt.Errorf("unknown scheme: %s (use '%s' or '%s')", ringScheme, schemeLSAG, schemeLattice)
	}

	// Load signer's key
	keySet, err := key.LoadKeySet(keyName)
	if err != nil {
		return fmt.Errorf("failed to load key '%s': %w", keyName, err)
	}

	// Build ring of public keys and find signer index
	ringPubKeys, signerIndex, err := buildRing(keyName, ringRingKeys, scheme, keySet)
	if err != nil {
		return err
	}

	ux.Logger.PrintToUser("Creating ring signature...")
	ux.Logger.PrintToUser("  Scheme:       %s", scheme.String())
	ux.Logger.PrintToUser("  Ring size:    %d", len(ringPubKeys))
	ux.Logger.PrintToUser("  Signer index: hidden (anonymous)")

	// Create signer based on scheme
	var signer ring.Signer
	switch scheme {
	case ring.LSAG:
		signer, err = ring.NewLSAGSignerFromPrivateKey(keySet.RingtailPrivateKey)
	case ring.LatticeLSAG:
		signer, err = ring.NewLatticeSignerFromPrivateKey(keySet.MLDSAPrivateKey)
	default:
		return fmt.Errorf("unsupported scheme")
	}
	if err != nil {
		return fmt.Errorf("failed to create signer: %w", err)
	}

	// Create signature
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = ctx // For future async signing

	sig, err := signer.Sign(message, ringPubKeys, signerIndex)
	if err != nil {
		return fmt.Errorf("failed to create signature: %w", err)
	}

	// Output signature
	sigBytes := sig.Bytes()
	sigHex := hex.EncodeToString(sigBytes)
	keyImageHex := hex.EncodeToString(sig.KeyImage())

	if ringOutputFile != "" {
		if err := os.WriteFile(ringOutputFile, []byte(sigHex), 0o644); err != nil {
			return fmt.Errorf("failed to write signature file: %w", err)
		}
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("Signature written to: %s", ringOutputFile)
	} else {
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("Signature (%d bytes):", len(sigBytes))
		ux.Logger.PrintToUser("  %s", sigHex)
	}

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Key Image (for linkability):")
	ux.Logger.PrintToUser("  %s", keyImageHex)
	ux.Logger.PrintToUser("")

	return nil
}

func newRingVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify <message>",
		Short: "Verify a ring signature",
		Long: `Verify a ring signature against a message and ring of public keys.

Examples:
  lux key ring verify "message" --signature <sig> --ring key1,key2,key3
  lux key ring verify --file message.txt --signature-file sig.txt --ring key1,key2,key3`,
		Args: cobra.MaximumNArgs(1),
		RunE: runRingVerify,
	}

	cmd.Flags().StringVar(&ringScheme, "scheme", schemeLSAG, "Signature scheme (lsag, lattice)")
	cmd.Flags().StringSliceVar(&ringRingKeys, "ring", nil, "Ring member key names (comma-separated)")
	cmd.Flags().String("signature", "", "Signature (hex-encoded)")
	cmd.Flags().String("signature-file", "", "Read signature from file")
	cmd.Flags().StringVarP(&ringInputFile, "file", "f", "", "Read message from file")
	_ = cmd.MarkFlagRequired("ring")

	return cmd
}

func runRingVerify(cmd *cobra.Command, args []string) error {
	// Get message
	var message []byte
	switch {
	case ringInputFile != "":
		var err error
		message, err = os.ReadFile(ringInputFile)
		if err != nil {
			return fmt.Errorf("failed to read message file: %w", err)
		}
	case len(args) > 0:
		message = []byte(args[0])
	default:
		return fmt.Errorf("message required: provide as argument or use --file")
	}

	// Get signature
	sigHex, _ := cmd.Flags().GetString("signature")
	sigFile, _ := cmd.Flags().GetString("signature-file")

	if sigHex == "" && sigFile == "" {
		return fmt.Errorf("signature required: use --signature or --signature-file")
	}

	if sigFile != "" {
		sigBytes, err := os.ReadFile(sigFile)
		if err != nil {
			return fmt.Errorf("failed to read signature file: %w", err)
		}
		sigHex = strings.TrimSpace(string(sigBytes))
	}

	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		return fmt.Errorf("invalid signature hex: %w", err)
	}

	// Determine scheme
	var scheme ring.Scheme
	switch strings.ToLower(ringScheme) {
	case schemeLSAG, "":
		scheme = ring.LSAG
	case schemeLattice, schemeLatticeFull, schemeLatticePQ:
		scheme = ring.LatticeLSAG
	default:
		return fmt.Errorf("unknown scheme: %s", ringScheme)
	}

	// Parse signature
	sig, err := ring.ParseSignature(scheme, sigBytes)
	if err != nil {
		return fmt.Errorf("failed to parse signature: %w", err)
	}

	// Build ring of public keys
	ringPubKeys, err := buildRingFromNames(ringRingKeys, scheme)
	if err != nil {
		return err
	}

	ux.Logger.PrintToUser("Verifying ring signature...")
	ux.Logger.PrintToUser("  Scheme:    %s", scheme.String())
	ux.Logger.PrintToUser("  Ring size: %d", len(ringPubKeys))

	// Verify
	valid := sig.Verify(message, ringPubKeys)

	ux.Logger.PrintToUser("")
	if valid {
		ux.Logger.PrintToUser("✓ Signature is VALID")
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("Key Image: %s", hex.EncodeToString(sig.KeyImage()))
	} else {
		ux.Logger.PrintToUser("✗ Signature is INVALID")
	}
	ux.Logger.PrintToUser("")

	if !valid {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}

func newRingKeyImageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keyimage <key-name>",
		Short: "Show key image for a key",
		Long: `Show the key image for a key. Key images are deterministic identifiers
derived from the private key that enable linkability - two signatures from
the same key will have the same key image.

This is used for double-spend detection in privacy-preserving transactions.

Examples:
  lux key ring keyimage mykey
  lux key ring keyimage mykey --scheme lattice`,
		Args: cobra.ExactArgs(1),
		RunE: runRingKeyImage,
	}

	cmd.Flags().StringVar(&ringScheme, "scheme", schemeLSAG, "Signature scheme (lsag, lattice)")

	return cmd
}

func runRingKeyImage(_ *cobra.Command, args []string) error {
	keyName := args[0]

	// Load key
	keySet, err := key.LoadKeySet(keyName)
	if err != nil {
		return fmt.Errorf("failed to load key '%s': %w", keyName, err)
	}

	// Determine scheme
	var scheme ring.Scheme
	switch strings.ToLower(ringScheme) {
	case schemeLSAG, "":
		scheme = ring.LSAG
	case schemeLattice, schemeLatticeFull, schemeLatticePQ:
		scheme = ring.LatticeLSAG
	default:
		return fmt.Errorf("unknown scheme: %s", ringScheme)
	}

	// Create signer to get key image
	var signer ring.Signer
	switch scheme {
	case ring.LSAG:
		signer, err = ring.NewLSAGSignerFromPrivateKey(keySet.RingtailPrivateKey)
	case ring.LatticeLSAG:
		signer, err = ring.NewLatticeSignerFromPrivateKey(keySet.MLDSAPrivateKey)
	}
	if err != nil {
		return fmt.Errorf("failed to create signer: %w", err)
	}

	keyImage := signer.KeyImage()

	ux.Logger.PrintToUser("Key Image for '%s' (%s):", keyName, scheme.String())
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("  Hex:    %s", hex.EncodeToString(keyImage))
	ux.Logger.PrintToUser("  Base64: %s", base64.StdEncoding.EncodeToString(keyImage))
	ux.Logger.PrintToUser("")

	return nil
}

func newRingSchemesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "schemes",
		Short: "List supported ring signature schemes",
		Long:  `List all supported ring signature schemes and their properties.`,
		Args:  cobra.NoArgs,
		RunE:  runRingSchemes,
	}
}

func runRingSchemes(_ *cobra.Command, _ []string) error {
	ux.Logger.PrintToUser("Supported Ring Signature Schemes")
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("  LSAG (Linkable Spontaneous Anonymous Group)")
	ux.Logger.PrintToUser("    - Based on secp256k1 elliptic curves")
	ux.Logger.PrintToUser("    - Uses Ringtail keys from ~/.lux/keys/<name>/rt/")
	ux.Logger.PrintToUser("    - Compact signatures, fast verification")
	ux.Logger.PrintToUser("    - Standard: Use '--scheme lsag' (default)")
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("  Lattice-LSAG (Post-Quantum)")
	ux.Logger.PrintToUser("    - Based on ML-DSA (FIPS 204) key material")
	ux.Logger.PrintToUser("    - Uses ML-DSA keys from ~/.lux/keys/<name>/mldsa/")
	ux.Logger.PrintToUser("    - Quantum-resistant security")
	ux.Logger.PrintToUser("    - Larger signatures, NIST Level 3 security")
	ux.Logger.PrintToUser("    - Use '--scheme lattice' or '--scheme pq'")
	ux.Logger.PrintToUser("")

	return nil
}

func newRingGenerateRingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate decoy keys for a ring",
		Long: `Generate random public keys to use as decoys in a ring signature.

In production, you should use real public keys from the network for better
anonymity. This command is mainly for testing and demonstration.

Examples:
  lux key ring generate --size 5
  lux key ring generate --size 10 --scheme lattice`,
		Args: cobra.NoArgs,
		RunE: runRingGenerate,
	}

	cmd.Flags().IntVarP(&ringSize, "size", "n", 5, "Number of keys to generate")
	cmd.Flags().StringVar(&ringScheme, "scheme", schemeLSAG, "Signature scheme (lsag, lattice)")

	return cmd
}

func runRingGenerate(_ *cobra.Command, _ []string) error {
	var scheme ring.Scheme
	switch strings.ToLower(ringScheme) {
	case schemeLSAG, "":
		scheme = ring.LSAG
	case schemeLattice, schemeLatticeFull, schemeLatticePQ:
		scheme = ring.LatticeLSAG
	default:
		return fmt.Errorf("unknown scheme: %s", ringScheme)
	}

	ux.Logger.PrintToUser("Generating %d random public keys for %s ring...", ringSize, scheme.String())

	ringKeys, err := ring.GenerateRing(scheme, ringSize)
	if err != nil {
		return fmt.Errorf("failed to generate ring: %w", err)
	}

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Generated Public Keys:")
	for i, pk := range ringKeys {
		pkHex := hex.EncodeToString(pk)
		if len(pkHex) > 64 {
			pkHex = pkHex[:64] + "..."
		}
		ux.Logger.PrintToUser("  [%d] %s", i, pkHex)
	}
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Note: For real anonymity, use public keys from actual network participants.")
	ux.Logger.PrintToUser("")

	return nil
}

// buildRing builds a ring of public keys from key names, including the signer
func buildRing(signerName string, ringNames []string, scheme ring.Scheme, signerKeySet *key.HDKeySet) ([][]byte, int, error) {
	// Ensure signer is in ring
	found := false
	for _, name := range ringNames {
		if name == signerName {
			found = true
			break
		}
	}
	if !found {
		return nil, 0, fmt.Errorf("signer key '%s' must be in the ring", signerName)
	}

	// Build ring
	ringPubKeys := make([][]byte, len(ringNames))
	signerIndex := -1

	for i, name := range ringNames {
		var pubKey []byte
		if name == signerName {
			signerIndex = i
			switch scheme {
			case ring.LSAG:
				pubKey = signerKeySet.RingtailPublicKey
			case ring.LatticeLSAG:
				pubKey = signerKeySet.MLDSAPublicKey
			}
		} else {
			ks, err := key.LoadKeySet(name)
			if err != nil {
				return nil, 0, fmt.Errorf("failed to load key '%s': %w", name, err)
			}
			switch scheme {
			case ring.LSAG:
				pubKey = ks.RingtailPublicKey
			case ring.LatticeLSAG:
				pubKey = ks.MLDSAPublicKey
			}
		}
		ringPubKeys[i] = pubKey
	}

	if signerIndex < 0 {
		return nil, 0, fmt.Errorf("signer not found in ring")
	}

	return ringPubKeys, signerIndex, nil
}

// buildRingFromNames builds a ring of public keys from key names (for verification)
func buildRingFromNames(ringNames []string, scheme ring.Scheme) ([][]byte, error) {
	ringPubKeys := make([][]byte, len(ringNames))

	for i, name := range ringNames {
		ks, err := key.LoadKeySet(name)
		if err != nil {
			return nil, fmt.Errorf("failed to load key '%s': %w", name, err)
		}
		switch scheme {
		case ring.LSAG:
			ringPubKeys[i] = ks.RingtailPublicKey
		case ring.LatticeLSAG:
			ringPubKeys[i] = ks.MLDSAPublicKey
		}
	}

	return ringPubKeys, nil
}
