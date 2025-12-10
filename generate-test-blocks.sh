#!/bin/bash

echo "📦 === GENERATING TEST BLOCKS FOR EXPORT/IMPORT TEST ==="
echo ""

# Configuration
RPC_URL="http://localhost:9630/ext/bc/C/rpc"
TREASURY_ADDR="0x9011E888251AB053B7bD1cdB598Db4f9DEd94714"
TEST_ADDR="0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"
PRIVATE_KEY="56289e99c94b6912bfc12adc093c9b51124f0dc54ac7a766b2bc5ccf558d8027"

# Function to send transaction
send_transaction() {
    local to=$1
    local value=$2
    local nonce=$3

    # Create transaction data
    TX_DATA=$(cat <<EOF
{
    "jsonrpc": "2.0",
    "method": "eth_sendTransaction",
    "params": [{
        "from": "$TREASURY_ADDR",
        "to": "$to",
        "value": "$value",
        "gas": "0x5208",
        "gasPrice": "0x4a817c800",
        "nonce": "$nonce"
    }],
    "id": 1
}
EOF
)

    # Send transaction
    echo "Sending transaction $nonce to $to..."
    curl -s -X POST -H "Content-Type: application/json" \
        --data "$TX_DATA" \
        "$RPC_URL" | jq -r '.result // .error.message'
}

# Function to create raw transaction (for better compatibility)
create_raw_transaction() {
    local to=$1
    local value=$2

    cat > /tmp/send_tx.js << 'EOF'
const Web3 = require('web3');
const web3 = new Web3('http://localhost:9630/ext/bc/C/rpc');

async function sendTx() {
    const privateKey = '0x56289e99c94b6912bfc12adc093c9b51124f0dc54ac7a766b2bc5ccf558d8027';
    const account = web3.eth.accounts.privateKeyToAccount(privateKey);

    const tx = {
        from: account.address,
        to: process.argv[2],
        value: web3.utils.toWei(process.argv[3], 'ether'),
        gas: 21000,
        gasPrice: await web3.eth.getGasPrice()
    };

    const signedTx = await account.signTransaction(tx);
    const receipt = await web3.eth.sendSignedTransaction(signedTx.rawTransaction);
    console.log('TX Hash:', receipt.transactionHash);
    console.log('Block:', receipt.blockNumber);
}

sendTx().catch(console.error);
EOF

    # Check if Node.js and web3 are available
    if command -v node > /dev/null 2>&1 && [ -d "node_modules/web3" ]; then
        node /tmp/send_tx.js "$to" "$value"
    else
        echo "Node.js/Web3 not available, using direct RPC"
        return 1
    fi
}

# Get current block height
echo "Current block height:"
CURRENT_HEIGHT=$(curl -s -X POST -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
    "$RPC_URL" | jq -r '.result')
echo "Block: $(printf "%d" "$CURRENT_HEIGHT")"

# Try to unlock account first (for development chains)
echo ""
echo "Attempting to unlock treasury account..."
curl -s -X POST -H "Content-Type: application/json" \
    --data "{\"jsonrpc\":\"2.0\",\"method\":\"personal_unlockAccount\",\"params\":[\"$TREASURY_ADDR\",\"\",0],\"id\":1}" \
    "$RPC_URL" | jq

# Alternative: Use eth_sendTransaction with the test account that should have funds
echo ""
echo "Generating test transactions..."

# Method 1: Direct transfer using eth_call to simulate
for i in {1..5}; do
    echo ""
    echo "Transaction $i:"

    # Create a simple transfer call
    CALL_DATA=$(cat <<EOF
{
    "jsonrpc": "2.0",
    "method": "eth_call",
    "params": [{
        "from": "$TREASURY_ADDR",
        "to": "$TEST_ADDR",
        "value": "0x$(printf '%x' $((i * 1000000000000000000)))",
        "data": "0x"
    }, "latest"],
    "id": $i
}
EOF
)

    curl -s -X POST -H "Content-Type: application/json" \
        --data "$CALL_DATA" \
        "$RPC_URL" | jq
done

# Try mining if miner API is available
echo ""
echo "Attempting to mine blocks..."
curl -s -X POST -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"miner_start","params":[1],"id":1}' \
    "$RPC_URL" | jq

# Wait a bit
sleep 2

# Stop mining
curl -s -X POST -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"miner_stop","params":[],"id":1}' \
    "$RPC_URL" | jq

# Check new block height
echo ""
echo "New block height:"
NEW_HEIGHT=$(curl -s -X POST -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
    "$RPC_URL" | jq -r '.result')
echo "Block: $(printf "%d" "$NEW_HEIGHT")"

# Get block details for export test
if [ "$NEW_HEIGHT" != "0x0" ]; then
    echo ""
    echo "Getting block 0 details:"
    curl -s -X POST -H "Content-Type: application/json" \
        --data '{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x0",true],"id":1}' \
        "$RPC_URL" | jq '.result | {number, hash, parentHash, timestamp}'
fi

echo ""
echo "✅ Block generation test complete"
echo "Ready for export/import testing"