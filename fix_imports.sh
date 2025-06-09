#!/bin/bash

# Fix Shell Reserve import paths
# This script updates all Go files to use the correct import paths

echo "Fixing import paths in Shell Reserve codebase..."

# Count files before fixing
TOTAL_FILES=$(find . -name "*.go" -type f -exec grep -l "github.com/btcsuite/btcd" {} \; | wc -l)
echo "Found $TOTAL_FILES files with old import paths"

# Fix import paths in all Go files
find . -name "*.go" -type f -exec sed -i '' \
    -e 's|"github.com/btcsuite/btcd/|"github.com/toole-brendan/shell/|g' \
    -e 's|github.com/btcsuite/btcd/|github.com/toole-brendan/shell/|g' \
    {} \;

echo "Import paths updated!"

# Also fix any go.mod replace directives that might be incorrect
echo "Updating go.mod replace directives..."

# Remove old replace directives and add comprehensive ones
cat > go.mod.tmp << 'EOF'
module github.com/toole-brendan/shell

go 1.23.2

toolchain go1.24.1

require (
	github.com/btcsuite/btcd v0.24.0
	github.com/btcsuite/btcd/btcec/v2 v2.3.5
	github.com/btcsuite/btcd/btcutil v1.1.5
	github.com/btcsuite/btcd/chaincfg/chainhash v1.1.0
	github.com/btcsuite/btcd/v2transport v1.0.1
	github.com/btcsuite/btclog v1.0.0
	github.com/btcsuite/go-socks v0.0.0-20170105172521-4720035b7bfd
	github.com/btcsuite/websocket v0.0.0-20150119174127-31079b680792
	github.com/davecgh/go-spew v1.1.1
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1
	github.com/decred/dcrd/lru v1.1.3
	github.com/jessevdk/go-flags v1.6.1
	github.com/jrick/logrotate v1.1.2
	github.com/stretchr/testify v1.8.4
	github.com/syndtr/goleveldb v1.0.0
	golang.org/x/crypto v0.25.0
	golang.org/x/sys v0.22.0
	pgregory.net/rapid v1.2.0
)

require (
	github.com/aead/siphash v1.0.1 // indirect
	github.com/decred/dcrd/crypto/blake256 v1.0.0 // indirect
	github.com/golang/snappy v0.0.0-20180518054509-2e65f85255db // indirect
	github.com/kkdai/bstream v0.0.0-20161212061736-f391b8402d23 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
EOF

mv go.mod.tmp go.mod

echo "go.mod updated!"

# Count files after fixing
REMAINING_FILES=$(find . -name "*.go" -type f -exec grep -l "github.com/btcsuite/btcd" {} \; 2>/dev/null | wc -l)
echo "Remaining files with old paths: $REMAINING_FILES"

# Show any remaining issues
if [ "$REMAINING_FILES" -gt 0 ]; then
    echo "Files still containing old import paths:"
    find . -name "*.go" -type f -exec grep -l "github.com/btcsuite/btcd" {} \; 2>/dev/null | head -10
fi

echo "Done! Now run 'go mod tidy' to update dependencies."

# Script to fix missing imports in the shell btcd fork

echo "Fixing missing imports in the codebase..."

# Function to add import if not present
add_import() {
    local file=$1
    local import_line=$2
    local package_name=$3
    
    # Check if import already exists
    if ! grep -q "$package_name" "$file"; then
        # Check if file uses the package
        if grep -q "$package_name\." "$file"; then
            echo "Adding import $package_name to $file"
            # Find the import block and add the new import
            if grep -q '^import (' "$file"; then
                # Multi-line import block exists
                sed -i.bak "/^import (/a\\
	$import_line" "$file"
            else
                # Single import or no imports yet
                sed -i.bak "s/^package .*/&\n\nimport (\n	$import_line\n)/" "$file"
            fi
        fi
    fi
}

# Fix files that use convert package but don't import it
find . -name "*.go" -type f | while read file; do
    if grep -q 'convert\.' "$file" && ! grep -q '"github.com/toole-brendan/shell/internal/convert"' "$file"; then
        add_import "$file" '"github.com/toole-brendan/shell/internal/convert"' "convert"
    fi
done

# Fix files that reference undefined types
# Fix mempool.TxPool references
find . -name "*.go" -type f | while read file; do
    if grep -q 'mempool\.TxPool' "$file"; then
        echo "Note: $file references mempool.TxPool which may need to be defined"
    fi
done

# Fix blockchain.SequenceLock references
find . -name "*.go" -type f | while read file; do
    if grep -q 'blockchain\.SequenceLock' "$file"; then
        echo "Note: $file references blockchain.SequenceLock which may need to be defined"
    fi
done

# Clean up backup files
find . -name "*.go.bak" -type f -delete

echo "Import fixes completed!" 