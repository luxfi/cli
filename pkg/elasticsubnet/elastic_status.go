// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package elasticsubnet

import (
	"os"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/sdk/models"
)

func GetLocalElasticChainsFromFile(app *application.Lux) ([]string, error) {
	allSubnetDirs, err := os.ReadDir(app.GetChainsDir())
	if err != nil {
		return nil, err
	}

	elasticSubnets := []string{}

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

		// check if sidecar contains local elastic subnets info in Elastic Subnets map
		// if so, add to list of elastic subnets
		if _, ok := sc.ElasticChain[models.Local.String()]; ok {
			elasticSubnets = append(elasticSubnets, sc.Name)
		}
	}

	return elasticSubnets, nil
}
