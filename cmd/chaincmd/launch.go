// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chaincmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/luxfi/cli/pkg/chainkit"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	launchDryRun    bool
	launchNetwork   string
	launchService   string
	launchOutputDir string
	launchApply     bool
)

func newLaunchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "launch <chain.yaml>",
		Short: "Launch a complete blockchain ecosystem from chain.yaml",
		Long: `Launch a complete blockchain ecosystem from a single chain.yaml configuration.

OVERVIEW:

  The launch command reads a chain.yaml file and generates Kubernetes CRDs
  that the lux-operator reconciles into a fully running ecosystem:
  nodes, indexer, explorer, gateway, exchange, and faucet.

GENERATED RESOURCES:

  LuxNetwork    Validator node fleet (StatefulSet, genesis, staking)
  LuxIndexer    Blockscout indexer per chain
  LuxExplorer   Branded explorer frontend
  LuxGateway    API gateway with rate limiting and CORS
  Exchange      DEX frontend deployment (branded)
  Faucet        Testnet/devnet token faucet

EXAMPLES:

  # Generate manifests for all networks (dry run)
  lux chain launch chain.yaml --dry-run

  # Generate and apply to devnet only
  lux chain launch chain.yaml --network=devnet --apply

  # Generate only explorer manifests
  lux chain launch chain.yaml --service=explorer --dry-run

  # Output manifests to custom directory
  lux chain launch chain.yaml --output=./k8s/generated --dry-run

WORKFLOW:

  1. Create chain.yaml in your project root
  2. Run: lux chain launch chain.yaml --dry-run
  3. Review generated manifests
  4. Run: lux chain launch chain.yaml --network=devnet --apply
  5. Monitor: kubectl get luxnet,luxidx,luxexp,luxgw -n <namespace>

NOTES:

  - chain.yaml is the single source of truth for the entire ecosystem
  - Generated CRDs require the lux-operator to be running in the cluster
  - Ingress uses hanzoai/ingress (never nginx/caddy)
  - All secrets are referenced via KMS, never stored in manifests`,
		Args: cobra.ExactArgs(1),
		RunE: runLaunch,
	}

	cmd.Flags().BoolVar(&launchDryRun, "dry-run", false, "Generate manifests without applying")
	cmd.Flags().StringVar(&launchNetwork, "network", "", "Target specific network (mainnet, testnet, devnet)")
	cmd.Flags().StringVar(&launchService, "service", "", "Generate only specific service (node, indexer, explorer, gateway, exchange, faucet)")
	cmd.Flags().StringVarP(&launchOutputDir, "output", "o", "", "Output directory for generated manifests")
	cmd.Flags().BoolVar(&launchApply, "apply", false, "Apply generated manifests to the cluster via kubectl")

	return cmd
}

func runLaunch(_ *cobra.Command, args []string) error {
	chainFile := args[0]

	// Resolve relative path
	if !filepath.IsAbs(chainFile) {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
		chainFile = filepath.Join(cwd, chainFile)
	}

	// Load and validate
	ux.Logger.PrintToUser("Loading %s", chainFile)
	cfg, err := chainkit.Load(chainFile)
	if err != nil {
		return err
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	ux.Logger.PrintToUser("Chain: %s (%s)", cfg.Chain.Name, cfg.Chain.Slug)
	ux.Logger.PrintToUser("Type: %s | VM: %s | Sequencer: %s", cfg.Chain.Type, cfg.Chain.VM, cfg.Chain.Sequencer)
	ux.Logger.PrintToUser("Token: %s (%s)", cfg.Token.Name, cfg.Token.Symbol)

	// Determine which networks to generate
	networks := make([]string, 0, len(cfg.Networks))
	if launchNetwork != "" {
		if _, ok := cfg.Networks[launchNetwork]; !ok {
			return fmt.Errorf("network %q not defined in chain.yaml (available: %s)",
				launchNetwork, availableNetworks(cfg))
		}
		networks = append(networks, launchNetwork)
	} else {
		for name := range cfg.Networks {
			networks = append(networks, name)
		}
	}

	// Determine output directory
	outDir := launchOutputDir
	if outDir == "" {
		outDir = filepath.Join(filepath.Dir(chainFile), "k8s", "generated")
	}

	// Generate manifests
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Generating manifests...")

	var allResults []*chainkit.GenerateResult
	for _, net := range networks {
		result, err := chainkit.Generate(cfg, net)
		if err != nil {
			return fmt.Errorf("generate %s: %w", net, err)
		}
		allResults = append(allResults, result)

		printResult(cfg, result)
	}

	// Write manifests to disk
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	var writtenFiles []string
	for _, r := range allResults {
		dir := filepath.Join(outDir, r.Network)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create %s directory: %w", r.Network, err)
		}

		pairs := []struct {
			name string
			data string
		}{
			{"namespace.yaml", r.Namespace_},
			{"luxnetwork.yaml", r.LuxNetwork},
			{"luxindexer.yaml", r.LuxIndexer},
			{"luxexplorer.yaml", r.LuxExplorer},
			{"luxgateway.yaml", r.LuxGateway},
			{"exchange.yaml", r.Exchange},
			{"faucet.yaml", r.Faucet},
		}
		for _, p := range pairs {
			if p.data == "" {
				continue
			}
			// Filter by service if specified
			if launchService != "" && !matchesService(p.name, launchService) {
				continue
			}
			path := filepath.Join(dir, p.name)
			if err := os.WriteFile(path, []byte(p.data), 0o644); err != nil {
				return fmt.Errorf("write %s: %w", path, err)
			}
			writtenFiles = append(writtenFiles, path)
		}
	}

	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Generated %d manifests in %s", len(writtenFiles), outDir)
	for _, f := range writtenFiles {
		rel, _ := filepath.Rel(outDir, f)
		ux.Logger.PrintToUser("  %s", rel)
	}

	// Apply if requested
	if launchApply && !launchDryRun {
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("Applying manifests to cluster...")
		for _, f := range writtenFiles {
			ux.Logger.PrintToUser("  kubectl apply -f %s", f)
			cmd := exec.Command("kubectl", "apply", "-f", f)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("kubectl apply -f %s: %w", f, err)
			}
		}
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("Monitor with:")
		for _, r := range allResults {
			ux.Logger.PrintToUser("  kubectl get luxnet,luxidx,luxexp,luxgw -n %s", r.Namespace)
		}
	} else if !launchDryRun && !launchApply {
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("To apply: lux chain launch %s --network=%s --apply", args[0], networks[0])
	}

	return nil
}

func printResult(cfg *chainkit.ChainConfig, r *chainkit.GenerateResult) {
	net := cfg.Networks[r.Network]
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("  Network: %s (chainId: %d, namespace: %s)", r.Network, net.ChainID, r.Namespace)

	services := []string{}
	if r.LuxNetwork != "" {
		services = append(services, fmt.Sprintf("node (%d validators)", net.Validators))
	}
	if r.LuxIndexer != "" {
		services = append(services, "indexer")
	}
	if r.LuxExplorer != "" {
		services = append(services, "explorer")
	}
	if r.LuxGateway != "" {
		services = append(services, "gateway")
	}
	if r.Exchange != "" {
		services = append(services, "exchange")
	}
	if r.Faucet != "" {
		services = append(services, "faucet")
	}
	ux.Logger.PrintToUser("  Services: %s", strings.Join(services, ", "))
}

func availableNetworks(cfg *chainkit.ChainConfig) string {
	names := make([]string, 0, len(cfg.Networks))
	for name := range cfg.Networks {
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}

func matchesService(filename, service string) bool {
	switch service {
	case "node":
		return filename == "luxnetwork.yaml" || filename == "namespace.yaml"
	case "indexer":
		return filename == "luxindexer.yaml" || filename == "namespace.yaml"
	case "explorer":
		return filename == "luxexplorer.yaml" || filename == "namespace.yaml"
	case "gateway":
		return filename == "luxgateway.yaml" || filename == "namespace.yaml"
	case "exchange":
		return filename == "exchange.yaml" || filename == "namespace.yaml"
	case "faucet":
		return filename == "faucet.yaml" || filename == "namespace.yaml"
	default:
		return true
	}
}
