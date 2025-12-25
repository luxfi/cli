# Lux CLI - AI Assistant Knowledge Base

**Last Updated**: 2024-12-20
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

# AMM Trading
lux amm balance                # Check token balances
lux amm swap --from LUX --to USDC --amount 100  # Swap tokens
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

Import RLP-encoded blockchain data to a running chain:

```bash
# Import to C-Chain (default)
lux chain import --chain=c --path=/Users/z/work/lux/state/rlp/lux-mainnet-96369.rlp

# Import to ZOO subnet (use blockchain ID)
lux chain import --chain=bXe2MhhAnXg6WGj6G8oDk55AKT1dMMsN72S8te7JdvzfZX1zM \
  --path=/Users/z/work/lux/state/rlp/zoo-mainnet/zoo-mainnet-200200.rlp

# Custom RPC endpoint (port 9630, NOT 9650)
lux chain import --path=/tmp/blocks.rlp --rpc=http://localhost:9630/ext/bc/C/admin
```

**RPC Endpoints** (internal port 9630):
- C-Chain admin: `http://localhost:9630/ext/bc/C/admin`
- Subnet admin: `http://localhost:9630/ext/bc/<blockchain-id>/admin`

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
│       └── genesis.json
├── plugins/                      # VM plugins
│   └── <vmid>                    # EVM plugin binary
└── runs/                         # Network runs
    └── local_network/            # Current local network
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

### Port 9650 vs 9630
- **9650**: Public API port (external)
- **9630**: Internal admin API port (use this for admin_importChain)

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
