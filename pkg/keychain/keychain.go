// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package keychain

import (
	"errors"
	"fmt"
	"slices"

	"github.com/luxfi/cli/v2/v2/cmd/flags"
	"github.com/luxfi/cli/v2/v2/pkg/application"
	"github.com/luxfi/cli/v2/v2/pkg/key"
	"github.com/luxfi/cli/v2/v2/pkg/models"
	"github.com/luxfi/cli/v2/v2/pkg/prompts"
	"github.com/luxfi/cli/v2/v2/pkg/utils"
	"github.com/luxfi/cli/v2/v2/pkg/ux"
	"github.com/luxfi/ids"
	"github.com/luxfi/node/v2/v2/utils/crypto/keychain"
	"github.com/luxfi/node/v2/v2/utils/crypto/ledger"
	"github.com/luxfi/node/v2/v2/utils/formatting/address"
	"github.com/luxfi/node/v2/v2/utils/logging"
	"github.com/luxfi/node/v2/v2/utils/set"
	"github.com/luxfi/node/v2/v2/utils/units"
	"github.com/luxfi/node/v2/v2/vms/platformvm"
)

const (
	numLedgerIndicesToSearch           = 1000
	numLedgerIndicesToSearchForBalance = 100
)

var (
	ErrMutuallyExlusiveKeySource = errors.New("key source flags --key, --ewoq, --ledger/--ledger-addrs are mutually exclusive")
	ErrEwoqKeyOnTestnetOrMainnet    = errors.New("key source ewoq is not available for mainnet/testnet operations")
)

type Keychain struct {
	Network       models.Network
	Keychain      keychain.Keychain
	Ledger        keychain.Ledger
	UsesLedger    bool
	LedgerIndices []uint32
}

func NewKeychain(network models.Network, keychain keychain.Keychain, ledger keychain.Ledger, ledgerIndices []uint32) *Keychain {
	usesLedger := len(ledgerIndices) > 0
	return &Keychain{
		Network:       network,
		Keychain:      keychain,
		Ledger:        ledger,
		UsesLedger:    usesLedger,
		LedgerIndices: ledgerIndices,
	}
}

func (kc *Keychain) HasOnlyOneKey() bool {
	return len(kc.Keychain.Addresses()) == 1
}

func (kc *Keychain) Addresses() set.Set[ids.ShortID] {
	return kc.Keychain.Addresses()
}

func (kc *Keychain) PChainFormattedStrAddresses() ([]string, error) {
	addrs := kc.Addresses().List()
	if len(addrs) == 0 {
		return nil, fmt.Errorf("no addresses in keychain")
	}
	hrp := key.GetHRP(kc.Network.ID())
	addrsStr := []string{}
	for _, addr := range addrs {
		addrStr, err := address.Format("P", hrp, addr[:])
		if err != nil {
			return nil, err
		}
		addrsStr = append(addrsStr, addrStr)
	}

	return addrsStr, nil
}

func (kc *Keychain) AddAddresses(addresses []string) error {
	if kc.UsesLedger {
		prevNumIndices := len(kc.LedgerIndices)
		ledgerIndicesAux, err := getLedgerIndices(kc.Ledger, addresses)
		if err != nil {
			return err
		}
		kc.LedgerIndices = append(kc.LedgerIndices, ledgerIndicesAux...)
		ledgerIndicesSet := set.Set[uint32]{}
		ledgerIndicesSet.Add(kc.LedgerIndices...)
		kc.LedgerIndices = ledgerIndicesSet.List()
		slices.Sort(kc.LedgerIndices)
		if len(kc.LedgerIndices) != prevNumIndices {
			if err := showLedgerAddresses(kc.Network, kc.Ledger, kc.LedgerIndices); err != nil {
				return err
			}
		}
		luxdKc, err := keychain.NewLedgerKeychainFromIndices(kc.Ledger, kc.LedgerIndices)
		if err != nil {
			return err
		}
		kc.Keychain = luxdKc
	}
	return nil
}

func GetKeychainFromCmdLineFlags(
	app *application.Lux,
	keychainGoal string,
	network models.Network,
	keyName string,
	useEwoq bool,
	useLedger bool,
	ledgerAddresses []string,
	requiredFunds uint64,
) (*Keychain, error) {
	// set ledger usage flag if ledger addresses are given
	if len(ledgerAddresses) > 0 {
		useLedger = true
	}
	// check mutually exclusive flags
	if !flags.EnsureMutuallyExclusive([]bool{useLedger, useEwoq, keyName != ""}) {
		return nil, ErrMutuallyExlusiveKeySource
	}
	switch {
	case network.Kind() == models.Local:
		// prompt the user if no key source was provided
		if !useEwoq && !useLedger && keyName == "" {
			keyFiles, err := app.GetKeyPaths()
			if err != nil {
				return nil, err
			}
			keyName, ledgerIndex, err := prompts.GetKeyOrLedger(app.Prompt, keychainGoal, keyFiles, keychainGoal, true, nil)
			if err != nil {
				return nil, err
			}
			useLedger = ledgerIndex >= 0
		}
	case network.Kind() == models.Devnet:
		// prompt the user if no key source was provided
		if !useEwoq && !useLedger && keyName == "" {
			keyFiles, err := app.GetKeyPaths()
			if err != nil {
				return nil, err
			}
			keyName, ledgerIndex, err := prompts.GetKeyOrLedger(app.Prompt, keychainGoal, keyFiles, keychainGoal, true, nil)
			if err != nil {
				return nil, err
			}
			useLedger = ledgerIndex >= 0
		}
	case network.Kind() == models.Testnet:
		if useEwoq || keyName == "ewoq" {
			return nil, ErrEwoqKeyOnTestnetOrMainnet
		}
		// prompt the user if no key source was provided
		if !useLedger && keyName == "" {
			keyFiles, err := app.GetKeyPaths()
			if err != nil {
				return nil, err
			}
			keyName, ledgerIndex, err := prompts.GetKeyOrLedger(app.Prompt, keychainGoal, keyFiles, keychainGoal, false, nil)
			if err != nil {
				return nil, err
			}
			useLedger = ledgerIndex >= 0
		}
	case network.Kind() == models.Mainnet:
		if useEwoq || keyName == "ewoq" {
			return nil, ErrEwoqKeyOnTestnetOrMainnet
		}
		if keyName == "" {
			useLedger = true
		} else {
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("WARNING: Storing keys locally in plain text is insecure. A hardware wallet is recommended for Mainnet.")
			ux.Logger.PrintToUser("")
		}
	}

	network.HandlePublicNetworkSimulation()

	// get keychain accessor
	return GetKeychain(app, useEwoq, useLedger, ledgerAddresses, keyName, network, requiredFunds)
}

func GetKeychain(
	app *application.Lux,
	useEwoq bool,
	useLedger bool,
	ledgerAddresses []string,
	keyName string,
	network models.Network,
	requiredFunds uint64,
) (*Keychain, error) {
	if !useEwoq && !useLedger && keyName == "" {
		return nil, fmt.Errorf("one of the options ewoq/ledger/keyName must be provided")
	}
	// get keychain accessor
	if useLedger {
		ledgerDevice, err := ledger.New()
		if err != nil {
			return nil, err
		}
		// always have index 0, for change
		ledgerIndices := []uint32{0}
		if requiredFunds > 0 {
			ledgerIndicesAux, err := searchForFundedLedgerIndices(network, ledgerDevice, requiredFunds)
			if err != nil {
				return nil, err
			}
			ledgerIndices = append(ledgerIndices, ledgerIndicesAux...)
		}
		if len(ledgerAddresses) > 0 {
			ledgerIndicesAux, err := getLedgerIndices(ledgerDevice, ledgerAddresses)
			if err != nil {
				return nil, err
			}
			ledgerIndices = append(ledgerIndices, ledgerIndicesAux...)
		}
		ledgerIndicesSet := set.Set[uint32]{}
		ledgerIndicesSet.Add(ledgerIndices...)
		ledgerIndices = ledgerIndicesSet.List()
		slices.Sort(ledgerIndices)
		if err := showLedgerAddresses(network, ledgerDevice, ledgerIndices); err != nil {
			return nil, err
		}
		kc, err := keychain.NewLedgerKeychainFromIndices(ledgerDevice, ledgerIndices)
		if err != nil {
			return nil, err
		}
		return NewKeychain(network, kc, ledgerDevice, ledgerIndices), nil
	}
	if useEwoq {
		sf, err := app.GetKey("ewoq", network, false)
		if err != nil {
			return nil, err
		}
		kc := sf.KeyChain()
		return NewKeychain(network, kc, nil, nil), nil
	}
	sf, err := app.GetKey(keyName, network, false)
	if err != nil {
		return nil, err
	}
	kc := sf.KeyChain()
	return NewKeychain(network, kc, nil, nil), nil
}

func getLedgerIndices(ledgerDevice keychain.Ledger, addressesStr []string) ([]uint32, error) {
	addresses, err := address.ParseToIDs(addressesStr)
	if err != nil {
		return []uint32{}, fmt.Errorf("failure parsing ledger addresses: %w", err)
	}
	// maps the indices of addresses to their corresponding ledger indices
	indexMap := map[int]uint32{}
	// for all ledger indices to search for, find if the ledger address belongs to the input
	// addresses and, if so, add the index pair to indexMap, breaking the loop if
	// all addresses were found
	for ledgerIndex := uint32(0); ledgerIndex < numLedgerIndicesToSearch; ledgerIndex++ {
		ledgerAddress, err := ledgerDevice.Addresses([]uint32{ledgerIndex})
		if err != nil {
			return []uint32{}, err
		}
		for addressesIndex, addr := range addresses {
			if addr == ledgerAddress[0] {
				ux.Logger.PrintToUser("  Found index %d for address %s", ledgerIndex, addressesStr[addressesIndex])
				indexMap[addressesIndex] = ledgerIndex
			}
		}
		if len(indexMap) == len(addresses) {
			break
		}
	}
	// create ledgerIndices from indexMap
	ledgerIndices := []uint32{}
	for addressesIndex := range addresses {
		ledgerIndex, ok := indexMap[addressesIndex]
		if !ok {
			continue
		}
		ledgerIndices = append(ledgerIndices, ledgerIndex)
	}
	return ledgerIndices, nil
}

// search for a set of indices that pay a given amount
func searchForFundedLedgerIndices(network models.Network, ledgerDevice keychain.Ledger, amount uint64) ([]uint32, error) {
	ux.Logger.PrintToUser("Looking for ledger indices to pay for %.9f LUX...", float64(amount)/float64(units.Lux))
	pClient := platformvm.NewClient(network.Endpoint)
	totalBalance := uint64(0)
	ledgerIndices := []uint32{}
	for ledgerIndex := uint32(0); ledgerIndex < numLedgerIndicesToSearchForBalance; ledgerIndex++ {
		ledgerAddress, err := ledgerDevice.Addresses([]uint32{ledgerIndex})
		if err != nil {
			return []uint32{}, err
		}
		ctx, cancel := utils.GetAPIContext()
		resp, err := pClient.GetBalance(ctx, ledgerAddress)
		cancel()
		if err != nil {
			return nil, err
		}
		if resp.Balance > 0 {
			ux.Logger.PrintToUser("  Found index %d with %.9f LUX", ledgerIndex, float64(resp.Balance)/float64(units.Lux))
			totalBalance += uint64(resp.Balance)
			ledgerIndices = append(ledgerIndices, ledgerIndex)
		}
		if totalBalance >= amount {
			break
		}
	}
	if totalBalance < amount {
		ux.Logger.PrintToUser(logging.Yellow.Wrap("Not enough funds in the first %d indices of Ledger"), numLedgerIndicesToSearchForBalance)
		return nil, fmt.Errorf("not enough funds on ledger")
	}
	return ledgerIndices, nil
}

func showLedgerAddresses(network models.Network, ledgerDevice keychain.Ledger, ledgerIndices []uint32) error {
	// get formatted addresses for ux
	addresses, err := ledgerDevice.Addresses(ledgerIndices)
	if err != nil {
		return err
	}
	addrStrs := []string{}
	for _, addr := range addresses {
		addrStr, err := address.Format("P", key.GetHRP(network.ID), addr[:])
		if err != nil {
			return err
		}
		addrStrs = append(addrStrs, addrStr)
	}
	ux.Logger.PrintToUser(logging.Yellow.Wrap("Ledger addresses: "))
	for _, addrStr := range addrStrs {
		ux.Logger.PrintToUser(logging.Yellow.Wrap(fmt.Sprintf("  %s", addrStr)))
	}
	return nil
}
