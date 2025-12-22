// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package chain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/exp/maps"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/cli/pkg/constants"
	keychainwrapper "github.com/luxfi/cli/pkg/keychain"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/evm/core"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/params"
	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/netrunner/client"
	anrnetwork "github.com/luxfi/netrunner/network"
	"github.com/luxfi/netrunner/rpcpb"
	"github.com/luxfi/netrunner/server"
	anrutils "github.com/luxfi/netrunner/utils"
	"github.com/luxfi/node/utils/crypto/keychain"
	"github.com/luxfi/node/utils/storage"
	"github.com/luxfi/node/vms/components/lux"
	"github.com/luxfi/node/vms/components/verify"
	"github.com/luxfi/node/vms/platformvm"
	platformapi "github.com/luxfi/node/vms/platformvm/api"
	"github.com/luxfi/node/vms/platformvm/reward"
	"github.com/luxfi/node/vms/platformvm/signer"
	"github.com/luxfi/node/vms/platformvm/txs"
	"github.com/luxfi/node/vms/secp256k1fx"
	walletkeychain "github.com/luxfi/node/wallet/keychain"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/sdk/wallet/chain/c"
	"github.com/luxfi/sdk/wallet/primary"
	"go.uber.org/zap"
)

const (
	WriteReadReadPerms = 0o644
	// ChainHealthTimeout is the maximum time to wait for a newly deployed chain to become healthy
	ChainHealthTimeout = 30 * time.Second
)

// emptyEthKeychain is a minimal implementation of EthKeychain for cases where ETH keys are not needed
type emptyEthKeychain struct{}

func (e *emptyEthKeychain) GetEth(addr common.Address) (walletkeychain.Signer, bool) {
	return nil, false
}

func (e *emptyEthKeychain) EthAddresses() set.Set[common.Address] {
	return set.NewSet[common.Address](0)
}

type LocalDeployer struct {
	procChecker        binutils.ProcessChecker
	binChecker         binutils.BinaryChecker
	getClientFunc      getGRPCClientFunc
	binaryDownloader   binutils.PluginBinaryDownloader
	app                *application.Lux
	backendStartedHere bool
	setDefaultSnapshot setDefaultSnapshotFunc
	luxVersion         string
	vmBin              string
}

func NewLocalDeployer(app *application.Lux, luxVersion string, vmBin string) *LocalDeployer {
	return &LocalDeployer{
		procChecker:        binutils.NewProcessChecker(),
		binChecker:         binutils.NewBinaryChecker(),
		getClientFunc:      binutils.NewGRPCClient,
		binaryDownloader:   binutils.NewPluginBinaryDownloader(app),
		app:                app,
		setDefaultSnapshot: SetDefaultSnapshot,
		luxVersion:         luxVersion,
		vmBin:              vmBin,
	}
}

type getGRPCClientFunc func(...binutils.GRPCClientOpOption) (client.Client, error)

type setDefaultSnapshotFunc func(string, bool) error

// DeployToLocalNetwork deploys to an already running network.
// It does NOT start the network - use 'lux network start' first.
func (d *LocalDeployer) DeployToLocalNetwork(chain string, chainGenesis []byte, genesisPath string) (ids.ID, ids.ID, error) {
	// Connect to existing gRPC server - do NOT start one
	cli, err := d.getClientFunc()
	if err != nil {
		return ids.Empty, ids.Empty, fmt.Errorf("failed to connect to network. Is it running? Start with: lux network start --mainnet\nError: %w", err)
	}
	defer cli.Close()

	ctx := binutils.GetAsyncContext()
	_, err = WaitForHealthy(ctx, cli)
	if err != nil {
		if server.IsServerError(err, server.ErrNotBootstrapped) {
			return ids.Empty, ids.Empty, fmt.Errorf("network is not running. Start it first with: lux network start --mainnet")
		}
		return ids.Empty, ids.Empty, fmt.Errorf("network is unhealthy: %w", err)
	}

	return d.doDeploy(chain, chainGenesis, genesisPath)
}

func getAssetID(wallet primary.Wallet, ownerAddr ids.ShortID, tokenName string, tokenSymbol string, maxSupply uint64) (ids.ID, error) {
	xWallet := wallet.X()
	owner := &secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs: []ids.ShortID{
			ownerAddr,
		},
	}
	_, cancel := context.WithTimeout(context.Background(), constants.DefaultWalletCreationTimeout)
	subnetAssetTx, err := xWallet.IssueCreateAssetTx(
		tokenName,
		tokenSymbol,
		9, // denomination for UI purposes only in explorer
		map[uint32][]verify.State{
			0: {
				&secp256k1fx.TransferOutput{
					Amt:          maxSupply,
					OutputOwners: *owner,
				},
			},
		},
	)
	defer cancel()
	if err != nil {
		return ids.Empty, err
	}
	return subnetAssetTx.ID(), nil
}

func exportToPChain(wallet primary.Wallet, owner *secp256k1fx.OutputOwners, subnetAssetID ids.ID, maxSupply uint64) error {
	xWallet := wallet.X()
	_, cancel := context.WithTimeout(context.Background(), constants.DefaultWalletCreationTimeout)

	_, err := xWallet.IssueExportTx(
		ids.Empty,
		[]*lux.TransferableOutput{
			{
				Asset: lux.Asset{
					ID: subnetAssetID,
				},
				Out: &secp256k1fx.TransferOutput{
					Amt:          maxSupply,
					OutputOwners: *owner,
				},
			},
		},
	)
	defer cancel()
	return err
}

func importFromXChain(wallet primary.Wallet, owner *secp256k1fx.OutputOwners) error {
	pWallet := wallet.P()
	xChainID := ids.FromStringOrPanic("2oYMBNV4eNHyqk2fjjV5nVQLDbtmNJzq5s3qs3Lo6ftnC6FByM") // X-Chain ID
	_, cancel := context.WithTimeout(context.Background(), constants.DefaultWalletCreationTimeout)
	_, err := pWallet.IssueImportTx(
		xChainID,
		owner,
	)
	defer cancel()
	return err
}

func IssueTransformSubnetTx(
	elasticSubnetConfig models.ElasticSubnetConfig,
	kc keychain.Keychain,
	subnetID ids.ID,
	tokenName string,
	tokenSymbol string,
	maxSupply uint64,
) (ids.ID, ids.ID, error) {
	ctx := context.Background()
	api := constants.LocalAPIEndpoint
	// Create empty EthKeychain if kc doesn't implement it
	var ethKc c.EthKeychain
	if ekc, ok := kc.(c.EthKeychain); ok {
		ethKc = ekc
	} else {
		// Create a minimal EthKeychain implementation
		ethKc = &emptyEthKeychain{}
	}
	wallet, err := primary.MakeWallet(ctx, &primary.WalletConfig{
		URI:         api,
		LUXKeychain: keychainwrapper.WrapCryptoKeychain(kc),
		EthKeychain: ethKc,
	})
	if err != nil {
		return ids.Empty, ids.Empty, err
	}

	// Get the first address from the keychain for ownership
	addrs := kc.Addresses()
	if addrs.Len() == 0 {
		return ids.Empty, ids.Empty, errors.New("keychain has no addresses")
	}
	ownerAddr := addrs.List()[0]

	subnetAssetID, err := getAssetID(wallet, ownerAddr, tokenName, tokenSymbol, maxSupply)
	if err != nil {
		return ids.Empty, ids.Empty, err
	}
	owner := &secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs: []ids.ShortID{
			ownerAddr,
		},
	}
	err = exportToPChain(wallet, owner, subnetAssetID, maxSupply)
	if err != nil {
		return ids.Empty, ids.Empty, err
	}
	err = importFromXChain(wallet, owner)
	if err != nil {
		return ids.Empty, ids.Empty, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultConfirmTxTimeout)
	transformSubnetTxID, err := wallet.P().IssueTransformChainTx(elasticSubnetConfig.SubnetID, subnetAssetID,
		elasticSubnetConfig.InitialSupply, elasticSubnetConfig.MaxSupply, elasticSubnetConfig.MinConsumptionRate,
		elasticSubnetConfig.MaxConsumptionRate, elasticSubnetConfig.MinValidatorStake, elasticSubnetConfig.MaxValidatorStake,
		elasticSubnetConfig.MinStakeDuration, elasticSubnetConfig.MaxStakeDuration, elasticSubnetConfig.MinDelegationFee,
		elasticSubnetConfig.MinDelegatorStake, elasticSubnetConfig.MaxValidatorWeightFactor, elasticSubnetConfig.UptimeRequirement,
	)
	defer cancel()
	if err != nil {
		return ids.Empty, ids.Empty, err
	}
	return transformSubnetTxID.ID(), subnetAssetID, err
}

func IssueAddPermissionlessValidatorTx(
	kc keychain.Keychain,
	subnetID ids.ID,
	nodeID ids.NodeID,
	stakeAmount uint64,
	assetID ids.ID,
	startTime uint64,
	endTime uint64,
) (ids.ID, error) {
	ctx := context.Background()
	api := constants.LocalAPIEndpoint
	// Create empty EthKeychain if kc doesn't implement it
	var ethKc c.EthKeychain
	if ekc, ok := kc.(c.EthKeychain); ok {
		ethKc = ekc
	} else {
		// Create a minimal EthKeychain implementation
		ethKc = &emptyEthKeychain{}
	}
	// Use P-Chain only wallet since our X-Chain uses exchangevm which doesn't
	// support standard AVM API methods.
	wallet, err := primary.MakePChainWallet(ctx, &primary.WalletConfig{
		URI:         api,
		LUXKeychain: keychainwrapper.WrapCryptoKeychain(kc),
		EthKeychain: ethKc,
	})
	if err != nil {
		return ids.Empty, err
	}

	// Get the first address from the keychain for ownership
	addrs := kc.Addresses()
	if addrs.Len() == 0 {
		return ids.Empty, errors.New("keychain has no addresses")
	}
	ownerAddr := addrs.List()[0]

	owner := &secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs: []ids.ShortID{
			ownerAddr,
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultConfirmTxTimeout)
	txID, err := wallet.P().IssueAddPermissionlessValidatorTx(
		&txs.ChainValidator{
			Validator: txs.Validator{
				NodeID: nodeID,
				Start:  startTime,
				End:    endTime,
				Wght:   stakeAmount,
			},
			Chain: subnetID,
		},
		&signer.Empty{},
		assetID,
		owner,
		&secp256k1fx.OutputOwners{},
		reward.PercentDenominator,
	)
	defer cancel()
	if err != nil {
		return ids.Empty, err
	}
	return txID.ID(), err
}

func (d *LocalDeployer) StartServer() error {
	isRunning, err := d.procChecker.IsServerProcessRunning(d.app)
	if err != nil {
		return fmt.Errorf("failed querying if server process is running: %w", err)
	}
	if !isRunning {
		d.app.Log.Debug("gRPC server is not running")
		if err := binutils.StartServerProcess(d.app); err != nil {
			return fmt.Errorf("failed starting gRPC server process: %w", err)
		}
		d.backendStartedHere = true
	}
	return nil
}

func GetCurrentSupply(subnetID ids.ID) error {
	api := constants.LocalAPIEndpoint
	pClient := platformvm.NewClient(api)
	ctx, cancel := context.WithTimeout(context.Background(), constants.E2ERequestTimeout)
	defer cancel()
	_, _, err := pClient.GetCurrentSupply(ctx, subnetID)
	return err
}

// BackendStartedHere returns true if the backend was started by this run,
// or false if it found it there already
func (d *LocalDeployer) BackendStartedHere() bool {
	return d.backendStartedHere
}

// DeployBlockchain deploys a blockchain to the local network
func (d *LocalDeployer) DeployBlockchain(chain string, chainGenesis []byte) (ids.ID, ids.ID, error) {
	// For local deployment, we just call the regular deployment function
	return d.DeployToLocalNetwork(chain, chainGenesis, "")
}

// doDeploy deploys a blockchain to an already running network.
// Network must already be running - this function only deploys.
// steps:
//   - install VM plugin binary
//   - deploy blockchain to network
//   - wait for completion
//   - show status
func (d *LocalDeployer) doDeploy(chain string, chainGenesis []byte, genesisPath string) (ids.ID, ids.ID, error) {
	backendLogFile, err := binutils.GetBackendLogFile(d.app)
	var backendLogDir string
	if err == nil {
		backendLogDir = filepath.Dir(backendLogFile)
	}

	cli, err := d.getClientFunc()
	if err != nil {
		return ids.Empty, ids.Empty, fmt.Errorf("error creating gRPC Client: %w", err)
	}
	defer cli.Close()

	ctx := binutils.GetAsyncContext()

	// loading sidecar before it's needed so we catch any error early
	sc, err := d.app.LoadSidecar(chain)
	if err != nil {
		return ids.Empty, ids.Empty, fmt.Errorf("failed to load sidecar: %w", err)
	}

	// Get the actual VM name based on VM type
	// The VMID is computed from the VM name, not the chain name
	// For EVM chains, we use "Lux EVM" as the VM name
	// For custom VMs, we use the chain name
	vmName := "Lux EVM" // Default for EVM chains
	if sc.VM == models.CustomVM {
		vmName = chain // For custom VMs, use chain name
	}

	// Network must already be running - get cluster info
	clusterInfo, err := WaitForHealthy(ctx, cli)
	if err != nil {
		return ids.Empty, ids.Empty, fmt.Errorf("network is not healthy: %w", err)
	}
	rootDir := clusterInfo.GetRootDataDir()

	chainVMID, err := anrutils.VMID(vmName)
	if err != nil {
		return ids.Empty, ids.Empty, fmt.Errorf("failed to create VM ID from %s: %w", vmName, err)
	}
	d.app.Log.Debug("this VM will get ID", zap.String("vm-id", chainVMID.String()), zap.String("vm-name", vmName))

	if alreadyDeployed(chainVMID, clusterInfo) {
		ux.Logger.PrintToUser("Net %s has already been deployed", chain)
		return ids.Empty, ids.Empty, nil
	}

	numBlockchains := len(clusterInfo.CustomChains)

	// Get existing chain parent IDs from the network
	chainParentIDs := maps.Keys(clusterInfo.Chains)
	sort.Strings(chainParentIDs)

	var chainParentID string
	if len(chainParentIDs) > 0 {
		// Select an existing chain parent for the new blockchain
		// Use round-robin to distribute across available parents
		chainParentID = chainParentIDs[numBlockchains%len(chainParentIDs)]
		d.app.Log.Debug("using existing chain parent", zap.String("parent-id", chainParentID))
	} else {
		// No chain parents exist - netrunner will create one deterministically
		// This happens on first deploy to a fresh network
		d.app.Log.Debug("no existing chain parents, netrunner will create one")
	}

	// if a chainConfig has been configured
	var (
		chainConfig            string
		chainConfigFile        = filepath.Join(d.app.GetChainsDir(), chain, constants.ChainConfigFileName)
		perNodeChainConfig     string
		perNodeChainConfigFile = filepath.Join(d.app.GetChainsDir(), chain, constants.PerNodeChainConfigFileName)
	)
	if _, err := os.Stat(chainConfigFile); err == nil {
		// currently the ANR only accepts the file as a path, not its content
		chainConfig = chainConfigFile
	}
	if _, err := os.Stat(perNodeChainConfigFile); err == nil {
		perNodeChainConfig = perNodeChainConfigFile
	}

	// install the plugin binary for the new VM
	if err := d.installPlugin(chainVMID, d.vmBin); err != nil {
		return ids.Empty, ids.Empty, err
	}

	ux.Logger.PrintToUser("VMs ready.")

	// Create a new blockchain on the running network
	// VmName must be the actual VM name (e.g., "Lux EVM") not the chain name
	// This is used by netrunner to compute the VMID
	spec := &rpcpb.BlockchainSpec{
		VmName:             vmName,
		Genesis:            genesisPath,
		ChainConfig:        chainConfig,
		BlockchainAlias:    chain,
		PerNodeChainConfig: perNodeChainConfig,
	}
	// Only set ChainId if we have an existing parent
	if chainParentID != "" {
		spec.ChainId = &chainParentID
	}
	blockchainSpecs := []*rpcpb.BlockchainSpec{spec}
	deployBlockchainsInfo, err := cli.CreateChains(
		ctx,
		blockchainSpecs,
	)
	if err != nil {
		utils.FindErrorLogs(rootDir, backendLogDir)
		pluginRemoveErr := d.removeInstalledPlugin(chainVMID)
		if pluginRemoveErr != nil {
			ux.Logger.PrintToUser("Failed to remove plugin binary: %s", pluginRemoveErr)
		}
		return ids.Empty, ids.Empty, fmt.Errorf("failed to deploy blockchain: %w", err)
	}
	rootDir = clusterInfo.GetRootDataDir()

	d.app.Log.Debug(deployBlockchainsInfo.String())

	fmt.Println()
	ux.Logger.PrintToUser("Blockchain has been deployed. Waiting for chain to become healthy (timeout: %s)...", ChainHealthTimeout)

	// Use timeout-based health check for chain deployment
	// This will fail fast if the chain doesn't become healthy within the timeout
	clusterInfo, err = WaitForChainHealthyWithTimeout(cli, chain, ChainHealthTimeout, rootDir, backendLogDir)
	if err != nil {
		pluginRemoveErr := d.removeInstalledPlugin(chainVMID)
		if pluginRemoveErr != nil {
			ux.Logger.PrintToUser("Failed to remove plugin binary: %s", pluginRemoveErr)
		}
		return ids.Empty, ids.Empty, err
	}

	endpoint := GetFirstEndpoint(clusterInfo, chain)

	fmt.Println()
	ux.Logger.PrintToUser("Network ready to use. Local network node endpoints:")
	ux.PrintTableEndpoints(clusterInfo)
	fmt.Println()

	ux.Logger.PrintToUser("Browser Extension connection details (any node URL from above works):")
	if endpoint != "" {
		httpIdx := strings.LastIndex(endpoint, "http")
		if httpIdx >= 0 {
			ux.Logger.PrintToUser("RPC URL:          %s", endpoint[httpIdx:])
		}
	}

	if sc.VM == models.EVM {
		if err := d.printExtraEvmInfo(chain, chainGenesis); err != nil {
			// not supposed to happen due to genesis pre validation
			return ids.Empty, ids.Empty, nil
		}
	}

	// Parse the chain parent ID
	parentID, _ := ids.FromString(chainParentID)
	var blockchainID ids.ID
	for _, info := range clusterInfo.CustomChains {
		if info.VmId == chainVMID.String() {
			blockchainID, _ = ids.FromString(info.BlockchainId)
		}
	}
	return parentID, blockchainID, nil
}

func (d *LocalDeployer) printExtraEvmInfo(chain string, chainGenesis []byte) error {
	var evmGenesis core.Genesis
	if err := json.Unmarshal(chainGenesis, &evmGenesis); err != nil {
		return fmt.Errorf("failed to unmarshall genesis: %w", err)
	}
	for address := range evmGenesis.Alloc {
		amount := evmGenesis.Alloc[address].Balance
		formattedAmount := new(big.Int).Div(amount, big.NewInt(params.Ether))
		ux.Logger.PrintToUser("Funded address:   %s with %s", address, formattedAmount.String())
	}
	ux.Logger.PrintToUser("Network name:     %s", chain)
	ux.Logger.PrintToUser("Chain ID:         %s", evmGenesis.Config.ChainID)
	ux.Logger.PrintToUser("Currency Symbol:  %s", d.app.GetTokenName(chain))
	return nil
}

// SetupLocalEnv also does some heavy lifting:
// * sets up default snapshot if not installed
// * checks if node is installed in the local binary path
// * if not, it downloads it and installs it (os - and archive dependent)
// * returns the location of the node path
func (d *LocalDeployer) SetupLocalEnv() (string, error) {
	err := d.setDefaultSnapshot(d.app.GetSnapshotsDir(), false)
	if err != nil {
		return "", fmt.Errorf("failed setting up snapshots: %w", err)
	}

	luxDir, err := d.setupLocalEnv()
	if err != nil {
		return "", fmt.Errorf("failed setting up local environment: %w", err)
	}

	pluginDir := d.app.GetPluginsDir()
	nodeBinPath := filepath.Join(luxDir, "luxd")

	if err := os.MkdirAll(pluginDir, constants.DefaultPerms755); err != nil {
		return "", fmt.Errorf("could not create pluginDir %s", pluginDir)
	}

	exists, err := storage.FolderExists(pluginDir)
	if !exists || err != nil {
		return "", fmt.Errorf("evaluated pluginDir to be %s but it does not exist", pluginDir)
	}

	// Version management: compare latest to local version
	// and update if necessary based on compatibility requirements
	exists, err = storage.FileExists(nodeBinPath)
	if !exists || err != nil {
		return "", fmt.Errorf(
			"evaluated nodeBinPath to be %s but it does not exist", nodeBinPath)
	}

	return nodeBinPath, nil
}

func (d *LocalDeployer) setupLocalEnv() (string, error) {
	return binutils.SetupLux(d.app, d.luxVersion)
}

// WaitForHealthy polls continuously until the network is ready to be used
func WaitForHealthy(
	ctx context.Context,
	cli client.Client,
) (*rpcpb.ClusterInfo, error) {
	cancel := make(chan struct{})
	defer close(cancel)
	go ux.PrintWait(cancel)
	resp, err := cli.WaitForHealthy(ctx)
	if err != nil {
		return nil, err
	}
	return resp.ClusterInfo, nil
}

// WaitForChainHealthyWithTimeout waits for a chain to become healthy with a specific timeout.
// Returns an error with helpful diagnostics if the chain fails to become healthy.
func WaitForChainHealthyWithTimeout(
	cli client.Client,
	chainName string,
	timeout time.Duration,
	rootDir string,
	backendLogDir string,
) (*rpcpb.ClusterInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cancelPrint := make(chan struct{})
	defer close(cancelPrint)
	go ux.PrintWait(cancelPrint)

	resp, err := cli.WaitForHealthy(ctx)
	if err != nil {
		// Check if it's a timeout
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, formatChainHealthError(cli, chainName, timeout, rootDir, backendLogDir)
		}
		// For other errors, still try to get diagnostic info
		if resp != nil && resp.ClusterInfo != nil {
			return resp.ClusterInfo, formatChainHealthError(cli, chainName, timeout, rootDir, backendLogDir)
		}
		return nil, fmt.Errorf("chain health check failed: %w", err)
	}

	// Even if WaitForHealthy returns without error, verify the chain is actually healthy
	if resp.ClusterInfo != nil && !resp.ClusterInfo.CustomChainsHealthy {
		return resp.ClusterInfo, formatChainHealthError(cli, chainName, timeout, rootDir, backendLogDir)
	}

	return resp.ClusterInfo, nil
}

// formatChainHealthError creates a detailed error message when a chain fails to become healthy
func formatChainHealthError(
	cli client.Client,
	chainName string,
	timeout time.Duration,
	rootDir string,
	backendLogDir string,
) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n\nERROR: Chain '%s' failed to become healthy within %s\n\n", chainName, timeout))

	// Try to get current health status for more info
	healthCtx, healthCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer healthCancel()
	healthResp, healthErr := cli.Health(healthCtx)
	if healthErr == nil && healthResp != nil && healthResp.ClusterInfo != nil {
		sb.WriteString("Node health check shows:\n")
		if !healthResp.ClusterInfo.Healthy {
			sb.WriteString("  - Network is not healthy\n")
		}
		if !healthResp.ClusterInfo.CustomChainsHealthy {
			sb.WriteString("  - Custom chains are not healthy\n")
		}
		// Show info about any custom chains that were being deployed
		for chainID, chainInfo := range healthResp.ClusterInfo.CustomChains {
			sb.WriteString(fmt.Sprintf("  - Chain %s (VM: %s): %s\n", chainID, chainInfo.VmId, chainInfo.ChainName))
		}
	}

	sb.WriteString("\nThe VM likely crashed during initialization. Common causes:\n")
	sb.WriteString("  - Invalid genesis configuration\n")
	sb.WriteString("  - VM binary incompatibility\n")
	sb.WriteString("  - Missing or incorrect chain configuration\n")

	sb.WriteString("\nTo debug:\n")
	if rootDir != "" {
		sb.WriteString(fmt.Sprintf("  tail -f %s/node1/logs/*.log\n", rootDir))
	}
	if backendLogDir != "" {
		sb.WriteString(fmt.Sprintf("  tail -f %s/*.log\n", backendLogDir))
	}
	sb.WriteString("  lux network status\n")

	// Try to find and print relevant error logs
	if rootDir != "" || backendLogDir != "" {
		utils.FindErrorLogs(rootDir, backendLogDir)
	}

	return errors.New(sb.String())
}

// GetFirstEndpoint get a human readable endpoint for the given chain
func GetFirstEndpoint(clusterInfo *rpcpb.ClusterInfo, chain string) string {
	var endpoint string
	for _, nodeInfo := range clusterInfo.NodeInfos {
		for blockchainID, chainInfo := range clusterInfo.CustomChains {
			if chainInfo.ChainName == chain && nodeInfo.Name == clusterInfo.NodeNames[0] {
				endpoint = fmt.Sprintf("Endpoint at node %s for blockchain %q with VM ID %q: %s/ext/bc/%s/rpc", nodeInfo.Name, blockchainID, chainInfo.VmId, nodeInfo.GetUri(), blockchainID)
			}
		}
	}
	return endpoint
}

// HasEndpoints returns true if cluster info contains custom blockchains
func HasEndpoints(clusterInfo *rpcpb.ClusterInfo) bool {
	return len(clusterInfo.CustomChains) > 0
}

// return true if vm has already been deployed
func alreadyDeployed(chainVMID ids.ID, clusterInfo *rpcpb.ClusterInfo) bool {
	if clusterInfo != nil {
		for _, chainInfo := range clusterInfo.CustomChains {
			if chainInfo.VmId == chainVMID.String() {
				return true
			}
		}
	}
	return false
}

// get list of all needed plugins and install them
func (d *LocalDeployer) installPlugin(
	vmID ids.ID,
	vmBin string,
) error {
	return d.binaryDownloader.InstallVM(vmID.String(), vmBin)
}

// get list of all needed plugins and install them
func (d *LocalDeployer) removeInstalledPlugin(
	vmID ids.ID,
) error {
	return d.binaryDownloader.RemoveVM(vmID.String())
}

func getExpectedDefaultSnapshotSHA256Sum() (string, error) {
	resp, err := http.Get(constants.BootstrapSnapshotSHA256URL)
	if err != nil {
		return "", fmt.Errorf("failed downloading sha256 sums: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed downloading sha256 sums: unexpected http status code: %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	sha256FileBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed downloading sha256 sums: %w", err)
	}
	expectedSum, err := utils.SearchSHA256File(sha256FileBytes, constants.BootstrapSnapshotLocalPath)
	if err != nil {
		return "", fmt.Errorf("failed obtaining snapshot sha256 sum: %w", err)
	}
	return expectedSum, nil
}

// Initialize default snapshot with bootstrap snapshot archive
// If force flag is set to true, overwrite the default snapshot if it exists
func SetDefaultSnapshot(snapshotsDir string, force bool) error {
	defaultSnapshotPath := filepath.Join(snapshotsDir, "anr-snapshot-"+constants.DefaultSnapshotName)
	if force {
		if err := os.RemoveAll(defaultSnapshotPath); err != nil {
			return fmt.Errorf("failed removing default snapshot: %w", err)
		}
	}
	// Always create a fresh snapshot with embedded genesis from netrunner
	// This avoids downloading potentially corrupted snapshots from GitHub
	if _, err := os.Stat(defaultSnapshotPath); os.IsNotExist(err) {
		if err := os.MkdirAll(defaultSnapshotPath, 0o755); err != nil {
			return fmt.Errorf("failed creating snapshot directory: %w", err)
		}
		// Create network.json with embedded genesis from netrunner
		genesis, err := anrnetwork.LoadLocalGenesis()
		if err != nil {
			return fmt.Errorf("failed loading local genesis: %w", err)
		}
		genesisBytes, err := json.Marshal(genesis)
		if err != nil {
			return fmt.Errorf("failed marshaling genesis: %w", err)
		}
		networkConfig := map[string]interface{}{
			"genesis":   string(genesisBytes),
			"networkID": 1337,
		}
		networkBytes, err := json.MarshalIndent(networkConfig, "", "  ")
		if err != nil {
			return fmt.Errorf("failed marshaling network config: %w", err)
		}
		networkJsonPath := filepath.Join(defaultSnapshotPath, "network.json")
		if err := os.WriteFile(networkJsonPath, networkBytes, WriteReadReadPerms); err != nil {
			return fmt.Errorf("failed writing network.json: %w", err)
		}
		ux.Logger.PrintToUser("Created fresh snapshot with embedded genesis")
	}
	return nil
}

// start the network
func (d *LocalDeployer) startNetwork(
	ctx context.Context,
	cli client.Client,
	nodeBinPath string,
	runDir string,
) error {
	opts := []client.OpOption{
		client.WithExecPath(nodeBinPath),
		client.WithRootDataDir(runDir),
		client.WithReassignPortsIfUsed(true),
		client.WithPluginDir(d.app.GetPluginsDir()),
	}

	// load global node configs if they exist
	configStr, err := d.app.Conf.LoadNodeConfig()
	if err != nil {
		return nil
	}
	if configStr != "" {
		opts = append(opts, client.WithGlobalNodeConfig(configStr))
	}

	// Try to load from snapshot first, if it has valid nodes
	snapshotPath := filepath.Join(d.app.GetSnapshotsDir(), "anr-snapshot-"+constants.DefaultSnapshotName)
	dbPath := filepath.Join(snapshotPath, "db")

	// Check if we have a valid snapshot with nodes (db directory with node subdirs)
	if fi, dbErr := os.Stat(dbPath); dbErr == nil && fi.IsDir() {
		// Check if there's at least one node directory
		entries, _ := os.ReadDir(dbPath)
		hasNodes := false
		for _, e := range entries {
			if e.IsDir() && strings.HasPrefix(e.Name(), "node") {
				hasNodes = true
				break
			}
		}
		if hasNodes {
			pp, err := cli.LoadSnapshot(
				ctx,
				constants.DefaultSnapshotName,
				opts...,
			)
			if err == nil {
				ux.Logger.PrintToUser("Node log path: %s/node<i>/logs", pp.ClusterInfo.RootDataDir)
				ux.Logger.PrintToUser("Starting network from snapshot...")
				return nil
			}
			// If LoadSnapshot fails, fall through to Start
			ux.Logger.PrintToUser("Snapshot load failed, starting fresh network: %s", err)
		}
	}

	// Start a fresh network using netrunner's embedded genesis
	ux.Logger.PrintToUser("Starting fresh local network...")
	pp, err := cli.Start(ctx, nodeBinPath, opts...)
	if err != nil {
		return fmt.Errorf("failed to start network: %w", err)
	}
	ux.Logger.PrintToUser("Node log path: %s/node<i>/logs", pp.ClusterInfo.RootDataDir)
	ux.Logger.PrintToUser("Network started successfully")
	return nil
}

// Returns an error if the server cannot be contacted. You may want to ignore this error.
func GetLocallyDeployedSubnets() (map[string]struct{}, error) {
	deployedNames := map[string]struct{}{}
	// if the server can not be contacted, or there is a problem with the query,
	// DO NOT FAIL, just print No for deployed status
	cli, err := binutils.NewGRPCClient()
	if err != nil {
		return nil, err
	}

	ctx := binutils.GetAsyncContext()
	resp, err := cli.Status(ctx)
	if err != nil {
		return nil, err
	}

	for _, chain := range resp.GetClusterInfo().CustomChains {
		deployedNames[chain.ChainName] = struct{}{}
	}

	return deployedNames, nil
}

func IssueRemoveSubnetValidatorTx(kc keychain.Keychain, subnetID ids.ID, nodeID ids.NodeID) (ids.ID, error) {
	ctx := context.Background()
	api := constants.LocalAPIEndpoint
	// Create empty EthKeychain if kc doesn't implement it
	var ethKc c.EthKeychain
	if ekc, ok := kc.(c.EthKeychain); ok {
		ethKc = ekc
	} else {
		// Create a minimal EthKeychain implementation
		ethKc = &emptyEthKeychain{}
	}
	// Use P-Chain only wallet since our X-Chain uses exchangevm which doesn't
	// support standard AVM API methods.
	wallet, err := primary.MakePChainWallet(ctx, &primary.WalletConfig{
		URI:         api,
		LUXKeychain: keychainwrapper.WrapCryptoKeychain(kc),
		EthKeychain: ethKc,
	})
	if err != nil {
		return ids.Empty, err
	}

	tx, err := wallet.P().IssueRemoveChainValidatorTx(nodeID, subnetID)
	if err != nil {
		return ids.Empty, err
	}
	return tx.ID(), nil
}

func GetSubnetValidators(subnetID ids.ID) ([]platformvm.ClientPermissionlessValidator, error) {
	api := constants.LocalAPIEndpoint
	pClient := platformvm.NewClient(api)
	ctx, cancel := context.WithTimeout(context.Background(), constants.E2ERequestTimeout)
	defer cancel()

	return pClient.GetCurrentValidators(ctx, subnetID, nil)
}

func CheckNodeIsInSubnetPendingValidators(subnetID ids.ID, nodeID string) (bool, error) {
	api := constants.LocalAPIEndpoint
	pClient := platformvm.NewClient(api)
	ctx, cancel := context.WithTimeout(context.Background(), constants.E2ERequestTimeout)
	defer cancel()

	// Get validators that will be active in the future (pending validators)
	futureTime := uint64(time.Now().Add(time.Hour).Unix())
	validators, err := pClient.GetValidatorsAt(ctx, subnetID, platformapi.Height(futureTime))
	if err != nil {
		return false, err
	}

	// Convert nodeID string to ids.NodeID for comparison
	nID, err := ids.NodeIDFromString(nodeID)
	if err != nil {
		return false, err
	}

	// Check current validators
	currentValidators, err := pClient.GetCurrentValidators(ctx, subnetID, nil)
	if err != nil {
		return false, err
	}

	// Check if the node is in future validators but not in current validators
	inFuture := false
	for id := range validators {
		if id == nID {
			inFuture = true
			break
		}
	}

	if !inFuture {
		return false, nil
	}

	// Check if it's already a current validator
	for _, v := range currentValidators {
		if v.NodeID == nID {
			return false, nil // Already active, not pending
		}
	}

	return true, nil // In future but not current = pending
}
