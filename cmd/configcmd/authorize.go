// Copyright (C) 2022, Lux Partners Limited, All rights reserved.
// See the file LICENSE for licensing terms.
package configcmd

import (
	"errors"

	"github.com/luxdefi/cli/pkg/constants"
	"github.com/luxdefi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

// lux config metrics command
func newAutorizeCloudAccessCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "authorize-cloud-access [enable | disable]",
		Short:        "authorize access to cloud resources",
		Long:         "set preferences to authorize access to cloud resources",
		RunE:         handleAutorizeCloudAccess,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
	}

	return cmd
}

func handleAutorizeCloudAccess(_ *cobra.Command, args []string) error {
	switch args[0] {
	case constants.Enable:
		ux.Logger.PrintToUser("Thank you for authorizing Lux-CLI to access your Cloud account(s)")
		ux.Logger.PrintToUser("By enabling this setting you are authorizing Lux-CLI to:")
		ux.Logger.PrintToUser("- Create Cloud instance(s) and other components (such as elastic IPs)")
		ux.Logger.PrintToUser("- Start/Stop Cloud instance(s) and other components (such as elastic IPs) previously created by Lux-CLI")
		ux.Logger.PrintToUser("- Delete Cloud instance(s) and other components (such as elastic IPs) previously created by Lux-CLI")
		err := saveAutorizeCloudAccessPreferences(true)
		if err != nil {
			return err
		}
	case constants.Disable:
		ux.Logger.PrintToUser("Lux-CLI Cloud access has been disabled.")
		ux.Logger.PrintToUser("You can re-enable Cloud access by running 'lux config authorize-cloud-access enable'")
		err := saveAutorizeCloudAccessPreferences(false)
		if err != nil {
			return err
		}
	default:
		return errors.New("Invalid authorize-cloud-access argument '" + args[0] + "'")
	}
	return nil
}

func saveAutorizeCloudAccessPreferences(enableAccess bool) error {
	return app.Conf.SetConfigValue(constants.ConfigAutorizeCloudAccessKey, enableAccess)
}
