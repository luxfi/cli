// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package blockchain provides helper functions for blockchain operations
// including peer management, URI handling, and blockchain state queries.
package blockchain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/luxfi/ids"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/localnet"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/math/set"
	"github.com/luxfi/p2p/peer"
	"github.com/luxfi/sdk/api/info"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/vm/vms/platformvm"
	"github.com/luxfi/vm/vms/platformvm/signer"
)

// GetAggregatorExtraPeers returns a list of peers for the aggregator from the cluster.
func GetAggregatorExtraPeers(
	app *application.Lux,
	clusterName string,
) ([]info.Peer, error) {
	uris, err := GetAggregatorNetworkUris(app, clusterName)
	if err != nil {
		return nil, err
	}
	urisSet := set.Of(uris...)
	uris = urisSet.List()
	return UrisToPeers(uris)
}

// GetAggregatorNetworkUris returns network URIs for the specified cluster.
func GetAggregatorNetworkUris(app *application.Lux, clusterName string) ([]string, error) {
	aggregatorExtraPeerEndpointsUris := []string{}
	if clusterName != "" {
		if localnet.LocalClusterExists(app, clusterName) {
			return localnet.GetLocalClusterURIs(app, clusterName)
		}
		// remote cluster case
		clustersConfig, err := app.LoadClustersConfig()
		if err != nil {
			return nil, err
		}
		// Type assertions for map[string]interface{}
		clustersMap, ok := clustersConfig["clusters"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid clusters config format")
		}
		clusterData, ok := clustersMap[clusterName].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("cluster %s not found", clusterName)
		}
		// Parse cluster config to extract node endpoints
		parseClusterConfig(clusterData, &aggregatorExtraPeerEndpointsUris)
	}
	return aggregatorExtraPeerEndpointsUris, nil
}

// parseClusterConfig extracts node endpoints from cluster configuration
func parseClusterConfig(clusterData map[string]interface{}, endpoints *[]string) {
	// Extract nodes field
	if nodes, ok := clusterData["Nodes"].([]interface{}); ok {
		for _, node := range nodes {
			if nodeStr, ok := node.(string); ok {
				// Construct endpoint URI from node ID
				endpoint := fmt.Sprintf("http://%s:9630", nodeStr)
				*endpoints = append(*endpoints, endpoint)
			}
		}
	}

	// Extract API nodes if available
	if apiNodes, ok := clusterData["APINodes"].([]interface{}); ok {
		for _, apiNode := range apiNodes {
			if apiNodeStr, ok := apiNode.(string); ok {
				// API nodes are already endpoints
				*endpoints = append(*endpoints, apiNodeStr)
			}
		}
	}

	// Extract network data if present
	if networkData, ok := clusterData["Network"].(map[string]interface{}); ok {
		if endpoint, ok := networkData["Endpoint"].(string); ok && endpoint != "" {
			*endpoints = append(*endpoints, endpoint)
		}
	}
}

// UrisToPeers converts a list of node URIs to peer information.
func UrisToPeers(uris []string) ([]info.Peer, error) {
	peers := []info.Peer{}
	ctx, cancel := utils.GetANRContext()
	defer cancel()
	for _, uri := range uris {
		client := info.NewClient(uri)
		nodeID, _, err := client.GetNodeID(ctx)
		if err != nil {
			return nil, err
		}
		ip, err := client.GetNodeIP(ctx)
		if err != nil {
			return nil, err
		}
		peers = append(peers, info.Peer{
			Info: peer.Info{
				ID:       nodeID,
				PublicIP: ip,
			},
		})
	}
	return peers, nil
}

// ConvertToBLSProofOfPossession converts public key and proof of possession strings to a ProofOfPossession struct.
func ConvertToBLSProofOfPossession(publicKey, proofOfPossesion string) (signer.ProofOfPossession, error) {
	type jsonProofOfPossession struct {
		PublicKey         string
		ProofOfPossession string
	}
	jsonPop := jsonProofOfPossession{
		PublicKey:         publicKey,
		ProofOfPossession: proofOfPossesion,
	}
	popBytes, err := json.Marshal(jsonPop)
	if err != nil {
		return signer.ProofOfPossession{}, err
	}
	pop := &signer.ProofOfPossession{}
	err = pop.UnmarshalJSON(popBytes)
	if err != nil {
		return signer.ProofOfPossession{}, err
	}
	return *pop, nil
}

// UpdatePChainHeight displays a progress bar while waiting for P-Chain height update.
func UpdatePChainHeight(
	title string,
) error {
	_, err := ux.TimedProgressBar(
		30*time.Second,
		title,
		0,
	)
	if err != nil {
		return err
	}
	fmt.Println()
	return nil
}

// GetBlockchainTimestamp returns the current timestamp from the blockchain.
func GetBlockchainTimestamp(network models.Network) (time.Time, error) {
	ctx, cancel := utils.GetAPIContext()
	defer cancel()
	platformCli := platformvm.NewClient(network.Endpoint())
	return platformCli.GetTimestamp(ctx)
}

// GetSubnet returns subnet validators information
func GetSubnet(subnetID ids.ID, network models.Network) (interface{}, error) {
	api := network.Endpoint()
	pClient := platformvm.NewClient(api)
	ctx, cancel := utils.GetAPIContext()
	defer cancel()
	// GetSubnet has been replaced, using GetCurrentValidators instead
	validators, err := pClient.GetCurrentValidators(ctx, subnetID, nil)
	if err != nil {
		return nil, err
	}
	return validators, nil
}

// GetSubnetIDFromBlockchainID returns the subnet ID that validates the given blockchain.
func GetSubnetIDFromBlockchainID(blockchainID ids.ID, network models.Network) (ids.ID, error) {
	api := network.Endpoint()
	pClient := platformvm.NewClient(api)
	ctx, cancel := utils.GetAPIContext()
	defer cancel()
	return pClient.ValidatedBy(ctx, blockchainID)
}
