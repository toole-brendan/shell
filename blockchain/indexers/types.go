package indexers

import (
	"github.com/toole-brendan/shell/blockchain"
	"github.com/toole-brendan/shell/chaincfg"
	"github.com/toole-brendan/shell/database"
	"github.com/btcsuite/btcd/btcutil"
)

// AddrIndex represents an address index
type AddrIndex struct {
	db          database.DB
	chainParams *chaincfg.Params
}

// NewAddrIndex creates a new address index
func NewAddrIndex(db database.DB, chainParams *chaincfg.Params) *AddrIndex {
	return &AddrIndex{
		db:          db,
		chainParams: chainParams,
	}
}
