// Copyright (c) 2025 Shell Reserve developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package randomx

import (
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

// RandomXParams contains the parameters for RandomX mining.
type RandomXParams struct {
	SeedRotation int32 // Blocks between seed rotations
	Memory       int64 // Memory requirement in bytes
}

// BlockTemplate is a template that valid blocks are based on.
type BlockTemplate struct {
	// Block is a block that is ready to be solved by miners.  Thus, it is
	// completely valid with the exception of satisfying the proof-of-work
	// requirement.
	Block *wire.MsgBlock

	// Fees contains the amount of fees each transaction in the generated
	// template pays in base units.  Since the first transaction is the
	// coinbase, the first entry (offset 0) will contain the negative of the
	// sum of the fees of all other transactions.
	Fees []int64

	// SigOpCounts contains the number of signature operations each
	// transaction in the generated template performs.
	SigOpCounts []int64

	// Height is the height at which the block template connects to the main
	// chain.
	Height int32

	// ValidPayAddress indicates whether or not the template coinbase pays
	// to an address or is redeemable by anyone.  See the documentation on
	// NewBlockTemplate for details on which this can be useful to generate
	// templates without a coinbase payment address.
	ValidPayAddress bool

	// WitnessCommitment is a commitment to the witness data (if any)
	// within the block.  This field will only be populated once the block
	// template is used to generate a block which is intended to be submitted
	// to the network for inclusion.  This field is only valid if the
	// template contains at least one transaction which has witness data.
	WitnessCommitment []byte
}

// Config is a descriptor which specifies the mining instance configuration.
type Config struct {
	// RandomXParams contains the parameters for RandomX operation.
	RandomXParams RandomXParams

	// GenesisHash is the hash of the genesis block for the network.
	GenesisHash *chainhash.Hash

	// RandomXSeedRotation specifies the number of blocks between RandomX seed rotations.
	RandomXSeedRotation int32

	// NumWorkers specifies the number of workers to create to solve blocks.
	NumWorkers uint32

	// UpdateNumWorkers is a channel that is listened to for updates to the
	// number of workers.
	UpdateNumWorkers chan struct{}

	// The following functions are required:

	// ConnectedCount should return the number of currently connected peers
	// for the passed chain and the current best known height.
	ConnectedCount func() (int32, int64, error)

	// IsCurrent should return whether or not the passed chain believes it
	// is current.  That is to say, whether or not it believes it has caught
	// up to the best known chain tip.
	IsCurrent func() bool

	// BlockTemplateGenerator should return a new block template that is
	// ready to be solved.
	BlockTemplateGenerator func() (*BlockTemplate, error)

	// BestSnapshot should return the current best known chain tip snapshot.
	BestSnapshot func() (int32, *chainhash.Hash, error)

	// SubmitBlock should submit the passed block to the network after
	// ensuring it passes all consensus validation rules.
	SubmitBlock func(*btcutil.Block) error
}
