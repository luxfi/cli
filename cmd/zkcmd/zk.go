// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package zkcmd

import (
	"github.com/luxfi/cli/pkg/application"
	"github.com/spf13/cobra"
)

var app *application.Lux

// NewCmd creates the zk command for zero-knowledge proof operations.
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	app = injectedApp
	cmd := &cobra.Command{
		Use:   "zk",
		Short: "Zero-knowledge proof tools (ceremony, prove, verify, SRS)",
		Long: `The zk command provides tools for zero-knowledge proof operations
on the Lux network, including powers-of-tau ceremony management,
proof generation, proof verification, and SRS (Structured Reference String)
management.

These operations integrate with the Z-Chain, Lux's dedicated ZK chain,
for on-chain proof verification via precompiled contracts.

USAGE:

  lux zk ceremony init     Initialize a new powers-of-tau ceremony
  lux zk ceremony contribute  Add randomness to a ceremony
  lux zk ceremony verify   Verify ceremony integrity
  lux zk ceremony export   Export final SRS binary
  lux zk ceremony status   Show ceremony state

  lux zk prove groth16     Generate a Groth16 proof
  lux zk prove plonk       Generate a PLONK proof

  lux zk verify groth16    Verify a Groth16 proof
  lux zk verify plonk      Verify a PLONK proof

  lux zk srs download      Download the official Lux SRS
  lux zk srs verify        Verify a downloaded SRS
  lux zk srs info          Show SRS metadata`,
	}

	cmd.AddCommand(newCeremonyCmd())
	cmd.AddCommand(newVerifyCmd())
	cmd.AddCommand(newProveCmd())
	cmd.AddCommand(newSRSCmd())

	return cmd
}
