#!/bin/bash

# Script to fix type conversion issues between btcsuite and shell types (macOS compatible)

echo "Fixing type conversion issues..."

# First, let's identify files that need fixing
echo "Identifying files with type conversion issues..."

# Files that use btcutil.NewBlock with shell types
files_with_newblock=$(grep -r "btcutil\.NewBlock" . --include="*.go" | grep -v "vendor" | grep -v "internal/convert" | cut -d: -f1 | sort -u)

# Files that use btcutil.NewTx with shell types  
files_with_newtx=$(grep -r "btcutil\.NewTx" . --include="*.go" | grep -v "vendor" | grep -v "internal/convert" | cut -d: -f1 | sort -u)

# Combine and deduplicate
all_files=$(echo -e "$files_with_newblock\n$files_with_newtx" | sort -u | grep -v "^$")

echo "Files to fix:"
echo "$all_files"

# Process each file
for file in $all_files; do
    echo "Processing $file..."
    
    # Check if the file already imports the convert package
    if ! grep -q "github.com/toole-brendan/shell/internal/convert" "$file"; then
        # Add the import after the last import statement
        # First, find the line number of the last import
        last_import_line=$(grep -n "^import (" "$file" | tail -1 | cut -d: -f1)
        
        if [ -n "$last_import_line" ]; then
            # Find the closing parenthesis of the import block
            closing_paren_line=$(tail -n +$last_import_line "$file" | grep -n "^)" | head -1 | cut -d: -f1)
            closing_paren_line=$((last_import_line + closing_paren_line - 1))
            
            # Insert the convert import before the closing parenthesis
            # Use a temporary file for macOS compatibility
            awk -v line="$closing_paren_line" 'NR==line{print "\t\"github.com/toole-brendan/shell/internal/convert\""}1' "$file" > "$file.tmp" && mv "$file.tmp" "$file"
        else
            # Single import style, need to convert to multi-import
            import_line=$(grep -n "^import " "$file" | head -1 | cut -d: -f1)
            if [ -n "$import_line" ]; then
                # Get the current import
                current_import=$(sed -n "${import_line}p" "$file" | sed 's/import //')
                
                # Create a temporary file with the new import block
                awk -v line="$import_line" -v import="$current_import" 'NR==line{print "import ("; print "\t" import; print "\t\"github.com/toole-brendan/shell/internal/convert\""; print ")"; next}1' "$file" > "$file.tmp" && mv "$file.tmp" "$file"
            fi
        fi
    fi
    
    # Now fix the actual type conversions
    # Fix btcutil.NewBlock calls
    sed -i '' 's/btcutil\.NewBlock(\([^)]*\))/convert.NewShellBlock(\1)/g' "$file"
    
    # Fix btcutil.NewTx calls
    sed -i '' 's/btcutil\.NewTx(\([^)]*\))/convert.NewShellTx(\1)/g' "$file"
    
done

echo "Type conversion fixes complete!"

# Now let's check if there are other type conversion issues
echo ""
echo "Checking for remaining type conversion issues..."

# Run a build to see what errors remain
echo "Running build to check for errors..."
go build ./... 2>&1 | grep -E "(cannot use|type mismatch)" | head -20

echo ""
echo "Done!" 