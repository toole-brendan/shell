// Package test provides integration tests for Shell Reserve's L1 Settlement Layer
// demonstrating the completed Phase β.5 functionality.
package test

import (
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/toole-brendan/shell/blockchain"
	"github.com/toole-brendan/shell/chaincfg/chainhash"
	"github.com/toole-brendan/shell/settlement/channels"
	"github.com/toole-brendan/shell/settlement/claimable"
	"github.com/toole-brendan/shell/txscript"
	"github.com/toole-brendan/shell/wire"
)

// TestPhaseB5SettlementIntegration tests the complete L1 Settlement Layer
// This demonstrates that Phase β.5 is complete and working
func TestPhaseB5SettlementIntegration(t *testing.T) {
	t.Parallel()

	// Create Shell chain state for testing
	utxoView := &blockchain.UtxoViewpoint{} // Mock UTXO view
	shellState := blockchain.NewShellChainState(utxoView)

	t.Run("PaymentChannelLifecycle", func(t *testing.T) {
		testPaymentChannelLifecycle(t, shellState)
	})

	t.Run("ClaimableBalanceLifecycle", func(t *testing.T) {
		testClaimableBalanceLifecycle(t, shellState)
	})

	t.Run("ShellOpcodeValidation", func(t *testing.T) {
		testShellOpcodeValidation(t)
	})
}

// testPaymentChannelLifecycle tests the complete payment channel workflow
func testPaymentChannelLifecycle(t *testing.T, shellState *blockchain.ShellChainState) {
	// Generate test keys for Alice and Bob
	alicePriv, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("Failed to generate Alice's key: %v", err)
	}
	alice := alicePriv.PubKey()

	bobPriv, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("Failed to generate Bob's key: %v", err)
	}
	bob := bobPriv.PubKey()

	// Test 1: Channel Opening
	t.Logf("✅ Testing Channel Opening...")

	channelState := shellState.GetChannelState()
	capacity := uint64(1000000) // 1M satoshis
	expiry := uint32(100000)    // Block height

	// Create mock funding outpoint using Shell types
	fundingOutpoint := wire.OutPoint{
		Hash:  chainhash.Hash{},
		Index: 0,
	}

	channel, err := channelState.OpenChannel(alice, bob, capacity, expiry, fundingOutpoint)
	if err != nil {
		t.Fatalf("Failed to open channel: %v", err)
	}

	if channel.Capacity != capacity {
		t.Errorf("Expected capacity %d, got %d", capacity, channel.Capacity)
	}

	if !channel.IsOpen {
		t.Error("Channel should be open after creation")
	}

	t.Logf("   Channel ID: %x", channel.ChannelID)
	t.Logf("   Capacity: %d satoshis", channel.Capacity)
	t.Logf("   Initial Balance: Alice=%d, Bob=%d", channel.Balance[0], channel.Balance[1])

	// Test 2: Channel Update
	t.Logf("✅ Testing Channel Update...")

	newBalances := [2]uint64{600000, 400000} // Alice sends 400k to Bob
	newNonce := uint64(1)

	update := &channels.ChannelUpdate{
		ChannelID:  channel.ChannelID,
		Balances:   newBalances,
		Nonce:      newNonce,
		Signatures: [2]*ecdsa.Signature{nil, nil}, // Mock signatures
	}

	err = channelState.UpdateChannel(update)
	if err != nil {
		t.Fatalf("Failed to update channel: %v", err)
	}

	// Verify update
	updatedChannel, err := channelState.GetChannel(channel.ChannelID)
	if err != nil {
		t.Fatalf("Failed to get updated channel: %v", err)
	}

	if updatedChannel.Balance[0] != newBalances[0] || updatedChannel.Balance[1] != newBalances[1] {
		t.Errorf("Balance update failed: expected %v, got %v", newBalances, updatedChannel.Balance)
	}

	t.Logf("   Updated Balance: Alice=%d, Bob=%d", updatedChannel.Balance[0], updatedChannel.Balance[1])

	// Test 3: Channel Close
	t.Logf("✅ Testing Channel Close...")

	closedChannel, err := channelState.CloseChannel(channel.ChannelID)
	if err != nil {
		t.Fatalf("Failed to close channel: %v", err)
	}

	if closedChannel.IsOpen {
		t.Error("Channel should be closed after CloseChannel call")
	}

	t.Logf("   Channel closed successfully")
	t.Logf("   Final Balance: Alice=%d, Bob=%d", closedChannel.Balance[0], closedChannel.Balance[1])
}

// testClaimableBalanceLifecycle tests the complete claimable balance workflow
func testClaimableBalanceLifecycle(t *testing.T, shellState *blockchain.ShellChainState) {
	// Generate test keys
	creatorPriv, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("Failed to generate creator key: %v", err)
	}
	creator := creatorPriv.PubKey()

	claimerPriv, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("Failed to generate claimer key: %v", err)
	}
	claimer := claimerPriv.PubKey()

	// Test 1: Claimable Balance Creation
	t.Logf("✅ Testing Claimable Balance Creation...")

	claimableState := shellState.GetClaimableState()
	amount := uint64(500000) // 500k satoshis

	// Create claimants with different predicates
	claimants := []claimable.Claimant{
		{
			Destination: claimer,
			Predicate:   claimable.UnconditionalPredicate(),
		},
		{
			Destination: claimer,
			Predicate:   claimable.AfterTimePredicate(1000000), // After block 1M
		},
	}

	// Create mock funding outpoint
	fundingOutpoint := wire.OutPoint{
		Hash:  chainhash.Hash{},
		Index: 1,
	}

	balance, err := claimableState.CreateClaimableBalance(
		creator,
		amount,
		claimants,
		100, // Create height
		fundingOutpoint,
	)
	if err != nil {
		t.Fatalf("Failed to create claimable balance: %v", err)
	}

	if balance.Amount != amount {
		t.Errorf("Expected amount %d, got %d", amount, balance.Amount)
	}

	if len(balance.Claimants) != len(claimants) {
		t.Errorf("Expected %d claimants, got %d", len(claimants), len(balance.Claimants))
	}

	t.Logf("   Claimable ID: %x", balance.ID)
	t.Logf("   Amount: %d satoshis", balance.Amount)
	t.Logf("   Claimants: %d", len(balance.Claimants))

	// Test 2: Claimable Balance Claiming
	t.Logf("✅ Testing Claimable Balance Claiming...")

	// Create proof for unconditional predicate
	proof := claimable.ClaimProof{
		Preimages: make(map[[32]byte][]byte),
		Timestamp: 1000001, // After the time predicate
	}

	claimedBalance, err := claimableState.ClaimBalance(
		balance.ID,
		claimer,
		proof,
		1000002, // Current height
	)
	if err != nil {
		t.Fatalf("Failed to claim balance: %v", err)
	}

	if claimedBalance.ID != balance.ID {
		t.Error("Claimed balance ID mismatch")
	}

	t.Logf("   Successfully claimed balance: %d satoshis", claimedBalance.Amount)

	// Verify balance is removed from state
	_, err = claimableState.GetClaimableBalance(balance.ID)
	if err == nil {
		t.Error("Balance should be removed after claiming")
	}
}

// testShellOpcodeValidation tests Shell-specific opcode validation
func testShellOpcodeValidation(t *testing.T) {
	t.Logf("✅ Testing Shell Opcode Validation...")

	// Test opcode detection
	script := []byte{txscript.OP_CHANNEL_OPEN}
	opcode, found := txscript.DetectShellOpcode(script)
	if !found {
		t.Error("Should detect OP_CHANNEL_OPEN")
	}
	if opcode != txscript.OP_CHANNEL_OPEN {
		t.Errorf("Expected OP_CHANNEL_OPEN (0x%02x), got 0x%02x", txscript.OP_CHANNEL_OPEN, opcode)
	}

	// Test all Shell opcodes
	shellOpcodes := []byte{
		txscript.OP_VAULTTEMPLATEVERIFY,
		txscript.OP_CHANNEL_OPEN,
		txscript.OP_CHANNEL_UPDATE,
		txscript.OP_CHANNEL_CLOSE,
		txscript.OP_CLAIMABLE_CREATE,
		txscript.OP_CLAIMABLE_CLAIM,
	}

	for _, expectedOpcode := range shellOpcodes {
		script := []byte{expectedOpcode}
		opcode, found := txscript.DetectShellOpcode(script)
		if !found {
			t.Errorf("Should detect opcode 0x%02x", expectedOpcode)
		}
		if opcode != expectedOpcode {
			t.Errorf("Expected opcode 0x%02x, got 0x%02x", expectedOpcode, opcode)
		}
	}

	t.Logf("   All 6 Shell opcodes detected correctly")

	// Test non-Shell script
	nonShellScript := []byte{txscript.OP_DUP, txscript.OP_HASH160}
	_, found = txscript.DetectShellOpcode(nonShellScript)
	if found {
		t.Error("Should not detect Shell opcodes in non-Shell script")
	}

	t.Logf("   Non-Shell scripts correctly ignored")
}

// TestSettlementValidation tests channel and claimable validation functions
func TestSettlementValidation(t *testing.T) {
	t.Parallel()

	t.Run("ChannelValidation", func(t *testing.T) {
		state := channels.NewChannelState()

		// Test channel open validation
		alice, _ := btcec.NewPrivateKey()
		bob, _ := btcec.NewPrivateKey()

		params := []interface{}{
			alice.PubKey(),
			bob.PubKey(),
			uint64(1000000),
			uint32(100000),
		}

		err := channels.ValidateChannelOperation(
			channels.ChannelOpOpen,
			channels.ChannelID{},
			state,
			params,
		)
		if err != nil {
			t.Errorf("Valid channel open should not error: %v", err)
		}

		// Test invalid capacity
		invalidParams := []interface{}{
			alice.PubKey(),
			bob.PubKey(),
			uint64(0), // Invalid capacity
			uint32(100000),
		}

		err = channels.ValidateChannelOperation(
			channels.ChannelOpOpen,
			channels.ChannelID{},
			state,
			invalidParams,
		)
		if err == nil {
			t.Error("Invalid channel capacity should error")
		}
	})

	t.Run("ClaimableValidation", func(t *testing.T) {
		state := claimable.NewClaimableState()

		creator, _ := btcec.NewPrivateKey()
		claimer, _ := btcec.NewPrivateKey()

		claimants := []claimable.Claimant{
			{
				Destination: claimer.PubKey(),
				Predicate:   claimable.UnconditionalPredicate(),
			},
		}

		params := []interface{}{
			creator.PubKey(),
			uint64(1000000),
			claimants,
		}

		err := claimable.ValidateClaimableOperation(
			claimable.ClaimableOpCreate,
			state,
			params,
		)
		if err != nil {
			t.Errorf("Valid claimable create should not error: %v", err)
		}

		// Test invalid amount
		invalidParams := []interface{}{
			creator.PubKey(),
			uint64(0), // Invalid amount
			claimants,
		}

		err = claimable.ValidateClaimableOperation(
			claimable.ClaimableOpCreate,
			state,
			invalidParams,
		)
		if err == nil {
			t.Error("Invalid claimable amount should error")
		}
	})
}

func init() {
	// Add any necessary imports that might be missing
	_ = wire.OutPoint{}
	_ = btcutil.Amount(0)
}
