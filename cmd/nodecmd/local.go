// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package nodecmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	sdkutils "github.com/luxfi/sdk/utils"

	"github.com/luxfi/cli/pkg/dependencies"

	"github.com/luxfi/cli/cmd/flags"
	"github.com/luxfi/cli/pkg/blockchain"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/keychain"
	"github.com/luxfi/cli/pkg/localnet"
	"github.com/luxfi/cli/pkg/networkoptions"
	"github.com/luxfi/cli/pkg/node"
	"github.com/luxfi/cli/pkg/signatureaggregator"
	"github.com/luxfi/cli/pkg/subnet"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/ids"
	luxlog "github.com/luxfi/log"
	"github.com/luxfi/node/api/info"
	"github.com/luxfi/node/config"
	"github.com/luxfi/node/utils/formatting/address"
	"github.com/luxfi/node/utils/units"
	"github.com/luxfi/node/vms/platformvm"
	"github.com/luxfi/sdk/contract"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/sdk/prompts"
	"github.com/luxfi/sdk/validatormanager"
	warpMessage "github.com/luxfi/sdk/validatormanager/warp"

	"github.com/luxfi/crypto"
	"github.com/spf13/cobra"
)

var (
	luxdBinaryPath string

	bootstrapIDs                []string
	bootstrapIPs                []string
	genesisPath                 string
	upgradePath                 string
	stakingTLSKeyPaths          []string
	stakingCertKeyPaths         []string
	stakingSignerKeyPaths       []string
	numNodes                    uint32
	nodeConfigPath              string
	partialSync                 bool
	stakeAmount                 uint64
	balanceLUX                  float64
	remainingBalanceOwnerAddr   string
	disableOwnerAddr            string
	delegationFee               uint16
	minimumStakeDuration        uint64
	rewardsRecipientAddr        string
	latestLuxdReleaseVersion    bool
	latestLuxdPreReleaseVersion bool
	validatorManagerAddress     string
	useACP99                    bool
	httpPorts                   []uint
	stakingPorts                []uint
	localValidateFlags          NodeLocalValidateFlags
)

// const snapshotName = "local_snapshot"
func newLocalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "local",
		Short: "Suite of commands for a local lux node",
		Long:  `The node local command suite provides a collection of commands related to local nodes`,
		RunE:  cobrautils.CommandSuiteUsage,
	}
	// node local start
	cmd.AddCommand(newLocalStartCmd())
	// node local stop
	cmd.AddCommand(newLocalStopCmd())
	// node local destroy
	cmd.AddCommand(newLocalDestroyCmd())
	// node local track
	cmd.AddCommand(newLocalTrackCmd())
	// node local status
	cmd.AddCommand(newLocalStatusCmd())
	// node local validate
	cmd.AddCommand(newLocalValidateCmd())
	return cmd
}

func newLocalStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start [clusterName]",
		Short: "Create or restart Lux nodes on local machine",
		Long: `The node local start command creates Lux nodes on the local machine,
or restarts previously created ones.
Once this command is completed, you will have to wait for the Lux node
to finish bootstrapping on the primary network before running further
commands on it, e.g. validating a Subnet. 

You can check the bootstrapping status by running lux node status local.
`,
		Args:              cobra.ExactArgs(1),
		RunE:              localStartNode,
		PersistentPostRun: handlePostRun,
	}
	// Network flags handled at higher level to avoid conflicts
	cmd.Flags().BoolVar(&latestLuxdReleaseVersion, "latest-luxd-version", true, "install latest luxd release version on node/s")
	cmd.Flags().BoolVar(&latestLuxdPreReleaseVersion, "latest-luxd-pre-release-version", false, "install latest luxd pre-release version on node/s")
	cmd.Flags().StringVar(&useCustomLuxgoVersion, "custom-luxd-version", "", "install given luxd version on node/s")
	cmd.Flags().StringVar(&luxdBinaryPath, "luxd-path", "", "use this luxd binary path")
	cmd.Flags().StringArrayVar(&bootstrapIDs, "bootstrap-id", []string{}, "nodeIDs of bootstrap nodes")
	cmd.Flags().StringArrayVar(&bootstrapIPs, "bootstrap-ip", []string{}, "IP:port pairs of bootstrap nodes")
	cmd.Flags().StringVar(&genesisPath, "genesis", "", "path to genesis file")
	cmd.Flags().StringVar(&upgradePath, "upgrade", "", "path to upgrade file")
	cmd.Flags().StringSliceVar(&stakingTLSKeyPaths, "staking-tls-key-path", []string{}, "path to provided staking tls key for node(s)")
	cmd.Flags().StringSliceVar(&stakingCertKeyPaths, "staking-cert-key-path", []string{}, "path to provided staking cert key for node(s)")
	cmd.Flags().StringSliceVar(&stakingSignerKeyPaths, "staking-signer-key-path", []string{}, "path to provided staking signer key for node(s)")
	cmd.Flags().Uint32Var(&numNodes, "num-nodes", 1, "number of Lux nodes to create on local machine")
	cmd.Flags().StringVar(&nodeConfigPath, "node-config", "", "path to common luxd config settings for all nodes")
	cmd.Flags().BoolVar(&partialSync, "partial-sync", true, "primary network partial sync")
	cmd.Flags().UintSliceVar(&httpPorts, "http-port", []uint{}, "http port for node(s)")
	cmd.Flags().UintSliceVar(&stakingPorts, "staking-port", []uint{}, "staking port for node(s)")
	return cmd
}

func newLocalStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop [clusterName]",
		Short: "Stop local nodes",
		Long:  `Stop local nodes.`,
		Args:  cobra.MaximumNArgs(1),
		RunE:  localStopNode,
	}
}

func newLocalTrackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "track [clusterName] [blockchainName]",
		Short: "Track specified blockchain with local node",
		Long:  "Track specified blockchain with local node",
		Args:  cobra.ExactArgs(2),
		RunE:  localTrack,
	}
	return cmd
}

func newLocalDestroyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "destroy [clusterName]",
		Short: "Cleanup local node",
		Long:  `Cleanup local node.`,
		Args:  cobra.ExactArgs(1),
		RunE:  localDestroyNode,
	}
}

func newLocalStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Get status of local node",
		Long:  `Get status of local node.`,
		Args:  cobra.MaximumNArgs(1),
		RunE:  localStatus,
	}

	cmd.Flags().StringVar(&blockchainName, "l1", "", "specify the blockchain the node is syncing with")
	cmd.Flags().StringVar(&blockchainName, "blockchain", "", "specify the blockchain the node is syncing with")

	return cmd
}

func localStartNode(_ *cobra.Command, args []string) error {
	clusterName := args[0]
	var (
		err     error
		genesis []byte
		upgrade []byte
	)
	if genesisPath != "" {
		genesis, err = os.ReadFile(genesisPath)
		if err != nil {
			return fmt.Errorf("could not read genesis at %s: %w", genesisPath, err)
		}
	}
	if upgradePath != "" {
		upgrade, err = os.ReadFile(upgradePath)
		if err != nil {
			return fmt.Errorf("could not read upgrade at %s: %w", upgradePath, err)
		}
	}
	connectionSettings := localnet.ConnectionSettings{
		Genesis:      genesis,
		Upgrade:      upgrade,
		BootstrapIDs: bootstrapIDs,
		BootstrapIPs: bootstrapIPs,
	}
	if len(stakingSignerKeyPaths) != len(stakingCertKeyPaths) || len(stakingSignerKeyPaths) != len(stakingTLSKeyPaths) {
		return fmt.Errorf("staking key inputs must be for the same number of nodes")
	}
	nodeSettingsLen := max(len(stakingSignerKeyPaths), len(httpPorts), len(stakingPorts))
	nodeSettings := make([]localnet.NodeSetting, nodeSettingsLen)
	for i := range nodeSettingsLen {
		nodeSetting := localnet.NodeSetting{}
		if i < len(stakingSignerKeyPaths) {
			stakingSignerKey, err := os.ReadFile(stakingSignerKeyPaths[i])
			if err != nil {
				return fmt.Errorf("could not read staking signer key at %s: %w", stakingSignerKeyPaths[i], err)
			}
			stakingCertKey, err := os.ReadFile(stakingCertKeyPaths[i])
			if err != nil {
				return fmt.Errorf("could not read staking cert key at %s: %w", stakingCertKeyPaths[i], err)
			}
			stakingTLSKey, err := os.ReadFile(stakingTLSKeyPaths[i])
			if err != nil {
				return fmt.Errorf("could not read staking TLS key at %s: %w", stakingTLSKeyPaths[i], err)
			}
			nodeSetting.StakingSignerKey = stakingSignerKey
			nodeSetting.StakingCertKey = stakingCertKey
			nodeSetting.StakingTLSKey = stakingTLSKey
		}
		if i < len(httpPorts) {
			nodeSetting.HTTPPort = uint64(httpPorts[i])
		}
		if i < len(stakingPorts) {
			nodeSetting.StakingPort = uint64(stakingPorts[i])
		}
		nodeSettings[i] = nodeSetting
	}

	network := models.UndefinedNetwork
	if !localnet.LocalClusterExists(app, clusterName) {
		network, err = networkoptions.GetNetworkFromCmdLineFlags(
			app,
			"",
			globalNetworkFlags,
			false,
			true,
			networkoptions.DefaultSupportedNetworkOptions,
			"",
		)
		if err != nil {
			return err
		}
	}

	if useCustomLuxgoVersion != "" {
		// Check version compatibility before proceeding
		if err = dependencies.CheckVersionIsOverMin(app, constants.LuxdRepoName, network, useCustomLuxgoVersion); err != nil {
			return err
		}
		latestLuxdPreReleaseVersion = false
		latestLuxdReleaseVersion = false
	}
	luxdVersionSetting := dependencies.LuxdVersionSettings{
		UseCustomLuxgoVersion:           useCustomLuxgoVersion,
		UseLatestLuxgoPreReleaseVersion: latestLuxdPreReleaseVersion,
		UseLatestLuxgoReleaseVersion:    latestLuxdReleaseVersion,
	}
	nodeConfig := make(map[string]interface{})
	if nodeConfigPath != "" {
		var err error
		nodeConfig = make(map[string]interface{})
		err = utils.ReadJSON(nodeConfigPath, &nodeConfig)
		if err != nil {
			return err
		}
	}
	if partialSync {
		nodeConfig[config.PartialSyncPrimaryNetworkKey] = true
	}
	return node.StartLocalNode(
		app,
		clusterName,
		luxdBinaryPath,
		numNodes,
		nodeConfig,
		connectionSettings,
		nodeSettings,
		luxdVersionSetting,
		network,
	)
}

func localStopNode(_ *cobra.Command, args []string) error {
	if len(args) == 1 {
		clusterName := args[0]

		// want to be able to stop clusters even if they are only partially operative
		if running, err := localnet.LocalClusterIsPartiallyRunning(app, clusterName); err != nil {
			return err
		} else if !running {
			ux.Logger.PrintToUser("cluster is not running")
		} else {
			if err := localnet.LocalClusterStop(app, clusterName); err != nil {
				return err
			}
			ux.Logger.GreenCheckmarkToUser("luxd stopped")
		}
		return nil
	}
	clusterNames, err := localnet.GetRunningLocalClusters(app)
	if err != nil {
		return err
	}
	if len(clusterNames) == 0 {
		ux.Logger.PrintToUser("no clusters to stop")
		return nil
	}
	for _, clusterName := range clusterNames {
		if err := localnet.LocalClusterStop(app, clusterName); err != nil {
			return err
		}
	}
	ux.Logger.GreenCheckmarkToUser("luxd stopped")
	return nil
}

func localDestroyNode(_ *cobra.Command, args []string) error {
	clusterName := args[0]
	if err := localnet.LocalClusterRemove(app, clusterName); err != nil {
		return err
	}
	ux.Logger.GreenCheckmarkToUser("Local node %s cleaned up.", clusterName)
	return nil
}

func localTrack(_ *cobra.Command, args []string) error {
	clusterName := args[0]
	blockchainName := args[1]
	return localnet.LocalClusterTrackSubnet(
		app,
		ux.Logger.PrintToUser,
		clusterName,
		blockchainName,
	)
}

func localStatus(_ *cobra.Command, args []string) error {
	clusterName := ""
	if len(args) > 0 {
		clusterName = args[0]
	}
	if blockchainName != "" && clusterName == "" {
		return fmt.Errorf("--blockchain flag is only supported if clusterName is specified")
	}
	return node.LocalStatus(app, clusterName, blockchainName)
}

func notImplementedForLocal(what string) error {
	ux.Logger.PrintToUser("Unsupported cmd: %s is not supported by local clusters", luxlog.LightBlue.Wrap(what))
	return nil
}

type NodeLocalValidateFlags struct {
	RPC         string
	SigAggFlags flags.SignatureAggregatorFlags
}

func newLocalValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate [clusterName]",
		Short: "Validate a specified L1 with an Lux Node set up on local machine (PoS only)",
		Long: `Use Lux Node set up on local machine to set up specified L1 by providing the
RPC URL of the L1. 

This command can only be used to validate Proof of Stake L1.`,
		RunE:    localValidate,
		PreRunE: cobra.ExactArgs(1),
	}
	flags.AddRPCFlagToCmd(cmd, app, &localValidateFlags.RPC)
	sigAggGroup := flags.AddSignatureAggregatorFlagsToCmd(cmd, &localValidateFlags.SigAggFlags)
	cmd.Flags().StringVar(&blockchainName, "l1", "", "specify the blockchain the node is syncing with")
	cmd.Flags().StringVar(&blockchainName, "blockchain", "", "specify the blockchain the node is syncing with")
	cmd.Flags().Uint64Var(&stakeAmount, "stake-amount", 0, "amount of tokens to stake")
	cmd.Flags().Float64Var(&balanceLUX, "balance", 0, "amount of LUX to increase validator's balance by")
	cmd.Flags().Uint16Var(&delegationFee, "delegation-fee", 100, "delegation fee (in bips)")
	cmd.Flags().StringVar(&remainingBalanceOwnerAddr, "remaining-balance-owner", "", "P-Chain address that will receive any leftover LUX from the validator when it is removed from Subnet")
	cmd.Flags().StringVar(&disableOwnerAddr, "disable-owner", "", "P-Chain address that will able to disable the validator with a P-Chain transaction")
	cmd.Flags().Uint64Var(&minimumStakeDuration, "minimum-stake-duration", constants.PoSL1MinimumStakeDurationSeconds, "minimum stake duration (in seconds)")
	cmd.Flags().StringVar(&rewardsRecipientAddr, "rewards-recipient", "", "EVM address that will receive the validation rewards")
	cmd.Flags().StringVar(&validatorManagerAddress, "validator-manager-address", "", "validator manager address")
	cmd.Flags().BoolVar(&useACP99, "acp99", true, "use ACP99 contracts instead of v1.0.0 for validator managers")
	cmd.SetHelpFunc(flags.WithGroupedHelp([]flags.GroupedFlags{sigAggGroup}))
	return cmd
}

func localValidate(_ *cobra.Command, args []string) error {
	clusterName := ""
	if len(args) > 0 {
		clusterName = args[0]
	}

	if clusterName == "" {
		return fmt.Errorf("local cluster name cannot be empty")
	}

	if !localnet.LocalClusterExists(app, clusterName) {
		return fmt.Errorf("local cluster %q not found, please create it first using lux node local start %q", clusterName, clusterName)
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

	// Estimate fee based on transaction complexity
	// Base fee for validator registration + delegation fee component
	baseFee := uint64(1000000)    // 0.001 LUX base fee
	txSizeEstimate := uint64(500) // Estimated transaction size for validator registration
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

	// should take input prior to here for stake amount, delegation fee, and min stake duration
	if stakeAmount == 0 {
		stakeAmount, err = app.Prompt.CaptureUint64Compare(
			"Enter the amount of token to stake for each validator",
			[]prompts.Comparator{
				{
					Label: "Positive",
					Type:  prompts.MoreThan,
					Value: 0,
				},
			},
		)
		if err != nil {
			return err
		}
	}

	if localValidateFlags.RPC == "" {
		localValidateFlags.RPC, err = app.Prompt.CaptureURL("What is the RPC endpoint?")
		if err != nil {
			return err
		}
	}
	_, blockchainID, err := utils.SplitLuxgoRPCURI(localValidateFlags.RPC)
	// if there is error that means RPC URL did not contain blockchain in it
	// RPC might be in the format of something like https://etna.lux-dev.network
	// We will prompt for blockchainID in that case
	if err != nil {
		blockchainID, err = app.Prompt.CaptureString("What is the Blockchain ID of the L1?")
		if err != nil {
			return err
		}
	}

	if validatorManagerAddress == "" {
		validatorManagerAddressAddrFmt, err := app.Prompt.CaptureAddress("What is the address of the Validator Manager?")
		if err != nil {
			return err
		}
		validatorManagerAddress = validatorManagerAddressAddrFmt.String()
	}

	chainSpec := contract.ChainSpec{
		BlockchainID: blockchainID,
	}
	if balanceLUX == 0 {
		addresses := kc.Addresses().List()
		availableBalance := uint64(0)
		if len(addresses) > 0 {
			availableBalance, err = utils.GetNetworkBalance(addresses[0], network)
		}
		if err != nil {
			return err
		}
		prompt := "How many LUX do you want to each validator to start with?"
		balanceLUX, err = blockchain.PromptValidatorBalance(app, float64(availableBalance)/float64(units.Lux), prompt)
		if err != nil {
			return err
		}
	}
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
	remainingBalanceOwners := warpMessage.PChainOwner{
		Threshold: 1,
		Addresses: remainingBalanceOwnerAddrID,
	}

	if disableOwnerAddr == "" {
		disableOwnerAddr, err = prompts.PromptAddress(
			app.Prompt,
			"Enter P-Chain address that will be able to disable the validator (Example: P-...)",
		)
		if err != nil {
			return err
		}
	}
	disableOwnerAddrID, err := address.ParseToIDs([]string{disableOwnerAddr})
	if err != nil {
		return fmt.Errorf("failure parsing disable owner address %s: %w", disableOwnerAddr, err)
	}
	disableOwners := warpMessage.PChainOwner{
		Threshold: 1,
		Addresses: disableOwnerAddrID,
	}

	ux.Logger.PrintToUser("A private key is needed to pay for initialization of the validator's registration (Blockchain gas token).")
	payerPrivateKey, err := prompts.PromptPrivateKey(
		app.Prompt,
		"Enter private key to pay the fee",
	)
	if err != nil {
		return err
	}

	extraAggregatorPeers, err := blockchain.GetAggregatorExtraPeers(app, clusterName)
	if err != nil {
		return err
	}
	aggregatorLogger, err := signatureaggregator.NewSignatureAggregatorLogger(
		localValidateFlags.SigAggFlags.AggregatorLogLevel,
		localValidateFlags.SigAggFlags.AggregatorLogToStdout,
		app.GetAggregatorLogDir(clusterName),
	)
	if err != nil {
		return err
	}

	net, err := localnet.GetLocalCluster(app, clusterName)
	if err != nil {
		return err
	}

	if useACP99 {
		ux.Logger.PrintToUser("%s", luxlog.Yellow.Wrap("Validator Manager Protocol: V2"))
	} else {
		ux.Logger.PrintToUser("%s", luxlog.Yellow.Wrap("Validator Manager Protocol: v1.0.0"))
	}

	for _, node := range net.Nodes {
		if err = addAsValidator(
			network,
			node.URI,
			chainSpec,
			remainingBalanceOwners, disableOwners,
			extraAggregatorPeers,
			aggregatorLogger,
			kc,
			balance,
			payerPrivateKey,
			validatorManagerAddress,
			useACP99,
		); err != nil {
			return err
		}
	}

	ux.Logger.PrintToUser(" ")
	ux.Logger.GreenCheckmarkToUser("All validators are successfully added to the L1")
	return nil
}

func addAsValidator(
	network models.Network,
	nodeURI string,
	chainSpec contract.ChainSpec,
	remainingBalanceOwners, disableOwners warpMessage.PChainOwner,
	extraAggregatorPeers []info.Peer,
	aggregatorLogger luxlog.Logger,
	kc *keychain.Keychain,
	balance uint64,
	payerPrivateKey string,
	validatorManagerAddressStr string,
	useACP99 bool,
) error {
	// get node data
	nodeIDStr, publicKey, pop, err := utils.GetNodeID(nodeURI)
	if err != nil {
		return err
	}
	nodeID, err := ids.NodeIDFromString(nodeIDStr)
	if err != nil {
		return err
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

	ux.Logger.PrintToUser(" ")
	ux.Logger.PrintToUser("Adding validator %s", nodeIDStr)
	ux.Logger.PrintToUser(" ")

	blockchainTimestamp, err := blockchain.GetBlockchainTimestamp(network)
	if err != nil {
		return fmt.Errorf("failed to get blockchain timestamp: %w", err)
	}
	expiry := uint64(blockchainTimestamp.Add(constants.DefaultValidationIDExpiryDuration).Unix())

	blsInfo, err := blockchain.ConvertToBLSProofOfPossession(publicKey, pop)
	if err != nil {
		return fmt.Errorf("failure parsing BLS info: %w", err)
	}

	// Convert []info.Peer to []string
	extraAggregatorPeerStrs := make([]string, len(extraAggregatorPeers))
	for i, peer := range extraAggregatorPeers {
		extraAggregatorPeerStrs[i] = fmt.Sprintf("%s-%s", peer.ID.String(), peer.IP.String())
	}
	if err = signatureaggregator.UpdateSignatureAggregatorPeers(app, network, extraAggregatorPeerStrs, aggregatorLogger); err != nil {
		return err
	}
	aggregatorCtx, aggregatorCancel := sdkutils.GetTimedContext(constants.SignatureAggregatorTimeout)
	defer aggregatorCancel()
	signatureAggregatorEndpoint, err := signatureaggregator.GetSignatureAggregatorEndpoint(app, network)
	if err != nil {
		return err
	}

	_, validationID, _, err := validatormanager.InitValidatorRegistration(
		aggregatorCtx,
		app.Lux,
		network,
		localValidateFlags.RPC,
		chainSpec,
		false,
		"",
		payerPrivateKey,
		nodeID,
		blsInfo.PublicKey[:],
		expiry,
		remainingBalanceOwners,
		disableOwners,
		0,
		aggregatorLogger,
		true,
		delegationFee,
		time.Duration(minimumStakeDuration)*time.Second,
		crypto.HexToAddress(rewardsRecipientAddr),
		validatorManagerAddressStr,
		useACP99,
		"",
		signatureAggregatorEndpoint,
	)
	if err != nil {
		return err
	}
	ux.Logger.PrintToUser("ValidationID: %s", validationID)

	// Use the underlying node keychain from the CLI keychain
	deployer := subnet.NewPublicDeployer(app, false, kc.Keychain, network)
	// Register the L1 validator on P-Chain
	txID, _, err := deployer.RegisterL1Validator(balance, blsInfo, nil)
	if err != nil {
		if !strings.Contains(err.Error(), "warp message already issued for validationID") {
			return err
		}
		ux.Logger.PrintToUser("%s", luxlog.LightBlue.Wrap("The Validation ID was already registered on the P-Chain. Proceeding to the next step"))
	} else {
		ux.Logger.PrintToUser("RegisterL1ValidatorTx ID: %s", txID)
	}
	if err := blockchain.UpdatePChainHeight(
		"Waiting for P-Chain to update validator information ...",
	); err != nil {
		return err
	}

	aggregatorCtx, aggregatorCancel = sdkutils.GetTimedContext(constants.SignatureAggregatorTimeout)
	defer aggregatorCancel()
	if _, err := validatormanager.FinishValidatorRegistration(
		aggregatorCtx,
		app.Lux,
		network,
		localValidateFlags.RPC,
		chainSpec,
		false,
		"",
		payerPrivateKey,
		validationID,
		aggregatorLogger,
		validatorManagerAddress,
		signatureAggregatorEndpoint,
	); err != nil {
		return err
	}

	validatorWeight, err := getPoSValidatorWeight(network, chainSpec, nodeID)
	if err != nil {
		return err
	}

	ux.Logger.PrintToUser("  NodeID: %s", nodeID)
	ux.Logger.PrintToUser("  Network: %s", network.Name())
	ux.Logger.PrintToUser("  Weight: %d", validatorWeight)
	ux.Logger.PrintToUser("  Balance: %.5f LUX", float64(balance)/float64(units.Lux))
	ux.Logger.GreenCheckmarkToUser("Validator %s successfully added to the L1", nodeIDStr)
	return nil
}

func getPoSValidatorWeight(network models.Network, chainSpec contract.ChainSpec, nodeID ids.NodeID) (uint64, error) {
	pClient := platformvm.NewClient(network.Endpoint())
	ctx, cancel := utils.GetAPIContext()
	defer cancel()
	subnetID, err := contract.GetSubnetID(
		app.Lux,
		network,
		chainSpec,
	)
	if err != nil {
		return 0, err
	}
	// Use GetCurrentValidators instead of GetValidatorsAt with ProposedHeight
	validatorsList, err := pClient.GetCurrentValidators(ctx, subnetID, nil)
	if err != nil {
		return 0, err
	}
	for _, validator := range validatorsList {
		if validator.NodeID == nodeID {
			return validator.Weight, nil
		}
	}
	return 0, fmt.Errorf("validator %s not found", nodeID)
}
