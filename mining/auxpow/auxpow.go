// Package auxpow implements auxiliary proof-of-work for Shell Reserve
// Phase γ.3: Bitcoin merge mining for initial network security
package auxpow

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"math/big"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/toole-brendan/shell/chaincfg"
)

// Phase γ.3: AuxPoW Integration for Shell Reserve

// AuxPoWConfig defines the configuration for auxiliary proof-of-work
type AuxPoWConfig struct {
	// Enable auxiliary proof-of-work
	Enabled bool

	// Chain ID for Shell Reserve in merge mining
	ChainID uint32

	// Required tag in Bitcoin coinbase
	CommitmentTag string

	// Native hashrate threshold for sunset (in TH/s)
	SunsetHashrateThreshold uint64

	// Monitoring period for hashrate assessment
	MonitoringBlocks uint32

	// Notice period before sunset activation
	SunsetNoticeBlocks uint32

	// Minimum merge mining reward (in satoshis)
	MinMergeReward uint64
}

// AuxPoWBlock represents a Shell block with auxiliary proof-of-work
type AuxPoWBlock struct {
	// Shell block header
	Header *wire.BlockHeader

	// Auxiliary proof-of-work data
	AuxData *AuxPoWData

	// Validation status
	IsValid bool

	// Validation timestamp
	ValidatedAt time.Time
}

// AuxPoWData contains the proof that links Shell block to Bitcoin block
type AuxPoWData struct {
	// Parent Bitcoin coinbase transaction containing Shell commitment
	ParentCoinbase *wire.MsgTx

	// Merkle branch proving coinbase inclusion in Bitcoin block
	MerkleBranch []chainhash.Hash

	// Number of transactions in Bitcoin block (for merkle proof)
	ParentBlockTxCount uint32

	// Bitcoin block header
	ParentBlock *wire.BlockHeader

	// Chain index (always 0 for Bitcoin)
	ChainIndex uint32

	// Shell block hash committed in Bitcoin coinbase
	ShellBlockHash chainhash.Hash
}

// AuxPoWValidator handles validation of auxiliary proof-of-work
type AuxPoWValidator struct {
	config      *AuxPoWConfig
	chainParams *chaincfg.Params

	// Hashrate tracking for sunset mechanism
	nativeHashrate uint64 // Current native hashrate (TH/s)
	mergeHashrate  uint64 // Current merge mining hashrate (TH/s)

	// Sunset state
	sunsetActivated    bool
	sunsetNoticeHeight uint32

	// Statistics
	totalAuxBlocks    uint64
	totalNativeBlocks uint64
	lastHashrateCheck uint32
}

// NewAuxPoWValidator creates a new auxiliary proof-of-work validator
func NewAuxPoWValidator(config *AuxPoWConfig, params *chaincfg.Params) *AuxPoWValidator {
	return &AuxPoWValidator{
		config:             config,
		chainParams:        params,
		nativeHashrate:     0,
		mergeHashrate:      0,
		sunsetActivated:    false,
		sunsetNoticeHeight: 0,
		totalAuxBlocks:     0,
		totalNativeBlocks:  0,
		lastHashrateCheck:  0,
	}
}

// ValidateAuxPoW validates an auxiliary proof-of-work block
func (v *AuxPoWValidator) ValidateAuxPoW(auxBlock *AuxPoWBlock) error {
	if !v.config.Enabled {
		return fmt.Errorf("auxiliary proof-of-work is disabled")
	}

	if v.sunsetActivated {
		return fmt.Errorf("auxiliary proof-of-work has been sunset")
	}

	auxData := auxBlock.AuxData
	if auxData == nil {
		return fmt.Errorf("missing auxiliary proof-of-work data")
	}

	// 1. Verify Shell block hash commitment in Bitcoin coinbase
	err := v.verifyShellCommitment(auxData)
	if err != nil {
		return fmt.Errorf("shell commitment verification failed: %v", err)
	}

	// 2. Verify merkle branch proving coinbase inclusion
	err = v.verifyMerkleBranch(auxData)
	if err != nil {
		return fmt.Errorf("merkle branch verification failed: %v", err)
	}

	// 3. Verify Bitcoin block meets Shell difficulty target
	err = v.verifyWorkSufficiency(auxBlock.Header, auxData.ParentBlock)
	if err != nil {
		return fmt.Errorf("work sufficiency verification failed: %v", err)
	}

	// 4. Verify chain index and other parameters
	err = v.verifyAuxParams(auxData)
	if err != nil {
		return fmt.Errorf("auxiliary parameters verification failed: %v", err)
	}

	// Mark as valid
	auxBlock.IsValid = true
	auxBlock.ValidatedAt = time.Now()

	// Update statistics
	v.totalAuxBlocks++

	return nil
}

// verifyShellCommitment verifies that Bitcoin coinbase contains Shell commitment
func (v *AuxPoWValidator) verifyShellCommitment(auxData *AuxPoWData) error {
	coinbase := auxData.ParentCoinbase
	if coinbase == nil || len(coinbase.TxIn) == 0 {
		return fmt.Errorf("invalid coinbase transaction")
	}

	// Extract coinbase script
	coinbaseScript := coinbase.TxIn[0].SignatureScript

	// Look for Shell commitment tag
	tagBytes := []byte(v.config.CommitmentTag)
	tagIndex := bytes.Index(coinbaseScript, tagBytes)
	if tagIndex == -1 {
		return fmt.Errorf("shell commitment tag not found in coinbase")
	}

	// Extract committed Shell block hash (32 bytes after tag)
	commitmentStart := tagIndex + len(tagBytes)
	if len(coinbaseScript) < commitmentStart+32 {
		return fmt.Errorf("insufficient data for shell block hash commitment")
	}

	committedHash := coinbaseScript[commitmentStart : commitmentStart+32]

	// Verify it matches the Shell block hash
	if !bytes.Equal(committedHash, auxData.ShellBlockHash[:]) {
		return fmt.Errorf("committed shell block hash mismatch")
	}

	return nil
}

// verifyMerkleBranch verifies the merkle branch proving coinbase inclusion
func (v *AuxPoWValidator) verifyMerkleBranch(auxData *AuxPoWData) error {
	// Calculate coinbase transaction hash
	coinbaseHash := auxData.ParentCoinbase.TxHash()

	// Verify merkle path from coinbase to block merkle root
	computedRoot := v.computeMerkleRoot(coinbaseHash, auxData.MerkleBranch, auxData.ParentBlockTxCount)

	// Compare with Bitcoin block merkle root
	if !computedRoot.IsEqual(&auxData.ParentBlock.MerkleRoot) {
		return fmt.Errorf("merkle root mismatch: computed %s, expected %s",
			computedRoot, auxData.ParentBlock.MerkleRoot)
	}

	return nil
}

// verifyWorkSufficiency verifies Bitcoin block work meets Shell difficulty
func (v *AuxPoWValidator) verifyWorkSufficiency(shellHeader *wire.BlockHeader, bitcoinHeader *wire.BlockHeader) error {
	// Calculate Bitcoin block work
	bitcoinWork := v.calculateWork(bitcoinHeader.Bits)

	// Get Shell difficulty target
	shellTarget := v.calculateTarget(shellHeader.Bits)

	// Convert Shell target to work
	shellWork := v.targetToWork(shellTarget)

	// Bitcoin work must exceed Shell work requirement
	if bitcoinWork.Cmp(shellWork) < 0 {
		return fmt.Errorf("insufficient work: bitcoin %s < shell %s", bitcoinWork, shellWork)
	}

	return nil
}

// verifyAuxParams verifies auxiliary proof-of-work parameters
func (v *AuxPoWValidator) verifyAuxParams(auxData *AuxPoWData) error {
	// Chain index must be 0 for Bitcoin
	if auxData.ChainIndex != 0 {
		return fmt.Errorf("invalid chain index: %d (must be 0 for Bitcoin)", auxData.ChainIndex)
	}

	// Verify parent block hash format
	if auxData.ParentBlock == nil {
		return fmt.Errorf("missing parent block header")
	}

	// Basic sanity checks on Bitcoin block header
	if auxData.ParentBlock.Version == 0 {
		return fmt.Errorf("invalid Bitcoin block version")
	}

	if auxData.ParentBlock.Timestamp.Before(time.Date(2009, 1, 3, 0, 0, 0, 0, time.UTC)) {
		return fmt.Errorf("Bitcoin block timestamp too early")
	}

	return nil
}

// computeMerkleRoot computes merkle root from coinbase hash and branch
func (v *AuxPoWValidator) computeMerkleRoot(coinbaseHash chainhash.Hash, branch []chainhash.Hash, txCount uint32) chainhash.Hash {
	hash := coinbaseHash

	// Apply merkle branch hashes
	for _, branchHash := range branch {
		// Concatenate and double-SHA256
		combined := append(hash[:], branchHash[:]...)
		first := sha256.Sum256(combined)
		second := sha256.Sum256(first[:])
		hash = chainhash.Hash(second)
	}

	return hash
}

// calculateWork calculates proof-of-work from difficulty bits
func (v *AuxPoWValidator) calculateWork(bits uint32) *big.Int {
	// Convert compact bits to target
	target := v.calculateTarget(bits)

	// Work = 2^256 / (target + 1)
	work := new(big.Int)
	work.Lsh(big.NewInt(1), 256)
	work.Div(work, new(big.Int).Add(target, big.NewInt(1)))

	return work
}

// calculateTarget converts compact difficulty bits to target
func (v *AuxPoWValidator) calculateTarget(bits uint32) *big.Int {
	// Extract mantissa and exponent from compact format
	mantissa := bits & 0x007fffff
	exponent := uint8(bits >> 24)

	if exponent <= 3 {
		mantissa >>= (8 * (3 - exponent))
		return big.NewInt(int64(mantissa))
	}

	result := big.NewInt(int64(mantissa))
	result.Lsh(result, uint(8*(exponent-3)))

	return result
}

// targetToWork converts difficulty target to work value
func (v *AuxPoWValidator) targetToWork(target *big.Int) *big.Int {
	if target.Sign() <= 0 {
		return big.NewInt(0)
	}

	// Work = 2^256 / (target + 1)
	work := new(big.Int)
	work.Lsh(big.NewInt(1), 256)
	work.Div(work, new(big.Int).Add(target, big.NewInt(1)))

	return work
}

// UpdateHashrateMetrics updates hashrate tracking for sunset mechanism
func (v *AuxPoWValidator) UpdateHashrateMetrics(blockHeight uint32, isAuxPoW bool, blockHash chainhash.Hash) {
	// Update block counts
	if isAuxPoW {
		v.totalAuxBlocks++
	} else {
		v.totalNativeBlocks++
	}

	// Check if it's time to assess hashrate
	if blockHeight >= v.lastHashrateCheck+v.config.MonitoringBlocks {
		v.assessHashrateForSunset(blockHeight)
		v.lastHashrateCheck = blockHeight
	}
}

// assessHashrateForSunset determines if sunset should be activated
func (v *AuxPoWValidator) assessHashrateForSunset(blockHeight uint32) {
	if v.sunsetActivated {
		return
	}

	// Calculate recent block ratio (last MonitoringBlocks)
	recentBlocks := v.config.MonitoringBlocks
	if recentBlocks == 0 {
		recentBlocks = 1008 // Default: ~1 week
	}

	// Estimate native hashrate based on block ratio and difficulty
	nativeRatio := float64(v.totalNativeBlocks) / float64(v.totalNativeBlocks+v.totalAuxBlocks)

	// Simplified hashrate estimation (would be more sophisticated in production)
	estimatedNativeHashrate := uint64(nativeRatio * 100) // TH/s (simplified)

	v.nativeHashrate = estimatedNativeHashrate

	// Check if native hashrate exceeds threshold
	if v.nativeHashrate >= v.config.SunsetHashrateThreshold && v.sunsetNoticeHeight == 0 {
		// Start sunset notice period
		v.sunsetNoticeHeight = blockHeight + v.config.SunsetNoticeBlocks

		// Log sunset notice (would use proper logging in production)
		fmt.Printf("AuxPoW Sunset Notice: Native hashrate %d TH/s exceeds threshold %d TH/s. "+
			"AuxPoW will be disabled at block %d\n",
			v.nativeHashrate, v.config.SunsetHashrateThreshold, v.sunsetNoticeHeight)
	}

	// Check if sunset should be activated
	if v.sunsetNoticeHeight > 0 && blockHeight >= v.sunsetNoticeHeight {
		v.activateSunset()
	}
}

// activateSunset disables auxiliary proof-of-work
func (v *AuxPoWValidator) activateSunset() {
	v.sunsetActivated = true
	v.config.Enabled = false

	// Log sunset activation (would use proper logging in production)
	fmt.Printf("AuxPoW Sunset Activated: Auxiliary proof-of-work disabled. " +
		"Shell Reserve now relies entirely on native RandomX mining.\n")
}

// GetSunsetStatus returns the current sunset status
func (v *AuxPoWValidator) GetSunsetStatus() (bool, uint32, uint64, uint64) {
	return v.sunsetActivated, v.sunsetNoticeHeight, v.nativeHashrate, v.mergeHashrate
}

// GetStatistics returns current AuxPoW statistics
func (v *AuxPoWValidator) GetStatistics() map[string]interface{} {
	stats := make(map[string]interface{})

	stats["aux_blocks"] = v.totalAuxBlocks
	stats["native_blocks"] = v.totalNativeBlocks
	stats["native_hashrate_ths"] = v.nativeHashrate
	stats["merge_hashrate_ths"] = v.mergeHashrate
	stats["sunset_activated"] = v.sunsetActivated
	stats["sunset_notice_height"] = v.sunsetNoticeHeight

	if v.totalAuxBlocks+v.totalNativeBlocks > 0 {
		auxRatio := float64(v.totalAuxBlocks) / float64(v.totalAuxBlocks+v.totalNativeBlocks)
		stats["aux_block_ratio"] = auxRatio
		stats["native_block_ratio"] = 1.0 - auxRatio
	} else {
		stats["aux_block_ratio"] = 0.0
		stats["native_block_ratio"] = 0.0
	}

	return stats
}

// DefaultAuxPoWConfig returns the default configuration for Shell Reserve
func DefaultAuxPoWConfig() *AuxPoWConfig {
	return &AuxPoWConfig{
		Enabled:                 true,
		ChainID:                 0x58534C, // "XSL" in hex
		CommitmentTag:           "XSLTAG",
		SunsetHashrateThreshold: 1000,    // 1 PH/s (1000 TH/s)
		MonitoringBlocks:        1008,    // ~1 week at 5-min blocks
		SunsetNoticeBlocks:      25920,   // ~6 months notice
		MinMergeReward:          1000000, // 0.01 XSL minimum
	}
}

// CreateShellCommitment creates a Bitcoin coinbase commitment for Shell block
func CreateShellCommitment(shellBlockHash chainhash.Hash, tag string) []byte {
	commitment := make([]byte, 0, len(tag)+32)
	commitment = append(commitment, []byte(tag)...)
	commitment = append(commitment, shellBlockHash[:]...)
	return commitment
}

// ExtractShellCommitment extracts Shell block hash from Bitcoin coinbase
func ExtractShellCommitment(coinbaseScript []byte, tag string) (*chainhash.Hash, error) {
	tagBytes := []byte(tag)
	tagIndex := bytes.Index(coinbaseScript, tagBytes)
	if tagIndex == -1 {
		return nil, fmt.Errorf("shell commitment tag not found")
	}

	commitmentStart := tagIndex + len(tagBytes)
	if len(coinbaseScript) < commitmentStart+32 {
		return nil, fmt.Errorf("insufficient data for shell block hash")
	}

	hashBytes := coinbaseScript[commitmentStart : commitmentStart+32]
	hash, err := chainhash.NewHash(hashBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid hash format: %v", err)
	}

	return hash, nil
}

// MergeMinable checks if a Bitcoin block can be used for merge mining Shell
func MergeMinable(bitcoinHeader *wire.BlockHeader, coinbase *wire.MsgTx, shellTarget *big.Int) bool {
	// Check if Bitcoin block meets Shell difficulty
	bitcoinWork := CalculateWorkFromHeader(bitcoinHeader)
	shellWork := new(big.Int).Div(new(big.Int).Lsh(big.NewInt(1), 256), new(big.Int).Add(shellTarget, big.NewInt(1)))

	if bitcoinWork.Cmp(shellWork) < 0 {
		return false
	}

	// Check if coinbase contains Shell commitment
	_, err := ExtractShellCommitment(coinbase.TxIn[0].SignatureScript, "XSLTAG")
	return err == nil
}

// CalculateWorkFromHeader calculates proof-of-work from block header
func CalculateWorkFromHeader(header *wire.BlockHeader) *big.Int {
	target := CompactToBig(header.Bits)
	if target.Sign() <= 0 {
		return big.NewInt(0)
	}

	work := new(big.Int)
	work.Lsh(big.NewInt(1), 256)
	work.Div(work, new(big.Int).Add(target, big.NewInt(1)))

	return work
}

// CompactToBig converts compact difficulty representation to big integer
func CompactToBig(compact uint32) *big.Int {
	mantissa := compact & 0x007fffff
	exponent := uint8(compact >> 24)

	if exponent <= 3 {
		mantissa >>= (8 * (3 - exponent))
		return big.NewInt(int64(mantissa))
	}

	result := big.NewInt(int64(mantissa))
	result.Lsh(result, uint(8*(exponent-3)))

	return result
}
