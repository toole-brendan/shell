// Copyright (c) 2025 Shell Reserve developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package confidential

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/toole-brendan/shell/wire"
)

var (
	// ErrInvalidConfidentialOutput is returned when a confidential output is malformed
	ErrInvalidConfidentialOutput = errors.New("invalid confidential output")

	// ErrBalanceValidationFailed is returned when transaction balance doesn't verify
	ErrBalanceValidationFailed = errors.New("confidential transaction balance validation failed")

	// ErrInsufficientBlindingData is returned when blinding data is incomplete
	ErrInsufficientBlindingData = errors.New("insufficient blinding data for transaction")
)

// ConfidentialOutput represents a transaction output with hidden amounts
type ConfidentialOutput struct {
	// Commitment to the output value (replaces the Value field)
	Commitment *PedersenCommitment

	// Range proof that the committed value is in valid range [0, 2^52)
	RangeProof *RangeProof

	// Standard script for spending conditions
	PkScript []byte

	// Optional explicit value (for fee outputs or non-confidential outputs)
	// If set to nil, the output is fully confidential
	ExplicitValue *uint64
}

// ConfidentialTx represents a transaction with confidential outputs
type ConfidentialTx struct {
	// Base transaction structure
	*wire.MsgTx

	// Confidential outputs (parallel to TxOut slice)
	ConfidentialOutputs []*ConfidentialOutput

	// Fee commitment (fee must be explicit for miners)
	FeeCommitment *PedersenCommitment

	// Explicit fee amount (must be public for miner incentives)
	ExplicitFee uint64
}

// NewConfidentialOutput creates a new confidential output
func NewConfidentialOutput(value uint64, blindingFactor *BlindingFactor, pkScript []byte) (*ConfidentialOutput, error) {
	// Create commitment
	commitment, err := CreateCommitment(value, blindingFactor)
	if err != nil {
		return nil, fmt.Errorf("failed to create commitment: %w", err)
	}

	// Generate range proof
	proofParams := &RangeProofParams{
		value:          value,
		blindingFactor: blindingFactor,
		commitment:     commitment,
	}

	rangeProof, err := GenerateRangeProof(proofParams)
	if err != nil {
		return nil, fmt.Errorf("failed to generate range proof: %w", err)
	}

	return &ConfidentialOutput{
		Commitment:    commitment,
		RangeProof:    rangeProof,
		PkScript:      pkScript,
		ExplicitValue: nil, // Fully confidential
	}, nil
}

// NewExplicitOutput creates a non-confidential output (for fees, etc.)
func NewExplicitOutput(value uint64, pkScript []byte) *ConfidentialOutput {
	return &ConfidentialOutput{
		Commitment:    nil,
		RangeProof:    nil,
		PkScript:      pkScript,
		ExplicitValue: &value,
	}
}

// IsConfidential returns true if the output has a hidden value
func (co *ConfidentialOutput) IsConfidential() bool {
	return co.ExplicitValue == nil && co.Commitment != nil
}

// GetValue returns the explicit value if available, otherwise returns 0
func (co *ConfidentialOutput) GetValue() uint64 {
	if co.ExplicitValue != nil {
		return *co.ExplicitValue
	}
	return 0 // Confidential outputs don't reveal their value
}

// SerializeSize returns the size needed to serialize the confidential output
func (co *ConfidentialOutput) SerializeSize() int {
	size := 0

	// Commitment (33 bytes if present)
	if co.Commitment != nil {
		size += CommitmentSize
	} else {
		size += 1 // Flag byte indicating no commitment
	}

	// Range proof (variable size)
	if co.RangeProof != nil {
		size += wire.VarIntSerializeSize(uint64(co.RangeProof.Size()))
		size += co.RangeProof.Size()
	} else {
		size += 1 // Flag byte indicating no range proof
	}

	// Script (variable size)
	size += wire.VarIntSerializeSize(uint64(len(co.PkScript)))
	size += len(co.PkScript)

	// Explicit value (8 bytes if present)
	if co.ExplicitValue != nil {
		size += 8
	} else {
		size += 1 // Flag byte indicating no explicit value
	}

	return size
}

// Serialize writes the confidential output to a writer
func (co *ConfidentialOutput) Serialize(w io.Writer) error {
	// Write commitment
	if co.Commitment != nil {
		if _, err := w.Write([]byte{0x01}); err != nil { // Has commitment flag
			return err
		}
		if _, err := w.Write(co.Commitment.Bytes()); err != nil {
			return err
		}
	} else {
		if _, err := w.Write([]byte{0x00}); err != nil { // No commitment flag
			return err
		}
	}

	// Write range proof
	if co.RangeProof != nil {
		if _, err := w.Write([]byte{0x01}); err != nil { // Has range proof flag
			return err
		}
		if err := wire.WriteVarInt(w, 0, uint64(co.RangeProof.Size())); err != nil {
			return err
		}
		if _, err := w.Write(co.RangeProof.Bytes()); err != nil {
			return err
		}
	} else {
		if _, err := w.Write([]byte{0x00}); err != nil { // No range proof flag
			return err
		}
	}

	// Write script
	if err := wire.WriteVarBytes(w, 0, co.PkScript); err != nil {
		return err
	}

	// Write explicit value
	if co.ExplicitValue != nil {
		if _, err := w.Write([]byte{0x01}); err != nil { // Has explicit value flag
			return err
		}
		// Write uint64 value using binary encoding
		var valueBytes [8]byte
		binary.LittleEndian.PutUint64(valueBytes[:], *co.ExplicitValue)
		_, err := w.Write(valueBytes[:])
		return err
	} else {
		_, err := w.Write([]byte{0x00}) // No explicit value flag
		return err
	}
}

// Deserialize reads a confidential output from a reader
func (co *ConfidentialOutput) Deserialize(r io.Reader) error {
	// Read commitment flag
	var commitmentFlag [1]byte
	if _, err := io.ReadFull(r, commitmentFlag[:]); err != nil {
		return err
	}

	if commitmentFlag[0] == 0x01 {
		// Read commitment
		commitmentData := make([]byte, CommitmentSize)
		if _, err := io.ReadFull(r, commitmentData); err != nil {
			return err
		}
		var err error
		co.Commitment, err = NewPedersenCommitment(commitmentData)
		if err != nil {
			return err
		}
	}

	// Read range proof flag
	var proofFlag [1]byte
	if _, err := io.ReadFull(r, proofFlag[:]); err != nil {
		return err
	}

	if proofFlag[0] == 0x01 {
		// Read range proof size
		proofSize, err := wire.ReadVarInt(r, 0)
		if err != nil {
			return err
		}

		// Read range proof data
		proofData := make([]byte, proofSize)
		if _, err := io.ReadFull(r, proofData); err != nil {
			return err
		}

		co.RangeProof, err = NewRangeProof(proofData)
		if err != nil {
			return err
		}
	}

	// Read script
	script, err := wire.ReadVarBytes(r, 0, wire.MaxMessagePayload, "script")
	if err != nil {
		return err
	}
	co.PkScript = script

	// Read explicit value flag
	var valueFlag [1]byte
	if _, err := io.ReadFull(r, valueFlag[:]); err != nil {
		return err
	}

	if valueFlag[0] == 0x01 {
		// Read explicit value using binary encoding
		var valueBytes [8]byte
		if _, err := io.ReadFull(r, valueBytes[:]); err != nil {
			return err
		}
		value := binary.LittleEndian.Uint64(valueBytes[:])
		co.ExplicitValue = &value
	}

	return nil
}

// NewConfidentialTx creates a new confidential transaction from a base transaction
func NewConfidentialTx(baseTx *wire.MsgTx) *ConfidentialTx {
	return &ConfidentialTx{
		MsgTx:               baseTx,
		ConfidentialOutputs: make([]*ConfidentialOutput, 0),
		ExplicitFee:         0,
	}
}

// AddConfidentialOutput adds a confidential output to the transaction
func (ctx *ConfidentialTx) AddConfidentialOutput(output *ConfidentialOutput) {
	ctx.ConfidentialOutputs = append(ctx.ConfidentialOutputs, output)

	// Also add to base transaction for compatibility
	// For confidential outputs, use a dummy value and mark with special script
	var value int64
	if output.ExplicitValue != nil {
		value = int64(*output.ExplicitValue)
	} else {
		value = 0 // Confidential output marker
	}

	baseTxOut := wire.NewTxOut(value, output.PkScript)
	ctx.MsgTx.AddTxOut(baseTxOut)
}

// ValidateBalance verifies that the transaction balance is correct using homomorphic properties
func (ctx *ConfidentialTx) ValidateBalance(inputCommitments []*PedersenCommitment, inputValues []uint64) bool {
	// Calculate sum of input commitments
	if len(inputCommitments) == 0 {
		return false
	}

	inputSum := inputCommitments[0]
	for i := 1; i < len(inputCommitments); i++ {
		inputSum = AddCommitments(inputSum, inputCommitments[i])
	}

	// Calculate sum of output commitments
	var outputSum *PedersenCommitment
	totalExplicitOutput := uint64(0)

	for _, output := range ctx.ConfidentialOutputs {
		if output.IsConfidential() {
			if outputSum == nil {
				outputSum = output.Commitment
			} else {
				outputSum = AddCommitments(outputSum, output.Commitment)
			}
		} else {
			totalExplicitOutput += output.GetValue()
		}
	}

	// Add explicit fee
	totalExplicitOutput += ctx.ExplicitFee

	// Create commitment to explicit outputs + fee
	if totalExplicitOutput > 0 {
		// Use zero blinding factor for explicit values
		zeroBlinding := &BlindingFactor{}
		explicitCommitment, err := CreateCommitment(totalExplicitOutput, zeroBlinding)
		if err != nil {
			return false
		}

		if outputSum == nil {
			outputSum = explicitCommitment
		} else {
			outputSum = AddCommitments(outputSum, explicitCommitment)
		}
	}

	// Verify: sum(inputs) = sum(outputs) + fee
	return inputSum.IsEqual(outputSum)
}

// ValidateRangeProofs validates all range proofs in the transaction
func (ctx *ConfidentialTx) ValidateRangeProofs() bool {
	for _, output := range ctx.ConfidentialOutputs {
		if output.IsConfidential() {
			if !VerifyRangeProof(output.Commitment, output.RangeProof) {
				return false
			}
		}
	}
	return true
}

// GetConfidentialSerializeSize returns the size needed to serialize confidential data
func (ctx *ConfidentialTx) GetConfidentialSerializeSize() int {
	size := 0

	// Number of confidential outputs
	size += wire.VarIntSerializeSize(uint64(len(ctx.ConfidentialOutputs)))

	// Each confidential output
	for _, output := range ctx.ConfidentialOutputs {
		size += output.SerializeSize()
	}

	// Fee commitment (if present)
	if ctx.FeeCommitment != nil {
		size += 1 + CommitmentSize // Flag + commitment
	} else {
		size += 1 // Flag only
	}

	// Explicit fee
	size += 8

	return size
}
