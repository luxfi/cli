# Loading Existing State with Lux CLI

The Lux CLI now supports loading existing blockchain state when starting a network. This is particularly useful for:
- Resuming from previous network runs
- Loading production database snapshots
- Testing with real blockchain data
- Migrating between different environments

## Quick Start

### Automatic Detection

If you have an existing mainnet-regenesis database (9.3GB with 1,074,616 blocks), the CLI will automatically detect and load it:

```bash
./bin/lux network start
```

The CLI will look for the database at:
`~/.lux-cli/runs/mainnet-regenesis/node1/chains/2G8mK7VCZX1dV8iPjkkTDMpYGZDCNLLVdTJVLmMsG5ZV7zKVmB/db`

### Manual Specification

You can also explicitly specify the database path:

```bash
# Load specific subnet database
./bin/lux network start \
  --subnet-state-path="/path/to/database/db" \
  --blockchain-id="2G8mK7VCZX1dV8iPjkkTDMpYGZDCNLLVdTJVLmMsG5ZV7zKVmB"

# Load from chaindata directory
./bin/lux network start \
  --state-path="/home/z/work/lux/state/chaindata/lux-mainnet-96369"
```

## Command Line Flags

The following flags are available for state loading:

| Flag | Description | Example |
|------|-------------|---------|
| `--subnet-state-path` | Path to existing subnet database | `/path/to/chains/blockchainID/db` |
| `--state-path` | Path to existing state directory | `~/work/lux/state/chaindata/lux-mainnet-96369` |
| `--subnet-id` | Subnet ID for the loaded state | `subnet-1234...` |
| `--blockchain-id` | Blockchain ID for the loaded state | `2G8mK7VCZX1dV8iPjkkTDMpYGZDCNLLVdTJVLmMsG5ZV7zKVmB` |

## Database Locations

The CLI looks for databases in these locations:

1. **Default mainnet-regenesis**: `~/.lux-cli/runs/mainnet-regenesis/node1/chains/[blockchainID]/db`
2. **State chaindata**: `~/work/lux/state/chaindata/[network]/db`
3. **Custom path**: Any path you specify with `--subnet-state-path`

## Known Blockchain IDs

| Network | Blockchain ID | Description |
|---------|--------------|-------------|
| LUX Mainnet Subnet | 2G8mK7VCZX1dV8iPjkkTDMpYGZDCNLLVdTJVLmMsG5ZV7zKVmB | Primary subnet with 1M+ blocks |
| LUX Testnet Subnet | 2sdADEgBC3NjLM4inKc1hY1PQpCT3JVyGVJxdmcq6sqrDndjFG | Test subnet |

## Verification

After starting the network with existing state, you can verify the database was loaded:

1. Check the logs for confirmation messages:
   ```
   Found existing mainnet-regenesis database at default location
   Successfully loaded existing subnet state for blockchain 2G8mK7VCZX1dV8iPjkkTDMpYGZDCNLLVdTJVLmMsG5ZV7zKVmB
   ```

2. Query the blockchain to check block height:
   ```bash
   curl -X POST --data '{
     "jsonrpc": "2.0",
     "method": "eth_blockNumber",
     "params": [],
     "id": 1
   }' -H "Content-Type: application/json" http://localhost:9630/ext/bc/C/rpc
   ```

3. Check database size in the running network:
   ```bash
   du -sh ~/.lux-cli/runs/*/node1/chains/*/db
   ```

## Troubleshooting

### Database Not Found

If the database is not automatically detected:
1. Verify the path exists: `ls -la /path/to/database`
2. Check permissions: `ls -ld /path/to/database`
3. Use explicit path with `--subnet-state-path`

### Wrong Blockchain ID

If you see an error about blockchain ID:
1. Check the actual blockchain ID from the path
2. Specify it explicitly with `--blockchain-id`

### Copy Failures

If database copying fails:
1. Ensure sufficient disk space: `df -h`
2. Check write permissions in target directory
3. Verify source database is not corrupted

## Implementation Details

The state loading feature:
1. Detects existing databases before network start
2. Copies the database to the appropriate node directory
3. Preserves all block data, state trie, and transaction history
4. Maintains database integrity during the copy process
5. Falls back gracefully if no state is found

## Example Use Cases

### Resume After Network Restart
```bash
# Network was previously running with data
./bin/lux network start  # Automatically loads previous state
```

### Load Production Snapshot
```bash
# Load a production database snapshot for testing
./bin/lux network start \
  --subnet-state-path="/backup/production-snapshot/db" \
  --blockchain-id="2G8mK7VCZX1dV8iPjkkTDMpYGZDCNLLVdTJVLmMsG5ZV7zKVmB"
```

### Migrate Between Environments
```bash
# Move database from one environment to another
./bin/lux network start \
  --state-path="/migration/source/chaindata/lux-mainnet-96369"
```

## Performance

Loading existing state:
- 9.3GB database: ~30-60 seconds copy time
- 1M+ blocks: Immediate availability after copy
- No replay or reprocessing required
- Full state trie preserved