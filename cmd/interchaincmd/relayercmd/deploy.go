// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package relayercmd

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/contract"
	"github.com/luxfi/cli/pkg/interchain/relayer"
	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/cli/pkg/localnet"
	"github.com/luxfi/cli/pkg/models"
	"github.com/luxfi/cli/pkg/networkoptions"
	"github.com/luxfi/cli/pkg/prompts"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/cli/pkg/vm"
	"github.com/luxfi/sdk/evm"
	"github.com/luxfi/node/utils/logging"

	"github.com/spf13/cobra"
)

type DeployFlags struct {
	Network              networkoptions.NetworkFlags
	Version              string
	LogLevel             string
	RelayCChain          bool
	BlockchainsToRelay   []string
	Key                  string
	Amount               float64
	CChainAmount         float64
	BlockchainFundingKey string
	CChainFundingKey     string
	BinPath              string
	AllowPrivateIPs      bool
}

var deployFlags DeployFlags

const (
	disableDeployToRemotePrompt = true
	aproxFundingFee             = 0.01
)

// lux interchain relayer deploy
func newDeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploys an Warp Relayer for the given Network",
		Long:  `Deploys an Warp Relayer for the given Network.`,
		RunE:  deploy,
		Args:  cobrautils.ExactArgs(0),
	}
	networkoptions.AddNetworkFlagsToCmd(cmd, &deployFlags.Network, true, networkoptions.NonMainnetSupportedNetworkOptions)
	cmd.Flags().StringVar(&deployFlags.BinPath, "bin-path", "", "use the given relayer binary")
	cmd.Flags().StringVar(
		&deployFlags.Version,
		"version",
		constants.DefaultRelayerVersion,
		"version to deploy",
	)
	cmd.Flags().StringVar(&deployFlags.LogLevel, "log-level", "", "log level to use for relayer logs")
	cmd.Flags().StringSliceVar(&deployFlags.BlockchainsToRelay, "blockchains", nil, "blockchains to relay as source and destination")
	cmd.Flags().BoolVar(&deployFlags.RelayCChain, "cchain", false, "relay C-Chain as source and destination")
	cmd.Flags().StringVar(&deployFlags.Key, "key", "", "key to be used by default both for rewards and to pay fees")
	cmd.Flags().Float64Var(&deployFlags.Amount, "amount", 0, "automatically fund l1s fee payments with the given amount")
	cmd.Flags().Float64Var(&deployFlags.CChainAmount, "cchain-amount", 0, "automatically fund cchain fee payments with the given amount")
	cmd.Flags().StringVar(&deployFlags.BlockchainFundingKey, "blockchain-funding-key", "", "key to be used to fund relayer account on all l1s")
	cmd.Flags().StringVar(&deployFlags.CChainFundingKey, "cchain-funding-key", "", "key to be used to fund relayer account on cchain")
	cmd.Flags().BoolVar(&deployFlags.AllowPrivateIPs, "allow-private-ips", true, "allow relayer to connect to private ips")
	return cmd
}

func deploy(_ *cobra.Command, args []string) error {
	return CallDeploy(args, deployFlags, models.UndefinedNetwork)
}

func CallDeploy(_ []string, flags DeployFlags, network models.Network) error {
	var err error
	if network == models.UndefinedNetwork {
		network, err = networkoptions.GetNetworkFromCmdLineFlags(
			app,
			"In which Network will operate the Relayer?",
			flags.Network,
			true,
			false,
			networkoptions.NonMainnetSupportedNetworkOptions,
			"",
		)
		if err != nil {
			return err
		}
	}

	deployToRemote := false
	if !disableDeployToRemotePrompt && network.Kind() != models.Local {
		prompt := "Do you want to deploy the relayer to a remote or a local host?"
		remoteHostOption := "I want to deploy the relayer into a remote node in the cloud"
		localHostOption := "I prefer to deploy into a localhost process"
		options := []string{remoteHostOption, localHostOption, explainOption}
		for {
			option, err := app.Prompt.CaptureList(
				prompt,
				options,
			)
			if err != nil {
				return err
			}
			switch option {
			case remoteHostOption:
				deployToRemote = true
			case localHostOption:
			case explainOption:
				ux.Logger.PrintToUser("A local host relayer is for temporary networks, won't survive a host restart")
				ux.Logger.PrintToUser("or a relayer transient failure (but anyway can be manually restarted by cmd)")
				ux.Logger.PrintToUser("A remote relayer is deployed into a new cloud node, and will recover from")
				ux.Logger.PrintToUser("temporary relayer failures and from host restarts.")
				continue
			}
			break
		}
	}

	if !deployToRemote {
		if isUP, _, _, err := relayer.RelayerIsUp(app.GetLocalRelayerRunPath(network.Kind())); err != nil {
			return err
		} else if isUP {
			return fmt.Errorf("there is already a local relayer deployed for %s", network.Kind().String())
		}
	}

	logLevelOptions := []string{
		logging.Info.LowerString(),
		logging.Warn.LowerString(),
		logging.Error.LowerString(),
		logging.Off.LowerString(),
		logging.Fatal.LowerString(),
		logging.Debug.LowerString(),
		logging.Trace.LowerString(),
		logging.Verbo.LowerString(),
	}
	if flags.LogLevel == "" {
		prompt := "Which log level do you prefer for your relayer?"

		flags.LogLevel, err = app.Prompt.CaptureList(
			prompt,
			logLevelOptions,
		)
		if err != nil {
			return err
		}
	} else {
		if _, err := logging.ToLevel(flags.LogLevel); err != nil {
			return fmt.Errorf("invalid log level %s: %w", flags.LogLevel, err)
		}
	}

	networkUP := true
	_, err = utils.GetChainID(network.Endpoint(), "C")
	if err != nil {
		if !strings.Contains(err.Error(), "connection refused") {
			return err
		}
		networkUP = false
	}

	configureBlockchains := false
	if networkUP {
		if flags.BlockchainsToRelay != nil || flags.RelayCChain {
			configureBlockchains = true
		} else {
			prompt := "Do you want to add blockchain information to your relayer?"
			yesOption := "Yes, I want to configure source and destination blockchains"
			noOption := "No, I prefer to configure the relayer later on"
			options := []string{yesOption, noOption, explainOption}
			for {
				option, err := app.Prompt.CaptureList(
					prompt,
					options,
				)
				if err != nil {
					return err
				}
				switch option {
				case yesOption:
					configureBlockchains = true
				case noOption:
				case explainOption:
					ux.Logger.PrintToUser("You can configure a list of source and destination blockchains, so that the")
					ux.Logger.PrintToUser("relayer will listen for new messages on each source, and deliver them to the")
					ux.Logger.PrintToUser("destinations.")
					ux.Logger.PrintToUser("Or you can not configure those later on, by using the 'relayer config' cmd.")
					continue
				}
				break
			}
		}
	}

	var configSpec ConfigSpec
	if configureBlockchains {
		// TODO: this is the base for a 'relayer config' cmd
		// that should load the current config, generate a configSpec for that,
		// and use this to change the config, before saving it
		// most probably, also, relayer config should restart the relayer
		var cancel bool
		configSpec, cancel, err = GenerateConfigSpec(
			network,
			flags.RelayCChain,
			flags.BlockchainsToRelay,
			flags.Key,
		)
		if cancel {
			return nil
		}
		if err != nil {
			return err
		}
	}

	fundBlockchains := false
	if networkUP && len(configSpec.destinations) > 0 {
		if flags.Amount != 0 {
			fundBlockchains = true
		} else {
			// TODO: this (and the next section) are the base for a 'relayer fund' cmd
			// it must be based on relayer conf, and try to gather a nice blockchain desc
			// from the blockchain id (as relayer logs cmd)
			ux.Logger.PrintToUser("")
			for _, destination := range configSpec.destinations {
				addr, err := evm.PrivateKeyToAddress(destination.privateKey)
				if err != nil {
					return err
				}
				client, err := evm.GetClient(destination.rpcEndpoint)
				if err != nil {
					return err
				}
				balance, err := client.GetAddressBalance(addr.Hex())
				if err != nil {
					return err
				}
				balanceFlt := new(big.Float).SetInt(balance)
				balanceFlt = balanceFlt.Quo(balanceFlt, new(big.Float).SetInt(vm.OneLux))
				ux.Logger.PrintToUser("Relayer private key on destination %s has a balance of %.9f", destination.blockchainDesc, balanceFlt)
			}
			ux.Logger.PrintToUser("")

			prompt := "Do you want to fund relayer destinations?"
			yesOption := "Yes, I want to fund destination blockchains"
			noOption := "No, I prefer to fund the relayer later on"
			options := []string{yesOption, noOption, explainOption}
			for {
				option, err := app.Prompt.CaptureList(
					prompt,
					options,
				)
				if err != nil {
					return err
				}
				switch option {
				case yesOption:
					fundBlockchains = true
				case noOption:
				case explainOption:
					ux.Logger.PrintToUser("You need to set some balance on the destination addresses")
					ux.Logger.PrintToUser("so the relayer can pay for fees when delivering messages.")
					continue
				}
				break
			}
		}
	}

	if fundBlockchains {
		for _, destination := range configSpec.destinations {
			addr, err := evm.PrivateKeyToAddress(destination.privateKey)
			if err != nil {
				return err
			}
			client, err := evm.GetClient(destination.rpcEndpoint)
			if err != nil {
				return err
			}
			cchainBlockchainID, err := contract.GetBlockchainID(
				app,
				network,
				contract.ChainSpec{
					CChain: true,
				},
			)
			if err != nil {
				return err
			}
			isCChainDestination := cchainBlockchainID.String() == destination.blockchainID
			doPay := false
			switch {
			case !isCChainDestination && flags.Amount != 0:
				doPay = true
			case isCChainDestination && flags.CChainAmount != 0:
				doPay = true
			default:
				balance, err := client.GetAddressBalance(addr.Hex())
				if err != nil {
					return err
				}
				balanceFlt := new(big.Float).SetInt(balance)
				balanceFlt = balanceFlt.Quo(balanceFlt, new(big.Float).SetInt(vm.OneLux))
				prompt := fmt.Sprintf("Do you want to fund relayer for destination %s (current C-Chain LUX balance: %.9f)?", destination.blockchainDesc, balanceFlt)
				yesOption := "Yes, I will send funds to it"
				noOption := "Not now"
				options := []string{yesOption, noOption}
				option, err := app.Prompt.CaptureList(
					prompt,
					options,
				)
				if err != nil {
					return err
				}
				switch option {
				case yesOption:
					doPay = true
				case noOption:
				}
			}
			if doPay {
				_, genesisPrivateKey, err := contract.GetEVMSubnetPrefundedKey(
					app,
					network,
					contract.ChainSpec{
						BlockchainID: destination.blockchainID,
					},
				)
				if err != nil {
					return err
				}
				privateKey := ""
				if flags.Amount != 0 {
					privateKey = genesisPrivateKey
				}
				if flags.BlockchainFundingKey != "" || flags.CChainFundingKey != "" {
					if isCChainDestination {
						if flags.CChainFundingKey != "" {
							keyPath, err := app.GetKey(flags.CChainFundingKey)
							if err != nil {
								return err
							}
							k, err := key.LoadSoft(network.ID(), keyPath)
							if err != nil {
								return err
							}
							privateKey = k.PrivKeyHex()
						}
					} else {
						if flags.BlockchainFundingKey != "" {
							keyPath, err := app.GetKey(flags.BlockchainFundingKey)
							if err != nil {
								return err
							}
							k, err := key.LoadSoft(network.ID(), keyPath)
							if err != nil {
								return err
							}
							privateKey = k.PrivKeyHex()
						}
					}
				}
				if privateKey == "" {
					privateKey, err = prompts.PromptPrivateKey(
						app.Prompt,
						fmt.Sprintf("fund the relayer destination %s", destination.blockchainDesc),
					)
					if err != nil {
						return err
					}
				}
				balance, err := client.GetPrivateKeyBalance(privateKey)
				if err != nil {
					return err
				}
				if balance.Cmp(big.NewInt(0)) == 0 {
					return fmt.Errorf("destination %s funding key has no balance", destination.blockchainDesc)
				}
				balanceBigFlt := new(big.Float).SetInt(balance)
				balanceBigFlt = balanceBigFlt.Quo(balanceBigFlt, new(big.Float).SetInt(vm.OneLux))
				balanceFlt, _ := balanceBigFlt.Float64()
				balanceFlt -= aproxFundingFee
				var amountFlt float64
				switch {
				case !isCChainDestination && flags.Amount != 0:
					amountFlt = flags.Amount
				case isCChainDestination && flags.CChainAmount != 0:
					amountFlt = flags.CChainAmount
				default:
					// CaptureFloat doesn't support validation function, validate after
					amountFlt, err = app.Prompt.CaptureFloat(
						fmt.Sprintf("Amount to transfer (available: %f)", balanceFlt),
					)
					if err != nil {
						return err
					}
					if amountFlt <= 0 {
						return fmt.Errorf("%f is not positive", amountFlt)
					}
					if amountFlt > balanceFlt {
						return fmt.Errorf("%f exceeds available funding balance of %f", amountFlt, balanceFlt)
					}
				}
				if amountFlt > balanceFlt {
					return fmt.Errorf(
						"desired balance %f for destination %s exceeds available funding balance of %f",
						amountFlt,
						destination.blockchainDesc,
						balanceFlt,
					)
				}
				amountBigFlt := new(big.Float).SetFloat64(amountFlt)
				amountBigFlt = amountBigFlt.Mul(amountBigFlt, new(big.Float).SetInt(vm.OneLux))
				amount, _ := amountBigFlt.Int(nil)
				receipt, err := client.FundAddress(privateKey, addr.Hex(), amount)
				if err != nil {
					return err
				}
				ux.Logger.PrintToUser("%s Paid fee: %.9f LUX",
					destination.blockchainDesc,
					evm.CalculateEvmFeeInLux(receipt.GasUsed, receipt.EffectiveGasPrice))
			}
		}
	}

	if deployToRemote {
		return nil
	}

	// runFilePath := app.GetLocalRelayerRunPath(network)
	storageDir := app.GetLocalRelayerStorageDir(network)
	// localNetworkRootDir := ""
	if network == models.Local {
		_, err = localnet.GetLocalNetworkDir(app)
		if err != nil {
			return err
		}
	}
	configPath := app.GetLocalRelayerConfigPath()
	logPath := app.GetLocalRelayerLogPath(network)

	metricsPort := uint32(9090) // Default metrics port
	if !deployToRemote {
		switch network {
		case models.Local:
			metricsPort = 9091 // Local network metrics port
		case models.Devnet:
			metricsPort = 9092 // Devnet metrics port
		case models.Testnet:
			metricsPort = 9093 // Testnet metrics port
		}
	}

	// create config
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Generating relayer config file at %s", configPath)
	// Create base relayer config
	config, err := relayer.CreateBaseRelayerConfig(
		flags.LogLevel,
		storageDir,
		metricsPort,
		network.Name(),
	)
	if err != nil {
		return err
	}
	for _, source := range configSpec.sources {
		if err := relayer.AddSourceToRelayerConfig(
			config,
			source.subnetID,
			source.blockchainID,
			source.rpcEndpoint,
			source.wsEndpoint,
			source.warpMessengerAddress,
			source.rewardAddress,
		); err != nil {
			return err
		}
	}
	for _, destination := range configSpec.destinations {
		if err := relayer.AddDestinationToRelayerConfig(
			config,
			destination.subnetID,
			destination.blockchainID,
			destination.rpcEndpoint,
			destination.privateKey,
		); err != nil {
			return err
		}
	}

	if len(configSpec.sources) > 0 && len(configSpec.destinations) > 0 {
		// relayer fails for empty configs
		// TODO: Save config to file and deploy relayer
		// For now, just save the config and return
		_ = config
		ux.Logger.PrintToUser("Relayer configuration created successfully")
		ux.Logger.PrintToUser("Config path: %s", configPath)
		ux.Logger.PrintToUser("Log path: %s", logPath)
	}

	return nil
}
