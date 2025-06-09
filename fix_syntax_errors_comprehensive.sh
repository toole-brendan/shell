#!/bin/bash

echo "=== Phase 1: Fixing Syntax Errors ==="

# Fix missing commas in composite literals
echo "Fixing missing commas in composite literals..."

# Fix in rpcwebsocket.go
sed -i 's/Hash:  \*convert\.HashToShell(tx\.Hash()))/Hash:  *convert.HashToShell(tx.Hash()),/g' rpcwebsocket.go

# Fix in mempool/mempool.go
sed -i 's/mp\.orphans\[convert\.HashToShell(tx\.Hash()))\]/mp.orphans[*convert.HashToShell(tx.Hash())]/g' mempool/mempool.go

# Fix in mempool/mempool_test.go
sed -i 's/prevOut := wire\.OutPoint{Hash: \*convert\.HashToShell(tx\.Hash()))/prevOut := wire.OutPoint{Hash: *convert.HashToShell(tx.Hash()),/g' mempool/mempool_test.go

# Fix in mining/mining.go
sed -i 's/deps := dependers\[convert\.HashToShell(tx\.Hash()))\]/deps := dependers[*convert.HashToShell(tx.Hash())]/g' mining/mining.go

# Fix in blockchain/validate.go
sed -i 's/prevOut := wire\.OutPoint{Hash: \*convert\.HashToShell(tx\.Hash()))/prevOut := wire.OutPoint{Hash: *convert.HashToShell(tx.Hash()),/g' blockchain/validate.go

# Fix in blockchain/utxoviewpoint.go
sed -i 's/prevOut := wire\.OutPoint{Hash: \*convert\.HashToShell(tx\.Hash())), Index: txOutIdx}/prevOut := wire.OutPoint{Hash: *convert.HashToShell(tx.Hash()), Index: txOutIdx}/g' blockchain/utxoviewpoint.go

# Fix in blockchain/utxocache.go
sed -i 's/prevOut := wire\.OutPoint{Hash: \*convert\.HashToShell(tx\.Hash()))/prevOut := wire.OutPoint{Hash: *convert.HashToShell(tx.Hash()),/g' blockchain/utxocache.go

# Fix in blockchain/chain.go
sed -i 's/b\.orphans\[convert\.HashToShell(block\.Hash()))\]/b.orphans[*convert.HashToShell(block.Hash())]/g' blockchain/chain.go

# Fix in blockchain/chain_test.go
sed -i 's/Hash:  \*convert\.HashToShell(targetTx\.Hash())),/Hash:  *convert.HashToShell(targetTx.Hash()),/g' blockchain/chain_test.go

# Fix in database/ffldb/driver_test.go
sed -i 's/blockHashMap\[convert\.HashToShell(block\.Hash()))\]/blockHashMap[*convert.HashToShell(block.Hash())]/g' database/ffldb/driver_test.go

# Fix in blockchain/indexers/addrindex.go
sed -i 's/addrIndexEntry\[convert\.HashToShell(tx\.Hash()))\]/addrIndexEntry[*convert.HashToShell(tx.Hash())]/g' blockchain/indexers/addrindex.go

echo "Syntax errors fixed!" 