// Package convert provides utilities for converting between btcsuite and shell types.
package convert

import (
	"github.com/btcsuite/btcd/btcutil"
	btcchaincfg "github.com/btcsuite/btcd/chaincfg"
	btcchainhash "github.com/btcsuite/btcd/chaincfg/chainhash"
	btcwire "github.com/btcsuite/btcd/wire"

	shellchainhash "github.com/toole-brendan/shell/chaincfg/chainhash"
	shellwire "github.com/toole-brendan/shell/wire"
	wire "github.com/toole-brendan/shell/wire"
)

// HashToShell converts a btcsuite chainhash.Hash to a shell chainhash.Hash
func HashToShell(hash *btcchainhash.Hash) *shellchainhash.Hash {
	if hash == nil {
		return nil
	}
	var shellHash shellchainhash.Hash
	copy(shellHash[:], hash[:])
	return &shellHash
}

// HashToBtc converts a shell chainhash.Hash to a btcsuite chainhash.Hash
func HashToBtc(hash *shellchainhash.Hash) *btcchainhash.Hash {
	if hash == nil {
		return nil
	}
	var btcHash btcchainhash.Hash
	copy(btcHash[:], hash[:])
	return &btcHash
}

// OutPointToShell converts a btcsuite wire.OutPoint to a shell wire.OutPoint
func OutPointToShell(op btcwire.OutPoint) shellwire.OutPoint {
	return shellwire.OutPoint{
		Hash:  *HashToShell(&op.Hash),
		Index: op.Index,
	}
}

// OutPointPtrToShell converts a btcsuite wire.OutPoint pointer to a shell wire.OutPoint
func OutPointPtrToShell(op *btcwire.OutPoint) shellwire.OutPoint {
	if op == nil {
		return shellwire.OutPoint{}
	}
	return OutPointToShell(*op)
}

// OutPointToBtc converts a shell wire.OutPoint to a btcsuite wire.OutPoint
func OutPointToBtc(op shellwire.OutPoint) btcwire.OutPoint {
	return btcwire.OutPoint{
		Hash:  *HashToBtc(&op.Hash),
		Index: op.Index,
	}
}

// OutPointPtrToBtc converts a shell wire.OutPoint pointer to a btcsuite wire.OutPoint
func OutPointPtrToBtc(op *shellwire.OutPoint) btcwire.OutPoint {
	if op == nil {
		return btcwire.OutPoint{}
	}
	return OutPointToBtc(*op)
}

// ParamsToBtc converts a shell network name to btcsuite chaincfg.Params
func ParamsToBtc(networkName string) *btcchaincfg.Params {
	if networkName == "" {
		return nil
	}
	// Map common network parameters
	switch networkName {
	case "mainnet":
		return &btcchaincfg.MainNetParams
	case "testnet3":
		return &btcchaincfg.TestNet3Params
	case "regtest":
		return &btcchaincfg.RegressionNetParams
	case "simnet":
		return &btcchaincfg.SimNetParams
	case "signet":
		return &btcchaincfg.SigNetParams
	default:
		// For custom networks, return mainnet as fallback
		// In production, you might want to create a proper mapping
		return &btcchaincfg.MainNetParams
	}
}

// TxLocToShell converts a btcsuite wire.TxLoc to a shell wire.TxLoc
func TxLocToShell(loc btcwire.TxLoc) shellwire.TxLoc {
	return shellwire.TxLoc{
		TxStart: loc.TxStart,
		TxLen:   loc.TxLen,
	}
}

// TxLocToBtc converts a shell wire.TxLoc to a btcsuite wire.TxLoc
func TxLocToBtc(loc shellwire.TxLoc) btcwire.TxLoc {
	return btcwire.TxLoc{
		TxStart: loc.TxStart,
		TxLen:   loc.TxLen,
	}
}

// TxLocsToShell converts a slice of btcsuite wire.TxLoc to shell wire.TxLoc
func TxLocsToShell(locs []btcwire.TxLoc) []shellwire.TxLoc {
	if locs == nil {
		return nil
	}
	shellLocs := make([]shellwire.TxLoc, len(locs))
	for i, loc := range locs {
		shellLocs[i] = TxLocToShell(loc)
	}
	return shellLocs
}

// TxLocsToBtc converts a slice of shell wire.TxLoc to btcsuite wire.TxLoc
func TxLocsToBtc(locs []shellwire.TxLoc) []btcwire.TxLoc {
	if locs == nil {
		return nil
	}
	btcLocs := make([]btcwire.TxLoc, len(locs))
	for i, loc := range locs {
		btcLocs[i] = TxLocToBtc(loc)
	}
	return btcLocs
}

// TxOutToShell converts a btcsuite wire.TxOut to a shell wire.TxOut
func TxOutToShell(txOut *btcwire.TxOut) *shellwire.TxOut {
	if txOut == nil {
		return nil
	}
	return &shellwire.TxOut{
		Value:    txOut.Value,
		PkScript: txOut.PkScript,
	}
}

// TxOutToBtc converts a shell wire.TxOut to a btcsuite wire.TxOut
func TxOutToBtc(txOut *shellwire.TxOut) *btcwire.TxOut {
	if txOut == nil {
		return nil
	}
	return &btcwire.TxOut{
		Value:    txOut.Value,
		PkScript: txOut.PkScript,
	}
}

// MsgBlockToShell converts a btcsuite wire.MsgBlock to a shell wire.MsgBlock
func MsgBlockToShell(block *btcwire.MsgBlock) *shellwire.MsgBlock {
	if block == nil {
		return nil
	}

	shellBlock := &shellwire.MsgBlock{
		Header: shellwire.BlockHeader{
			Version:    block.Header.Version,
			PrevBlock:  *HashToShell(&block.Header.PrevBlock),
			MerkleRoot: *HashToShell(&block.Header.MerkleRoot),
			Timestamp:  block.Header.Timestamp,
			Bits:       block.Header.Bits,
			Nonce:      block.Header.Nonce,
		},
		Transactions: make([]*shellwire.MsgTx, len(block.Transactions)),
	}

	for i, tx := range block.Transactions {
		shellBlock.Transactions[i] = MsgTxToShell(tx)
	}

	return shellBlock
}

// MsgTxToShell converts a btcsuite wire.MsgTx to a shell wire.MsgTx
func MsgTxToShell(tx *btcwire.MsgTx) *shellwire.MsgTx {
	if tx == nil {
		return nil
	}

	shellTx := &shellwire.MsgTx{
		Version:  tx.Version,
		TxIn:     make([]*shellwire.TxIn, len(tx.TxIn)),
		TxOut:    make([]*shellwire.TxOut, len(tx.TxOut)),
		LockTime: tx.LockTime,
	}

	for i, txIn := range tx.TxIn {
		shellTx.TxIn[i] = &shellwire.TxIn{
			PreviousOutPoint: OutPointToShell(txIn.PreviousOutPoint),
			SignatureScript:  txIn.SignatureScript,
			Witness:          TxWitnessToShell(txIn.Witness),
			Sequence:         txIn.Sequence,
		}
	}

	for i, txOut := range tx.TxOut {
		shellTx.TxOut[i] = TxOutToShell(txOut)
	}

	return shellTx
}

// TxWitnessToShell converts a btcsuite wire.TxWitness to a shell wire.TxWitness
func TxWitnessToShell(witness btcwire.TxWitness) shellwire.TxWitness {
	if witness == nil {
		return nil
	}
	shellWitness := make(shellwire.TxWitness, len(witness))
	copy(shellWitness, witness)
	return shellWitness
}

// BlockHeaderToShell converts a btcsuite wire.BlockHeader to a shell wire.BlockHeader
func BlockHeaderToShell(header *btcwire.BlockHeader) *shellwire.BlockHeader {
	if header == nil {
		return nil
	}
	return &shellwire.BlockHeader{
		Version:    header.Version,
		PrevBlock:  *HashToShell(&header.PrevBlock),
		MerkleRoot: *HashToShell(&header.MerkleRoot),
		Timestamp:  header.Timestamp,
		Bits:       header.Bits,
		Nonce:      header.Nonce,
	}
}

// NewShellBlock creates a new shell btcutil.Block from a shell wire.MsgBlock
func NewShellBlock(msgBlock *shellwire.MsgBlock) *btcutil.Block {
	// Convert shell MsgBlock to btcsuite MsgBlock
	btcMsgBlock := &btcwire.MsgBlock{
		Header: btcwire.BlockHeader{
			Version:    msgBlock.Header.Version,
			PrevBlock:  *HashToBtc(&msgBlock.Header.PrevBlock),
			MerkleRoot: *HashToBtc(&msgBlock.Header.MerkleRoot),
			Timestamp:  msgBlock.Header.Timestamp,
			Bits:       msgBlock.Header.Bits,
			Nonce:      msgBlock.Header.Nonce,
		},
		Transactions: make([]*btcwire.MsgTx, len(msgBlock.Transactions)),
	}

	for i, tx := range msgBlock.Transactions {
		btcMsgBlock.Transactions[i] = MsgTxToBtc(tx)
	}

	return btcutil.NewBlock(btcMsgBlock)
}

// MsgTxToBtc converts a shell wire.MsgTx to a btcsuite wire.MsgTx
func MsgTxToBtc(tx *shellwire.MsgTx) *btcwire.MsgTx {
	if tx == nil {
		return nil
	}

	btcTx := &btcwire.MsgTx{
		Version:  tx.Version,
		TxIn:     make([]*btcwire.TxIn, len(tx.TxIn)),
		TxOut:    make([]*btcwire.TxOut, len(tx.TxOut)),
		LockTime: tx.LockTime,
	}

	for i, txIn := range tx.TxIn {
		btcTx.TxIn[i] = &btcwire.TxIn{
			PreviousOutPoint: OutPointToBtc(txIn.PreviousOutPoint),
			SignatureScript:  txIn.SignatureScript,
			Witness:          TxWitnessToBtc(txIn.Witness),
			Sequence:         txIn.Sequence,
		}
	}

	for i, txOut := range tx.TxOut {
		btcTx.TxOut[i] = TxOutToBtc(txOut)
	}

	return btcTx
}

// TxWitnessToBtc converts a shell wire.TxWitness to a btcsuite wire.TxWitness
func TxWitnessToBtc(witness shellwire.TxWitness) btcwire.TxWitness {
	if witness == nil {
		return nil
	}
	btcWitness := make(btcwire.TxWitness, len(witness))
	copy(btcWitness, witness)
	return btcWitness
}

// NewShellTx creates a new shell btcutil.Tx from a shell wire.MsgTx
func NewShellTx(msgTx *shellwire.MsgTx) *btcutil.Tx {
	// Convert shell MsgTx to btcsuite MsgTx
	btcMsgTx := MsgTxToBtc(msgTx)
	return btcutil.NewTx(btcMsgTx)
}

// MsgFilterLoadToShell converts a btcsuite wire.MsgFilterLoad to shell wire.MsgFilterLoad
func MsgFilterLoadToShell(msg *btcwire.MsgFilterLoad) *shellwire.MsgFilterLoad {
	if msg == nil {
		return nil
	}
	return &shellwire.MsgFilterLoad{
		Filter:    msg.Filter,
		HashFuncs: msg.HashFuncs,
		Tweak:     msg.Tweak,
		Flags:     shellwire.BloomUpdateType(msg.Flags),
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
		Flags:     btcwire.BloomUpdateType(msg.Flags),
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

func ToShellBlockHeader(header *btcwire.BlockHeader) *wire.BlockHeader {
	if header == nil {
		return nil
	}

	return &wire.BlockHeader{
		Version:    header.Version,
		PrevBlock:  *HashToShell(&header.PrevBlock),
		MerkleRoot: *HashToShell(&header.MerkleRoot),
		Timestamp:  header.Timestamp,
		Bits:       header.Bits,
		Nonce:      header.Nonce,
	}
}

func ToShellOutPoint(out *btcwire.OutPoint) *wire.OutPoint {
	if out == nil {
		return nil
	}

	return &wire.OutPoint{
		Hash:  *HashToShell(&out.Hash),
		Index: out.Index,
	}
}

func ToShellTxIn(txIn *btcwire.TxIn) *wire.TxIn {
	if txIn == nil {
		return nil
	}

	witness := make([][]byte, len(txIn.Witness))
	for i, w := range txIn.Witness {
		witness[i] = make([]byte, len(w))
		copy(witness[i], w)
	}

	return &wire.TxIn{
		PreviousOutPoint: *ToShellOutPoint(&txIn.PreviousOutPoint),
		SignatureScript:  txIn.SignatureScript,
		Witness:          witness,
		Sequence:         txIn.Sequence,
	}
}

func ToShellTxOut(txOut *btcwire.TxOut) *wire.TxOut {
	if txOut == nil {
		return nil
	}

	return &wire.TxOut{
		Value:    txOut.Value,
		PkScript: txOut.PkScript,
	}
}

func ToShellMsgTx(tx *btcwire.MsgTx) *wire.MsgTx {
	if tx == nil {
		return nil
	}

	shellTxIns := make([]*wire.TxIn, len(tx.TxIn))
	for i, txIn := range tx.TxIn {
		shellTxIns[i] = ToShellTxIn(txIn)
	}

	shellTxOuts := make([]*wire.TxOut, len(tx.TxOut))
	for i, txOut := range tx.TxOut {
		shellTxOuts[i] = ToShellTxOut(txOut)
	}

	return &wire.MsgTx{
		Version:  tx.Version,
		TxIn:     shellTxIns,
		TxOut:    shellTxOuts,
		LockTime: tx.LockTime,
	}
}
