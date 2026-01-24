#!/bin/bash
# Create snapshots from local networks and push to GitHub

set -e

SNAPSHOTS_DIR="$HOME/work/lux/snapshots"
DATE=$(date +%Y-%m-%d)

echo "=== Creating Lux Network Snapshots ==="

# Check block counts first
echo "Checking current block heights..."
MAINNET_BLOCKS=$(curl -s -X POST --data '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' -H 'content-type:application/json;' http://localhost:9630/ext/bc/C/rpc 2>/dev/null | jq -r '.result' | xargs printf "%d")
TESTNET_BLOCKS=$(curl -s -X POST --data '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' -H 'content-type:application/json;' http://localhost:9640/ext/bc/C/rpc 2>/dev/null | jq -r '.result' | xargs printf "%d")
DEVNET_BLOCKS=$(curl -s -X POST --data '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' -H 'content-type:application/json;' http://localhost:9650/ext/bc/C/rpc 2>/dev/null | jq -r '.result' | xargs printf "%d")

echo "Mainnet: $MAINNET_BLOCKS blocks"
echo "Testnet: $TESTNET_BLOCKS blocks"
echo "Devnet: $DEVNET_BLOCKS blocks"

# Minimum blocks required (mainnet should have ~1M)
if [ "$MAINNET_BLOCKS" -lt 1000000 ]; then
    echo "WARNING: Mainnet only has $MAINNET_BLOCKS blocks, expected ~1,082,780"
    echo "Continue anyway? (y/n)"
    read -r response
    if [ "$response" != "y" ]; then
        exit 1
    fi
fi

# Create snapshots using lux CLI
cd ~/work/lux/cli

echo "Creating mainnet snapshot..."
./bin/lux snapshot --mainnet --name mainnet-$DATE --full

echo "Creating testnet snapshot..."
./bin/lux snapshot --testnet --name testnet-$DATE --full

echo "Creating devnet snapshot..."
./bin/lux snapshot --devnet --name devnet-$DATE --full

# List snapshots
./bin/lux snapshot list

echo ""
echo "=== Pushing to GitHub ==="

cd $SNAPSHOTS_DIR
git pull

# Copy snapshots
for network in mainnet testnet devnet; do
    echo "Preparing $network snapshot..."
    mkdir -p $network
    
    # Find the snapshot directory
    SNAP_DIR="$HOME/.lux/snapshots/$network-$DATE"
    if [ -d "$SNAP_DIR" ]; then
        # Create tarball with zstd compression
        tar -cf - -C "$HOME/.lux/snapshots" "$network-$DATE" | zstd -19 -T0 > /tmp/$network-$DATE.tar.zst
        
        # Split into 99MB chunks for Git LFS
        split -b 99m /tmp/$network-$DATE.tar.zst $network/$network-$DATE.tar.zst.part
        
        # Create manifest
        cat > $network/manifest.json << MANIFEST
{
  "name": "$network-$DATE",
  "date": "$DATE",
  "network": "$network",
  "blocks": $((${network^^}_BLOCKS)),
  "parts": $(ls $network/$network-$DATE.tar.zst.part* | jq -R -s 'split("\n") | map(select(length > 0) | split("/")[-1])'),
  "restore": "cat $network-$DATE.tar.zst.part* | zstd -d | tar xf -"
}
MANIFEST
    fi
done

# Add and commit
git add .
git commit -m "Update snapshots $DATE" || true
git push origin main

echo "Done! Snapshots pushed to GitHub"
