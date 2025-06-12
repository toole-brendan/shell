// Copyright (c) 2025 The Shell developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mining

import (
	"errors"
	"fmt"

	"github.com/toole-brendan/shell/chaincfg"
	"github.com/toole-brendan/shell/chaincfg/chainhash"
	"github.com/toole-brendan/shell/mining/randomx"
	"github.com/toole-brendan/shell/wire"
)

// MiningPolicy manages the mining algorithm policy and validation for Shell Reserve.
// It supports both RandomX (traditional) and MobileX (mobile-optimized) algorithms.
type MiningPolicy struct {
	chainParams    *chaincfg.Params
	randomXEnabled bool
	mobileXEnabled bool
	dualMining     bool
}

// AlgorithmType represents the detected mining algorithm used for a block
type AlgorithmType int

const (
	// AlgorithmUnknown indicates the algorithm could not be determined
	AlgorithmUnknown AlgorithmType = iota
	// AlgorithmRandomX indicates RandomX was used
	AlgorithmRandomX
	// AlgorithmMobileX indicates MobileX was used
	AlgorithmMobileX
)

// NewMiningPolicy creates a new mining policy manager
func NewMiningPolicy(chainParams *chaincfg.Params) *MiningPolicy {
	return &MiningPolicy{
		chainParams:    chainParams,
		randomXEnabled: true, // RandomX always enabled
		mobileXEnabled: chainParams.MobileXEnabled,
		dualMining:     chainParams.MobileXEnabled, // Dual mining when MobileX enabled
	}
}

// DetectAlgorithm determines which algorithm was used to mine a block
func (mp *MiningPolicy) DetectAlgorithm(blockHeader *wire.BlockHeader) AlgorithmType {
	// Check for MobileX indicators
	if mp.isMobileXBlock(blockHeader) {
		return AlgorithmMobileX
	}

	// Default to RandomX for compatibility
	return AlgorithmRandomX
}

// isMobileXBlock determines if a block was mined using MobileX
func (mp *MiningPolicy) isMobileXBlock(blockHeader *wire.BlockHeader) bool {
	// MobileX blocks have thermal proof field
	if blockHeader.ThermalProof != 0 {
		return true
	}

	// Could add additional MobileX detection heuristics here:
	// - Specific nonce patterns
	// - Block timing characteristics
	// - Other mobile-specific signatures

	return false
}

// ValidateBlockAlgorithm validates that a block was mined with an acceptable algorithm
func (mp *MiningPolicy) ValidateBlockAlgorithm(blockHeader *wire.BlockHeader, blockHeight int32) error {
	algorithm := mp.DetectAlgorithm(blockHeader)

	switch algorithm {
	case AlgorithmRandomX:
		if !mp.randomXEnabled {
			return errors.New("RandomX mining is disabled")
		}
		return mp.validateRandomXBlock(blockHeader, blockHeight)

	case AlgorithmMobileX:
		if !mp.mobileXEnabled {
			return errors.New("MobileX mining is not yet activated")
		}
		return mp.validateMobileXBlock(blockHeader, blockHeight)

	default:
		return fmt.Errorf("unknown or invalid mining algorithm for block at height %d", blockHeight)
	}
}

// validateRandomXBlock validates a RandomX-mined block
func (mp *MiningPolicy) validateRandomXBlock(blockHeader *wire.BlockHeader, blockHeight int32) error {
	// Standard RandomX validation
	// This would integrate with the existing RandomX validation logic

	// Check difficulty meets target
	hash := mp.computeRandomXHash(blockHeader)
	hashBig := randomx.HashToBig(&hash)
	target := randomx.CompactToBig(blockHeader.Bits)

	if hashBig.Cmp(target) > 0 {
		return fmt.Errorf("block hash %s is higher than target %s", hashBig.Text(16), target.Text(16))
	}

	return nil
}

// validateMobileXBlock validates a MobileX-mined block
func (mp *MiningPolicy) validateMobileXBlock(blockHeader *wire.BlockHeader, blockHeight int32) error {
	// MobileX-specific validation

	// 1. Validate thermal proof is present
	if blockHeader.ThermalProof == 0 {
		return errors.New("MobileX block missing thermal proof")
	}

	// 2. Validate thermal proof integrity
	if err := mp.validateThermalProof(blockHeader); err != nil {
		return fmt.Errorf("thermal proof validation failed: %w", err)
	}

	// 3. Check MobileX hash meets difficulty target
	hash := mp.computeMobileXHash(blockHeader)
	hashBig := randomx.HashToBig(&hash)
	target := randomx.CompactToBig(blockHeader.Bits)

	if hashBig.Cmp(target) > 0 {
		return fmt.Errorf("MobileX block hash %s is higher than target %s", hashBig.Text(16), target.Text(16))
	}

	return nil
}

// validateThermalProof validates the thermal compliance proof
func (mp *MiningPolicy) validateThermalProof(blockHeader *wire.BlockHeader) error {
	// Basic thermal proof validation
	// In a real implementation, this would:
	// 1. Verify the proof was generated correctly
	// 2. Check thermal compliance within tolerance
	// 3. Validate against block header contents

	thermalProof := blockHeader.ThermalProof

	// Thermal proof should be non-zero and within reasonable bounds
	if thermalProof == 0 {
		return errors.New("thermal proof cannot be zero")
	}

	// Example validation: thermal proof should indicate reasonable thermal state
	// This is a simplified check - real implementation would be more sophisticated
	if thermalProof > 0xFFFFFFFFFFFF {
		return errors.New("thermal proof value out of range")
	}

	// TODO: Implement proper thermal proof cryptographic validation
	// This would involve:
	// - Verifying PMU counter consistency
	// - Checking thermal compliance within tolerance (±5%)
	// - Validating proof against header hash

	return nil
}

// computeRandomXHash computes the RandomX hash for a block header
func (mp *MiningPolicy) computeRandomXHash(blockHeader *wire.BlockHeader) chainhash.Hash {
	// This would integrate with the actual RandomX implementation
	// For now, return a placeholder
	// In real implementation: return randomx.ComputeHash(blockHeader)

	var hash chainhash.Hash
	// Placeholder hash computation
	return hash
}

// computeMobileXHash computes the MobileX hash for a block header
func (mp *MiningPolicy) computeMobileXHash(blockHeader *wire.BlockHeader) chainhash.Hash {
	// This would integrate with the MobileX implementation
	// For now, return a placeholder
	// In real implementation: return mobilex.ComputeHash(blockHeader)

	var hash chainhash.Hash
	// Placeholder hash computation
	return hash
}

// GetSupportedAlgorithms returns the currently supported mining algorithms
func (mp *MiningPolicy) GetSupportedAlgorithms() []AlgorithmType {
	var algorithms []AlgorithmType

	if mp.randomXEnabled {
		algorithms = append(algorithms, AlgorithmRandomX)
	}

	if mp.mobileXEnabled {
		algorithms = append(algorithms, AlgorithmMobileX)
	}

	return algorithms
}

// IsDualMiningEnabled returns whether dual-algorithm mining is enabled
func (mp *MiningPolicy) IsDualMiningEnabled() bool {
	return mp.dualMining && mp.randomXEnabled && mp.mobileXEnabled
}

// GetAlgorithmRatio returns the target ratio of RandomX to MobileX blocks
func (mp *MiningPolicy) GetAlgorithmRatio() (randomXPercent, mobileXPercent float64) {
	if !mp.IsDualMiningEnabled() {
		if mp.randomXEnabled {
			return 100.0, 0.0
		}
		return 0.0, 100.0
	}

	// Default 50/50 split during dual mining period
	// This could be configurable or dynamic based on network conditions
	return 50.0, 50.0
}

// UpdateMobileXStatus updates the MobileX activation status
func (mp *MiningPolicy) UpdateMobileXStatus(enabled bool) {
	mp.mobileXEnabled = enabled
	mp.dualMining = enabled && mp.randomXEnabled
}

// GetPolicyInfo returns information about the current mining policy
func (mp *MiningPolicy) GetPolicyInfo() PolicyInfo {
	randomXPercent, mobileXPercent := mp.GetAlgorithmRatio()

	return PolicyInfo{
		RandomXEnabled:      mp.randomXEnabled,
		MobileXEnabled:      mp.mobileXEnabled,
		DualMiningEnabled:   mp.IsDualMiningEnabled(),
		RandomXPercent:      randomXPercent,
		MobileXPercent:      mobileXPercent,
		SupportedAlgorithms: mp.GetSupportedAlgorithms(),
	}
}

// PolicyInfo contains information about the current mining policy
type PolicyInfo struct {
	RandomXEnabled      bool
	MobileXEnabled      bool
	DualMiningEnabled   bool
	RandomXPercent      float64
	MobileXPercent      float64
	SupportedAlgorithms []AlgorithmType
}

// String returns a human-readable description of the algorithm type
func (at AlgorithmType) String() string {
	switch at {
	case AlgorithmRandomX:
		return "RandomX"
	case AlgorithmMobileX:
		return "MobileX"
	default:
		return "Unknown"
	}
}
