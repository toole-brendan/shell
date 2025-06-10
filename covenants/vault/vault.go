// Package vault implements Shell Reserve's vault covenant functionality
// for institutional-grade custody with time-delayed spending policies.
package vault

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/toole-brendan/shell/chaincfg/chainhash"
)

// VaultTemplate defines the spending policy for a Shell institutional vault.
// This implements time-delayed recovery mechanisms as specified in the implementation plan.
type VaultTemplate struct {
	// Version for future upgrades
	Version uint16

	// CSVTimeout is the number of blocks until cold recovery is enabled
	CSVTimeout uint32

	// HotThreshold is the number of signatures required for immediate spending
	// Example: 11 for 11-of-15 multisig
	HotThreshold uint8

	// ColdScriptHash is the hash of the recovery script (20 bytes)
	ColdScriptHash [20]byte

	// Reserved for future extensions
	Reserved [8]byte
}

// VaultScript represents a complete vault spending policy
type VaultScript struct {
	Template    VaultTemplate
	HotKeys     []btcec.PublicKey // Hot wallet keys (e.g., 15 keys for 11-of-15)
	ColdKeys    []btcec.PublicKey // Cold recovery keys (e.g., 5 keys for 3-of-5)
	InternalKey *btcec.PublicKey  // Taproot internal key
}

// CentralBankVaultConfig provides a pre-configured vault for central bank use
type CentralBankVaultConfig struct {
	// Hot path: 11-of-15 multisig for day-to-day operations
	HotKeys      []btcec.PublicKey
	HotThreshold uint8 // 11

	// Cold path: 3-of-5 multisig after 30 days (4320 blocks)
	ColdKeys       []btcec.PublicKey
	ColdThreshold  uint8  // 3
	RecoveryBlocks uint32 // 4320 (~30 days)
}

// Hash calculates the SHA256 hash of the vault template for OP_VAULTTEMPLATEVERIFY
func (vt *VaultTemplate) Hash() chainhash.Hash {
	data := make([]byte, 0, 32)

	// Serialize template for hashing
	versionBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(versionBytes, vt.Version)
	data = append(data, versionBytes...)

	timeoutBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(timeoutBytes, vt.CSVTimeout)
	data = append(data, timeoutBytes...)

	data = append(data, vt.HotThreshold)
	data = append(data, vt.ColdScriptHash[:]...)
	data = append(data, vt.Reserved[:]...)

	hash := sha256.Sum256(data)
	return chainhash.Hash(hash)
}

// Serialize converts the vault template to bytes for storage
func (vt *VaultTemplate) Serialize() []byte {
	data := make([]byte, 0, 32)

	versionBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(versionBytes, vt.Version)
	data = append(data, versionBytes...)

	timeoutBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(timeoutBytes, vt.CSVTimeout)
	data = append(data, timeoutBytes...)

	data = append(data, vt.HotThreshold)
	data = append(data, vt.ColdScriptHash[:]...)
	data = append(data, vt.Reserved[:]...)

	return data
}

// DeserializeVaultTemplate creates a VaultTemplate from bytes
func DeserializeVaultTemplate(data []byte) (*VaultTemplate, error) {
	if len(data) < 32 {
		return nil, fmt.Errorf("vault template data too short: %d bytes", len(data))
	}

	vt := &VaultTemplate{}

	vt.Version = binary.LittleEndian.Uint16(data[0:2])
	vt.CSVTimeout = binary.LittleEndian.Uint32(data[2:6])
	vt.HotThreshold = data[6]
	copy(vt.ColdScriptHash[:], data[7:27])
	copy(vt.Reserved[:], data[27:35])

	return vt, nil
}

// CreateCentralBankVault creates a standard vault configuration for central banks
// with 11-of-15 hot spending and 3-of-5 cold recovery after 30 days
func CreateCentralBankVault(config CentralBankVaultConfig) (*VaultScript, error) {
	if len(config.HotKeys) < int(config.HotThreshold) {
		return nil, fmt.Errorf("insufficient hot keys: need %d, have %d",
			config.HotThreshold, len(config.HotKeys))
	}

	if len(config.ColdKeys) < int(config.ColdThreshold) {
		return nil, fmt.Errorf("insufficient cold keys: need %d, have %d",
			config.ColdThreshold, len(config.ColdKeys))
	}

	// Create cold script hash (simplified for now - real implementation would use multisig)
	coldScriptData := make([]byte, 0)
	for _, key := range config.ColdKeys {
		coldScriptData = append(coldScriptData, key.SerializeCompressed()...)
	}
	coldScriptData = append(coldScriptData, config.ColdThreshold)

	coldHash := sha256.Sum256(coldScriptData)
	var coldScriptHash [20]byte
	copy(coldScriptHash[:], coldHash[:20])

	template := VaultTemplate{
		Version:        1,
		CSVTimeout:     config.RecoveryBlocks,
		HotThreshold:   config.HotThreshold,
		ColdScriptHash: coldScriptHash,
	}

	vault := &VaultScript{
		Template: template,
		HotKeys:  config.HotKeys,
		ColdKeys: config.ColdKeys,
	}

	return vault, nil
}

// IsHotSpendValid checks if a hot spend satisfies the vault template requirements
func (vs *VaultScript) IsHotSpendValid(sigCount uint8, lockTime uint32) bool {
	// Hot spend must have sufficient signatures
	if sigCount < vs.Template.HotThreshold {
		return false
	}

	// No time restriction for hot spend
	return true
}

// IsColdRecoveryValid checks if a cold recovery satisfies the vault template requirements
func (vs *VaultScript) IsColdRecoveryValid(lockTime uint32, currentHeight uint32) bool {
	// Cold recovery requires timeout to be satisfied
	if currentHeight < lockTime+vs.Template.CSVTimeout {
		return false
	}

	return true
}

// VaultSpendType represents the type of vault spending operation
type VaultSpendType uint8

const (
	VaultSpendHot VaultSpendType = iota
	VaultSpendCold
)

// VaultSpendInfo contains information about a vault spending attempt
type VaultSpendInfo struct {
	Type        VaultSpendType
	SigCount    uint8
	LockTime    uint32
	BlockHeight uint32
	Template    VaultTemplate
}

// ValidateVaultSpend performs complete validation of a vault spending operation
func ValidateVaultSpend(info VaultSpendInfo) error {
	switch info.Type {
	case VaultSpendHot:
		if info.SigCount < info.Template.HotThreshold {
			return fmt.Errorf("insufficient hot signatures: need %d, have %d",
				info.Template.HotThreshold, info.SigCount)
		}

	case VaultSpendCold:
		requiredHeight := info.LockTime + info.Template.CSVTimeout
		if info.BlockHeight < requiredHeight {
			return fmt.Errorf("cold recovery timeout not met: need height %d, current %d",
				requiredHeight, info.BlockHeight)
		}

	default:
		return fmt.Errorf("unknown vault spend type: %d", info.Type)
	}

	return nil
}

// StandardCentralBankConfig returns a standard vault configuration for central banks
func StandardCentralBankConfig() CentralBankVaultConfig {
	return CentralBankVaultConfig{
		HotThreshold:   11,   // 11-of-15 for hot spending
		ColdThreshold:  3,    // 3-of-5 for cold recovery
		RecoveryBlocks: 4320, // ~30 days at 5-minute blocks
		// HotKeys and ColdKeys would be populated by the institution
	}
}

// GetTimeToRecovery calculates the time remaining until cold recovery is available
func (vs *VaultScript) GetTimeToRecovery(lockTime, currentHeight uint32) time.Duration {
	if currentHeight >= lockTime+vs.Template.CSVTimeout {
		return 0 // Recovery already available
	}

	blocksRemaining := (lockTime + vs.Template.CSVTimeout) - currentHeight
	// Shell uses 5-minute blocks
	minutesRemaining := int64(blocksRemaining) * 5

	return time.Duration(minutesRemaining) * time.Minute
}
