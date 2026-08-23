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
	// lightMnemonic re-exports the single source of truth (key.LightMnemonic
	// → github.com/luxfi/light.Mnemonic); do not re-declare the literal.
	lightMnemonic = key.LightMnemonic
)

// StartLocal starts a 3-node localnet on K8s via the operator.
// No netrunner — the operator manages StatefulSets, chain creation and deployment.
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
	operatorDir := filepath.Join(home, "work", "lux", "operator")

	// The localnet CRs live in their own namespace; the operator (deployed by
	// config/default into its own namespace) watches cluster-wide. AlreadyExists
	// is the expected steady state, so this is the one apply whose error is fine.
	_ = kubectl(ctx, "create", "namespace", "lux-system")

	// Operator CRDs — canonical home is spec/crd/bases/<group>.
	ux.Logger.PrintToUser("-> Operator CRDs")
	if err := kubectl(ctx, "apply", "-f", filepath.Join(operatorDir, "spec", "crd", "bases", "lux.cloud")); err != nil {
		return err
	}

	// Operator RBAC + Deployment — canonical home is the config/default kustomize.
	ux.Logger.PrintToUser("-> Operator RBAC + Deployment")
	if err := kubectl(ctx, "apply", "-k", filepath.Join(operatorDir, "config", "default")); err != nil {
		return err
	}

	// LuxNetwork CR for the localnet.
	ux.Logger.PrintToUser("-> LuxNetwork (%d validators, network ID %d)", localnetValidators, localnetNetworkID)
	if err := kubectl(ctx, "apply", "-f", filepath.Join(operatorDir, "spec", "examples", "luxnetwork-devnet.yaml")); err != nil {
		return err
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

func kubectl(ctx string, args ...string) error {
	fullArgs := append([]string{"--context", ctx}, args...)
	cmd := exec.Command("kubectl", fullArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("kubectl %v: %w", args, err)
	}
	return nil
}
