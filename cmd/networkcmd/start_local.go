// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package networkcmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/cli/pkg/ux"
)

const (
	localnetNetworkID  = uint32(1337)
	localnetValidators = 3
	localEVMChainID    = 1337
	lightMnemonic      = "light light light light light light light light light light light energy"
)

// StartLocal starts a 3-node localnet on K8s via the operator.
// No netrunner — the operator manages StatefulSets, subnet creation, chain deployment.
//
//   lux network start --local
//   lux network start --local --k8s colima
func StartLocal() error {
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("╔══════════════════════════════════════════════╗")
	ux.Logger.PrintToUser("║  Lux Network — Localnet (3 nodes, K8s)       ║")
	ux.Logger.PrintToUser("╚══════════════════════════════════════════════╝")
	ux.Logger.PrintToUser("")

	ctx := k8sCluster
	if ctx == "" {
		ctx = "colima"
	}

	if err := checkK8s(ctx); err != nil {
		return fmt.Errorf("K8s not available (context: %s): %w\nStart colima: colima start --kubernetes --cpu 4 --memory 8", ctx, err)
	}
	ux.Logger.PrintToUser("K8s context: %s", ctx)

	// Show funded accounts
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Funded Accounts (light mnemonic):")
	for i := 0; i < localnetValidators; i++ {
		sf, err := key.NewSoftFromMnemonicWithAccount(localEVMChainID, lightMnemonic, uint32(i))
		if err != nil {
			continue
		}
		label := ""
		if i == 0 {
			label = " (deployer)"
		}
		ux.Logger.PrintToUser("  [%d] %s%s", i, sf.C(), label)
	}
	ux.Logger.PrintToUser("")

	home, _ := os.UserHomeDir()

	// Apply operator CRDs + deployment
	ux.Logger.PrintToUser("-> Operator CRDs")
	operatorDir := filepath.Join(home, "work", "lux", "operator", "k8s")
	crds, _ := filepath.Glob(filepath.Join(operatorDir, "crds", "*.yaml"))
	for _, crd := range crds {
		kubectl(ctx, "apply", "-f", crd)
	}

	ux.Logger.PrintToUser("-> Operator RBAC + Deployment")
	kubectl(ctx, "apply",
		"-f", filepath.Join(operatorDir, "rbac", "serviceaccount.yaml"),
		"-f", filepath.Join(operatorDir, "rbac", "clusterrole.yaml"),
		"-f", filepath.Join(operatorDir, "rbac", "clusterrolebinding.yaml"),
		"-f", filepath.Join(operatorDir, "deployment.yaml"))

	// Apply network CR for localnet
	ux.Logger.PrintToUser("-> LuxNetwork (3 validators, network ID %d)", localnetNetworkID)
	networkCR := filepath.Join(operatorDir, "networks", "devnet.yaml")
	if _, err := os.Stat(networkCR); err == nil {
		kubectl(ctx, "apply", "-f", networkCR)
	}

	// Apply platform services if they exist
	platformCR := filepath.Join(operatorDir, "platforms", "devnet.yaml")
	if _, err := os.Stat(platformCR); err == nil {
		ux.Logger.PrintToUser("-> Platform services")
		kubectl(ctx, "apply", "-f", platformCR)
	}

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("╔══════════════════════════════════════════════╗")
	ux.Logger.PrintToUser("║  Localnet deploying via operator              ║")
	ux.Logger.PrintToUser("║  Check: lux network status                    ║")
	ux.Logger.PrintToUser("║  Stop:  lux network stop                      ║")
	ux.Logger.PrintToUser("╚══════════════════════════════════════════════╝")

	return nil
}

func checkK8s(ctx string) error {
	cmd := exec.Command("kubectl", "--context", ctx, "cluster-info")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

func kubectl(ctx string, args ...string) {
	fullArgs := append([]string{"--context", ctx}, args...)
	cmd := exec.Command("kubectl", fullArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}
