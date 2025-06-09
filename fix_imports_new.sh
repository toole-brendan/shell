#!/bin/bash

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