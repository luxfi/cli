// Copyright (C) 2022, Lux Partners Limited, All rights reserved.
// See the file LICENSE for licensing terms.

package lpmintegration

import (
	"os"

	"github.com/luxdefi/lpm/lpm"
	"github.com/luxdefi/lpm/config"
	"github.com/luxdefi/cli/pkg/application"
	"github.com/luxdefi/cli/pkg/constants"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

// Note, you can only call this method once per run
func SetupLpm(app *application.Lux, lpmBaseDir string) error {
	credentials, err := initCredentials(app)
	if err != nil {
		return err
	}

	// Need to initialize a afero filesystem object to run lpm
	fs := afero.NewOsFs()

	err = os.MkdirAll(app.GetLPMPluginDir(), constants.DefaultPerms755)
	if err != nil {
		return err
	}

	// The New() function has a lot of prints we'd like to hide from the user,
	// so going to divert stdout to the log temporarily
	stdOutHolder := os.Stdout
	lpmLog, err := os.OpenFile(app.GetLPMLog(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, constants.DefaultPerms755)
	if err != nil {
		return err
	}
	defer lpmLog.Close()
	os.Stdout = lpmLog
	lpmConfig := lpm.Config{
		Directory:        lpmBaseDir,
		Auth:             credentials,
		AdminAPIEndpoint: app.Conf.GetConfigStringValue(constants.ConfigLPMAdminAPIEndpointKey),
		PluginDir:        app.GetLPMPluginDir(),
		Fs:               fs,
	}
	lpmInstance, err := lpm.New(lpmConfig)
	if err != nil {
		return err
	}
	os.Stdout = stdOutHolder
	app.Lpm = lpmInstance

	app.LpmDir = lpmBaseDir
	return err
}

// If we need to use custom git credentials (say for private repos).
// the zero value for credentials is safe to use.
// Stolen from LPM repo
func initCredentials(app *application.Lux) (http.BasicAuth, error) {
	result := http.BasicAuth{}

	if app.Conf.ConfigValueIsSet(constants.ConfigLPMCredentialsFileKey) {
		credentials := &config.Credential{}

		bytes, err := os.ReadFile(app.Conf.GetConfigStringValue(constants.ConfigLPMCredentialsFileKey))
		if err != nil {
			return result, err
		}
		if err := yaml.Unmarshal(bytes, credentials); err != nil {
			return result, err
		}

		result.Username = credentials.Username
		result.Password = credentials.Password
	}

	return result, nil
}
