// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"encoding/json"
	"fmt"

	"github.com/luxfi/cli/pkg/blockchain"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/networkoptions"
	"github.com/luxfi/cli/pkg/precompiles"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/cli/pkg/vm"
	"github.com/luxfi/crypto"
	"github.com/luxfi/geth/core"
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/contract"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/sdk/prompts"
	validatorManagerSDK "github.com/luxfi/sdk/validatormanager"
	"github.com/luxfi/sdk/validatormanager/validatormanagertypes"

	"github.com/luxfi/geth/common"
	"github.com/spf13/cobra"
)

var (
	blockchainIDStr string
	subnetIDstr     string
	useSubnetEvm    bool
	useCustomVM     bool
	rpcURL          string
)

// lux blockchain import public
func newImportPublicCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "public [blockchainPath]",
		Short: "Import an existing blockchain config from running blockchains on a public network",
		RunE:  importPublic,
		Args:  cobrautils.MaximumNArgs(1),
		Long: `The blockchain import public command imports a Blockchain configuration from a running network.

By default, an imported Blockchain
doesn't overwrite an existing Blockchain with the same name. To allow overwrites, provide the --force
flag.`,
	}

	// Network flags are registered at the parent blockchain command level

	cmd.Flags().BoolVar(&useSubnetEvm, "evm", false, "import a subnet-evm")
	cmd.Flags().BoolVar(&useCustomVM, "custom", false, "use a custom VM template")
	cmd.Flags().BoolVar(
		&overwriteImport,
		"force",
		false,
		"overwrite the existing configuration if one exists",
	)
	cmd.Flags().StringVar(
		&blockchainIDStr,
		"blockchain-id",
		"",
		"the blockchain ID",
	)
	cmd.Flags().StringVar(&rpcURL, "rpc", "", "rpc endpoint for the blockchain")
	return cmd
}

func importPublic(*cobra.Command, []string) error {
	network, err := networkoptions.GetNetworkFromCmdLineFlags(
		app,
		"",
		globalNetworkFlags,
		true,
		false,
		networkoptions.DefaultSupportedNetworkOptions,
		"",
	)
	if err != nil {
		return err
	}

	var blockchainID ids.ID
	if blockchainIDStr != "" {
		blockchainID, err = ids.FromString(blockchainIDStr)
		if err != nil {
			return err
		}
	}

	sc, genBytes, err := importBlockchain(network, rpcURL, blockchainID, ux.Logger.PrintToUser)
	if err != nil {
		return err
	}

	sc.TokenName = constants.DefaultTokenName
	sc.TokenSymbol = "TEST" // Default test token symbol

	sc.VM, err = vm.PromptVMType(app, useSubnetEvm, useCustomVM)
	if err != nil {
		return err
	}

	if sc.VM == models.SubnetEvm {
		versions, err := app.Downloader.GetAllReleasesForRepo(constants.LuxOrg, constants.SubnetEVMRepoName)
		if err != nil {
			return err
		}
		sc.VMVersion, err = app.Prompt.CaptureList("Pick the version for this VM", versions)
		if err != nil {
			return err
		}
		sc.RPCVersion, err = vm.GetRPCProtocolVersion(app, sc.VM, sc.VMVersion)
		if err != nil {
			return fmt.Errorf("failed getting RPCVersion for VM type %s with version %s", sc.VM, sc.VMVersion)
		}
		var genesis core.Genesis
		if err := json.Unmarshal(genBytes, &genesis); err != nil {
			return err
		}
		sc.ChainID = genesis.Config.ChainID.String()
	}

	if err := app.CreateSidecar(&sc); err != nil {
		return fmt.Errorf("failed creating the sidecar for import: %w", err)
	}

	if err = app.WriteGenesisFile(sc.Name, genBytes); err != nil {
		return err
	}

	ux.Logger.PrintToUser("Blockchain %q imported successfully", sc.Name)

	return nil
}

func importBlockchain(
	network models.Network,
	rpcURL string,
	blockchainID ids.ID,
	printFunc func(msg string, args ...interface{}),
) (models.Sidecar, []byte, error) {
	var err error

	if rpcURL == "" {
		rpcURL, err = app.Prompt.CaptureStringAllowEmpty("What is the RPC endpoint?")
		if err != nil {
			return models.Sidecar{}, nil, err
		}
		if rpcURL != "" {
			if err := prompts.ValidateURLFormat(rpcURL); err != nil {
				return models.Sidecar{}, nil, fmt.Errorf("invalid url format: %w", err)
			}
		}
	}

	if blockchainID == ids.Empty {
		var err error
		if rpcURL != "" {
			blockchainID, _ = precompiles.WarpPrecompileGetBlockchainID(rpcURL)
		}
		if blockchainID == ids.Empty {
			blockchainID, err = app.Prompt.CaptureID("What is the Blockchain ID?")
			if err != nil {
				return models.Sidecar{}, nil, err
			}
		}
	}

	createChainTx, err := utils.GetBlockchainTx(network.Endpoint(), blockchainID)
	if err != nil {
		return models.Sidecar{}, nil, err
	}

	subnetID := createChainTx.NetID
	vmID := createChainTx.VMID
	blockchainName := createChainTx.ChainName
	genBytes := createChainTx.GenesisData

	printFunc("Retrieved information:")
	printFunc("  Name: %s", blockchainName)
	printFunc("  BlockchainID: %s", blockchainID.String())
	printFunc("  SubnetID: %s", subnetID.String())
	printFunc("  VMID: %s", vmID.String())

	// GetSubnet returns validator info, not permissioning details anymore
	_, err = blockchain.GetSubnet(subnetID, network)
	if err != nil {
		return models.Sidecar{}, nil, err
	}
	// Permissioning check removed as API no longer provides this

	sc := models.Sidecar{
		Name: blockchainName,
		Networks: map[string]models.NetworkData{
			network.Name(): {
				SubnetID:     subnetID,
				BlockchainID: blockchainID,
			},
		},
		Subnet:          blockchainName,
		Version:         constants.SidecarVersion,
		ImportedVMID:    vmID.String(),
		ImportedFromLPM: true,
	}

	if rpcURL != "" {
		e := sc.Networks[network.Name()]
		e.RPCEndpoints = []string{rpcURL}
		sc.Networks[network.Name()] = e
	}

	// Always treat as sovereign for now since API changed
	if true {
		sc.Sovereign = true
		sc.UseACP99 = true
		// ManagerAddress is retrieved from the validator manager contract
		// validatorManagerAddress = "0x" + hex.EncodeToString(subnetInfo.ManagerAddress)
		validatorManagerAddress = "" // Will be populated from contract
		e := sc.Networks[network.Name()]
		e.ValidatorManagerAddress = validatorManagerAddress
		sc.Networks[network.Name()] = e
		printFunc("  Validator Manager Address: %s", validatorManagerAddress)
		if rpcURL != "" && validatorManagerAddress != "" {
			// Convert hex address to crypto.Address
			addr := crypto.Address(common.HexToAddress(validatorManagerAddress).Bytes())
			vmType := validatorManagerSDK.GetValidatorManagerType(rpcURL, addr)
			// Convert type to string
			sc.ValidatorManagement = string(vmType)
			if sc.ValidatorManagement == validatormanagertypes.UndefinedValidatorManagement {
				return models.Sidecar{}, nil, fmt.Errorf("could not obtain validator manager type")
			}
			if sc.ValidatorManagement == validatormanagertypes.ProofOfAuthority {
				// a v2.0.0 validator manager can be identified as PoA for two cases:
				// - it is PoA
				// - it is a validator manager used by v2.0.0 PoS or another specialized validator manager,
				//   in which case the main manager interacts with the P-Chain, and the specialized manager, which is the
				//   owner of this main manager, interacts with the users
				// Convert to crypto.Address for SDK call
				addr := crypto.Address(common.HexToAddress(validatorManagerAddress).Bytes())
				owner, err := contract.GetContractOwner(rpcURL, addr)
				if err != nil {
					return models.Sidecar{}, nil, err
				}
				// check if the owner is a specialized PoS validator manager
				// if this is the case, GetValidatorManagerType will return the corresponding type
				validatorManagement := validatorManagerSDK.GetValidatorManagerType(rpcURL, owner)
				if validatorManagement != validatormanagertypes.UndefinedValidatorManagement {
					printFunc("  Specialized Validator Manager Address: %s", owner)
					e := sc.Networks[network.Name()]
					e.ValidatorManagerAddress = owner.String()
					sc.Networks[network.Name()] = e
					sc.ValidatorManagement = string(validatorManagement)
				} else {
					sc.ValidatorManagerOwner = owner.String()
				}
			}
			printFunc("  Validation Kind: %s", sc.ValidatorManagement)
			if sc.ValidatorManagement == validatormanagertypes.ProofOfAuthority {
				printFunc("  Validator Manager Owner: %s", sc.ValidatorManagerOwner)
			}
		}
	}

	return sc, genBytes, err
}
