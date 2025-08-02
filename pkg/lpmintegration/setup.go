// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package lpmintegration

import (
	"os"

	"github.com/luxfi/cli/v2/pkg/application"
	"github.com/luxfi/cli/v2/pkg/constants"
	"github.com/luxfi/cli/v2/pkg/lpm"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

const (
	credentialsFileKey  = "credentials-file"
	adminAPIEndpointKey = "admin-api-endpoint"
)

// Credential represents git authentication credentials
type Credential struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// Note, you can only call this method once per run
func SetupLpm(app *application.Lux, lpmBaseDir string) error {
	// Note: credentials not used in LPM currently, but keeping for future auth
	_, err := initCredentials()
	if err != nil {
		return err
	}


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
	lpmInstance, err := lpm.NewClient(
		lpmBaseDir,
		app.GetLPMPluginDir(),
		viper.GetString(adminAPIEndpointKey),
	)
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
func initCredentials() (http.BasicAuth, error) {
	result := http.BasicAuth{}

	if viper.IsSet(credentialsFileKey) {
		credentials := &Credential{}

		bytes, err := os.ReadFile(viper.GetString(credentialsFileKey))
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
