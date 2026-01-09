// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package chain provides chain deployment and management utilities.
package chain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/binutils"
	keychainwrapper "github.com/luxfi/cli/pkg/keychain"
	climodels "github.com/luxfi/cli/pkg/models"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/constants"
	"github.com/luxfi/evm/core"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/params"
	"github.com/luxfi/ids"
	"github.com/luxfi/keychain"
	walletkeychain "github.com/luxfi/keychain"
	"github.com/luxfi/math/set"
	"github.com/luxfi/netrunner/client"
	anrnetwork "github.com/luxfi/netrunner/network"
	"github.com/luxfi/netrunner/rpcpb"
	"github.com/luxfi/netrunner/server"
	anrutils "github.com/luxfi/netrunner/utils"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/vm/utils/storage"
	"github.com/luxfi/sdk/wallet/chain/c"
	"github.com/luxfi/sdk/wallet/primary"
	"github.com/luxfi/vm/vms/components/lux"
	"github.com/luxfi/vm/vms/components/verify"
	"github.com/luxfi/vm/vms/platformvm"
	platformapi "github.com/luxfi/vm/vms/platformvm/api"
	"github.com/luxfi/vm/vms/platformvm/reward"
	"github.com/luxfi/vm/vms/platformvm/signer"
	"github.com/luxfi/vm/vms/platformvm/txs"
	"github.com/luxfi/vm/vms/secp256k1fx"
	"go.uber.org/zap"
)

// Chain deployment constants.
const (
	WriteReadReadPerms = 0o644
	// ChainHealthTimeout is the maximum time to wait for a newly deployed chain to become healthy
	// For local networks (5 nodes on localhost), chains should be healthy in <10s
	ChainHealthTimeout = 10 * time.Second
	// LocalNetworkHealthTimeout is for checking if the network itself is running
	LocalNetworkHealthTimeout = 5 * time.Second
	// BlockchainCreationTimeout is the maximum time to wait for CreateChains RPC call
	// This involves a P-chain transaction, subnet creation, chain creation, node restarts,
	// and P-chain height sync across all 5 validators. Needs 90s minimum for stability.
	BlockchainCreationTimeout = 90 * time.Second
)

// DeploymentError represents a chain deployment failure that does NOT crash the network.
// The network remains running and can accept new deployments.
type DeploymentError struct {
	ChainName string
	Cause     error
	// NetworkHealthy indicates if the primary network is still running after the failure
	NetworkHealthy bool
	// Recoverable indicates if the error can be fixed and retried
	Recoverable bool
	// Suggestion provides actionable guidance to fix the issue
	Suggestion string
}

func (e *DeploymentError) Error() string {
	status := "network crashed"
	if e.NetworkHealthy {
		status = "network still running"
	}
	msg := fmt.Sprintf("chain '%s' deployment failed (%s): %v", e.ChainName, status, e.Cause)
	if e.Suggestion != "" {
		msg += "\n\nTo fix: " + e.Suggestion
	}
	return msg
}

func (e *DeploymentError) Unwrap() error {
	return e.Cause
}

// NewRecoverableDeploymentError creates a deployment error that can be retried
func NewRecoverableDeploymentError(chainName string, cause error, suggestion string) *DeploymentError {
	return &DeploymentError{
		ChainName:      chainName,
		Cause:          cause,
		NetworkHealthy: true, // Recoverable errors shouldn't crash the network
		Recoverable:    true,
		Suggestion:     suggestion,
	}
}

// emptyEthKeychain is a minimal implementation of EthKeychain for cases where ETH keys are not needed
type emptyEthKeychain struct{}

func (*emptyEthKeychain) GetEth(_ common.Address) (walletkeychain.Signer, bool) {
	return nil, false
}

func (*emptyEthKeychain) EthAddresses() set.Set[common.Address] {
	return set.NewSet[common.Address](0)
}

// LocalDeployer handles local chain deployment.
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
	networkType        string // "mainnet", "testnet", or "local"
}

// NewLocalDeployer creates a new LocalDeployer instance.
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
		networkType:        "", // Auto-detect from network state
	}
}

// NewLocalDeployerForNetwork creates a deployer for a specific network type
func NewLocalDeployerForNetwork(app *application.Lux, luxVersion, vmBin, networkType string) *LocalDeployer {
	d := NewLocalDeployer(app, luxVersion, vmBin)
	d.networkType = networkType
	// Use network-aware gRPC client
	d.getClientFunc = func(opts ...binutils.GRPCClientOpOption) (client.Client, error) {
		opts = append(opts, binutils.WithNetworkType(networkType))
		return binutils.NewGRPCClient(opts...)
	}
	return d
}

type getGRPCClientFunc func(...binutils.GRPCClientOpOption) (client.Client, error)

type setDefaultSnapshotFunc func(string, bool) error

// DeployToLocalNetwork deploys to an already running network.
// It does NOT start the network - use 'lux network start' first.
func (d *LocalDeployer) DeployToLocalNetwork(chain string, chainGenesis []byte, genesisPath string) (ids.ID, ids.ID, error) {
	// Create step tracker that warns after 5 seconds
	tracker := ux.NewStepTracker(ux.Logger, 5*time.Second)

	// Connect to existing gRPC server - do NOT start one
	tracker.Start("Connecting to network")
	cli, err := d.getClientFunc()
	if err != nil {
		tracker.Failed("connection failed")
		return ids.Empty, ids.Empty, fmt.Errorf("failed to connect to network. Is it running? Start with: lux network start --mainnet\nError: %w", err)
	}
	defer func() { _ = cli.Close() }()
	tracker.Complete("")

	// Quick health check with short timeout for local network (5s max)
	ctx, cancel := context.WithTimeout(context.Background(), LocalNetworkHealthTimeout)
	defer cancel()

	tracker.Start("Checking network health")
	_, err = WaitForHealthy(ctx, cli)
	if err != nil {
		if server.IsServerError(err, server.ErrNotBootstrapped) {
			tracker.Failed("network not running")
			return ids.Empty, ids.Empty, fmt.Errorf("network is not running. Start it first with: lux network start --mainnet")
		}
		tracker.Failed(err.Error())
		return ids.Empty, ids.Empty, fmt.Errorf("network is unhealthy: %w", err)
	}
	tracker.CompleteSuccess()

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

// IssueTransformSubnetTx transforms a subnet to a permissionless elastic subnet.
func IssueTransformSubnetTx(
	elasticSubnetConfig climodels.ElasticChainConfig,
	kc keychain.Keychain,
	_ ids.ID, // subnetID comes from elasticSubnetConfig
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

	_, cancel := context.WithTimeout(context.Background(), constants.DefaultConfirmTxTimeout)
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

// IssueAddPermissionlessValidatorTx issues an add permissionless validator transaction.
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
	_, cancel := context.WithTimeout(context.Background(), constants.DefaultConfirmTxTimeout)
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

// StartServer starts the gRPC server for the deployer's network type.
// If no network type is set, defaults to mainnet for backward compatibility.
func (d *LocalDeployer) StartServer() error {
	networkType := d.networkType
	if networkType == "" {
		networkType = "mainnet" // Default for backward compatibility
	}
	return d.StartServerForNetwork(networkType)
}

// StartServerForNetwork starts the gRPC server for a specific network type.
func (d *LocalDeployer) StartServerForNetwork(networkType string) error {
	isRunning, err := binutils.IsServerProcessRunningForNetwork(d.app, networkType)
	if err != nil {
		return fmt.Errorf("failed querying if server process is running: %w", err)
	}
	if !isRunning {
		d.app.Log.Debug("gRPC server is not running", zap.String("network", networkType))
		if err := binutils.StartServerProcessForNetwork(d.app, networkType); err != nil {
			return fmt.Errorf("failed starting gRPC server for %s: %w", networkType, err)
		}
		d.backendStartedHere = true
	}
	return nil
}

// GetCurrentSupply prints the current supply of a subnet.
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
// IMPORTANT: This function is designed to NEVER crash the primary network.
// If deployment fails, it returns a DeploymentError but the network continues running.
//
// Steps:
//  1. Preflight validation (VM binary, genesis, config)
//  2. Install VM plugin binary
//  3. Deploy blockchain to network
//  4. Wait for chain health
//  5. Show status
func (d *LocalDeployer) doDeploy(chain string, chainGenesis []byte, genesisPath string) (ids.ID, ids.ID, error) {
	// Create step tracker that warns after 5 seconds
	tracker := ux.NewStepTracker(ux.Logger, 5*time.Second)

	backendLogFile, err := binutils.GetBackendLogFile(d.app)
	var backendLogDir string
	if err == nil {
		backendLogDir = filepath.Dir(backendLogFile)
	}

	tracker.Start("Connecting to gRPC server")
	cli, err := d.getClientFunc()
	if err != nil {
		tracker.Failed("connection failed")
		return ids.Empty, ids.Empty, fmt.Errorf("error creating gRPC Client: %w", err)
	}
	defer func() { _ = cli.Close() }()
	tracker.Complete("")

	// loading sidecar before it's needed so we catch any error early
	tracker.Start("Loading chain configuration")
	sc, err := d.app.LoadSidecar(chain)
	if err != nil {
		tracker.Failed("config not found")
		return ids.Empty, ids.Empty, fmt.Errorf("failed to load sidecar: %w", err)
	}
	tracker.Complete("")

	// Get the actual VM name based on VM type
	// The VMID is computed from the VM name, not the chain name
	// For EVM chains, we use "Lux EVM" as the VM name
	// For custom VMs, we use the chain name
	vmName := "Lux EVM" // Default for EVM chains
	if sc.VM == models.CustomVM {
		vmName = chain // For custom VMs, use chain name
	}

	// Network must already be running - get cluster info
	// Use short timeout for local network health check
	healthCtx, healthCancel := context.WithTimeout(context.Background(), LocalNetworkHealthTimeout)
	defer healthCancel()

	tracker.Start("Verifying network is ready")
	clusterInfo, err := WaitForHealthy(healthCtx, cli)
	if err != nil {
		tracker.Failed("network unhealthy")
		return ids.Empty, ids.Empty, fmt.Errorf("network is not healthy: %w", err)
	}
	tracker.CompleteSuccess()
	rootDir := clusterInfo.GetRootDataDir()

	chainVMID, err := anrutils.VMID(vmName)
	if err != nil {
		return ids.Empty, ids.Empty, fmt.Errorf("failed to create VM ID from %s: %w", vmName, err)
	}
	d.app.Log.Debug("this VM will get ID", zap.String("vm-id", chainVMID.String()), zap.String("vm-name", vmName))

	// Check if this specific chain is already deployed (by chain name, not VM ID)
	// Multiple chains can use the same VM (e.g., multiple EVM chains using Lux EVM)
	if alreadyDeployedByName(chain, clusterInfo) {
		ux.Logger.GreenCheckmarkToUser("Chain %s already deployed", chain)
		return ids.Empty, ids.Empty, nil
	}

	// Each blockchain gets its own subnet unless explicitly configured to share one.
	// The netrunner will create a new subnet for this chain.
	// NOTE: Removed the round-robin logic that incorrectly assigned new chains to existing subnets.
	// Each chain is independent and needs its own subnet for proper isolation.
	var chainParentID string
	d.app.Log.Debug("no chain parent specified, netrunner will create a new subnet for this chain")

	// if a chainConfig has been configured
	var (
		chainConfig            string
		chainConfigFile        = filepath.Join(d.app.GetChainsDir(), chain, constants.ChainConfigFile)
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

	// === PREFLIGHT VALIDATION ===
	// Validate VM binary BEFORE installing to catch issues early.
	// This prevents deploying a broken VM that would crash nodes.
	tracker.Start("Validating VM binary")
	if err := d.validateVMBinary(d.vmBin, chainVMID); err != nil {
		tracker.Failed("validation failed")
		return ids.Empty, ids.Empty, NewRecoverableDeploymentError(
			chain,
			fmt.Errorf("VM preflight validation failed: %w", err),
			"Rebuild the VM binary or check the VM path",
		)
	}
	tracker.CompleteSuccess()

	// install the plugin binary for the new VM
	tracker.Start("Installing VM plugin")
	if err := d.installPlugin(chainVMID, d.vmBin); err != nil {
		tracker.Failed("installation failed")
		return ids.Empty, ids.Empty, NewRecoverableDeploymentError(
			chain,
			fmt.Errorf("failed to install VM plugin: %w", err),
			"Check plugin directory permissions and disk space",
		)
	}
	tracker.CompleteSuccess()

	// Create a new blockchain on the running network
	// VmName must be the actual VM name (e.g., "Lux EVM") not the chain name
	// This is used by netrunner to compute the VMID
	tracker.Start(fmt.Sprintf("Creating blockchain '%s' on P-chain", chain))
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

	// Use short timeout for blockchain creation - should complete in <15s on local network
	// If it takes longer, the network or VM has a problem
	createCtx, createCancel := context.WithTimeout(context.Background(), BlockchainCreationTimeout)
	defer createCancel()

	// Start a goroutine to check for warnings during long operations
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				tracker.CheckWarn()
			}
		}
	}()

	deployBlockchainsInfo, err := cli.CreateChains(
		createCtx,
		blockchainSpecs,
	)
	close(done) // Stop the warning checker

	if err != nil {
		tracker.Failed(err.Error())
		utils.FindErrorLogs(rootDir, backendLogDir)
		// Check if the network is still healthy after the failure
		networkHealthy := d.checkNetworkHealthQuick(cli)

		// Provide specific error message based on failure type
		var errMsg string
		if errors.Is(err, context.DeadlineExceeded) {
			errMsg = fmt.Sprintf("blockchain creation timed out after %s (limit: %s)", tracker.Elapsed().Round(time.Millisecond), BlockchainCreationTimeout)
		} else {
			errMsg = fmt.Sprintf("blockchain creation failed after %s: %v", tracker.Elapsed().Round(time.Millisecond), err)
		}

		return ids.Empty, ids.Empty, &DeploymentError{
			ChainName:      chain,
			Cause:          errors.New(errMsg),
			NetworkHealthy: networkHealthy,
		}
	}
	tracker.CompleteSuccess()

	// Wait for validators to track the chain
	tracker.Start("Waiting for validators to track chain")
	// Start warning checker for health wait
	healthDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-healthDone:
				return
			case <-ticker.C:
				tracker.CheckWarn()
			}
		}
	}()

	// Quick status check to verify chain is tracked
	statusCtx, statusCancel := context.WithTimeout(context.Background(), ChainHealthTimeout)
	defer statusCancel()
	if statusResp, statusErr := cli.Status(statusCtx); statusErr == nil && statusResp.ClusterInfo != nil {
		clusterInfo = statusResp.ClusterInfo
	}
	close(healthDone)
	tracker.CompleteSuccess()

	d.app.Log.Debug(deployBlockchainsInfo.String())

	fmt.Println()
	ux.Logger.GreenCheckmarkToUser("Blockchain deployed successfully")
	ux.Logger.PrintToUser("Chain is now available - nodes will sync in background.")

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

	// Find the blockchain and subnet IDs from the cluster info
	var subnetID, blockchainID ids.ID
	for _, info := range clusterInfo.CustomChains {
		if info.VmId == chainVMID.String() {
			blockchainID, _ = ids.FromString(info.BlockchainId)
			subnetID, _ = ids.FromString(info.PchainId)
		}
	}
	return subnetID, blockchainID, nil
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

	pluginDir := d.app.GetCurrentPluginsDir()
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

// alreadyDeployedByName returns true if a chain with the given name is already deployed
// This is the correct check for multi-chain deployments using the same VM (e.g., multiple EVM subnets)
func alreadyDeployedByName(chainName string, clusterInfo *rpcpb.ClusterInfo) bool {
	if clusterInfo != nil {
		for _, chainInfo := range clusterInfo.CustomChains {
			if chainInfo.ChainName == chainName {
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

// checkNetworkHealthQuick performs a fast health check to see if the network is still running.
// This is used after deployment failures to determine if the network crashed.
// Returns true if network appears healthy, false otherwise.
func (d *LocalDeployer) checkNetworkHealthQuick(cli client.Client) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := cli.Health(ctx)
	if err != nil {
		d.app.Log.Debug("quick health check failed", zap.Error(err))
		return false
	}
	if resp == nil || resp.ClusterInfo == nil {
		return false
	}
	return resp.ClusterInfo.Healthy
}

// validateVMBinary performs preflight checks on a VM binary before deployment.
// This catches common issues that would crash nodes when loading the VM.
// Returns nil if validation passes, error with actionable message otherwise.
func (d *LocalDeployer) validateVMBinary(vmBin string, vmID ids.ID) error {
	// Check binary exists
	info, err := os.Stat(vmBin)
	if os.IsNotExist(err) {
		return fmt.Errorf("VM binary not found: %s\n\nTo fix: build or download the VM binary first", vmBin)
	}
	if err != nil {
		return fmt.Errorf("cannot access VM binary %s: %w", vmBin, err)
	}

	// Check it's a regular file (not directory, symlink, etc)
	if !info.Mode().IsRegular() {
		// If symlink, check target exists
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(vmBin)
			if err != nil {
				return fmt.Errorf("VM binary %s is a symlink but cannot read target: %w", vmBin, err)
			}
			if _, err := os.Stat(target); os.IsNotExist(err) {
				return fmt.Errorf("VM binary symlink %s points to missing target: %s\n\nTo fix: rebuild the VM or update the symlink", vmBin, target)
			}
		}
	}

	// Check executable permissions
	if info.Mode()&0o111 == 0 {
		return fmt.Errorf("VM binary %s is not executable\n\nTo fix: chmod +x %s", vmBin, vmBin)
	}

	// Check minimum file size (VM binaries should be at least a few KB)
	const minVMSize = 1024 // 1KB minimum
	if info.Size() < minVMSize {
		return fmt.Errorf("VM binary %s is too small (%d bytes) - may be corrupted\n\nTo fix: rebuild the VM", vmBin, info.Size())
	}

	d.app.Log.Debug("VM binary validation passed",
		zap.String("binary", vmBin),
		zap.String("vmid", vmID.String()),
		zap.Int64("size", info.Size()),
	)
	return nil
}

// SetDefaultSnapshot initializes the default snapshot with the bootstrap snapshot archive.
// If force flag is set to true, it overwrites the default snapshot if it exists.
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
		if err := os.MkdirAll(defaultSnapshotPath, 0o750); err != nil {
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
		networkJSONPath := filepath.Join(defaultSnapshotPath, "network.json")
		if err := os.WriteFile(networkJSONPath, networkBytes, WriteReadReadPerms); err != nil {
			return fmt.Errorf("failed writing network.json: %w", err)
		}
		ux.Logger.PrintToUser("Created fresh snapshot with embedded genesis")
	}
	return nil
}

// GetLocallyDeployedSubnets returns the locally deployed subnets. Returns an error if the server cannot be contacted.
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

// IssueRemoveSubnetValidatorTx issues a remove subnet validator transaction.
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

// GetSubnetValidators returns the validators for a subnet.
func GetSubnetValidators(subnetID ids.ID) ([]platformvm.ClientPermissionlessValidator, error) {
	api := constants.LocalAPIEndpoint
	pClient := platformvm.NewClient(api)
	ctx, cancel := context.WithTimeout(context.Background(), constants.E2ERequestTimeout)
	defer cancel()

	return pClient.GetCurrentValidators(ctx, subnetID, nil)
}

// CheckNodeIsInSubnetPendingValidators checks if a node is in the pending validators for a subnet.
func CheckNodeIsInSubnetPendingValidators(subnetID ids.ID, nodeID string) (bool, error) {
	api := constants.LocalAPIEndpoint
	pClient := platformvm.NewClient(api)
	ctx, cancel := context.WithTimeout(context.Background(), constants.E2ERequestTimeout)
	defer cancel()

	// Get validators that will be active in the future (pending validators)
	futureTime := uint64(time.Now().Add(time.Hour).Unix()) //nolint:gosec // G115: Unix time is positive
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
