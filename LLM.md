# Lux CLI - AI Assistant Knowledge Base

**Last Updated**: 2025-12-25
**Version**: 1.21.24
**Organization**: Lux Industries

## Quick Reference

### Essential Commands

```bash
# Network Management (5-node local network)
lux network start              # Start local network
lux network start --mainnet    # Start with mainnet genesis (ID: 96369)
lux network stop               # Stop local network
lux network status             # Check network status
lux network clean              # Remove all network data

# Chain Operations
lux chain import --chain=c --path=/path/to/blocks.rlp  # Import RLP blocks to C-Chain
lux chain import --chain=zoo --path=/path/to/zoo.rlp   # Import to subnet

# L2/Subnet Management (alias: lux l2, lux subnet)
lux l2 create mychain          # Create L2 configuration
lux l2 deploy mychain --local  # Deploy to local network
lux l2 list                    # List configured L2s
lux l2 describe mychain        # Show L2 details

# L1 Sovereign Chain
lux l1 create mychain          # Create sovereign L1
lux l1 deploy mychain          # Deploy L1

# AMM Trading (LUX_MNEMONIC supported)
lux amm balance                # Check token balances
lux amm status                 # Show AMM contract status
lux amm pools                  # List liquidity pools
lux amm swap --from 0x... --to 0x... --amount 100  # Swap tokens
lux amm quote --from 0x... --to 0x... --amount 100 # Get swap quote
```

## Command Architecture

The CLI is organized into these main command groups:

| Command | Purpose | Notes |
|---------|---------|-------|
| `lux network` | Local 5-node network management | start, stop, status, clean |
| `lux chain` | Chain-level data operations | import, export |
| `lux l1` | Sovereign L1 chains | create, deploy, describe |
| `lux l2` | L2s/Subnets (alias: `subnet`) | create, deploy, validators |
| `lux l3` | App-specific L3 chains | create, deploy |
| `lux key` | Key management | create, list, export |
| `lux validator` | P-Chain validator balance | |
| `lux amm` | AMM/DEX trading | swap, balance, add-liquidity |
| `lux warp` | Cross-chain messaging | |
| `lux contract` | Smart contract tools | deploy, verify |
| `lux config` | CLI configuration | |

## Chain Import

Import RLP-encoded blockchain data to a running chain.

### Admin API Differences: Coreth vs EVM

**IMPORTANT**: The C-Chain (Coreth) and Subnets (EVM) have DIFFERENT admin API implementations:

| Chain Type | Endpoint | Method Format | Parameters |
|------------|----------|---------------|------------|
| C-Chain (Coreth) | `/ext/bc/C/rpc` | `admin_importChain` | Array: `["/path/to/file.rlp"]` |
| Subnets (EVM) | `/ext/bc/<id>/admin` | `admin.importChain` | Object: `{"file":"/path/to/file.rlp"}` |

### C-Chain Import (Coreth)

```bash
# Via direct curl
curl -X POST "http://127.0.0.1:9630/ext/bc/C/rpc" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"admin_importChain","params":["/path/to/blocks.rlp"],"id":1}'
```

### Subnet Import (EVM Plugin)

```bash
# Via direct curl (use blockchain ID from deploy output)
curl -X POST "http://127.0.0.1:9630/ext/bc/<blockchain-id>/admin" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"admin.importChain","params":{"file":"/path/to/blocks.rlp"},"id":1}'
```

### CLI Import (Planned)

```bash
# Import to C-Chain
lux chain import --chain=c --path=/path/to/blocks.rlp

# Import to subnet
lux chain import --chain=<blockchain-id> --path=/path/to/blocks.rlp
```

### Block Import Results (2025-12-25)

Successfully imported blocks to local networks:

| Network | Chain | Blocks | Genesis Hash |
|---------|-------|--------|--------------|
| Mainnet | C-Chain (96369) | ~1,082,780 | `0x3f4fa...` |
| Mainnet | Zoo (200200) | 799 | `0x7c548...` |
| Testnet | C-Chain (96368) | 218 | `0x1c5fe...` |
| Testnet | Zootest (200201) | 84 | `0x0652f...` |

**RPC Endpoints** (internal port 9630/9640):
- Mainnet C-Chain: `http://127.0.0.1:9630/ext/bc/C/rpc`
- Mainnet Zoo: `http://127.0.0.1:9630/ext/bc/<zoo-id>/rpc`
- Testnet C-Chain: `http://127.0.0.1:9640/ext/bc/C/rpc`
- Testnet Zootest: `http://127.0.0.1:9640/ext/bc/<zootest-id>/rpc`

## Network Start Options

```bash
# Default: starts with Lux mainnet genesis
lux network start --mainnet

# Testnet genesis
lux network start --testnet

# From snapshot
lux network start --snapshot=mybackup
```

## Historic Chains

| Chain | Chain ID | Blockchain ID | Status |
|-------|----------|---------------|--------|
| LUX (C-Chain) | 96369 | dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ | Active |
| ZOO | 200200 | bXe2MhhAnXg6WGj6G8oDk55AKT1dMMsN72S8te7JdvzfZX1zM | Subnet |
| SPC | 36911 | QFAFyn1hh59mh7kokA55dJq5ywskF5A1yn8dDpLhmKApS6FP1 | Subnet |

## Data Locations

```
/Users/z/work/lux/state/
├── rlp/                          # Exported RLP blockchain data
│   ├── lux-mainnet-96369.rlp     # C-Chain blocks (1.28GB)
│   └── zoo-mainnet/
│       └── zoo-mainnet-200200.rlp # ZOO blocks (1.3MB, 100 blocks)
│
└── chaindata/                    # Raw database files
    ├── lux-mainnet-96369/        # C-Chain PebbleDB
    ├── zoo-mainnet-200200/       # ZOO chaindata
    └── spc-mainnet-36911/        # SPC chaindata

~/.lux/
├── chains/                       # Chain configurations
│   └── zoo/
│       ├── genesis.json
│       └── sidecar.json          # VM metadata
├── chain-configs/                # Chain-specific configs
│   └── C/
│       └── config.json           # C-Chain config (admin API, pruning)
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
# Stop network with snapshot
lux network stop --snapshot-name=my_snapshot

# Start from snapshot
lux network start --snapshot=my_snapshot

# Manual snapshot (while network running)
tar -czf ~/.lux/snapshots/backup.tar.gz -C ~/.lux/runs/mainnet run_*
```

## Development

```bash
# Build CLI
cd /Users/z/work/lux/cli
go build -o bin/lux ./main.go

# Install globally
go install

# Run tests
go test ./...
```

## VM Implementations

| Chain Type | VM | Source |
|------------|----|----|
| C-Chain | Coreth | `/Users/z/work/lux/coreth` |
| Subnets (ZOO, SPC) | EVM | `/Users/z/work/lux/evm` |

The admin_importChain RPC is implemented in both:
- Coreth: `eth/api_admin.go`
- EVM: `plugin/evm/admin.go`

## Troubleshooting

### "404 page not found" on subnet RPC
The subnet is not tracked or deployed. Ensure:
1. Network is running: `lux network status`
2. Subnet is deployed: `lux l2 deploy <name> --local`
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

### Port 9650 vs 9630
- **9650**: Public API port (external)
- **9630**: Internal admin API port (use this for admin_importChain)

### Import hangs or timeouts
The `admin_importChain` RPC may timeout for large imports (>10k blocks), but the import continues in the background. Monitor progress via logs:

```bash
# Check import progress
tail -f ~/.lux/runs/mainnet/current/node1/db/mainnet/main.log | grep "Inserted new block"
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

## Token Denominations

| Chain | Decimals | Notes |
|-------|----------|-------|
| P-Chain/X-Chain | 6 | 1 LUX = 1,000,000 µLUX |
| C-Chain (EVM) | 18 | Standard EVM decimals |

## Rules for AI Assistants

1. **ALWAYS** update this LLM.md with significant discoveries
2. **NEVER** create random summary files - update THIS file
3. **NEVER** use ava-labs packages - use luxfi packages
4. Use port 9630 for admin APIs, not 9650
5. Test changes with actual data from `/Users/z/work/lux/state/`

---

*This file is the single source of truth for AI assistants working on this project.*
