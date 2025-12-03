// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package localnet

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/utils"
)

// SubnetStateInfo stores information about a deployed subnet for state persistence.
type SubnetStateInfo struct {
	SubnetID     string `json:"subnetId"`
	BlockchainID string `json:"blockchainId"`
	VMID         string `json:"vmId"`
	Name         string `json:"name"`
}

// ValidatorStateInfo stores information about a validator for state persistence.
type ValidatorStateInfo struct {
	NodeID    string `json:"nodeId"`
	SubnetID  string `json:"subnetId"`
	Weight    uint64 `json:"weight"`
	StartTime uint64 `json:"startTime"`
	EndTime   uint64 `json:"endTime"`
}

// NetworkStateData stores state persistence data for the local network.
// This data survives start/stop cycles and ensures P-Chain state, subnet registrations,
// validator sets, and balances are preserved across restarts.
type NetworkStateData struct {
	// TrackedSubnets is a list of subnet IDs that should be automatically tracked on restart
	TrackedSubnets []string `json:"trackedSubnets,omitempty"`
	// DevMode when true, tracks all subnets automatically
	DevMode bool `json:"devMode,omitempty"`
	// State persistence for P-Chain data
	Subnets     []SubnetStateInfo    `json:"subnets,omitempty"`
	Validators  []ValidatorStateInfo `json:"validators,omitempty"`
	NetworkID   uint32               `json:"networkId,omitempty"`
	LastSavedAt string               `json:"lastSavedAt,omitempty"`
}

const networkStateFilename = "network_state.json"

// SaveNetworkState saves the current P-Chain state (subnets, validators) to disk.
func SaveNetworkState(app *application.Lux) error {
	rootDataDir, err := GetLocalNetworkDir(app)
	if err != nil {
		return err
	}

	stateData := NetworkStateData{}
	statePath := filepath.Join(rootDataDir, networkStateFilename)
	if utils.FileExists(statePath) {
		bs, err := os.ReadFile(statePath)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(bs, &stateData); err != nil {
			return err
		}
	}

	blockchains, err := GetLocalNetworkBlockchainsInfo(app)
	if err == nil && len(blockchains) > 0 {
		stateData.Subnets = make([]SubnetStateInfo, 0, len(blockchains))
		for _, chain := range blockchains {
			stateData.Subnets = append(stateData.Subnets, SubnetStateInfo{
				SubnetID:     chain.SubnetID.String(),
				BlockchainID: chain.ID.String(),
				VMID:         chain.VMID.String(),
			})
		}
	}

	stateData.LastSavedAt = time.Now().UTC().Format(time.RFC3339)

	bs, err := json.MarshalIndent(&stateData, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath, bs, constants.WriteReadReadPerms)
}

// GetSavedNetworkState returns saved state from a previous session if it exists.
func GetSavedNetworkState(app *application.Lux) (bool, NetworkStateData, error) {
	stateData := NetworkStateData{}
	rootDataDir, err := GetLocalNetworkDir(app)
	if err != nil {
		return false, stateData, err
	}

	statePath := filepath.Join(rootDataDir, networkStateFilename)
	if !utils.FileExists(statePath) {
		return false, stateData, nil
	}

	bs, err := os.ReadFile(statePath)
	if err != nil {
		return false, stateData, err
	}
	if err := json.Unmarshal(bs, &stateData); err != nil {
		return false, stateData, err
	}
	return true, stateData, nil
}

// ClearNetworkState removes all saved network state (for fresh starts).
func ClearNetworkState(app *application.Lux) error {
	rootDataDir, err := GetLocalNetworkDir(app)
	if err != nil {
		return err
	}

	statePath := filepath.Join(rootDataDir, networkStateFilename)
	if utils.FileExists(statePath) {
		return os.Remove(statePath)
	}
	return nil
}

// AddTrackedSubnet adds a subnet ID to the list of tracked subnets.
func AddTrackedSubnet(app *application.Lux, subnetID string) error {
	rootDataDir, err := GetLocalNetworkDir(app)
	if err != nil {
		return err
	}

	stateData := NetworkStateData{}
	statePath := filepath.Join(rootDataDir, networkStateFilename)
	if utils.FileExists(statePath) {
		bs, err := os.ReadFile(statePath)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(bs, &stateData); err != nil {
			return err
		}
	}

	for _, tracked := range stateData.TrackedSubnets {
		if tracked == subnetID {
			return nil
		}
	}

	stateData.TrackedSubnets = append(stateData.TrackedSubnets, subnetID)

	bs, err := json.MarshalIndent(&stateData, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath, bs, constants.WriteReadReadPerms)
}

// GetTrackedSubnets returns the list of subnet IDs that should be tracked on restart.
func GetTrackedSubnets(app *application.Lux) ([]string, error) {
	exists, data, err := GetSavedNetworkState(app)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	return data.TrackedSubnets, nil
}

// SetDevMode enables or disables dev mode for the local network.
func SetDevMode(app *application.Lux, enabled bool) error {
	rootDataDir, err := GetLocalNetworkDir(app)
	if err != nil {
		return err
	}

	stateData := NetworkStateData{}
	statePath := filepath.Join(rootDataDir, networkStateFilename)
	if utils.FileExists(statePath) {
		bs, err := os.ReadFile(statePath)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(bs, &stateData); err != nil {
			return err
		}
	}

	stateData.DevMode = enabled

	bs, err := json.MarshalIndent(&stateData, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath, bs, constants.WriteReadReadPerms)
}

// IsDevModeEnabled returns true if dev mode is enabled for the local network.
func IsDevModeEnabled(app *application.Lux) (bool, error) {
	exists, data, err := GetSavedNetworkState(app)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}
	return data.DevMode, nil
}

// RemoveTrackedSubnet removes a subnet ID from the list of tracked subnets.
func RemoveTrackedSubnet(app *application.Lux, subnetID string) error {
	rootDataDir, err := GetLocalNetworkDir(app)
	if err != nil {
		return err
	}

	stateData := NetworkStateData{}
	statePath := filepath.Join(rootDataDir, networkStateFilename)
	if utils.FileExists(statePath) {
		bs, err := os.ReadFile(statePath)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(bs, &stateData); err != nil {
			return err
		}
	}

	var newTracked []string
	for _, tracked := range stateData.TrackedSubnets {
		if tracked != subnetID {
			newTracked = append(newTracked, tracked)
		}
	}
	stateData.TrackedSubnets = newTracked

	bs, err := json.MarshalIndent(&stateData, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath, bs, constants.WriteReadReadPerms)
}
