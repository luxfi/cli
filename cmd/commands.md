# Lux CLI reference

_Generated from the command tree by `make docs` — do not edit by hand._

<a id="lux-ai"></a>
## lux ai

The ai command provides tools for interacting with the Lux AI network,
including chat completion, model listing, and agent execution.

Connects to a local lux-ai node or the Hanzo AI gateway (api.hanzo.ai).

ENDPOINTS:

  Local:    http://localhost:9090 (default, via lux ai serve)
  Gateway:  https://api.hanzo.ai/v1

ENVIRONMENT:

  LUX_AI_ENDPOINT   Override the default AI endpoint
  LUX_AI_API_KEY    API key for gateway authentication

**Usage:**

```bash
lux ai
```

<a id="lux-ai-agent"></a>
### lux ai agent

Run an AI agent that executes a task using the connected AI endpoint.

The agent sends the task as a system-prompted chat request with
tool-use capabilities when supported by the model.

Examples:
  lux ai agent "Deploy a new EVM chain on devnet"
  lux ai agent --model qwen3-8b "Analyze the validator set"

**Usage:**

```bash
lux ai agent [task] [flags]
```

**Flags:**

```
      --model string   Model to use (default "qwen3-8b")
```

<a id="lux-ai-chat"></a>
### lux ai chat

Send a chat message to the AI model and print the response.

Examples:
  lux ai chat "What is the Lux network?"
  lux ai chat --model qwen3-8b "Explain post-quantum cryptography"
  lux ai chat --system "You are a blockchain expert" "What is BFT?"

**Usage:**

```bash
lux ai chat [message] [flags]
```

**Flags:**

```
      --model string    Model to use (default "qwen3-8b")
      --system string   System prompt
```

<a id="lux-ai-complete"></a>
### lux ai complete

Generate a text completion from a prompt.

Examples:
  lux ai complete "The Lux blockchain uses"
  lux ai complete --model zen-coder-1.5b --max-tokens 256 "func main() {"

**Usage:**

```bash
lux ai complete [prompt] [flags]
```

**Flags:**

```
      --max-tokens int   Maximum tokens to generate (0 = model default)
      --model string     Model to use (default "qwen3-8b")
```

<a id="lux-ai-models"></a>
### lux ai models

List all models available on the connected AI endpoint.

Examples:
  lux ai models
  LUX_AI_ENDPOINT=https://api.hanzo.ai lux ai models

**Usage:**

```bash
lux ai models
```

<a id="lux-amm"></a>
## lux amm

Commands for trading on Lux Exchange AMM pools.

Supported networks:
  - lux (Lux Mainnet C-Chain, chain ID 96369)
  - zoo (Zoo Mainnet, chain ID 200200)
  - lux-testnet (Lux Testnet, chain ID 96368)

Wallet access via:
  - MNEMONIC environment variable (BIP39 mnemonic)
  - PRIVATE_KEY environment variable (hex private key)
  - --private-key flag (hex private key)

Example usage:
  lux amm balance --network zoo
  lux amm swap --network zoo --from LUX --to USDT --amount 100
  lux amm pools --network zoo
  lux amm quote --network zoo --from LUX --to USDT --amount 100
  lux amm balance --network zoo --private-key 0x...

**Usage:**

```bash
lux amm
```

**Flags:**

```
      --network string       Network: lux, zoo, or lux-testnet (default "zoo")
      --private-key string   Private key (hex) for wallet access
      --rpc string           Custom RPC endpoint (overrides network default)
```

<a id="lux-amm-balance"></a>
### lux amm balance

Display native token and ERC20 token balances.

Examples:
  lux amm balance --network zoo
  lux amm balance --network zoo --token 0x...

**Usage:**

```bash
lux amm balance [flags]
```

**Flags:**

```
      --token string   ERC20 token address to check
```

<a id="lux-amm-pools"></a>
### lux amm pools

List all liquidity pools on the AMM.

Examples:
  lux amm pools --network zoo

**Usage:**

```bash
lux amm pools
```

<a id="lux-amm-quote"></a>
### lux amm quote

Get a quote for swapping tokens without executing.
Tries V2 pools first, then V3 if no V2 pool exists.

Examples:
  lux amm quote --network zoo --from 0x... --to 0x... --amount 100
  lux amm quote --network zoo --from 0x... --to 0x... --amount 100 --v3

**Usage:**

```bash
lux amm quote [flags]
```

**Flags:**

```
      --amount float   Amount to quote
      --from string    Token address to swap from
      --to string      Token address to swap to
      --v3             Force V3 pool
```

<a id="lux-amm-status"></a>
### lux amm status

Show AMM contract status and network info.

Examples:
  lux amm status --network zoo

**Usage:**

```bash
lux amm status
```

<a id="lux-amm-swap"></a>
### lux amm swap

Swap tokens using Uniswap V2/V3 style AMM.
Tries V2 pools first, then V3 if no V2 pool exists.

Examples:
  lux amm swap --network zoo --from 0x... --to 0x... --amount 100
  lux amm swap --network zoo --from 0x... --to 0x... --amount 100 --slippage 1.0
  lux amm swap --network zoo --from 0x... --to 0x... --amount 100 --v3
  lux amm swap --network zoo --from 0x... --to 0x... --amount 100 --dry-run

**Usage:**

```bash
lux amm swap [flags]
```

**Flags:**

```
      --amount float     Amount to swap
      --dry-run          Only show quote, don't execute
      --from string      Token address to swap from
      --slippage float   Max slippage tolerance (%) (default 0.5)
      --to string        Token address to swap to
      --v3               Force V3 pool
```

<a id="lux-amm-tokens"></a>
### lux amm tokens

Get information about ERC20 tokens.

Examples:
  lux amm tokens --network zoo 0x...

**Usage:**

```bash
lux amm tokens [address...]
```

<a id="lux-call"></a>
## lux call

Asks a node what it can do, and runs one of the answers.

With no arguments it lists every service the node publishes; with a
service it lists that service's operations; with both it runs one. The
flags of an operation are the fields of its input, and its help is the
prose the handler carries — all of it read from the node, none of it
written here.

Examples:
  lux call
  lux call platform
  lux call platform get-height
  lux call --at node.lux.svc:9653 platform get-validators --net-id 8675309

**Usage:**

```bash
lux call [service] [operation] [flags]
```

**Flags:**

```
      --at string   the node to ask (a socket path, host:port, or URL) (default "/run/zip/luxd.sock")
```

<a id="lux-chain"></a>
## lux chain

The chain command provides unified operations for blockchain management.

OVERVIEW:

  The chain command suite handles the complete blockchain lifecycle from
  configuration creation through deployment and operation. It works with
  chain configurations stored in ~/.lux/chains/.

CHAIN TYPES:

  L1 (Sovereign)  - Independent validator set, own tokenomics
  L2 (Rollup)     - Based on L1 sequencing (Lux, Ethereum, etc.)
  L3 (App Chain)  - Built on L2 for application-specific use

CORE COMMANDS:

  create       Create a new blockchain configuration
  deploy       Deploy to local network, testnet, or mainnet
  list         List all configured blockchains
  describe     Show detailed blockchain information
  delete       Delete a blockchain configuration

DATA OPERATIONS:

  import       Import blocks from RLP file to running chain

NETWORK FLAGS (for deployment):

  --mainnet, -m    Deploy to mainnet (port 9630)
  --testnet, -t    Deploy to testnet (port 9640)
  --devnet, -d     Deploy to devnet (port 9650)
  --custom         Deploy to custom network

EXAMPLES:

  # Create a new L2 blockchain
  lux chain create mychain

  # Create a sovereign L1
  lux chain create mychain --type=l1

  # Deploy to local devnet
  lux chain deploy mychain --devnet

  # Deploy to testnet
  lux chain deploy mychain --testnet

  # List all configured chains
  lux chain list

  # Import historical blocks
  lux chain import c ~/work/lux/state/rlp/mainnet.rlp --mainnet

  # Delete a chain configuration
  lux chain delete mychain

TYPICAL WORKFLOW:

  1. Create configuration:  lux chain create mychain
  2. Start network:         lux network start --devnet
  3. Deploy chain:          lux chain deploy mychain --devnet
  4. Verify deployment:     lux chain list
  5. Check endpoints:       lux network status

NOTES:

  - Chain configurations are stored in ~/.lux/chains/<name>/
  - Each chain has a genesis.json and sidecar.json
  - Chains can be deployed to multiple networks (local, testnet, mainnet)
  - Use 'lux chain delete' to remove configurations
  - Network must be running before deployment

**Usage:**

```bash
lux chain
```

<a id="lux-chain-create"></a>
### lux chain create

Create a new blockchain configuration for deployment.

OVERVIEW:

  Creates a blockchain configuration with genesis file and metadata.
  The configuration is stored in ~/.lux/chains/<chainName>/ and can
  be deployed to any network (local, testnet, mainnet).

CHAIN TYPES:

  l1    Sovereign L1 with independent validation
  l2    Layer 2 rollup/chain (default)
  l3    App-specific L3 chain

SEQUENCER OPTIONS (for L2):

  lux       Lux-based rollup, 100ms blocks (default, lowest cost)
  ethereum  Ethereum-based rollup, 12s blocks (highest security)
  op        OP Stack compatible
  external  External/custom sequencer

VM OPTIONS:

  --evm          Use Lux EVM (default)
  --pars         Use Pars VM (post-quantum messaging)
  --custom-vm    Use custom VM binary
  --vm           Path to custom VM binary
  --vm-version   Specific VM version (default: latest)
  --latest       Use latest VM version

GENESIS OPTIONS:

  --genesis           Path to custom genesis.json file
                      If not provided, generates default EVM genesis
  --evm-chain-id      EVM chain ID (default: 200200)
  --token-name        Native token name (default: TOKEN)
  --token-symbol      Native token symbol (default: TKN)
  --airdrop-address   Address to airdrop tokens to (default: test account)
  --airdrop-amount    Amount to airdrop in wei (default: 1000000000000000000000000)

NON-INTERACTIVE MODE:

  Non-interactive mode is automatically enabled when:
    - NON_INTERACTIVE=1 environment variable is set
    - CI=1 environment variable is set (common in CI/CD pipelines)
    - stdin is not a TTY (piped input, scripts, etc.)

  In non-interactive mode, sensible defaults are used for optional values.
  Required values must be provided via flags.

OTHER OPTIONS:

  --force, -f              Overwrite existing configuration
  --enable-preconfirm      Enable pre-confirmations (<100ms acknowledgment)

EXAMPLES:

  # Create default L2 chain with Lux sequencing
  lux chain create mychain

  # Create sovereign L1
  lux chain create mychain --type=l1

  # Create with Ethereum sequencing (12s blocks)
  lux chain create mychain --sequencer=ethereum

  # Create with custom genesis
  lux chain create mychain --genesis=~/custom-genesis.json

  # Create L3 on existing L2
  lux chain create myapp --type=l3

  # Overwrite existing configuration
  lux chain create mychain --force

  # Create with pre-confirmations enabled
  lux chain create mychain --enable-preconfirm

  # Non-interactive in CI/CD (env var triggers non-interactive mode)
  CI=1 lux chain create mychain

  # Non-interactive with custom chain ID
  NON_INTERACTIVE=1 lux chain create mychain --evm-chain-id=12345

  # Piped input also triggers non-interactive mode
  echo "" | lux chain create mychain --evm-chain-id=12345

OUTPUT:

  Creates two files in ~/.lux/chains/<chainName>/:
  - genesis.json    Blockchain genesis configuration
  - sidecar.json    Metadata (VM type, versions, deployment info)

NEXT STEPS:

  After creating a chain configuration:
  1. Start a network:     lux network start --devnet
  2. Deploy the chain:    lux chain deploy mychain --devnet
  3. Verify deployment:   lux network status

NOTES:

  - Chain names must be unique and ≤32 characters
  - Reserved names: c, p, x, primary, platform
  - Default genesis includes funded test account
  - Genesis can be customized after creation

**Usage:**

```bash
lux chain create [chainName] [flags]
```

**Flags:**

```
      --airdrop-address string   Address to airdrop tokens to
      --airdrop-amount string    Amount to airdrop in wei
      --custom                   Target custom network
      --custom-vm                Use custom VM
  -d, --devnet                   Target devnet
      --enable-preconfirm        Enable pre-confirmations
      --evm                      Use Lux EVM
      --evm-chain-id uint        EVM chain ID (default: 200200)
  -f, --force                    Overwrite existing configuration
      --genesis string           Path to custom genesis file
      --latest                   Use latest VM version
  -m, --mainnet                  Target mainnet
      --pars                     Use Pars VM (post-quantum messaging)
      --sequencer string         Sequencer: lux, ethereum, op, external (default "lux")
  -t, --testnet                  Target testnet
      --token-name string        Native token name (default: TOKEN)
      --token-symbol string      Native token symbol (default: TKN)
      --type string              Chain type: l1, l2, l3 (default "l2")
      --vm string                Path to custom VM binary
      --vm-version string        VM version to use
```

<a id="lux-chain-delete"></a>
### lux chain delete

Delete a blockchain configuration

**Usage:**

```bash
lux chain delete [chainName] [flags]
```

**Flags:**

```
      --custom    Target custom network
  -d, --devnet    Target devnet
  -f, --force     Skip confirmation prompt (required in non-interactive mode)
  -m, --mainnet   Target mainnet
  -t, --testnet   Target testnet
```

<a id="lux-chain-deploy"></a>
### lux chain deploy

Deploy a configured blockchain to the network.

OVERVIEW:

  Deploys a blockchain configuration to a running network. The blockchain
  must be created first with 'lux chain create'. The target network must
  be running before deployment.

NETWORK FLAGS (choose one):

  --mainnet, -m    Deploy to mainnet (port 9630, Network ID 1)
  --testnet, -t    Deploy to testnet (port 9640, Network ID 2)
  --devnet, -d     Deploy to devnet (port 9650, Network ID 3)
  --local, -l      Deploy to local/custom network

  Default: --local (deploys to custom/local network)

PREREQUISITES:

  1. Chain must be created:
     lux chain create mychain

  2. For local networks, network must be running:
     lux network start --devnet

  3. For remote networks (devnet, testnet, mainnet), a funded key is needed:
     Set MNEMONIC or PRIVATE_KEY env var, or use --key flag

  4. VM must be installed (for custom VMs):
     lux vm link "Lux EVM" --path ~/work/lux/evm/build/evm

OPTIONS:

  --node-version   Specific luxd version to use (default: latest)
  --key            Key name for remote network deployment (from ~/.lux/keys/)

EXAMPLES:

  # Deploy to remote devnet (auto-detects remote endpoint)
  lux chain deploy mychain --devnet

  # Deploy to remote devnet with specific key
  lux chain deploy mychain --devnet --key mykey

  # Deploy to local devnet (if local network is running)
  lux chain deploy mychain --devnet

  # Deploy to testnet
  lux chain deploy mychain --testnet
  lux chain deploy mychain -t

  # Deploy to mainnet
  lux chain deploy mychain --mainnet
  lux chain deploy mychain -m

  # Deploy with specific node version
  lux chain deploy mychain --devnet --node-version v1.11.0

DEPLOYMENT PROCESS:

  Local network:
  1. Validates chain configuration exists
  2. Verifies local gRPC network is running
  3. Checks VM plugin is installed
  4. Creates blockchain via netrunner gRPC
  5. Updates sidecar with deployment info

  Remote network:
  1. Validates chain configuration exists
  2. Probes remote endpoint (e.g., https://api.lux-dev.network)
  3. Creates chain on P-chain via wallet transaction
  4. Creates blockchain on P-chain via wallet transaction
  5. Updates sidecar with deployment info

OUTPUT:

  On success, displays:
  - Blockchain ID
  - Chain ID
  - RPC endpoints

TROUBLESHOOTING:

  "Network not running" → Start network first:
    lux network start --devnet

  "Chain mychain not found" → Create chain first:
    lux chain create mychain

  "VM not installed" → Link VM binary:
    lux vm link "Lux EVM" --path ~/path/to/evm

  "RPC version mismatch" → Chain VM version incompatible with running node

NOTES:

  - Deployment info is saved to the chain's sidecar.json
  - Same chain can be deployed to multiple networks
  - Each deployment gets unique blockchain ID
  - Use 'lux network status' to see deployed chain endpoints

**Usage:**

```bash
lux chain deploy [chainName] [flags]
```

**Flags:**

```
  -d, --devnet                Deploy to devnet
      --key string            Key name for remote network deployment (from ~/.lux/keys/)
  -l, --local                 Deploy to local/custom network
  -m, --mainnet               Deploy to mainnet
      --node-version string   Node version to use (default "latest")
  -t, --testnet               Deploy to testnet
      --timeout duration      Maximum time to wait for chain deployment (e.g., 60s, 2m) (default 30s)
```

<a id="lux-chain-describe"></a>
### lux chain describe

Show detailed information about a blockchain

**Usage:**

```bash
lux chain describe [chainName] [flags]
```

**Flags:**

```
      --custom    Target custom network
  -d, --devnet    Target devnet
  -m, --mainnet   Target mainnet
  -t, --testnet   Target testnet
```

<a id="lux-chain-import"></a>
### lux chain import

Import blocks from an RLP-encoded file to a running chain.

OVERVIEW:

  Imports historical blockchain data from RLP files into a running chain.
  This is useful for bootstrapping chains with existing state or syncing
  from canonical snapshots.

  Uses the admin_importChain RPC method. The network must be running and
  the admin API must be enabled (default when started via CLI).

CHAIN IDENTIFIERS:

  c, C         C-Chain (primary EVM chain)
  <name>       Chain name (looks up blockchain ID from sidecar)
  <blockchainID>  Direct blockchain ID

NETWORK FLAGS (auto-detects port):

  --mainnet, -m    Import to mainnet chain (port 9630)
  --testnet, -t    Import to testnet chain (port 9640)
  --devnet, -d     Import to devnet chain (port 9650)

  Default: auto-detects running network or uses custom (port 9660)

OPTIONS:

  --rpc <url>      Custom RPC endpoint (overrides network flag)

PREREQUISITES:

  1. Network must be running:
     lux network start --mainnet

  2. RLP file must exist and be readable by the node

EXAMPLES:

  # Import C-Chain mainnet blocks
  lux chain import c ~/work/lux/state/rlp/lux-mainnet-96369.rlp --mainnet

  # Import to custom chain on devnet
  lux chain import zoo ~/work/lux/state/rlp/zoo-mainnet-200200.rlp --devnet

  # Import with custom RPC endpoint
  lux chain import c blocks.rlp --rpc http://localhost:9630/v1/chain/C/rpc

  # Import to blockchain by ID
  lux chain import 2ebCneCbwthjQ1rYT41nhd7M76Hc6YmosMAQrTFhBq8qeqh6tt blocks.rlp --mainnet

RLP FILE LOCATIONS:

  Canonical RLP files are stored in:
    ~/work/lux/state/rlp/<network>/<chain>-<chainid>.rlp

  Examples:
    ~/work/lux/state/rlp/lux-mainnet/lux-mainnet-96369.rlp
    ~/work/lux/state/rlp/zoo-mainnet/zoo-mainnet-200200.rlp

IMPORT PROCESS:

  1. Validates file exists
  2. Detects or connects to RPC endpoint
  3. Gets current block height
  4. Calls admin_importChain with file path
  5. Monitors import progress
  6. Reports final block height and import rate

OUTPUT:

  Import complete!
    Blocks imported: 1082780
    Final height: 1082780
    Time: 45m12s
    Rate: 399.2 blocks/sec

TROUBLESHOOTING:

  "Network not running" → Start network first:
    lux network start --mainnet

  "RPC connection refused" → Check network is running:
    lux network status

  "File not found" → Use absolute path or verify file exists

  "Import timeout" → Import continues in background, check node logs

NOTES:

  - Import runs asynchronously - RPC may timeout but import continues
  - Large imports (1M+ blocks) can take 30min - 2hrs depending on hardware
  - The node must have read access to the RLP file
  - Genesis config must match the RLP file exactly for successful import
  - Use 'lux chain export' to create RLP files from running chains

**Usage:**

```bash
lux chain import <chain> <path> [flags]
```

**Flags:**

```
      --custom       Target custom network
  -d, --devnet       Target devnet
  -m, --mainnet      Target mainnet
      --rpc string   Custom RPC endpoint (default: auto-detected)
  -t, --testnet      Target testnet
```

<a id="lux-chain-launch"></a>
### lux chain launch

Launch a complete blockchain ecosystem from a single chain.yaml configuration.

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
  - All secrets are referenced via KMS, never stored in manifests

**Usage:**

```bash
lux chain launch <chain.yaml> [flags]
```

**Flags:**

```
      --apply            Apply generated manifests to the cluster via kubectl
      --dry-run          Generate manifests without applying
      --network string   Target specific network (mainnet, testnet, devnet)
  -o, --output string    Output directory for generated manifests
      --service string   Generate only specific service (node, indexer, explorer, gateway, exchange, faucet)
```

<a id="lux-chain-list"></a>
### lux chain list

List all configured blockchains with their details.

OVERVIEW:

  Displays a table of all blockchain configurations stored in ~/.lux/chains/.
  Shows configuration details and deployment status across networks.

OUTPUT COLUMNS:

  Name        Blockchain configuration name
  Type        Chain type (L1, L2, L3)
  Chain ID    EVM chain ID
  VM          Virtual machine type (EVM, CustomVM)
  Sequencer   Sequencer type (lux, ethereum, op)
  Deployed    Whether chain is deployed to any network

EXAMPLES:

  # List all configured chains
  lux chain list

TYPICAL OUTPUT:

  +----------+------+----------+-----+-----------+----------+
  | NAME     | TYPE | CHAIN ID | VM  | SEQUENCER | DEPLOYED |
  +----------+------+----------+-----+-----------+----------+
  | mychain  | L2   | 200200   | EVM | lux       | Yes      |
  | testnet  | L1   | 36911    | EVM | lux       | No       |
  +----------+------+----------+-----+-----------+----------+

NOTES:

  - Only shows chains with valid configurations
  - "Deployed: Yes" means chain is deployed to at least one network
  - Use 'lux chain describe <name>' for detailed chain information
  - Use 'lux network status' to see endpoints of deployed chains

**Usage:**

```bash
lux chain list [flags]
```

**Flags:**

```
      --custom    Target custom network
  -d, --devnet    Target devnet
  -m, --mainnet   Target mainnet
  -t, --testnet   Target testnet
```

<a id="lux-chain-upgrade"></a>
### lux chain upgrade

The blockchain upgrade command suite provides a collection of tools for
updating your developmental and deployed Blockchains.

**Usage:**

```bash
lux chain upgrade
```

<a id="lux-chain-upgrade-apply"></a>
#### lux chain upgrade apply

Apply generated upgrade bytes to running Blockchain nodes to trigger a network upgrade.

For public networks (Testnet or Mainnet), to complete this process,
you must have access to the machine running your validator.
If the CLI is running on the same machine as your validator, it can manipulate your node's
configuration automatically. Alternatively, the command can print the necessary instructions
to upgrade your node manually.

After you update your validator's configuration, you need to restart your validator manually.
If you provide the --luxd-chain-config-dir flag, this command attempts to write the upgrade file at that path.
Refer to https://docs.lux.network/nodes/maintain/chain-config-flags#chain-chain-configs for related documentation.

In non-interactive mode (CI/scripts), use --force to skip confirmation prompts for
timestamps in the past. The --luxd-chain-config-dir defaults to ~/.luxd/chains and
will be used without confirmation prompts.

Examples:
  # Interactive mode
  lux blockchain upgrade apply mychain --local

  # Non-interactive mode with custom config directory
  lux blockchain upgrade apply mychain --testnet --luxd-chain-config-dir /path/to/chains --force

  # Print manual instructions (non-interactive friendly)
  lux blockchain upgrade apply mychain --mainnet --print

**Usage:**

```bash
lux chain upgrade apply [blockchainName] [flags]
```

**Flags:**

```
      --config                         Create upgrade config for future chain deployments (same as generate)
  -f, --force                          Skip confirmation prompts (e.g., for timestamps in the past)
      --local                          Apply upgrade to existing local deployment
      --luxd-chain-config-dir string   Luxd chain config directory (e.g., ~/.luxd/chains) (default "/Users/z/.luxd/chains")
      --mainnet                        Apply upgrade to existing mainnet deployment
      --print                          Print manual config instructions (for public networks only, non-interactive friendly)
      --testnet                        Apply upgrade to existing testnet deployment
```

<a id="lux-chain-upgrade-export"></a>
#### lux chain upgrade export

Export the upgrade bytes file to a location of choice on disk.

In non-interactive mode (CI/scripts), use --output to specify the file path
and --force to overwrite existing files without confirmation.

Examples:
  # Interactive mode (prompts for path)
  lux blockchain upgrade export mychain

  # Non-interactive mode
  lux blockchain upgrade export mychain --output ./upgrade.json --force

**Usage:**

```bash
lux chain upgrade export [blockchainName] [flags]
```

**Flags:**

```
  -f, --force           Overwrite existing file without confirmation
  -o, --output string   Output file path for upgrade bytes (required in non-interactive mode)
```

<a id="lux-chain-upgrade-generate"></a>
#### lux chain upgrade generate

The blockchain upgrade generate command builds a new upgrade.json file to customize your Blockchain.
It guides the user through the process using an interactive wizard.

IMPORTANT: This command requires interactive mode (TTY) due to the complexity of precompile
configuration. For non-interactive/CI environments, create the upgrade.json file manually
or use 'lux blockchain upgrade import' to import a pre-created configuration.

Use --yes/-y to skip the initial warning confirmation when running interactively.

Examples:
  # Interactive mode (wizard)
  lux blockchain upgrade generate mychain

  # Skip initial warning
  lux blockchain upgrade generate mychain --yes

  # For CI/non-interactive: import a pre-created upgrade file instead
  lux blockchain upgrade import mychain --upgrade-filepath ./upgrade.json

**Usage:**

```bash
lux chain upgrade generate [blockchainName] [flags]
```

**Flags:**

```
  -y, --yes   Skip initial warning confirmation prompt
```

<a id="lux-chain-upgrade-import"></a>
#### lux chain upgrade import

Import the upgrade bytes file into the local environment

**Usage:**

```bash
lux chain upgrade import [blockchainName] [flags]
```

**Flags:**

```
      --upgrade-filepath string   Import upgrade bytes file into local environment
```

<a id="lux-chain-upgrade-print"></a>
#### lux chain upgrade print

Print the upgrade.json file content

**Usage:**

```bash
lux chain upgrade print [blockchainName]
```

<a id="lux-chain-upgrade-vm"></a>
#### lux chain upgrade vm

The blockchain upgrade vm command enables the user to upgrade their Blockchain's VM binary. The command
can upgrade both local Blockchains and publicly deployed Blockchains on Testnet and Mainnet.

The command walks the user through an interactive wizard. The user can skip the wizard by providing
command line flags.

**Usage:**

```bash
lux chain upgrade vm [blockchainName] [flags]
```

**Flags:**

```
      --binary string       Upgrade to custom binary
      --config              upgrade config for future chain deployments
      --latest              upgrade to latest version
      --local local         upgrade existing local deployment
      --mainnet mainnet     upgrade existing mainnet deployment
      --plugin-dir string   plugin directory to automatically upgrade VM
      --print               print instructions for upgrading
      --testnet testnet     upgrade existing testnet deployment (alias for `testnet`)
      --version string      Upgrade to custom version
```

<a id="lux-config"></a>
## lux config

Customize configuration for Lux CLI

**Usage:**

```bash
lux config
```

<a id="lux-config-lint"></a>
### lux config lint

Validate a luxd configuration file for errors.

Reports:
  - Unknown configuration keys (with typo suggestions)
  - Invalid value types (e.g., "abc" for a duration)
  - Deprecated keys (with replacement hints)

Uses the authoritative flag spec from github.com/luxfi/config/spec,
which is generated from the node's source of truth.

Example:
  lux config lint myconfig.json

**Usage:**

```bash
lux config lint <config-file.json>
```

<a id="lux-config-metrics"></a>
### lux config metrics

set user metrics collection preferences

**Usage:**

```bash
lux config metrics [enable | disable]
```

<a id="lux-contract"></a>
## lux contract

The contract command suite provides a collection of tools for deploying
and interacting with smart contracts on Lux networks.

**Usage:**

```bash
lux contract
```

<a id="lux-contract-deploy"></a>
### lux contract deploy

The contract command suite provides a collection of tools for deploying
smart contracts on Lux networks.

**Usage:**

```bash
lux contract deploy
```

<a id="lux-contract-deploy-erc20"></a>
#### lux contract deploy erc20

Deploy an ERC20 token into a given Network and Blockchain.

The command deploys a standard ERC20 token contract with the specified
symbol, initial supply, and recipient address for the minted tokens.

Examples:
  # Interactive mode (prompts for missing values)
  lux contract deploy erc20

  # Non-interactive mode (all flags required)
  lux contract deploy erc20 --symbol USDC --supply 1000000 \
    --funded 0x1234...abcd --private-key-file ./key.txt \
    --c-chain --mainnet

  # Deploy to a specific blockchain
  lux contract deploy erc20 --symbol LUX --supply 100000000 \
    --funded 0xYourAddress --blockchain-id <ID> --testnet

**Usage:**

```bash
lux contract deploy erc20 [flags]
```

**Flags:**

```
      --blockchain string      deploy the ERC20 contract into the given CLI blockchain
      --blockchain-id string   deploy the ERC20 contract into the given blockchain ID/Alias
      --c-chain                deploy the ERC20 contract into C-Chain
      --funded string          address to receive the initial token supply (0x...)
      --genesis-key            use genesis allocated key as contract deployer
      --key string             CLI stored key to use as contract deployer
      --private-key string     private key to use as contract deployer
      --rpc string             RPC endpoint URL (auto-detected if not specified)
      --supply uint            total token supply to mint
      --symbol string          token symbol (e.g., USDC, LUX)
```

<a id="lux-contract-deploy-l2"></a>
#### lux contract deploy l2

Deploy the canonical lux/standard contract stack (Safe + Bridge + Exchange +
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
--i-know-this-is-real-money.

**Usage:**

```bash
lux contract deploy l2 [flags]
```

**Flags:**

```
      --brand string                restrict to a single brand (default: all in inventory)
      --confirm                     broadcast instead of dry-run
      --deployer-index uint         BIP44 mnemonic index for the deployer key
      --env string                  mainnet|testnet|devnet
      --i-know-this-is-real-money   mainnet broadcast safeguard
      --inventory string            inventory JSON path
      --kms-fetch string            kms-fetch binary path (default: ~/work/hanzo/kms/cmd/kms-fetch/kms-fetch)
      --liquid                      after standard succeeds, also deploy lux/liquid
      --liquid-repo string          lux/liquid repo root (default: ~/work/lux/liquid)
      --liquid-script string        forge script in lux/liquid (default: script/DeployL2.s.sol)
      --repo string                 lux/standard repo root (default: ~/work/lux/standard)
      --resume                      pass --resume to forge to continue a partial broadcast
      --script string               forge script (default: contracts/script/DeployMultiNetwork.s.sol)
```

<a id="lux-contract-initValidatorManager"></a>
### lux contract initValidatorManager

Initializes Proof of Authority(PoA) or Proof of Stake(PoS)Validator Manager contract on a Blockchain and sets up initial validator set on the Blockchain. For more info on Validator Manager, please head to https://github.com/luxfi/warp-contracts/tree/main/contracts/validator-manager

**Usage:**

```bash
lux contract initValidatorManager blockchainName [flags]
```

**Flags:**

```
      --genesis-key                            use genesis allocated key as contract deployer
      --key string                             CLI stored key to use as contract deployer
      --pos-maximum-stake-amount uint          (PoS only) maximum stake amount (default 1000)
      --pos-maximum-stake-multiplier uint8     (PoS only )maximum stake multiplier (default 1)
      --pos-minimum-delegation-fee uint16      (PoS only) minimum delegation fee (default 1)
      --pos-minimum-stake-amount uint          (PoS only) minimum stake amount (default 1)
      --pos-minimum-stake-duration uint        (PoS only) minimum stake duration (in seconds) (default 86400)
      --pos-reward-calculator-address string   (PoS only) initialize the ValidatorManager with reward calculator address
      --pos-weight-to-value-factor uint        (PoS only) weight to value factor (default 1)
      --private-key string                     private key to use as contract deployer
      --rpc string                             blockchain rpc endpoint
```

<a id="lux-ctx"></a>
## lux ctx

Scans $LUX_NETWORK_PATH (default ~/work/{lux,zoo,hanzo,pars,osage,adnexus}/universe)
for chain.yaml files and prints every (network, env) tuple they declare.

**Usage:**

```bash
lux ctx
```

<a id="lux-cycle"></a>
## lux cycle

Atomically boots the network's L1, deploys the standard contract
suite, stops the node cleanly, and writes a tar.zst snapshot.

Pipeline:
  1. lux up    <ref>           (background within this process)
  2. wait healthy (networkID match + C-chain responding)
  3. lux deploy <ref>          (forge against the local RPC)
  4. lux down  <ref>           (SIGTERM + wait drain)
  5. lux snap  <ref>           (tar.zst the data-dir)

Examples:
  lux cycle zoo/localnet
  lux cycle lux/devnet --skip-deploy   # boot, stop, snap only
  lux cycle hanzo/testnet --skip-snap

**Usage:**

```bash
lux cycle <network>/<env> [flags]
```

**Flags:**

```
      --mnemonic string    BIP44 mnemonic for the deployer (defaults to $LUX_MNEMONIC)
      --node-path string   path to luxd binary
      --skip-deploy        do not run contract deploy stage
      --skip-snap          do not run snapshot stage
```

<a id="lux-dev"></a>
## lux dev

The dev command provides local development environment tools.

This runs a single-node Lux network with K=1 consensus for instant
block finality. All chains (C/P/X) are enabled with full validator
signing capabilities.

Commands:
  start   - Start local dev node (default port 8545)
  stop    - Stop the dev node

Features:
  • K=1 consensus (instant finality, no validator sampling)
  • Full validator signing for all chains
  • Compatible with Hardhat/Foundry/Anvil tooling
  • Test accounts pre-funded in genesis

**Usage:**

```bash
lux dev
```

<a id="lux-dev-stack"></a>
### lux dev stack

Manage a multi-app local development stack.

The stack runs luxd (one or more nodes) plus companion apps:
explorer, bridge, exchange, safe, dao, wallet, faucet.

Config lives at ~/.lux/dev/stack.yaml and is auto-created on first run.

Examples:
  lux dev stack up                # Start stack with defaults
  lux dev stack up --chains 3     # Start 3 luxd nodes + apps
  lux dev stack down              # Graceful shutdown
  lux dev stack status            # Show running processes
  lux dev stack logs explorer     # Tail explorer logs

**Usage:**

```bash
lux dev stack
```

<a id="lux-dev-stack-down"></a>
#### lux dev stack down

Gracefully stop all running stack processes. Sends SIGTERM, waits 10s, then SIGKILL.

**Usage:**

```bash
lux dev stack down
```

<a id="lux-dev-stack-logs"></a>
#### lux dev stack logs

Tail the log file for a stack application.

The app name can be a base name (e.g., "explorer") which tails instance 0,
or a full instance name (e.g., "explorer-1") for a specific chain instance.

**Usage:**

```bash
lux dev stack logs <app>
```

<a id="lux-dev-stack-status"></a>
#### lux dev stack status

Display a table of all stack processes with PID, port, state, and uptime.

**Usage:**

```bash
lux dev stack status
```

<a id="lux-dev-stack-up"></a>
#### lux dev stack up

Start all enabled apps in the dev stack.

luxd nodes start first and must pass health checks before companion
apps are launched. Port deconfliction for multi-chain: chain i gets
ports at port_base + 100*i.

**Usage:**

```bash
lux dev stack up [flags]
```

**Flags:**

```
      --chains int      number of luxd nodes (overrides stack.yaml)
      --config string   path to stack.yaml (default: ~/.lux/dev/stack.yaml)
```

<a id="lux-dev-start"></a>
### lux dev start

Start a single-node Lux development network.

The dev node uses K=1 consensus for instant block finality without
validator sampling. All chains are enabled with full validator signing:
  • C-Chain: EVM-compatible smart contracts
  • P-Chain: Platform staking and validation
  • X-Chain: UTXO-based asset exchange
  • T-Chain: Threshold FHE operations

Default port is 8545 (Anvil-compatible) so it works seamlessly with
Hardhat, Foundry, and other Ethereum tooling.

FHE Support:
  The T-Chain provides threshold homomorphic encryption for confidential
  smart contracts. Use FHE precompiles at 0x0200...0080 or the @luxfi/fhe SDK.

DEX / D-Chain:
  The public default luxd does NOT include the DEX D-Chain. To launch the
  dchain-tagged luxd (which bakes D-Chain on localnet 1337) pass:
    lux dev start --build-tags dchain
  This resolves the dchain-tagged binary and points --plugin-dir at the plugin
  directory so the EVM plugin subprocess is found deterministically.

Examples:
  lux dev start                    # Start on default port 8545
  lux dev start --port 9650        # Start on custom port
  lux dev start --automine 1s      # Mine blocks every 1 second
  lux dev start --automine 500ms   # Mine blocks every 500ms
  lux dev start --build-tags dchain # Start the D-Chain-enabled (dexvm) node

**Usage:**

```bash
lux dev start [flags]
```

**Flags:**

```
      --automine string       auto-mine interval (e.g., '1s', '500ms'); empty = mine as blocks arrive
      --build-tags string     luxd build-tag selector; 'dchain' launches the D-Chain-enabled (dexvm) node
      --clean                 clean state before starting (fresh genesis)
      --data-dir string       luxd data-dir (default ~/.lux/devnet)
      --genesis-file string   genesis file path (uses luxd embedded if empty)
      --log-level string      log level (debug, info, warn, error) (default "info")
      --network-id uint32     sovereign-L1 networkID (override 1337 default) (default 1337)
      --node-path string      path to luxd binary (auto-detected if not set)
      --plugin-dir string     VM plugin directory passed to luxd (default: luxd's own ~/.lux/plugins/current)
      --port int              HTTP port for RPC (Anvil-compatible default) (default 8545)
```

<a id="lux-dev-stop"></a>
### lux dev stop

Stops the dev node that `lux dev start` writes to its default
data-dir (~/.lux/devnet). For sovereign-L1 nodes booted via
`lux up <brand>/<env>`, use `lux down <brand>/<env>` instead.

**Usage:**

```bash
lux dev stop
```

<a id="lux-dex"></a>
## lux dex

Commands for interacting with Lux DEX - a high-performance
decentralized exchange with spot trading, AMM pools, and perpetual futures.

Features:
  - Central Limit Order Book (CLOB) for spot trading
  - AMM pools (Constant Product, StableSwap, Concentrated Liquidity)
  - Perpetual futures with up to 100x leverage
  - Cross-chain swaps via Warp messaging
  - 1ms block times for ultra-low latency HFT

Example usage:
  lux dex market list              # List all markets
  lux dex order place              # Place an order
  lux dex pool create              # Create liquidity pool
  lux dex perp open                # Open perpetual position

**Usage:**

```bash
lux dex
```

<a id="lux-dex-account"></a>
### lux dex account

Commands for managing your DEX trading account, deposits, and withdrawals

<a id="lux-dex-account-balance"></a>
#### lux dex account balance

View account balances

**Usage:**

```bash
lux dex account balance
```

<a id="lux-dex-account-deposit"></a>
#### lux dex account deposit

Deposit funds to trading account

**Usage:**

```bash
lux dex account deposit [flags]
```

**Flags:**

```
      --amount float   Amount to deposit
      --token string   Token to deposit
```

<a id="lux-dex-account-history"></a>
#### lux dex account history

View transaction history

**Usage:**

```bash
lux dex account history
```

<a id="lux-dex-account-withdraw"></a>
#### lux dex account withdraw

Withdraw funds from trading account

**Usage:**

```bash
lux dex account withdraw [flags]
```

**Flags:**

```
      --amount float   Amount to withdraw
      --token string   Token to withdraw
```

<a id="lux-dex-market"></a>
### lux dex market

Commands for listing, creating, and managing trading markets

<a id="lux-dex-market-create"></a>
#### lux dex market create

Create a new spot or perpetual market with specified parameters

**Usage:**

```bash
lux dex market create
```

<a id="lux-dex-market-info"></a>
#### lux dex market info

Display detailed information about a specific market including orderbook depth, recent trades, and statistics

**Usage:**

```bash
lux dex market info [symbol]
```

<a id="lux-dex-market-list"></a>
#### lux dex market list

Display all spot and perpetual markets with current prices and volume

**Usage:**

```bash
lux dex market list
```

<a id="lux-dex-order"></a>
### lux dex order

Commands for placing, cancelling, and viewing orders

<a id="lux-dex-order-cancel"></a>
#### lux dex order cancel

Cancel an order

**Usage:**

```bash
lux dex order cancel [order-id]
```

<a id="lux-dex-order-history"></a>
#### lux dex order history

View order history

**Usage:**

```bash
lux dex order history
```

<a id="lux-dex-order-list"></a>
#### lux dex order list

List open orders

**Usage:**

```bash
lux dex order list
```

<a id="lux-dex-order-place"></a>
#### lux dex order place

Place a limit or market order on a trading pair.

Examples:
  lux dex order place --market LUX/USDT --side buy --type limit --price 10.50 --amount 100
  lux dex order place --market BTC/USDT --side sell --type market --amount 0.5

**Usage:**

```bash
lux dex order place [flags]
```

**Flags:**

```
      --amount float    Order amount
      --market string   Trading pair symbol (e.g., LUX/USDT)
      --price float     Limit price (required for limit orders)
      --side string     Order side: buy or sell
      --tif string      Time in force: gtc, ioc, fok (default "gtc")
      --type string     Order type: limit or market (default "limit")
```

<a id="lux-dex-perp"></a>
### lux dex perp

Commands for trading perpetual futures contracts.

Features:
  - Up to 100x leverage
  - Cross and isolated margin modes
  - Automatic liquidation protection
  - 8-hour funding rate intervals

Similar to Hyperliquid and GMX perpetual trading.

<a id="lux-dex-perp-close"></a>
#### lux dex perp close

Close a perpetual position

**Usage:**

```bash
lux dex perp close [market] [flags]
```

**Flags:**

```
      --percent float   Percentage of position to close (0-100) (default 100)
```

<a id="lux-dex-perp-funding"></a>
#### lux dex perp funding

View funding rate information

**Usage:**

```bash
lux dex perp funding
```

<a id="lux-dex-perp-markets"></a>
#### lux dex perp markets

List perpetual markets

**Usage:**

```bash
lux dex perp markets
```

<a id="lux-dex-perp-open"></a>
#### lux dex perp open

Open a new perpetual futures position.

Examples:
  lux dex perp open --market BTC-PERP --side long --size 0.1 --leverage 10
  lux dex perp open --market ETH-PERP --side short --size 1 --leverage 5 --margin isolated

**Usage:**

```bash
lux dex perp open [flags]
```

**Flags:**

```
      --leverage uint16   Leverage multiplier (1-100) (default 10)
      --margin string     Margin mode: cross or isolated (default "cross")
      --market string     Perpetual market symbol (e.g., BTC-PERP)
      --side string       Position side: long or short
      --size float        Position size in base units
```

<a id="lux-dex-perp-pnl"></a>
#### lux dex perp pnl

View profit/loss summary

**Usage:**

```bash
lux dex perp pnl
```

<a id="lux-dex-perp-positions"></a>
#### lux dex perp positions

List open positions

**Usage:**

```bash
lux dex perp positions
```

<a id="lux-dex-pool"></a>
### lux dex pool

Commands for creating, managing, and interacting with AMM liquidity pools

<a id="lux-dex-pool-add"></a>
#### lux dex pool add

Add liquidity to a pool

**Usage:**

```bash
lux dex pool add [pool-id] [flags]
```

**Flags:**

```
      --amount0 float   Amount of token0 to add
      --amount1 float   Amount of token1 to add
```

<a id="lux-dex-pool-create"></a>
#### lux dex pool create

Create a new AMM liquidity pool.

Pool types:
  - constant-product: Standard x*y=k AMM (like Uniswap V2)
  - stableswap: Optimized for stable pairs (like Curve)
  - concentrated: Concentrated liquidity (like Uniswap V3)

Examples:
  lux dex pool create --token0 LUX --token1 USDT --amount0 1000 --amount1 10000 --type constant-product --fee 30

**Usage:**

```bash
lux dex pool create [flags]
```

**Flags:**

```
      --amount0 float   Initial amount of token0
      --amount1 float   Initial amount of token1
      --fee uint16      Fee in basis points (30 = 0.3%) (default 30)
      --token0 string   First token symbol
      --token1 string   Second token symbol
      --type string     Pool type: constant-product, stableswap, concentrated (default "constant-product")
```

<a id="lux-dex-pool-list"></a>
#### lux dex pool list

List all liquidity pools

**Usage:**

```bash
lux dex pool list
```

<a id="lux-dex-pool-remove"></a>
#### lux dex pool remove

Remove liquidity from a pool

**Usage:**

```bash
lux dex pool remove [pool-id] [flags]
```

**Flags:**

```
      --percent float   Percentage of liquidity to remove (0-100)
```

<a id="lux-dex-pool-swap"></a>
#### lux dex pool swap

Swap tokens using the best available route through AMM pools.

Examples:
  lux dex pool swap --from LUX --to USDT --amount 100 --slippage 0.5

**Usage:**

```bash
lux dex pool swap [flags]
```

**Flags:**

```
      --amount float     Amount to swap
      --from string      Token to swap from
      --slippage float   Maximum slippage tolerance (%) (default 0.5)
      --to string        Token to swap to
```

<a id="lux-dex-status"></a>
### lux dex status

Display DEX network status including:
  - Connected nodes
  - Market statistics
  - Recent trades
  - Network health

**Usage:**

```bash
lux dex status
```

<a id="lux-down"></a>
## lux down

Stops the luxd node for <network>/<env> by:

  1. Probing http://127.0.0.1:<httpPort>/v1/info → info.getNetworkID
  2. Verifying the response equals the expected networkID from chain.yaml
  3. Looking up the PID listening on httpPort via lsof
  4. SIGTERM → wait drain → SIGKILL if still up

The (network, env) tuple alone determines what gets stopped. No --data-dir,
no --pid-file. Refusal-by-default: if the responder's networkID does not
match, we leave it alone.

Examples:
  lux down zoo/localnet
  lux down lux/devnet

**Usage:**

```bash
lux down <network>/<env>
```

<a id="lux-explore"></a>
## lux explore

The explore command starts a local block explorer that indexes
chain data and serves the explorer API + frontend.

USAGE:

  lux explore                     Start explorer for the running local network
  lux explore --rpc <url>         Start explorer for a specific RPC endpoint
  lux explore --chain cchain      Index a specific chain (default: cchain)
  lux explore --port 8090         API port (default: 8090)

The explorer runs as a background process. Use 'lux explore stop' to stop it.
Data is stored in ~/.lux/explorer/ and persists across restarts.

ENDPOINTS:

  http://localhost:8090/v1/explorer/stats     Chain statistics
  http://localhost:8090/v1/explorer/blocks     Block list
  http://localhost:8090/v1/explorer/search     Search
  http://localhost:8090/health                 Health check

**Usage:**

```bash
lux explore [flags]
```

**Flags:**

```
      --chain string   Chain to index (cchain, xchain, pchain, or chain name) (default "cchain")
      --data string    Data directory (default: ~/.lux/explorer/)
      --open           Open browser after starting (default true)
      --port int       HTTP port for explorer API (default 8090)
      --rpc string     RPC endpoint (auto-detected from running network if not set)
```

<a id="lux-explore-status"></a>
### lux explore status

Show explorer status

**Usage:**

```bash
lux explore status
```

<a id="lux-explore-stop"></a>
### lux explore stop

Stop the running explorer

**Usage:**

```bash
lux explore stop
```

<a id="lux-fhe"></a>
## lux fhe

The fhe command provides tools for Fully Homomorphic Encryption (FHE)
on the Lux network, including key generation, encryption, computation
on encrypted data, and decryption.

These operations integrate with the T-Chain (Threshold chain) for
on-chain FHE computation via precompiled contracts.

SCHEMES:

  TFHE    - Fast bootstrapping, boolean/small integer circuits
  BGV     - Batched integer arithmetic
  CKKS    - Approximate fixed-point arithmetic

**Usage:**

```bash
lux fhe
```

<a id="lux-fhe-decrypt"></a>
### lux fhe decrypt

Decrypt a hex-encoded ciphertext using the FHE secret key.

Examples:
  lux fhe decrypt --key ./keys/secret.key --input <hex>

**Usage:**

```bash
lux fhe decrypt [flags]
```

**Flags:**

```
      --input string   Hex-encoded ciphertext
      --key string     Path to secret key file
```

<a id="lux-fhe-encrypt"></a>
### lux fhe encrypt

Encrypt a boolean value using an FHE secret key.
Outputs hex-encoded ciphertext to stdout.

Examples:
  lux fhe encrypt --key ./keys/secret.key --value true
  lux fhe encrypt --key ./keys/secret.key --value false

**Usage:**

```bash
lux fhe encrypt [flags]
```

**Flags:**

```
      --key string   Path to secret key file
      --value        Boolean value to encrypt
```

<a id="lux-fhe-eval"></a>
### lux fhe eval

Evaluate a boolean operation on FHE-encrypted data without decrypting.

Requires bootstrap key for homomorphic evaluation.

Supported gates: AND, OR, XOR, NAND, NOR, XNOR, NOT

Examples:
  lux fhe eval --op AND --inputs ct1.hex,ct2.hex --bsk ./keys/bootstrap.key

**Usage:**

```bash
lux fhe eval [flags]
```

**Flags:**

```
      --op string   Boolean gate (AND, OR, XOR, NAND, NOR, XNOR, NOT) (default "AND")
```

<a id="lux-fhe-keygen"></a>
### lux fhe keygen

Generate a Fully Homomorphic Encryption key pair.

Produces a secret key (for decryption), public key (for encryption),
and bootstrap key (for homomorphic operations).

Examples:
  lux fhe keygen
  lux fhe keygen --scheme PN11QP54 --output ./keys/

**Usage:**

```bash
lux fhe keygen [flags]
```

**Flags:**

```
      --output string   Output directory for keys (default: current dir)
      --scheme string   Parameter set (PN10QP27, PN11QP54, STD128, STD128Q) (default "PN10QP27")
```

<a id="lux-gpu"></a>
## lux gpu

The gpu command provides utilities for managing GPU acceleration
in the Lux node. Use subcommands to check GPU status, availability,
and configuration.

GPU acceleration is used for:
  - NTT operations in Corona consensus
  - FHE operations in ThresholdVM
  - Lattice cryptography operations

**Usage:**

```bash
lux gpu
```

<a id="lux-gpu-status"></a>
### lux gpu status

Show the current GPU acceleration status including:
  - GPU availability on this system
  - Active backend (Metal, CUDA, or CPU)
  - Platform and architecture information
  - Available GPU-accelerated features
  - Default configuration settings

**Usage:**

```bash
lux gpu status [flags]
```

**Flags:**

```
      --json   output status in JSON format
```

<a id="lux-info"></a>
## lux info

Resolves <name>/<env> against the registry and prints every field
of the resulting profile: distinct identifiers (network ID + EVM chain
ID), ports, paths, snapshot name, service label, RPC endpoints, and
live state.

The wire identifiers are deliberately distinct:
  network ID       uint32, validator wire (--network-id, info.getNetworkID)
  EVM chain ID     uint64, EIP-155 (eth_chainId, MetaMask)

Lux brand keeps them separate (NID 1 / EVM 96369). Sovereign-L1 brand
forks (Zoo, Hanzo, Pars, Osage, Liquidity) collapse them by convention.

Examples:
  lux info zoo/mainnet
  lux info lux/devnet

**Usage:**

```bash
lux info <name>/<env>
```

<a id="lux-key"></a>
## lux key

The key command suite provides tools for managing all cryptographic keys
used in the Lux network.

Key types managed:
- EC (secp256k1): Transaction signing, Ethereum compatibility
- BLS: Consensus participation, aggregated signatures
- Ring-sig (LSAG over secp256k1) for privacy
- ML-DSA: Post-quantum digital signatures (NIST Level 3)

All keys are derived from a single BIP39 mnemonic phrase using HKDF,
stored in ~/.lux/keys/<name>/ with separate subdirectories for each type.

Examples:
  lux key create validator1              # Create new key set
  lux key create validator1 --mnemonic   # Create from existing mnemonic
  lux key generate -n 5                  # Batch generate 5 key sets (key-0 to key-4)
  lux key generate -n 10 -p validator    # Generate validator-0 to validator-9
  lux key derive -n 5                    # Derive 5 keys from MNEMONIC
  lux key derive -n 5 --show             # Show derived addresses without saving
  lux key list                           # List all key sets
  lux key show validator1                # Show public keys and addresses
  lux key delete validator1              # Delete key set
  lux key export validator1              # Export mnemonic (DANGER!)
  lux key lock validator1                # Lock key (clear from memory)
  lux key lock --all                     # Lock all keys
  lux key unlock validator1              # Unlock key for use
  lux key backend list                   # List available backends
  lux key backend set keychain           # Set default backend
  lux key kchain status                  # Check K-Chain service
  lux key kchain create mykey            # Create distributed key
  lux key kchain sign mykey "data"       # Threshold sign data

**Usage:**

```bash
lux key
```

<a id="lux-key-backend"></a>
### lux key backend

Manage key storage backends for cryptographic keys.

Available backends:
  software       - Encrypted file storage (AES-256-GCM + Argon2id)
  keychain       - macOS Keychain with optional TouchID
  secret-service - Linux Secret Service (GNOME Keyring, KWallet)
  yubikey        - Yubikey hardware token
  zymbit         - Zymbit HSM (Raspberry Pi)
  walletconnect  - Remote signing via mobile wallet
  ledger         - Ledger hardware wallet
  env            - Environment variable storage

Examples:
  lux key backend list          # List available backends
  lux key backend set keychain  # Set default backend
  lux key backend info          # Show current backend info

**Usage:**

```bash
lux key backend
```

<a id="lux-key-backend-info"></a>
#### lux key backend info

Display detailed information about the current default key storage backend.

**Usage:**

```bash
lux key backend info
```

<a id="lux-key-backend-list"></a>
#### lux key backend list

List all key storage backends and their availability status.

Backends marked as 'available' can be used on this system.
Some backends require specific hardware or services to be present.

**Usage:**

```bash
lux key backend list
```

<a id="lux-key-backend-set"></a>
#### lux key backend set

Set the default key storage backend.

The default backend is used when creating new keys.
Existing keys remain in their original backend.

Valid backend types:
  software, keychain, secret-service, yubikey, zymbit, walletconnect, ledger, env

**Usage:**

```bash
lux key backend set <type>
```

<a id="lux-key-create"></a>
### lux key create

Create a new key set with all cryptographic key types.

Generates a BIP39 mnemonic phrase and derives:
- EC (secp256k1) key for transactions
- BLS key for consensus
- Ring-signature (LSAG) key over secp256k1
- ML-DSA key for post-quantum signatures

Keys are stored in ~/.lux/keys/<name>/

Examples:
  lux key create validator1                           # Generate new mnemonic
  lux key create validator1 --mnemonic                # Prompt for existing mnemonic
  lux key create validator1 --phrase "word1 word2..." # Use provided mnemonic
  lux key create mainnet-key-01 --phrase "$MNEMONIC" --account 1  # Derive account 1

**Usage:**

```bash
lux key create <name> [flags]
```

**Flags:**

```
      --account uint32   Account index for HD derivation (0-based)
  -m, --mnemonic         Import from existing mnemonic (prompts for input)
      --phrase string    Mnemonic phrase to import (12 or 24 words)
```

<a id="lux-key-delete"></a>
### lux key delete

Delete a key set from ~/.lux/keys/

WARNING: This permanently deletes all keys! Make sure you have backed up
the mnemonic phrase before deleting.

Example:
  lux key delete validator1
  lux key delete validator1 --force  # Skip confirmation

**Usage:**

```bash
lux key delete <name> [flags]
```

**Flags:**

```
  -f, --force   Skip confirmation prompt
```

<a id="lux-key-derive"></a>
### lux key derive

Derive multiple key sets from a single mnemonic phrase.

Uses the MNEMONIC environment variable to derive keys deterministically.
Each key uses Lux P/X-Chain BIP-44 path: m/44'/9000'/0'/0/{index}

This ensures compatibility with MetaMask, cast, and other Ethereum tools.
The same mnemonic always produces the same keys across all tools.

Examples:
  # Derive 5 validator keys from mnemonic
  export MNEMONIC="your 24 words here"
  lux key derive -n 5 --prefix validator

  # Derive keys starting at index 5
  lux key derive -n 3 --start 5 --prefix backup

  # Show addresses without saving (for verification)
  lux key derive -n 5 --show

  # Show addresses with private keys (DANGER)
  lux key derive -n 1 --show --export

**Usage:**

```bash
lux key derive [flags]
```

**Flags:**

```
  -n, --count int        Number of keys to derive (default 5)
      --export           Show private keys in output (DANGER - keep secret!)
      --network string   Network for P/X address HRP: mainnet (P-lux1…) | testnet (P-test1…) | devnet (P-dev1…) | local (P-local1…) | custom (P-custom1…) (default "mainnet")
  -p, --prefix string    Prefix for key names (default "mainnet-key")
      --show             Only show addresses, don't save keys
  -s, --start int        Starting account index
```

<a id="lux-key-export"></a>
### lux key export

Export key set data.

By default, exports public keys. Use --mnemonic to export the seed phrase.

WARNING: Exporting the mnemonic exposes your private keys!

Examples:
  lux key export validator1                    # Export public keys
  lux key export validator1 --mnemonic         # Export mnemonic (DANGER!)
  lux key export validator1 -o keys.json       # Export to file

**Usage:**

```bash
lux key export <name> [flags]
```

**Flags:**

```
      --mnemonic        Export mnemonic phrase (DANGEROUS!)
  -o, --output string   Output file (default: stdout)
```

<a id="lux-key-export-signer"></a>
### lux key export-signer

Export BLS signer keys derived from MNEMONIC for use as luxd
staking signer keys. Each key is written as a raw 32-byte file.

This is needed when starting luxd nodes manually (outside of netrunner)
that need to use mnemonic-derived BLS keys for consensus.

Examples:
  # Export signer keys for accounts 5-9
  export MNEMONIC="your mnemonic here"
  lux key export-signer -n 5 --start 5 --output ~/.lux/local-validators

  # This creates:
  #   ~/.lux/local-validators/node5/signer.key
  #   ~/.lux/local-validators/node6/signer.key
  #   ...

**Usage:**

```bash
lux key export-signer [flags]
```

**Flags:**

```
  -n, --count int       Number of signer keys to export (default 5)
  -o, --output string   Output directory (required)
  -s, --start int       Starting account index
```

<a id="lux-key-generate"></a>
### lux key generate

Generate multiple key sets with indexed names.

Creates keys with names like: prefix-0, prefix-1, prefix-2, etc.
Each key set contains EC, BLS, ring-sig, and ML-DSA keys.

Examples:
  lux key generate -n 5                    # Creates key-0 through key-4
  lux key generate -n 10 --prefix validator # Creates validator-0 through validator-9
  lux key generate -n 3 --start 5          # Creates key-5, key-6, key-7

**Usage:**

```bash
lux key generate [flags]
```

**Flags:**

```
  -n, --count int       Number of key sets to generate (default 1)
  -p, --prefix string   Prefix for key names (default "key")
  -s, --start int       Starting index number
```

<a id="lux-key-genesis"></a>
### lux key genesis

Generate a genesis.json file with P-Chain and X-Chain allocations.

Network modes:
  --mainnet   Use mainnet configuration (Network ID: 1, Chain ID: 96369)
              - Uses mainnet-key-01 through mainnet-key-11
              - 5 P-Chain keys (first unlocked, rest 100-year vesting)
              - 5 X-Chain keys (100-year vesting)

  --testnet   Use testnet configuration (Network ID: 2, Chain ID: 96368)
              - Uses testnet-key-01 through testnet-key-10
              - Shorter vesting for testing

  --devnet    Use devnet configuration (Network ID: 3, Chain ID: 96370)
              - Uses devnet-key-01 through devnet-key-05
              - No vesting, fully unlocked

  (no flag)   Local development (Network ID: 1337, Chain ID: 1337)
              - Generates new keys if needed
              - Single validator, fully unlocked

The command will generate keys if they don't exist (use --generate-keys to force).

Examples:
  # Generate mainnet genesis using existing mainnet keys
  lux key genesis --mainnet -o /path/to/genesis.json

  # Generate testnet genesis
  lux key genesis --testnet -o /path/to/genesis.json

  # Generate devnet genesis with new keys
  lux key genesis --devnet --generate-keys -o /path/to/genesis.json

  # Custom configuration with manual key selection
  lux key genesis --p-chain key1,key2 --x-chain key3 -o genesis.json

**Usage:**

```bash
lux key genesis [flags]
```

**Flags:**

```
      --amount uint              Amount per key in nLUX (default 1B LUX) (default 1000000000000000000)
      --c-chain-genesis string   Path to existing genesis to preserve C-Chain config
      --devnet                   Generate devnet genesis (Network ID: 3, Chain ID: 96370)
      --generate-keys            Generate new keys if they don't exist
      --mainnet                  Generate mainnet genesis (Network ID: 1, Chain ID: 96369)
      --network-id uint32        Network ID (overrides network preset)
  -n, --num-keys int             Number of keys to generate (for mainnet/testnet) (default 11)
  -o, --output string            Output file path (default: ~/.lux/networks/<network>/genesis.json)
      --p-chain strings          P-Chain allocation keys (overrides network preset)
      --save                     Save genesis to ~/.lux/networks/<network>/genesis.json
      --testnet                  Generate testnet genesis (Network ID: 2, Chain ID: 96368)
      --vesting-percent float    Percentage unlocked per year (overrides network preset)
      --vesting-years int        Vesting period in years (overrides network preset)
      --x-chain strings          X-Chain allocation keys (overrides network preset)
```

<a id="lux-key-import"></a>
### lux key import

Import a key set by recovering from a mnemonic phrase.

Derives all key types (EC, BLS, ring-sig, ML-DSA) from the mnemonic.

Example:
  lux key import validator1

**Usage:**

```bash
lux key import <name>
```

<a id="lux-key-kchain"></a>
### lux key kchain

K-Chain provides distributed key management using threshold cryptography.

Keys are split across multiple validators using Shamir Secret Sharing,
requiring a threshold of shares to reconstruct or sign.

Features:
  - Distributed key storage across validators
  - Threshold signing without key reconstruction
  - Proactive secret resharing
  - ML-KEM post-quantum encryption
  - ML-DSA post-quantum signatures

Default port range: 963N (9630-9639)

Examples:
  lux key kchain status                    # Check K-Chain service status
  lux key kchain distribute mykey          # Distribute key to validators
  lux key kchain sign mykey "data"         # Threshold sign data
  lux key kchain encrypt mykey "plaintext" # Encrypt with ML-KEM
  lux key kchain algorithms                # List supported algorithms

**Usage:**

```bash
lux key kchain
```

**Flags:**

```
      --endpoint string   K-Chain RPC endpoint (default "http://localhost:9630")
```

<a id="lux-key-kchain-algorithms"></a>
#### lux key kchain algorithms

List all cryptographic algorithms supported by K-Chain.

**Usage:**

```bash
lux key kchain algorithms
```

<a id="lux-key-kchain-create"></a>
#### lux key kchain create

Create a new key and distribute it across K-Chain validators.

Examples:
  lux key kchain create mykey
  lux key kchain create mykey -a ml-kem-768 -t 3 -n 5

**Usage:**

```bash
lux key kchain create <key-name> [flags]
```

**Flags:**

```
  -a, --algorithm string   Key algorithm (default "ml-kem-768")
  -n, --shares int         Total shares (default 5)
  -t, --threshold int      Threshold for reconstruction (default 3)
```

<a id="lux-key-kchain-decrypt"></a>
#### lux key kchain decrypt

Decrypt data using threshold key reconstruction.

Requires gathering shares from validators to reconstruct the
decryption key. The key is immediately cleared after decryption.

Examples:
  lux key kchain decrypt mykey <ciphertext>

**Usage:**

```bash
lux key kchain decrypt <key-name> <ciphertext>
```

<a id="lux-key-kchain-delete"></a>
#### lux key kchain delete

Delete a key and securely wipe all shares from validators.

Examples:
  lux key kchain delete mykey
  lux key kchain delete mykey --force

**Usage:**

```bash
lux key kchain delete <key-name> [flags]
```

**Flags:**

```
      --force   Force deletion even if shares exist
```

<a id="lux-key-kchain-distribute"></a>
#### lux key kchain distribute

Distribute a key across K-Chain validators using Shamir Secret Sharing.

The key is split into shares, each stored on a different validator.
A threshold number of shares is required to reconstruct or sign.

Examples:
  lux key kchain distribute mykey                    # Use defaults (3-of-5)
  lux key kchain distribute mykey -t 2 -n 3          # 2-of-3 threshold
  lux key kchain distribute mykey --validators v1:9630,v2:9631,v3:9632

**Usage:**

```bash
lux key kchain distribute <key-name> [flags]
```

**Flags:**

```
  -n, --shares int           Total number of shares to create (default 5)
  -t, --threshold int        Number of shares required to reconstruct (default 3)
      --validators strings   Validator endpoints (host:port)
```

<a id="lux-key-kchain-encrypt"></a>
#### lux key kchain encrypt

Encrypt data using the key's ML-KEM public key.

ML-KEM (Module-Lattice Key Encapsulation Mechanism) provides
post-quantum secure encryption.

Examples:
  lux key kchain encrypt mykey "secret message"
  lux key kchain encrypt mykey --algorithm ml-kem-768 "data"

**Usage:**

```bash
lux key kchain encrypt <key-name> <plaintext> [flags]
```

**Flags:**

```
  -a, --algorithm string   Encryption algorithm (default "ml-kem-768")
```

<a id="lux-key-kchain-gather"></a>
#### lux key kchain gather

Gather threshold shares from validators to reconstruct a key.

This command contacts validators to retrieve shares and reconstructs
the original key material. Requires threshold number of responsive validators.

Examples:
  lux key kchain gather mykey

**Usage:**

```bash
lux key kchain gather <key-name>
```

<a id="lux-key-kchain-list"></a>
#### lux key kchain list

List all keys stored in K-Chain.

**Usage:**

```bash
lux key kchain list [flags]
```

**Flags:**

```
  -a, --algorithm string   Filter by algorithm
```

<a id="lux-key-kchain-reshare"></a>
#### lux key kchain reshare

Perform proactive secret resharing to rotate key shares.

This creates new shares without changing the underlying key,
limiting the window of exposure if any shares are compromised.

Examples:
  lux key kchain reshare mykey
  lux key kchain reshare mykey -t 4 -n 7   # Change to 4-of-7

**Usage:**

```bash
lux key kchain reshare <key-name> [flags]
```

**Flags:**

```
  -n, --shares int           New total shares (0 = keep current)
  -t, --threshold int        New threshold (0 = keep current)
      --validators strings   New validator set
```

<a id="lux-key-kchain-show"></a>
#### lux key kchain show

Show detailed information about a distributed key.

**Usage:**

```bash
lux key kchain show <key-name> [flags]
```

**Flags:**

```
  -f, --format string   Public key format (pem, der, raw) (default "pem")
```

<a id="lux-key-kchain-sign"></a>
#### lux key kchain sign

Sign data using threshold signatures without reconstructing the key.

Each validator computes a partial signature using their share.
Partial signatures are combined to produce the final signature.

Examples:
  lux key kchain sign mykey "message to sign"
  lux key kchain sign mykey --hex 48656c6c6f
  lux key kchain sign mykey --algorithm bls-threshold "data"

**Usage:**

```bash
lux key kchain sign <key-name> <data> [flags]
```

**Flags:**

```
  -a, --algorithm string   Signing algorithm (default "bls-threshold")
      --hex                Interpret data as hex-encoded
```

<a id="lux-key-kchain-status"></a>
#### lux key kchain status

Check the health and status of the K-Chain distributed key management service.

**Usage:**

```bash
lux key kchain status
```

<a id="lux-key-kchain-verify"></a>
#### lux key kchain verify

Verify a signature against the key's public key.

Examples:
  lux key kchain verify mykey "message" <signature>

**Usage:**

```bash
lux key kchain verify <key-name> <data> <signature> [flags]
```

**Flags:**

```
  -a, --algorithm string   Signature algorithm (default "bls-threshold")
```

<a id="lux-key-list"></a>
### lux key list

List all key sets stored in ~/.lux/keys/

Shows the name of each key set. Use 'lux key show <name>' for details.

Example:
  lux key list
  lux key ls

**Usage:**

```bash
lux key list
```

<a id="lux-key-lock"></a>
### lux key lock

Lock a key to clear it from the memory session.

A locked key requires password authentication to use again.
This is a security measure to protect keys when not in use.

Examples:
  lux key lock validator1    # Lock a specific key
  lux key lock --all         # Lock all keys

**Usage:**

```bash
lux key lock [name] [flags]
```

**Flags:**

```
  -a, --all   Lock all keys
```

<a id="lux-key-migrate"></a>
### lux key migrate

Migrate legacy plaintext key files to encrypted keystore.enc format.

This command reads plaintext key files (ec/private.key, bls/secret.key, staker.key)
and encrypts them using AES-256-GCM with Argon2id key derivation.

After migration, the plaintext originals can be securely deleted with --secure.

Examples:
  lux key migrate node0              # Migrate single node
  lux key migrate node0 node1 node2  # Migrate multiple nodes
  lux key migrate --all              # Migrate all keys with plaintext files
  lux key migrate node0 --secure     # Migrate and securely delete originals

**Usage:**

```bash
lux key migrate [name...] [flags]
```

**Flags:**

```
      --all      Migrate all keys with plaintext files
      --force    Overwrite existing keystore.enc files
      --secure   Securely delete plaintext files after migration
```

<a id="lux-key-ring"></a>
### lux key ring

Ring signatures allow signing messages such that the signature can be
verified as coming from someone in a group (the "ring"), without revealing
which member actually signed. This provides strong anonymity guarantees.

Features:
  - LSAG (Linkable Spontaneous Anonymous Group) signatures using secp256k1
  - Lattice-based ring signatures for post-quantum security
  - Key images for linkability (double-spend detection)

The ring signature uses your ring-signature key (LSAG over secp256k1) from ~/.lux/keys/<name>/rt/

Examples:
  lux key ring sign mykey "message" --ring key1,key2,key3
  lux key ring verify "message" --signature <sig> --ring key1,key2,key3
  lux key ring keyimage mykey
  lux key ring schemes

**Usage:**

```bash
lux key ring
```

<a id="lux-key-ring-generate"></a>
#### lux key ring generate

Generate random public keys to use as decoys in a ring signature.

In production, you should use real public keys from the network for better
anonymity. This command is mainly for testing and demonstration.

Examples:
  lux key ring generate --size 5
  lux key ring generate --size 10 --scheme lattice

**Usage:**

```bash
lux key ring generate [flags]
```

**Flags:**

```
      --scheme string   Signature scheme (lsag, lattice) (default "lsag")
  -n, --size int        Number of keys to generate (default 5)
```

<a id="lux-key-ring-keyimage"></a>
#### lux key ring keyimage

Show the key image for a key. Key images are deterministic identifiers
derived from the private key that enable linkability - two signatures from
the same key will have the same key image.

This is used for double-spend detection in privacy-preserving transactions.

Examples:
  lux key ring keyimage mykey
  lux key ring keyimage mykey --scheme lattice

**Usage:**

```bash
lux key ring keyimage <key-name> [flags]
```

**Flags:**

```
      --scheme string   Signature scheme (lsag, lattice) (default "lsag")
```

<a id="lux-key-ring-schemes"></a>
#### lux key ring schemes

List all supported ring signature schemes and their properties.

**Usage:**

```bash
lux key ring schemes
```

<a id="lux-key-ring-sign"></a>
#### lux key ring sign

Create a ring signature for a message using your key and a ring of public keys.

Your key must be one of the keys in the ring. The signature proves you're a member
of the ring without revealing which member you are.

Examples:
  lux key ring sign mykey "message to sign" --ring key1,key2,key3
  lux key ring sign mykey --file message.txt --ring key1,key2,key3,key4
  lux key ring sign mykey "data" --ring key1,key2,key3 --scheme lattice

**Usage:**

```bash
lux key ring sign <key-name> <message> [flags]
```

**Flags:**

```
  -f, --file string     Read message from file
  -o, --output string   Write signature to file
      --ring strings    Ring member key names (comma-separated)
  -s, --scheme string   Signature scheme (lsag, lattice) (default "lsag")
```

<a id="lux-key-ring-verify"></a>
#### lux key ring verify

Verify a ring signature against a message and ring of public keys.

Examples:
  lux key ring verify "message" --signature <sig> --ring key1,key2,key3
  lux key ring verify --file message.txt --signature-file sig.txt --ring key1,key2,key3

**Usage:**

```bash
lux key ring verify <message> [flags]
```

**Flags:**

```
  -f, --file string             Read message from file
      --ring strings            Ring member key names (comma-separated)
      --scheme string           Signature scheme (lsag, lattice) (default "lsag")
      --signature string        Signature (hex-encoded)
      --signature-file string   Read signature from file
```

<a id="lux-key-show"></a>
### lux key show

Show public keys and addresses for a key set.

Displays:
- EC (secp256k1) address (Ethereum format)
- BLS public key (consensus)
- Ring-signature public key (LSAG over secp256k1)
- ML-DSA public key (post-quantum)

With --export flag, also displays private keys (DANGER - keep secret!).

Example:
  lux key show validator1
  lux key show validator1 --export

**Usage:**

```bash
lux key show <name> [flags]
```

**Flags:**

```
      --export   Export private keys (DANGER - keep secret!)
```

<a id="lux-key-staker"></a>
### lux key staker

Writes staker.crt, staker.key and signer.key into <dir> and prints the
NodeID, BLS public key and proof of possession.

Point luxd at the directory, then register the printed values:

  lux key staker ~/.luxd/staking
  lux primary addValidator --mainnet \
      --node-id <NodeID> --public-key <pub> --proof-of-possession <pop> \
      --stake 2 --duration 336h

**Usage:**

```bash
lux key staker <dir>
```

<a id="lux-key-unlock"></a>
### lux key unlock

Unlock a key by providing the password.

The key remains unlocked for the session duration (default 30 seconds).
After the timeout without access, the key is automatically locked and
requires re-authentication. The timeout resets on each key access.

Session timeout can be configured via:
  KEY_SESSION_TIMEOUT environment variable (e.g., "30s", "5m", "1h")

Password can be provided via:
  --password flag
  KEY_PASSWORD environment variable
  Interactive prompt (most secure)

Examples:
  lux key unlock validator1                    # Prompts for password
  lux key unlock validator1 --password secret  # Password via flag (less secure)
  KEY_SESSION_TIMEOUT=5m lux key unlock validator1  # 5 minute session

**Usage:**

```bash
lux key unlock <name> [flags]
```

**Flags:**

```
  -p, --password string   Password for the key
```

<a id="lux-kms"></a>
## lux kms

Key Management Service (KMS) for managing cryptographic keys and secrets.

The KMS provides:
  - Key generation (AES-256, RSA, ECDSA, Ed25519)
  - Encryption/decryption operations
  - Digital signatures
  - Secret management
  - MPC wallet integration

QUICK START:

  # Start the KMS server
  lux kms server start

  # Generate a new key
  lux kms key create --name mykey --type aes-256-gcm --usage encrypt-decrypt

  # List keys
  lux kms key list

  # Create a secret
  lux kms secret create --name API_KEY --value "sk-xxx" --env production

STORAGE:

  KMS data is stored in ~/.lux/kms/ by default.
  The root encryption key is derived from your system keychain or environment.

API:

  The KMS server exposes a REST API compatible with the kms-go SDK.
  Default address: http://localhost:8200

Available subcommands:
  server  - Manage the KMS server
  key     - Key management operations
  secret  - Secret management operations

<a id="lux-kms-key"></a>
### lux kms key

Commands for managing cryptographic keys.

<a id="lux-kms-key-create"></a>
#### lux kms key create

Create a new cryptographic key.

Supported key types:
  - aes-256-gcm   : Symmetric encryption (default)
  - rsa-4096      : RSA asymmetric key
  - ecdsa-p256    : ECDSA P-256 curve
  - ecdsa-p384    : ECDSA P-384 curve
  - ed25519       : EdDSA Ed25519

Usage types:
  - encrypt-decrypt : For encryption operations
  - sign-verify     : For digital signatures

Examples:
  lux kms key create --name mykey --type aes-256-gcm
  lux kms key create --name signing --type ecdsa-p256 --usage sign-verify

**Usage:**

```bash
lux kms key create [flags]
```

**Flags:**

```
      --description string   Key description
      --name string          Key name (required)
      --project string       Project ID
      --type string          Key type (default "aes-256-gcm")
      --usage string         Key usage (default "encrypt-decrypt")
```

<a id="lux-kms-key-delete"></a>
#### lux kms key delete

Delete a key

**Usage:**

```bash
lux kms key delete [keyID]
```

<a id="lux-kms-key-list"></a>
#### lux kms key list

List all keys

**Usage:**

```bash
lux kms key list
```

<a id="lux-kms-secret"></a>
### lux kms secret

Commands for managing encrypted secrets.

<a id="lux-kms-secret-create"></a>
#### lux kms secret create

Create a new encrypted secret.

Examples:
  lux kms secret create --name API_KEY --value "sk-xxx"
  lux kms secret create --name DB_PASSWORD --value "secret" --env production

**Usage:**

```bash
lux kms secret create [flags]
```

**Flags:**

```
      --env string     Environment (dev, staging, prod)
      --name string    Secret name (required)
      --path string    Secret path (default "/")
      --value string   Secret value (required)
```

<a id="lux-kms-secret-get"></a>
#### lux kms secret get

Get a secret value

**Usage:**

```bash
lux kms secret get [secretName]
```

<a id="lux-kms-secret-list"></a>
#### lux kms secret list

List all secrets

**Usage:**

```bash
lux kms secret list
```

<a id="lux-kms-server"></a>
### lux kms server

Commands for starting and managing the KMS server.

<a id="lux-kms-server-start"></a>
#### lux kms server start

Start the KMS HTTP API server.

The server provides a REST API for key management, encryption, and secret
operations. It is compatible with the kms-go SDK client.

Examples:
  # Start with default settings
  lux kms server start

  # Start on a custom port
  lux kms server start --addr :9200

  # Start with in-memory storage (for testing)
  lux kms server start --in-memory

  # Start with API key authentication
  lux kms server start --api-key your-secret-key

**Usage:**

```bash
lux kms server start [flags]
```

**Flags:**

```
      --addr string       Server listen address (default ":8200")
      --api-key string    API key for authentication
      --data-dir string   Data directory (default: ~/.lux/kms)
      --in-memory         Use in-memory storage (data lost on restart)
```

<a id="lux-link"></a>
## lux link

Link Lux binaries for system-wide use.

Creates ~/.lux/bin directory if needed and symlinks binaries.
Add ~/.lux/bin to your PATH for easy access.

SUPPORTED BINARIES:
  all        - All binaries (lux, luxd, netrunner)
  lux        - CLI binary
  luxd       - Node binary
  netrunner  - Network runner

EXAMPLES:

  # Link all binaries (auto-detect from workspace)
  lux link all

  # Link specific binary (auto-detect)
  lux link luxd
  lux link netrunner

  # Link specific binary with explicit path
  lux link luxd /path/to/luxd

**Usage:**

```bash
lux link [binary] [path]
```

<a id="lux-mcp"></a>
## lux mcp

Relays MCP between an agent on stdio and a node's door on ZAP.

The node's tools ARE its typed operations — one tool per op, named by the
op — so nothing here enumerates them and nothing here can fall behind
them. Every frame is passed whole.

Point an MCP client at this command:

  {"command": "lux", "args": ["mcp", "--at", "/run/zip/luxd.sock"]}

Examples:
  lux mcp
  lux mcp --at /run/zip/luxd.sock
  lux mcp --at node.lux.svc:9655

**Usage:**

```bash
lux mcp [flags]
```

**Flags:**

```
      --at string   the node's MCP address (a socket path, or host:port) (default "/run/zip/luxd.sock")
```

<a id="lux-mpc"></a>
## lux mpc

Multi-Party Computation (MPC) management commands.

MPC enables threshold signing for blockchain wallets, where multiple
parties must cooperate to sign transactions without any single party
having access to the complete private key.

Each MPC node holds exactly one key shard. For a t-of-n threshold scheme,
at least t nodes must cooperate to produce a valid signature.

NETWORK TYPES:

  --mainnet   Production MPC network (ports 9700-9799)
  --testnet   Test MPC network (ports 9710-9809)
  --devnet    Development MPC network (ports 9720-9819)

QUICK START:

  # Initialize a 3-of-5 MPC network
  lux mpc node init --threshold 3 --nodes 5 --devnet

  # Start all MPC nodes
  lux mpc node start

  # Check status
  lux mpc node status

  # Create a wallet
  lux mpc wallet create --name "Treasury"

  # Stop the network
  lux mpc node stop

SECURITY:

  Key shards are stored encrypted in ~/.lux/keys/mpc/
  Backups are stored in ~/.lux/mpc/backups/ by default

CLOUD DEPLOYMENT:

  Deploy MPC nodes to cloud providers:
  lux mpc deploy create mpc-devnet-xxx --provider aws --region us-east-1
  lux mpc deploy status mpc-devnet-xxx
  lux mpc deploy ssh mpc-devnet-xxx mpc-node-1

Available subcommands:
  node     - Manage MPC nodes (local)
  deploy   - Deploy MPC nodes to cloud
  backup   - Backup and restore node data
  wallet   - Manage MPC wallets
  sign     - Threshold signing operations

<a id="lux-mpc-backup"></a>
### lux mpc backup

Backup and restore MPC node data.

By default, backups are stored locally in ~/.lux/mpc/backups.
For cloud storage, specify a destination URI.

Supports multiple storage backends:
  - Local filesystem (default: ~/.lux/mpc/backups)
  - S3 (AWS, MinIO, Cloudflare R2, etc.)
  - GCS (Google Cloud Storage)
  - Azure Blob Storage

Examples:
  # Backup to default local directory (~/.lux/mpc/backups)
  lux mpc backup create

  # Backup to S3
  lux mpc backup create --destination s3://my-bucket/backups

  # Backup to custom local directory
  lux mpc backup create --destination file:///backups/mpc

  # List local backups (default)
  lux mpc backup list

  # List S3 backups
  lux mpc backup list --destination s3://my-bucket/backups

  # Restore from local backup
  lux mpc backup restore my-backup-20250125

  # Restore from S3
  lux mpc backup restore my-backup-20250125 --destination s3://my-bucket/backups

<a id="lux-mpc-backup-create"></a>
#### lux mpc backup create

Create a backup of MPC node data.

The backup includes:
  - BadgerDB database
  - Key shares and wallet data
  - Node configuration

By default, backups are stored in ~/.lux/mpc/backups.
Backups are compressed with zstd by default and can be encrypted
with age encryption for secure storage.

**Usage:**

```bash
lux mpc backup create [flags]
```

**Flags:**

```
      --age-recipient strings   Age recipient public key(s)
      --compression string      Compression algorithm (zstd, gzip, none) (default "zstd")
  -d, --destination string      Storage destination (default: ~/.lux/mpc/backups)
      --encrypt                 Encrypt backup with age
      --incremental             Create incremental backup
```

<a id="lux-mpc-backup-delete"></a>
#### lux mpc backup delete

Delete a backup

**Usage:**

```bash
lux mpc backup delete <backup-name> [flags]
```

**Flags:**

```
  -d, --destination string   Storage destination (default: ~/.lux/mpc/backups)
```

<a id="lux-mpc-backup-list"></a>
#### lux mpc backup list

List available backups

**Usage:**

```bash
lux mpc backup list [flags]
```

**Flags:**

```
  -d, --destination string   Storage destination (default: ~/.lux/mpc/backups)
```

<a id="lux-mpc-backup-restore"></a>
#### lux mpc backup restore

Restore MPC node data from a backup.

This will stop the MPC node if running, restore the data,
and optionally restart the node.

**Usage:**

```bash
lux mpc backup restore <backup-name> [flags]
```

**Flags:**

```
      --age-identity strings   Age identity file(s) for decryption
  -d, --destination string     Storage destination (default: ~/.lux/mpc/backups)
      --target string          Target path (default: original location)
```

<a id="lux-mpc-backup-verify"></a>
#### lux mpc backup verify

Download and verify backup integrity without restoring.

**Usage:**

```bash
lux mpc backup verify <backup-name> [flags]
```

**Flags:**

```
      --age-identity strings   Age identity file(s) for decryption
  -d, --destination string     Storage destination (default: ~/.lux/mpc/backups)
```

<a id="lux-mpc-deploy"></a>
### lux mpc deploy

Deploy MPC nodes to cloud providers for production use.

Each MPC node is deployed to a separate server for security.
Key shards are encrypted and stored securely on each node.

SUPPORTED PROVIDERS:

  aws           Amazon Web Services (EC2)
  gcp           Google Cloud Platform (Compute Engine)
  azure         Microsoft Azure (Virtual Machines)
  digitalocean  DigitalOcean (Droplets)

SECURITY CONSIDERATIONS:

  - Each node should be in a different availability zone/region
  - Key shards are encrypted with age before storage
  - SSH access is required for node management
  - Use private networks where possible

Examples:
  # Deploy to AWS
  lux mpc deploy create mpc-devnet-xxx --provider aws --region us-east-1

  # Deploy to DigitalOcean
  lux mpc deploy create mpc-devnet-xxx --provider digitalocean --region nyc1

  # Check deployment status
  lux mpc deploy status mpc-devnet-xxx

  # SSH to a specific node
  lux mpc deploy ssh mpc-devnet-xxx mpc-node-1

  # Destroy deployment
  lux mpc deploy destroy mpc-devnet-xxx

<a id="lux-mpc-deploy-create"></a>
#### lux mpc deploy create

Deploy an initialized MPC network to cloud infrastructure.

The network must be initialized first with 'lux mpc node init'.
Each node will be deployed to a separate cloud instance.

**Usage:**

```bash
lux mpc deploy create <network-name> [flags]
```

**Flags:**

```
      --aws-profile string            AWS profile name
      --aws-vpc string                AWS VPC ID
      --azure-resource-group string   Azure resource group
      --azure-subscription string     Azure subscription ID
      --gcp-project string            GCP project ID
      --gcp-zone string               GCP zone
      --instance-type string          Instance type (default: provider-specific)
  -p, --provider string               Cloud provider (aws, gcp, azure, digitalocean)
  -r, --region string                 Cloud region
      --ssh-key string                Path to SSH private key
      --ssh-user string               SSH username (default "ubuntu")
```

<a id="lux-mpc-deploy-destroy"></a>
#### lux mpc deploy destroy

Terminate all cloud instances and clean up resources.

WARNING: This will delete all deployed instances!
Make sure you have backups of key shards before destroying.

**Usage:**

```bash
lux mpc deploy destroy <network-name> [flags]
```

**Flags:**

```
  -f, --force   Skip confirmation
```

<a id="lux-mpc-deploy-list"></a>
#### lux mpc deploy list

List deployments

**Usage:**

```bash
lux mpc deploy list
```

<a id="lux-mpc-deploy-ssh"></a>
#### lux mpc deploy ssh

Open an SSH session to a deployed MPC node.

Examples:
  lux mpc deploy ssh mpc-devnet-xxx mpc-node-1

**Usage:**

```bash
lux mpc deploy ssh <network-name> <node-name>
```

<a id="lux-mpc-deploy-status"></a>
#### lux mpc deploy status

Show deployment status

**Usage:**

```bash
lux mpc deploy status <network-name>
```

<a id="lux-mpc-node"></a>
### lux mpc node

Commands for managing MPC node lifecycle.

MPC nodes form a threshold signing network. Each node holds one key shard
and participates in distributed signing operations.

Examples:
  # Initialize a new 3-of-5 MPC network
  lux mpc node init --threshold 3 --nodes 5 --devnet

  # Start all nodes in the network
  lux mpc node start

  # Check status of all nodes
  lux mpc node status

  # Stop all nodes
  lux mpc node stop

  # Clean up (stop and remove data)
  lux mpc node clean

<a id="lux-mpc-node-clean"></a>
#### lux mpc node clean

Stop all nodes and remove network data.

WARNING: This will delete all node data including key shards!
Make sure you have backups before running this command.

Examples:
  # Clean the default network
  lux mpc node clean

  # Force clean without confirmation
  lux mpc node clean --force

**Usage:**

```bash
lux mpc node clean [network-name] [flags]
```

**Flags:**

```
  -f, --force   Skip confirmation
```

<a id="lux-mpc-node-init"></a>
#### lux mpc node init

Initialize a new MPC network with the specified threshold configuration.

This creates the network directory structure, generates node configurations,
and prepares encrypted key storage directories.

Examples:
  # Create a 2-of-3 devnet MPC network
  lux mpc node init --threshold 2 --nodes 3 --devnet

  # Create a 3-of-5 mainnet MPC network
  lux mpc node init --threshold 3 --nodes 5 --mainnet

**Usage:**

```bash
lux mpc node init [flags]
```

**Flags:**

```
      --devnet          Initialize devnet MPC network (default)
      --mainnet         Initialize mainnet MPC network
  -n, --nodes int       Total number of nodes (default 3)
      --testnet         Initialize testnet MPC network
  -t, --threshold int   Signing threshold (t in t-of-n) (default 2)
```

<a id="lux-mpc-node-list"></a>
#### lux mpc node list

List all initialized MPC networks.

**Usage:**

```bash
lux mpc node list
```

<a id="lux-mpc-node-start"></a>
#### lux mpc node start

Start all MPC nodes in a network.

If no network name is specified, starts the most recently created network.

Examples:
  # Start all nodes in the default network
  lux mpc node start

  # Start a specific network
  lux mpc node start mpc-devnet-abc123

**Usage:**

```bash
lux mpc node start [network-name] [flags]
```

**Flags:**

```
      --devnet    Start devnet MPC network
      --mainnet   Start mainnet MPC network
      --testnet   Start testnet MPC network
```

<a id="lux-mpc-node-status"></a>
#### lux mpc node status

Display status of all MPC nodes in a network.

Shows running status, uptime, endpoints, and health information.

Examples:
  # Show status of default network
  lux mpc node status

  # Show status of specific network
  lux mpc node status mpc-devnet-abc123

**Usage:**

```bash
lux mpc node status [network-name]
```

<a id="lux-mpc-node-stop"></a>
#### lux mpc node stop

Stop all MPC nodes in a network.

This gracefully shuts down nodes and saves state for later restart.

Examples:
  # Stop the default network
  lux mpc node stop

  # Stop a specific network
  lux mpc node stop mpc-devnet-abc123

**Usage:**

```bash
lux mpc node stop [network-name]
```

<a id="lux-mpc-sign"></a>
### lux mpc sign

Commands for threshold signing operations.

Examples:
  # Initiate a signing request
  lux mpc sign request --wallet <wallet-id> --message "0x..."

  # Approve a signing request
  lux mpc sign approve <request-id>

  # Check signing status
  lux mpc sign status <request-id>

<a id="lux-mpc-sign-approve"></a>
#### lux mpc sign approve

Approve a signing request

**Usage:**

```bash
lux mpc sign approve <request-id>
```

<a id="lux-mpc-sign-request"></a>
#### lux mpc sign request

Initiate a signing request

**Usage:**

```bash
lux mpc sign request
```

<a id="lux-mpc-sign-status"></a>
#### lux mpc sign status

Check signing status

**Usage:**

```bash
lux mpc sign status <request-id>
```

<a id="lux-mpc-wallet"></a>
### lux mpc wallet

Commands for managing MPC wallets and their key shares.

Examples:
  # List wallets
  lux mpc wallet list

  # Create a new wallet
  lux mpc wallet create --name "Treasury" --threshold 2 --parties 3

  # Show wallet details
  lux mpc wallet show <wallet-id>

<a id="lux-mpc-wallet-create"></a>
#### lux mpc wallet create

Create a new wallet

**Usage:**

```bash
lux mpc wallet create
```

<a id="lux-mpc-wallet-export"></a>
#### lux mpc wallet export

Export wallet public key

**Usage:**

```bash
lux mpc wallet export <wallet-id>
```

<a id="lux-mpc-wallet-list"></a>
#### lux mpc wallet list

List wallets

**Usage:**

```bash
lux mpc wallet list
```

<a id="lux-mpc-wallet-show"></a>
#### lux mpc wallet show

Show wallet details

**Usage:**

```bash
lux mpc wallet show <wallet-id>
```

<a id="lux-netrunner"></a>
## lux netrunner

Commands for managing the Lux network runner.

The netrunner is used for local network testing and development.

**Usage:**

```bash
lux netrunner
```

<a id="lux-netrunner-link"></a>
### lux netrunner link

Link netrunner binary for the CLI to use.

Creates ~/.lux/bin directory if needed and symlinks the netrunner binary.

EXAMPLES:

  # Link netrunner (auto-detect from ../netrunner/bin/netrunner)
  lux netrunner link --auto

  # Link specific path
  lux netrunner link /path/to/netrunner

**Usage:**

```bash
lux netrunner link [path] [flags]
```

**Flags:**

```
      --auto   auto-detect netrunner from standard locations
```

<a id="lux-network"></a>
## lux network

The network command manages local network runtime operations.

OVERVIEW:

  The network command suite controls the lifecycle of local Lux networks
  used for development and testing. It manages the node processes and runtime
  state, but does NOT manage blockchain configurations (use 'lux chain' for that).

COMMANDS:

  start     Start a local network (mainnet/testnet/devnet/dev mode)
  stop      Stop the running network and save a snapshot
  status    Show network status and endpoints
  clean     Stop network and delete runtime data (preserves chains)
  snapshot  Manage network snapshots

NETWORK TYPES:

  mainnet   Production network (3 validators, port 9630)
  testnet   Test network (3 validators, port 9640)
  devnet    Development network (3 validators, port 9650)
  dev       Single-node dev mode with K=1 consensus

TYPICAL WORKFLOW:

  # Start a development network
  lux network start --devnet

  # Check it's running
  lux network status

  # Deploy a chain (see 'lux chain --help')
  lux chain deploy mychain

  # Stop and save state
  lux network stop

  # Clean everything (preserves chain configs)
  lux network clean

NOTES:

  - Only one network type can run at a time
  - Chain configurations are managed separately via 'lux chain'
  - Runtime data is stored in ~/.lux/networks/<type>
  - Use 'lux network clean' to wipe runtime data but keep chain configs

**Usage:**

```bash
lux network
```

<a id="lux-network-bootstrap"></a>
### lux network bootstrap

The bootstrap command downloads and installs a network snapshot from a remote source.
It supports downloading split archives (parts) in parallel and reassembling them.

This is useful for quickly syncing a new node by starting from a recent snapshot
instead of syncing from genesis.

**Usage:**

```bash
lux network bootstrap [flags]
```

**Flags:**

```
      --network-type string    network type to bootstrap (mainnet, testnet) (default "mainnet")
      --snapshot-name string   specific snapshot name to download (optional)
      --url string             base URL for the snapshot parts (optional, defaults to official repo)
```

<a id="lux-network-clean"></a>
### lux network clean

The network clean command stops the network and deletes runtime data.

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

STORAGE CLEANUP:

  The CLI can accumulate significant storage over time:
  - netrunner-server.log files (can grow to 100GB+)
  - .backup.* directories from snapshot loads
  - Stale run directories from previous sessions

  Use --logs, --backups, --stale-runs, or --all to clean these:
    lux network clean --all              # Clean everything
    lux network clean --logs             # Clean large logs only
    lux network clean --all --dry-run    # Preview what would be deleted

NOTE: Chain configurations are explicitly preserved. To delete a chain
configuration, use: lux chain delete <chainName>

**Usage:**

```bash
lux network clean [flags]
```

**Flags:**

```
      --all                clean all: logs, backups, and stale runs
      --backups            clean up old .backup.* directories
      --dry-run            show what would be deleted without actually deleting
      --logs               clean up large netrunner-server.log files
      --max-age-days int   maximum age in days for backups and stale runs (default 7)
      --max-log-mb int     maximum log file size in MB before cleanup (default 100)
      --reset-plugins      also reset the plugins directory (removes user-installed VMs)
      --stale-runs         clean up stale run directories from previous sessions
```

<a id="lux-network-describe"></a>
### lux network describe

Show detailed information about a network including:
- Genesis configuration
- C-chain allocations and precompiles
- Initial validators/stakers
- Network parameters

Network must be one of: mainnet, testnet, devnet, local

**Usage:**

```bash
lux network describe <network>
```

<a id="lux-network-monitor"></a>
### lux network monitor

The monitor command shows real-time network status updates.

OVERVIEW:

  Continuously monitors network health, validator nodes, endpoints, and custom chains.
  Updates display every second by default, showing live statistics.

OPTIONS:

  --interval, -i   Update interval in seconds (default: 1)
  --format         Output format (full, summary, chains, nodes)
  --compact        Use compact output format

EXAMPLES:

  # Monitor with default 1-second updates
  lux network monitor

  # Monitor with 5-second updates
  lux network monitor --interval 5

  # Monitor with compact format
  lux network monitor --compact

  # Monitor only chain status
  lux network monitor --format chains

**Usage:**

```bash
lux network monitor [flags]
```

**Flags:**

```
      --compact         use compact output format
      --format string   output format (full, summary, chains, nodes) (default "full")
  -i, --interval int    update interval in seconds (default 1)
  -o, --output string   output format (text, json) (default "text")
```

<a id="lux-network-send"></a>
### lux network send

Send funds on the C-Chain of the running local network.

This command uses a local key (MNEMONIC or --from) to sign a C-Chain
transfer and submit it to the running network's C-Chain RPC.

Examples:
  # Send 100 LUX to a C-Chain address (uses MNEMONIC account 0)
  lux network send --amount 100 --to 0x9011E888251AB053B7bD1cdB598Db4f9DEd94714

  # Send with a specific stored key
  lux network send --amount 25 --to 0x... --from node1

Notes:
  - Amount is in LUX (converted to wei)
  - Requires a running local network (mainnet/testnet/devnet)
  - Source/dest flags are accepted but only C->C is supported right now

**Usage:**

```bash
lux network send [flags]
```

**Flags:**

```
      --amount float    Amount to send in LUX (required)
      --dest string     Destination chain (only C supported) (default "C")
      --from string     Key name to use for signing (default: MNEMONIC account 0)
      --source string   Source chain (only C supported) (default "C")
      --to string       Destination address (C-Chain hex address)
```

<a id="lux-network-snapshot"></a>
### lux network snapshot

The snapshot command allows you to save, load, list, and delete snapshots of your local network state.

Snapshots capture the entire network state including all node data, databases, and configurations.

Commands:
  save <name>      - Save current network state as a named snapshot (Legacy)
  load <name>      - Load a snapshot and restart the network (Legacy)
  list             - List all available snapshots
  delete <name>    - Delete a snapshot
  advanced         - Advanced coordinated snapshots (incremental, squash, etc)

Examples:
  lux network snapshot save my-test-state
  lux network snapshot advanced create my-prod-state --incremental
  lux network snapshot list

**Usage:**

```bash
lux network snapshot
```

<a id="lux-network-snapshot-advanced"></a>
#### lux network snapshot advanced

Advanced snapshot commands for coordinated multi-node snapshots.

Commands:
  create <name>    - Create advanced snapshot of all nodes (base or incremental)
  restore <name>   - Restore network from advanced snapshot
  squash <network> <chain-id> - Squash incrementals into base
  download <name>  - Download from GitHub (placeholder)
  upload <name>    - Upload to GitHub (placeholder)

Examples:
  lux network snapshot advanced create production-backup --incremental
  lux network snapshot advanced restore production-backup
  lux network snapshot advanced squash mainnet 1

**Usage:**

```bash
lux network snapshot advanced
```

<a id="lux-network-snapshot-advanced-create"></a>
##### lux network snapshot advanced create

Create a coordinated snapshot of all nodes in the network.
If --incremental is set, tries to create an incremental backup from the last checkpoint.
Otherwise creates a full base snapshot.

**Usage:**

```bash
lux network snapshot advanced create <name> [flags]
```

**Flags:**

```
      --incremental   Create incremental snapshot if possible
```

<a id="lux-network-snapshot-advanced-download"></a>
##### lux network snapshot advanced download

Download a snapshot from GitHub releases.

This feature will download chunked snapshot files from GitHub releases
and verify SHA256 hashes before restoring.

Note: This is a planned feature. For now, manually download snapshot
chunks and use 'lux network snapshot advanced restore' to restore.

**Usage:**

```bash
lux network snapshot advanced download <name>
```

<a id="lux-network-snapshot-advanced-restore"></a>
##### lux network snapshot advanced restore

Restore network from advanced snapshot

**Usage:**

```bash
lux network snapshot advanced restore <name>
```

<a id="lux-network-snapshot-advanced-squash"></a>
##### lux network snapshot advanced squash

Squashes all incremental snapshots for a specific chain into the base snapshot.
This creates a new base snapshot and removes the incrementals, saving space.

**Usage:**

```bash
lux network snapshot advanced squash <network> <chain-id> <snapshot-name>
```

<a id="lux-network-snapshot-advanced-upload"></a>
##### lux network snapshot advanced upload

Upload a snapshot to GitHub releases.

This feature will upload chunked snapshot files (99MB each) to GitHub
releases for distribution.

Note: This is a planned feature. For now, manually upload the snapshot
chunks from ~/.lux/snapshots/<name>/.

**Usage:**

```bash
lux network snapshot advanced upload <name>
```

<a id="lux-network-snapshot-delete"></a>
#### lux network snapshot delete

The snapshot delete command removes a saved snapshot from disk.

Example:
  lux network snapshot delete my-test-state

**Usage:**

```bash
lux network snapshot delete <name>
```

<a id="lux-network-snapshot-list"></a>
#### lux network snapshot list

The snapshot list command displays all saved snapshots with their metadata.

Example:
  lux network snapshot list

**Usage:**

```bash
lux network snapshot list
```

<a id="lux-network-snapshot-load"></a>
#### lux network snapshot load

The snapshot load command loads a previously saved snapshot.

If the network is currently running, it will be stopped first. The snapshot
data will be copied to the active network directory and the network will be restarted.

Example:
  lux network snapshot load my-test-state

**Usage:**

```bash
lux network snapshot load <name> [flags]
```

**Flags:**

```
      --network-type string   network type to load snapshot into (mainnet, testnet, devnet, custom)
```

<a id="lux-network-snapshot-save"></a>
#### lux network snapshot save

The snapshot save command saves the current network state to a named snapshot.

Uses native BadgerDB backup API for reliable, consistent snapshots. Works on both
running and stopped networks - BadgerDB supports concurrent reads during backup.

Use --incremental to create a smaller incremental backup if a previous backup exists.
Incremental backups only store changes since the last backup, saving significant space.

Example:
  lux network snapshot save my-test-state              # Full backup (works while running)
  lux network snapshot save my-backup --incremental    # Incremental backup (smaller, faster)

**Usage:**

```bash
lux network snapshot save <name> [flags]
```

**Flags:**

```
      --incremental           create incremental backup (smaller, faster if previous backup exists)
      --network-type string   network type to snapshot (mainnet, testnet, devnet, custom)
```

<a id="lux-network-start"></a>
### lux network start

The network start command starts a local, multi-node Lux network.

NETWORK TYPES (choose one, required):

  --mainnet, -m    Production mainnet with 3 validators (port 9630)
                   - Network ID: 1
                   - HTTP API: ports 9630-9638
                   - Use for mainnet testing and development

  --testnet, -t    Test network with 3 validators (port 9640)
                   - Network ID: 2
                   - HTTP API: ports 9640-9648
                   - Use for testnet deployment testing

  --devnet, -d     Development network with 3 validators (port 9650)
                   - Network ID: 3
                   - HTTP API: ports 9650-9658
                   - Use for rapid local development

  --dev            Dev mode (port 8545) - Anvil/Hardhat compatible
                   - Single-node: K=1 consensus, instant finality
                   - Multi-node: --dev --num-validators=3 (turbo profile)
                   - Primary chains: C/P/X (Contract/Platform/Exchange)
                   - App chain VMs: A(AI) B(Bridge) D(DEX) G(Graph) I(Identity)
                             K(Key) O(Oracle) Q(Quantum) R(Relay) T(Threshold) Z(ZK)
                   - Set MNEMONIC to auto-fund derived accounts

OPTIONS:

  --num-validators    Number of validator nodes (default: 3)
                      With --dev: 1 = K=1 single-node, >1 = turbo multi-node
  --node-path         Path to custom luxd binary
  --node-version      luxd version to use (default: latest)
  --snapshot-name     Resume from named snapshot
  --port              Base port for APIs (overrides defaults)
  --profile           Consensus profile: standard, fast, turbo (default: auto)

EXAMPLES:

  # Start mainnet (3 validators, port 9630)
  lux network start --mainnet
  lux network start -m

  # Start testnet with custom validator count
  lux network start --testnet --num-validators 2

  # Start devnet (most common for development)
  lux network start --devnet

  # Start single-node dev mode for rapid testing (K=1)
  lux network start --dev

  # Start 3-validator dev mode with turbo consensus
  lux network start --dev --num-validators=3

  # 3-node dev with mnemonic-funded accounts
  export LIGHT_MNEMONIC="light light light light light light light light light light light energy"
  lux network start --dev --num-validators=3

  # Use custom luxd binary
  lux network start --devnet --node-path ~/work/lux/node/build/luxd

NOTES:

  - Only one network type can run at a time
  - Each network type uses different ports to avoid conflicts
  - Network data is stored in ~/.lux/networks/<type>
  - Use 'lux network status' to verify the network is running
  - Use 'lux network stop' to stop and save a snapshot
  - Admin APIs are enabled by default for chain deployment

TYPICAL WORKFLOW:

  1. Start network:    lux network start --devnet
  2. Deploy chain:     lux chain deploy mychain
  3. Test your dapp:   (connect to http://localhost:9650/v1/chain/C/rpc)
  4. Stop network:     lux network stop

**Usage:**

```bash
lux network start [flags]
```

**Flags:**

```
      --archive-path string        path to BadgerDB archive database (enables dual-database mode)
      --archive-shared             enable shared read-only access to archive database
      --blockchain-id string       blockchain ID for the loaded state
      --chain-id string            chain ID for the loaded state
      --chain-state-path string    path to existing chain database to load
      --db-backend string          database backend to use (pebble, leveldb, or badgerdb)
      --dev                        single-node dev mode with K=1 consensus
  -d, --devnet                     start devnet with 3 validators (port 9650)
      --import-chain-data string   path to import blockchain data from another chain into C-Chain
      --k8s string                 deploy to Kubernetes cluster (use kubeconfig context name)
      --k8s-image string           Docker image for K8s deployment (default "ghcr.io/luxfi/node:latest")
  -l, --local                      start 3-node localnet on K8s (operator-native, light mnemonic)
  -m, --mainnet                    start mainnet with 3 validators (port 9630)
      --node-path string           path to local luxd binary (overrides --node-version)
      --node-version string        use this version of node (ex: v1.17.12) (default "latest")
      --num-validators int         number of validators to start (default 3)
      --port int                   base port for node APIs (each node uses 2 ports: HTTP and staking) (default 9630)
      --profile string             performance profile: standard, fast, turbo (default: per-network)
      --snapshot-name string       name of snapshot to use to start the network from (default "default-20251225")
      --state-path string          path to existing state directory (e.g., ~/work/lux/state/chaindata/lux-mainnet-96369)
  -t, --testnet                    start testnet with 3 validators (port 9640)
```

<a id="lux-network-status"></a>
### lux network status

The improved network status command shows detailed information about running networks.

OVERVIEW:

  Displays network health, validator nodes, endpoints, and custom chains.
  Uses clean, structured output suitable for scripting and human reading.

FORMAT OPTIONS:

  --format full     Show full detailed status (default)
  --format summary  Show only network summary
  --format chains   Show only chain status
  --format nodes    Show only node status
  --compact         Use compact output format

EXAMPLES:

  # Show full status
  lux network status-new

  # Show only chain status
  lux network status-new --format chains

  # Show compact summary
  lux network status-new --compact

OUTPUT FORMAT:

  status  mainnet  up   grpc=8369  nodes=5  vms=1  controller=on
  status  testnet  up   grpc=8368  nodes=5  vms=1  controller=on
  
  mainnet nodes
  node   http                         version       peers  uptime     ok
  1      http://127.0.0.1:9630        luxd/1.22.75   12     01:22:10   yes
  
  mainnet chains (heights)
  chain  kind  height     block_time           rpc_ok  latency
  p      p     12345      2026-01-06 14:27:03   yes     18ms
  c      evm   218        2026-01-06 14:27:01   yes     16ms

**Usage:**

```bash
lux network status [flags]
```

**Flags:**

```
      --compact         use compact output format
      --format string   output format (full, summary, chains, nodes) (default "full")
  -o, --output string   output format (text, json, yaml, wide) (default "text")
      --verbose         show verbose progress information
```

<a id="lux-network-stop"></a>
### lux network stop

The network stop command gracefully shuts down the running network and saves state.

SNAPSHOT BEHAVIOR:

  By default, the network saves its state to a snapshot when stopping. This includes:
  - Blockchain state (C-Chain, P-Chain, X-Chain, deployed chains)
  - Validator state
  - Database contents

  The snapshot allows you to resume exactly where you left off with:
    lux network start --<type> --snapshot-name <name>

OPTIONS:

  --mainnet           Stop mainnet network (network-id=1)
  --testnet           Stop testnet network (network-id=2)
  --devnet            Stop devnet network (network-id=3)
  --network-id        Stop network by ID (for custom networks)
  --snapshot-name     Name for the snapshot (default: default-snapshot)
  --force             Force stop without confirmation (use with caution)

SAFETY CHECKS:

  If multiple networks are running, you MUST specify which one to stop:
    lux network stop --devnet
    lux network stop --testnet

  Stopping mainnet or testnet requires explicit --mainnet/--testnet flag.
  This prevents accidental disruption of production deployments.

EXAMPLES:

  # Stop the running network (when only one is running)
  lux network stop

  # Stop specific network type (required when multiple running)
  lux network stop --devnet
  lux network stop --testnet

  # Stop with named snapshot
  lux network stop --devnet --snapshot-name my-snapshot

  # Resume from snapshot later
  lux network start --devnet --snapshot-name my-snapshot

NOTES:

  - Snapshots preserve ALL network state including deployed chains
  - Chain configurations (in ~/.lux/chains/) are NOT affected
  - Use 'lux network clean' to wipe runtime data completely
  - Only the specified network type is stopped (others remain running)
  - Use 'lux dev stop' for the dev mode node (separate from network command)

SNAPSHOT vs CLEAN:

  lux network stop    - Saves state for resuming later
  lux network clean   - Deletes runtime data, preserves chain configs

**Usage:**

```bash
lux network stop [flags]
```

**Flags:**

```
      --cleanup                clean up old log files and stale run directories
      --devnet                 stop devnet network (network-id=3)
      --force                  force stop without confirmation (use with caution for mainnet/testnet)
      --mainnet                stop mainnet network (network-id=1)
      --network-id uint32      stop network by ID (for custom networks)
      --snapshot-name string   name of snapshot to use to save network state into (default "default-20251225")
      --testnet                stop testnet network (network-id=2)
```

<a id="lux-node"></a>
## lux node

Commands for managing luxd nodes — locally and on Kubernetes.

LOCAL COMMANDS:
  link        Symlink a luxd binary to ~/.lux/bin/luxd
  join        Join this box to the node pool (run a validator anywhere)

KUBERNETES COMMANDS (via Helm chart):
  deploy      Deploy/update luxd via Helm (single source of truth)
  upgrade     Rolling upgrade with zero downtime (partition-based)
  status      Show pod status, images, and health
  logs        Stream logs from a luxd pod
  rollback    Revert to previous StatefulSet revision

The deploy command uses the canonical Helm chart at ~/work/lux/devops/charts/lux/
(configurable via --chart-path or $CHART_PATH). All other k8s commands use
the Kubernetes API directly for fast read/write operations.

All k8s commands require one of --mainnet, --testnet, --devnet, or --namespace.
Use --context to target a specific kubeconfig context.

EXAMPLES:
  # Local
  lux node link --auto

  # Deploy via Helm (uses canonical chart + values-{network}.yaml)
  lux node deploy --mainnet
  lux node deploy --testnet --set image.tag=luxd-v1.23.15

  # Zero-downtime upgrade (partition-based, per-pod health checks)
  lux node upgrade --mainnet --image ghcr.io/luxfi/node:v1.23.6

  # Check status
  lux node status --mainnet

  # Stream logs
  lux node logs --mainnet luxd-0 -f

  # Rollback
  lux node rollback --mainnet

**Usage:**

```bash
lux node
```

<a id="lux-node-deploy"></a>
### lux node deploy

Deploys the luxd Helm chart to Kubernetes using helm upgrade --install.

Uses the canonical Helm chart from ~/work/lux/devops/charts/lux/ as the
single source of truth. This ensures the CLI creates identical deployments
to the deploy-all.sh script — same startup.sh, staking keys, bootstrap
nodes, upgrade-file-content, chain configs, and per-pod services.

CHART DISCOVERY (in order):
  1. --chart-path flag
  2. $CHART_PATH environment variable
  3. ~/work/lux/devops/charts/lux/

EXAMPLES:
  lux node deploy --mainnet
  lux node deploy --testnet --set image.tag=luxd-v1.23.15
  lux node deploy --devnet --replicas 3
  lux node deploy --mainnet --chart-path /path/to/chart
  lux node deploy --mainnet --dry-run

**Usage:**

```bash
lux node deploy [flags]
```

**Flags:**

```
      --chart-path string   path to Helm chart (default: auto-detect)
      --context string      kubeconfig context to use
      --devnet              target lux-devnet namespace
      --dry-run             helm dry-run mode (template only, no apply)
      --image string        override image tag (shorthand for --set image.tag=TAG)
      --mainnet             target lux-mainnet namespace
      --namespace string    k8s namespace (overrides network flags)
      --replicas int32      override replica count (0 = use chart default)
      --set stringArray     additional Helm --set overrides (repeatable)
      --testnet             target lux-testnet namespace
```

<a id="lux-node-join"></a>
### lux node join

Run this on any box and it joins the pool; the operator schedules a
compact validator onto it and rebalances the fleet. Drain a box and its
validator reschedules onto the others — run a node anywhere, the network spreads.

  # First box — become the control plane and hold history:
  lux node join --init --role archive

  # Any other box — join and run a compact validator:
  lux node join --server https://<control>:6443 --token <token>

Off-LAN boxes: put them on one tailnet first (tailscale up, or hanzozt) and pass
the tailnet IP as --server, so "anywhere" really means anywhere. The role is a
node label (lux.cloud/validator|archive=true) the NodeFleet schedules against —
see operator/spec/examples/nodefleet-lab.yaml.

**Usage:**

```bash
lux node join [flags]
```

**Flags:**

```
      --init            make THIS box the pool's control plane (k3s server)
      --name string     node name (default: hostname)
      --print           print the k3s command instead of running it
      --role string     node role: validator (compact) or archive (full history) (default "validator")
      --server string   control-plane API URL, e.g. https://<ip>:6443 (agent mode)
      --token string    node token from the control plane (agent mode)
```

<a id="lux-node-link"></a>
### lux node link

Link luxd binary for the CLI to use.

Creates ~/.lux/bin directory if needed and symlinks the luxd binary.

PRIORITY ORDER for binary lookup:
  1. Command-line flags (--node-path)
  2. ~/.lux/bin/luxd (this symlink)
  3. Environment variable (NODE_PATH)
  4. Config file settings
  5. PATH lookup
  6. Relative paths from CLI location

EXAMPLES:

  # Link luxd (auto-detect from ../node/bin/luxd)
  lux node link --auto

  # Link specific path
  lux node link /path/to/luxd

**Usage:**

```bash
lux node link [path] [flags]
```

**Flags:**

```
      --auto   auto-detect luxd from standard locations
```

<a id="lux-node-logs"></a>
### lux node logs

Streams logs from a specific luxd pod in the StatefulSet.

If no pod name is given, defaults to luxd-0.

EXAMPLES:
  lux node logs --mainnet
  lux node logs --mainnet luxd-2
  lux node logs --mainnet luxd-0 -f
  lux node logs --testnet --tail 100

**Usage:**

```bash
lux node logs [pod-name] [flags]
```

**Flags:**

```
      --context string     kubeconfig context to use
      --devnet             target lux-devnet namespace
  -f, --follow             follow log output
      --mainnet            target lux-mainnet namespace
      --namespace string   k8s namespace (overrides network flags)
      --tail int           number of recent lines to show (default 200)
      --testnet            target lux-testnet namespace
```

<a id="lux-node-rollback"></a>
### lux node rollback

Reverts the luxd StatefulSet to its previous ControllerRevision.

By default rolls back to the immediately previous revision. Use --revision
to target a specific revision number.

After rollback, performs the same partition-based rolling update to ensure
zero downtime — pods are updated one at a time with health checks.

EXAMPLES:
  lux node rollback --mainnet
  lux node rollback --mainnet --revision 3
  lux node rollback --testnet --timeout 10m

**Usage:**

```bash
lux node rollback [flags]
```

**Flags:**

```
      --context string     kubeconfig context to use
      --devnet             target lux-devnet namespace
      --mainnet            target lux-mainnet namespace
      --namespace string   k8s namespace (overrides network flags)
      --revision int       target revision number (0 = previous)
      --testnet            target lux-testnet namespace
      --timeout duration   max time to wait per pod (default 5m0s)
```

<a id="lux-node-status"></a>
### lux node status

Displays the current state of the luxd Kubernetes deployment.

Shows:
  - StatefulSet metadata (replicas, revision, update strategy)
  - Per-pod status (ready, image, restarts, age)
  - LoadBalancer external IP
  - Revision history for rollback

EXAMPLES:
  lux node status --mainnet
  lux node status --testnet
  lux node status --namespace my-custom-ns

**Usage:**

```bash
lux node status [flags]
```

**Flags:**

```
      --context string     kubeconfig context to use
      --devnet             target lux-devnet namespace
      --mainnet            target lux-mainnet namespace
      --namespace string   k8s namespace (overrides network flags)
      --testnet            target lux-testnet namespace
```

<a id="lux-node-upgrade"></a>
### lux node upgrade

Performs a partition-based rolling upgrade of the luxd StatefulSet.

Upgrades one pod at a time (highest ordinal first), waiting for each pod
to become ready and stable before proceeding. This ensures C-chain RPC
clients experience no downtime since a quorum of validators remains
available throughout the upgrade.

PROCESS:
  1. Validates the new image exists and differs from current
  2. Updates the StatefulSet pod template with new image
  3. Sets partition = replicas (no pods restart yet)
  4. Lowers partition one at a time (pod N-1, N-2, ... 0)
  5. After each pod restart: waits for readiness + stability period
  6. If any pod fails health check, stops and prints rollback command

EXAMPLES:
  lux node upgrade --mainnet --image ghcr.io/luxfi/node:v1.23.5
  lux node upgrade --testnet --image ghcr.io/luxfi/node:v1.23.5 --stability-wait 60s
  lux node upgrade --devnet --image ghcr.io/luxfi/node:v1.23.5 --dry-run

**Usage:**

```bash
lux node upgrade [flags]
```

**Flags:**

```
      --context string            kubeconfig context to use
      --devnet                    target lux-devnet namespace
      --dry-run                   show what would happen without making changes
      --evm-version string        EVM plugin version to update in init container
      --force                     proceed even if image is the same
      --health-timeout duration   max time to wait for a pod to become ready (default 5m0s)
      --image string              new container image (required)
      --mainnet                   target lux-mainnet namespace
      --namespace string          k8s namespace (overrides network flags)
      --stability-wait duration   wait time after pod ready before proceeding (default 30s)
      --testnet                   target lux-testnet namespace
```

<a id="lux-primary"></a>
## lux primary

The primary command suite provides a collection of tools for interacting with the
Primary Network

**Usage:**

```bash
lux primary
```

<a id="lux-primary-addValidator"></a>
### lux primary addValidator

Issues an AddPermissionlessValidatorTx for a node identity.

Create the identity first — it prints every value this command needs:

  lux key staker ~/.luxd/staking
  lux primary addValidator --mainnet \
      --node-id NodeID-... --public-key 0x... --proof-of-possession 0x... \
      --stake 2000000000 --duration 336h

The stake is in nLUX (1 LUX = 1e9 nLUX) and the chain reads the start time
from its own clock, so --duration measures from when the tx is accepted.

**Usage:**

```bash
lux primary addValidator [flags]
```

**Flags:**

```
      --cluster string               operate on the given cluster
      --delegation-fee uint32        share of delegation rewards the validator keeps, out of 1,000,000 (default 20000)
      --devnet                       operate on a devnet network
      --duration duration            how long the validator stays in the set
      --endpoint string              use the given endpoint for network operations
  -k, --key string                   name of the stored key that pays and owns the rewards
  -g, --ledger                       sign with a ledger device instead of a stored key
      --ledger-addrs strings         ledger addresses to search
  -m, --mainnet                      operate on mainnet
      --node-id string               NodeID of the validator
      --proof-of-possession string   BLS proof of possession of the validator
      --public-key string            BLS public key of the validator
      --reward-address string        P-Chain address to own the staking reward (default: the paying key)
      --stake uint                   amount to stake, in nLUX
  -t, --testnet                      operate on testnet
```

<a id="lux-primary-describe"></a>
### lux primary describe

The chain describe command prints details of the primary network configuration to the console.

**Usage:**

```bash
lux primary describe
```

<a id="lux-ps"></a>
## lux ps

Probes the resolved httpPort for every (network, env) tuple in the
registry and reports up/down + networkID match.

**Usage:**

```bash
lux ps
```

<a id="lux-rpc"></a>
## lux rpc

Make JSON-RPC calls to a Lux node.

Examples:
  # Get P-Chain height
  lux rpc call --method platform.getHeight --endpoint http://localhost:9630/v1/chain/P

  # Get blockchains with params
  lux rpc call --method platform.getBlockchains --params '{}' --endpoint http://localhost:9630/v1/chain/P

  # Create blockchain
  lux rpc call --method platform.createBlockchain \
    --params '{"vmID":"...", "name":"mychain", "genesis":"..."}' \
    --endpoint http://localhost:9630/v1/chain/P


<a id="lux-rpc-call"></a>
### lux rpc call

Make a JSON-RPC call to the specified endpoint with the given method and parameters

**Usage:**

```bash
lux rpc call [flags]
```

**Flags:**

```
      --endpoint string   RPC endpoint URL (default "http://localhost:9630/v1/chain/P")
      --method string     RPC method to call (required)
      --params string     JSON params object (optional)
      --timeout int       Request timeout in seconds (default 30)
```

<a id="lux-rpc-transfer"></a>
### lux rpc transfer

Transfer LUX across chains using atomic export/import.

Supported:
  - P -> C
  - X -> C
  - C -> P
  - C -> X

Example:
  lux rpc transfer --from-chain P --to-chain C --to 0x9011... --amount 10


**Usage:**

```bash
lux rpc transfer [flags]
```

**Flags:**

```
      --amount float        Amount to transfer in LUX
      --from string         Key name to use (default: MNEMONIC account 0)
      --from-chain string   Source chain: P, X, or C (default "P")
      --rpc-url string      Base RPC URL (default: RPC_URL or running network endpoint)
      --to string           Destination address (C-Chain hex for C, bech32 for P/X)
      --to-chain string     Destination chain: P, X, or C (default "C")
      --wait                Wait for export acceptance before import (default true)
```

<a id="lux-rt"></a>
## lux rt

The rt (corona) command provides tools for Corona threshold signing,
a post-quantum threshold signature scheme using Module-LWE.

Corona is part of the triple consensus (BLS + Corona + ML-DSA) used
by Lux validators. It provides threshold signatures where t-of-n parties
can cooperatively produce a valid signature without reconstructing the
full private key.

KEY PROPERTIES:

  - Post-quantum secure (lattice-based)
  - t-of-n threshold without trusted dealer (via DKG)
  - Proactive resharing for key rotation
  - Compatible with QuasarCert attestations

**Usage:**

```bash
lux rt
```

<a id="lux-rt-keygen"></a>
### lux rt keygen

Generate t-of-n threshold key shares for Corona signing.

Examples:
  lux rt keygen --threshold 3 --parties 5 --output ./shares/
  lux rt keygen --threshold 2 --parties 3

**Usage:**

```bash
lux rt keygen [flags]
```

**Flags:**

```
      --output string   Output directory for key shares (default: current dir)
      --parties int     Total number of parties (n)
      --threshold int   Signing threshold (t)
```

<a id="lux-rt-reshare"></a>
### lux rt reshare

Reshare existing key shares to a new committee configuration.

Proactive resharing allows rotating key shares without changing the
group public key. Used for committee membership changes.

Examples:
  lux rt reshare --old-shares ./old/ --new-threshold 3 --new-parties 7

**Usage:**

```bash
lux rt reshare [flags]
```

**Flags:**

```
      --new-parties int     New total number of parties
      --new-threshold int   New signing threshold
      --old-shares string   Directory containing old key shares
```

<a id="lux-rt-sign"></a>
### lux rt sign

Initiate a Corona threshold signing session.

Requires t-of-n key holders to participate in the 2-round signing protocol:
  Round 1: Each party broadcasts D matrix + MACs
  Round 2: Each party broadcasts z share
  Finalize: Any party aggregates into final signature

Examples:
  lux rt sign --message "hello" --share ./shares/share-0.json
  lux rt sign --tx-file unsigned.tx --share ./shares/share-0.json

**Usage:**

```bash
lux rt sign [flags]
```

**Flags:**

```
      --message string   Message to sign (hex or string)
      --share string     Path to key share file
```

<a id="lux-rt-verify"></a>
### lux rt verify

Verify a Corona threshold signature against the group public key.

Examples:
  lux rt verify --signature sig.json --message "hello" --group-key group.json

**Usage:**

```bash
lux rt verify [flags]
```

**Flags:**

```
      --group-key string   Path to group key file
      --message string     Message that was signed
      --signature string   Path to signature file
```

<a id="lux-self"></a>
## lux self

Commands for managing the Lux CLI installation.

Similar to nvm for Node.js, this allows you to:
- Link development builds to ~/.lux/bin/
- Install specific versions
- Switch between versions
- Self-update

EXAMPLES:

  # Link current binary to ~/.lux/bin/lux
  lux self link

  # Install a specific version
  lux self install v1.22.5

  # List installed versions
  lux self list

  # Use a specific version
  lux self use v1.22.5

**Usage:**

```bash
lux self
```

<a id="lux-self-install"></a>
### lux self install

Install a specific version of the Lux CLI.

Downloads and installs the specified version to ~/.lux/versions/<version>/.

If no version is specified, installs the latest version.

EXAMPLES:

  # Install latest version
  lux self install

  # Install specific version
  lux self install v1.22.5

**Usage:**

```bash
lux self install [version]
```

<a id="lux-self-link"></a>
### lux self link

Link the currently running CLI binary to ~/.lux/bin/lux.

This makes the development build available system-wide when ~/.lux/bin
is in your PATH.

EXAMPLES:

  # Link current binary
  lux self link

**Usage:**

```bash
lux self link
```

<a id="lux-self-list"></a>
### lux self list

List all installed versions of the Lux CLI.

Shows installed versions and indicates which one is currently active.

EXAMPLES:

  lux self list

**Usage:**

```bash
lux self list
```

<a id="lux-self-use"></a>
### lux self use

Switch to a specific installed version of the Lux CLI.

Updates the ~/.lux/bin/lux symlink to point to the specified version.

Use 'dev' to switch back to your development build.

EXAMPLES:

  # Switch to a specific version
  lux self use v1.22.5

  # Switch back to development build
  lux self use dev

**Usage:**

```bash
lux self use <version>
```

<a id="lux-snap"></a>
## lux snap

Snapshots the resolved data-dir to <snapshotDir>/<snapshotName>.tar.zst.

The node should be stopped first (`lux down`) — snapshotting a
live database produces corrupt archives. Use --live to override the
safety check.

Examples:
  lux snap zoo/devnet              # → ~/work/lux/snapshots/200202-zoo-devnet-with-contracts.tar.zst
  lux snap lux/mainnet --tag pre-merge

**Usage:**

```bash
lux snap <network>/<env> [flags]
```

**Flags:**

```
      --live         allow snapshot while node is still up (unsafe)
      --tag string   override the trailing token in the snapshot name (default: with-contracts)
```

<a id="lux-snapshot"></a>
## lux snapshot

The snapshot command creates native incremental backups of running networks.

This uses BadgerDB's native backup API for:
  - Incremental backups (only changes since last backup)
  - Consistent snapshots (atomic database state)
  - Fast restore times
  - Smaller backup sizes (zstd compressed)

USAGE:

  # Create snapshot of running network (auto-detects which network)
  lux snapshot

  # Create snapshot of specific network
  lux snapshot --mainnet
  lux snapshot --testnet

  # Create snapshot with custom name
  lux snapshot --name my-backup

  # Force full backup (not incremental)
  lux snapshot --full

  # Restore from snapshot
  lux snapshot restore my-backup

  # List available snapshots
  lux snapshot list

INCREMENTAL BACKUPS:

  By default, snapshots are incremental - they only include data that changed
  since the last backup. This makes them much smaller and faster.

  First backup: Full backup (~90MB compressed for fresh network)
  Subsequent:   Incremental (~1-10MB for typical changes)

  Use --full to force a complete backup.

**Usage:**

```bash
lux snapshot [flags]
```

**Flags:**

```
      --devnet        snapshot devnet network
      --full          create full backup instead of incremental
      --mainnet       snapshot mainnet network
      --name string   snapshot name (default: <network>-<date>)
      --testnet       snapshot testnet network
```

<a id="lux-snapshot-clean"></a>
### lux snapshot clean

Clean up old snapshots, large log files, and stale run directories.

This command frees disk space by removing:
  - Old backup directories
  - Large netrunner log files (>100MB)
  - Stale run directories from previous sessions

EXAMPLES:

  # Preview what would be cleaned
  lux snapshot clean --dry-run

  # Clean everything, keep last 3 snapshots
  lux snapshot clean --keep 3

  # Clean all old data
  lux snapshot clean

**Usage:**

```bash
lux snapshot clean [flags]
```

**Flags:**

```
      --dry-run    show what would be cleaned without deleting
      --keep int   number of recent snapshots to keep (default 3)
```

<a id="lux-snapshot-list"></a>
### lux snapshot list

List available snapshots

**Usage:**

```bash
lux snapshot list
```

<a id="lux-snapshot-restore"></a>
### lux snapshot restore

Restore a network from a previously created snapshot.

The network must be stopped before restoring. After restore, start the
network with 'lux network start'.

EXAMPLES:

  # Restore from snapshot
  lux snapshot restore my-backup

  # Restore mainnet snapshot
  lux snapshot restore mainnet-2026-01-19 --mainnet

**Usage:**

```bash
lux snapshot restore [name] [flags]
```

**Flags:**

```
      --devnet    restore to devnet
      --mainnet   restore to mainnet
      --testnet   restore to testnet
```

<a id="lux-status"></a>
## lux status

The improved network status command shows detailed information about running networks.

OVERVIEW:

  Displays network health, validator nodes, endpoints, and custom chains.
  Uses clean, structured output suitable for scripting and human reading.

FORMAT OPTIONS:

  --format full     Show full detailed status (default)
  --format summary  Show only network summary
  --format chains   Show only chain status
  --format nodes    Show only node status
  --compact         Use compact output format

EXAMPLES:

  # Show full status
  lux network status-new

  # Show only chain status
  lux network status-new --format chains

  # Show compact summary
  lux network status-new --compact

OUTPUT FORMAT:

  status  mainnet  up   grpc=8369  nodes=5  vms=1  controller=on
  status  testnet  up   grpc=8368  nodes=5  vms=1  controller=on
  
  mainnet nodes
  node   http                         version       peers  uptime     ok
  1      http://127.0.0.1:9630        luxd/1.22.75   12     01:22:10   yes
  
  mainnet chains (heights)
  chain  kind  height     block_time           rpc_ok  latency
  p      p     12345      2026-01-06 14:27:03   yes     18ms
  c      evm   218        2026-01-06 14:27:01   yes     16ms

**Usage:**

```bash
lux status [flags]
```

**Flags:**

```
      --compact         use compact output format
      --format string   output format (full, summary, chains, nodes) (default "full")
  -o, --output string   output format (text, json, yaml, wide) (default "text")
      --verbose         show verbose progress information
```

<a id="lux-tui"></a>
## lux tui

Launch the interactive terminal UI for monitoring and managing
the Lux blockchain stack.

The TUI provides a dashboard with tabs for:
  - Dashboard:   Network overview and health
  - Nodes:       Node status and management
  - Chains:      Chain status and block heights
  - Validators:  Validator set and uptime
  - Logs:        Live log streaming

NAVIGATION:

  Tab/Arrow   Navigate between tabs
  1-5         Jump to tab by number
  r           Refresh data
  q           Quit

Examples:
  lux tui
  lux tui --endpoint http://localhost:9640

**Usage:**

```bash
lux tui [flags]
```

**Flags:**

```
      --endpoint string   Lux node API endpoint (default: http://127.0.0.1:9630)
```

<a id="lux-up"></a>
## lux up

Boots the luxd node for <network>/<env> in K=1 PoA mode.

The (network, env) tuple identifies one L1 instance globally. Every
parameter — port, networkID, dataDir, genesisFile — is derived from that
network's chain.yaml under $LUX_NETWORK_PATH. Foreground: `lux up`
blocks until the node exits or Ctrl-C is pressed.

Examples:
  lux up zoo/localnet                 # K=1 dev node for Zoo, networkID 200203
  lux up lux/devnet                   # K=1 dev node for Lux, networkID 3
  lux up hanzo/testnet --clean        # wipe state then boot

Stop with: lux down <network>/<env>

**Usage:**

```bash
lux up <network>/<env> [flags]
```

**Flags:**

```
      --automine string    auto-mine interval (e.g., '1s'); empty = instant
      --clean              remove data-dir before boot (fresh genesis)
      --node-path string   path to luxd binary (auto-detected if empty)
```

<a id="lux-update"></a>
## lux update

Check if an update is available, and prompt the user to install it

**Usage:**

```bash
lux update [flags]
```

**Flags:**

```
  -c, --confirm   Assume yes for installation
```

<a id="lux-validator"></a>
## lux validator

The validator command suite provides a collection of tools for managing validator
balance on P-Chain.

Validator's balance is used to pay for continuous fee to the P-Chain. When this Balance reaches 0, 
the validator will be considered inactive and will no longer participate in validating the L1

**Usage:**

```bash
lux validator
```

<a id="lux-validator-getBalance"></a>
### lux validator getBalance

This command gets the remaining validator P-Chain balance that is available to pay
P-Chain continuous fee

**Usage:**

```bash
lux validator getBalance [flags]
```

**Flags:**

```
      --l1 string              name of L1
      --node-id string         node ID of the validator
      --validation-id string   validation ID of the validator
```

<a id="lux-validator-increaseBalance"></a>
### lux validator increaseBalance

This command increases the validator P-Chain balance

**Usage:**

```bash
lux validator increaseBalance [flags]
```

**Flags:**

```
      --balance float          amount of LUX to increase validator's balance by
  -k, --key string             select the key to use [testnet/devnet deploy only]
      --l1 string              name of L1 (to increase balance of bootstrap validators only)
      --node-id string         node ID of the validator
      --validation-id string   validationIDStr of the validator
```

<a id="lux-validator-list"></a>
### lux validator list

This command gets a list of the validators of the L1

**Usage:**

```bash
lux validator list [blockchainName]
```

<a id="lux-vm"></a>
## lux vm

Commands for installing, linking, and managing VM plugins.

VM plugins are stored as symlinks in ~/.lux/plugins/<vmid>.
The VMID is calculated from the VM name (padded to 32 bytes, CB58 encoded).

Examples:
  lux vm link lux-evm --path ~/work/lux/evm/build/evm
  lux vm status
  lux vm unlink lux-evm
  lux vm reload

**Usage:**

```bash
lux vm
```

<a id="lux-vm-install"></a>
### lux vm install

Install a VM plugin from GitHub releases.

Downloads the latest (or specified) version from GitHub releases and installs it
to ~/.lux/plugins/packages/<org>/<name>/<version>/.

Package format: <org>/<name> or <org>/<name>@<version>

Examples:
  lux vm install luxfi/evm           # Install latest
  lux vm install luxfi/evm@v1.0.0    # Install specific version
  lux vm install myuser/myvm         # Install from any org

**Usage:**

```bash
lux vm install <org/name>[@version] [flags]
```

**Flags:**

```
  -v, --version string   Version to install (default: latest)
```

<a id="lux-vm-link"></a>
### lux vm link

Link a local VM binary to the plugins directory for development.

Creates a proper package entry and VMID symlink for a locally built VM binary.
Use this during development to test local builds with the node.

Package format: <org>/<name> (e.g., luxfi/evm, myuser/myvm)

The binary must exist and be executable.

Examples:
  lux vm link luxfi/evm ~/work/lux/evm/build/evm
  lux vm link luxfi/evm ~/work/lux/evm/build/evm --version v1.2.3-dev
  lux vm link myuser/myvm /path/to/myvm/build/myvm

**Usage:**

```bash
lux vm link <org/name> <path> [flags]
```

**Flags:**

```
  -v, --version string   Version label (default: v0.0.0-local)
```

<a id="lux-vm-reload"></a>
### lux vm reload

Reload VMs on network nodes by calling admin.loadVMs.

This triggers the node to scan the plugins directory and load any new VMs.

Examples:
  lux vm reload
  lux vm reload --endpoint http://127.0.0.1:9630

**Usage:**

```bash
lux vm reload [flags]
```

**Flags:**

```
  -e, --endpoint string    Node endpoint to call admin.loadVMs on (default "http://127.0.0.1:9630")
  -t, --timeout duration   Timeout for the reload request (default 30s)
```

<a id="lux-vm-status"></a>
### lux vm status

Show all linked VMs in the plugins directory.

Displays VMID, name (if known), target path, and whether the target exists.

Examples:
  lux vm status
  lux vm status --json

**Usage:**

```bash
lux vm status [flags]
```

**Flags:**

```
      --json   Output in JSON format
```

<a id="lux-vm-unlink"></a>
### lux vm unlink

Remove a VM symlink from the plugins directory.

Removes the symlink at ~/.lux/plugins/<vmid> for the given VM name.

Examples:
  lux vm unlink lux-evm
  lux vm unlink "Lux EVM"

**Usage:**

```bash
lux vm unlink <vm-name>
```

<a id="lux-warp"></a>
## lux warp

Warp V2 provides cross-chain messaging with post-quantum safety.

This command provides tools for creating, signing, verifying, and relaying
cross-chain messages between Lux networks.

Commands:
  create    Create a new cross-chain message
  sign      Sign a message with validator key
  verify    Verify a signed message
  relay     Start message relayer

**Usage:**

```bash
lux warp
```

<a id="lux-warp-create"></a>
### lux warp create

Create a new Warp message to send between chains.

Example:
  lux warp create --source 0xAA --dest 0xBB --payload "Hello from chain A"

**Usage:**

```bash
lux warp create [flags]
```

**Flags:**

```
  -d, --dest string      Destination chain ID (hex)
  -p, --payload string   Message payload
  -s, --source string    Source chain ID (hex)
```

<a id="lux-warp-relay"></a>
### lux warp relay

Start a Warp message relayer to bridge messages between chains.

The relayer monitors source chains for new messages and delivers them
to destination chains after signature verification.

**Usage:**

```bash
lux warp relay
```

<a id="lux-warp-sign"></a>
### lux warp sign

Sign a cross-chain message with your validator key.

Example:
  lux warp sign --message <hex> --key ~/.lux/staking/signer.key

**Usage:**

```bash
lux warp sign [flags]
```

**Flags:**

```
  -k, --key string       Path to signing key
  -m, --message string   Message to sign (hex)
```

<a id="lux-warp-verify"></a>
### lux warp verify

Verify a Warp message signature against the validator set.

Example:
  lux warp verify --message <hex> --signature <hex>

**Usage:**

```bash
lux warp verify [flags]
```

**Flags:**

```
  -m, --message string     Message to verify (hex)
  -s, --signature string   Signature to verify (hex)
```

<a id="lux-zk"></a>
## lux zk

The zk command provides tools for zero-knowledge proof operations
on the Lux network, including powers-of-tau ceremony management,
proof generation, proof verification, and SRS (Structured Reference String)
management.

These operations integrate with the Z-Chain, Lux's dedicated ZK chain,
for on-chain proof verification via precompiled contracts.

USAGE:

  lux zk ceremony init     Initialize a new powers-of-tau ceremony
  lux zk ceremony contribute  Add randomness to a ceremony
  lux zk ceremony verify   Verify ceremony integrity
  lux zk ceremony export   Export final SRS binary
  lux zk ceremony status   Show ceremony state

  lux zk prove groth16     Generate a Groth16 proof
  lux zk prove plonk       Generate a PLONK proof

  lux zk verify groth16    Verify a Groth16 proof
  lux zk verify plonk      Verify a PLONK proof

  lux zk srs download      Download the official Lux SRS
  lux zk srs verify        Verify a downloaded SRS
  lux zk srs info          Show SRS metadata

<a id="lux-zk-ceremony"></a>
### lux zk ceremony

Manage powers-of-tau ceremonies for generating trusted SRS
(Structured Reference Strings) used in Groth16 and PLONK proof systems.

The ceremony requires the 'ceremony' binary from the Lux node repo.
If not found, build it with:
  cd ~/work/lux/node && go build -o /usr/local/bin/ceremony ./cmd/ceremony/

<a id="lux-zk-ceremony-contribute"></a>
#### lux zk ceremony contribute

Apply a random contribution to the ceremony state. Generates
cryptographically secure random scalars (tau, alpha, beta) and mixes them
into the SRS. The random values are zeroed from memory after use.

**Usage:**

```bash
lux zk ceremony contribute [flags]
```

**Flags:**

```
      --input string         Input ceremony file (required)
      --output string        Output ceremony file (required)
      --participant string   Participant name (required)
```

<a id="lux-zk-ceremony-export"></a>
#### lux zk ceremony export

Export the SRS (Structured Reference String) from a completed and
verified ceremony. The ceremony is verified before export. Output is
uncompressed binary (G1: 64 bytes, G2: 128 bytes per point).

**Usage:**

```bash
lux zk ceremony export [flags]
```

**Flags:**

```
      --input string    Ceremony file to export (required)
      --output string   Output SRS binary file (required)
```

<a id="lux-zk-ceremony-init"></a>
#### lux zk ceremony init

Create a new ceremony state file with initial powers of the BN254
generators. This is the starting point before any contributions.

**Usage:**

```bash
lux zk ceremony init [flags]
```

**Flags:**

```
      --circuit string     Circuit name (required)
      --output string      Output file path (required)
      --participants int   Expected number of participants (default 3)
      --power int          Power of 2 for constraint count (2^power) (default 20)
```

<a id="lux-zk-ceremony-status"></a>
#### lux zk ceremony status

Display the current state of a ceremony file including circuit info, contributions, and participant hashes.

**Usage:**

```bash
lux zk ceremony status [flags]
```

**Flags:**

```
      --input string   Ceremony file to inspect (required)
```

<a id="lux-zk-ceremony-verify"></a>
#### lux zk ceremony verify

Check the consistency of a ceremony state file by verifying:
- TauG1/TauG2 form consistent geometric sequences (pairing checks)
- AlphaG1/BetaG1 use the same tau ratio
- BetaG1 and BetaG2 encode the same beta scalar
- Contribution hash chain integrity
- No points at infinity

**Usage:**

```bash
lux zk ceremony verify [flags]
```

**Flags:**

```
      --input string   Ceremony file to verify (required)
```

<a id="lux-zk-prove"></a>
### lux zk prove

Generate zero-knowledge proofs using the Lux SRS.

Proof generation runs locally using the SRS from the trusted setup ceremony.
The resulting proof can be verified on-chain via the Z-Chain verifier precompiles
or off-chain using 'lux zk verify'.

<a id="lux-zk-prove-groth16"></a>
#### lux zk prove groth16

Generate a Groth16 proof from a circuit, witness, and SRS.

The proving key is derived from the SRS generated by the ceremony.
The witness contains both public inputs and private values.

**Usage:**

```bash
lux zk prove groth16 [flags]
```

**Flags:**

```
      --circuit string   Compiled circuit file path (required)
      --output string    Output proof file path (required)
      --srs string       SRS file path (required)
      --witness string   Witness file path (required)
```

<a id="lux-zk-prove-plonk"></a>
#### lux zk prove plonk

Generate a PLONK proof from a circuit, witness, and SRS.

PLONK uses a universal SRS that works with any circuit of bounded size,
unlike Groth16 which requires a circuit-specific trusted setup.

**Usage:**

```bash
lux zk prove plonk [flags]
```

**Flags:**

```
      --circuit string   Compiled circuit file path (required)
      --output string    Output proof file path (required)
      --srs string       SRS file path (required)
      --witness string   Witness file path (required)
```

<a id="lux-zk-srs"></a>
### lux zk srs

Manage Structured Reference Strings (SRS) for ZK proof systems.

The SRS is the output of the trusted setup ceremony and is required
for both proof generation and verification. The official Lux SRS
is published on the Z-Chain.

<a id="lux-zk-srs-download"></a>
#### lux zk srs download

Download the official SRS binary from the Z-Chain. The file is
verified after download using the ceremony binary.

**Usage:**

```bash
lux zk srs download [flags]
```

**Flags:**

```
      --output string   Output file path (default: ~/.lux/zk/srs.bin)
      --url string      SRS download URL (default "https://api.lux.network/mainnet/v1/chain/Z/srs")
```

<a id="lux-zk-srs-info"></a>
#### lux zk srs info

Display metadata about an SRS file including the number of powers, file size, and SHA-256 hash.

**Usage:**

```bash
lux zk srs info [flags]
```

**Flags:**

```
      --input string   SRS file path (required)
```

<a id="lux-zk-srs-verify"></a>
#### lux zk srs verify

Verify the integrity of a downloaded SRS file by checking its
structure and computing its SHA-256 hash.

**Usage:**

```bash
lux zk srs verify [flags]
```

**Flags:**

```
      --input string   SRS file path (required)
```

<a id="lux-zk-verify"></a>
### lux zk verify

Verify zero-knowledge proofs by calling Z-Chain precompiled contracts.

The Z-Chain provides on-chain verification for Groth16 and PLONK proofs via
precompiled contracts at fixed addresses. This command submits the proof and
public inputs to the verifier precompile and returns the result.

<a id="lux-zk-verify-groth16"></a>
#### lux zk verify groth16

Verify a Groth16 proof against the Z-Chain Groth16 verifier precompile.

Requires the proof file, verification key, and public inputs.
Connects to the Z-Chain RPC endpoint to call the verifier contract.

**Usage:**

```bash
lux zk verify groth16 [flags]
```

**Flags:**

```
      --inputs string   Public inputs file path (required)
      --proof string    Proof file path (required)
      --rpc string      Z-Chain RPC endpoint (default "http://localhost:9630/v1/chain/Z/rpc")
      --vk string       Verification key file path (required)
```

<a id="lux-zk-verify-plonk"></a>
#### lux zk verify plonk

Verify a PLONK proof against the Z-Chain PLONK verifier precompile.

Requires the proof file, verification key, and public inputs.
Connects to the Z-Chain RPC endpoint to call the verifier contract.

**Usage:**

```bash
lux zk verify plonk [flags]
```

**Flags:**

```
      --inputs string   Public inputs file path (required)
      --proof string    Proof file path (required)
      --rpc string      Z-Chain RPC endpoint (default "http://localhost:9630/v1/chain/Z/rpc")
      --vk string       Verification key file path (required)
```

