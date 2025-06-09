#!/bin/bash

# Script to fix remaining issues in the shell btcd fork

echo "Fixing remaining issues..."

# Fix 1: Fix btcutil.NewBlockFromBlockAndBytes calls
echo "Fixing btcutil.NewBlockFromBlockAndBytes calls..."
find . -name "*.go" -type f | while read file; do
    if grep -q 'btcutil\.NewBlockFromBlockAndBytes' "$file"; then
        echo "Fixing NewBlockFromBlockAndBytes in $file"
        # Need to convert msg to btcsuite type first
        sed -i.bak 's/block := btcutil\.NewBlockFromBlockAndBytes(msg, buf)/block := convert.NewShellBlock(msg)/g' "$file"
    fi
done

# Fix 2: Fix filter.Reload calls
echo "Fixing filter.Reload calls..."
find . -name "*.go" -type f | while read file; do
    if grep -q 'sp\.filter\.Reload(msg)' "$file"; then
        echo "Fixing filter.Reload in $file"
        # Need to convert msg to btcsuite type
        sed -i.bak 's/sp\.filter\.Reload(msg)/sp.filter.Reload(convert.MsgFilterLoadToBtc(msg))/g' "$file"
    fi
done

# Fix 3: Add missing convert functions to convert.go
echo "Adding missing conversion functions..."
cat >> internal/convert/convert.go << 'EOF'

// MsgFilterLoadToShell converts a btcsuite wire.MsgFilterLoad to shell wire.MsgFilterLoad
func MsgFilterLoadToShell(msg *btcwire.MsgFilterLoad) *shellwire.MsgFilterLoad {
	if msg == nil {
		return nil
	}
	return &shellwire.MsgFilterLoad{
		Filter:    msg.Filter,
		HashFuncs: msg.HashFuncs,
		Tweak:     msg.Tweak,
		Flags:     msg.Flags,
	}
}

// MsgFilterLoadToBtc converts a shell wire.MsgFilterLoad to btcsuite wire.MsgFilterLoad
func MsgFilterLoadToBtc(msg *shellwire.MsgFilterLoad) *btcwire.MsgFilterLoad {
	if msg == nil {
		return nil
	}
	return &btcwire.MsgFilterLoad{
		Filter:    msg.Filter,
		HashFuncs: msg.HashFuncs,
		Tweak:     msg.Tweak,
		Flags:     msg.Flags,
	}
}

// MsgMerkleBlockToShell converts a btcsuite wire.MsgMerkleBlock to shell wire.MsgMerkleBlock
func MsgMerkleBlockToShell(msg *btcwire.MsgMerkleBlock) *shellwire.MsgMerkleBlock {
	if msg == nil {
		return nil
	}
	shellMsg := &shellwire.MsgMerkleBlock{
		Header:       *BlockHeaderToShell(&msg.Header),
		Transactions: msg.Transactions,
		Hashes:       make([]*shellchainhash.Hash, len(msg.Hashes)),
		Flags:        msg.Flags,
	}
	for i, hash := range msg.Hashes {
		shellMsg.Hashes[i] = HashToShell(hash)
	}
	return shellMsg
}

// MsgMerkleBlockToBtc converts a shell wire.MsgMerkleBlock to btcsuite wire.MsgMerkleBlock
func MsgMerkleBlockToBtc(msg *shellwire.MsgMerkleBlock) *btcwire.MsgMerkleBlock {
	if msg == nil {
		return nil
	}
	btcMsg := &btcwire.MsgMerkleBlock{
		Header:       *BlockHeaderToBtc(&msg.Header),
		Transactions: msg.Transactions,
		Hashes:       make([]*btcchainhash.Hash, len(msg.Hashes)),
		Flags:        msg.Flags,
	}
	for i, hash := range msg.Hashes {
		btcMsg.Hashes[i] = HashToBtc(hash)
	}
	return btcMsg
}

// BlockHeaderToBtc converts a shell wire.BlockHeader to btcsuite wire.BlockHeader
func BlockHeaderToBtc(header *shellwire.BlockHeader) *btcwire.BlockHeader {
	if header == nil {
		return nil
	}
	return &btcwire.BlockHeader{
		Version:    header.Version,
		PrevBlock:  *HashToBtc(&header.PrevBlock),
		MerkleRoot: *HashToBtc(&header.MerkleRoot),
		Timestamp:  header.Timestamp,
		Bits:       header.Bits,
		Nonce:      header.Nonce,
	}
}
EOF

# Fix 4: Fix undefined types in netsync/manager.go
echo "Fixing undefined types in netsync/manager.go..."
if [ -f "./netsync/manager.go" ]; then
    # Replace mempool.TxPool with *mempool.TxPool
    sed -i.bak 's/txMemPool\s*mempool\.TxPool/txMemPool *mempool.TxPool/g' ./netsync/manager.go
    
    # Add mempool.Tag type definition if needed
    if ! grep -q "type Tag " ./mempool/mempool.go; then
        echo "Adding Tag type to mempool package..."
        sed -i.bak '/^package mempool/a\
\
// Tag represents a tag for a transaction\
type Tag uint64' ./mempool/mempool.go
    fi
fi

# Fix 5: Fix indexers.NewAddrIndex issue
echo "Checking for NewAddrIndex function..."
if ! grep -q "func NewAddrIndex" ./blockchain/indexers/addrindex.go; then
    echo "Note: NewAddrIndex function may be missing from indexers package"
fi

# Fix 6: Fix mempool.New issue
echo "Checking for mempool.New function..."
if ! grep -q "func New" ./mempool/mempool.go; then
    echo "Note: mempool.New function may be missing, use NewTxPool instead"
    find . -name "*.go" -type f | while read file; do
        if grep -q 'mempool\.New(' "$file"; then
            sed -i.bak 's/mempool\.New(/mempool.NewTxPool(/g' "$file"
        fi
    done
fi

# Clean up backup files
find . -name "*.go.bak" -type f -delete

echo "Remaining issues fixed!"
echo ""
echo "Next steps:"
echo "1. Run 'go mod tidy' to update dependencies"
echo "2. Check for any remaining linter errors with 'go vet ./...'"
echo "3. Some issues may still require manual intervention" 