#!/bin/bash
# Update all imports from github.com/luxfi/cli/ to github.com/luxfi/cli/v2/

echo "Updating CLI imports to v2..."

# Find all Go files and update imports
find . -name "*.go" -type f -not -path "./vendor/*" -not -path "./.git/*" | while read -r file; do
    # Update imports from github.com/luxfi/cli/ to github.com/luxfi/cli/v2/
    sed -i '' 's|"github.com/luxfi/cli/|"github.com/luxfi/cli/v2/|g' "$file"
done

echo "CLI import update complete!"