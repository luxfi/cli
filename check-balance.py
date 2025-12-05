#!/usr/bin/env python3
import json
import requests
import sys

RPC_URL = "http://localhost:9630/ext/bc/C/rpc"
TREASURY = "0x9011E888251AB053B7bD1cdB598Db4f9DEd94714"

def check_balance(address):
    payload = {
        "jsonrpc": "2.0",
        "method": "eth_getBalance",
        "params": [address, "latest"],
        "id": 1
    }

    try:
        response = requests.post(RPC_URL, json=payload, timeout=5)
        result = response.json()

        if 'result' in result:
            balance_hex = result['result']
            balance_wei = int(balance_hex, 16)
            balance_lux = balance_wei / 10**18

            print(f"Address: {address}")
            print(f"Balance: {balance_wei} wei")
            print(f"Balance: {balance_lux:.6f} LUX")
            return balance_wei
        else:
            print(f"Error: {result}")
            return None
    except Exception as e:
        print(f"Connection failed: {e}")
        print("Make sure the node is running on port 9630")
        return None

def get_block_height():
    payload = {
        "jsonrpc": "2.0",
        "method": "eth_blockNumber",
        "params": [],
        "id": 1
    }

    try:
        response = requests.post(RPC_URL, json=payload, timeout=5)
        result = response.json()

        if 'result' in result:
            height = int(result['result'], 16)
            print(f"Current block height: {height}")
            return height
        else:
            print(f"Error: {result}")
            return None
    except Exception as e:
        print(f"Connection failed: {e}")
        return None

if __name__ == "__main__":
    print("=== LUX Migrated Chain Balance Checker ===")
    print(f"RPC: {RPC_URL}")
    print()

    # Check block height
    get_block_height()
    print()

    # Check treasury balance
    print("Treasury Account:")
    check_balance(TREASURY)
    print()

    # Check additional addresses if provided
    if len(sys.argv) > 1:
        for addr in sys.argv[1:]:
            print(f"Checking {addr}:")
            check_balance(addr)
            print()
