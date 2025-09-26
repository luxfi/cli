// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package blockchaincmd

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/sdk/contract"
	warpgenesis "github.com/luxfi/cli/pkg/interchain/genesis"
	"github.com/luxfi/cli/pkg/localnet"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/cli/pkg/subnet"
	"github.com/luxfi/cli/pkg/txutils"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/cli/pkg/vm"
	validatorManagerSDK "github.com/luxfi/sdk/validatormanager"
	"github.com/luxfi/evm/core"
	"github.com/luxfi/evm/params"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/ids"
	luxlog "github.com/luxfi/log"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var printGenesisOnly bool

// lux blockchain describe
func newDescribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe [blockchainName]",
		Short: "Print a summary of the blockchain’s configuration",
		Long: `The blockchain describe command prints the details of a Blockchain configuration to the console.
By default, the command prints a summary of the configuration. By providing the --genesis
flag, the command instead prints out the raw genesis file.`,
		RunE: describe,
		Args: cobrautils.ExactArgs(1),
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

func printGenesis(blockchainName string) error {
	genesisFile := app.GetGenesisPath(blockchainName)
	gen, err := os.ReadFile(genesisFile)
	if err != nil {
		return err
	}
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser(string(gen))
	return nil
}

func PrintSubnetInfo(blockchainName string, onlyLocalnetInfo bool) error {
	sc, err := app.LoadSidecar(blockchainName)
	if err != nil {
		return err
	}

	genesisBytes, err := app.LoadRawGenesis(sc.Subnet)
	if err != nil {
		return err
	}

	// VM/Deploys
	t := ux.DefaultTable(sc.Name, nil)
	// SetColumnConfigs not available in tablewriter, skip it
	t.Append([]string{"Name", sc.Name, sc.Name})
	vmIDstr := sc.ImportedVMID
	if vmIDstr == "" {
		vmID, err := utils.VMID(sc.Name)
		if err == nil {
			vmIDstr = vmID.String()
		} else {
			vmIDstr = constants.NotAvailableLabel
		}
	}
	t.Append([]string{"VM ID", vmIDstr, vmIDstr})
	t.Append([]string{"VM Version", sc.VMVersion, sc.VMVersion})
	t.Append([]string{"Validation", sc.ValidatorManagement, sc.ValidatorManagement})

	locallyDeployed := false
	localEndpoint := ""
	localChainID := ""
	for net, data := range sc.Networks {
		network, err := app.GetNetworkFromSidecarNetworkName(net)
		if err != nil {
			ux.Logger.RedXToUser("%s is supposed to be deployed to network %s: %s ", blockchainName, network.Name(), err)
			ux.Logger.PrintToUser("")
			continue
		}
		if network.Kind() != models.Local && onlyLocalnetInfo {
			continue
		}
		genesisBytes, err := contract.GetBlockchainGenesis(
			app.GetSDKApp(),
			network,
			contract.ChainSpec{
				BlockchainName: sc.Name,
			},
		)
		if err != nil {
			if network.Kind() != models.Local {
				return err
			}
			// ignore local network errors for cases
			// where local network is down but sidecar contains
			// local network metadata
			// (eg host restarts)
			continue
		} else if network.Kind() == models.Local {
			locallyDeployed = true
		}
		if utils.ByteSliceIsSubnetEvmGenesis(genesisBytes) {
			genesis, err := utils.ByteSliceToSubnetEvmGenesis(genesisBytes)
			if err != nil {
				return err
			}
			t.Append([]string{net, "ChainID", genesis.Config.ChainID.String()})
			if network.Kind() == models.Local {
				localChainID = genesis.Config.ChainID.String()
			}
		}
		if data.SubnetID != ids.Empty {
			t.Append([]string{net, "SubnetID", data.SubnetID.String()})
			_, owners, threshold, err := txutils.GetOwners(network, data.SubnetID)
			if err != nil {
				return err
			}
			t.Append([]string{net, fmt.Sprintf("Owners (Threhold=%d)", threshold), strings.Join(owners, "\n")})
		}
		if data.BlockchainID != ids.Empty {
			hexEncoding := "0x" + hex.EncodeToString(data.BlockchainID[:])
			t.Append([]string{net, "BlockchainID (CB58)", data.BlockchainID.String()})
			t.Append([]string{net, "BlockchainID (HEX)", hexEncoding})
		}
		endpoint, _, err := contract.GetBlockchainEndpoints(
			app.GetSDKApp(),
			network,
			contract.ChainSpec{
				BlockchainName: sc.Name,
			},
			false,
			false,
		)
		if err != nil {
			return err
		}
		if network.Kind() == models.Local {
			localEndpoint = endpoint
		}
		t.Append([]string{net, "RPC Endpoint", endpoint})
		if data.ValidatorManagerAddress != "" {
			t.Append([]string{net, "Manager", data.ValidatorManagerAddress})
		}
	}
	t.Render()

	// Warp
	t = ux.DefaultTable("Warp", nil)
	// SetColumnConfigs not available in tablewriter
	hasWarpInfo := false
	for net, data := range sc.Networks {
		network, err := app.GetNetworkFromSidecarNetworkName(net)
		if err != nil {
			continue
		}
		if network.Kind() == models.Local && !locallyDeployed {
			continue
		}
		if network.Kind() != models.Local && onlyLocalnetInfo {
			continue
		}
		if data.TeleporterMessengerAddress != "" {
			t.Append([]string{net, "Warp Messenger Address", data.TeleporterMessengerAddress})
			hasWarpInfo = true
		}
		if data.TeleporterRegistryAddress != "" {
			t.Append([]string{net, "Warp Registry Address", data.TeleporterRegistryAddress})
			hasWarpInfo = true
		}
	}
	if hasWarpInfo {
		ux.Logger.PrintToUser("")
		t.Render()
	}

	// Token
	ux.Logger.PrintToUser("")
	t = ux.DefaultTable("Token", nil)
	t.Append([]string{"Token Name", sc.TokenName})
	t.Append([]string{"Token Symbol", sc.TokenSymbol})
	t.Render()

	if utils.ByteSliceIsSubnetEvmGenesis(genesisBytes) {
		genesis, err := utils.ByteSliceToSubnetEvmGenesis(genesisBytes)
		if err != nil {
			return err
		}
		if err := printAllocations(sc, genesis); err != nil {
			return err
		}
		printSmartContracts(sc, genesis)
		printPrecompiles(genesis)
	}

	if locallyDeployed {
		ux.Logger.PrintToUser("")
		if err := localnet.PrintEndpoints(app, ux.Logger.PrintToUser, sc.Name); err != nil {
			return err
		}

		codespaceEndpoint, err := utils.GetCodespaceURL(localEndpoint)
		if err != nil {
			return err
		}
		if codespaceEndpoint != "" {
			_, port, _, err := utils.GetURIHostPortAndPath(localEndpoint)
			if err != nil {
				return err
			}
			localEndpoint = codespaceEndpoint + "\n" + luxlog.Orange.Wrap(
				fmt.Sprintf("Please make sure to set visibility of port %d to public", port),
			)
		}

		// Wallet
		t = ux.DefaultTable("Wallet Connection", nil)
		t.Append([]string{"Network RPC URL", localEndpoint})
		t.Append([]string{"Network Name", sc.Name})
		t.Append([]string{"Chain ID", localChainID})
		t.Append([]string{"Token Symbol", sc.TokenSymbol})
		t.Append([]string{"Token Name", sc.TokenName})
		ux.Logger.PrintToUser("")
		t.Render()
	}

	return nil
}

func printAllocations(sc models.Sidecar, genesis core.Genesis) error {
	warpKeyAddress := ""
	if sc.TeleporterReady {
		// TeleporterKey is managed through the warp configuration
		// The key address is stored in the validator manager contract
		// k, err := key.LoadSoft(models.NewLocalNetwork().NetworkID(), app.GetKeyPath(sc.TeleporterKey))
		// if err != nil {
		//     return err
		// }
		// warpKeyAddress = k.C()
	}
	_, subnetAirdropAddress, _, err := subnet.GetDefaultSubnetAirdropKeyInfo(app, sc.Name)
	if err != nil {
		return err
	}
	if len(genesis.Alloc) > 0 {
		ux.Logger.PrintToUser("")
		t := ux.DefaultTable(
			"Initial Token Allocation",
			[]string{
				"Description",
				"Address and Private Key",
				fmt.Sprintf("Amount (%s)", sc.TokenSymbol),
				"Amount (wei)",
			},
		)
		for address, allocation := range genesis.Alloc {
			amount := allocation.Balance
			// we are only interested in supply distribution here
			if amount == nil || big.NewInt(0).Cmp(amount) == 0 {
				continue
			}
			formattedAmount := new(big.Int).Div(amount, big.NewInt(params.Ether))
			description := ""
			privKey := ""
			switch address.Hex() {
			case warpKeyAddress:
				description = luxlog.Orange.Wrap("Used by Warp")
			case subnetAirdropAddress:
				description = luxlog.Orange.Wrap("Main funded account")
			case vm.PrefundedEwoqAddress.Hex():
				description = luxlog.Orange.Wrap("Main funded account")
			case sc.ValidatorManagerOwner:
				description = luxlog.Orange.Wrap("Validator Manager Owner")
			case sc.ProxyContractOwner:
				description = luxlog.Orange.Wrap("Proxy Admin Owner")
			}
			var (
				found bool
				name  string
			)
			found, name, _, privKey, err = contract.SearchForManagedKey(app.GetSDKApp(), models.NewLocalNetwork(), address.Hex(), true)
			if err != nil {
				return err
			}
			if found {
				description = fmt.Sprintf("%s\n%s", description, name)
			}
			t.Append([]string{description, address.Hex() + "\n" + privKey, formattedAmount.String(), amount.String()})
		}
		t.Render()
	}
	return nil
}

func printSmartContracts(sc models.Sidecar, genesis core.Genesis) {
	if len(genesis.Alloc) == 0 {
		return
	}
	ux.Logger.PrintToUser("")
	t := ux.DefaultTable(
		"Smart Contracts",
		[]string{"Description", "Address", "Deployer"},
	)
	for address, allocation := range genesis.Alloc {
		if len(allocation.Code) == 0 {
			continue
		}
		var description, deployer string
		switch {
		case address == common.HexToAddress(warpgenesis.MessengerContractAddress):
			description = "Warp Messenger"
			deployer = warpgenesis.MessengerDeployerAddress
		case address == common.HexToAddress(validatorManagerSDK.ValidatorMessagesContractAddress):
			description = "Validator Messages Lib"
		case address == common.HexToAddress(validatorManagerSDK.ValidatorContractAddress):
			if sc.ValidatorManagement == "proof-of-authority" {
				description = "PoA Validator Manager"
			} else {
				description = "Native Token Staking Manager"
			}
			if sc.UseACP99 {
				description = "ACP99 Compatible " + description
			} else {
				description = "v1.0.0 Compatible " + description
			}
		case address == common.HexToAddress(validatorManagerSDK.ValidatorProxyContractAddress):
			description = "Validator Transparent Proxy"
		case address == common.HexToAddress(validatorManagerSDK.ValidatorProxyAdminContractAddress):
			description = "Validator Proxy Admin"
			deployer = sc.ProxyContractOwner
		case address == common.HexToAddress(validatorManagerSDK.SpecializationProxyContractAddress):
			description = "Validator Specialization Transparent Proxy"
		case address == common.HexToAddress(validatorManagerSDK.SpecializationProxyAdminContractAddress):
			description = "Validator Specialization Proxy Admin"
		case address == common.HexToAddress(validatorManagerSDK.RewardCalculatorAddress):
			description = "Reward Calculator"
		}
		t.Append([]string{description, address.Hex(), deployer})
	}
	t.Render()
}

func printPrecompiles(genesis core.Genesis) {
	ux.Logger.PrintToUser("")
	t := ux.DefaultTable(
		"Initial Precompile Configs",
		[]string{"Precompile", "Admin Addresses", "Manager Addresses", "Enabled Addresses"},
	)
	// SetColumnConfigs and Style not available in tablewriter
	// SetColumnConfigs not available in tablewriter

	warpSet := false
	allowListSet := false
	
	// GenesisPrecompiles are now handled through the EVM config extensions
	// The precompile configuration is stored in the upgraded chain config structure
	/*
	// Warp
	if genesis.Config.GenesisPrecompiles[warp.ConfigKey] != nil {
		t.Append([]string{"Warp", "n/a", "n/a", "n/a"})
		warpSet = true
	}
	// Native Minting
	if genesis.Config.GenesisPrecompiles[nativeminter.ConfigKey] != nil {
		cfg := genesis.Config.GenesisPrecompiles[nativeminter.ConfigKey].(*nativeminter.Config)
		addPrecompileAllowListToTable(t, "Native Minter", cfg.AdminAddresses, cfg.ManagerAddresses, cfg.EnabledAddresses)
		allowListSet = true
	}
	// Contract allow list
	if genesis.Config.GenesisPrecompiles[deployerallowlist.ConfigKey] != nil {
		cfg := genesis.Config.GenesisPrecompiles[deployerallowlist.ConfigKey].(*deployerallowlist.Config)
		addPrecompileAllowListToTable(t, "Contract Allow List", cfg.AdminAddresses, cfg.ManagerAddresses, cfg.EnabledAddresses)
		allowListSet = true
	}
	// TX allow list
	if genesis.Config.GenesisPrecompiles[txallowlist.ConfigKey] != nil {
		cfg := genesis.Config.GenesisPrecompiles[txallowlist.Module.ConfigKey].(*txallowlist.Config)
		addPrecompileAllowListToTable(t, "Tx Allow List", cfg.AdminAddresses, cfg.ManagerAddresses, cfg.EnabledAddresses)
		allowListSet = true
	}
	// Fee config allow list
	if genesis.Config.GenesisPrecompiles[feemanager.ConfigKey] != nil {
		cfg := genesis.Config.GenesisPrecompiles[feemanager.ConfigKey].(*feemanager.Config)
		addPrecompileAllowListToTable(t, "Fee Config Allow List", cfg.AdminAddresses, cfg.ManagerAddresses, cfg.EnabledAddresses)
		allowListSet = true
	}
	// Reward config allow list
	if genesis.Config.GenesisPrecompiles[rewardmanager.ConfigKey] != nil {
		cfg := genesis.Config.GenesisPrecompiles[rewardmanager.ConfigKey].(*rewardmanager.Config)
		addPrecompileAllowListToTable(t, "Reward Manager Allow List", cfg.AdminAddresses, cfg.ManagerAddresses, cfg.EnabledAddresses)
		allowListSet = true
	}
	*/
	if warpSet || allowListSet {
		t.Render()
		if allowListSet {
			note := luxlog.Orange.Wrap("The allowlist is taken from the genesis and is not being updated if you make adjustments\nvia the precompile. Use readAllowList(address) instead.")
			ux.Logger.PrintToUser(note)
		}
	}
}

// Function temporarily disabled while precompile display is being refactored
// Will be re-enabled when the new precompile configuration format is finalized
/*
func addPrecompileAllowListToTable(
	t table.Writer,
	label string,
	adminAddresses []common.Address,
	managerAddresses []common.Address,
	enabledAddresses []common.Address,
) {
	t.AppendSeparator()
	admins := len(adminAddresses)
	managers := len(managerAddresses)
	enabled := len(enabledAddresses)
	max := max(admins, managers, enabled)
	for i := 0; i < max; i++ {
		var admin, manager, enable string
		if i < len(adminAddresses) && adminAddresses[i] != (common.Address{}) {
			admin = adminAddresses[i].Hex()
		}
		if i < len(managerAddresses) && managerAddresses[i] != (common.Address{}) {
			manager = managerAddresses[i].Hex()
		}
		if i < len(enabledAddresses) && enabledAddresses[i] != (common.Address{}) {
			enable = enabledAddresses[i].Hex()
		}
		t.Append([]string{label, admin, manager, enable})
	}
}
*/

func describe(_ *cobra.Command, args []string) error {
	blockchainName := args[0]
	if !app.GenesisExists(blockchainName) {
		ux.Logger.PrintToUser("The provided blockchain name %q does not exist", blockchainName)
		return nil
	}
	if printGenesisOnly {
		return printGenesis(blockchainName)
	}
	if err := PrintSubnetInfo(blockchainName, false); err != nil {
		return err
	}
	if isEVM, _, err := app.HasSubnetEVMGenesis(blockchainName); err != nil {
		return err
	} else if !isEVM {
		sc, err := app.LoadSidecar(blockchainName)
		if err != nil {
			return err
		}
		app.Log.Warn("Unknown genesis format", zap.Any("vm-type", sc.VM))
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("Printing genesis")
		return printGenesis(blockchainName)
	}
	return nil
}
