// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package keycmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/constants"
	"github.com/luxfi/crypto"
	genesiscfg "github.com/luxfi/genesis/configs"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ripemd160" //nolint:gosec // G507: Required for legacy address derivation
)

// GenesisAllocation represents a genesis allocation entry in genesis.json.
type GenesisAllocation struct {
	EthAddr        string           `json:"ethAddr"`
	LuxAddr        string           `json:"luxAddr"`
	InitialAmount  uint64           `json:"initialAmount"`
	UnlockSchedule []UnlockSchedule `json:"unlockSchedule"`
}

type UnlockSchedule struct {
	Amount   uint64 `json:"amount"`
	Locktime uint64 `json:"locktime"`
}

type InitialStaker struct {
	NodeID        string `json:"nodeID"`
	RewardAddress string `json:"rewardAddress"`
	DelegationFee uint64 `json:"delegationFee"`
	Signer        struct {
		PublicKey         string `json:"publicKey"`
		ProofOfPossession string `json:"proofOfPossession"`
	} `json:"signer"`
}

type Genesis struct {
	NetworkID                  uint32              `json:"networkID"`
	Allocations                []GenesisAllocation `json:"allocations"`
	StartTime                  uint64              `json:"startTime"`
	InitialStakeDuration       uint64              `json:"initialStakeDuration"`
	InitialStakeDurationOffset uint64              `json:"initialStakeDurationOffset"`
	InitialStakedFunds         []string            `json:"initialStakedFunds"`
	InitialStakers             []InitialStaker     `json:"initialStakers"`
	CChainGenesis              string              `json:"cChainGenesis"`
	XChainGenesis              string              `json:"xChainGenesis"`
	Message                    string              `json:"message"`
}

type XChainGenesis struct {
	Allocations                []GenesisAllocation `json:"allocations"`
	StartTime                  uint64              `json:"startTime"`
	InitialStakeDuration       uint64              `json:"initialStakeDuration"`
	InitialStakeDurationOffset uint64              `json:"initialStakeDurationOffset"`
	InitialStakedFunds         []string            `json:"initialStakedFunds"`
	InitialStakers             []interface{}       `json:"initialStakers"`
}

// Network configuration
type NetworkConfig struct {
	NetworkID      uint32
	ChainID        uint32
	KeyPrefix      string
	NumPChainKeys  int
	NumXChainKeys  int
	HRP            string // Human-readable part for bech32 (lux, test, local)
	VestingYears   int
	VestingPercent float64
	Message        string
}

var (
	outputFile       string
	networkIDFlag    uint32
	pChainKeys       []string
	xChainKeys       []string
	vestingYears     int
	vestingPercent   float64
	amountPerKey     uint64
	preserveCGenesis string
	useMainnet       bool
	useTestnet       bool
	useDevnet        bool
	generateKeys     bool
	numKeys          int
	saveToLux        bool // Save genesis to ~/.lux/networks/<network>/genesis.json
)

const (
	// 1 billion LUX in nLUX (1B * 10^9)
	oneBillionLUX = 1_000_000_000_000_000_000
	// Seconds per year
	secondsPerYear = 365 * 24 * 60 * 60
	// Jan 1, 2020 00:00:00 UTC
	jan2020 = 1577836800
)

// Predefined network configurations
var networkConfigs = map[string]NetworkConfig{
	"mainnet": {
		NetworkID:      constants.MainnetID,      // 1 (P-Chain network identifier)
		ChainID:        constants.MainnetChainID, // 96369 (C-Chain EVM identifier)
		KeyPrefix:      "mainnet-key",
		NumPChainKeys:  5,
		NumXChainKeys:  5,
		HRP:            constants.MainnetHRP, // "lux"
		VestingYears:   100,
		VestingPercent: 1.0,
		Message:        "Lux Mainnet Genesis - Quantum-Safe BLS Signatures",
	},
	"testnet": {
		NetworkID:      constants.TestnetID,      // 2 (P-Chain network identifier)
		ChainID:        constants.TestnetChainID, // 96368 (C-Chain EVM identifier)
		KeyPrefix:      "testnet-key",
		NumPChainKeys:  5,
		NumXChainKeys:  5,
		HRP:            constants.TestnetHRP, // "test"
		VestingYears:   1,
		VestingPercent: 100.0, // Fully unlocked after 1 year for testing
		Message:        "Lux Testnet Genesis",
	},
	"devnet": {
		NetworkID:      constants.DevnetID,      // 3 (P-Chain network identifier)
		ChainID:        constants.DevnetChainID, // 96370 (C-Chain EVM identifier)
		KeyPrefix:      "devnet-key",
		NumPChainKeys:  3,
		NumXChainKeys:  2,
		HRP:            constants.DevnetHRP, // "dev"
		VestingYears:   0,                   // No vesting for devnet
		VestingPercent: 100.0,
		Message:        "Lux Devnet Genesis - Development Only",
	},
	"custom": {
		NetworkID:      constants.CustomID,      // 1337 (P-Chain network identifier)
		ChainID:        constants.CustomChainID, // 1337 (C-Chain EVM identifier)
		KeyPrefix:      "custom-key",
		NumPChainKeys:  1,
		NumXChainKeys:  0,
		HRP:            constants.CustomHRP, // "custom"
		VestingYears:   0,
		VestingPercent: 100.0,
		Message:        "Lux Custom Genesis - Single Node Development",
	},
}

func newGenesisCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "genesis",
		Short: "Generate genesis.json for different networks",
		Long: `Generate a genesis.json file with P-Chain and X-Chain allocations.

Network modes:
  --mainnet   Use mainnet configuration (Network ID: 1, Chain ID: 96369)
              - Uses mainnet-key-01 through mainnet-key-11
              - 5 P-Chain keys (first unlocked, rest 100-year vesting)
              - 5 X-Chain keys (100-year vesting)

  --testnet   Use testnet configuration (Network ID: 2, Chain ID: 96368)
              - Uses testnet-key-01 through testnet-key-10
              - Shorter vesting for testing

  --devnet    Use devnet configuration (Network ID: 3, Chain ID: 96370)
              - Uses devnet-key-01 through devnet-key-05
              - No vesting, fully unlocked

  (no flag)   Local development (Network ID: 1337, Chain ID: 1337)
              - Generates new keys if needed
              - Single validator, fully unlocked

The command will generate keys if they don't exist (use --generate-keys to force).

Examples:
  # Generate mainnet genesis using existing mainnet keys
  lux key genesis --mainnet -o /path/to/genesis.json

  # Generate testnet genesis
  lux key genesis --testnet -o /path/to/genesis.json

  # Generate devnet genesis with new keys
  lux key genesis --devnet --generate-keys -o /path/to/genesis.json

  # Custom configuration with manual key selection
  lux key genesis --p-chain key1,key2 --x-chain key3 -o genesis.json`,
		RunE: runGenesisCmd,
	}

	// Network selection flags (mutually exclusive)
	cmd.Flags().BoolVar(&useMainnet, "mainnet", false, "Generate mainnet genesis (Network ID: 1, Chain ID: 96369)")
	cmd.Flags().BoolVar(&useTestnet, "testnet", false, "Generate testnet genesis (Network ID: 2, Chain ID: 96368)")
	cmd.Flags().BoolVar(&useDevnet, "devnet", false, "Generate devnet genesis (Network ID: 3, Chain ID: 96370)")

	// Key generation
	cmd.Flags().BoolVar(&generateKeys, "generate-keys", false, "Generate new keys if they don't exist")
	cmd.Flags().IntVarP(&numKeys, "num-keys", "n", 11, "Number of keys to generate (for mainnet/testnet)")

	// Output
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (default: ~/.lux/networks/<network>/genesis.json)")
	cmd.Flags().BoolVar(&saveToLux, "save", false, "Save genesis to ~/.lux/networks/<network>/genesis.json")

	// Custom configuration (overrides network defaults)
	cmd.Flags().Uint32Var(&networkIDFlag, "network-id", 0, "Network ID (overrides network preset)")
	cmd.Flags().StringSliceVar(&pChainKeys, "p-chain", nil, "P-Chain allocation keys (overrides network preset)")
	cmd.Flags().StringSliceVar(&xChainKeys, "x-chain", nil, "X-Chain allocation keys (overrides network preset)")
	cmd.Flags().IntVar(&vestingYears, "vesting-years", 0, "Vesting period in years (overrides network preset)")
	cmd.Flags().Float64Var(&vestingPercent, "vesting-percent", 0, "Percentage unlocked per year (overrides network preset)")
	cmd.Flags().Uint64Var(&amountPerKey, "amount", oneBillionLUX, "Amount per key in nLUX (default 1B LUX)")
	cmd.Flags().StringVar(&preserveCGenesis, "c-chain-genesis", "", "Path to existing genesis to preserve C-Chain config")

	return cmd
}

func runGenesisCmd(_ *cobra.Command, _ []string) error {
	// Determine network configuration
	var config NetworkConfig
	networkName := "local"

	// Count how many network flags are set
	networkFlags := 0
	if useMainnet {
		networkFlags++
		networkName = "mainnet"
	}
	if useTestnet {
		networkFlags++
		networkName = "testnet"
	}
	if useDevnet {
		networkFlags++
		networkName = "devnet"
	}

	if networkFlags > 1 {
		return fmt.Errorf("only one of --mainnet, --testnet, --devnet can be specified")
	}

	config = networkConfigs[networkName]

	// Apply overrides
	if networkIDFlag != 0 {
		config.NetworkID = networkIDFlag
		config.ChainID = networkIDFlag
	}
	if vestingYears != 0 {
		config.VestingYears = vestingYears
	}
	if vestingPercent != 0 {
		config.VestingPercent = vestingPercent
	}

	// Apply -n flag to adjust number of keys (split evenly between P and X chains)
	if numKeys > 0 && len(pChainKeys) == 0 && len(xChainKeys) == 0 {
		// User specified -n, distribute keys between P and X chains
		// For mainnet/testnet: typically 50/50 split
		// For devnet/local: mostly P-Chain keys
		if networkName == "mainnet" || networkName == "testnet" {
			config.NumPChainKeys = (numKeys + 1) / 2 // Ceiling division
			config.NumXChainKeys = numKeys / 2
		} else {
			config.NumPChainKeys = numKeys
			config.NumXChainKeys = 0
		}
	}

	// Determine output path - default to ~/.lux/networks/<network>/genesis.json
	actualOutput := outputFile
	if actualOutput == "" || saveToLux {
		networksDir := filepath.Join(app.GetBaseDir(), "networks", networkName)
		if err := os.MkdirAll(networksDir, 0o750); err != nil {
			return fmt.Errorf("failed to create networks directory: %w", err)
		}
		defaultOutput := filepath.Join(networksDir, "genesis.json")
		if actualOutput == "" {
			actualOutput = defaultOutput
		}
		// If --save flag is set, also save to the default location
		if saveToLux && outputFile != "" && outputFile != defaultOutput {
			ux.Logger.Info("Will save genesis to both: %s and %s", outputFile, defaultOutput)
		}
	}

	keysDir := filepath.Join(app.GetBaseDir(), "keys")

	// Determine which keys to use
	var pKeys, xKeys []string

	if len(pChainKeys) > 0 {
		// Use manually specified keys
		pKeys = pChainKeys
		xKeys = xChainKeys
	} else {
		// Use network preset keys based on numKeys
		pKeys = make([]string, config.NumPChainKeys)
		for i := 0; i < config.NumPChainKeys; i++ {
			pKeys[i] = fmt.Sprintf("%s-%02d", config.KeyPrefix, i+1)
		}
		xKeys = make([]string, config.NumXChainKeys)
		for i := 0; i < config.NumXChainKeys; i++ {
			xKeys[i] = fmt.Sprintf("%s-%02d", config.KeyPrefix, config.NumPChainKeys+i+1)
		}
	}

	// Check if keys exist, generate if needed
	allKeys := make([]string, 0, len(pKeys)+len(xKeys))
	allKeys = append(allKeys, pKeys...)
	allKeys = append(allKeys, xKeys...)
	missingKeys := []string{}
	for _, keyName := range allKeys {
		keyDir := filepath.Join(keysDir, keyName)
		if _, err := os.Stat(keyDir); os.IsNotExist(err) {
			missingKeys = append(missingKeys, keyName)
		}
	}

	if len(missingKeys) > 0 {
		if !generateKeys {
			return fmt.Errorf("missing keys: %s\nUse --generate-keys to create them, or create manually with 'lux key create'",
				strings.Join(missingKeys, ", "))
		}
		ux.Logger.Info("Generating missing keys: %s", strings.Join(missingKeys, ", "))
		for _, keyName := range missingKeys {
			if err := generateKeySet(keyName); err != nil {
				return fmt.Errorf("failed to generate key %s: %w", keyName, err)
			}
			ux.Logger.Info("Generated key set: %s", keyName)
		}
	}

	ux.Logger.Info("Generating %s genesis...", networkName)
	ux.Logger.Info("Network ID: %d, Chain ID: %d", config.NetworkID, config.ChainID)

	// Generate P-Chain allocations
	pAllocations := []GenesisAllocation{}
	initialStakers := []InitialStaker{}
	initialStakedFunds := []string{}

	for i, keyName := range pKeys {
		keyDir := filepath.Join(keysDir, keyName)

		// Read EC public key
		ecPubBytes, err := os.ReadFile(filepath.Join(keyDir, "ec", "public.key")) //nolint:gosec // G304: Reading from app's key directory
		if err != nil {
			return fmt.Errorf("failed to read EC public key for %s: %w", keyName, err)
		}
		ecPubHex := strings.TrimSpace(string(ecPubBytes))

		// Derive EVM address from EC public key
		ecPubDecoded, err := hex.DecodeString(ecPubHex)
		if err != nil {
			return fmt.Errorf("failed to decode EC public key for %s: %w", keyName, err)
		}
		evmAddr := crypto.Keccak256(ecPubDecoded)[12:]
		ethAddr := fmt.Sprintf("0x%x", evmAddr)

		// Derive P-Chain address (bech32)
		sha256Hash := sha256.Sum256(ecPubDecoded)
		ripemdHasher := ripemd160.New() //nolint:gosec // G406: Required for legacy address derivation
		ripemdHasher.Write(sha256Hash[:])
		shortID := ripemdHasher.Sum(nil)
		luxAddr, err := formatLuxAddress("P", config.HRP, shortID)
		if err != nil {
			return fmt.Errorf("failed to format P-Chain address for %s: %w", keyName, err)
		}

		// Read BLS keys for staker info
		blsPubBytes, err := os.ReadFile(filepath.Join(keyDir, "bls", "public.key")) //nolint:gosec // G304: Reading from app's key directory
		if err != nil {
			return fmt.Errorf("failed to read BLS public key for %s: %w", keyName, err)
		}
		blsPopBytes, err := os.ReadFile(filepath.Join(keyDir, "bls", "pop.key")) //nolint:gosec // G304: Reading from app's key directory
		if err != nil {
			return fmt.Errorf("failed to read BLS PoP for %s: %w", keyName, err)
		}

		// Create allocation
		var alloc GenesisAllocation
		if i == 0 || config.VestingYears == 0 {
			// First key or no vesting: fully unlocked
			alloc = GenesisAllocation{
				EthAddr:       ethAddr,
				LuxAddr:       luxAddr,
				InitialAmount: amountPerKey,
				UnlockSchedule: []UnlockSchedule{
					{Amount: amountPerKey, Locktime: 0},
				},
			}
			ux.Logger.Info("P-Chain Key %d (%s): fully unlocked - %s", i+1, keyName, luxAddr)
		} else {
			// Vesting schedule
			alloc = createVestingAllocation(ethAddr, luxAddr, amountPerKey, config.VestingYears, config.VestingPercent)
			ux.Logger.Info("P-Chain Key %d (%s): %d-year vesting - %s", i+1, keyName, config.VestingYears, luxAddr)
		}
		pAllocations = append(pAllocations, alloc)

		// Create staker entry
		initialStakedFunds = append(initialStakedFunds, luxAddr)
		staker := InitialStaker{
			NodeID:        fmt.Sprintf("NodeID-PLACEHOLDER-%d", i+1),
			RewardAddress: luxAddr,
			DelegationFee: 20000, // 2%
		}
		staker.Signer.PublicKey = "0x" + strings.TrimSpace(string(blsPubBytes))
		staker.Signer.ProofOfPossession = "0x" + strings.TrimSpace(string(blsPopBytes))
		initialStakers = append(initialStakers, staker)
	}

	// Generate X-Chain allocations
	xAllocations := []GenesisAllocation{}
	for i, keyName := range xKeys {
		keyDir := filepath.Join(keysDir, keyName)

		ecPubBytes, err := os.ReadFile(filepath.Join(keyDir, "ec", "public.key")) //nolint:gosec // G304: Reading from app's key directory
		if err != nil {
			return fmt.Errorf("failed to read EC public key for %s: %w", keyName, err)
		}
		ecPubHex := strings.TrimSpace(string(ecPubBytes))

		ecPubDecoded, err := hex.DecodeString(ecPubHex)
		if err != nil {
			return fmt.Errorf("failed to decode EC public key for %s: %w", keyName, err)
		}
		evmAddr := crypto.Keccak256(ecPubDecoded)[12:]
		ethAddr := fmt.Sprintf("0x%x", evmAddr)

		sha256HashX := sha256.Sum256(ecPubDecoded)
		ripemdHasherX := ripemd160.New() //nolint:gosec // G406: Required for legacy address derivation
		ripemdHasherX.Write(sha256HashX[:])
		shortID := ripemdHasherX.Sum(nil)
		luxAddr, err := formatLuxAddress("X", config.HRP, shortID)
		if err != nil {
			return fmt.Errorf("failed to format X-Chain address for %s: %w", keyName, err)
		}

		var alloc GenesisAllocation
		if config.VestingYears == 0 {
			alloc = GenesisAllocation{
				EthAddr:       ethAddr,
				LuxAddr:       luxAddr,
				InitialAmount: amountPerKey,
				UnlockSchedule: []UnlockSchedule{
					{Amount: amountPerKey, Locktime: 0},
				},
			}
			ux.Logger.Info("X-Chain Key %d (%s): fully unlocked - %s", i+1, keyName, luxAddr)
		} else {
			alloc = createVestingAllocation(ethAddr, luxAddr, amountPerKey, config.VestingYears, config.VestingPercent)
			ux.Logger.Info("X-Chain Key %d (%s): %d-year vesting - %s", i+1, keyName, config.VestingYears, luxAddr)
		}
		xAllocations = append(xAllocations, alloc)
	}

	// Create X-Chain genesis JSON
	xGenesis := XChainGenesis{
		Allocations:                xAllocations,
		StartTime:                  uint64(time.Now().Unix()), //nolint:gosec // G115: Unix time is positive
		InitialStakeDuration:       secondsPerYear,
		InitialStakeDurationOffset: 5400,
		InitialStakedFunds:         []string{},
		InitialStakers:             []interface{}{},
	}
	xGenesisBytes, err := json.Marshal(xGenesis)
	if err != nil {
		return fmt.Errorf("failed to marshal X-Chain genesis: %w", err)
	}

	// Load existing C-Chain genesis if specified
	cChainGenesis := getDefaultCChainGenesis(config.ChainID)
	if preserveCGenesis != "" {
		existingGenesis, err := os.ReadFile(preserveCGenesis) //nolint:gosec // G304: User-specified file for genesis preservation
		if err != nil {
			return fmt.Errorf("failed to read existing genesis: %w", err)
		}
		var existing Genesis
		if err := json.Unmarshal(existingGenesis, &existing); err != nil {
			return fmt.Errorf("failed to parse existing genesis: %w", err)
		}
		cChainGenesis = existing.CChainGenesis
		ux.Logger.Info("Preserving C-Chain genesis from %s", preserveCGenesis)
	}

	// Create final genesis
	genesis := Genesis{
		NetworkID:                  config.NetworkID,
		Allocations:                pAllocations,
		StartTime:                  uint64(time.Now().Unix()), //nolint:gosec // G115: Unix time is positive
		InitialStakeDuration:       secondsPerYear,
		InitialStakeDurationOffset: 5400,
		InitialStakedFunds:         initialStakedFunds,
		InitialStakers:             initialStakers,
		CChainGenesis:              cChainGenesis,
		XChainGenesis:              string(xGenesisBytes),
		Message:                    fmt.Sprintf("%s - %d Validators", config.Message, len(initialStakers)),
	}

	// Write output
	output, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal genesis: %w", err)
	}

	if err := os.WriteFile(actualOutput, output, 0o644); err != nil { //nolint:gosec // G306: Genesis file needs to be readable
		return fmt.Errorf("failed to write genesis file: %w", err)
	}

	ux.Logger.Info("")
	ux.Logger.Info("Genesis file written to: %s", actualOutput)

	// Also save to ~/.lux/networks/<network>/genesis.json if --save flag is set
	if saveToLux && outputFile != "" {
		networksDir := filepath.Join(app.GetBaseDir(), "networks", networkName)
		if err := os.MkdirAll(networksDir, 0o750); err != nil {
			return fmt.Errorf("failed to create networks directory: %w", err)
		}
		defaultOutput := filepath.Join(networksDir, "genesis.json")
		if actualOutput != defaultOutput {
			if err := os.WriteFile(defaultOutput, output, 0o644); err != nil { //nolint:gosec // G306: Genesis file needs to be readable
				return fmt.Errorf("failed to write genesis to ~/.lux: %w", err)
			}
			ux.Logger.Info("Genesis also saved to: %s", defaultOutput)
		}
	}

	ux.Logger.Info("Network: %s (ID: %d)", networkName, config.NetworkID)
	ux.Logger.Info("P-Chain allocations: %d", len(pAllocations))
	ux.Logger.Info("X-Chain allocations: %d", len(xAllocations))
	ux.Logger.Info("Initial stakers: %d", len(initialStakers))
	if len(initialStakers) > 0 {
		ux.Logger.Info("")
		ux.Logger.Info("NOTE: Update initialStakers with actual NodeIDs before deployment")
	}

	return nil
}

// generateKeySet creates a new key set using the existing key creation logic
func generateKeySet(name string) error {
	// Use the existing create command logic
	// This is a simplified version - in production, call the actual key creation
	cmd := newCreateCmd()
	cmd.SetArgs([]string{name})
	return cmd.Execute()
}

func createVestingAllocation(ethAddr, luxAddr string, totalAmount uint64, years int, percentPerYear float64) GenesisAllocation {
	schedule := []UnlockSchedule{}
	unlockPerPeriod := uint64(float64(totalAmount) * percentPerYear / 100)
	remaining := totalAmount

	for i := 0; i < years && remaining > 0; i++ {
		unlock := unlockPerPeriod
		if unlock > remaining {
			unlock = remaining
		}
		locktime := uint64(jan2020 + (i+1)*secondsPerYear) //nolint:gosec // G115: Vesting timestamps are bounded
		schedule = append(schedule, UnlockSchedule{
			Amount:   unlock,
			Locktime: locktime,
		})
		remaining -= unlock
	}

	// Handle any remainder in final unlock
	if remaining > 0 && len(schedule) > 0 {
		schedule[len(schedule)-1].Amount += remaining
	}

	return GenesisAllocation{
		EthAddr:        ethAddr,
		LuxAddr:        luxAddr,
		InitialAmount:  totalAmount,
		UnlockSchedule: schedule,
	}
}

// formatLuxAddress creates a Lux address with proper bech32 encoding
// chainPrefix: "P", "X", "C", etc.
// hrp: "lux", "test", "local" - the bech32 Human Readable Part
// data: 20-byte address (RIPEMD-160 hash of SHA256 of public key)
//
// The result is: chainPrefix-hrp1<bech32data>
// Example: P-lux1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq8qwm4a
//
// IMPORTANT: The bech32 checksum is computed using ONLY the hrp ("lux"),
// NOT the chain prefix ("P-"). This matches the node's address.Format().
func formatLuxAddress(chainPrefix, hrp string, data []byte) (string, error) {
	converted, err := bech32ConvertBits(data, 8, 5, true)
	if err != nil {
		return "", err
	}
	// Compute bech32 with just the HRP (lux, test, local)
	bech32Addr := bech32Encode(hrp, converted)
	// Prepend chain prefix: P-lux1..., X-lux1..., etc.
	return chainPrefix + "-" + bech32Addr, nil
}

// Bech32 encoding helpers
const bech32Charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"

func bech32ConvertBits(data []byte, fromBits, toBits uint, pad bool) ([]byte, error) {
	acc := uint(0)
	bits := uint(0)
	ret := []byte{}
	maxv := uint((1 << toBits) - 1)

	for _, b := range data {
		acc = (acc << fromBits) | uint(b)
		bits += fromBits
		for bits >= toBits {
			bits -= toBits
			ret = append(ret, byte((acc>>bits)&maxv))
		}
	}

	if pad {
		if bits > 0 {
			ret = append(ret, byte((acc<<(toBits-bits))&maxv))
		}
	} else if bits >= fromBits || ((acc<<(toBits-bits))&maxv) != 0 {
		return nil, fmt.Errorf("invalid padding")
	}

	return ret, nil
}

func bech32Encode(hrp string, data []byte) string {
	combined := append([]byte{}, data...)
	checksum := bech32Checksum(hrp, combined)
	combined = append(combined, checksum...)

	result := hrp + "1"
	for _, b := range combined {
		result += string(bech32Charset[b])
	}
	return result
}

func bech32Checksum(hrp string, data []byte) []byte {
	values := append(bech32HrpExpand(hrp), data...)
	values = append(values, 0, 0, 0, 0, 0, 0)
	polymod := bech32Polymod(values) ^ 1
	checksum := make([]byte, 6)
	for i := 0; i < 6; i++ {
		checksum[i] = byte((polymod >> (5 * (5 - i))) & 31)
	}
	return checksum
}

func bech32HrpExpand(hrp string) []byte {
	ret := make([]byte, len(hrp)*2+1)
	for i, c := range hrp {
		ret[i] = byte(c >> 5)
		ret[len(hrp)+1+i] = byte(c & 31)
	}
	ret[len(hrp)] = 0
	return ret
}

func bech32Polymod(values []byte) uint32 {
	gen := []uint32{0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3}
	chk := uint32(1)
	for _, v := range values {
		b := chk >> 25
		chk = (chk&0x1ffffff)<<5 ^ uint32(v)
		for i := 0; i < 5; i++ {
			if (b>>i)&1 == 1 {
				chk ^= gen[i]
			}
		}
	}
	return chk
}

// getDefaultCChainGenesis returns the canonical C-chain genesis from the genesis repo.
// Falls back to a minimal genesis only if canonical config is not available.
func getDefaultCChainGenesis(networkID uint32) string {
	// Try to get canonical genesis from github.com/luxfi/genesis/configs
	genesisBytes, err := genesiscfg.GetCanonicalGenesisBytes(networkID)
	if err == nil {
		// Parse and extract cChainGenesis
		var fullGenesis struct {
			CChainGenesis string `json:"cChainGenesis"`
		}
		if err := json.Unmarshal(genesisBytes, &fullGenesis); err == nil && fullGenesis.CChainGenesis != "" {
			return fullGenesis.CChainGenesis
		}
	}

	// Fallback: try GetGenesis
	genesisBytes, err = genesiscfg.GetGenesis(networkID)
	if err == nil {
		var fullGenesis struct {
			CChainGenesis string `json:"cChainGenesis"`
		}
		if err := json.Unmarshal(genesisBytes, &fullGenesis); err == nil && fullGenesis.CChainGenesis != "" {
			return fullGenesis.CChainGenesis
		}
	}

	// Final fallback: minimal genesis (should rarely be used)
	// This is only for truly custom networks with no canonical config
	ux.Logger.Info("Warning: Using minimal C-chain genesis for network %d (no canonical config found)", networkID)
	return fmt.Sprintf(`{"config":{"chainId":%d,"homesteadBlock":0,"eip150Block":0,"eip150Hash":"0x2086799aeebeae135c246c65021c82b4e15a2c451340993aacfd2751886514f0","eip155Block":0,"eip158Block":0,"byzantiumBlock":0,"constantinopleBlock":0,"petersburgBlock":0,"istanbulBlock":0,"muirGlacierBlock":0,"berlinBlock":0,"londonBlock":0,"apricotPhase1BlockTimestamp":0,"apricotPhase2BlockTimestamp":0,"apricotPhase3BlockTimestamp":0,"apricotPhase4BlockTimestamp":0,"apricotPhase5BlockTimestamp":0,"durangoBlockTimestamp":0,"etnaTimestamp":1800000000,"feeConfig":{"gasLimit":30000000,"minBaseFee":25000000000,"targetGas":100000000,"baseFeeChangeDenominator":36,"minBlockGasCost":0,"maxBlockGasCost":10000000,"targetBlockRate":2,"blockGasCostStep":500000}},"alloc":{},"nonce":"0x0","timestamp":"0x0","extraData":"0x00","gasLimit":"0x1C9C380","difficulty":"0x0","mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000","coinbase":"0x0000000000000000000000000000000000000000","number":"0x0","gasUsed":"0x0","parentHash":"0x0000000000000000000000000000000000000000000000000000000000000000"}`, networkID)
}
