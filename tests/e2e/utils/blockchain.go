// Copyright (C) 2024, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/models"
	"github.com/luxfi/geth/accounts/abi/bind"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/ethclient"
)

// GetChainID retrieves the chain ID from an RPC endpoint
func GetChainID(rpcURL string) (uint64, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return 0, err
	}
	defer client.Close()

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return 0, err
	}

	return chainID.Uint64(), nil
}

// GetBalance retrieves the balance of an address
func GetBalance(rpcURL string, address string) (*big.Int, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	addr := common.HexToAddress(address)
	balance, err := client.BalanceAt(context.Background(), addr, nil)
	if err != nil {
		return nil, err
	}

	return balance, nil
}

// DeployTestContract deploys a simple test contract
func DeployTestContract(rpcURL string) (string, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return "", err
	}
	defer client.Close()

	// Use the default funded key
	privateKey := "56289e99c94b6912bfc12adc093c9b51124f0dc54ac7a766b2bc5ccf558d8027"
	auth, err := GetAuth(client, privateKey)
	if err != nil {
		return "", err
	}

	// Simple storage contract bytecode
	bytecode := common.FromHex("608060405234801561001057600080fd5b50610150806100206000396000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c80632e64cec11461003b5780636057361d14610059575b600080fd5b610043610075565b60405161005091906100d9565b60405180910390f35b610073600480360381019061006e919061009d565b61007e565b005b60008054905090565b8060008190555050565b60008135905061009781610103565b92915050565b6000602082840312156100b3576100b26100fe565b5b60006100c184828501610088565b91505092915050565b6100d3816100f4565b82525050565b60006020820190506100ee60008301846100ca565b92915050565b6000819050919050565b600080fd5b61010c816100f4565b811461011757600080fd5b5056fea264697066735822122")

	address, tx, _, err := bind.DeployContract(auth, parsed, bytecode, client)
	if err != nil {
		return "", err
	}

	// Wait for transaction to be mined
	_, err = bind.WaitMined(context.Background(), client, tx)
	if err != nil {
		return "", err
	}

	return address.Hex(), nil
}

// GetContractCode retrieves the code of a deployed contract
func GetContractCode(rpcURL string, address string) ([]byte, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	addr := common.HexToAddress(address)
	code, err := client.CodeAt(context.Background(), addr, nil)
	if err != nil {
		return nil, err
	}

	return code, nil
}

// GetSubnetConfig retrieves the subnet configuration
func GetSubnetConfig(subnetName string) (*models.Sidecar, error) {
	app := application.New()
	subnetDir := filepath.Join(app.GetSubnetDir(), subnetName)
	sidecarPath := filepath.Join(subnetDir, "sidecar.json")

	data, err := os.ReadFile(sidecarPath)
	if err != nil {
		return nil, err
	}

	var sidecar models.Sidecar
	if err := json.Unmarshal(data, &sidecar); err != nil {
		return nil, err
	}

	return &sidecar, nil
}

// GetNodeInfo gets information about a specific node
type NodeInfo struct {
	ID          string
	StakingCert string
}

func GetNodeInfo(ctx context.Context, nodeName string) (*NodeInfo, error) {
	// This would integrate with the netrunner client
	// For now, return mock data based on expected node IDs
	nodeIDs := map[string]string{
		"node1": "NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg",
		"node2": "NodeID-MFrZFVCXPv5iCn6M9K6XduxGTYp891xXZ",
		"node3": "NodeID-NFBbbJ4qCmNaCzeW7sxErhvWqvEQMnYcN",
		"node4": "NodeID-GWPcbFJZFfZreETSoWjPimr846mXEKCtu",
		"node5": "NodeID-P7oB2McjBGgW2NXXWVYjV8JEDFoW9xDE5",
	}

	id, ok := nodeIDs[nodeName]
	if !ok {
		return nil, fmt.Errorf("unknown node: %s", nodeName)
	}

	return &NodeInfo{
		ID:          id,
		StakingCert: "mock-cert-data",
	}, nil
}