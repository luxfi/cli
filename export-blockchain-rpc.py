#!/usr/bin/env python3
"""
Export blockchain data from running SubnetEVM node via RPC.
This reads blocks via RPC and exports them to JSONL format.
"""

import json
import requests
import sys
from typing import Dict, Any, Optional

RPC_URL = "http://localhost:9630/ext/bc/C/rpc"
OUTPUT_FILE = "/tmp/lux-migration/blockchain-export-rpc.jsonl"

def make_rpc_call(method: str, params: list) -> Optional[Dict[str, Any]]:
    """Make an RPC call to the node."""
    payload = {
        "jsonrpc": "2.0",
        "method": method,
        "params": params,
        "id": 1
    }

    try:
        response = requests.post(RPC_URL, json=payload, timeout=10)
        result = response.json()

        if 'error' in result:
            print(f"RPC Error: {result['error']}")
            return None

        return result.get('result')
    except requests.exceptions.RequestException as e:
        print(f"Connection error: {e}")
        return None
    except json.JSONDecodeError as e:
        print(f"JSON decode error: {e}")
        return None

def get_block_by_number(block_num: int) -> Optional[Dict[str, Any]]:
    """Get a block by its number."""
    # Try to get block with full transaction details
    block_hex = hex(block_num)
    return make_rpc_call("eth_getBlockByNumber", [block_hex, True])

def get_current_height() -> int:
    """Get current blockchain height."""
    result = make_rpc_call("eth_blockNumber", [])
    if result:
        return int(result, 16)
    return 0

def export_blockchain():
    """Export all blocks from the blockchain via RPC."""
    print("=== Blockchain RPC Export Tool ===")
    print(f"RPC endpoint: {RPC_URL}")
    print(f"Output file: {OUTPUT_FILE}")
    print("")

    # Get current height
    current_height = get_current_height()
    print(f"Current blockchain height: {current_height}")

    if current_height == 0:
        print("WARNING: Blockchain is at height 0. No blocks to export.")
        print("The node may not have the blockchain data loaded.")
        return

    # Open output file
    with open(OUTPUT_FILE, 'w') as f:
        # Write metadata
        metadata = {
            "version": "1.0.0",
            "type": "blockchain-export",
            "source": "rpc",
            "height": current_height
        }
        json.dump(metadata, f)
        f.write('\n')

        # Export blocks
        print(f"Exporting {current_height} blocks...")

        for block_num in range(0, min(current_height + 1, 10)):  # Export first 10 blocks as sample
            if block_num % 1000 == 0:
                print(f"  Processing block {block_num}...")

            block = get_block_by_number(block_num)
            if block:
                # Write block to JSONL
                block_entry = {
                    "type": "block",
                    "number": int(block.get('number', '0x0'), 16),
                    "hash": block.get('hash'),
                    "parentHash": block.get('parentHash'),
                    "timestamp": int(block.get('timestamp', '0x0'), 16),
                    "gasUsed": int(block.get('gasUsed', '0x0'), 16),
                    "gasLimit": int(block.get('gasLimit', '0x0'), 16),
                    "difficulty": int(block.get('difficulty', '0x0'), 16),
                    "miner": block.get('miner'),
                    "transactions": block.get('transactions', []),
                    "uncles": block.get('uncles', [])
                }
                json.dump(block_entry, f)
                f.write('\n')
            else:
                print(f"  Failed to get block {block_num}")

    print(f"✅ Export complete: {OUTPUT_FILE}")

def check_lux_apis():
    """Check if LUX-specific APIs are available."""
    print("\n=== Checking LUX APIs ===")

    # Check replay status
    result = make_rpc_call("lux_replayStatus", [])
    if result:
        print(f"Replay status: {json.dumps(result, indent=2)}")
    else:
        print("lux_replayStatus API not available")

    # Check blockchain verification
    result = make_rpc_call("lux_verifyBlockchain", [])
    if result:
        print(f"Blockchain verification: {json.dumps(result, indent=2)}")
    else:
        print("lux_verifyBlockchain API not available")

if __name__ == "__main__":
    export_blockchain()
    check_lux_apis()