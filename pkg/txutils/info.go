// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txutils

import (
	"context"
	"fmt"

	"github.com/luxfi/address"
	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/constants"
	"github.com/luxfi/ids"
	"github.com/luxfi/node/vms/platformvm"
	"github.com/luxfi/protocol/p/txs"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/node/vms/secp256k1fx"
)

// GetNetwork returns the network model associated with a tx.
// Expected tx.Unsigned types: txs.AddChainValidatorTx, txs.CreateChainTx, etc.
func GetNetwork(tx *txs.Tx) (models.Network, error) {
	unsignedTx := tx.Unsigned
	var networkID uint32
	switch unsignedTx := unsignedTx.(type) {
	case *txs.RemoveChainValidatorTx:
		networkID = unsignedTx.BaseTx.NetworkID
	case *txs.AddChainValidatorTx:
		networkID = unsignedTx.BaseTx.NetworkID
	case *txs.CreateChainTx:
		networkID = unsignedTx.BaseTx.NetworkID
	case *txs.ConvertChainToL1Tx:
		networkID = unsignedTx.BaseTx.NetworkID
	default:
		return models.Undefined, fmt.Errorf("unexpected unsigned tx type %T", unsignedTx)
	}
	network := models.NetworkFromNetworkID(networkID)
	if network == models.Undefined {
		return models.Undefined, fmt.Errorf("undefined network model for tx")
	}
	return network, nil
}

// GetLedgerDisplayName returns the display name for a tx on the ledger.
func GetLedgerDisplayName(tx *txs.Tx) string {
	unsignedTx := tx.Unsigned
	switch unsignedTx.(type) {
	case *txs.AddChainValidatorTx:
		return "ChainValidator"
	case *txs.CreateChainTx:
		return "CreateChain"
	default:
		return ""
	}
}

// IsCreateChainTx returns true if the tx is a CreateChainTx.
func IsCreateChainTx(tx *txs.Tx) bool {
	_, ok := tx.Unsigned.(*txs.CreateChainTx)
	return ok
}

// ChainOwners contains the ownership information for a chain
type ChainOwners struct {
	IsPermissioned bool
	ControlKeys    []string
	Threshold      uint32
}

// GetChainOwners retrieves ownership information for a chain
func GetChainOwners(network models.Network, chainID ids.ID) (*ChainOwners, error) {
	pClient, err := getPlatformClient(network)
	if err != nil {
		return nil, err
	}

	tx, err := getChainTx(pClient, chainID)
	if err != nil {
		return nil, err
	}

	createNetworkTx, ok := tx.Unsigned.(*txs.CreateNetworkTx)
	if !ok {
		return nil, fmt.Errorf("got unexpected type %T for network tx %s", tx.Unsigned, chainID)
	}

	owner, ok := createNetworkTx.Owner.(*secp256k1fx.OutputOwners)
	if !ok {
		// If not a standard OutputOwners, it might be a different owner type
		// For now, treat as non-permissioned
		return &ChainOwners{IsPermissioned: false}, nil
	}

	// Format control keys as strings
	controlKeysStrs, err := formatControlKeys(network, owner.Addrs)
	if err != nil {
		return nil, err
	}

	return &ChainOwners{
		IsPermissioned: true,
		ControlKeys:    controlKeysStrs,
		Threshold:      owner.Threshold,
	}, nil
}

// GetOwners returns ownership information in the legacy format (for backward compatibility)
func GetOwners(network models.Network, chainID ids.ID) (bool, []string, uint32, error) {
	owners, err := GetChainOwners(network, chainID)
	if err != nil {
		return false, nil, 0, err
	}
	return owners.IsPermissioned, owners.ControlKeys, owners.Threshold, nil
}

// Helper functions for better composability

func getPlatformClient(network models.Network) (*platformvm.Client, error) {
	var api string
	switch network {
	case models.Testnet:
		api = constants.TestnetAPIEndpoint
	case models.Mainnet:
		api = constants.MainnetAPIEndpoint
	case models.Local:
		api = constants.LocalAPIEndpoint
	default:
		return nil, fmt.Errorf("network not supported: %v", network)
	}
	return platformvm.NewClient(api), nil
}

func getChainTx(pClient *platformvm.Client, chainID ids.ID) (*txs.Tx, error) {
	ctx := context.Background()
	txBytes, err := pClient.GetTx(ctx, chainID)
	if err != nil {
		return nil, fmt.Errorf("chain tx %s query error: %w", chainID, err)
	}

	var tx txs.Tx
	if _, err := txs.Codec.Unmarshal(txBytes, &tx); err != nil {
		return nil, fmt.Errorf("couldn't unmarshal tx %s: %w", chainID, err)
	}
	return &tx, nil
}

func formatControlKeys(network models.Network, addrs []ids.ShortID) ([]string, error) {
	networkID, err := network.NetworkID()
	if err != nil {
		return nil, err
	}

	hrp := key.GetHRP(networkID)
	controlKeysStrs := make([]string, 0, len(addrs))

	for _, addr := range addrs {
		addrStr, err := address.Format("P", hrp, addr[:])
		if err != nil {
			return nil, err
		}
		controlKeysStrs = append(controlKeysStrs, addrStr)
	}
	return controlKeysStrs, nil
}
