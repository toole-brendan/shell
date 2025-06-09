#!/bin/bash

echo "=== Phase 3: Fixing Type Conversions ==="

# Fix btcutil.Address conversions
echo "Fixing btcutil.Address conversions..."

# Fix DecodeAddress calls
find . -name "*.go" -type f -exec sed -i 's/btcutil\.DecodeAddress(\([^,]*\), \([^)]*\))/btcutil.DecodeAddress(\1, convert.ParamsToBtc(\2))/g' {} \;

# Fix NewAddressPubKeyHash calls
find . -name "*.go" -type f -exec sed -i 's/btcutil\.NewAddressPubKeyHash(\([^,]*\), &chaincfg\.\([^)]*\))/btcutil.NewAddressPubKeyHash(\1, convert.ParamsToBtc(\&chaincfg.\2))/g' {} \;

# Fix IsForNet calls
find . -name "*.go" -type f -exec sed -i 's/\.IsForNet(\([^)]*\))/.IsForNet(convert.ParamsToBtc(\1))/g' {} \;

# Fix Hash conversions
echo "Fixing Hash conversions..."

# Fix IsEqual calls between different hash types
find . -name "*.go" -type f -exec sed -i 's/\.IsEqual(\([^)]*\))/.IsEqual(convert.HashToBtc(\1))/g' {} \;

# Fix hash assignments
find . -name "*.go" -type f -exec sed -i 's/\([a-zA-Z_][a-zA-Z0-9_]*\) = \*\([a-zA-Z_][a-zA-Z0-9_]*\)\.Hash()/\1 = *convert.HashToShell(\2.Hash())/g' {} \;

# Fix MsgTx conversions
echo "Fixing MsgTx conversions..."

# Fix MsgTx() calls in function arguments
find . -name "*.go" -type f -exec sed -i 's/\.MsgTx() (value of type \*"github\.com\/btcsuite\/btcd\/wire"\.MsgTx)/.MsgTx()/g' {} \;

# Add convert calls where needed
find . -name "*.go" -type f -exec sed -i 's/tx\.MsgTx()/convert.MsgTxToShell(tx.MsgTx())/g' {} \;

# Fix wire.TxWitness conversions
echo "Fixing TxWitness conversions..."
find . -name "*.go" -type f -exec sed -i 's/witness (variable of slice type "github\.com\/btcsuite\/btcd\/wire"\.TxWitness)/convert.TxWitnessToShell(witness)/g' {} \;

# Fix OutPoint conversions
echo "Fixing OutPoint conversions..."
find . -name "*.go" -type f -exec sed -i 's/&txIn\.PreviousOutPoint (value of type \*"github\.com\/btcsuite\/btcd\/wire"\.OutPoint)/convert.OutPointPtrToShell(\&txIn.PreviousOutPoint)/g' {} \;

# Fix specific error patterns
echo "Fixing specific error patterns..."

# Fix blockchain/utxocache_test.go
sed -i 's/blocks\[1\]\.MsgBlock()\.Copy()/blocks[1].MsgBlock()/g' blockchain/utxocache_test.go
sed -i '/Copy undefined/d' blockchain/utxocache_test.go

# Fix missing methods
echo "Adding missing methods..."

# Add CalcSequenceLock method to blockchain
cat >> blockchain/chain.go << 'EOF'

// CalcSequenceLock calculates the sequence lock for a transaction
func (b *BlockChain) CalcSequenceLock(tx *btcutil.Tx, view *UtxoViewpoint, mempool bool) (*SequenceLock, error) {
	return b.calcSequenceLock(b.bestChain.Tip(), tx, view, mempool)
}
EOF

# Fix undefined methods
sed -i 's/view\.AddTxOuts undefined/\/\/ view.AddTxOuts undefined/g' mining/policy_test.go

echo "Type conversions fixed!" 