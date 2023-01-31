// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package vm

import (
	"bytes"
	"encoding/json"
	"errors"
	"math/big"
	"os"

	"github.com/luxdefi/cli/pkg/application"
	"github.com/luxdefi/cli/pkg/constants"
	"github.com/luxdefi/cli/pkg/models"
	"github.com/luxdefi/cli/pkg/statemachine"
	"github.com/luxdefi/cli/pkg/ux"
	"github.com/luxdefi/subnet-evm/core"
	"github.com/luxdefi/subnet-evm/params"
	"github.com/ethereum/go-ethereum/common"
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

		subnetEVMVersion, _, err = getVMVersion(app, "Subnet-EVM", constants.SubnetEVMRepoName, subnetEVMVersion, false)
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

	if conf != nil && conf.TxAllowListConfig != nil {
		if err := ensureAdminsHaveBalance(conf.TxAllowListConfig.AllowListAdmins, allocation); err != nil {
			return nil, nil, err
		}
	}

	conf.ChainID = chainID

	genesis.Alloc = allocation
	genesis.Config = conf
	genesis.Difficulty = Difficulty
	genesis.GasLimit = GasLimit

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
	return getAllocation(app, defaultEvmAirdropAmount, oneAvax, "Amount to airdrop (in AVAX units)")
}
