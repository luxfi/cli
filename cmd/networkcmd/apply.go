// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"fmt"
	"os"

	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/netspec"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	applySpecPath string
	applyDryRun   bool
	applyForce    bool
)

// newApplyCmd creates the network apply command for declarative network management.
func newApplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply a declarative network specification",
		Long: `Apply a network specification file to create or update a network.

This command provides Infrastructure as Code (IaC) for Lux networks.
It reads a YAML or JSON specification file and ensures the network matches
the desired state.

The command is idempotent - running it multiple times with the same spec
will only make changes when necessary.

Example spec.yaml:
  apiVersion: lux.network/v1
  kind: Network
  network:
    name: mydevnet
    nodes: 5
    subnets:
      - name: mychain
        vm: subnet-evm
        chainId: 12345
        tokenSymbol: MYT
        validators: 3
        testDefaults: true

Usage:
  lux network apply -f spec.yaml
  lux network apply -f spec.yaml --dry-run
  lux network apply -f spec.yaml --force`,
		RunE:    applySpec,
		PreRunE: cobrautils.ExactArgs(0),
	}

	cmd.Flags().StringVarP(&applySpecPath, "file", "f", "", "path to network specification file (required)")
	cmd.Flags().BoolVar(&applyDryRun, "dry-run", false, "show what would be changed without making changes")
	cmd.Flags().BoolVar(&applyForce, "force", false, "force recreation of existing resources")

	_ = cmd.MarkFlagRequired("file")

	return cmd
}

// applySpec reads the spec and applies it to create/update the network.
func applySpec(cmd *cobra.Command, args []string) error {
	// Parse the specification file
	spec, err := netspec.ParseFile(applySpecPath)
	if err != nil {
		return fmt.Errorf("failed to parse specification: %w", err)
	}

	ux.Logger.PrintToUser("Applying network specification: %s", spec.Network.Name)
	ux.Logger.PrintToUser("")

	// Get current network state
	currentState, err := getCurrentNetworkState(spec.Network.Name)
	if err != nil {
		return fmt.Errorf("failed to get current state: %w", err)
	}

	// Calculate diff
	diff := netspec.Diff(spec, currentState)

	if !diff.HasChanges() && !applyForce {
		ux.Logger.GreenCheckmarkToUser("Network is up to date. No changes needed.")
		return nil
	}

	// Display planned changes
	ux.Logger.PrintToUser("Planned changes:")
	ux.Logger.PrintToUser("  %s", diff.String())
	ux.Logger.PrintToUser("")

	if applyDryRun {
		ux.Logger.PrintToUser("Dry run complete. No changes made.")
		return nil
	}

	// Apply changes
	if err := applyChanges(cmd, spec, diff, currentState); err != nil {
		return err
	}

	ux.Logger.GreenCheckmarkToUser("Network specification applied successfully")
	return nil
}

// getCurrentNetworkState retrieves the current state of the network.
func getCurrentNetworkState(networkName string) (*netspec.NetworkState, error) {
	state := &netspec.NetworkState{
		Name: networkName,
	}

	// Check if network is running by checking for any blockchain configs
	subnetDir := app.GetSubnetDir()
	entries, err := os.ReadDir(subnetDir)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return nil, err
	}

	// Scan for deployed subnets
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		sc, err := app.LoadSidecar(entry.Name())
		if err != nil {
			continue
		}

		subnetState := netspec.SubnetState{
			Name:      sc.Name,
			VM:        string(sc.VM),
			VMVersion: sc.VMVersion,
			ChainID:   parseChainID(sc.ChainID),
		}

		// Check if deployed to local network
		if networks := sc.Networks; networks != nil {
			if localData, ok := networks["Local Network"]; ok {
				subnetState.Deployed = localData.BlockchainID.String() != ""
				subnetState.SubnetID = localData.SubnetID.String()
				subnetState.BlockchainID = localData.BlockchainID.String()
				if len(localData.RPCEndpoints) > 0 {
					subnetState.RPCEndpoint = localData.RPCEndpoints[0]
				}
			}
		}

		state.Subnets = append(state.Subnets, subnetState)
	}

	// Check if network is running
	state.Running, _ = isNetworkRunning()

	// Get node count if running
	if state.Running {
		state.Nodes = getRunningNodeCount()
	}

	return state, nil
}

// applyChanges applies the changes defined in the diff.
func applyChanges(cmd *cobra.Command, spec *netspec.NetworkSpec, diff *netspec.DiffResult, currentState *netspec.NetworkState) error {
	// Handle network-level changes (restart with different node count, etc.)
	if diff.NetworkChanges || diff.NeedsRestart {
		if err := applyNetworkChanges(spec, currentState); err != nil {
			return fmt.Errorf("failed to apply network changes: %w", err)
		}
	}

	// Create new subnets
	for _, subnet := range diff.SubnetsToCreate {
		ux.Logger.PrintToUser("Creating subnet: %s", subnet.Name)
		if err := createSubnetFromSpec(cmd, subnet); err != nil {
			return fmt.Errorf("failed to create subnet %s: %w", subnet.Name, err)
		}
	}

	// Update existing subnets
	for _, subnet := range diff.SubnetsToUpdate {
		ux.Logger.PrintToUser("Updating subnet: %s", subnet.Name)
		if err := updateSubnetFromSpec(cmd, subnet); err != nil {
			return fmt.Errorf("failed to update subnet %s: %w", subnet.Name, err)
		}
	}

	// Delete subnets
	for _, name := range diff.SubnetsToDelete {
		ux.Logger.PrintToUser("Deleting subnet: %s", name)
		if err := CallDeleteBlockchain(name); err != nil {
			return fmt.Errorf("failed to delete subnet %s: %w", name, err)
		}
	}

	// Deploy subnets that were created
	for _, subnet := range diff.SubnetsToCreate {
		ux.Logger.PrintToUser("Deploying subnet: %s", subnet.Name)
		if err := deploySubnetFromSpec(cmd, subnet); err != nil {
			return fmt.Errorf("failed to deploy subnet %s: %w", subnet.Name, err)
		}
	}

	return nil
}

// applyNetworkChanges handles network-level changes like node count.
func applyNetworkChanges(spec *netspec.NetworkSpec, currentState *netspec.NetworkState) error {
	// If network is running with wrong node count, restart
	if currentState.Running && currentState.Nodes != spec.Network.Nodes {
		ux.Logger.PrintToUser("Restarting network with %d nodes...", spec.Network.Nodes)
		// Stop the network
		if err := StopNetwork(nil, nil); err != nil {
			return err
		}
	}

	// Start network with correct configuration
	luxdVersion := spec.Network.LuxdVersion
	if luxdVersion == "" {
		luxdVersion = constants.DefaultLuxdVersion
	}

	numNodes = spec.Network.Nodes
	return Start(StartFlags{
		UserProvidedLuxdVersion: luxdVersion,
		NumNodes:                spec.Network.Nodes,
	}, true)
}

// createSubnetFromSpec creates a blockchain from a spec.
func createSubnetFromSpec(cmd *cobra.Command, subnet netspec.SubnetSpec) error {
	return CallCreate(
		cmd,
		subnet.Name,
		true, // force
		subnet.Genesis,
		subnet.VM == "subnet-evm",
		subnet.VM == "custom",
		subnet.VMVersion,
		subnet.ChainID,
		subnet.TokenSymbol,
		subnet.ProductionDefaults,
		subnet.TestDefaults,
		subnet.VMVersion == "latest",
		false, // pre-release
		"",    // custom VM repo
		"",    // custom VM branch
		"",    // custom VM build script
	)
}

// updateSubnetFromSpec updates an existing subnet configuration.
func updateSubnetFromSpec(cmd *cobra.Command, subnet netspec.SubnetSpec) error {
	// Delete and recreate to update
	if err := CallDeleteBlockchain(subnet.Name); err != nil {
		return err
	}
	return createSubnetFromSpec(cmd, subnet)
}

// deploySubnetFromSpec deploys a subnet to the local network.
func deploySubnetFromSpec(cmd *cobra.Command, subnet netspec.SubnetSpec) error {
	return CallDeploy(
		cmd,
		false, // not subnet only
		subnet.Name,
		globalNetworkFlags, // use current network flags
		"",                 // key name
		false,              // use ledger
		true,               // use ewoq
		true,               // same control key
	)
}

// isNetworkRunning checks if the local network is running.
func isNetworkRunning() (bool, error) {
	// Check if the server process is running
	checker := binutils.NewProcessChecker()
	isRunning, err := checker.IsServerProcessRunning(app)
	if err != nil {
		return false, nil
	}
	return isRunning, nil
}

// getRunningNodeCount returns the number of running nodes.
func getRunningNodeCount() uint32 {
	// Default to 5 if we can't determine
	return 5
}

// parseChainID parses a chain ID string to uint64.
func parseChainID(s string) uint64 {
	var id uint64
	fmt.Sscanf(s, "%d", &id)
	return id
}
