// Package txscript provides Shell Reserve script parsing utilities
// for extracting parameters from Shell-specific opcodes.
package txscript

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/wire"
	"github.com/toole-brendan/shell/settlement/channels"
	"github.com/toole-brendan/shell/settlement/claimable"
)

// ShellScriptParams contains extracted parameters from Shell opcodes
type ShellScriptParams struct {
	// Channel parameters
	ChannelAlice    *btcec.PublicKey
	ChannelBob      *btcec.PublicKey
	ChannelAmount   uint64
	ChannelID       channels.ChannelID
	ChannelBalances [2]uint64
	ChannelNonce    uint64

	// Claimable balance parameters
	ClaimableAmount    uint64
	ClaimableClaimants []claimable.Claimant
	ClaimableID        claimable.ClaimableID
	ClaimableClaimer   *btcec.PublicKey
	ClaimableProof     claimable.ClaimProof
}

// ExtractChannelOpenParams extracts parameters from OP_CHANNEL_OPEN script
func ExtractChannelOpenParams(script []byte, witness wire.TxWitness) (*ShellScriptParams, error) {
	// For OP_CHANNEL_OPEN, parameters are in witness:
	// [alice_pubkey] [bob_pubkey] [amount] [signatures...]

	if len(witness) < 3 {
		return nil, errors.New("insufficient witness items for channel open")
	}

	// Parse Alice's public key
	aliceBytes := witness[0]
	alice, err := btcec.ParsePubKey(aliceBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid alice public key: %v", err)
	}

	// Parse Bob's public key
	bobBytes := witness[1]
	bob, err := btcec.ParsePubKey(bobBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid bob public key: %v", err)
	}

	// Parse amount
	amountBytes := witness[2]
	if len(amountBytes) != 8 {
		return nil, fmt.Errorf("invalid amount length: expected 8, got %d", len(amountBytes))
	}
	amount := binary.LittleEndian.Uint64(amountBytes)

	if amount == 0 {
		return nil, errors.New("channel amount must be positive")
	}

	return &ShellScriptParams{
		ChannelAlice:  alice,
		ChannelBob:    bob,
		ChannelAmount: amount,
	}, nil
}

// ExtractChannelUpdateParams extracts parameters from OP_CHANNEL_UPDATE script
func ExtractChannelUpdateParams(script []byte, witness wire.TxWitness) (*ShellScriptParams, error) {
	// For OP_CHANNEL_UPDATE, parameters are in witness:
	// [channel_id] [balance_a] [balance_b] [nonce] [signatures...]

	if len(witness) < 4 {
		return nil, errors.New("insufficient witness items for channel update")
	}

	// Parse channel ID
	channelIDBytes := witness[0]
	if len(channelIDBytes) != 32 {
		return nil, fmt.Errorf("invalid channel ID length: expected 32, got %d", len(channelIDBytes))
	}
	var channelID channels.ChannelID
	copy(channelID[:], channelIDBytes)

	// Parse balance A
	balanceABytes := witness[1]
	if len(balanceABytes) != 8 {
		return nil, fmt.Errorf("invalid balance A length: expected 8, got %d", len(balanceABytes))
	}
	balanceA := binary.LittleEndian.Uint64(balanceABytes)

	// Parse balance B
	balanceBBytes := witness[2]
	if len(balanceBBytes) != 8 {
		return nil, fmt.Errorf("invalid balance B length: expected 8, got %d", len(balanceBBytes))
	}
	balanceB := binary.LittleEndian.Uint64(balanceBBytes)

	// Parse nonce
	nonceBytes := witness[3]
	if len(nonceBytes) != 8 {
		return nil, fmt.Errorf("invalid nonce length: expected 8, got %d", len(nonceBytes))
	}
	nonce := binary.LittleEndian.Uint64(nonceBytes)

	return &ShellScriptParams{
		ChannelID:       channelID,
		ChannelBalances: [2]uint64{balanceA, balanceB},
		ChannelNonce:    nonce,
	}, nil
}

// ExtractChannelCloseParams extracts parameters from OP_CHANNEL_CLOSE script
func ExtractChannelCloseParams(script []byte, witness wire.TxWitness) (*ShellScriptParams, error) {
	// For OP_CHANNEL_CLOSE, parameters are in witness:
	// [channel_id] [signatures...]

	if len(witness) < 1 {
		return nil, errors.New("insufficient witness items for channel close")
	}

	// Parse channel ID
	channelIDBytes := witness[0]
	if len(channelIDBytes) != 32 {
		return nil, fmt.Errorf("invalid channel ID length: expected 32, got %d", len(channelIDBytes))
	}
	var channelID channels.ChannelID
	copy(channelID[:], channelIDBytes)

	return &ShellScriptParams{
		ChannelID: channelID,
	}, nil
}

// ExtractClaimableCreateParams extracts parameters from OP_CLAIMABLE_CREATE script
func ExtractClaimableCreateParams(script []byte, witness wire.TxWitness) (*ShellScriptParams, error) {
	// For OP_CLAIMABLE_CREATE, parameters are in witness:
	// [amount] [num_claimants] [claimant_data...] [signature]

	if len(witness) < 3 {
		return nil, errors.New("insufficient witness items for claimable create")
	}

	// Parse amount
	amountBytes := witness[0]
	if len(amountBytes) != 8 {
		return nil, fmt.Errorf("invalid amount length: expected 8, got %d", len(amountBytes))
	}
	amount := binary.LittleEndian.Uint64(amountBytes)

	if amount == 0 {
		return nil, errors.New("claimable amount must be positive")
	}

	// Parse number of claimants
	numClaimantsBytes := witness[1]
	if len(numClaimantsBytes) != 1 {
		return nil, errors.New("invalid num claimants format")
	}
	numClaimants := int(numClaimantsBytes[0])

	if numClaimants == 0 {
		return nil, errors.New("must have at least one claimant")
	}

	// Parse claimant data (simplified - real implementation would be more complex)
	claimants := make([]claimable.Claimant, 0, numClaimants)
	witnessIdx := 2

	for i := 0; i < numClaimants; i++ {
		if witnessIdx >= len(witness) {
			return nil, errors.New("insufficient witness data for claimants")
		}

		// Parse destination public key
		destBytes := witness[witnessIdx]
		dest, err := btcec.ParsePubKey(destBytes)
		if err != nil {
			return nil, fmt.Errorf("invalid destination public key for claimant %d: %v", i, err)
		}
		witnessIdx++

		// For now, use unconditional predicate (simplified)
		// Real implementation would parse predicate from witness
		predicate := claimable.UnconditionalPredicate()

		claimants = append(claimants, claimable.Claimant{
			Destination: dest,
			Predicate:   predicate,
		})
	}

	return &ShellScriptParams{
		ClaimableAmount:    amount,
		ClaimableClaimants: claimants,
	}, nil
}

// ExtractClaimableClaimParams extracts parameters from OP_CLAIMABLE_CLAIM script
func ExtractClaimableClaimParams(script []byte, witness wire.TxWitness) (*ShellScriptParams, error) {
	// For OP_CLAIMABLE_CLAIM, parameters are in witness:
	// [balance_id] [claimer_pubkey] [proof_data] [signature]

	if len(witness) < 3 {
		return nil, errors.New("insufficient witness items for claimable claim")
	}

	// Parse balance ID
	balanceIDBytes := witness[0]
	if len(balanceIDBytes) != 32 {
		return nil, fmt.Errorf("invalid balance ID length: expected 32, got %d", len(balanceIDBytes))
	}
	var balanceID claimable.ClaimableID
	copy(balanceID[:], balanceIDBytes)

	// Parse claimer public key
	claimerBytes := witness[1]
	claimer, err := btcec.ParsePubKey(claimerBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid claimer public key: %v", err)
	}

	// Parse proof data (simplified for now)
	proofBytes := witness[2]
	proof := claimable.ClaimProof{
		Preimages: make(map[[32]byte][]byte),
		Timestamp: 0, // Would be filled from block context
	}

	// If proof is not empty, assume it's a hash preimage
	if len(proofBytes) > 0 {
		// For now, just store the raw proof data
		// Real implementation would parse structured proof data
	}

	return &ShellScriptParams{
		ClaimableID:      balanceID,
		ClaimableClaimer: claimer,
		ClaimableProof:   proof,
	}, nil
}

// DetectShellOpcode scans a script for Shell-specific opcodes
func DetectShellOpcode(script []byte) (byte, bool) {
	shellOpcodes := []byte{
		OP_CHANNEL_OPEN,
		OP_CHANNEL_UPDATE,
		OP_CHANNEL_CLOSE,
		OP_CLAIMABLE_CREATE,
		OP_CLAIMABLE_CLAIM,
	}

	for _, opcode := range shellOpcodes {
		for _, scriptByte := range script {
			if scriptByte == opcode {
				return opcode, true
			}
		}
	}

	return 0, false
}
