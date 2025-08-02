// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package wallet

import (
	"context"

	"github.com/luxfi/cli/sdk/keychain"
	"github.com/luxfi/node/ids"
	luxdkeychain "github.com/luxfi/node/utils/crypto/keychain"
	"github.com/luxfi/node/utils/set"
	"github.com/luxfi/node/vms/secp256k1fx"
	"github.com/luxfi/node/wallet/subnet/primary"
	"github.com/luxfi/node/wallet/subnet/primary/common"
)

type Wallet struct {
	*primary.Wallet
	Keychain keychain.Keychain
	options  []common.Option
	config   primary.WalletConfig
}

func New(ctx context.Context, uri string, luxKeychain luxdkeychain.Keychain, config primary.WalletConfig) (Wallet, error) {
	wallet, err := primary.MakeWallet(
		ctx,
		uri,
		luxKeychain,
		nil,
		config,
	)
	return Wallet{
		Wallet: wallet,
		Keychain: keychain.Keychain{
			Keychain: luxKeychain,
		},
		config: config,
	}, err
}

// SecureWalletIsChangeOwner ensures that a fee paying address (wallet's keychain) will receive
// the change UTXO and not a randomly selected auth key that may not be paying fees
func (w *Wallet) SecureWalletIsChangeOwner() {
	addrs := w.Addresses()
	changeAddr := addrs[0]
	// sets change to go to wallet addr (instead of any other subnet auth key)
	changeOwner := &secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs:     []ids.ShortID{changeAddr},
	}
	w.options = append(w.options, common.WithChangeOwner(changeOwner))
	w.Wallet = primary.NewWalletWithOptions(w.Wallet, w.options...)
}

// SetAuthKeys sets auth keys that will be used when signing txs, besides the wallet's Keychain fee
// paying ones
func (w *Wallet) SetAuthKeys(authKeys []ids.ShortID) {
	addrs := w.Addresses()
	addrsSet := set.Set[ids.ShortID]{}
	addrsSet.Add(addrs...)
	addrsSet.Add(authKeys...)
	w.options = append(w.options, common.WithCustomAddresses(addrsSet))
	w.Wallet = primary.NewWalletWithOptions(w.Wallet, w.options...)
}

func (w *Wallet) SetSubnetAuthMultisig(authKeys []ids.ShortID) {
	w.SecureWalletIsChangeOwner()
	w.SetAuthKeys(authKeys)
}

func (w *Wallet) Addresses() []ids.ShortID {
	return w.Keychain.Addresses().List()
}
