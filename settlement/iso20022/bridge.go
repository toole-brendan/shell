// Package iso20022 provides ISO 20022 integration for Shell Reserve
// enabling compatibility with SWIFT messaging standards for institutional use.
package iso20022

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

// MessageType represents ISO 20022 message types supported by Shell
type MessageType string

const (
	// Core payment message types
	PACS008 MessageType = "pacs.008.001.08" // FIToFICstmrCdtTrf - Credit Transfer
	PACS009 MessageType = "pacs.009.001.08" // FIToFICtmrMsgMkrGrpRpt - FI Transfer
	CAMT056 MessageType = "camt.056.001.08" // FIToFIPmtCxlReq - Payment Cancellation
	PAIN001 MessageType = "pain.001.001.09" // CstmrCdtTrfInitn - Payment Initiation
)

// ISO20022Message represents a mapped Shell transaction in ISO 20022 format
type ISO20022Message struct {
	Type            MessageType      `json:"msgType"`
	MessageID       string           `json:"msgId"`
	CreationDate    time.Time        `json:"creDtTm"`
	SenderBIC       string           `json:"instgAgt,omitempty"`
	ReceiverBIC     string           `json:"instdAgt,omitempty"`
	EndToEndID      string           `json:"endToEndId"`
	TransactionID   string           `json:"txId"`
	Amount          uint64           `json:"instrAmt"`
	Currency        string           `json:"ccy"`
	ValueDate       time.Time        `json:"reqdExctnDt"`
	Reference       string           `json:"rmtInf,omitempty"`
	ShellTxHash     chainhash.Hash   `json:"shellTxHash"`
	ShellBlockHash  chainhash.Hash   `json:"shellBlkHash,omitempty"`
	Confirmations   int32            `json:"confirmations"`
	SettlementProof *SettlementProof `json:"settlementProof,omitempty"`
}

// BankIdentifier represents bank identification in SWIFT format
type BankIdentifier struct {
	BIC     string `json:"bic"`
	Name    string `json:"name"`
	Account string `json:"account"`
}

// SettlementProof provides cryptographic proof of settlement finality
type SettlementProof struct {
	TransactionHash  chainhash.Hash `json:"txHash"`
	BlockHash        chainhash.Hash `json:"blockHash"`
	BlockHeight      int32          `json:"blockHeight"`
	Confirmations    int32          `json:"confirmations"`
	Timestamp        time.Time      `json:"timestamp"`
	ISOReference     string         `json:"isoReference"`
	ProofHash        [32]byte       `json:"proofHash"`
	IsIrrevocable    bool           `json:"irrevocable"`
	FinalizationTime time.Time      `json:"finalizationTime"`
}

// MapToISO20022 converts a Shell transaction to ISO 20022 message format
func MapToISO20022(tx *wire.MsgTx, msgType MessageType, metadata *TransactionMetadata) (*ISO20022Message, error) {
	if tx == nil {
		return nil, fmt.Errorf("transaction cannot be nil")
	}

	txHash := tx.TxHash()
	now := time.Now()

	msg := &ISO20022Message{
		Type:          msgType,
		MessageID:     generateMessageID(txHash),
		CreationDate:  now,
		EndToEndID:    generateEndToEndID(txHash),
		TransactionID: txHash.String(),
		Currency:      "XSL",
		ValueDate:     now,
		ShellTxHash:   txHash,
	}

	// Add metadata if provided
	if metadata != nil {
		msg.SenderBIC = metadata.SenderBIC
		msg.ReceiverBIC = metadata.ReceiverBIC
		msg.Reference = metadata.Reference
		msg.Amount = metadata.Amount

		if !metadata.ValueDate.IsZero() {
			msg.ValueDate = metadata.ValueDate
		}
	}

	return msg, nil
}

// TransactionMetadata contains additional information for ISO 20022 mapping
type TransactionMetadata struct {
	SenderBIC   string    `json:"senderBic"`
	ReceiverBIC string    `json:"receiverBic"`
	Reference   string    `json:"reference"`
	Amount      uint64    `json:"amount"`
	ValueDate   time.Time `json:"valueDate"`
}

// GenerateSWIFTReference creates a SWIFT-compatible reference for Shell transactions
func GenerateSWIFTReference(tx *wire.MsgTx) string {
	hash := tx.TxHash()
	timestamp := time.Now().Format("060102150405") // YYMMDDHHMMSS

	// Format: XSL + timestamp + first 6 chars of hash
	return fmt.Sprintf("XSL%s%s", timestamp, hex.EncodeToString(hash[:3]))
}

// generateMessageID creates ISO 20022 compliant message ID
func generateMessageID(txHash chainhash.Hash) string {
	timestamp := time.Now().Format("20060102150405")
	return fmt.Sprintf("SHELL%s%s", timestamp, hex.EncodeToString(txHash[:4]))
}

// generateEndToEndID creates end-to-end identification
func generateEndToEndID(txHash chainhash.Hash) string {
	return fmt.Sprintf("E2E%s", hex.EncodeToString(txHash[:8]))
}

// GenerateSettlementProof creates cryptographic proof of settlement finality
func GenerateSettlementProof(tx *wire.MsgTx, blockHash chainhash.Hash, blockHeight int32, confirmations int32) *SettlementProof {
	txHash := tx.TxHash()
	now := time.Now()

	// Create proof hash combining transaction and block information
	proofData := fmt.Sprintf("%s:%s:%d:%d", txHash.String(), blockHash.String(), blockHeight, confirmations)
	proofHash := sha256.Sum256([]byte(proofData))

	proof := &SettlementProof{
		TransactionHash:  txHash,
		BlockHash:        blockHash,
		BlockHeight:      blockHeight,
		Confirmations:    confirmations,
		Timestamp:        now,
		ISOReference:     GenerateSWIFTReference(tx),
		ProofHash:        proofHash,
		IsIrrevocable:    confirmations >= 6, // 6 confirmations = final
		FinalizationTime: now,
	}

	// If settlement is final, set finalization time
	if proof.IsIrrevocable {
		proof.FinalizationTime = now
	}

	return proof
}

// ValidateSettlementProof verifies the settlement proof integrity
func ValidateSettlementProof(proof *SettlementProof) error {
	if proof == nil {
		return fmt.Errorf("settlement proof cannot be nil")
	}

	// Recreate proof hash and verify
	proofData := fmt.Sprintf("%s:%s:%d:%d",
		proof.TransactionHash.String(),
		proof.BlockHash.String(),
		proof.BlockHeight,
		proof.Confirmations)
	expectedHash := sha256.Sum256([]byte(proofData))

	if proof.ProofHash != expectedHash {
		return fmt.Errorf("settlement proof hash verification failed")
	}

	// Verify irrevocability rules
	if proof.IsIrrevocable && proof.Confirmations < 6 {
		return fmt.Errorf("insufficient confirmations for irrevocable settlement")
	}

	return nil
}

// CreatePACS008Message creates a credit transfer message (pacs.008)
func CreatePACS008Message(tx *wire.MsgTx, sender, receiver BankIdentifier, amount uint64, reference string) (*ISO20022Message, error) {
	metadata := &TransactionMetadata{
		SenderBIC:   sender.BIC,
		ReceiverBIC: receiver.BIC,
		Reference:   reference,
		Amount:      amount,
		ValueDate:   time.Now(),
	}

	return MapToISO20022(tx, PACS008, metadata)
}

// CreatePACS009Message creates a financial institution transfer message (pacs.009)
func CreatePACS009Message(tx *wire.MsgTx, sender, receiver BankIdentifier, amount uint64) (*ISO20022Message, error) {
	metadata := &TransactionMetadata{
		SenderBIC:   sender.BIC,
		ReceiverBIC: receiver.BIC,
		Amount:      amount,
		ValueDate:   time.Now(),
	}

	return MapToISO20022(tx, PACS009, metadata)
}

// GetSupportedMessageTypes returns all supported ISO 20022 message types
func GetSupportedMessageTypes() []MessageType {
	return []MessageType{PACS008, PACS009, CAMT056, PAIN001}
}

// IsSupported checks if a message type is supported
func IsSupported(msgType MessageType) bool {
	supported := GetSupportedMessageTypes()
	for _, t := range supported {
		if t == msgType {
			return true
		}
	}
	return false
}
