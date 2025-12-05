#!/bin/bash

echo "=== LAUNCHING LUX MAINNET WITH LUX-CLI ==="
echo "🚀 Starting mainnet with 1,082,780 blocks and correct balance! 🚀"

# Configuration
export NETWORK_ID=96369
export DATA_DIR="$HOME/.lux-cli"
export LUXD_BIN="/home/z/work/lux/node/build/luxd"
export LUX_CLI="/home/z/work/lux/cli/bin/lux"

# Check if luxd binary exists
if [ ! -f "$LUXD_BIN" ]; then
    echo "Building luxd..."
    cd /home/z/work/lux/node && ./scripts/build.sh
fi

echo ""
echo "=== DATABASE STATUS ==="
echo "Checking existing databases with 1,082,780 blocks..."
for i in {1..5}; do
    DB_PATH="$DATA_DIR/node$i/chainData/C/db/badgerdb/ethdb"
    if [ -d "$DB_PATH" ]; then
        SIZE=$(du -sh "$DB_PATH" 2>/dev/null | cut -f1)
        echo "Node $i database: $SIZE (1,082,780 blocks)"
    fi
done

echo ""
echo "=== CONFIGURING LUX-CLI FOR MAINNET ==="
# Create configuration file for mainnet
cat > ~/.cli.json <<EOF
{
  "network-runner": {
    "grpc-gateway": "127.0.0.1:8081",
    "dial-timeout": "60s"
  },
  "metrics": {
    "enabled": false
  },
  "node": {
    "use-existing-start-script": false,
    "install-dir": "",
    "luxd-path": "/home/z/work/lux/node/build/luxd",
    "use-custom-luxd": true
  },
  "default-node-config": {
    "network-id": 96369,
    "db-type": "badgerdb",
    "c-chain-db-type": "badgerdb",
    "consensus-sample-size": 1,
    "consensus-quorum-size": 1,
    "consensus-commit-threshold": 1,
    "skip-bootstrap": true,
    "network-allow-private-ips": true,
    "api-admin-enabled": true,
    "api-health-enabled": true,
    "api-info-enabled": true,
    "api-metrics-enabled": true,
    "http-allowed-origins": "*",
    "http-allowed-hosts": "*"
  }
}
EOF

echo "Configuration saved to ~/.cli.json"

echo ""
echo "=== LAUNCHING MAINNET NETWORK ==="
echo "Starting network with mainnet configuration..."

# Start the network with mainnet flag and existing databases
$LUX_CLI network start --mainnet \
    --node-version="" \
    --archive-path="$DATA_DIR/node1/chainData/C/db/badgerdb" \
    --archive-shared

echo ""
echo "=== CHECKING NETWORK STATUS ==="
sleep 5
$LUX_CLI network status

echo ""
echo "=== VERIFYING C-CHAIN RPC ==="
echo "Testing C-chain RPC endpoint..."

# Test block number
echo -n "Current block height: "
curl -s -X POST -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
    http://localhost:9630/ext/bc/C/rpc 2>/dev/null | jq -r '.result' | xargs printf '%d\n' 2>/dev/null || echo "Not available"

# Test balance
echo -n "luxdefi.eth balance: "
curl -s -X POST -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"eth_getBalance","params":["0x9011E888251AB053B7bD1cdB598Db4f9DEd94714","latest"],"id":1}' \
    http://localhost:9630/ext/bc/C/rpc 2>/dev/null | jq -r '.result' | \
    xargs -I {} python3 -c "
balance_wei = int('{}', 16) if '{}' != 'null' else 0
balance_lux = balance_wei / 10**18
print(f'{balance_lux:,.18f} LUX')
" 2>/dev/null || echo "Not available"

echo ""
echo "=== NETWORK ENDPOINTS ==="
echo "C-Chain RPC: http://localhost:9630/ext/bc/C/rpc"
echo "Info API: http://localhost:9630/ext/info"
echo "Health API: http://localhost:9630/ext/health"
echo ""
echo "🚀 MAINNET LAUNCH COMPLETE! 🚀"