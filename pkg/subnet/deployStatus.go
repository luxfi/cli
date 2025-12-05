// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package subnet

import (
	"os"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/sdk/models"
)

func GetLocallyDeployedSubnetsFromFile(app *application.Lux) ([]string, error) {
	allSubnetDirs, err := os.ReadDir(app.GetSubnetDir())
	if err != nil {
		return nil, err
	}

	deployedSubnets := []string{}

	for _, subnetDir := range allSubnetDirs {
		if !subnetDir.IsDir() {
			continue
		}
		// read sidecar file
		sc, err := app.LoadSidecar(subnetDir.Name())
		if err == os.ErrNotExist {
			// don't fail on missing sidecar file, just warn
			ux.Logger.PrintToUser("warning: inconsistent subnet directory. No sidecar file found for subnet %s", subnetDir.Name())
			continue
		}
		if err != nil {
			return nil, err
		}

		// check if sidecar contains local deployment info in Networks map
		// if so, add to list of deployed subnets
		if _, ok := sc.Networks[models.Local.String()]; ok {
			deployedSubnets = append(deployedSubnets, sc.Name)
		}
	}

	return deployedSubnets, nil
}

// GetLocallyDeployedSubnetIDs returns a list of subnet IDs for locally deployed subnets
// This is used for auto-tracking subnets when starting the local network
func GetLocallyDeployedSubnetIDs(app *application.Lux) ([]string, error) {
	allSubnetDirs, err := os.ReadDir(app.GetSubnetDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	subnetIDs := []string{}

	for _, subnetDir := range allSubnetDirs {
		if !subnetDir.IsDir() {
			continue
		}
		// read sidecar file
		sc, err := app.LoadSidecar(subnetDir.Name())
		if err != nil {
			continue // skip on any error
		}

		// check if sidecar contains local deployment info with a valid SubnetID
		if network, ok := sc.Networks[models.Local.String()]; ok {
			if network.SubnetID.String() != "" && network.SubnetID.String() != "11111111111111111111111111111111LpoYY" {
				subnetIDs = append(subnetIDs, network.SubnetID.String())
			}
		}
	}

	return subnetIDs, nil
}
