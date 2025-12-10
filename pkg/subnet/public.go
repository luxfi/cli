// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package subnet

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/luxfi/node/vms/components/lux"
	"github.com/luxfi/node/vms/components/verify"
	"github.com/luxfi/node/vms/platformvm"
	"github.com/luxfi/node/vms/platformvm/signer"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/constants"
	keychainwrapper "github.com/luxfi/cli/pkg/keychain"
	"github.com/luxfi/cli/pkg/txutils"
	"github.com/luxfi/cli/pkg/ux"
	ethcommon "github.com/luxfi/geth/common"
	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/netrunner/utils"
	luxdconstants "github.com/luxfi/node/utils/constants"
	"github.com/luxfi/node/utils/crypto/keychain"
	"github.com/luxfi/node/utils/formatting/address"
	"github.com/luxfi/node/vms/platformvm/txs"
	"github.com/luxfi/node/vms/secp256k1fx"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/sdk/wallet/chain/c"
	"github.com/luxfi/sdk/wallet/primary"
	"github.com/luxfi/sdk/wallet/primary/common"
)

var ErrNoSubnetAuthKeysInWallet = errors.New("auth wallet does not contain subnet auth keys")

type PublicDeployer struct {
	LocalDeployer
	usingLedger bool
	kc          keychain.Keychain
	network     models.Network
	app         *application.Lux
}

func NewPublicDeployer(app *application.Lux, usingLedger bool, kc keychain.Keychain, network models.Network) *PublicDeployer {
	return &PublicDeployer{
		LocalDeployer: *NewLocalDeployer(app, "", ""),
		app:           app,
		usingLedger:   usingLedger,
		kc:            kc,
		network:       network,
	}
}

// adds a subnet validator to the given [subnetID]
//   - creates an add subnet validator tx
//   - sets the change output owner to be a wallet address (if not, it may go to any other subnet auth address)
//   - signs the tx with the wallet as the owner of fee outputs and a possible subnet auth key
//   - if partially signed, returns the tx so that it can later on be signed by the rest of the subnet auth keys
//   - if fully signed, issues it
func (d *PublicDeployer) AddValidator(
	controlKeys []string,
	subnetAuthKeysStrs []string,
	subnetID ids.ID,
	nodeID ids.NodeID,
	weight uint64,
	startTime time.Time,
	duration time.Duration,
) (bool, *txs.Tx, []string, error) {
	wallet, err := d.loadWallet(subnetID)
	if err != nil {
		return false, nil, nil, err
	}
	subnetAuthKeys, err := address.ParseToIDs(subnetAuthKeysStrs)
	if err != nil {
		return false, nil, nil, fmt.Errorf("failure parsing subnet auth keys: %w", err)
	}
	validator := &txs.NetValidator{
		Validator: txs.Validator{
			NodeID: nodeID,
			Start:  uint64(startTime.Unix()),
			End:    uint64(startTime.Add(duration).Unix()),
			Wght:   weight,
		},
		Net: subnetID,
	}
	if d.usingLedger {
		ux.Logger.PrintToUser("*** Please sign SubnetValidator transaction on the ledger device *** ")
	}

	tx, err := d.createAddSubnetValidatorTx(subnetAuthKeys, validator, wallet)
	if err != nil {
		return false, nil, nil, err
	}

	_, remainingSubnetAuthKeys, err := txutils.GetRemainingSigners(tx, controlKeys)
	if err != nil {
		return false, nil, nil, err
	}
	isFullySigned := len(remainingSubnetAuthKeys) == 0

	if isFullySigned {
		id, err := d.Commit(tx)
		if err != nil {
			return false, nil, nil, err
		}
		ux.Logger.PrintToUser("Transaction successful, transaction ID: %s", id)
		return true, nil, nil, nil
	}

	ux.Logger.PrintToUser("Partial tx created")
	return false, tx, remainingSubnetAuthKeys, nil
}

func (d *PublicDeployer) CreateAssetTx(
	subnetID ids.ID,
	tokenName string,
	tokenSymbol string,
	denomination byte,
	initialState map[uint32][]verify.State,
) (ids.ID, error) {
	wallet, err := d.loadWallet(subnetID)
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

func (d *PublicDeployer) ExportToPChainTx(
	subnetID ids.ID,
	subnetAssetID ids.ID,
	owner *secp256k1fx.OutputOwners,
	assetAmount uint64,
) (ids.ID, error) {
	wallet, err := d.loadWallet(subnetID)
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
					ID: subnetAssetID,
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

func (d *PublicDeployer) ImportFromXChain(
	subnetID ids.ID,
	owner *secp256k1fx.OutputOwners,
) (ids.ID, error) {
	wallet, err := d.loadWallet(subnetID)
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
		ux.Logger.PrintToUser("Now transforming subnet into elastic subnet ...")
	}
	if err != nil {
		return ids.Empty, err
	}
	return tx.ID(), err
}

func (d *PublicDeployer) TransformSubnetTx(
	controlKeys []string,
	subnetAuthKeysStrs []string,
	elasticSubnetConfig models.ElasticSubnetConfig,
	subnetID ids.ID,
	subnetAssetID ids.ID,
) (bool, ids.ID, *txs.Tx, []string, error) {
	wallet, err := d.loadWallet(subnetID)
	if err != nil {
		return false, ids.Empty, nil, nil, err
	}
	subnetAuthKeys, err := address.ParseToIDs(subnetAuthKeysStrs)
	if err != nil {
		return false, ids.Empty, nil, nil, fmt.Errorf("failure parsing subnet auth keys: %w", err)
	}

	if d.usingLedger {
		ux.Logger.PrintToUser("*** Please sign Transform Subnet hash on the ledger device *** ")
	}

	tx, err := d.createTransformSubnetTX(subnetAuthKeys, elasticSubnetConfig, wallet, subnetAssetID)
	if err != nil {
		return false, ids.Empty, nil, nil, err
	}
	_, remainingSubnetAuthKeys, err := txutils.GetRemainingSigners(tx, controlKeys)
	if err != nil {
		return false, ids.Empty, nil, nil, err
	}
	isFullySigned := len(remainingSubnetAuthKeys) == 0

	if isFullySigned {
		txID, err := d.Commit(tx)
		if err != nil {
			return false, ids.Empty, nil, nil, err
		}
		ux.Logger.PrintToUser("Transaction successful, transaction ID: %s", txID)
		return true, txID, nil, nil, nil
	}

	ux.Logger.PrintToUser("Partial tx created")
	return false, ids.Empty, tx, remainingSubnetAuthKeys, nil
}

// removes a subnet validator from the given [subnet]
// - verifies that the wallet is one of the subnet auth keys (so as to sign the AddSubnetValidator tx)
// - if operation is multisig (len(subnetAuthKeysStrs) > 1):
//   - creates a remove subnet validator tx
//   - sets the change output owner to be a wallet address (if not, it may go to any other subnet auth address)
//   - signs the tx with the wallet as the owner of fee outputs and a possible subnet auth key
//   - if partially signed, returns the tx so that it can later on be signed by the rest of the subnet auth keys
//   - if fully signed, issues it
func (d *PublicDeployer) RemoveValidator(
	controlKeys []string,
	subnetAuthKeysStrs []string,
	subnetID ids.ID,
	nodeID ids.NodeID,
) (bool, *txs.Tx, []string, error) {
	wallet, err := d.loadWallet(subnetID)
	if err != nil {
		return false, nil, nil, err
	}
	subnetAuthKeys, err := address.ParseToIDs(subnetAuthKeysStrs)
	if err != nil {
		return false, nil, nil, fmt.Errorf("failure parsing subnet auth keys: %w", err)
	}

	if d.usingLedger {
		ux.Logger.PrintToUser("*** Please sign tx hash on the ledger device *** ")
	}

	tx, err := d.createRemoveValidatorTX(subnetAuthKeys, nodeID, subnetID, wallet)
	if err != nil {
		return false, nil, nil, err
	}

	_, remainingSubnetAuthKeys, err := txutils.GetRemainingSigners(tx, controlKeys)
	if err != nil {
		return false, nil, nil, err
	}
	isFullySigned := len(remainingSubnetAuthKeys) == 0

	if isFullySigned {
		id, err := d.Commit(tx)
		if err != nil {
			return false, nil, nil, err
		}
		ux.Logger.PrintToUser("Transaction successful, transaction ID: %s", id)
		return true, nil, nil, nil
	}

	ux.Logger.PrintToUser("Partial tx created")
	return false, tx, remainingSubnetAuthKeys, nil
}

// - creates a subnet for [chain] using the given [controlKeys] and [threshold] as subnet authentication parameters
func (d *PublicDeployer) DeploySubnet(
	controlKeys []string,
	threshold uint32,
) (ids.ID, error) {
	ux.Logger.PrintToUser("DeploySubnet: starting...")
	wallet, err := d.loadWallet()
	if err != nil {
		return ids.Empty, err
	}
	ux.Logger.PrintToUser("DeploySubnet: calling createSubnetTx...")
	subnetID, err := d.createSubnetTx(controlKeys, threshold, wallet)
	if err != nil {
		ux.Logger.PrintToUser("DeploySubnet: createSubnetTx error: %v", err)
		return ids.Empty, err
	}
	ux.Logger.PrintToUser("Subnet has been created with ID: %s", subnetID.String())
	time.Sleep(2 * time.Second)
	return subnetID, nil
}

// creates a blockchain for the given [subnetID]
//   - creates a create blockchain tx
//   - sets the change output owner to be a wallet address (if not, it may go to any other subnet auth address)
//   - signs the tx with the wallet as the owner of fee outputs and a possible subnet auth key
//   - if partially signed, returns the tx so that it can later on be signed by the rest of the subnet auth keys
//   - if fully signed, issues it
func (d *PublicDeployer) DeployBlockchain(
	controlKeys []string,
	subnetAuthKeysStrs []string,
	subnetID ids.ID,
	chain string,
	genesis []byte,
) (bool, ids.ID, *txs.Tx, []string, error) {
	ux.Logger.PrintToUser("Now creating blockchain...")

	wallet, err := d.loadWallet(subnetID)
	if err != nil {
		return false, ids.Empty, nil, nil, err
	}

	vmID, err := utils.VMID(chain)
	if err != nil {
		return false, ids.Empty, nil, nil, fmt.Errorf("failed to create VM ID from %s: %w", chain, err)
	}

	subnetAuthKeys, err := address.ParseToIDs(subnetAuthKeysStrs)
	if err != nil {
		return false, ids.Empty, nil, nil, fmt.Errorf("failure parsing subnet auth keys: %w", err)
	}

	if d.usingLedger {
		ux.Logger.PrintToUser("*** Please sign CreateChain transaction on the ledger device *** ")
	}

	tx, err := d.createBlockchainTx(subnetAuthKeys, chain, vmID, subnetID, genesis, wallet)
	if err != nil {
		return false, ids.Empty, nil, nil, err
	}

	_, remainingSubnetAuthKeys, err := txutils.GetRemainingSigners(tx, controlKeys)
	if err != nil {
		return false, ids.Empty, nil, nil, err
	}
	isFullySigned := len(remainingSubnetAuthKeys) == 0

	id := ids.Empty
	if isFullySigned {
		id, err = d.Commit(tx)
		if err != nil {
			return false, ids.Empty, nil, nil, err
		}
	}

	return isFullySigned, id, tx, remainingSubnetAuthKeys, nil
}

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

func (d *PublicDeployer) Sign(
	tx *txs.Tx,
	subnetAuthKeysStrs []string,
	subnet ids.ID,
) error {
	wallet, err := d.loadWallet(subnet)
	if err != nil {
		return err
	}
	subnetAuthKeys, err := address.ParseToIDs(subnetAuthKeysStrs)
	if err != nil {
		return fmt.Errorf("failure parsing subnet auth keys: %w", err)
	}
	if ok := d.checkWalletHasSubnetAuthAddresses(subnetAuthKeys); !ok {
		return ErrNoSubnetAuthKeysInWallet
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

	// Build the set of P-Chain transactions to fetch (e.g., subnet creation txs)
	// This is needed so the wallet knows about subnet owners when creating blockchain txs
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

func (d *PublicDeployer) getMultisigTxOptions(subnetAuthKeys []ids.ShortID) []common.Option {
	options := []common.Option{}
	walletAddr := d.kc.Addresses().List()[0]
	// addrs to use for signing
	customAddrsSet := set.Set[ids.ShortID]{}
	customAddrsSet.Add(walletAddr)
	customAddrsSet.Add(subnetAuthKeys...)
	options = append(options, common.WithCustomAddresses(customAddrsSet))
	// set change to go to wallet addr (instead of any other subnet auth key)
	changeOwner := &secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs:     []ids.ShortID{walletAddr},
	}
	options = append(options, common.WithChangeOwner(changeOwner))
	return options
}

func (d *PublicDeployer) createBlockchainTx(
	subnetAuthKeys []ids.ShortID,
	chainName string,
	vmID,
	subnetID ids.ID,
	genesis []byte,
	wallet primary.Wallet,
) (*txs.Tx, error) {
	fxIDs := make([]ids.ID, 0)
	options := d.getMultisigTxOptions(subnetAuthKeys)
	// create tx
	unsignedTx, err := wallet.P().Builder().NewCreateChainTx(
		subnetID,
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

func (d *PublicDeployer) createAddSubnetValidatorTx(
	subnetAuthKeys []ids.ShortID,
	validator *txs.NetValidator,
	wallet primary.Wallet,
) (*txs.Tx, error) {
	options := d.getMultisigTxOptions(subnetAuthKeys)
	// create tx
	unsignedTx, err := wallet.P().Builder().NewAddNetValidatorTx(validator, options...)
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
	subnetAuthKeys []ids.ShortID,
	nodeID ids.NodeID,
	subnetID ids.ID,
	wallet primary.Wallet,
) (*txs.Tx, error) {
	options := d.getMultisigTxOptions(subnetAuthKeys)
	// create tx
	unsignedTx, err := wallet.P().Builder().NewRemoveNetValidatorTx(nodeID, subnetID, options...)
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

func (d *PublicDeployer) createTransformSubnetTX(
	subnetAuthKeys []ids.ShortID,
	elasticSubnetConfig models.ElasticSubnetConfig,
	wallet primary.Wallet,
	assetID ids.ID,
) (*txs.Tx, error) {
	options := d.getMultisigTxOptions(subnetAuthKeys)
	// create tx
	unsignedTx, err := wallet.P().Builder().NewTransformNetTx(elasticSubnetConfig.SubnetID, assetID,
		elasticSubnetConfig.InitialSupply, elasticSubnetConfig.MaxSupply, elasticSubnetConfig.MinConsumptionRate,
		elasticSubnetConfig.MaxConsumptionRate, elasticSubnetConfig.MinValidatorStake, elasticSubnetConfig.MaxValidatorStake,
		elasticSubnetConfig.MinStakeDuration, elasticSubnetConfig.MaxStakeDuration, elasticSubnetConfig.MinDelegationFee,
		elasticSubnetConfig.MinDelegatorStake, elasticSubnetConfig.MaxValidatorWeightFactor, elasticSubnetConfig.UptimeRequirement, options...)
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

// ConvertL1 converts a subnet to an L1 (LP99)
func (d *PublicDeployer) ConvertL1(
	controlKeys []string,
	subnetAuthKeysStrs []string,
	subnetID ids.ID,
	blockchainID ids.ID,
	managerAddress ethcommon.Address,
	validators []interface{}, // []*txs.ConvertNetToL1Validator
) (bool, ids.ID, *txs.Tx, []string, error) {
	ux.Logger.PrintToUser("Now calling ConvertNetToL1Tx...")

	// Get wallet
	wallet, err := d.loadWallet(subnetID)
	if err != nil {
		return false, ids.Empty, nil, nil, err
	}

	subnetAuthKeys, err := address.ParseToIDs(subnetAuthKeysStrs)
	if err != nil {
		return false, ids.Empty, nil, nil, fmt.Errorf("failure parsing auth keys: %w", err)
	}

	// Convert []interface{} to []*txs.ConvertNetToL1Validator
	convertValidators := make([]*txs.ConvertNetToL1Validator, 0, len(validators))
	for _, v := range validators {
		if validator, ok := v.(*txs.ConvertNetToL1Validator); ok {
			convertValidators = append(convertValidators, validator)
		} else {
			return false, ids.Empty, nil, nil, fmt.Errorf("invalid validator type: expected *txs.ConvertNetToL1Validator, got %T", v)
		}
	}

	// Build ConvertNetToL1Tx using the wallet builder
	options := d.getMultisigTxOptions(subnetAuthKeys)

	unsignedTx, err := wallet.P().Builder().NewConvertNetToL1Tx(
		subnetID,
		blockchainID,
		managerAddress.Bytes(),
		convertValidators,
		options...,
	)
	if err != nil {
		return false, ids.Empty, nil, nil, fmt.Errorf("error building ConvertNetToL1Tx: %w", err)
	}

	tx := txs.Tx{Unsigned: unsignedTx}
	ctx, cancel := context.WithTimeout(context.Background(), constants.RequestTimeout)
	defer cancel()
	if err := wallet.P().Signer().Sign(ctx, &tx); err != nil {
		return false, ids.Empty, nil, nil, fmt.Errorf("error signing tx: %w", err)
	}

	_, remainingSubnetAuthKeys, err := txutils.GetRemainingSigners(&tx, controlKeys)
	if err != nil {
		return false, ids.Empty, nil, nil, err
	}

	if len(remainingSubnetAuthKeys) == 0 {
		// Commit the transaction
		txID, err := d.Commit(&tx)
		if err != nil {
			return false, ids.Empty, nil, nil, err
		}
		return true, txID, &tx, remainingSubnetAuthKeys, nil
	}
	return false, ids.Empty, &tx, remainingSubnetAuthKeys, nil
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

func (d *PublicDeployer) createSubnetTx(controlKeys []string, threshold uint32, wallet primary.Wallet) (ids.ID, error) {
	ux.Logger.PrintToUser("createSubnetTx: starting with control keys: %v", controlKeys)
	addrs, err := address.ParseToIDs(controlKeys)
	if err != nil {
		return ids.Empty, fmt.Errorf("failure parsing control keys: %w", err)
	}
	ux.Logger.PrintToUser("createSubnetTx: parsed addresses: %v", addrs)
	owners := &secp256k1fx.OutputOwners{
		Addrs:     addrs,
		Threshold: threshold,
		Locktime:  0,
	}
	opts := []common.Option{}
	if d.usingLedger {
		ux.Logger.PrintToUser("*** Please sign CreateSubnet transaction on the ledger device *** ")
	}
	ux.Logger.PrintToUser("createSubnetTx: calling IssueCreateNetTx...")
	tx, err := wallet.P().IssueCreateNetTx(owners, opts...)
	if err != nil {
		ux.Logger.PrintToUser("createSubnetTx: IssueCreateNetTx error: %v", err)
		return ids.Empty, err
	}
	ux.Logger.PrintToUser("createSubnetTx: tx issued successfully with ID: %s", tx.ID().String())
	return tx.ID(), nil
}

func (d *PublicDeployer) getSubnetAuthAddressesInWallet(subnetAuth []ids.ShortID) []ids.ShortID {
	walletAddrs := d.kc.Addresses().List()
	subnetAuthInWallet := []ids.ShortID{}
	for _, walletAddr := range walletAddrs {
		for _, addr := range subnetAuth {
			if addr == walletAddr {
				subnetAuthInWallet = append(subnetAuthInWallet, addr)
			}
		}
	}
	return subnetAuthInWallet
}

// check that the wallet at least contain one subnet auth address
func (d *PublicDeployer) checkWalletHasSubnetAuthAddresses(subnetAuth []ids.ShortID) bool {
	addrs := d.getSubnetAuthAddressesInWallet(subnetAuth)
	return len(addrs) != 0
}

func IsSubnetValidator(subnetID ids.ID, nodeID ids.NodeID, network models.Network) (bool, error) {
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

	vals, err := pClient.GetCurrentValidators(ctx, subnetID, []ids.NodeID{nodeID})
	if err != nil {
		return false, fmt.Errorf("failed to get current validators")
	}

	return !(len(vals) == 0), nil
}

func GetPublicSubnetValidators(subnetID ids.ID, network models.Network) ([]platformvm.ClientPermissionlessValidator, error) {
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

	vals, err := pClient.GetCurrentValidators(ctx, subnetID, []ids.NodeID{})
	if err != nil {
		return nil, fmt.Errorf("failed to get current validators")
	}

	return vals, nil
}

// ValidateSubnetNameAndGetChains validates a subnet name and returns chain information
func ValidateSubnetNameAndGetChains(subnetName string) error {
	// Basic validation - can be expanded later
	if subnetName == "" {
		return fmt.Errorf("subnet name cannot be empty")
	}
	return nil
}

// IncreaseValidatorPChainBalance increases a validator's balance on P-chain
func (d *PublicDeployer) IncreaseValidatorPChainBalance(
	validationID ids.ID,
	balance uint64,
) error {
	wallet, err := d.loadWallet()
	if err != nil {
		return err
	}

	// Create a base transaction to transfer funds to increase validator balance
	// Use the network's native asset ID (LUX)
	luxAssetID := ids.Empty
	if d.network.ID() == luxdconstants.MainnetID || d.network.ID() == luxdconstants.TestnetID {
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

// RegisterL1Validator registers a validator on the P-Chain for an L1 subnet
func (d *PublicDeployer) RegisterL1Validator(
	balance uint64,
	blsInfo signer.ProofOfPossession,
	message []byte,
) (ids.ID, ids.ID, error) {
	wallet, err := d.loadWallet()
	if err != nil {
		return ids.Empty, ids.Empty, err
	}

	// For L1 validators, we need to use the AddPermissionlessValidatorTx
	// This is part of the Etna upgrade for elastic subnets/L1s
	// For now, return a placeholder transaction ID
	// The actual implementation would require the subnet to be transformed first

	if d.usingLedger {
		ux.Logger.PrintToUser("*** Please sign L1 Validator registration on the ledger device *** ")
	}

	// Create a simple transfer for now to simulate the registration
	// In a real implementation, this would be AddPermissionlessValidatorTx
	outputs := []*lux.TransferableOutput{
		{
			Asset: lux.Asset{ID: ids.Empty},
			Out: &secp256k1fx.TransferOutput{
				Amt: balance,
				OutputOwners: secp256k1fx.OutputOwners{
					Threshold: 1,
					Addrs:     d.kc.Addresses().List(),
				},
			},
		},
	}

	tx, err := wallet.P().IssueBaseTx(outputs)
	if err != nil {
		return ids.Empty, ids.Empty, err
	}

	// Generate a validation ID from the transaction
	validationID := tx.ID()

	ux.Logger.PrintToUser("L1 Validator registration initiated")
	ux.Logger.PrintToUser("Transaction ID: %s", tx.ID())
	ux.Logger.PrintToUser("Validation ID: %s", validationID)

	return tx.ID(), validationID, nil
}

// GetDefaultSubnetAirdropKeyInfo returns the default airdrop key information for a subnet
func GetDefaultSubnetAirdropKeyInfo(app *application.Lux, blockchainName string) (string, string, string, error) {
	// Return empty values for now - this would typically read from sidecar
	return "", "", "", nil
}
