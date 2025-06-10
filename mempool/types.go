package mempool

import (
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/toole-brendan/shell/blockchain"
	"github.com/toole-brendan/shell/blockchain/indexers"
	"github.com/toole-brendan/shell/chaincfg"
	"github.com/toole-brendan/shell/mining"
	"github.com/toole-brendan/shell/txscript"
)

// TxDesc wraps a mining.TxDesc with additional mempool-specific fields
type TxDesc struct {
	mining.TxDesc
	StartingPriority float64
}

// Tag represents a tag for tracking transaction sources
type Tag uint64

// Policy houses the policy (configuration parameters) which is used to
// control the mempool.
type Policy struct {
	// MaxTxVersion is the transaction version that the mempool should
	// accept.  All transactions above this version are rejected as
	// non-standard.
	MaxTxVersion int32

	// DisableRelayPriority defines whether to relay free or low-fee
	// transactions that do not have enough priority to be relayed.
	DisableRelayPriority bool

	// AcceptNonStd defines whether to accept non-standard transactions. If
	// true, non-standard transactions will be accepted into the mempool.
	// Otherwise, all non-standard transactions will be rejected.
	AcceptNonStd bool

	// FreeTxRelayLimit defines the given amount in thousands of bytes
	// per minute that transactions with no fee are rate limited to.
	FreeTxRelayLimit float64

	// MaxOrphanTxs is the maximum number of orphan transactions
	// that can be queued.
	MaxOrphanTxs int

	// MaxOrphanTxSize is the maximum size allowed for orphan transactions.
	// This helps prevent memory exhaustion attacks from sending a lot of
	// of big orphans.
	MaxOrphanTxSize int

	// MaxSigOpCostPerTx is the cumulative maximum cost of all the signature
	// operations in a single transaction we will relay or mine.  It is a
	// fraction of the max signature operations for a block.
	MaxSigOpCostPerTx int

	// MinRelayTxFee defines the minimum transaction fee in BTC/kB to be
	// considered a non-zero fee.
	MinRelayTxFee btcutil.Amount

	// RejectReplacement, if true, rejects accepting replacement
	// transactions using the Replace-By-Fee (RBF) signaling policy into
	// the mempool.
	RejectReplacement bool
}

// Config represents the configuration for the mempool
type Config struct {
	Policy             Policy
	ChainParams        *chaincfg.Params
	FetchUtxoView      func(*btcutil.Tx) (*blockchain.UtxoViewpoint, error)
	BestHeight         func() int32
	MedianTimePast     func() time.Time
	CalcSequenceLock   func(*btcutil.Tx, *blockchain.UtxoViewpoint) (*blockchain.SequenceLock, error)
	IsDeploymentActive func(deploymentID uint32) (bool, error)
	SigCache           *txscript.SigCache
	HashCache          *txscript.HashCache
	AddrIndex          *indexers.AddrIndex
	FeeEstimator       *FeeEstimator
}

// DefaultBlockPrioritySize is the default size for high-priority/low-fee transactions
const DefaultBlockPrioritySize = 50000

// TxMempool interface for RPC server
type TxMempool interface {
	// Add methods that are used by RPC server
}
