// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package blockchaincmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/models"
	"github.com/luxfi/cli/pkg/networkoptions"
	"github.com/luxfi/cli/pkg/plugins"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/node/ids"
	"github.com/luxfi/node/utils/logging"
	"github.com/luxfi/node/vms/platformvm"
	"github.com/spf13/cobra"
)

var (
	// path to luxd config file
	luxdConfigPath string
	// path to luxd plugin dir
	pluginDir string
	// path to luxd datadir dir
	dataDir string
	// if true, print the manual instructions to screen
	printManual bool
	// if true, doesn't ask for overwriting the config file
	forceWrite bool
	// for permissionless subnet only: how much native token will be staked in the validator
	stakeAmount uint64
)

// lux blockchain join
func newJoinCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "join [blockchainName]",
		Short: "Configure your validator node to begin validating a new blockchain",
		Long: `The blockchain join command configures your validator node to begin validating a new Blockchain.

To complete this process, you must have access to the machine running your validator. If the
CLI is running on the same machine as your validator, it can generate or update your node's
config file automatically. Alternatively, the command can print the necessary instructions
to update your node manually. To complete the validation process, the Blockchain's admins must add
the NodeID of your validator to the Blockchain's allow list by calling addValidator with your
NodeID.

After you update your validator's config, you need to restart your validator manually. If
you provide the --luxd-config flag, this command attempts to edit the config file
at that path.

This command currently only supports Blockchains deployed on the Testnet and Mainnet.`,
		RunE: joinCmd,
		Args: cobrautils.ExactArgs(1),
	}
	networkoptions.AddNetworkFlagsToCmd(cmd, &globalNetworkFlags, false, networkoptions.DefaultSupportedNetworkOptions)
	cmd.Flags().StringVar(&luxdConfigPath, "luxd-config", "", "file path of the luxd config file")
	cmd.Flags().StringVar(&pluginDir, "plugin-dir", "", "file path of luxd's plugin directory")
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "path of luxd's data dir directory")
	cmd.Flags().BoolVar(&printManual, "print", false, "if true, print the manual config without prompting")
	cmd.Flags().StringVar(&nodeIDStr, "node-id", "", "set the NodeID of the validator to check")
	cmd.Flags().BoolVar(&forceWrite, "force-write", false, "if true, skip to prompt to overwrite the config file")
	cmd.Flags().Uint64Var(&stakeAmount, "stake-amount", 0, "amount of tokens to stake on validator")
	cmd.Flags().StringVar(&startTimeStr, "start-time", "", "start time that validator starts validating")
	cmd.Flags().DurationVar(&duration, "staking-period", 0, "how long validator validates for after start time")
	cmd.Flags().StringVarP(&keyName, "key", "k", "", "select the key to use [testnet only]")
	cmd.Flags().BoolVarP(&useLedger, "ledger", "g", false, "use ledger instead of key (always true on mainnet, defaults to false on testnet)")
	cmd.Flags().StringSliceVar(&ledgerAddresses, "ledger-addrs", []string{}, "use the given ledger addresses")
	return cmd
}

func joinCmd(_ *cobra.Command, args []string) error {
	if printManual && (luxdConfigPath != "" || pluginDir != "") {
		return errors.New("--print cannot be used with --luxd-config or --plugin-dir")
	}

	chains, err := ValidateSubnetNameAndGetChains(args)
	if err != nil {
		return err
	}

	blockchainName := chains[0]

	sc, err := app.LoadSidecar(blockchainName)
	if err != nil {
		return err
	}

	if sc.Sovereign {
		return errors.New("lux blockchain join command cannot be used on sovereign blockchains")
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

	network.HandlePublicNetworkSimulation()

	subnetID := sc.Networks[network.Name()].SubnetID
	if subnetID == ids.Empty {
		return errNoSubnetID
	}
	subnetIDStr := subnetID.String()

	if printManual {
		pluginDir = app.GetTmpPluginDir()
		vmPath, err := plugins.CreatePlugin(app, sc.Name, pluginDir)
		if err != nil {
			return err
		}
		printJoinCmd(subnetIDStr, network, vmPath)
		return nil
	}

	// if **both** flags were set, nothing special needs to be done
	// just check the following blocks
	if luxdConfigPath == "" && pluginDir == "" {
		// both flags are NOT set
		const (
			choiceManual    = "Manual"
			choiceAutomatic = "Automatic"
		)
		choice, err := app.Prompt.CaptureList(
			"How would you like to update the luxd config?",
			[]string{choiceAutomatic, choiceManual},
		)
		if err != nil {
			return err
		}
		if choice == choiceManual {
			pluginDir = app.GetTmpPluginDir()
			vmPath, err := plugins.CreatePlugin(app, sc.Name, pluginDir)
			if err != nil {
				return err
			}
			printJoinCmd(subnetIDStr, network, vmPath)
			return nil
		}
	}

	// if choice is automatic, we just pass through this block
	// or, pluginDir was set but not luxdConfigPath
	// if **both** flags were set, this will be skipped...
	if luxdConfigPath == "" {
		luxdConfigPath, err = plugins.FindLuxdConfigPath()
		if err != nil {
			return err
		}
		if luxdConfigPath != "" {
			ux.Logger.PrintToUser(logging.Bold.Wrap(logging.Green.Wrap("Found a config file at %s")), luxdConfigPath)
			yes, err := app.Prompt.CaptureYesNo("Is this the file we should update?")
			if err != nil {
				return err
			}
			if yes {
				ux.Logger.PrintToUser("Will use file at path %s to update the configuration", luxdConfigPath)
			} else {
				luxdConfigPath = ""
			}
		}
		if luxdConfigPath == "" {
			luxdConfigPath, err = app.Prompt.CaptureString("Path to your existing config file (or where it will be generated)")
			if err != nil {
				return err
			}
		}
	}

	// ...but not this
	luxdConfigPath, err := plugins.SanitizePath(luxdConfigPath)
	if err != nil {
		return err
	}

	// luxdConfigPath was set but not pluginDir
	// if **both** flags were set, this will be skipped...
	if pluginDir == "" {
		pluginDir, err = plugins.FindPluginDir()
		if err != nil {
			return err
		}
		if pluginDir != "" {
			ux.Logger.PrintToUser(logging.Bold.Wrap(logging.Green.Wrap("Found the VM plugin directory at %s")), pluginDir)
			yes, err := app.Prompt.CaptureYesNo("Is this where we should install the VM?")
			if err != nil {
				return err
			}
			if yes {
				ux.Logger.PrintToUser("Will use plugin directory at %s to install the VM", pluginDir)
			} else {
				pluginDir = ""
			}
		}
		if pluginDir == "" {
			pluginDir, err = app.Prompt.CaptureString("Path to your luxd plugin dir (likely .luxd/plugins)")
			if err != nil {
				return err
			}
		}
	}

	// ...but not this
	pluginDir, err := plugins.SanitizePath(pluginDir)
	if err != nil {
		return err
	}

	vmPath, err := plugins.CreatePlugin(app, sc.Name, pluginDir)
	if err != nil {
		return err
	}

	ux.Logger.PrintToUser("VM binary written to %s", vmPath)

	if forceWrite {
		if err := writeLuxdChainConfigFiles(app, dataDir, blockchainName, sc, network); err != nil {
			return err
		}
	}

	subnetLuxdConfigFile := ""
	if app.LuxdNodeConfigExists(blockchainName) {
		subnetLuxdConfigFile = app.GetLuxdNodeConfigPath(blockchainName)
	}

	if err := plugins.EditConfigFile(
		app,
		subnetIDStr,
		network,
		luxdConfigPath,
		forceWrite,
		subnetLuxdConfigFile,
	); err != nil {
		return err
	}

	return nil
}

func writeLuxdChainConfigFiles(
	app *application.Lux,
	dataDir string,
	blockchainName string,
	sc models.Sidecar,
	network models.Network,
) error {
	if dataDir == "" {
		dataDir = utils.UserHomePath(".luxd")
	}

	subnetID := sc.Networks[network.Name()].SubnetID
	if subnetID == ids.Empty {
		return errNoSubnetID
	}
	subnetIDStr := subnetID.String()
	blockchainID := sc.Networks[network.Name()].BlockchainID

	configsPath := filepath.Join(dataDir, "configs")

	subnetConfigsPath := filepath.Join(configsPath, "subnets")
	subnetConfigPath := filepath.Join(subnetConfigsPath, subnetIDStr+".json")
	if app.LuxdSubnetConfigExists(blockchainName) {
		if err := os.MkdirAll(subnetConfigsPath, constants.DefaultPerms755); err != nil {
			return err
		}
		subnetConfig, err := app.LoadRawLuxdSubnetConfig(blockchainName)
		if err != nil {
			return err
		}
		if err := os.WriteFile(subnetConfigPath, subnetConfig, constants.DefaultPerms755); err != nil {
			return err
		}
	} else {
		_ = os.RemoveAll(subnetConfigPath)
	}

	if blockchainID != ids.Empty && app.ChainConfigExists(blockchainName) || app.NetworkUpgradeExists(blockchainName) {
		chainConfigsPath := filepath.Join(configsPath, "chains", blockchainID.String())
		if err := os.MkdirAll(chainConfigsPath, constants.DefaultPerms755); err != nil {
			return err
		}
		chainConfigPath := filepath.Join(chainConfigsPath, "config.json")
		if app.ChainConfigExists(blockchainName) {
			chainConfig, err := app.LoadRawChainConfig(blockchainName)
			if err != nil {
				return err
			}
			if err := os.WriteFile(chainConfigPath, chainConfig, constants.DefaultPerms755); err != nil {
				return err
			}
		} else {
			_ = os.RemoveAll(chainConfigPath)
		}
		networkUpgradesPath := filepath.Join(chainConfigsPath, "upgrade.json")
		if app.NetworkUpgradeExists(blockchainName) {
			networkUpgrades, err := app.LoadRawNetworkUpgrades(blockchainName)
			if err != nil {
				return err
			}
			if err := os.WriteFile(networkUpgradesPath, networkUpgrades, constants.DefaultPerms755); err != nil {
				return err
			}
		} else {
			_ = os.RemoveAll(networkUpgradesPath)
		}
	}

	return nil
}

func checkIsValidating(subnetID ids.ID, nodeID ids.NodeID, pClient platformvm.Client) (bool, error) {
	// first check if the node is already an accepted validator on the subnet
	ctx := context.Background()
	nodeIDs := []ids.NodeID{nodeID}
	vals, err := pClient.GetCurrentValidators(ctx, subnetID, nodeIDs)
	if err != nil {
		return false, err
	}
	for _, v := range vals {
		// strictly this is not needed, as we are providing the nodeID as param
		// just a double check
		if v.NodeID == nodeID {
			return true, nil
		}
	}
	return false, nil
}

func printJoinCmd(subnetID string, network models.Network, vmPath string) {
	msg := `
To setup your node, you must do two things:

1. Add your VM binary to your node's plugin directory
2. Update your node config to start validating the subnet

To add the VM to your plugin directory, copy or scp from %s

If you installed luxd with the install script, your plugin directory is likely
~/.luxd/plugins.

If you start your node from the command line WITHOUT a config file (e.g. via command
line or systemd script), add the following flag to your node's startup command:

--track-subnets=%s
(if the node already has a track-subnets config, append the new value by
comma-separating it).

For example:
./build/luxd --network-id=%s --track-subnets=%s

If you start the node via a JSON config file, add this to your config file:
track-subnets: %s

NOTE: The flag --track-subnets is a replacement of the deprecated --whitelisted-subnets.
If the later is present in config, please rename it to track-subnets first.

TIP: Try this command with the --luxd-config flag pointing to your config file,
this tool will try to update the file automatically (make sure it can write to it).

After you update your config, you will need to restart your node for the changes to
take effect.`

	ux.Logger.PrintToUser(msg, vmPath, subnetID, network.NetworkIDFlagValue(), subnetID, subnetID)
}
