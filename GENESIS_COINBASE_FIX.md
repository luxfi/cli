# Genesis Coinbase Address Fix

## Problem
The Platform VM was generating C-Chain genesis with incorrect coinbase addresses that had 64 hex characters instead of the correct 40 characters required for Ethereum addresses.

### Error Message
```
hex string has length 64, want 40 for common.Address
```

## Root Cause
The genesis files had coinbase addresses like:
```json
"coinbase": "0x0000000000000000000000000000000000000000000000000000000000000000"
```

This is 64 hex characters (32 bytes), but Ethereum addresses are only 20 bytes (40 hex characters).

## Solution
Fixed all genesis files to use the correct 40-character format:
```json
"coinbase": "0x0000000000000000000000000000000000000000"
```

## Files Fixed
1. `/home/z/work/lux/node/genesis/cchain_genesis_mainnet.json`
2. `/home/z/work/lux/node/genesis/cchain_genesis_final.json`
3. `/home/z/work/lux/node/genesis/genesis_mainnet.json` (embedded cChainGenesis)
4. `/home/z/work/lux/node/genesis/genesis_testnet.json` (embedded cChainGenesis)
5. `/home/z/work/lux/node/genesis/genesis_local.json` (embedded cChainGenesis)
6. `/home/z/work/lux/node/genesis/genesis_test.json` (embedded cChainGenesis)
7. `/home/z/work/lux/node/genesis/genesis_96369_migrated.json` (embedded cChainGenesis)

## Fix Script
A Python script was created to automatically fix all genesis files:
```bash
/home/z/work/lux/cli/fix_genesis_coinbase.py
```

## Verification
After the fix, all coinbase addresses now correctly use 40 hex characters (20 bytes).

## Rebuild
After fixing the genesis files, rebuild the node:
```bash
cd /home/z/work/lux/node
./scripts/build.sh
```

## Testing
The C-Chain can now initialize properly without the address format error.

---
*Fixed on: 2025-01-05*