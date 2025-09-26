// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package txutils

import (
	"context"
	"fmt"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/ids"
	"github.com/luxfi/node/utils/formatting/address"
	"github.com/luxfi/node/vms/platformvm"
	"github.com/luxfi/node/vms/platformvm/txs"
	"github.com/luxfi/node/vms/secp256k1fx"
)

// get network model associated to tx
// expect tx.Unsigned type to be in [txs.AddNetValidatorTx, txs.CreateChainTx]
func GetNetwork(tx *txs.Tx) (models.Network, error) {
	unsignedTx := tx.Unsigned
	var networkID uint32
	switch unsignedTx := unsignedTx.(type) {
	case *txs.RemoveNetValidatorTx:
		networkID = unsignedTx.NetworkID
	case *txs.AddNetValidatorTx:
		networkID = unsignedTx.NetworkID
	case *txs.CreateChainTx:
		networkID = unsignedTx.NetworkID
	case *txs.ConvertNetToL1Tx:
		networkID = unsignedTx.NetworkID
	default:
		return models.Undefined, fmt.Errorf("unexpected unsigned tx type %T", unsignedTx)
	}
	network := models.NetworkFromNetworkID(networkID)
	if network == models.Undefined {
		return models.Undefined, fmt.Errorf("undefined network model for tx")
	}
	return network, nil
}

func GetLedgerDisplayName(tx *txs.Tx) string {
	unsignedTx := tx.Unsigned
	switch unsignedTx.(type) {
	case *txs.AddNetValidatorTx:
		return "SubnetValidator"
	case *txs.CreateChainTx:
		return "CreateChain"
	default:
		return ""
	}
}

func IsCreateChainTx(tx *txs.Tx) bool {
	_, ok := tx.Unsigned.(*txs.CreateChainTx)
	return ok
}

// SubnetOwners contains the ownership information for a subnet
type SubnetOwners struct {
	IsPermissioned bool
	ControlKeys    []string
	Threshold      uint32
}

// GetSubnetOwners retrieves ownership information for a subnet
func GetSubnetOwners(network models.Network, subnetID ids.ID) (*SubnetOwners, error) {
	pClient, err := getPlatformClient(network)
	if err != nil {
		return nil, err
	}
	
	tx, err := getSubnetTx(pClient, subnetID)
	if err != nil {
		return nil, err
	}
	
	createSubnetTx, ok := tx.Unsigned.(*txs.CreateNetTx)
	if !ok {
		return nil, fmt.Errorf("got unexpected type %T for subnet tx %s", tx.Unsigned, subnetID)
	}
	
	owner, ok := createSubnetTx.Owner.(*secp256k1fx.OutputOwners)
	if !ok {
		// If not a standard OutputOwners, it might be a different owner type
		// For now, treat as non-permissioned
		return &SubnetOwners{IsPermissioned: false}, nil
	}
	
	// Format control keys as strings
	controlKeysStrs, err := formatControlKeys(network, owner.Addrs)
	if err != nil {
		return nil, err
	}
	
	return &SubnetOwners{
		IsPermissioned: true,
		ControlKeys:    controlKeysStrs,
		Threshold:      owner.Threshold,
	}, nil
}

// GetOwners returns ownership information in the legacy format (for backward compatibility)
func GetOwners(network models.Network, subnetID ids.ID) (bool, []string, uint32, error) {
	owners, err := GetSubnetOwners(network, subnetID)
	if err != nil {
		return false, nil, 0, err
	}
	return owners.IsPermissioned, owners.ControlKeys, owners.Threshold, nil
}

// Helper functions for better composability

func getPlatformClient(network models.Network) (platformvm.Client, error) {
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

func getSubnetTx(pClient platformvm.Client, subnetID ids.ID) (*txs.Tx, error) {
	ctx := context.Background()
	txBytes, err := pClient.GetTx(ctx, subnetID)
	if err != nil {
		return nil, fmt.Errorf("subnet tx %s query error: %w", subnetID, err)
	}
	
	var tx txs.Tx
	if _, err := txs.Codec.Unmarshal(txBytes, &tx); err != nil {
		return nil, fmt.Errorf("couldn't unmarshal tx %s: %w", subnetID, err)
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
