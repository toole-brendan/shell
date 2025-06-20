// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"compress/bzip2"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/toole-brendan/shell/blockchain/internal/testhelper"
	"github.com/toole-brendan/shell/chaincfg"
	"github.com/toole-brendan/shell/chaincfg/chainhash"
	"github.com/toole-brendan/shell/database"
	_ "github.com/toole-brendan/shell/database/ffldb"
	"github.com/toole-brendan/shell/internal/convert"
	"github.com/toole-brendan/shell/txscript"
	"github.com/toole-brendan/shell/wire"
)

const (
	// testDbType is the database backend type to use for the tests.
	testDbType = "ffldb"

	// testDbRoot is the root directory used to create all test databases.
	testDbRoot = "testdbs"

	// blockDataNet is the expected network in the test block data.
	blockDataNet = wire.MainNet
)

// filesExists returns whether or not the named file or directory exists.
func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// isSupportedDbType returns whether or not the passed database type is
// currently supported.
func isSupportedDbType(dbType string) bool {
	supportedDrivers := database.SupportedDrivers()
	for _, driver := range supportedDrivers {
		if dbType == driver {
			return true
		}
	}

	return false
}

// loadBlocks reads files containing bitcoin block data (gzipped but otherwise
// in the format bitcoind writes) from disk and returns them as an array of
// btcutil.Block.  This is largely borrowed from the test code in btcdb.
func loadBlocks(filename string) (blocks []*btcutil.Block, err error) {
	filename = filepath.Join("testdata/", filename)

	var network = wire.MainNet
	var dr io.Reader
	var fi io.ReadCloser

	fi, err = os.Open(filename)
	if err != nil {
		return
	}

	if strings.HasSuffix(filename, ".bz2") {
		dr = bzip2.NewReader(fi)
	} else {
		dr = fi
	}
	defer fi.Close()

	var block *btcutil.Block

	err = nil
	for height := int64(1); err == nil; height++ {
		var rintbuf uint32
		err = binary.Read(dr, binary.LittleEndian, &rintbuf)
		if err == io.EOF {
			// hit end of file at expected offset: no warning
			height--
			err = nil
			break
		}
		if err != nil {
			break
		}
		if rintbuf != uint32(network) {
			break
		}
		err = binary.Read(dr, binary.LittleEndian, &rintbuf)
		blocklen := rintbuf

		rbytes := make([]byte, blocklen)

		// read block
		dr.Read(rbytes)

		block, err = btcutil.NewBlockFromBytes(rbytes)
		if err != nil {
			return
		}
		blocks = append(blocks, block)
	}

	return
}

// chainSetup is used to create a new db and chain instance with the genesis
// block already inserted.  In addition to the new chain instance, it returns
// a teardown function the caller should invoke when done testing to clean up.
func chainSetup(dbName string, params *chaincfg.Params) (*BlockChain, func(), error) {
	if !isSupportedDbType(testDbType) {
		return nil, nil, fmt.Errorf("unsupported db type %v", testDbType)
	}

	// Handle memory database specially since it doesn't need the disk
	// specific handling.
	var db database.DB
	var teardown func()
	if testDbType == "memdb" {
		ndb, err := database.Create(testDbType)
		if err != nil {
			return nil, nil, fmt.Errorf("error creating db: %v", err)
		}
		db = ndb

		// Setup a teardown function for cleaning up.  This function is
		// returned to the caller to be invoked when it is done testing.
		teardown = func() {
			db.Close()
		}
	} else {
		// Create the root directory for test databases.
		if !fileExists(testDbRoot) {
			if err := os.MkdirAll(testDbRoot, 0700); err != nil {
				err := fmt.Errorf("unable to create test db "+
					"root: %v", err)
				return nil, nil, err
			}
		}

		// Create a new database to store the accepted blocks into.
		dbPath := filepath.Join(testDbRoot, dbName)
		_ = os.RemoveAll(dbPath)
		ndb, err := database.Create(testDbType, dbPath, blockDataNet)
		if err != nil {
			return nil, nil, fmt.Errorf("error creating db: %v", err)
		}
		db = ndb

		// Setup a teardown function for cleaning up.  This function is
		// returned to the caller to be invoked when it is done testing.
		teardown = func() {
			db.Close()
			os.RemoveAll(dbPath)
			os.RemoveAll(testDbRoot)
		}
	}

	// Copy the chain params to ensure any modifications the tests do to
	// the chain parameters do not affect the global instance.
	paramsCopy := *params

	// Create the main chain instance.
	chain, err := New(&Config{
		DB:          db,
		ChainParams: &paramsCopy,
		Checkpoints: nil,
		TimeSource:  NewMedianTime(),
		SigCache:    txscript.NewSigCache(1000),
	})
	if err != nil {
		teardown()
		err := fmt.Errorf("failed to create chain instance: %v", err)
		return nil, nil, err
	}
	return chain, teardown, nil
}

// loadUtxoView returns a utxo view loaded from a file.
func loadUtxoView(filename string) (*UtxoViewpoint, error) {
	// The utxostore file format is:
	// <tx hash><output index><serialized utxo len><serialized utxo>
	//
	// The output index and serialized utxo len are little endian uint32s
	// and the serialized utxo uses the format described in chainio.go.

	filename = filepath.Join("testdata", filename)
	fi, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	// Choose read based on whether the file is compressed or not.
	var r io.Reader
	if strings.HasSuffix(filename, ".bz2") {
		r = bzip2.NewReader(fi)
	} else {
		r = fi
	}
	defer fi.Close()

	view := NewUtxoViewpoint()
	for {
		// Hash of the utxo entry.
		var hash chainhash.Hash
		_, err := io.ReadAtLeast(r, hash[:], len(hash[:]))
		if err != nil {
			// Expected EOF at the right offset.
			if err == io.EOF {
				break
			}
			return nil, err
		}

		// Output index of the utxo entry.
		var index uint32
		err = binary.Read(r, binary.LittleEndian, &index)
		if err != nil {
			return nil, err
		}

		// Num of serialized utxo entry bytes.
		var numBytes uint32
		err = binary.Read(r, binary.LittleEndian, &numBytes)
		if err != nil {
			return nil, err
		}

		// Serialized utxo entry.
		serialized := make([]byte, numBytes)
		_, err = io.ReadAtLeast(r, serialized, int(numBytes))
		if err != nil {
			return nil, err
		}

		// Deserialize it and add it to the view.
		entry, err := deserializeUtxoEntry(serialized)
		if err != nil {
			return nil, err
		}
		view.Entries()[wire.OutPoint{Hash: hash, Index: index}] = entry
	}

	return view, nil
}

// convertUtxoStore reads a utxostore from the legacy format and writes it back
// out using the latest format.  It is only useful for converting utxostore data
// used in the tests, which has already been done.  However, the code is left
// available for future reference.
func convertUtxoStore(r io.Reader, w io.Writer) error {
	// The old utxostore file format was:
	// <tx hash><serialized utxo len><serialized utxo>
	//
	// The serialized utxo len was a little endian uint32 and the serialized
	// utxo uses the format described in upgrade.go.

	littleEndian := binary.LittleEndian
	for {
		// Hash of the utxo entry.
		var hash chainhash.Hash
		_, err := io.ReadAtLeast(r, hash[:], len(hash[:]))
		if err != nil {
			// Expected EOF at the right offset.
			if err == io.EOF {
				break
			}
			return err
		}

		// Num of serialized utxo entry bytes.
		var numBytes uint32
		err = binary.Read(r, littleEndian, &numBytes)
		if err != nil {
			return err
		}

		// Serialized utxo entry.
		serialized := make([]byte, numBytes)
		_, err = io.ReadAtLeast(r, serialized, int(numBytes))
		if err != nil {
			return err
		}

		// Deserialize the entry.
		entries, err := deserializeUtxoEntryV0(serialized)
		if err != nil {
			return err
		}

		// Loop through all of the utxos and write them out in the new
		// format.
		for outputIdx, entry := range entries {
			// Reserialize the entries using the new format.
			serialized, err := serializeUtxoEntry(entry)
			if err != nil {
				return err
			}

			// Write the hash of the utxo entry.
			_, err = w.Write(hash[:])
			if err != nil {
				return err
			}

			// Write the output index of the utxo entry.
			err = binary.Write(w, littleEndian, outputIdx)
			if err != nil {
				return err
			}

			// Write num of serialized utxo entry bytes.
			err = binary.Write(w, littleEndian, uint32(len(serialized)))
			if err != nil {
				return err
			}

			// Write the serialized utxo.
			_, err = w.Write(serialized)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// TstSetCoinbaseMaturity makes the ability to set the coinbase maturity
// available when running tests.
func (b *BlockChain) TstSetCoinbaseMaturity(maturity uint16) {
	b.chainParams.CoinbaseMaturity = maturity
}

// newFakeChain returns a chain that is usable for synthetic tests.  It is
// important to note that this chain has no database associated with it, so
// it is not usable with all functions and the tests must take care when making
// use of it.
func newFakeChain(params *chaincfg.Params) *BlockChain {
	// Create a genesis block node and block index index populated with it
	// for use when creating the fake chain below.
	node := newBlockNode(&params.GenesisBlock.Header, nil)
	index := newBlockIndex(nil, params)
	index.AddNode(node)

	targetTimespan := int64(params.TargetTimespan / time.Second)
	targetTimePerBlock := int64(params.TargetTimePerBlock / time.Second)
	adjustmentFactor := params.RetargetAdjustmentFactor
	b := &BlockChain{
		chainParams:         params,
		timeSource:          NewMedianTime(),
		minRetargetTimespan: targetTimespan / adjustmentFactor,
		maxRetargetTimespan: targetTimespan * adjustmentFactor,
		blocksPerRetarget:   int32(targetTimespan / targetTimePerBlock),
		index:               index,
		bestChain:           newChainView(node),
		warningCaches:       newThresholdCaches(vbNumBits),
		deploymentCaches:    newThresholdCaches(chaincfg.DefinedDeployments),
	}

	for _, deployment := range params.Deployments {
		deploymentStarter := deployment.DeploymentStarter
		if clockStarter, ok := deploymentStarter.(chaincfg.ClockConsensusDeploymentStarter); ok {
			clockStarter.SynchronizeClock(b)
		}

		deploymentEnder := deployment.DeploymentEnder
		if clockEnder, ok := deploymentEnder.(chaincfg.ClockConsensusDeploymentEnder); ok {
			clockEnder.SynchronizeClock(b)
		}
	}

	return b
}

// newFakeNode creates a block node connected to the passed parent with the
// provided fields populated and fake values for the other fields.
func newFakeNode(parent *blockNode, blockVersion int32, bits uint32, timestamp time.Time) *blockNode {
	// Make up a header and create a block node from it.
	header := &wire.BlockHeader{
		Version:   blockVersion,
		PrevBlock: parent.hash,
		Bits:      bits,
		Timestamp: timestamp,
	}
	return newBlockNode(header, parent)
}

// addBlock adds a block to the blockchain that succeeds the previous block.
// The blocks spends all the provided spendable outputs.  The new block and
// the new spendable outputs created in the block are returned.
func addBlock(chain *BlockChain, prev *btcutil.Block, spends []*testhelper.SpendableOut) (
	*btcutil.Block, []*testhelper.SpendableOut, error) {

	block, outs, err := newBlock(chain, prev, spends)
	if err != nil {
		return nil, nil, err
	}

	_, _, err = chain.ProcessBlock(block, BFNone)
	if err != nil {
		return nil, nil, err
	}

	return block, outs, nil
}

// calcMerkleRoot creates a merkle tree from the slice of transactions and
// returns the root of the tree.
func calcMerkleRoot(txns []*wire.MsgTx) chainhash.Hash {
	if len(txns) == 0 {
		return chainhash.Hash{}
	}

	utilTxns := make([]*btcutil.Tx, 0, len(txns))
	for _, tx := range txns {
		utilTxns = append(utilTxns, convert.NewShellTx(tx))
	}
	return CalcMerkleRoot(utilTxns, false)
}

// newBlock creates a block to the blockchain that succeeds the previous block.
// The blocks spends all the provided spendable outputs.  The new block and the
// newly spendable outputs created in the block are returned.
func newBlock(chain *BlockChain, prev *btcutil.Block,
	spends []*testhelper.SpendableOut) (*btcutil.Block, []*testhelper.SpendableOut, error) {

	blockHeight := prev.Height() + 1
	txns := make([]*wire.MsgTx, 0, 1+len(spends))

	// Create and add coinbase tx.
	cb := testhelper.CreateCoinbaseTx(blockHeight, CalcBlockSubsidy(blockHeight, chain.chainParams))
	txns = append(txns, cb)

	// Spend all txs to be spent.
	for _, spend := range spends {
		cb.TxOut[0].Value += int64(testhelper.LowFee)

		spendTx := testhelper.CreateSpendTx(spend, testhelper.LowFee)
		txns = append(txns, spendTx)
	}

	// Use a timestamp that is one second after the previous block unless
	// this is the first block in which case the current time is used.
	var ts time.Time
	if blockHeight == 1 {
		ts = time.Unix(time.Now().Unix(), 0)
	} else {
		ts = prev.MsgBlock().Header.Timestamp.Add(time.Second)
	}

	var prevBlockHash chainhash.Hash
	copy(prevBlockHash[:], prev.Hash()[:])

	// Create the block. The nonce will be solved in the below code in
	// SolveBlock.
	msgBlock := &wire.MsgBlock{
		Header: wire.BlockHeader{
			Version:    1,
			PrevBlock:  prevBlockHash,
			MerkleRoot: calcMerkleRoot(txns),
			Bits:       chain.chainParams.PowLimitBits,
			Timestamp:  ts,
			Nonce:      0, // To be solved.
		},
		Transactions: txns,
	}

	// Solve the block.
	if !testhelper.SolveBlock(&msgBlock.Header) {
		return nil, nil, fmt.Errorf("Unable to solve block at height %d", blockHeight)
	}

	block := convert.NewShellBlock(msgBlock)
	block.SetHeight(blockHeight)

	// Create spendable outs to return.
	outs := make([]*testhelper.SpendableOut, len(txns))
	for i, tx := range txns {
		out := testhelper.MakeSpendableOutForTx(tx, 0)
		outs[i] = &out
	}

	return block, outs, nil
}
