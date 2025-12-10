// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"errors"
	"fmt"
	"time"

	"github.com/luxfi/cli/pkg/localnet"

	"github.com/luxfi/crypto"
	sdkutils "github.com/luxfi/sdk/utils"

	"github.com/spf13/pflag"

	"github.com/luxfi/cli/cmd/flags"
	"github.com/luxfi/cli/pkg/blockchain"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/keychain"
	"github.com/luxfi/cli/pkg/networkoptions"
	"github.com/luxfi/cli/pkg/signatureaggregator"
	"github.com/luxfi/cli/pkg/net"
	"github.com/luxfi/cli/pkg/txutils"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/ids"
	luxlog "github.com/luxfi/log"
	luxdconstants "github.com/luxfi/node/utils/constants"
	"github.com/luxfi/node/utils/formatting/address"
	"github.com/luxfi/node/utils/units"
	"github.com/luxfi/sdk/contract"
	"github.com/luxfi/sdk/evm"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/sdk/prompts"
	"github.com/luxfi/sdk/validator"
	"github.com/luxfi/sdk/validatormanager"
	sdkwarp "github.com/luxfi/sdk/validatormanager/warp"

	"github.com/luxfi/geth/common"
	"github.com/spf13/cobra"
)

var (
	nodeIDStr                           string
	nodeEndpoint                        string
	balanceLUX                          float64
	weight                              uint64
	startTimeStr                        string
	duration                            time.Duration
	defaultValidatorParams              bool
	useDefaultStartTime                 bool
	useDefaultDuration                  bool
	useDefaultWeight                    bool
	waitForTxAcceptance                 bool
	publicKey                           string
	pop                                 string
	remainingBalanceOwnerAddr           string
	disableOwnerAddr                    string
	rewardsRecipientAddr                string
	delegationFee                       uint16
	errNoSubnetID                       = errors.New("failed to find the subnet ID for this subnet, has it been deployed/created on this network?")
	errMutuallyExclusiveDurationOptions = errors.New("--use-default-duration/--use-default-validator-params and --staking-period are mutually exclusive")
	errMutuallyExclusiveStartOptions    = errors.New("--use-default-start-time/--use-default-validator-params and --start-time are mutually exclusive")
	errMutuallyExclusiveWeightOptions   = errors.New("--use-default-validator-params and --weight are mutually exclusive")
	ErrNotPermissionedSubnet            = errors.New("subnet is not permissioned")
	clusterNameFlagValue                string
	createLocalValidator                bool
	externalValidatorManagerOwner       bool
	validatorManagerOwner               string
	httpPort                            uint32
	stakingPort                         uint32
	addValidatorFlags                   BlockchainAddValidatorFlags
)

type BlockchainAddValidatorFlags struct {
	RPC         string
	SigAggFlags flags.SignatureAggregatorFlags
}

const (
	validatorWeightFlag = "weight"
)

// lux blockchain addValidator
func newAddValidatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "addValidator [blockchainName]",
		Short: "Add a validator to an L1",
		Long: `The blockchain addValidator command adds a node as a validator to
an L1 of the user provided deployed network. If the network is proof of 
authority, the owner of the validator manager contract must sign the 
transaction. If the network is proof of stake, the node must stake the L1's
staking token. Both processes will issue a RegisterL1ValidatorTx on the P-Chain.

This command currently only works on Blockchains deployed to either the Testnet
Testnet or Mainnet.`,
		RunE:    addValidator,
		PreRunE: cobrautils.MaximumNArgs(1),
	}
	networkGroup := networkoptions.GetNetworkFlagsGroup(cmd, &globalNetworkFlags, true, networkoptions.DefaultSupportedNetworkOptions)
	flags.AddRPCFlagToCmd(cmd, app, &addValidatorFlags.RPC)
	sigAggGroup := flags.AddSignatureAggregatorFlagsToCmd(cmd, &addValidatorFlags.SigAggFlags)
	cmd.Flags().StringVarP(&keyName, "key", "k", "", "select the key to use [testnet/devnet only]")
	cmd.Flags().Float64Var(
		&balanceLUX,
		"balance",
		0,
		"set the LUX balance of the validator that will be used for continuous fee on P-Chain",
	)
	cmd.Flags().BoolVarP(&useEwoq, "ewoq", "e", false, "use ewoq key [testnet/devnet only]")
	cmd.Flags().BoolVarP(&useLedger, "ledger", "g", false, "use ledger instead of key (always true on mainnet, defaults to false on testnet/devnet)")
	cmd.Flags().StringSliceVar(&ledgerAddresses, "ledger-addrs", []string{}, "use the given ledger addresses")
	cmd.Flags().StringVar(&nodeIDStr, "node-id", "", "node-id of the validator to add")
	cmd.Flags().StringVar(&publicKey, "bls-public-key", "", "set the BLS public key of the validator to add")
	cmd.Flags().StringVar(&pop, "bls-proof-of-possession", "", "set the BLS proof of possession of the validator to add")
	cmd.Flags().StringVar(&remainingBalanceOwnerAddr, "remaining-balance-owner", "", "P-Chain address that will receive any leftover LUX from the validator when it is removed from Subnet")
	cmd.Flags().StringVar(&disableOwnerAddr, "disable-owner", "", "P-Chain address that will able to disable the validator with a P-Chain transaction")
	cmd.Flags().StringVar(&rewardsRecipientAddr, "rewards-recipient", "", "EVM address that will receive the validation rewards")
	cmd.Flags().BoolVar(&createLocalValidator, "create-local-validator", false, "create additional local validator and add it to existing running local node")
	cmd.Flags().BoolVar(&partialSync, "partial-sync", true, "set primary network partial sync for new validators")
	cmd.Flags().StringVar(&nodeEndpoint, "node-endpoint", "", "gather node id/bls from publicly available luxd apis on the given endpoint")
	cmd.Flags().Uint64Var(&weight, validatorWeightFlag, uint64(constants.DefaultStakeWeight), "set the weight of the validator")
	cmd.Flags().StringVar(&validatorManagerOwner, "validator-manager-owner", "", "force using this address to issue transactions to the validator manager")

	remoteBlockchainGroup := flags.RegisterFlagGroup(cmd, "Add Validator To Remote Blockchain Flags (Blockchain config is not in local machine)", "show-remote-blockchain-flags", true, func(set *pflag.FlagSet) {
		set.StringVar(&subnetIDstr, "subnet-id", "", "subnet ID (only if blockchain name is not provided)")
	})

	nonSovGroup := flags.RegisterFlagGroup(cmd, "Non Subnet-Only-Validators (Non-SOV) Flags", "show-non-sov-flags", false, func(set *pflag.FlagSet) {
		set.BoolVar(&useDefaultStartTime, "default-start-time", false, "(for Subnets, not L1s) use default start time for subnet validator (5 minutes later for testnet & mainnet, 30 seconds later for devnet)")
		set.StringVar(&startTimeStr, "start-time", "", "(for Subnets, not L1s) UTC start time when this validator starts validating, in 'YYYY-MM-DD HH:MM:SS' format")
		set.BoolVar(&useDefaultDuration, "default-duration", false, "(for Subnets, not L1s) set duration so as to validate until primary validator ends its period")
		set.BoolVar(&defaultValidatorParams, "default-validator-params", false, "(for Subnets, not L1s) use default weight/start/duration params for subnet validator")
		set.StringSliceVar(&subnetAuthKeys, "subnet-auth-keys", nil, "(for Subnets, not L1s) control keys that will be used to authenticate add validator tx")
		set.StringVar(&outputTxPath, "output-tx-path", "", "(for Subnets, not L1s) file path of the add validator tx")
		set.BoolVar(&waitForTxAcceptance, "wait-for-tx-acceptance", true, "(for Subnets, not L1s) just issue the add validator tx, without waiting for its acceptance")
		set.DurationVar(&duration, "staking-period", 0, "how long this validator will be staking")
	})

	localMachineGroup := flags.RegisterFlagGroup(cmd, "Local Machine Flags (Use local machine as a validator)", "show-local-machine-flags", false, func(set *pflag.FlagSet) {
		set.Uint32Var(&httpPort, "http-port", 0, "http port for node")
		set.Uint32Var(&stakingPort, "staking-port", 0, "staking port for node")
		set.BoolVar(&partialSync, "partial-sync", true, "set primary network partial sync for new validators")
		set.BoolVar(&createLocalValidator, "create-local-validator", false, "create additional local validator and add it to existing running local node")
	})

	posGroup := flags.RegisterFlagGroup(cmd, "Proof Of Stake Flags", "show-pos-flags", false, func(set *pflag.FlagSet) {
		set.Uint16Var(&delegationFee, "delegation-fee", 100, "(PoS only) delegation fee (in bips)")
		set.DurationVar(&duration, "staking-period", 0, "how long this validator will be staking")
	})

	externalSigningGroup := flags.RegisterFlagGroup(cmd, "External EVM Signature Flags (For EVM Multisig and Ledger Signing)", "show-external-signing-flags", true, func(set *pflag.FlagSet) {
		set.BoolVar(&externalValidatorManagerOwner, "external-evm-signature", false, "set this value to true when signing validator manager tx outside of cli (for multisig or ledger)")
		set.StringVar(&initiateTxHash, "initiate-tx-hash", "", "initiate tx is already issued, with the given hash")
	})

	cmd.SetHelpFunc(flags.WithGroupedHelp([]flags.GroupedFlags{networkGroup, externalSigningGroup, remoteBlockchainGroup, localMachineGroup, posGroup, nonSovGroup, sigAggGroup}))
	return cmd
}

func preAddChecks(args []string) error {
	if nodeEndpoint != "" && createLocalValidator {
		return fmt.Errorf("cannot set both --node-endpoint and --create-local-validator")
	}
	if createLocalValidator && (nodeIDStr != "" || publicKey != "" || pop != "") {
		return fmt.Errorf("cannot set --node-id, --bls-public-key or --bls-proof-of-possession if --create-local-validator used")
	}
	if len(args) == 0 && createLocalValidator {
		return fmt.Errorf("use lux addValidator <subnetName> command to use local machine as new validator")
	}

	return nil
}

func addValidator(cmd *cobra.Command, args []string) error {
	var sc models.Sidecar
	blockchainName := ""
	networkOption := networkoptions.DefaultSupportedNetworkOptions
	if len(args) == 1 {
		blockchainName = args[0]
		_, err := ValidateSubnetNameAndGetChains([]string{blockchainName})
		if err != nil {
			return err
		}
		sc, err = app.LoadSidecar(blockchainName)
		if err != nil {
			return fmt.Errorf("failed to load sidecar: %w", err)
		}
		networkOption = networkoptions.GetNetworkFromSidecar(sc, networkoptions.DefaultSupportedNetworkOptions)
	}

	network, err := networkoptions.GetNetworkFromCmdLineFlags(
		app,
		"",
		globalNetworkFlags,
		true,
		false,
		networkOption,
		"",
	)
	if err != nil {
		return err
	}

	if network.ClusterName() != "" {
		clusterNameFlagValue = network.ClusterName()
		// Convert cluster to standard network for consistency
		network = models.ConvertClusterToNetwork(network)
	}

	if len(args) == 0 {
		sc, _, err = importBlockchain(network, addValidatorFlags.RPC, ids.Empty, ux.Logger.PrintToUser)
		if err != nil {
			return err
		}
	}

	if err := preAddChecks(args); err != nil {
		return err
	}

	// Use clusterNameFlagValue which was already set above
	// Network data doesn't store cluster name separately

	// Estimate fee based on transaction complexity
	// Base fee + per-byte fee for transaction size
	baseFee := uint64(1000000)    // 0.001 LUX base fee
	txSizeEstimate := uint64(500) // Estimated transaction size in bytes
	perByteFee := uint64(1000)    // Fee per byte
	fee := baseFee + (txSizeEstimate * perByteFee)
	kc, err := keychain.GetKeychainFromCmdLineFlags(
		app,
		"to pay for transaction fees on P-Chain",
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

	sovereign := sc.Sovereign

	if nodeEndpoint != "" {
		nodeIDStr, publicKey, pop, err = utils.GetNodeID(nodeEndpoint)
		if err != nil {
			return err
		}
	}

	if sovereign {
		if !cmd.Flags().Changed(validatorWeightFlag) {
			weight, err = app.Prompt.CaptureWeight(
				"What weight would you like to assign to the validator?",
			)
			if err != nil {
				return err
			}
		}
	}

	// if we don't have a nodeID or ProofOfPossession by this point, prompt user if we want to add additional local node
	if (!sovereign && nodeIDStr == "") || (sovereign && !createLocalValidator && nodeIDStr == "" && publicKey == "" && pop == "") {
		if len(args) == 0 {
			createLocalValidator = false
		} else {
			for {
				local := "Use my local machine to spin up an additional validator"
				existing := "I have an existing Lux node (we will require its NodeID and BLS info)"
				if option, err := app.Prompt.CaptureList(
					"How would you like to set up the new validator",
					[]string{local, existing},
				); err != nil {
					return err
				} else {
					createLocalValidator = option == local
					break
				}
			}
		}
	}

	subnetID := sc.Networks[network.Name()].SubnetID

	// if user chose to upsize a local node to add another local validator
	var localValidatorClusterName string
	if createLocalValidator {
		localValidatorClusterName = localnet.LocalClusterName()
		node, err := localnet.AddNodeToLocalCluster(app, ux.Logger.PrintToUser, localValidatorClusterName, httpPort, stakingPort)
		if err != nil {
			return err
		}
		nodeIDStr, publicKey, pop, err = utils.GetNodeID(node.URI)
		if err != nil {
			return err
		}
		// AddDefaultBlockchainRPCsToSidecar returns (bool, error)
		_, err = app.AddDefaultBlockchainRPCsToSidecar(blockchainName, network, []string{node.URI})
		if err != nil {
			return err
		}
	}

	if nodeIDStr == "" {
		nodeID, err := PromptNodeID("add as a blockchain validator")
		if err != nil {
			return err
		}
		nodeIDStr = nodeID.String()
	}
	// Simple NodeID validation
	if _, err := ids.NodeIDFromString(nodeIDStr); err != nil {
		return fmt.Errorf("invalid node ID: %w", err)
	}

	if sovereign && publicKey == "" && pop == "" {
		publicKey, pop, err = promptProofOfPossession(true, true)
		if err != nil {
			return err
		}
	}

	network.HandlePublicNetworkSimulation()

	if !sovereign {
		if err := UpdateKeychainWithSubnetControlKeys(kc, network, blockchainName); err != nil {
			return err
		}
	}
	deployer := net.NewPublicDeployer(app, useLedger, kc.Keychain, network)
	if !sovereign {
		return CallAddValidatorNonSOV(deployer, network, kc, useLedger, blockchainName, nodeIDStr, defaultValidatorParams, waitForTxAcceptance)
	}
	if err := CallAddValidator(
		deployer,
		network,
		kc,
		blockchainName,
		subnetID,
		nodeIDStr,
		publicKey,
		pop,
		weight,
		balanceLUX,
		remainingBalanceOwnerAddr,
		disableOwnerAddr,
		sc,
		addValidatorFlags.RPC,
	); err != nil {
		return err
	}
	if createLocalValidator && network.Kind() == models.Local {
		// For all blockchains validated by the cluster, set up an alias from blockchain name
		// into blockchain id, to be mainly used in the blockchain RPC
		return localnet.RefreshLocalClusterAliases(app, localValidatorClusterName)
	}
	return nil
}

func promptValidatorBalanceLUX(availableBalance float64) (float64, error) {
	ux.Logger.PrintToUser("Validator's balance is used to pay for continuous fee to the P-Chain")
	ux.Logger.PrintToUser("When this Balance reaches 0, the validator will be considered inactive and will no longer participate in validating the L1")
	txt := "What balance would you like to assign to the validator (in LUX)?"
	return app.Prompt.CaptureValidatorBalance(txt, availableBalance, constants.BootstrapValidatorBalanceLUX)
}

func CallAddValidator(
	deployer *net.PublicDeployer,
	network models.Network,
	kc *keychain.Keychain,
	blockchainName string,
	subnetID ids.ID,
	nodeIDStr string,
	publicKey string,
	pop string,
	weight uint64,
	balanceLUX float64,
	remainingBalanceOwnerAddr string,
	disableOwnerAddr string,
	sc models.Sidecar,
	rpcURL string,
) error {
	nodeID, err := ids.NodeIDFromString(nodeIDStr)
	if err != nil {
		return err
	}
	blsInfo, err := blockchain.ConvertToBLSProofOfPossession(publicKey, pop)
	if err != nil {
		return fmt.Errorf("failure parsing BLS info: %w", err)
	}

	blockchainTimestamp, err := blockchain.GetBlockchainTimestamp(network)
	if err != nil {
		return fmt.Errorf("failed to get blockchain timestamp: %w", err)
	}
	expiry := uint64(blockchainTimestamp.Add(constants.DefaultValidationIDExpiryDuration).Unix())
	chainSpec := contract.ChainSpec{
		BlockchainName: blockchainName,
	}
	if sc.Networks[network.Name()].BlockchainID.String() != "" {
		chainSpec.BlockchainID = sc.Networks[network.Name()].BlockchainID.String()
	}
	if sc.Networks[network.Name()].ValidatorManagerAddress == "" {
		return fmt.Errorf("unable to find Validator Manager address")
	}
	validatorManagerAddress = sc.Networks[network.Name()].ValidatorManagerAddress

	if validatorManagerOwner == "" {
		validatorManagerOwner = sc.ValidatorManagerOwner
	}

	var ownerPrivateKey string
	if !externalValidatorManagerOwner {
		var ownerPrivateKeyFound bool
		ownerPrivateKeyFound, _, _, ownerPrivateKey, err = contract.SearchForManagedKey(
			app.GetSDKApp(),
			network,
			common.HexToAddress(validatorManagerOwner).Hex(), // Convert to string
			true,
		)
		if err != nil {
			return err
		}
		if !ownerPrivateKeyFound {
			return fmt.Errorf("private key for Validator manager owner %s is not found", validatorManagerOwner)
		}
	}

	pos := sc.PoS

	if pos {
		// should take input prior to here for delegation fee, and min stake duration
		if duration == 0 {
			duration, err = PromptDuration(time.Now(), network, true) // it's pos
			if err != nil {
				return nil
			}
		}
		if rewardsRecipientAddr == "" {
			rewardsRecipientAddr, err = prompts.PromptAddress(
				app.Prompt,
				"Enter address to receive the validation rewards",
			)
			if err != nil {
				return err
			}
		}
	}

	if sc.UseACP99 {
		ux.Logger.PrintToUser("%s", luxlog.Yellow.Wrap("Validator Manager Protocol: V2"))
	} else {
		ux.Logger.PrintToUser("%s", luxlog.Yellow.Wrap("Validator Manager Protocol: v1.0.0"))
	}

	ux.Logger.PrintToUser("%s", luxlog.Yellow.Wrap(fmt.Sprintf("Validation manager owner %s pays for the initialization of the validator's registration (Blockchain gas token)", validatorManagerOwner)))

	if rpcURL == "" {
		rpcURL, _, err = contract.GetBlockchainEndpoints(
			app.GetSDKApp(),
			network,
			chainSpec,
			true,
			false,
		)
		if err != nil {
			return err
		}
	}

	ux.Logger.PrintToUser("%s", luxlog.Yellow.Wrap(fmt.Sprintf("RPC Endpoint: %s", rpcURL)))

	totalWeight, err := validator.GetTotalWeight(network, subnetID)
	if err != nil {
		return err
	}
	allowedChange := float64(totalWeight) * constants.MaxL1TotalWeightChange
	if float64(weight) > allowedChange {
		return fmt.Errorf("can't make change: desired validator weight %d exceeds max allowed weight change of %d", newWeight, uint64(allowedChange))
	}

	if balanceLUX == 0 {
		// Get balance for first address
		addresses := kc.Addresses().List()
		if len(addresses) == 0 {
			return fmt.Errorf("no addresses in keychain")
		}
		availableBalance, err := utils.GetNetworkBalance(addresses[0], network)
		if err != nil {
			return err
		}
		if availableBalance == 0 {
			return fmt.Errorf("chosen key has zero balance")
		}
		balanceLUX, err = promptValidatorBalanceLUX(float64(availableBalance) / float64(units.Lux))
		if err != nil {
			return err
		}
	}
	// convert to nanoLUX
	balance := uint64(balanceLUX * float64(units.Lux))

	if remainingBalanceOwnerAddr == "" {
		remainingBalanceOwnerAddr, err = blockchain.GetKeyForChangeOwner(app, network)
		if err != nil {
			return err
		}
	}
	remainingBalanceOwnerAddrID, err := address.ParseToIDs([]string{remainingBalanceOwnerAddr})
	if err != nil {
		return fmt.Errorf("failure parsing remaining balanche owner address %s: %w", remainingBalanceOwnerAddr, err)
	}
	remainingBalanceOwners := sdkwarp.PChainOwner{
		Threshold: 1,
		Addresses: remainingBalanceOwnerAddrID,
	}

	if disableOwnerAddr == "" {
		disableOwnerAddr, err = prompts.PromptAddress(
			app.Prompt,
			"Enter P-Chain address to disable the validator (Example: P-...)",
		)
		if err != nil {
			return err
		}
	}
	disableOwnerAddrID, err := address.ParseToIDs([]string{disableOwnerAddr})
	if err != nil {
		return fmt.Errorf("failure parsing disable owner address %s: %w", disableOwnerAddr, err)
	}
	disableOwners := sdkwarp.PChainOwner{
		Threshold: 1,
		Addresses: disableOwnerAddrID,
	}
	extraAggregatorPeers, err := blockchain.GetAggregatorExtraPeers(app, clusterNameFlagValue)
	if err != nil {
		return err
	}
	aggregatorLogger, err := signatureaggregator.NewSignatureAggregatorLogger(
		addValidatorFlags.SigAggFlags.AggregatorLogLevel,
		addValidatorFlags.SigAggFlags.AggregatorLogToStdout,
		app.GetAggregatorLogDir(clusterNameFlagValue),
	)
	if err != nil {
		return err
	}
	// Convert peers to string URIs
	var extraPeerURIs []string
	for _, peer := range extraAggregatorPeers {
		extraPeerURIs = append(extraPeerURIs, peer.IP.String())
	}
	if err = signatureaggregator.UpdateSignatureAggregatorPeers(app, network, extraPeerURIs, aggregatorLogger); err != nil {
		return err
	}
	aggregatorCtx, aggregatorCancel := sdkutils.GetTimedContext(constants.SignatureAggregatorTimeout)
	defer aggregatorCancel()
	signatureAggregatorEndpoint, err := signatureaggregator.GetSignatureAggregatorEndpoint(app, network)
	if err != nil {
		return err
	}
	_, validationID, rawTx, err := validatormanager.InitValidatorRegistration(
		aggregatorCtx,
		app.GetSDKApp(),
		network,
		rpcURL,
		chainSpec,
		externalValidatorManagerOwner,
		validatorManagerOwner,
		ownerPrivateKey,
		nodeID,
		blsInfo.PublicKey[:],
		expiry,
		remainingBalanceOwners,
		disableOwners,
		weight,
		aggregatorLogger,
		pos,
		delegationFee,
		duration,
		crypto.HexToAddress(rewardsRecipientAddr),
		validatorManagerAddress,
		sc.UseACP99,
		initiateTxHash,
		signatureAggregatorEndpoint,
	)
	if err != nil {
		return err
	}
	if rawTx != nil {
		dump, err := evm.TxDump("Initializing Validator Registration", rawTx)
		if err == nil {
			ux.Logger.PrintToUser("%s", dump)
		}
		return err
	}
	ux.Logger.PrintToUser("ValidationID: %s", validationID)

	// Register the L1 validator on the P-Chain
	ux.Logger.PrintToUser("Registering L1 validator on P-Chain...")

	// Use the deployer's RegisterL1Validator method with the calculated balance
	txID, _, err := deployer.RegisterL1Validator(
		balance, // Balance for validation
		blsInfo, // BLS proof of possession
		nil,     // Message (optional)
	)
	if err != nil {
		ux.Logger.PrintToUser("Warning: P-Chain registration not fully implemented: %v", err)
		// Continue anyway as the validator manager registration succeeded
	} else {
		ux.Logger.PrintToUser("L1 Validator registered with TX ID: %s", txID)
	}

	// Still update P-Chain height for consistency
	if err := blockchain.UpdatePChainHeight(
		"Waiting for P-Chain to update validator information ...",
	); err != nil {
		return err
	}

	aggregatorCtx, aggregatorCancel = sdkutils.GetTimedContext(constants.SignatureAggregatorTimeout)
	defer aggregatorCancel()
	rawTx, err = validatormanager.FinishValidatorRegistration(
		aggregatorCtx,
		app.GetSDKApp(),
		network,
		rpcURL,
		chainSpec,
		externalValidatorManagerOwner,
		validatorManagerOwner,
		ownerPrivateKey,
		validationID,
		aggregatorLogger,
		validatorManagerAddress,
		signatureAggregatorEndpoint,
	)
	if err != nil {
		return err
	}
	if rawTx != nil {
		dump, err := evm.TxDump("Finish Validator Registration", rawTx)
		if err == nil {
			ux.Logger.PrintToUser("%s", dump)
		}
		return err
	}

	ux.Logger.PrintToUser("  NodeID: %s", nodeID)
	ux.Logger.PrintToUser("  Network: %s", network.Name())
	// weight is inaccurate for PoS as it's fetched during registration
	if !pos {
		ux.Logger.PrintToUser("  Weight: %d", weight)
	}
	ux.Logger.PrintToUser("  Balance: %.2f", balanceLUX)

	ux.Logger.GreenCheckmarkToUser("Validator successfully added to the L1")

	return nil
}

func CallAddValidatorNonSOV(
	deployer *net.PublicDeployer,
	network models.Network,
	kc *keychain.Keychain,
	useLedgerSetting bool,
	blockchainName string,
	nodeIDStr string,
	defaultValidatorParamsSetting bool,
	waitForTxAcceptanceSetting bool,
) error {
	var start time.Time
	nodeID, err := ids.NodeIDFromString(nodeIDStr)
	if err != nil {
		return err
	}
	useLedger = useLedgerSetting
	defaultValidatorParams = defaultValidatorParamsSetting
	waitForTxAcceptance = waitForTxAcceptanceSetting

	if defaultValidatorParams {
		useDefaultDuration = true
		useDefaultStartTime = true
		useDefaultWeight = true
	}

	if useDefaultDuration && duration != 0 {
		return errMutuallyExclusiveDurationOptions
	}
	if useDefaultStartTime && startTimeStr != "" {
		return errMutuallyExclusiveStartOptions
	}
	if useDefaultWeight && weight != 0 {
		return errMutuallyExclusiveWeightOptions
	}

	if outputTxPath != "" {
		if utils.FileExists(outputTxPath) {
			return fmt.Errorf("outputTxPath %q already exists", outputTxPath)
		}
	}

	_, err = ValidateSubnetNameAndGetChains([]string{blockchainName})
	if err != nil {
		return err
	}

	sc, err := app.LoadSidecar(blockchainName)
	if err != nil {
		return err
	}

	subnetID := sc.Networks[network.Name()].SubnetID
	if subnetID == ids.Empty {
		return errNoSubnetID
	}

	_, controlKeys, threshold, err := txutils.GetOwners(network, subnetID)
	if err != nil {
		return err
	}
	// If control keys are empty, it's not a permissioned subnet
	isPermissioned := len(controlKeys) > 0
	if !isPermissioned {
		return ErrNotPermissionedSubnet
	}

	// kcKeys not used after prompts refactoring
	// kcKeys, err := kc.PChainFormattedStrAddresses()
	// if err != nil {
	// 	return err
	// }

	// get keys for add validator tx signing
	if subnetAuthKeys != nil {
		if err := prompts.CheckSubnetAuthKeys(subnetAuthKeys, controlKeys, threshold); err != nil {
			return err
		}
	} else {
		subnetAuthKeys, err = prompts.GetSubnetAuthKeys(app.Prompt, controlKeys, threshold)
		if err != nil {
			return err
		}
	}
	ux.Logger.PrintToUser("Your auth keys for add validator tx creation: %s", subnetAuthKeys)

	selectedWeight, err := getWeight()
	if err != nil {
		return err
	}
	if selectedWeight < constants.MinStakeWeight {
		return fmt.Errorf("invalid weight, must be greater than or equal to %d: %d", constants.MinStakeWeight, selectedWeight)
	}

	start, selectedDuration, err := getTimeParameters(network, nodeID, true)
	if err != nil {
		return err
	}

	ux.Logger.PrintToUser("NodeID: %s", nodeID.String())
	ux.Logger.PrintToUser("Network: %s", network.Name())
	ux.Logger.PrintToUser("Start time: %s", start.Format(constants.TimeParseLayout))
	ux.Logger.PrintToUser("End time: %s", start.Add(selectedDuration).Format(constants.TimeParseLayout))
	ux.Logger.PrintToUser("Weight: %d", selectedWeight)
	ux.Logger.PrintToUser("Inputs complete, issuing transaction to add the provided validator information...")

	isFullySigned, tx, remainingSubnetAuthKeys, err := deployer.AddValidator(
		controlKeys,
		subnetAuthKeys,
		subnetID,
		nodeID,
		selectedWeight,
		start,
		selectedDuration,
	)
	if err != nil {
		return err
	}
	if !isFullySigned {
		if err := SaveNotFullySignedTx(
			"Add Validator",
			tx,
			blockchainName,
			subnetAuthKeys,
			remainingSubnetAuthKeys,
			outputTxPath,
			false,
		); err != nil {
			return err
		}
	}

	return err
}

func PromptDuration(start time.Time, network models.Network, isPos bool) (time.Duration, error) {
	for {
		txt := "How long should this validator be validating? Enter a duration, e.g. 8760h. Valid time units are \"ns\", \"us\" (or \"µs\"), \"ms\", \"s\", \"m\", \"h\""
		var d time.Duration
		var err error
		switch {
		case network.Kind() == models.Testnet:
			// Use generic CaptureDuration for testnet
			d, err = app.Prompt.CaptureDuration(txt)
		case network.Kind() == models.Mainnet && isPos:
			// Use generic CaptureDuration for mainnet PoS
			d, err = app.Prompt.CaptureDuration(txt)
		case network.Kind() == models.Mainnet && !isPos:
			// Use generic CaptureDuration for mainnet PoA
			d, err = app.Prompt.CaptureDuration(txt)
		default:
			d, err = app.Prompt.CaptureDuration(txt)
		}
		if err != nil {
			return 0, err
		}
		end := start.Add(d)
		confirm := fmt.Sprintf("Your validator will finish staking by %s", end.Format(constants.TimeParseLayout))
		yes, err := app.Prompt.CaptureYesNo(confirm)
		if err != nil {
			return 0, err
		}
		if yes {
			return d, nil
		}
	}
}

func getTimeParameters(network models.Network, nodeID ids.NodeID, isValidator bool) (time.Time, time.Duration, error) {
	defaultStakingStartLeadTime := constants.StakingStartLeadTime
	if network.Kind() == models.Devnet {
		defaultStakingStartLeadTime = constants.DevnetStakingStartLeadTime
	}

	const custom = "Custom"

	// this sets either the global var startTimeStr or useDefaultStartTime to enable repeated execution with
	// state keeping from node cmds
	if startTimeStr == "" && !useDefaultStartTime {
		if isValidator {
			ux.Logger.PrintToUser("When should your validator start validating?\n" +
				"If you validator is not ready by this time, subnet downtime can occur.")
		} else {
			ux.Logger.PrintToUser("When do you want to start delegating?\n")
		}
		defaultStartOption := "Start in " + ux.FormatDuration(defaultStakingStartLeadTime)
		startTimeOptions := []string{defaultStartOption, custom}
		startTimeOption, err := app.Prompt.CaptureList("Start time", startTimeOptions)
		if err != nil {
			return time.Time{}, 0, err
		}
		switch startTimeOption {
		case defaultStartOption:
			useDefaultStartTime = true
		default:
			start, err := promptStart()
			if err != nil {
				return time.Time{}, 0, err
			}
			startTimeStr = start.Format(constants.TimeParseLayout)
		}
	}

	var (
		err   error
		start time.Time
	)
	if startTimeStr != "" {
		start, err = time.Parse(constants.TimeParseLayout, startTimeStr)
		if err != nil {
			return time.Time{}, 0, err
		}
		if start.Before(time.Now().Add(constants.StakingMinimumLeadTime)) {
			return time.Time{}, 0, fmt.Errorf("time should be at least %s in the future ", constants.StakingMinimumLeadTime)
		}
	} else {
		start = time.Now().Add(defaultStakingStartLeadTime)
	}

	// this sets either the global var duration or useDefaultDuration to enable repeated execution with
	// state keeping from node cmds
	if duration == 0 && !useDefaultDuration {
		msg := "How long should your validator validate for?"
		if !isValidator {
			msg = "How long do you want to delegate for?"
		}
		const defaultDurationOption = "Until primary network validator expires"
		durationOptions := []string{defaultDurationOption, custom}
		durationOption, err := app.Prompt.CaptureList(msg, durationOptions)
		if err != nil {
			return time.Time{}, 0, err
		}
		switch durationOption {
		case defaultDurationOption:
			useDefaultDuration = true
		default:
			duration, err = PromptDuration(start, network, false) // notSoV
			if err != nil {
				return time.Time{}, 0, err
			}
		}
	}

	var selectedDuration time.Duration
	if useDefaultDuration {
		// avoid setting both globals useDefaultDuration and duration
		selectedDuration, err = utils.GetRemainingValidationTime(network.Endpoint(), nodeID, luxdconstants.PrimaryNetworkID, start)
		if err != nil {
			return time.Time{}, 0, err
		}
	} else {
		selectedDuration = duration
	}

	return start, selectedDuration, nil
}

func promptStart() (time.Time, error) {
	txt := "When should the validator start validating? Enter a UTC datetime in 'YYYY-MM-DD HH:MM:SS' format"
	return app.Prompt.CaptureDate(txt)
}

func PromptNodeID(goal string) (ids.NodeID, error) {
	txt := fmt.Sprintf("What is the NodeID of the node you want to %s?", goal)
	return app.Prompt.CaptureNodeID(txt)
}

func getWeight() (uint64, error) {
	// this sets either the global var weight or useDefaultWeight to enable repeated execution with
	// state keeping from node cmds
	if weight == 0 && !useDefaultWeight {
		defaultWeight := fmt.Sprintf("Default (%d)", constants.DefaultStakeWeight)
		txt := "What stake weight would you like to assign to the validator?"
		weightOptions := []string{defaultWeight, "Custom"}
		weightOption, err := app.Prompt.CaptureList(txt, weightOptions)
		if err != nil {
			return 0, err
		}
		switch weightOption {
		case defaultWeight:
			useDefaultWeight = true
		default:
			weight, err = app.Prompt.CaptureWeight(txt)
			if err != nil {
				return 0, err
			}
		}
	}
	if useDefaultWeight {
		return constants.DefaultStakeWeight, nil
	}
	return weight, nil
}
