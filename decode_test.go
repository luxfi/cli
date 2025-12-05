package main

import (
	"fmt"
	"encoding/hex"
	"github.com/luxfi/ids"
)

func main() {
	fmt.Println("=== Address Analysis ===\n")

	// The address bytes we derived from secp256k1
	computedBytes, _ := hex.DecodeString("4f35e1d82b3983b7313c792723bf4ea66ebc5200")
	fmt.Printf("Computed from secp256k1:\n  Hex: %x\n", computedBytes)
	
	shortID := ids.ShortID{}
	copy(shortID[:], computedBytes)
	fmt.Printf("  As ShortID: %v\n\n", shortID)

	// Genesis address bytes (from P-lux13kuhcl8vufyu9wvtmspzdnzv9ftm75hus3wuf7)
	genesisBytes, _ := hex.DecodeString("8db97c7cece249c2b98bdc0226cc4c2a57bf52fc")
	fmt.Printf("Genesis P-Chain address bytes:\n  Hex: %x\n", genesisBytes)
	
	genesisShortID := ids.ShortID{}
	copy(genesisShortID[:], genesisBytes)
	fmt.Printf("  As ShortID: %v\n\n", genesisShortID)

	// CLI address bytes (from P-custom18jma8ppw3nhx5r4ap8clazz0dps7rv5u9xde7p)
	cliBytes, _ := hex.DecodeString("3cb7d3842e8cee6a0ebd09f1fe884f6861e1b29c")
	fmt.Printf("CLI P-Chain address bytes:\n  Hex: %x\n", cliBytes)
	
	cliShortID := ids.ShortID{}
	copy(cliShortID[:], cliBytes)
	fmt.Printf("  As ShortID: %v\n\n", cliShortID)

	// Check: is the genesis bytes the same as the C-Chain address?
	cChainAddr := "0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"
	fmt.Printf("C-Chain address from genesis:\n  %s\n\n", cChainAddr)
	
	fmt.Printf("FINDING: Genesis P-Chain address bytes (8db97c7cece249c2b98bdc0226cc4c2a57bf52fc)\n")
	fmt.Printf("         match the C-Chain ETH address (0x8db97c7cece249c2b98bdc0226cc4c2a57bf52fc)\n")
	fmt.Printf("         when the 0x prefix is removed!\n")
}
