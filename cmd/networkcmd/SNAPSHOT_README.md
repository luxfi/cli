# Network Snapshot Feature

## Overview

The network snapshot feature allows you to save, list, load, and delete network state snapshots. This is useful for:
- Preserving network state before major changes
- Testing different scenarios with saved states
- Creating checkpoints during development
- Sharing network configurations

## Implementation

### Files Created
- `snapshot.go` - Main implementation file containing all snapshot commands

### Commands Implemented

#### 1. `lux network snapshot save <name>`
Saves the current network state to a named snapshot.

**Usage:**
```bash
# Save all networks
lux network snapshot save my-snapshot

# Save specific network
lux network snapshot save my-snapshot --network mainnet
lux network snapshot save my-snapshot --network testnet
lux network snapshot save my-snapshot --network devnet
```

**Options:**
- `--network` - Specify which network(s) to snapshot (mainnet, testnet, devnet, or all)

**Behavior:**
- Creates snapshot directory at `~/.lux/snapshots/<name>/`
- Copies network data from `~/.lux/runs/<network>/current/` to snapshot
- Stores metadata (timestamp, networks included) in `metadata.json`
- Fails if snapshot name already exists

#### 2. `lux network snapshot list`
Lists all available snapshots.

**Usage:**
```bash
lux network snapshot list
```

**Output:**
```
Available snapshots:

  my-snapshot (created: 2025-01-01 12:00:00)
    Networks: mainnet, testnet
  
  test-state (created: 2025-01-02 15:30:00)
    Networks: devnet
```

#### 3. `lux network snapshot load <name>`
Loads a previously saved snapshot.

**Usage:**
```bash
# Load all networks from snapshot
lux network snapshot load my-snapshot

# Load specific network from snapshot
lux network snapshot load my-snapshot --network mainnet
```

**Options:**
- `--network` - Specify which network(s) to load (mainnet, testnet, devnet, or all)

**Behavior:**
- Stops any running networks
- Creates new run directory with timestamp
- Copies snapshot data to new run directory
- Updates `current` symlink to point to new run
- Network can then be started with `lux network start`

#### 4. `lux network snapshot delete <name>`
Deletes a saved snapshot.

**Usage:**
```bash
lux network snapshot delete my-snapshot
```

**Behavior:**
- Removes snapshot directory and all contents
- Fails if snapshot doesn't exist

## Directory Structure

```
~/.lux/
├── runs/                          # Active network data
│   ├── mainnet/
│   │   ├── current -> run_20250101_120000
│   │   ├── run_20250101_120000/  # Network state
│   │   └── run_20250101_130000/
│   ├── testnet/
│   └── devnet/
└── snapshots/                     # Saved snapshots
    ├── my-snapshot/
    │   ├── metadata.json
    │   ├── mainnet/              # Snapshot of mainnet state
    │   └── testnet/              # Snapshot of testnet state
    └── test-state/
        ├── metadata.json
        └── devnet/
```

## Implementation Details

### Key Features
1. **Network-aware**: Supports mainnet, testnet, and devnet independently
2. **Safe**: Won't overwrite existing snapshots
3. **Metadata tracking**: Stores creation timestamp and networks included
4. **Graceful degradation**: Skips networks that don't exist in snapshot
5. **Atomic operations**: Uses temp files and renames for safety

### Helper Functions
- `copyDir()` - Recursively copies directory contents
- `copyFile()` - Copies individual files preserving permissions
- `saveJSON()` - Saves metadata to JSON file
- `loadJSON()` - Loads metadata from JSON file

### Integration Points
- Uses `app.GetSnapshotsDir()` for snapshot storage location
- Uses `app.GetRunDir()` for run directory location
- Integrates with `StopNetwork()` for graceful network shutdown
- Uses `binutils.KillgRPCServerProcessForNetwork()` for cleanup

## Building

The code has been implemented and compiles successfully:

```bash
cd ~/work/lux/cli/cmd/networkcmd
go build .
```

**Note:** There is currently a global build issue with the CLI project related to missing precompile dependencies. This affects building the full CLI binary but does not affect the correctness of the snapshot implementation.

Once the precompile dependency issue is resolved, build with:

```bash
cd ~/work/lux/cli
make build
# or
CGO_ENABLED=0 GOSUMDB=off GOPROXY=direct go build -o ./bin/lux main.go
```

## Testing

### Manual Testing

1. **Start a network:**
   ```bash
   lux network start --mainnet
   ```

2. **Create a snapshot:**
   ```bash
   lux network snapshot save test-state
   ```

3. **List snapshots:**
   ```bash
   lux network snapshot list
   ```

4. **Stop and modify network (optional):**
   ```bash
   lux network stop
   # Make some changes
   ```

5. **Load the snapshot:**
   ```bash
   lux network snapshot load test-state
   ```

6. **Delete the snapshot:**
   ```bash
   lux network snapshot delete test-state
   ```

### Expected Behavior

1. **Save**: Should copy all network data and create metadata
2. **List**: Should show all snapshots with timestamps
3. **Load**: Should restore network to saved state
4. **Delete**: Should remove snapshot completely

## Error Handling

The implementation includes error handling for:
- Invalid snapshot names (containing `/` or `..`)
- Non-existent snapshots
- Missing network data
- File system errors (permissions, disk space, etc.)
- Already running networks (stops them gracefully)

## Future Enhancements

Possible future improvements:
1. Snapshot compression to save disk space
2. Snapshot export/import for sharing
3. Incremental snapshots
4. Snapshot verification/integrity checks
5. Automatic snapshot cleanup (delete old snapshots)
6. Snapshot tags and descriptions

## Notes

- Snapshots are stored locally in `~/.lux/snapshots/`
- Each snapshot can contain one or more network types
- Loading a snapshot creates a new run directory (doesn't overwrite existing)
- The `--network all` flag is the default for save/load operations
- Snapshot names must be valid directory names (no special characters)
