#!/bin/bash
# Deploy snapshot directly to DigitalOcean nodes

set -e

MAINNET_IP="164.92.101.46"
TESTNET_IP="24.144.93.58"
DEVNET_IP="143.110.230.60"

DATE=$(date +%Y-%m-%d)

usage() {
    echo "Usage: $0 <network> [snapshot-name]"
    echo "  network: mainnet, testnet, or devnet"
    echo "  snapshot-name: optional, defaults to network-YYYY-MM-DD"
    exit 1
}

NETWORK=$1
SNAPSHOT_NAME=${2:-$NETWORK-$DATE}

case $NETWORK in
    mainnet) IP=$MAINNET_IP ;;
    testnet) IP=$TESTNET_IP ;;
    devnet) IP=$DEVNET_IP ;;
    *) usage ;;
esac

echo "=== Deploying $SNAPSHOT_NAME to $NETWORK ($IP) ==="

# Create snapshot tarball from local ~/.lux/snapshots
SNAP_DIR="$HOME/.lux/snapshots/$NETWORK"
if [ ! -d "$SNAP_DIR" ]; then
    echo "Error: Snapshot directory not found: $SNAP_DIR"
    echo "Run: lux snapshot --$NETWORK --full"
    exit 1
fi

echo "Creating compressed snapshot..."
cd $HOME/.lux/snapshots
tar -cf - $NETWORK | zstd -T0 > /tmp/$SNAPSHOT_NAME.tar.zst
ls -lh /tmp/$SNAPSHOT_NAME.tar.zst

echo "Uploading to $IP..."
scp /tmp/$SNAPSHOT_NAME.tar.zst root@$IP:/tmp/

echo "Restoring on remote..."
# shellcheck disable=SC2087 # Intentional: local vars ($SNAPSHOT_NAME, $NETWORK) must expand client-side
ssh root@$IP << REMOTE
    set -e
    systemctl stop luxd || true
    rm -rf /data/lux/db/*
    mkdir -p /data/lux/db
    cd /data/lux/db
    zstd -d /tmp/$SNAPSHOT_NAME.tar.zst -c | tar xf -
    mv $NETWORK/* . || true
    rm -rf $NETWORK
    rm /tmp/$SNAPSHOT_NAME.tar.zst
    ls -la
    systemctl start luxd
    sleep 10
    curl -s -X POST --data '{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}' -H 'content-type:application/json;' http://localhost:9630/ext/bc/C/rpc | jq
REMOTE

echo "Done! $NETWORK node deployed"
