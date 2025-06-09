#!/bin/bash

# Script to fix remaining type conversion issues in the Shell codebase

echo "Fixing remaining type conversion issues..."

# First, let's add the convert import to files that need it
echo "Adding convert imports where needed..."

# Files that need convert import based on the errors
files_needing_convert=(
    "server.go"
    "rpcwebsocket.go"
    "rpcserver.go"
    "netsync/manager.go"
    "mempool/mempool.go"
    "mempool/mempool_test.go"
    "blockchain/validate.go"
    "blockchain/utxoviewpoint.go"
    "blockchain/scriptval.go"
    "blockchain/merkle.go"
    "blockchain/process.go"
    "blockchain/chain.go"
    "blockchain/chainio.go"
    "blockchain/accept.go"
    "blockchain/indexers/txindex.go"
    "blockchain/indexers/cfindex.go"
    "blockchain/indexers/addrindex.go"
    "database/ffldb/db.go"
    "mining/mining.go"
)

for file in "${files_needing_convert[@]}"; do
    if [ -f "$file" ]; then
        # Check if convert is already imported
        if ! grep -q '"github.com/toole-brendan/shell/internal/convert"' "$file"; then
            echo "Adding convert import to $file"
            # Add the import after the last import line
            awk '
            /^import \(/ { in_import = 1 }
            in_import && /^\)/ { 
                print "\t\"github.com/toole-brendan/shell/internal/convert\""
                in_import = 0 
            }
            { print }
            ' "$file" > "$file.tmp" && mv "$file.tmp" "$file"
        fi
    fi
done

# Now let's fix the type conversions
echo "Fixing hash type conversions..."

# Fix Hash conversions in map indexes and struct literals
find . -name "*.go" -type f | while read -r file; do
    # Skip vendor and test data directories
    if [[ "$file" == *"/vendor/"* ]] || [[ "$file" == *"/testdata/"* ]]; then
        continue
    fi
    
    # Create a temporary file
    tmp_file="${file}.tmp"
    
    # Fix map index issues with Hash types
    sed -E 's/\[\*([a-zA-Z0-9_]+)\.Hash\(\)\]/[convert.HashToShell(\1.Hash())]/g' "$file" > "$tmp_file"
    
    # Fix Hash field assignments in struct literals
    sed -i '' -E 's/Hash:([[:space:]]+)\*([a-zA-Z0-9_]+)\.Hash\(\)/Hash:\1*convert.HashToShell(\2.Hash())/g' "$tmp_file"
    
    # Fix IsEqual comparisons
    sed -i '' -E 's/\.IsEqual\(&([a-zA-Z0-9_]+)\.([a-zA-Z0-9_]+)\)/\.IsEqual(convert.HashToBtc(\&\1.\2))/g' "$tmp_file"
    
    # Fix function arguments expecting shell Hash
    sed -i '' -E 's/([a-zA-Z0-9_]+)\.Hash\(\)([[:space:]]*\))/convert.HashToShell(\1.Hash())\2/g' "$tmp_file"
    
    # Move the temp file back
    mv "$tmp_file" "$file"
done

# Fix specific patterns that need manual attention
echo "Fixing specific type conversion patterns..."

# Fix wire.OutPoint conversions
find . -name "*.go" -type f | while read -r file; do
    if [[ "$file" == *"/vendor/"* ]] || [[ "$file" == *"/testdata/"* ]]; then
        continue
    fi
    
    tmp_file="${file}.tmp"
    
    # Fix OutPoint in map indexes
    sed -E 's/\[txIn\.PreviousOutPoint\]/[convert.OutPointToShell(\&txIn.PreviousOutPoint)]/g' "$file" > "$tmp_file"
    
    # Fix OutPoint in function arguments
    sed -i '' -E 's/LookupEntry\(txIn\.PreviousOutPoint\)/LookupEntry(convert.OutPointToShell(\&txIn.PreviousOutPoint))/g' "$tmp_file"
    
    mv "$tmp_file" "$file"
done

# Fix btcutil.DecodeAddress calls
echo "Fixing btcutil.DecodeAddress calls..."
find . -name "*.go" -type f | while read -r file; do
    if [[ "$file" == *"/vendor/"* ]] || [[ "$file" == *"/testdata/"* ]]; then
        continue
    fi
    
    # Check if file contains DecodeAddress calls with shell params
    if grep -q "btcutil\.DecodeAddress.*chaincfg.*Params" "$file"; then
        tmp_file="${file}.tmp"
        
        # Replace the params argument in DecodeAddress calls
        sed -E 's/btcutil\.DecodeAddress\(([^,]+), ([^)]+)\)/btcutil.DecodeAddress(\1, convert.ParamsToBtc(\2))/g' "$file" > "$tmp_file"
        
        mv "$tmp_file" "$file"
    fi
done

# Fix NewAddress* functions
echo "Fixing NewAddress* function calls..."
find . -name "*.go" -type f | while read -r file; do
    if [[ "$file" == *"/vendor/"* ]] || [[ "$file" == *"/testdata/"* ]]; then
        continue
    fi
    
    tmp_file="${file}.tmp"
    
    # Fix various NewAddress* calls
    sed -E 's/btcutil\.NewAddressPubKeyHash\(([^,]+), ([^)]+)\)/btcutil.NewAddressPubKeyHash(\1, convert.ParamsToBtc(\2))/g' "$file" > "$tmp_file"
    sed -i '' -E 's/btcutil\.NewAddressPubKey\(([^,]+), ([^)]+)\)/btcutil.NewAddressPubKey(\1, convert.ParamsToBtc(\2))/g' "$tmp_file"
    sed -i '' -E 's/btcutil\.NewAddressScriptHash\(([^,]+), ([^)]+)\)/btcutil.NewAddressScriptHash(\1, convert.ParamsToBtc(\2))/g' "$tmp_file"
    sed -i '' -E 's/btcutil\.NewAddressScriptHashFromHash\(([^,]+), ([^)]+)\)/btcutil.NewAddressScriptHashFromHash(\1, convert.ParamsToBtc(\2))/g' "$tmp_file"
    sed -i '' -E 's/btcutil\.NewAddressWitnessPubKeyHash\(([^,]+), ([^)]+)\)/btcutil.NewAddressWitnessPubKeyHash(\1, convert.ParamsToBtc(\2))/g' "$tmp_file"
    sed -i '' -E 's/btcutil\.NewAddressWitnessScriptHash\(([^,]+), ([^)]+)\)/btcutil.NewAddressWitnessScriptHash(\1, convert.ParamsToBtc(\2))/g' "$tmp_file"
    sed -i '' -E 's/btcutil\.NewAddressTaproot\(([^,]+), ([^)]+)\)/btcutil.NewAddressTaproot(\1, convert.ParamsToBtc(\2))/g' "$tmp_file"
    
    mv "$tmp_file" "$file"
done

# Fix IsForNet calls
echo "Fixing IsForNet calls..."
find . -name "*.go" -type f | while read -r file; do
    if [[ "$file" == *"/vendor/"* ]] || [[ "$file" == *"/testdata/"* ]]; then
        continue
    fi
    
    if grep -q "\.IsForNet.*chaincfg.*Params" "$file"; then
        tmp_file="${file}.tmp"
        sed -E 's/\.IsForNet\(([^)]+)\)/.IsForNet(convert.ParamsToBtc(\1))/g' "$file" > "$tmp_file"
        mv "$tmp_file" "$file"
    fi
done

# Remove unused imports
echo "Removing unused imports..."
for file in "${files_needing_convert[@]}"; do
    if [ -f "$file" ]; then
        # Check if convert is actually used in the file
        if ! grep -q "convert\." "$file"; then
            # Remove the convert import if not used
            sed -i '' '/github.com\/toole-brendan\/shell\/internal\/convert/d' "$file"
        fi
    fi
done

# Clean up any double convert calls that might have been created
echo "Cleaning up double conversions..."
find . -name "*.go" -type f | while read -r file; do
    if [[ "$file" == *"/vendor/"* ]] || [[ "$file" == *"/testdata/"* ]]; then
        continue
    fi
    
    # Remove double conversions
    sed -i '' 's/convert\.HashToShell(convert\.HashToShell(/convert.HashToShell(/g' "$file"
    sed -i '' 's/convert\.ParamsToBtc(convert\.ParamsToBtc(/convert.ParamsToBtc(/g' "$file"
done

echo "Type conversion fixes complete!"
echo "Please run 'go build ./...' to check for remaining issues." 