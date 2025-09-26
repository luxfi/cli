// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package blockchaincmd

import (
	"fmt"

	"github.com/luxfi/cli/cmd/flags"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/blockchain"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/sdk/prompts"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/sdk/validatormanager/validatormanagertypes"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/ids"
	"github.com/luxfi/node/staking"
	"github.com/luxfi/node/utils/formatting"
	"github.com/luxfi/node/vms/platformvm/signer"
)

// captureInt wraps SDK's CapturePositiveInt to provide the validator function interface
func captureInt(prompt string, validator func(int) error) (int, error) {
	// Use CapturePositiveInt without comparators and validate afterwards
	result, err := app.Prompt.CapturePositiveInt(prompt, nil)
	if err != nil {
		return 0, err
	}
	
	// Apply the validator if provided
	if validator != nil {
		if err := validator(result); err != nil {
			return 0, err
		}
	}
	
	return result, nil
}

func getValidatorContractManagerAddr() (string, error) {
	return prompts.PromptAddress(
		app.Prompt,
		"enable as controller of ValidatorManager contract (C-Chain address)",
	)
}

func promptProofOfPossession(promptPublicKey, promptPop bool) (string, string, error) {
	if promptPublicKey || promptPop {
		ux.Logger.PrintToUser("Next, we need the public key and proof of possession of the node's BLS")
		ux.Logger.PrintToUser("Check https://docs.lux.network/api-reference/info-api#infogetnodeid for instructions on calling info.getNodeID API")
	}
	var err error
	publicKey := ""
	proofOfPossesion := ""
	if promptPublicKey {
		txt := "What is the node's BLS public key?"
		// Capture and validate BLS public key
		publicKey, err = app.Prompt.CaptureString(txt)
		if err != nil {
			return "", "", err
		}
	}
	if promptPop {
		txt := "What is the node's BLS proof of possession?"
		// Capture and validate BLS proof of possession
		proofOfPossesion, err = app.Prompt.CaptureString(txt)
		if err != nil {
			return "", "", err
		}
	}
	return publicKey, proofOfPossesion, nil
}

// promptValidatorManagementType allows the user to select between different validator management types
// with an option to explain the differences between them
func promptValidatorManagementType(
	app *application.Lux,
	sidecar *models.Sidecar,
) error {
	explainOption := "Explain the difference"
	if createFlags.proofOfStake {
		sidecar.ValidatorManagement = validatormanagertypes.ProofOfStake
		return nil
	}
	if createFlags.proofOfAuthority {
		sidecar.ValidatorManagement = validatormanagertypes.ProofOfAuthority
		return nil
	}

	options := []string{validatormanagertypes.ProofOfAuthority, validatormanagertypes.ProofOfStake, explainOption}
	for {
		option, err := app.Prompt.CaptureList(
			"Which validator management type would you like to use in your blockchain?",
			options,
		)
		if err != nil {
			return err
		}
		switch option {
		case validatormanagertypes.ProofOfAuthority:
			sidecar.ValidatorManagement = string(validatormanagertypes.ValidatorManagementTypeFromString(option))
		case validatormanagertypes.ProofOfStake:
			sidecar.ValidatorManagement = string(validatormanagertypes.ValidatorManagementTypeFromString(option))
		case explainOption:
			continue
		}
		break
	}
	return nil
}

// generateNewNodeAndBLS returns node id, bls public key and bls pop
func generateNewNodeAndBLS() (string, string, string, error) {
	certBytes, _, err := staking.NewCertAndKeyBytes()
	if err != nil {
		return "", "", "", err
	}
	nodeID, err := utils.ToNodeID(certBytes)
	if err != nil {
		return "", "", "", err
	}
	// Generate a new BLS secret key for proof of possession
	blsSecretKey, err := bls.NewSecretKey()
	if err != nil {
		return "", "", "", err
	}
	p, err := signer.NewProofOfPossession(blsSecretKey)
	if err != nil {
		return "", "", "", err
	}
	publicKey, err := formatting.Encode(formatting.HexNC, p.PublicKey[:])
	if err != nil {
		return "", "", "", err
	}
	pop, err := formatting.Encode(formatting.HexNC, p.ProofOfPossession[:])
	if err != nil {
		return "", "", "", err
	}
	return nodeID.String(), publicKey, pop, nil
}

func promptBootstrapValidators(
	network models.Network,
	validatorBalance uint64,
	availableBalance uint64,
	bootstrapValidatorFlags *flags.BootstrapValidatorFlags,
) ([]models.SubnetValidator, error) {
	var subnetValidators []models.SubnetValidator
	var err error
	if bootstrapValidatorFlags.NumBootstrapValidators == 0 {
		maxNumValidators := availableBalance / validatorBalance
		bootstrapValidatorFlags.NumBootstrapValidators, err = captureInt(
			"How many bootstrap validators do you want to set up?",
			func(n int) error {
				if err := prompts.ValidatePositiveInt(n); err != nil {
					return err
				}
				if n > int(maxNumValidators) {
					return fmt.Errorf(
						"given available balance %d, the maximum number of validators with balance %d is %d",
						availableBalance,
						validatorBalance,
						maxNumValidators,
					)
				}
				return nil
			},
		)
	}
	if err != nil {
		return nil, err
	}
	var setUpNodes bool
	if bootstrapValidatorFlags.GenerateNodeID {
		setUpNodes = false
	} else {
		setUpNodes, err = promptSetUpNodes()
		if err != nil {
			return nil, err
		}
		bootstrapValidatorFlags.GenerateNodeID = !setUpNodes
	}
	if bootstrapValidatorFlags.ChangeOwnerAddress == "" {
		bootstrapValidatorFlags.ChangeOwnerAddress, err = blockchain.GetKeyForChangeOwner(app, network)
		if err != nil {
			return nil, err
		}
	}
	for len(subnetValidators) < bootstrapValidatorFlags.NumBootstrapValidators {
		ux.Logger.PrintToUser("Getting info for bootstrap validator %d", len(subnetValidators)+1)
		var nodeID ids.NodeID
		var publicKey, pop string
		if setUpNodes {
			nodeID, err = PromptNodeID("add as bootstrap validator")
			if err != nil {
				return nil, err
			}
			publicKey, pop, err = promptProofOfPossession(true, true)
			if err != nil {
				return nil, err
			}
		} else {
			nodeIDStr, publicKey, pop, err = generateNewNodeAndBLS()
			if err != nil {
				return nil, err
			}
			nodeID, err = ids.NodeIDFromString(nodeIDStr)
			if err != nil {
				return nil, err
			}
		}
		subnetValidator := models.SubnetValidator{
			NodeID:               nodeID.String(),
			Weight:               constants.BootstrapValidatorWeight,
			Balance:              validatorBalance,
			BLSPublicKey:         publicKey,
			BLSProofOfPossession: pop,
			ChangeOwnerAddr:      bootstrapValidatorFlags.ChangeOwnerAddress,
		}
		subnetValidators = append(subnetValidators, subnetValidator)
		ux.Logger.GreenCheckmarkToUser("Bootstrap Validator %d:", len(subnetValidators))
		ux.Logger.PrintToUser("- Node ID: %s", nodeID)
		ux.Logger.PrintToUser("- Change Address: %s", bootstrapValidatorFlags.ChangeOwnerAddress)
	}
	return subnetValidators, nil
}

func validateBLS(publicKey, pop string) error {
	if err := prompts.ValidateHexa(publicKey); err != nil {
		return fmt.Errorf("format error in given public key: %w", err)
	}
	if err := prompts.ValidateHexa(pop); err != nil {
		return fmt.Errorf("format error in given proof of possession: %w", err)
	}
	return nil
}

func validateSubnetValidatorsJSON(generateNewNodeID bool, validatorJSONS []models.SubnetValidator) error {
	for _, validatorJSON := range validatorJSONS {
		if !generateNewNodeID {
			if validatorJSON.NodeID == "" || validatorJSON.BLSPublicKey == "" || validatorJSON.BLSProofOfPossession == "" {
				return fmt.Errorf("no Node ID or BLS info provided, use --generate-node-id flag to generate new Node ID and BLS info")
			}
			_, err := ids.NodeIDFromString(validatorJSON.NodeID)
			if err != nil {
				return fmt.Errorf("invalid node id %s", validatorJSON.NodeID)
			}
			if err = validateBLS(validatorJSON.BLSPublicKey, validatorJSON.BLSProofOfPossession); err != nil {
				return err
			}
		}
		if validatorJSON.Weight == 0 {
			return fmt.Errorf("bootstrap validator weight has to be greater than 0")
		}
		if validatorJSON.Balance == 0 {
			return fmt.Errorf("bootstrap validator balance has to be greater than 0")
		}
	}
	return nil
}

// promptProvideNodeID returns false if user doesn't have any Lux node set up yet to be
// bootstrap validators
func promptSetUpNodes() (bool, error) {
	ux.Logger.PrintToUser("If you have set up your own Lux Nodes, you can provide the Node ID and BLS Key from those nodes in the next step.")
	ux.Logger.PrintToUser("Otherwise, we will generate new Node IDs and BLS Key for you.")
	setUpNodes, err := app.Prompt.CaptureYesNo("Have you set up your own Lux Nodes?")
	if err != nil {
		return false, err
	}
	return setUpNodes, nil
}
