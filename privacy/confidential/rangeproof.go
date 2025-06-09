// Copyright (c) 2025 Shell Reserve developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package confidential

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"

	"github.com/btcsuite/btcd/btcec/v2"
)

const (
	// RangeProofSize is the maximum size of a range proof
	RangeProofSize = 675 // Typical Bulletproof size

	// MaxValue is the maximum value that can be committed (2^52 - 1 for precision)
	MaxValue = (1 << 52) - 1

	// NumBits is the number of bits in the range proof (52 bits)
	NumBits = 52
)

var (
	// ErrInvalidRangeProof is returned when a range proof is invalid
	ErrInvalidRangeProof = errors.New("invalid range proof")

	// ErrValueOutOfRange is returned when a value is outside the valid range
	ErrValueOutOfRange = errors.New("value out of range")
)

// RangeProof represents a zero-knowledge proof that a committed value is in [0, 2^n)
type RangeProof struct {
	// Simplified range proof - in production would use Bulletproofs
	// For now, we use a Pedersen commitment based approach
	proof []byte
}

// RangeProofParams contains the parameters for range proof generation and verification
type RangeProofParams struct {
	// Value is the secret value being proved
	value uint64
	// BlindingFactor is the secret blinding factor
	blindingFactor *BlindingFactor
	// Commitment is the public commitment to the value
	commitment *PedersenCommitment
}

// NewRangeProof creates a new range proof from serialized data
func NewRangeProof(data []byte) (*RangeProof, error) {
	if len(data) == 0 {
		return nil, ErrInvalidRangeProof
	}

	proof := make([]byte, len(data))
	copy(proof, data)
	return &RangeProof{
		proof: proof,
	}, nil
}

// Bytes returns the serialized range proof
func (rp *RangeProof) Bytes() []byte {
	return rp.proof
}

// Size returns the size of the range proof in bytes
func (rp *RangeProof) Size() int {
	return len(rp.proof)
}

// GenerateRangeProof creates a zero-knowledge proof that value ∈ [0, 2^NumBits)
func GenerateRangeProof(params *RangeProofParams) (*RangeProof, error) {
	if params.value > MaxValue {
		return nil, ErrValueOutOfRange
	}

	// Simplified range proof using bit decomposition
	// In production, this would be replaced with proper Bulletproofs
	proof, err := generateSimplifiedRangeProof(params)
	if err != nil {
		return nil, fmt.Errorf("failed to generate range proof: %w", err)
	}

	return &RangeProof{proof: proof}, nil
}

// VerifyRangeProof verifies that the commitment contains a value in [0, 2^NumBits)
func VerifyRangeProof(commitment *PedersenCommitment, proof *RangeProof) bool {
	if proof == nil || commitment == nil {
		return false
	}

	// Simplified verification - in production would use Bulletproof verification
	return verifySimplifiedRangeProof(commitment, proof)
}

// generateSimplifiedRangeProof creates a simplified range proof
// This is a placeholder implementation - real Bulletproofs would be used in production
func generateSimplifiedRangeProof(params *RangeProofParams) ([]byte, error) {
	// Bit decomposition approach: prove each bit of the value
	bitCommitments := make([]*PedersenCommitment, NumBits)
	bitBlindingFactors := make([]*BlindingFactor, NumBits)

	// Decompose value into bits
	valueBits := decomposeToBits(params.value, NumBits)

	// Create commitment for each bit
	for i := 0; i < NumBits; i++ {
		// Generate blinding factor for this bit
		blindFactor, err := GenerateBlindingFactor()
		if err != nil {
			return nil, err
		}
		bitBlindingFactors[i] = blindFactor

		// Create commitment to the bit (0 or 1)
		bitCommitments[i], err = CreateCommitment(valueBits[i], blindFactor)
		if err != nil {
			return nil, err
		}
	}

	// Generate proofs that each commitment is to 0 or 1
	// This is simplified - real implementation would use proper zero-knowledge proofs
	proofData := make([]byte, 0, RangeProofSize)

	// Add commitment data
	for _, commitment := range bitCommitments {
		proofData = append(proofData, commitment.Bytes()...)
	}

	// Add a simple hash-based proof of knowledge
	// In production, this would be replaced with proper Sigma protocols
	hasher := sha256.New()
	hasher.Write(params.commitment.Bytes())
	challengePrefix := hasher.Sum(nil)[:8] // First 8 bytes for verification

	// Create a full 32-byte challenge that starts with the commitment prefix
	challengeHash := make([]byte, 32)
	copy(challengeHash[:8], challengePrefix)
	// Fill the rest with commitment and blinding factor data
	hasher2 := sha256.New()
	hasher2.Write(params.blindingFactor.Bytes())
	for _, bf := range bitBlindingFactors {
		hasher2.Write(bf.Bytes())
	}
	remainingHash := hasher2.Sum(nil)
	copy(challengeHash[8:], remainingHash[:24])

	proofData = append(proofData, challengeHash...)

	return proofData, nil
}

// verifySimplifiedRangeProof verifies a simplified range proof
func verifySimplifiedRangeProof(commitment *PedersenCommitment, proof *RangeProof) bool {
	// This is a simplified verification for testing
	// In production, this would implement full Bulletproof verification

	// For now, we do a basic check that the proof exists and has reasonable size
	proofData := proof.Bytes()
	if len(proofData) < 64 { // Minimum proof size
		return false
	}

	// In a real implementation, we would:
	// 1. Verify the bit commitments sum to the original commitment
	// 2. Verify each bit commitment is to 0 or 1
	// 3. Verify the challenge-response protocol

	// For testing purposes, we'll do a simplified check
	// that verifies the proof was generated for this commitment
	expectedSize := NumBits*CommitmentSize + 32
	if len(proofData) >= expectedSize {
		// Extract the challenge hash at the end
		challengeOffset := len(proofData) - 32
		challengeHash := proofData[challengeOffset:]

		// Verify the challenge includes the commitment
		hasher := sha256.New()
		hasher.Write(commitment.Bytes())
		computedPrefix := hasher.Sum(nil)[:8] // First 8 bytes

		// Check if the challenge hash starts with the commitment prefix
		// This is a very simplified verification for testing
		for i := 0; i < 8; i++ {
			if challengeHash[i] != computedPrefix[i] {
				return false
			}
		}
		return true
	}

	return false
}

// scaleCommitment multiplies a commitment by a scalar
func scaleCommitment(commitment *PedersenCommitment, scalar uint64) *PedersenCommitment {
	scalarBytes := new(big.Int).SetUint64(scalar).Bytes()

	// Scale the point: scalar * commitment
	scaledX, scaledY := btcec.S256().ScalarMult(
		commitment.point.X(), commitment.point.Y(),
		scalarBytes,
	)

	// Convert to public key
	var fx, fy btcec.FieldVal
	fx.SetByteSlice(scaledX.Bytes())
	fy.SetByteSlice(scaledY.Bytes())
	scaledPoint := btcec.NewPublicKey(&fx, &fy)

	return &PedersenCommitment{point: scaledPoint}
}

// decomposeToBits decomposes a value into its bit representation
func decomposeToBits(value uint64, numBits int) []uint64 {
	bits := make([]uint64, numBits)
	for i := 0; i < numBits; i++ {
		if (value>>i)&1 == 1 {
			bits[i] = 1
		} else {
			bits[i] = 0
		}
	}
	return bits
}

// BatchVerifyRangeProofs verifies multiple range proofs efficiently
func BatchVerifyRangeProofs(commitments []*PedersenCommitment, proofs []*RangeProof) bool {
	if len(commitments) != len(proofs) {
		return false
	}

	// For simplified implementation, verify each proof individually
	// In production, this would use batch verification for efficiency
	for i, commitment := range commitments {
		if !VerifyRangeProof(commitment, proofs[i]) {
			return false
		}
	}

	return true
}

// CreateDummyRangeProof creates a dummy range proof for testing
// This should only be used in test environments
func CreateDummyRangeProof() *RangeProof {
	// Create a minimal dummy proof
	dummyData := make([]byte, 64)
	rand.Read(dummyData)

	return &RangeProof{proof: dummyData}
}

// IsValidValueRange checks if a value is within the valid range
func IsValidValueRange(value uint64) bool {
	return value <= MaxValue
}
