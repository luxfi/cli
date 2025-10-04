// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package qchaincmd

import (
	"fmt"
	"os"

	"github.com/luxfi/cli/pkg/networkoptions"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/node/utils/constants"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func newDescribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe",
		Short: "Show Q-Chain information and status",
		Long:  `Display detailed information about the Q-Chain including its configuration, status, and quantum-resistant features.`,
		RunE:  describeQChain,
	}

	networkoptions.AddNetworkFlagsToCmd(cmd, &globalNetworkFlags, true, networkoptions.DefaultSupportedNetworkOptions)

	return cmd
}

var globalNetworkFlags networkoptions.NetworkFlags

func describeQChain(cmd *cobra.Command, args []string) error {
	network, err := networkoptions.GetNetworkFromCmdLineFlags(
		app,
		"",
		globalNetworkFlags,
		true,
		false,
		networkoptions.DefaultSupportedNetworkOptions,
		"",
	)
	if err != nil {
		return err
	}

	ux.Logger.PrintToUser("Q-Chain Information")
	ux.Logger.PrintToUser("==================")

	// Create table for display
	table := tablewriter.NewWriter(os.Stdout)

	// Basic Information
	table.Append([]string{"Property", "Value"})

	// Network information
	rows := [][]string{
		{"Network", network.Name()},
		{"Chain ID", constants.QChainID.String()},
		{"Chain Alias", "Q"},
		{"VM Type", "QuantumVM"},
		{"Consensus", "Quantum-Resistant Snow"},
	}

	// Q-Chain specific features
	rows = append(rows,
		[]string{"Signature Algorithm", "Ringtail (Post-Quantum)"},
		[]string{"Key Size", "256-bit quantum-safe"},
		[]string{"Hash Function", "SHA3-256 (Quantum-resistant)"},
		[]string{"Transaction Validation", "Quantum-safe verification"},
		[]string{"Cross-chain Protocol", "Quantum Teleport"},
	)

	// Network configuration
	if network.Name() == "Local Network" {
		rows = append(rows,
			[]string{"Network ID", fmt.Sprintf("%d", constants.QChainMainnetID)},
			[]string{"RPC Endpoint", fmt.Sprintf("http://127.0.0.1:9630/ext/bc/%s/rpc", constants.QChainID)},
			[]string{"WS Endpoint", fmt.Sprintf("ws://127.0.0.1:9630/ext/bc/%s/ws", constants.QChainID)},
		)
	} else {
		endpoint := network.Endpoint()
		rows = append(rows,
			[]string{"Network ID", fmt.Sprintf("%d", constants.QChainMainnetID)},
			[]string{"RPC Endpoint", fmt.Sprintf("%s/ext/bc/%s/rpc", endpoint, constants.QChainID)},
			[]string{"WS Endpoint", fmt.Sprintf("%s/ext/bc/%s/ws", endpoint, constants.QChainID)},
		)
	}

	// Status information
	rows = append(rows,
		[]string{"Status", "Ready for Deployment"},
		[]string{"Block Time", "100ms (Fast Finality)"},
		[]string{"Security Level", "Post-Quantum (Level 5)"},
	)

	for _, row := range rows {
		table.Append(row)
	}
	table.Render()

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Q-Chain Features:")
	ux.Logger.PrintToUser("• Quantum-resistant signatures using Ringtail algorithm")
	ux.Logger.PrintToUser("• Secure against attacks from quantum computers")
	ux.Logger.PrintToUser("• Fast finality with 100ms block times")
	ux.Logger.PrintToUser("• Cross-chain quantum-safe communication")
	ux.Logger.PrintToUser("• Compatible with existing Lux infrastructure")

	return nil
}
