// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package configcmd

import (
	"errors"

	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

// lux config metrics command
func newAuthorizeCloudAccessCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "authorize-cloud-access [enable | disable]",
		Short: "authorize access to cloud resources",
		Long:  "set preferences to authorize access to cloud resources",
		RunE:  handleAuthorizeCloudAccess,
		Args:  cobrautils.ExactArgs(1),
	}

	return cmd
}

func handleAuthorizeCloudAccess(_ *cobra.Command, args []string) error {
	switch args[0] {
	case constants.Enable:
		ux.Logger.PrintToUser("Thank you for authorizing Lux-CLI to access your Cloud account(s)")
		ux.Logger.PrintToUser("By enabling this setting you are authorizing Lux-CLI to:")
		ux.Logger.PrintToUser("- Create Cloud instance(s) and other components (such as elastic IPs)")
		ux.Logger.PrintToUser("- Start/Stop Cloud instance(s) and other components (such as elastic IPs) previously created by Lux-CLI")
		ux.Logger.PrintToUser("- Delete Cloud instance(s) and other components (such as elastic IPs) previously created by Lux-CLI")
		err := saveAuthorizeCloudAccessPreferences(true)
		if err != nil {
			return err
		}
	case constants.Disable:
		ux.Logger.PrintToUser("Lux-CLI Cloud access has been disabled.")
		ux.Logger.PrintToUser("You can re-enable Cloud access by running 'lux config authorize-cloud-access enable'")
		err := saveAuthorizeCloudAccessPreferences(false)
		if err != nil {
			return err
		}
	default:
		return errors.New("Invalid authorize-cloud-access argument '" + args[0] + "'")
	}
	return nil
}

func saveAuthorizeCloudAccessPreferences(enableAccess bool) error {
	return app.Conf.SetConfigValue(constants.ConfigAuthorizeCloudAccessKey, enableAccess)
}
