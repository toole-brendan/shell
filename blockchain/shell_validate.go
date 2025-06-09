// Copyright (c) 2025 Shell Reserve developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"fmt"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/toole-brendan/shell/chaincfg"
	"github.com/toole-brendan/shell/privacy/confidential"
)

// Shell-specific validation errors
var (
	ErrConfidentialTxValidation = ruleError(ErrScriptValidation, "confidential transaction validation failed")
	ErrInvalidRangeProof        = ruleError(ErrScriptValidation, "invalid range proof")
	ErrBalanceValidationFailed  = ruleError(ErrScriptValidation, "confidential balance validation failed")
)

// CheckShellTransactionSanity performs Shell-specific transaction validation
// This includes confidential transaction validation on top of standard Bitcoin validation
func CheckShellTransactionSanity(tx *btcutil.Tx, chainParams *chaincfg.Params) error {
	// First run standard Bitcoin validation
	if err := CheckTransactionSanity(tx); err != nil {
		return err
	}

	// Shell-specific validation
	return validateShellSpecificRules(tx, chainParams)
}

// validateShellSpecificRules checks Shell-specific consensus rules
func validateShellSpecificRules(tx *btcutil.Tx, chainParams *chaincfg.Params) error {
	msgTx := tx.MsgTx()

	// Check if this transaction has confidential outputs
	// Look for confidential transaction markers in witness data
	if msgTx.HasWitness() {
		for _, txIn := range msgTx.TxIn {
			if isConfidentialTransactionWitness(txIn.Witness) {
				return validateConfidentialTransaction(tx, chainParams)
			}
		}
	}

	// For now, allow standard transactions alongside confidential ones
	return nil
}

// isConfidentialTransactionWitness checks if witness data indicates confidential transaction
func isConfidentialTransactionWitness(witness [][]byte) bool {
	// Look for confidential transaction markers
	// This is a simplified check - in practice would be more sophisticated
	for _, item := range witness {
		if len(item) >= 4 && string(item[:4]) == "CONF" {
			return true
		}
	}
	return false
}

// validateConfidentialTransaction validates a confidential transaction
func validateConfidentialTransaction(tx *btcutil.Tx, chainParams *chaincfg.Params) error {
	// Extract confidential transaction data from witness
	confTx, err := extractConfidentialTxFromWitness(tx)
	if err != nil {
		return fmt.Errorf("failed to extract confidential tx data: %w", err)
	}

	// Validate range proofs for all confidential outputs
	if !confTx.ValidateRangeProofs() {
		return ErrInvalidRangeProof
	}

	// Note: Balance validation requires input commitments which would come from UTXO set
	// This is a placeholder for the full implementation
	// TODO: Implement full balance validation with input commitments

	return nil
}

// extractConfidentialTxFromWitness extracts confidential transaction data from witness
func extractConfidentialTxFromWitness(tx *btcutil.Tx) (*confidential.ConfidentialTx, error) {
	// This is a placeholder implementation
	// In practice, confidential transaction data would be encoded in witness

	msgTx := tx.MsgTx()
	confTx := confidential.NewConfidentialTx(msgTx)

	// Extract confidential outputs from witness data
	// TODO: Implement proper witness parsing for confidential transaction data

	return confTx, nil
}

// ValidateShellBlock performs Shell-specific block validation
func ValidateShellBlock(block *btcutil.Block, chainParams *chaincfg.Params) error {
	// Validate all transactions in the block
	for _, tx := range block.Transactions() {
		if err := CheckShellTransactionSanity(tx, chainParams); err != nil {
			return err
		}
	}

	// Shell-specific block validation rules
	return validateShellBlockRules(block, chainParams)
}

// validateShellBlockRules checks Shell-specific block consensus rules
func validateShellBlockRules(block *btcutil.Block, chainParams *chaincfg.Params) error {
	msgBlock := block.MsgBlock()

	// Validate block time (5-minute target)
	// Additional Shell-specific validations would go here

	// Validate coinbase has proper Shell subsidy
	if len(msgBlock.Transactions) == 0 {
		return ruleError(ErrNoTransactions, "block has no transactions")
	}

	coinbaseTx := msgBlock.Transactions[0]
	if !IsCoinBaseTx(coinbaseTx) {
		return ruleError(ErrFirstTxNotCoinbase, "first transaction is not coinbase")
	}

	// TODO: Validate Shell-specific coinbase subsidy calculation

	return nil
}

// CalcShellBlockSubsidy calculates the Shell block subsidy
func CalcShellBlockSubsidy(height int32, chainParams *chaincfg.Params) int64 {
	// Shell starts with 95 XSL per block
	// Halves every 262,800 blocks (~10 years)

	baseSubsidy := int64(95 * 1e8) // 95 XSL in satoshis

	if chainParams.SubsidyReductionInterval == 0 {
		return baseSubsidy
	}

	// Calculate number of halvings
	halvings := height / chainParams.SubsidyReductionInterval

	// Subsidy halves for each halving period
	// After 64 halvings, subsidy becomes 0
	if halvings >= 64 {
		return 0
	}

	return baseSubsidy >> uint(halvings)
}

// IsShellTransactionConfidential checks if a transaction uses confidential features
func IsShellTransactionConfidential(tx *btcutil.Tx) bool {
	if !tx.HasWitness() {
		return false
	}

	msgTx := tx.MsgTx()
	for _, txIn := range msgTx.TxIn {
		if isConfidentialTransactionWitness(txIn.Witness) {
			return true
		}
	}

	return false
}
