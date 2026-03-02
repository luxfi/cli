#!/bin/bash
# Update Cloudflare DNS for Lux ecosystem
# Usage: ./update-cloudflare-dns.sh
# Requires: CF_TOKEN or CLOUDFLARE_API_TOKEN environment variable
# Can also extract from K8s: kubectl get secret cloudflare-api-credentials -n lux-system

set -e

CF_TOKEN="${CF_TOKEN:-$CLOUDFLARE_API_TOKEN}"
if [ -z "$CF_TOKEN" ]; then
  echo "Error: CF_TOKEN or CLOUDFLARE_API_TOKEN not set"
  echo "  export CF_TOKEN=\$(kubectl --context do-sfo3-lux-k8s get secret cloudflare-api-credentials -n lux-system -o jsonpath='{.data.apiKey}' | base64 -d)"
  exit 1
fi

# Zone IDs (Cloudflare)
LUX_ZONE="287bdfd07cf016bd102f6394892e4759"      # lux.network
TEST_ZONE="3a67860079d4780ef609a2d3753be437"      # lux-test.network
DEV_ZONE="6db2a37606cb2cdf5b9751f199595677"       # lux-dev.network
BUILD_ZONE="72486d2ed359900298da1e99a0c1a0fc"     # lux.build
MARKET_ZONE="f9e50bf2d0f12aba59b4f142dbc774b3"    # lux.market
EXCHANGE_ZONE="99ba4822517691fef46757c763490ad5"   # lux.exchange
SHOP_ZONE="55e83e0fe6054e761b139115077e2a1f"       # lux.shop

# Infrastructure IPs - UPDATE THESE WHEN CHANGING INFRASTRUCTURE
# lux-k8s (do-sfo3-lux-k8s) is the SINGLE cluster for all Lux services
LB_IP="24.199.71.113"  # lux-k8s hanzo-ingress DaemonSet (Traefik)

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
      --data "{\"type\":\"A\",\"name\":\"$NAME\",\"content\":\"$IP\",\"ttl\":1,\"proxied\":false}")
  else
    # Create new
    RESULT=$(curl -s -X POST "https://api.cloudflare.com/client/v4/zones/$ZONE/dns_records" \
      -H "Authorization: Bearer $CF_TOKEN" \
      -H "Content-Type: application/json" \
      --data "{\"type\":\"A\",\"name\":\"$NAME\",\"content\":\"$IP\",\"ttl\":1,\"proxied\":false}")
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
echo "LB IP: $LB_IP (lux-k8s)"
echo ""

echo "--- lux.network (API & Services) ---"
create_or_update_record $LUX_ZONE "api.lux.network" $LB_IP
create_or_update_record $LUX_ZONE "cloud.lux.network" $LB_IP

echo ""
echo "--- lux.network (Explorers) ---"
create_or_update_record $LUX_ZONE "explore.lux.network" $LB_IP
create_or_update_record $LUX_ZONE "explore-zoo.lux.network" $LB_IP
create_or_update_record $LUX_ZONE "explore-hanzo.lux.network" $LB_IP
create_or_update_record $LUX_ZONE "explore-spc.lux.network" $LB_IP
create_or_update_record $LUX_ZONE "explore-pars.lux.network" $LB_IP
create_or_update_record $LUX_ZONE "api-explore.lux.network" $LB_IP

echo ""
echo "--- lux.network (Indexers) ---"
create_or_update_record $LUX_ZONE "api-indexer.lux.network" $LB_IP
create_or_update_record $LUX_ZONE "api-indexer-pchain.lux.network" $LB_IP
create_or_update_record $LUX_ZONE "api-indexer-xchain.lux.network" $LB_IP
create_or_update_record $LUX_ZONE "api-indexer-achain.lux.network" $LB_IP
create_or_update_record $LUX_ZONE "api-indexer-bchain.lux.network" $LB_IP
create_or_update_record $LUX_ZONE "api-indexer-qchain.lux.network" $LB_IP
create_or_update_record $LUX_ZONE "api-indexer-tchain.lux.network" $LB_IP
create_or_update_record $LUX_ZONE "api-indexer-zchain.lux.network" $LB_IP
create_or_update_record $LUX_ZONE "api-indexer-kchain.lux.network" $LB_IP

echo ""
echo "--- lux.network (Apps) ---"
create_or_update_record $LUX_ZONE "exchange.lux.network" $LB_IP
create_or_update_record $LUX_ZONE "bridge.lux.network" $LB_IP
create_or_update_record $LUX_ZONE "bridge-api.lux.network" $LB_IP
create_or_update_record $LUX_ZONE "dex.lux.network" $LB_IP
create_or_update_record $LUX_ZONE "wallet.lux.network" $LB_IP
create_or_update_record $LUX_ZONE "safe.lux.network" $LB_IP
create_or_update_record $LUX_ZONE "mpc.lux.network" $LB_IP
create_or_update_record $LUX_ZONE "mpc-api.lux.network" $LB_IP
create_or_update_record $LUX_ZONE "market.lux.network" $LB_IP

echo ""
echo "--- TLD Domains ---"
create_or_update_record $BUILD_ZONE "lux.build" $LB_IP
create_or_update_record $BUILD_ZONE "www.lux.build" $LB_IP
create_or_update_record $MARKET_ZONE "lux.market" $LB_IP
create_or_update_record $MARKET_ZONE "www.lux.market" $LB_IP
create_or_update_record $EXCHANGE_ZONE "lux.exchange" $LB_IP
create_or_update_record $EXCHANGE_ZONE "www.lux.exchange" $LB_IP

echo ""
echo "--- lux-test.network (Testnet) ---"
create_or_update_record $TEST_ZONE "api.lux-test.network" $LB_IP
create_or_update_record $TEST_ZONE "explore.lux-test.network" $LB_IP
create_or_update_record $TEST_ZONE "exchange.lux-test.network" $LB_IP

echo ""
echo "--- lux-dev.network (Devnet) ---"
create_or_update_record $DEV_ZONE "api.lux-dev.network" $LB_IP
create_or_update_record $DEV_ZONE "explore.lux-dev.network" $LB_IP

echo ""
echo "=== DNS Update Complete ==="
echo ""
echo "All records point to LB: $LB_IP (lux-k8s cluster)"
echo ""
echo "Verify: dig +short lux.market @1.1.1.1"
