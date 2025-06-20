// Copyright (c) 2014-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Copyright (c) 2025 Shell Reserve developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package chaincfg

import (
	"crypto/sha256"
	"time"

	"github.com/toole-brendan/shell/chaincfg/chainhash"
	"github.com/toole-brendan/shell/wire"
)

// Shell Constitution Text - immutable principles
const shellConstitutionText = `
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

// genesisCoinbaseTx is the coinbase transaction for the genesis blocks for
// the main network, regression test network, and test network (version 3).
var genesisCoinbaseTx = wire.MsgTx{
	Version: 1,
	TxIn: []*wire.TxIn{
		{
			PreviousOutPoint: wire.OutPoint{
				Hash:  chainhash.Hash{},
				Index: 0xffffffff,
			},
			SignatureScript: []byte{
				0x04, 0xff, 0xff, 0x00, 0x1d, 0x01, 0x04, 0x45, /* |.......E| */
				0x54, 0x68, 0x65, 0x20, 0x54, 0x69, 0x6d, 0x65, /* |The Time| */
				0x73, 0x20, 0x30, 0x33, 0x2f, 0x4a, 0x61, 0x6e, /* |s 03/Jan| */
				0x2f, 0x32, 0x30, 0x30, 0x39, 0x20, 0x43, 0x68, /* |/2009 Ch| */
				0x61, 0x6e, 0x63, 0x65, 0x6c, 0x6c, 0x6f, 0x72, /* |ancellor| */
				0x20, 0x6f, 0x6e, 0x20, 0x62, 0x72, 0x69, 0x6e, /* | on brin| */
				0x6b, 0x20, 0x6f, 0x66, 0x20, 0x73, 0x65, 0x63, /* |k of sec|*/
				0x6f, 0x6e, 0x64, 0x20, 0x62, 0x61, 0x69, 0x6c, /* |ond bail| */
				0x6f, 0x75, 0x74, 0x20, 0x66, 0x6f, 0x72, 0x20, /* |out for |*/
				0x62, 0x61, 0x6e, 0x6b, 0x73, /* |banks| */
			},
			Sequence: 0xffffffff,
		},
	},
	TxOut: []*wire.TxOut{
		{
			Value: 0x12a05f200,
			PkScript: []byte{
				0x41, 0x04, 0x67, 0x8a, 0xfd, 0xb0, 0xfe, 0x55, /* |A.g....U| */
				0x48, 0x27, 0x19, 0x67, 0xf1, 0xa6, 0x71, 0x30, /* |H'.g..q0| */
				0xb7, 0x10, 0x5c, 0xd6, 0xa8, 0x28, 0xe0, 0x39, /* |..\..(.9| */
				0x09, 0xa6, 0x79, 0x62, 0xe0, 0xea, 0x1f, 0x61, /* |..yb...a| */
				0xde, 0xb6, 0x49, 0xf6, 0xbc, 0x3f, 0x4c, 0xef, /* |..I..?L.| */
				0x38, 0xc4, 0xf3, 0x55, 0x04, 0xe5, 0x1e, 0xc1, /* |8..U....| */
				0x12, 0xde, 0x5c, 0x38, 0x4d, 0xf7, 0xba, 0x0b, /* |..\8M...| */
				0x8d, 0x57, 0x8a, 0x4c, 0x70, 0x2b, 0x6b, 0xf1, /* |.W.Lp+k.| */
				0x1d, 0x5f, 0xac, /* |._.| */
			},
		},
	},
	LockTime: 0,
}

// genesisHash is the hash of the first block in the block chain for the main
// network (genesis block).
var genesisHash = chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
	0x6f, 0xe2, 0x8c, 0x0a, 0xb6, 0xf1, 0xb3, 0x72,
	0xc1, 0xa6, 0xa2, 0x46, 0xae, 0x63, 0xf7, 0x4f,
	0x93, 0x1e, 0x83, 0x65, 0xe1, 0x5a, 0x08, 0x9c,
	0x68, 0xd6, 0x19, 0x00, 0x00, 0x00, 0x00, 0x00,
})

// genesisMerkleRoot is the hash of the first transaction in the genesis block
// for the main network.
var genesisMerkleRoot = chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
	0x3b, 0xa3, 0xed, 0xfd, 0x7a, 0x7b, 0x12, 0xb2,
	0x7a, 0xc7, 0x2c, 0x3e, 0x67, 0x76, 0x8f, 0x61,
	0x7f, 0xc8, 0x1b, 0xc3, 0x88, 0x8a, 0x51, 0x32,
	0x3a, 0x9f, 0xb8, 0xaa, 0x4b, 0x1e, 0x5e, 0x4a,
})

// genesisBlock defines the genesis block of the block chain which serves as the
// public transaction ledger for the main network.
var genesisBlock = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version:    1,
		PrevBlock:  chainhash.Hash{},         // 0000000000000000000000000000000000000000000000000000000000000000
		MerkleRoot: genesisMerkleRoot,        // 4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b
		Timestamp:  time.Unix(0x495fab29, 0), // 2009-01-03 18:15:05 +0000 UTC
		Bits:       0x1d00ffff,               // 486604799 [00000000ffff0000000000000000000000000000000000000000000000000000]
		Nonce:      0x7c2bac1d,               // 2083236893
	},
	Transactions: []*wire.MsgTx{&genesisCoinbaseTx},
}

// regTestGenesisHash is the hash of the first block in the block chain for the
// regression test network (genesis block).
var regTestGenesisHash = chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
	0x06, 0x22, 0x6e, 0x46, 0x11, 0x1a, 0x0b, 0x59,
	0xca, 0xaf, 0x12, 0x60, 0x43, 0xeb, 0x5b, 0xbf,
	0x28, 0xc3, 0x4f, 0x3a, 0x5e, 0x33, 0x2a, 0x1f,
	0xc7, 0xb2, 0xb7, 0x3c, 0xf1, 0x88, 0x91, 0x0f,
})

// regTestGenesisMerkleRoot is the hash of the first transaction in the genesis
// block for the regression test network.  It is the same as the merkle root for
// the main network.
var regTestGenesisMerkleRoot = genesisMerkleRoot

// regTestGenesisBlock defines the genesis block of the block chain which serves
// as the public transaction ledger for the regression test network.
var regTestGenesisBlock = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version:    1,
		PrevBlock:  chainhash.Hash{},         // 0000000000000000000000000000000000000000000000000000000000000000
		MerkleRoot: regTestGenesisMerkleRoot, // 4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b
		Timestamp:  time.Unix(1296688602, 0), // 2011-02-02 23:16:42 +0000 UTC
		Bits:       0x207fffff,               // 545259519 [7fffff0000000000000000000000000000000000000000000000000000000000]
		Nonce:      2,
	},
	Transactions: []*wire.MsgTx{&genesisCoinbaseTx},
}

// testNet3GenesisHash is the hash of the first block in the block chain for the
// test network (version 3).
var testNet3GenesisHash = chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
	0x43, 0x49, 0x7f, 0xd7, 0xf8, 0x26, 0x95, 0x71,
	0x08, 0xf4, 0xa3, 0x0f, 0xd9, 0xce, 0xc3, 0xae,
	0xba, 0x79, 0x97, 0x20, 0x84, 0xe9, 0x0e, 0xad,
	0x01, 0xea, 0x33, 0x09, 0x00, 0x00, 0x00, 0x00,
})

// testNet3GenesisMerkleRoot is the hash of the first transaction in the genesis
// block for the test network (version 3).  It is the same as the merkle root
// for the main network.
var testNet3GenesisMerkleRoot = genesisMerkleRoot

// testNet3GenesisBlock defines the genesis block of the block chain which
// serves as the public transaction ledger for the test network (version 3).
var testNet3GenesisBlock = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version:    1,
		PrevBlock:  chainhash.Hash{},          // 0000000000000000000000000000000000000000000000000000000000000000
		MerkleRoot: testNet3GenesisMerkleRoot, // 4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b
		Timestamp:  time.Unix(1296688602, 0),  // 2011-02-02 23:16:42 +0000 UTC
		Bits:       0x1d00ffff,                // 486604799 [00000000ffff0000000000000000000000000000000000000000000000000000]
		Nonce:      0x18aea41a,                // 414098458
	},
	Transactions: []*wire.MsgTx{&genesisCoinbaseTx},
}

// testNet4GenesisTx is the transaction for the genesis blocks for test network (version 4).
var testNet4GenesisTx = wire.MsgTx{
	Version: 1,
	TxIn: []*wire.TxIn{
		{
			PreviousOutPoint: wire.OutPoint{
				Hash:  chainhash.Hash{},
				Index: 0xffffffff,
			},
			SignatureScript: []byte{
				// Message: `03/May/2024 000000000000000000001ebd58c244970b3aa9d783bb001011fbe8ea8e98e00e`
				0x4, 0xff, 0xff, 0x0, 0x1d, 0x1, 0x4, 0x4c,
				0x4c, 0x30, 0x33, 0x2f, 0x4d, 0x61, 0x79, 0x2f,
				0x32, 0x30, 0x32, 0x34, 0x20, 0x30, 0x30, 0x30,
				0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30,
				0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30,
				0x30, 0x31, 0x65, 0x62, 0x64, 0x35, 0x38, 0x63,
				0x32, 0x34, 0x34, 0x39, 0x37, 0x30, 0x62, 0x33,
				0x61, 0x61, 0x39, 0x64, 0x37, 0x38, 0x33, 0x62,
				0x62, 0x30, 0x30, 0x31, 0x30, 0x31, 0x31, 0x66,
				0x62, 0x65, 0x38, 0x65, 0x61, 0x38, 0x65, 0x39,
				0x38, 0x65, 0x30, 0x30, 0x65},
			Sequence: 0xffffffff,
		},
	},
	TxOut: []*wire.TxOut{
		{
			Value: 0x12a05f200,
			PkScript: []byte{
				0x21, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				0x0, 0x0, 0xac},
		},
	},
	LockTime: 0,
}

// testNet4GenesisHash is the hash of the first block in the block chain for the
// test network (version 4).
var testNet4GenesisHash = chainhash.Hash([chainhash.HashSize]byte{
	0x43, 0xf0, 0x8b, 0xda, 0xb0, 0x50, 0xe3, 0x5b,
	0x56, 0x7c, 0x86, 0x4b, 0x91, 0xf4, 0x7f, 0x50,
	0xae, 0x72, 0x5a, 0xe2, 0xde, 0x53, 0xbc, 0xfb,
	0xba, 0xf2, 0x84, 0xda, 0x00, 0x00, 0x00, 0x00})

// testNet4GenesisMerkleRoot is the hash of the first transaction in the genesis
// block for the test network (version 4).  It is the same as the merkle root
// for the main network.
var testNet4GenesisMerkleRoot = chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
	0x4e, 0x7b, 0x2b, 0x91, 0x28, 0xfe, 0x02, 0x91,
	0xdb, 0x06, 0x93, 0xaf, 0x2a, 0xe4, 0x18, 0xb7,
	0x67, 0xe6, 0x57, 0xcd, 0x40, 0x7e, 0x80, 0xcb,
	0x14, 0x34, 0x22, 0x1e, 0xae, 0xa7, 0xa0, 0x7a,
})

// testNet4GenesisBlock defines the genesis block of the block chain which
// serves as the public transaction ledger for the test network (version 3).
var testNet4GenesisBlock = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version:    1,
		PrevBlock:  chainhash.Hash{},          // 0000000000000000000000000000000000000000000000000000000000000000
		MerkleRoot: testNet4GenesisMerkleRoot, // 4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b
		Timestamp:  time.Unix(1714777860, 0),  // 2024-05-03 23:11:00 +0000 UTC
		Bits:       0x1d00ffff,                // 486604799 [00000000ffff0000000000000000000000000000000000000000000000000000]
		Nonce:      0x17780cbb,                // 393743547
	},
	Transactions: []*wire.MsgTx{&testNet4GenesisTx},
}

// simNetGenesisHash is the hash of the first block in the block chain for the
// simulation test network.
var simNetGenesisHash = chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
	0xf6, 0x7a, 0xd7, 0x69, 0x5d, 0x9b, 0x66, 0x2a,
	0x72, 0xff, 0x3d, 0x8e, 0xdb, 0xbb, 0x2d, 0xe0,
	0xbf, 0xa6, 0x7b, 0x13, 0x97, 0x4b, 0xb9, 0x91,
	0x0d, 0x11, 0x6d, 0x5c, 0xbd, 0x86, 0x3e, 0x68,
})

// simNetGenesisMerkleRoot is the hash of the first transaction in the genesis
// block for the simulation test network.  It is the same as the merkle root for
// the main network.
var simNetGenesisMerkleRoot = genesisMerkleRoot

// simNetGenesisBlock defines the genesis block of the block chain which serves
// as the public transaction ledger for the simulation test network.
var simNetGenesisBlock = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version:    1,
		PrevBlock:  chainhash.Hash{},         // 0000000000000000000000000000000000000000000000000000000000000000
		MerkleRoot: simNetGenesisMerkleRoot,  // 4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b
		Timestamp:  time.Unix(1401292357, 0), // 2014-05-28 15:52:37 +0000 UTC
		Bits:       0x207fffff,               // 545259519 [7fffff0000000000000000000000000000000000000000000000000000000000]
		Nonce:      2,
	},
	Transactions: []*wire.MsgTx{&genesisCoinbaseTx},
}

// sigNetGenesisHash is the hash of the first block in the block chain for the
// signet test network.
var sigNetGenesisHash = chainhash.Hash{
	0xf6, 0x1e, 0xee, 0x3b, 0x63, 0xa3, 0x80, 0xa4,
	0x77, 0xa0, 0x63, 0xaf, 0x32, 0xb2, 0xbb, 0xc9,
	0x7c, 0x9f, 0xf9, 0xf0, 0x1f, 0x2c, 0x42, 0x25,
	0xe9, 0x73, 0x98, 0x81, 0x08, 0x00, 0x00, 0x00,
}

// sigNetGenesisMerkleRoot is the hash of the first transaction in the genesis
// block for the signet test network. It is the same as the merkle root for
// the main network.
var sigNetGenesisMerkleRoot = genesisMerkleRoot

// sigNetGenesisBlock defines the genesis block of the block chain which serves
// as the public transaction ledger for the signet test network.
var sigNetGenesisBlock = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version:    1,
		PrevBlock:  chainhash.Hash{},         // 0000000000000000000000000000000000000000000000000000000000000000
		MerkleRoot: sigNetGenesisMerkleRoot,  // 4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b
		Timestamp:  time.Unix(1598918400, 0), // 2020-09-01 00:00:00 +0000 UTC
		Bits:       0x1e0377ae,               // 503543726 [00000377ae000000000000000000000000000000000000000000000000000000]
		Nonce:      52613770,
	},
	Transactions: []*wire.MsgTx{&genesisCoinbaseTx},
}

// shellGenesisCoinbaseTx is the coinbase transaction for the Shell Reserve genesis block.
// It includes the constitution hash and launch message with no premine.
var shellGenesisCoinbaseTx = wire.MsgTx{
	Version: 1,
	TxIn: []*wire.TxIn{
		{
			PreviousOutPoint: wire.OutPoint{
				Hash:  chainhash.Hash{},
				Index: 0xffffffff,
			},
			SignatureScript: []byte{
				// Message: "Shell Reserve 2026-01-01: Digital Gold for Central Banks | Constitution: [hash]"
				0x04, 0xff, 0xff, 0x00, 0x1d, 0x01, 0x04, 0x5a, /* |.......Z| */
				0x53, 0x68, 0x65, 0x6c, 0x6c, 0x20, 0x52, 0x65, /* |Shell Re| */
				0x73, 0x65, 0x72, 0x76, 0x65, 0x20, 0x32, 0x30, /* |serve 20| */
				0x32, 0x36, 0x2d, 0x30, 0x31, 0x2d, 0x30, 0x31, /* |26-01-01| */
				0x3a, 0x20, 0x44, 0x69, 0x67, 0x69, 0x74, 0x61, /* |: Digita| */
				0x6c, 0x20, 0x47, 0x6f, 0x6c, 0x64, 0x20, 0x66, /* |l Gold f| */
				0x6f, 0x72, 0x20, 0x43, 0x65, 0x6e, 0x74, 0x72, /* |or Centr| */
				0x61, 0x6c, 0x20, 0x42, 0x61, 0x6e, 0x6b, 0x73, /* |al Banks| */
				0x20, 0x7c, 0x20, 0x46, 0x61, 0x69, 0x72, 0x20, /* | | Fair | */
				0x4c, 0x61, 0x75, 0x6e, 0x63, 0x68, 0x2c, 0x20, /* |Launch, | */
				0x4e, 0x6f, 0x20, 0x50, 0x72, 0x65, 0x6d, 0x69, /* |No Premi| */
				0x6e, 0x65, /* |ne| */
			},
			Sequence: 0xffffffff,
		},
	},
	TxOut: []*wire.TxOut{
		{
			// Zero value output - no premine, unspendable
			Value: 0,
			PkScript: []byte{
				0x6a,                                           // OP_RETURN - makes output unspendable
				0x24,                                           // 36 bytes of data
				0x53, 0x68, 0x65, 0x6c, 0x6c, 0x20, 0x52, 0x65, /* |Shell Re| */
				0x73, 0x65, 0x72, 0x76, 0x65, 0x3a, 0x20, 0x44, /* |serve: D| */
				0x69, 0x67, 0x69, 0x74, 0x61, 0x6c, 0x20, 0x47, /* |igital G| */
				0x6f, 0x6c, 0x64, 0x20, 0x66, 0x6f, 0x72, 0x20, /* |old for | */
				0x32, 0x31, 0x73, 0x74, 0x20, 0x43, 0x65, 0x6e, /* |21st Cen| */
				0x74, 0x75, 0x72, 0x79, /* |tury| */
			},
		},
	},
	LockTime: 0,
}

// shellGenesisHash is the hash of the first block in the Shell Reserve chain.
// This will be computed after the genesis block is finalized.
var shellGenesisHash = chainhash.Hash{
	// Placeholder - will be computed from actual genesis block
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

// shellGenesisMerkleRoot is the hash of the coinbase transaction in the Shell genesis block.
var shellGenesisMerkleRoot = chainhash.Hash{
	// Placeholder - will be computed from actual coinbase transaction
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

// shellGenesisBlock defines the genesis block for the Shell Reserve network.
// Launch date: January 1, 2026, 00:00 UTC
var shellGenesisBlock = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version:    1,
		PrevBlock:  chainhash.Hash{},                            // 0000000000000000000000000000000000000000000000000000000000000000
		MerkleRoot: shellGenesisMerkleRoot,                      // Will be computed from coinbase tx
		Timestamp:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), // Shell Reserve launch
		Bits:       0x1d00ffff,                                  // Initial difficulty same as Bitcoin
		Nonce:      0,                                           // Will be mined with RandomX
	},
	Transactions: []*wire.MsgTx{&shellGenesisCoinbaseTx},
}

// createShellGenesisBlock creates the Shell Reserve genesis block
func createShellGenesisBlock() *wire.MsgBlock {
	// Constitution hash for verifiable commitment to principles
	constitutionHash := sha256.Sum256([]byte(shellConstitutionText))

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
			Value:    0,            // NO PREMINE - zero value output
			PkScript: []byte{0x6a}, // Unspendable OP_RETURN (0x6a)
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
