// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package primarycmd

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/cli/pkg/localnet"
	"github.com/luxfi/cli/pkg/networkoptions"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	luxlog "github.com/luxfi/log"
	"github.com/luxfi/node/utils/units"
	"github.com/luxfi/sdk/evm"
	"github.com/luxfi/sdk/models"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

const art = `
   _____       _____ _           _         _____
  / ____|     / ____| |         (_)       |  __ \
 | |   ______| |    | |__   __ _ _ _ __   | |__) |_ _ _ __ __ _ _ __ ___  ___ 
 | |  |______| |    | '_ \ / _  | | '_ \  |  ___/ _  | '__/ _  | '_   _ \/ __|
 | |____     | |____| | | | (_| | | | | | | |  | (_| | | | (_| | | | | | \__ \
  \_____|     \_____|_| |_|\__,_|_|_| |_| |_|   \__,_|_|  \__,_|_| |_| |_|___/
`

// lux primary describe
func newDescribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe",
		Short: "Print details of the primary network configuration",
		Long:  `The subnet describe command prints details of the primary network configuration to the console.`,
		RunE:  describe,
		Args:  cobrautils.ExactArgs(0),
	}
	// Network flags handled at higher level to avoid conflicts
	return cmd
}

func describe(_ *cobra.Command, _ []string) error {
	network, err := networkoptions.GetNetworkFromCmdLineFlags(
		app,
		"",
		globalNetworkFlags,
		false,
		false,
		networkoptions.LocalClusterSupportedNetworkOptions,
		"",
	)
	if err != nil {
		return err
	}
	var (
		warpMessengerAddress string
		warpRegistryAddress  string
	)
	blockchainID, err := utils.GetChainID(network.Endpoint(), "C")
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			networkUpMsg := ""
			if network.Kind() != models.Testnet && network.Kind() != models.Mainnet {
				networkUpMsg = fmt.Sprintf(" Is the %s up?", network.Name())
			}
			ux.Logger.RedXToUser("Could not connect to Primary Network at %s.%s", network.Endpoint(), networkUpMsg)
			return nil
		}
		return err
	}
	if network.Kind() == models.Local {
		if b, extraLocalNetworkData, err := localnet.GetExtraLocalNetworkData(app, ""); err != nil {
			return err
		} else if b {
			warpMessengerAddress = extraLocalNetworkData.CChainTeleporterMessengerAddress
			warpRegistryAddress = extraLocalNetworkData.CChainTeleporterRegistryAddress
		}
	} else if network.ClusterName() != "" {
		if clusterConfig, err := app.GetClusterConfig(network.ClusterName()); err != nil {
			return err
		} else {
			// TODO: Fix cluster config access - might need type assertion or different method
			// warpMessengerAddress = clusterConfig.ExtraNetworkData.CChainTeleporterMessengerAddress
			// warpRegistryAddress = clusterConfig.ExtraNetworkData.CChainTeleporterRegistryAddress
			_ = clusterConfig
		}
	}
	fmt.Print(luxlog.LightBlue.Wrap(art))
	blockchainIDHexEncoding := "0x" + hex.EncodeToString(blockchainID[:])
	rpcURL := network.CChainEndpoint()
	client, err := evm.GetClient(rpcURL)
	if err != nil {
		return err
	}
	evmChainID, err := client.GetChainID()
	if err != nil {
		return err
	}
	// Load the ewoq key for local networks
	k, err := key.NewSoft(network.ID(), key.WithPrivateKeyEncoded(key.EwoqPrivateKey))
	if err != nil {
		return err
	}
	address := k.C()
	privKey := k.PrivKeyHex()
	balance, err := client.GetAddressBalance(address)
	if err != nil {
		return err
	}
	balance = balance.Div(balance, big.NewInt(int64(units.Lux)))
	balanceStr := fmt.Sprintf("%.9f", float64(balance.Uint64())/float64(units.Lux))
	table := tablewriter.NewWriter(os.Stdout)
	_ = []string{"Parameter", "Value"}
	// table.SetHeader(header)
	// table.SetRowLine(true)
	// table.SetAlignment(tablewriter.ALIGN_LEFT)
	// table.SetAutoMergeCellsByColumnIndex([]int{0})
	table.Append([]string{"RPC URL", rpcURL})
	codespaceURL, err := utils.GetCodespaceURL(rpcURL)
	if err != nil {
		return err
	}
	if codespaceURL != "" {
		table.Append([]string{"Codespace RPC URL", codespaceURL})
	}
	table.Append([]string{"EVM Chain ID", fmt.Sprint(evmChainID)})
	table.Append([]string{"TOKEN SYMBOL", "LUX"})
	table.Append([]string{"Address", address})
	table.Append([]string{"Balance", balanceStr})
	table.Append([]string{"Private Key", privKey})
	table.Append([]string{"BlockchainID (CB58)", blockchainID.String()})
	table.Append([]string{"BlockchainID (HEX)", blockchainIDHexEncoding})
	if warpMessengerAddress != "" {
		table.Append([]string{"Warp Messenger Address", warpMessengerAddress})
	}
	if warpRegistryAddress != "" {
		table.Append([]string{"Warp Registry Address", warpRegistryAddress})
	}
	table.Render()
	return nil
}
