package key_test

import (
	"encoding/hex"
	"fmt"
	"testing"
	
	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/address"
)

func TestFee0Address(t *testing.T) {
	privKeyHex := "abd51d463510cb17d7ba09e535828383d9c2c817aa386024aacce1660a1ee625"
	privKeyBytes, _ := hex.DecodeString(privKeyHex)
	
	// Direct crypto
	privKey, _ := secp256k1.ToPrivateKey(privKeyBytes)
	pubKey := privKey.PublicKey()
	addr := pubKey.Address()
	fmt.Printf("Direct crypto Address(): %x\n", addr[:])
	
	pAddr, _ := address.Format("P", "dev", addr[:])
	fmt.Printf("Direct crypto P-addr: %s\n", pAddr)
	
	// Via LoadSoft
	// Network ID 3 = devnet
	sf, err := key.NewSoftFromBytes(3, privKeyBytes)
	if err != nil {
		t.Fatal(err)
	}
	pAddrs := sf.P()
	fmt.Printf("SoftKey P-addrs: %v\n", pAddrs)
	fmt.Printf("SoftKey C-addr: %s\n", sf.C())
	
	fmt.Printf("\nExpected P-addr: P-dev1e44zjaddy52vjqa40ws90uwu9c2ryp7egufeqg\n")
}
