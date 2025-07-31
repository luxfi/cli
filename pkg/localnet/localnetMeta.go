// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package localnet

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/utils"
)

// Local network metadata keeps reference to the tmpnet directory
// of the currently executing local network
type LocalNetworkMeta struct {
	NetworkDir string `json:"networkDir"`
}

// localNetworkMetaPath returns the path of the metadata file
func localNetworkMetaPath(app *application.Lux) string {
	return filepath.Join(app.GetBaseDir(), constants.LocalNetworkMetaFile)
}

// LocalNetworkMetaExists indicates if the metadata file exists
func LocalNetworkMetaExists(
	app *application.Lux,
) bool {
	return utils.FileExists(localNetworkMetaPath(app))
}

// GetLocalNetworkMeta returns the metadata contents
func GetLocalNetworkMeta(
	app *application.Lux,
) (*LocalNetworkMeta, error) {
	path := localNetworkMetaPath(app)
	bs, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed reading executing localnet meta file at %s: %w", path, err)
	}
	var meta LocalNetworkMeta
	if err := json.Unmarshal(bs, &meta); err != nil {
		return nil, fmt.Errorf("failed unmarshalling executing localnet meta file at %s: %w", path, err)
	}
	return &meta, nil
}

// SaveLocalNetworkMeta saves the tmpnet directory of the currently executing local network
// to the metadata file
func SaveLocalNetworkMeta(
	app *application.Lux,
	networkDir string,
) error {
	meta := LocalNetworkMeta{
		NetworkDir: networkDir,
	}
	bs, err := json.Marshal(&meta)
	if err != nil {
		return err
	}
	path := localNetworkMetaPath(app)
	if err := os.WriteFile(path, bs, constants.WriteReadUserOnlyPerms); err != nil {
		return fmt.Errorf("could not write executing localnet meta file %s: %w", path, err)
	}
	return nil
}

// RemoveLocalNetworkMeta removes the metadata file
func RemoveLocalNetworkMeta(
	app *application.Lux,
) error {
	path := localNetworkMetaPath(app)
	return os.RemoveAll(path)
}
