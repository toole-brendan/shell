// Package channels implements Shell Reserve's payment channel functionality
// for institutional instant settlement between parties.
package channels

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/toole-brendan/shell/chaincfg/chainhash"
	"github.com/toole-brendan/shell/wire"
)

// ChannelID is a unique identifier for a payment channel
type ChannelID [32]byte

// PaymentChannel represents a unidirectional payment channel between two parties
type PaymentChannel struct {
	ChannelID    ChannelID
	Participants [2]*btcec.PublicKey // [0] is sender, [1] is receiver
	Capacity     uint64              // Total locked amount in satoshis
	Balance      [2]uint64           // Current balance for each party
	Nonce        uint64              // Monotonically increasing counter
	Expiry       uint32              // Block height when channel expires
	IsOpen       bool                // Channel state
}

// ChannelUpdate represents a state update for a payment channel
type ChannelUpdate struct {
	ChannelID  ChannelID
	Balances   [2]uint64
	Nonce      uint64
	Signatures [2]*ecdsa.Signature
}

// ChannelState tracks the global state of all channels
type ChannelState struct {
	channels map[ChannelID]*PaymentChannel
	utxos    map[wire.OutPoint]*PaymentChannel
}

// NewChannelState creates a new channel state tracker
func NewChannelState() *ChannelState {
	return &ChannelState{
		channels: make(map[ChannelID]*PaymentChannel),
		utxos:    make(map[wire.OutPoint]*PaymentChannel),
	}
}

// GenerateChannelID creates a unique channel ID from participants and funding transaction
func GenerateChannelID(alice, bob *btcec.PublicKey, fundingTx *chainhash.Hash, outputIdx uint32) ChannelID {
	data := make([]byte, 0, 100)

	// Add participant public keys
	data = append(data, alice.SerializeCompressed()...)
	data = append(data, bob.SerializeCompressed()...)

	// Add funding transaction info
	data = append(data, fundingTx[:]...)
	idxBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(idxBytes, outputIdx)
	data = append(data, idxBytes...)

	hash := sha256.Sum256(data)
	var channelID ChannelID
	copy(channelID[:], hash[:])

	return channelID
}

// OpenChannel creates a new payment channel
func (cs *ChannelState) OpenChannel(alice, bob *btcec.PublicKey, capacity uint64, expiry uint32, fundingOutpoint wire.OutPoint) (*PaymentChannel, error) {
	if capacity == 0 {
		return nil, errors.New("channel capacity must be greater than zero")
	}

	if alice == nil || bob == nil {
		return nil, errors.New("both participants must have valid public keys")
	}

	// Generate channel ID
	channelID := GenerateChannelID(alice, bob, &fundingOutpoint.Hash, fundingOutpoint.Index)

	// Check if channel already exists
	if _, exists := cs.channels[channelID]; exists {
		return nil, fmt.Errorf("channel %x already exists", channelID)
	}

	// Create new channel
	channel := &PaymentChannel{
		ChannelID:    channelID,
		Participants: [2]*btcec.PublicKey{alice, bob},
		Capacity:     capacity,
		Balance:      [2]uint64{capacity, 0}, // Initially all balance goes to sender
		Nonce:        0,
		Expiry:       expiry,
		IsOpen:       true,
	}

	// Store channel
	cs.channels[channelID] = channel
	cs.utxos[fundingOutpoint] = channel

	return channel, nil
}

// UpdateChannel processes a channel state update
func (cs *ChannelState) UpdateChannel(update *ChannelUpdate) error {
	channel, exists := cs.channels[update.ChannelID]
	if !exists {
		return fmt.Errorf("channel %x not found", update.ChannelID)
	}

	if !channel.IsOpen {
		return fmt.Errorf("channel %x is closed", update.ChannelID)
	}

	// Verify nonce is strictly increasing
	if update.Nonce <= channel.Nonce {
		return fmt.Errorf("invalid nonce: got %d, expected > %d", update.Nonce, channel.Nonce)
	}

	// Verify balance conservation
	totalBalance := update.Balances[0] + update.Balances[1]
	if totalBalance != channel.Capacity {
		return fmt.Errorf("balance mismatch: %d + %d != %d",
			update.Balances[0], update.Balances[1], channel.Capacity)
	}

	// Verify signatures (simplified - would need actual message signing in production)
	if update.Signatures[0] == nil || update.Signatures[1] == nil {
		return errors.New("both participants must sign the update")
	}

	// Apply update
	channel.Balance = update.Balances
	channel.Nonce = update.Nonce

	return nil
}

// CloseChannel finalizes a payment channel
func (cs *ChannelState) CloseChannel(channelID ChannelID) (*PaymentChannel, error) {
	channel, exists := cs.channels[channelID]
	if !exists {
		return nil, fmt.Errorf("channel %x not found", channelID)
	}

	if !channel.IsOpen {
		return nil, fmt.Errorf("channel %x already closed", channelID)
	}

	// Mark as closed
	channel.IsOpen = false

	// Clean up UTXO mapping
	for outpoint, ch := range cs.utxos {
		if ch.ChannelID == channelID {
			delete(cs.utxos, outpoint)
			break
		}
	}

	return channel, nil
}

// GetChannel retrieves a channel by ID
func (cs *ChannelState) GetChannel(channelID ChannelID) (*PaymentChannel, error) {
	channel, exists := cs.channels[channelID]
	if !exists {
		return nil, fmt.Errorf("channel %x not found", channelID)
	}

	return channel, nil
}

// ValidateChannelOperation validates channel operations for consensus
func ValidateChannelOperation(op ChannelOpType, channelID ChannelID, state *ChannelState, params []interface{}) error {
	switch op {
	case ChannelOpOpen:
		// Validate open parameters
		if len(params) < 4 {
			return errors.New("insufficient parameters for channel open")
		}

		alice, ok := params[0].(*btcec.PublicKey)
		if !ok {
			return errors.New("invalid alice public key")
		}

		bob, ok := params[1].(*btcec.PublicKey)
		if !ok {
			return errors.New("invalid bob public key")
		}

		capacity, ok := params[2].(uint64)
		if !ok {
			return errors.New("invalid capacity")
		}

		if capacity == 0 {
			return errors.New("channel capacity must be positive")
		}

		// Check if channel already exists
		genID := GenerateChannelID(alice, bob, &chainhash.Hash{}, 0)
		if _, exists := state.channels[genID]; exists {
			return errors.New("channel already exists")
		}

		return nil

	case ChannelOpUpdate:
		// Validate update parameters
		if len(params) < 3 {
			return errors.New("insufficient parameters for channel update")
		}

		channel, err := state.GetChannel(channelID)
		if err != nil {
			return err
		}

		if !channel.IsOpen {
			return errors.New("cannot update closed channel")
		}

		balances, ok := params[0].([2]uint64)
		if !ok {
			return errors.New("invalid balances")
		}

		nonce, ok := params[1].(uint64)
		if !ok {
			return errors.New("invalid nonce")
		}

		// Verify nonce increment
		if nonce <= channel.Nonce {
			return fmt.Errorf("nonce must increase: got %d, current %d", nonce, channel.Nonce)
		}

		// Verify balance conservation
		if balances[0]+balances[1] != channel.Capacity {
			return errors.New("balance conservation violated")
		}

		return nil

	case ChannelOpClose:
		// Validate close parameters
		channel, err := state.GetChannel(channelID)
		if err != nil {
			return err
		}

		if !channel.IsOpen {
			return errors.New("channel already closed")
		}

		return nil

	default:
		return fmt.Errorf("unknown channel operation: %v", op)
	}
}

// ChannelOpType represents the type of channel operation
type ChannelOpType uint8

const (
	ChannelOpOpen ChannelOpType = iota
	ChannelOpUpdate
	ChannelOpClose
)

// CreateChannelOpenScript creates a script for opening a payment channel
func CreateChannelOpenScript(alice, bob *btcec.PublicKey, amount uint64) []byte {
	// This would integrate with txscript package in production
	// For now, return a placeholder
	return []byte{0xc6} // OP_CHANNEL_OPEN
}

// CreateChannelUpdateScript creates a script for updating channel state
func CreateChannelUpdateScript(channelID ChannelID, balances [2]uint64, nonce uint64) []byte {
	// This would integrate with txscript package in production
	// For now, return a placeholder
	return []byte{0xc7} // OP_CHANNEL_UPDATE
}

// CreateChannelCloseScript creates a script for closing a channel
func CreateChannelCloseScript(channelID ChannelID) []byte {
	// This would integrate with txscript package in production
	// For now, return a placeholder
	return []byte{0xc8} // OP_CHANNEL_CLOSE
}
