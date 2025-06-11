package test

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/toole-brendan/shell/settlement/swaps"
)

// TestAtomicSwapCreation tests the creation of atomic swaps
func TestAtomicSwapCreation(t *testing.T) {
	// Generate test keys
	initiatorPriv, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("Failed to generate initiator key: %v", err)
	}
	initiator := initiatorPriv.PubKey()

	participantPriv, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("Failed to generate participant key: %v", err)
	}
	participant := participantPriv.PubKey()

	// Create swap parameters
	secret := []byte("this-is-a-test-secret-for-atomic-swap")
	params := &swaps.AtomicSwapParams{
		Initiator:   initiator,
		Participant: participant,
		Amount:      1000000, // 1M satoshis
		Timeout:     3600,    // 1 hour
		Chain:       swaps.ChainShell,
		Secret:      secret,
	}

	// Create atomic swap
	swap, err := swaps.NewAtomicSwap(params)
	if err != nil {
		t.Fatalf("Failed to create atomic swap: %v", err)
	}

	// Verify swap properties
	if swap.Initiator != initiator {
		t.Error("Initiator mismatch")
	}

	if swap.Participant != participant {
		t.Error("Participant mismatch")
	}

	if swap.Amount != params.Amount {
		t.Error("Amount mismatch")
	}

	if swap.Timeout != params.Timeout {
		t.Error("Timeout mismatch")
	}

	if swap.Chain != params.Chain {
		t.Error("Chain mismatch")
	}

	if swap.Status != swaps.SwapStatusPending {
		t.Error("Initial status should be pending")
	}

	// Verify secret hash
	expectedHash := sha256.Sum256(secret)
	if swap.SecretHash != expectedHash {
		t.Error("Secret hash mismatch")
	}

	t.Logf("✅ Atomic swap creation successful")
	t.Logf("   Swap ID: %x", swap.SwapID)
	t.Logf("   Amount: %d satoshis", swap.Amount)
	t.Logf("   Timeout: %d seconds", swap.Timeout)
	t.Logf("   Status: %s", swap.Status)
}

// TestHTLCScriptCreation tests HTLC script generation
func TestHTLCScriptCreation(t *testing.T) {
	swap := createTestSwap(t)

	script, err := swap.CreateHTLCScript()
	if err != nil {
		t.Fatalf("Failed to create HTLC script: %v", err)
	}

	if len(script) == 0 {
		t.Error("HTLC script should not be empty")
	}

	// Verify script contains expected opcodes
	scriptHex := fmt.Sprintf("%x", script)

	// Should contain IF/ELSE/ENDIF structure
	if !strings.Contains(scriptHex, "63") { // OP_IF
		t.Error("Script should contain OP_IF")
	}

	if !strings.Contains(scriptHex, "67") { // OP_ELSE
		t.Error("Script should contain OP_ELSE")
	}

	if !strings.Contains(scriptHex, "68") { // OP_ENDIF
		t.Error("Script should contain OP_ENDIF")
	}

	t.Logf("✅ HTLC script creation successful")
	t.Logf("   Script length: %d bytes", len(script))
	t.Logf("   Script hex: %x", script)
}

// TestContractTransactionCreation tests contract transaction creation
func TestContractTransactionCreation(t *testing.T) {
	swap := createTestSwap(t)

	// Create mock funding transaction
	fundingTx := createMockFundingTransaction(t, swap.Amount)

	// Create contract transaction
	contractTx, err := swap.CreateContractTransaction(fundingTx, 0)
	if err != nil {
		t.Fatalf("Failed to create contract transaction: %v", err)
	}

	// Verify contract transaction
	if len(contractTx.TxIn) != 1 {
		t.Error("Contract transaction should have exactly one input")
	}

	if len(contractTx.TxOut) != 1 {
		t.Error("Contract transaction should have exactly one output")
	}

	if contractTx.TxOut[0].Value != int64(swap.Amount) {
		t.Error("Contract output value mismatch")
	}

	if swap.Status != swaps.SwapStatusActive {
		t.Error("Swap status should be active after contract creation")
	}

	t.Logf("✅ Contract transaction creation successful")
	t.Logf("   Contract TX Hash: %s", contractTx.TxHash())
	t.Logf("   Amount: %d satoshis", contractTx.TxOut[0].Value)
	t.Logf("   Status: %s", swap.Status)
}

// TestRedeemTransaction tests redeem transaction creation and secret extraction
func TestRedeemTransaction(t *testing.T) {
	swap := createTestSwap(t)
	secret := []byte("this-is-a-test-secret-for-atomic-swap")

	// Create funding and contract transactions
	fundingTx := createMockFundingTransaction(t, swap.Amount)
	_, err := swap.CreateContractTransaction(fundingTx, 0)
	if err != nil {
		t.Fatalf("Failed to create contract transaction: %v", err)
	}

	// Create participant address (mock)
	participantAddr := []byte{0x76, 0xa9, 0x14} // P2PKH prefix + 20 bytes + suffix

	// Create redeem transaction
	redeemTx, err := swap.CreateRedeemTransaction(secret, participantAddr)
	if err != nil {
		t.Fatalf("Failed to create redeem transaction: %v", err)
	}

	// Verify redeem transaction
	if len(redeemTx.TxIn) != 1 {
		t.Error("Redeem transaction should have exactly one input")
	}

	if len(redeemTx.TxOut) != 1 {
		t.Error("Redeem transaction should have exactly one output")
	}

	if redeemTx.TxOut[0].Value != int64(swap.Amount) {
		t.Error("Redeem output value mismatch")
	}

	if swap.Status != swaps.SwapStatusRedeemed {
		t.Error("Swap status should be redeemed")
	}

	// Test secret extraction
	extractedSecret, err := swaps.ExtractSecretFromRedeemTx(redeemTx)
	if err != nil {
		t.Fatalf("Failed to extract secret from redeem transaction: %v", err)
	}

	if string(extractedSecret) != string(secret) {
		t.Error("Extracted secret does not match original")
	}

	t.Logf("✅ Redeem transaction successful")
	t.Logf("   Redeem TX Hash: %s", redeemTx.TxHash())
	t.Logf("   Secret extracted: %s", string(extractedSecret))
	t.Logf("   Status: %s", swap.Status)
}

// TestRefundTransaction tests refund transaction creation after timeout
func TestRefundTransaction(t *testing.T) {
	// Create swap with short timeout for testing
	initiatorPriv, _ := btcec.NewPrivateKey()
	participantPriv, _ := btcec.NewPrivateKey()

	params := &swaps.AtomicSwapParams{
		Initiator:   initiatorPriv.PubKey(),
		Participant: participantPriv.PubKey(),
		Amount:      1000000,
		Timeout:     1, // 1 second timeout
		Chain:       swaps.ChainShell,
		Secret:      []byte("test-secret"),
	}

	swap, err := swaps.NewAtomicSwap(params)
	if err != nil {
		t.Fatalf("Failed to create swap: %v", err)
	}

	// Create funding and contract transactions
	fundingTx := createMockFundingTransaction(t, swap.Amount)
	_, err = swap.CreateContractTransaction(fundingTx, 0)
	if err != nil {
		t.Fatalf("Failed to create contract transaction: %v", err)
	}

	// Wait for timeout
	time.Sleep(2 * time.Second)

	// Create initiator address (mock)
	initiatorAddr := []byte{0x76, 0xa9, 0x14} // P2PKH prefix + 20 bytes + suffix

	// Create refund transaction
	refundTx, err := swap.CreateRefundTransaction(initiatorAddr)
	if err != nil {
		t.Fatalf("Failed to create refund transaction: %v", err)
	}

	// Verify refund transaction
	if len(refundTx.TxIn) != 1 {
		t.Error("Refund transaction should have exactly one input")
	}

	if len(refundTx.TxOut) != 1 {
		t.Error("Refund transaction should have exactly one output")
	}

	if refundTx.TxOut[0].Value != int64(swap.Amount) {
		t.Error("Refund output value mismatch")
	}

	if swap.Status != swaps.SwapStatusRefunded {
		t.Error("Swap status should be refunded")
	}

	t.Logf("✅ Refund transaction successful")
	t.Logf("   Refund TX Hash: %s", refundTx.TxHash())
	t.Logf("   Status: %s", swap.Status)
}

// TestSwapManager tests the swap management functionality
func TestSwapManager(t *testing.T) {
	manager := swaps.NewSwapManager()

	// Create test swaps
	swap1 := createTestSwap(t)
	swap2 := createTestSwap(t)

	// Add swaps to manager
	err := manager.AddSwap(swap1)
	if err != nil {
		t.Fatalf("Failed to add swap1: %v", err)
	}

	err = manager.AddSwap(swap2)
	if err != nil {
		t.Fatalf("Failed to add swap2: %v", err)
	}

	// Retrieve swap
	retrievedSwap, err := manager.GetSwap(swap1.SwapID)
	if err != nil {
		t.Fatalf("Failed to retrieve swap: %v", err)
	}

	if retrievedSwap.SwapID != swap1.SwapID {
		t.Error("Retrieved swap ID mismatch")
	}

	// List active swaps (both should be pending, not active yet)
	activeSwaps := manager.ListActiveSwaps()
	if len(activeSwaps) != 0 {
		t.Errorf("Expected 0 active swaps, got %d", len(activeSwaps))
	}

	// Test cleanup (no expired swaps yet)
	manager.CleanupExpiredSwaps()

	t.Logf("✅ Swap manager tests successful")
	t.Logf("   Swaps managed: 2")
	t.Logf("   Active swaps: %d", len(activeSwaps))
}

// TestCrossChainSwapCreation tests cross-chain swap creation
func TestCrossChainSwapCreation(t *testing.T) {
	initiatorPriv, _ := btcec.NewPrivateKey()
	participantPriv, _ := btcec.NewPrivateKey()

	params := &swaps.AtomicSwapParams{
		Initiator:   initiatorPriv.PubKey(),
		Participant: participantPriv.PubKey(),
		Amount:      5000000, // 5M satoshis for cross-chain
		Timeout:     7200,    // 2 hours
		Chain:       swaps.ChainShell,
		Secret:      []byte("cross-chain-secret"),
	}

	// Test Shell <-> Bitcoin swap
	btcSwap, err := swaps.CreateCrossChainSwap(params, swaps.ChainBitcoin)
	if err != nil {
		t.Fatalf("Failed to create BTC cross-chain swap: %v", err)
	}

	if btcSwap.Chain != swaps.ChainBitcoin {
		t.Error("Cross-chain type mismatch for BTC")
	}

	if btcSwap.ShellSwap.Amount != params.Amount {
		t.Error("Amount mismatch in cross-chain swap")
	}

	// Test Shell <-> Ethereum swap
	ethSwap, err := swaps.CreateCrossChainSwap(params, swaps.ChainEthereum)
	if err != nil {
		t.Fatalf("Failed to create ETH cross-chain swap: %v", err)
	}

	if ethSwap.Chain != swaps.ChainEthereum {
		t.Error("Cross-chain type mismatch for ETH")
	}

	t.Logf("✅ Cross-chain swap creation successful")
	t.Logf("   BTC Swap ID: %x", btcSwap.ShellSwap.SwapID)
	t.Logf("   ETH Swap ID: %x", ethSwap.ShellSwap.SwapID)
	t.Logf("   Amount: %d satoshis", params.Amount)
}

// TestSwapValidation tests swap validation logic
func TestSwapValidation(t *testing.T) {
	// Test valid swap
	validSwap := createTestSwap(t)
	err := swaps.ValidateSwap(validSwap)
	if err != nil {
		t.Errorf("Valid swap should not error: %v", err)
	}

	// Test nil swap
	err = swaps.ValidateSwap(nil)
	if err == nil {
		t.Error("Nil swap should error")
	}

	// Test swap with missing initiator
	invalidSwap := createTestSwap(t)
	invalidSwap.Initiator = nil
	err = swaps.ValidateSwap(invalidSwap)
	if err == nil {
		t.Error("Swap with nil initiator should error")
	}

	// Test swap with zero amount
	invalidSwap2 := createTestSwap(t)
	invalidSwap2.Amount = 0
	err = swaps.ValidateSwap(invalidSwap2)
	if err == nil {
		t.Error("Swap with zero amount should error")
	}

	t.Logf("✅ Swap validation tests successful")
}

// TestAtomicSwapRealWorldScenario tests a complete atomic swap scenario
func TestAtomicSwapRealWorldScenario(t *testing.T) {
	t.Log("💱 Testing Real-World Cross-Chain Atomic Swap Scenario")

	// Scenario: Central Bank wants to swap 10 XSL for 0.3 BTC
	initiatorPriv, _ := btcec.NewPrivateKey()
	participantPriv, _ := btcec.NewPrivateKey()

	secret := []byte("central-bank-swap-secret-2026-q1")
	params := &swaps.AtomicSwapParams{
		Initiator:   initiatorPriv.PubKey(),
		Participant: participantPriv.PubKey(),
		Amount:      1000000000, // 10 XSL (1B satoshis)
		Timeout:     86400,      // 24 hours
		Chain:       swaps.ChainShell,
		Secret:      secret,
	}

	// Step 1: Create cross-chain swap
	crossSwap, err := swaps.CreateCrossChainSwap(params, swaps.ChainBitcoin)
	if err != nil {
		t.Fatalf("Failed to create cross-chain swap: %v", err)
	}

	shellSwap := crossSwap.ShellSwap

	// Step 2: Create funding transaction
	fundingTx := createMockFundingTransaction(t, shellSwap.Amount)

	// Step 3: Create contract transaction (initiator locks XSL)
	contractTx, err := shellSwap.CreateContractTransaction(fundingTx, 0)
	if err != nil {
		t.Fatalf("Failed to create contract transaction: %v", err)
	}

	// Step 4: Participant redeems with secret
	participantAddr := []byte{0x76, 0xa9, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20, 0x21, 0x22, 0x23, 0x24, 0x88, 0xac}
	redeemTx, err := shellSwap.CreateRedeemTransaction(secret, participantAddr)
	if err != nil {
		t.Fatalf("Failed to create redeem transaction: %v", err)
	}

	// Step 5: Verify secret can be extracted (for Bitcoin side redemption)
	extractedSecret, err := swaps.ExtractSecretFromRedeemTx(redeemTx)
	if err != nil {
		t.Fatalf("Failed to extract secret: %v", err)
	}

	// Verify the complete swap
	if shellSwap.Status != swaps.SwapStatusRedeemed {
		t.Error("Shell swap should be redeemed")
	}

	if string(extractedSecret) != string(secret) {
		t.Error("Extracted secret mismatch")
	}

	t.Logf("✅ Real-world scenario completed successfully")
	t.Logf("   💰 Swap Details:")
	t.Logf("      Amount: 10 XSL (%d satoshis)", shellSwap.Amount)
	t.Logf("      Chains: Shell ↔ Bitcoin")
	t.Logf("      Timeout: 24 hours")
	t.Logf("   🔐 Security:")
	t.Logf("      Secret: %s", string(secret))
	t.Logf("      Secret Hash: %x", shellSwap.SecretHash)
	t.Logf("   📋 Transactions:")
	t.Logf("      Contract TX: %s", contractTx.TxHash())
	t.Logf("      Redeem TX: %s", redeemTx.TxHash())
	t.Logf("      Status: %s", shellSwap.Status)
}

// Helper functions

func createTestSwap(t *testing.T) *swaps.AtomicSwap {
	initiatorPriv, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("Failed to generate initiator key: %v", err)
	}

	participantPriv, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("Failed to generate participant key: %v", err)
	}

	params := &swaps.AtomicSwapParams{
		Initiator:   initiatorPriv.PubKey(),
		Participant: participantPriv.PubKey(),
		Amount:      1000000,
		Timeout:     3600,
		Chain:       swaps.ChainShell,
		Secret:      []byte("this-is-a-test-secret-for-atomic-swap"),
	}

	swap, err := swaps.NewAtomicSwap(params)
	if err != nil {
		t.Fatalf("Failed to create test swap: %v", err)
	}

	return swap
}

func createMockFundingTransaction(t *testing.T, amount uint64) *wire.MsgTx {
	tx := wire.NewMsgTx(wire.TxVersion)

	// Add a mock input
	prevOut := wire.NewOutPoint(&chainhash.Hash{}, 0)
	txIn := wire.NewTxIn(prevOut, []byte{}, nil)
	tx.AddTxIn(txIn)

	// Add output with the specified amount
	txOut := wire.NewTxOut(int64(amount), []byte{})
	tx.AddTxOut(txOut)

	return tx
}
