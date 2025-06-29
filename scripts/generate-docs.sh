#!/bin/bash
set -euo pipefail

# Create docs directory
mkdir -p docs

# Generate package list documentation
echo "# PCF-MCP Package Documentation" > docs/godoc.txt
echo "Generated on $(date)" >> docs/godoc.txt
echo "" >> docs/godoc.txt

# Get all packages
PACKAGES=$(go list ./...)

# Generate documentation for each package
for pkg in $PACKAGES; do
    echo "=================================================================================" >> docs/godoc.txt
    echo "PACKAGE: $pkg" >> docs/godoc.txt
    echo "=================================================================================" >> docs/godoc.txt
    go doc -all "$pkg" >> docs/godoc.txt 2>/dev/null || echo "No exported symbols in $pkg" >> docs/godoc.txt
    echo "" >> docs/godoc.txt
done

echo "Documentation generated in docs/godoc.txt"