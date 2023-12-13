// Copyright (C) 2022, Lux Partners Limited, All rights reserved.
// See the file LICENSE for licensing terms.
package nodecmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/luxdefi/cli/pkg/constants"
	"github.com/luxdefi/cli/pkg/models"
	"github.com/luxdefi/cli/pkg/ssh"
	"github.com/luxdefi/cli/pkg/utils"
	"github.com/luxdefi/cli/pkg/ux"
	"github.com/luxdefi/cli/pkg/vm"

	"golang.org/x/exp/slices"
)

func checkHostsAreHealthy(hosts []*models.Host) ([]string, error) {
	ux.Logger.PrintToUser("Checking if node(s) are healthy ...")
	wg := sync.WaitGroup{}
	wgResults := models.NodeResults{}
	for _, host := range hosts {
		wg.Add(1)
		go func(nodeResults *models.NodeResults, host *models.Host) {
			defer wg.Done()
			if resp, err := ssh.RunSSHCheckHealthy(host); err != nil {
				nodeResults.AddResult(host.NodeID, nil, err)
				return
			} else {
				if isHealthy, err := parseHealthyOutput(resp); err != nil {
					nodeResults.AddResult(host.NodeID, nil, err)
				} else {
					nodeResults.AddResult(host.NodeID, isHealthy, err)
				}
			}
		}(&wgResults, host)
	}
	wg.Wait()
	if wgResults.HasErrors() {
		return nil, fmt.Errorf("failed to get health status for node(s) %s", wgResults.GetErrorHostMap())
	}
	return utils.Filter(wgResults.GetNodeList(), func(nodeID string) bool {
		return !wgResults.GetResultMap()[nodeID].(bool)
	}), nil
}

func parseHealthyOutput(byteValue []byte) (bool, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(byteValue, &result); err != nil {
		return false, err
	}
	isHealthyInterface, ok := result["result"].(map[string]interface{})
	if ok {
		isHealthy, ok := isHealthyInterface["healthy"].(bool)
		if ok {
			return isHealthy, nil
		}
	}
	return false, fmt.Errorf("unable to parse node healthy status")
}

func checkHostsAreBootstrapped(hosts []*models.Host) ([]string, error) {
	ux.Logger.PrintToUser("Checking if node(s) are bootstrapped to Primary Network ...")
	wg := sync.WaitGroup{}
	wgResults := models.NodeResults{}
	for _, host := range hosts {
		wg.Add(1)
		go func(nodeResults *models.NodeResults, host *models.Host) {
			defer wg.Done()
			if resp, err := ssh.RunSSHCheckBootstrapped(host); err != nil {
				nodeResults.AddResult(host.NodeID, nil, err)
				return
			} else {
				if isBootstrapped, err := parseBootstrappedOutput(resp); err != nil {
					nodeResults.AddResult(host.NodeID, nil, err)
				} else {
					nodeResults.AddResult(host.NodeID, isBootstrapped, err)
				}
			}
		}(&wgResults, host)
	}
	wg.Wait()
	if wgResults.HasErrors() {
		return nil, fmt.Errorf("failed to get node bootrapp status for node(s) %s", wgResults.GetErrorHostMap())
	}
	return utils.Filter(wgResults.GetNodeList(), func(nodeID string) bool {
		return !wgResults.GetResultMap()[nodeID].(bool)
	}), nil
}

func parseBootstrappedOutput(byteValue []byte) (bool, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(byteValue, &result); err != nil {
		return false, err
	}
	isBootstrappedInterface, ok := result["result"].(map[string]interface{})
	if ok {
		isBootstrapped, ok := isBootstrappedInterface["isBootstrapped"].(bool)
		if ok {
			return isBootstrapped, nil
		}
	}
	return false, errors.New("unable to parse node bootstrap status")
}

func checkLuxdVersionCompatible(hosts []*models.Host, subnetName string) ([]string, error) {
	ux.Logger.PrintToUser("Checking compatibility of node(s) lux go version with Subnet EVM RPC of subnet %s ...", subnetName)
	sc, err := app.LoadSidecar(subnetName)
	if err != nil {
		return nil, err
	}
	compatibleVersions, err := checkForCompatibleAvagoVersion(sc.RPCVersion)
	if err != nil {
		return nil, err
	}
	wg := sync.WaitGroup{}
	wgResults := models.NodeResults{}
	for _, host := range hosts {
		wg.Add(1)
		go func(nodeResults *models.NodeResults, host *models.Host) {
			defer wg.Done()
			if resp, err := ssh.RunSSHCheckLuxdVersion(host); err != nil {
				nodeResults.AddResult(host.NodeID, nil, err)
				return
			} else {
				if luxdVersion, err := parseLuxdOutput(resp); err != nil {
					nodeResults.AddResult(host.NodeID, nil, err)
				} else {
					nodeResults.AddResult(host.NodeID, luxdVersion, err)
				}
			}
		}(&wgResults, host)
	}
	wg.Wait()
	if wgResults.HasErrors() {
		return nil, fmt.Errorf("failed to get node version for node(s) %s", wgResults.GetErrorHostMap())
	}
	incompatibleNodes := []string{}
	for nodeID, luxdVersion := range wgResults.GetResultMap() {
		if !slices.Contains(compatibleVersions, fmt.Sprintf("%v", luxdVersion)) {
			incompatibleNodes = append(incompatibleNodes, nodeID)
		}
	}
	if len(incompatibleNodes) > 0 {
		ux.Logger.PrintToUser(fmt.Sprintf("Compatible Lux Go versions are %s", strings.Join(compatibleVersions, ", ")))
	}
	return incompatibleNodes, nil
}

func parseLuxdOutput(byteValue []byte) (string, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(byteValue, &result); err != nil {
		return "", err
	}
	nodeIDInterface, ok := result["result"].(map[string]interface{})
	if ok {
		vmVersions, ok := nodeIDInterface["vmVersions"].(map[string]interface{})
		if ok {
			luxdVersion, ok := vmVersions["platform"].(string)
			if ok {
				return luxdVersion, nil
			}
		}
	}
	return "", nil
}

func checkForCompatibleAvagoVersion(configuredRPCVersion int) ([]string, error) {
	compatibleAvagoVersions, err := vm.GetAvailableLuxdVersions(
		app, configuredRPCVersion, constants.LuxdCompatibilityURL)
	if err != nil {
		return nil, err
	}
	return compatibleAvagoVersions, nil
}

func disconnectHosts(hosts []*models.Host) {
	for _, host := range hosts {
		_ = host.Disconnect()
	}
}

func authorizedAccessFromSettings() bool {
	return app.Conf.GetConfigBoolValue(constants.ConfigAutorizeCloudAccessKey)
}
