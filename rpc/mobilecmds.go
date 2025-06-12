// Copyright (c) 2025 The Shell developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/toole-brendan/shell/blockchain"
	"github.com/toole-brendan/shell/btcjson"
	"github.com/toole-brendan/shell/chaincfg/chainhash"
	"github.com/toole-brendan/shell/mining/mobilex"
	"github.com/toole-brendan/shell/wire"
)

// Mobile mining state tracking
type mobileMinersState struct {
	sync.RWMutex
	activeMinerCount   int64
	mobileHashrate     float64
	deviceBreakdown    map[string]int64
	thermalViolations  int64
	blocksFoundMobile  int64
	averageTemperature float64
	npuUtilization     float64
}

var (
	// Global mobile mining state
	mobileState = &mobileMinersState{
		deviceBreakdown: make(map[string]int64),
	}
)

// Mobile mining RPC handlers follow the same pattern as regular handlers
// They would be registered in the main rpcserver initialization

// handleGetMobileBlockTemplate implements the getmobileblocktemplate command.
// This is a simplified implementation for mobile miners.
func handleGetMobileBlockTemplate(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetMobileBlockTemplateCmd)
	request := c.Request

	// Get best block state
	best := s.cfg.Chain.BestSnapshot()

	// Create a simplified template response
	// In production, this would integrate with the existing getblocktemplate
	reply := btcjson.GetMobileBlockTemplateResult{
		// Standard fields
		Bits:         fmt.Sprintf("%08x", best.Bits),
		CurTime:      time.Now().Unix(),
		Height:       int64(best.Height + 1),
		PreviousHash: best.Hash.String(),
		Target:       fmt.Sprintf("%064x", blockchain.CompactToBig(best.Bits)),

		// Mobile-specific fields
		MobileTarget:    calculateMobileTarget(best.Bits, request.DeviceInfo),
		NPUWork:         generateNPUWorkForHeight(best.Height + 1),
		ThermalTarget:   45.0, // Default thermal target
		WorkSize:        determineWorkSize(request.DeviceInfo),
		DeviceOptimized: request.DeviceInfo != nil,
	}

	// For now, include empty transaction list
	reply.Transactions = []string{}

	return &reply, nil
}

// handleGetMobileMiningInfo implements the getmobilemininginfo command.
func handleGetMobileMiningInfo(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	// Get current blockchain state
	best := s.cfg.Chain.BestSnapshot()

	// Get mobile-specific stats
	mobileState.RLock()
	mobileHashrate := mobileState.mobileHashrate
	activeMobileMiners := mobileState.activeMinerCount
	mobileState.RUnlock()

	// Calculate difficulty ratio
	difficulty := getDifficultyRatio(best.Bits, s.cfg.ChainParams)
	mobileDifficulty := difficulty * 0.1 // 10% of main difficulty

	result := btcjson.GetMobileMiningInfoResult{
		Blocks:             int64(best.Height),
		CurrentBlockSize:   uint64(best.BlockSize),
		CurrentBlockTx:     uint64(best.NumTxns),
		Difficulty:         difficulty,
		MobileDifficulty:   mobileDifficulty,
		Errors:             "",
		Generate:           s.cfg.CPUMiner.IsMining(),
		HashesPerSec:       int64(s.cfg.CPUMiner.HashesPerSecond()),
		MobileMinersActive: activeMobileMiners,
		MobileHashrate:     mobileHashrate,
		NetworkHashPerSec:  0, // TODO: Calculate from recent blocks
		PooledTx:           uint64(s.cfg.TxMemPool.Count()),
		TestNet:            cfg.TestNet3,
		ThermalCompliance:  95.0, // Placeholder
	}

	return &result, nil
}

// handleSubmitMobileBlock implements the submitmobileblock command.
func handleSubmitMobileBlock(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.SubmitMobileBlockCmd)

	// Decode the serialized block
	blockBytes, err := hex.DecodeString(c.HexBlock)
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidParameter,
			Message: fmt.Sprintf("invalid hex block: %v", err),
		}
	}

	// Deserialize the block
	block, err := btcutil.NewBlockFromBytes(blockBytes)
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCDeserialization,
			Message: fmt.Sprintf("block decode failed: %v", err),
		}
	}

	// Validate thermal proof if provided
	if c.ThermalProof != nil {
		thermal := mobilex.NewThermalVerification(2000, 5.0)
		thermal.UpdateTemperature(c.ThermalProof.Temperature)

		// Get the header directly from Shell's wire.MsgBlock
		msgBlock := &wire.MsgBlock{}
		err := msgBlock.Deserialize(bytes.NewReader(blockBytes))
		if err != nil {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCDeserialization,
				Message: fmt.Sprintf("block deserialize failed: %v", err),
			}
		}

		if err := thermal.ValidateThermalProof(&msgBlock.Header); err != nil {
			mobileState.Lock()
			mobileState.thermalViolations++
			mobileState.Unlock()

			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCInvalidParameter,
				Message: fmt.Sprintf("thermal validation failed: %v", err),
			}
		}

		// Update temperature statistics
		mobileState.Lock()
		mobileState.averageTemperature = (mobileState.averageTemperature*float64(mobileState.blocksFoundMobile) +
			c.ThermalProof.Temperature) / float64(mobileState.blocksFoundMobile+1)
		mobileState.Unlock()
	}

	// Process the block - handle all 3 return values
	_, isOrphan, err := s.cfg.Chain.ProcessBlock(block, blockchain.BFNone)
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCVerify,
			Message: fmt.Sprintf("rejected: %v", err),
		}
	}

	// If not orphan, it was accepted
	if !isOrphan {
		// Update mobile mining stats
		mobileState.Lock()
		mobileState.blocksFoundMobile++
		mobileState.Unlock()

		// Notify websocket clients
		s.ntfnMgr.NotifyBlockConnected(block)
	}

	return nil, nil
}

// handleGetMobileWork implements the getmobilework command (simplified interface).
func handleGetMobileWork(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetMobileWorkCmd)

	// Get current state
	best := s.cfg.Chain.BestSnapshot()

	// Create simplified work
	workID := fmt.Sprintf("%d-%d", best.Height+1, time.Now().Unix())

	// Create a basic header template
	var buf [80]byte
	binary.LittleEndian.PutUint32(buf[0:4], 1) // Version
	copy(buf[4:36], best.Hash[:])              // Previous block hash
	// Merkle root would need to be calculated from transactions
	// For now, leave it empty
	binary.LittleEndian.PutUint32(buf[68:72], uint32(time.Now().Unix()))
	binary.LittleEndian.PutUint32(buf[72:76], best.Bits)
	// Nonce starts at 0

	// Determine difficulty based on device class
	targetBits := adjustBitsForDevice(best.Bits, c.DeviceClass)
	target := blockchain.CompactToBig(targetBits)

	// Generate NPU work if applicable
	npuWork := ""
	if c.DeviceClass == "flagship" || c.DeviceClass == "midrange" {
		npuWork = generateNPUWorkForHeight(best.Height + 1)
	}

	result := btcjson.GetMobileWorkResult{
		WorkID:       workID,
		Data:         hex.EncodeToString(buf[:]),
		Target:       fmt.Sprintf("%064x", target),
		NPUWork:      npuWork,
		ThermalLimit: 50.0, // Conservative default
	}

	return &result, nil
}

// handleSubmitMobileWork implements the submitmobilework command.
func handleSubmitMobileWork(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.SubmitMobileWorkCmd)

	// Parse nonce
	_, err := strconv.ParseUint(c.Nonce, 16, 32)
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidParameter,
			Message: "invalid nonce",
		}
	}

	// Parse thermal proof
	_, err = strconv.ParseUint(c.ThermalProof, 16, 64)
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidParameter,
			Message: "invalid thermal proof",
		}
	}

	// TODO: Validate work ID and reconstruct full block
	// This is a simplified implementation

	// For now, just return success
	return true, nil
}

// handleValidateThermalProof implements the validatethermalproof command.
func handleValidateThermalProof(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.ValidateThermalProofCmd)

	// Get block by hash
	hash, err := chainhash.NewHashFromStr(c.BlockHash)
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidParameter,
			Message: "invalid block hash",
		}
	}

	block, err := s.cfg.Chain.BlockByHash(hash)
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCBlockNotFound,
			Message: "block not found",
		}
	}

	// Get block bytes and deserialize using Shell's wire package
	blockBytes, err := block.Bytes()
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInternal.Code,
			Message: "failed to serialize block",
		}
	}

	msgBlock := &wire.MsgBlock{}
	err = msgBlock.Deserialize(bytes.NewReader(blockBytes))
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCDeserialization,
			Message: "block deserialize failed",
		}
	}

	// Create thermal verifier
	thermal := mobilex.NewThermalVerification(2000, 5.0)

	// Validate the thermal proof
	if msgBlock.Header.ThermalProof != c.ThermalProof {
		return false, nil
	}

	err = thermal.ValidateThermalProof(&msgBlock.Header)
	return err == nil, nil
}

// handleGetMobileStats implements the getmobilestats command.
func handleGetMobileStats(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	// Get stats from state
	mobileState.RLock()
	result := btcjson.GetMobileStatsResult{
		TotalMobileMiners:   mobileState.activeMinerCount,
		MobileHashrate:      mobileState.mobileHashrate,
		DeviceBreakdown:     copyDeviceBreakdown(mobileState.deviceBreakdown),
		GeographicBreakdown: make(map[string]int64), // TODO: Implement geo tracking
		ThermalViolations:   mobileState.thermalViolations,
		AverageTemperature:  mobileState.averageTemperature,
		NPUUtilization:      mobileState.npuUtilization,
		BlocksFoundMobile:   mobileState.blocksFoundMobile,
	}
	mobileState.RUnlock()

	return &result, nil
}

// Helper functions

// calculateMobileTarget returns the target difficulty for mobile miners as a hex string.
func calculateMobileTarget(bits uint32, deviceInfo *btcjson.DeviceInfo) string {
	// Start with network target
	target := blockchain.CompactToBig(bits)

	// Adjust based on device type
	var multiplier int64 = 10 // Default 10x easier
	if deviceInfo != nil {
		switch deviceInfo.SocModel {
		case "Snapdragon 8 Gen 3", "A17 Pro", "Tensor G3":
			multiplier = 10 // Flagship: 10x easier
		case "Snapdragon 7 Gen 3", "A16", "Tensor G2":
			multiplier = 20 // Mid-range: 20x easier
		default:
			multiplier = 50 // Budget: 50x easier
		}
	}

	// Make it easier for mobile miners
	mobileTarget := new(big.Int).Mul(target, big.NewInt(multiplier))
	return fmt.Sprintf("%064x", mobileTarget)
}

// adjustBitsForDevice adjusts difficulty bits for device class.
func adjustBitsForDevice(bits uint32, deviceClass string) uint32 {
	// Convert to target
	target := blockchain.CompactToBig(bits)

	// Adjust based on device class
	var multiplier int64
	switch deviceClass {
	case "flagship":
		multiplier = 10
	case "midrange":
		multiplier = 20
	default: // budget
		multiplier = 50
	}

	// Make easier
	newTarget := new(big.Int).Mul(target, big.NewInt(multiplier))

	// Convert back to bits
	return blockchain.BigToCompact(newTarget)
}

// generateNPUWorkForHeight generates NPU work parameters for a given height.
func generateNPUWorkForHeight(height int32) string {
	// Generate deterministic NPU work based on height
	work := make([]byte, 256) // Smaller than pool version
	for i := range work {
		work[i] = byte((height + int32(i)) % 256)
	}
	return hex.EncodeToString(work)
}

// determineWorkSize returns optimized work parameters for device.
func determineWorkSize(deviceInfo *btcjson.DeviceInfo) btcjson.MobileWorkSize {
	if deviceInfo == nil {
		return btcjson.MobileWorkSize{
			SearchSpace:   0x40000, // 256K default
			NPUIterations: 200,
			CacheSize:     1024 * 1024,
		}
	}

	switch deviceInfo.SocModel {
	case "Snapdragon 8 Gen 3", "A17 Pro":
		return btcjson.MobileWorkSize{
			SearchSpace:   0x100000, // 1M nonces
			NPUIterations: 100,
			CacheSize:     3 * 1024 * 1024,
		}
	case "Snapdragon 7 Gen 3", "A16":
		return btcjson.MobileWorkSize{
			SearchSpace:   0x80000, // 512K nonces
			NPUIterations: 150,
			CacheSize:     2 * 1024 * 1024,
		}
	default:
		return btcjson.MobileWorkSize{
			SearchSpace:   0x40000, // 256K nonces
			NPUIterations: 200,
			CacheSize:     1024 * 1024,
		}
	}
}

// copyDeviceBreakdown creates a copy of device breakdown map.
func copyDeviceBreakdown(original map[string]int64) map[string]int64 {
	copy := make(map[string]int64)
	for k, v := range original {
		copy[k] = v
	}
	return copy
}

// Mobile mining RPC command map for registration
var mobileRPCHandlers = map[string]commandHandler{
	"getmobileblocktemplate": handleGetMobileBlockTemplate,
	"getmobilemininginfo":    handleGetMobileMiningInfo,
	"submitmobileblock":      handleSubmitMobileBlock,
	"getmobilework":          handleGetMobileWork,
	"submitmobilework":       handleSubmitMobileWork,
	"validatethermalproof":   handleValidateThermalProof,
	"getmobilestats":         handleGetMobileStats,
}
