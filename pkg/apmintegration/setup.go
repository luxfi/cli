// Copyright (C) 2022, Lux Partners Limited, All rights reserved.
// See the file LICENSE for licensing terms.

package apmintegration

import (
	"os"

	"github.com/luxdefi/apm/apm"
	"github.com/luxdefi/apm/config"
	"github.com/luxdefi/cli/pkg/application"
	"github.com/luxdefi/cli/pkg/constants"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

// Note, you can only call this method once per run
func SetupApm(app *application.Lux, apmBaseDir string) error {
	credentials, err := initCredentials(app)
	if err != nil {
		return err
	}

	// Need to initialize a afero filesystem object to run apm
	fs := afero.NewOsFs()

	err = os.MkdirAll(app.GetAPMPluginDir(), constants.DefaultPerms755)
	if err != nil {
		return err
	}

	// The New() function has a lot of prints we'd like to hide from the user,
	// so going to divert stdout to the log temporarily
	stdOutHolder := os.Stdout
	apmLog, err := os.OpenFile(app.GetAPMLog(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, constants.DefaultPerms755)
	if err != nil {
		return err
	}
	defer apmLog.Close()
	os.Stdout = apmLog
	apmConfig := apm.Config{
		Directory:        apmBaseDir,
		Auth:             credentials,
		AdminAPIEndpoint: app.Conf.GetConfigStringValue(constants.ConfigAPMAdminAPIEndpointKey),
		PluginDir:        app.GetAPMPluginDir(),
		Fs:               fs,
	}
	apmInstance, err := apm.New(apmConfig)
	if err != nil {
		return err
	}
	os.Stdout = stdOutHolder
	app.Apm = apmInstance

	app.ApmDir = apmBaseDir
	return err
}

// If we need to use custom git credentials (say for private repos).
// the zero value for credentials is safe to use.
// Stolen from APM repo
func initCredentials(app *application.Lux) (http.BasicAuth, error) {
	result := http.BasicAuth{}

	if app.Conf.ConfigValueIsSet(constants.ConfigAPMCredentialsFileKey) {
		credentials := &config.Credential{}

		bytes, err := os.ReadFile(app.Conf.GetConfigStringValue(constants.ConfigAPMCredentialsFileKey))
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
