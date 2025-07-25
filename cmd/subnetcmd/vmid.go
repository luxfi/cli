// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package subnetcmd

import (
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/netrunner/utils"
	"github.com/spf13/cobra"
)

// lux subnet create
func vmidCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "vmid [vmName]",
		Short:        "Prints the VMID of a VM",
		Long:         `The subnet vmid command prints the virtual machine ID (VMID) for the given Subnet.`,
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		RunE:         printVMID,
	}
	return cmd
}

func printVMID(_ *cobra.Command, args []string) error {
	chains, err := validateSubnetNameAndGetChains(args)
	if err != nil {
		return err
	}

	chain := chains[0]
	vmID, err := utils.VMID(chain)
	if err != nil {
		return err
	}

	ux.Logger.PrintToUser("VM ID : %s", vmID.String())
	return nil
}
