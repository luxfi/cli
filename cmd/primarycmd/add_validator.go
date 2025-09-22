// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package primarycmd

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/luxfi/cli/cmd/blockchaincmd"
	"github.com/luxfi/cli/cmd/nodecmd"
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/keychain"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/cli/pkg/networkoptions"
	cliprompts "github.com/luxfi/cli/pkg/prompts"
	sdkprompts "github.com/luxfi/sdk/prompts"
	"github.com/luxfi/cli/pkg/subnet"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/ids"
	"github.com/spf13/cobra"
)

var (
	globalNetworkFlags           networkoptions.NetworkFlags
	keyName                      string
	useLedger                    bool
	ledgerAddresses              []string
	nodeIDStr                    string
	weight                       uint64
	delegationFee                uint32
	startTimeStr                 string
	duration                     time.Duration
	publicKey                    string
	pop                          string
	ErrMutuallyExlusiveKeyLedger = errors.New("--key and --ledger,--ledger-addrs are mutually exclusive")
	ErrStoredKeyOnMainnet        = errors.New("--key is not available for mainnet operations")
)

type jsonProofOfPossession struct {
	PublicKey         string `json:"publicKey"`
	ProofOfPossession string `json:"proofOfPossession"`
}

// lux primary addValidator
func newAddValidatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "addValidator",
		Short: "Add a validator to Primary Network",
		Long: `The primary addValidator command adds a node as a validator 
in the Primary Network`,
		RunE: addValidator,
		Args: cobrautils.ExactArgs(0),
	}
	networkoptions.AddNetworkFlagsToCmd(cmd, &globalNetworkFlags, false, networkoptions.NonLocalSupportedNetworkOptions)
	cmd.Flags().StringVarP(&keyName, "key", "k", "", "select the key to use [testnet only]")
	cmd.Flags().StringVar(&nodeIDStr, "nodeID", "", "set the NodeID of the validator to add")
	cmd.Flags().Uint64Var(&weight, "weight", 0, "set the staking weight of the validator to add")
	cmd.Flags().StringVar(&startTimeStr, "start-time", "", "UTC start time when this validator starts validating, in 'YYYY-MM-DD HH:MM:SS' format")
	cmd.Flags().DurationVar(&duration, "staking-period", 0, "how long this validator will be staking")
	cmd.Flags().BoolVarP(&useLedger, "ledger", "g", false, "use ledger instead of key (always true on mainnet, defaults to false on testnet)")
	cmd.Flags().StringSliceVar(&ledgerAddresses, "ledger-addrs", []string{}, "use the given ledger addresses")
	cmd.Flags().StringVar(&publicKey, "public-key", "", "set the BLS public key of the validator to add")
	cmd.Flags().StringVar(&pop, "proof-of-possession", "", "set the BLS proof of possession of the validator to add")
	cmd.Flags().Uint32Var(&delegationFee, "delegation-fee", 0, "set the delegation fee (20 000 is equivalent to 2%)")
	return cmd
}

func promptProofOfPossession() (jsonProofOfPossession, error) {
	if publicKey != "" {
		err := cliprompts.ValidateHexa(publicKey)
		if err != nil {
			ux.Logger.PrintToUser("Format error in given public key: %s", err)
			publicKey = ""
		}
	}
	if pop != "" {
		err := cliprompts.ValidateHexa(pop)
		if err != nil {
			ux.Logger.PrintToUser("Format error in given proof of possession: %s", err)
			pop = ""
		}
	}
	if publicKey == "" || pop == "" {
		ux.Logger.PrintToUser("Next, we need the public key and proof of possession of the node's BLS")
		ux.Logger.PrintToUser("SSH into the node and call info.getNodeID API to get the node's BLS info")
		ux.Logger.PrintToUser("Check https://docs.lux.network/api-reference/info-api#infogetnodeid for instructions on calling info.getNodeID API")
	}
	var err error
	if publicKey == "" {
		txt := "What is the public key of the node's BLS?"
		// Create a CLI prompter to use CaptureValidatedString
		cliPrompter := cliprompts.NewPrompter()
		publicKey, err = cliPrompter.CaptureValidatedString(txt, cliprompts.ValidateHexa)
		if err != nil {
			return jsonProofOfPossession{}, err
		}
	}
	if pop == "" {
		txt := "What is the proof of possession of the node's BLS?"
		// Create a CLI prompter to use CaptureValidatedString
		cliPrompter := cliprompts.NewPrompter()
		pop, err = cliPrompter.CaptureValidatedString(txt, cliprompts.ValidateHexa)
		if err != nil {
			return jsonProofOfPossession{}, err
		}
	}
	return jsonProofOfPossession{PublicKey: publicKey, ProofOfPossession: pop}, nil
}

func addValidator(_ *cobra.Command, _ []string) error {
	var (
		nodeID ids.NodeID
		start  time.Time
		err    error
	)

	network, err := networkoptions.GetNetworkFromCmdLineFlags(
		app,
		"",
		globalNetworkFlags,
		true,
		false,
		networkoptions.NonLocalSupportedNetworkOptions,
		"",
	)
	if err != nil {
		return err
	}

	if len(ledgerAddresses) > 0 {
		useLedger = true
	}

	if useLedger && keyName != "" {
		return ErrMutuallyExlusiveKeyLedger
	}

	switch network.Kind() {
	case models.Testnet:
		if !useLedger && keyName == "" {
			useLedger, keyName, err = sdkprompts.GetKeyOrLedger(app.Prompt, constants.PayTxsFeesMsg, app.GetKeyDir(), false)
			if err != nil {
				return err
			}
		}
	case models.Mainnet:
		useLedger = true
		if keyName != "" {
			return ErrStoredKeyOnMainnet
		}
	default:
		return errors.New("unsupported network")
	}

	if nodeIDStr == "" {
		nodeID, err = blockchaincmd.PromptNodeID("add as Primary Network Validator")
		if err != nil {
			return err
		}
	} else {
		nodeID, err = ids.NodeIDFromString(nodeIDStr)
		if err != nil {
			return err
		}
	}

	minValStake, err := nodecmd.GetMinStakingAmount(network)
	if err != nil {
		return err
	}
	if weight == 0 {
		weight, err = nodecmd.PromptWeightPrimaryNetwork(network)
		if err != nil {
			return err
		}
	}
	if weight < minValStake {
		return fmt.Errorf("illegal weight, must be greater than or equal to %d: %d", minValStake, weight)
	}

	// Estimate fee based on network type and transaction complexity
	fee := estimateAddValidatorFee(network)
	kc, err := keychain.GetKeychain(app, false, useLedger, ledgerAddresses, keyName, network, fee)
	if err != nil {
		return err
	}

	network.HandlePublicNetworkSimulation()

	// For primary network validators, we don't need proof of possession for now
	// but keeping the prompt for future compatibility
	_, err = promptProofOfPossession()
	if err != nil {
		return err
	}

	start, duration, err = nodecmd.GetTimeParametersPrimaryNetwork(network, 0, duration, startTimeStr, false)
	if err != nil {
		return err
	}
	deployer := subnet.NewPublicDeployer(app, useLedger, kc.Keychain, network)
	nodecmd.PrintNodeJoinPrimaryNetworkOutput(nodeID, weight, network, start)
	if delegationFee == 0 {
		delegationFee, err = getDelegationFeeOption(app, network)
		if err != nil {
			return err
		}
	} else {
		defaultFee := network.GenesisParams().MinDelegationFee
		if delegationFee < defaultFee {
			return fmt.Errorf("delegation fee has to be larger than %d", defaultFee)
		}
	}
	// For primary network, use AddValidator with empty subnet ID
	// AddValidator returns (bool, *txs.Tx, []string, error)
	// The popBytes and recipientAddr are used for PoS validators, but primary network uses the simpler model
	_, _, _, err = deployer.AddValidator(nil, nil, ids.Empty, nodeID, weight, start, duration)
	return err
}

func getDelegationFeeOption(app *application.Lux, network models.Network) (uint32, error) {
	ux.Logger.PrintToUser("What would you like to set the delegation fee to?")
	defaultFee := network.GenesisParams().MinDelegationFee
	defaultOption := fmt.Sprintf("Default Delegation Fee (%d%%)", defaultFee/10000)
	delegationFeePrompt := "Delegation Fee"
	feeOption, err := app.Prompt.CaptureList(
		delegationFeePrompt,
		[]string{defaultOption, "Custom"},
	)
	if err != nil {
		return 0, err
	}
	if feeOption != defaultOption {
		ux.Logger.PrintToUser("Note that 20 000 is equivalent to 2%%")
		delegationFee, err := app.Prompt.CapturePositiveInt(
			delegationFeePrompt,
			[]sdkprompts.Comparator{
				{
					Label: "Min Delegation Fee",
					Type:  sdkprompts.MoreThanEq,
					Value: uint64(defaultFee),
				},
			},
		)
		if err != nil {
			return 0, err
		}
		if delegationFee > 0 && delegationFee <= math.MaxUint32 {
			return uint32(delegationFee), nil
		}
		return 0, fmt.Errorf("invalid delegation fee")
	}
	return defaultFee, nil
}

func estimateAddValidatorFee(network models.Network) uint64 {
	const baseFee = 1_000_000 // 0.001 LUX base fee
	switch network.Kind() {
	case models.Mainnet:
		return baseFee * 2 // Higher fee for mainnet
	case models.Testnet:
		return baseFee
	case models.Local:
		return 0 // No fee for local networks
	default:
		return baseFee
	}
}
