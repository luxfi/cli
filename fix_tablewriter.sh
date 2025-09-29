#!/bin/bash
# Quick script to comment out old tablewriter API calls
# This is a temporary fix to get CLI building

echo "Fixing tablewriter API issues temporarily..."

# Comment out all SetHeader, SetRowLine, SetAutoMergeCells, SetAlignment calls
find cmd pkg -name "*.go" -exec sed -i 's/^\([[:space:]]*\)table\.SetHeader/\1\/\/ table.SetHeader/' {} \;
find cmd pkg -name "*.go" -exec sed -i 's/^\([[:space:]]*\)table\.SetRowLine/\1\/\/ table.SetRowLine/' {} \;
find cmd pkg -name "*.go" -exec sed -i 's/^\([[:space:]]*\)table\.SetAutoMergeCells/\1\/\/ table.SetAutoMergeCells/' {} \;
find cmd pkg -name "*.go" -exec sed -i 's/^\([[:space:]]*\)table\.SetAlignment/\1\/\/ table.SetAlignment/' {} \;
find cmd pkg -name "*.go" -exec sed -i 's/^\([[:space:]]*\)table\.SetBorder/\1\/\/ table.SetBorder/' {} \;
find cmd pkg -name "*.go" -exec sed -i 's/^\([[:space:]]*\)table\.SetAutoWrapText/\1\/\/ table.SetAutoWrapText/' {} \;
find cmd pkg -name "*.go" -exec sed -i 's/^\([[:space:]]*\)table\.SetCaption/\1\/\/ table.SetCaption/' {} \;

echo "Done. Tables will render without formatting but CLI should build now."