// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package networkcmd

import (
	"errors"

	"github.com/luxfi/cli/pkg/chain"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/localnet"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/cli/pkg/warp/relayer"
	"github.com/luxfi/cli/pkg/warp/signatureaggregator"
	"github.com/luxfi/constants"
	luxlog "github.com/luxfi/log"
	"github.com/luxfi/sdk/models"

	"github.com/spf13/cobra"
)

var resetPlugins bool

func newCleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Stop network and delete runtime data (preserves chain configs)",
		Long: `The network clean command stops the network and deletes runtime data.

⚠️  IMPORTANT - WHAT GETS DELETED:

  Runtime Data (DELETED):
  - Network snapshots (blockchain state, databases)
  - Validator state
  - Log files
  - Running processes

  Chain Configs (PRESERVED):
  - Chain configurations in ~/.lux/chains/
  - Genesis files
  - Sidecar metadata

BEHAVIOR:

  1. Stops the running network gracefully
  2. Deletes network runtime data and snapshots
  3. Removes local deployment info from sidecars
  4. Preserves chain configurations for redeployment

  After cleaning, you can redeploy your chains to a fresh network:
    lux network start --devnet
    lux chain deploy mychain

OPTIONS:

  --reset-plugins    Also delete the plugins directory (removes user-installed VMs)

EXAMPLES:

  # Clean network runtime (most common)
  lux network clean

  # Clean and also remove custom VM plugins
  lux network clean --reset-plugins

WHEN TO USE:

  ✓ Network state is corrupted
  ✓ Want to start fresh but keep chain configs
  ✓ Testing deployment from scratch
  ✓ Cleaning up after development session

  ✗ Just want to stop the network (use 'lux network stop')
  ✗ Want to delete a specific chain (use 'lux chain delete <name>')

CLEAN vs STOP:

  lux network stop     - Saves state for resuming later
  lux network clean    - Deletes runtime data, preserves chain configs
  lux chain delete     - Deletes a specific chain configuration

NOTE: Chain configurations are explicitly preserved. To delete a chain
configuration, use: lux chain delete <chainName>`,
		RunE: clean,
		Args: cobrautils.ExactArgs(0),
	}
	cmd.Flags().BoolVar(&resetPlugins, "reset-plugins", false, "also reset the plugins directory (removes user-installed VMs)")

	return cmd
}

func clean(*cobra.Command, []string) error {
	if err := localnet.LocalNetworkStop(app); err != nil && !errors.Is(err, localnet.ErrNetworkNotRunning) {
		return err
	} else if err == nil {
		ux.Logger.PrintToUser("Process terminated.")
	} else {
		ux.Logger.PrintToUser("%s", luxlog.Red.Wrap("No network is running."))
	}

	if err := relayer.Cleanup(
		app.GetLocalRelayerRunPath(models.Local),
		app.GetLocalRelayerLogPath(models.Local),
		app.GetLocalRelayerStorageDir(models.Local),
	); err != nil {
		return err
	}

	// Clean up signature aggregator
	network := models.NewLocalNetwork()
	if err := signatureaggregator.Cleanup(
		app.GetLocalRelayerRunPath(network),
		app.GetLocalRelayerStorageDir(network),
	); err != nil {
		return err
	}

	if resetPlugins {
		if err := app.ResetPluginsDir(); err != nil {
			return err
		}
	}

	if err := removeLocalDeployInfoFromSidecars(); err != nil {
		return err
	}

	// SAFETY: Use SafeRemoveAll to prevent accidental deletion of protected directories
	// Only delete the snapshot, NOT the chains directory
	snapshotPath := app.GetSnapshotPath(constants.DefaultSnapshotName)
	if err := app.SafeRemoveAll(snapshotPath); err != nil {
		// Log warning but don't fail - the snapshot may not exist
		ux.Logger.PrintToUser("Warning: could not clean snapshot: %v", err)
	}

	clusterNames, err := localnet.GetRunningLocalClustersConnectedToLocalNetwork(app)
	if err != nil {
		return err
	}
	for _, clusterName := range clusterNames {
		if err := localnet.LocalClusterRemove(app, clusterName); err != nil {
			return err
		}
	}

	// Explicitly note that chain configs are preserved
	ux.Logger.PrintToUser("Note: Chain configurations in %s are preserved. Use 'lux chain delete <name>' to remove individual chains.", app.GetChainsDir())

	return nil
}

func removeLocalDeployInfoFromSidecars() error {
	// Remove all local deployment info from sidecar files
	deployedChains, err := chain.GetLocallyDeployedChainsFromFile(app)
	if err != nil {
		return err
	}

	for _, chain := range deployedChains {
		sc, err := app.LoadSidecar(chain)
		if err != nil {
			return err
		}

		delete(sc.Networks, models.Local.String())
		if err = app.UpdateSidecar(&sc); err != nil {
			return err
		}
	}
	return nil
}
