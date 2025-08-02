// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package plugins

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/luxfi/cli/v2/pkg/application"
	"github.com/luxfi/cli/v2/pkg/constants"
	"github.com/luxfi/cli/v2/pkg/models"
	"github.com/luxfi/cli/v2/pkg/ux"
)

// Edits an Luxgo config file or creates one if it doesn't exist. Contains prompts unless forceWrite is set to true.
func EditConfigFile(
	app *application.Lux,
	subnetID string,
	network models.Network,
	configFile string,
	forceWrite bool,
	subnetLuxdConfigFile string,
) error {
	if !forceWrite {
		warn := "This will edit your existing config file. This edit is nondestructive,\n" +
			"but it's always good to have a backup."
		ux.Logger.PrintToUser(warn)
		yes, err := app.Prompt.CaptureYesNo("Proceed?")
		if err != nil {
			return err
		}
		if !yes {
			ux.Logger.PrintToUser("Canceled by user")
			return nil
		}
	}
	fileBytes, err := os.ReadFile(configFile)
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

	if subnetLuxdConfigFile != "" {
		subnetLuxdConfigFileBytes, err := os.ReadFile(subnetLuxdConfigFile)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to load extra flags from blockchain luxd config file %s: %w", subnetLuxdConfigFile, err)
		}
		var subnetLuxdConfig map[string]interface{}
		if err := json.Unmarshal(subnetLuxdConfigFileBytes, &subnetLuxdConfig); err != nil {
			return fmt.Errorf("failed to unpack the config file %s to JSON: %w", subnetLuxdConfigFile, err)
		}
		for k, v := range subnetLuxdConfig {
			if k == "track-subnets" || k == "whitelisted-subnets" {
				ux.Logger.PrintToUser("ignoring configuration setting for %q, a blockchain luxd config file should not change it", k)
				continue
			}
			luxdConfig[k] = v
		}
	}

	// Banff.10: "track-subnets" instead of "whitelisted-subnets"
	oldVal := luxdConfig["track-subnets"]
	if oldVal == nil {
		// check the old key in the config file for tracked-subnets
		oldVal = luxdConfig["whitelisted-subnets"]
	}

	newVal := ""
	if oldVal != nil {
		// if an entry already exists, we check if the subnetID already is part
		// of the whitelisted-subnets...
		exists := false
		var oldValStr string
		var ok bool
		if oldValStr, ok = oldVal.(string); !ok {
			return fmt.Errorf("expected a string value, but got %T", oldVal)
		}
		elems := strings.Split(oldValStr, ",")
		for _, s := range elems {
			if s == subnetID {
				// ...if it is, we just don't need to update the value...
				newVal = oldVal.(string)
				exists = true
			}
		}
		// ...but if it is not, we concatenate the new subnet to the existing ones
		if !exists {
			newVal = strings.Join([]string{oldVal.(string), subnetID}, ",")
		}
	} else {
		// there were no entries yet, so add this subnet as its new value
		newVal = subnetID
	}

	// Banf.10 changes from "whitelisted-subnets" to "track-subnets"
	delete(luxdConfig, "whitelisted-subnets")
	luxdConfig["track-subnets"] = newVal
	networkID, err := network.NetworkID()
	if err != nil {
		return err
	}
	luxdConfig["network-id"] = networkID

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
