#!/bin/bash

# Dev Mode Launch Script for LUX - Simplest approach
# Uses --dev flag for single-node development

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

echo ""
echo -e "${CYAN}═══════════════════════════════════════════════════════════${NC}"
echo -e "${CYAN}    LUX Dev Mode - Single Node Development${NC}"
echo -e "${CYAN}═══════════════════════════════════════════════════════════${NC}"
echo ""

# Configuration
HTTP_PORT=9630
STAKING_PORT=9631

# Paths
LUXD_BIN="/home/z/work/lux/node/build/luxd"
DATA_DIR="${HOME}/.luxd-dev"
LOG_FILE="${DATA_DIR}/logs/luxd-dev.log"
PID_FILE="${DATA_DIR}/luxd-dev.pid"

# Kill any existing luxd processes
echo -e "${YELLOW}Stopping any existing luxd processes...${NC}"
pkill -f luxd 2>/dev/null || true
sleep 2

# Clean up old data
if [ -d "${DATA_DIR}" ]; then
    echo -e "${YELLOW}Removing old dev data directory...${NC}"
    rm -rf "${DATA_DIR}"
fi

# Ensure directories exist
echo -e "${BLUE}Creating required directories...${NC}"
mkdir -p "${DATA_DIR}"
mkdir -p "$(dirname ${LOG_FILE})"
echo -e "${GREEN}✓ Directories created${NC}"

echo ""
echo -e "${CYAN}Starting luxd in DEV mode...${NC}"
echo -e "${CYAN}═══════════════════════════════════════════════════════════${NC}"
echo ""

cd /home/z/work/lux/node

echo -e "${YELLOW}Launching luxd with --dev flag...${NC}"
DISABLE_MIGRATION_DETECTION=1 nohup ./build/luxd \
  --dev \
  --data-dir="${DATA_DIR}" \
  --http-host=0.0.0.0 \
  --http-port=${HTTP_PORT} \
  --staking-port=${STAKING_PORT} \
  --http-allowed-hosts="*" \
  --http-allowed-origins="*" \
  --api-admin-enabled \
  --api-health-enabled \
  --api-info-enabled \
  --api-metrics-enabled \
  --log-level=info \
  --log-display-level=info \
  > "${LOG_FILE}" 2>&1 &

PID=$!
echo ${PID} > "${PID_FILE}"

echo -e "${GREEN}✓ luxd started with PID: ${PID}${NC}"
echo ""
echo -e "${YELLOW}Waiting for node initialization (30 seconds)...${NC}"

# Wait and show progress
for i in {1..30}; do
    sleep 1
    echo -n "."
done
echo ""

# Check if node is running
if ps -p ${PID} > /dev/null; then
    echo -e "${GREEN}✓ Node is running!${NC}"
    echo ""

    # Test C-Chain RPC
    echo -e "${BLUE}Testing C-Chain RPC endpoint...${NC}"
    BLOCK_RESULT=$(curl -s -X POST --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
         -H 'content-type:application/json;' http://localhost:${HTTP_PORT}/ext/bc/C/rpc)

    if echo "$BLOCK_RESULT" | grep -q "result"; then
        echo -e "${GREEN}✓ C-Chain RPC is responding!${NC}"
        BLOCK_HEX=$(echo "$BLOCK_RESULT" | grep -o '"result":"0x[^"]*"' | cut -d'"' -f4)
        BLOCK_DEC=$((16#${BLOCK_HEX#0x}))
        echo -e "${BLUE}Current block: ${BLOCK_DEC}${NC}"
    else
        echo -e "${YELLOW}⚠ C-Chain may still be initializing...${NC}"
        echo -e "${BLUE}Response: ${BLOCK_RESULT}${NC}"
    fi

    echo ""
    echo -e "${CYAN}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}    LUX Dev Node Started Successfully!${NC}"
    echo -e "${CYAN}═══════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "${BLUE}API Endpoints:${NC}"
    echo -e "  Info:     ${GREEN}http://localhost:${HTTP_PORT}/ext/info${NC}"
    echo -e "  Health:   ${GREEN}http://localhost:${HTTP_PORT}/ext/health${NC}"
    echo -e "  Metrics:  ${GREEN}http://localhost:${HTTP_PORT}/ext/metrics${NC}"
    echo -e "  C-Chain:  ${GREEN}http://localhost:${HTTP_PORT}/ext/bc/C/rpc${NC}"
    echo -e "  P-Chain:  ${GREEN}http://localhost:${HTTP_PORT}/ext/bc/P${NC}"
    echo -e "  X-Chain:  ${GREEN}http://localhost:${HTTP_PORT}/ext/bc/X${NC}"
    echo ""
    echo -e "${BLUE}Quick Tests:${NC}"
    echo -e "  Get block height:"
    echo -e "  ${YELLOW}curl -X POST --data '{\"jsonrpc\":\"2.0\",\"method\":\"eth_blockNumber\",\"params\":[],\"id\":1}' -H 'content-type:application/json;' http://localhost:${HTTP_PORT}/ext/bc/C/rpc${NC}"
    echo ""
    echo -e "  Get balance for any address:"
    echo -e "  ${YELLOW}curl -X POST --data '{\"jsonrpc\":\"2.0\",\"method\":\"eth_getBalance\",\"params\":[\"0x9011E888251AB053B7bD1cdB598Db4f9DEd94714\",\"latest\"],\"id\":1}' -H 'content-type:application/json;' http://localhost:${HTTP_PORT}/ext/bc/C/rpc${NC}"
    echo ""
    echo -e "${BLUE}Management:${NC}"
    echo -e "  View logs:    ${YELLOW}tail -f ${LOG_FILE}${NC}"
    echo -e "  Stop node:    ${YELLOW}kill ${PID}${NC}"
    echo ""
else
    echo -e "${RED}✗ Node failed to start!${NC}"
    echo -e "${YELLOW}Check the log file for details:${NC}"
    echo -e "${YELLOW}tail -100 ${LOG_FILE}${NC}"
    exit 1
fi
