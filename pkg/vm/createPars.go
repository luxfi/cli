// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"encoding/json"
	"math/big"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/cli/pkg/statemachine"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/sdk/models"
)

const (
	// ParsDefaultChainID is the default chain ID for Pars networks
	ParsDefaultChainID = 7070
	// ParsOrg is the GitHub organization for Pars
	ParsOrg = "parsdao"
	// ParsRepoName is the repository name for parsd
	ParsRepoName = "node"
)

// ParsGenesisConfig represents the genesis configuration for a Pars chain
type ParsGenesisConfig struct {
	ChainID uint64 `json:"chainId"`
	Network struct {
		RPCAddr   string `json:"rpcAddr"`
		P2PAddr   string `json:"p2pAddr"`
		ChainID   uint64 `json:"chainId"`
		NetworkID uint64 `json:"networkId"`
	} `json:"network"`
	EVM struct {
		Enabled     bool `json:"enabled"`
		Precompiles struct {
			MLDSA    string `json:"mldsa"`
			MLKEM    string `json:"mlkem"`
			BLS      string `json:"bls"`
			Ringtail string `json:"ringtail"`
			FHE      string `json:"fhe"`
		} `json:"precompiles"`
	} `json:"evm"`
	Pars struct {
		Enabled bool `json:"enabled"`
		Storage struct {
			MaxSize       int64 `json:"maxSize"`
			RetentionDays int   `json:"retentionDays"`
		} `json:"storage"`
		Onion struct {
			HopCount int `json:"hopCount"`
		} `json:"onion"`
		Session struct {
			IDPrefix string `json:"idPrefix"`
		} `json:"session"`
	} `json:"pars"`
	Warp struct {
		Enabled     bool   `json:"enabled"`
		LuxEndpoint string `json:"luxEndpoint"`
	} `json:"warp"`
	Crypto struct {
		GPUEnabled      bool   `json:"gpuEnabled"`
		SignatureScheme string `json:"signatureScheme"`
		KEMScheme       string `json:"kemScheme"`
		ThresholdScheme string `json:"thresholdScheme"`
	} `json:"crypto"`
	Consensus struct {
		Engine      string `json:"engine"`
		BlockTimeMs int    `json:"blockTimeMs"`
	} `json:"consensus"`
}

// DefaultParsGenesis returns the default genesis configuration for Pars
func DefaultParsGenesis(chainID uint64) *ParsGenesisConfig {
	cfg := &ParsGenesisConfig{
		ChainID: chainID,
	}

	// Network settings
	cfg.Network.RPCAddr = "127.0.0.1:9650"
	cfg.Network.P2PAddr = "0.0.0.0:9651"
	cfg.Network.ChainID = chainID
	cfg.Network.NetworkID = chainID

	// EVM with PQ precompiles
	cfg.EVM.Enabled = true
	cfg.EVM.Precompiles.MLDSA = "0x0601"
	cfg.EVM.Precompiles.MLKEM = "0x0603"
	cfg.EVM.Precompiles.BLS = "0x0B00"
	cfg.EVM.Precompiles.Ringtail = "0x0700"
	cfg.EVM.Precompiles.FHE = "0x0800"

	// Pars messaging
	cfg.Pars.Enabled = true
	cfg.Pars.Storage.MaxSize = 10737418240 // 10GB
	cfg.Pars.Storage.RetentionDays = 30
	cfg.Pars.Onion.HopCount = 3
	cfg.Pars.Session.IDPrefix = "07" // Post-quantum prefix

	// Warp cross-chain
	cfg.Warp.Enabled = true
	cfg.Warp.LuxEndpoint = "https://api.lux.network"

	// Post-quantum crypto
	cfg.Crypto.GPUEnabled = true
	cfg.Crypto.SignatureScheme = "ML-DSA-65"
	cfg.Crypto.KEMScheme = "ML-KEM-768"
	cfg.Crypto.ThresholdScheme = "Ringtail"

	// Quasar consensus
	cfg.Consensus.Engine = "quasar"
	cfg.Consensus.BlockTimeMs = 2000

	return cfg
}

// CreateParsChainConfig creates a new Pars chain configuration
func CreateParsChainConfig(
	app *application.Lux,
	chainName string,
	vmVersion string,
) ([]byte, *models.Sidecar, error) {
	ux.Logger.PrintToUser("Creating Pars VM chain %s", chainName)

	// Get chain ID
	chainID, err := getParsChainID(app)
	if err != nil {
		return nil, nil, err
	}

	// Get VM version
	vmVersion, err = getParsVersion(app, vmVersion)
	if err != nil {
		return nil, nil, err
	}

	// Create genesis
	genesis := DefaultParsGenesis(chainID.Uint64())
	genesisBytes, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return nil, nil, err
	}

	// Create sidecar
	sc := &models.Sidecar{
		Name:       chainName,
		VM:         models.ParsVM,
		Chain:      chainName,
		VMVersion:  vmVersion,
		TokenName:  "PARS",
		EVMChainID: chainID.String(),
	}

	return genesisBytes, sc, nil
}

func getParsChainID(app *application.Lux) (*big.Int, error) {
	ux.Logger.PrintToUser("Enter chain ID for Pars network (default: %d)", ParsDefaultChainID)

	defaultID := big.NewInt(ParsDefaultChainID)
	chainID, err := app.Prompt.CapturePositiveBigInt("ChainId")
	if err != nil {
		return defaultID, nil
	}
	if chainID.Cmp(big.NewInt(0)) == 0 {
		return defaultID, nil
	}
	return chainID, nil
}

func getParsVersion(app *application.Lux, vmVersion string) (string, error) {
	if vmVersion == "latest" || vmVersion == "" {
		// Get latest release from parsdao/node
		version, err := app.Downloader.GetLatestReleaseVersion(
			binutils.GetGithubLatestReleaseURL(ParsOrg, ParsRepoName),
		)
		if err != nil {
			// Fall back to a default version if release not found
			ux.Logger.PrintToUser("Could not fetch latest version, using v0.1.0")
			return "v0.1.0", nil
		}
		return version, nil
	}
	return vmVersion, nil
}

// GetParsDescriptors prompts for Pars chain configuration
func GetParsDescriptors(app *application.Lux, vmVersion string) (
	*big.Int,
	string,
	string,
	statemachine.StateDirection,
	error,
) {
	chainID, err := getParsChainID(app)
	if err != nil {
		return nil, "", "", statemachine.Stop, err
	}

	tokenName := "PARS" // Fixed token name for Pars

	vmVersion, err = getParsVersion(app, vmVersion)
	if err != nil {
		return nil, "", "", statemachine.Stop, err
	}

	return chainID, tokenName, vmVersion, statemachine.Forward, nil
}
