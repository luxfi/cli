#!/usr/bin/env python3
"""
Fix coinbase addresses in genesis files.
Coinbase addresses should be 40 hex characters (20 bytes), not 64 characters.
"""

import json
import os
import sys
from pathlib import Path

def fix_coinbase_in_json(data):
    """Recursively fix coinbase addresses in JSON data"""
    if isinstance(data, dict):
        if 'coinbase' in data:
            coinbase = data['coinbase']
            if coinbase.startswith('0x'):
                hex_part = coinbase[2:]
                # If it's 64 characters, truncate to 40
                if len(hex_part) == 64:
                    data['coinbase'] = '0x' + hex_part[:40]
                    print(f"  Fixed coinbase: {coinbase} -> {data['coinbase']}")
                elif len(hex_part) != 40:
                    print(f"  Warning: Unexpected coinbase length: {len(hex_part)} in {coinbase}")

        # Recurse into all dict values
        for key, value in data.items():
            if isinstance(value, (dict, list)):
                fix_coinbase_in_json(value)
            elif isinstance(value, str) and key == 'cChainGenesis':
                # Handle embedded JSON strings
                try:
                    embedded = json.loads(value)
                    fix_coinbase_in_json(embedded)
                    data[key] = json.dumps(embedded, separators=(',', ':'))
                except (json.JSONDecodeError, TypeError):
                    pass
    elif isinstance(data, list):
        for item in data:
            fix_coinbase_in_json(item)

def fix_genesis_file(filepath):
    """Fix a single genesis file"""
    print(f"Processing {filepath}...")

    try:
        with open(filepath, 'r') as f:
            data = json.load(f)

        # Make a backup
        backup_path = str(filepath) + '.backup'
        if not os.path.exists(backup_path):
            with open(backup_path, 'w') as f:
                json.dump(data, f, indent=2)
            print(f"  Created backup: {backup_path}")

        # Fix coinbase addresses
        fix_coinbase_in_json(data)

        # Write back
        with open(filepath, 'w') as f:
            json.dump(data, f, indent=2)

        print(f"  Saved fixed file: {filepath}")

    except Exception as e:
        print(f"  Error processing {filepath}: {e}")

def main():
    # Fix all genesis files in the node/genesis directory
    genesis_dir = Path("/home/z/work/lux/node/genesis")

    # Files to fix
    files_to_fix = [
        "genesis_mainnet.json",
        "genesis_testnet.json",
        "genesis_local.json",
        "genesis_test.json",
        "genesis_96369_migrated.json",
        "cchain_genesis_mainnet.json",
        "cchain_genesis_final.json"
    ]

    for filename in files_to_fix:
        filepath = genesis_dir / filename
        if filepath.exists():
            fix_genesis_file(filepath)
        else:
            print(f"File not found: {filepath}")

    print("\nGenesis files fixed!")
    print("Now rebuild the node to use the corrected genesis configuration.")

if __name__ == "__main__":
    main()