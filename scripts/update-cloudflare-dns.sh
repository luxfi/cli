#!/bin/bash
# Update Cloudflare DNS for Lux ecosystem
# Usage: ./update-cloudflare-dns.sh
# Requires: CF_TOKEN environment variable

set -e

CF_TOKEN="${CF_TOKEN:-$CLOUDFLARE_API_TOKEN}"
if [ -z "$CF_TOKEN" ]; then
  echo "Error: CF_TOKEN or CLOUDFLARE_API_TOKEN not set"
  exit 1
fi

# Zone IDs
LUX_ZONE="287bdfd07cf016bd102f6394892e4759"
TEST_ZONE="3a67860079d4780ef609a2d3753be437"
DEV_ZONE="6db2a37606cb2cdf5b9751f199595677"
BUILD_ZONE="72486d2ed359900298da1e99a0c1a0fc"

# Server IPs - UPDATE THESE WHEN CHANGING INFRASTRUCTURE
MAINNET_IP="164.92.101.46"
TESTNET_IP="24.144.93.58"
DEVNET_IP="143.110.230.60"
EXPLORER_IP="67.205.158.227"

create_or_update_record() {
  local ZONE=$1
  local NAME=$2
  local IP=$3

  # Check if record exists
  EXISTING=$(curl -s -X GET "https://api.cloudflare.com/client/v4/zones/$ZONE/dns_records?type=A&name=$NAME" \
    -H "Authorization: Bearer $CF_TOKEN" | jq -r '.result[0].id // empty')

  if [ -n "$EXISTING" ]; then
    # Update existing
    RESULT=$(curl -s -X PUT "https://api.cloudflare.com/client/v4/zones/$ZONE/dns_records/$EXISTING" \
      -H "Authorization: Bearer $CF_TOKEN" \
      -H "Content-Type: application/json" \
      --data "{\"type\":\"A\",\"name\":\"$NAME\",\"content\":\"$IP\",\"ttl\":300,\"proxied\":false}")
  else
    # Create new
    RESULT=$(curl -s -X POST "https://api.cloudflare.com/client/v4/zones/$ZONE/dns_records" \
      -H "Authorization: Bearer $CF_TOKEN" \
      -H "Content-Type: application/json" \
      --data "{\"type\":\"A\",\"name\":\"$NAME\",\"content\":\"$IP\",\"ttl\":300,\"proxied\":false}")
  fi

  SUCCESS=$(echo $RESULT | jq -r '.success')
  if [ "$SUCCESS" = "true" ]; then
    echo "  ✓ $NAME -> $IP"
  else
    ERROR=$(echo $RESULT | jq -r '.errors[0].message // "unknown"')
    echo "  ✗ $NAME -> $IP ($ERROR)"
  fi
}

echo "=== Updating Cloudflare DNS for Lux Ecosystem ==="
echo ""

echo "lux.network (Mainnet):"
create_or_update_record $LUX_ZONE "api.lux.network" $MAINNET_IP
create_or_update_record $LUX_ZONE "rpc.lux.network" $MAINNET_IP
create_or_update_record $LUX_ZONE "explorer.lux.network" $EXPLORER_IP
create_or_update_record $LUX_ZONE "ipfs.lux.network" $EXPLORER_IP
create_or_update_record $LUX_ZONE "graph.lux.network" $EXPLORER_IP
create_or_update_record $LUX_ZONE "indexer.lux.network" $EXPLORER_IP
create_or_update_record $LUX_ZONE "exchange.lux.network" $EXPLORER_IP
create_or_update_record $LUX_ZONE "wallet.lux.network" $EXPLORER_IP

echo ""
echo "lux-test.network (Testnet):"
create_or_update_record $TEST_ZONE "lux-test.network" $TESTNET_IP
create_or_update_record $TEST_ZONE "api.lux-test.network" $TESTNET_IP
create_or_update_record $TEST_ZONE "rpc.lux-test.network" $TESTNET_IP
create_or_update_record $TEST_ZONE "explorer.lux-test.network" $EXPLORER_IP
create_or_update_record $TEST_ZONE "ipfs.lux-test.network" $EXPLORER_IP
create_or_update_record $TEST_ZONE "faucet.lux-test.network" $EXPLORER_IP
create_or_update_record $TEST_ZONE "graph.lux-test.network" $EXPLORER_IP
create_or_update_record $TEST_ZONE "exchange.lux-test.network" $EXPLORER_IP
create_or_update_record $TEST_ZONE "wallet.lux-test.network" $EXPLORER_IP

echo ""
echo "lux-dev.network (Devnet):"
create_or_update_record $DEV_ZONE "lux-dev.network" $DEVNET_IP
create_or_update_record $DEV_ZONE "api.lux-dev.network" $DEVNET_IP
create_or_update_record $DEV_ZONE "rpc.lux-dev.network" $DEVNET_IP
create_or_update_record $DEV_ZONE "explorer.lux-dev.network" $EXPLORER_IP
create_or_update_record $DEV_ZONE "ipfs.lux-dev.network" $EXPLORER_IP
create_or_update_record $DEV_ZONE "faucet.lux-dev.network" $EXPLORER_IP
create_or_update_record $DEV_ZONE "graph.lux-dev.network" $EXPLORER_IP
create_or_update_record $DEV_ZONE "exchange.lux-dev.network" $EXPLORER_IP
create_or_update_record $DEV_ZONE "wallet.lux-dev.network" $EXPLORER_IP

echo ""
echo "lux.build:"
create_or_update_record $BUILD_ZONE "lux.build" $EXPLORER_IP
create_or_update_record $BUILD_ZONE "www.lux.build" $EXPLORER_IP

echo ""
echo "=== DNS Update Complete ==="
echo ""
echo "Server IPs:"
echo "  Mainnet Node:  $MAINNET_IP"
echo "  Testnet Node:  $TESTNET_IP"
echo "  Devnet Node:   $DEVNET_IP"
echo "  Explorer Box:  $EXPLORER_IP"
