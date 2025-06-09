// Copyright (c) 2025 Shell Reserve developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package genesis

import (
	"crypto/sha256"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

// Shell Constitution Text - immutable principles
const ConstitutionText = `
Shell Reserve Constitutional Principles (Immutable)

1. Single Purpose: Store value securely for decades, nothing else
2. Political Neutrality: No privileged parties, no premine, pure fair launch
3. Institutional First: Designed for central banks and sovereign wealth funds
4. Generational Thinking: Built for 100-year operation, not quarterly profits
5. Boring by Design: Stability and predictability over innovation
6. Mathematical Security: Governed by consensus and cryptography, not committees
7. Reserve Asset Mandate: Digital gold that acts like gold - rare, boring, reliable

Launch Commitment: January 1, 2026, 00:00 UTC
No premine. No special allocations. No privileged parties.
Pure proof-of-work distribution from block zero.

"Built to last, not to impress."
`

// CreateShellGenesisBlock creates the Shell Reserve genesis block
func CreateShellGenesisBlock() *wire.MsgBlock {
	// Constitution hash for verifiable commitment to principles
	constitutionHash := sha256.Sum256([]byte(ConstitutionText))

	// Genesis block message - includes constitution commitment and timestamp proof
	genesisMessage := []byte("Shell Reserve Genesis Block - Fair Launch January 1, 2026")
	genesisMessage = append(genesisMessage, constitutionHash[:]...)

	// Add newspaper headline for timestamp proof
	timestampProof := []byte("FT 2025-12-31: Central Banks Accelerate Gold Buying as Dollar Weaponization Concerns Mount")
	genesisMessage = append(genesisMessage, timestampProof...)

	// Genesis coinbase transaction (no premine - unspendable output)
	genesisCoinbase := &wire.MsgTx{
		Version: 2, // Shell starts with version 2+ blocks
		TxIn: []*wire.TxIn{{
			PreviousOutPoint: wire.OutPoint{
				Hash:  chainhash.Hash{}, // Null hash for coinbase
				Index: 0xffffffff,       // Max index for coinbase
			},
			SignatureScript: genesisMessage,
			Sequence:        0xffffffff,
		}},
		TxOut: []*wire.TxOut{{
			Value:    0,                          // NO PREMINE - zero value output
			PkScript: []byte{txscript.OP_RETURN}, // Unspendable OP_RETURN
		}},
		LockTime: 0,
	}

	// Genesis block header
	genesisHeader := wire.BlockHeader{
		Version:    2,                                           // Shell starts with v2+ for height serialization
		PrevBlock:  chainhash.Hash{},                            // Null previous block
		MerkleRoot: genesisCoinbase.TxHash(),                    // Merkle root of single coinbase
		Timestamp:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), // Fair launch time
		Bits:       0x1d00ffff,                                  // Initial difficulty (same as Bitcoin genesis)
		Nonce:      0,                                           // To be filled by miner
	}

	// Genesis block
	genesisBlock := &wire.MsgBlock{
		Header:       genesisHeader,
		Transactions: []*wire.MsgTx{genesisCoinbase},
	}

	return genesisBlock
}

// GetShellGenesisHash returns the hash of the Shell genesis block
func GetShellGenesisHash() *chainhash.Hash {
	genesisBlock := CreateShellGenesisBlock()
	hash := genesisBlock.Header.BlockHash()
	return &hash
}

// GetConstitutionHash returns the SHA256 hash of the Shell constitution
func GetConstitutionHash() [32]byte {
	return sha256.Sum256([]byte(ConstitutionText))
}

// VerifyConstitutionCommitment verifies that a block contains the constitution commitment
func VerifyConstitutionCommitment(block *wire.MsgBlock) bool {
	if len(block.Transactions) == 0 {
		return false
	}

	coinbase := block.Transactions[0]
	if len(coinbase.TxIn) == 0 {
		return false
	}

	script := coinbase.TxIn[0].SignatureScript
	constitutionHash := GetConstitutionHash()

	// Check if constitution hash is present in coinbase script
	for i := 0; i <= len(script)-32; i++ {
		if script[i] == constitutionHash[0] { // First byte match
			// Check full hash
			match := true
			for j := 0; j < 32; j++ {
				if i+j >= len(script) || script[i+j] != constitutionHash[j] {
					match = false
					break
				}
			}
			if match {
				return true
			}
		}
	}

	return false
}
