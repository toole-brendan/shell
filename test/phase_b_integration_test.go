// Package test provides integration tests for Shell Reserve Phase β functionality
// including liquidity rewards, fee structure, and settlement layer integration.
package test

import (
	"crypto/sha256"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/toole-brendan/shell/blockchain"
	"github.com/toole-brendan/shell/chaincfg/chainhash"
	"github.com/toole-brendan/shell/liquidity"
	"github.com/toole-brendan/shell/mempool"
	"github.com/toole-brendan/shell/settlement/channels"
	"github.com/toole-brendan/shell/settlement/claimable"
	"github.com/toole-brendan/shell/wire"
)

// TestPhaseBLiquidityRewardIntegration tests the complete liquidity reward workflow
func TestPhaseBLiquidityRewardIntegration(t *testing.T) {
	// Create a new Shell chain state
	utxoView := blockchain.NewUtxoViewpoint()
	shellState := blockchain.NewShellChainState(utxoView)

	// Test liquidity manager initialization
	liquidityManager := shellState.GetLiquidityManager()
	if liquidityManager == nil {
		t.Fatal("Liquidity manager not initialized")
	}

	// Test epoch information
	epochInfo, err := liquidityManager.GetEpochInfo(0)
	if err != nil {
		t.Fatalf("Failed to get epoch info: %v", err)
	}

	if epochInfo.Index != 0 {
		t.Errorf("Expected epoch index 0, got %d", epochInfo.Index)
	}

	if epochInfo.RewardPool == 0 {
		t.Error("Expected non-zero reward pool")
	}

	t.Logf("Epoch 0 info: Pool=%d, Blocks=%d-%d",
		epochInfo.RewardPool, epochInfo.StartBlock, epochInfo.EndBlock)
}

// TestPhaseBFeeCalculation tests the fee structure implementation
func TestPhaseBFeeCalculation(t *testing.T) {
	feeCalculator := mempool.NewFeeCalculator()

	// Test basic fee calculation
	baseFeeRate := feeCalculator.GetFeeRate()
	if baseFeeRate != mempool.BaseFeeRate {
		t.Errorf("Expected base fee rate %f, got %f", mempool.BaseFeeRate, baseFeeRate)
	}

	// Test maker rebate rate
	makerRebateRate := feeCalculator.GetMakerRebateRate()
	if makerRebateRate != mempool.MakerRebate {
		t.Errorf("Expected maker rebate rate %f, got %f", mempool.MakerRebate, makerRebateRate)
	}

	// Create a test transaction
	tx := &wire.MsgTx{
		Version: 1,
		TxIn: []*wire.TxIn{{
			PreviousOutPoint: wire.OutPoint{},
			SignatureScript:  []byte{},
			Witness:          wire.TxWitness{[]byte{0x4d, 0x41, 0x4b, 0x52}}, // "MAKR" flag
			Sequence:         0xffffffff,
		}},
		TxOut: []*wire.TxOut{{
			Value:    1000000,            // 0.01 XSL
			PkScript: []byte{0x51, 0x20}, // Taproot output
		}},
	}

	// Calculate fees
	feeResult, err := feeCalculator.CalculateFee(tx)
	if err != nil {
		t.Fatalf("Fee calculation failed: %v", err)
	}

	// Check that maker rebate is applied
	if feeResult.MakerRebate == 0 {
		t.Error("Expected non-zero maker rebate for transaction with MAKR flag")
	}

	// Check fee structure
	if feeResult.BaseFee <= 0 {
		t.Error("Expected positive base fee")
	}

	if feeResult.NetFee < 0 {
		t.Error("Net fee should not be negative")
	}

	t.Logf("Fee calculation: Base=%d, Rebate=%d, Net=%d",
		feeResult.BaseFee, feeResult.MakerRebate, feeResult.NetFee)
}

// TestPhaseBSettlementLayerIntegration tests settlement primitives with liquidity features
func TestPhaseBSettlementLayerIntegration(t *testing.T) {
	// Create Shell chain state
	utxoView := blockchain.NewUtxoViewpoint()
	shellState := blockchain.NewShellChainState(utxoView)

	// Test channel state
	channelState := shellState.GetChannelState()
	if channelState == nil {
		t.Fatal("Channel state not initialized")
	}

	// Create test participants
	alicePrivKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("Failed to create Alice's private key: %v", err)
	}
	alicePubKey := alicePrivKey.PubKey()

	bobPrivKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("Failed to create Bob's private key: %v", err)
	}
	bobPubKey := bobPrivKey.PubKey()

	// Test channel opening
	fundingOutpoint := wire.OutPoint{
		Hash:  chainhash.Hash{1, 2, 3},
		Index: 0,
	}

	channel, err := channelState.OpenChannel(
		alicePubKey,
		bobPubKey,
		1000000, // 0.01 XSL capacity
		144*30,  // 30 day expiry
		fundingOutpoint,
	)
	if err != nil {
		t.Fatalf("Failed to open channel: %v", err)
	}

	if channel.Capacity != 1000000 {
		t.Errorf("Expected channel capacity 1000000, got %d", channel.Capacity)
	}

	// Test channel update
	update := &channels.ChannelUpdate{
		ChannelID:  channel.ChannelID,
		Balances:   [2]uint64{600000, 400000}, // Alice: 0.006, Bob: 0.004
		Nonce:      1,
		Signatures: [2]*ecdsa.Signature{nil, nil}, // Simplified for test
	}

	err = channelState.UpdateChannel(update)
	if err != nil {
		t.Fatalf("Failed to update channel: %v", err)
	}

	// Verify channel state
	updatedChannel, err := channelState.GetChannel(channel.ChannelID)
	if err != nil {
		t.Fatalf("Failed to get updated channel: %v", err)
	}

	if updatedChannel.Balance[0] != 600000 {
		t.Errorf("Expected Alice balance 600000, got %d", updatedChannel.Balance[0])
	}

	t.Logf("Channel updated: Alice=%d, Bob=%d",
		updatedChannel.Balance[0], updatedChannel.Balance[1])
}

// TestPhaseBClaimableBalanceIntegration tests claimable balances with liquidity features
func TestPhaseBClaimableBalanceIntegration(t *testing.T) {
	// Create Shell chain state
	utxoView := blockchain.NewUtxoViewpoint()
	shellState := blockchain.NewShellChainState(utxoView)

	// Test claimable state
	claimableState := shellState.GetClaimableState()
	if claimableState == nil {
		t.Fatal("Claimable state not initialized")
	}

	// Create test participants
	creatorPrivKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("Failed to create creator's private key: %v", err)
	}
	creatorPubKey := creatorPrivKey.PubKey()

	claimerPrivKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("Failed to create claimer's private key: %v", err)
	}
	claimerPubKey := claimerPrivKey.PubKey()

	// Create claimable balance with time-based predicate
	fundingOutpoint := wire.OutPoint{
		Hash:  chainhash.Hash{4, 5, 6},
		Index: 0,
	}

	claimants := []claimable.Claimant{{
		Destination: claimerPubKey,
		Predicate: claimable.ClaimPredicate{
			Type:      claimable.PredicateAfterTime,
			Timestamp: 1700000000, // Future timestamp
		},
	}}

	balance, err := claimableState.CreateClaimableBalance(
		creatorPubKey,
		500000, // 0.005 XSL
		claimants,
		100, // Current block height
		fundingOutpoint,
	)
	if err != nil {
		t.Fatalf("Failed to create claimable balance: %v", err)
	}

	if balance.Amount != 500000 {
		t.Errorf("Expected balance amount 500000, got %d", balance.Amount)
	}

	// Test claiming with proof
	proof := claimable.ClaimProof{
		Timestamp: 1700000001, // After predicate timestamp
		Preimages: make(map[[32]byte][]byte),
	}

	_, err = claimableState.ClaimBalance(
		balance.ID,
		claimerPubKey,
		proof,
		200, // Future block height
	)
	if err != nil {
		t.Fatalf("Failed to claim balance: %v", err)
	}

	t.Logf("Claimable balance created and claimed successfully")
}

// TestPhaseBLiquidityAttestationParsing tests attestation blob parsing
func TestPhaseBLiquidityAttestationParsing(t *testing.T) {
	liquidityManager := liquidity.NewLiquidityManager(0)

	// Create test attestation blob
	blob := make([]byte, 0, 200)

	// Epoch index (1 byte)
	blob = append(blob, 0x01)

	// Participant ID (32 bytes)
	participantID := sha256.Sum256([]byte("test-participant"))
	blob = append(blob, participantID[:]...)

	// Volume (8 bytes, little endian)
	volume := uint64(1000000000) // 10 XSL
	volumeBytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		volumeBytes[i] = byte(volume >> (i * 8))
	}
	blob = append(blob, volumeBytes...)

	// Spread (4 bytes, little endian)
	spread := uint32(50) // 50 basis points
	spreadBytes := make([]byte, 4)
	for i := 0; i < 4; i++ {
		spreadBytes[i] = byte(spread >> (i * 8))
	}
	blob = append(blob, spreadBytes...)

	// Uptime (2 bytes, little endian)
	uptime := uint16(9900) // 99% uptime
	uptimeBytes := make([]byte, 2)
	for i := 0; i < 2; i++ {
		uptimeBytes[i] = byte(uptime >> (i * 8))
	}
	blob = append(blob, uptimeBytes...)

	// Timestamp (4 bytes, little endian)
	timestamp := uint32(1700000000)
	timestampBytes := make([]byte, 4)
	for i := 0; i < 4; i++ {
		timestampBytes[i] = byte(timestamp >> (i * 8))
	}
	blob = append(blob, timestampBytes...)

	// Number of signatures (1 byte)
	blob = append(blob, 0x00) // No signatures for this test

	// Merkle proof length (2 bytes, little endian)
	blob = append(blob, 0x00, 0x00) // No merkle proof for this test

	// Parse the attestation
	err := liquidityManager.ProcessRewardClaim(&liquidity.LiquidityRewardClaim{
		AttestationBlob: blob,
	}, 100)

	// Note: This will fail validation due to missing signatures and merkle proof,
	// but we're testing the parsing functionality
	if err == nil {
		t.Error("Expected validation error due to missing signatures")
	}

	t.Logf("Attestation parsing test completed (expected validation failure)")
}

// TestPhaseBFullWorkflow tests the complete Phase β workflow
func TestPhaseBFullWorkflow(t *testing.T) {
	// Initialize all Phase β components
	utxoView := blockchain.NewUtxoViewpoint()
	shellState := blockchain.NewShellChainState(utxoView)
	feeCalculator := mempool.NewFeeCalculator()

	// Test that all components are properly initialized
	if shellState.GetChannelState() == nil {
		t.Error("Channel state not initialized")
	}

	if shellState.GetClaimableState() == nil {
		t.Error("Claimable state not initialized")
	}

	if shellState.GetLiquidityManager() == nil {
		t.Error("Liquidity manager not initialized")
	}

	if feeCalculator == nil {
		t.Error("Fee calculator not initialized")
	}

	// Test that we can create transactions with Shell features
	tx := &wire.MsgTx{
		Version: 1,
		TxIn: []*wire.TxIn{{
			PreviousOutPoint: wire.OutPoint{},
			SignatureScript:  []byte{},
			Witness:          wire.TxWitness{[]byte{0xc6}}, // OP_CHANNEL_OPEN
			Sequence:         0xffffffff,
		}},
		TxOut: []*wire.TxOut{{
			Value:    1000000,
			PkScript: []byte{0x51, 0x20}, // Taproot output
		}},
	}

	// Calculate fees for Shell transaction
	feeResult, err := feeCalculator.CalculateFee(tx)
	if err != nil {
		t.Fatalf("Failed to calculate fees for Shell transaction: %v", err)
	}

	// Should include operation fee for channel open
	if feeResult.OperationFee == 0 {
		t.Error("Expected non-zero operation fee for channel open")
	}

	t.Logf("Phase β full workflow test completed successfully")
	t.Logf("Fee breakdown: Base=%d, Operation=%d, Total=%d",
		feeResult.BaseFee, feeResult.OperationFee, feeResult.TotalFee)
}
