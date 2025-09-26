// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package flags

import (
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/sdk/prompts"
	"github.com/spf13/cobra"
)

const (
	rpcURLFLag = "rpc"
)

func AddRPCFlagToCmd(cmd *cobra.Command, app *application.Lux, rpc *string) {
	cmd.Flags().StringVar(rpc, rpcURLFLag, "", "blockchain rpc endpoint")

	rpcPreRun := func(cmd *cobra.Command, args []string) error {
		if err := ValidateRPC(app, rpc, cmd, args); err != nil {
			return err
		}
		return nil
	}

	existingPreRunE := cmd.PreRunE
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if existingPreRunE != nil {
			if err := existingPreRunE(cmd, args); err != nil {
				return err
			}
		}
		return rpcPreRun(cmd, args)
	}
}

func ValidateRPC(app *application.Lux, rpc *string, cmd *cobra.Command, args []string) error {
	var err error
	// Prompt for RPC endpoint when needed for certain commands
	if *rpc == "" {
		// Commands that require RPC endpoint
		requiresRPC := map[string]bool{
			"addValidator":     true,
			"deploy":          true,
			"removeValidator": true,
			"status":          true,
		}
		
		if requiresRPC[cmd.Name()] && len(args) == 0 {
			*rpc, err = app.Prompt.CaptureURL("What is the RPC endpoint?")
			if err != nil {
				return err
			}
		}
		return nil
	}
	return prompts.ValidateURLFormat(*rpc)
}
