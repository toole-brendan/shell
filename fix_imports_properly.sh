#!/bin/bash

# Fix Shell Reserve import paths properly
# This script updates only internal package imports, not external dependencies

echo "Fixing import paths in Shell Reserve codebase properly..."

# First, revert all changes to restore original imports
echo "Reverting all import changes..."
find . -name "*.go" -type f -exec sed -i '' \
    -e 's|"github.com/toole-brendan/shell/|"github.com/btcsuite/btcd/|g' \
    {} \;

# Now, selectively update only the internal packages that exist in our fork
echo "Updating only internal package imports..."

# List of internal packages that exist in our fork
INTERNAL_PACKAGES=(
    "addrmgr"
    "addresses"
    "blockchain"
    "btcjson"
    "chaincfg"
    "connmgr"
    "database"
    "genesis"
    "mempool"
    "mining"
    "netsync"
    "peer"
    "privacy"
    "rpcclient"
    "rpcserver"
    "txscript"
    "wire"
)

# Update imports for each internal package
for pkg in "${INTERNAL_PACKAGES[@]}"; do
    echo "Updating imports for package: $pkg"
    find . -name "*.go" -type f -exec sed -i '' \
        -e "s|\"github.com/btcsuite/btcd/${pkg}|\"github.com/toole-brendan/shell/${pkg}|g" \
        {} \;
done

# Also update the chaincfg/chainhash submodule since we have it locally
echo "Updating chaincfg/chainhash imports..."
find . -name "*.go" -type f -exec sed -i '' \
    -e 's|"github.com/btcsuite/btcd/chaincfg/chainhash|"github.com/toole-brendan/shell/chaincfg/chainhash|g' \
    {} \;

echo "Import paths fixed properly!"

# Show summary
echo ""
echo "Summary:"
echo "- External dependencies (btcutil, btcec, etc.) remain as github.com/btcsuite/btcd imports"
echo "- Internal packages that exist in our fork use github.com/toole-brendan/shell imports"
echo ""
echo "Next steps:"
echo "1. Run 'go mod tidy' to update dependencies"
echo "2. Fix any remaining compilation issues" 