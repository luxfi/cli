// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package vm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/models"
	"github.com/luxfi/cli/pkg/statemachine"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/evm/core"
	"github.com/luxfi/evm/params"
	"github.com/luxfi/evm/precompile/contracts/txallowlist"
	"github.com/luxfi/geth/common"
)

func CreateEvmSubnetConfig(app *application.Lux, subnetName string, genesisPath string, subnetEVMVersion string) ([]byte, *models.Sidecar, error) {
	var (
		genesisBytes []byte
		sc           *models.Sidecar
		err          error
	)

	if genesisPath == "" {
		genesisBytes, sc, err = createEvmGenesis(app, subnetName, subnetEVMVersion)
		if err != nil {
			return nil, &models.Sidecar{}, err
		}
	} else {
		ux.Logger.PrintToUser("Importing genesis")
		genesisBytes, err = os.ReadFile(genesisPath)
		if err != nil {
			return nil, &models.Sidecar{}, err
		}

		subnetEVMVersion, err = getVMVersion(app, "Subnet-EVM", constants.SubnetEVMRepoName, subnetEVMVersion, false)
		if err != nil {
			return nil, &models.Sidecar{}, err
		}

		rpcVersion, err := GetRPCProtocolVersion(app, models.SubnetEvm, subnetEVMVersion)
		if err != nil {
			return nil, &models.Sidecar{}, err
		}

		sc = &models.Sidecar{
			Name:       subnetName,
			VM:         models.SubnetEvm,
			VMVersion:  subnetEVMVersion,
			RPCVersion: rpcVersion,
			Subnet:     subnetName,
			TokenName:  "",
		}
	}

	return genesisBytes, sc, nil
}

func createEvmGenesis(
	app *application.Lux,
	subnetName string,
	subnetEVMVersion string,
) ([]byte, *models.Sidecar, error) {
	ux.Logger.PrintToUser("creating subnet %s", subnetName)

	genesis := core.Genesis{}
	conf := params.SubnetEVMDefaultChainConfig

	const (
		descriptorsState = "descriptors"
		feeState         = "fee"
		airdropState     = "airdrop"
		precompilesState = "precompiles"
	)

	var (
		chainID    *big.Int
		tokenName  string
		vmVersion  string
		allocation core.GenesisAlloc
		direction  statemachine.StateDirection
		err        error
	)

	subnetEvmState, err := statemachine.NewStateMachine(
		[]string{descriptorsState, feeState, airdropState, precompilesState},
	)
	if err != nil {
		return nil, nil, err
	}
	for subnetEvmState.Running() {
		switch subnetEvmState.CurrentState() {
		case descriptorsState:
			chainID, tokenName, vmVersion, direction, err = getDescriptors(app, subnetEVMVersion)
		case feeState:
			*conf, direction, err = GetFeeConfig(*conf, app)
		case airdropState:
			allocation, direction, err = getEVMAllocation(app)
		case precompilesState:
			*conf, direction, err = getPrecompiles(*conf, app)
		default:
			err = errors.New("invalid creation stage")
		}
		if err != nil {
			return nil, nil, err
		}
		subnetEvmState.NextState(direction)
	}

	if conf != nil && conf.GenesisPrecompiles[txallowlist.ConfigKey] != nil {
		allowListCfg, ok := conf.GenesisPrecompiles[txallowlist.ConfigKey].(*txallowlist.Config)
		if !ok {
			return nil, nil, fmt.Errorf("expected config of type txallowlist.AllowListConfig, but got %T", allowListCfg)
		}

		if err := ensureAdminsHaveBalance(
			allowListCfg.AdminAddresses,
			allocation); err != nil {
			return nil, nil, err
		}
	}

	conf.ChainID = chainID

	genesis.Alloc = allocation
	genesis.Config = conf
	genesis.Difficulty = Difficulty
	genesis.GasLimit = conf.FeeConfig.GasLimit.Uint64()

	jsonBytes, err := genesis.MarshalJSON()
	if err != nil {
		return nil, nil, err
	}

	var prettyJSON bytes.Buffer
	err = json.Indent(&prettyJSON, jsonBytes, "", "    ")
	if err != nil {
		return nil, nil, err
	}

	rpcVersion, err := GetRPCProtocolVersion(app, models.SubnetEvm, vmVersion)
	if err != nil {
		return nil, &models.Sidecar{}, err
	}

	sc := &models.Sidecar{
		Name:       subnetName,
		VM:         models.SubnetEvm,
		VMVersion:  vmVersion,
		RPCVersion: rpcVersion,
		Subnet:     subnetName,
		TokenName:  tokenName,
	}

	return prettyJSON.Bytes(), sc, nil
}

func ensureAdminsHaveBalance(admins []common.Address, alloc core.GenesisAlloc) error {
	if len(admins) < 1 {
		return nil
	}

	for _, admin := range admins {
		// we can break at the first admin who has a non-zero balance
		if bal, ok := alloc[admin]; ok &&
			bal.Balance != nil &&
			bal.Balance.Uint64() > uint64(0) {
			return nil
		}
	}
	return errors.New("none of the addresses in the transaction allow list precompile have any tokens allocated to them. Currently, no address can transact on the network. Airdrop some funds to one of the allow list addresses to continue")
}

// In own function to facilitate testing
func getEVMAllocation(app *application.Lux) (core.GenesisAlloc, statemachine.StateDirection, error) {
	return getAllocation(app, defaultEvmAirdropAmount, oneLux, "Amount to airdrop (in LUX units)")
}

// CreateEVMGenesis creates a new EVM genesis configuration
func CreateEVMGenesis(chainID *big.Int, allocations core.GenesisAlloc, timestamps map[string]uint64) map[string]interface{} {
	// Default configuration
	config := map[string]interface{}{
		"config": map[string]interface{}{
			"chainId":             chainID,
			"homesteadBlock":      0,
			"eip150Block":         0,
			"eip155Block":         0,
			"eip158Block":         0,
			"byzantiumBlock":      0,
			"constantinopleBlock": 0,
			"petersburgBlock":     0,
			"istanbulBlock":       0,
			"muirGlacierBlock":    0,
			"berlinBlock":         0,
			"londonBlock":         0,
			"feeConfig": map[string]interface{}{
				"gasLimit":        8000000,
				"targetBlockRate": 2,
				"minBaseFee":      25000000000,
				"targetGas":       15000000,
				"baseFeeChangeDenominator": 36,
				"minBlockGasCost":  0,
				"maxBlockGasCost":  1000000,
				"blockGasCostStep": 200000,
			},
		},
		"alloc":      allocations,
		"difficulty": "0x0",
		"gasLimit":   "0x7A1200",
		"timestamp":  "0x0",
	}

	// Apply custom timestamps if provided
	if timestamps != nil {
		for key, value := range timestamps {
			config[key] = fmt.Sprintf("0x%x", value)
		}
	}

	return config
}
