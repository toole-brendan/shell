package mempool

import (
	"github.com/toole-brendan/shell/mining"
)

// TxPool represents a transaction memory pool
type TxPool struct {
	*TxPool
}

// TxDesc wraps a mining.TxDesc with additional mempool-specific fields
type TxDesc struct {
	mining.TxDesc
	StartingPriority float64
}

// Tag represents a tag for tracking transaction sources
type Tag uint64

// Config represents the configuration for the mempool
type Config struct {
	Policy               Policy
	ChainParams         *chaincfg.Params
	FetchUtxoView       func(*btcutil.Tx) (*blockchain.UtxoViewpoint, error)
	BestHeight          func() int32
	MedianTimePast      func() time.Time
	CalcSequenceLock    func(*btcutil.Tx, *blockchain.UtxoViewpoint) (*blockchain.SequenceLock, error)
	IsDeploymentActive  func(deploymentID uint32) (bool, error)
	SigCache            *txscript.SigCache
	HashCache           *txscript.HashCache
	AddrIndex           *indexers.AddrIndex
	FeeEstimator        *FeeEstimator
}

// DefaultBlockPrioritySize is the default size for high-priority/low-fee transactions
const DefaultBlockPrioritySize = 50000

// TxMempool interface for RPC server
type TxMempool interface {
	// Add methods that are used by RPC server
}

// New creates a new transaction memory pool
func New(cfg *Config) *TxPool {
	// Implementation will be added
	return &TxPool{}
}
