// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chainkit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"
)

// GenerateResult holds all generated manifests for a network.
type GenerateResult struct {
	Network   string // e.g. "mainnet", "testnet", "devnet"
	Namespace string // K8s namespace

	// CRD manifests (consumed by lux-operator)
	LuxNetwork string // LuxNetwork CR YAML
	LuxIndexer string // LuxIndexer CR YAML
	LuxExplorer string // LuxExplorer CR YAML
	LuxGateway string // LuxGateway CR YAML

	// Standard K8s manifests (for services without CRDs)
	Namespace_  string // Namespace YAML
	Exchange    string // Exchange Deployment YAML (if enabled)
	Faucet      string // Faucet Deployment YAML (if enabled)
}

// Generate produces all K8s manifests for a single network from chain.yaml.
func Generate(cfg *ChainConfig, network string) (*GenerateResult, error) {
	net, ok := cfg.Networks[network]
	if !ok {
		return nil, fmt.Errorf("network %q not defined in chain.yaml", network)
	}

	ns := cfg.NamespaceFor(network)
	result := &GenerateResult{
		Network:   network,
		Namespace: ns,
	}

	// Context passed to all templates
	ctx := &templateCtx{
		Config:    cfg,
		Network:   network,
		NetSpec:   net,
		Namespace: ns,
	}

	var err error

	// Namespace
	result.Namespace_, err = renderTemplate("namespace", tplNamespace, ctx)
	if err != nil {
		return nil, fmt.Errorf("generate namespace: %w", err)
	}

	// LuxNetwork CR
	if cfg.Services.Node.Enabled {
		ctx.GenesisJSON, err = cfg.LoadGenesisJSON()
		if err != nil {
			return nil, fmt.Errorf("load genesis: %w", err)
		}
		ctx.PrecompileUpgrades = buildPrecompileUpgrades(cfg.Precompiles)
		result.LuxNetwork, err = renderTemplate("luxnetwork", tplLuxNetwork, ctx)
		if err != nil {
			return nil, fmt.Errorf("generate LuxNetwork: %w", err)
		}
	}

	// LuxIndexer CR
	if cfg.Services.Indexer.Enabled {
		result.LuxIndexer, err = renderTemplate("luxindexer", tplLuxIndexer, ctx)
		if err != nil {
			return nil, fmt.Errorf("generate LuxIndexer: %w", err)
		}
	}

	// LuxExplorer CR
	if cfg.Services.Explorer.Enabled {
		result.LuxExplorer, err = renderTemplate("luxexplorer", tplLuxExplorer, ctx)
		if err != nil {
			return nil, fmt.Errorf("generate LuxExplorer: %w", err)
		}
	}

	// LuxGateway CR
	if cfg.Services.Gateway.Enabled {
		domain := cfg.Brand.Domains.RPC
		if domain == "" {
			domain = fmt.Sprintf("api.%s.%s.network", network, cfg.Chain.Slug)
		}
		ctx.GatewayHost = domain
		result.LuxGateway, err = renderTemplate("luxgateway", tplLuxGateway, ctx)
		if err != nil {
			return nil, fmt.Errorf("generate LuxGateway: %w", err)
		}
	}

	// Exchange Deployment (not a CRD yet)
	if cfg.Services.Exchange.Enabled {
		result.Exchange, err = renderTemplate("exchange", tplExchange, ctx)
		if err != nil {
			return nil, fmt.Errorf("generate Exchange: %w", err)
		}
	}

	// Faucet Deployment
	if cfg.Services.Faucet.Enabled && network != "mainnet" {
		result.Faucet, err = renderTemplate("faucet", tplFaucet, ctx)
		if err != nil {
			return nil, fmt.Errorf("generate Faucet: %w", err)
		}
	}

	return result, nil
}

// GenerateAll produces manifests for all networks defined in chain.yaml.
func GenerateAll(cfg *ChainConfig) ([]*GenerateResult, error) {
	var results []*GenerateResult
	for name := range cfg.Networks {
		r, err := Generate(cfg, name)
		if err != nil {
			return nil, fmt.Errorf("network %s: %w", name, err)
		}
		results = append(results, r)
	}
	return results, nil
}

// WriteManifests writes all generated manifests to an output directory.
func WriteManifests(results []*GenerateResult, outDir string) ([]string, error) {
	var files []string
	for _, r := range results {
		dir := filepath.Join(outDir, r.Network)
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
			path := filepath.Join(dir, p.name)
			files = append(files, path)
		}
	}
	return files, nil
}

// --- template context and helpers ---

type templateCtx struct {
	Config             *ChainConfig
	Network            string
	NetSpec            NetworkSpec
	Namespace          string
	GenesisJSON        json.RawMessage
	PrecompileUpgrades string
	GatewayHost        string
}

func renderTemplate(name, tpl string, ctx *templateCtx) (string, error) {
	funcMap := template.FuncMap{
		"indent": func(n int, s string) string {
			pad := strings.Repeat(" ", n)
			lines := strings.Split(s, "\n")
			for i, l := range lines {
				if l != "" {
					lines[i] = pad + l
				}
			}
			return strings.Join(lines, "\n")
		},
		"lower": strings.ToLower,
		"split": strings.Split,
		"boolDefault": func(b *bool, def bool) bool {
			if b == nil {
				return def
			}
			return *b
		},
	}

	t, err := template.New(name).Funcs(funcMap).Parse(tpl)
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("execute template %s: %w", name, err)
	}
	return buf.String(), nil
}

func buildPrecompileUpgrades(precompiles []PrecompileSpec) string {
	if len(precompiles) == 0 {
		return ""
	}
	var lines []string
	for _, p := range precompiles {
		lines = append(lines, fmt.Sprintf("      - %s: {blockTimestamp: %d}", p.Name, p.BlockTimestamp))
	}
	return strings.Join(lines, "\n")
}

// --- templates ---

const tplNamespace = `apiVersion: v1
kind: Namespace
metadata:
  name: {{.Namespace}}
  labels:
    app.kubernetes.io/part-of: {{.Config.Chain.Slug}}-network
    lux.network/chain: {{.Config.Chain.Slug}}
    lux.network/network: {{.Network}}
`

const tplLuxNetwork = `apiVersion: lux.network/v1alpha1
kind: LuxNetwork
metadata:
  name: {{.Config.Chain.Slug}}d
  namespace: {{.Namespace}}
  labels:
    app.kubernetes.io/part-of: {{.Config.Chain.Slug}}-network
    lux.network/network: {{.Network}}
spec:
  networkId: {{.NetSpec.NetworkID}}
  validators: {{.NetSpec.Validators}}
  dbType: {{.Config.Chain.DBType}}
  networkCompressionType: {{.Config.Chain.Compression}}
  image:
    repository: {{.Config.Services.Node.Image}}
{{- if .NetSpec.ImageTag}}
    tag: "{{.NetSpec.ImageTag}}"
{{- end}}
  ports:
    http: 9650
    staking: 9651
  consensus:
    sybilProtectionEnabled: {{boolDefault .NetSpec.SybilProtection true}}
  storage:
    size: "{{.Config.Services.Node.StorageSize}}"
{{- if .Config.Services.Node.StorageClass}}
    storageClass: {{.Config.Services.Node.StorageClass}}
{{- end}}
{{- if .Config.Services.Node.StakingKMS}}
  staking:
    kms:
      hostApi: {{.Config.Services.Node.StakingKMS.HostAPI}}
      projectSlug: {{.Config.Services.Node.StakingKMS.ProjectSlug}}
      envSlug: {{.Config.Services.Node.StakingKMS.EnvSlug}}
      secretsPath: {{.Config.Services.Node.StakingKMS.SecretsPath}}
{{- end}}
{{- if .NetSpec.SeedRestoreURL}}
  seedRestore:
    enabled: true
    sourceType: ObjectStore
    objectStoreUrl: "{{.NetSpec.SeedRestoreURL}}"
{{- end}}
{{- if gt .NetSpec.SnapshotInterval 0}}
  snapshotSchedule:
    enabled: true
    intervalSeconds: {{.NetSpec.SnapshotInterval}}
{{- end}}
  chainTracking:
    trackAllChains: true
  startupGate:
    onTimeout: StartAnyway
    timeoutSeconds: 30
{{- if .PrecompileUpgrades}}
  chainUpgradeConfig:
    precompileUpgrades:
{{.PrecompileUpgrades}}
{{- end}}
{{- if .GenesisJSON}}
  genesis: {{printf "%s" .GenesisJSON}}
{{- end}}
`

const tplLuxIndexer = `apiVersion: lux.network/v1alpha1
kind: LuxIndexer
metadata:
  name: indexer-{{.Config.Chain.Slug}}
  namespace: {{.Namespace}}
  labels:
    app.kubernetes.io/part-of: {{.Config.Chain.Slug}}-network
    lux.network/network: {{.Network}}
    lux.network/chain: {{.Config.Chain.Slug}}
spec:
  networkRef: {{.Config.Chain.Slug}}d
  chainAlias: "{{.Config.Chain.Slug}}"
  chainId: {{.NetSpec.ChainID}}
{{- if .NetSpec.BlockchainID}}
  blockchainId: "{{.NetSpec.BlockchainID}}"
{{- end}}
  image:
    repository: {{.Config.Services.Indexer.Image}}
    tag: {{.Config.Services.Indexer.ImageTag}}
  database:
    managed: true
    storageSize: "{{.Config.Services.Indexer.DBStorageSize}}"
  port: 4000
  replicas: {{.Config.Services.Indexer.Replicas}}
  traceEnabled: {{.Config.Services.Indexer.TraceEnabled}}
  contractVerification: {{.Config.Services.Indexer.ContractVerification}}
  pollInterval: {{.Config.Services.Indexer.PollInterval}}
  storage:
    size: "10Gi"
`

const tplLuxExplorer = `apiVersion: lux.network/v1alpha1
kind: LuxExplorer
metadata:
  name: explorer
  namespace: {{.Namespace}}
  labels:
    app.kubernetes.io/part-of: {{.Config.Chain.Slug}}-network
    lux.network/network: {{.Network}}
spec:
  networkRef: {{.Config.Chain.Slug}}d
  indexerRefs:
    - indexerName: indexer-{{.Config.Chain.Slug}}
      displayName: "{{.Config.Brand.DisplayName}}"
      default: true
{{- if .Config.Brand.PrimaryColor}}
      color: "{{.Config.Brand.PrimaryColor}}"
{{- end}}
  replicas: {{.Config.Services.Explorer.Replicas}}
  port: 3000
  ingress:
{{- if .Config.Brand.Domains.Explorer}}
    host: {{.Config.Brand.Domains.Explorer}}
{{- else}}
    host: explorer.{{.Network}}.{{.Config.Chain.Slug}}.network
{{- end}}
    ingressClass: {{.Config.Services.Explorer.IngressClass}}
  branding:
    networkName: "{{.Config.Brand.DisplayName}} ({{.Network}})"
{{- if .Config.Brand.Logo}}
    logo: "{{.Config.Brand.Logo}}"
{{- end}}
{{- if .Config.Brand.PrimaryColor}}
    primaryColor: "{{.Config.Brand.PrimaryColor}}"
{{- end}}
`

const tplLuxGateway = `apiVersion: lux.network/v1alpha1
kind: LuxGateway
metadata:
  name: api-gateway
  namespace: {{.Namespace}}
  labels:
    app.kubernetes.io/part-of: {{.Config.Chain.Slug}}-network
    lux.network/network: {{.Network}}
spec:
  networkRef: {{.Config.Chain.Slug}}d
  host: {{.GatewayHost}}
  replicas: {{.Config.Services.Gateway.Replicas}}
  port: 8080
  autoRoutes: true
  cors:
    allowedOrigins:
{{- if .Config.Services.Gateway.CORSAllowOrigins}}
{{- range $origin := split .Config.Services.Gateway.CORSAllowOrigins ","}}
      - "{{$origin}}"
{{- end}}
{{- else}}
      - "*"
{{- end}}
    allowedMethods:
      - "GET"
      - "POST"
      - "OPTIONS"
    allowedHeaders:
      - "Content-Type"
      - "Authorization"
  rateLimit:
    requestsPerSecond: {{.Config.Services.Gateway.RateLimitRPS}}
    burst: {{.Config.Services.Gateway.RateLimitBurst}}
`

const tplExchange = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.Config.Chain.Slug}}-exchange
  namespace: {{.Namespace}}
  labels:
    app.kubernetes.io/part-of: {{.Config.Chain.Slug}}-network
    app.kubernetes.io/component: exchange
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{.Config.Chain.Slug}}-exchange
  template:
    metadata:
      labels:
        app: {{.Config.Chain.Slug}}-exchange
    spec:
      containers:
        - name: exchange
          image: {{.Config.Services.Exchange.Image}}
          ports:
            - containerPort: 3000
          env:
            - name: NEXT_PUBLIC_BRAND_PACKAGE
              value: "{{.Config.Services.Exchange.BrandPackage}}"
            - name: NEXT_PUBLIC_CHAIN_ID
              value: "{{.NetSpec.ChainID}}"
---
apiVersion: v1
kind: Service
metadata:
  name: {{.Config.Chain.Slug}}-exchange
  namespace: {{.Namespace}}
spec:
  selector:
    app: {{.Config.Chain.Slug}}-exchange
  ports:
    - port: 3000
      targetPort: 3000
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{.Config.Chain.Slug}}-exchange
  namespace: {{.Namespace}}
spec:
  ingressClassName: {{.Config.Deploy.IngressClass}}
  rules:
    - host: {{.Config.Brand.Domains.Exchange}}
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: {{.Config.Chain.Slug}}-exchange
                port:
                  number: 3000
`

const tplFaucet = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.Config.Chain.Slug}}-faucet
  namespace: {{.Namespace}}
  labels:
    app.kubernetes.io/part-of: {{.Config.Chain.Slug}}-network
    app.kubernetes.io/component: faucet
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{.Config.Chain.Slug}}-faucet
  template:
    metadata:
      labels:
        app: {{.Config.Chain.Slug}}-faucet
    spec:
      containers:
        - name: faucet
          image: {{.Config.Deploy.Registry}}/faucet:latest
          ports:
            - containerPort: 8080
          env:
            - name: CHAIN_ID
              value: "{{.NetSpec.ChainID}}"
            - name: RPC_URL
              value: "http://{{.Config.Chain.Slug}}d-0.{{.Config.Chain.Slug}}d:9650/ext/bc/C/rpc"
            - name: DRIP_AMOUNT
              value: "{{.Config.Services.Faucet.DripAmount}}"
            - name: RATE_LIMIT
              value: "{{.Config.Services.Faucet.RateLimit}}"
---
apiVersion: v1
kind: Service
metadata:
  name: {{.Config.Chain.Slug}}-faucet
  namespace: {{.Namespace}}
spec:
  selector:
    app: {{.Config.Chain.Slug}}-faucet
  ports:
    - port: 8080
      targetPort: 8080
`
