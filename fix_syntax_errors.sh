#!/bin/bash

# Script to fix common syntax errors in the shell btcd fork

echo "Fixing syntax errors in the codebase..."

# Fix 1: Missing commas in composite literals with convert.HashToShell
# Pattern: convert.HashToShell(...))] should be convert.HashToShell(...),
find . -name "*.go" -type f | while read file; do
    if grep -q 'convert\.HashToShell([^)]*))]\s*=' "$file"; then
        echo "Fixing missing comma in $file"
        sed -i.bak 's/convert\.HashToShell(\([^)]*\)))]\s*=/convert.HashToShell(\1),/g' "$file"
    fi
done

# Fix 2: Extra closing parenthesis in map access
# Pattern: map[convert.HashToShell(...)))] should be map[*convert.HashToShell(...)]
find . -name "*.go" -type f | while read file; do
    if grep -q '\[convert\.HashToShell([^)]*)))\]' "$file"; then
        echo "Fixing extra parenthesis in map access in $file"
        sed -i.bak 's/\[convert\.HashToShell(\([^)]*\)))\]/[*convert.HashToShell(\1)]/g' "$file"
    fi
done

# Fix 3: Missing comma in wire.OutPoint composite literal
# Pattern: Hash: *convert.HashToShell(...))} should be Hash: *convert.HashToShell(...),}
find . -name "*.go" -type f | while read file; do
    if grep -q 'Hash:\s*\*convert\.HashToShell([^)]*))}\s*$' "$file"; then
        echo "Fixing missing comma in OutPoint literal in $file"
        sed -i.bak 's/Hash:\s*\*convert\.HashToShell(\([^)]*\))}/Hash: *convert.HashToShell(\1),}/g' "$file"
    fi
done

# Fix 4: Direct hash assignment issues
# Pattern: Hash: *tx.Hash())} should be Hash: *tx.Hash(),}
find . -name "*.go" -type f | while read file; do
    if grep -q 'Hash:\s*\*[^,]*\.Hash())}\s*$' "$file"; then
        echo "Fixing missing comma in hash assignment in $file"
        sed -i.bak 's/Hash:\s*\*\([^,]*\.Hash()\)}/Hash: *\1,}/g' "$file"
    fi
done

# Fix 5: Expected ']', found ')' errors
# Pattern: ]...)] should be ]...]
find . -name "*.go" -type f | while read file; do
    if grep -q '\]\s*)\s*=' "$file"; then
        echo "Fixing extra parenthesis after bracket in $file"
        sed -i.bak 's/\]\s*)\s*=/] =/g' "$file"
    fi
done

# Clean up backup files
find . -name "*.go.bak" -type f -delete

echo "Syntax error fixes completed!" 