#!/bin/bash

echo "=== Blockchain Regenesis Demo ==="
echo "Demonstrating idempotent export/import functionality for Lux Network"
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check C-Chain current state
echo -e "${BLUE}Step 1: Checking C-Chain current state...${NC}"
CURRENT_HEIGHT=$(curl -s -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  http://localhost:9630/ext/bc/C/rpc | jq -r '.result' | xargs printf "%d\n")
echo "C-Chain is at block: $CURRENT_HEIGHT"

# Export current C-Chain data
echo -e "\n${BLUE}Step 2: Exporting C-Chain blocks 0 to $CURRENT_HEIGHT...${NC}"
./bin/lux export \
  --rpc http://localhost:9630/ext/bc/C/rpc \
  --start 0 \
  --end $CURRENT_HEIGHT \
  --output regenesis-export.json \
  --parallel 10

# Check export file
echo -e "\n${BLUE}Step 3: Verifying export file...${NC}"
BLOCKS_EXPORTED=$(cat regenesis-export.json | jq '.blocks | length')
echo "Blocks exported: $BLOCKS_EXPORTED"
echo "Export metadata:"
cat regenesis-export.json | jq '{version, chainId, exportTime, blockCount: .metadata.blockCount}'

# Test idempotent import (dry-run first)
echo -e "\n${YELLOW}Step 4: Testing import with dry-run mode...${NC}"
./bin/lux import \
  --file regenesis-export.json \
  --dest http://localhost:9630/ext/bc/C/rpc \
  --parallel 50 \
  --dry-run

# Actual import (idempotent - safe to run multiple times)
echo -e "\n${GREEN}Step 5: Performing idempotent import...${NC}"
./bin/lux import \
  --file regenesis-export.json \
  --dest http://localhost:9630/ext/bc/C/rpc \
  --parallel 50 \
  --skip-existing

# Verify import
echo -e "\n${BLUE}Step 6: Verifying import results...${NC}"
NEW_HEIGHT=$(curl -s -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  http://localhost:9630/ext/bc/C/rpc | jq -r '.result' | xargs printf "%d\n")
echo "C-Chain now at block: $NEW_HEIGHT"

# Check treasury balance preservation
echo -e "\n${BLUE}Step 7: Checking treasury balance...${NC}"
TREASURY_BALANCE=$(curl -s -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_getBalance","params":["0x9011E888251AB053B7bD1cdB598Db4f9DEd94714","latest"],"id":1}' \
  http://localhost:9630/ext/bc/C/rpc | jq -r '.result')
echo "Treasury balance at 0x9011E888251AB053B7bD1cdB598Db4f9DEd94714: $TREASURY_BALANCE"

# Convert hex to decimal and format
if [ "$TREASURY_BALANCE" != "null" ] && [ "$TREASURY_BALANCE" != "" ]; then
    BALANCE_DEC=$(printf "%d\n" "$TREASURY_BALANCE")
    BALANCE_LUX=$(echo "scale=2; $BALANCE_DEC / 1000000000000000000" | bc)
    echo "Treasury balance: $BALANCE_LUX LUX"
fi

echo -e "\n${GREEN}=== Regenesis Demo Complete ===${NC}"
echo ""
echo "The export/import commands provide:"
echo "✅ Complete blockchain data export in portable JSON format"
echo "✅ Idempotent import (safe to re-run without duplicating data)"
echo "✅ Parallel processing with configurable workers"
echo "✅ Dry-run mode for testing"
echo "✅ State preservation including account balances"
echo ""
echo "For full SubnetEVM to C-Chain migration with 1,082,780+ blocks:"
echo "  1. Deploy SubnetEVM with existing data"
echo "  2. Export: lux export --rpc [SUBNET_RPC] --start 0 --end 1082780 --output subnet-export.json"
echo "  3. Import: lux import --file subnet-export.json --dest [C_CHAIN_RPC] --parallel 200"