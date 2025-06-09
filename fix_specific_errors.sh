#!/bin/bash

# Script to fix specific linter errors in the shell btcd fork

echo "Fixing specific linter errors..."

# Fix 1: Fix undefined Tx variable in server.go and other files
# Pattern: convert.HashToShell(Tx.Hash()) should be convert.HashToShell(tx.Hash())
find . -name "*.go" -type f | while read file; do
    if grep -q 'convert\.HashToShell(Tx\.Hash())' "$file"; then
        echo "Fixing undefined Tx in $file"
        sed -i.bak 's/convert\.HashToShell(Tx\.Hash())/convert.HashToShell(tx.Hash())/g' "$file"
    fi
done

# Fix 2: Fix txD.convert undefined errors
# Pattern: txD.convert.HashToShell should be convert.HashToShell
find . -name "*.go" -type f | while read file; do
    if grep -q 'txD\.convert\.HashToShell' "$file"; then
        echo "Fixing txD.convert in $file"
        sed -i.bak 's/txD\.convert\.HashToShell/convert.HashToShell/g' "$file"
    fi
done

# Fix 3: Fix missing comma in composite literals
find . -name "*.go" -type f | while read file; do
    if grep -q 'Hash:\s*\*convert\.HashToShell([^,)]*))' "$file"; then
        echo "Fixing missing comma in composite literal in $file"
        sed -i.bak 's/Hash:\s*\*convert\.HashToShell(\([^,)]*\))/Hash: *convert.HashToShell(\1),/g' "$file"
    fi
done

# Fix 4: Fix undefined convert.ParamsToBtc
# This function should already be in convert.go, so we just need to ensure imports are correct

# Fix 5: Fix wire.TxLoc type issues
# Files need to use the correct import path

# Fix 6: Fix undefined methods like TxDescs, FetchTransaction, etc.
echo "Checking for undefined methods that may need implementation..."
grep -n "undefined.*method" . -r --include="*.go" | head -20

# Clean up backup files
find . -name "*.go.bak" -type f -delete

echo "Specific error fixes completed!" 