// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package contractcmd

import (
	"fmt"
	"math/big"

	"github.com/luxfi/cli/cmd/flags"
	"github.com/luxfi/cli/cmd/networkcmd"
	"github.com/luxfi/cli/pkg/blockchain"
	"github.com/luxfi/cli/pkg/chainvalidators"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/networkoptions"
	"github.com/luxfi/cli/pkg/signatureaggregator"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/constantsants"
	"github.com/luxfi/ids"
	luxlog "github.com/luxfi/log"
	blockchainSDK "github.com/luxfi/sdk/blockchain"
	"github.com/luxfi/sdk/contract"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/sdk/prompts"
	validatormanagerSDK "github.com/luxfi/sdk/validatormanager"

	"github.com/luxfi/geth/common"
	"github.com/spf13/cobra"
)

type POSManagerSpecFlags struct {
	rewardCalculatorAddress string
	minimumStakeAmount      uint64 // big.Int
	maximumStakeAmount      uint64 // big.Int
	minimumStakeDuration    uint64
	minimumDelegationFee    uint16
	maximumStakeMultiplier  uint8
	weightToValueFactor     uint64 // big.Int
}

var (
	initPOSManagerFlags       POSManagerSpecFlags
	network                   networkoptions.NetworkFlags
	privateKeyFlags           contract.PrivateKeyFlags
	initValidatorManagerFlags ContractInitValidatorManagerFlags
)

type ContractInitValidatorManagerFlags struct {
	RPC         string
	SigAggFlags flags.SignatureAggregatorFlags
}

// lux contract initValidatorManager
func newInitValidatorManagerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "initValidatorManager blockchainName",
		Short:   "Initializes Proof of Authority(PoA) or Proof of Stake(PoS) Validator Manager on a given Network and Blockchain",
		Long:    "Initializes Proof of Authority(PoA) or Proof of Stake(PoS)Validator Manager contract on a Blockchain and sets up initial validator set on the Blockchain. For more info on Validator Manager, please head to https://github.com/luxfi/warp-contracts/tree/main/contracts/validator-manager",
		RunE:    initValidatorManager,
		PreRunE: cobrautils.ExactArgs(1),
	}
	// Network flags handled globally to avoid conflicts
	privateKeyFlags.AddToCmd(cmd, "as contract deployer")
	flags.AddRPCFlagToCmd(cmd, app, &initValidatorManagerFlags.RPC)
	sigAggGroup := flags.AddSignatureAggregatorFlagsToCmd(cmd, &initValidatorManagerFlags.SigAggFlags)

	cmd.Flags().StringVar(&initPOSManagerFlags.rewardCalculatorAddress, "pos-reward-calculator-address", "", "(PoS only) initialize the ValidatorManager with reward calculator address")
	cmd.Flags().Uint64Var(&initPOSManagerFlags.minimumStakeAmount, "pos-minimum-stake-amount", 1, "(PoS only) minimum stake amount")
	cmd.Flags().Uint64Var(&initPOSManagerFlags.maximumStakeAmount, "pos-maximum-stake-amount", 1000, "(PoS only) maximum stake amount")
	cmd.Flags().Uint64Var(&initPOSManagerFlags.minimumStakeDuration, "pos-minimum-stake-duration", constants.PoSL1MinimumStakeDurationSeconds, "(PoS only) minimum stake duration (in seconds)")
	cmd.Flags().Uint16Var(&initPOSManagerFlags.minimumDelegationFee, "pos-minimum-delegation-fee", 1, "(PoS only) minimum delegation fee")
	cmd.Flags().Uint8Var(&initPOSManagerFlags.maximumStakeMultiplier, "pos-maximum-stake-multiplier", 1, "(PoS only )maximum stake multiplier")
	cmd.Flags().Uint64Var(&initPOSManagerFlags.weightToValueFactor, "pos-weight-to-value-factor", 1, "(PoS only) weight to value factor")
	cmd.SetHelpFunc(flags.WithGroupedHelp([]flags.GroupedFlags{sigAggGroup}))
	return cmd
}

func initValidatorManager(_ *cobra.Command, args []string) error {
	blockchainName := args[0]
	chainSpec := contract.ChainSpec{
		BlockchainName: blockchainName,
	}
	network, err := networkoptions.GetNetworkFromCmdLineFlags(
		app,
		"",
		network,
		true,
		false,
		networkoptions.DefaultSupportedNetworkOptions,
		"",
	)
	if err != nil {
		return err
	}
	// Convert local cluster networks to appropriate type
	network = models.ConvertClusterToNetwork(network)
	if initValidatorManagerFlags.RPC == "" {
		initValidatorManagerFlags.RPC, _, err = contract.GetBlockchainEndpoints(
			app.GetSDKApp(),
			network,
			chainSpec,
			true,
			false,
		)
		if err != nil {
			return err
		}
	}
	ux.Logger.PrintToUser(luxlog.Yellow.Wrap("RPC Endpoint: %s"), initValidatorManagerFlags.RPC)
	_, genesisPrivateKey, err := contract.GetEVMSubnetPrefundedKey(
		app.GetSDKApp(),
		network,
		chainSpec,
	)
	if err != nil {
		return err
	}
	privateKey, err := privateKeyFlags.GetPrivateKey(app.GetSDKApp(), genesisPrivateKey)
	if err != nil {
		return err
	}
	if privateKey == "" {
		privateKey, err = prompts.PromptPrivateKey(
			app.Prompt,
			"pay for initializing Proof of Authority Validator Manager contract? (Uses Blockchain gas token)",
		)
		if err != nil {
			return err
		}
	}
	sc, err := app.LoadSidecar(chainSpec.BlockchainName)
	if err != nil {
		return fmt.Errorf("failed to load sidecar: %w", err)
	}
	if sc.Networks[network.Name()].ValidatorManagerAddress == "" {
		return fmt.Errorf("unable to find Validator Manager address")
	}
	managerAddress := sc.Networks[network.Name()].ValidatorManagerAddress
	scNetwork := sc.Networks[network.Name()]
	if scNetwork.BlockchainID == ids.Empty {
		return fmt.Errorf("blockchain has not been deployed to %s", network.Name())
	}
	// Get bootstrap validators from the blockchain configuration
	// Note: Using empty validator list as NetworkData doesn't have validators
	var bootstrapValidators []models.SubnetValidator
	luxdBootstrapValidators, err := chainvalidators.ToL1Validators(bootstrapValidators)
	if err != nil {
		return err
	}
	// Use network name as cluster identifier
	clusterName := network.Name()
	extraAggregatorPeers, err := blockchain.GetAggregatorExtraPeers(app, clusterName)
	if err != nil {
		return err
	}
	aggregatorLogger, err := signatureaggregator.NewSignatureAggregatorLogger(
		initValidatorManagerFlags.SigAggFlags.AggregatorLogLevel,
		initValidatorManagerFlags.SigAggFlags.AggregatorLogToStdout,
		app.GetAggregatorLogDir(clusterName),
	)
	if err != nil {
		return err
	}
	subnetID, err := contract.GetSubnetID(
		app.GetSDKApp(),
		network,
		chainSpec,
	)
	if err != nil {
		return err
	}
	blockchainID, err := contract.GetBlockchainID(
		app.GetSDKApp(),
		network,
		chainSpec,
	)
	if err != nil {
		return err
	}
	ownerAddress := common.HexToAddress(sc.ProxyContractOwner)
	// Convert validators to []interface{}
	validators := make([]interface{}, len(luxdBootstrapValidators))
	for i, v := range luxdBootstrapValidators {
		validators[i] = v
	}
	netSDK := blockchainSDK.Net{
		NetID:               subnetID,
		BlockchainID:        blockchainID,
		BootstrapValidators: validators,
		OwnerAddress:        &ownerAddress,
		RPC:                 initValidatorManagerFlags.RPC,
	}
	// Convert extraAggregatorPeers to []interface{}
	extraPeers := make([]interface{}, len(extraAggregatorPeers))
	for i, p := range extraAggregatorPeers {
		extraPeers[i] = p
	}
	err = signatureaggregator.CreateSignatureAggregatorInstance(app, subnetID.String(), network, extraPeers, aggregatorLogger, "latest")
	if err != nil {
		return err
	}
	signatureAggregatorEndpoint, err := signatureaggregator.GetSignatureAggregatorEndpoint(app, network)
	if err != nil {
		return err
	}
	switch {
	case sc.ValidatorManagement == "proof-of-authority": // PoA
		ux.Logger.PrintToUser(luxlog.Yellow.Wrap("Initializing Proof of Authority Validator Manager contract on blockchain %s"), blockchainName)
		if err := validatormanagerSDK.SetupPoA(
			aggregatorLogger, // Use aggregatorLogger instead of app.Log
			netSDK,
			network,
			privateKey,
			aggregatorLogger,
			managerAddress,
			sc.UseACP99,
			signatureAggregatorEndpoint,
		); err != nil {
			return err
		}
		ux.Logger.GreenCheckmarkToUser("Proof of Authority Validator Manager contract successfully initialized on blockchain %s", blockchainName)
	case sc.PoS: // PoS
		deployed, err := validatormanagerSDK.ValidatorProxyHasImplementationSet(initValidatorManagerFlags.RPC)
		if err != nil {
			return err
		}
		if !deployed {
			// it is not in genesis
			ux.Logger.PrintToUser("Deploying Proof of Stake Validator Manager contract on blockchain %s ...", blockchainName)
			proxyOwnerPrivateKey, err := networkcmd.GetProxyOwnerPrivateKey(
				app,
				network,
				sc.ProxyContractOwner,
				ux.Logger.PrintToUser,
			)
			if err != nil {
				return err
			}
			if sc.UseACP99 {
				_, err := validatormanagerSDK.DeployAndRegisterValidatorManagerV2_0_0Contract(
					initValidatorManagerFlags.RPC,
					genesisPrivateKey,
					proxyOwnerPrivateKey,
				)
				if err != nil {
					return err
				}
				_, err = validatormanagerSDK.DeployAndRegisterPoSValidatorManagerV2_0_0Contract(
					initValidatorManagerFlags.RPC,
					genesisPrivateKey,
					proxyOwnerPrivateKey,
				)
				if err != nil {
					return err
				}
			} else {
				if _, err := validatormanagerSDK.DeployAndRegisterPoSValidatorManagerV1_0_0Contract(
					initValidatorManagerFlags.RPC,
					genesisPrivateKey,
					proxyOwnerPrivateKey,
				); err != nil {
					return err
				}
			}
		}
		ux.Logger.PrintToUser(luxlog.Yellow.Wrap("Initializing Proof of Stake Validator Manager contract on blockchain %s"), blockchainName)
		if initPOSManagerFlags.rewardCalculatorAddress == "" {
			initPOSManagerFlags.rewardCalculatorAddress = validatormanagerSDK.RewardCalculatorAddress
		}
		found, _, _, managerOwnerPrivateKey, err := contract.SearchForManagedKey(
			app.GetSDKApp(),
			network,
			ownerAddress.Hex(),
			true,
		)
		if err != nil {
			return err
		}
		if !found {
			return fmt.Errorf("could not find validator manager owner private key")
		}
		if err := validatormanagerSDK.SetupPoS(
			aggregatorLogger, // Use aggregatorLogger instead of app.Log
			netSDK,
			network,
			privateKey,
			aggregatorLogger,
			validatormanagerSDK.PoSParams{
				MinimumStakeAmount:      big.NewInt(int64(initPOSManagerFlags.minimumStakeAmount)), //nolint:gosec // G115: Stake amounts are bounded
				MaximumStakeAmount:      big.NewInt(int64(initPOSManagerFlags.maximumStakeAmount)), //nolint:gosec // G115: Stake amounts are bounded
				MinimumStakeDuration:    initPOSManagerFlags.minimumStakeDuration,
				MinimumDelegationFee:    initPOSManagerFlags.minimumDelegationFee,
				MaximumStakeMultiplier:  initPOSManagerFlags.maximumStakeMultiplier,
				WeightToValueFactor:     big.NewInt(int64(initPOSManagerFlags.weightToValueFactor)), //nolint:gosec // G115: Weight factor is bounded
				RewardCalculatorAddress: initPOSManagerFlags.rewardCalculatorAddress,
				UptimeBlockchainID:      blockchainID,
			},
			managerAddress,
			validatormanagerSDK.SpecializationProxyContractAddress,
			managerOwnerPrivateKey,
			sc.UseACP99,
			signatureAggregatorEndpoint,
		); err != nil {
			return err
		}
		sidecar, err := app.LoadSidecar(blockchainName)
		if err != nil {
			return err
		}
		networkInfo := sidecar.Networks[network.Name()]
		networkInfo.ValidatorManagerAddress = validatormanagerSDK.SpecializationProxyContractAddress
		sidecar.Networks[network.Name()] = networkInfo
		if err := app.UpdateSidecar(&sidecar); err != nil {
			return err
		}
		ux.Logger.GreenCheckmarkToUser("Native Token Proof of Stake Validator Manager contract successfully initialized on blockchain %s", blockchainName)
	default: // unsupported
		return fmt.Errorf("only PoA and PoS supported")
	}
	return nil
}
