// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package remoteconfig

import (
	"bytes"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/luxfi/cli/pkg/constants"
)

type LuxConfigInputs struct {
	HTTPHost                   string
	APIAdminEnabled            bool
	IndexEnabled               bool
	NetworkID                  string
	DBDir                      string
	LogDir                     string
	PublicIP                   string
	StateSyncEnabled           bool
	PruningEnabled             bool
	Aliases                    []string
	BlockChainID               string
	TrackSubnets               string
	BootstrapIDs               string
	BootstrapIPs               string
	PartialSync                bool
	GenesisPath                string
	UpgradePath                string
	ProposerVMUseCurrentHeight bool
}

func PrepareLuxConfig(publicIP string, networkID string, subnets []string) LuxConfigInputs {
	return LuxConfigInputs{
		HTTPHost:                   "127.0.0.1",
		NetworkID:                  networkID,
		DBDir:                      "/.luxd/db/",
		LogDir:                     "/.luxd/logs/",
		PublicIP:                   publicIP,
		StateSyncEnabled:           true,
		PruningEnabled:             false,
		TrackSubnets:               strings.Join(subnets, ","),
		Aliases:                    nil,
		BlockChainID:               "",
		ProposerVMUseCurrentHeight: constants.DevnetFlagsProposerVMUseCurrentHeight,
	}
}

func RenderLuxTemplate(templateName string, config LuxConfigInputs) ([]byte, error) {
	templateBytes, err := templates.ReadFile(templateName)
	if err != nil {
		return nil, err
	}
	helperFuncs := template.FuncMap{
		"join": strings.Join,
	}
	tmpl, err := template.New("config").Funcs(helperFuncs).Parse(string(templateBytes))
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func RenderLuxNodeConfig(config LuxConfigInputs) ([]byte, error) {
	if output, err := RenderLuxTemplate("templates/lux-node.tmpl", config); err != nil {
		return nil, err
	} else {
		return output, nil
	}
}

func RenderLuxCChainConfig(config LuxConfigInputs) ([]byte, error) {
	if output, err := RenderLuxTemplate("templates/lux-cchain.tmpl", config); err != nil {
		return nil, err
	} else {
		return output, nil
	}
}

func RenderLuxAliasesConfig(config LuxConfigInputs) ([]byte, error) {
	if output, err := RenderLuxTemplate("templates/lux-aliases.tmpl", config); err != nil {
		return nil, err
	} else {
		return output, nil
	}
}

func GetRemoteLuxNodeConfig() string {
	return filepath.Join(constants.CloudNodeConfigPath, constants.NodeFileName)
}

func GetRemoteLuxCChainConfig() string {
	return filepath.Join(constants.CloudNodeConfigPath, "chains", "C", "config.json")
}

func GetRemoteLuxGenesis() string {
	return filepath.Join(constants.CloudNodeConfigPath, constants.GenesisFileName)
}

func GetRemoteLuxUpgrade() string {
	return filepath.Join(constants.CloudNodeConfigPath, constants.UpgradeFileName)
}

func GetRemoteLuxAliasesConfig() string {
	return filepath.Join(constants.CloudNodeConfigPath, "chains", constants.AliasesFileName)
}

func LuxFolderToCreate() []string {
	return []string{
		"/home/ubuntu/.luxd/db",
		"/home/ubuntu/.luxd/logs",
		"/home/ubuntu/.luxd/configs",
		"/home/ubuntu/.luxd/configs/subnets/",
		"/home/ubuntu/.luxd/configs/chains/C",
		"/home/ubuntu/.luxd/staking",
		"/home/ubuntu/.luxd/plugins",
		"/home/ubuntu/.lux-cli/services/icm-relayer",
	}
}
