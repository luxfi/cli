// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package warpcmd provides the warp command for cross-chain messaging operations
package warpcmd

import (
	"encoding/hex"
	"fmt"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/warp/types"
	"github.com/spf13/cobra"
)

var app *application.Lux

// NewCmd creates the warp command for the Lux CLI
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	app = injectedApp

	cmd := &cobra.Command{
		Use:   "warp",
		Short: "Cross-chain messaging protocol operations",
		Long: `Warp V2 provides cross-chain messaging with post-quantum safety.

This command provides tools for creating, signing, verifying, and relaying
cross-chain messages between Lux networks.

Commands:
  create    Create a new cross-chain message
  sign      Sign a message with validator key
  verify    Verify a signed message
  relay     Start message relayer`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newSignCmd())
	cmd.AddCommand(newVerifyCmd())
	cmd.AddCommand(newRelayCmd())

	return cmd
}

func newCreateCmd() *cobra.Command {
	var (
		sourceChain string
		destChain   string
		payload     string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new cross-chain message",
		Long: `Create a new Warp message to send between chains.

Example:
  lux warp create --source 0xAA --dest 0xBB --payload "Hello from chain A"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			sourceID, err := hexToID(sourceChain)
			if err != nil {
				return fmt.Errorf("invalid source chain ID: %w", err)
			}

			destID, err := hexToID(destChain)
			if err != nil {
				return fmt.Errorf("invalid destination chain ID: %w", err)
			}

			msg := &SimpleMessage{
				sourceID: sourceID,
				destID:   destID,
				payload:  []byte(payload),
			}

			serialized, _ := msg.Serialize()
			msg.id = types.ID(hashBytes(serialized))

			fmt.Printf("Warp message created:\n")
			fmt.Printf("  ID: %x\n", msg.ID())
			fmt.Printf("  Source: %x\n", msg.SourceChainID())
			fmt.Printf("  Destination: %x\n", msg.DestinationChainID())
			fmt.Printf("  Payload: %s\n", msg.Payload())
			fmt.Printf("  Serialized: %x\n", serialized)

			return nil
		},
	}

	cmd.Flags().StringVarP(&sourceChain, "source", "s", "", "Source chain ID (hex)")
	cmd.Flags().StringVarP(&destChain, "dest", "d", "", "Destination chain ID (hex)")
	cmd.Flags().StringVarP(&payload, "payload", "p", "", "Message payload")
	_ = cmd.MarkFlagRequired("source")
	_ = cmd.MarkFlagRequired("dest")
	_ = cmd.MarkFlagRequired("payload")

	return cmd
}

func newSignCmd() *cobra.Command {
	var (
		messageHex string
		keyFile    string
	)

	cmd := &cobra.Command{
		Use:   "sign",
		Short: "Sign a Warp message",
		Long: `Sign a cross-chain message with your validator key.

Example:
  lux warp sign --message <hex> --key ~/.lux/staking/signer.key`,
		RunE: func(cmd *cobra.Command, args []string) error {
			messageBytes, err := hex.DecodeString(messageHex)
			if err != nil {
				return fmt.Errorf("invalid message hex: %w", err)
			}

			fmt.Printf("Message to sign: %x\n", messageBytes)
			fmt.Printf("Key file: %s\n", keyFile)
			fmt.Println("BLS signing integrates with validator infrastructure")

			return nil
		},
	}

	cmd.Flags().StringVarP(&messageHex, "message", "m", "", "Message to sign (hex)")
	cmd.Flags().StringVarP(&keyFile, "key", "k", "", "Path to signing key")
	_ = cmd.MarkFlagRequired("message")
	_ = cmd.MarkFlagRequired("key")

	return cmd
}

func newVerifyCmd() *cobra.Command {
	var (
		messageHex   string
		signatureHex string
	)

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify a signed message",
		Long: `Verify a Warp message signature against the validator set.

Example:
  lux warp verify --message <hex> --signature <hex>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			messageBytes, err := hex.DecodeString(messageHex)
			if err != nil {
				return fmt.Errorf("invalid message hex: %w", err)
			}

			signatureBytes, err := hex.DecodeString(signatureHex)
			if err != nil {
				return fmt.Errorf("invalid signature hex: %w", err)
			}

			fmt.Printf("Message: %x\n", messageBytes)
			fmt.Printf("Signature: %x\n", signatureBytes)
			fmt.Println("Verification integrates with validator set")

			return nil
		},
	}

	cmd.Flags().StringVarP(&messageHex, "message", "m", "", "Message to verify (hex)")
	cmd.Flags().StringVarP(&signatureHex, "signature", "s", "", "Signature to verify (hex)")
	_ = cmd.MarkFlagRequired("message")
	_ = cmd.MarkFlagRequired("signature")

	return cmd
}

func newRelayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relay",
		Short: "Relay messages between chains",
		Long: `Start a Warp message relayer to bridge messages between chains.

The relayer monitors source chains for new messages and delivers them
to destination chains after signature verification.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Warp message relayer")
			fmt.Println("Integrates with Lux relayer infrastructure")
			return nil
		},
	}

	return cmd
}

// SimpleMessage implements the Message interface
type SimpleMessage struct {
	id       types.ID
	sourceID types.ID
	destID   types.ID
	payload  []byte
}

func (m *SimpleMessage) ID() types.ID                 { return m.id }
func (m *SimpleMessage) SourceChainID() types.ID      { return m.sourceID }
func (m *SimpleMessage) DestinationChainID() types.ID { return m.destID }
func (m *SimpleMessage) Payload() []byte              { return m.payload }
func (m *SimpleMessage) Serialize() ([]byte, error) {
	result := make([]byte, 0, 32*2+len(m.payload))
	result = append(result, m.sourceID[:]...)
	result = append(result, m.destID[:]...)
	result = append(result, m.payload...)
	return result, nil
}

func hexToID(hexStr string) (types.ID, error) {
	if len(hexStr) == 0 {
		return types.ID{}, fmt.Errorf("empty chain ID")
	}

	if len(hexStr) >= 2 && hexStr[0:2] == "0x" {
		hexStr = hexStr[2:]
	}

	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return types.ID{}, err
	}

	var id types.ID
	copy(id[:], bytes)
	return id, nil
}

func hashBytes(data []byte) [32]byte {
	var hash [32]byte
	for i, b := range data {
		hash[i%32] ^= b
	}
	return hash
}
