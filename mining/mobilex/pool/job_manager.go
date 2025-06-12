// Copyright (c) 2025 The Shell developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package pool

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/toole-brendan/shell/blockchain"
	"github.com/toole-brendan/shell/chaincfg"
	"github.com/toole-brendan/shell/chaincfg/chainhash"
	"github.com/toole-brendan/shell/wire"
)

// JobManager manages mining jobs for the pool.
type JobManager struct {
	cfg         *PoolConfig
	chainParams *chaincfg.Params

	// Current job
	currentJob   atomic.Value // *MiningJob
	jobIDCounter uint64

	// Template management
	currentTemplate *BlockTemplate
	templateMutex   sync.RWMutex

	// Update tracking
	lastUpdate     time.Time
	updateInterval time.Duration

	// Shutdown
	quit chan struct{}
}

// BlockTemplate represents a block template from the node.
type BlockTemplate struct {
	Height        int32
	PreviousBlock chainhash.Hash
	Transactions  []*wire.MsgTx
	CoinbaseValue int64
	Target        *big.Int
	MinTime       int64
	CurTime       int64
}

// NewJobManager creates a new job manager.
func NewJobManager(cfg *PoolConfig, chainParams *chaincfg.Params) *JobManager {
	jm := &JobManager{
		cfg:            cfg,
		chainParams:    chainParams,
		updateInterval: 30 * time.Second,
		quit:           make(chan struct{}),
	}

	// Initialize with empty job
	jm.currentJob.Store(&MiningJob{
		ID:               "0",
		Height:           0,
		PreviousHash:     "0000000000000000000000000000000000000000000000000000000000000000",
		Target:           "00000000ffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		MobileDifficulty: cfg.InitialDifficulty,
	})

	return jm
}

// Start begins the job management loop.
func (jm *JobManager) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	// Initial job update
	jm.updateJob()

	ticker := time.NewTicker(jm.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			jm.updateJob()
		case <-jm.quit:
			return
		}
	}
}

// Stop stops the job manager.
func (jm *JobManager) Stop() {
	close(jm.quit)
}

// GetCurrentJob returns the current mining job.
func (jm *JobManager) GetCurrentJob() *MiningJob {
	return jm.currentJob.Load().(*MiningJob)
}

// updateJob fetches a new block template and creates a job.
func (jm *JobManager) updateJob() {
	// Get block template from node
	template, err := jm.getBlockTemplate()
	if err != nil {
		// Log error but continue with existing job
		return
	}

	// Store template
	jm.templateMutex.Lock()
	jm.currentTemplate = template
	jm.templateMutex.Unlock()

	// Create new job
	jobID := atomic.AddUint64(&jm.jobIDCounter, 1)

	job := &MiningJob{
		ID:               fmt.Sprintf("%d", jobID),
		Height:           template.Height,
		PreviousHash:     template.PreviousBlock.String(),
		CoinbaseValue:    template.CoinbaseValue,
		Target:           targetToHex(template.Target),
		MobileDifficulty: jm.cfg.InitialDifficulty,

		// Mobile-specific fields
		ThermalTarget: 45.0, // 45°C target
		NPUWork:       generateNPUWork(template.Height),
		WorkSize: WorkSizeConfig{
			SearchSpace:   0x100000,        // 1M nonces default
			NPUIterations: 150,             // Every 150 hashes
			CacheSize:     2 * 1024 * 1024, // 2MB default
		},
	}

	// Store new job
	jm.currentJob.Store(job)
	jm.lastUpdate = time.Now()
}

// getBlockTemplate fetches a block template from the node.
func (jm *JobManager) getBlockTemplate() (*BlockTemplate, error) {
	// This would connect to the Shell node via RPC
	// For now, return a mock template

	// In production, this would use btcjson.NewGetBlockTemplateCmd
	// and make an RPC call to the node

	mockTemplate := &BlockTemplate{
		Height:        100000,
		PreviousBlock: chainhash.Hash{}, // Would be real previous block
		Transactions:  []*wire.MsgTx{},
		CoinbaseValue: 95 * 1e8, // 95 XSL
		Target:        blockchain.CompactToBig(0x1d00ffff),
		MinTime:       time.Now().Unix() - 600,
		CurTime:       time.Now().Unix(),
	}

	return mockTemplate, nil
}

// CreateCoinbase creates a coinbase transaction for a miner.
func (jm *JobManager) CreateCoinbase(extraNonce1, extraNonce2 string, minerAddress string) (*wire.MsgTx, error) {
	jm.templateMutex.RLock()
	template := jm.currentTemplate
	jm.templateMutex.RUnlock()

	if template == nil {
		return nil, fmt.Errorf("no block template available")
	}

	// Create coinbase transaction
	coinbaseTx := wire.NewMsgTx(wire.TxVersion)

	// Coinbase input
	prevOut := wire.NewOutPoint(&chainhash.Hash{}, wire.MaxPrevOutIndex)
	txIn := wire.NewTxIn(prevOut, nil, nil)

	// Build coinbase script
	// Height + ExtraNonce1 + ExtraNonce2 + Pool signature
	scriptSig := BuildCoinbaseScript(template.Height, extraNonce1, extraNonce2, "Shell Mobile Pool")
	txIn.SignatureScript = scriptSig

	coinbaseTx.AddTxIn(txIn)

	// Calculate pool fee
	poolFee := int64(float64(template.CoinbaseValue) * jm.cfg.PoolFeePercent / 100.0)
	minerReward := template.CoinbaseValue - poolFee

	// Miner output
	minerScript, err := AddrToScript(minerAddress)
	if err != nil {
		return nil, err
	}
	coinbaseTx.AddTxOut(wire.NewTxOut(minerReward, minerScript))

	// Pool fee output
	if poolFee > 0 && jm.cfg.PoolAddress != "" {
		poolScript, err := AddrToScript(jm.cfg.PoolAddress)
		if err != nil {
			return nil, err
		}
		coinbaseTx.AddTxOut(wire.NewTxOut(poolFee, poolScript))
	}

	return coinbaseTx, nil
}

// Helper functions

// targetToHex converts a big.Int target to hex string.
func targetToHex(target *big.Int) string {
	// Convert to 32-byte hex string
	bytes := target.Bytes()
	padded := make([]byte, 32)
	copy(padded[32-len(bytes):], bytes)
	return hex.EncodeToString(padded)
}

// generateNPUWork generates NPU work data for a given height.
func generateNPUWork(height int32) []byte {
	// Generate deterministic NPU work based on height
	// This would contain the neural network weights/parameters
	// For now, return placeholder data
	work := make([]byte, 1024) // 1KB of NPU parameters
	for i := range work {
		work[i] = byte((height + int32(i)) % 256)
	}
	return work
}

// BuildCoinbaseScript builds the coinbase script.
func BuildCoinbaseScript(height int32, extraNonce1, extraNonce2, poolSig string) []byte {
	// Simplified coinbase script building
	// In production, this would properly serialize all components

	script := make([]byte, 0, 100)

	// Add block height (BIP34)
	heightBytes := make([]byte, 4)
	heightBytes[0] = byte(height)
	heightBytes[1] = byte(height >> 8)
	heightBytes[2] = byte(height >> 16)
	heightBytes[3] = byte(height >> 24)
	script = append(script, byte(len(heightBytes)))
	script = append(script, heightBytes...)

	// Add extra nonces
	en1, _ := hex.DecodeString(extraNonce1)
	script = append(script, en1...)
	en2, _ := hex.DecodeString(extraNonce2)
	script = append(script, en2...)

	// Add pool signature
	script = append(script, byte(len(poolSig)))
	script = append(script, []byte(poolSig)...)

	return script
}

// AddrToScript converts an address string to a script.
func AddrToScript(addr string) ([]byte, error) {
	// This would decode the address and create the appropriate script
	// For now, return a placeholder P2PKH script
	return []byte{0x76, 0xa9, 0x14}, nil // OP_DUP OP_HASH160 <push 20>
}
