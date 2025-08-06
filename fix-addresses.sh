#!/bin/bash

echo "Replacing common.Address with crypto.Address..."

# First, add crypto import where needed and replace common.Address
find . -name "*.go" -type f ! -path "./vendor/*" ! -path "./cmd/*" -exec grep -l "common\.Address" {} \; | while read file; do
    echo "Processing $file"
    
    # Check if luxfi/crypto is already imported
    if ! grep -q '"github.com/luxfi/crypto"' "$file"; then
        # Check if there's already a geth/common import
        if grep -q '"github.com/luxfi/geth/common"' "$file"; then
            # Replace the geth/common import with crypto
            sed -i 's|"github.com/luxfi/geth/common"|"github.com/luxfi/crypto"|g' "$file"
        else
            # Add crypto import after package declaration
            sed -i '/^package /a\\nimport "github.com/luxfi/crypto"' "$file"
        fi
    fi
    
    # Replace common.Address with crypto.Address
    sed -i 's/common\.Address/crypto.Address/g' "$file"
    
    # Replace common.HexToAddress with crypto.HexToAddress
    sed -i 's/common\.HexToAddress/crypto.HexToAddress/g' "$file"
    
    # Replace common.BytesToAddress with crypto.BytesToAddress
    sed -i 's/common\.BytesToAddress/crypto.BytesToAddress/g' "$file"
    
    # Replace common.IsHexAddress with crypto.IsHexAddress
    sed -i 's/common\.IsHexAddress/crypto.IsHexAddress/g' "$file"
done

# Fix any remaining common.Hash and common.Bytes references
find . -name "*.go" -type f ! -path "./vendor/*" ! -path "./cmd/*" -exec grep -l "common\." {} \; | while read file; do
    echo "Checking $file for remaining common types"
    
    # If only using common.Hash or common.Hex2Bytes, we might need both imports
    if grep -q "common\.Hash\|common\.Hex2Bytes\|common\.Big" "$file"; then
        # Keep geth/common for these types but ensure crypto is also imported
        if ! grep -q '"github.com/luxfi/crypto"' "$file"; then
            # Add crypto import if not present
            if grep -q '^import (' "$file"; then
                # Multi-line import
                sed -i '/^import (/a\\t"github.com/luxfi/crypto"' "$file"
            else
                # Single import, convert to multi-line
                sed -i '/^package /a\\nimport (\n\t"github.com/luxfi/crypto"\n)' "$file"
            fi
        fi
    fi
done

echo "Done! Now fixing import organization..."

# Clean up imports
goimports -w . 2>/dev/null || echo "goimports not found, skipping import organization"

echo "Address replacement complete!"