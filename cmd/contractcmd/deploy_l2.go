// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package contractcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// newDeployL2Cmd ports lux/standard/script/deploy_l2.sh. Deploys the
// standard contract suite (Safe + Bridge + Exchange + sToken) to one or
// more L2 chains using forge.
//
// Inventory source: lux/genesis/configs/inventory/l2-<latest>.json.
//
// Default mode is dry-run. --confirm switches to broadcast. Mainnet broadcast
// additionally requires --i-know-this-is-real-money to prevent accidental
// real-money deploys.
//
// Per-(brand,env) deploy is idempotent: if the recorded WLUX address has
// bytecode on chain, the deploy is skipped. Use --resume to override and
// have forge continue a partial broadcast.
//
// Identity resolution (one wins, never mixed):
//  1. KMS path: if LUX_KMS_AUTH_TOKEN set, shell out to kms-fetch for
//     brand/<brand>/<env>/deployer/private-key.
//  2. Mnemonic fallback: LUX_MNEMONIC env + --mnemonic-index flag.
func newDeployL2Cmd() *cobra.Command {
	var (
		env             string
		brand           string
		inventoryPath   string
		deployScript    string
		liquid          bool
		resume          bool
		confirm         bool
		realMoneyOK     bool
		deployerIdx     uint
		kmsFetchBin     string
		liquidScript    string
		repoRoot        string
		liquidRepoRoot  string
		broadcastDirRel string
	)
	cmd := &cobra.Command{
		Use:   "l2",
		Short: "Deploy the lux/standard contract suite to L2 chains",
		Long: `Deploy the canonical lux/standard contract stack (Safe + Bridge + Exchange +
sToken + WLUX/BridgedETH/BridgedBTC) to one or more L2 chains.

The inventory JSON describes which chains exist for a given env and what
their evmChainId values should be. The command per-brand:

  1. Probes the L2's RPC for eth_chainId and checks it matches inventory.
  2. Reads the deployments manifest; if WLUX is already deployed (cast code
     returns non-empty), the deploy is skipped (use --resume to override).
  3. Invokes forge script with the appropriate identity, RPC, and resume
     flags.
  4. Parses forge's broadcast output and writes a manifest at
     lux/standard/deployments/l2-<env>/<brand>.json.

Mainnet broadcast (--confirm with --env mainnet) is gated behind
--i-know-this-is-real-money.`,
		RunE: func(c *cobra.Command, _ []string) error {
			if env == "" {
				return fmt.Errorf("--env required (mainnet|testnet|devnet)")
			}
			gw, err := gatewayFor(env)
			if err != nil {
				return err
			}
			if env == "mainnet" && confirm && !realMoneyOK {
				return fmt.Errorf("mainnet broadcast requires --i-know-this-is-real-money")
			}
			if confirm && os.Getenv("LUX_MNEMONIC") == "" && os.Getenv("LUX_KMS_AUTH_TOKEN") == "" {
				return fmt.Errorf("broadcast requires LUX_MNEMONIC (or LUX_KMS_AUTH_TOKEN + KMS-provisioned key)")
			}

			home, _ := os.UserHomeDir()
			if inventoryPath == "" {
				inventoryPath = filepath.Join(home, "work/lux/genesis/configs/inventory/l2-2026-06-06.json")
			}
			if repoRoot == "" {
				repoRoot = filepath.Join(home, "work/lux/standard")
			}
			if liquidRepoRoot == "" {
				liquidRepoRoot = filepath.Join(home, "work/lux/liquid")
			}
			if kmsFetchBin == "" {
				kmsFetchBin = filepath.Join(home, "work/hanzo/kms/cmd/kms-fetch/kms-fetch")
			}
			if deployScript == "" {
				deployScript = "contracts/script/DeployMultiNetwork.s.sol"
			}
			if liquidScript == "" {
				liquidScript = "script/DeployL2.s.sol"
			}
			if broadcastDirRel == "" {
				broadcastDirRel = "broadcast"
			}

			inv, err := loadInventory(inventoryPath)
			if err != nil {
				return fmt.Errorf("inventory: %w", err)
			}
			envEntry, ok := inv.Envs[env]
			if !ok {
				return fmt.Errorf("inventory has no env=%s", env)
			}
			var brands []inventoryChain
			if brand != "" {
				for _, ch := range envEntry.Chains {
					if ch.Brand == brand {
						brands = append(brands, ch)
						break
					}
				}
				if len(brands) == 0 {
					return fmt.Errorf("brand=%s not found in env=%s", brand, env)
				}
			} else {
				brands = envEntry.Chains
			}

			outDir := filepath.Join(repoRoot, "deployments", "l2-"+env)
			if err := os.MkdirAll(outDir, 0o755); err != nil {
				return fmt.Errorf("mkdir outDir: %w", err)
			}

			fmt.Printf("============================================================\n")
			fmt.Printf("  lux contract deploy l2\n  env: %s\n  mode: %s\n", env, modeOf(confirm))
			brandNames := make([]string, len(brands))
			for i, b := range brands {
				brandNames[i] = b.Brand
			}
			fmt.Printf("  brands: %s\n  gateway: %s\n", strings.Join(brandNames, " "), gw)
			fmt.Printf("  inventory: %s\n  deployer-idx: %d\n  out: %s\n", inventoryPath, deployerIdx, outDir)
			fmt.Printf("============================================================\n")

			failures := 0
			for _, ch := range brands {
				if err := deployBrand(c.Context(), brandDeployArgs{
					Brand:          ch.Brand,
					ExpectedCID:    ch.EVMChainID,
					Gateway:        gw,
					Env:            env,
					RepoRoot:       repoRoot,
					DeployScript:   deployScript,
					OutDir:         outDir,
					BroadcastRel:   broadcastDirRel,
					Confirm:        confirm,
					Resume:         resume,
					DeployerIdx:    deployerIdx,
					KMSFetch:       kmsFetchBin,
					ScriptBaseName: scriptBaseName(deployScript),
				}); err != nil {
					failures++
					fmt.Printf("  ✗ %s: %v\n", ch.Brand, err)
				}
			}

			if liquid {
				for _, ch := range brands {
					if err := deployLiquid(c.Context(), liquidDeployArgs{
						Brand:        ch.Brand,
						Gateway:      gw,
						LiquidRoot:   liquidRepoRoot,
						LiquidScript: liquidScript,
						Manifest:     filepath.Join(outDir, ch.Brand+".json"),
						Confirm:      confirm,
						DeployerIdx:  deployerIdx,
					}); err != nil {
						fmt.Printf("  liquid/%s: %v\n", ch.Brand, err)
					}
				}
			}

			fmt.Printf("\n============================================================\n  done: %d brands processed, %d failures\n============================================================\n",
				len(brands), failures)
			if failures > 0 {
				return fmt.Errorf("%d deploy failure(s)", failures)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&env, "env", "", "mainnet|testnet|devnet")
	cmd.Flags().StringVar(&brand, "brand", "", "restrict to a single brand (default: all in inventory)")
	cmd.Flags().StringVar(&inventoryPath, "inventory", "", "inventory JSON path")
	cmd.Flags().StringVar(&deployScript, "script", "", "forge script (default: contracts/script/DeployMultiNetwork.s.sol)")
	cmd.Flags().BoolVar(&liquid, "liquid", false, "after standard succeeds, also deploy lux/liquid")
	cmd.Flags().BoolVar(&resume, "resume", false, "pass --resume to forge to continue a partial broadcast")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "broadcast instead of dry-run")
	cmd.Flags().BoolVar(&realMoneyOK, "i-know-this-is-real-money", false, "mainnet broadcast safeguard")
	cmd.Flags().UintVar(&deployerIdx, "deployer-index", 0, "BIP44 mnemonic index for the deployer key")
	cmd.Flags().StringVar(&kmsFetchBin, "kms-fetch", "", "kms-fetch binary path (default: ~/work/hanzo/kms/cmd/kms-fetch/kms-fetch)")
	cmd.Flags().StringVar(&repoRoot, "repo", "", "lux/standard repo root (default: ~/work/lux/standard)")
	cmd.Flags().StringVar(&liquidRepoRoot, "liquid-repo", "", "lux/liquid repo root (default: ~/work/lux/liquid)")
	cmd.Flags().StringVar(&liquidScript, "liquid-script", "", "forge script in lux/liquid (default: script/DeployL2.s.sol)")
	return cmd
}

// --- inventory model

type inventoryFile struct {
	Envs map[string]inventoryEnv `json:"envs"`
}

type inventoryEnv struct {
	Deployer string           `json:"deployer"`
	Chains   []inventoryChain `json:"chains"`
}

type inventoryChain struct {
	Brand       string `json:"brand"`
	ChainID     string `json:"chainId"`
	EVMChainID  uint64 `json:"evmChainId"`
	HistoricRLP string `json:"historicRLP,omitempty"`
}

func loadInventory(p string) (*inventoryFile, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var inv inventoryFile
	if err := json.Unmarshal(b, &inv); err != nil {
		return nil, err
	}
	return &inv, nil
}

func gatewayFor(env string) (string, error) {
	switch env {
	case "mainnet":
		return "https://api.lux.network", nil
	case "testnet":
		return "https://api.lux-test.network", nil
	case "devnet":
		return "https://api.lux-dev.network", nil
	}
	return "", fmt.Errorf("unknown env=%s", env)
}

func modeOf(b bool) string {
	if b {
		return "broadcast"
	}
	return "dry-run"
}

func scriptBaseName(p string) string { return strings.TrimSuffix(filepath.Base(p), ".sol") }

// --- per-brand deploy

type brandDeployArgs struct {
	Brand          string
	ExpectedCID    uint64
	Gateway        string
	Env            string
	RepoRoot       string
	DeployScript   string
	OutDir         string
	BroadcastRel   string
	Confirm        bool
	Resume         bool
	DeployerIdx    uint
	KMSFetch       string
	ScriptBaseName string
}

func deployBrand(ctx context.Context, a brandDeployArgs) error {
	rpc := fmt.Sprintf("%s/v1/bc/%s/rpc", a.Gateway, a.Brand)
	manifest := filepath.Join(a.OutDir, a.Brand+".json")
	fmt.Printf("\n--- %s @ %s ---\n", a.Brand, rpc)

	// Health check
	gotCID, err := ethChainID(ctx, rpc)
	if err != nil {
		return fmt.Errorf("rpc unreachable: %w", err)
	}
	if gotCID != a.ExpectedCID {
		return fmt.Errorf("chainId mismatch: got %d, expected %d", gotCID, a.ExpectedCID)
	}
	fmt.Printf("  ✓ chain alive at chainId %d\n", gotCID)

	// Idempotency
	if !a.Resume {
		if addr, ok := readDeployedAddr(manifest, "WLUX"); ok {
			code, err := castCode(ctx, addr, rpc)
			if err == nil && code != "" && code != "0x" {
				fmt.Printf("  ✓ already deployed (WLUX @ %s has bytecode) — skip (pass --resume to override)\n", addr)
				return nil
			}
		}
	}

	// Identity
	flags := []string{"--rpc-url", rpc}
	pk, kmsSrc := resolveDeployerKey(ctx, a.Brand, a.Env, a.KMSFetch)
	if pk != "" {
		fmt.Printf("  ✓ deployer key sourced from %s\n", kmsSrc)
		flags = append(flags, "--private-key", pk)
	} else {
		mn := os.Getenv("LUX_MNEMONIC")
		if mn == "" {
			return fmt.Errorf("no KMS key and LUX_MNEMONIC unset")
		}
		fmt.Printf("  ✓ deployer key derived from LUX_MNEMONIC idx %d\n", a.DeployerIdx)
		flags = append(flags, "--mnemonics", mn, "--mnemonic-indexes", strconv.FormatUint(uint64(a.DeployerIdx), 10))
	}
	if a.Confirm {
		flags = append(flags, "--broadcast", "--skip-simulation", "--slow")
	}
	if a.Resume {
		flags = append(flags, "--resume")
	}

	// forge script ...
	args := append([]string{"script", a.DeployScript}, flags...)
	cmd := exec.CommandContext(ctx, "forge", args...)
	cmd.Dir = a.RepoRoot
	out, runErr := cmd.CombinedOutput()
	tailLines(out, 50)
	if runErr != nil {
		return fmt.Errorf("forge script: %w", runErr)
	}
	fmt.Printf("  ✓ deploy script completed\n")

	// Manifest write
	if a.Confirm {
		bcFile := filepath.Join(a.RepoRoot, a.BroadcastRel, a.ScriptBaseName, strconv.FormatUint(a.ExpectedCID, 10), "run-latest.json")
		if err := mergeManifest(bcFile, manifest, a.Brand, a.Env, a.ExpectedCID, rpc); err != nil {
			fmt.Printf("  WARN: manifest merge: %v\n", err)
		} else {
			fmt.Printf("  ✓ manifest written: %s\n", manifest)
		}
	}
	return nil
}

// --- liquid deploy

type liquidDeployArgs struct {
	Brand        string
	Gateway      string
	LiquidRoot   string
	LiquidScript string
	Manifest     string
	Confirm      bool
	DeployerIdx  uint
}

func deployLiquid(ctx context.Context, a liquidDeployArgs) error {
	if _, err := os.Stat(filepath.Join(a.LiquidRoot, a.LiquidScript)); err != nil {
		return fmt.Errorf("liquid script missing: %w", err)
	}
	b, err := os.ReadFile(a.Manifest)
	if err != nil {
		return fmt.Errorf("manifest missing: %w", err)
	}
	var m struct {
		Contracts map[string]string `json:"contracts"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return fmt.Errorf("manifest parse: %w", err)
	}
	wlux, leth, lbtc := m.Contracts["WLUX"], m.Contracts["BridgedETH"], m.Contracts["BridgedBTC"]
	if wlux == "" || leth == "" || lbtc == "" {
		return fmt.Errorf("missing one of WLUX/BridgedETH/BridgedBTC in manifest")
	}
	rpc := fmt.Sprintf("%s/v1/bc/%s/rpc", a.Gateway, a.Brand)
	args := []string{"script", a.LiquidScript, "--rpc-url", rpc}
	mn := os.Getenv("LUX_MNEMONIC")
	if mn != "" {
		args = append(args, "--mnemonics", mn, "--mnemonic-indexes", strconv.FormatUint(uint64(a.DeployerIdx), 10))
	}
	if a.Confirm {
		args = append(args, "--broadcast")
	}
	cmd := exec.CommandContext(ctx, "forge", args...)
	cmd.Dir = a.LiquidRoot
	cmd.Env = append(os.Environ(), "WLUX="+wlux, "LETH="+leth, "LBTC="+lbtc, "BRAND="+a.Brand)
	out, runErr := cmd.CombinedOutput()
	tailLines(out, 30)
	if runErr != nil {
		return fmt.Errorf("forge script: %w", runErr)
	}
	return nil
}

// --- helpers

type rpcResponse struct {
	Result string `json:"result"`
}

func ethChainID(ctx context.Context, rpc string) (uint64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	body := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}`)
	req, _ := http.NewRequestWithContext(ctx, "POST", rpc, body)
	req.Header.Set("content-type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var r rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return 0, err
	}
	if !strings.HasPrefix(r.Result, "0x") {
		return 0, fmt.Errorf("unexpected: %q", r.Result)
	}
	v, err := strconv.ParseUint(strings.TrimPrefix(r.Result, "0x"), 16, 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func castCode(ctx context.Context, addr, rpc string) (string, error) {
	out, err := exec.CommandContext(ctx, "cast", "code", addr, "--rpc-url", rpc).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func readDeployedAddr(manifest, contract string) (string, bool) {
	b, err := os.ReadFile(manifest)
	if err != nil {
		return "", false
	}
	var m struct {
		Contracts map[string]string `json:"contracts"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return "", false
	}
	addr := m.Contracts[contract]
	return addr, addr != ""
}

func mergeManifest(bcFile, manifest, brand, env string, cid uint64, rpc string) error {
	b, err := os.ReadFile(bcFile)
	if err != nil {
		return err
	}
	var bc struct {
		Transactions []struct {
			TransactionType string `json:"transactionType"`
			ContractName    string `json:"contractName"`
			ContractAddress string `json:"contractAddress"`
		} `json:"transactions"`
	}
	if err := json.Unmarshal(b, &bc); err != nil {
		return err
	}
	contracts := map[string]string{}
	if eb, err := os.ReadFile(manifest); err == nil {
		var existing struct {
			Contracts map[string]string `json:"contracts"`
		}
		if err := json.Unmarshal(eb, &existing); err == nil {
			for k, v := range existing.Contracts {
				contracts[k] = v
			}
		}
	}
	for _, tx := range bc.Transactions {
		if tx.TransactionType == "CREATE" && tx.ContractName != "" {
			contracts[tx.ContractName] = tx.ContractAddress
		}
	}
	out := struct {
		Brand     string            `json:"brand"`
		Env       string            `json:"env"`
		ChainID   uint64            `json:"chainId"`
		RPC       string            `json:"rpc"`
		Contracts map[string]string `json:"contracts"`
	}{brand, env, cid, rpc, contracts}
	buf, _ := json.MarshalIndent(out, "", "  ")
	return os.WriteFile(manifest, buf, 0o644)
}

// resolveDeployerKey returns (pkHex, source) — pkHex empty if not found.
func resolveDeployerKey(ctx context.Context, brand, env, kmsFetchBin string) (string, string) {
	if os.Getenv("LUX_KMS_AUTH_TOKEN") == "" || kmsFetchBin == "" {
		return "", ""
	}
	if _, err := os.Stat(kmsFetchBin); err != nil {
		return "", ""
	}
	tmp, err := os.MkdirTemp("", "kms-pk-*")
	if err != nil {
		return "", ""
	}
	defer os.RemoveAll(tmp)
	cmd := exec.CommandContext(ctx, kmsFetchBin)
	cmd.Env = append(os.Environ(),
		"KMS_SECRETS=PK=brand/"+brand+"/"+env+"/deployer/private-key",
		"OUT_DIR="+tmp, "WRITE_ENV_FILE=false")
	if err := cmd.Run(); err != nil {
		return "", ""
	}
	pk, err := os.ReadFile(filepath.Join(tmp, "PK"))
	if err != nil {
		return "", ""
	}
	pkStr := strings.TrimSpace(string(pk))
	if pkStr == "" {
		return "", ""
	}
	return pkStr, "KMS"
}

func tailLines(out []byte, n int) {
	lines := bytes.Split(bytes.TrimRight(out, "\n"), []byte{'\n'})
	start := 0
	if len(lines) > n {
		start = len(lines) - n
	}
	for _, l := range lines[start:] {
		fmt.Printf("    %s\n", l)
	}
}
