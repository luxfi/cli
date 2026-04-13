// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package zkcmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify a ZK proof (groth16 or plonk)",
		Long: `Verify zero-knowledge proofs by calling Z-Chain precompiled contracts.

The Z-Chain provides on-chain verification for Groth16 and PLONK proofs via
precompiled contracts at fixed addresses. This command submits the proof and
public inputs to the verifier precompile and returns the result.`,
	}

	cmd.AddCommand(newVerifyGroth16Cmd())
	cmd.AddCommand(newVerifyPlonkCmd())

	return cmd
}

func newVerifyGroth16Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "groth16",
		Short: "Verify a Groth16 proof",
		Long: `Verify a Groth16 proof against the Z-Chain Groth16 verifier precompile.

Requires the proof file, verification key, and public inputs.
Connects to the Z-Chain RPC endpoint to call the verifier contract.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			rpc, _ := cmd.Flags().GetString("rpc")
			proof, _ := cmd.Flags().GetString("proof")
			vk, _ := cmd.Flags().GetString("vk")
			inputs, _ := cmd.Flags().GetString("inputs")
			return verifyProof("groth16", rpc, proof, vk, inputs)
		},
	}

	cmd.Flags().String("rpc", "http://localhost:9630/ext/bc/Z/rpc", "Z-Chain RPC endpoint")
	cmd.Flags().String("proof", "", "Proof file path (required)")
	cmd.Flags().String("vk", "", "Verification key file path (required)")
	cmd.Flags().String("inputs", "", "Public inputs file path (required)")
	cmd.MarkFlagRequired("proof")
	cmd.MarkFlagRequired("vk")
	cmd.MarkFlagRequired("inputs")

	return cmd
}

func newVerifyPlonkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plonk",
		Short: "Verify a PLONK proof",
		Long: `Verify a PLONK proof against the Z-Chain PLONK verifier precompile.

Requires the proof file, verification key, and public inputs.
Connects to the Z-Chain RPC endpoint to call the verifier contract.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			rpc, _ := cmd.Flags().GetString("rpc")
			proof, _ := cmd.Flags().GetString("proof")
			vk, _ := cmd.Flags().GetString("vk")
			inputs, _ := cmd.Flags().GetString("inputs")
			return verifyProof("plonk", rpc, proof, vk, inputs)
		},
	}

	cmd.Flags().String("rpc", "http://localhost:9630/ext/bc/Z/rpc", "Z-Chain RPC endpoint")
	cmd.Flags().String("proof", "", "Proof file path (required)")
	cmd.Flags().String("vk", "", "Verification key file path (required)")
	cmd.Flags().String("inputs", "", "Public inputs file path (required)")
	cmd.MarkFlagRequired("proof")
	cmd.MarkFlagRequired("vk")
	cmd.MarkFlagRequired("inputs")

	return cmd
}

func verifyProof(scheme, rpc, proofPath, vkPath, inputsPath string) error {
	fmt.Printf("Verifying %s proof via Z-Chain at %s\n", scheme, rpc)
	fmt.Printf("  Proof:   %s\n", proofPath)
	fmt.Printf("  VK:      %s\n", vkPath)
	fmt.Printf("  Inputs:  %s\n", inputsPath)
	fmt.Println()
	return fmt.Errorf(
		"Z-Chain %s verifier precompile call not yet implemented (rpc: %s)\n\n"+
			"The Z-Chain exposes verifier precompiles for on-chain proof verification.\n"+
			"Track progress: https://github.com/luxfi/node/issues?q=zkvm+verifier",
		scheme, rpc,
	)
}
