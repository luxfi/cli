# Migration Tools

This directory contains the chainmigrate binary used by `lux migrate` commands.

## Binary

- **migrate** - Symlink to `/Users/z/work/lux/node/cmd/chainmigrate/chainmigrate`

The binary is built from the node repository and uses the ChainExporter interface to export blockchain data.

## Usage via lux-cli

```bash
# Prepare migration from EVM to C-Chain
lux migrate prepare \
  --source-db /path/to/evm/pebble \
  --output ./migration-output \
  --network-id 96369 \
  --validators 5

# Bootstrap the migrated network
lux migrate bootstrap --output ./migration-output

# Import into existing chain
lux migrate import --source ./migration-output/c-chain-db

# Validate migration
lux migrate validate --endpoint http://localhost:9650
```

## Direct Usage

The migrate binary can also be used directly:

```bash
./migrate \
  --src-pebble /path/to/source/db \
  --dst-leveldb /path/to/dest/db \
  --chain-id 96369 \
  --start-block 0 \
  --end-block 1000 \
  --batch-size 100
```

## Flags

- `--src-pebble` - Source PebbleDB path
- `--dst-leveldb` - Destination LevelDB path
- `--chain-id` - Chain ID for the migration
- `--start-block` - Starting block number (default: 0)
- `--end-block` - Ending block number (default: latest)
- `--batch-size` - Blocks per batch (default: 100)
- `--export-only` - Only export, don't import
- `--import-only` - Only import from existing export

## Implementation

The tool uses:
- **ChainExporter** interface from `luxfi/node/chainmigrate`
- **EVM Exporter** from `luxfi/evm/plugin/evm`
- **luxfi/log** for logging
- **luxfi/geth** for Ethereum types

## Building from Source

```bash
cd /Users/z/work/lux/node/cmd/chainmigrate
go build -o chainmigrate .
```

The built binary will be available at `/Users/z/work/lux/node/cmd/chainmigrate/chainmigrate`.
