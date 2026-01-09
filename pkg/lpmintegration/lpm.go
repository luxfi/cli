// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package lpmintegration

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/constantsants"
)

const gitExtension = ".git"

// AddRepo adds a new LPM repository and returns its alias.
func AddRepo(app *application.Lux, repoURL *url.URL, branch string) (string, error) {
	alias, err := getAlias(repoURL)
	if err != nil {
		return "", err
	}

	if alias == constants.DefaultLuxPackage {
		ux.Logger.PrintToUser("Lux Plugins Core already installed, skipping...")
		return "", nil
	}

	repoStr := repoURL.String()

	if path.Ext(repoStr) != gitExtension {
		repoStr += gitExtension
	}

	fmt.Println("Installing repo")

	return alias, app.Lpm.AddRepository(alias, repoStr, branch)
}

// UpdateRepos updates all LPM repositories.
func UpdateRepos(app *application.Lux) error {
	return app.Lpm.Update()
}

// InstallVM installs all VMs for a given subnet from LPM.
func InstallVM(app *application.Lux, subnetKey string) error {
	vms, err := getVMsInSubnet(app, subnetKey)
	if err != nil {
		return err
	}

	splitKey := strings.Split(subnetKey, ":")
	if len(splitKey) != 2 {
		return fmt.Errorf("invalid key: %s", subnetKey)
	}

	repo := splitKey[0]

	for _, vm := range vms {
		toInstall := repo + ":" + vm
		fmt.Println("Installing vm:", toInstall)
		err = app.Lpm.Install(toInstall)
		if err != nil {
			return err
		}
	}

	return nil
}
