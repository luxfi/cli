// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package validatorcmd

import (
	"fmt"

	"github.com/luxfi/cli/v2/pkg/cobrautils"
	"github.com/luxfi/cli/v2/pkg/contract"
	"github.com/luxfi/cli/v2/pkg/networkoptions"
	"github.com/luxfi/cli/v2/pkg/ux"
	"github.com/luxfi/cli/v2/sdk/validator"
	"github.com/luxfi/node/v2/utils/units"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func NewListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [blockchainName]",
		Short: "Lists the validators of an L1",
		Long:  `This command gets a list of the validators of the L1`,
		RunE:  list,
		Args:  cobrautils.ExactArgs(1),
	}
	networkoptions.AddNetworkFlagsToCmd(cmd, &globalNetworkFlags, true, networkoptions.DefaultSupportedNetworkOptions)
	return cmd
}

func list(_ *cobra.Command, args []string) error {
	blockchainName := args[0]
	sc, err := app.LoadSidecar(blockchainName)
	if err != nil {
		return fmt.Errorf("failed to load sidecar: %w", err)
	}
	if !sc.Sovereign {
		return fmt.Errorf("lux validator commands are only applicable to sovereign L1s")
	}

	network, err := networkoptions.GetNetworkFromCmdLineFlags(
		app,
		"",
		globalNetworkFlags,
		true,
		false,
		networkoptions.DefaultSupportedNetworkOptions,
		"",
	)
	if err != nil {
		return err
	}

	chainSpec := contract.ChainSpec{
		BlockchainName: blockchainName,
	}

	subnetID, err := contract.GetSubnetID(app, network, chainSpec)
	if err != nil {
		return err
	}

	validators, err := validator.GetCurrentValidators(network.SDKNetwork(), subnetID)
	if err != nil {
		return err
	}

	t := ux.DefaultTable(
		fmt.Sprintf("%s Validators", blockchainName),
		table.Row{"Node ID", "Validation ID", "Weight", "Remaining Balance (LUX)"},
	)
	for _, validator := range validators {
		t.AppendRow(table.Row{
			validator.NodeID,
			validator.ValidationID,
			validator.Weight,
			float64(validator.Balance) / float64(units.Lux),
		})
	}
	fmt.Println(t.Render())

	return nil
}
