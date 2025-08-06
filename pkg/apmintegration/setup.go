// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package lpmintegration

import (
	"os"

	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/constants"
	clilpm "github.com/luxfi/cli/pkg/lpm"
	"github.com/luxfi/lpm/config"
	"github.com/luxfi/lpm/lpm"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

// Note, you can only call this method once per run
func SetupApm(app *application.Lux, lpmBaseDir string) error {
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
	_, err = lpm.New(lpmConfig) // We create but don't use directly
	if err != nil {
		return err
	}
	os.Stdout = stdOutHolder
	
	// Create a CLI LPM client using the same configuration
	app.Apm, err = clilpm.NewClient(
		lpmBaseDir,
		app.GetLPMPluginDir(),
		app.Conf.GetConfigStringValue(constants.ConfigLPMAdminAPIEndpointKey),
	)
	if err != nil {
		return err
	}

	app.ApmDir = func() string {
		return lpmBaseDir
	}
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
