// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package blockchain

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/luxfi/evm/utils"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/cli/sdk/keychain"
	"github.com/luxfi/cli/sdk/network"
	"github.com/luxfi/cli/sdk/vm"
	"github.com/luxfi/cli/sdk/wallet"
	"github.com/luxfi/node/ids"
	"github.com/luxfi/node/utils/formatting/address"
	"github.com/luxfi/node/wallet/subnet/primary"
	"github.com/luxfi/evm/core"
	"github.com/luxfi/evm/params"
	"github.com/ethereum/go-ethereum/common"
)

func getDefaultSubnetEVMGenesis() SubnetParams {
	allocation := core.GenesisAlloc{}
	defaultAmount, _ := new(big.Int).SetString(vm.DefaultEvmAirdropAmount, 10)
	allocation[common.HexToAddress("INITIAL_ALLOCATION_ADDRESS")] = core.GenesisAccount{
		Balance: defaultAmount,
	}
	genesisBlock0Timestamp := utils.TimeToNewUint64(time.Now())
	return SubnetParams{
		SubnetEVM: &SubnetEVMParams{
			ChainID:     big.NewInt(123456),
			FeeConfig:   vm.StarterFeeConfig,
			Allocation:  allocation,
			Precompiles: params.Precompiles{},
			Timestamp:   genesisBlock0Timestamp,
		},
		Name: "TestSubnet",
	}
}

func TestSubnetDeploy(t *testing.T) {
	require := require.New(t)
	subnetParams := getDefaultSubnetEVMGenesis()
	newSubnet, err := New(&subnetParams)
	require.NoError(err)
	network := network.TestnetNetwork()

	keychain, err := keychain.NewKeychain(network, "KEY_PATH", nil)
	require.NoError(err)

	controlKeys := keychain.Addresses().List()
	subnetAuthKeys := keychain.Addresses().List()
	threshold := 1
	newSubnet.SetSubnetControlParams(controlKeys, uint32(threshold))
	wallet, err := wallet.New(
		context.Background(),
		network.Endpoint,
		keychain.Keychain,
		primary.WalletConfig{
			SubnetIDs: nil,
		},
	)
	require.NoError(err)
	deploySubnetTx, err := newSubnet.CreateSubnetTx(wallet)
	require.NoError(err)
	subnetID, err := newSubnet.Commit(*deploySubnetTx, wallet, true)
	require.NoError(err)
	fmt.Printf("subnetID %s \n", subnetID.String())
	time.Sleep(2 * time.Second)
	newSubnet.SetSubnetAuthKeys(subnetAuthKeys)
	deployChainTx, err := newSubnet.CreateBlockchainTx(wallet)
	require.NoError(err)
	blockchainID, err := newSubnet.Commit(*deployChainTx, wallet, true)
	require.NoError(err)
	fmt.Printf("blockchainID %s \n", blockchainID.String())
}

func TestSubnetDeployMultiSig(t *testing.T) {
	require := require.New(t)
	subnetParams := getDefaultSubnetEVMGenesis()
	newSubnet, _ := New(&subnetParams)
	network := network.TestnetNetwork()

	keychainA, err := keychain.NewKeychain(network, "KEY_PATH_A", nil)
	require.NoError(err)
	keychainB, err := keychain.NewKeychain(network, "KEY_PATH_B", nil)
	require.NoError(err)
	keychainC, err := keychain.NewKeychain(network, "KEY_PATH_C", nil)
	require.NoError(err)

	controlKeys := []ids.ShortID{}
	controlKeys = append(controlKeys, keychainA.Addresses().List()[0])
	controlKeys = append(controlKeys, keychainB.Addresses().List()[0])
	controlKeys = append(controlKeys, keychainC.Addresses().List()[0])

	subnetAuthKeys := []ids.ShortID{}
	subnetAuthKeys = append(subnetAuthKeys, keychainA.Addresses().List()[0])
	subnetAuthKeys = append(subnetAuthKeys, keychainB.Addresses().List()[0])
	threshold := 2
	newSubnet.SetSubnetControlParams(controlKeys, uint32(threshold))

	walletA, err := wallet.New(
		context.Background(),
		network.Endpoint,
		keychainA.Keychain,
		primary.WalletConfig{
			SubnetIDs: nil,
		},
	)
	require.NoError(err)

	deploySubnetTx, err := newSubnet.CreateSubnetTx(walletA)
	require.NoError(err)
	subnetID, err := newSubnet.Commit(*deploySubnetTx, walletA, true)
	require.NoError(err)
	fmt.Printf("subnetID %s \n", subnetID.String())

	// we need to wait to allow the transaction to reach other nodes in Testnet
	time.Sleep(2 * time.Second)

	newSubnet.SetSubnetAuthKeys(subnetAuthKeys)
	// first signature of CreateChainTx using keychain A
	deployChainTx, err := newSubnet.CreateBlockchainTx(walletA)
	require.NoError(err)

	// include subnetID in PChainTxsToFetch when creating second wallet
	walletB, err := wallet.New(
		context.Background(),
		network.Endpoint,
		keychainB.Keychain,
		primary.WalletConfig{
			SubnetIDs: []ids.ID{subnetID},
		},
	)
	require.NoError(err)

	// second signature using keychain B
	err = walletB.P().Signer().Sign(context.Background(), deployChainTx.PChainTx)
	require.NoError(err)

	// since we are using the fee paying key as control key too, we can commit the transaction
	// on chain immediately since the number of signatures has been reached
	blockchainID, err := newSubnet.Commit(*deployChainTx, walletA, true)
	require.NoError(err)
	fmt.Printf("blockchainID %s \n", blockchainID.String())
}

func TestSubnetDeployLedger(t *testing.T) {
	require := require.New(t)
	subnetParams := getDefaultSubnetEVMGenesis()
	newSubnet, err := New(&subnetParams)
	require.NoError(err)
	network := network.TestnetNetwork()

	ledgerInfo := keychain.LedgerParams{
		LedgerAddresses: []string{"P-testnetxxxxxxxxx"},
	}
	keychainA, err := keychain.NewKeychain(network, "", &ledgerInfo)
	require.NoError(err)

	addressesIDs, err := address.ParseToIDs([]string{"P-testnetyyyyyyyy"})
	require.NoError(err)
	controlKeys := addressesIDs
	subnetAuthKeys := addressesIDs
	threshold := 1

	newSubnet.SetSubnetControlParams(controlKeys, uint32(threshold))

	walletA, err := wallet.New(
		context.Background(),
		network.Endpoint,
		keychainA.Keychain,
		primary.WalletConfig{
			SubnetIDs: nil,
		},
	)

	require.NoError(err)
	deploySubnetTx, err := newSubnet.CreateSubnetTx(walletA)
	require.NoError(err)
	subnetID, err := newSubnet.Commit(*deploySubnetTx, walletA, true)
	require.NoError(err)
	fmt.Printf("subnetID %s \n", subnetID.String())

	time.Sleep(2 * time.Second)

	newSubnet.SetSubnetAuthKeys(subnetAuthKeys)
	deployChainTx, err := newSubnet.CreateBlockchainTx(walletA)
	require.NoError(err)

	ledgerInfoB := keychain.LedgerParams{
		LedgerAddresses: []string{"P-testnetyyyyyyyy"},
	}
	err = keychainA.Ledger.LedgerDevice.Disconnect()
	require.NoError(err)

	keychainB, err := keychain.NewKeychain(network, "", &ledgerInfoB)
	require.NoError(err)

	walletB, err := wallet.New(
		context.Background(),
		network.Endpoint,
		keychainB.Keychain,
		primary.WalletConfig{
			SubnetIDs: []ids.ID{subnetID},
		},
	)
	require.NoError(err)

	// second signature
	err = walletB.P().Signer().Sign(context.Background(), deployChainTx.PChainTx)
	require.NoError(err)

	blockchainID, err := newSubnet.Commit(*deployChainTx, walletB, true)
	require.NoError(err)

	fmt.Printf("blockchainID %s \n", blockchainID.String())
}
