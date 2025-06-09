#!/bin/bash

# Script to analyze type errors and categorize them

echo "Analyzing type errors..."

# Create output directory
mkdir -p type_error_analysis

# Extract unique error patterns
echo "=== Unique Error Patterns ===" > type_error_analysis/patterns.txt

# Common patterns we're seeing:
# 1. btcutil.NewBlock/NewTx conversions
# 2. Hash type mismatches
# 3. OutPoint type mismatches
# 4. TxOut/TxIn type mismatches
# 5. Interface implementation mismatches

# Count occurrences of each pattern
echo "Pattern counts:" > type_error_analysis/counts.txt
echo "" >> type_error_analysis/counts.txt

# Pattern 1: btcutil.NewBlock
echo -n "btcutil.NewBlock type errors: " >> type_error_analysis/counts.txt
grep -r "cannot use .* as \*\"github.com/btcsuite/btcd/wire\".MsgBlock value in argument to btcutil.NewBlock" . --include="*.go" 2>/dev/null | wc -l >> type_error_analysis/counts.txt

# Pattern 2: btcutil.NewTx
echo -n "btcutil.NewTx type errors: " >> type_error_analysis/counts.txt
grep -r "cannot use .* as \*\"github.com/btcsuite/btcd/wire\".MsgTx value in argument to btcutil.NewTx" . --include="*.go" 2>/dev/null | wc -l >> type_error_analysis/counts.txt

# Pattern 3: Hash type mismatches
echo -n "Hash type mismatches: " >> type_error_analysis/counts.txt
grep -r "cannot use .* as \*\"github.com/toole-brendan/shell/chaincfg/chainhash\".Hash value" . --include="*.go" 2>/dev/null | wc -l >> type_error_analysis/counts.txt

# Pattern 4: OutPoint type mismatches
echo -n "OutPoint type mismatches: " >> type_error_analysis/counts.txt
grep -r "cannot use .* as \"github.com/toole-brendan/shell/wire\".OutPoint value" . --include="*.go" 2>/dev/null | wc -l >> type_error_analysis/counts.txt

# Pattern 5: TxOut type mismatches
echo -n "TxOut type mismatches: " >> type_error_analysis/counts.txt
grep -r "cannot use .* as \*\"github.com/toole-brendan/shell/wire\".TxOut value" . --include="*.go" 2>/dev/null | wc -l >> type_error_analysis/counts.txt

# Pattern 6: Interface implementation issues
echo -n "Interface implementation issues: " >> type_error_analysis/counts.txt
grep -r "does not implement.*wrong type for method" . --include="*.go" 2>/dev/null | wc -l >> type_error_analysis/counts.txt

echo ""
echo "Analysis complete. Results saved to type_error_analysis/"
cat type_error_analysis/counts.txt 