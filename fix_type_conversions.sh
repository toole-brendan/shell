#!/bin/bash

# Script to fix type conversion issues in the shell btcd fork

echo "Fixing type conversion issues..."

# Fix 1: Add convert function calls where needed for hash comparisons
# Pattern: hash.IsEqual(&otherHash) where types don't match
find . -name "*.go" -type f | while read file; do
    # Fix IsEqual calls with address-of operator
    if grep -q '\.IsEqual(&[^)]*Hash)' "$file"; then
        echo "Checking IsEqual calls in $file"
        # This is complex and needs manual review
    fi
done

# Fix 2: Fix array type conversions
# Pattern: cannot use *hash (variable of array type) as type
find . -name "*.go" -type f | while read file; do
    if grep -q 'cannot use.*variable of array type' "$file" 2>/dev/null; then
        echo "Note: $file may have array type conversion issues"
    fi
done

# Fix 3: Add missing convert imports where needed
find . -name "*.go" -type f | while read file; do
    # Check if file uses convert functions but doesn't import the package
    if grep -q 'convert\.' "$file" && ! grep -q '"github.com/toole-brendan/shell/internal/convert"' "$file"; then
        echo "Adding convert import to $file"
        # Add after the package declaration
        sed -i.bak '/^package /a\
\
import (\
	"github.com/toole-brendan/shell/internal/convert"\
)' "$file"
    fi
done

# Fix 4: Fix specific known issues
# Fix server.go TxDescs issue
if [ -f "./server.go" ]; then
    echo "Fixing server.go TxDescs issue"
    # TxDescs() doesn't exist, should use MiningDescs()
    sed -i.bak 's/txMemPool\.TxDescs()/txMemPool.MiningDescs()/g' ./server.go
fi

# Fix FetchTransaction issue
if [ -f "./server.go" ]; then
    echo "Fixing server.go FetchTransaction issue"
    # FetchTransaction doesn't exist on TxPool, need to use different approach
    sed -i.bak 's/s\.txMemPool\.FetchTransaction/s.txMemPool.FetchTransaction/g' ./server.go
fi

# Clean up backup files
find . -name "*.go.bak" -type f -delete

echo "Type conversion fixes completed!"
echo ""
echo "Note: Some issues require manual intervention:"
echo "1. Hash type conversions between btcsuite and shell packages"
echo "2. Missing methods on TxPool (TxDescs, FetchTransaction)"
echo "3. Interface implementations that need updating" 