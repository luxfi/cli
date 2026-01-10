// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package chain provides chain deployment and management utilities.
package chain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/luxfi/node/vms/platformvm"
	lux "github.com/luxfi/utxo"
	"github.com/luxfi/vm/components/verify"

	"github.com/luxfi/address"
	"github.com/luxfi/cli/pkg/application"
	keychainwrapper "github.com/luxfi/cli/pkg/keychain"
	climodels "github.com/luxfi/cli/pkg/models"
	"github.com/luxfi/cli/pkg/txutils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/constants"
	ethcommon "github.com/luxfi/geth/common"
	"github.com/luxfi/ids"
	"github.com/luxfi/keychain"
	"github.com/luxfi/math/set"
	"github.com/luxfi/netrunner/utils"
	"github.com/luxfi/protocol/p/txs"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/sdk/wallet/chain/c"
	"github.com/luxfi/sdk/wallet/primary"
	"github.com/luxfi/sdk/wallet/primary/common"
	"github.com/luxfi/node/vms/secp256k1fx"
)

// ErrNoChainAuthKeysInWallet indicates the wallet doesn't contain required chain auth keys.
var ErrNoChainAuthKeysInWallet = errors.New("auth wallet does not contain chain auth keys")

// PublicDeployer handles chain deployment to public networks.
type PublicDeployer struct {
	LocalDeployer
	usingLedger bool
	kc          keychain.Keychain
	network     models.Network
	app         *application.Lux
}

// NewPublicDeployer creates a new PublicDeployer instance.
func NewPublicDeployer(app *application.Lux, usingLedger bool, kc keychain.Keychain, network models.Network) *PublicDeployer {
	return &PublicDeployer{
		LocalDeployer: *NewLocalDeployer(app, "", ""),
		app:           app,
		usingLedger:   usingLedger,
		kc:            kc,
		network:       network,
	}
}

// AddValidator adds a chain validator to the given chainID.
// It creates an add chain validator tx, signs it with the wallet,
// and if fully signed, issues it. If partially signed, returns the tx for additional signatures.
func (d *PublicDeployer) AddValidator(
	controlKeys []string,
	chainAuthKeysStrs []string,
	chainID ids.ID,
	nodeID ids.NodeID,
	weight uint64,
	startTime time.Time,
	duration time.Duration,
) (bool, *txs.Tx, []string, error) {
	wallet, err := d.loadWallet(chainID)
	if err != nil {
		return false, nil, nil, err
	}
	chainAuthKeys, err := address.ParseToIDs(chainAuthKeysStrs)
	if err != nil {
		return false, nil, nil, fmt.Errorf("failure parsing chain auth keys: %w", err)
	}
	validator := &txs.ChainValidator{
		Validator: txs.Validator{
			NodeID: nodeID,
			Start:  uint64(startTime.Unix()),               //nolint:gosec // G115: Unix time is positive
			End:    uint64(startTime.Add(duration).Unix()), //nolint:gosec // G115: Unix time is positive
			Wght:   weight,
		},
		Chain: chainID,
	}
	if d.usingLedger {
		ux.Logger.PrintToUser("*** Please sign ChainValidator transaction on the ledger device *** ")
	}

	tx, err := d.createAddChainValidatorTx(chainAuthKeys, validator, wallet)
	if err != nil {
		return false, nil, nil, err
	}

	_, remainingChainAuthKeys, err := txutils.GetRemainingSigners(tx, controlKeys)
	if err != nil {
		return false, nil, nil, err
	}
	isFullySigned := len(remainingChainAuthKeys) == 0

	if isFullySigned {
		id, err := d.Commit(tx)
		if err != nil {
			return false, nil, nil, err
		}
		ux.Logger.PrintToUser("Transaction successful, transaction ID: %s", id)
		return true, nil, nil, nil
	}

	ux.Logger.PrintToUser("Partial tx created")
	return false, tx, remainingChainAuthKeys, nil
}

// CreateAssetTx creates a new asset on the X-Chain.
func (d *PublicDeployer) CreateAssetTx(
	chainID ids.ID,
	tokenName string,
	tokenSymbol string,
	denomination byte,
	initialState map[uint32][]verify.State,
) (ids.ID, error) {
	wallet, err := d.loadWallet(chainID)
	if err != nil {
		return ids.Empty, err
	}

	if d.usingLedger {
		ux.Logger.PrintToUser("*** Please sign Create Asset Transaction hash on the ledger device *** ")
	}

	tx, err := wallet.X().IssueCreateAssetTx(tokenName, tokenSymbol, denomination, initialState)
	if err != nil {
		return ids.Empty, err
	}
	ux.Logger.PrintToUser("Create Asset Transaction successful, transaction ID: %s", tx.ID())
	ux.Logger.PrintToUser("Now exporting asset to P-Chain ...")
	return tx.ID(), err
}

// ExportToPChainTx exports assets from X-Chain to P-Chain.
func (d *PublicDeployer) ExportToPChainTx(
	chainID ids.ID,
	chainAssetID ids.ID,
	owner *secp256k1fx.OutputOwners,
	assetAmount uint64,
) (ids.ID, error) {
	wallet, err := d.loadWallet(chainID)
	if err != nil {
		return ids.Empty, err
	}

	if d.usingLedger {
		ux.Logger.PrintToUser("*** Please sign X -> P Chain Export Transaction hash on the ledger device *** ")
	}

	tx, err := wallet.X().IssueExportTx(ids.Empty,
		[]*lux.TransferableOutput{
			{
				Asset: lux.Asset{
					ID: chainAssetID,
				},
				Out: &secp256k1fx.TransferOutput{
					Amt:          assetAmount,
					OutputOwners: *owner,
				},
			},
		})
	if err == nil {
		ux.Logger.PrintToUser("Export to P-Chain Transaction successful, transaction ID: %s", tx.ID())
		ux.Logger.PrintToUser("Now importing asset from X-Chain ...")
	}
	return tx.ID(), err
}

// ImportFromXChain imports assets from X-Chain to P-Chain.
func (d *PublicDeployer) ImportFromXChain(
	chainID ids.ID,
	owner *secp256k1fx.OutputOwners,
) (ids.ID, error) {
	wallet, err := d.loadWallet(chainID)
	if err != nil {
		return ids.Empty, err
	}
	if d.usingLedger {
		ux.Logger.PrintToUser("*** Please sign X -> P Chain Import Transaction hash on the ledger device *** ")
	}
	xChainID := ids.FromStringOrPanic("2oYMBNV4eNHyqk2fjjV5nVQLDbtmNJzq5s3qs3Lo6ftnC6FByM") // X-Chain ID

	tx, err := wallet.P().IssueImportTx(xChainID, owner)
	if err == nil {
		ux.Logger.PrintToUser("Import from X Chain Transaction successful, transaction ID: %s", tx.ID())
		ux.Logger.PrintToUser("Now transforming net into elastic net ...")
	}
	if err != nil {
		return ids.Empty, err
	}
	return tx.ID(), err
}

// TransformChainTx transforms a chain to a permissionless elastic chain.
func (d *PublicDeployer) TransformChainTx(
	controlKeys []string,
	chainAuthKeysStrs []string,
	elasticChainConfig climodels.ElasticChainConfig,
	chainID ids.ID,
	chainAssetID ids.ID,
) (bool, ids.ID, *txs.Tx, []string, error) {
	wallet, err := d.loadWallet(chainID)
	if err != nil {
		return false, ids.Empty, nil, nil, err
	}
	chainAuthKeys, err := address.ParseToIDs(chainAuthKeysStrs)
	if err != nil {
		return false, ids.Empty, nil, nil, fmt.Errorf("failure parsing chain auth keys: %w", err)
	}

	if d.usingLedger {
		ux.Logger.PrintToUser("*** Please sign Transform Net hash on the ledger device *** ")
	}

	tx, err := d.createTransformChainTX(chainAuthKeys, elasticChainConfig, wallet, chainAssetID)
	if err != nil {
		return false, ids.Empty, nil, nil, err
	}
	_, remainingChainAuthKeys, err := txutils.GetRemainingSigners(tx, controlKeys)
	if err != nil {
		return false, ids.Empty, nil, nil, err
	}
	isFullySigned := len(remainingChainAuthKeys) == 0

	if isFullySigned {
		txID, err := d.Commit(tx)
		if err != nil {
			return false, ids.Empty, nil, nil, err
		}
		ux.Logger.PrintToUser("Transaction successful, transaction ID: %s", txID)
		return true, txID, nil, nil, nil
	}

	ux.Logger.PrintToUser("Partial tx created")
	return false, ids.Empty, tx, remainingChainAuthKeys, nil
}

// RemoveValidator removes a chain validator from the given chain.
// It verifies that the wallet is one of the chain auth keys (so as to sign the tx).
// If operation is multisig (len(chainAuthKeysStrs) > 1), it creates a remove
// chain validator tx and sets the change output owner to be a wallet address.
func (d *PublicDeployer) RemoveValidator(
	controlKeys []string,
	chainAuthKeysStrs []string,
	chainID ids.ID,
	nodeID ids.NodeID,
) (bool, *txs.Tx, []string, error) {
	wallet, err := d.loadWallet(chainID)
	if err != nil {
		return false, nil, nil, err
	}
	chainAuthKeys, err := address.ParseToIDs(chainAuthKeysStrs)
	if err != nil {
		return false, nil, nil, fmt.Errorf("failure parsing chain auth keys: %w", err)
	}

	if d.usingLedger {
		ux.Logger.PrintToUser("*** Please sign tx hash on the ledger device *** ")
	}

	tx, err := d.createRemoveValidatorTX(chainAuthKeys, nodeID, chainID, wallet)
	if err != nil {
		return false, nil, nil, err
	}

	_, remainingChainAuthKeys, err := txutils.GetRemainingSigners(tx, controlKeys)
	if err != nil {
		return false, nil, nil, err
	}
	isFullySigned := len(remainingChainAuthKeys) == 0

	if isFullySigned {
		id, err := d.Commit(tx)
		if err != nil {
			return false, nil, nil, err
		}
		ux.Logger.PrintToUser("Transaction successful, transaction ID: %s", id)
		return true, nil, nil, nil
	}

	ux.Logger.PrintToUser("Partial tx created")
	return false, tx, remainingChainAuthKeys, nil
}

// DeployChain creates a chain using the given control keys and threshold.
func (d *PublicDeployer) DeployChain(
	controlKeys []string,
	threshold uint32,
) (ids.ID, error) {
	ux.Logger.PrintToUser("DeployNet: starting...")
	wallet, err := d.loadWallet()
	if err != nil {
		return ids.Empty, err
	}
	ux.Logger.PrintToUser("DeployNet: calling createNetTx...")
	chainID, err := d.createChainTx(controlKeys, threshold, wallet)
	if err != nil {
		ux.Logger.PrintToUser("DeployNet: createNetTx error: %v", err)
		return ids.Empty, err
	}
	ux.Logger.PrintToUser("Net has been created with ID: %s", chainID.String())
	time.Sleep(2 * time.Second)
	return chainID, nil
}

// DeployBlockchain creates a blockchain for the given chain.
// It creates a create blockchain tx and sets the change output owner
// to be a wallet address (if not, it may go to any other chain auth address).
func (d *PublicDeployer) DeployBlockchain(
	controlKeys []string,
	chainAuthKeysStrs []string,
	chainID ids.ID,
	chain string,
	genesis []byte,
) (bool, ids.ID, *txs.Tx, []string, error) {
	ux.Logger.PrintToUser("Now creating blockchain...")

	wallet, err := d.loadWallet(chainID)
	if err != nil {
		return false, ids.Empty, nil, nil, err
	}

	vmID, err := utils.VMID(chain)
	if err != nil {
		return false, ids.Empty, nil, nil, fmt.Errorf("failed to create VM ID from %s: %w", chain, err)
	}

	chainAuthKeys, err := address.ParseToIDs(chainAuthKeysStrs)
	if err != nil {
		return false, ids.Empty, nil, nil, fmt.Errorf("failure parsing chain auth keys: %w", err)
	}

	if d.usingLedger {
		ux.Logger.PrintToUser("*** Please sign CreateChain transaction on the ledger device *** ")
	}

	tx, err := d.createBlockchainTx(chainAuthKeys, chain, vmID, chainID, genesis, wallet)
	if err != nil {
		return false, ids.Empty, nil, nil, err
	}

	_, remainingChainAuthKeys, err := txutils.GetRemainingSigners(tx, controlKeys)
	if err != nil {
		return false, ids.Empty, nil, nil, err
	}
	isFullySigned := len(remainingChainAuthKeys) == 0

	id := ids.Empty
	if isFullySigned {
		id, err = d.Commit(tx)
		if err != nil {
			return false, ids.Empty, nil, nil, err
		}
	}

	return isFullySigned, id, tx, remainingChainAuthKeys, nil
}

// Commit issues a fully signed transaction to the network.
func (d *PublicDeployer) Commit(
	tx *txs.Tx,
) (ids.ID, error) {
	wallet, err := d.loadWallet()
	if err != nil {
		return ids.Empty, err
	}
	err = wallet.P().IssueTx(tx)
	if err != nil {
		return ids.Empty, err
	}
	return tx.ID(), nil
}

// Sign signs a transaction with the wallet's keys.
func (d *PublicDeployer) Sign(
	tx *txs.Tx,
	chainAuthKeysStrs []string,
	chain ids.ID,
) error {
	wallet, err := d.loadWallet(chain)
	if err != nil {
		return err
	}
	chainAuthKeys, err := address.ParseToIDs(chainAuthKeysStrs)
	if err != nil {
		return fmt.Errorf("failure parsing chain auth keys: %w", err)
	}
	if ok := d.checkWalletHasChainAuthAddresses(chainAuthKeys); !ok {
		return ErrNoChainAuthKeysInWallet
	}
	if d.usingLedger {
		txName := txutils.GetLedgerDisplayName(tx)
		if len(txName) == 0 {
			ux.Logger.PrintToUser("*** Please sign tx hash on the ledger device *** ")
		} else {
			ux.Logger.PrintToUser("*** Please sign %s transaction on the ledger device *** ", txName)
		}
	}
	if err := d.signTx(tx, wallet); err != nil {
		return err
	}
	return nil
}

func (d *PublicDeployer) loadWallet(preloadTxs ...ids.ID) (primary.Wallet, error) {
	ctx := context.Background()
	ux.Logger.PrintToUser("loadWallet: starting...")

	var api string
	switch d.network {
	case models.Testnet:
		api = constants.TestnetAPIEndpoint
	case models.Mainnet:
		api = constants.MainnetAPIEndpoint
	case models.Local:
		// used for E2E testing of public related paths
		api = constants.LocalAPIEndpoint
	default:
		return nil, fmt.Errorf("unsupported public network")
	}
	ux.Logger.PrintToUser("loadWallet: using API endpoint %s", api)

	// Create empty EthKeychain if kc doesn't implement it
	var ethKc c.EthKeychain
	if ekc, ok := d.kc.(c.EthKeychain); ok {
		ethKc = ekc
	} else {
		// Create a minimal EthKeychain implementation
		ethKc = &emptyEthKeychain{}
	}

	// Build the set of P-Chain transactions to fetch (e.g., chain creation txs)
	// This is needed so the wallet knows about chain owners when creating blockchain txs
	pChainTxsToFetch := set.Set[ids.ID]{}
	for _, txID := range preloadTxs {
		pChainTxsToFetch.Add(txID)
	}

	ux.Logger.PrintToUser("loadWallet: creating P-Chain wallet...")
	// Use P-Chain only wallet since our X-Chain uses exchangevm which doesn't
	// support standard AVM API methods.
	wallet, err := primary.MakePChainWallet(ctx, &primary.WalletConfig{
		URI:              api,
		LUXKeychain:      keychainwrapper.WrapCryptoKeychain(d.kc),
		EthKeychain:      ethKc,
		PChainTxsToFetch: pChainTxsToFetch,
	})
	if err != nil {
		ux.Logger.PrintToUser("loadWallet: error creating wallet: %v", err)
		return nil, err
	}
	ux.Logger.PrintToUser("loadWallet: wallet created successfully")
	return wallet, nil
}

func (d *PublicDeployer) getMultisigTxOptions(chainAuthKeys []ids.ShortID) []common.Option {
	options := []common.Option{}
	walletAddr := d.kc.Addresses().List()[0]
	// addrs to use for signing
	customAddrsSet := set.Set[ids.ShortID]{}
	customAddrsSet.Add(walletAddr)
	customAddrsSet.Add(chainAuthKeys...)
	options = append(options, common.WithCustomAddresses(customAddrsSet))
	// set change to go to wallet addr (instead of any other chain auth key)
	changeOwner := &secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs:     []ids.ShortID{walletAddr},
	}
	options = append(options, common.WithChangeOwner(changeOwner))
	return options
}

func (d *PublicDeployer) createBlockchainTx(
	chainAuthKeys []ids.ShortID,
	chainName string,
	vmID,
	chainID ids.ID,
	genesis []byte,
	wallet primary.Wallet,
) (*txs.Tx, error) {
	fxIDs := make([]ids.ID, 0)
	options := d.getMultisigTxOptions(chainAuthKeys)
	// create tx
	unsignedTx, err := wallet.P().Builder().NewCreateChainTx(
		chainID,
		genesis,
		vmID,
		fxIDs,
		chainName,
		options...,
	)
	if err != nil {
		return nil, err
	}
	tx := txs.Tx{Unsigned: unsignedTx}
	// sign with current wallet
	if err := wallet.P().Signer().Sign(context.Background(), &tx); err != nil {
		return nil, err
	}
	return &tx, nil
}

func (d *PublicDeployer) createAddChainValidatorTx(
	chainAuthKeys []ids.ShortID,
	validator *txs.ChainValidator,
	wallet primary.Wallet,
) (*txs.Tx, error) {
	options := d.getMultisigTxOptions(chainAuthKeys)
	// create tx
	unsignedTx, err := wallet.P().Builder().NewAddChainValidatorTx(validator, options...)
	if err != nil {
		return nil, err
	}
	tx := txs.Tx{Unsigned: unsignedTx}
	// sign with current wallet
	if err := wallet.P().Signer().Sign(context.Background(), &tx); err != nil {
		return nil, err
	}
	return &tx, nil
}

func (d *PublicDeployer) createRemoveValidatorTX(
	chainAuthKeys []ids.ShortID,
	nodeID ids.NodeID,
	chainID ids.ID,
	wallet primary.Wallet,
) (*txs.Tx, error) {
	options := d.getMultisigTxOptions(chainAuthKeys)
	// create tx
	unsignedTx, err := wallet.P().Builder().NewRemoveChainValidatorTx(nodeID, chainID, options...)
	if err != nil {
		return nil, err
	}
	tx := txs.Tx{Unsigned: unsignedTx}
	// sign with current wallet
	if err := wallet.P().Signer().Sign(context.Background(), &tx); err != nil {
		return nil, err
	}
	return &tx, nil
}

func (d *PublicDeployer) createTransformChainTX(
	chainAuthKeys []ids.ShortID,
	elasticChainConfig climodels.ElasticChainConfig,
	wallet primary.Wallet,
	assetID ids.ID,
) (*txs.Tx, error) {
	options := d.getMultisigTxOptions(chainAuthKeys)
	// create tx
	unsignedTx, err := wallet.P().Builder().NewTransformChainTx(elasticChainConfig.ChainID, assetID,
		elasticChainConfig.InitialSupply, elasticChainConfig.MaxSupply, elasticChainConfig.MinConsumptionRate,
		elasticChainConfig.MaxConsumptionRate, elasticChainConfig.MinValidatorStake, elasticChainConfig.MaxValidatorStake,
		elasticChainConfig.MinStakeDuration, elasticChainConfig.MaxStakeDuration, elasticChainConfig.MinDelegationFee,
		elasticChainConfig.MinDelegatorStake, elasticChainConfig.MaxValidatorWeightFactor, elasticChainConfig.UptimeRequirement, options...)
	if err != nil {
		return nil, err
	}
	tx := txs.Tx{Unsigned: unsignedTx}
	// sign with current wallet
	if err := wallet.P().Signer().Sign(context.Background(), &tx); err != nil {
		return nil, err
	}
	return &tx, nil
}

// ConvertL1 converts a chain to an L1 (LP99)
func (d *PublicDeployer) ConvertL1(
	controlKeys []string,
	chainAuthKeysStrs []string,
	chainID ids.ID,
	blockchainID ids.ID,
	managerAddress ethcommon.Address,
	validators []interface{}, // []*txs.ConvertChainToL1Validator
) (bool, ids.ID, *txs.Tx, []string, error) {
	ux.Logger.PrintToUser("Now calling ConvertChainToL1Tx...")

	// Get wallet
	wallet, err := d.loadWallet(chainID)
	if err != nil {
		return false, ids.Empty, nil, nil, err
	}

	chainAuthKeys, err := address.ParseToIDs(chainAuthKeysStrs)
	if err != nil {
		return false, ids.Empty, nil, nil, fmt.Errorf("failure parsing auth keys: %w", err)
	}

	// Convert []interface{} to []*txs.ConvertChainToL1Validator
	convertValidators := make([]*txs.ConvertChainToL1Validator, 0, len(validators))
	for _, v := range validators {
		validator, ok := v.(*txs.ConvertChainToL1Validator)
		if !ok {
			return false, ids.Empty, nil, nil, fmt.Errorf("invalid validator type: expected *txs.ConvertChainToL1Validator, got %T", v)
		}
		convertValidators = append(convertValidators, validator)
	}

	// Build ConvertChainToL1Tx using the wallet builder
	options := d.getMultisigTxOptions(chainAuthKeys)

	unsignedTx, err := wallet.P().Builder().NewConvertChainToL1Tx(
		chainID,
		blockchainID,
		managerAddress.Bytes(),
		convertValidators,
		options...,
	)
	if err != nil {
		return false, ids.Empty, nil, nil, fmt.Errorf("error building ConvertChainToL1Tx: %w", err)
	}

	tx := txs.Tx{Unsigned: unsignedTx}
	ctx, cancel := context.WithTimeout(context.Background(), constants.RequestTimeout)
	defer cancel()
	if err := wallet.P().Signer().Sign(ctx, &tx); err != nil {
		return false, ids.Empty, nil, nil, fmt.Errorf("error signing tx: %w", err)
	}

	_, remainingChainAuthKeys, err := txutils.GetRemainingSigners(&tx, controlKeys)
	if err != nil {
		return false, ids.Empty, nil, nil, err
	}

	if len(remainingChainAuthKeys) == 0 {
		// Commit the transaction
		txID, err := d.Commit(&tx)
		if err != nil {
			return false, ids.Empty, nil, nil, err
		}
		return true, txID, &tx, remainingChainAuthKeys, nil
	}
	return false, ids.Empty, &tx, remainingChainAuthKeys, nil
}

func (*PublicDeployer) signTx(
	tx *txs.Tx,
	wallet primary.Wallet,
) error {
	if err := wallet.P().Signer().Sign(context.Background(), tx); err != nil {
		return err
	}
	return nil
}

func (d *PublicDeployer) createChainTx(controlKeys []string, threshold uint32, wallet primary.Wallet) (ids.ID, error) {
	ux.Logger.PrintToUser("createChainTx: starting with control keys: %v", controlKeys)
	addrs, err := address.ParseToIDs(controlKeys)
	if err != nil {
		return ids.Empty, fmt.Errorf("failure parsing control keys: %w", err)
	}
	ux.Logger.PrintToUser("createChainTx: parsed addresses: %v", addrs)
	owners := &secp256k1fx.OutputOwners{
		Addrs:     addrs,
		Threshold: threshold,
		Locktime:  0,
	}
	opts := []common.Option{}
	if d.usingLedger {
		ux.Logger.PrintToUser("*** Please sign CreateChain transaction on the ledger device *** ")
	}
	ux.Logger.PrintToUser("createNetworkTx: calling IssueCreateNetworkTx...")
	tx, err := wallet.P().IssueCreateNetworkTx(owners, opts...)
	if err != nil {
		ux.Logger.PrintToUser("createNetworkTx: IssueCreateNetworkTx error: %v", err)
		return ids.Empty, err
	}
	ux.Logger.PrintToUser("createNetworkTx: tx issued successfully with ID: %s", tx.ID().String())
	return tx.ID(), nil
}

func (d *PublicDeployer) getChainAuthAddressesInWallet(chainAuth []ids.ShortID) []ids.ShortID {
	walletAddrs := d.kc.Addresses().List()
	chainAuthInWallet := []ids.ShortID{}
	for _, walletAddr := range walletAddrs {
		for _, addr := range chainAuth {
			if addr == walletAddr {
				chainAuthInWallet = append(chainAuthInWallet, addr)
			}
		}
	}
	return chainAuthInWallet
}

// check that the wallet at least contain one chain auth address
func (d *PublicDeployer) checkWalletHasChainAuthAddresses(chainAuth []ids.ShortID) bool {
	addrs := d.getChainAuthAddressesInWallet(chainAuth)
	return len(addrs) != 0
}

// IsChainValidator checks if a node is a validator for the given chain.
func IsChainValidator(chainID ids.ID, nodeID ids.NodeID, network models.Network) (bool, error) {
	var apiURL string
	switch network {
	case models.Mainnet:
		apiURL = constants.MainnetAPIEndpoint
	case models.Testnet:
		apiURL = constants.TestnetAPIEndpoint
	default:
		return false, fmt.Errorf("invalid network: %s", network)
	}
	pClient := platformvm.NewClient(apiURL)
	ctx, cancel := context.WithTimeout(context.Background(), constants.E2ERequestTimeout)
	defer cancel()

	vals, err := pClient.GetCurrentValidators(ctx, chainID, []ids.NodeID{nodeID})
	if err != nil {
		return false, fmt.Errorf("failed to get current validators")
	}

	return len(vals) != 0, nil
}

// GetPublicChainValidators returns the validators for a chain on a public network.
func GetPublicChainValidators(chainID ids.ID, network models.Network) ([]platformvm.ClientPermissionlessValidator, error) {
	var apiURL string
	switch network {
	case models.Mainnet:
		apiURL = constants.MainnetAPIEndpoint
	case models.Testnet:
		apiURL = constants.TestnetAPIEndpoint
	default:
		return nil, fmt.Errorf("invalid network: %s", network)
	}
	pClient := platformvm.NewClient(apiURL)
	ctx, cancel := context.WithTimeout(context.Background(), constants.E2ERequestTimeout)
	defer cancel()

	vals, err := pClient.GetCurrentValidators(ctx, chainID, []ids.NodeID{})
	if err != nil {
		return nil, fmt.Errorf("failed to get current validators")
	}

	return vals, nil
}

// ValidateChainNameAndGetChains validates a chain name and returns chain information
func ValidateChainNameAndGetChains(chainName string) error {
	// Basic validation - can be expanded later
	if chainName == "" {
		return fmt.Errorf("chain name cannot be empty")
	}
	return nil
}

// IncreaseValidatorPChainBalance increases a validator's balance on P-chain.
func (d *PublicDeployer) IncreaseValidatorPChainBalance(
	_ ids.ID, // validationID reserved for future use
	balance uint64,
) error {
	wallet, err := d.loadWallet()
	if err != nil {
		return err
	}

	// Create a base transaction to transfer funds to increase validator balance
	// Use the network's native asset ID (LUX)
	luxAssetID := ids.Empty
	if d.network.ID() == constants.MainnetID || d.network.ID() == constants.TestnetID {
		luxAssetID = ids.Empty // Native asset on mainnet/testnet
	}

	outputs := []*lux.TransferableOutput{
		{
			Asset: lux.Asset{ID: luxAssetID},
			Out: &secp256k1fx.TransferOutput{
				Amt: balance,
				OutputOwners: secp256k1fx.OutputOwners{
					Threshold: 1,
					Addrs:     []ids.ShortID{},
				},
			},
		},
	}

	tx, err := wallet.P().IssueBaseTx(outputs)
	if err != nil {
		return fmt.Errorf("failed to create balance increase transaction: %w", err)
	}

	ux.Logger.PrintToUser("Increased validator balance by %d nLUX", balance)
	ux.Logger.PrintToUser("Transaction ID: %s", tx.ID())
	return nil
}

// GetDefaultChainAirdropKeyInfo returns the default airdrop key information for a chain.
func GetDefaultChainAirdropKeyInfo(_ *application.Lux, _ string) (string, string, string, error) {
	// Return empty values for now - this would typically read from sidecar
	return "", "", "", nil
}
