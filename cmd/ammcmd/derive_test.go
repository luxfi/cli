package ammcmd

import (
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/luxfi/crypto"
	"github.com/luxfi/go-bip39"
)

func TestDerive(t *testing.T) {
	mnemonic := "know defense install season surface planet hobby borrow theory security aisle toast"
	
	// Generate seed with empty passphrase
	seed := bip39.NewSeed(mnemonic, "")
	t.Logf("Seed: %s", hex.EncodeToString(seed))
	
	// Create master key
	masterKey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		t.Fatalf("Error creating master: %v", err)
	}
	
	// Derive m/44'/60'/0'/0/0
	purpose, _ := masterKey.Derive(hdkeychain.HardenedKeyStart + 44)
	coinType, _ := purpose.Derive(hdkeychain.HardenedKeyStart + 60)
	account, _ := coinType.Derive(hdkeychain.HardenedKeyStart + 0)
	change, _ := account.Derive(0)
	addressKey, _ := change.Derive(0)
	
	// Get private key
	ecPrivKey, _ := addressKey.ECPrivKey()
	privKey := ecPrivKey.ToECDSA()
	
	t.Logf("Private key: %s", hex.EncodeToString(crypto.FromECDSA(privKey)))
	
	// Get address
	addr := crypto.PubkeyToAddress(privKey.PublicKey)
	t.Logf("Address: %s", addr.Hex())
	
	// Expected address
	expected := "0x9011E888251AB053B7bD1cdB598Db4f9DEd94714"
	if addr.Hex() != expected {
		t.Errorf("Address mismatch: got %s, want %s", addr.Hex(), expected)
	}
}
