#!/bin/bash
# Full deployment script for Lux DO nodes
# This copies pre-generated keys from ~/.lux/keys/ to DO nodes

set -e

MAINNET_IP="164.92.101.46"
TESTNET_IP="24.144.93.58"
DEVNET_IP="143.110.230.60"

NETWORK=$1
KEY_NAME=${2:-node0}  # Default to node0

if [ -z "$NETWORK" ]; then
    echo "Usage: $0 <network> [key-name]"
    echo "  network: mainnet, testnet, or devnet"
    echo "  key-name: name of key directory in ~/.lux/keys/ (default: node0)"
    exit 1
fi

case $NETWORK in
    mainnet) IP=$MAINNET_IP; NETWORK_ID=1 ;;
    testnet) IP=$TESTNET_IP; NETWORK_ID=2 ;;
    devnet) IP=$DEVNET_IP; NETWORK_ID=3 ;;
    *) echo "Invalid network: $NETWORK"; exit 1 ;;
esac

KEY_DIR="$HOME/.lux/keys/$KEY_NAME"
if [ ! -d "$KEY_DIR" ]; then
    echo "Error: Key directory not found: $KEY_DIR"
    echo "Available keys:"
    ls ~/.lux/keys/
    exit 1
fi

echo "=== Deploying Lux $NETWORK Node to $IP ==="
echo "Using keys from: $KEY_DIR"

# Create directories on DO
ssh root@$IP "mkdir -p /data/lux/{config,staking,plugins,db,logs}"

# Upload staking keys
echo "Uploading staking keys..."
scp "$KEY_DIR/staker.crt" root@$IP:/data/lux/staking/
scp "$KEY_DIR/staker.key" root@$IP:/data/lux/staking/

# Check for signer key in staking subdir
if [ -f "$KEY_DIR/staking/signer.key" ]; then
    scp "$KEY_DIR/staking/signer.key" root@$IP:/data/lux/staking/
elif [ -f "$KEY_DIR/bls/signer.key" ]; then
    scp "$KEY_DIR/bls/signer.key" root@$IP:/data/lux/staking/
fi

# Set correct permissions
ssh root@$IP "chmod 600 /data/lux/staking/*.key"

# Upload genesis files
echo "Uploading genesis files..."
GENESIS_DIR="$HOME/work/lux/genesis/configs/$NETWORK"
scp "$GENESIS_DIR/genesis.json" root@$IP:/data/lux/config/
scp "$GENESIS_DIR/cchain.json" root@$IP:/data/lux/config/ 2>/dev/null || true

# Create systemd service
echo "Creating systemd service..."
ssh root@$IP << REMOTE
cat > /etc/systemd/system/luxd.service << 'SYSTEMD'
[Unit]
Description=Lux $NETWORK Node
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/luxd \\
    --network-id=$NETWORK_ID \\
    --genesis-file=/data/lux/config/genesis.json \\
    --http-host=0.0.0.0 \\
    --http-port=9630 \\
    --staking-port=9631 \\
    --data-dir=/data/lux/db \\
    --plugin-dir=/data/lux/plugins \\
    --staking-tls-cert-file=/data/lux/staking/staker.crt \\
    --staking-tls-key-file=/data/lux/staking/staker.key \\
    --staking-signer-key-file=/data/lux/staking/signer.key \\
    --index-enabled \\
    --api-admin-enabled \\
    --log-level=warn \\
    --log-dir=/data/lux/logs \\
    --http-allowed-hosts=*
Restart=always
RestartSec=10
LimitNOFILE=65536
StandardOutput=null
StandardError=journal

[Install]
WantedBy=multi-user.target
SYSTEMD

systemctl daemon-reload
REMOTE

echo ""
echo "=== Deployment Complete ==="
echo "Node configured on $IP"
echo ""
echo "Next steps:"
echo "1. Upload snapshot: ./scripts/deploy-snapshot-to-do.sh $NETWORK"
echo "2. Start node: ssh root@$IP systemctl start luxd"
echo "3. Check status: ssh root@$IP systemctl status luxd"
