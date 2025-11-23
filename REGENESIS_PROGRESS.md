# Lux Network 96369 C-Chain Regenesis Progress

## Session Date: 2025-11-22

### Objective
Restore Lux network 96369 state (1,082,780 blocks, 7.1GB PebbleDB) into mainnet C-Chain.

### Latest Attempt: Automatic State Migration (2025-11-22 18:12)
**Status**: ⚠️ FAILED - Database transaction size limit exceeded

**What Happened**:
1. Started mainnet node with data-dir containing network 96369 PebbleDB state
2. C-Chain detected chainId 96369 genesis and automatically initiated migration
3. Successfully copied 1,080,000+ database entries from PebbleDB
4. **FAILURE**: BadgerDB transaction flush failed with "Txn is too big to fit into one request"
5. Backend creation failed, leaving backend=nil
6. C-Chain RPC endpoint unavailable (404 error)

**Root Cause**: BadgerDB has a maximum transaction size limit. The migration code attempted to copy all 1,080,000+ entries in batches, but the batch size was still too large for BadgerDB's transaction limits.

**Key Error Messages**:
```
Backend creation result: err=failed to copy database entries: failed to copy database: failed to flush batch: Txn is too big to fit into one request, backend=false
VM initialization failed, continuing anyway
Backend is nil in CreateHandlers
failed to create HTTP handlers
```

### ✅ Completed Steps

#### 1. EVM Plugin Build (Fixed All Errors)
- **Location**: `~/work/lux/evm`
- **Binary**: `~/.luxd/plugins/srEXiWaHuhNyGwPUi444Tu47ZEDwxTWrbQiuD7FmgSAQ6X7Dy` (59MB)
- **Commit**: `2829dff68543cae82e3d139ed62e3b3b6310e4b3`
- **Fixes Applied**:
  - Changed imports from `github.com/luxfi/consensus` to `github.com/luxfi/consensus/context`
  - Removed `rawdb.FindCommonAncestor()` dependency
  - Fixed `database.Reader` to `database.StateReader`
  - Fixed 7 compilation errors total

#### 2. PebbleDB State Prepared
- **Source**: `/Users/z/work/lux/state/chaindata/lux-mainnet-96369/db/pebbledb`
- **Size**: 7.1GB containing 1,082,780 blocks
- **Blockchain ID**: `dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ`
- **Chain ID**: 96369
- **Copied To**: `~/.luxd/evm-readonly/chainData/dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ/pebbledb`

#### 3. Read-Only Chain Configuration
- **Config File**: `~/.luxd/evm-readonly/chains/dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ/config.json`
- **Settings**:
  ```json
  {
    "database-read-only": true,
    "database-type": "pebbledb",
    "pruning-enabled": false,
    "enabled-eth-apis": ["eth", "eth-filter", "net", "web3", "internal-eth", "internal-blockchain", "internal-transaction"]
  }
  ```

#### 4. Genesis File Ready
- **Location**: `/Users/z/work/lux/state/cmd/genesis/configs/C/genesis.json`
- **Chain ID**: 96369
- **Base64 Encoded**: 1,828 bytes

### 🔄 Current Status

**Node Running**:
- PID: 9491
- Network: Mainnet (network-id=1)
- Data Dir: `/Users/z/.luxd/evm-readonly`
- P-Chain: Bootstrapped ✅

**Issue Identified**:
- Blockchain ID `dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ` is from network 96369
- Currently running on mainnet (network-id=1)
- Need to switch to network 96369 for proper blockchain registration

### ⏳ Next Steps - Solutions for BadgerDB Transaction Limit

#### Option A: Fix Batch Size in Migration Code (RECOMMENDED)
**Goal**: Reduce batch size to avoid exceeding BadgerDB transaction limits

**Location**: `/Users/z/work/lux/node/vms/cchainvm/backend.go` (copyFromMigratedDB function)

**Required Changes**:
1. Identify current batch size (likely 10,000+ entries per batch)
2. Reduce to 1,000 or 500 entries per batch
3. Add proper batch flush interval
4. Test with PebbleDB state to verify it completes successfully

**Code Pattern** (example):
```go
const batchSize = 1000 // Reduced from 10000
batch := db.NewBatch()
count := 0

for entries {
    batch.Put(key, value)
    count++

    if count >= batchSize {
        if err := batch.Write(); err != nil {
            return err
        }
        batch.Reset()
        count = 0
    }
}
// Final flush
if count > 0 {
    return batch.Write()
}
```

#### Option B: Use PebbleDB Directly (Read-Only Mode)
**Goal**: Serve the PebbleDB state directly without copying to BadgerDB

**Approach**:
1. Modify C-Chain to support PebbleDB as a read-only database backend
2. Configure chain to use existing PebbleDB at:
   `~/.luxd/evm-readonly/chainData/dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ/pebbledb/`
3. Skip migration/copy step entirely
4. Serve RPC queries directly from PebbleDB

**Advantages**:
- No data copy required (instant)
- No transaction size limits
- Preserves original network 96369 state perfectly

**Disadvantages**:
- Read-only (can't add new blocks)
- May require code changes to support PebbleDB in C-Chain

#### Option C: Incremental Migration via RPC
**Goal**: Migrate blocks incrementally using eth_getBlockByNumber RPC calls

**Approach**:
1. Keep network 96369 node running with PebbleDB (read-only)
2. Use mainnet C-Chain with fresh BadgerDB
3. Write migration tool that:
   - Queries network 96369 node via RPC (eth_getBlockByNumber)
   - Inserts blocks into mainnet C-Chain via internal API
   - Processes 1,000 blocks at a time
   - Saves progress checkpoint
4. Run migration tool until all 1,082,780 blocks are copied

**Advantages**:
- Avoids database transaction limits (one block at a time)
- Can resume if interrupted
- Progress tracking built-in

**Disadvantages**:
- Slower (may take hours)
- Requires both nodes running simultaneously

### 📊 Key Metrics

- **Total Blocks**: 1,082,780
- **Database Size**: 7.1GB
- **Genesis**: Block 0 - `0x3f4fa2a0b0ce089f52bf0ae9199c75ffdd76ecafc987794050cb0d286f1ec61e`
- **Latest Block**: Block 1,082,780 - `0x32dede1fc8e0f11ecde12fb42aef7933fc6c5fcf863bc277b5eac08ae4d461f0`
- **Treasury Address**: `0x9011E888251AB053B7bD1cdB598Db4f9DEd94714`
- **Expected Balance**: > 1.9T LUX

### 🎯 Final Goal

Deploy read-only EVM chain with existing state, then use native cross-chain RPC to import all blocks into mainnet C-Chain via running VM instances.

**User Requirements**:
- Primary interaction via `lux-cli` (~/work/lux/cli)
- Use RPC/CLI for all management
- Deploy with pre-existing state
- Work properly with P-Chain

### 📝 Notes

- EVM plugin successfully built with all consensus context fixes
- Read-only mode configured to prevent accidental state modifications
- Genesis file matches archived network 96369 configuration
- All tooling ready for final deployment step
