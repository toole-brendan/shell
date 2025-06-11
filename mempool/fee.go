// Package mempool implements Shell Reserve's fee calculation and validation
// for the liquidity reward program and market maker rebates.
package mempool

import (
	"errors"
	"fmt"

	"github.com/toole-brendan/shell/txscript"
	"github.com/toole-brendan/shell/wire"
)

// Fee constants for Shell Reserve
const (
	BaseFeeRate = 0.0003 // 0.0003 XSL per byte (burned)
	MakerRebate = 0.0001 // 0.0001 XSL per byte rebate for makers

	// Special fee amounts for Shell operations
	ChannelOpenFee   = 0.1 * 1e8  // 0.1 XSL to open payment channel
	ChannelUpdateFee = 0.01 * 1e8 // 0.01 XSL for channel updates
	AtomicSwapFee    = 0.05 * 1e8 // 0.05 XSL for atomic swaps
	ClaimableFee     = 0.02 * 1e8 // 0.02 XSL for claimable balances
)

// FeeCalculator handles Shell Reserve fee calculations
type FeeCalculator struct {
	baseFeeRate float64
	makerRebate float64
}

// NewFeeCalculator creates a new fee calculator with Shell parameters
func NewFeeCalculator() *FeeCalculator {
	return &FeeCalculator{
		baseFeeRate: BaseFeeRate,
		makerRebate: MakerRebate,
	}
}

// FeeResult contains the calculated fee and any rebate
type FeeResult struct {
	BaseFee      int64 // Fee to be burned
	MakerRebate  int64 // Rebate for market makers (if applicable)
	OperationFee int64 // Additional fee for Shell operations
	TotalFee     int64 // Total fee to pay
	NetFee       int64 // Net fee after rebates
}

// CalculateFee computes fees for a Shell transaction
func (fc *FeeCalculator) CalculateFee(tx *wire.MsgTx) (*FeeResult, error) {
	if tx == nil {
		return nil, errors.New("transaction cannot be nil")
	}

	// Calculate base fee based on transaction size
	size := tx.SerializeSize()
	baseFee := int64(float64(size) * fc.baseFeeRate * 1e8)

	result := &FeeResult{
		BaseFee: baseFee,
	}

	// Check for maker flag in witness data
	isMaker := fc.checkMakerFlag(tx)
	if isMaker {
		result.MakerRebate = int64(float64(size) * fc.makerRebate * 1e8)
	}

	// Add operation-specific fees for Shell opcodes
	operationFee, err := fc.calculateOperationFee(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate operation fee: %v", err)
	}
	result.OperationFee = operationFee

	// Calculate totals
	result.TotalFee = result.BaseFee + result.OperationFee
	result.NetFee = result.TotalFee - result.MakerRebate

	// Ensure net fee is not negative
	if result.NetFee < 0 {
		result.NetFee = 0
	}

	return result, nil
}

// checkMakerFlag examines witness data for market maker indicators
func (fc *FeeCalculator) checkMakerFlag(tx *wire.MsgTx) bool {
	// Check each input's witness for maker flag
	for _, txIn := range tx.TxIn {
		if len(txIn.Witness) > 0 {
			// Look for maker flag in witness data
			// This is a simplified implementation - in production,
			// the maker flag would be part of the attestation system
			for _, witnessItem := range txIn.Witness {
				if len(witnessItem) >= 4 &&
					witnessItem[0] == 0x4d && // 'M'
					witnessItem[1] == 0x41 && // 'A'
					witnessItem[2] == 0x4b && // 'K'
					witnessItem[3] == 0x52 { // 'R'
					return true
				}
			}
		}
	}
	return false
}

// calculateOperationFee adds fees for Shell-specific operations
func (fc *FeeCalculator) calculateOperationFee(tx *wire.MsgTx) (int64, error) {
	var totalOperationFee int64

	// Check for Shell opcodes in outputs and inputs
	for _, txOut := range tx.TxOut {
		opcodeFee, err := fc.getOpcodeDefinedFee(txOut.PkScript)
		if err != nil {
			return 0, err
		}
		totalOperationFee += opcodeFee
	}

	// Check witness data for Shell operations
	for _, txIn := range tx.TxIn {
		for _, witnessItem := range txIn.Witness {
			opcodeFee, err := fc.getOpcodeDefinedFee(witnessItem)
			if err != nil {
				return 0, err
			}
			totalOperationFee += opcodeFee
		}
	}

	return totalOperationFee, nil
}

// getOpcodeDefinedFee returns the fee for Shell-specific opcodes
func (fc *FeeCalculator) getOpcodeDefinedFee(script []byte) (int64, error) {
	if len(script) == 0 {
		return 0, nil
	}

	// Parse script for Shell opcodes
	tokenizer := txscript.MakeScriptTokenizer(0, script)

	for tokenizer.Next() {
		opcode := tokenizer.Opcode()

		switch opcode {
		case 0xc6: // OP_CHANNEL_OPEN
			return int64(ChannelOpenFee), nil

		case 0xc7: // OP_CHANNEL_UPDATE
			return int64(ChannelUpdateFee), nil

		case 0xc8: // OP_CHANNEL_CLOSE
			return 0, nil // No additional fee for closing

		case 0xc9: // OP_CLAIMABLE_CREATE
			return int64(ClaimableFee), nil

		case 0xca: // OP_CLAIMABLE_CLAIM
			return 0, nil // No additional fee for claiming
		}
	}

	if err := tokenizer.Err(); err != nil {
		return 0, fmt.Errorf("script parsing error: %v", err)
	}

	return 0, nil
}

// ValidateFee checks if the transaction pays sufficient fees
func (fc *FeeCalculator) ValidateFee(tx *wire.MsgTx, inputValue int64, outputValue int64) error {
	feeResult, err := fc.CalculateFee(tx)
	if err != nil {
		return fmt.Errorf("fee calculation failed: %v", err)
	}

	actualFee := inputValue - outputValue
	requiredFee := feeResult.NetFee

	if actualFee < requiredFee {
		return fmt.Errorf("insufficient fee: paid %d, required %d", actualFee, requiredFee)
	}

	return nil
}

// CalculateMinimumFee returns the minimum fee required for a transaction
func (fc *FeeCalculator) CalculateMinimumFee(tx *wire.MsgTx) (int64, error) {
	result, err := fc.CalculateFee(tx)
	if err != nil {
		return 0, err
	}
	return result.NetFee, nil
}

// GetFeeRate returns the current base fee rate
func (fc *FeeCalculator) GetFeeRate() float64 {
	return fc.baseFeeRate
}

// GetMakerRebateRate returns the current maker rebate rate
func (fc *FeeCalculator) GetMakerRebateRate() float64 {
	return fc.makerRebate
}

// EstimateFee provides a fee estimate without fully parsing the transaction
func (fc *FeeCalculator) EstimateFee(txSize int, hasShellOpcodes bool, isMaker bool) int64 {
	baseFee := int64(float64(txSize) * fc.baseFeeRate * 1e8)

	var rebate int64
	if isMaker {
		rebate = int64(float64(txSize) * fc.makerRebate * 1e8)
	}

	var operationFee int64
	if hasShellOpcodes {
		operationFee = int64(ChannelOpenFee) // Conservative estimate
	}

	netFee := baseFee + operationFee - rebate
	if netFee < 0 {
		netFee = 0
	}

	return netFee
}
