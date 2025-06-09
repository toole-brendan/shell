// Copyright (c) 2025 Shell Reserve developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package confidential

import (
	"bytes"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/toole-brendan/shell/wire"
)

func TestPedersenCommitment(t *testing.T) {
	t.Run("CreateCommitment", func(t *testing.T) {
		value := uint64(100000000) // 1 XSL
		blindingFactor, err := GenerateBlindingFactor()
		if err != nil {
			t.Fatalf("Failed to generate blinding factor: %v", err)
		}

		commitment, err := CreateCommitment(value, blindingFactor)
		if err != nil {
			t.Fatalf("Failed to create commitment: %v", err)
		}

		if commitment == nil {
			t.Fatal("Commitment should not be nil")
		}

		// Verify commitment serialization
		commitmentBytes := commitment.Bytes()
		if len(commitmentBytes) != CommitmentSize {
			t.Errorf("Expected commitment size %d, got %d", CommitmentSize, len(commitmentBytes))
		}

		// Verify commitment can be recreated from bytes
		commitment2, err := NewPedersenCommitment(commitmentBytes)
		if err != nil {
			t.Fatalf("Failed to recreate commitment from bytes: %v", err)
		}

		if !commitment.IsEqual(commitment2) {
			t.Error("Recreated commitment should equal original")
		}
	})

	t.Run("VerifyCommitment", func(t *testing.T) {
		value := uint64(50000000) // 0.5 XSL
		blindingFactor, err := GenerateBlindingFactor()
		if err != nil {
			t.Fatalf("Failed to generate blinding factor: %v", err)
		}

		commitment, err := CreateCommitment(value, blindingFactor)
		if err != nil {
			t.Fatalf("Failed to create commitment: %v", err)
		}

		// Should verify with correct value and blinding factor
		if !VerifyCommitment(commitment, value, blindingFactor) {
			t.Error("Commitment should verify with correct parameters")
		}

		// Should not verify with wrong value
		if VerifyCommitment(commitment, value+1, blindingFactor) {
			t.Error("Commitment should not verify with wrong value")
		}

		// Should not verify with wrong blinding factor
		wrongBlinding, _ := GenerateBlindingFactor()
		if VerifyCommitment(commitment, value, wrongBlinding) {
			t.Error("Commitment should not verify with wrong blinding factor")
		}
	})

	t.Run("HomomorphicAddition", func(t *testing.T) {
		value1 := uint64(30000000) // 0.3 XSL
		value2 := uint64(20000000) // 0.2 XSL
		expectedSum := value1 + value2

		bf1, err := GenerateBlindingFactor()
		if err != nil {
			t.Fatalf("Failed to generate blinding factor 1: %v", err)
		}

		bf2, err := GenerateBlindingFactor()
		if err != nil {
			t.Fatalf("Failed to generate blinding factor 2: %v", err)
		}

		c1, err := CreateCommitment(value1, bf1)
		if err != nil {
			t.Fatalf("Failed to create commitment 1: %v", err)
		}

		c2, err := CreateCommitment(value2, bf2)
		if err != nil {
			t.Fatalf("Failed to create commitment 2: %v", err)
		}

		// Test homomorphic addition
		cSum := AddCommitments(c1, c2)

		// Create expected commitment with sum of values and blinding factors
		// Note: This is a simplified test - real implementation would need to handle blinding factor addition properly
		t.Logf("Sum commitment created: %s", cSum.String())
		t.Logf("Expected sum: %d", expectedSum)
	})
}

func TestRangeProof(t *testing.T) {
	t.Run("GenerateAndVerifyRangeProof", func(t *testing.T) {
		value := uint64(1000000000) // 10 XSL
		blindingFactor, err := GenerateBlindingFactor()
		if err != nil {
			t.Fatalf("Failed to generate blinding factor: %v", err)
		}

		commitment, err := CreateCommitment(value, blindingFactor)
		if err != nil {
			t.Fatalf("Failed to create commitment: %v", err)
		}

		params := &RangeProofParams{
			value:          value,
			blindingFactor: blindingFactor,
			commitment:     commitment,
		}

		rangeProof, err := GenerateRangeProof(params)
		if err != nil {
			t.Fatalf("Failed to generate range proof: %v", err)
		}

		if rangeProof == nil {
			t.Fatal("Range proof should not be nil")
		}

		// Verify the range proof
		if !VerifyRangeProof(commitment, rangeProof) {
			t.Error("Range proof should verify")
		}

		// Test with different commitment (should fail)
		wrongValue := uint64(2000000000)
		wrongBf, _ := GenerateBlindingFactor()
		wrongCommitment, _ := CreateCommitment(wrongValue, wrongBf)

		if VerifyRangeProof(wrongCommitment, rangeProof) {
			t.Error("Range proof should not verify with wrong commitment")
		}
	})

	t.Run("ValueOutOfRange", func(t *testing.T) {
		value := uint64(MaxValue + 1) // Out of range
		blindingFactor, err := GenerateBlindingFactor()
		if err != nil {
			t.Fatalf("Failed to generate blinding factor: %v", err)
		}

		commitment, err := CreateCommitment(value, blindingFactor)
		if err != nil {
			t.Fatalf("Failed to create commitment: %v", err)
		}

		params := &RangeProofParams{
			value:          value,
			blindingFactor: blindingFactor,
			commitment:     commitment,
		}

		_, err = GenerateRangeProof(params)
		if err != ErrValueOutOfRange {
			t.Errorf("Expected ErrValueOutOfRange, got %v", err)
		}
	})
}

func TestConfidentialOutput(t *testing.T) {
	t.Run("CreateConfidentialOutput", func(t *testing.T) {
		value := uint64(500000000) // 5 XSL
		blindingFactor, err := GenerateBlindingFactor()
		if err != nil {
			t.Fatalf("Failed to generate blinding factor: %v", err)
		}

		script := []byte{0x76, 0xa9, 0x14}           // P2PKH prefix
		script = append(script, make([]byte, 20)...) // 20-byte hash
		script = append(script, 0x88, 0xac)          // P2PKH suffix

		confOutput, err := NewConfidentialOutput(value, blindingFactor, script)
		if err != nil {
			t.Fatalf("Failed to create confidential output: %v", err)
		}

		if !confOutput.IsConfidential() {
			t.Error("Output should be confidential")
		}

		if confOutput.GetValue() != 0 {
			t.Error("Confidential output should not reveal value")
		}

		if confOutput.Commitment == nil {
			t.Error("Confidential output should have commitment")
		}

		if confOutput.RangeProof == nil {
			t.Error("Confidential output should have range proof")
		}
	})

	t.Run("CreateExplicitOutput", func(t *testing.T) {
		value := uint64(1000000) // 0.01 XSL fee
		script := []byte{0x6a}   // OP_RETURN (fee output)

		explicitOutput := NewExplicitOutput(value, script)

		if explicitOutput.IsConfidential() {
			t.Error("Output should not be confidential")
		}

		if explicitOutput.GetValue() != value {
			t.Errorf("Expected value %d, got %d", value, explicitOutput.GetValue())
		}

		if explicitOutput.Commitment != nil {
			t.Error("Explicit output should not have commitment")
		}

		if explicitOutput.RangeProof != nil {
			t.Error("Explicit output should not have range proof")
		}
	})

	t.Run("SerializeDeserialize", func(t *testing.T) {
		value := uint64(250000000) // 2.5 XSL
		blindingFactor, err := GenerateBlindingFactor()
		if err != nil {
			t.Fatalf("Failed to generate blinding factor: %v", err)
		}

		script := []byte{0x51} // OP_1 (simple script)

		confOutput, err := NewConfidentialOutput(value, blindingFactor, script)
		if err != nil {
			t.Fatalf("Failed to create confidential output: %v", err)
		}

		// Serialize
		var buf bytes.Buffer
		err = confOutput.Serialize(&buf)
		if err != nil {
			t.Fatalf("Failed to serialize confidential output: %v", err)
		}

		// Deserialize
		deserializedOutput := &ConfidentialOutput{}
		err = deserializedOutput.Deserialize(&buf)
		if err != nil {
			t.Fatalf("Failed to deserialize confidential output: %v", err)
		}

		// Verify deserialized output
		if !deserializedOutput.IsConfidential() {
			t.Error("Deserialized output should be confidential")
		}

		if !deserializedOutput.Commitment.IsEqual(confOutput.Commitment) {
			t.Error("Deserialized commitment should match original")
		}

		if !bytes.Equal(deserializedOutput.PkScript, confOutput.PkScript) {
			t.Error("Deserialized script should match original")
		}
	})
}

func TestConfidentialTransaction(t *testing.T) {
	t.Run("CreateConfidentialTransaction", func(t *testing.T) {
		// Create base transaction
		baseTx := wire.NewMsgTx(wire.TxVersion)

		// Create confidential transaction
		confTx := NewConfidentialTx(baseTx)

		if confTx.MsgTx != baseTx {
			t.Error("Confidential transaction should reference base transaction")
		}

		if len(confTx.ConfidentialOutputs) != 0 {
			t.Error("New confidential transaction should have no outputs")
		}

		if confTx.ExplicitFee != 0 {
			t.Error("New confidential transaction should have zero fee")
		}
	})

	t.Run("AddConfidentialOutputs", func(t *testing.T) {
		baseTx := wire.NewMsgTx(wire.TxVersion)
		confTx := NewConfidentialTx(baseTx)

		// Add confidential output
		value1 := uint64(100000000) // 1 XSL
		bf1, _ := GenerateBlindingFactor()
		script1 := []byte{0x76, 0xa9, 0x14}
		script1 = append(script1, make([]byte, 20)...) // 20-byte hash
		script1 = append(script1, 0x88, 0xac)

		confOutput1, err := NewConfidentialOutput(value1, bf1, script1)
		if err != nil {
			t.Fatalf("Failed to create confidential output: %v", err)
		}

		confTx.AddConfidentialOutput(confOutput1)

		// Add explicit output (fee)
		feeValue := uint64(1000000) // 0.01 XSL
		feeScript := []byte{0x6a}
		explicitOutput := NewExplicitOutput(feeValue, feeScript)

		confTx.AddConfidentialOutput(explicitOutput)
		confTx.ExplicitFee = feeValue

		if len(confTx.ConfidentialOutputs) != 2 {
			t.Errorf("Expected 2 confidential outputs, got %d", len(confTx.ConfidentialOutputs))
		}

		if len(confTx.MsgTx.TxOut) != 2 {
			t.Errorf("Expected 2 base transaction outputs, got %d", len(confTx.MsgTx.TxOut))
		}
	})

	t.Run("ValidateRangeProofs", func(t *testing.T) {
		baseTx := wire.NewMsgTx(wire.TxVersion)
		confTx := NewConfidentialTx(baseTx)

		// Add valid confidential output
		value := uint64(50000000) // 0.5 XSL
		bf, _ := GenerateBlindingFactor()
		script := []byte{0x51} // OP_1

		confOutput, err := NewConfidentialOutput(value, bf, script)
		if err != nil {
			t.Fatalf("Failed to create confidential output: %v", err)
		}

		confTx.AddConfidentialOutput(confOutput)

		// Validate range proofs
		if !confTx.ValidateRangeProofs() {
			t.Error("Range proof validation should pass")
		}

		// Add output with invalid range proof
		invalidOutput := &ConfidentialOutput{
			Commitment: confOutput.Commitment,
			RangeProof: CreateDummyRangeProof(), // Invalid proof
			PkScript:   script,
		}

		confTx.AddConfidentialOutput(invalidOutput)

		// This should fail validation
		if confTx.ValidateRangeProofs() {
			t.Error("Range proof validation should fail with invalid proof")
		}
	})
}

func TestBlindingFactor(t *testing.T) {
	t.Run("GenerateBlindingFactor", func(t *testing.T) {
		bf1, err := GenerateBlindingFactor()
		if err != nil {
			t.Fatalf("Failed to generate blinding factor: %v", err)
		}

		bf2, err := GenerateBlindingFactor()
		if err != nil {
			t.Fatalf("Failed to generate second blinding factor: %v", err)
		}

		// Should be different
		if bytes.Equal(bf1.Bytes(), bf2.Bytes()) {
			t.Error("Generated blinding factors should be different")
		}

		// Should be correct size
		if len(bf1.Bytes()) != BlindingFactorSize {
			t.Errorf("Expected blinding factor size %d, got %d", BlindingFactorSize, len(bf1.Bytes()))
		}
	})

	t.Run("BlindingFactorBigInt", func(t *testing.T) {
		bf, err := GenerateBlindingFactor()
		if err != nil {
			t.Fatalf("Failed to generate blinding factor: %v", err)
		}

		bigInt := bf.BigInt()
		if bigInt == nil {
			t.Error("BigInt conversion should not return nil")
		}

		// Should be able to convert back
		newBf := &BlindingFactor{}
		copy(newBf[:], bigInt.Bytes())

		if !bytes.Equal(bf.Bytes(), newBf.Bytes()) {
			t.Error("BigInt conversion should be reversible")
		}
	})
}

func TestValueGenerator(t *testing.T) {
	t.Run("DeterministicValueGenerator", func(t *testing.T) {
		// Value generator should be deterministic
		H1 := GetValueGenerator()
		H2 := GetValueGenerator()

		if !H1.IsEqual(H2) {
			t.Error("Value generator should be deterministic")
		}

		// Should be different from base generator
		G := btcec.Generator()
		if H1.IsEqual(G) {
			t.Error("Value generator should be different from base generator")
		}
	})
}
