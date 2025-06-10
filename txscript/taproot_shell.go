// Package txscript implements Shell Reserve's Taproot integration
// with BIP 340/341/342 support and Shell-specific validation rules.
package txscript

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/toole-brendan/shell/chaincfg"
	"github.com/toole-brendan/shell/covenants/vault"
	"github.com/toole-brendan/shell/wire"
)

// ShellTaprootVersion is the witness version for Shell Taproot outputs (same as Bitcoin)
const ShellTaprootVersion = 0x01

// ShellTaprootLeafVersion is the leaf version for Shell tapscript
// Using 0xC2 to differentiate from BaseLeafVersion (0xC0)
const ShellTaprootLeafVersion = 0xC2

// ShellTaprootBuilder builds Taproot outputs with Shell-specific features
type ShellTaprootBuilder struct {
	internalKey *btcec.PublicKey
	leaves      []TapLeaf

	// Shell-specific fields
	vaultTemplate *vault.VaultTemplate
	isVault       bool
	isChannel     bool
	isClaimable   bool
}

// NewShellTaprootBuilder creates a new builder for Shell Taproot outputs
func NewShellTaprootBuilder(internalKey *btcec.PublicKey) *ShellTaprootBuilder {
	return &ShellTaprootBuilder{
		internalKey: internalKey,
		leaves:      make([]TapLeaf, 0),
	}
}

// AddVaultLeaf adds a vault covenant leaf to the Taproot tree
func (stb *ShellTaprootBuilder) AddVaultLeaf(template *vault.VaultTemplate, script []byte) error {
	if stb.isChannel || stb.isClaimable {
		return errors.New("cannot mix vault with channel/claimable outputs")
	}

	stb.isVault = true
	stb.vaultTemplate = template

	// Create leaf with vault-specific version
	leaf := NewTapLeaf(ShellTaprootLeafVersion, script)
	stb.leaves = append(stb.leaves, leaf)

	return nil
}

// AddChannelLeaf adds a payment channel leaf to the Taproot tree
func (stb *ShellTaprootBuilder) AddChannelLeaf(script []byte) error {
	if stb.isVault || stb.isClaimable {
		return errors.New("cannot mix channel with vault/claimable outputs")
	}

	stb.isChannel = true

	// Create leaf with standard tapscript version
	leaf := NewTapLeaf(BaseLeafVersion, script)
	stb.leaves = append(stb.leaves, leaf)

	return nil
}

// AddClaimableLeaf adds a claimable balance leaf to the Taproot tree
func (stb *ShellTaprootBuilder) AddClaimableLeaf(script []byte) error {
	if stb.isVault || stb.isChannel {
		return errors.New("cannot mix claimable with vault/channel outputs")
	}

	stb.isClaimable = true

	// Create leaf with standard tapscript version
	leaf := NewTapLeaf(BaseLeafVersion, script)
	stb.leaves = append(stb.leaves, leaf)

	return nil
}

// Build constructs the final Taproot output
func (stb *ShellTaprootBuilder) Build() ([]byte, error) {
	if stb.internalKey == nil {
		return nil, errors.New("internal key required")
	}

	// Build standard Taproot tree
	tapscriptTree := AssembleTaprootScriptTree(stb.leaves...)
	tapscriptRoot := tapscriptTree.RootNode.TapHash()

	// Compute output key
	outputKey := ComputeTaprootOutputKey(stb.internalKey, tapscriptRoot[:])

	// Create witness v1 output script
	builder := NewScriptBuilder()
	builder.AddOp(OP_1)
	builder.AddData(schnorr.SerializePubKey(outputKey))

	return builder.Script()
}

// VerifyShellTaprootSpend verifies a Shell Taproot spend with institutional features
func VerifyShellTaprootSpend(vm *Engine) error {
	// Extract witness from transaction input
	if vm.txIdx >= len(vm.tx.TxIn) {
		return errors.New("invalid transaction index")
	}

	witness := vm.tx.TxIn[vm.txIdx].Witness
	if len(witness) < 2 {
		return errors.New("insufficient witness items")
	}

	// Extract control block to get leaf version
	controlBlockBytes := witness[len(witness)-1]
	if len(controlBlockBytes) < 33 {
		return errors.New("control block too small")
	}

	// Parse leaf version from control block
	leafVersion := controlBlockBytes[0] & 0xfe

	// Apply Shell-specific validation rules based on leaf version
	switch leafVersion {
	case ShellTaprootLeafVersion:
		// This is a vault covenant - apply vault rules
		return verifyShellVaultRules(vm)

	case byte(BaseLeafVersion):
		// Standard tapscript - check for Shell opcodes
		return verifyShellOpcodeRules(vm)

	default:
		// Unknown leaf version
		return fmt.Errorf("unknown Shell leaf version: %d", leafVersion)
	}
}

// verifyShellVaultRules applies Shell vault covenant validation
func verifyShellVaultRules(vm *Engine) error {
	// Extract vault template from witness
	if vm.txIdx >= len(vm.tx.TxIn) {
		return errors.New("invalid transaction index")
	}

	witness := vm.tx.TxIn[vm.txIdx].Witness
	if len(witness) < 3 {
		return errors.New("insufficient witness data for vault")
	}

	// Witness stack for vault spend:
	// [signature(s)] [vault_template] [script] [control_block]

	// TODO: Full vault validation including:
	// - Template hash verification
	// - Time lock checking
	// - Signature threshold validation
	// - Hot/cold key verification

	return nil
}

// verifyShellOpcodeRules ensures Shell opcodes are used correctly
func verifyShellOpcodeRules(vm *Engine) error {
	// Scan script for Shell-specific opcodes
	script := vm.scripts[vm.scriptIdx]

	hasVaultOp := bytes.Contains(script, []byte{OP_VAULTTEMPLATEVERIFY})
	hasChannelOp := bytes.Contains(script, []byte{OP_CHANNEL_OPEN}) ||
		bytes.Contains(script, []byte{OP_CHANNEL_UPDATE}) ||
		bytes.Contains(script, []byte{OP_CHANNEL_CLOSE})
	hasClaimableOp := bytes.Contains(script, []byte{OP_CLAIMABLE_CREATE}) ||
		bytes.Contains(script, []byte{OP_CLAIMABLE_CLAIM})

	// Ensure opcodes aren't mixed inappropriately
	opCount := 0
	if hasVaultOp {
		opCount++
	}
	if hasChannelOp {
		opCount++
	}
	if hasClaimableOp {
		opCount++
	}

	if opCount > 1 {
		return errors.New("cannot mix vault, channel, and claimable opcodes")
	}

	// Additional validation based on transaction type
	if hasChannelOp {
		return verifyChannelTransaction(vm)
	}

	if hasClaimableOp {
		return verifyClaimableTransaction(vm)
	}

	return nil
}

// verifyChannelTransaction performs channel-specific validation
func verifyChannelTransaction(vm *Engine) error {
	// Channel transactions must:
	// 1. Have appropriate witness structure
	// 2. Include valid channel state
	// 3. Satisfy signature requirements

	// TODO: Full channel validation

	return nil
}

// verifyClaimableTransaction performs claimable balance validation
func verifyClaimableTransaction(vm *Engine) error {
	// Claimable transactions must:
	// 1. Reference valid claimable balance
	// 2. Satisfy predicates
	// 3. Have valid proofs

	// TODO: Full claimable validation

	return nil
}

// ComputeShellTaprootAddress generates a Shell Taproot address
func ComputeShellTaprootAddress(internalKey *btcec.PublicKey, scriptRoot []byte, params *chaincfg.Params) (string, error) {
	// Compute output key
	outputKey := ComputeTaprootOutputKey(internalKey, scriptRoot)

	// Create witness v1 program
	witnessProgram := schnorr.SerializePubKey(outputKey)

	// Encode as bech32m with Shell prefix
	// TODO: Implement proper bech32m encoding for Shell addresses
	// For now, return a placeholder
	return fmt.Sprintf("xsl1%x", witnessProgram), nil
}

// CreateShellVaultOutput creates a Taproot output with vault covenant
func CreateShellVaultOutput(amount int64, vaultConfig vault.CentralBankVaultConfig) (*wire.TxOut, error) {
	// Generate internal key for key-spend path (emergency recovery)
	internalPrivKey, err := btcec.NewPrivateKey()
	if err != nil {
		return nil, err
	}
	internalKey := internalPrivKey.PubKey()

	// Create vault
	vaultScript, err := vault.CreateCentralBankVault(vaultConfig)
	if err != nil {
		return nil, err
	}

	// Build Taproot output
	builder := NewShellTaprootBuilder(internalKey)

	// Add vault template leaf
	if err := builder.AddVaultLeaf(&vaultScript.Template, nil); err != nil {
		return nil, err
	}

	// Build output script
	outputScript, err := builder.Build()
	if err != nil {
		return nil, err
	}

	return &wire.TxOut{
		Value:    amount,
		PkScript: outputScript,
	}, nil
}

// ShellTaprootSigHashType represents allowed sighash types for Shell
type ShellTaprootSigHashType uint8

const (
	// ShellSigHashDefault is the default sighash type (0x00)
	ShellSigHashDefault ShellTaprootSigHashType = 0x00

	// ShellSigHashAll signs all inputs and outputs
	ShellSigHashAll ShellTaprootSigHashType = 0x01

	// ShellSigHashNone signs all inputs, no outputs
	ShellSigHashNone ShellTaprootSigHashType = 0x02

	// ShellSigHashSingle signs all inputs and corresponding output
	ShellSigHashSingle ShellTaprootSigHashType = 0x03

	// ShellSigHashAnyOneCanPay can be combined with above
	ShellSigHashAnyOneCanPay ShellTaprootSigHashType = 0x80
)

// ComputeShellTaprootSigHash computes the signature hash for Shell Taproot spends
func ComputeShellTaprootSigHash(tx *wire.MsgTx, idx int, prevOuts []wire.TxOut,
	sigHashType ShellTaprootSigHashType, leafScript []byte, leafVersion uint8) ([32]byte, error) {

	// Use standard BIP 341 sighash computation
	sigHashes := NewTxSigHashes(tx, NewMultiPrevOutFetcher(prevOuts))

	// Convert Shell sighash type to standard type
	standardType := SigHashType(sigHashType)

	// Compute hash based on spend type
	if leafScript == nil {
		// Key spend path
		hash, err := calcTaprootSignatureHashRaw(
			sigHashes, standardType, tx, idx, NewMultiPrevOutFetcher(prevOuts),
		)
		if err != nil {
			return [32]byte{}, err
		}

		var result [32]byte
		copy(result[:], hash)
		return result, nil
	} else {
		// Script spend path
		hash, err := calcTaprootSignatureHashRaw(
			sigHashes, standardType, tx, idx, NewMultiPrevOutFetcher(prevOuts),
		)
		if err != nil {
			return [32]byte{}, err
		}

		// Add leaf-specific data for Shell validation
		h := sha256.New()
		h.Write(hash)
		h.Write([]byte{leafVersion})
		h.Write(leafScript)

		var result [32]byte
		copy(result[:], h.Sum(nil))
		return result, nil
	}
}

// ValidateShellTaprootWitness validates witness data for Shell Taproot spends
func ValidateShellTaprootWitness(witness wire.TxWitness, prevOut *wire.TxOut) error {
	// Minimum witness: [signature] [script] [control_block]
	if len(witness) < 2 {
		return errors.New("insufficient witness items for Taproot spend")
	}

	// Extract control block (last item)
	controlBlockBytes := witness[len(witness)-1]
	if len(controlBlockBytes) < 33 {
		return errors.New("control block too small")
	}

	// Parse leaf version
	leafVersion := controlBlockBytes[0] & 0xfe

	// Apply Shell-specific validation based on leaf version
	switch leafVersion {
	case ShellTaprootLeafVersion:
		// Vault covenant validation
		if len(witness) < 4 {
			return errors.New("vault spend requires additional witness data")
		}
		// TODO: Validate vault template and signatures

	case BaseLeafVersion:
		// Standard tapscript validation
		// Shell opcodes will be validated during script execution

	default:
		return fmt.Errorf("unsupported leaf version: %d", leafVersion)
	}

	return nil
}
