// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mempool

import (
	"container/list"
	"fmt"
	"maps"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/davecgh/go-spew/spew"
	"github.com/toole-brendan/shell/blockchain"
	"github.com/toole-brendan/shell/btcjson"
	"github.com/toole-brendan/shell/chaincfg"
	"github.com/toole-brendan/shell/chaincfg/chainhash"
	"github.com/toole-brendan/shell/internal/convert"
	"github.com/toole-brendan/shell/mining"
	"github.com/toole-brendan/shell/txscript"
	"github.com/toole-brendan/shell/wire"
)

const (
	// orphanTTL is the maximum amount of time an orphan is allowed to
	// stay in the orphan pool before it expires and is evicted during the
	// next scan.
	orphanTTL = time.Minute * 15

	// orphanExpireScanInterval is the minimum amount of time in between
	// scans of the orphan pool to evict expired transactions.
	orphanExpireScanInterval = time.Minute * 5

	// MaxRBFSequence is the maximum sequence number an input can use to
	// signal that the transaction spending it can be replaced using the
	// Replace-By-Fee (RBF) policy.
	MaxRBFSequence = 0xfffffffd

	// MaxReplacementEvictions is the maximum number of transactions that
	// can be evicted from the mempool when accepting a transaction
	// replacement.
	MaxReplacementEvictions = 100

	// Transactions smaller than 65 non-witness bytes are not relayed to
	// mitigate CVE-2017-12842.
	MinStandardTxNonWitnessSize = 65
)

// orphanTx is normal transaction that references an ancestor transaction
// that is not yet available.  It also contains additional information related
// to it such as an expiration time to help prevent caching the orphan forever.
type orphanTx struct {
	tx         *btcutil.Tx
	tag        Tag
	expiration time.Time
}

// TxPool is used as a source of transactions that need to be mined into blocks
// and relayed to other peers.  It is safe for concurrent access from multiple
// peers.
type TxPool struct {
	// The following variables must only be used atomically.
	lastUpdated int64 // last time pool was updated

	mtx           sync.RWMutex
	cfg           Config
	pool          map[chainhash.Hash]*TxDesc
	orphans       map[chainhash.Hash]*orphanTx
	orphansByPrev map[wire.OutPoint]map[chainhash.Hash]*btcutil.Tx
	outpoints     map[wire.OutPoint]*btcutil.Tx
	pennyTotal    float64 // exponentially decaying total for penny spends.
	lastPennyUnix int64   // unix time of last ``penny spend''

	// nextExpireScan is the time after which the orphan pool will be
	// scanned in order to evict orphans.  This is NOT a hard deadline as
	// the scan will only run when an orphan is added to the pool as opposed
	// to on an unconditional timer.
	nextExpireScan time.Time
}

// New returns a new memory pool for validating and storing standalone
// transactions.
func New(cfg *Config) *TxPool {
	return &TxPool{
		cfg:            *cfg,
		pool:           make(map[chainhash.Hash]*TxDesc),
		orphans:        make(map[chainhash.Hash]*orphanTx),
		orphansByPrev:  make(map[wire.OutPoint]map[chainhash.Hash]*btcutil.Tx),
		outpoints:      make(map[wire.OutPoint]*btcutil.Tx),
		nextExpireScan: time.Now().Add(orphanExpireScanInterval),
	}
}

// Ensure the TxPool type implements the mining.TxSource interface.
var _ mining.TxSource = (*TxPool)(nil)

// Ensure the TxPool type implements the TxMemPool interface.
var _ TxMempool = (*TxPool)(nil)

// removeOrphan is the internal function which implements the public
// RemoveOrphan.  See the comment for RemoveOrphan for more details.
//
// This function MUST be called with the mempool lock held (for writes).
func (mp *TxPool) removeOrphan(tx *btcutil.Tx, removeRedeemers bool) {
	// Nothing to do if passed tx is not an orphan.
	txHash := convert.HashToShell(tx.Hash())
	otx, exists := mp.orphans[*txHash]
	if !exists {
		return
	}

	// Remove the reference from the previous orphan index.
	for _, txIn := range otx.tx.MsgTx().TxIn {
		orphans, exists := mp.orphansByPrev[convert.OutPointToShell(txIn.PreviousOutPoint)]
		if exists {
			delete(orphans, *txHash)

			// Remove the map entry altogether if there are no
			// longer any orphans which depend on it.
			if len(orphans) == 0 {
				delete(mp.orphansByPrev, convert.OutPointToShell(txIn.PreviousOutPoint))
			}
		}
	}

	// Remove any orphans that redeem outputs from this one if requested.
	if removeRedeemers {
		prevOut := wire.OutPoint{Hash: *txHash}
		for txOutIdx := range tx.MsgTx().TxOut {
			prevOut.Index = uint32(txOutIdx)
			for _, orphan := range mp.orphansByPrev[prevOut] {
				mp.removeOrphan(orphan, true)
			}
		}
	}

	// Remove the transaction from the orphan pool.
	delete(mp.orphans, *txHash)
}

// RemoveOrphan removes the passed orphan transaction from the orphan pool and
// previous orphan index.
//
// This function is safe for concurrent access.
func (mp *TxPool) RemoveOrphan(tx *btcutil.Tx) {
	mp.mtx.Lock()
	mp.removeOrphan(tx, false)
	mp.mtx.Unlock()
}

// RemoveOrphansByTag removes all orphan transactions tagged with the provided
// identifier.
//
// This function is safe for concurrent access.
func (mp *TxPool) RemoveOrphansByTag(tag Tag) uint64 {
	var numEvicted uint64
	mp.mtx.Lock()
	for _, otx := range mp.orphans {
		if otx.tag == tag {
			mp.removeOrphan(otx.tx, true)
			numEvicted++
		}
	}
	mp.mtx.Unlock()
	return numEvicted
}

// limitNumOrphans limits the number of orphan transactions by evicting a random
// orphan if adding a new one would cause it to overflow the max allowed.
//
// This function MUST be called with the mempool lock held (for writes).
func (mp *TxPool) limitNumOrphans() error {
	// Scan through the orphan pool and remove any expired orphans when it's
	// time.  This is done for efficiency so the scan only happens
	// periodically instead of on every orphan added to the pool.
	if now := time.Now(); now.After(mp.nextExpireScan) {
		origNumOrphans := len(mp.orphans)
		for _, otx := range mp.orphans {
			if now.After(otx.expiration) {
				// Remove redeemers too because the missing
				// parents are very unlikely to ever materialize
				// since the orphan has already been around more
				// than long enough for them to be delivered.
				mp.removeOrphan(otx.tx, true)
			}
		}

		// Set next expiration scan to occur after the scan interval.
		mp.nextExpireScan = now.Add(orphanExpireScanInterval)

		numOrphans := len(mp.orphans)
		if numExpired := origNumOrphans - numOrphans; numExpired > 0 {
			log.Debugf("Expired %d %s (remaining: %d)", numExpired,
				pickNoun(numExpired, "orphan", "orphans"),
				numOrphans)
		}
	}

	// Nothing to do if adding another orphan will not cause the pool to
	// exceed the limit.
	if len(mp.orphans)+1 <= mp.cfg.Policy.MaxOrphanTxs {
		return nil
	}

	// Remove a random entry from the map.  For most compilers, Go's
	// range statement iterates starting at a random item although
	// that is not 100% guaranteed by the spec.  The iteration order
	// is not important here because an adversary would have to be
	// able to pull off preimage attacks on the hashing function in
	// order to target eviction of specific entries anyways.
	for _, otx := range mp.orphans {
		// Don't remove redeemers in the case of a random eviction since
		// it is quite possible it might be needed again shortly.
		mp.removeOrphan(otx.tx, false)
		break
	}

	return nil
}

// addOrphan adds an orphan transaction to the orphan pool.
//
// This function MUST be called with the mempool lock held (for writes).
func (mp *TxPool) addOrphan(tx *btcutil.Tx, tag Tag) {
	// Nothing to do if no orphans are allowed.
	if mp.cfg.Policy.MaxOrphanTxs <= 0 {
		return
	}

	// Limit the number orphan transactions to prevent memory exhaustion.
	// This will periodically remove any expired orphans and evict a random
	// orphan if space is still needed.
	mp.limitNumOrphans()

	mp.orphans[*convert.HashToShell(tx.Hash())] = &orphanTx{
		tx:         tx,
		tag:        tag,
		expiration: time.Now().Add(orphanTTL),
	}
	for _, txIn := range tx.MsgTx().TxIn {
		outPoint := convert.OutPointToShell(txIn.PreviousOutPoint)
		if _, exists := mp.orphansByPrev[outPoint]; !exists {
			mp.orphansByPrev[outPoint] =
				make(map[chainhash.Hash]*btcutil.Tx)
		}
		mp.orphansByPrev[outPoint][*convert.HashToShell(tx.Hash())] = tx
	}

	log.Debugf("Stored orphan transaction %v (total: %d)", tx.Hash(),
		len(mp.orphans))
}

// maybeAddOrphan potentially adds an orphan to the orphan pool.
//
// This function MUST be called with the mempool lock held (for writes).
func (mp *TxPool) maybeAddOrphan(tx *btcutil.Tx, tag Tag) error {
	// Ignore orphan transactions that are too large.  This helps avoid
	// a memory exhaustion attack based on sending a lot of really large
	// orphans.  In the case there is a valid transaction larger than this,
	// it will ultimtely be rebroadcast after the parent transactions
	// have been mined or otherwise received.
	//
	// Note that the number of orphan transactions in the orphan pool is
	// also limited, so this equates to a maximum memory used of
	// mp.cfg.Policy.MaxOrphanTxSize * mp.cfg.Policy.MaxOrphanTxs (which is ~5MB
	// using the default values at the time this comment was written).
	serializedLen := tx.MsgTx().SerializeSize()
	if serializedLen > mp.cfg.Policy.MaxOrphanTxSize {
		str := fmt.Sprintf("orphan transaction size of %d bytes is "+
			"larger than max allowed size of %d bytes",
			serializedLen, mp.cfg.Policy.MaxOrphanTxSize)
		return txRuleError(wire.RejectNonstandard, str)
	}

	// Add the orphan if the none of the above disqualified it.
	mp.addOrphan(tx, tag)

	return nil
}

// removeOrphanDoubleSpends removes all orphans which spend outputs spent by the
// passed transaction from the orphan pool.  Removing those orphans then leads
// to removing all orphans which rely on them, recursively.  This is necessary
// when a transaction is added to the main pool because it may spend outputs
// that orphans also spend.
//
// This function MUST be called with the mempool lock held (for writes).
func (mp *TxPool) removeOrphanDoubleSpends(tx *btcutil.Tx) {
	msgTx := tx.MsgTx()
	for _, txIn := range msgTx.TxIn {
		for _, orphan := range mp.orphansByPrev[convert.OutPointToShell(txIn.PreviousOutPoint)] {
			mp.removeOrphan(orphan, true)
		}
	}
}

// isTransactionInPool returns whether or not the passed transaction already
// exists in the main pool.
//
// This function MUST be called with the mempool lock held (for reads).
func (mp *TxPool) isTransactionInPool(hash *chainhash.Hash) bool {
	if _, exists := mp.pool[*hash]; exists {
		return true
	}

	return false
}

// IsTransactionInPool returns whether or not the passed transaction already
// exists in the main pool.
//
// This function is safe for concurrent access.
func (mp *TxPool) IsTransactionInPool(hash *chainhash.Hash) bool {
	// Protect concurrent access.
	mp.mtx.RLock()
	inPool := mp.isTransactionInPool(hash)
	mp.mtx.RUnlock()

	return inPool
}

// isOrphanInPool returns whether or not the passed transaction already exists
// in the orphan pool.
//
// This function MUST be called with the mempool lock held (for reads).
func (mp *TxPool) isOrphanInPool(hash *chainhash.Hash) bool {
	if _, exists := mp.orphans[*hash]; exists {
		return true
	}

	return false
}

// IsOrphanInPool returns whether or not the passed transaction already exists
// in the orphan pool.
//
// This function is safe for concurrent access.
func (mp *TxPool) IsOrphanInPool(hash *chainhash.Hash) bool {
	// Protect concurrent access.
	mp.mtx.RLock()
	inPool := mp.isOrphanInPool(hash)
	mp.mtx.RUnlock()

	return inPool
}

// haveTransaction returns whether or not the passed transaction already exists
// in the main pool or in the orphan pool.
//
// This function MUST be called with the mempool lock held (for reads).
func (mp *TxPool) haveTransaction(hash *chainhash.Hash) bool {
	return mp.isTransactionInPool(hash) || mp.isOrphanInPool(hash)
}

// HaveTransaction returns whether or not the passed transaction already exists
// in the main pool or in the orphan pool.
//
// This function is safe for concurrent access.
func (mp *TxPool) HaveTransaction(hash *chainhash.Hash) bool {
	// Protect concurrent access.
	mp.mtx.RLock()
	haveTx := mp.haveTransaction(hash)
	mp.mtx.RUnlock()

	return haveTx
}

// removeTransaction is the internal function which implements the public
// RemoveTransaction.  See the comment for RemoveTransaction for more details.
//
// This function MUST be called with the mempool lock held (for writes).
func (mp *TxPool) removeTransaction(tx *btcutil.Tx, removeRedeemers bool) {
	txHash := convert.HashToShell(tx.Hash())
	if removeRedeemers {
		// Remove any transactions which rely on this one.
		for i := uint32(0); i < uint32(len(tx.MsgTx().TxOut)); i++ {
			prevOut := wire.OutPoint{Hash: *txHash, Index: i}
			if txRedeemer, exists := mp.outpoints[prevOut]; exists {
				mp.removeTransaction(txRedeemer, true)
			}
		}
	}

	// Remove the transaction if needed.
	if txDesc, exists := mp.pool[*txHash]; exists {
		// Remove unconfirmed address index entries associated with the
		// transaction if enabled.
		if mp.cfg.AddrIndex != nil {
			mp.cfg.AddrIndex.RemoveUnconfirmedTx(txHash)
		}

		// Mark the referenced outpoints as unspent by the pool.
		for _, txIn := range txDesc.Tx.MsgTx().TxIn {
			delete(mp.outpoints, convert.OutPointToShell(txIn.PreviousOutPoint))
		}
		delete(mp.pool, *txHash)
		atomic.StoreInt64(&mp.lastUpdated, time.Now().Unix())
	}
}

// RemoveTransaction removes the passed transaction from the mempool. When the
// removeRedeemers flag is set, any transactions that redeem outputs from the
// removed transaction will also be removed recursively from the mempool, as
// they would otherwise become orphans.
//
// This function is safe for concurrent access.
func (mp *TxPool) RemoveTransaction(tx *btcutil.Tx, removeRedeemers bool) {
	// Protect concurrent access.
	mp.mtx.Lock()
	mp.removeTransaction(tx, removeRedeemers)
	mp.mtx.Unlock()
}

// RemoveDoubleSpends removes all transactions which spend outputs spent by the
// passed transaction from the memory pool.  Removing those transactions then
// leads to removing all transactions which rely on them, recursively.  This is
// necessary when a block is connected to the main chain because the block may
// contain transactions which were previously unknown to the memory pool.
//
// This function is safe for concurrent access.
func (mp *TxPool) RemoveDoubleSpends(tx *btcutil.Tx) {
	// Protect concurrent access.
	mp.mtx.Lock()
	for _, txIn := range tx.MsgTx().TxIn {
		if txRedeemer, ok := mp.outpoints[convert.OutPointToShell(txIn.PreviousOutPoint)]; ok {
			if !txRedeemer.Hash().IsEqual(tx.Hash()) {
				mp.removeTransaction(txRedeemer, true)
			}
		}
	}
	mp.mtx.Unlock()
}

// addTransaction adds the passed transaction to the memory pool.  It should
// not be called directly as it doesn't perform any validation.  This is a
// helper for maybeAcceptTransaction.
//
// This function MUST be called with the mempool lock held (for writes).
func (mp *TxPool) addTransaction(utxoView *blockchain.UtxoViewpoint, tx *btcutil.Tx, height int32, fee int64) *TxDesc {
	// Add the transaction to the pool and mark the referenced outpoints
	// as spent by the pool.
	txD := &TxDesc{
		TxDesc: mining.TxDesc{
			Tx:       tx,
			Added:    time.Now(),
			Height:   height,
			Fee:      fee,
			FeePerKB: fee * 1000 / GetTxVirtualSize(tx),
		},
		StartingPriority: mining.CalcPriority(tx.MsgTx(), utxoView, height),
	}

	mp.pool[*convert.HashToShell(tx.Hash())] = txD
	for _, txIn := range tx.MsgTx().TxIn {
		mp.outpoints[convert.OutPointToShell(txIn.PreviousOutPoint)] = tx
	}
	atomic.StoreInt64(&mp.lastUpdated, time.Now().Unix())

	// Add unconfirmed address index entries associated with the transaction
	// if enabled.
	if mp.cfg.AddrIndex != nil {
		mp.cfg.AddrIndex.AddUnconfirmedTx(tx, utxoView)
	}

	// Record this tx for fee estimation if enabled.
	if mp.cfg.FeeEstimator != nil {
		mp.cfg.FeeEstimator.ObserveTransaction(txD)
	}

	return txD
}

// checkPoolDoubleSpend checks whether or not the passed transaction is
// attempting to spend coins already spent by other transactions in the pool.
// If it does, we'll check whether each of those transactions are signaling for
// replacement. If just one of them isn't, an error is returned. Otherwise, a
// boolean is returned signaling that the transaction is a replacement. Note it
// does not check for double spends against transactions already in the main
// chain.
//
// This function MUST be called with the mempool lock held (for reads).
func (mp *TxPool) checkPoolDoubleSpend(tx *btcutil.Tx) (bool, error) {
	var isReplacement bool
	for _, txIn := range tx.MsgTx().TxIn {
		conflict, ok := mp.outpoints[convert.OutPointToShell(txIn.PreviousOutPoint)]
		if !ok {
			continue
		}

		// Reject the transaction if we don't accept replacement
		// transactions or if it doesn't signal replacement.
		if mp.cfg.Policy.RejectReplacement ||
			!mp.signalsReplacement(conflict, nil) {
			str := fmt.Sprintf("output already spent in mempool: "+
				"output=%v, tx=%v", txIn.PreviousOutPoint,
				conflict.Hash())
			return false, txRuleError(wire.RejectDuplicate, str)
		}

		isReplacement = true
	}

	return isReplacement, nil
}

// signalsReplacement determines if a transaction is signaling that it can be
// replaced using the Replace-By-Fee (RBF) policy. This policy specifies two
// ways a transaction can signal that it is replaceable:
//
// Explicit signaling: A transaction is considered to have opted in to allowing
// replacement of itself if any of its inputs have a sequence number less than
// 0xfffffffe.
//
// Inherited signaling: Transactions that don't explicitly signal replaceability
// are replaceable under this policy for as long as any one of their ancestors
// signals replaceability and remains unconfirmed.
//
// The cache is optional and serves as an optimization to avoid visiting
// transactions we've already determined don't signal replacement.
//
// This function MUST be called with the mempool lock held (for reads).
func (mp *TxPool) signalsReplacement(tx *btcutil.Tx,
	cache map[chainhash.Hash]struct{}) bool {

	// If a cache was not provided, we'll initialize one now to use for the
	// recursive calls.
	if cache == nil {
		cache = make(map[chainhash.Hash]struct{})
	}

	for _, txIn := range tx.MsgTx().TxIn {
		if txIn.Sequence <= MaxRBFSequence {
			return true
		}

		hash := txIn.PreviousOutPoint.Hash
		shellHash := convert.HashToShellValue(hash)
		unconfirmedAncestor, ok := mp.pool[shellHash]
		if !ok {
			continue
		}

		// If we've already determined the transaction doesn't signal
		// replacement, we can avoid visiting it again.
		if _, ok := cache[shellHash]; ok {
			continue
		}

		if mp.signalsReplacement(unconfirmedAncestor.Tx, cache) {
			return true
		}

		// Since the transaction doesn't signal replacement, we'll cache
		// its result to ensure we don't attempt to determine so again.
		cache[shellHash] = struct{}{}
	}

	return false
}

// txAncestors returns all of the unconfirmed ancestors of the given
// transaction. Given transactions A, B, and C where C spends B and B spends A,
// A and B are considered ancestors of C.
//
// The cache is optional and serves as an optimization to avoid visiting
// transactions we've already determined ancestors of.
//
// This function MUST be called with the mempool lock held (for reads).
func (mp *TxPool) txAncestors(tx *btcutil.Tx,
	cache map[chainhash.Hash]map[chainhash.Hash]*btcutil.Tx) map[chainhash.Hash]*btcutil.Tx {

	// If a cache was not provided, we'll initialize one now to use for the
	// recursive calls.
	if cache == nil {
		cache = make(map[chainhash.Hash]map[chainhash.Hash]*btcutil.Tx)
	}

	ancestors := make(map[chainhash.Hash]*btcutil.Tx)
	for _, txIn := range tx.MsgTx().TxIn {
		parent, ok := mp.pool[convert.HashToShellValue(txIn.PreviousOutPoint.Hash)]
		if !ok {
			continue
		}
		ancestors[*convert.HashToShell(parent.Tx.Hash())] = parent.Tx

		// Determine if the ancestors of this ancestor have already been
		// computed. If they haven't, we'll do so now and cache them to
		// use them later on if necessary.
		moreAncestors, ok := cache[*convert.HashToShell(parent.Tx.Hash())]
		if !ok {
			moreAncestors = mp.txAncestors(parent.Tx, cache)
			cache[*convert.HashToShell(parent.Tx.Hash())] = moreAncestors
		}

		maps.Copy(ancestors, moreAncestors)
	}

	return ancestors
}

// txDescendants returns all of the unconfirmed descendants of the given
// transaction. Given transactions A, B, and C where C spends B and B spends A,
// B and C are considered descendants of A. A cache can be provided in order to
// easily retrieve the descendants of transactions we've already determined the
// descendants of.
//
// This function MUST be called with the mempool lock held (for reads).
func (mp *TxPool) txDescendants(tx *btcutil.Tx,
	cache map[chainhash.Hash]map[chainhash.Hash]*btcutil.Tx) map[chainhash.Hash]*btcutil.Tx {

	// If a cache was not provided, we'll initialize one now to use for the
	// recursive calls.
	if cache == nil {
		cache = make(map[chainhash.Hash]map[chainhash.Hash]*btcutil.Tx)
	}

	// We'll go through all of the outputs of the transaction to determine
	// if they are spent by any other mempool transactions.
	descendants := make(map[chainhash.Hash]*btcutil.Tx)
	op := wire.OutPoint{Hash: *convert.HashToShell(tx.Hash())}
	for i := range tx.MsgTx().TxOut {
		op.Index = uint32(i)
		descendant, ok := mp.outpoints[op]
		if !ok {
			continue
		}
		descendantShellHash := *convert.HashToShell(descendant.Hash())
		descendants[descendantShellHash] = descendant

		// Determine if the descendants of this descendant have already
		// been computed. If they haven't, we'll do so now and cache
		// them to use them later on if necessary.
		moreDescendants, ok := cache[descendantShellHash]
		if !ok {
			moreDescendants = mp.txDescendants(descendant, cache)
			cache[descendantShellHash] = moreDescendants
		}

		for _, moreDescendant := range moreDescendants {
			descendants[*convert.HashToShell(moreDescendant.Hash())] = moreDescendant
		}
	}

	return descendants
}

// txConflicts returns all of the unconfirmed transactions that would become
// conflicts if we were to accept the given transaction into the mempool. An
// unconfirmed conflict is known as a transaction that spends an output already
// spent by a different transaction within the mempool. Any descendants of these
// transactions are also considered conflicts as they would no longer exist.
// These are generally not allowed except for transactions that signal RBF
// support.
//
// This function MUST be called with the mempool lock held (for reads).
func (mp *TxPool) txConflicts(tx *btcutil.Tx) map[chainhash.Hash]*btcutil.Tx {
	conflicts := make(map[chainhash.Hash]*btcutil.Tx)
	for _, txIn := range tx.MsgTx().TxIn {
		conflict, ok := mp.outpoints[convert.OutPointToShell(txIn.PreviousOutPoint)]
		if !ok {
			continue
		}
		conflicts[*convert.HashToShell(conflict.Hash())] = conflict
		descendants := mp.txDescendants(conflict, nil)
		maps.Copy(conflicts, descendants)
	}
	return conflicts
}

// CheckSpend checks whether the passed outpoint is already spent by a
// transaction in the mempool. If that's the case the spending transaction will
// be returned, if not nil will be returned.
func (mp *TxPool) CheckSpend(op wire.OutPoint) *btcutil.Tx {
	mp.mtx.RLock()
	txR := mp.outpoints[op]
	mp.mtx.RUnlock()

	return txR
}

// fetchInputUtxos loads utxo details about the input transactions referenced by
// the passed transaction.  First, it loads the details form the viewpoint of
// the main chain, then it adjusts them based upon the contents of the
// transaction pool.
//
// This function MUST be called with the mempool lock held (for reads).
func (mp *TxPool) fetchInputUtxos(tx *btcutil.Tx) (*blockchain.UtxoViewpoint, error) {
	utxoView, err := mp.cfg.FetchUtxoView(tx)
	if err != nil {
		return nil, err
	}

	// Attempt to populate any missing inputs from the transaction pool.
	for _, txIn := range tx.MsgTx().TxIn {
		prevOut := &txIn.PreviousOutPoint
		entry := utxoView.LookupEntry(convert.OutPointToShell(*prevOut))
		if entry != nil && !entry.IsSpent() {
			continue
		}

		if poolTxDesc, exists := mp.pool[convert.HashToShellValue(prevOut.Hash)]; exists {
			// AddTxOut ignores out of range index values, so it is
			// safe to call without bounds checking here.
			utxoView.AddTxOut(poolTxDesc.Tx, prevOut.Index,
				mining.UnminedHeight)
		}
	}

	return utxoView, nil
}

// FetchTransaction returns the requested transaction from the transaction pool.
// This only fetches from the main transaction pool and does not include
// orphans.
//
// This function is safe for concurrent access.
func (mp *TxPool) FetchTransaction(txHash *chainhash.Hash) (*btcutil.Tx, error) {
	// Protect concurrent access.
	mp.mtx.RLock()
	txDesc, exists := mp.pool[*txHash]
	mp.mtx.RUnlock()

	if exists {
		return txDesc.Tx, nil
	}

	return nil, fmt.Errorf("transaction is not in the pool")
}

// validateReplacement determines whether a transaction is deemed as a valid
// replacement of all of its conflicts according to the RBF policy. If it is
// valid, no error is returned. Otherwise, an error is returned indicating what
// went wrong.
//
// This function MUST be called with the mempool lock held (for reads).
func (mp *TxPool) validateReplacement(tx *btcutil.Tx,
	txFee int64) (map[chainhash.Hash]*btcutil.Tx, error) {

	// First, we'll make sure the set of conflicting transactions doesn't
	// exceed the maximum allowed.
	conflicts := mp.txConflicts(tx)
	if len(conflicts) > MaxReplacementEvictions {
		str := fmt.Sprintf("%v: replacement transaction evicts more "+
			"transactions than permitted: max is %v, evicts %v",
			tx.Hash(), MaxReplacementEvictions, len(conflicts))
		return nil, txRuleError(wire.RejectNonstandard, str)
	}

	// The set of conflicts (transactions we'll replace) and ancestors
	// should not overlap, otherwise the replacement would be spending an
	// output that no longer exists.
	for ancestorHash := range mp.txAncestors(tx, nil) {
		if _, ok := conflicts[ancestorHash]; !ok {
			continue
		}
		str := fmt.Sprintf("%v: replacement transaction spends parent "+
			"transaction %v", tx.Hash(), ancestorHash)
		return nil, txRuleError(wire.RejectInvalid, str)
	}

	// The replacement should have a higher fee rate than each of the
	// conflicting transactions and a higher absolute fee than the fee sum
	// of all the conflicting transactions.
	//
	// We usually don't want to accept replacements with lower fee rates
	// than what they replaced as that would lower the fee rate of the next
	// block. Requiring that the fee rate always be increased is also an
	// easy-to-reason about way to prevent DoS attacks via replacements.
	var (
		txSize           = GetTxVirtualSize(tx)
		txFeeRate        = txFee * 1000 / txSize
		conflictsFee     int64
		conflictsParents = make(map[chainhash.Hash]struct{})
	)
	for hash, conflict := range conflicts {
		if txFeeRate <= mp.pool[hash].FeePerKB {
			str := fmt.Sprintf("%v: replacement transaction has an "+
				"insufficient fee rate: needs more than %v, "+
				"has %v", tx.Hash(), mp.pool[hash].FeePerKB,
				txFeeRate)
			return nil, txRuleError(wire.RejectInsufficientFee, str)
		}

		conflictsFee += mp.pool[hash].Fee

		// We'll track each conflict's parents to ensure the replacement
		// isn't spending any new unconfirmed inputs.
		for _, txIn := range conflict.MsgTx().TxIn {
			conflictsParents[*convert.HashToShell(&txIn.PreviousOutPoint.Hash)] = struct{}{}
		}
	}

	// It should also have an absolute fee greater than all of the
	// transactions it intends to replace and pay for its own bandwidth,
	// which is determined by our minimum relay fee.
	minFee := calcMinRequiredTxRelayFee(txSize, mp.cfg.Policy.MinRelayTxFee)
	if txFee < conflictsFee+minFee {
		str := fmt.Sprintf("%v: replacement transaction has an "+
			"insufficient absolute fee: needs %v, has %v",
			tx.Hash(), conflictsFee+minFee, txFee)
		return nil, txRuleError(wire.RejectInsufficientFee, str)
	}

	// Finally, it should not spend any new unconfirmed outputs, other than
	// the ones already included in the parents of the conflicting
	// transactions it'll replace.
	for _, txIn := range tx.MsgTx().TxIn {
		shellHash := *convert.HashToShell(&txIn.PreviousOutPoint.Hash)
		if _, ok := conflictsParents[shellHash]; ok {
			continue
		}
		// Confirmed outputs are valid to spend in the replacement.
		if _, ok := mp.pool[shellHash]; !ok {
			continue
		}
		str := fmt.Sprintf("replacement transaction spends new "+
			"unconfirmed input %v not found in conflicting "+
			"transactions", txIn.PreviousOutPoint)
		return nil, txRuleError(wire.RejectInvalid, str)
	}

	return conflicts, nil
}

// maybeAcceptTransaction is the internal function which implements the public
// MaybeAcceptTransaction.  See the comment for MaybeAcceptTransaction for
// more details.
//
// This function MUST be called with the mempool lock held (for writes).
func (mp *TxPool) maybeAcceptTransaction(tx *btcutil.Tx, isNew, rateLimit,
	rejectDupOrphans bool) ([]*chainhash.Hash, *TxDesc, error) {

	txHash := tx.Hash()

	// Check for mempool acceptance.
	r, err := mp.checkMempoolAcceptance(
		tx, isNew, rateLimit, rejectDupOrphans,
	)
	if err != nil {
		return nil, nil, err
	}

	// Exit early if this transaction is missing parents.
	if len(r.MissingParents) > 0 {
		return r.MissingParents, nil, nil
	}

	// Now that we've deemed the transaction as valid, we can add it to the
	// mempool. If it ended up replacing any transactions, we'll remove them
	// first.
	for _, conflict := range r.Conflicts {
		log.Debugf("Replacing transaction %v (fee_rate=%v sat/kb) "+
			"with %v (fee_rate=%v sat/kb)\n", conflict.Hash(),
			mp.pool[*convert.HashToShell(conflict.Hash())].FeePerKB, tx.Hash(),
			int64(r.TxFee)*1000/r.TxSize)

		// The conflict set should already include the descendants for
		// each one, so we don't need to remove the redeemers within
		// this call as they'll be removed eventually.
		mp.removeTransaction(conflict, false)
	}
	txD := mp.addTransaction(r.utxoView, tx, r.bestHeight, int64(r.TxFee))

	log.Debugf("Accepted transaction %v (pool size: %v)", txHash,
		len(mp.pool))

	return nil, txD, nil
}

// MaybeAcceptTransaction is the main workhorse for handling insertion of new
// free-standing transactions into a memory pool.  It includes functionality
// such as rejecting duplicate transactions, ensuring transactions follow all
// rules, detecting orphan transactions, and insertion into the memory pool.
//
// If the transaction is an orphan (missing parent transactions), the
// transaction is NOT added to the orphan pool, but each unknown referenced
// parent is returned.  Use ProcessTransaction instead if new orphans should
// be added to the orphan pool.
//
// This function is safe for concurrent access.
func (mp *TxPool) MaybeAcceptTransaction(tx *btcutil.Tx, isNew, rateLimit bool) ([]*chainhash.Hash, *TxDesc, error) {
	// Protect concurrent access.
	mp.mtx.Lock()
	hashes, txD, err := mp.maybeAcceptTransaction(tx, isNew, rateLimit, true)
	mp.mtx.Unlock()

	return hashes, txD, err
}

// processOrphans is the internal function which implements the public
// ProcessOrphans.  See the comment for ProcessOrphans for more details.
//
// This function MUST be called with the mempool lock held (for writes).
func (mp *TxPool) processOrphans(acceptedTx *btcutil.Tx) []*TxDesc {
	var acceptedTxns []*TxDesc

	// Start with processing at least the passed transaction.
	processList := list.New()
	processList.PushBack(acceptedTx)
	for processList.Len() > 0 {
		// Pop the transaction to process from the front of the list.
		firstElement := processList.Remove(processList.Front())
		processItem := firstElement.(*btcutil.Tx)

		prevOut := wire.OutPoint{Hash: *convert.HashToShell(processItem.Hash())}
		for txOutIdx := range processItem.MsgTx().TxOut {
			// Look up all orphans that redeem the output that is
			// now available.  This will typically only be one, but
			// it could be multiple if the orphan pool contains
			// double spends.  While it may seem odd that the orphan
			// pool would allow this since there can only possibly
			// ultimately be a single redeemer, it's important to
			// track it this way to prevent malicious actors from
			// being able to purposely constructing orphans that
			// would otherwise make outputs unspendable.
			//
			// Skip to the next available output if there are none.
			prevOut.Index = uint32(txOutIdx)
			orphans, exists := mp.orphansByPrev[prevOut]
			if !exists {
				continue
			}

			// Potentially accept an orphan into the tx pool.
			for _, tx := range orphans {
				missing, txD, err := mp.maybeAcceptTransaction(
					tx, true, true, false)
				if err != nil {
					// The orphan is now invalid, so there
					// is no way any other orphans which
					// redeem any of its outputs can be
					// accepted.  Remove them.
					mp.removeOrphan(tx, true)
					break
				}

				// Transaction is still an orphan.  Try the next
				// orphan which redeems this output.
				if len(missing) > 0 {
					continue
				}

				// Transaction was accepted into the main pool.
				//
				// Add it to the list of accepted transactions
				// that are no longer orphans, remove it from
				// the orphan pool, and add it to the list of
				// transactions to process so any orphans that
				// depend on it are handled too.
				acceptedTxns = append(acceptedTxns, txD)
				mp.removeOrphan(tx, false)
				processList.PushBack(tx)

				// Only one transaction for this outpoint can be
				// accepted, so the rest are now double spends
				// and are removed later.
				break
			}
		}
	}

	// Recursively remove any orphans that also redeem any outputs redeemed
	// by the accepted transactions since those are now definitive double
	// spends.
	mp.removeOrphanDoubleSpends(acceptedTx)
	for _, txD := range acceptedTxns {
		mp.removeOrphanDoubleSpends(txD.Tx)
	}

	return acceptedTxns
}

// ProcessOrphans determines if there are any orphans which depend on the passed
// transaction hash (it is possible that they are no longer orphans) and
// potentially accepts them to the memory pool.  It repeats the process for the
// newly accepted transactions (to detect further orphans which may no longer be
// orphans) until there are no more.
//
// It returns a slice of transactions added to the mempool.  A nil slice means
// no transactions were moved from the orphan pool to the mempool.
//
// This function is safe for concurrent access.
func (mp *TxPool) ProcessOrphans(acceptedTx *btcutil.Tx) []*TxDesc {
	mp.mtx.Lock()
	acceptedTxns := mp.processOrphans(acceptedTx)
	mp.mtx.Unlock()

	return acceptedTxns
}

// ProcessTransaction is the main workhorse for handling insertion of new
// free-standing transactions into the memory pool.  It includes functionality
// such as rejecting duplicate transactions, ensuring transactions follow all
// rules, orphan transaction handling, and insertion into the memory pool.
//
// It returns a slice of transactions added to the mempool.  When the
// error is nil, the list will include the passed transaction itself along
// with any additional orphan transactions that were added as a result of
// the passed one being accepted.
//
// This function is safe for concurrent access.
func (mp *TxPool) ProcessTransaction(tx *btcutil.Tx, allowOrphan, rateLimit bool, tag Tag) ([]*TxDesc, error) {
	log.Tracef("Processing transaction %v", tx.Hash())

	// Protect concurrent access.
	mp.mtx.Lock()
	defer mp.mtx.Unlock()

	// Potentially accept the transaction to the memory pool.
	missingParents, txD, err := mp.maybeAcceptTransaction(tx, true, rateLimit,
		true)
	if err != nil {
		return nil, err
	}

	if len(missingParents) == 0 {
		// Accept any orphan transactions that depend on this
		// transaction (they may no longer be orphans if all inputs
		// are now available) and repeat for those accepted
		// transactions until there are no more.
		newTxs := mp.processOrphans(tx)
		acceptedTxs := make([]*TxDesc, len(newTxs)+1)

		// Add the parent transaction first so remote nodes
		// do not add orphans.
		acceptedTxs[0] = txD
		copy(acceptedTxs[1:], newTxs)

		return acceptedTxs, nil
	}

	// The transaction is an orphan (has inputs missing).  Reject
	// it if the flag to allow orphans is not set.
	if !allowOrphan {
		// Only use the first missing parent transaction in
		// the error message.
		//
		// NOTE: RejectDuplicate is really not an accurate
		// reject code here, but it matches the reference
		// implementation and there isn't a better choice due
		// to the limited number of reject codes.  Missing
		// inputs is assumed to mean they are already spent
		// which is not really always the case.
		str := fmt.Sprintf("orphan transaction %v references "+
			"outputs of unknown or fully-spent "+
			"transaction %v", tx.Hash(), missingParents[0])
		return nil, txRuleError(wire.RejectDuplicate, str)
	}

	// Potentially add the orphan transaction to the orphan pool.
	err = mp.maybeAddOrphan(tx, tag)
	return nil, err
}

// Count returns the number of transactions in the main pool.  It does not
// include the orphan pool.
//
// This function is safe for concurrent access.
func (mp *TxPool) Count() int {
	mp.mtx.RLock()
	count := len(mp.pool)
	mp.mtx.RUnlock()

	return count
}

// TxHashes returns a slice of hashes for all the transactions in the memory
// pool.
//
// This function is safe for concurrent access.
func (mp *TxPool) TxHashes() []*chainhash.Hash {
	mp.mtx.RLock()
	hashes := make([]*chainhash.Hash, len(mp.pool))
	i := 0
	for hash := range mp.pool {
		hashCopy := hash
		hashes[i] = &hashCopy
		i++
	}
	mp.mtx.RUnlock()

	return hashes
}

// TxDescs returns a slice of descriptors for all the transactions in the pool.
// The descriptors are to be treated as read only.
//
// This function is safe for concurrent access.
func (mp *TxPool) TxDescs() []*TxDesc {
	mp.mtx.RLock()
	descs := make([]*TxDesc, len(mp.pool))
	i := 0
	for _, desc := range mp.pool {
		descs[i] = desc
		i++
	}
	mp.mtx.RUnlock()

	return descs
}

// MiningDescs returns a slice of mining descriptors for all the transactions
// in the pool.
//
// This is part of the mining.TxSource interface implementation and is safe for
// concurrent access as required by the interface contract.
func (mp *TxPool) MiningDescs() []*mining.TxDesc {
	mp.mtx.RLock()
	descs := make([]*mining.TxDesc, len(mp.pool))
	i := 0
	for _, desc := range mp.pool {
		descs[i] = &desc.TxDesc
		i++
	}
	mp.mtx.RUnlock()

	return descs
}

// RawMempoolVerbose returns all the entries in the mempool as a fully
// populated btcjson result.
//
// This function is safe for concurrent access.
func (mp *TxPool) RawMempoolVerbose() map[string]*btcjson.GetRawMempoolVerboseResult {
	mp.mtx.RLock()
	defer mp.mtx.RUnlock()

	result := make(map[string]*btcjson.GetRawMempoolVerboseResult,
		len(mp.pool))
	bestHeight := mp.cfg.BestHeight()

	for _, desc := range mp.pool {
		// Calculate the current priority based on the inputs to
		// the transaction.  Use zero if one or more of the
		// input transactions can't be found for some reason.
		tx := desc.Tx
		var currentPriority float64
		utxos, err := mp.fetchInputUtxos(tx)
		if err == nil {
			currentPriority = mining.CalcPriority(
				tx.MsgTx(), utxos,
				bestHeight+1,
			)
		}

		mpd := &btcjson.GetRawMempoolVerboseResult{
			Size:             int32(tx.MsgTx().SerializeSize()),
			Vsize:            int32(GetTxVirtualSize(tx)),
			Weight:           int32(blockchain.GetTransactionWeight(tx)),
			Fee:              btcutil.Amount(desc.Fee).ToBTC(),
			Time:             desc.Added.Unix(),
			Height:           int64(desc.Height),
			StartingPriority: desc.StartingPriority,
			CurrentPriority:  currentPriority,
			Depends:          make([]string, 0),
		}
		for _, txIn := range tx.MsgTx().TxIn {
			hash := &txIn.PreviousOutPoint.Hash
			if mp.haveTransaction(convert.HashToShell(hash)) {
				mpd.Depends = append(mpd.Depends,
					hash.String())
			}
		}

		result[tx.Hash().String()] = mpd
	}

	return result
}

// LastUpdated returns the last time a transaction was added to or removed from
// the main pool.  It does not include the orphan pool.
//
// This function is safe for concurrent access.
func (mp *TxPool) LastUpdated() time.Time {
	return time.Unix(atomic.LoadInt64(&mp.lastUpdated), 0)
}

// CheckMempoolAcceptance behaves similarly to bitcoind's `testmempoolaccept`
// RPC method. It will perform a series of checks to decide whether this
// transaction can be accepted to the mempool. If not, the specific error is
// returned and the caller needs to take actions based on it.
func (mp *TxPool) CheckMempoolAcceptance(tx *btcutil.Tx) (
	*MempoolAcceptResult, error) {

	mp.mtx.RLock()
	defer mp.mtx.RUnlock()

	// Call checkMempoolAcceptance with isNew=true and rateLimit=true,
	// which has the effect that we always check the fee paid from this tx
	// is greater than min relay fee. We also reject this tx if it's
	// already an orphan.
	result, err := mp.checkMempoolAcceptance(tx, true, true, true)
	if err != nil {
		log.Errorf("CheckMempoolAcceptance: %v", err)
		return nil, err
	}

	log.Tracef("Tx %v passed mempool acceptance check: %v", tx.Hash(),
		spew.Sdump(result))

	return result, nil
}

// checkMempoolAcceptance performs a series of validations on the given
// transaction. It returns an error when the transaction fails to meet the
// mempool policy, otherwise a `mempoolAcceptResult` is returned.
func (mp *TxPool) checkMempoolAcceptance(tx *btcutil.Tx,
	isNew, rateLimit, rejectDupOrphans bool) (*MempoolAcceptResult, error) {

	txHash := convert.HashToShell(tx.Hash())

	// Check for segwit activeness.
	if err := mp.validateSegWitDeployment(tx); err != nil {
		return nil, err
	}

	// Don't accept the transaction if it already exists in the pool. This
	// applies to orphan transactions as well when the reject duplicate
	// orphans flag is set. This check is intended to be a quick check to
	// weed out duplicates.
	if mp.isTransactionInPool(txHash) || (rejectDupOrphans &&
		mp.isOrphanInPool(txHash)) {

		str := fmt.Sprintf("already have transaction in mempool %v",
			txHash)
		return nil, txRuleError(wire.RejectDuplicate, str)
	}

	// Disallow transactions under the minimum standardness size.
	if tx.MsgTx().SerializeSizeStripped() < MinStandardTxNonWitnessSize {
		str := fmt.Sprintf("tx %v is too small", txHash)
		return nil, txRuleError(wire.RejectNonstandard, str)
	}

	// Perform preliminary sanity checks on the transaction. This makes use
	// of blockchain which contains the invariant rules for what
	// transactions are allowed into blocks.
	err := blockchain.CheckTransactionSanity(tx)
	if err != nil {
		if cerr, ok := err.(blockchain.RuleError); ok {
			return nil, chainRuleError(cerr)
		}

		return nil, err
	}

	// A standalone transaction must not be a coinbase transaction.
	if blockchain.IsCoinBase(tx) {
		str := fmt.Sprintf("transaction is an individual coinbase %v",
			txHash)

		return nil, txRuleError(wire.RejectInvalid, str)
	}

	// Get the current height of the main chain. A standalone transaction
	// will be mined into the next block at best, so its height is at least
	// one more than the current height.
	bestHeight := mp.cfg.BestHeight()
	nextBlockHeight := bestHeight + 1

	medianTimePast := mp.cfg.MedianTimePast()

	// The transaction may not use any of the same outputs as other
	// transactions already in the pool as that would ultimately result in
	// a double spend, unless those transactions signal for RBF. This check
	// is intended to be quick and therefore only detects double spends
	// within the transaction pool itself. The transaction could still be
	// double spending coins from the main chain at this point. There is a
	// more in-depth check that happens later after fetching the referenced
	// transaction inputs from the main chain which examines the actual
	// spend data and prevents double spends.
	isReplacement, err := mp.checkPoolDoubleSpend(tx)
	if err != nil {
		return nil, err
	}

	// Fetch all of the unspent transaction outputs referenced by the
	// inputs to this transaction. This function also attempts to fetch the
	// transaction itself to be used for detecting a duplicate transaction
	// without needing to do a separate lookup.
	utxoView, err := mp.fetchInputUtxos(tx)
	if err != nil {
		if cerr, ok := err.(blockchain.RuleError); ok {
			return nil, chainRuleError(cerr)
		}

		return nil, err
	}

	// Don't allow the transaction if it exists in the main chain and is
	// already fully spent.
	prevOut := wire.OutPoint{Hash: *txHash}
	for txOutIdx := range tx.MsgTx().TxOut {
		prevOut.Index = uint32(txOutIdx)

		entry := utxoView.LookupEntry(prevOut)
		if entry != nil && !entry.IsSpent() {
			return nil, txRuleError(wire.RejectDuplicate,
				"transaction already exists in blockchain")
		}

		utxoView.RemoveEntry(prevOut)
	}

	// Transaction is an orphan if any of the referenced transaction
	// outputs don't exist or are already spent. Adding orphans to the
	// orphan pool is not handled by this function, and the caller should
	// use maybeAddOrphan if this behavior is desired.
	var missingParents []*chainhash.Hash
	for outpoint, entry := range utxoView.Entries() {
		if entry == nil || entry.IsSpent() {
			// Must make a copy of the hash here since the iterator
			// is replaced and taking its address directly would
			// result in all the entries pointing to the same
			// memory location and thus all be the final hash.
			hashCopy := outpoint.Hash
			missingParents = append(missingParents, &hashCopy)
		}
	}

	// Exit early if this transaction is missing parents.
	if len(missingParents) > 0 {
		log.Debugf("Tx %v is an orphan with missing parents: %v",
			txHash, missingParents)

		return &MempoolAcceptResult{
			MissingParents: missingParents,
		}, nil
	}

	// Perform several checks on the transaction inputs using the invariant
	// rules in blockchain for what transactions are allowed into blocks.
	// Also returns the fees associated with the transaction which will be
	// used later.
	//
	// NOTE: this check must be performed before `validateStandardness` to
	// make sure a nil entry is not returned from `utxoView.LookupEntry`.
	txFee, err := blockchain.CheckTransactionInputs(
		tx, nextBlockHeight, utxoView, mp.cfg.ChainParams,
	)
	if err != nil {
		if cerr, ok := err.(blockchain.RuleError); ok {
			return nil, chainRuleError(cerr)
		}
		return nil, err
	}

	// Don't allow non-standard transactions or non-standard inputs if the
	// network parameters forbid their acceptance.
	err = mp.validateStandardness(
		tx, nextBlockHeight, medianTimePast, utxoView,
	)
	if err != nil {
		return nil, err
	}

	// Don't allow the transaction into the mempool unless its sequence
	// lock is active, meaning that it'll be allowed into the next block
	// with respect to its defined relative lock times.
	sequenceLock, err := mp.cfg.CalcSequenceLock(tx, utxoView)
	if err != nil {
		if cerr, ok := err.(blockchain.RuleError); ok {
			return nil, chainRuleError(cerr)
		}

		return nil, err
	}

	if !blockchain.SequenceLockActive(
		sequenceLock, nextBlockHeight, medianTimePast,
	) {

		return nil, txRuleError(wire.RejectNonstandard,
			"transaction's sequence locks on inputs not met")
	}

	// Don't allow transactions with an excessive number of signature
	// operations which would result in making it impossible to mine.
	if err := mp.validateSigCost(tx, utxoView); err != nil {
		return nil, err
	}

	txSize := GetTxVirtualSize(tx)

	// Don't allow transactions with fees too low to get into a mined
	// block.
	err = mp.validateRelayFeeMet(
		tx, txFee, txSize, utxoView, nextBlockHeight, isNew, rateLimit,
	)
	if err != nil {
		return nil, err
	}

	// If the transaction has any conflicts, and we've made it this far,
	// then we're processing a potential replacement.
	var conflicts map[chainhash.Hash]*btcutil.Tx
	if isReplacement {
		conflicts, err = mp.validateReplacement(tx, txFee)
		if err != nil {
			return nil, err
		}
	}

	// Verify crypto signatures for each input and reject the transaction
	// if any don't verify.
	err = blockchain.ValidateTransactionScripts(tx, utxoView,
		txscript.StandardVerifyFlags, mp.cfg.SigCache,
		mp.cfg.HashCache)
	if err != nil {
		if cerr, ok := err.(blockchain.RuleError); ok {
			return nil, chainRuleError(cerr)
		}
		return nil, err
	}

	result := &MempoolAcceptResult{
		TxFee:      btcutil.Amount(txFee),
		TxSize:     txSize,
		Conflicts:  conflicts,
		utxoView:   utxoView,
		bestHeight: bestHeight,
	}

	return result, nil
}

// validateSegWitDeployment checks that when a transaction has witness data,
// segwit must be active.
func (mp *TxPool) validateSegWitDeployment(tx *btcutil.Tx) error {
	// Exit early if this transaction doesn't have witness data.
	if !tx.MsgTx().HasWitness() {
		return nil
	}

	// If a transaction has witness data, and segwit isn't active yet, then
	// we won't accept it into the mempool as it can't be mined yet.
	segwitActive, err := mp.cfg.IsDeploymentActive(
		chaincfg.DeploymentSegwit,
	)
	if err != nil {
		return err
	}

	// Exit early if segwit is active.
	if segwitActive {
		return nil
	}

	simnetHint := ""
	if mp.cfg.ChainParams.Net == wire.SimNet {
		bestHeight := mp.cfg.BestHeight()
		simnetHint = fmt.Sprintf(" (The threshold for segwit "+
			"activation is 300 blocks on simnet, current best "+
			"height is %d)", bestHeight)
	}
	str := fmt.Sprintf("transaction %v has witness data, "+
		"but segwit isn't active yet%s", tx.Hash(), simnetHint)

	return txRuleError(wire.RejectNonstandard, str)
}

// validateStandardness checks the transaction passes both transaction standard
// and input standard.
func (mp *TxPool) validateStandardness(tx *btcutil.Tx, nextBlockHeight int32,
	medianTimePast time.Time, utxoView *blockchain.UtxoViewpoint) error {

	// Exit early if we accept non-standard transactions.
	//
	// NOTE: if you modify this code to accept non-standard transactions,
	// you should add code here to check that the transaction does a
	// reasonable number of ECDSA signature verifications.
	if mp.cfg.Policy.AcceptNonStd {
		return nil
	}

	// Check the transaction standard.
	err := CheckTransactionStandard(
		tx, nextBlockHeight, medianTimePast,
		mp.cfg.Policy.MinRelayTxFee, mp.cfg.Policy.MaxTxVersion,
	)
	if err != nil {
		// Attempt to extract a reject code from the error so it can be
		// retained. When not possible, fall back to a non standard
		// error.
		rejectCode, found := extractRejectCode(err)
		if !found {
			rejectCode = wire.RejectNonstandard
		}
		str := fmt.Sprintf("transaction %v is not standard: %v",
			tx.Hash(), err)

		return txRuleError(rejectCode, str)
	}

	// Check the inputs standard.
	err = checkInputsStandard(tx, utxoView)
	if err != nil {
		// Attempt to extract a reject code from the error so it can be
		// retained. When not possible, fall back to a non-standard
		// error.
		rejectCode, found := extractRejectCode(err)
		if !found {
			rejectCode = wire.RejectNonstandard
		}
		str := fmt.Sprintf("transaction %v has a non-standard "+
			"input: %v", tx.Hash(), err)

		return txRuleError(rejectCode, str)
	}

	return nil
}

// validateSigCost checks the cost to run the signature operations to make sure
// the number of signatures are sane.
func (mp *TxPool) validateSigCost(tx *btcutil.Tx,
	utxoView *blockchain.UtxoViewpoint) error {

	// Since the coinbase address itself can contain signature operations,
	// the maximum allowed signature operations per transaction is less
	// than the maximum allowed signature operations per block.
	//
	// TODO(roasbeef): last bool should be conditional on segwit activation
	sigOpCost, err := blockchain.GetSigOpCost(
		tx, false, utxoView, true, true,
	)
	if err != nil {
		if cerr, ok := err.(blockchain.RuleError); ok {
			return chainRuleError(cerr)
		}

		return err
	}

	// Exit early if the sig cost is under limit.
	if sigOpCost <= mp.cfg.Policy.MaxSigOpCostPerTx {
		return nil
	}

	str := fmt.Sprintf("transaction %v sigop cost is too high: %d > %d",
		tx.Hash(), sigOpCost, mp.cfg.Policy.MaxSigOpCostPerTx)

	return txRuleError(wire.RejectNonstandard, str)
}

// validateRelayFeeMet checks that the min relay fee is covered by this
// transaction.
func (mp *TxPool) validateRelayFeeMet(tx *btcutil.Tx, txFee, txSize int64,
	utxoView *blockchain.UtxoViewpoint, nextBlockHeight int32,
	isNew, rateLimit bool) error {

	txHash := tx.Hash()

	// Most miners allow a free transaction area in blocks they mine to go
	// alongside the area used for high-priority transactions as well as
	// transactions with fees. A transaction size of up to 1000 bytes is
	// considered safe to go into this section. Further, the minimum fee
	// calculated below on its own would encourage several small
	// transactions to avoid fees rather than one single larger transaction
	// which is more desirable. Therefore, as long as the size of the
	// transaction does not exceed 1000 less than the reserved space for
	// high-priority transactions, don't require a fee for it.
	minFee := calcMinRequiredTxRelayFee(txSize, mp.cfg.Policy.MinRelayTxFee)

	if txSize >= (DefaultBlockPrioritySize-1000) && txFee < minFee {
		str := fmt.Sprintf("transaction %v has %d fees which is under "+
			"the required amount of %d", txHash, txFee, minFee)

		return txRuleError(wire.RejectInsufficientFee, str)
	}

	// Exit early if the min relay fee is met.
	if txFee >= minFee {
		return nil
	}

	// Exit early if this is neither a new tx or rate limited.
	if !isNew && !rateLimit {
		return nil
	}

	// Require that free transactions have sufficient priority to be mined
	// in the next block. Transactions which are being added back to the
	// memory pool from blocks that have been disconnected during a reorg
	// are exempted.
	if isNew && !mp.cfg.Policy.DisableRelayPriority {
		currentPriority := mining.CalcPriority(
			tx.MsgTx(), utxoView, nextBlockHeight,
		)
		if currentPriority <= mining.MinHighPriority {
			str := fmt.Sprintf("transaction %v has insufficient "+
				"priority (%g <= %g)", txHash,
				currentPriority, mining.MinHighPriority)

			return txRuleError(wire.RejectInsufficientFee, str)
		}
	}

	// We can only end up here when the rateLimit is true. Free-to-relay
	// transactions are rate limited here to prevent penny-flooding with
	// tiny transactions as a form of attack.
	nowUnix := time.Now().Unix()

	// Decay passed data with an exponentially decaying ~10 minute window -
	// matches bitcoind handling.
	mp.pennyTotal *= math.Pow(
		1.0-1.0/600.0, float64(nowUnix-mp.lastPennyUnix),
	)
	mp.lastPennyUnix = nowUnix

	// Are we still over the limit?
	if mp.pennyTotal >= mp.cfg.Policy.FreeTxRelayLimit*10*1000 {
		str := fmt.Sprintf("transaction %v has been rejected "+
			"by the rate limiter due to low fees", txHash)

		return txRuleError(wire.RejectInsufficientFee, str)
	}

	oldTotal := mp.pennyTotal
	mp.pennyTotal += float64(txSize)
	log.Tracef("rate limit: curTotal %v, nextTotal: %v, limit %v",
		oldTotal, mp.pennyTotal, mp.cfg.Policy.FreeTxRelayLimit*10*1000)

	return nil
}
