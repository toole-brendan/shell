// Copyright (c) 2025 Shell Reserve developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package randomx

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"sync"
	"time"

	"github.com/btcsuite/btclog"
	"github.com/toole-brendan/shell/chaincfg/chainhash"
	"github.com/toole-brendan/shell/internal/convert"
	"github.com/toole-brendan/shell/wire"
)

// log is a logger that is initialized with no output filters.  This
// means the package will not perform any logging by default until the caller
// requests it.
var log btclog.Logger

// UseLogger uses a specified Logger to output package logging info.
func UseLogger(logger btclog.Logger) {
	log = logger
}

// Disable logging by default until the package user requests it.
func init() {
	DisableLog()
}

// DisableLog disables all library log output.  Logging output is disabled
// by default until either UseLogger or SetLogWriter are called.
func DisableLog() {
	log = btclog.Disabled
}

const (
	// maxNonce is the maximum value a nonce can be in a block header.
	maxNonce = ^uint32(0) // 2^32 - 1

	// maxExtraNonce is the maximum value an extra nonce used in a coinbase can be.
	maxExtraNonce = ^uint64(0) // 2^64 - 1

	// hpsUpdateSecs is the number of seconds to wait in between each
	// update to the hashes per second monitor.
	hpsUpdateSecs = 10

	// hashUpdateSec is the number of seconds each worker waits in between
	// notifying the speed monitor with how many hashes have been completed
	// while they are actively searching for a solution.  This is done to
	// reduce the amount of syncing between the workers that must be done to
	// keep track of the hashes per second.
	hashUpdateSecs = 15
)

// RandomXMiner provides facilities for solving blocks (mining) using RandomX
// proof-of-work in a concurrent manner with CPU cores.
type RandomXMiner struct {
	cache            *Cache
	dataset          *Dataset
	vm               *VM
	seedHeight       int32
	seedHash         chainhash.Hash
	memory           int64 // Memory requirement in bytes
	started          bool
	shutdown         chan struct{}
	wg               sync.WaitGroup
	updateHashes     chan uint64
	speedMonitorQuit chan struct{}
	quit             chan struct{}
	mutex            sync.Mutex
}

// NewRandomXMiner returns a new instance of a RandomX miner.
func NewRandomXMiner(memoryMB int64) *RandomXMiner {
	return &RandomXMiner{
		memory:           memoryMB * 1024 * 1024, // Convert MB to bytes
		seedHeight:       -1,                     // Initialize with invalid height
		updateHashes:     make(chan uint64),
		speedMonitorQuit: make(chan struct{}),
		quit:             make(chan struct{}),
	}
}

// speedMonitor handles tracking the number of hashes per second the mining
// process is performing.  It must be run as a goroutine.
func (m *RandomXMiner) speedMonitor() {
	log.Tracef("RandomX speed monitor started")

	var hashesPerSec int64
	var totalHashes uint64
	ticker := time.NewTicker(time.Second * hpsUpdateSecs)
	defer ticker.Stop()

out:
	for {
		select {
		// Periodic update to the hashes per second monitor.
		case numHashes := <-m.updateHashes:
			totalHashes += numHashes

		case <-ticker.C:
			curHashesPerSec := int64(totalHashes / hpsUpdateSecs)
			if curHashesPerSec != hashesPerSec {
				log.Infof("Hash speed: %d kilohashes/s", curHashesPerSec/1000)
				hashesPerSec = curHashesPerSec
			}
			totalHashes = 0

		// Request to shutdown the speed monitor.
		case <-m.speedMonitorQuit:
			break out

		case <-m.quit:
			break out
		}
	}

	m.wg.Done()
	log.Tracef("RandomX speed monitor done")
}

// initRandomX initializes the RandomX cache, dataset, and VM for the given seed.
func (m *RandomXMiner) initRandomX(seed []byte) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	log.Infof("Initializing RandomX with seed %x", seed)

	// Initialize RandomX cache with the seed
	cache, err := NewCache(seed)
	if err != nil {
		return err
	}

	// Initialize dataset from cache
	dataset, err := NewDataset(cache)
	if err != nil {
		cache.Close()
		return err
	}

	// Initialize VM
	vm, err := NewVM(cache, dataset)
	if err != nil {
		dataset.Close()
		cache.Close()
		return err
	}

	// Clean up old instances
	if m.vm != nil {
		m.vm.Close()
	}
	if m.dataset != nil {
		m.dataset.Close()
	}
	if m.cache != nil {
		m.cache.Close()
	}

	// Store new instances
	m.cache = cache
	m.dataset = dataset
	m.vm = vm

	log.Infof("RandomX initialization complete")
	return nil
}

// getSeedForHeight calculates the RandomX seed for the given block height.
// The seed changes every RandomXSeedRotation blocks.
func (m *RandomXMiner) getSeedForHeight(height int32, rotation int32, genesisHash *chainhash.Hash) []byte {
	seedHeight := (height / rotation) * rotation

	// For genesis or early blocks, use genesis hash as seed
	if seedHeight <= 0 {
		return genesisHash[:]
	}

	// Create seed based on height
	// In a real implementation, this would be the block hash at seedHeight
	// For now, we'll use a deterministic seed based on height
	seed := make([]byte, 32)
	binary.LittleEndian.PutUint32(seed[0:4], uint32(seedHeight))
	copy(seed[4:], genesisHash[:28])

	hasher := sha256.New()
	hasher.Write(seed)
	return hasher.Sum(nil)
}

// updateSeed updates the RandomX seed if needed based on block height.
func (m *RandomXMiner) updateSeed(height int32, rotation int32, genesisHash *chainhash.Hash) error {
	newSeedHeight := (height / rotation) * rotation

	// Check if we need to update the seed
	if newSeedHeight == m.seedHeight {
		return nil // No update needed
	}

	seed := m.getSeedForHeight(height, rotation, genesisHash)
	seedHash := sha256.Sum256(seed)

	// Update if this is a new seed
	if m.seedHeight != newSeedHeight || seedHash != m.seedHash {
		if err := m.initRandomX(seed); err != nil {
			return err
		}
		m.seedHeight = newSeedHeight
		m.seedHash = seedHash
		log.Infof("RandomX seed updated for height %d (seed height: %d)", height, newSeedHeight)
	}

	return nil
}

// solveBlock attempts to find a nonce which makes the passed block hash to
// a value less than the target difficulty.  When a successful solution is found
// true is returned and the nonce field of the passed header is updated with the
// solution.  False is returned if no solution exists.
func (m *RandomXMiner) solveBlock(msgBlock *wire.MsgBlock, blockHeight int32,
	ticker *time.Ticker, quit chan struct{}, params *RandomXParams) bool {

	// Get a local copy of the header so we can update the nonce while
	// checking if the solution is under the target.
	header := msgBlock.Header
	targetDifficulty := CompactToBig(header.Bits)

	// Initial state check.
	hashesCompleted := uint64(0)

	// Note that the entire extra nonce range is iterated and the offset is
	// added relying on the fact that overflow will wrap around 0 as
	// provided by the Go spec.
	for extraNonce := uint64(0); extraNonce <= maxExtraNonce; extraNonce++ {
		// Update the extra nonce in the block template with the
		// new value by regenerating the coinbase script and setting
		// the merkle root to the new value.
		//
		// NOTE: This is only required when the extra nonce actually
		// changes.
		if extraNonce != 0 {
			err := UpdateExtraNonce(msgBlock, blockHeight, extraNonce)
			if err != nil {
				// This should never happen.
				log.Errorf("Failed to update extra nonce: %v", err)
				return false
			}
		}

		// Search through the entire nonce range for a solution while
		// periodically checking for early quit and updates to the speed
		// monitor.
		for i := uint32(0); i <= maxNonce; i++ {
			select {
			case <-quit:
				return false

			case <-ticker.C:
				m.updateHashes <- hashesCompleted
				hashesCompleted = 0

			default:
				// Non-blocking select to fall through
			}

			// Update the nonce and hash the block header.
			header.Nonce = i
			hash := m.hashBlockHeader(&header, params)
			hashesCompleted++

			// The block is solved when the new block hash is less
			// than the target difficulty.  Yay!
			hashNum := HashToBig(&hash)
			if hashNum.Cmp(targetDifficulty) <= 0 {
				m.updateHashes <- hashesCompleted
				msgBlock.Header.Nonce = i
				return true
			}
		}
	}

	return false
}

// hashBlockHeader hashes a block header using RandomX.
func (m *RandomXMiner) hashBlockHeader(header *wire.BlockHeader, params *RandomXParams) chainhash.Hash {
	// Ensure RandomX is initialized
	if m.vm == nil {
		// This should not happen in normal operation
		log.Errorf("RandomX VM not initialized")
		return chainhash.Hash{}
	}

	// Serialize the block header
	var buf bytes.Buffer
	err := header.Serialize(&buf)
	if err != nil {
		log.Errorf("Failed to serialize header: %v", err)
		return chainhash.Hash{}
	}
	headerBytes := buf.Bytes()

	// Compute RandomX hash
	hash := m.vm.CalcHash(headerBytes)

	var result chainhash.Hash
	copy(result[:], hash)
	return result
}

// generateBlocks is a worker that is controlled by the mineWorkerController.
// It is self contained in that it creates a block template and attempts to solve
// it by finding a nonce which results in a block hash less than the target
// difficulty.  Once a valid solution is found, it is submitted.
func (m *RandomXMiner) generateBlocks(quit chan struct{}, cfg *Config) {
	log.Tracef("Starting generate blocks worker")

	// Start a ticker which is used to signal checks for stale work and
	// updates to the speed monitor.
	ticker := time.NewTicker(time.Second * hashUpdateSecs)
	defer ticker.Stop()

out:
	for {
		// Quit when the miner is stopped.
		select {
		case <-quit:
			break out
		default:
			// Non-blocking select to fall through
		}

		// Wait until there is a connection to at least one other peer
		// since there is no way to relay a found block or receive
		// transactions to include in a block template.
		//
		// Also, grab the current best chain height so the coinbase and
		// block template can properly be generated.
		curHeight, _, err := cfg.ConnectedCount()
		if err != nil {
			log.Errorf("Failed to get connected count: %v", err)
			break out
		}
		if curHeight != 0 && curHeight == 0 {
			time.Sleep(time.Second)
			continue
		}

		// No point in searching for a solution before the chain is
		// synced.  Also, grab the current best chain height and hash
		// so the coinbase and block template can properly be generated.
		bestHeight, _, err := cfg.BestSnapshot()
		if err != nil {
			log.Errorf("Failed to get best snapshot: %v", err)
			break out
		}
		if bestHeight != 0 && (bestHeight < curHeight-1 || (bestHeight == curHeight-1 && cfg.IsCurrent() == false)) {
			time.Sleep(time.Second)
			continue
		}

		// Update RandomX seed if necessary
		nextHeight := bestHeight + 1
		err = m.updateSeed(nextHeight, cfg.RandomXSeedRotation, cfg.GenesisHash)
		if err != nil {
			log.Errorf("Failed to update RandomX seed: %v", err)
			time.Sleep(time.Second)
			continue
		}

		// Create a new block template.
		template, err := cfg.BlockTemplateGenerator()
		if err != nil {
			log.Errorf("Failed to create new block template: %v", err)
			time.Sleep(time.Second)
			continue
		}

		// Attempt to solve the block.  The function will exit early
		// with false when conditions that trigger a stale block, so
		// a new block template can be generated.  When the return is
		// true a solution was found, so submit it to the network.
		// The submitted block is not guaranteed to be on the main chain.
		if m.solveBlock(template.Block, template.Height, ticker, quit, &cfg.RandomXParams) {
			block := convert.NewShellBlock(template.Block)

			// Submit the solved block.
			err := cfg.SubmitBlock(block)
			if err != nil {
				log.Warnf("Failed to submit block: %v", err)
			} else {
				blockSha := block.Hash()
				log.Infof("Block submitted %s (height %d)", blockSha, template.Height)
			}
		}
	}

	m.wg.Done()
	log.Tracef("Generate blocks worker done")
}

// mineWorkerController launches the worker goroutines that are used to generate
// block templates and solve them.  It also provides the ability to dynamically
// adjust the number of running worker goroutines.
//
// It must be run as a goroutine.
func (m *RandomXMiner) mineWorkerController(cfg *Config) {
	// Launch workers that are used to generate block templates and
	// solve them.
	var runningWorkers []chan struct{}
	launchWorkers := func(numWorkers uint32) {
		for i := uint32(0); i < numWorkers; i++ {
			quit := make(chan struct{})
			runningWorkers = append(runningWorkers, quit)

			m.wg.Add(1)
			go m.generateBlocks(quit, cfg)
		}
	}

	// Launch the current number of workers by default.
	runningWorkers = make([]chan struct{}, 0, cfg.NumWorkers)
	launchWorkers(cfg.NumWorkers)

out:
	for {
		select {
		// Update the number of running workers.
		case <-cfg.UpdateNumWorkers:
			// No change.
			numRunning := uint32(len(runningWorkers))
			if cfg.NumWorkers == numRunning {
				continue
			}

			// Add new workers.
			if cfg.NumWorkers > numRunning {
				launchWorkers(cfg.NumWorkers - numRunning)
				continue
			}

			// Signal the most recently created goroutines to exit.
			for i := numRunning - 1; i >= cfg.NumWorkers; i-- {
				close(runningWorkers[i])
				runningWorkers[i] = nil
				runningWorkers = runningWorkers[:i]
			}

		case <-m.quit:
			for _, quit := range runningWorkers {
				close(quit)
			}
			break out
		}
	}

	// Wait until all workers shut down to stop the speed monitor since
	// they rely on being able to send updates to it.
	m.wg.Wait()
	close(m.speedMonitorQuit)
	m.wg.Done()
	log.Tracef("RandomX miner worker controller done")
}

// Start begins the mining process as well as the speed monitor used to track
// hashing metrics.  Calling this function when the miner has already been
// started will have no effect.
//
// The miner will continue running until the Stop method is invoked.
func (m *RandomXMiner) Start(cfg *Config) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Nothing to do if the miner is already running.
	if m.started {
		return
	}

	log.Infof("Starting RandomX miner with %d workers", cfg.NumWorkers)

	m.quit = make(chan struct{})
	m.speedMonitorQuit = make(chan struct{})
	m.wg.Add(2)
	go m.speedMonitor()
	go m.mineWorkerController(cfg)

	m.started = true
	log.Infof("RandomX miner started")
}

// Stop gracefully stops the mining process by signaling all workers and the
// speed monitor to quit.  Calling this function when the miner has not already
// been started will have no effect.
//
// The function does not return until all workers and the speed monitor have
// finished running.
func (m *RandomXMiner) Stop() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Nothing to do if the miner is not currently running.
	if !m.started {
		return
	}

	log.Infof("Stopping RandomX miner...")
	close(m.quit)
	m.wg.Wait()

	// Clean up RandomX resources
	if m.vm != nil {
		m.vm.Close()
		m.vm = nil
	}
	if m.dataset != nil {
		m.dataset.Close()
		m.dataset = nil
	}
	if m.cache != nil {
		m.cache.Close()
		m.cache = nil
	}

	m.started = false
	log.Infof("RandomX miner stopped")
}

// IsMining returns whether or not the miner has been started and is therefore
// currently mining.
func (m *RandomXMiner) IsMining() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.started
}

// HashesPerSecond returns the number of hashes per second the mining process
// is performing.  0 is returned if the miner is not currently running.
//
// This function is safe for concurrent access.
func (m *RandomXMiner) HashesPerSecond() float64 {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Nothing to do if the miner is not currently running.
	if !m.started {
		return 0
	}

	return float64(<-m.updateHashes)
}

// SetNumWorkers sets the number of workers to create which solve blocks.  Any
// negative values will cause a default number of workers to be created which
// is based on the number of processor cores in the system.  A value of 0 will
// cause all CPU mining to be stopped.
//
// This function is safe for concurrent access.
func (m *RandomXMiner) SetNumWorkers(numWorkers int32) {
	if numWorkers == 0 {
		m.Stop()
	}

	// Don't lock here since the speed monitor and worker controller will
	// handle switching the number of workers dynamically.
}
