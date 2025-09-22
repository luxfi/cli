// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package validatormanager

import (
	"context"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/sdk/contract"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/sdk/evm"
	sdkwarp "github.com/luxfi/sdk/warp"
	sdkutils "github.com/luxfi/sdk/utils"
	"github.com/luxfi/sdk/validator"
	localWarpMessage "github.com/luxfi/sdk/validatormanager/warp"
	"github.com/luxfi/crypto"
	subnetEvmWarp "github.com/luxfi/evm/precompile/contracts/warp"
	ethereum "github.com/luxfi/geth"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/ids"
	"github.com/luxfi/node/proto/pb/platformvm"
	luxdconstants "github.com/luxfi/node/utils/constants"
	"github.com/luxfi/node/utils/logging"
	luxlog "github.com/luxfi/log"
	warp "github.com/luxfi/warp"
	warpPayload "github.com/luxfi/node/vms/platformvm/warp/payload"
	warpMessage "github.com/luxfi/sdk/validatormanager/warp"
)

func InitializeValidatorRegistrationPoSNative(
	rpcURL string,
	managerAddress crypto.Address,
	managerOwnerPrivateKey string,
	nodeID ids.NodeID,
	blsPublicKey []byte,
	expiry uint64,
	balanceOwners localWarpMessage.PChainOwner,
	disableOwners localWarpMessage.PChainOwner,
	delegationFeeBips uint16,
	minStakeDuration time.Duration,
	stakeAmount *big.Int,
	rewardRecipient crypto.Address,
	useACP99 bool,
) (*types.Transaction, *types.Receipt, error) {
	type PChainOwner struct {
		Threshold uint32
		Addresses []crypto.Address
	}

	type ValidatorRegistrationInput struct {
		NodeID                []byte
		BlsPublicKey          []byte
		RegistrationExpiry    uint64
		RemainingBalanceOwner PChainOwner
		DisableOwner          PChainOwner
	}

	balanceOwnersAux := PChainOwner{
		Threshold: balanceOwners.Threshold,
		Addresses: sdkutils.Map(balanceOwners.Addresses, func(addr ids.ShortID) crypto.Address {
			return crypto.BytesToAddress(addr[:])
		}),
	}
	disableOwnersAux := PChainOwner{
		Threshold: disableOwners.Threshold,
		Addresses: sdkutils.Map(disableOwners.Addresses, func(addr ids.ShortID) crypto.Address {
			return crypto.BytesToAddress(addr[:])
		}),
	}

	if useACP99 {
		return contract.TxToMethod(
			rpcURL,
			false,
			crypto.Address{},
			managerOwnerPrivateKey,
			managerAddress,
			stakeAmount,
			"initialize validator registration with stake",
			ErrorSignatureToError,
			"initiateValidatorRegistration(bytes,bytes,(uint32,[address]),(uint32,[address]),uint16,uint64,address)",
			nodeID[:],
			blsPublicKey,
			balanceOwnersAux,
			disableOwnersAux,
			delegationFeeBips,
			uint64(minStakeDuration.Seconds()),
			rewardRecipient,
		)
	}

	return contract.TxToMethod(
		rpcURL,
		false,
		crypto.Address{},
		managerOwnerPrivateKey,
		managerAddress,
		stakeAmount,
		"initialize validator registration with stake",
		ErrorSignatureToError,
		"initializeValidatorRegistration((bytes,bytes,uint64,(uint32,[address]),(uint32,[address])),uint16,uint64)",
		ValidatorRegistrationInput{
			NodeID:                nodeID[:],
			BlsPublicKey:          blsPublicKey,
			RegistrationExpiry:    expiry,
			RemainingBalanceOwner: balanceOwnersAux,
			DisableOwner:          disableOwnersAux,
		},
		delegationFeeBips,
		uint64(minStakeDuration.Seconds()),
	)
}

// step 1 of flow for adding a new validator
func InitializeValidatorRegistrationPoA(
	rpcURL string,
	managerAddress crypto.Address,
	generateRawTxOnly bool,
	managerOwnerAddress crypto.Address,
	managerOwnerPrivateKey string,
	nodeID ids.NodeID,
	blsPublicKey []byte,
	expiry uint64,
	balanceOwners localWarpMessage.PChainOwner,
	disableOwners localWarpMessage.PChainOwner,
	weight uint64,
	useACP99 bool,
) (*types.Transaction, *types.Receipt, error) {
	type PChainOwner struct {
		Threshold uint32
		Addresses []crypto.Address
	}
	balanceOwnersAux := PChainOwner{
		Threshold: balanceOwners.Threshold,
		Addresses: sdkutils.Map(balanceOwners.Addresses, func(addr ids.ShortID) crypto.Address {
			return crypto.BytesToAddress(addr[:])
		}),
	}
	disableOwnersAux := PChainOwner{
		Threshold: disableOwners.Threshold,
		Addresses: sdkutils.Map(disableOwners.Addresses, func(addr ids.ShortID) crypto.Address {
			return crypto.BytesToAddress(addr[:])
		}),
	}
	if useACP99 {
		return contract.TxToMethod(
			rpcURL,
			generateRawTxOnly,
			managerOwnerAddress,
			managerOwnerPrivateKey,
			managerAddress,
			big.NewInt(0),
			"initialize validator registration",
			ErrorSignatureToError,
			"initiateValidatorRegistration(bytes,bytes,(uint32,[address]),(uint32,[address]),uint64)",
			nodeID[:],
			blsPublicKey,
			balanceOwnersAux,
			disableOwnersAux,
			weight,
		)
	}
	type ValidatorRegistrationInput struct {
		NodeID                []byte
		BlsPublicKey          []byte
		RegistrationExpiry    uint64
		RemainingBalanceOwner PChainOwner
		DisableOwner          PChainOwner
	}
	return contract.TxToMethod(
		rpcURL,
		generateRawTxOnly,
		managerOwnerAddress,
		managerOwnerPrivateKey,
		managerAddress,
		big.NewInt(0),
		"initialize validator registration",
		ErrorSignatureToError,
		"initializeValidatorRegistration((bytes,bytes,uint64,(uint32,[address]),(uint32,[address])),uint64)",
		ValidatorRegistrationInput{
			NodeID:                nodeID[:],
			BlsPublicKey:          blsPublicKey,
			RegistrationExpiry:    expiry,
			RemainingBalanceOwner: balanceOwnersAux,
			DisableOwner:          disableOwnersAux,
		},
		weight,
	)
}

func GetRegisterL1ValidatorMessage(
	ctx context.Context,
	rpcURL string,
	network models.Network,
	aggregatorLogger luxlog.Logger,
	aggregatorQuorumPercentage uint64,
	subnetID ids.ID,
	blockchainID ids.ID,
	managerAddress crypto.Address,
	nodeID ids.NodeID,
	blsPublicKey [48]byte,
	expiry uint64,
	balanceOwners localWarpMessage.PChainOwner,
	disableOwners localWarpMessage.PChainOwner,
	weight uint64,
	alreadyInitialized bool,
	initiateTxHash string,
	registerSubnetValidatorUnsignedMessage *warp.UnsignedMessage,
	signatureAggregatorEndpoint string,
) (*warp.Message, ids.ID, error) {
	var (
		validationID ids.ID
		err          error
	)
	if registerSubnetValidatorUnsignedMessage == nil {
		if alreadyInitialized {
			validationID, err = validator.GetValidationID(
				rpcURL,
				managerAddress,
				nodeID,
			)
			if err != nil {
				return nil, ids.Empty, err
			}
			if initiateTxHash != "" {
				registerSubnetValidatorUnsignedMessage, err = GetRegisterL1ValidatorMessageFromTx(
					rpcURL,
					validationID,
					initiateTxHash,
				)
				if err != nil {
					return nil, ids.Empty, err
				}
			} else {
				registerSubnetValidatorUnsignedMessage, err = SearchForRegisterL1ValidatorMessage(
					ctx,
					rpcURL,
					validationID,
				)
				if err != nil {
					return nil, ids.Empty, err
				}
			}
		} else {
			addressedCallPayload, err := warpMessage.NewRegisterL1Validator(
				subnetID,
				nodeID,
				blsPublicKey[:],
				expiry,
				balanceOwners,
				disableOwners,
				weight,
			)
			if err != nil {
				return nil, ids.Empty, err
			}
			validationID = addressedCallPayload.ValidationID()
			registerSubnetValidatorAddressedCall, err := warpPayload.NewAddressedCall(
				managerAddress.Bytes(),
				addressedCallPayload.Bytes(),
			)
			if err != nil {
				return nil, ids.Empty, err
			}
			registerSubnetValidatorUnsignedMessage, err = warp.NewUnsignedMessage(
				network.ID(),
				blockchainID[:],
				registerSubnetValidatorAddressedCall.Bytes(),
			)
			if err != nil {
				return nil, ids.Empty, err
			}
		}
	} else {
		payload := registerSubnetValidatorUnsignedMessage.Payload
		addressedCall, err := warpPayload.ParseAddressedCall(payload)
		if err != nil {
			return nil, ids.Empty, fmt.Errorf("unexpected format on given registration warp message: %w", err)
		}
		reg, err := warpMessage.ParseRegisterL1Validator(addressedCall.Payload)
		if err != nil {
			return nil, ids.Empty, fmt.Errorf("unexpected format on given registration warp message: %w", err)
		}
		validationID = reg.ValidationID()
	}

	messageHexStr := hex.EncodeToString(registerSubnetValidatorUnsignedMessage.Bytes())
	standaloneSignedMessage, err := sdkwarp.SignMessage(aggregatorLogger, signatureAggregatorEndpoint, messageHexStr, "", subnetID.String(), aggregatorQuorumPercentage)
	if err != nil {
		return nil, ids.Empty, fmt.Errorf("failed to get signed message: %w", err)
	}
	signedMessageInterface, err := warpMessage.ConvertStandaloneToNodeWarpMessage(standaloneSignedMessage)
	if err != nil {
		return nil, ids.Empty, fmt.Errorf("failed to convert warp message: %w", err)
	}
	signedMessage := signedMessageInterface.(*warp.Message)
	return signedMessage, validationID, nil
}


func GetPChainL1ValidatorRegistrationMessage(
	ctx context.Context,
	network models.Network,
	rpcURL string,
	aggregatorLogger luxlog.Logger,
	aggregatorQuorumPercentage uint64,
	subnetID ids.ID,
	validationID ids.ID,
	registered bool,
	signatureAggregatorEndpoint string,
) (*warp.Message, error) {
	addressedCallPayload, err := warpMessage.NewL1ValidatorRegistration(validationID, registered)
	if err != nil {
		return nil, err
	}
	subnetValidatorRegistrationAddressedCall, err := warpPayload.NewAddressedCall(
		nil,
		addressedCallPayload.Bytes(),
	)
	if err != nil {
		return nil, err
	}
	subnetConversionUnsignedMessage, err := warp.NewUnsignedMessage(
		network.ID(),
		luxdconstants.PlatformChainID[:],
		subnetValidatorRegistrationAddressedCall.Bytes(),
	)
	if err != nil {
		return nil, err
	}
	var justificationBytes []byte
	if !registered {
		justificationBytes, err = GetRegistrationJustification(ctx, rpcURL, validationID, subnetID)
		if err != nil {
			return nil, err
		}
	}
	justification := hex.EncodeToString(justificationBytes)
	messageHexStr := hex.EncodeToString(subnetConversionUnsignedMessage.Bytes())
	standaloneSignedMessage, err := sdkwarp.SignMessage(aggregatorLogger, signatureAggregatorEndpoint, messageHexStr, justification, subnetID.String(), aggregatorQuorumPercentage)
	if err != nil {
		return nil, err
	}
	signedMessageInterface, err := warpMessage.ConvertStandaloneToNodeWarpMessage(standaloneSignedMessage)
	if err != nil {
		return nil, err
	}
	return signedMessageInterface.(*warp.Message), nil
}

// last step of flow for adding a new validator
func CompleteValidatorRegistration(
	rpcURL string,
	managerAddress crypto.Address,
	generateRawTxOnly bool,
	ownerAddress crypto.Address,
	privateKey string, // not need to be owner atm
	l1ValidatorRegistrationSignedMessage *warp.Message,
) (*types.Transaction, *types.Receipt, error) {
	return contract.TxToMethodWithWarpMessage(
		rpcURL,
		generateRawTxOnly,
		ownerAddress,
		privateKey,
		managerAddress,
		l1ValidatorRegistrationSignedMessage,
		big.NewInt(0),
		"complete validator registration",
		ErrorSignatureToError,
		"completeValidatorRegistration(uint32)",
		uint32(0),
	)
}

func InitValidatorRegistration(
	ctx context.Context,
	app *application.Lux,
	network models.Network,
	rpcURL string,
	chainSpec contract.ChainSpec,
	generateRawTxOnly bool,
	ownerAddressStr string,
	ownerPrivateKey string,
	nodeID ids.NodeID,
	blsPublicKey []byte,
	expiry uint64,
	balanceOwners localWarpMessage.PChainOwner,
	disableOwners localWarpMessage.PChainOwner,
	weight uint64,
	aggregatorLogger luxlog.Logger,
	isPos bool,
	delegationFee uint16,
	stakeDuration time.Duration,
	rewardRecipient crypto.Address,
	validatorManagerAddressStr string,
	useACP99 bool,
	initiateTxHash string,
	signatureAggregatorEndpoint string,
) (*warp.Message, ids.ID, *types.Transaction, error) {
	subnetID, err := contract.GetSubnetID(
		app.GetSDKApp(),
		network,
		chainSpec,
	)
	if err != nil {
		return nil, ids.Empty, nil, err
	}
	blockchainID, err := contract.GetBlockchainID(
		app.GetSDKApp(),
		network,
		chainSpec,
	)
	if err != nil {
		return nil, ids.Empty, nil, err
	}
	managerAddress := crypto.HexToAddress(validatorManagerAddressStr)
	ownerAddress := crypto.HexToAddress(ownerAddressStr)

	alreadyInitialized := initiateTxHash != ""
	if validationID, err := validator.GetValidationID(
		rpcURL,
		managerAddress,
		nodeID,
	); err != nil {
		return nil, ids.Empty, nil, err
	} else if validationID != ids.Empty {
		alreadyInitialized = true
	}

	var receipt *types.Receipt
	if !alreadyInitialized {
		var tx *types.Transaction
		if isPos {
			stakeAmount, err := PoSWeightToValue(
				rpcURL,
				managerAddress,
				weight,
			)
			if err != nil {
				return nil, ids.Empty, nil, fmt.Errorf("failure obtaining value from weight: %w", err)
			}
			ux.Logger.PrintLineSeparator()
			ux.Logger.PrintToUser("Initializing validator registration with PoS validator manager")
			ux.Logger.PrintToUser("Using RPC URL: %s", rpcURL)
			ux.Logger.PrintToUser("NodeID: %s staking %s tokens", nodeID.String(), stakeAmount)
			ux.Logger.PrintLineSeparator()
			tx, receipt, err = InitializeValidatorRegistrationPoSNative(
				rpcURL,
				managerAddress,
				ownerPrivateKey,
				nodeID,
				blsPublicKey,
				expiry,
				balanceOwners,
				disableOwners,
				delegationFee,
				stakeDuration,
				stakeAmount,
				rewardRecipient,
				useACP99,
			)
			if err != nil {
				if !errors.Is(err, ErrNodeAlreadyRegistered) {
					return nil, ids.Empty, nil, evm.TransactionError(tx, err, "failure initializing validator registration")
				}
				ux.Logger.PrintToUser(logging.LightBlue.Wrap("The validator registration was already initialized. Proceeding to the next step"))
				alreadyInitialized = true
			} else {
				ux.Logger.PrintToUser("Validator registration initialized. InitiateTxHash: %s", tx.Hash())
			}
			ux.Logger.PrintToUser(fmt.Sprintf("Validator staked amount: %d", stakeAmount))
		} else {
			managerAddress = crypto.HexToAddress(validatorManagerAddressStr)
			tx, receipt, err = InitializeValidatorRegistrationPoA(
				rpcURL,
				managerAddress,
				generateRawTxOnly,
				ownerAddress,
				ownerPrivateKey,
				nodeID,
				blsPublicKey,
				expiry,
				balanceOwners,
				disableOwners,
				weight,
				useACP99,
			)
			if err != nil {
				if !errors.Is(err, ErrNodeAlreadyRegistered) {
					return nil, ids.Empty, nil, evm.TransactionError(tx, err, "failure initializing validator registration")
				}
				ux.Logger.PrintToUser(logging.LightBlue.Wrap("The validator registration was already initialized. Proceeding to the next step"))
				alreadyInitialized = true
			} else if generateRawTxOnly {
				return nil, ids.Empty, tx, nil
			}
			ux.Logger.PrintToUser(fmt.Sprintf("Validator weight: %d", weight))
		}
	} else {
		ux.Logger.PrintToUser(logging.LightBlue.Wrap("The validator registration was already initialized. Proceeding to the next step"))
	}

	var unsignedMessage *warp.UnsignedMessage
	if receipt != nil {
		unsignedMessage, err = evm.ExtractWarpMessageFromReceipt(receipt)
		if err != nil {
			return nil, ids.Empty, nil, err
		}
	}

	signedMessage, validationID, err := GetRegisterL1ValidatorMessage(
		ctx,
		rpcURL,
		network,
		aggregatorLogger,
		0,
		subnetID,
		blockchainID,
		managerAddress,
		nodeID,
		[48]byte(blsPublicKey),
		expiry,
		balanceOwners,
		disableOwners,
		weight,
		alreadyInitialized,
		initiateTxHash,
		unsignedMessage,
		signatureAggregatorEndpoint,
	)

	return signedMessage, validationID, nil, err
}

func FinishValidatorRegistration(
	ctx context.Context,
	app *application.Lux,
	network models.Network,
	rpcURL string,
	chainSpec contract.ChainSpec,
	generateRawTxOnly bool,
	ownerAddressStr string,
	privateKey string,
	validationID ids.ID,
	aggregatorLogger luxlog.Logger,
	validatorManagerAddressStr string,
	signatureAggregatorEndpoint string,
) (*types.Transaction, error) {
	subnetID, err := contract.GetSubnetID(
		app.GetSDKApp(),
		network,
		chainSpec,
	)
	if err != nil {
		return nil, err
	}
	managerAddress := crypto.HexToAddress(validatorManagerAddressStr)
	signedMessage, err := GetPChainL1ValidatorRegistrationMessage(
		ctx,
		network,
		rpcURL,
		aggregatorLogger,
		0,
		subnetID,
		validationID,
		true,
		signatureAggregatorEndpoint,
	)
	if err != nil {
		return nil, err
	}
	if privateKey != "" {
		if client, err := evm.GetClient(rpcURL); err != nil {
			ux.Logger.RedXToUser("failure connecting to L1 to setup proposer VM: %w", err)
		} else {
			if err := client.SetupProposerVM(privateKey); err != nil {
				ux.Logger.RedXToUser("failure setting proposer VM on L1: %w", err)
			}
			client.Close()
		}
	}
	ownerAddress := crypto.HexToAddress(ownerAddressStr)
	tx, _, err := CompleteValidatorRegistration(
		rpcURL,
		managerAddress,
		generateRawTxOnly,
		ownerAddress,
		privateKey,
		signedMessage,
	)
	if err != nil {
		if !errors.Is(err, ErrInvalidValidationID) {
			return nil, evm.TransactionError(tx, err, "failure completing validator registration")
		} else {
			return nil, fmt.Errorf("the Validator was already fully registered on the Manager")
		}
	}
	if generateRawTxOnly {
		return tx, nil
	}
	return nil, nil
}

func SearchForRegisterL1ValidatorMessage(
	ctx context.Context,
	rpcURL string,
	validationID ids.ID,
) (*warp.UnsignedMessage, error) {
	client, err := evm.GetClient(rpcURL)
	if err != nil {
		return nil, err
	}
	height, err := client.BlockNumber()
	if err != nil {
		return nil, err
	}
	maxBlock := int64(height)
	minBlock := int64(0)
	blockStep := int64(5000)
	for blockNumber := maxBlock; blockNumber >= minBlock; blockNumber -= blockStep {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		fromBlock := big.NewInt(blockNumber - blockStep)
		if fromBlock.Sign() < 0 {
			fromBlock = big.NewInt(0)
		}
		toBlock := big.NewInt(blockNumber)
		logs, err := client.FilterLogs(ethereum.FilterQuery{
			FromBlock: fromBlock,
			ToBlock:   toBlock,
			Addresses: []common.Address{subnetEvmWarp.Module.Address},
		})
		if err != nil {
			return nil, err
		}
		msgs := evm.GetWarpMessagesFromLogs(utils.PointersSlice(logs))
		for _, msg := range msgs {
			payload := msg.Payload
			addressedCall, err := warpPayload.ParseAddressedCall(payload)
			if err == nil {
				reg, err := warpMessage.ParseRegisterL1Validator(addressedCall.Payload)
				if err == nil {
					if reg.ValidationID() == validationID {
						return msg, nil
					}
				}
			}
		}
	}
	return nil, fmt.Errorf("validation id %s not found on warp events", validationID)
}

func GetRegistrationJustification(
	ctx context.Context,
	rpcURL string,
	validationID ids.ID,
	subnetID ids.ID,
) ([]byte, error) {
	const numBootstrapValidatorsToSearch = 100
	for validationIndex := uint32(0); validationIndex < numBootstrapValidatorsToSearch; validationIndex++ {
		bootstrapValidationID := subnetID.Append(validationIndex)
		if bootstrapValidationID == validationID {
			justification := platformvm.L1ValidatorRegistrationJustification{
				Preimage: &platformvm.L1ValidatorRegistrationJustification_ConvertSubnetToL1TxData{
					ConvertSubnetToL1TxData: &platformvm.SubnetIDIndex{
						SubnetId: subnetID[:],
						Index:    validationIndex,
					},
				},
			}
			// Use JSON marshaling as a workaround since we don't have real protobuf
			justBytes, _ := json.Marshal(&justification)
			return justBytes, nil
		}
	}
	msg, err := SearchForRegisterL1ValidatorMessage(
		ctx,
		rpcURL,
		validationID,
	)
	if err != nil {
		return nil, err
	}
	payload := msg.Payload
	addressedCall, err := warpPayload.ParseAddressedCall(payload)
	if err != nil {
		return nil, err
	}
	justification := platformvm.L1ValidatorRegistrationJustification{
		Preimage: &platformvm.L1ValidatorRegistrationJustification_RegisterL1ValidatorMessage{
			RegisterL1ValidatorMessage: addressedCall.Payload,
		},
	}
	// Use JSON marshaling as a workaround since we don't have real protobuf
	justBytes, _ := json.Marshal(&justification)
	return justBytes, nil
}

func GetRegisterL1ValidatorMessageFromTx(
	rpcURL string,
	validationID ids.ID,
	txHash string,
) (*warp.UnsignedMessage, error) {
	client, err := evm.GetClient(rpcURL)
	if err != nil {
		return nil, err
	}
	receipt, err := client.TransactionReceipt(common.HexToHash(txHash))
	if err != nil {
		return nil, err
	}
	msgs := evm.GetWarpMessagesFromLogs(receipt.Logs)
	for _, msg := range msgs {
		payload := msg.Payload
		addressedCall, err := warpPayload.ParseAddressedCall(payload)
		if err == nil {
			reg, err := warpMessage.ParseRegisterL1Validator(addressedCall.Payload)
			if err == nil {
				if reg.ValidationID() == validationID {
					return msg, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("register validator message not found on tx %s", txHash)
}
