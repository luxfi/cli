// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package networkcmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// newDescribeCmd creates the describe command
func newDescribeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "describe <network>",
		Short: "Describe network configuration, genesis, and allocations",
		Long: `Show detailed information about a network including:
- Genesis configuration
- C-chain allocations and precompiles
- Initial validators/stakers
- Network parameters

Network must be one of: mainnet, testnet, devnet, local`,
		Args: cobra.ExactArgs(1),
		RunE: describeNetwork,
	}
}

// Genesis structures
type GenesisAllocation struct {
	EthAddr        string `json:"ethAddr"`
	LuxAddr        string `json:"luxAddr"`
	InitialAmount  uint64 `json:"initialAmount"`
	UnlockSchedule []struct {
		Amount   uint64 `json:"amount"`
		Locktime uint64 `json:"locktime"`
	} `json:"unlockSchedule"`
}

type InitialStaker struct {
	NodeID        string `json:"nodeID"`
	RewardAddress string `json:"rewardAddress"`
	DelegationFee uint64 `json:"delegationFee"`
	Weight        uint64 `json:"weight"`
	Signer        struct {
		PublicKey         string `json:"publicKey"`
		ProofOfPossession string `json:"proofOfPossession"`
	} `json:"signer"`
}

type PChainGenesis struct {
	NetworkID      uint32          `json:"networkID"`
	InitialStakers []InitialStaker `json:"initialStakers"`
}

type CChainGenesis struct {
	Config struct {
		ChainID uint64 `json:"chainId"`
	} `json:"config"`
	Alloc map[string]struct {
		Balance string `json:"balance"`
		Nonce   string `json:"nonce,omitempty"`
		Code    string `json:"code,omitempty"`
	} `json:"alloc"`
}

type MainGenesis struct {
	NetworkID   uint32              `json:"networkID"`
	Allocations []GenesisAllocation `json:"allocations"`
}

func describeNetwork(cmd *cobra.Command, args []string) error {
	networkType := strings.ToLower(args[0])

	// Validate network type
	validNetworks := map[string]bool{
		"mainnet": true,
		"testnet": true,
		"devnet":  true,
		"local":   true,
	}
	if !validNetworks[networkType] {
		return fmt.Errorf("invalid network type: %s (valid: mainnet, testnet, devnet, local)", networkType)
	}

	// Find genesis configs directory
	genesisDir := findGenesisDir(networkType)
	if genesisDir == "" {
		return fmt.Errorf("genesis configuration not found for %s", networkType)
	}

	fmt.Printf("=== %s Network Configuration ===\n\n", strings.ToUpper(networkType))

	// Load and display main genesis
	if err := displayMainGenesis(genesisDir); err != nil {
		fmt.Printf("Main genesis: %v\n\n", err)
	}

	// Load and display P-chain genesis (validators)
	if err := displayPChainGenesis(genesisDir); err != nil {
		fmt.Printf("P-chain genesis: %v\n\n", err)
	}

	// Load and display C-chain genesis (allocations)
	if err := displayCChainGenesis(genesisDir); err != nil {
		fmt.Printf("C-chain genesis: %v\n\n", err)
	}

	// Display precompile addresses
	displayPrecompiles()

	return nil
}

func findGenesisDir(networkType string) string {
	// Try multiple locations
	possiblePaths := []string{
		filepath.Join(os.Getenv("HOME"), "work/lux/genesis/configs", networkType),
		filepath.Join(os.Getenv("HOME"), ".lux/genesis", networkType),
		filepath.Join("/usr/local/share/lux/genesis", networkType),
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(filepath.Join(path, "genesis.json")); err == nil {
			return path
		}
		if _, err := os.Stat(filepath.Join(path, "cchain.json")); err == nil {
			return path
		}
	}

	return ""
}

func displayMainGenesis(dir string) error {
	genesisPath := filepath.Join(dir, "genesis.json")
	data, err := os.ReadFile(genesisPath)
	if err != nil {
		return err
	}

	var genesis MainGenesis
	if err := json.Unmarshal(data, &genesis); err != nil {
		return err
	}

	fmt.Printf("Network ID: %d\n", genesis.NetworkID)
	fmt.Printf("Allocations: %d accounts\n\n", len(genesis.Allocations))

	fmt.Println("genesis allocations")
	fmt.Println("idx   eth_address                                  p-chain_address                              initial_amount")
	for i, alloc := range genesis.Allocations {
		if i >= 10 {
			fmt.Printf("... and %d more allocations\n", len(genesis.Allocations)-10)
			break
		}
		fmt.Printf("%-5d %-44s %-44s %d\n",
			i+1,
			alloc.EthAddr,
			alloc.LuxAddr,
			alloc.InitialAmount)
	}
	fmt.Println()

	return nil
}

func displayPChainGenesis(dir string) error {
	pchainPath := filepath.Join(dir, "pchain.json")
	data, err := os.ReadFile(pchainPath)
	if err != nil {
		return err
	}

	var pgenesis PChainGenesis
	if err := json.Unmarshal(data, &pgenesis); err != nil {
		return err
	}

	fmt.Println("initial validators (p-chain)")
	fmt.Println("idx   node_id                                      reward_address                               weight           delegation_fee")
	for i, staker := range pgenesis.InitialStakers {
		fmt.Printf("%-5d %-44s %-44s %-16d %.2f%%\n",
			i+1,
			staker.NodeID,
			staker.RewardAddress,
			staker.Weight,
			float64(staker.DelegationFee)/10000.0)
	}
	fmt.Println()

	return nil
}

func displayCChainGenesis(dir string) error {
	cchainPath := filepath.Join(dir, "cchain.json")
	data, err := os.ReadFile(cchainPath)
	if err != nil {
		return err
	}

	var cgenesis CChainGenesis
	if err := json.Unmarshal(data, &cgenesis); err != nil {
		return err
	}

	fmt.Printf("C-Chain ID: %d\n\n", cgenesis.Config.ChainID)

	fmt.Println("c-chain allocations")
	fmt.Println("address                                      balance                              type")
	for addr, alloc := range cgenesis.Alloc {
		allocType := "account"
		if alloc.Code != "" && alloc.Code != "0x" {
			allocType = "precompile"
		}

		// Format balance (it's in hex)
		balance := alloc.Balance
		if len(balance) > 20 {
			balance = balance[:20] + "..."
		}

		fmt.Printf("0x%-42s %-36s %s\n",
			addr,
			balance,
			allocType)
	}
	fmt.Println()

	return nil
}

func displayPrecompiles() {
	fmt.Println("active precompiles (lx defi)")
	fmt.Println("name         lp        address                                      description")

	defiPrecompiles := []struct {
		Name    string
		LP      string
		Address string
		Desc    string
	}{
		{"LXPool", "LP-9010", "0x0000000000000000000000000000000000009010", "v4 PoolManager AMM core"},
		{"LXOracle", "LP-9011", "0x0000000000000000000000000000000000009011", "Multi-source price aggregation"},
		{"LXRouter", "LP-9012", "0x0000000000000000000000000000000000009012", "Swap routing"},
		{"LXHooks", "LP-9013", "0x0000000000000000000000000000000000009013", "Hook contract registry"},
		{"LXFlash", "LP-9014", "0x0000000000000000000000000000000000009014", "Flash loan facility"},
		{"LXBook", "LP-9020", "0x0000000000000000000000000000000000009020", "CLOB matching engine"},
		{"LXVault", "LP-9030", "0x0000000000000000000000000000000000009030", "DeFi vault operations"},
		{"LXFeed", "LP-9040", "0x0000000000000000000000000000000000009040", "Price feed aggregator"},
		{"LXLend", "LP-9050", "0x0000000000000000000000000000000000009050", "Lending pool (Aave-style)"},
		{"LXLiquid", "LP-9060", "0x0000000000000000000000000000000000009060", "Self-repaying loans"},
		{"Liquidator", "LP-9070", "0x0000000000000000000000000000000000009070", "Position liquidation engine"},
		{"LiquidFX", "LP-9080", "0x0000000000000000000000000000000000009080", "Transmuter (liquid token)"},
	}

	for _, p := range defiPrecompiles {
		fmt.Printf("%-12s %-9s %-44s %s\n", p.Name, p.LP, p.Address, p.Desc)
	}
	fmt.Println()

	// AI/ML precompiles
	fmt.Println("ai/ml precompiles")
	fmt.Println("name         address                                      description")
	aiPrecompiles := []struct {
		Name    string
		Address string
		Desc    string
	}{
		{"ML-DSA", "0x0000000000000000000000000000000000000300", "Post-quantum ML-DSA signature verification"},
		{"NVTrust", "0x0000000000000000000000000000000000000301", "NVIDIA GPU attestation verification"},
		{"Inference", "0x0000000000000000000000000000000000000302", "On-chain inference verification"},
		{"ModelReg", "0x0000000000000000000000000000000000000303", "AI model registry"},
	}
	for _, p := range aiPrecompiles {
		fmt.Printf("%-12s %-44s %s\n", p.Name, p.Address, p.Desc)
	}
	fmt.Println()

	// Threshold cryptography precompiles
	fmt.Println("threshold cryptography precompiles")
	fmt.Println("name         address                                      description")
	thresholdPrecompiles := []struct {
		Name    string
		Address string
		Desc    string
	}{
		{"TSS-ECDSA", "0x0000000000000000000000000000000000000400", "Threshold ECDSA (CMP/CGGMP21)"},
		{"TSS-EdDSA", "0x0000000000000000000000000000000000000401", "Threshold EdDSA (FROST)"},
		{"TSS-BLS", "0x0000000000000000000000000000000000000402", "Threshold BLS signatures"},
		{"LSS", "0x0000000000000000000000000000000000000403", "Lux Secret Sharing"},
		{"MPC", "0x0000000000000000000000000000000000000404", "Multi-party computation"},
	}
	for _, p := range thresholdPrecompiles {
		fmt.Printf("%-12s %-44s %s\n", p.Name, p.Address, p.Desc)
	}
	fmt.Println()

	// FHE precompiles
	fmt.Println("fully homomorphic encryption (fhe) precompiles")
	fmt.Println("name         address                                      description")
	fhePrecompiles := []struct {
		Name    string
		Address string
		Desc    string
	}{
		{"FHE-Add", "0x0000000000000000000000000000000000000500", "Homomorphic addition"},
		{"FHE-Mul", "0x0000000000000000000000000000000000000501", "Homomorphic multiplication"},
		{"FHE-Cmp", "0x0000000000000000000000000000000000000502", "Homomorphic comparison"},
		{"FHE-Enc", "0x0000000000000000000000000000000000000503", "FHE encryption"},
		{"FHE-Dec", "0x0000000000000000000000000000000000000504", "FHE decryption (threshold)"},
		{"FHE-Key", "0x0000000000000000000000000000000000000505", "FHE key management"},
	}
	for _, p := range fhePrecompiles {
		fmt.Printf("%-12s %-44s %s\n", p.Name, p.Address, p.Desc)
	}
	fmt.Println()

	// ZKP precompiles
	fmt.Println("zero-knowledge proof (zkp) precompiles")
	fmt.Println("name         address                                      description")
	zkpPrecompiles := []struct {
		Name    string
		Address string
		Desc    string
	}{
		{"Groth16", "0x0000000000000000000000000000000000000600", "Groth16 proof verification"},
		{"PLONK", "0x0000000000000000000000000000000000000601", "PLONK proof verification"},
		{"STARK", "0x0000000000000000000000000000000000000602", "STARK proof verification"},
		{"Halo2", "0x0000000000000000000000000000000000000603", "Halo2 proof verification"},
		{"Poseidon", "0x0000000000000000000000000000000000000604", "Poseidon hash (ZK-friendly)"},
		{"Rescue", "0x0000000000000000000000000000000000000605", "Rescue hash (ZK-friendly)"},
	}
	for _, p := range zkpPrecompiles {
		fmt.Printf("%-12s %-44s %s\n", p.Name, p.Address, p.Desc)
	}
	fmt.Println()

	// Standard EIP precompiles
	fmt.Println("standard precompiles (eip)")
	fmt.Println("name         address                                      eip")
	stdPrecompiles := []struct {
		Name    string
		Address string
		EIP     string
	}{
		{"ECADD", "0x0000000000000000000000000000000000000006", "EIP-1108 (BN254)"},
		{"ECMUL", "0x0000000000000000000000000000000000000007", "EIP-1108 (BN254)"},
		{"ECPAIRING", "0x0000000000000000000000000000000000000008", "EIP-1108 (BN254)"},
		{"BLS G1ADD", "0x000000000000000000000000000000000000000b", "EIP-2537 (BLS12-381)"},
		{"BLS G1MUL", "0x000000000000000000000000000000000000000c", "EIP-2537 (BLS12-381)"},
		{"BLS G2ADD", "0x000000000000000000000000000000000000000d", "EIP-2537 (BLS12-381)"},
		{"BLS G2MUL", "0x000000000000000000000000000000000000000e", "EIP-2537 (BLS12-381)"},
		{"BLS PAIRING", "0x000000000000000000000000000000000000000f", "EIP-2537 (BLS12-381)"},
		{"BLS MAP G1", "0x0000000000000000000000000000000000000010", "EIP-2537 (BLS12-381)"},
		{"BLS MAP G2", "0x0000000000000000000000000000000000000011", "EIP-2537 (BLS12-381)"},
	}

	for _, p := range stdPrecompiles {
		fmt.Printf("%-12s %-44s %s\n", p.Name, p.Address, p.EIP)
	}
}
