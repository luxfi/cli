// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chainkit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempChainYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "chain.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const validChainYAML = `
version: "1"
chain:
  name: "Test Chain"
  slug: test
  type: l1
  vm: evm
networks:
  devnet:
    networkId: 3
    chainId: 99999
    validators: 3
token:
  name: "Test Token"
  symbol: "TST"
  decimals: 18
brand:
  displayName: "Test Chain"
  domains:
    explorer: "explorer.test.network"
    rpc: "api.test.network"
services:
  node:
    enabled: true
  indexer:
    enabled: true
  explorer:
    enabled: true
  gateway:
    enabled: true
  exchange:
    enabled: false
  wallet:
    enabled: false
  faucet:
    enabled: true
    dripAmount: "1000000000000000000"
    rateLimit: "1/hour"
deploy:
  platform: hanzo
  namespace: "test-{network}"
  ingressClass: hanzo
`

func TestLoad(t *testing.T) {
	path := writeTempChainYAML(t, validChainYAML)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Chain.Name != "Test Chain" {
		t.Errorf("name = %q, want %q", cfg.Chain.Name, "Test Chain")
	}
	if cfg.Chain.Slug != "test" {
		t.Errorf("slug = %q, want %q", cfg.Chain.Slug, "test")
	}
	if cfg.Token.Symbol != "TST" {
		t.Errorf("token.symbol = %q, want %q", cfg.Token.Symbol, "TST")
	}
}

func TestLoadDefaults(t *testing.T) {
	path := writeTempChainYAML(t, validChainYAML)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	// Check defaults were applied
	if cfg.Chain.Sequencer != "lux" {
		t.Errorf("sequencer default = %q, want %q", cfg.Chain.Sequencer, "lux")
	}
	if cfg.Chain.DBType != "zapdb" {
		t.Errorf("dbType default = %q, want %q", cfg.Chain.DBType, "zapdb")
	}
	if cfg.Deploy.IngressClass != "hanzo" {
		t.Errorf("ingressClass default = %q, want %q", cfg.Deploy.IngressClass, "hanzo")
	}
	if cfg.Services.Node.Image != "ghcr.io/luxfi/node" {
		t.Errorf("node.image default = %q, want ghcr.io/luxfi/node", cfg.Services.Node.Image)
	}
	if cfg.Services.Indexer.Replicas != 1 {
		t.Errorf("indexer.replicas default = %d, want 1", cfg.Services.Indexer.Replicas)
	}
}

func TestValidate(t *testing.T) {
	path := writeTempChainYAML(t, validChainYAML)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("valid config should pass validation: %v", err)
	}
}

func TestValidateMissingSlug(t *testing.T) {
	yaml := strings.Replace(validChainYAML, "slug: test", "slug: \"\"", 1)
	path := writeTempChainYAML(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for missing slug")
	} else if !strings.Contains(err.Error(), "chain.slug") {
		t.Errorf("error should mention chain.slug: %v", err)
	}
}

func TestValidateNginxRejected(t *testing.T) {
	yaml := strings.Replace(validChainYAML, "ingressClass: hanzo", "ingressClass: nginx", 1)
	path := writeTempChainYAML(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for nginx ingressClass")
	} else if !strings.Contains(err.Error(), "nginx") {
		t.Errorf("error should mention nginx: %v", err)
	}
}

func TestValidateBadChainType(t *testing.T) {
	yaml := strings.Replace(validChainYAML, "type: l1", "type: l4", 1)
	path := writeTempChainYAML(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for bad chain type")
	}
}

func TestNamespaceFor(t *testing.T) {
	path := writeTempChainYAML(t, validChainYAML)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	ns := cfg.NamespaceFor("devnet")
	if ns != "test-devnet" {
		t.Errorf("namespace = %q, want %q", ns, "test-devnet")
	}
	ns = cfg.NamespaceFor("mainnet")
	if ns != "test-mainnet" {
		t.Errorf("namespace = %q, want %q", ns, "test-mainnet")
	}
}

func TestGenerate(t *testing.T) {
	path := writeTempChainYAML(t, validChainYAML)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}

	result, err := Generate(cfg, "devnet")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if result.Network != "devnet" {
		t.Errorf("network = %q, want devnet", result.Network)
	}
	if result.Namespace != "test-devnet" {
		t.Errorf("namespace = %q, want test-devnet", result.Namespace)
	}

	// Verify LuxNetwork CR was generated
	if result.LuxNetwork == "" {
		t.Fatal("expected LuxNetwork manifest")
	}
	if !strings.Contains(result.LuxNetwork, "kind: LuxNetwork") {
		t.Error("LuxNetwork should contain 'kind: LuxNetwork'")
	}
	if !strings.Contains(result.LuxNetwork, "networkId: 3") {
		t.Error("LuxNetwork should contain networkId: 3")
	}
	if !strings.Contains(result.LuxNetwork, "validators: 3") {
		t.Error("LuxNetwork should contain validators: 3")
	}

	// Verify LuxIndexer CR was generated
	if result.LuxIndexer == "" {
		t.Fatal("expected LuxIndexer manifest")
	}
	if !strings.Contains(result.LuxIndexer, "kind: LuxIndexer") {
		t.Error("LuxIndexer should contain 'kind: LuxIndexer'")
	}
	if !strings.Contains(result.LuxIndexer, "chainId: 99999") {
		t.Error("LuxIndexer should contain chainId: 99999")
	}

	// Verify LuxExplorer CR
	if result.LuxExplorer == "" {
		t.Fatal("expected LuxExplorer manifest")
	}
	if !strings.Contains(result.LuxExplorer, "kind: LuxExplorer") {
		t.Error("LuxExplorer should contain 'kind: LuxExplorer'")
	}
	if !strings.Contains(result.LuxExplorer, "explorer.test.network") {
		t.Error("LuxExplorer should contain explorer domain")
	}
	if !strings.Contains(result.LuxExplorer, "ingressClass: hanzo") {
		t.Error("LuxExplorer should use hanzo ingress")
	}

	// Verify LuxGateway CR
	if result.LuxGateway == "" {
		t.Fatal("expected LuxGateway manifest")
	}
	if !strings.Contains(result.LuxGateway, "kind: LuxGateway") {
		t.Error("LuxGateway should contain 'kind: LuxGateway'")
	}

	// Faucet should be generated for devnet
	if result.Faucet == "" {
		t.Fatal("expected Faucet manifest for devnet")
	}
	if !strings.Contains(result.Faucet, "DRIP_AMOUNT") {
		t.Error("Faucet should contain DRIP_AMOUNT env var")
	}

	// Exchange should NOT be generated (disabled)
	if result.Exchange != "" {
		t.Error("exchange should not be generated when disabled")
	}
}

func TestGenerateNoFaucetOnMainnet(t *testing.T) {
	yaml := strings.Replace(validChainYAML, "validators: 3\n", "validators: 3\n  mainnet:\n    networkId: 1\n    chainId: 88888\n    validators: 5\n", 1)
	path := writeTempChainYAML(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	result, err := Generate(cfg, "mainnet")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if result.Faucet != "" {
		t.Error("faucet should not be generated for mainnet")
	}
}

func TestGenerateAll(t *testing.T) {
	path := writeTempChainYAML(t, validChainYAML)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	results, err := GenerateAll(cfg)
	if err != nil {
		t.Fatalf("GenerateAll: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result (devnet only), got %d", len(results))
	}
}

func TestGeneratePrecompiles(t *testing.T) {
	yaml := validChainYAML + `
precompiles:
  - name: mldsaVerify
    blockTimestamp: 0
  - name: dexConfig
    blockTimestamp: 0
`
	path := writeTempChainYAML(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	result, err := Generate(cfg, "devnet")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.LuxNetwork, "mldsaVerify") {
		t.Error("LuxNetwork should contain mldsaVerify precompile")
	}
	if !strings.Contains(result.LuxNetwork, "dexConfig") {
		t.Error("LuxNetwork should contain dexConfig precompile")
	}
}

func TestGenerateWithKMS(t *testing.T) {
	yaml := strings.Replace(validChainYAML, "node:\n    enabled: true", `node:
    enabled: true
    stakingKms:
      hostApi: "http://kms.lux-system.svc.cluster.local/api"
      projectSlug: test-infra
      envSlug: devnet
      secretsPath: /staking`, 1)
	path := writeTempChainYAML(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	result, err := Generate(cfg, "devnet")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.LuxNetwork, "kms:") {
		t.Error("LuxNetwork should contain KMS staking config")
	}
	if !strings.Contains(result.LuxNetwork, "test-infra") {
		t.Error("LuxNetwork KMS should reference test-infra project")
	}
}

func TestGenerateBadNetwork(t *testing.T) {
	path := writeTempChainYAML(t, validChainYAML)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Generate(cfg, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent network")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	path := writeTempChainYAML(t, "not: [valid: yaml: {{")
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/chain.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}
