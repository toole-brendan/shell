package mempool

import (
	"github.com/btcsuite/btcd/btcutil"
	"github.com/toole-brendan/shell/blockchain"
	"github.com/toole-brendan/shell/chaincfg/chainhash"
)

// MempoolAcceptResult holds the result from mempool acceptance check.
type MempoolAcceptResult struct {
	// TxFee is the fees paid in satoshi.
	TxFee btcutil.Amount

	// TxSize is the virtual size(vb) of the tx.
	TxSize int64

	// conflicts is a set of transactions whose inputs are spent by this
	// transaction(RBF).
	Conflicts map[chainhash.Hash]*btcutil.Tx

	// MissingParents is a set of outpoints that are used by this
	// transaction which cannot be found. Transaction is an orphan if any
	// of the referenced transaction outputs don't exist or are already
	// spent.
	//
	// NOTE: this field is mutually exclusive with other fields. If this
	// field is not nil, then other fields must be empty.
	MissingParents []*chainhash.Hash

	// utxoView is a set of the unspent transaction outputs referenced by
	// the inputs to this transaction.
	utxoView *blockchain.UtxoViewpoint

	// bestHeight is the best known height by the mempool.
	bestHeight int32
}
