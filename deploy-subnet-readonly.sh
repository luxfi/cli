#!/bin/bash

# Deploy SubnetEVM in read-only mode using existing PebbleDB
# NO COPYING - just point at the existing database

set -e

echo "🚀 === SubnetEVM Read-Only Deployment ==="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
SUBNET_ID="2G8mK7VCZX1dV8iPjkkTDMpYGZDCNLLVdTJVLmMsG5ZV7zKVmB"
EXISTING_DB="/home/z/.luxd-5node-rpc/node2/chains/${SUBNET_ID}/db"
HTTP_PORT=9640
DATA_DIR="/home/z/.luxd-subnet-readonly"
LOG_LEVEL="info"

# Check if database exists
if [ ! -d "$EXISTING_DB" ]; then
    echo -e "${RED}❌ SubnetEVM database not found at: $EXISTING_DB${NC}"
    echo "Looking for alternative locations..."

    # Try to find the database
    FOUND_DB=$(find /home/z -name "$SUBNET_ID" -type d 2>/dev/null | grep "chains.*db$" | head -1)

    if [ -n "$FOUND_DB" ]; then
        EXISTING_DB="$FOUND_DB"
        echo -e "${GREEN}✅ Found database at: $EXISTING_DB${NC}"
    else
        echo -e "${RED}Cannot find SubnetEVM database${NC}"
        exit 1
    fi
fi

echo -e "${GREEN}✅ Using existing database: $EXISTING_DB${NC}"

# Clean and prepare directory structure
echo "Preparing directory structure..."
rm -rf "$DATA_DIR"
mkdir -p "$DATA_DIR"/{staking,plugins,logs,configs/chains}

# Generate staking certificates
echo "Generating staking certificates..."
cd "$DATA_DIR/staking"
openssl req -x509 -newkey rsa:4096 -keyout staker.key -out staker.crt \
    -sha256 -days 365 -nodes -subj "/CN=SubnetReadOnly" 2>/dev/null
cp staker.key signer.key

# Copy the EVM plugin
echo "Setting up EVM plugin..."
EVM_PLUGIN="/home/z/.luxd/plugins/srEXiWaHuhNyGwPUi444Tu47ZEDwxTWrbQiuD7FmgSAQ6X7Dy"
if [ ! -f "$EVM_PLUGIN" ]; then
    echo "Building EVM plugin..."
    cd /home/z/work/lux/evm
    ./scripts/build.sh
    EVM_PLUGIN="/home/z/.luxd/plugins/srEXiWaHuhNyGwPUi444Tu47ZEDwxTWrbQiuD7FmgSAQ6X7Dy"
fi

cp "$EVM_PLUGIN" "$DATA_DIR/plugins/srEXiWaHuhNyGwPUi444Tu47ZEDwxTWrbQiuD7FmgSAQ6X7Dy"

# Create chain config directory
mkdir -p "$DATA_DIR/configs/chains/$SUBNET_ID"

# Create chain config for read-only mode
cat > "$DATA_DIR/configs/chains/$SUBNET_ID/config.json" << EOF
{
  "chain-id": 96369,
  "network-id": 96369,
  "state-sync-enabled": false,
  "pruning-enabled": false,
  "eth-apis": ["eth", "eth-filter", "net", "web3", "internal-eth", "internal-blockchain", "internal-transaction"],
  "rpc-gas-cap": 50000000,
  "rpc-tx-fee-cap": 100,
  "allow-unfinalized-queries": true,
  "allow-unprotected-txs": true,
  "local-txs-enabled": false,
  "api-max-duration": 0,
  "api-max-blocks-per-request": 0,
  "ws-cpu-refill-rate": 0,
  "ws-cpu-max-stored": 0,
  "preimages-enabled": false,
  "log-level": "info",
  "external-db-path": "$EXISTING_DB",
  "read-only-db": true
}
EOF

# Create node config
cat > "$DATA_DIR/config.json" << EOF
{
  "network-id": 96369,
  "data-dir": "$DATA_DIR",
  "db-dir": "$DATA_DIR/db",
  "log-dir": "$DATA_DIR/logs",
  "plugin-dir": "$DATA_DIR/plugins",
  "chain-config-dir": "$DATA_DIR/configs/chains",
  "log-level": "$LOG_LEVEL",
  "http-host": "0.0.0.0",
  "http-port": $HTTP_PORT,
  "staking-enabled": false,
  "sybil-protection-enabled": false,
  "consensus-sample-size": 1,
  "consensus-quorum-size": 1,
  "bootstrap-ips": "",
  "bootstrap-ids": "",
  "api-admin-enabled": true,
  "api-metrics-enabled": true,
  "api-health-enabled": true,
  "api-info-enabled": true,
  "index-enabled": false,
  "db-type": "pebbledb",
  "http-allowed-origins": "*",
  "http-allowed-hosts": "*",
  "whitelisted-subnets": "$SUBNET_ID"
}
EOF

# Start the node
echo ""
echo -e "${BLUE}Starting SubnetEVM in read-only mode...${NC}"
echo -e "${BLUE}HTTP Port: $HTTP_PORT${NC}"
echo -e "${BLUE}Database: $EXISTING_DB${NC}"
echo ""

cd "$DATA_DIR"
exec /home/z/work/lux/node/build/luxd --config-file="$DATA_DIR/config.json"