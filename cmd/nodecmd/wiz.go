// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package nodecmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/luxfi/cli/cmd/blockchaincmd"
	"github.com/luxfi/cli/cmd/interchaincmd/messengercmd"
	"github.com/luxfi/cli/pkg/ansible"
	awsAPI "github.com/luxfi/cli/pkg/cloud/aws"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/sdk/contract"
	"github.com/luxfi/cli/pkg/docker"
	"github.com/luxfi/cli/pkg/interchain"
	"github.com/luxfi/cli/pkg/interchain/relayer"
	"github.com/luxfi/cli/pkg/metrics"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/cli/pkg/networkoptions"
	"github.com/luxfi/cli/pkg/node"
	"github.com/luxfi/cli/pkg/ssh"
	"github.com/luxfi/cli/pkg/subnet"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/ids"
	luxlog "github.com/luxfi/log"
	"github.com/luxfi/node/utils/set"
	"github.com/luxfi/node/vms/platformvm/status"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
)

const (
	healthCheckPoolTime   = 60 * time.Second
	healthCheckTimeout    = 3 * time.Minute
	syncCheckPoolTime     = 10 * time.Second
	syncCheckTimeout      = 1 * time.Minute
	validateCheckPoolTime = 10 * time.Second
	validateCheckTimeout  = 1 * time.Minute
)

var (
	forceSubnetCreate                bool
	subnetGenesisFile                string
	useEvmSubnet                     bool
	useCustomSubnet                  bool
	evmVersion                       string
	evmChainID                       uint64
	evmToken                         string
	evmTestDefaults                  bool
	evmProductionDefaults            bool
	useLatestEvmReleasedVersion      bool
	useLatestEvmPreReleasedVersion   bool
	customVMRepoURL                  string
	customVMBranch                   string
	customVMBuildScript              string
	nodeConf                         string
	subnetConf                       string
	chainConf                        string
	validators                       []string
	customGrafanaDashboardPath       string
	warpReady                        bool
	runRelayer                       bool
	warpVersion                      string
	warpMessengerContractAddressPath string
	warpMessengerDeployerAddressPath string
	warpMessengerDeployerTxPath      string
	warpRegistryBydecodePath         string
	deployWarpMessenger              bool
	deployWarpRegistry               bool
	replaceKeyPair                   bool
)

func newWizCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wiz [clusterName] [subnetName]",
		Short: "(ALPHA Warning) Creates a devnet together with a fully validated subnet into it.",
		Long: `(ALPHA Warning) This command is currently in experimental mode.

The node wiz command creates a devnet and deploys, sync and validate a subnet into it. It creates the subnet if so needed.
`,
		Args:              cobrautils.RangeArgs(1, 2),
		RunE:              wiz,
		PersistentPostRun: handlePostRun,
	}
	cmd.Flags().BoolVar(&useStaticIP, "use-static-ip", true, "attach static Public IP on cloud servers")
	cmd.Flags().BoolVar(&useAWS, "aws", false, "create node/s in AWS cloud")
	cmd.Flags().BoolVar(&useGCP, "gcp", false, "create node/s in GCP cloud")
	cmd.Flags().StringSliceVar(&cmdLineRegion, "region", []string{}, "create node/s in given region(s). Use comma to separate multiple regions")
	cmd.Flags().BoolVar(&authorizeAccess, "authorize-access", false, "authorize CLI to create cloud resources")
	cmd.Flags().IntSliceVar(&numValidatorsNodes, "num-validators", []int{}, "number of nodes to create per region(s). Use comma to separate multiple numbers for each region in the same order as --region flag")
	cmd.Flags().StringVar(&nodeType, "node-type", "", "cloud instance type. Use 'default' to use recommended default instance type")
	cmd.Flags().StringVar(&cmdLineGCPCredentialsPath, "gcp-credentials", "", "use given GCP credentials")
	cmd.Flags().StringVar(&cmdLineGCPProjectName, "gcp-project", "", "use given GCP project")
	cmd.Flags().StringVar(&cmdLineAlternativeKeyPairName, "alternative-key-pair-name", "", "key pair name to use if default one generates conflicts")
	cmd.Flags().StringVar(&awsProfile, "aws-profile", constants.AWSDefaultCredential, "aws profile to use")
	cmd.Flags().BoolVar(&defaultValidatorParams, "default-validator-params", false, "use default weight/start/duration params for subnet validator")
	cmd.Flags().BoolVar(&forceSubnetCreate, "force-subnet-create", false, "overwrite the existing subnet configuration if one exists")
	cmd.Flags().StringVar(&subnetGenesisFile, "subnet-genesis", "", "file path of the subnet genesis")
	cmd.Flags().BoolVar(&warpReady, "teleporter", false, "generate an warp-ready vm")
	cmd.Flags().BoolVar(&warpReady, "warp", false, "generate an warp-ready vm")
	cmd.Flags().BoolVar(&runRelayer, "relayer", false, "run AWM relayer when deploying the vm")
	cmd.Flags().BoolVar(&useEvmSubnet, "evm-subnet", false, "use Subnet-EVM as the subnet virtual machine")
	cmd.Flags().BoolVar(&useCustomSubnet, "custom-subnet", false, "use a custom VM as the subnet virtual machine")
	cmd.Flags().StringVar(&evmVersion, "evm-version", "", "version of Subnet-EVM to use")
	cmd.Flags().Uint64Var(&evmChainID, "evm-chain-id", 0, "chain ID to use with Subnet-EVM")
	cmd.Flags().StringVar(&evmToken, "evm-token", "", "token name to use with Subnet-EVM")
	cmd.Flags().BoolVar(&evmProductionDefaults, "evm-defaults", false, "use default production settings with Subnet-EVM")
	cmd.Flags().BoolVar(&evmProductionDefaults, "evm-production-defaults", false, "use default production settings for your blockchain")
	cmd.Flags().BoolVar(&evmTestDefaults, "evm-test-defaults", false, "use default test settings for your blockchain")
	cmd.Flags().BoolVar(&useLatestEvmReleasedVersion, "latest-evm-version", false, "use latest Subnet-EVM released version")
	cmd.Flags().BoolVar(&useLatestEvmPreReleasedVersion, "latest-pre-released-evm-version", false, "use latest Subnet-EVM pre-released version")
	cmd.Flags().StringVar(&customVMRepoURL, "custom-vm-repo-url", "", "custom vm repository url")
	cmd.Flags().StringVar(&customVMBranch, "custom-vm-branch", "", "custom vm branch or commit")
	cmd.Flags().StringVar(&customVMBuildScript, "custom-vm-build-script", "", "custom vm build-script")
	cmd.Flags().StringVar(&customGrafanaDashboardPath, "add-grafana-dashboard", "", "path to additional grafana dashboard json file")
	cmd.Flags().StringVar(&nodeConf, "node-config", "", "path to luxd node configuration for subnet")
	cmd.Flags().StringVar(&subnetConf, "subnet-config", "", "path to the subnet configuration for subnet")
	cmd.Flags().StringVar(&chainConf, "chain-config", "", "path to the chain configuration for subnet")
	cmd.Flags().BoolVar(&useSSHAgent, "use-ssh-agent", false, "use ssh agent for ssh")
	cmd.Flags().StringVar(&sshIdentity, "ssh-agent-identity", "", "use given ssh identity(only for ssh agent). If not set, default will be used.")
	cmd.Flags().BoolVar(&useLatestLuxgoReleaseVersion, "latest-luxd-version", false, "install latest luxd release version on node/s")
	cmd.Flags().BoolVar(&useLatestLuxgoPreReleaseVersion, "latest-luxd-pre-release-version", false, "install latest luxd pre-release version on node/s")
	cmd.Flags().StringVar(&useCustomLuxgoVersion, "custom-luxd-version", "", "install given luxd version on node/s")
	cmd.Flags().StringSliceVar(&validators, "validators", []string{}, "deploy subnet into given comma separated list of validators. defaults to all cluster nodes")
	cmd.Flags().BoolVar(&addMonitoring, enableMonitoringFlag, false, " set up Prometheus monitoring for created nodes. Please note that this option creates a separate monitoring instance and incures additional cost")
	cmd.Flags().IntSliceVar(&numAPINodes, "num-apis", []int{}, "number of API nodes(nodes without stake) to create in the new Devnet")
	cmd.Flags().IntVar(&iops, "aws-volume-iops", constants.AWSGP3DefaultIOPS, "AWS iops (for gp3, io1, and io2 volume types only)")
	cmd.Flags().IntVar(&throughput, "aws-volume-throughput", constants.AWSGP3DefaultThroughput, "AWS throughput in MiB/s (for gp3 volume type only)")
	cmd.Flags().StringVar(&volumeType, "aws-volume-type", "gp3", "AWS volume type")
	cmd.Flags().IntVar(&volumeSize, "aws-volume-size", constants.CloudServerStorageSize, "AWS volume size in GB")
	cmd.Flags().StringVar(&grafanaPkg, "grafana-pkg", "", "use grafana pkg instead of apt repo(by default), for example https://dl.grafana.com/oss/release/grafana_10.4.1_amd64.deb")
	cmd.Flags().StringVar(&warpVersion, "teleporter-version", "latest", "warp version to deploy")
	cmd.Flags().StringVar(&warpMessengerContractAddressPath, "teleporter-messenger-contract-address-path", "", "path to an warp messenger contract address file")
	cmd.Flags().StringVar(&warpMessengerDeployerAddressPath, "teleporter-messenger-deployer-address-path", "", "path to an warp messenger deployer address file")
	cmd.Flags().StringVar(&warpMessengerDeployerTxPath, "teleporter-messenger-deployer-tx-path", "", "path to an warp messenger deployer tx file")
	cmd.Flags().StringVar(&warpRegistryBydecodePath, "teleporter-registry-bytecode-path", "", "path to an warp registry bytecode file")
	cmd.Flags().BoolVar(&deployWarpMessenger, "deploy-teleporter-messenger", true, "deploy Interchain Messenger")
	cmd.Flags().BoolVar(&deployWarpRegistry, "deploy-teleporter-registry", true, "deploy Interchain Registry")
	cmd.Flags().StringVar(&warpVersion, "warp-version", "latest", "warp version to deploy")
	cmd.Flags().StringVar(&warpMessengerContractAddressPath, "warp-messenger-contract-address-path", "", "path to an warp messenger contract address file")
	cmd.Flags().StringVar(&warpMessengerDeployerAddressPath, "warp-messenger-deployer-address-path", "", "path to an warp messenger deployer address file")
	cmd.Flags().StringVar(&warpMessengerDeployerTxPath, "warp-messenger-deployer-tx-path", "", "path to an warp messenger deployer tx file")
	cmd.Flags().StringVar(&warpRegistryBydecodePath, "warp-registry-bytecode-path", "", "path to an warp registry bytecode file")
	cmd.Flags().BoolVar(&deployWarpMessenger, "deploy-warp-messenger", true, "deploy Interchain Messenger")
	cmd.Flags().BoolVar(&deployWarpRegistry, "deploy-warp-registry", true, "deploy Interchain Registry")
	cmd.Flags().BoolVar(&replaceKeyPair, "auto-replace-keypair", false, "automatically replaces key pair to access node if previous key pair is not found")
	cmd.Flags().BoolVar(&publicHTTPPortAccess, "public-http-port", false, "allow public access to luxd HTTP port")
	cmd.Flags().StringSliceVar(&subnetAliases, "subnet-aliases", nil, "additional subnet aliases to be used for RPC calls in addition to subnet blockchain name")
	return cmd
}

func wiz(cmd *cobra.Command, args []string) error {
	clusterName := args[0]
	subnetName := ""
	if len(args) > 1 {
		subnetName = args[1]
	}
	c := make(chan os.Signal, 1)
	// Destroy cluster if user calls ctrl ^ c
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		for range c {
			if err := CallDestroyNode(clusterName); err != nil {
				ux.Logger.RedXToUser("Unable to delete cluster %s due to %s", clusterName, err)
				ux.Logger.RedXToUser("Please try again by calling lux node destroy %s", clusterName)
			}
			os.Exit(0)
		}
	}()
	clusterAlreadyExists, err := app.ClusterExists(clusterName)
	if err != nil {
		return err
	}
	if clusterAlreadyExists {
		if err := checkClusterIsADevnet(clusterName); err != nil {
			return err
		}
	}
	if clusterAlreadyExists && subnetName == "" {
		return fmt.Errorf("expecting to add subnet to existing cluster but no subnet-name was provided")
	}
	if subnetName != "" && (!app.SidecarExists(subnetName) || forceSubnetCreate) {
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser(luxlog.Green.Wrap("Creating the subnet"))
		ux.Logger.PrintToUser("")
		if err := blockchaincmd.CallCreate(
			cmd,
			subnetName,
			forceSubnetCreate,
			subnetGenesisFile,
			useEvmSubnet,
			useCustomSubnet,
			evmVersion,
			evmChainID,
			evmToken,
			evmProductionDefaults,
			evmTestDefaults,
			useLatestEvmReleasedVersion,
			useLatestEvmPreReleasedVersion,
			customVMRepoURL,
			customVMBranch,
			customVMBuildScript,
		); err != nil {
			return err
		}
		if chainConf != "" || subnetConf != "" || nodeConf != "" {
			if err := blockchaincmd.CallConfigure(
				cmd,
				subnetName,
				chainConf,
				subnetConf,
				nodeConf,
			); err != nil {
				return err
			}
		}
	}

	if !clusterAlreadyExists {
		globalNetworkFlags.UseDevnet = true
		if len(useCustomLuxgoVersion) == 0 && !useLatestLuxgoReleaseVersion && !useLatestLuxgoPreReleaseVersion {
			useLuxgoVersionFromSubnet = subnetName
		}
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser(luxlog.Green.Wrap("Creating the devnet..."))
		ux.Logger.PrintToUser("")
		// wizSubnet is used to get more metrics sent from node create command on whether if vm is custom or subnetEVM
		wizSubnet = subnetName
		if err := createNodes(cmd, []string{clusterName}); err != nil {
			return err
		}
	} else {
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser(luxlog.Green.Wrap("Adding subnet into existing devnet %s..."), clusterName)
	}

	// check all validators are found
	if len(validators) != 0 {
		allHosts, err := ansible.GetInventoryFromAnsibleInventoryFile(app.GetAnsibleInventoryDirPath(clusterName))
		if err != nil {
			return err
		}
		clustersConfig, err := app.GetClustersConfig()
		if err != nil {
			return err
		}
		clusters, _ := clustersConfig["Clusters"].(map[string]interface{})
		_, ok := clusters[clusterName].(map[string]interface{})
		if !ok {
			return fmt.Errorf("cluster %s does not exist", clusterName)
		}
		// Filter to get only validator hosts (exclude API nodes)
		hosts := allHosts
		_, err = filterHosts(hosts, validators)
		if err != nil {
			return err
		}
	}

	if err := node.WaitForHealthyCluster(app, clusterName, healthCheckTimeout, healthCheckPoolTime); err != nil {
		return err
	}

	if subnetName == "" {
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser(luxlog.Green.Wrap("Devnet %s has been created!"), clusterName)
		return nil
	}

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser(luxlog.Green.Wrap("Checking subnet compatibility"))
	ux.Logger.PrintToUser("")
	if err := checkRPCCompatibility(clusterName, subnetName); err != nil {
		return err
	}

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser(luxlog.Green.Wrap("Creating the blockchain"))
	ux.Logger.PrintToUser("")
	avoidChecks = true
	if err := deploySubnet(cmd, []string{clusterName, subnetName}); err != nil {
		return err
	}

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser(luxlog.Green.Wrap("Adding nodes as subnet validators"))
	ux.Logger.PrintToUser("")
	avoidSubnetValidationChecks = true
	useEwoq = true
	if err := validateSubnet(cmd, []string{clusterName, subnetName}); err != nil {
		return err
	}

	network, err := app.GetClusterNetwork(clusterName)
	if err != nil {
		return err
	}
	sc, err := app.LoadSidecar(subnetName)
	if err != nil {
		return err
	}
	subnetID := sc.Networks[network.Name()].SubnetID
	if subnetID == ids.Empty {
		return constants.ErrNoSubnetID
	}

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser(luxlog.Green.Wrap("Waiting for nodes to be validating the subnet"))
	ux.Logger.PrintToUser("")
	if err := waitForSubnetValidators(network, clusterName, subnetID, validateCheckTimeout, validateCheckPoolTime); err != nil {
		return err
	}

	isEVMGenesis, _, err := app.HasSubnetEVMGenesis(subnetName)
	if err != nil {
		return err
	}

	var awmRelayerHost *models.Host
	if sc.TeleporterReady && sc.RunRelayer && isEVMGenesis {
		// get or set AWM Relayer host and configure/stop service
		awmRelayerHost, err = node.GetWarpRelayerHost(app, clusterName)
		if err != nil {
			return err
		}
		if awmRelayerHost == nil {
			awmRelayerHost, err = chooseWarpRelayerHost(clusterName)
			if err != nil {
				return err
			}
			// get awm-relayer latest version
			relayerVersion, err := relayer.GetLatestRelayerReleaseVersion(app)
			if err != nil {
				return err
			}
			if err := setWarpRelayerHost(awmRelayerHost, relayerVersion); err != nil {
				return err
			}
			if err := setWarpRelayerSecurityGroupRule(clusterName, awmRelayerHost); err != nil {
				return err
			}
		} else {
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser(luxlog.Green.Wrap("Stopping AWM Relayer Service"))
			if err := ssh.RunSSHStopWarpRelayerService(awmRelayerHost); err != nil {
				return err
			}
		}
	}

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser(luxlog.Green.Wrap("Setting the nodes as subnet trackers"))
	ux.Logger.PrintToUser("")
	if err := syncSubnet(cmd, []string{clusterName, subnetName}); err != nil {
		return err
	}
	if err := node.WaitForHealthyCluster(app, clusterName, healthCheckTimeout, healthCheckPoolTime); err != nil {
		return err
	}
	blockchainID := sc.Networks[network.Name()].BlockchainID
	if blockchainID == ids.Empty {
		return constants.ErrNoBlockchainID
	}
	// update logging
	if addMonitoring {
		// set up subnet logs in Loki
		if err = setUpSubnetLogging(clusterName, subnetName); err != nil {
			return err
		}
	}
	if err := waitForClusterSubnetStatus(clusterName, subnetName, blockchainID, status.Validating, validateCheckTimeout, validateCheckPoolTime); err != nil {
		return err
	}

	if b, err := hasWarpDeploys(clusterName); err != nil {
		return err
	} else if b {
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser(luxlog.Green.Wrap("Updating Proposer VMs"))
		ux.Logger.PrintToUser("")
		if err := updateProposerVMs(network); err != nil {
			// not going to consider fatal, as warp messaging will be working fine after a failed first msg
			ux.Logger.PrintToUser(luxlog.Yellow.Wrap("failure setting proposer: %s"), err)
		}
	}

	if sc.TeleporterReady && sc.RunRelayer && isEVMGenesis {
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser(luxlog.Green.Wrap("Setting up Warp on subnet"))
		ux.Logger.PrintToUser("")
		flags := messengercmd.DeployFlags{
			ChainFlags: contract.ChainSpec{
				BlockchainName: subnetName,
			},
			PrivateKeyFlags: contract.PrivateKeyFlags{
				KeyName: constants.WarpKeyName,
			},
			Network: networkoptions.NetworkFlags{
				ClusterName: clusterName,
			},
			DeployMessenger:              deployWarpMessenger,
			DeployRegistry:               deployWarpRegistry,
			ForceRegistryDeploy:          true,
			Version:                      warpVersion,
			MessengerContractAddressPath: warpMessengerContractAddressPath,
			MessengerDeployerAddressPath: warpMessengerDeployerAddressPath,
			MessengerDeployerTxPath:      warpMessengerDeployerTxPath,
			RegistryBydecodePath:         warpRegistryBydecodePath,
			IncludeCChain:                true,
		}
		if err := messengercmd.CallDeploy([]string{}, flags, models.UndefinedNetwork); err != nil {
			return err
		}
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser(luxlog.Green.Wrap("Starting AWM Relayer Service"))
		ux.Logger.PrintToUser("")
		if err := updateWarpRelayerFunds(network, sc, blockchainID); err != nil {
			return err
		}
		if err := updateWarpRelayerHostConfig(network, awmRelayerHost, subnetName); err != nil {
			return err
		}
	}

	ux.Logger.PrintToUser("")
	if clusterAlreadyExists {
		ux.Logger.PrintToUser(luxlog.Green.Wrap("Devnet %s is now validating subnet %s"), clusterName, subnetName)
	} else {
		ux.Logger.PrintToUser(luxlog.Green.Wrap("Devnet %s is successfully created and is now validating subnet %s!"), clusterName, subnetName)
	}
	ux.Logger.PrintToUser("")

	ux.Logger.PrintToUser(luxlog.Green.Wrap("Subnet %s RPC URL: %s"), subnetName, network.BlockchainEndpoint(blockchainID.String()))
	ux.Logger.PrintToUser("")

	if addMonitoring {
		if customGrafanaDashboardPath != "" {
			if err = addCustomDashboard(clusterName, subnetName); err != nil {
				return err
			}
		}
		// no need to check for error, as it's ok not to have monitoring host
		monitoringHosts, _ := ansible.GetInventoryFromAnsibleInventoryFile(app.GetMonitoringInventoryDir(clusterName))
		if len(monitoringHosts) > 0 {
			getMonitoringHint(monitoringHosts[0].IP)
		}
	}

	if err := deployClusterYAMLFile(clusterName, subnetName); err != nil {
		return err
	}
	sendNodeWizMetrics()
	return nil
}

func hasWarpDeploys(
	clusterName string,
) (bool, error) {
	clusterConfig, err := app.GetClusterConfig(clusterName)
	if err != nil {
		return false, err
	}
	subnets, _ := clusterConfig["Subnets"].([]interface{})
	for _, subnet := range subnets {
		deployedSubnetName, _ := subnet.(string)
		deployedSubnetIsEVMGenesis, _, err := app.HasSubnetEVMGenesis(deployedSubnetName)
		if err != nil {
			return false, err
		}
		deployedSubnetSc, err := app.LoadSidecar(deployedSubnetName)
		if err != nil {
			return false, err
		}
		if deployedSubnetSc.TeleporterReady && deployedSubnetIsEVMGenesis {
			return true, nil
		}
	}
	return false, nil
}

func updateProposerVMs(
	network models.Network,
) error {
	clusterConfig, err := app.GetClusterConfig(network.ClusterName())
	if err != nil {
		return err
	}
	subnets, _ := clusterConfig["Subnets"].([]interface{})
	for _, subnet := range subnets {
		deployedSubnetName, _ := subnet.(string)
		deployedSubnetIsEVMGenesis, _, err := app.HasSubnetEVMGenesis(deployedSubnetName)
		if err != nil {
			return err
		}
		deployedSubnetSc, err := app.LoadSidecar(deployedSubnetName)
		if err != nil {
			return err
		}
		if deployedSubnetSc.TeleporterReady && deployedSubnetIsEVMGenesis {
			ux.Logger.PrintToUser("Updating proposerVM on %s", deployedSubnetName)
			blockchainID := deployedSubnetSc.Networks[network.Name()].BlockchainID
			if blockchainID == ids.Empty {
				return constants.ErrNoBlockchainID
			}
			if err := interchain.SetProposerVM(app, network, blockchainID.String(), deployedSubnetSc.TeleporterKey); err != nil {
				return err
			}
		}
	}
	ux.Logger.PrintToUser("Updating proposerVM on c-chain")
	return interchain.SetProposerVM(app, network, "C", "")
}

func setWarpRelayerHost(host *models.Host, relayerVersion string) error {
	cloudID := host.GetCloudID()
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("configuring AWM Relayer on host %s", cloudID)
	// Need to determine cluster name from host
	clusterName := ""
	clustersConfig, _ := app.GetClustersConfig()
	clusters, _ := clustersConfig["Clusters"].(map[string]interface{})
	for cName, cluster := range clusters {
		c, _ := cluster.(map[string]interface{})
		nodes, _ := c["Nodes"].([]interface{})
		for _, n := range nodes {
			if n == cloudID {
				clusterName = cName
				break
			}
		}
		if clusterName != "" {
			break
		}
	}
	nodeConfig, err := app.LoadClusterNodeConfig(clusterName, cloudID)
	if err != nil {
		return err
	}
	if err := ssh.ComposeSSHSetupWarpRelayer(host, relayerVersion); err != nil {
		return err
	}
	nodeConfig["IsWarpRelayer"] = true
	return app.CreateNodeCloudConfigFile(cloudID, nodeConfig)
}

func updateWarpRelayerHostConfig(network models.Network, host *models.Host, blockchainName string) error {
	ux.Logger.PrintToUser("setting AWM Relayer on host %s to relay blockchain %s", host.GetCloudID(), blockchainName)
	if err := addBlockchainToRelayerConf(network, host.GetCloudID(), blockchainName); err != nil {
		return err
	}
	if err := ssh.RunSSHUploadNodeWarpRelayerConfig(host, app.GetNodeInstanceDirPath(host.GetCloudID())); err != nil {
		return err
	}
	return ssh.RunSSHStartWarpRelayerService(host)
}

func chooseWarpRelayerHost(clusterName string) (*models.Host, error) {
	// first look up for separate monitoring host
	monitoringInventoryFile := app.GetMonitoringInventoryDir(clusterName)
	if utils.FileExists(monitoringInventoryFile) {
		monitoringHosts, err := ansible.GetInventoryFromAnsibleInventoryFile(monitoringInventoryFile)
		if err != nil {
			return nil, err
		}
		if len(monitoringHosts) > 0 {
			return monitoringHosts[0], nil
		}
	}
	// then look up for API nodes
	clusterConfig, err := app.GetClusterConfig(clusterName)
	if err != nil {
		return nil, err
	}
	apiNodes, _ := clusterConfig["APINodes"].([]interface{})
	if len(apiNodes) > 0 {
		apiNode, _ := apiNodes[0].(string)
		return node.GetHostWithCloudID(app, clusterName, apiNode)
	}
	// finally go for other hosts
	nodes, _ := clusterConfig["Nodes"].([]interface{})
	if len(nodes) > 0 {
		nodeID, _ := nodes[0].(string)
		return node.GetHostWithCloudID(app, clusterName, nodeID)
	}
	return nil, fmt.Errorf("no hosts found on cluster")
}

func updateWarpRelayerFunds(network models.Network, sc models.Sidecar, blockchainID ids.ID) error {
	_, relayerAddress, _, err := relayer.GetDefaultRelayerKeyInfo(app, blockchainID.String())
	if err != nil {
		return err
	}
	// Use a placeholder key for now - proper key management would be needed
	keyAddress := "0x0000000000000000000000000000000000000000"
	chainSpec := map[string]interface{}{
		"blockchainID": blockchainID.String(),
		"amount":       0.1,
	}
	if err := relayer.FundRelayer(app, network, chainSpec, keyAddress, relayerAddress); err != nil {
		return err
	}
	// Fund from ewoq as well
	ewoqAddress := "0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"
	return relayer.FundRelayer(app, network, chainSpec, ewoqAddress, relayerAddress)
}

func deployClusterYAMLFile(clusterName, subnetName string) error {
	var separateHosts []*models.Host
	var err error
	loadTestInventoryDir := app.GetLoadTestInventoryDir(clusterName)
	if utils.FileExists(loadTestInventoryDir) {
		separateHosts, err = ansible.GetInventoryFromAnsibleInventoryFile(loadTestInventoryDir)
		if err != nil {
			return err
		}
	}
	subnetID, chainID, err := getDeployedSubnetInfo(clusterName, subnetName)
	if err != nil {
		return err
	}
	var externalHost *models.Host
	if len(separateHosts) > 0 {
		externalHost = separateHosts[0]
	}
	if err = createClusterYAMLFile(clusterName, subnetID, chainID, externalHost); err != nil {
		return err
	}
	ux.Logger.GreenCheckmarkToUser("Cluster information YAML file can be found at %s at local host", app.GetClusterYAMLFilePath(clusterName))
	// deploy YAML file to external host, if it exists
	if len(separateHosts) > 0 {
		if err = ssh.RunSSHCopyYAMLFile(separateHosts[0], app.GetClusterYAMLFilePath(clusterName)); err != nil {
			return err
		}
		ux.Logger.GreenCheckmarkToUser("Cluster information YAML file can be found at /home/ubuntu/%s at external host", constants.ClusterYAMLFileName)
	}
	return nil
}

func checkRPCCompatibility(
	clusterName string,
	subnetName string,
) error {
	_, err := app.GetClusterConfig(clusterName)
	if err != nil {
		return err
	}
	allHosts, err := ansible.GetInventoryFromAnsibleInventoryFile(app.GetAnsibleInventoryDirPath(clusterName))
	if err != nil {
		return err
	}
	// Filter to get only validator hosts (exclude API nodes)
	// For now, include all hosts
	hosts := allHosts
	if len(validators) != 0 {
		hosts, err = filterHosts(hosts, validators)
		if err != nil {
			return err
		}
	}
	defer node.DisconnectHosts(hosts)
	return node.CheckHostsAreRPCCompatible(app, hosts, subnetName)
}

func waitForSubnetValidators(
	network models.Network,
	clusterName string,
	subnetID ids.ID,
	timeout time.Duration,
	poolTime time.Duration,
) error {
	ux.Logger.PrintToUser("Waiting for node(s) in cluster %s to be validators of subnet ID %s...", clusterName, subnetID)
	_, err := app.GetClusterConfig(clusterName)
	if err != nil {
		return err
	}
	allHosts, err := ansible.GetInventoryFromAnsibleInventoryFile(app.GetAnsibleInventoryDirPath(clusterName))
	if err != nil {
		return err
	}
	// Filter to get only validator hosts (exclude API nodes)
	// For now, include all hosts
	hosts := allHosts
	if len(validators) != 0 {
		hosts, err = filterHosts(hosts, validators)
		if err != nil {
			return err
		}
	}
	defer node.DisconnectHosts(hosts)
	nodeIDMap, failedNodesMap := getNodeIDs(hosts)
	startTime := time.Now()
	for {
		failedNodes := []string{}
		for _, host := range hosts {
			nodeID, b := nodeIDMap[host.NodeID]
			if !b {
				err, b := failedNodesMap[host.NodeID]
				if !b {
					return fmt.Errorf("expected to found an error for non mapped node")
				}
				return err
			}
			isValidator, err := subnet.IsSubnetValidator(subnetID, nodeID, network)
			if err != nil {
				return err
			}
			if !isValidator {
				failedNodes = append(failedNodes, host.GetCloudID())
			}
		}
		if len(failedNodes) == 0 {
			ux.Logger.PrintToUser("Nodes validating subnet ID %s after %d seconds", subnetID, uint32(time.Since(startTime).Seconds()))
			return nil
		}
		if time.Since(startTime) > timeout {
			ux.Logger.PrintToUser("Nodes not validating subnet ID %sf", subnetID)
			for _, failedNode := range failedNodes {
				ux.Logger.PrintToUser("  " + failedNode)
			}
			ux.Logger.PrintToUser("")
			return fmt.Errorf("cluster %s not validating subnet ID %s after %d seconds", clusterName, subnetID, uint32(timeout.Seconds()))
		}
		time.Sleep(poolTime)
	}
}

func waitForClusterSubnetStatus(
	clusterName string,
	subnetName string,
	blockchainID ids.ID,
	targetStatus status.BlockchainStatus,
	timeout time.Duration,
	poolTime time.Duration,
) error {
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Waiting for node(s) in cluster %s to be %s subnet %s...", clusterName, strings.ToLower(targetStatus.String()), subnetName)
	clustersConfig, err := app.GetClustersConfig()
	if err != nil {
		return err
	}
	clusters, _ := clustersConfig["Clusters"].(map[string]interface{})
	_, ok := clusters[clusterName]
	if !ok {
		return fmt.Errorf("cluster %s does not exist", clusterName)
	}
	allHosts, err := ansible.GetInventoryFromAnsibleInventoryFile(app.GetAnsibleInventoryDirPath(clusterName))
	if err != nil {
		return err
	}
	// Filter to get only validator hosts (exclude API nodes)
	// For now, include all hosts
	hosts := allHosts
	if len(validators) != 0 {
		hosts, err = filterHosts(hosts, validators)
		if err != nil {
			return err
		}
	}
	defer node.DisconnectHosts(hosts)
	startTime := time.Now()
	for {
		wg := sync.WaitGroup{}
		wgResults := models.NodeResults{}
		for _, host := range hosts {
			wg.Add(1)
			go func(nodeResults *models.NodeResults, host *models.Host) {
				defer wg.Done()
				if syncstatus, err := ssh.RunSSHSubnetSyncStatus(host, blockchainID.String()); err != nil {
					nodeResults.AddResult(host.NodeID, nil, err)
					return
				} else {
					if subnetSyncStatus, err := parseSubnetSyncOutput(syncstatus); err != nil {
						nodeResults.AddResult(host.NodeID, nil, err)
						return
					} else {
						nodeResults.AddResult(host.NodeID, subnetSyncStatus, err)
					}
				}
			}(&wgResults, host)
		}
		wg.Wait()
		if wgResults.HasErrors() {
			return fmt.Errorf("failed to check sync status for node(s) %s", wgResults.GetErrorHostMap())
		}
		failedNodes := []string{}
		for host, subnetSyncStatus := range wgResults.GetResultMap() {
			if subnetSyncStatus != targetStatus.String() {
				failedNodes = append(failedNodes, host)
			}
		}
		if len(failedNodes) == 0 {
			ux.Logger.PrintToUser("Nodes %s %s after %d seconds", targetStatus.String(), subnetName, uint32(time.Since(startTime).Seconds()))
			return nil
		}
		if time.Since(startTime) > timeout {
			ux.Logger.PrintToUser("Nodes not %s %s", targetStatus.String(), subnetName)
			for _, failedNode := range failedNodes {
				ux.Logger.PrintToUser("  " + failedNode)
			}
			ux.Logger.PrintToUser("")
			return fmt.Errorf("cluster not %s subnet %s after %d seconds", strings.ToLower(targetStatus.String()), subnetName, uint32(timeout.Seconds()))
		}
		time.Sleep(poolTime)
	}
}

func checkClusterIsADevnet(clusterName string) error {
	exists, err := app.ClusterExists(clusterName)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("cluster %q does not exists", clusterName)
	}
	clustersConfig, err := app.GetClustersConfig()
	if err != nil {
		return err
	}
	clusters, _ := clustersConfig["Clusters"].(map[string]interface{})
	cluster, ok := clusters[clusterName].(map[string]interface{})
	if !ok {
		return fmt.Errorf("cluster %q does not exist", clusterName)
	}
	networkMap, _ := cluster["Network"].(map[string]interface{})
	if kind, _ := networkMap["Kind"].(string); kind != "Devnet" {
		return fmt.Errorf("cluster %q is not a Devnet", clusterName)
	}
	return nil
}

func filterHosts(hosts []*models.Host, nodes []string) ([]*models.Host, error) {
	indices := set.Set[int]{}
	for _, node := range nodes {
		added := false
		for i, host := range hosts {
			cloudID := host.GetCloudID()
			ip := host.IP
			nodeID, err := getNodeID(app.GetNodeInstanceDirPath(cloudID))
			if err != nil {
				return nil, err
			}
			if slices.Contains([]string{cloudID, ip, nodeID.String()}, node) {
				added = true
				indices.Add(i)
			}
		}
		if !added {
			return nil, fmt.Errorf("node %q not found", node)
		}
	}
	filteredHosts := []*models.Host{}
	for i, host := range hosts {
		if indices.Contains(i) {
			filteredHosts = append(filteredHosts, host)
		}
	}
	return filteredHosts, nil
}

func setWarpRelayerSecurityGroupRule(clusterName string, awmRelayerHost *models.Host) error {
	clusterConfig, err := app.GetClusterConfig(clusterName)
	if err != nil {
		return err
	}
	hasGCPNodes := false
	lastRegion := ""
	var ec2Svc *awsAPI.AwsCloud
	// Get cloud IDs from cluster nodes
	nodes, _ := clusterConfig["Nodes"].([]interface{})
	for _, node := range nodes {
		cloudID, _ := node.(string)
		nodeConfig, err := app.LoadClusterNodeConfig(clusterName, cloudID)
		if err != nil {
			return err
		}
		cloudService, _ := nodeConfig["CloudService"].(string)
		region, _ := nodeConfig["Region"].(string)
		switch {
		case cloudService == "" || cloudService == constants.AWSCloudService:
			if region != lastRegion {
				ec2Svc, err = awsAPI.NewAwsCloud(awsProfile, region)
				if err != nil {
					return err
				}
				lastRegion = region
			}
			securityGroup, _ := nodeConfig["SecurityGroup"].(string)
			securityGroupExists, sg, err := ec2Svc.CheckSecurityGroupExists(securityGroup)
			if err != nil {
				return err
			}
			if !securityGroupExists {
				return fmt.Errorf("security group %s doesn't exist in region %s", securityGroup, region)
			}
			if inSG := awsAPI.CheckIPInSg(&sg, awmRelayerHost.IP, constants.LuxdAPIPort); !inSG {
				if err = ec2Svc.AddSecurityGroupRule(
					*sg.GroupId,
					"ingress",
					"tcp",
					awmRelayerHost.IP+constants.IPAddressSuffix,
					constants.LuxdAPIPort,
				); err != nil {
					return err
				}
			}
		case cloudService == constants.GCPCloudService:
			hasGCPNodes = true
		default:
			return fmt.Errorf("cloud %s is not supported", cloudService)
		}
	}
	if hasGCPNodes {
		if err := setGCPWarpRelayerSecurityGroupRule(awmRelayerHost); err != nil {
			return err
		}
	}
	return nil
}

func sendNodeWizMetrics() {
	flags := make(map[string]string)
	populateSubnetVMMetrics(flags, wizSubnet)
	metrics.HandleTracking(app, flags, nil)
}

func populateSubnetVMMetrics(flags map[string]string, subnetName string) {
	sc, err := app.LoadSidecar(subnetName)
	if err == nil {
		switch sc.VM {
		case models.SubnetEvm:
			flags[constants.MetricsSubnetVM] = "Subnet-EVM"
		case models.CustomVM:
			flags[constants.MetricsSubnetVM] = "Custom-VM"
			flags[constants.MetricsCustomVMRepoURL] = sc.CustomVMRepoURL
			flags[constants.MetricsCustomVMBranch] = sc.CustomVMBranch
			flags[constants.MetricsCustomVMBuildScript] = sc.CustomVMBuildScript
		}
	}
	flags[constants.MetricsEnableMonitoring] = strconv.FormatBool(addMonitoring)
}

// setUPSubnetLogging sets up the subnet logging for the subnet
func setUpSubnetLogging(clusterName, subnetName string) error {
	_, chainID, err := getDeployedSubnetInfo(clusterName, subnetName)
	if err != nil {
		return err
	}
	wg := sync.WaitGroup{}
	wgResults := models.NodeResults{}
	spinSession := ux.NewUserSpinner()
	hosts, err := ansible.GetInventoryFromAnsibleInventoryFile(app.GetAnsibleInventoryDirPath(clusterName))
	if err != nil {
		return err
	}
	monitoringInventoryPath := app.GetMonitoringInventoryDir(clusterName)
	monitoringHosts, err := ansible.GetInventoryFromAnsibleInventoryFile(monitoringInventoryPath)
	if err != nil {
		return err
	}
	for _, host := range hosts {
		if !addMonitoring {
			continue
		}
		wg.Add(1)
		go func(host *models.Host) {
			defer wg.Done()
			spinner := spinSession.SpinToUser(utils.ScriptLog(host.NodeID, "Setup Subnet Logs"))
			cloudID := host.GetCloudID()
			nodeID, err := getNodeID(app.GetNodeInstanceDirPath(cloudID))
			if err != nil {
				wgResults.AddResult(host.NodeID, nil, err)
				ux.SpinFailWithError(spinner, "", err)
				return
			}
			if err = ssh.RunSSHSetupPromtailConfig(host, monitoringHosts[0].IP, constants.LuxdLokiPort, cloudID, nodeID.String(), chainID); err != nil {
				wgResults.AddResult(host.NodeID, nil, err)
				ux.SpinFailWithError(spinner, "", err)
				return
			}
			if err := docker.RestartDockerComposeService(host, utils.GetRemoteComposeFile(), "promtail", constants.SSHLongRunningScriptTimeout); err != nil {
				wgResults.AddResult(host.NodeID, nil, err)
				ux.SpinFailWithError(spinner, "", err)
				return
			}
			ux.SpinComplete(spinner)
		}(host)
	}
	wg.Wait()
	for _, node := range hosts {
		if wgResults.HasIDWithError(node.NodeID) {
			ux.Logger.RedXToUser("Node %s is ERROR with error: %s", node.NodeID, wgResults.GetErrorHostMap()[node.NodeID])
		}
	}
	spinSession.Stop()
	return nil
}

func addBlockchainToRelayerConf(network models.Network, cloudNodeID string, blockchainName string) error {
	_, _, _, err := relayer.GetDefaultRelayerKeyInfo(app, blockchainName)
	if err != nil {
		return err
	}

	configBasePath := app.GetNodeInstanceDirPath(cloudNodeID)

	configPath := app.GetWarpRelayerServiceConfigPath(configBasePath)
	if err := os.MkdirAll(filepath.Dir(configPath), constants.DefaultPerms755); err != nil {
		return err
	}
	ux.Logger.PrintToUser("updating configuration file %s", configPath)

	if err := relayer.CreateBaseRelayerConfigIfMissing(
		configPath,
		"info",
		app.GetWarpRelayerServiceStorageDir(),
		9090, // Default warp relayer metrics port
		network,
		true,
	); err != nil {
		return err
	}

	chainSpec := contract.ChainSpec{CChain: true}
	subnetID, err := contract.GetSubnetID(app.GetSDKApp(), network, chainSpec)
	if err != nil {
		return err
	}
	blockchainID, err := contract.GetBlockchainID(app.GetSDKApp(), network, chainSpec)
	if err != nil {
		return err
	}
	registryAddress, messengerAddress, err := contract.GetWarpInfo(app.GetSDKApp(), network, chainSpec, false, false, false)
	if err != nil {
		return err
	}
	_, _, err = contract.GetBlockchainEndpoints(app.GetSDKApp(), network, chainSpec, false, false)
	if err != nil {
		return err
	}

	// Use storage directory for relayer config
	storageDir := app.GetKeyDir()
	if err = relayer.AddSourceAndDestinationToRelayerConfig(
		app,
		storageDir,
		network,
		subnetID.String(),
		blockchainID.String(),
		registryAddress,
		messengerAddress,
		true, // isSource
	); err != nil {
		return err
	}

	chainSpec = contract.ChainSpec{BlockchainName: blockchainName}
	subnetID, err = contract.GetSubnetID(app.GetSDKApp(), network, chainSpec)
	if err != nil {
		return err
	}
	blockchainID, err = contract.GetBlockchainID(app.GetSDKApp(), network, chainSpec)
	if err != nil {
		return err
	}
	registryAddress, messengerAddress, err = contract.GetWarpInfo(app.GetSDKApp(), network, chainSpec, false, false, false)
	if err != nil {
		return err
	}
	_, _, err = contract.GetBlockchainEndpoints(app.GetSDKApp(), network, chainSpec, false, false)
	if err != nil {
		return err
	}

	// Reuse storage directory for relayer config
	if err = relayer.AddSourceAndDestinationToRelayerConfig(
		app,
		storageDir,
		network,
		subnetID.String(),
		blockchainID.String(),
		registryAddress,
		messengerAddress,
		true, // isSource
	); err != nil {
		return err
	}

	return nil
}
