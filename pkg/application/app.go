// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package application

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/luxdefi/apm/apm"
	"github.com/luxdefi/cli/pkg/config"
	"github.com/luxdefi/cli/pkg/constants"
	"github.com/luxdefi/cli/pkg/models"
	"github.com/luxdefi/cli/pkg/prompts"
	"github.com/luxdefi/node/ids"
	"github.com/luxdefi/node/utils/logging"
	"github.com/luxdefi/subnet-evm/core"
)

const (
	WriteReadReadPerms = 0o644
)

type Lux struct {
	Log        logging.Logger
	baseDir    string
	Conf       *config.Config
	Prompt     prompts.Prompter
	Apm        *apm.APM
	ApmDir     string
	Downloader Downloader
}

func New() *Lux {
	return &Lux{}
}

func (app *Lux) Setup(baseDir string, log logging.Logger, conf *config.Config, prompt prompts.Prompter, downloader Downloader) {
	app.baseDir = baseDir
	app.Log = log
	app.Conf = conf
	app.Prompt = prompt
	app.Downloader = downloader
}

func (app *Lux) GetUpgradeFilesDir() string {
	return filepath.Join(app.baseDir, constants.UpgradeFilesDir)
}

func (app *Lux) GetRunFile() string {
	return filepath.Join(app.GetRunDir(), constants.ServerRunFile)
}

func (app *Lux) GetSnapshotsDir() string {
	return filepath.Join(app.baseDir, constants.SnapshotsDirName)
}

func (app *Lux) GetBaseDir() string {
	return app.baseDir
}

func (app *Lux) GetSubnetDir() string {
	return filepath.Join(app.baseDir, constants.SubnetDir)
}

func (app *Lux) GetReposDir() string {
	return filepath.Join(app.baseDir, constants.ReposDir)
}

func (app *Lux) GetRunDir() string {
	return filepath.Join(app.baseDir, constants.RunDir)
}

func (app *Lux) GetCustomVMDir() string {
	return filepath.Join(app.baseDir, constants.CustomVMDir)
}

func (app *Lux) GetNodeBinDir() string {
	return filepath.Join(app.baseDir, constants.LuxCliBinDir, constants.NodeInstallDir)
}

func (app *Lux) GetSubnetEVMBinDir() string {
	return filepath.Join(app.baseDir, constants.LuxCliBinDir, constants.SubnetEVMInstallDir)
}

func (app *Lux) GetSpacesVMBinDir() string {
	return filepath.Join(app.baseDir, constants.LuxCliBinDir, constants.SpacesVMInstallDir)
}

func (app *Lux) GetCustomVMPath(subnetName string) string {
	return filepath.Join(app.GetCustomVMDir(), subnetName)
}

func (app *Lux) GetAPMVMPath(vmid string) string {
	return filepath.Join(app.GetAPMPluginDir(), vmid)
}

func (app *Lux) GetGenesisPath(subnetName string) string {
	return filepath.Join(app.GetSubnetDir(), subnetName, constants.GenesisFileName)
}

func (app *Lux) GetSidecarPath(subnetName string) string {
	return filepath.Join(app.GetSubnetDir(), subnetName, constants.SidecarFileName)
}

func (app *Lux) GetKeyDir() string {
	return filepath.Join(app.baseDir, constants.KeyDir)
}

func (*Lux) GetTmpPluginDir() string {
	return os.TempDir()
}

func (app *Lux) GetAPMBaseDir() string {
	return filepath.Join(app.baseDir, "apm")
}

func (app *Lux) GetAPMLog() string {
	return filepath.Join(app.baseDir, constants.LogDir, constants.APMLogName)
}

func (app *Lux) GetAPMPluginDir() string {
	return filepath.Join(app.baseDir, constants.APMPluginDir)
}

func (app *Lux) GetKeyPath(keyName string) string {
	return filepath.Join(app.baseDir, constants.KeyDir, keyName+constants.KeySuffix)
}

func (app *Lux) GetDownloader() Downloader {
	return app.Downloader
}

func (*Lux) GetNodeCompatibilityURL() string {
	return constants.NodeCompatibilityURL
}

func (app *Lux) WriteGenesisFile(subnetName string, genesisBytes []byte) error {
	genesisPath := app.GetGenesisPath(subnetName)
	if err := os.MkdirAll(filepath.Dir(genesisPath), constants.DefaultPerms755); err != nil {
		return err
	}

	return os.WriteFile(genesisPath, genesisBytes, WriteReadReadPerms)
}

func (app *Lux) GenesisExists(subnetName string) bool {
	genesisPath := app.GetGenesisPath(subnetName)
	_, err := os.Stat(genesisPath)
	return err == nil
}

func (app *Lux) SidecarExists(subnetName string) bool {
	sidecarPath := app.GetSidecarPath(subnetName)
	_, err := os.Stat(sidecarPath)
	return err == nil
}

func (app *Lux) SubnetConfigExists(subnetName string) bool {
	// There's always a sidecar, but imported subnets don't have a genesis right now
	return app.SidecarExists(subnetName)
}

func (app *Lux) KeyExists(keyName string) bool {
	keyPath := app.GetKeyPath(keyName)
	_, err := os.Stat(keyPath)
	return err == nil
}

func (app *Lux) CopyGenesisFile(inputFilename string, subnetName string) error {
	genesisBytes, err := os.ReadFile(inputFilename)
	if err != nil {
		return err
	}
	genesisPath := app.GetGenesisPath(subnetName)
	if err := os.MkdirAll(filepath.Dir(genesisPath), constants.DefaultPerms755); err != nil {
		return err
	}

	return os.WriteFile(genesisPath, genesisBytes, WriteReadReadPerms)
}

func (app *Lux) CopyVMBinary(inputFilename string, subnetName string) error {
	vmBytes, err := os.ReadFile(inputFilename)
	if err != nil {
		return err
	}
	vmPath := app.GetCustomVMPath(subnetName)
	return os.WriteFile(vmPath, vmBytes, WriteReadReadPerms)
}

func (app *Lux) CopyKeyFile(inputFilename string, keyName string) error {
	keyBytes, err := os.ReadFile(inputFilename)
	if err != nil {
		return err
	}
	keyPath := app.GetKeyPath(keyName)
	return os.WriteFile(keyPath, keyBytes, WriteReadReadPerms)
}

func (app *Lux) LoadEvmGenesis(subnetName string) (core.Genesis, error) {
	genesisPath := app.GetGenesisPath(subnetName)
	jsonBytes, err := os.ReadFile(genesisPath)
	if err != nil {
		return core.Genesis{}, err
	}

	var gen core.Genesis
	err = json.Unmarshal(jsonBytes, &gen)
	return gen, err
}

func (app *Lux) LoadRawGenesis(subnetName string) ([]byte, error) {
	genesisPath := app.GetGenesisPath(subnetName)
	genesisBytes, err := os.ReadFile(genesisPath)
	if err != nil {
		return nil, err
	}

	return genesisBytes, err
}

func (app *Lux) CreateSidecar(sc *models.Sidecar) error {
	if sc.TokenName == "" {
		sc.TokenName = constants.DefaultTokenName
	}

	sidecarPath := app.GetSidecarPath(sc.Name)
	if err := os.MkdirAll(filepath.Dir(sidecarPath), constants.DefaultPerms755); err != nil {
		return err
	}

	// only apply the version on a write
	sc.Version = constants.SidecarVersion
	scBytes, err := json.MarshalIndent(sc, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile(sidecarPath, scBytes, WriteReadReadPerms)
}

func (app *Lux) LoadSidecar(subnetName string) (models.Sidecar, error) {
	sidecarPath := app.GetSidecarPath(subnetName)
	jsonBytes, err := os.ReadFile(sidecarPath)
	if err != nil {
		return models.Sidecar{}, err
	}

	var sc models.Sidecar
	err = json.Unmarshal(jsonBytes, &sc)

	if sc.TokenName == "" {
		sc.TokenName = constants.DefaultTokenName
	}

	return sc, err
}

func (app *Lux) UpdateSidecar(sc *models.Sidecar) error {
	sc.Version = constants.SidecarVersion
	scBytes, err := json.MarshalIndent(sc, "", "    ")
	if err != nil {
		return err
	}

	sidecarPath := app.GetSidecarPath(sc.Name)
	return os.WriteFile(sidecarPath, scBytes, WriteReadReadPerms)
}

func (app *Lux) UpdateSidecarNetworks(
	sc *models.Sidecar,
	network models.Network,
	subnetID ids.ID,
	blockchainID ids.ID,
) error {
	if sc.Networks == nil {
		sc.Networks = make(map[string]models.NetworkData)
	}
	sc.Networks[network.String()] = models.NetworkData{
		SubnetID:     subnetID,
		BlockchainID: blockchainID,
	}
	if err := app.UpdateSidecar(sc); err != nil {
		return fmt.Errorf("creation of chains and subnet was successful, but failed to update sidecar: %w", err)
	}
	return nil
}

func (app *Lux) GetTokenName(subnetName string) string {
	sidecar, err := app.LoadSidecar(subnetName)
	if err != nil {
		return constants.DefaultTokenName
	}
	return sidecar.TokenName
}

func (app *Lux) GetSidecarNames() ([]string, error) {
	matches, err := os.ReadDir(app.GetSubnetDir())
	if err != nil {
		return nil, err
	}

	var names []string
	for _, m := range matches {
		// a subnet dir could theoretically exist without a sidecar yet...
		if _, err := os.Stat(filepath.Join(app.GetSubnetDir(), m.Name(), constants.SidecarFileName)); err == nil {
			names = append(names, m.Name())
		}
	}
	return names, nil
}
