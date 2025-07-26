#!/bin/bash

# Test script for local network setup and EVM deployment
set -e

echo "Testing Local Network Setup..."
echo "=============================="

# Build the CLI
echo "1. Building lux-cli..."
go build -o lux-cli ./

# Clean any existing network
echo "2. Cleaning existing network..."
./lux-cli network clean --hard || true

# Start the local network
echo "3. Starting local network..."
./lux-cli network start

# Wait for network to be ready
echo "4. Waiting for network to be ready..."
sleep 10

# Check network status
echo "5. Checking network status..."
./lux-cli network status

# Create a test subnet with EVM
echo "6. Creating test EVM subnet..."
cat > test-genesis.json << EOF
{
  "config": {
    "chainId": 1337,
    "homesteadBlock": 0,
    "eip150Block": 0,
    "eip155Block": 0,
    "eip158Block": 0,
    "byzantiumBlock": 0,
    "constantinopleBlock": 0,
    "petersburgBlock": 0,
    "istanbulBlock": 0,
    "muirGlacierBlock": 0,
    "subnetEVMTimestamp": 0,
    "feeConfig": {
      "gasLimit": 8000000,
      "targetBlockRate": 2,
      "minBaseFee": 1000000000,
      "targetGas": 15000000,
      "baseFeeChangeDenominator": 36,
      "minBlockGasCost": 0,
      "maxBlockGasCost": 1000000,
      "blockGasCostStep": 200000
    }
  },
  "alloc": {
    "8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC": {
      "balance": "0x295BE96E64066972000000"
    }
  },
  "nonce": "0x0",
  "timestamp": "0x0",
  "extraData": "0x00",
  "gasLimit": "0x7A1200",
  "difficulty": "0x0",
  "mixHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "coinbase": "0x0000000000000000000000000000000000000000",
  "number": "0x0",
  "gasUsed": "0x0",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000"
}
EOF

echo "7. Creating subnet configuration..."
./lux-cli subnet create testsubnet --evm --genesis test-genesis.json

echo "8. Deploying subnet to local network..."
./lux-cli subnet deploy testsubnet --local

echo "9. Getting subnet info..."
./lux-cli subnet describe testsubnet

echo "10. Testing completed successfully!"
echo "==================================="

# Cleanup
rm -f test-genesis.json