// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package subnetcmd

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/luxfi/cli/cmd/flags"
	"github.com/luxfi/cli/pkg/models"
	"github.com/luxfi/cli/pkg/subnet"
	"github.com/luxfi/ids"
	"github.com/luxfi/node/vms/platformvm"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var (
	validatorsLocal   bool
	validatorsTestnet bool
	validatorsMainnet bool
)

// lux subnet validators
func newValidatorsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validators [subnetName]",
		Short: "List a subnet's validators",
		Long: `The subnet validators command lists the validators of a subnet and provides
severarl statistics about them.`,
		RunE:         printValidators,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
	}
	cmd.Flags().BoolVarP(&validatorsLocal, "local", "l", false, "deploy to a local network")
	cmd.Flags().BoolVarP(&validatorsTestnet, "testnet", "t", false, "deploy to testnet (alias to `testnet`)")
	cmd.Flags().BoolVarP(&validatorsTestnet, "testnet", "f", false, "deploy to testnet (alias to `testnet`")
	cmd.Flags().BoolVarP(&validatorsMainnet, "mainnet", "m", false, "deploy to mainnet")
	return cmd
}

func printValidators(_ *cobra.Command, args []string) error {
	if !flags.EnsureMutuallyExclusive([]bool{validatorsLocal, validatorsTestnet, validatorsMainnet}) {
		return errMutuallyExlusiveNetworks
	}

	var network models.Network
	switch {
	case validatorsLocal:
		network = models.Local
	case validatorsTestnet:
		network = models.Testnet
	case validatorsMainnet:
		network = models.Mainnet
	}

	if network == models.Undefined {
		// no flag was set, prompt user
		networkStr, err := app.Prompt.CaptureList(
			"Choose a network to list validators from",
			[]string{models.Local.String(), models.Testnet.String(), models.Mainnet.String()},
		)
		if err != nil {
			return err
		}
		network = models.NetworkFromString(networkStr)
	}

	// get the subnetID
	sc, err := app.LoadSidecar(args[0])
	if err != nil {
		return err
	}

	deployInfo, ok := sc.Networks[network.String()]
	if !ok {
		return errors.New("no deployment found for subnet")
	}

	subnetID := deployInfo.SubnetID

	if network == models.Local {
		return printLocalValidators(subnetID)
	} else {
		return printPublicValidators(subnetID, network)
	}
}

func printLocalValidators(subnetID ids.ID) error {
	validators, err := subnet.GetSubnetValidators(subnetID)
	if err != nil {
		return err
	}

	return printValidatorsFromList(validators)
}

func printPublicValidators(subnetID ids.ID, network models.Network) error {
	validators, err := subnet.GetPublicSubnetValidators(subnetID, network)
	if err != nil {
		return err
	}

	return printValidatorsFromList(validators)
}

func printValidatorsFromList(validators []platformvm.ClientPermissionlessValidator) error {
	header := []string{"NodeID", "Stake Amount", "Delegator Weight", "Start Time", "End Time", "Type"}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(header)
	table.SetRowLine(true)

	for _, validator := range validators {
		var delegatorWeight uint64
		if validator.DelegatorWeight != nil {
			delegatorWeight = *validator.DelegatorWeight
		}

		validatorType := "permissioned"
		if validator.PotentialReward != nil && *validator.PotentialReward > 0 {
			validatorType = "elastic"
		}

		table.Append([]string{
			validator.NodeID.String(),
			strconv.FormatUint(validator.Weight, 10),
			strconv.FormatUint(delegatorWeight, 10),
			formatUnixTime(validator.StartTime),
			formatUnixTime(validator.EndTime),
			validatorType,
		})
	}

	table.Render()

	return nil
}

func formatUnixTime(unixTime uint64) string {
	return time.Unix(int64(unixTime), 0).Format(time.RFC3339)
}
