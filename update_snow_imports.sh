#!/bin/bash

# Update all imports from snow/ to consensus/
echo "Updating imports from snow/ to consensus/ in cli..."

# Find all Go files and update imports
find . -name "*.go" -type f | while read -r file; do
    # Skip vendor directories
    if [[ "$file" == *"/vendor/"* ]]; then
        continue
    fi
    
    # Update imports
    sed -i 's|"github.com/luxfi/node/snow/|"github.com/luxfi/node/consensus/|g' "$file"
    sed -i 's|"github.com/luxfi/node/snow"|"github.com/luxfi/node/consensus"|g' "$file"
done

echo "Import updates complete."