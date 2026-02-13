// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package keychain

import (
	"errors"
	"fmt"
	"slices"

	"github.com/luxfi/address"
	"github.com/luxfi/cli/cmd/flags"
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/constants"
	"github.com/luxfi/ids"
	"github.com/luxfi/keychain"
	"github.com/luxfi/ledger"
	luxlog "github.com/luxfi/log"
	"github.com/luxfi/math/set"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/sdk/platformvm"
	"github.com/luxfi/sdk/prompts"
)

const (
	numLedgerIndicesToSearch           = 1000
	numLedgerIndicesToSearchForBalance = 100
)

var (
	ErrMutuallyExlusiveKeySource = errors.New("key source flags --key and --ledger/--ledger-addrs are mutually exclusive")
	// AllowInsecureKeysOnMainnet is a flag that allows use of software keys on mainnet
	// This should only be set to true for testing or if you understand the security risks
	AllowInsecureKeysOnMainnet = false
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
		luxdKc, err := keychain.NewLedgerKeychain(kc.Ledger, kc.LedgerIndices)
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
	useLocalKey bool,
	useLedger bool,
	ledgerAddresses []string,
	requiredFunds uint64,
) (*Keychain, error) {
	// set ledger usage flag if ledger addresses are given
	if len(ledgerAddresses) > 0 {
		useLedger = true
	}
	// check mutually exclusive flags
	if !flags.EnsureMutuallyExclusive([]bool{useLedger, useLocalKey, keyName != ""}) {
		return nil, ErrMutuallyExlusiveKeySource
	}
	switch network {
	case models.Local:
		// prompt the user if no key source was provided
		if !useLocalKey && !useLedger && keyName == "" {
			var err error
			useLedger, keyName, err = prompts.GetKeyOrLedger(app.Prompt, keychainGoal, app.GetKeyDir(), false)
			if err != nil {
				return nil, err
			}
		}
	case models.Devnet:
		// prompt the user if no key source was provided
		if !useLocalKey && !useLedger && keyName == "" {
			var err error
			useLedger, keyName, err = prompts.GetKeyOrLedger(app.Prompt, keychainGoal, app.GetKeyDir(), false)
			if err != nil {
				return nil, err
			}
		}
	case models.Testnet:
		// prompt the user if no key source was provided
		if !useLedger && keyName == "" {
			var err error
			useLedger, keyName, err = prompts.GetKeyOrLedger(app.Prompt, keychainGoal, app.GetKeyDir(), false)
			if err != nil {
				return nil, err
			}
		}
	case models.Mainnet:
		if keyName == "" && !useLocalKey {
			useLedger = true
		} else {
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("%s", luxlog.Red.Wrap("WARNING: Storing keys locally in plain text is insecure. A hardware wallet is recommended for Mainnet."))
			ux.Logger.PrintToUser("")
		}
	}

	// get keychain accessor
	return GetKeychain(app, useLocalKey, useLedger, ledgerAddresses, keyName, network, requiredFunds)
}

func GetKeychain(
	app *application.Lux,
	useLocalKey bool,
	useLedger bool,
	ledgerAddresses []string,
	keyName string,
	network models.Network,
	requiredFunds uint64,
) (*Keychain, error) {
	// Check for LUX_MNEMONIC environment variable first
	// This allows automated deployment without interactive key selection
	if mnemonic := key.GetMnemonicFromEnv(); mnemonic != "" && !useLedger && !useLocalKey && keyName == "" {
		ux.Logger.PrintToUser("Using key from LUX_MNEMONIC environment variable (BIP-44 derivation)")
		// Use BIP-44 standard derivation path: m/44'/60'/0'/0/0
		sf, err := key.NewSoftFromMnemonic(network.ID(), mnemonic)
		if err != nil {
			return nil, fmt.Errorf("failed to create soft key from mnemonic: %w", err)
		}
		pAddrs := sf.P()
		if len(pAddrs) > 0 {
			ux.Logger.PrintToUser("  P-Chain address: %s", pAddrs[0])
		}
		cAddr := sf.C()
		if cAddr != "" {
			ux.Logger.PrintToUser("  C-Chain address: %s", cAddr)
		}
		kc := sf.KeyChain()
		wrappedKc := WrapSecp256k1fxKeychain(kc)
		return NewKeychain(network, wrappedKc, nil, nil), nil
	}

	if !useLocalKey && !useLedger && keyName == "" {
		return nil, fmt.Errorf("one of the options local-key/ledger/keyName must be provided (or set LUX_MNEMONIC env)")
	}
	// get keychain accessor
	if useLedger {
		ledgerDevice, err := ledger.NewLedger()
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
		kc, err := keychain.NewLedgerKeychain(ledgerDevice, ledgerIndices)
		if err != nil {
			return nil, err
		}
		return NewKeychain(network, kc, ledgerDevice, ledgerIndices), nil
	}
	if useLocalKey {
		// Use the local-key from ~/.lux/keys/local-key.pk which is generated
		// on first use with a unique random key per machine.
		sf, err := key.GetOrCreateLocalKey(network.ID())
		if err != nil {
			return nil, fmt.Errorf("failed to get local key: %w", err)
		}
		kc := sf.KeyChain()
		wrappedKc := WrapSecp256k1fxKeychain(kc)
		return NewKeychain(network, wrappedKc, nil, nil), nil
	}
	keyPath := app.GetKeyPath(keyName)
	sf, err := key.LoadSoft(network.ID(), keyPath)
	if err != nil {
		return nil, err
	}
	kc := sf.KeyChain()
	wrappedKc := WrapSecp256k1fxKeychain(kc)
	return NewKeychain(network, wrappedKc, nil, nil), nil
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
		ledgerAddress, err := ledgerDevice.GetAddresses([]uint32{ledgerIndex})
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

// getNetworkEndpoint returns the endpoint for the given network
func getNetworkEndpoint(network models.Network) string {
	switch network {
	case models.Mainnet:
		return "https://api.lux-test.network"
	case models.Testnet:
		return "https://api.lux-test.network"
	case models.Local:
		return "http://127.0.0.1:9630"
	default:
		return "http://127.0.0.1:9630"
	}
}

// search for a set of indices that pay a given amount
func searchForFundedLedgerIndices(network models.Network, ledgerDevice keychain.Ledger, amount uint64) ([]uint32, error) {
	ux.Logger.PrintToUser("Looking for ledger indices to pay for %.9f LUX...", float64(amount)/float64(constants.Lux))
	endpoint := getNetworkEndpoint(network)
	pClient := platformvm.NewClient(endpoint)
	totalBalance := uint64(0)
	ledgerIndices := []uint32{}
	for ledgerIndex := uint32(0); ledgerIndex < numLedgerIndicesToSearchForBalance; ledgerIndex++ {
		ledgerAddress, err := ledgerDevice.GetAddresses([]uint32{ledgerIndex})
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
			ux.Logger.PrintToUser("  Found index %d with %.9f LUX", ledgerIndex, float64(resp.Balance)/float64(constants.Lux))
			totalBalance += uint64(resp.Balance)
			ledgerIndices = append(ledgerIndices, ledgerIndex)
		}
		if totalBalance >= amount {
			break
		}
	}
	if totalBalance < amount {
		ux.Logger.PrintToUser(luxlog.Yellow.Wrap("Not enough funds in the first %d indices of Ledger"), numLedgerIndicesToSearchForBalance)
		return nil, fmt.Errorf("not enough funds on ledger")
	}
	return ledgerIndices, nil
}

func showLedgerAddresses(network models.Network, ledgerDevice keychain.Ledger, ledgerIndices []uint32) error {
	// get formatted addresses for ux
	addresses, err := ledgerDevice.GetAddresses(ledgerIndices)
	if err != nil {
		return err
	}
	addrStrs := []string{}
	for _, addr := range addresses {
		addrStr, err := address.Format("P", key.GetHRP(network.ID()), addr[:])
		if err != nil {
			return err
		}
		addrStrs = append(addrStrs, addrStr)
	}
	ux.Logger.PrintToUser("%s", luxlog.Yellow.Wrap("Ledger addresses: "))
	for _, addrStr := range addrStrs {
		ux.Logger.PrintToUser(luxlog.Yellow.Wrap("  %s"), addrStr)
	}
	return nil
}
