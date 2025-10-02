// / Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package blockchaincmd

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/cmd/flags"
	"github.com/luxfi/cli/cmd/interchaincmd/messengercmd"
	"github.com/luxfi/cli/cmd/interchaincmd/relayercmd"
	"github.com/luxfi/cli/cmd/networkcmd"
	"github.com/luxfi/cli/pkg/blockchain"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/dependencies"
	"github.com/luxfi/cli/pkg/interchain/relayer"
	"github.com/luxfi/cli/pkg/keychain"
	"github.com/luxfi/cli/pkg/localnet"
	"github.com/luxfi/cli/pkg/metrics"
	"github.com/luxfi/cli/pkg/networkoptions"
	"github.com/luxfi/cli/pkg/subnet"
	"github.com/luxfi/cli/pkg/txutils"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/cli/pkg/vm"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/ids"
	luxlog "github.com/luxfi/log"
	"github.com/luxfi/node/api/info"
	"github.com/luxfi/node/utils/formatting/address"
	"github.com/luxfi/node/utils/set"
	"github.com/luxfi/node/utils/units"
	"github.com/luxfi/node/vms/platformvm/fx"
	"github.com/luxfi/node/vms/platformvm/signer"
	"github.com/luxfi/node/vms/platformvm/txs"
	"github.com/luxfi/node/vms/platformvm/warp/message"
	"github.com/luxfi/node/vms/types"
	"github.com/luxfi/sdk/contract"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/sdk/prompts"
	"github.com/luxfi/sdk/prompts/comparator"
	sdkutils "github.com/luxfi/sdk/utils"
	validatormanagerSDK "github.com/luxfi/sdk/validatormanager"
	"github.com/luxfi/sdk/validatormanager/validatormanagertypes"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const skipRelayerFlagName = "skip-relayer"

var (
	sameControlKey         bool
	keyName                string
	threshold              uint32
	controlKeys            []string
	subnetAuthKeys         []string
	outputTxPath           string
	useLedger              bool
	useEwoq                bool
	ledgerAddresses        []string
	subnetIDStr            string
	mainnetChainID         uint32
	skipCreatePrompt       bool
	partialSync            bool
	subnetOnly             bool
	warpSpec               subnet.WarpSpec
	numNodes               uint32
	relayerAmount          float64
	relayerKeyName         string
	relayCChain            bool
	cChainFundingKey       string
	warpKeyName            string
	cchainIcmKeyName       string
	relayerAllowPrivateIPs bool

	validatorManagerAddress        string
	deployFlags                    BlockchainDeployFlags
	errMutuallyExlusiveControlKeys = errors.New("--control-keys and --same-control-key are mutually exclusive")
	ErrMutuallyExlusiveKeyLedger   = errors.New("key source flags --key, --ledger/--ledger-addrs are mutually exclusive")
	ErrStoredKeyOnMainnet          = errors.New("key --key is not available for mainnet operations")
	errMutuallyExlusiveSubnetFlags = errors.New("--subnet-only and --subnet-id are mutually exclusive")
)

type BlockchainDeployFlags struct {
	SigAggFlags             flags.SignatureAggregatorFlags
	LocalMachineFlags       flags.LocalMachineFlags
	ProofOfStakeFlags       flags.POSFlags
	BootstrapValidatorFlags flags.BootstrapValidatorFlags
	ConvertOnly             bool
}

// lux blockchain deploy
func newDeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy [blockchainName]",
		Short: "Deploys a blockchain configuration",
		Long: `The blockchain deploy command deploys your Blockchain configuration to Local Network, to Testnet, DevNet or to Mainnet.

At the end of the call, the command prints the RPC URL you can use to interact with the L1 / Subnet.

When deploying an L1, Lux-CLI lets you use your local machine as a bootstrap validator, so you don't need to run separate Lux nodes. 
This is controlled by the --use-local-machine flag (enabled by default on Local Network).

If --use-local-machine is set to true: 
- Lux-CLI will call CreateSubnetTx, CreateChainTx, ConvertSubnetToL1Tx, followed by syncing the local machine bootstrap validator to the L1 and initialize 
Validator Manager Contract on the L1

If using your own Lux Nodes as bootstrap validators: 
- Lux-CLI will call CreateSubnetTx, CreateChainTx, ConvertSubnetToL1Tx 
- You will have to sync your bootstrap validators to the L1 
- Next, Initialize Validator Manager contract on the L1 using lux contract initValidatorManager [L1_Name]

Lux-CLI only supports deploying an individual Blockchain once per network. Subsequent
attempts to deploy the same Blockchain to the same network (Local Network, Testnet, Mainnet) aren't
allowed. If you'd like to redeploy a Blockchain locally for testing, you must first call
lux network clean to reset all deployed chain state. Subsequent local deploys
redeploy the chain with fresh state. You can deploy the same Blockchain to multiple networks,
so you can take your locally tested Blockchain and deploy it on Testnet or Mainnet.`,
		RunE:              deployBlockchain,
		PersistentPostRun: handlePostRun,
		PreRunE:           cobrautils.ExactArgs(1),
	}
	networkGroup := networkoptions.GetNetworkFlagsGroup(cmd, &globalNetworkFlags, true, networkoptions.DefaultSupportedNetworkOptions)
	sigAggGroup := flags.AddSignatureAggregatorFlagsToCmd(cmd, &deployFlags.SigAggFlags)

	cmd.Flags().StringVarP(&keyName, "key", "k", "", "select the key to use [testnet/devnet deploy only]")
	cmd.Flags().StringVar(&outputTxPath, "output-tx-path", "", "file path of the blockchain creation tx (for multi-sig signing)")
	cmd.Flags().BoolVarP(&useEwoq, "ewoq", "e", false, "use ewoq key [local/devnet deploy only]")
	cmd.Flags().BoolVarP(&useLedger, "ledger", "g", false, "use ledger instead of key")
	cmd.Flags().StringSliceVar(&ledgerAddresses, "ledger-addrs", []string{}, "use the given ledger addresses")
	cmd.Flags().StringVarP(&subnetIDStr, "subnet-id", "u", "", "do not create a subnet, deploy the blockchain into the given subnet id")
	cmd.Flags().Uint32Var(&mainnetChainID, "mainnet-chain-id", 0, "use different ChainID for mainnet deployment")
	cmd.Flags().BoolVar(&subnetOnly, "subnet-only", false, "command stops after CreateSubnetTx and returns SubnetID")
	cmd.Flags().BoolVar(&deployFlags.ConvertOnly, "convert-only", false, "avoid node track, restart and poa manager setup")

	localNetworkGroup := flags.RegisterFlagGroup(cmd, "Local Network Flags", "show-local-network-flags", true, func(set *pflag.FlagSet) {
		set.Uint32Var(&numNodes, "num-nodes", constants.LocalNetworkNumNodes, "number of nodes to be created on local network deploy")
		set.StringVar(&deployFlags.LocalMachineFlags.LuxdBinaryPath, "luxd-path", "", "use this luxd binary path")
		set.StringVar(
			&deployFlags.LocalMachineFlags.UserProvidedLuxdVersion,
			"luxd-version",
			constants.DefaultLuxdVersion,
			"use this version of luxd (ex: v1.17.12)",
		)
	})

	nonSovGroup := flags.RegisterFlagGroup(cmd, "Non Subnet-Only-Validators (Non-SOV) Flags", "show-non-sov-flags", true, func(set *pflag.FlagSet) {
		set.BoolVar(&sameControlKey, "same-control-key", false, "use the fee-paying key as control key")
		set.Uint32Var(&threshold, "threshold", 0, "required number of control key signatures to make blockchain changes")
		set.StringSliceVar(&controlKeys, "control-keys", nil, "addresses that may make blockchain changes")
		set.StringSliceVar(&subnetAuthKeys, "auth-keys", nil, "control keys that will be used to authenticate chain creation")
	})

	bootstrapValidatorGroup := flags.AddBootstrapValidatorFlagsToCmd(cmd, &deployFlags.BootstrapValidatorFlags)
	localMachineGroup := flags.AddLocalMachineFlagsToCmd(cmd, &deployFlags.LocalMachineFlags)

	warpGroup := flags.RegisterFlagGroup(cmd, "Warp Flags", "show-warp-flags", false, func(set *pflag.FlagSet) {
		set.BoolVar(&warpSpec.SkipWarpDeploy, "skip-warp-deploy", false, "Skip automatic Warp deploy")
		set.BoolVar(&warpSpec.SkipRelayerDeploy, skipRelayerFlagName, false, "skip relayer deploy")
		set.StringVar(&warpSpec.WarpVersion, "warp-version", constants.LatestReleaseVersionTag, "Warp version to deploy")
		set.StringVar(&warpSpec.RelayerVersion, "relayer-version", constants.DefaultRelayerVersion, "relayer version to deploy")
		set.StringVar(&warpSpec.RelayerBinPath, "relayer-path", "", "relayer binary to use")
		set.StringVar(&warpSpec.RelayerLogLevel, "relayer-log-level", "info", "log level to be used for relayer logs")
		set.Float64Var(&relayerAmount, "relayer-amount", 0, "automatically fund relayer fee payments with the given amount")
		set.StringVar(&relayerKeyName, "relayer-key", "", "key to be used by default both for rewards and to pay fees")
		set.StringVar(&warpKeyName, "warp-key", constants.WarpKeyName, "key to be used to pay for Warp deploys")
		set.StringVar(&cchainIcmKeyName, "cchain-warp-key", "", "key to be used to pay for Warp deploys on C-Chain")
		set.BoolVar(&relayCChain, "relay-cchain", true, "relay C-Chain as source and destination")
		set.StringVar(&cChainFundingKey, "cchain-funding-key", "", "key to be used to fund relayer account on cchain")
		set.BoolVar(&relayerAllowPrivateIPs, "relayer-allow-private-ips", true, "allow relayer to connec to private ips")
		set.StringVar(&warpSpec.MessengerContractAddressPath, "teleporter-messenger-contract-address-path", "", "path to an Warp Messenger contract address file")
		set.StringVar(&warpSpec.MessengerDeployerAddressPath, "teleporter-messenger-deployer-address-path", "", "path to an Warp Messenger deployer address file")
		set.StringVar(&warpSpec.MessengerDeployerTxPath, "teleporter-messenger-deployer-tx-path", "", "path to an Warp Messenger deployer tx file")
		set.StringVar(&warpSpec.RegistryBydecodePath, "teleporter-registry-bytecode-path", "", "path to an Warp Registry bytecode file")
	})
	posGroup := flags.AddProofOfStakeToCmd(cmd, &deployFlags.ProofOfStakeFlags)

	cmd.SetHelpFunc(flags.WithGroupedHelp([]flags.GroupedFlags{networkGroup, bootstrapValidatorGroup, localMachineGroup, localNetworkGroup, nonSovGroup, warpGroup, posGroup, sigAggGroup}))
	return cmd
}

type SubnetValidator struct {
	// Must be Ed25519 NodeID
	NodeID ids.NodeID `json:"nodeID"`
	// Weight of this validator used when sampling
	Weight uint64 `json:"weight"`
	// When this validator will stop validating the Subnet
	EndTime uint64 `json:"endTime"`
	// Initial balance for this validator
	Balance uint64 `json:"balance"`
	// [Signer] is the BLS key for this validator.
	// Note: We do not enforce that the BLS key is unique across all validators.
	//       This means that validators can share a key if they so choose.
	//       However, a NodeID + Subnet does uniquely map to a BLS key
	Signer signer.Signer `json:"signer"`
	// Leftover $LUX from the [Balance] will be issued to this
	// owner once it is removed from the validator set.
	ChangeOwner fx.Owner `json:"changeOwner"`
}

func CallDeploy(
	cmd *cobra.Command,
	subnetOnlyParam bool,
	blockchainName string,
	networkFlags networkoptions.NetworkFlags,
	keyNameParam string,
	useLedgerParam bool,
	useEwoqParam bool,
	sameControlKeyParam bool,
) error {
	subnetOnly = subnetOnlyParam
	globalNetworkFlags = networkFlags
	sameControlKey = sameControlKeyParam
	keyName = keyNameParam
	useLedger = useLedgerParam
	useEwoq = useEwoqParam
	return deployBlockchain(cmd, []string{blockchainName})
}

func getChainsInSubnet(blockchainName string) ([]string, error) {
	subnets, err := os.ReadDir(app.GetSubnetDir())
	if err != nil {
		return nil, fmt.Errorf("failed to read baseDir: %w", err)
	}

	chains := []string{}

	for _, s := range subnets {
		if !s.IsDir() {
			continue
		}
		sidecarFile := filepath.Join(app.GetSubnetDir(), s.Name(), constants.SidecarFileName)
		if _, err := os.Stat(sidecarFile); err == nil {
			// read in sidecar file
			jsonBytes, err := os.ReadFile(sidecarFile)
			if err != nil {
				return nil, fmt.Errorf("failed reading file %s: %w", sidecarFile, err)
			}

			var sc models.Sidecar
			err = json.Unmarshal(jsonBytes, &sc)
			if err != nil {
				return nil, fmt.Errorf("failed unmarshaling file %s: %w", sidecarFile, err)
			}
			if sc.Subnet == blockchainName {
				chains = append(chains, sc.Name)
			}
		}
	}
	return chains, nil
}

func checkSubnetEVMDefaultAddressNotInAlloc(network models.Network, chain string) error {
	if network != models.Local &&
		network != models.Devnet &&
		!simulatedPublicNetwork() {
		genesis, err := app.LoadEvmGenesis(chain)
		if err != nil {
			return err
		}
		allocAddressMap := genesis.Alloc
		for address := range allocAddressMap {
			if address.String() == vm.PrefundedEwoqAddress.String() {
				return fmt.Errorf("can't airdrop to default address on public networks, please edit the genesis by calling `lux blockchain create %s --force`", chain)
			}
		}
	}
	return nil
}

func runDeploy(cmd *cobra.Command, args []string) error {
	skipCreatePrompt = true
	return deployBlockchain(cmd, args)
}

func updateSubnetEVMGenesisChainID(genesisBytes []byte, newChainID uint) ([]byte, error) {
	var genesisMap map[string]interface{}
	if err := json.Unmarshal(genesisBytes, &genesisMap); err != nil {
		return nil, err
	}
	configI, ok := genesisMap["config"]
	if !ok {
		return nil, fmt.Errorf("config field not found on genesis")
	}
	config, ok := configI.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected genesis config field to be a map[string]interface, found %T", configI)
	}
	config["chainId"] = float64(newChainID)
	return json.MarshalIndent(genesisMap, "", "  ")
}

// updates sidecar with genesis mainnet id to use
// given either by cmdline flag, original genesis id, or id obtained from the user
func getSubnetEVMMainnetChainID(sc *models.Sidecar, blockchainName string) error {
	// get original chain id
	evmGenesis, err := app.LoadEvmGenesis(blockchainName)
	if err != nil {
		return err
	}
	if evmGenesis.Config == nil {
		return fmt.Errorf("invalid subnet evm genesis format: config is nil")
	}
	if evmGenesis.Config.ChainID == nil {
		return fmt.Errorf("invalid subnet evm genesis format: config chain id is nil")
	}
	originalChainID := evmGenesis.Config.ChainID.Uint64()
	// handle cmdline flag if given
	if mainnetChainID != 0 {
		sc.SubnetEVMMainnetChainID = uint32(mainnetChainID)
	}
	// prompt the user
	if sc.SubnetEVMMainnetChainID == 0 {
		useSameChainID := "Use same ChainID"
		useNewChainID := "Use new ChainID"
		listOptions := []string{useNewChainID, useSameChainID}
		newChainIDPrompt := "Using the same ChainID for both Testnet and Mainnet could lead to a replay attack. Do you want to use a different ChainID?"
		var (
			err      error
			decision string
		)
		decision, err = app.Prompt.CaptureList(newChainIDPrompt, listOptions)
		if err != nil {
			return err
		}
		if decision == useSameChainID {
			sc.SubnetEVMMainnetChainID = uint32(originalChainID)
		} else {
			ux.Logger.PrintToUser("Enter your blockchain's ChainID. It can be any positive integer != %d.", originalChainID)
			newChainID, err := app.Prompt.CapturePositiveInt(
				"ChainID",
				[]prompts.Comparator{
					{
						Label: "Zero",
						Type:  comparator.MoreThan,
						Value: 0,
					},
					{
						Label: "Original Chain ID",
						Type:  comparator.NotEq,
						Value: originalChainID,
					},
				},
			)
			if err != nil {
				return err
			}
			sc.SubnetEVMMainnetChainID = uint32(newChainID)
		}
	}
	return app.UpdateSidecar(sc)
}

func deployLocalNetworkPreCheck(cmd *cobra.Command, network models.Network, bootstrapValidatorFlags flags.BootstrapValidatorFlags) error {
	if network == models.Local {
		if cmd.Flags().Changed("use-local-machine") && !deployFlags.LocalMachineFlags.UseLocalMachine &&
			bootstrapValidatorFlags.BootstrapEndpoints == nil &&
			bootstrapValidatorFlags.BootstrapValidatorsJSONFilePath == "" &&
			!bootstrapValidatorFlags.GenerateNodeID {
			return fmt.Errorf("deploying blockchain on local network requires local machine to be used as bootstrap validator")
		}
	}

	return nil
}

// checks for flags that will conflict if user sets convert only to false or if user sets use-local-machine to true
// if any of generateNodeID, bootstrapValidatorsJSONFilePath or bootstrapEndpoints is used by user,
// convertOnly will be set to true
func validateConvertOnlyFlag(cmd *cobra.Command, bootstrapValidatorFlags flags.BootstrapValidatorFlags, convertOnly *bool, useLocalMachine bool) error {
	if bootstrapValidatorFlags.GenerateNodeID ||
		bootstrapValidatorFlags.BootstrapValidatorsJSONFilePath != "" ||
		bootstrapValidatorFlags.BootstrapEndpoints != nil {
		flagName := ""
		switch {
		case bootstrapValidatorFlags.GenerateNodeID:
			flagName = "--generate-node-id=true"
		case bootstrapValidatorFlags.BootstrapValidatorsJSONFilePath != "":
			flagName = "--bootstrap-filepath is not empty"
		case bootstrapValidatorFlags.BootstrapEndpoints != nil:
			flagName = "--bootstrap-endpoints is not empty"
		}
		if cmd.Flags().Changed("use-local-machine") && useLocalMachine {
			return fmt.Errorf("cannot use local machine as bootstrap validator if %s", flagName)
		}
		if cmd.Flags().Changed("convert-only") && !*convertOnly {
			return fmt.Errorf("cannot set --convert-only=false if %s", flagName)
		}
		*convertOnly = true
	}
	return nil
}

func prepareBootstrapValidators(
	bootstrapValidators *[]models.SubnetValidator,
	network models.Network,
	sidecar models.Sidecar,
	kc keychain.Keychain,
	blockchainName string,
	deployBalance,
	availableBalance uint64,
	localMachineFlags *flags.LocalMachineFlags,
	bootstrapValidatorFlags *flags.BootstrapValidatorFlags,
) error {
	var err error
	if bootstrapValidatorFlags.ChangeOwnerAddress == "" {
		// use provided key as change owner unless already set
		if pAddr, err := kc.PChainFormattedStrAddresses(); err == nil && len(pAddr) > 0 {
			bootstrapValidatorFlags.ChangeOwnerAddress = pAddr[0]
			ux.Logger.PrintToUser("Using [%s] to be set as a change owner for leftover LUX", bootstrapValidatorFlags.ChangeOwnerAddress)
		}
	}
	if !bootstrapValidatorFlags.GenerateNodeID && bootstrapValidatorFlags.BootstrapEndpoints == nil && bootstrapValidatorFlags.BootstrapValidatorsJSONFilePath == "" {
		if cancel, err := StartLocalMachine(
			network,
			sidecar,
			blockchainName,
			deployBalance,
			availableBalance,
			localMachineFlags,
			bootstrapValidatorFlags,
		); err != nil {
			return err
		} else if cancel {
			return nil
		}
	}
	switch {
	case len(bootstrapValidatorFlags.BootstrapEndpoints) > 0:
		for _, endpoint := range bootstrapValidatorFlags.BootstrapEndpoints {
			infoClient := info.NewClient(endpoint)
			ctx, cancel := utils.GetAPILargeContext()
			defer cancel()
			nodeID, proofOfPossession, err := infoClient.GetNodeID(ctx)
			if err != nil {
				return err
			}
			publicKey = "0x" + hex.EncodeToString(proofOfPossession.PublicKey[:])
			pop = "0x" + hex.EncodeToString(proofOfPossession.ProofOfPossession[:])

			*bootstrapValidators = append(*bootstrapValidators, models.SubnetValidator{
				NodeID:               nodeID.String(),
				Weight:               constants.BootstrapValidatorWeight,
				Balance:              deployBalance,
				BLSPublicKey:         publicKey,
				BLSProofOfPossession: pop,
				ChangeOwnerAddr:      bootstrapValidatorFlags.ChangeOwnerAddress,
			})
		}
	case clusterNameFlagValue != "":
		// for remote clusters we don't need to ask for bootstrap validators and can read it from filesystem
		*bootstrapValidators, err = getClusterBootstrapValidators(clusterNameFlagValue, network, deployBalance)
		if err != nil {
			return fmt.Errorf("error getting bootstrap validators from cluster %s: %w", clusterNameFlagValue, err)
		}

	default:
		if len(*bootstrapValidators) == 0 {
			*bootstrapValidators, err = promptBootstrapValidators(
				network,
				deployBalance,
				availableBalance,
				bootstrapValidatorFlags,
			)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// deployBlockchain is the cobra command run for deploying subnets
func deployBlockchain(cmd *cobra.Command, args []string) error {
	blockchainName := args[0]

	if err := CreateBlockchainFirst(cmd, blockchainName, skipCreatePrompt); err != nil {
		return err
	}

	chains, err := ValidateSubnetNameAndGetChains(args)
	if err != nil {
		return err
	}

	if warpSpec.MessengerContractAddressPath != "" || warpSpec.MessengerDeployerAddressPath != "" || warpSpec.MessengerDeployerTxPath != "" || warpSpec.RegistryBydecodePath != "" {
		if warpSpec.MessengerContractAddressPath == "" || warpSpec.MessengerDeployerAddressPath == "" || warpSpec.MessengerDeployerTxPath == "" || warpSpec.RegistryBydecodePath == "" {
			return fmt.Errorf("if setting any Warp asset path, you must set all Warp asset paths")
		}
	}

	var bootstrapValidators []models.SubnetValidator
	if deployFlags.BootstrapValidatorFlags.BootstrapValidatorsJSONFilePath != "" {
		bootstrapValidators, err = LoadBootstrapValidator(deployFlags.BootstrapValidatorFlags)
		if err != nil {
			return err
		}
		deployFlags.BootstrapValidatorFlags.NumBootstrapValidators = len(bootstrapValidators)
	}

	chain := chains[0]

	sidecar, err := app.LoadSidecar(chain)
	if err != nil {
		return fmt.Errorf("failed to load sidecar for later update: %w", err)
	}

	if sidecar.ImportedFromLPM {
		return errors.New("unable to deploy blockchains imported from a repo")
	}

	if outputTxPath != "" {
		if _, err := os.Stat(outputTxPath); err == nil {
			return fmt.Errorf("outputTxPath %q already exists", outputTxPath)
		}
	}

	if !sidecar.Sovereign && deployFlags.BootstrapValidatorFlags.BootstrapValidatorsJSONFilePath != "" {
		return fmt.Errorf("--bootstrap-filepath flag is only applicable to sovereign blockchains")
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
	clusterNameFlagValue = globalNetworkFlags.ClusterName

	isEVMGenesis, validationErr, err := app.HasSubnetEVMGenesis(chain)
	if err != nil {
		return err
	}
	if sidecar.VM == models.SubnetEvm && !isEVMGenesis {
		return fmt.Errorf("failed to validate SubnetEVM genesis format: %s", validationErr)
	}

	chainGenesis, err := app.LoadRawGenesis(chain)
	if err != nil {
		return err
	}

	if isEVMGenesis {
		// is is a subnet evm or a custom vm based on subnet evm
		if network == models.Mainnet {
			err = getSubnetEVMMainnetChainID(&sidecar, chain)
			if err != nil {
				return err
			}
			chainGenesis, err = updateSubnetEVMGenesisChainID(chainGenesis, uint(sidecar.SubnetEVMMainnetChainID))
			if err != nil {
				return err
			}
		}
		err = checkSubnetEVMDefaultAddressNotInAlloc(network, chain)
		if err != nil {
			return err
		}
	}

	if err = validateConvertOnlyFlag(cmd, deployFlags.BootstrapValidatorFlags, &deployFlags.ConvertOnly, deployFlags.LocalMachineFlags.UseLocalMachine); err != nil {
		return err
	}

	ux.Logger.PrintToUser("Deploying %s to %s", chains, network.Name())

	if network == models.Local {
		if err = deployLocalNetworkPreCheck(cmd, network, deployFlags.BootstrapValidatorFlags); err != nil {
			return err
		}
		app.Log.Debug("Deploy local")

		luxdVersion := deployFlags.LocalMachineFlags.UserProvidedLuxdVersion

		if luxdVersion == constants.DefaultLuxdVersion && deployFlags.LocalMachineFlags.LuxdBinaryPath == "" {
			luxdVersion, err = dependencies.GetLatestCLISupportedDependencyVersion(app, constants.LuxdRepoName, network, &sidecar.RPCVersion)
			if err != nil {
				if err != dependencies.ErrNoLuxdVersion {
					return err
				}
				luxdVersion = constants.LatestPreReleaseVersionTag
			}
		}

		ux.Logger.PrintToUser("")
		if err := networkcmd.Start(
			networkcmd.StartFlags{
				UserProvidedLuxdVersion: luxdVersion,
				LuxdBinaryPath:          deployFlags.LocalMachineFlags.LuxdBinaryPath,
				NumNodes:                numNodes,
			},
			false,
		); err != nil {
			return err
		}

		// check if blockchain rpc version matches what is currently running
		// for the case version or binary was provided
		_, _, networkRPCVersion, err := localnet.GetLocalNetworkLuxdVersion(app)
		if err != nil {
			return err
		}
		if networkRPCVersion != sidecar.RPCVersion {
			return fmt.Errorf(
				"the current local network uses rpc version %d but your blockchain has version %d and is not compatible",
				networkRPCVersion,
				sidecar.RPCVersion,
			)
		}

		useEwoq = true

		if !sidecar.Sovereign {
			// sovereign blockchains are deployed into new local clusters,
			// non sovereign blockchains are deployed into the local network itself
			if b, err := localnet.BlockchainAlreadyDeployedOnLocalNetwork(app, blockchainName); err != nil {
				return err
			} else if b {
				return fmt.Errorf("blockchain %s has already been deployed", blockchainName)
			}
		}
	}

	createSubnet := true
	var subnetID ids.ID
	if subnetIDStr != "" {
		subnetID, err = ids.FromString(subnetIDStr)
		if err != nil {
			return err
		}
		createSubnet = false
	} else if !subnetOnly && sidecar.Networks != nil {
		model, ok := sidecar.Networks[network.Name()]
		if ok {
			if model.SubnetID != ids.Empty && model.BlockchainID == ids.Empty {
				subnetID = model.SubnetID
				createSubnet = false
			}
		}
	}

	// Calculate estimated fees based on operation type
	fee := uint64(0)
	if !subnetOnly {
		// Add blockchain creation fee (typically 1 LUX)
		fee += 1_000_000_000 // 1 LUX in nLUX
	}
	if createSubnet {
		// Add subnet creation fee (typically 1 LUX)
		fee += 1_000_000_000 // 1 LUX in nLUX
	}
	// Add buffer for transaction fees (0.01 LUX)
	fee += 10_000_000 // 0.01 LUX in nLUX

	kc, err := keychain.GetKeychainFromCmdLineFlags(
		app,
		constants.PayTxsFeesMsg,
		network,
		keyName,
		useEwoq,
		useLedger,
		ledgerAddresses,
		fee,
	)
	if err != nil {
		return err
	}

	addresses := kc.Addresses().List()
	if len(addresses) == 0 {
		return fmt.Errorf("no addresses available in keychain")
	}
	availableBalance, err := utils.GetNetworkBalance(addresses[0], network)
	if err != nil {
		return err
	}
	deployBalance := uint64(deployFlags.BootstrapValidatorFlags.DeployBalanceLUX * float64(units.Lux))
	// whether user has created Lux Nodes when blockchain deploy command is called
	if sidecar.Sovereign && !subnetOnly {
		err = prepareBootstrapValidators(&bootstrapValidators, network, sidecar, *kc, blockchainName, deployBalance, availableBalance, &deployFlags.LocalMachineFlags, &deployFlags.BootstrapValidatorFlags)
		if err != nil {
			return err
		}
	} else if network == models.Local {
		sameControlKey = true
	}

	// from here on we are assuming a public deploy
	if subnetOnly && subnetIDStr != "" {
		return errMutuallyExlusiveSubnetFlags
	}

	if sidecar.Sovereign {
		requiredBalance := deployBalance * uint64(len(bootstrapValidators))
		if availableBalance < requiredBalance {
			return fmt.Errorf(
				"required balance for %d validators dynamic fee on PChain is %d but the given key has %d",
				len(bootstrapValidators),
				requiredBalance,
				availableBalance,
			)
		}
	}

	network.HandlePublicNetworkSimulation()

	if createSubnet {
		if sidecar.Sovereign {
			sameControlKey = true
		}
		controlKeys, threshold, err = promptOwners(
			kc,
			controlKeys,
			sameControlKey,
			threshold,
			subnetAuthKeys,
			true,
		)
		if err != nil {
			return err
		}
	} else {
		ux.Logger.PrintToUser("%s", luxlog.Blue.Wrap(
			fmt.Sprintf("Deploying into pre-existent subnet ID %s", subnetID.String()),
		))
		var isPermissioned bool
		isPermissioned, controlKeys, threshold, err = txutils.GetOwners(network, subnetID)
		if err != nil {
			return err
		}
		if !isPermissioned {
			return ErrNotPermissionedSubnet
		}
	}

	// add control keys to the keychain whenever possible
	if err := kc.AddAddresses(controlKeys); err != nil {
		return err
	}

	kcKeys, err := kc.PChainFormattedStrAddresses()
	if err != nil {
		return err
	}

	// get keys for blockchain tx signing
	if subnetAuthKeys != nil {
		if err := prompts.CheckSubnetAuthKeys(subnetAuthKeys, controlKeys, threshold); err != nil {
			return err
		}
	} else {
		// Filter control keys that are in the keychain
		filteredControlKeys := []string{}
		for _, controlKey := range controlKeys {
			for _, kcKey := range kcKeys {
				if controlKey == kcKey {
					filteredControlKeys = append(filteredControlKeys, controlKey)
					break
				}
			}
		}
		subnetAuthKeys, err = prompts.GetSubnetAuthKeys(app.Prompt, filteredControlKeys, threshold)
		if err != nil {
			return err
		}
	}
	ux.Logger.PrintToUser("Your blockchain auth keys for chain creation: %s", subnetAuthKeys)

	// deploy to public network
	deployer := subnet.NewPublicDeployer(app, useLedger, kc.Keychain, network)

	if createSubnet {
		subnetID, err = deployer.DeploySubnet(controlKeys, threshold)
		if err != nil {
			return err
		}
		// get the control keys in the same order as the tx
		_, controlKeys, threshold, err = txutils.GetOwners(network, subnetID)
		if err != nil {
			return err
		}
	}
	var (
		savePartialTx           bool
		blockchainID            ids.ID
		tx                      *txs.Tx
		remainingSubnetAuthKeys []string
		isFullySigned           bool
	)

	if !subnetOnly {
		isFullySigned, blockchainID, tx, remainingSubnetAuthKeys, err = deployer.DeployBlockchain(
			controlKeys,
			subnetAuthKeys,
			subnetID,
			chain,
			chainGenesis,
		)
		if err != nil {
			ux.Logger.PrintToUser("%s", luxlog.Red.Wrap(
				fmt.Sprintf("error deploying blockchain: %s. fix the issue and try again with a new deploy cmd", err),
			))
			return err
		}
		// Save partial transaction if not fully signed
		savePartialTx = !isFullySigned && err == nil
	}

	if err := PrintDeployResults(chain, subnetID, blockchainID); err != nil {
		return err
	}

	if savePartialTx {
		if err := SaveNotFullySignedTx(
			"Blockchain Creation",
			tx,
			chain,
			subnetAuthKeys,
			remainingSubnetAuthKeys,
			outputTxPath,
			false,
		); err != nil {
			return err
		}
	}

	// stop here if subnetOnly is true
	if subnetOnly {
		return nil
	}

	// needs to first stop relayer so non sovereign subnets successfully restart
	if sidecar.TeleporterReady && !warpSpec.SkipWarpDeploy && !warpSpec.SkipRelayerDeploy && network != models.Mainnet {
		_ = relayercmd.CallStop(nil, relayercmd.StopFlags{}, network)
	}

	tracked := false

	if sidecar.Sovereign {
		validatorManagerStr := validatormanagerSDK.ValidatorProxyContractAddress
		luxdBootstrapValidators, cancel, savePartialTx, err := convertSubnetToL1(
			bootstrapValidators,
			deployer,
			subnetID,
			blockchainID,
			network,
			chain,
			sidecar,
			controlKeys,
			subnetAuthKeys,
			validatorManagerStr,
			false,
		)
		if err != nil {
			return err
		}
		if cancel {
			return nil
		}

		if savePartialTx {
			return nil
		}
		if deployFlags.ConvertOnly || (!deployFlags.LocalMachineFlags.UseLocalMachine && clusterNameFlagValue == "") {
			printSuccessfulConvertOnlyOutput(blockchainName, subnetID.String(), deployFlags.BootstrapValidatorFlags.GenerateNodeID)
			return nil
		}

		tracked, err = InitializeValidatorManager(
			blockchainName,
			sidecar.ValidatorManagerOwner,
			subnetID,
			blockchainID,
			network,
			luxdBootstrapValidators,
			sidecar.ValidatorManagement == validatormanagertypes.ProofOfStake,
			validatorManagerStr,
			sidecar.ProxyContractOwner,
			sidecar.UseACP99,
			deployFlags.LocalMachineFlags.UseLocalMachine,
			deployFlags.SigAggFlags,
			deployFlags.ProofOfStakeFlags,
		)
		if err != nil {
			return err
		}
		if sidecar.UseACP99 && sidecar.ValidatorManagement == validatormanagertypes.ProofOfStake {
			sidecar, err := app.LoadSidecar(chain)
			if err != nil {
				return err
			}
			networkInfo := sidecar.Networks[network.Name()]
			networkInfo.ValidatorManagerAddress = validatormanagerSDK.SpecializationProxyContractAddress
			sidecar.Networks[network.Name()] = networkInfo
			if err := app.UpdateSidecar(&sidecar); err != nil {
				return err
			}
		}
	} else {
		if err := app.UpdateSidecarNetworks(
			&sidecar,
			network,
			subnetID,
			blockchainID,
		); err != nil {
			return err
		}
		if network == models.Local && !simulatedPublicNetwork() {
			ux.Logger.PrintToUser("")
			if err := localnet.LocalNetworkTrackSubnet(
				app,
				ux.Logger.PrintToUser,
				blockchainName,
			); err != nil {
				return err
			}
			tracked = true
		}
	}

	if sidecar.Sovereign && tracked {
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("%s", luxlog.Green.Wrap("Your L1 is ready for on-chain interactions."))
	}

	var warpErr, relayerErr error
	if sidecar.TeleporterReady && tracked && !warpSpec.SkipWarpDeploy {
		chainSpec := contract.ChainSpec{
			BlockchainName: blockchainName,
		}
		chainSpec.SetEnabled(true, false, false, false, false)
		deployWarpFlags := messengercmd.DeployFlags{
			ChainFlags: chainSpec,
			PrivateKeyFlags: contract.PrivateKeyFlags{
				KeyName: warpKeyName,
			},
			DeployMessenger:              true,
			DeployRegistry:               true,
			ForceRegistryDeploy:          true,
			Version:                      warpSpec.WarpVersion,
			MessengerContractAddressPath: warpSpec.MessengerContractAddressPath,
			MessengerDeployerAddressPath: warpSpec.MessengerDeployerAddressPath,
			MessengerDeployerTxPath:      warpSpec.MessengerDeployerTxPath,
			RegistryBydecodePath:         warpSpec.RegistryBydecodePath,
			CChainKeyName:                cchainIcmKeyName,
		}
		ux.Logger.PrintToUser("")
		if err := messengercmd.CallDeploy([]string{}, deployWarpFlags, network); err != nil {
			warpErr = err
			ux.Logger.RedXToUser("Interchain Messaging is not deployed due to: %v", warpErr)
		} else {
			ux.Logger.GreenCheckmarkToUser("Warp is successfully deployed")
			if network != models.Local && !deployFlags.LocalMachineFlags.UseLocalMachine {
				if flag := cmd.Flags().Lookup(skipRelayerFlagName); flag != nil && !flag.Changed {
					ux.Logger.PrintToUser("")
					yes, err := app.Prompt.CaptureYesNo("Do you want to setup local relayer for the messages to be interchanged, as Interchain Messaging was deployed to your blockchain?")
					if err != nil {
						return err
					}
					warpSpec.SkipRelayerDeploy = !yes
				}
			}
			if !warpSpec.SkipRelayerDeploy && network != models.Mainnet {
				if network == models.Local && warpSpec.RelayerBinPath == "" && warpSpec.RelayerVersion == constants.DefaultRelayerVersion {
					if b, extraLocalNetworkData, err := localnet.GetExtraLocalNetworkData(app, ""); err != nil {
						return err
					} else if b {
						warpSpec.RelayerBinPath = extraLocalNetworkData.RelayerPath
					}
				}
				deployRelayerFlags := relayercmd.DeployFlags{
					Version:            warpSpec.RelayerVersion,
					BinPath:            warpSpec.RelayerBinPath,
					LogLevel:           warpSpec.RelayerLogLevel,
					RelayCChain:        relayCChain,
					CChainFundingKey:   cChainFundingKey,
					BlockchainsToRelay: []string{blockchainName},
					Key:                relayerKeyName,
					Amount:             relayerAmount,
					AllowPrivateIPs:    relayerAllowPrivateIPs,
				}
				if network == models.Local {
					blockchains, err := localnet.GetLocalNetworkBlockchainsInfo(app)
					if err != nil {
						return err
					}
					deployRelayerFlags.BlockchainsToRelay = utils.Unique(sdkutils.Map(blockchains, func(i localnet.BlockchainInfo) string { return i.Name }))
				}
				if network == models.Local || deployFlags.LocalMachineFlags.UseLocalMachine {
					relayerKeyName, _, _, err := relayer.GetDefaultRelayerKeyInfo(app, blockchainName)
					if err != nil {
						return err
					}
					deployRelayerFlags.Key = relayerKeyName
					deployRelayerFlags.Amount = constants.DefaultRelayerAmount
					deployRelayerFlags.BlockchainFundingKey = constants.WarpKeyName
				}
				if network == models.Local {
					deployRelayerFlags.CChainFundingKey = "ewoq"
					deployRelayerFlags.CChainAmount = constants.DefaultRelayerAmount
				}
				if err := relayercmd.CallDeploy(nil, deployRelayerFlags, network); err != nil {
					relayerErr = err
					ux.Logger.RedXToUser("Relayer is not deployed due to: %v", relayerErr)
				} else {
					ux.Logger.GreenCheckmarkToUser("Relayer is successfully deployed")
				}
			}
		}
	}

	flags := make(map[string]string)
	flags[constants.MetricsNetwork] = network.Name()
	metrics.HandleTracking(app, flags, nil)

	if network.Kind() == models.Local && !simulatedPublicNetwork() {
		ux.Logger.PrintToUser("")
		_ = PrintSubnetInfo(blockchainName, true)
	}
	if warpErr != nil {
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("Interchain Messaging is not deployed due to: %v", warpErr)
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("To deploy Warp later on, call `lux warp deploy`")
		ux.Logger.PrintToUser("This does not affect L1 operations besides Interchain Messaging")
	}
	if relayerErr != nil {
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("Relayer is not deployed due to: %v", relayerErr)
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("To deploy a local relayer later on, call `lux interchain relayer deploy`")
		ux.Logger.PrintToUser("This does not affect L1 operations besides Interchain Messaging")
	}

	if tracked {
		if sidecar.Sovereign {
			ux.Logger.GreenCheckmarkToUser("L1 is successfully deployed on %s", network.Name())
		} else {
			ux.Logger.GreenCheckmarkToUser("Subnet is successfully deployed on %s", network.Name())
		}
	}

	return nil
}

func setBootstrapValidatorValidationID(luxdBootstrapValidators []*txs.ConvertNetToL1Validator, bootstrapValidators []models.SubnetValidator, subnetID ids.ID) {
	for index, luxdValidator := range luxdBootstrapValidators {
		for bootstrapValidatorIndex, validator := range bootstrapValidators {
			luxdValidatorNodeID, _ := ids.ToNodeID(luxdValidator.NodeID[:])
			if validator.NodeID == luxdValidatorNodeID.String() {
				validationID := subnetID.Append(uint32(index))
				bootstrapValidators[bootstrapValidatorIndex].ValidationID = validationID.String()
			}
		}
	}
}

func getClusterBootstrapValidators(
	clusterName string,
	network models.Network,
	deployBalance uint64,
) ([]models.SubnetValidator, error) {
	_, err := app.GetClusterConfig(clusterName)
	if err != nil {
		return nil, err
	}
	subnetValidators := []models.SubnetValidator{}
	// Cluster config parsing is handled differently in the new architecture
	// Remote cluster nodes are not currently supported
	hostIDs := []string{}
	for _, h := range hostIDs {
		// Node instance paths are managed by the cluster configuration
		_ = h
		// nodeID, pub, pop, err := utils.GetNodeParams(app.GetNodeInstanceDirPath(h))
		// if err != nil {
		//     return nil, fmt.Errorf("failed to parse nodeID: %w", err)
		// }
		// changeAddr, err = blockchain.GetKeyForChangeOwner(app, network)
		// if err != nil {
		//     return nil, err
		// }
		// ux.Logger.Info("Bootstrap validator info for Host: %s | Node ID: %s | Public Key: %s | Proof of Possession: %s", h, nodeID, hex.EncodeToString(pub), hex.EncodeToString(pop))
		// subnetValidators = append(subnetValidators, models.SubnetValidator{
		//     NodeID:               nodeID.String(),
		//     Weight:               constants.BootstrapValidatorWeight,
		//     Balance:              deployBalance,
		//     BLSPublicKey:         fmt.Sprintf("%s%s", "0x", hex.EncodeToString(pub)),
		//     BLSProofOfPossession: fmt.Sprintf("%s%s", "0x", hex.EncodeToString(pop)),
		//     ChangeOwnerAddr:      changeAddr,
		// })
	}
	return subnetValidators, nil
}

// ConvertToLuxdSubnetValidator converts subnet validators to L1 validator format
// Deactivation owner is handled through the validator management contract
func ConvertToLuxdSubnetValidator(subnetValidators []models.SubnetValidator) ([]*txs.ConvertNetToL1Validator, error) {
	bootstrapValidators := []*txs.ConvertNetToL1Validator{}
	for _, validator := range subnetValidators {
		nodeID, err := ids.NodeIDFromString(validator.NodeID)
		if err != nil {
			return nil, err
		}
		blsInfo, err := blockchain.ConvertToBLSProofOfPossession(validator.BLSPublicKey, validator.BLSProofOfPossession)
		if err != nil {
			return nil, fmt.Errorf("failure parsing BLS info: %w", err)
		}
		// Convert BLS public key from byte array to *bls.PublicKey
		blsPubKey, err := bls.PublicKeyFromCompressedBytes(blsInfo.PublicKey[:])
		if err != nil {
			return nil, fmt.Errorf("failed to parse BLS public key: %w", err)
		}
		// Parse change owner address when provided (for future use)
		if validator.ChangeOwnerAddr != "" {
			// For now, we'll just validate the address format
			_, err := address.ParseToIDs([]string{validator.ChangeOwnerAddr})
			if err != nil {
				return nil, fmt.Errorf("failure parsing change owner address: %w", err)
			}
			// The owner handling might need to be done differently with LP99
		}
		// Convert nodeID to byte slice for types.JSONByteSlice
		nodeIDBytes := nodeID.Bytes()

		// Create ProofOfPossession from BLS public key
		// For now, using empty proof since we don't have the private key
		var blsPubKeyBytes [bls.PublicKeyLen]byte
		copy(blsPubKeyBytes[:], bls.PublicKeyToCompressedBytes(blsPubKey))

		bootstrapValidator := &txs.ConvertNetToL1Validator{
			NodeID:  types.JSONByteSlice(nodeIDBytes[:]),
			Weight:  validator.Weight,
			Balance: validator.Balance,
			Signer: signer.ProofOfPossession{
				PublicKey: blsPubKeyBytes,
				// ProofOfPossession would need to be generated with private key
			},
			// These fields are required but we'll use empty owners for now
			RemainingBalanceOwner: message.PChainOwner{
				Threshold: 1,
				Addresses: []ids.ShortID{},
			},
			DeactivationOwner: message.PChainOwner{
				Threshold: 1,
				Addresses: []ids.ShortID{},
			},
		}
		bootstrapValidators = append(bootstrapValidators, bootstrapValidator)
	}
	// Sorting is not needed as ConvertSubnetToL1Validator doesn't implement Sortable
	// luxdutils.Sort(bootstrapValidators)
	return bootstrapValidators, nil
}

func scanChainsInSubnet(subnetName string) ([]string, error) {
	// Scan the subnet directory for chain configurations
	subnetDir := app.GetSubnetDir()
	entries, err := os.ReadDir(subnetDir)
	if err != nil {
		return nil, err
	}

	var chains []string
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() == subnetName {
			chains = append(chains, entry.Name())
		}
	}
	return chains, nil
}

func ValidateSubnetNameAndGetChains(args []string) ([]string, error) {
	// this should not be necessary but some bright guy might just be creating
	// the genesis by hand or something...
	if err := checkInvalidSubnetNames(args[0]); err != nil {
		return nil, fmt.Errorf("blockchain name %s is invalid: %w", args[0], err)
	}
	// Check subnet exists
	// Load chains from cached index for fast querying
	chains, err := getChainsInSubnet(args[0])
	if err != nil {
		// If no cache exists, scan directory directly
		chains, err = scanChainsInSubnet(args[0])
		if err != nil {
			return nil, fmt.Errorf("failed to getChainsInSubnet: %w", err)
		}
	}

	if len(chains) == 0 {
		return nil, errors.New("Invalid blockchain " + args[0])
	}

	return chains, nil
}

func SaveNotFullySignedTx(
	txName string,
	tx *txs.Tx,
	blockchainName string,
	subnetAuthKeys []string,
	remainingSubnetAuthKeys []string,
	outputTxPath string,
	forceOverwrite bool,
) error {
	signedCount := len(subnetAuthKeys) - len(remainingSubnetAuthKeys)
	ux.Logger.PrintToUser("")
	if signedCount == len(subnetAuthKeys) {
		ux.Logger.PrintToUser("All %d required %s signatures have been signed. "+
			"Saving tx to disk to enable commit.", len(subnetAuthKeys), txName)
	} else {
		ux.Logger.PrintToUser("%d of %d required %s signatures have been signed. "+
			"Saving tx to disk to enable remaining signing.", signedCount, len(subnetAuthKeys), txName)
	}
	if outputTxPath == "" {
		ux.Logger.PrintToUser("")
		var err error
		if forceOverwrite {
			outputTxPath, err = app.Prompt.CaptureString("Path to export partially signed tx to")
		} else {
			outputTxPath, err = app.Prompt.CaptureNewFilepath("Path to export partially signed tx to")
		}
		if err != nil {
			return err
		}
	}
	if forceOverwrite {
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("Overwriting %s", outputTxPath)
	}
	if err := txutils.SaveToDisk(tx, outputTxPath, forceOverwrite); err != nil {
		return err
	}
	if signedCount == len(subnetAuthKeys) {
		PrintReadyToSignMsg(blockchainName, outputTxPath)
	} else {
		PrintRemainingToSignMsg(blockchainName, remainingSubnetAuthKeys, outputTxPath)
	}
	return nil
}

func PrintReadyToSignMsg(
	blockchainName string,
	outputTxPath string,
) {
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Tx is fully signed, and ready to be committed")
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Commit command:")
	cmdLine := fmt.Sprintf("  lux transaction commit %s --input-tx-filepath %s", blockchainName, outputTxPath)
	if blockchainName == "" {
		cmdLine = fmt.Sprintf("  lux transaction commit --input-tx-filepath %s", outputTxPath)
	}
	ux.Logger.PrintToUser("%s", cmdLine)
}

func PrintRemainingToSignMsg(
	blockchainName string,
	remainingSubnetAuthKeys []string,
	outputTxPath string,
) {
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Addresses remaining to sign the tx")
	for _, subnetAuthKey := range remainingSubnetAuthKeys {
		ux.Logger.PrintToUser("  %s", subnetAuthKey)
	}
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Connect a ledger with one of the remaining addresses or choose a stored key "+
		"and run the signing command, or send %q to another user for signing.", outputTxPath)
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Signing command:")
	cmdline := fmt.Sprintf("  lux transaction sign %s --input-tx-filepath %s", blockchainName, outputTxPath)
	if blockchainName == "" {
		cmdline = fmt.Sprintf("  lux transaction sign --input-tx-filepath %s", outputTxPath)
	}
	ux.Logger.PrintToUser("%s", cmdline)
	ux.Logger.PrintToUser("")
}

func PrintDeployResults(blockchainName string, subnetID ids.ID, blockchainID ids.ID) error {
	t := ux.DefaultTable("Deployment results", nil)
	// SetColumnConfigs doesn't exist for tablewriter, skip it
	if blockchainName != "" {
		t.Append([]string{"Chain Name", blockchainName})
	}
	t.Append([]string{"Subnet ID", subnetID.String()})
	if blockchainName != "" {
		vmID, err := utils.VMID(blockchainName)
		if err != nil {
			return fmt.Errorf("failed to create VM ID from %s: %w", blockchainName, err)
		}
		t.Append([]string{"VM ID", vmID.String()})
	}
	if blockchainID != ids.Empty {
		t.Append([]string{"Blockchain ID", blockchainID.String()})
		t.Append([]string{"P-Chain TXID", blockchainID.String()})
	}
	t.Render()
	return nil
}

func LoadBootstrapValidator(bootstrapValidatorFlags flags.BootstrapValidatorFlags) ([]models.SubnetValidator, error) {
	if !utils.FileExists(bootstrapValidatorFlags.BootstrapValidatorsJSONFilePath) {
		return nil, fmt.Errorf("file path %q doesn't exist", bootstrapValidatorFlags.BootstrapValidatorsJSONFilePath)
	}
	jsonBytes, err := os.ReadFile(bootstrapValidatorFlags.BootstrapValidatorsJSONFilePath)
	if err != nil {
		return nil, err
	}
	var subnetValidators []models.SubnetValidator
	if err = json.Unmarshal(jsonBytes, &subnetValidators); err != nil {
		return nil, err
	}
	if err = validateSubnetValidatorsJSON(bootstrapValidatorFlags.GenerateNodeID, subnetValidators); err != nil {
		return nil, err
	}
	return subnetValidators, nil
}

func ConvertURIToPeers(uris []string) ([]info.Peer, error) {
	aggregatorPeers, err := blockchain.UrisToPeers(uris)
	if err != nil {
		return nil, err
	}
	nodeIDs := sdkutils.Map(aggregatorPeers, func(peer info.Peer) ids.NodeID {
		return peer.Info.ID
	})
	nodeIDsSet := set.Of(nodeIDs...)
	for _, uri := range uris {
		infoClient := info.NewClient(uri)
		ctx, cancel := utils.GetAPILargeContext()
		defer cancel()
		peers, err := infoClient.Peers(ctx, nil)
		if err != nil {
			return nil, err
		}
		for _, peer := range peers {
			if !nodeIDsSet.Contains(peer.Info.ID) {
				aggregatorPeers = append(aggregatorPeers, peer)
				nodeIDsSet.Add(peer.Info.ID)
			}
		}
	}
	return aggregatorPeers, nil
}

func simulatedPublicNetwork() bool {
	return os.Getenv(constants.SimulatePublicNetwork) != ""
}
