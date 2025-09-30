// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
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
	"github.com/luxfi/node/api/info"
	"github.com/luxfi/node/network/peer"
	"github.com/luxfi/node/utils/set"
	"github.com/luxfi/node/vms/platformvm"
	"github.com/luxfi/node/vms/platformvm/signer"
	"github.com/luxfi/sdk/models"
)

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

func GetAggregatorNetworkUris(app *application.Lux, clusterName string) ([]string, error) {
	aggregatorExtraPeerEndpointsUris := []string{}
	if clusterName != "" {
		if localnet.LocalClusterExists(app, clusterName) {
			return localnet.GetLocalClusterURIs(app, clusterName)
		} else { // remote cluster case
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
			if err := parseClusterConfig(clusterData, &aggregatorExtraPeerEndpointsUris); err != nil {
				return nil, fmt.Errorf("failed to parse cluster config: %w", err)
			}
		}
	}
	return aggregatorExtraPeerEndpointsUris, nil
}

// parseClusterConfig extracts node endpoints from cluster configuration
func parseClusterConfig(clusterData map[string]interface{}, endpoints *[]string) error {
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

	return nil
}

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

func GetBlockchainTimestamp(network models.Network) (time.Time, error) {
	ctx, cancel := utils.GetAPIContext()
	defer cancel()
	platformCli := platformvm.NewClient(network.Endpoint())
	return platformCli.GetTimestamp(ctx)
}

func GetSubnet(subnetID ids.ID, network models.Network) (platformvm.GetSubnetClientResponse, error) {
	api := network.Endpoint()
	pClient := platformvm.NewClient(api)
	ctx, cancel := utils.GetAPIContext()
	defer cancel()
	return pClient.GetSubnet(ctx, subnetID)
}

func GetSubnetIDFromBlockchainID(blockchainID ids.ID, network models.Network) (ids.ID, error) {
	api := network.Endpoint()
	pClient := platformvm.NewClient(api)
	ctx, cancel := utils.GetAPIContext()
	defer cancel()
	return pClient.ValidatedBy(ctx, blockchainID)
}
