// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chaincmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/luxfi/cli/pkg/prompts"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/constants"
	"github.com/luxfi/evm/core"
	"github.com/luxfi/sdk/models"
	"github.com/spf13/cobra"
)

var (
	chainType        string // l1, l2, l3
	sequencerType    string // lux, ethereum, op, external
	forceCreate      bool
	genesisFile      string
	customVMBin      string
	useEVM           bool
	useCustomVM      bool
	vmVersion        string
	useLatestVM      bool
	enablePreconfirm bool

	// Genesis configuration flags
	evmChainID     uint64 // EVM chain ID (default: 200200)
	tokenName      string // Token name (default: TOKEN)
	tokenSymbol    string // Token symbol (default: TKN)
	airdropAddress string // Address to airdrop tokens to
	airdropAmount  string // Amount to airdrop (in wei, default: 1000000 ether)
)

func newCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [chainName]",
		Short: "Create a new blockchain configuration",
		Long: `Create a new blockchain configuration for deployment.

OVERVIEW:

  Creates a blockchain configuration with genesis file and metadata.
  The configuration is stored in ~/.lux/chains/<chainName>/ and can
  be deployed to any network (local, testnet, mainnet).

CHAIN TYPES:

  l1    Sovereign L1 with independent validation
  l2    Layer 2 rollup/subnet (default)
  l3    App-specific L3 chain

SEQUENCER OPTIONS (for L2):

  lux       Lux-based rollup, 100ms blocks (default, lowest cost)
  ethereum  Ethereum-based rollup, 12s blocks (highest security)
  op        OP Stack compatible
  external  External/custom sequencer

VM OPTIONS:

  --evm          Use Lux EVM (default)
  --custom-vm    Use custom VM binary
  --vm           Path to custom VM binary
  --vm-version   Specific VM version (default: latest)
  --latest       Use latest VM version

GENESIS OPTIONS:

  --genesis           Path to custom genesis.json file
                      If not provided, generates default EVM genesis
  --evm-chain-id      EVM chain ID (default: 200200)
  --token-name        Native token name (default: TOKEN)
  --token-symbol      Native token symbol (default: TKN)
  --airdrop-address   Address to airdrop tokens to (default: test account)
  --airdrop-amount    Amount to airdrop in wei (default: 1000000000000000000000000)

NON-INTERACTIVE MODE:

  Non-interactive mode is automatically enabled when:
    - LUX_NON_INTERACTIVE=1 environment variable is set
    - CI=1 environment variable is set (common in CI/CD pipelines)
    - stdin is not a TTY (piped input, scripts, etc.)

  In non-interactive mode, sensible defaults are used for optional values.
  Required values must be provided via flags.

OTHER OPTIONS:

  --force, -f              Overwrite existing configuration
  --enable-preconfirm      Enable pre-confirmations (<100ms acknowledgment)

EXAMPLES:

  # Create default L2 chain with Lux sequencing
  lux chain create mychain

  # Create sovereign L1
  lux chain create mychain --type=l1

  # Create with Ethereum sequencing (12s blocks)
  lux chain create mychain --sequencer=ethereum

  # Create with custom genesis
  lux chain create mychain --genesis=~/custom-genesis.json

  # Create L3 on existing L2
  lux chain create myapp --type=l3

  # Overwrite existing configuration
  lux chain create mychain --force

  # Create with pre-confirmations enabled
  lux chain create mychain --enable-preconfirm

  # Non-interactive in CI/CD (env var triggers non-interactive mode)
  CI=1 lux chain create mychain

  # Non-interactive with custom chain ID
  LUX_NON_INTERACTIVE=1 lux chain create mychain --evm-chain-id=12345

  # Piped input also triggers non-interactive mode
  echo "" | lux chain create mychain --evm-chain-id=12345

OUTPUT:

  Creates two files in ~/.lux/chains/<chainName>/:
  - genesis.json    Blockchain genesis configuration
  - sidecar.json    Metadata (VM type, versions, deployment info)

NEXT STEPS:

  After creating a chain configuration:
  1. Start a network:     lux network start --devnet
  2. Deploy the chain:    lux chain deploy mychain --devnet
  3. Verify deployment:   lux network status

NOTES:

  - Chain names must be unique and ≤32 characters
  - Reserved names: c, p, x, primary, platform
  - Default genesis includes funded test account
  - Genesis can be customized after creation`,
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		RunE:         createChain,
	}

	cmd.Flags().StringVar(&chainType, "type", "l2", "Chain type: l1, l2, l3")
	cmd.Flags().StringVar(&sequencerType, "sequencer", "lux", "Sequencer: lux, ethereum, op, external")
	cmd.Flags().BoolVarP(&forceCreate, "force", "f", false, "Overwrite existing configuration")
	cmd.Flags().StringVar(&genesisFile, "genesis", "", "Path to custom genesis file")
	cmd.Flags().StringVar(&customVMBin, "vm", "", "Path to custom VM binary")
	cmd.Flags().BoolVar(&useEVM, "evm", false, "Use Lux EVM")
	cmd.Flags().BoolVar(&useCustomVM, "custom-vm", false, "Use custom VM")
	cmd.Flags().StringVar(&vmVersion, "vm-version", "", "VM version to use")
	cmd.Flags().BoolVar(&useLatestVM, "latest", false, "Use latest VM version")
	cmd.Flags().BoolVar(&enablePreconfirm, "enable-preconfirm", false, "Enable pre-confirmations")

	// Genesis configuration flags
	cmd.Flags().Uint64Var(&evmChainID, "evm-chain-id", 0, "EVM chain ID (default: 200200)")
	cmd.Flags().StringVar(&tokenName, "token-name", "", "Native token name (default: TOKEN)")
	cmd.Flags().StringVar(&tokenSymbol, "token-symbol", "", "Native token symbol (default: TKN)")
	cmd.Flags().StringVar(&airdropAddress, "airdrop-address", "", "Address to airdrop tokens to")
	cmd.Flags().StringVar(&airdropAmount, "airdrop-amount", "", "Amount to airdrop in wei")

	return cmd
}

func createChain(cmd *cobra.Command, args []string) error {
	chainName := args[0]

	// Validate chain name
	if err := validateChainName(chainName); err != nil {
		return err
	}

	// Check if configuration already exists
	if app.ChainConfigExists(chainName) && !forceCreate {
		return fmt.Errorf("chain %s already exists. Use --force to overwrite", chainName)
	}

	// Determine VM type
	var vmType models.VMType
	switch {
	case useEVM:
		vmType = models.EVM
	case useCustomVM:
		vmType = models.CustomVM
	default:
		// Default to EVM for all chain types
		vmType = models.EVM
	}

	// Handle genesis
	var chainGenesis []byte
	var err error
	if genesisFile != "" {
		chainGenesis, err = os.ReadFile(genesisFile) //nolint:gosec // G304: User-specified genesis file
		if err != nil {
			return fmt.Errorf("failed to read genesis file: %w", err)
		}
		ux.Logger.PrintToUser("Importing genesis")
	} else {
		// Generate default genesis
		chainGenesis, err = generateDefaultGenesis(chainName, chainType)
		if err != nil {
			return fmt.Errorf("failed to generate genesis: %w", err)
		}
	}

	// Validate genesis
	if vmType == models.EVM {
		var genesis core.Genesis
		if err := json.Unmarshal(chainGenesis, &genesis); err != nil {
			return fmt.Errorf("invalid genesis format: %w", err)
		}
	}

	// Resolve chain ID - prompt if not provided and interactive
	resolvedChainID := evmChainID
	if resolvedChainID == 0 {
		if !prompts.IsInteractive() {
			resolvedChainID = 200200 // default
		} else {
			chainIDStr, err := app.Prompt.CaptureString("Enter EVM chain ID (default: 200200)")
			if err != nil {
				return err
			}
			if chainIDStr == "" {
				resolvedChainID = 200200
			} else {
				parsed, err := strconv.ParseUint(chainIDStr, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid chain ID: %w", err)
				}
				resolvedChainID = parsed
			}
		}
	}

	// Resolve token name - prompt if not provided and interactive
	resolvedTokenName := tokenName
	if resolvedTokenName == "" {
		if !prompts.IsInteractive() {
			resolvedTokenName = "TOKEN" // default
		} else {
			name, err := app.Prompt.CaptureString("Enter token name (default: TOKEN)")
			if err != nil {
				return err
			}
			if name == "" {
				resolvedTokenName = "TOKEN"
			} else {
				resolvedTokenName = name
			}
		}
	}

	// Resolve token symbol - prompt if not provided and interactive
	resolvedTokenSymbol := tokenSymbol
	if resolvedTokenSymbol == "" {
		if !prompts.IsInteractive() {
			resolvedTokenSymbol = "TKN" // default
		} else {
			symbol, err := app.Prompt.CaptureString("Enter token symbol (default: TKN)")
			if err != nil {
				return err
			}
			if symbol == "" {
				resolvedTokenSymbol = "TKN"
			} else {
				resolvedTokenSymbol = symbol
			}
		}
	}

	// Create sidecar configuration
	sc := models.Sidecar{
		Name:              chainName,
		VM:                vmType,
		Net:               chainName, // Network name (not subnet)
		TokenName:         resolvedTokenName,
		TokenSymbol:       resolvedTokenSymbol,
		ChainID:           fmt.Sprintf("%d", resolvedChainID),
		Version:           "1.4.0",
		BasedRollup:       chainType == "l2",
		Sovereign:         chainType == "l1",
		SequencerType:     sequencerType,
		PreconfirmEnabled: enablePreconfirm,
		ChainLayer:        getChainLayer(chainType),
	}

	// Set L1 block time based on sequencer
	switch sequencerType {
	case "lux":
		sc.L1BlockTime = 100 // 100ms
	case "ethereum":
		sc.L1BlockTime = 12000 // 12s
	default:
		sc.L1BlockTime = 2000 // 2s default
	}

	// Handle custom VM
	if useCustomVM && customVMBin != "" {
		// TODO: Implement custom VM copy
		ux.Logger.PrintToUser("Custom VM support coming soon")
	}

	// Get VM version and RPC version if using EVM
	if vmType == models.EVM {
		if vmVersion != "" {
			sc.VMVersion = vmVersion
		} else {
			sc.VMVersion = constants.DefaultEVMVersion // Default EVM version
		}
		// Set correct RPC version for Lux EVM
		// This must match the running node's EVM RPC version
		sc.RPCVersion = constants.DefaultEVMRPCVersion
	}

	// Create chain directory
	chainDir := filepath.Join(app.GetChainsDir(), chainName)
	if err := os.MkdirAll(chainDir, constants.DefaultPerms755); err != nil {
		return fmt.Errorf("failed to create chain directory: %w", err)
	}

	// Write genesis
	genesisPath := filepath.Join(chainDir, constants.GenesisFileName)
	if err := os.WriteFile(genesisPath, chainGenesis, constants.WriteReadReadPerms); err != nil {
		return fmt.Errorf("failed to write genesis: %w", err)
	}

	// Write sidecar
	if err := app.CreateSidecar(&sc); err != nil {
		return fmt.Errorf("failed to create sidecar: %w", err)
	}

	// Success message
	ux.Logger.PrintToUser("Creating %s chain %s", chainType, chainName)
	ux.Logger.PrintToUser("Chain Configuration:")
	ux.Logger.PrintToUser("   Type: %s", strings.ToUpper(chainType))
	ux.Logger.PrintToUser("   Chain ID: %s", sc.ChainID)
	ux.Logger.PrintToUser("   Token: %s (%s)", sc.TokenName, sc.TokenSymbol)
	if chainType == "l2" {
		ux.Logger.PrintToUser("   Sequencer: %s", sequencerType)
		ux.Logger.PrintToUser("   Block Time: %dms", sc.L1BlockTime)
	}
	ux.Logger.PrintToUser("Successfully created chain configuration")
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Next steps:")
	ux.Logger.PrintToUser("   1. Start network:  lux network start --devnet")
	ux.Logger.PrintToUser("   2. Deploy chain:   lux chain deploy %s --devnet", chainName)

	return nil
}

func validateChainName(name string) error {
	if name == "" {
		return errors.New("chain name cannot be empty")
	}
	if len(name) > 32 {
		return errors.New("chain name must be 32 characters or less")
	}
	// Check for reserved names
	reserved := []string{"c", "p", "x", "primary", "platform"}
	for _, r := range reserved {
		if strings.EqualFold(name, r) {
			return fmt.Errorf("%s is a reserved chain name", name)
		}
	}
	return nil
}

// getChainLayer returns the chain layer (1=L1, 2=L2, 3=L3)
func getChainLayer(chainType string) int {
	switch chainType {
	case "l1":
		return 1
	case "l2":
		return 2
	case "l3":
		return 3
	default:
		return 2 // Default to L2
	}
}

// genesisParams contains parameters for genesis generation
type genesisParams struct {
	chainID        uint64
	airdropAddress string
	airdropAmount  string // hex-encoded balance
}

// getGenesisParams resolves genesis parameters from flags or defaults
func getGenesisParams() genesisParams {
	params := genesisParams{
		chainID:        200200,                                     // Default chain ID
		airdropAddress: "9011E888251AB053B7bD1cdB598Db4f9DEd94714", // Default test account
		airdropAmount:  "0x193e5939a08ce9dbd480000000",             // ~500M tokens
	}

	// Override with flags if provided
	if evmChainID != 0 {
		params.chainID = evmChainID
	}

	if airdropAddress != "" {
		// Strip 0x prefix if present for consistency
		addr := airdropAddress
		if strings.HasPrefix(addr, "0x") || strings.HasPrefix(addr, "0X") {
			addr = addr[2:]
		}
		params.airdropAddress = addr
	}

	if airdropAmount != "" {
		// If provided as decimal, convert to hex
		if !strings.HasPrefix(airdropAmount, "0x") {
			// Parse as decimal and convert to hex
			if n, ok := new(big.Int).SetString(airdropAmount, 10); ok {
				params.airdropAmount = "0x" + n.Text(16)
			} else {
				// Already hex or invalid, use as-is
				params.airdropAmount = airdropAmount
			}
		} else {
			params.airdropAmount = airdropAmount
		}
	}

	return params
}

func generateDefaultGenesis(_, _ string) ([]byte, error) {
	params := getGenesisParams()

	// Default genesis for EVM-compatible chains
	genesis := map[string]interface{}{
		"config": map[string]interface{}{
			"chainId":             params.chainID,
			"homesteadBlock":      0,
			"eip150Block":         0,
			"eip155Block":         0,
			"eip158Block":         0,
			"byzantiumBlock":      0,
			"constantinopleBlock": 0,
			"petersburgBlock":     0,
			"istanbulBlock":       0,
			"muirGlacierBlock":    0,
			"evmTimestamp":        0,
			"feeConfig": map[string]interface{}{
				"gasLimit":                 8000000,
				"targetBlockRate":          2,
				"minBaseFee":               25000000000,
				"targetGas":                15000000,
				"baseFeeChangeDenominator": 36,
				"minBlockGasCost":          0,
				"maxBlockGasCost":          1000000,
				"blockGasCostStep":         200000,
			},
			"allowFeeRecipients": true,
		},
		"alloc": map[string]interface{}{
			params.airdropAddress: map[string]interface{}{
				"balance": params.airdropAmount,
			},
		},
		"nonce":      "0x0",
		"timestamp":  "0x6727e9c3",
		"extraData":  "0x",
		"gasLimit":   "0x7a1200",
		"difficulty": "0x0",
		"mixHash":    "0x0000000000000000000000000000000000000000000000000000000000000000",
		"coinbase":   "0x0000000000000000000000000000000000000000",
	}

	return json.MarshalIndent(genesis, "", "  ")
}
