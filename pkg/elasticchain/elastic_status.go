// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package elasticchain

import (
	"errors"
	"os"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/models"
	"github.com/luxfi/cli/pkg/ux"
)

// GetLocalElasticChainsFromFile returns the list of local elastic chains.
func GetLocalElasticChainsFromFile(app *application.Lux) ([]string, error) {
	allChainDirs, err := os.ReadDir(app.GetChainsDir())
	if err != nil {
		return nil, err
	}

	elasticChains := []string{}

	for _, chainDir := range allChainDirs {
		if !chainDir.IsDir() {
			continue
		}
		// read sidecar file
		sc, err := app.LoadSidecar(chainDir.Name())
		if errors.Is(err, os.ErrNotExist) {
			// don't fail on missing sidecar file, just warn
			ux.Logger.PrintToUser("warning: inconsistent chain directory. No sidecar file found for chain %s", chainDir.Name())
			continue
		}
		if err != nil {
			return nil, err
		}

		// check if sidecar contains local elastic chains info in Elastic Chains map
		// if so, add to list of elastic chains
		if _, ok := sc.ElasticChain[models.Local.String()]; ok {
			elasticChains = append(elasticChains, sc.Name)
		}
	}

	return elasticChains, nil
}
