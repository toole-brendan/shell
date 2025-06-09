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
	"github.com/toole-brendan/shell/chaincfg/chainhash"
)

const (
	// CommitmentSize is the size of a Pedersen commitment in bytes (33 bytes compressed point)
	CommitmentSize = 33

	// BlindingFactorSize is the size of a blinding factor in bytes
	BlindingFactorSize = 32
)

var (
	// ErrInvalidCommitment is returned when a commitment is malformed
	ErrInvalidCommitment = errors.New("invalid commitment")

	// ErrInvalidBlindingFactor is returned when a blinding factor is invalid
	ErrInvalidBlindingFactor = errors.New("invalid blinding factor")

	// ErrCommitmentMismatch is returned when commitments don't verify
	ErrCommitmentMismatch = errors.New("commitment verification failed")

	// Secp256k1 curve order
	curveOrder = btcec.S256().N
)

// PedersenCommitment represents a Pedersen commitment: C = vH + rG
// where v is the value, r is the blinding factor, H is the value generator, G is the base point
type PedersenCommitment struct {
	point *btcec.PublicKey
}

// BlindingFactor represents a random blinding factor used in commitments
type BlindingFactor [BlindingFactorSize]byte

// NewPedersenCommitment creates a commitment from a serialized point
func NewPedersenCommitment(data []byte) (*PedersenCommitment, error) {
	if len(data) != CommitmentSize {
		return nil, ErrInvalidCommitment
	}

	pubKey, err := btcec.ParsePubKey(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse commitment point: %w", err)
	}

	return &PedersenCommitment{point: pubKey}, nil
}

// Bytes returns the serialized commitment (33 bytes compressed)
func (c *PedersenCommitment) Bytes() []byte {
	return c.point.SerializeCompressed()
}

// Point returns the underlying elliptic curve point
func (c *PedersenCommitment) Point() *btcec.PublicKey {
	return c.point
}

// GenerateBlindingFactor creates a cryptographically secure random blinding factor
func GenerateBlindingFactor() (*BlindingFactor, error) {
	var bf BlindingFactor
	_, err := rand.Read(bf[:])
	if err != nil {
		return nil, fmt.Errorf("failed to generate blinding factor: %w", err)
	}

	// Ensure the blinding factor is valid (less than curve order)
	blindInt := new(big.Int).SetBytes(bf[:])
	if blindInt.Cmp(curveOrder) >= 0 {
		// Reduce modulo curve order
		blindInt.Mod(blindInt, curveOrder)
		copy(bf[:], blindInt.FillBytes(make([]byte, BlindingFactorSize)))
	}

	return &bf, nil
}

// BigInt returns the blinding factor as a big integer
func (bf *BlindingFactor) BigInt() *big.Int {
	return new(big.Int).SetBytes(bf[:])
}

// Bytes returns the blinding factor as bytes
func (bf *BlindingFactor) Bytes() []byte {
	result := make([]byte, BlindingFactorSize)
	copy(result, bf[:])
	return result
}

// GetValueGenerator returns the "H" generator point for values
// This is derived deterministically from the base point G
func GetValueGenerator() *btcec.PublicKey {
	// Use a standard derivation method: H = SHA256("Shell Value Generator") * G
	hasher := sha256.New()
	hasher.Write([]byte("Shell Reserve Value Generator v1.0"))
	seed := hasher.Sum(nil)

	// Convert seed to scalar and multiply by generator
	scalar := new(big.Int).SetBytes(seed)
	scalar.Mod(scalar, curveOrder)

	// Generate H = scalar * G using ScalarBaseMult
	hx, hy := btcec.S256().ScalarBaseMult(scalar.Bytes())

	// Convert to btcec.FieldVal and create public key
	var fx, fy btcec.FieldVal
	fx.SetByteSlice(hx.Bytes())
	fy.SetByteSlice(hy.Bytes())
	return btcec.NewPublicKey(&fx, &fy)
}

// CreateCommitment creates a Pedersen commitment: C = vH + rG
func CreateCommitment(value uint64, blindingFactor *BlindingFactor) (*PedersenCommitment, error) {
	if blindingFactor == nil {
		return nil, ErrInvalidBlindingFactor
	}

	// Get value generator H
	H := GetValueGenerator() // Value generator H

	// Convert value to big.Int
	valueBig := new(big.Int).SetUint64(value)

	// Calculate vH (value * H)
	valuePointX, valuePointY := btcec.S256().ScalarMult(
		H.X(), H.Y(),
		valueBig.Bytes(),
	)

	// Calculate rG (blinding factor * G)
	blindPointX, blindPointY := btcec.S256().ScalarBaseMult(blindingFactor.Bytes())

	// Calculate C = vH + rG
	commitX, commitY := btcec.S256().Add(
		valuePointX, valuePointY,
		blindPointX, blindPointY,
	)

	// Convert to public key
	var fx, fy btcec.FieldVal
	fx.SetByteSlice(commitX.Bytes())
	fy.SetByteSlice(commitY.Bytes())
	commitmentPoint := btcec.NewPublicKey(&fx, &fy)

	return &PedersenCommitment{point: commitmentPoint}, nil
}

// VerifyCommitment verifies that a commitment opens to the given value and blinding factor
func VerifyCommitment(commitment *PedersenCommitment, value uint64, blindingFactor *BlindingFactor) bool {
	// Recreate the commitment
	expectedCommitment, err := CreateCommitment(value, blindingFactor)
	if err != nil {
		return false
	}

	// Compare the points
	return commitment.point.IsEqual(expectedCommitment.point)
}

// AddCommitments adds two Pedersen commitments homomorphically
// Useful for verifying transaction balance: sum(inputs) = sum(outputs) + fee
func AddCommitments(c1, c2 *PedersenCommitment) *PedersenCommitment {
	// Add the points: C1 + C2
	sumX, sumY := btcec.S256().Add(
		c1.point.X(), c1.point.Y(),
		c2.point.X(), c2.point.Y(),
	)

	// Convert to public key
	var fx, fy btcec.FieldVal
	fx.SetByteSlice(sumX.Bytes())
	fy.SetByteSlice(sumY.Bytes())
	sumPoint := btcec.NewPublicKey(&fx, &fy)

	return &PedersenCommitment{point: sumPoint}
}

// SubtractCommitments subtracts the second commitment from the first
func SubtractCommitments(c1, c2 *PedersenCommitment) *PedersenCommitment {
	// Negate c2 by negating its Y coordinate
	negY := new(big.Int).Neg(c2.point.Y())
	negY.Mod(negY, btcec.S256().P)

	// Add c1 + (-c2)
	sumX, sumY := btcec.S256().Add(
		c1.point.X(), c1.point.Y(),
		c2.point.X(), negY,
	)

	// Convert to public key
	var fx, fy btcec.FieldVal
	fx.SetByteSlice(sumX.Bytes())
	fy.SetByteSlice(sumY.Bytes())
	diffPoint := btcec.NewPublicKey(&fx, &fy)

	return &PedersenCommitment{point: diffPoint}
}

// Hash returns a hash of the commitment for use in other protocols
func (c *PedersenCommitment) Hash() chainhash.Hash {
	return chainhash.DoubleHashH(c.Bytes())
}

// String returns a hex representation of the commitment
func (c *PedersenCommitment) String() string {
	return fmt.Sprintf("%x", c.Bytes())
}

// IsEqual checks if two commitments are equal
func (c *PedersenCommitment) IsEqual(other *PedersenCommitment) bool {
	if other == nil {
		return false
	}
	return c.point.IsEqual(other.point)
}
