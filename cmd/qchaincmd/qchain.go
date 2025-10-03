// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package qchaincmd

import (
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/spf13/cobra"
)

var app *application.Lux

// lux qchain
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "qchain",
		Short: "Interact with the Q-Chain (Quantum-Resistant Chain)",
		Long: `The qchain command suite provides tools for interacting with the Q-Chain,
which implements post-quantum cryptography for quantum-resistant security.

Features:
- Ringtail post-quantum signatures
- Quantum-safe key generation
- Secure transaction validation
- Cross-chain quantum-safe communication`,
		RunE: cobrautils.CommandSuiteUsage,
	}
	app = injectedApp

	// qchain generate-keys
	cmd.AddCommand(newGenerateKeysCmd())
	// qchain describe
	cmd.AddCommand(newDescribeCmd())
	// qchain deploy
	cmd.AddCommand(newDeployCmd())
	// qchain transaction
	cmd.AddCommand(newTransactionCmd())
	// qchain verify
	cmd.AddCommand(newVerifyCmd())

	return cmd
}
