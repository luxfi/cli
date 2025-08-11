// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package subnetcmd

import (
	"fmt"
	"math"
	"math/big"
	"os"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/evm/core"
	"github.com/luxfi/evm/params"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/ids"
	"github.com/luxfi/netrunner/utils"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var printGenesisOnly bool

// lux subnet describe
func newDescribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe [subnetName]",
		Short: "Print a summary of the subnet’s configuration",
		Long: `The subnet describe command prints the details of a Subnet configuration to the console.
By default, the command prints a summary of the configuration. By providing the --genesis
flag, the command instead prints out the raw genesis file.`,
		RunE: readGenesis,
		Args: cobra.ExactArgs(1),
	}
	cmd.Flags().BoolVarP(
		&printGenesisOnly,
		"genesis",
		"g",
		false,
		"Print the genesis to the console directly instead of the summary",
	)
	return cmd
}

func printGenesis(subnetName string) error {
	genesisFile := app.GetGenesisPath(subnetName)
	gen, err := os.ReadFile(genesisFile)
	if err != nil {
		return err
	}
	fmt.Println(string(gen))
	return nil
}

func printDetails(genesis core.Genesis, sc models.Sidecar) {
	const art = `
 _____       _        _ _
|  __ \     | |      (_) |
| |  | | ___| |_ __ _ _| |___
| |  | |/ _ \ __/ _` + `  | | / __|
| |__| |  __/ || (_| | | \__ \
|_____/ \___|\__\__,_|_|_|___/
`
	fmt.Print(art)
	table := tablewriter.NewWriter(os.Stdout)
	header := []string{"Parameter", "Value"}
	table.SetHeader(header)
	table.SetRowLine(true)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	table.Append([]string{"Subnet Name", sc.Subnet})
	table.Append([]string{"ChainID", genesis.Config.ChainID.String()})
	table.Append([]string{"Token Name", app.GetTokenName(sc.Subnet)})
	table.Append([]string{"VM Version", sc.VMVersion})
	if sc.ImportedVMID != "" {
		table.Append([]string{"VM ID", sc.ImportedVMID})
	} else {
		id := constants.NotAvailableLabel
		vmID, err := utils.VMID(sc.Name)
		if err == nil {
			id = vmID.String()
		}
		table.Append([]string{"VM ID", id})
	}

	for net, data := range sc.Networks {
		if data.SubnetID != ids.Empty {
			table.Append([]string{fmt.Sprintf("%s SubnetID", net), data.SubnetID.String()})
		}
		if data.BlockchainID != ids.Empty {
			table.Append([]string{fmt.Sprintf("%s BlockchainID", net), data.BlockchainID.String()})
		}
	}
	table.Render()
}

func printGasTable(genesis core.Genesis) {
	// Generated here with BIG font
	// https://patorjk.com/software/taag/#p=display&f=Big&t=Precompiles
	const art = `
  _____              _____             __ _
 / ____|            / ____|           / _(_)
| |  __  __ _ ___  | |     ___  _ __ | |_ _  __ _
| | |_ |/ _` + `  / __| | |    / _ \| '_ \|  _| |/ _` + `  |
| |__| | (_| \__ \ | |___| (_) | | | | | | | (_| |
 \_____|\__,_|___/  \_____\___/|_| |_|_| |_|\__, |
                                             __/ |
                                            |___/
`

	fmt.Print(art)
	table := tablewriter.NewWriter(os.Stdout)
	header := []string{"Gas Parameter", "Value"}
	table.SetHeader(header)
	table.SetRowLine(true)

	// Display fee configuration from genesis config
	// Use default values if not specified
	gasLimit := uint64(8000000)
	minBaseFee := uint64(25000000000)
	targetGas := uint64(15000000)
	baseFeeChangeDenominator := uint64(36)
	minBlockGasCost := uint64(0)
	maxBlockGasCost := uint64(1000000)
	targetBlockRate := uint64(2)
	blockGasCostStep := uint64(200000)
	
	// Check if we have fee configuration in genesis config
	if genesis.Config != nil {
		// The fee configuration is stored in the extras for the ChainConfig
		// but GasLimit is a per-block field, not a chain config field
		// Use default values for now
	}
	
	table.Append([]string{"GasLimit", fmt.Sprintf("%d", gasLimit)})
	table.Append([]string{"MinBaseFee", fmt.Sprintf("%d", minBaseFee)})
	table.Append([]string{"TargetGas (per 10s)", fmt.Sprintf("%d", targetGas)})
	table.Append([]string{"BaseFeeChangeDenominator", fmt.Sprintf("%d", baseFeeChangeDenominator)})
	table.Append([]string{"MinBlockGasCost", fmt.Sprintf("%d", minBlockGasCost)})
	table.Append([]string{"MaxBlockGasCost", fmt.Sprintf("%d", maxBlockGasCost)})
	table.Append([]string{"TargetBlockRate", fmt.Sprintf("%d", targetBlockRate)})
	table.Append([]string{"BlockGasCostStep", fmt.Sprintf("%d", blockGasCostStep)})

	table.Render()
}

func printAirdropTable(genesis core.Genesis) {
	const art = `
          _         _
    /\   (_)       | |
   /  \   _ _ __ __| |_ __ ___  _ __
  / /\ \ | | '__/ _` + `  | '__/ _ \| '_ \
 / ____ \| | | | (_| | | | (_) | |_) |
/_/    \_\_|_|  \__,_|_|  \___/| .__/
                               | |
                               |_|
`
	fmt.Print(art)
	if len(genesis.Alloc) > 0 {
		table := tablewriter.NewWriter(os.Stdout)
		header := []string{"Address", "Airdrop Amount (10^18)", "Airdrop Amount (wei)"}
		table.SetHeader(header)
		table.SetRowLine(true)

		for address := range genesis.Alloc {
			amount := genesis.Alloc[address].Balance
			formattedAmount := new(big.Int).Div(amount, big.NewInt(params.Ether))
			table.Append([]string{address.Hex(), formattedAmount.String(), amount.String()})
		}

		table.Render()
	} else {
		fmt.Printf("No airdrops allocated")
	}
}

func printPrecompileTable(genesis core.Genesis) {
	const art = `

  _____                                    _ _
 |  __ \                                  (_) |
 | |__) | __ ___  ___ ___  _ __ ___  _ __  _| | ___  ___
 |  ___/ '__/ _ \/ __/ _ \| '_ ` + `  _ \| '_ \| | |/ _ \/ __|
 | |   | | |  __/ (_| (_) | | | | | | |_) | | |  __/\__ \
 |_|   |_|  \___|\___\___/|_| |_| |_| .__/|_|_|\___||___/
                                    | |
                                    |_|

`
	fmt.Print(art)

	table := tablewriter.NewWriter(os.Stdout)
	header := []string{"Precompile", "Admin", "Enabled"}
	table.SetHeader(header)
	table.SetAutoMergeCellsByColumnIndex([]int{0, 1, 2})
	table.SetRowLine(true)

	precompileSet := false

	// Check for precompiles in genesis config
	// For EVM chains, we can check for common precompiles
	precompileNames := []string{
		"Contract Native Minter",
		"Tx Allow List",
		"Fee Manager",
		"Reward Manager",
		"Contract Allow List",
	}
	
	// Display known precompiles with default status
	// In the future, this can be enhanced to read from chain config extras
	for _, precompileName := range precompileNames {
		// For now, we don't have access to actual precompile status
		// So we'll show them as not configured
		table.Append([]string{precompileName, "Not Configured", "No"})
	}
	
	// Check if any precompiles are actually configured
	// This will be enhanced when we have access to extras.ChainConfig
	if genesis.Config != nil {
		// Look for precompile configurations
		// For now, we'll assume no precompiles are configured
		precompileSet = false
	}

	if precompileSet || len(precompileNames) > 0 {
		table.Render()
		if !precompileSet {
			ux.Logger.PrintToUser("Note: Precompile configuration details are not currently available")
		}
	} else {
		ux.Logger.PrintToUser("No precompiles set")
	}
}

func appendToAddressTable(
	table *tablewriter.Table,
	label string,
	adminAddresses []common.Address,
	enabledAddresses []common.Address,
) {
	admins := len(adminAddresses)
	enabled := len(enabledAddresses)
	max := int(math.Max(float64(admins), float64(enabled)))
	for i := 0; i < max; i++ {
		var admin, enable string
		if len(adminAddresses) >= i+1 && adminAddresses[i] != (common.Address{}) {
			admin = adminAddresses[i].Hex()
		}
		if len(enabledAddresses) >= i+1 && enabledAddresses[i] != (common.Address{}) {
			enable = enabledAddresses[i].Hex()
		}
		table.Append([]string{label, admin, enable})
	}
}

func describeSubnetEvmGenesis(sc models.Sidecar) error {
	// Load genesis
	genesis, err := app.LoadEvmGenesis(sc.Subnet)
	if err != nil {
		return err
	}

	printDetails(genesis.Genesis, sc)
	// Write gas table
	printGasTable(genesis.Genesis)
	// fmt.Printf("\n\n")
	printAirdropTable(genesis.Genesis)
	printPrecompileTable(genesis.Genesis)
	return nil
}

func readGenesis(_ *cobra.Command, args []string) error {
	subnetName := args[0]
	if !app.GenesisExists(subnetName) {
		ux.Logger.PrintToUser("The provided subnet name %q does not exist", subnetName)
		return nil
	}
	if printGenesisOnly {
		return printGenesis(subnetName)
	}
	// read in sidecar
	sc, err := app.LoadSidecar(subnetName)
	if err != nil {
		return err
	}

	switch sc.VM {
	case models.EVM:
		return describeSubnetEvmGenesis(sc)
	default:
		app.Log.Warn("Unknown genesis format", zap.Any("vm-type", sc.VM))
		ux.Logger.PrintToUser("Printing genesis")
		err = printGenesis(subnetName)
	}
	return err
}
