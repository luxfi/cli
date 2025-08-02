// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package configcmd

import (
	"errors"

	"github.com/luxfi/cli/v2/v2/pkg/cobrautils"
	"github.com/luxfi/cli/v2/v2/pkg/constants"
	"github.com/luxfi/cli/v2/v2/pkg/ux"
	"github.com/spf13/cobra"
)

// lux config metrics command
func newMetricsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics [enable | disable]",
		Short: "opt in or out of metrics collection",
		Long:  "set user metrics collection preferences",
		RunE:  handleMetricsSettings,
		Args:  cobrautils.ExactArgs(1),
	}

	return cmd
}

func handleMetricsSettings(_ *cobra.Command, args []string) error {
	switch args[0] {
	case constants.Enable:
		ux.Logger.PrintToUser("Thank you for opting in Lux CLI usage metrics collection")
		err := saveMetricsPreferences(true)
		if err != nil {
			return err
		}
	case constants.Disable:
		ux.Logger.PrintToUser("Lux CLI usage metrics will no longer be collected")
		err := saveMetricsPreferences(false)
		if err != nil {
			return err
		}
	default:
		return errors.New("Invalid metrics argument '" + args[0] + "'")
	}
	return nil
}

func saveMetricsPreferences(enableMetrics bool) error {
	return app.Conf.SetConfigValue(constants.ConfigMetricsEnabledKey, enableMetrics)
}
