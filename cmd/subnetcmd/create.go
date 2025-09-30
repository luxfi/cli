// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package subnetcmd

import (
	"errors"
	"fmt"
	"unicode"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/utils"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/cli/pkg/vm"
	"github.com/luxfi/sdk/models"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

const (
	forceFlag = "force"
	latest    = "latest"
)

var (
	forceCreate      bool
	useSubnetEvm     bool
	genesisFile      string
	vmFile           string
	useCustom        bool
	vmVersion        string
	useLatestVersion bool

	// L2/Sequencer flags
	sequencer        string
	enablePreconfirm bool

	errIllegalNameCharacter = errors.New(
		"illegal name character: only letters, no special characters allowed")
)

// lux subnet create
func newCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [subnetName]",
		Short: "Create a new subnet configuration",
		Long: `The subnet create command builds a new subnet configuration that can be
deployed as an L2 (using any L1 as sequencer) or as a sovereign L1.

Subnets are L2s that can use different sequencing models:
- Based rollups: Use L1 block proposers as sequencers (Ethereum, Lux L1, etc.)
- Centralized: Traditional single sequencer model
- Distributed: Multiple sequencers with consensus

By default, the command runs an interactive wizard. It supports:
- Lux EVM and custom VMs
- Multiple base chains for sequencing
- Pre-confirmations for fast UX
- Migration paths between different models

Use --sequencer to specify the sequencing model (lux, ethereum, lux, op, external).
Use -f to overwrite existing configurations.`,
		SilenceUsage:      true,
		Args:              cobra.ExactArgs(1),
		RunE:              createSubnetConfig,
		PersistentPostRun: handlePostRun,
	}
	cmd.Flags().StringVar(&genesisFile, "genesis", "", "file path of genesis to use")
	cmd.Flags().StringVar(&vmFile, "vm", "", "file path of custom vm to use")
	cmd.Flags().BoolVar(&useSubnetEvm, "evm", false, "use the Lux EVM as the base template")
	cmd.Flags().StringVar(&vmVersion, "vm-version", "", "version of vm template to use")
	cmd.Flags().BoolVar(&useCustom, "custom", false, "use a custom VM template")
	cmd.Flags().BoolVar(&useLatestVersion, latest, false, "use latest VM version, takes precedence over --vm-version")
	cmd.Flags().BoolVarP(&forceCreate, forceFlag, "f", false, "overwrite the existing configuration if one exists")

	// L2/Sequencer flags
	cmd.Flags().StringVar(&sequencer, "sequencer", "", "sequencer for the L2 (lux, ethereum, lux, op, external)")
	cmd.Flags().BoolVar(&enablePreconfirm, "enable-preconfirm", false, "enable pre-confirmations for fast UX")
	return cmd
}

func moreThanOneVMSelected() bool {
	vmVars := []bool{useSubnetEvm, useCustom}
	firstSelect := false
	for _, val := range vmVars {
		if firstSelect && val {
			return true
		} else if val {
			firstSelect = true
		}
	}
	return false
}

func getVMFromFlag() models.VMType {
	if useSubnetEvm {
		return models.EVM
	}
	if useCustom {
		return models.CustomVM
	}
	return ""
}

// override postrun function from root.go, so that we don't double send metrics for the same command
func handlePostRun(_ *cobra.Command, _ []string) {}

func createSubnetConfig(cmd *cobra.Command, args []string) error {
	subnetName := args[0]
	if app.GenesisExists(subnetName) && !forceCreate {
		return errors.New("configuration already exists. Use --" + forceFlag + " parameter to overwrite")
	}

	if err := checkInvalidSubnetNames(subnetName); err != nil {
		return fmt.Errorf("subnet name %q is invalid: %w", subnetName, err)
	}

	if moreThanOneVMSelected() {
		return errors.New("too many VMs selected. Provide at most one VM selection flag")
	}

	subnetType := getVMFromFlag()

	if subnetType == "" {
		subnetTypeStr, err := app.Prompt.CaptureList(
			"Choose your VM",
			[]string{models.EVM, models.CustomVM},
		)
		if err != nil {
			return err
		}
		subnetType = models.VMTypeFromString(subnetTypeStr)
	}

	var (
		genesisBytes []byte
		sc           *models.Sidecar
		err          error
	)

	if useLatestVersion {
		vmVersion = latest
	}

	if vmVersion != latest && vmVersion != "" && !semver.IsValid(vmVersion) {
		return fmt.Errorf("invalid version string, should be semantic version (ex: v1.1.1): %s", vmVersion)
	}

	switch subnetType {
	case models.EVM:
		genesisBytes, sc, err = vm.CreateEvmConfig(app, subnetName, genesisFile, vmVersion)
		if err != nil {
			return err
		}
	case models.CustomVM:
		genesisBytes, sc, err = vm.CreateCustomSubnetConfig(app, subnetName, genesisFile, vmFile)
		if err != nil {
			return err
		}
	default:
		return errors.New("not implemented")
	}

	// Configure L2/Sequencer settings
	if sequencer == "" && !cmd.Flags().Changed("sequencer") {
		// Interactive sequencer selection
		sequencerOptions := []string{
			"Lux (100ms blocks, lowest cost, based rollup)",
			"Ethereum (12s blocks, highest security, based rollup)",
			"Lux (2s blocks, fast finality, based rollup)",
			"OP Stack (Optimism compatible)",
			"External (Traditional sequencer)",
			"None (Deploy as sovereign L1)",
		}

		choice, err := app.Prompt.CaptureList(
			"Select sequencer for your L2",
			sequencerOptions,
		)
		if err != nil {
			return err
		}

		switch choice {
		case "Lux (100ms blocks, lowest cost, based rollup)":
			sequencer = "lux"
		case "Ethereum (12s blocks, highest security, based rollup)":
			sequencer = "ethereum"
		case "Lux (2s blocks, fast finality, based rollup)":
			sequencer = "lux"
		case "OP Stack (Optimism compatible)":
			sequencer = "op"
		case "External (Traditional sequencer)":
			sequencer = "external"
		case "None (Deploy as sovereign L1)":
			sc.Sovereign = true
		}
	}

	// Apply L2 configuration
	if sequencer != "" {
		sc.BaseChain = sequencer
		sc.BasedRollup = isBasedRollup(sequencer)
		sc.Sovereign = false // L2s are not sovereign
		sc.SequencerType = sequencer
		sc.L1BlockTime = getBlockTime(sequencer)
		sc.PreconfirmEnabled = enablePreconfirm

		ux.Logger.PrintToUser("🔧 L2 Configuration:")
		ux.Logger.PrintToUser("   Sequencer: %s", sequencer)
		if isBasedRollup(sequencer) {
			ux.Logger.PrintToUser("   Type: Based rollup (L1-sequenced)")
		} else if sequencer == "op" {
			ux.Logger.PrintToUser("   Type: OP Stack compatible")
		} else {
			ux.Logger.PrintToUser("   Type: External sequencer")
		}
		ux.Logger.PrintToUser("   Block Time: %dms", sc.L1BlockTime)
		if enablePreconfirm {
			ux.Logger.PrintToUser("   Pre-confirmations: Enabled")
		}
	} else if sc.Sovereign {
		ux.Logger.PrintToUser("🔧 L1 Configuration:")
		ux.Logger.PrintToUser("   Type: Sovereign L1")
		ux.Logger.PrintToUser("   Validation: Independent")
	}

	if err = app.WriteGenesisFile(subnetName, genesisBytes); err != nil {
		return err
	}

	sc.ImportedFromLPM = false
	if err = app.CreateSidecar(sc); err != nil {
		return err
	}
	flags := make(map[string]string)
	flags[constants.SubnetType] = subnetType.RepoName()
	utils.HandleTracking(cmd, app, flags)
	ux.Logger.PrintToUser("Successfully created subnet configuration")
	return nil
}

func checkInvalidSubnetNames(name string) error {
	// this is currently exactly the same code as in node/vms/platformvm/create_chain_tx.go
	for _, r := range name {
		if r > unicode.MaxASCII || !(unicode.IsLetter(r) || unicode.IsNumber(r) || r == ' ') {
			return errIllegalNameCharacter
		}
	}

	return nil
}
