// Copyright (C) 2019-2025, Lux Industries Inc All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/ids"
	"github.com/luxfi/node/utils/constants"
	"github.com/luxfi/node/vms/secp256k1fx"
	"github.com/luxfi/node/wallet/net/primary"
)

// WalletKeyInfo holds information about a wallet key loaded from ~/.lux/keys/
type WalletKeyInfo struct {
	NetworkID  uint32 `json:"network_id"`
	PrivateKey string `json:"private_key"`
	EthAddress string `json:"eth_address"`
	PChainAddr string `json:"p_chain"`
	XChainAddr string `json:"x_chain"`
	CreatedAt  int64  `json:"created_at"`
}

func main() {
	// Flags
	uri := flag.String("uri", "http://127.0.0.1:9650", "Node API URI")
	genesisFile := flag.String("genesis", "", "Path to genesis JSON file")
	name := flag.String("name", "ZOO", "Name of the chain")
	networkName := flag.String("network", "lux-testnet", "Network name (lux-mainnet, lux-testnet, etc.)")
	keyFile := flag.String("key", "", "Path to wallet key file (default: ~/.lux/keys/<network>_wallet.json)")
	flag.Parse()

	if *genesisFile == "" {
		log.Fatal("--genesis flag is required")
	}

	// Load genesis file
	genesisBytes, err := os.ReadFile(*genesisFile)
	if err != nil {
		log.Fatalf("failed to read genesis file: %s", err)
	}

	// Validate genesis is valid JSON
	var genesisData interface{}
	if err := json.Unmarshal(genesisBytes, &genesisData); err != nil {
		log.Fatalf("genesis file is not valid JSON: %s", err)
	}
	log.Printf("Loaded genesis file: %s (%d bytes)", *genesisFile, len(genesisBytes))

	// Load wallet key from file
	keyPath := *keyFile
	if keyPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("failed to get home directory: %s", err)
		}
		keyPath = filepath.Join(homeDir, ".lux", "keys", fmt.Sprintf("%s_wallet.json", *networkName))
	}

	log.Printf("Loading wallet key from %s...", keyPath)
	keyJSON, err := os.ReadFile(keyPath)
	if err != nil {
		log.Fatalf("failed to read wallet key file: %s\nHint: Start a network first with 'lux network start --testnet' to generate keys", err)
	}

	var keyInfo WalletKeyInfo
	if err := json.Unmarshal(keyJSON, &keyInfo); err != nil {
		log.Fatalf("failed to parse wallet key file: %s", err)
	}

	// Parse the private key - use secp256k1.ToPrivateKey() directly
	privKeyBytes, err := hex.DecodeString(keyInfo.PrivateKey)
	if err != nil {
		log.Fatalf("failed to decode private key: %s", err)
	}

	key, err := secp256k1.ToPrivateKey(privKeyBytes)
	if err != nil {
		log.Fatalf("failed to parse secp256k1 key: %s", err)
	}

	log.Printf("Using wallet key:")
	log.Printf("  ETH Address: %s", keyInfo.EthAddress)
	log.Printf("  P-Chain:     %s", keyInfo.PChainAddr)

	// Create keychain
	kc := primary.NewKeychainAdapter(secp256k1fx.NewKeychain(key))

	ctx := context.Background()

	// Create wallet connected to the running network
	log.Printf("Connecting to %s...", *uri)
	walletSyncStartTime := time.Now()
	wallet, err := primary.MakeWallet(ctx, &primary.WalletConfig{
		URI:         *uri,
		LUXKeychain: kc,
		EthKeychain: kc,
	})
	if err != nil {
		log.Fatalf("failed to initialize wallet: %s", err)
	}
	log.Printf("Synced wallet in %s", time.Since(walletSyncStartTime))

	// Get the P-chain wallet
	pWallet := wallet.P()

	// Create the subnet owner (key that can manage the subnet)
	owner := &secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs: []ids.ShortID{
			key.Address(),
		},
	}

	// Step 1: Create subnet (net)
	log.Println("Creating subnet...")
	createNetStartTime := time.Now()
	createNetTx, err := pWallet.IssueCreateNetTx(owner)
	if err != nil {
		log.Fatalf("failed to issue create net transaction: %s", err)
	}
	netID := createNetTx.ID()
	log.Printf("Created subnet %s in %s", netID, time.Since(createNetStartTime))

	// Step 2: Create blockchain on the subnet
	// Use the Subnet EVM VM ID
	vmID := constants.EVMID
	log.Printf("Creating chain with VM ID: %s", vmID)

	log.Printf("Creating blockchain '%s' on subnet %s...", *name, netID)
	createChainStartTime := time.Now()
	createChainTx, err := pWallet.IssueCreateChainTx(
		netID,
		genesisBytes,
		vmID,
		nil, // fxIDs (no additional feature extensions)
		*name,
	)
	if err != nil {
		log.Fatalf("failed to issue create chain transaction: %s", err)
	}
	chainID := createChainTx.ID()
	log.Printf("Created blockchain %s in %s", chainID, time.Since(createChainStartTime))

	// Print summary
	fmt.Println("\n========================================")
	fmt.Println("Subnet Deployment Complete!")
	fmt.Println("========================================")
	fmt.Printf("Subnet ID:     %s\n", netID)
	fmt.Printf("Blockchain ID: %s\n", chainID)
	fmt.Printf("Chain Name:    %s\n", *name)
	fmt.Printf("VM ID:         %s\n", vmID)
	fmt.Println("========================================")
	fmt.Println("\nTo access the chain RPC:")
	fmt.Printf("  %s/ext/bc/%s/rpc\n", *uri, chainID)
	fmt.Println("\nNote: You may need to add validators to this subnet")
	fmt.Println("and whitelist the chain on your nodes for it to start.")
}
