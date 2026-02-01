# Lux CLI Documentation

**Version**: 1.22.5
**Last Updated**: 2026-01-23

## DigitalOcean Production Nodes (SFO3)

| Network | Reserved IP | Network ID | C-Chain ID | Port | Status |
|---------|-------------|------------|------------|------|--------|
| Mainnet | 209.38.175.130 | 1337 | 1337 | 9630 | Running (dev) |
| Testnet | 24.199.70.106 | 1337 | 1337 | 9640 | Running (dev) |
| Devnet | 24.199.74.128 | 1337 | 1337 | 9650 | Running (dev) |

### RPC Endpoints

```bash
# Mainnet C-Chain RPC
http://209.38.175.130:9630/ext/bc/C/rpc

# Testnet C-Chain RPC
http://24.199.70.106:9640/ext/bc/C/rpc

# Devnet C-Chain RPC
http://24.199.74.128:9650/ext/bc/C/rpc
```

### DNS Records (pending setup)

| Subdomain | Domain | IP |
|-----------|--------|-----|
| api, rpc, explorer | lux.network | 209.38.175.130 |
| @, api, rpc | lux-test.network | 24.199.70.106 |
| @, api, rpc | lux-dev.network | 24.199.74.128 |

## Quick Reference

### Essential Commands

```bash
# Network Management (5-node local network)
lux network start --mainnet    # Start mainnet (5 validators, port 9630)
lux network start --testnet    # Start testnet (5 validators, port 9640)
lux network start --dev        # Single-node dev mode (port 8545, anvil-compatible)
lux network stop               # Stop local network
lux network status             # Check network status
lux network clean              # Remove all network data

# Chain Operations (unified command - supports L1/L2/L3)
lux chain create mychain                      # Create L2 configuration (default)
lux chain create mychain --type=l1            # Create sovereign L1
lux chain create mychain --type=l3            # Create app-specific L3
lux chain deploy mychain --devnet             # Deploy to local network
lux chain list                                # List configured chains
lux chain describe mychain                    # Show chain details

# Chain creation with custom values
lux chain create mychain --evm-chain-id=12345
lux chain create mychain --token-name=MYTOKEN --token-symbol=MTK

# Import/Export
lux chain import --chain=c --path=/path/to/blocks.rlp  # Import RLP blocks to C-Chain
lux chain import --chain=zoo --path=/path/to/zoo.rlp   # Import to subnet

# AMM Trading (LUX_MNEMONIC supported)
lux amm balance                # Check token balances
lux amm status                 # Show AMM contract status
lux amm pools                  # List liquidity pools
lux amm swap --from 0x... --to 0x... --amount 100  # Swap tokens
lux amm quote --from 0x... --to 0x... --amount 100 # Get swap quote

# DEX Trading (High-performance exchange)
lux dex market list            # List all trading markets
lux dex order place            # Place limit/market orders
lux dex pool create            # Create liquidity pools
lux dex perp open              # Open perpetual positions
```

## Command Architecture

The CLI is organized into these main command groups:

| Command | Purpose | Notes |
|---------|---------|-------|
| `lux network` | Local 5-node network management | start, stop, status, clean |
| `lux chain` | Unified chain lifecycle | create, deploy, import, export, list |
| `lux key` | Key management | create, list, export |
| `lux validator` | P-Chain validator balance | |
| `lux amm` | AMM/DEX trading | balance, swap, quote, pools, status |
| `lux dex` | High-performance DEX | market, order, pool, perp, account |
| `lux warp` | Cross-chain messaging | |
| `lux contract` | Smart contract tools | deploy, verify |
| `lux config` | CLI configuration | |

## Network Modes

| Mode | Network ID | C-Chain ID | Base Port | Validators | Status |
|------|------------|------------|-----------|------------|--------|
| `--mainnet` | 1 | 96369 | 9630 | 5 | ✅ Production |
| `--testnet` | 2 | 96368 | 9640 | 5 | ✅ Testing |
| `--devnet` | 3 | - | 9650 | 5 | ⚠️ Missing genesis |
| `--dev` | 1337 | 1337 | 8545 | 1 | ✅ Rapid dev |

```bash
# Start mainnet with 5 validators
lux network start --mainnet

# Start testnet
lux network start --testnet

# Single-node dev mode (anvil/hardhat compatible port)
lux network start --dev

# Resume from snapshot
lux network start --snapshot-name=mybackup
```

## Core Network Chains

All validators on mainnet/testnet run these **11 core chains** natively:

| Chain | Name | Purpose |
|-------|------|---------|
| P | Platform | Staking, validator management |
| C | Contract | EVM smart contracts |
| X | Exchange | UTXO asset transfers |
| Q | Quantum | Post-quantum cryptography |
| A | AI | Artificial intelligence |
| B | Bridge | Cross-chain bridging |
| T | Threshold | Threshold FHE |
| Z | ZK | Zero-knowledge proofs |
| G | Graph | Graph database |
| K | KMS | Key management |
| D | DEX | Decentralized exchange |

**Endpoints** (mainnet on port 9630):
```
P-Chain:  http://localhost:9630/ext/bc/P
C-Chain:  http://localhost:9630/ext/bc/C/rpc
X-Chain:  http://localhost:9630/ext/bc/X
Q-Chain:  http://localhost:9630/ext/bc/Q/rpc
A-Chain:  http://localhost:9630/ext/bc/A/rpc
...
```

## Historic Chains

| Chain | Chain ID | Blockchain ID | Status |
|-------|----------|---------------|--------|
| LUX (C-Chain) | 96369 | dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ | Active |
| ZOO | 200200 | bXe2MhhAnXg6WGj6G8oDk55AKT1dMMsN72S8te7JdvzfZX1zM | Subnet |
| SPC | 36911 | QFAFyn1hh59mh7kokA55dJq5ywskF5A1yn8dDpLhmKApS6FP1 | Subnet |

## Data Locations

```
~/.lux/
├── chains/                       # All chain configs (consolidated)
│   ├── C/                        # C-Chain config
│   │   └── config.json           # Admin API, pruning, etc.
│   ├── zoo/                      # Chain definition + config
│   │   ├── genesis.json
│   │   ├── sidecar.json          # VM metadata
│   │   └── config.json           # Runtime config (optional)
│   └── <blockchain-id>/          # Deployed chain configs
│       └── config.json
├── dev/                          # Dev mode data (lux dev start)
│   ├── db/                       # Dev chain database
│   ├── logs/                     # Dev node logs
│   └── luxd.pid                  # Dev node PID file
├── plugins/
│   └── current/                  # Active plugins
│       └── <vmid>                # EVM plugin binary
├── runs/                         # Network runs
│   ├── mainnet/
│   │   └── run_YYYYMMDD_HHMMSS/  # Mainnet network data
│   └── testnet/
│       └── run_YYYYMMDD_HHMMSS/  # Testnet network data
└── snapshots/                    # Network snapshots
    └── testnet_complete_*.tar.gz # Saved state with imported blocks
```

## Network Snapshots

Save and restore network state:

```bash
# Hot snapshot while network is running (zero downtime)
lux network snapshot save my-backup --network-type=mainnet

# Stop network with snapshot
lux network stop --snapshot-name=my_snapshot

# Start from snapshot
lux network start --snapshot-name=my_snapshot

# Resume from hot snapshot
lux network start --mainnet --snapshot-name=my-backup
```

### Hot Snapshots (Zero Downtime)

As of 2026-01-22, the CLI supports zero-downtime hot snapshots:

- **Running network**: Uses gRPC `SaveHotSnapshot` via admin.snapshot API
- **Stopped network**: Uses direct BadgerDB access

Hot snapshots work even during active operations like block imports. The snapshot captures consistent state without interrupting the network.

```bash
# Create hot snapshot for each network type
lux network snapshot save mainnet-backup --network-type=mainnet
lux network snapshot save testnet-backup --network-type=testnet
lux network snapshot save devnet-backup --network-type=devnet
```

Notes:
- Hot snapshots use BadgerDB native incremental backups (~100KB vs 75GB for directory copies)
- Advanced snapshots (base/incremental/squash) live under `lux network snapshot advanced`

## Development

```bash
# Build CLI
cd /path/to/lux/cli
go build -o bin/lux ./main.go

# Install globally
go install

# Run tests
go test ./...
```

## AMM Trading

The AMM CLI supports Uniswap V2/V3 style DEX trading on Lux and Zoo networks.

### Wallet Access (Priority Order)

1. `--private-key` flag (hex private key)
2. `LUX_PRIVATE_KEY` environment variable
3. `LUX_MNEMONIC` environment variable (BIP39 mnemonic)

```bash
# Using mnemonic
export LUX_MNEMONIC="word1 word2 ... word12"
lux amm balance --network lux-testnet

# Using private key
export LUX_PRIVATE_KEY="0x..."
lux amm balance --network zoo
```

### Network Configuration

| Network | Flag | Chain ID | Default RPC |
|---------|------|----------|-------------|
| Lux Mainnet | `--network lux` | 96369 | localhost:8545 |
| Zoo Mainnet | `--network zoo` | 200200 | localhost:8546 |
| Lux Testnet | `--network lux-testnet` | 96368 | localhost:8547 |
| Zoo Testnet | `--network zoo-testnet` | 200201 | localhost:9640 |

Override RPC with `--rpc` flag:
```bash
lux amm status --network lux-testnet --rpc "http://127.0.0.1:9642/ext/bc/C/rpc"
```

### AMM Contract Addresses

All networks use the same contract addresses (CREATE2 deployed):

| Contract | Address |
|----------|---------|
| V2 Factory | `0xD173926A10A0C4eCd3A51B1422270b65Df0551c1` |
| V2 Router | `0xAe2cf1E403aAFE6C05A5b8Ef63EB19ba591d8511` |
| V3 Factory | `0x80bBc7C4C7a59C899D1B37BC14539A22D5830a84` |
| V3 Router | `0x939bC0Bca6F9B9c52E6e3AD8A3C590b5d9B9D10E` |
| Multicall | `0xd25F88CBdAe3c2CCA3Bb75FC4E723b44C0Ea362F` |
| Quoter | `0x12e2B76FaF4dDA5a173a4532916bb6Bfa3645275` |

### Key Derivation

Mnemonics are derived using BIP44 path `m/44'/60'/0'/0/0`:
- BIP39 mnemonic → seed
- BIP32 derivation → private key
- ECDSA → Ethereum address

Treasury address `0x9011E888251AB053B7bD1cdB598Db4f9DEd94714` is derived from the production mnemonic.

## DEX Trading

The Lux DEX provides a high-performance decentralized exchange with spot trading, AMM pools, and perpetual futures.

### Key Features

- **Central Limit Order Book (CLOB)**: Limit and market orders with 1ms block times
- **AMM Pools**: Constant Product, StableSwap, and Concentrated Liquidity
- **Perpetual Futures**: Up to 100x leverage on major assets
- **Cross-Chain Swaps**: Via Warp messaging between L1/L2/L3 chains
- **High-Frequency Trading**: Ultra-low latency for professional traders

### Network Configuration

| Network | Chain ID | Features |
|---------|----------|----------|
| Lux Mainnet | 96369 | Full DEX functionality |
| Zoo Mainnet | 200200 | AMM pools only |
| Lux Testnet | 96368 | Full DEX (test tokens) |

### Command Structure

```bash
# Market operations
lux dex market list              # List all markets
lux dex market info LUX/USDT     # Market details
lux dex market create            # Create new market

# Order operations
lux dex order place --market LUX/USDT --side buy --type limit --price 10.50 --amount 100
lux dex order cancel --order-id 12345
lux dex order history           # View order history

# Pool operations
lux dex pool create --type constant-product --token-a LUX --token-b USDT
lux dex pool add-liquidity --pool-id 1 --amount-a 1000 --amount-b 10000
lux dex pool list                # List available pools

# Perpetual futures
lux dex perp open --market BTC/USD --side long --leverage 10x --amount 0.1
lux dex perp close --position-id 5678
lux dex perp positions          # View open positions

# Account management
lux dex account balance          # View account balances
lux dex account history          # View trading history
lux dex account positions        # View open positions
```

### Order Types

| Type | Description | Example |
|------|-------------|---------|
| `limit` | Order at specific price | `--type limit --price 10.50` |
| `market` | Immediate execution | `--type market` |
| `stop-limit` | Triggered limit order | `--type stop-limit --stop 9.50 --limit 9.40` |
| `stop-market` | Triggered market order | `--type stop-market --stop 11.00` |

### Pool Types

| Type | Description | Fee |
|------|-------------|-----|
| `constant-product` | Uniswap-style AMM | 0.3% |
| `stable` | Curve-style stablecoin pool | 0.04% |
| `concentrated` | Uniswap V3-style | 0.05%-1% |

### Perpetual Futures

- **Leverage**: 2x to 100x (configurable per market)
- **Funding Rate**: Dynamic based on market conditions
- **Liquidation**: Automatic with partial liquidation support
- **Markets**: BTC/USD, ETH/USD, LUX/USD, and more

### Cross-Chain Trading

```bash
# Cross-chain swap via Warp
lux dex swap --from-chain my-l1 --to-chain my-l2 \
  --from-token 0xUSDC_L1... --to-token 0xWETH_L2... \
  --amount 1000000000 --cross-chain

# Cross-chain order execution
lux dex order place --market LUX/USDT \
  --side buy --type limit --price 10.50 --amount 100 \
  --cross-chain --target-chain my-l2
```

### Configuration

DEX configuration is stored in `~/.lux-cli/dex/config.json`:

```json
{
  "defaultChain": "my-l1",
  "slippage": 0.5,
  "deadline": "20m",
  "gasMultiplier": 1.1,
  "routers": {
    "my-l1": {
      "default": "0xRouterAddr...",
      "amm": "0xAMMRouter...",
      "clob": "0xCLOBRouter..."
    }
  }
}
```

## Troubleshooting

### "404 page not found" on subnet RPC
The subnet is not tracked or deployed. Ensure:
1. Network is running: `lux network status`
2. Subnet is deployed: `lux chain deploy <name> --local`
3. Using correct blockchain ID in RPC path

### "ErrPrunedAncestor" during import
The genesis state is not accessible. This is a known issue with fresh subnet deployments. The genesis state trie must be properly committed before imports can work.

### "invalid gas limit" during import
If you see `invalid gas limit: have 12000000, want 10000000`, the Fortuna upgrade is activated prematurely.

**Fix**: Set far-future timestamps for Fortuna and related upgrades in genesis:

```json
{
  "config": {
    "etnaTimestamp": 253399622400,
    "fortunaTimestamp": 253399622400,
    "graniteTimestamp": 253399622400
  }
}
```

The value `253399622400` is year 9999, effectively disabling these upgrades.

### Port Configuration
| Network | Base Port | Range |
|---------|-----------|-------|
| mainnet | 9630 | 9630-9638 |
| testnet | 9640 | 9640-9648 |
| devnet | 9650 | 9650-9658 |
| dev | 8545 | 8545 only |

Each 5-validator network uses 5 ports (one per validator): base, base+2, base+4, base+6, base+8

### Import hangs or timeouts
The `admin_importChain` RPC may timeout for large imports (>10k blocks), but the import continues in the background. Monitor progress via logs:

```bash
# Check import progress
tail -f ~/.lux/runs/mainnet/current/node1/db/mainnet/main.log | grep "Inserted new block"
```

## Token Denominations

| Chain | Decimals | Notes |
|-------|----------|-------|
| P-Chain/X-Chain | 6 | 1 LUX = 1,000,000 µLUX |
| C-Chain (EVM) | 18 | Standard EVM decimals |

---

*This file contains essential documentation for the Lux CLI project.*
