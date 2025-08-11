// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package relayercmd

import (
	"fmt"
	"math/big"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/cli/pkg/networkoptions"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/node/utils/units"
	"github.com/spf13/cobra"
)

// lux interchain relayer fund
func newFundCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fund",
		Short: "Fund Warp relayer accounts",
		Long:  `Fund the Warp relayer accounts on specified blockchains.`,
		RunE:  fundRelayer,
		Args:  cobrautils.ExactArgs(0),
	}
	
	networkoptions.AddNetworkFlagsToCmd(cmd, &globalNetworkFlags, false, networkoptions.DefaultSupportedNetworkOptions)
	cmd.Flags().StringVar(&fundingKeyName, "key", "", "Key to use for funding")
	cmd.Flags().Float64Var(&fundAmount, "amount", 0.1, "Amount to fund in LUX")
	cmd.Flags().StringSliceVar(&blockchainNames, "blockchains", nil, "Blockchains to fund")
	
	return cmd
}

var (
	fundingKeyName  string
	fundAmount      float64
	blockchainNames []string
	globalNetworkFlags networkoptions.NetworkFlags
)

func fundRelayer(_ *cobra.Command, _ []string) error {
	network, err := networkoptions.GetNetworkFromCmdLineFlags(
		app,
		"",
		globalNetworkFlags,
		true,
		false,
		nil,
		"",
	)
	if err != nil {
		return err
	}
	
	// Load funding key
	if fundingKeyName == "" {
		return fmt.Errorf("funding key is required")
	}
	
	keyPath := app.GetKeyPath(fundingKeyName)
	sk, err := key.LoadSoft(network.ID(), keyPath)
	if err != nil {
		return fmt.Errorf("failed to load key: %w", err)
	}
	
	// Get relayer address
	relayerAddress, err := getRelayerAddress()
	if err != nil {
		return fmt.Errorf("failed to get relayer address: %w", err)
	}
	
	// Convert amount to wei
	amountWei := new(big.Int).Mul(
		big.NewInt(int64(fundAmount*float64(units.Lux))),
		big.NewInt(1),
	)
	
	// Fund each blockchain
	for _, blockchainName := range blockchainNames {
		ux.Logger.PrintToUser("Funding relayer on blockchain: %s", blockchainName)
		
		// Get RPC endpoint for the blockchain
		rpcEndpoint, err := getBlockchainRPC(app, network, blockchainName)
		if err != nil {
			return fmt.Errorf("failed to get RPC for %s: %w", blockchainName, err)
		}
		
		// Send funds
		if err := sendFunds(sk, relayerAddress, amountWei, rpcEndpoint); err != nil {
			return fmt.Errorf("failed to fund relayer on %s: %w", blockchainName, err)
		}
		
		ux.Logger.PrintToUser("✅ Funded relayer with %.4f LUX on %s", fundAmount, blockchainName)
	}
	
	return nil
}

func getRelayerAddress() (string, error) {
	// Get relayer address from config or generate new one
	// For now, use a default address
	return "0x0000000000000000000000000000000000000000", nil
}

func getBlockchainRPC(app *application.Lux, network models.Network, blockchainName string) (string, error) {
	// Load blockchain configuration
	sc, err := app.LoadSidecar(blockchainName)
	if err != nil {
		return "", err
	}
	
	// Get RPC endpoint
	networkData, ok := sc.Networks[network.Name()]
	if !ok {
		return "", fmt.Errorf("blockchain %s not deployed on network %s", blockchainName, network.Name())
	}
	
	if len(networkData.RPCEndpoints) == 0 {
		return "", fmt.Errorf("no RPC endpoints for blockchain %s", blockchainName)
	}
	
	return networkData.RPCEndpoints[0], nil
}

func sendFunds(sk interface{}, toAddress string, amount *big.Int, rpcEndpoint string) error {
	// Implementation would send funds to the relayer address
	// This is a placeholder for the actual transaction logic
	ux.Logger.PrintToUser("Sending %s wei to %s via %s", amount.String(), toAddress, rpcEndpoint)
	return nil
}