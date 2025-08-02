// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package version

import (
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/luxfi/cli/v2/pkg/application"
	"github.com/luxfi/cli/v2/pkg/constants"
)

type CLIMinVersionMap struct {
	MinVersion string `json:"min-version"`
}

func CheckCLIVersionIsOverMin(app *application.Lux, version string) error {
	minVersionBytes, err := app.Downloader.Download(constants.CLIMinVersionURL)
	if err != nil {
		return err
	}

	var parsedMinVersion CLIMinVersionMap
	if err = json.Unmarshal(minVersionBytes, &parsedMinVersion); err != nil {
		return err
	}

	minVersion := parsedMinVersion.MinVersion
	// Add 'v' prefix if missing
	if !strings.HasPrefix(minVersion, "v") {
		minVersion = "v" + minVersion
	}
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	versionComparison := semver.Compare(version, minVersion)
	if versionComparison == -1 {
		return fmt.Errorf("CLI version is required to be at least %s, current CLI version is %s, please upgrade CLI by calling `lux update`", minVersion, version)
	}
	return nil
}
