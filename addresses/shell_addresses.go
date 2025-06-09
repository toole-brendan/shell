// Copyright (c) 2025 Shell Reserve developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Package addresses implements Shell Reserve address generation and validation.
package addresses

import (
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/btcsuite/btcd/btcutil/bech32"
	"github.com/toole-brendan/shell/chaincfg"
	"github.com/toole-brendan/shell/chaincfg/chainhash"
	"github.com/toole-brendan/shell/txscript"
)

const (
	// ShellSegwitHRP is the human-readable part for Shell bech32 addresses
	ShellSegwitHRP = "xsl"

	// AddressTypeTaproot represents Shell Taproot addresses (xsl1...)
	AddressTypeTaproot = "taproot"

	// AddressTypeP2PKH represents Shell P2PKH addresses (legacy style)
	AddressTypeP2PKH = "p2pkh"

	// AddressTypeP2SH represents Shell P2SH addresses (script hash)
	AddressTypeP2SH = "p2sh"
)

var (
	// ErrInvalidAddress is returned when an address format is invalid
	ErrInvalidAddress = errors.New("invalid Shell address format")

	// ErrUnsupportedAddressType is returned for unsupported address types
	ErrUnsupportedAddressType = errors.New("unsupported Shell address type")

	// ErrInvalidPublicKey is returned when a public key is malformed
	ErrInvalidPublicKey = errors.New("invalid public key")
)

// ShellAddress represents a Shell Reserve address
type ShellAddress interface {
	// String returns the human-readable address
	String() string

	// ScriptAddress returns the raw script hash or public key hash
	ScriptAddress() []byte

	// AddressType returns the type of address
	AddressType() string

	// IsForNetwork returns true if the address is for the given network
	IsForNetwork(params *chaincfg.Params) bool
}

// ShellTaprootAddress represents a Shell Taproot address (xsl1...)
type ShellTaprootAddress struct {
	witnessVersion byte
	witnessProgram []byte
	netParams      *chaincfg.Params
}

// NewShellTaprootAddress creates a new Shell Taproot address from a public key
func NewShellTaprootAddress(pubKey *btcec.PublicKey, params *chaincfg.Params) (*ShellTaprootAddress, error) {
	if pubKey == nil {
		return nil, ErrInvalidPublicKey
	}

	// For Taproot, we use the 32-byte x-coordinate of the public key
	witnessProgram := pubKey.SerializeCompressed()[1:] // Remove the 0x02/0x03 prefix

	return &ShellTaprootAddress{
		witnessVersion: 1, // Taproot is witness version 1
		witnessProgram: witnessProgram,
		netParams:      params,
	}, nil
}

// String returns the bech32 encoded Shell Taproot address
func (addr *ShellTaprootAddress) String() string {
	// Convert witness program to 5-bit groups for bech32
	conv, err := bech32.ConvertBits(addr.witnessProgram, 8, 5, true)
	if err != nil {
		return ""
	}

	// Prepend witness version
	data := append([]byte{addr.witnessVersion}, conv...)

	// Encode with Shell HRP
	encoded, err := bech32.Encode(ShellSegwitHRP, data)
	if err != nil {
		return ""
	}

	return encoded
}

// ScriptAddress returns the witness program
func (addr *ShellTaprootAddress) ScriptAddress() []byte {
	return addr.witnessProgram
}

// AddressType returns the address type
func (addr *ShellTaprootAddress) AddressType() string {
	return AddressTypeTaproot
}

// IsForNetwork checks if the address is for the given network
func (addr *ShellTaprootAddress) IsForNetwork(params *chaincfg.Params) bool {
	return addr.netParams.Name == params.Name
}

// ShellP2PKHAddress represents a legacy Shell P2PKH address
type ShellP2PKHAddress struct {
	hash      [20]byte
	netParams *chaincfg.Params
}

// NewShellP2PKHAddress creates a new Shell P2PKH address from a public key hash
func NewShellP2PKHAddress(pubKeyHash []byte, params *chaincfg.Params) (*ShellP2PKHAddress, error) {
	if len(pubKeyHash) != 20 {
		return nil, fmt.Errorf("public key hash must be 20 bytes")
	}

	var hash [20]byte
	copy(hash[:], pubKeyHash)

	return &ShellP2PKHAddress{
		hash:      hash,
		netParams: params,
	}, nil
}

// String returns the base58 encoded Shell P2PKH address
func (addr *ShellP2PKHAddress) String() string {
	// Create versioned payload
	payload := make([]byte, 21)
	payload[0] = addr.netParams.PubKeyHashAddrID
	copy(payload[1:], addr.hash[:])

	// Calculate checksum
	checksum := chainhash.DoubleHashB(payload)[:4]

	// Encode with base58
	fullPayload := append(payload, checksum...)
	return base58.Encode(fullPayload)
}

// ScriptAddress returns the public key hash
func (addr *ShellP2PKHAddress) ScriptAddress() []byte {
	return addr.hash[:]
}

// AddressType returns the address type
func (addr *ShellP2PKHAddress) AddressType() string {
	return AddressTypeP2PKH
}

// IsForNetwork checks if the address is for the given network
func (addr *ShellP2PKHAddress) IsForNetwork(params *chaincfg.Params) bool {
	return addr.netParams.Name == params.Name
}

// GenerateShellAddress generates a Shell address from a public key
func GenerateShellAddress(pubKey *btcec.PublicKey, addressType string, params *chaincfg.Params) (ShellAddress, error) {
	switch addressType {
	case AddressTypeTaproot:
		return NewShellTaprootAddress(pubKey, params)

	case AddressTypeP2PKH:
		// Generate P2PKH address from public key
		pubKeyBytes := pubKey.SerializeCompressed()
		pubKeyHash := btcutil.Hash160(pubKeyBytes)
		return NewShellP2PKHAddress(pubKeyHash, params)

	default:
		return nil, ErrUnsupportedAddressType
	}
}

// ParseShellAddress parses a Shell address string into a ShellAddress
func ParseShellAddress(address string, params *chaincfg.Params) (ShellAddress, error) {
	// Try to parse as bech32 (Taproot)
	if hrp, data, err := bech32.Decode(address); err == nil {
		if hrp == ShellSegwitHRP {
			return parseShellBech32Address(data, params)
		}
	}

	// Try to parse as base58 (P2PKH/P2SH)
	decoded := base58.Decode(address)
	if len(decoded) != 25 {
		return nil, ErrInvalidAddress
	}

	// Verify checksum
	payload := decoded[:21]
	checksum := decoded[21:]
	expectedChecksum := chainhash.DoubleHashB(payload)[:4]

	for i := 0; i < 4; i++ {
		if checksum[i] != expectedChecksum[i] {
			return nil, ErrInvalidAddress
		}
	}

	version := payload[0]
	hash := payload[1:]

	// Check address version and create appropriate address type
	if version == params.PubKeyHashAddrID {
		return NewShellP2PKHAddress(hash, params)
	}

	return nil, ErrUnsupportedAddressType
}

// parseShellBech32Address parses a Shell bech32 address
func parseShellBech32Address(data []byte, params *chaincfg.Params) (ShellAddress, error) {
	if len(data) < 1 {
		return nil, ErrInvalidAddress
	}

	witnessVersion := data[0]
	witnessProgram, err := bech32.ConvertBits(data[1:], 5, 8, false)
	if err != nil {
		return nil, ErrInvalidAddress
	}

	switch witnessVersion {
	case 1: // Taproot
		if len(witnessProgram) != 32 {
			return nil, ErrInvalidAddress
		}
		return &ShellTaprootAddress{
			witnessVersion: witnessVersion,
			witnessProgram: witnessProgram,
			netParams:      params,
		}, nil

	default:
		return nil, ErrUnsupportedAddressType
	}
}

// CreateShellScript creates a script for a Shell address
func CreateShellScript(addr ShellAddress) ([]byte, error) {
	switch a := addr.(type) {
	case *ShellTaprootAddress:
		// Taproot script: OP_1 <32-byte-pubkey>
		return txscript.NewScriptBuilder().
			AddOp(txscript.OP_1).
			AddData(a.witnessProgram).
			Script()

	case *ShellP2PKHAddress:
		// P2PKH script: OP_DUP OP_HASH160 <pubkey-hash> OP_EQUALVERIFY OP_CHECKSIG
		return txscript.NewScriptBuilder().
			AddOp(txscript.OP_DUP).
			AddOp(txscript.OP_HASH160).
			AddData(a.hash[:]).
			AddOp(txscript.OP_EQUALVERIFY).
			AddOp(txscript.OP_CHECKSIG).
			Script()

	default:
		return nil, ErrUnsupportedAddressType
	}
}

// ValidateShellAddress validates a Shell address format and network
func ValidateShellAddress(address string, params *chaincfg.Params) error {
	addr, err := ParseShellAddress(address, params)
	if err != nil {
		return err
	}

	if !addr.IsForNetwork(params) {
		return fmt.Errorf("address is not for network %s", params.Name)
	}

	return nil
}

// GetAddressInfo returns detailed information about a Shell address
func GetAddressInfo(address string, params *chaincfg.Params) (map[string]interface{}, error) {
	addr, err := ParseShellAddress(address, params)
	if err != nil {
		return nil, err
	}

	script, err := CreateShellScript(addr)
	if err != nil {
		return nil, err
	}

	info := map[string]interface{}{
		"address":     addr.String(),
		"type":        addr.AddressType(),
		"network":     params.Name,
		"script_hex":  fmt.Sprintf("%x", script),
		"script_hash": fmt.Sprintf("%x", chainhash.HashB(script)),
	}

	// Add type-specific information
	switch a := addr.(type) {
	case *ShellTaprootAddress:
		info["witness_version"] = a.witnessVersion
		info["witness_program"] = fmt.Sprintf("%x", a.witnessProgram)

	case *ShellP2PKHAddress:
		info["pubkey_hash"] = fmt.Sprintf("%x", a.hash[:])
	}

	return info, nil
}

// IsValidShellAddressFormat performs a quick format validation without network checks
func IsValidShellAddressFormat(address string) bool {
	// Check bech32 format
	if hrp, _, err := bech32.Decode(address); err == nil {
		return hrp == ShellSegwitHRP
	}

	// Check base58 format
	decoded := base58.Decode(address)
	if len(decoded) != 25 {
		return false
	}

	// Verify checksum
	payload := decoded[:21]
	checksum := decoded[21:]
	expectedChecksum := chainhash.DoubleHashB(payload)[:4]

	for i := 0; i < 4; i++ {
		if checksum[i] != expectedChecksum[i] {
			return false
		}
	}

	return true
}

// GenerateMultiSigAddress generates a multi-signature Shell address
func GenerateMultiSigAddress(pubKeys []*btcec.PublicKey, required int, params *chaincfg.Params) (ShellAddress, error) {
	if required <= 0 || required > len(pubKeys) {
		return nil, fmt.Errorf("invalid required signatures: %d of %d", required, len(pubKeys))
	}

	if len(pubKeys) > 15 {
		return nil, fmt.Errorf("too many public keys: %d (max 15)", len(pubKeys))
	}

	// Create multi-sig script
	builder := txscript.NewScriptBuilder()
	builder.AddOp(txscript.OP_1 + byte(required) - 1) // OP_M

	for _, pubKey := range pubKeys {
		builder.AddData(pubKey.SerializeCompressed())
	}

	builder.AddOp(txscript.OP_1 + byte(len(pubKeys)) - 1) // OP_N
	builder.AddOp(txscript.OP_CHECKMULTISIG)

	script, err := builder.Script()
	if err != nil {
		return nil, err
	}

	// Hash the script for P2SH-like functionality
	scriptHash := btcutil.Hash160(script)

	// For now, return as P2PKH-style address (in practice, would need P2SH support)
	return NewShellP2PKHAddress(scriptHash, params)
}
