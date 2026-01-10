// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package plugins

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/constants"
	"github.com/luxfi/sdk/models"
)

// Edits an Luxgo config file or creates one if it doesn't exist. Contains prompts unless forceWrite is set to true.
func EditConfigFile(
	app *application.Lux,
	chainID string,
	network models.Network,
	configFile string,
	forceWrite bool,
	chainLuxdConfigFile string,
) error {
	if !forceWrite {
		warn := "This will edit your existing config file. This edit is nondestructive,\n" +
			"but it's always good to have a backup."
		ux.Logger.PrintToUser("%s", warn)
		yes, err := app.Prompt.CaptureYesNo("Proceed?")
		if err != nil {
			return err
		}
		if !yes {
			ux.Logger.PrintToUser("Canceled by user")
			return nil
		}
	}
	fileBytes, err := os.ReadFile(configFile) //nolint:gosec // G304: Reading config from known location
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to load luxd config file %s: %w", configFile, err)
	}
	if fileBytes == nil {
		fileBytes = []byte("{}")
	}
	var luxdConfig map[string]interface{}
	if err := json.Unmarshal(fileBytes, &luxdConfig); err != nil {
		return fmt.Errorf("failed to unpack the config file %s to JSON: %w", configFile, err)
	}

	if chainLuxdConfigFile != "" {
		chainLuxdConfigFileBytes, err := os.ReadFile(chainLuxdConfigFile) //nolint:gosec // G304: Reading config from known location
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to load extra flags from blockchain luxd config file %s: %w", chainLuxdConfigFile, err)
		}
		var chainLuxdConfig map[string]interface{}
		if err := json.Unmarshal(chainLuxdConfigFileBytes, &chainLuxdConfig); err != nil {
			return fmt.Errorf("failed to unpack the config file %s to JSON: %w", chainLuxdConfigFile, err)
		}
		for k, v := range chainLuxdConfig {
			if k == "track-chains" || k == "whitelisted-chains" {
				ux.Logger.PrintToUser("ignoring configuration setting for %q, a blockchain luxd config file should not change it", k)
				continue
			}
			luxdConfig[k] = v
		}
	}

	// Banff.10: "track-chains" instead of "whitelisted-chains"
	oldVal := luxdConfig["track-chains"]
	if oldVal == nil {
		// check the old key in the config file for tracked-chains
		oldVal = luxdConfig["whitelisted-chains"]
	}

	newVal := ""
	if oldVal != nil {
		// if an entry already exists, we check if the chainID already is part
		// of the whitelisted-chains...
		exists := false
		var oldValStr string
		var ok bool
		if oldValStr, ok = oldVal.(string); !ok {
			return fmt.Errorf("expected a string value, but got %T", oldVal)
		}
		elems := strings.Split(oldValStr, ",")
		for _, s := range elems {
			if s == chainID {
				// ...if it is, we just don't need to update the value...
				newVal = oldVal.(string)
				exists = true
			}
		}
		// ...but if it is not, we concatenate the new chain to the existing ones
		if !exists {
			newVal = strings.Join([]string{oldVal.(string), chainID}, ",")
		}
	} else {
		// there were no entries yet, so add this chain as its new value
		newVal = chainID
	}

	// Banf.10 changes from "whitelisted-chains" to "track-chains"
	delete(luxdConfig, "whitelisted-chains")
	luxdConfig["track-chains"] = newVal
	luxdConfig["network-id"] = network.NetworkIDFlagValue()

	writeBytes, err := json.MarshalIndent(luxdConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to pack JSON to bytes for the config file: %w", err)
	}
	if err := os.WriteFile(configFile, writeBytes, constants.DefaultPerms755); err != nil {
		return fmt.Errorf("failed to write JSON config file, check permissions? %w", err)
	}
	msg := `The config file has been edited. To use it, make sure to start the node with the '--config-file' option, e.g.

./build/luxd --config-file %s

(using your binary location). The node has to be restarted for the changes to take effect.`
	ux.Logger.PrintToUser(msg, configFile)
	return nil
}
