// Package liquidity implements Shell Reserve's liquidity reward program
// for bootstrapping professional market making during the first 3 years.
package liquidity

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/toole-brendan/shell/chaincfg/chainhash"
	"github.com/toole-brendan/shell/wire"
)

// Liquidity reward program constants
const (
	RewardPoolSize  = 2000000 * 1e8 // 2% of total supply (2M XSL)
	EpochCount      = 12            // 12 quarterly epochs
	EpochBlocks     = 26280         // ~3 months at 5-minute blocks
	MinAttestorSigs = 3             // Minimum attestor signatures required
)

// LiquidityEpoch represents a 3-month reward period
type LiquidityEpoch struct {
	Index       uint8
	StartBlock  int32
	EndBlock    int32
	RewardPool  uint64
	MerkleRoot  chainhash.Hash
	TotalWeight uint64
}

// AttestorInfo represents a known market data provider
type AttestorInfo struct {
	PublicKey *btcec.PublicKey
	Name      string
	Endpoint  string
	Weight    uint8 // Voting weight (1-5)
}

// KnownAttestors are the authorized market data providers
var KnownAttestors = []AttestorInfo{
	{
		// Kaiko attestor (placeholder - real keys would be provided)
		Name:     "Kaiko",
		Endpoint: "https://kaiko.shell-attestation.com",
		Weight:   5,
	},
	{
		// Coin Metrics attestor
		Name:     "Coin Metrics",
		Endpoint: "https://coinmetrics.shell-attestation.com",
		Weight:   5,
	},
	{
		// CME CF Benchmarks attestor
		Name:     "CME CF Benchmarks",
		Endpoint: "https://cf.shell-attestation.com",
		Weight:   4,
	},
	{
		// State Street attestor
		Name:     "State Street",
		Endpoint: "https://statestreet.shell-attestation.com",
		Weight:   3,
	},
	{
		// Anchorage Digital attestor
		Name:     "Anchorage Digital",
		Endpoint: "https://anchorage.shell-attestation.com",
		Weight:   3,
	},
}

// LiquidityAttestation represents signed market making data
type LiquidityAttestation struct {
	EpochIndex    uint8
	ParticipantID [32]byte // Hash of participant's public key
	Volume        uint64   // Verified trading volume in satoshis
	Spread        uint32   // Average spread in basis points
	Uptime        uint16   // Percentage uptime (0-10000)
	Timestamp     uint32   // Unix timestamp
	AttestorSigs  []ecdsa.Signature
	MerkleProof   []chainhash.Hash
}

// LiquidityRewardClaim represents a claim for liquidity rewards
type LiquidityRewardClaim struct {
	Version         int32
	EpochIndex      uint8
	AttestationBlob []byte
	MerklePath      []chainhash.Hash
	Output          *wire.TxOut
}

// LiquidityManager handles the reward program state
type LiquidityManager struct {
	epochs     [EpochCount]LiquidityEpoch
	claims     map[[32]byte]bool // Track claimed rewards
	startBlock int32
}

// NewLiquidityManager creates a new liquidity reward manager
func NewLiquidityManager(startBlock int32) *LiquidityManager {
	lm := &LiquidityManager{
		claims:     make(map[[32]byte]bool),
		startBlock: startBlock,
	}

	// Initialize epochs
	for i := 0; i < EpochCount; i++ {
		lm.epochs[i] = LiquidityEpoch{
			Index:      uint8(i),
			StartBlock: startBlock + int32(i*EpochBlocks),
			EndBlock:   startBlock + int32((i+1)*EpochBlocks) - 1,
			RewardPool: uint64(RewardPoolSize) / uint64(EpochCount),
		}
	}

	return lm
}

// ValidateAttestation verifies an attestation from market data providers
func (lm *LiquidityManager) ValidateAttestation(attestation *LiquidityAttestation) error {
	// Check epoch validity
	if attestation.EpochIndex >= EpochCount {
		return errors.New("invalid epoch index")
	}

	epoch := lm.epochs[attestation.EpochIndex]
	if epoch.MerkleRoot == (chainhash.Hash{}) {
		return errors.New("epoch not finalized")
	}

	// Verify minimum number of attestor signatures
	if len(attestation.AttestorSigs) < MinAttestorSigs {
		return fmt.Errorf("insufficient attestor signatures: got %d, need %d",
			len(attestation.AttestorSigs), MinAttestorSigs)
	}

	// Verify attestor signatures
	validSigs := 0
	attestationHash := lm.hashAttestation(attestation)

	for i, sig := range attestation.AttestorSigs {
		if i >= len(KnownAttestors) {
			continue
		}

		attestor := KnownAttestors[i]
		if attestor.PublicKey != nil && sig.Verify(attestationHash[:], attestor.PublicKey) {
			validSigs++
		}
	}

	if validSigs < MinAttestorSigs {
		return fmt.Errorf("insufficient valid signatures: got %d, need %d",
			validSigs, MinAttestorSigs)
	}

	// Verify merkle inclusion in epoch root
	if !lm.verifyMerkleProof(attestationHash, attestation.MerkleProof, epoch.MerkleRoot) {
		return errors.New("invalid merkle proof")
	}

	return nil
}

// ProcessRewardClaim processes a liquidity reward claim
func (lm *LiquidityManager) ProcessRewardClaim(claim *LiquidityRewardClaim, currentBlock int32) error {
	// Parse attestation from blob
	attestation, err := lm.parseAttestationBlob(claim.AttestationBlob)
	if err != nil {
		return fmt.Errorf("failed to parse attestation: %w", err)
	}

	// Validate attestation
	if err := lm.ValidateAttestation(attestation); err != nil {
		return fmt.Errorf("attestation validation failed: %w", err)
	}

	// Check if reward already claimed
	claimHash := lm.hashClaim(claim)
	if lm.claims[claimHash] {
		return errors.New("reward already claimed")
	}

	// Calculate reward amount
	epoch := lm.epochs[claim.EpochIndex]
	weight := lm.calculateMarketMakerWeight(attestation)
	reward := (weight * epoch.RewardPool) / epoch.TotalWeight

	// Verify output amount doesn't exceed calculated reward
	if uint64(claim.Output.Value) > reward {
		return fmt.Errorf("excessive reward claim: requested %d, max %d",
			claim.Output.Value, reward)
	}

	// Mark as claimed
	lm.claims[claimHash] = true

	return nil
}

// hashAttestation computes the hash of an attestation for verification
func (lm *LiquidityManager) hashAttestation(attestation *LiquidityAttestation) chainhash.Hash {
	data := make([]byte, 0, 128)

	data = append(data, attestation.EpochIndex)
	data = append(data, attestation.ParticipantID[:]...)

	volumeBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(volumeBytes, attestation.Volume)
	data = append(data, volumeBytes...)

	spreadBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(spreadBytes, attestation.Spread)
	data = append(data, spreadBytes...)

	uptimeBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(uptimeBytes, attestation.Uptime)
	data = append(data, uptimeBytes...)

	timestampBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(timestampBytes, attestation.Timestamp)
	data = append(data, timestampBytes...)

	hash := sha256.Sum256(data)
	return chainhash.Hash(hash)
}

// hashClaim computes the hash of a reward claim
func (lm *LiquidityManager) hashClaim(claim *LiquidityRewardClaim) [32]byte {
	data := make([]byte, 0, 128)

	versionBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(versionBytes, uint32(claim.Version))
	data = append(data, versionBytes...)

	data = append(data, claim.EpochIndex)
	data = append(data, claim.AttestationBlob...)

	for _, proof := range claim.MerklePath {
		data = append(data, proof[:]...)
	}

	return sha256.Sum256(data)
}

// calculateMarketMakerWeight computes weight based on volume, spread, and uptime
func (lm *LiquidityManager) calculateMarketMakerWeight(attestation *LiquidityAttestation) uint64 {
	// Simplified weight calculation
	// In production, this would use a more sophisticated formula
	// Weight = Volume * UptimeFactor * SpreadFactor

	volume := attestation.Volume
	uptimeFactor := uint64(attestation.Uptime) // Already in basis points (0-10000)

	// Better spreads get higher weight (inverse relationship)
	// Maximum spread factor of 10000, minimum of 1000
	spreadFactor := uint64(10000)
	if attestation.Spread > 0 {
		spreadFactor = 10000 / (1 + uint64(attestation.Spread)/100)
		if spreadFactor < 1000 {
			spreadFactor = 1000
		}
	}

	weight := (volume * uptimeFactor * spreadFactor) / (10000 * 10000)
	return weight
}

// verifyMerkleProof verifies inclusion in a merkle tree
func (lm *LiquidityManager) verifyMerkleProof(leaf chainhash.Hash, proof []chainhash.Hash, root chainhash.Hash) bool {
	current := leaf

	for _, sibling := range proof {
		// Combine current with sibling
		combined := make([]byte, 64)
		copy(combined[:32], current[:])
		copy(combined[32:], sibling[:])

		hash := sha256.Sum256(combined)
		current = chainhash.Hash(hash)
	}

	return current == root
}

// parseAttestationBlob extracts attestation from binary blob
func (lm *LiquidityManager) parseAttestationBlob(blob []byte) (*LiquidityAttestation, error) {
	// TODO: Implement proper binary deserialization
	// For now, return a placeholder
	return &LiquidityAttestation{}, errors.New("attestation parsing not yet implemented")
}

// FinalizeEpoch computes final rewards for an epoch
func (lm *LiquidityManager) FinalizeEpoch(epochIndex uint8, merkleRoot chainhash.Hash, totalWeight uint64) error {
	if epochIndex >= EpochCount {
		return errors.New("invalid epoch index")
	}

	epoch := &lm.epochs[epochIndex]
	epoch.MerkleRoot = merkleRoot
	epoch.TotalWeight = totalWeight

	return nil
}

// GetEpochInfo returns information about a specific epoch
func (lm *LiquidityManager) GetEpochInfo(epochIndex uint8) (*LiquidityEpoch, error) {
	if epochIndex >= EpochCount {
		return nil, errors.New("invalid epoch index")
	}

	return &lm.epochs[epochIndex], nil
}

// IsEpochActive checks if an epoch is currently active
func (lm *LiquidityManager) IsEpochActive(epochIndex uint8, currentBlock int32) bool {
	if epochIndex >= EpochCount {
		return false
	}

	epoch := lm.epochs[epochIndex]
	return currentBlock >= epoch.StartBlock && currentBlock <= epoch.EndBlock
}

// GetActiveEpoch returns the currently active epoch
func (lm *LiquidityManager) GetActiveEpoch(currentBlock int32) *LiquidityEpoch {
	for i := 0; i < EpochCount; i++ {
		if lm.IsEpochActive(uint8(i), currentBlock) {
			return &lm.epochs[i]
		}
	}
	return nil
}

// GetRewardPoolRemaining returns unclaimed rewards for an epoch
func (lm *LiquidityManager) GetRewardPoolRemaining(epochIndex uint8) uint64 {
	// This would track actual claims in production
	// For now, return the full pool
	if epochIndex >= EpochCount {
		return 0
	}
	return lm.epochs[epochIndex].RewardPool
}
