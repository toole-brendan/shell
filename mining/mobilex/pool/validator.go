// Copyright (c) 2025 The Shell developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package pool

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/toole-brendan/shell/blockchain"
	"github.com/toole-brendan/shell/chaincfg"
	"github.com/toole-brendan/shell/chaincfg/chainhash"
	"github.com/toole-brendan/shell/mining/mobilex"
	"github.com/toole-brendan/shell/wire"
)

// Share represents a submitted mining share.
type Share struct {
	ClientID     uint64
	WorkerName   string
	JobID        string
	Extranonce2  string
	Ntime        string
	Nonce        string
	ThermalProof string
	Difficulty   float64
	SubmittedAt  time.Time
}

// ShareResult contains the validation result.
type ShareResult struct {
	Valid                  bool
	MeetsPoolDifficulty    bool
	MeetsNetworkDifficulty bool
	Block                  *wire.MsgBlock
	Error                  error
}

// ShareValidator validates mining shares.
type ShareValidator struct {
	cfg         *PoolConfig
	chainParams *chaincfg.Params

	// Duplicate detection
	recentShares map[string]time.Time
	shareExpiry  time.Duration
}

// NewShareValidator creates a new share validator.
func NewShareValidator(cfg *PoolConfig, chainParams *chaincfg.Params) *ShareValidator {
	return &ShareValidator{
		cfg:          cfg,
		chainParams:  chainParams,
		recentShares: make(map[string]time.Time),
		shareExpiry:  5 * time.Minute,
	}
}

// ValidateShare validates a submitted share.
func (sv *ShareValidator) ValidateShare(share *Share, job *MiningJob) (*ShareResult, error) {
	result := &ShareResult{}

	// Basic validation
	if err := sv.validateBasic(share); err != nil {
		result.Error = err
		return result, err
	}

	// Check for duplicate
	if sv.isDuplicate(share) {
		result.Error = errors.New("duplicate share")
		return result, result.Error
	}

	// Validate job ID
	if share.JobID != job.ID {
		result.Error = errors.New("invalid job ID")
		return result, result.Error
	}

	// Parse share components
	extranonce2, err := hex.DecodeString(share.Extranonce2)
	if err != nil || len(extranonce2) != 4 {
		result.Error = errors.New("invalid extranonce2")
		return result, result.Error
	}

	ntime, err := strconv.ParseInt(share.Ntime, 16, 64)
	if err != nil {
		result.Error = errors.New("invalid ntime")
		return result, result.Error
	}

	nonce, err := strconv.ParseUint(share.Nonce, 16, 32)
	if err != nil {
		result.Error = errors.New("invalid nonce")
		return result, result.Error
	}

	thermalProof, err := strconv.ParseUint(share.ThermalProof, 16, 64)
	if err != nil {
		result.Error = errors.New("invalid thermal proof")
		return result, result.Error
	}

	// Validate time
	if !sv.validateTime(ntime, job) {
		result.Error = errors.New("time out of range")
		return result, result.Error
	}

	// Build block header
	header, err := sv.buildBlockHeader(share, job, ntime, uint32(nonce), thermalProof)
	if err != nil {
		result.Error = err
		return result, err
	}

	// Validate thermal proof
	if sv.cfg.ThermalCompliance {
		if err := sv.validateThermalProof(header); err != nil {
			result.Error = fmt.Errorf("thermal validation failed: %w", err)
			return result, result.Error
		}
	}

	// Compute MobileX hash
	hash := sv.computeMobileXHash(header)

	// Check difficulty
	hashBig := mobilex.HashToBig(&hash)

	// Pool difficulty target
	poolTarget := sv.difficultyToTarget(share.Difficulty)
	if hashBig.Cmp(poolTarget) > 0 {
		result.Error = errors.New("share above target")
		return result, result.Error
	}
	result.MeetsPoolDifficulty = true

	// Network difficulty target
	networkTarget, _ := new(big.Int).SetString(job.Target, 16)
	if hashBig.Cmp(networkTarget) <= 0 {
		result.MeetsNetworkDifficulty = true

		// Build full block
		block, err := sv.buildFullBlock(header, job)
		if err != nil {
			result.Error = err
			return result, err
		}
		result.Block = block
	}

	// Mark share as valid
	result.Valid = true

	// Record share to prevent duplicates
	sv.recordShare(share)

	return result, nil
}

// validateBasic performs basic share validation.
func (sv *ShareValidator) validateBasic(share *Share) error {
	if share.WorkerName == "" {
		return errors.New("missing worker name")
	}
	if share.JobID == "" {
		return errors.New("missing job ID")
	}
	if len(share.Extranonce2) != 8 { // 4 bytes hex
		return errors.New("invalid extranonce2 length")
	}
	if len(share.Ntime) != 8 { // 4 bytes hex
		return errors.New("invalid ntime length")
	}
	if len(share.Nonce) != 8 { // 4 bytes hex
		return errors.New("invalid nonce length")
	}
	if len(share.ThermalProof) != 16 { // 8 bytes hex
		return errors.New("invalid thermal proof length")
	}
	return nil
}

// isDuplicate checks if a share is a duplicate.
func (sv *ShareValidator) isDuplicate(share *Share) bool {
	// Create unique key
	key := fmt.Sprintf("%s:%s:%s:%s:%s",
		share.WorkerName,
		share.JobID,
		share.Extranonce2,
		share.Ntime,
		share.Nonce,
	)

	// Check if exists
	if submitTime, exists := sv.recentShares[key]; exists {
		// Still within expiry window
		if time.Since(submitTime) < sv.shareExpiry {
			return true
		}
	}

	return false
}

// recordShare records a share to prevent duplicates.
func (sv *ShareValidator) recordShare(share *Share) {
	key := fmt.Sprintf("%s:%s:%s:%s:%s",
		share.WorkerName,
		share.JobID,
		share.Extranonce2,
		share.Ntime,
		share.Nonce,
	)

	sv.recentShares[key] = time.Now()

	// Clean old entries periodically
	if len(sv.recentShares) > 10000 {
		sv.cleanOldShares()
	}
}

// cleanOldShares removes expired share records.
func (sv *ShareValidator) cleanOldShares() {
	now := time.Now()
	for key, submitTime := range sv.recentShares {
		if now.Sub(submitTime) > sv.shareExpiry {
			delete(sv.recentShares, key)
		}
	}
}

// validateTime validates the share timestamp.
func (sv *ShareValidator) validateTime(ntime int64, job *MiningJob) bool {
	// Allow some flexibility in time
	now := time.Now().Unix()

	// Not too far in the past (10 minutes)
	if ntime < now-600 {
		return false
	}

	// Not too far in the future (2 minutes)
	if ntime > now+120 {
		return false
	}

	return true
}

// buildBlockHeader builds a block header from share data.
func (sv *ShareValidator) buildBlockHeader(share *Share, job *MiningJob, ntime int64, nonce uint32, thermalProof uint64) (*wire.BlockHeader, error) {
	// Parse previous block hash
	prevHash, err := chainhash.NewHashFromStr(job.PreviousHash)
	if err != nil {
		return nil, err
	}

	// TODO: Calculate merkle root from transactions
	// For now, use empty merkle root
	merkleRoot := chainhash.Hash{}

	// Build header
	header := &wire.BlockHeader{
		Version:      1, // Block version 1
		PrevBlock:    *prevHash,
		MerkleRoot:   merkleRoot,
		Timestamp:    time.Unix(ntime, 0),
		Bits:         sv.targetToBits(job.Target),
		Nonce:        nonce,
		ThermalProof: thermalProof,
	}

	return header, nil
}

// validateThermalProof validates the thermal compliance proof.
func (sv *ShareValidator) validateThermalProof(header *wire.BlockHeader) error {
	// Create thermal verifier
	thermal := mobilex.NewThermalVerification(2000, 5.0) // 2GHz base, 5% tolerance

	// Validate thermal proof
	if err := thermal.ValidateThermalProof(header); err != nil {
		return err
	}

	return nil
}

// computeMobileXHash computes the MobileX hash for validation.
func (sv *ShareValidator) computeMobileXHash(header *wire.BlockHeader) chainhash.Hash {
	// This would use the actual MobileX miner implementation
	// For now, return a simple hash

	headerBytes := sv.serializeHeader(header)
	return chainhash.DoubleHashH(headerBytes)
}

// serializeHeader serializes a block header.
func (sv *ShareValidator) serializeHeader(header *wire.BlockHeader) []byte {
	// Simplified serialization
	// In production, use proper wire protocol serialization
	buf := make([]byte, 88)
	// ... serialization logic ...
	return buf
}

// difficultyToTarget converts pool difficulty to target.
func (sv *ShareValidator) difficultyToTarget(difficulty float64) *big.Int {
	// Difficulty 1 = 0x00000000ffff0000000000000000000000000000000000000000000000000000
	// Higher difficulty = lower target

	diffOne := new(big.Int)
	diffOne.SetString("00000000ffff0000000000000000000000000000000000000000000000000000", 16)

	// Target = diffOne / difficulty
	target := new(big.Int).Div(diffOne, big.NewInt(int64(difficulty)))

	return target
}

// targetToBits converts a target hex string to compact bits.
func (sv *ShareValidator) targetToBits(targetHex string) uint32 {
	// Convert hex target to compact bits representation
	// This is a simplified version
	target, _ := new(big.Int).SetString(targetHex, 16)
	return blockchain.BigToCompact(target)
}

// buildFullBlock builds a complete block from a valid share.
func (sv *ShareValidator) buildFullBlock(header *wire.BlockHeader, job *MiningJob) (*wire.MsgBlock, error) {
	block := &wire.MsgBlock{
		Header: *header,
	}

	// TODO: Add transactions from job template
	// For now, just add coinbase

	return block, nil
}
