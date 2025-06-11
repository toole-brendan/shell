package test

import (
	"crypto/sha256"
	"encoding/binary"
	"testing"
	"time"

	"github.com/btcsuite/btcd/wire"
	"github.com/toole-brendan/shell/txscript"
)

// TestDocumentHashOpcode tests the OP_DOC_HASH opcode functionality
func TestDocumentHashOpcode(t *testing.T) {
	// Create a mock document and compute its hash
	document := []byte("This is a test trade document for Bills of Lading BL-2026-001")
	docHash := sha256.Sum256(document)

	// Create timestamp (current time)
	timestamp := time.Now().Unix()
	timestampBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(timestampBytes, uint64(timestamp))

	// Create reference string
	reference := "BL-2026-001-TRADE-ABC-TO-XYZ"
	referenceBytes := []byte(reference)

	// Create witness with document hash parameters
	witness := wire.TxWitness{
		docHash[:],     // 32-byte hash
		timestampBytes, // 8-byte timestamp
		referenceBytes, // reference string
	}

	// Test ExtractDocumentHashParams
	params, err := txscript.ExtractDocumentHashParams(nil, witness)
	if err != nil {
		t.Fatalf("Failed to extract document hash params: %v", err)
	}

	// Verify extracted parameters
	if params.DocumentHash != docHash {
		t.Errorf("Hash mismatch: expected %x, got %x", docHash, params.DocumentHash)
	}

	if params.DocumentTimestamp != timestamp {
		t.Errorf("Timestamp mismatch: expected %d, got %d", timestamp, params.DocumentTimestamp)
	}

	if params.DocumentReference != reference {
		t.Errorf("Reference mismatch: expected %s, got %s", reference, params.DocumentReference)
	}
}

// TestDocumentHashDetection tests Shell opcode detection
func TestDocumentHashDetection(t *testing.T) {
	// Create script with OP_DOC_HASH
	script := []byte{txscript.OP_DOC_HASH}

	opcode, found := txscript.DetectShellOpcode(script)
	if !found {
		t.Error("Should detect OP_DOC_HASH in script")
	}

	if opcode != txscript.OP_DOC_HASH {
		t.Errorf("Expected OP_DOC_HASH (0x%02x), got 0x%02x", txscript.OP_DOC_HASH, opcode)
	}
}

// TestDocumentHashValidation tests parameter validation
func TestDocumentHashValidation(t *testing.T) {
	tests := []struct {
		name        string
		witness     wire.TxWitness
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid parameters",
			witness: wire.TxWitness{
				make([]byte, 32),               // valid 32-byte hash
				[]byte{1, 0, 0, 0, 0, 0, 0, 0}, // valid timestamp
				[]byte("VALID-REF"),            // valid reference
			},
			expectError: false,
		},
		{
			name: "Invalid hash length",
			witness: wire.TxWitness{
				make([]byte, 31),               // invalid 31-byte hash
				[]byte{1, 0, 0, 0, 0, 0, 0, 0}, // valid timestamp
				[]byte("REF"),                  // valid reference
			},
			expectError: true,
			errorMsg:    "invalid hash length",
		},
		{
			name: "Invalid timestamp",
			witness: wire.TxWitness{
				make([]byte, 32),               // valid hash
				[]byte{0, 0, 0, 0, 0, 0, 0, 0}, // zero timestamp
				[]byte("REF"),                  // valid reference
			},
			expectError: true,
			errorMsg:    "timestamp must be positive",
		},
		{
			name: "Reference too long",
			witness: wire.TxWitness{
				make([]byte, 32),               // valid hash
				[]byte{1, 0, 0, 0, 0, 0, 0, 0}, // valid timestamp
				make([]byte, 300),              // too long reference (300 > 256)
			},
			expectError: true,
			errorMsg:    "reference too long",
		},
		{
			name: "Insufficient witness items",
			witness: wire.TxWitness{
				make([]byte, 32), // only hash, missing timestamp and reference
			},
			expectError: true,
			errorMsg:    "insufficient witness items",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := txscript.ExtractDocumentHashParams(nil, tt.witness)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorMsg)
				} else if err.Error()[:len(tt.errorMsg)] != tt.errorMsg {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

// TestDocumentHashRealWorldExample tests with realistic trade finance data
func TestDocumentHashRealWorldExample(t *testing.T) {
	// Simulate a real trade finance document
	billOfLading := `
	BILL OF LADING
	Shipper: ABC Trading Corp, New York
	Consignee: XYZ Manufacturing Ltd, Singapore  
	Vessel: MV SHELL RESERVE
	Voyage: SR-2026-001
	Port of Loading: New York, USA
	Port of Discharge: Singapore
	Commodity: Steel Products
	Weight: 50,000 MT
	Date: 2026-01-15
	`

	// Compute hash
	docHash := sha256.Sum256([]byte(billOfLading))

	// Trade timestamp
	tradeTime := time.Date(2026, 1, 15, 14, 30, 0, 0, time.UTC).Unix()
	timestampBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(timestampBytes, uint64(tradeTime))

	// Trade reference
	reference := "BL-ABC-XYZ-SR2026001-STEEL"

	// Create witness
	witness := wire.TxWitness{
		docHash[:],
		timestampBytes,
		[]byte(reference),
	}

	// Extract and verify
	params, err := txscript.ExtractDocumentHashParams(nil, witness)
	if err != nil {
		t.Fatalf("Failed to extract trade document params: %v", err)
	}

	// Verify the commitment enables institutional verification
	if params.DocumentHash != docHash {
		t.Error("Document hash verification failed")
	}

	if params.DocumentReference != reference {
		t.Error("Trade reference verification failed")
	}

	// Verify timestamp is reasonable
	if params.DocumentTimestamp != tradeTime {
		t.Error("Trade timestamp verification failed")
	}

	t.Logf("Successfully committed trade document:")
	t.Logf("  Hash: %x", params.DocumentHash)
	t.Logf("  Reference: %s", params.DocumentReference)
	t.Logf("  Timestamp: %d", params.DocumentTimestamp)
}
