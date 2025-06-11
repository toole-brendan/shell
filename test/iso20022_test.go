package test

import (
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/toole-brendan/shell/settlement/iso20022"
)

// TestISO20022MessageMapping tests basic message mapping functionality
func TestISO20022MessageMapping(t *testing.T) {
	// Create a mock transaction
	tx := createMockTransaction(t)

	// Test basic mapping without metadata
	msg, err := iso20022.MapToISO20022(tx, iso20022.PACS008, nil)
	if err != nil {
		t.Fatalf("Failed to map transaction to ISO 20022: %v", err)
	}

	// Verify basic fields
	if msg.Type != iso20022.PACS008 {
		t.Errorf("Expected message type %s, got %s", iso20022.PACS008, msg.Type)
	}

	if msg.Currency != "XSL" {
		t.Errorf("Expected currency XSL, got %s", msg.Currency)
	}

	if msg.TransactionID != tx.TxHash().String() {
		t.Errorf("Transaction ID mismatch")
	}

	if msg.MessageID == "" {
		t.Error("Message ID should not be empty")
	}

	if msg.EndToEndID == "" {
		t.Error("End-to-end ID should not be empty")
	}

	t.Logf("✅ Basic ISO 20022 mapping successful")
	t.Logf("   Message Type: %s", msg.Type)
	t.Logf("   Message ID: %s", msg.MessageID)
	t.Logf("   Transaction ID: %s", msg.TransactionID)
	t.Logf("   End-to-End ID: %s", msg.EndToEndID)
}

// TestISO20022MessageMappingWithMetadata tests mapping with full metadata
func TestISO20022MessageMappingWithMetadata(t *testing.T) {
	tx := createMockTransaction(t)

	// Create metadata for institutional transfer
	metadata := &iso20022.TransactionMetadata{
		SenderBIC:   "CHASUS33XXX", // Chase Bank US
		ReceiverBIC: "DEUTDEFFXXX", // Deutsche Bank Germany
		Reference:   "TRADE-REF-001",
		Amount:      1000000, // 1M satoshis
		ValueDate:   time.Date(2026, 1, 15, 14, 30, 0, 0, time.UTC),
	}

	msg, err := iso20022.MapToISO20022(tx, iso20022.PACS008, metadata)
	if err != nil {
		t.Fatalf("Failed to map transaction with metadata: %v", err)
	}

	// Verify metadata fields
	if msg.SenderBIC != metadata.SenderBIC {
		t.Errorf("Expected sender BIC %s, got %s", metadata.SenderBIC, msg.SenderBIC)
	}

	if msg.ReceiverBIC != metadata.ReceiverBIC {
		t.Errorf("Expected receiver BIC %s, got %s", metadata.ReceiverBIC, msg.ReceiverBIC)
	}

	if msg.Reference != metadata.Reference {
		t.Errorf("Expected reference %s, got %s", metadata.Reference, msg.Reference)
	}

	if msg.Amount != metadata.Amount {
		t.Errorf("Expected amount %d, got %d", metadata.Amount, msg.Amount)
	}

	if !msg.ValueDate.Equal(metadata.ValueDate) {
		t.Errorf("Value date mismatch")
	}

	t.Logf("✅ ISO 20022 mapping with metadata successful")
	t.Logf("   Sender BIC: %s", msg.SenderBIC)
	t.Logf("   Receiver BIC: %s", msg.ReceiverBIC)
	t.Logf("   Reference: %s", msg.Reference)
	t.Logf("   Amount: %d satoshis", msg.Amount)
}

// TestPACS008MessageCreation tests credit transfer message creation
func TestPACS008MessageCreation(t *testing.T) {
	tx := createMockTransaction(t)

	sender := iso20022.BankIdentifier{
		BIC:     "CHASUS33XXX",
		Name:    "JPMorgan Chase Bank",
		Account: "001234567890",
	}

	receiver := iso20022.BankIdentifier{
		BIC:     "DEUTDEFFXXX",
		Name:    "Deutsche Bank AG",
		Account: "987654321000",
	}

	amount := uint64(5000000) // 5M satoshis
	reference := "TRADE-SETTLEMENT-XYZ-001"

	msg, err := iso20022.CreatePACS008Message(tx, sender, receiver, amount, reference)
	if err != nil {
		t.Fatalf("Failed to create PACS.008 message: %v", err)
	}

	if msg.Type != iso20022.PACS008 {
		t.Errorf("Expected PACS.008 message type, got %s", msg.Type)
	}

	if msg.SenderBIC != sender.BIC {
		t.Errorf("Sender BIC mismatch")
	}

	if msg.ReceiverBIC != receiver.BIC {
		t.Errorf("Receiver BIC mismatch")
	}

	if msg.Amount != amount {
		t.Errorf("Amount mismatch")
	}

	if msg.Reference != reference {
		t.Errorf("Reference mismatch")
	}

	t.Logf("✅ PACS.008 message creation successful")
	t.Logf("   From: %s (%s)", sender.Name, sender.BIC)
	t.Logf("   To: %s (%s)", receiver.Name, receiver.BIC)
	t.Logf("   Amount: %d XSL", amount)
	t.Logf("   Reference: %s", reference)
}

// TestPACS009MessageCreation tests FI transfer message creation
func TestPACS009MessageCreation(t *testing.T) {
	tx := createMockTransaction(t)

	sender := iso20022.BankIdentifier{
		BIC:     "SWIFT000XXX",
		Name:    "Central Bank A",
		Account: "CB-RESERVE-001",
	}

	receiver := iso20022.BankIdentifier{
		BIC:     "SWIFT001XXX",
		Name:    "Central Bank B",
		Account: "CB-RESERVE-002",
	}

	amount := uint64(50000000) // 50M satoshis (institutional transfer)

	msg, err := iso20022.CreatePACS009Message(tx, sender, receiver, amount)
	if err != nil {
		t.Fatalf("Failed to create PACS.009 message: %v", err)
	}

	if msg.Type != iso20022.PACS009 {
		t.Errorf("Expected PACS.009 message type, got %s", msg.Type)
	}

	if msg.SenderBIC != sender.BIC {
		t.Errorf("Sender BIC mismatch")
	}

	if msg.ReceiverBIC != receiver.BIC {
		t.Errorf("Receiver BIC mismatch")
	}

	if msg.Amount != amount {
		t.Errorf("Amount mismatch")
	}

	t.Logf("✅ PACS.009 message creation successful")
	t.Logf("   From: %s (%s)", sender.Name, sender.BIC)
	t.Logf("   To: %s (%s)", receiver.Name, receiver.BIC)
	t.Logf("   Amount: %d XSL", amount)
}

// TestSettlementProofGeneration tests settlement finality proof generation
func TestSettlementProofGeneration(t *testing.T) {
	tx := createMockTransaction(t)
	blockHash := chainhash.Hash([32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32})
	blockHeight := int32(100000)
	confirmations := int32(6)

	proof := iso20022.GenerateSettlementProof(tx, blockHash, blockHeight, confirmations)
	if proof == nil {
		t.Fatal("Settlement proof should not be nil")
	}

	// Verify proof fields
	if proof.TransactionHash != tx.TxHash() {
		t.Error("Transaction hash mismatch in proof")
	}

	if proof.BlockHash != blockHash {
		t.Error("Block hash mismatch in proof")
	}

	if proof.BlockHeight != blockHeight {
		t.Error("Block height mismatch in proof")
	}

	if proof.Confirmations != confirmations {
		t.Error("Confirmations mismatch in proof")
	}

	if !proof.IsIrrevocable {
		t.Error("Settlement should be irrevocable with 6+ confirmations")
	}

	if proof.ISOReference == "" {
		t.Error("ISO reference should not be empty")
	}

	t.Logf("✅ Settlement proof generation successful")
	t.Logf("   Transaction Hash: %s", proof.TransactionHash)
	t.Logf("   Block Height: %d", proof.BlockHeight)
	t.Logf("   Confirmations: %d", proof.Confirmations)
	t.Logf("   Irrevocable: %t", proof.IsIrrevocable)
	t.Logf("   ISO Reference: %s", proof.ISOReference)
}

// TestSettlementProofValidation tests settlement proof validation
func TestSettlementProofValidation(t *testing.T) {
	tx := createMockTransaction(t)
	blockHash := chainhash.Hash([32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32})
	blockHeight := int32(100000)
	confirmations := int32(6)

	// Generate valid proof
	proof := iso20022.GenerateSettlementProof(tx, blockHash, blockHeight, confirmations)

	// Validate the proof
	err := iso20022.ValidateSettlementProof(proof)
	if err != nil {
		t.Errorf("Valid proof should not error: %v", err)
	}

	// Test invalid proof (tampered)
	tamperedProof := *proof
	tamperedProof.ProofHash[0] = ^tamperedProof.ProofHash[0] // Flip first bit

	err = iso20022.ValidateSettlementProof(&tamperedProof)
	if err == nil {
		t.Error("Tampered proof should fail validation")
	}

	// Test invalid confirmations
	invalidProof := *proof
	invalidProof.Confirmations = 3
	invalidProof.IsIrrevocable = true // This should fail validation

	err = iso20022.ValidateSettlementProof(&invalidProof)
	if err == nil {
		t.Error("Proof with insufficient confirmations should fail validation")
	}

	t.Logf("✅ Settlement proof validation working correctly")
}

// TestSWIFTReferenceGeneration tests SWIFT reference generation
func TestSWIFTReferenceGeneration(t *testing.T) {
	tx := createMockTransaction(t)

	ref := iso20022.GenerateSWIFTReference(tx)

	// Verify format: XSL + 12 digits timestamp + 6 hex chars
	if len(ref) != 21 { // XSL(3) + timestamp(12) + hash(6)
		t.Errorf("Expected reference length 21, got %d: %s", len(ref), ref)
	}

	if ref[:3] != "XSL" {
		t.Errorf("Reference should start with XSL, got: %s", ref[:3])
	}

	// Test uniqueness - two calls should generate different references
	time.Sleep(time.Second) // Ensure different timestamp
	ref2 := iso20022.GenerateSWIFTReference(tx)
	if ref == ref2 {
		t.Error("Two SWIFT references should be different (timestamp based)")
	}

	t.Logf("✅ SWIFT reference generation successful")
	t.Logf("   Reference 1: %s", ref)
	t.Logf("   Reference 2: %s", ref2)
}

// TestSupportedMessageTypes tests message type support
func TestSupportedMessageTypes(t *testing.T) {
	supportedTypes := iso20022.GetSupportedMessageTypes()

	expectedTypes := []iso20022.MessageType{
		iso20022.PACS008,
		iso20022.PACS009,
		iso20022.CAMT056,
		iso20022.PAIN001,
	}

	if len(supportedTypes) != len(expectedTypes) {
		t.Errorf("Expected %d supported types, got %d", len(expectedTypes), len(supportedTypes))
	}

	// Verify each expected type is supported
	for _, expected := range expectedTypes {
		if !iso20022.IsSupported(expected) {
			t.Errorf("Message type %s should be supported", expected)
		}
	}

	// Test unsupported type
	unsupported := iso20022.MessageType("unsupported.message.type")
	if iso20022.IsSupported(unsupported) {
		t.Error("Unsupported message type should return false")
	}

	t.Logf("✅ Message type support verification successful")
	t.Logf("   Supported types: %v", supportedTypes)
}

// TestRealWorldScenario tests a complete institutional transfer scenario
func TestRealWorldScenario(t *testing.T) {
	t.Log("🏦 Testing Real-World Central Bank Settlement Scenario")

	// Create transaction representing $50M XSL transfer between central banks
	tx := createMockTransaction(t)

	// Central Bank A (sender)
	centralBankA := iso20022.BankIdentifier{
		BIC:     "RBOZAU2SXXX", // Reserve Bank of Australia (example)
		Name:    "Reserve Bank of Australia",
		Account: "RBA-SHELL-RESERVE-001",
	}

	// Central Bank B (receiver)
	centralBankB := iso20022.BankIdentifier{
		BIC:     "BANKSGSGXXX", // Bank of Singapore (example)
		Name:    "Monetary Authority of Singapore",
		Account: "MAS-SHELL-RESERVE-001",
	}

	amount := uint64(5000000000) // 50M satoshis (50 XSL)
	reference := "BILATERAL-SETTLEMENT-Q1-2026-001"

	// Step 1: Create PACS.008 message for the transfer
	msg, err := iso20022.CreatePACS008Message(tx, centralBankA, centralBankB, amount, reference)
	if err != nil {
		t.Fatalf("Failed to create institutional transfer message: %v", err)
	}

	// Step 2: Generate settlement proof (assuming transaction is confirmed)
	blockHash := chainhash.Hash([32]byte{})
	blockHeight := int32(262800) // One halving period
	confirmations := int32(12)   // High security for central banks

	proof := iso20022.GenerateSettlementProof(tx, blockHash, blockHeight, confirmations)
	msg.SettlementProof = proof
	msg.Confirmations = confirmations

	// Step 3: Validate the complete message
	if msg.Type != iso20022.PACS008 {
		t.Error("Should be PACS.008 credit transfer")
	}

	if msg.Amount != amount {
		t.Error("Amount mismatch")
	}

	if !msg.SettlementProof.IsIrrevocable {
		t.Error("Settlement should be irrevocable")
	}

	// Step 4: Validate settlement proof
	err = iso20022.ValidateSettlementProof(msg.SettlementProof)
	if err != nil {
		t.Errorf("Settlement proof validation failed: %v", err)
	}

	t.Logf("✅ Real-world scenario completed successfully")
	t.Logf("   📊 Transfer Details:")
	t.Logf("      From: %s (%s)", centralBankA.Name, centralBankA.BIC)
	t.Logf("      To: %s (%s)", centralBankB.Name, centralBankB.BIC)
	t.Logf("      Amount: 50 XSL (%d satoshis)", amount)
	t.Logf("      Reference: %s", reference)
	t.Logf("   🔒 Settlement Proof:")
	t.Logf("      Block Height: %d", proof.BlockHeight)
	t.Logf("      Confirmations: %d", proof.Confirmations)
	t.Logf("      Irrevocable: %t", proof.IsIrrevocable)
	t.Logf("      ISO Reference: %s", proof.ISOReference)
}

// Helper function to create a mock transaction for testing
func createMockTransaction(t *testing.T) *wire.MsgTx {
	tx := wire.NewMsgTx(wire.TxVersion)

	// Add a mock input
	prevOut := wire.NewOutPoint(&chainhash.Hash{}, 0)
	txIn := wire.NewTxIn(prevOut, []byte{}, nil)
	tx.AddTxIn(txIn)

	// Add a mock output
	txOut := wire.NewTxOut(1000000, []byte{}) // 1M satoshis
	tx.AddTxOut(txOut)

	return tx
}
