#!/bin/bash

# Update node/ids imports to ids
find pkg sdk -name "*.go" -type f ! -name "*.disabled" -exec sed -i.bak \
  -e 's|"github.com/luxfi/node/ids"|"github.com/luxfi/ids"|g' \
  {} \;

# Clean up backup files
find . -name "*.bak" -delete