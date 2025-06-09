#!/bin/bash

echo "=== Phase 4: Fixing Import Issues ==="

# Remove unused imports
echo "Removing unused imports..."

# Fix txscript/reference_test.go
sed -i '/"github.com\/toole-brendan\/shell\/internal\/convert"/d' txscript/reference_test.go

# Fix mempool/estimatefee_test.go
sed -i '/"github.com\/toole-brendan\/shell\/internal\/convert"/d' mempool/estimatefee_test.go

# Fix mining/policy_test.go
sed -i '/"github.com\/btcsuite\/btcd\/btcutil"/d' mining/policy_test.go

# Fix mining/randomx/miner.go
sed -i '/"github.com\/btcsuite\/btcd\/btcutil"/d' mining/randomx/miner.go

# Fix database/example_test.go
sed -i '/"github.com\/btcsuite\/btcd\/btcutil"/d' database/example_test.go

# Fix database/ffldb/bench_test.go
sed -i '/"github.com\/btcsuite\/btcd\/btcutil"/d' database/ffldb/bench_test.go

# Fix blockchain/example_test.go
sed -i '/"github.com\/btcsuite\/btcd\/btcutil"/d' blockchain/example_test.go

# Fix blockchain/bench_test.go
sed -i '/"github.com\/btcsuite\/btcd\/btcutil"/d' blockchain/bench_test.go

# Add missing imports where needed
echo "Adding missing imports..."

# Add convert import where it's used but not imported
for file in $(grep -l "convert\." --include="*.go" -r . | grep -v "internal/convert"); do
    if ! grep -q '"github.com/toole-brendan/shell/internal/convert"' "$file"; then
        sed -i '/^import (/a\\t"github.com/toole-brendan/shell/internal/convert"' "$file"
    fi
done

echo "Import issues fixed!" 