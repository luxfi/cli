#!/bin/bash
# Deploy Lux node to DigitalOcean with mnemonic-based key derivation
# Usage: ./deploy-do-node.sh <network> <mnemonic>

set -e

MAINNET_IP="164.92.101.46"
TESTNET_IP="24.144.93.58"
DEVNET_IP="143.110.230.60"

NETWORK=$1
MNEMONIC="${2:-$LUX_MNEMONIC}"

if [ -z "$NETWORK" ] || [ -z "$MNEMONIC" ]; then
    echo "Usage: $0 <network> [mnemonic]"
    echo "  network: mainnet, testnet, or devnet"
    echo "  mnemonic: optional, uses LUX_MNEMONIC env var if not provided"
    exit 1
fi

case $NETWORK in
    mainnet) IP=$MAINNET_IP; NETWORK_ID=1; PORT=9630 ;;
    testnet) IP=$TESTNET_IP; NETWORK_ID=2; PORT=9640 ;;
    devnet) IP=$DEVNET_IP; NETWORK_ID=3; PORT=9650 ;;
    *) echo "Invalid network: $NETWORK"; exit 1 ;;
esac

echo "=== Deploying Lux $NETWORK Node to $IP ==="

# Upload genesis files
echo "Uploading genesis files..."
scp /Users/z/work/lux/genesis/configs/$NETWORK/genesis.json root@$IP:/data/lux/config/
scp /Users/z/work/lux/genesis/configs/$NETWORK/cchain.json root@$IP:/data/lux/config/ 2>/dev/null || true

# Create encrypted mnemonic file and setup on remote
echo "Setting up node with mnemonic..."
ssh root@$IP << REMOTE
set -e

# Store mnemonic in memory-backed file system (tmpfs) for security
mkdir -p /run/lux
echo "$MNEMONIC" > /run/lux/mnemonic
chmod 600 /run/lux/mnemonic

# Create systemd service that derives keys from mnemonic
cat > /etc/systemd/system/luxd.service << 'SYSTEMD'
[Unit]
Description=Lux $NETWORK Node
After=network.target

[Service]
Type=simple
User=root
Environment="LUX_MNEMONIC_FILE=/run/lux/mnemonic"
Environment="NETWORK_ID=$NETWORK_ID"
ExecStartPre=/usr/local/bin/luxd-keygen --mnemonic-file=/run/lux/mnemonic --output=/data/lux/staking
ExecStart=/usr/local/bin/luxd \
    --network-id=$NETWORK_ID \
    --genesis-file=/data/lux/config/genesis.json \
    --http-host=0.0.0.0 \
    --http-port=9630 \
    --staking-port=9631 \
    --data-dir=/data/lux/db \
    --plugin-dir=/data/lux/plugins \
    --staking-tls-cert-file=/data/lux/staking/staker.crt \
    --staking-tls-key-file=/data/lux/staking/staker.key \
    --staking-signer-key-file=/data/lux/staking/signer.key \
    --index-enabled \
    --api-admin-enabled \
    --log-level=warn \
    --log-dir=/data/lux/logs \
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
echo "Node configured. Start with: systemctl start luxd"
REMOTE

echo "Done! Node configured on $IP"
echo "Next: Upload snapshot and start the node"
