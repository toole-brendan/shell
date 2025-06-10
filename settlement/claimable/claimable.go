// Package claimable implements Shell Reserve's claimable balance functionality
// for conditional payments inspired by Stellar's design.
package claimable

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/toole-brendan/shell/wire"
)

// ClaimableID is a unique identifier for a claimable balance
type ClaimableID [32]byte

// PredicateType defines the type of condition for claiming
type PredicateType uint8

const (
	PredicateUnconditional PredicateType = iota
	PredicateBeforeTime
	PredicateAfterTime
	PredicateHashPreimage
	PredicateAnd
	PredicateOr
	PredicateNot
)

// ClaimPredicate defines conditions that must be met to claim a balance
type ClaimPredicate struct {
	Type      PredicateType
	Timestamp uint32           // Unix timestamp for time-based predicates
	Hash      [32]byte         // Hash for preimage predicates
	Children  []ClaimPredicate // For composite predicates (AND, OR, NOT)
}

// Claimant represents someone who can claim a balance
type Claimant struct {
	Destination *btcec.PublicKey
	Predicate   ClaimPredicate
}

// ClaimableBalance represents a balance that can be claimed by satisfying conditions
type ClaimableBalance struct {
	ID         ClaimableID
	Amount     uint64 // Amount in satoshis (not confidential for simplicity)
	Claimants  []Claimant
	CreateTime uint32 // Block height when created
	Creator    *btcec.PublicKey
}

// ClaimProof contains the data needed to satisfy claim predicates
type ClaimProof struct {
	Preimages map[[32]byte][]byte // Hash preimages
	Timestamp uint32              // Current timestamp for time checks
}

// ClaimableState tracks the global state of all claimable balances
type ClaimableState struct {
	balances map[ClaimableID]*ClaimableBalance
	utxos    map[wire.OutPoint]*ClaimableBalance
}

// NewClaimableState creates a new claimable balance state tracker
func NewClaimableState() *ClaimableState {
	return &ClaimableState{
		balances: make(map[ClaimableID]*ClaimableBalance),
		utxos:    make(map[wire.OutPoint]*ClaimableBalance),
	}
}

// GenerateClaimableID creates a unique ID for a claimable balance
func GenerateClaimableID(creator *btcec.PublicKey, amount uint64, nonce uint64) ClaimableID {
	data := make([]byte, 0, 100)

	// Add creator public key
	data = append(data, creator.SerializeCompressed()...)

	// Add amount
	amountBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(amountBytes, amount)
	data = append(data, amountBytes...)

	// Add nonce for uniqueness
	nonceBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(nonceBytes, nonce)
	data = append(data, nonceBytes...)

	hash := sha256.Sum256(data)
	var claimableID ClaimableID
	copy(claimableID[:], hash[:])

	return claimableID
}

// CreateClaimableBalance creates a new claimable balance
func (cs *ClaimableState) CreateClaimableBalance(creator *btcec.PublicKey, amount uint64, claimants []Claimant, createHeight uint32, fundingOutpoint wire.OutPoint) (*ClaimableBalance, error) {
	if amount == 0 {
		return nil, errors.New("claimable amount must be greater than zero")
	}

	if len(claimants) == 0 {
		return nil, errors.New("must have at least one claimant")
	}

	// Validate claimants
	for i, claimant := range claimants {
		if claimant.Destination == nil {
			return nil, fmt.Errorf("claimant %d has nil destination", i)
		}
		if err := validatePredicate(claimant.Predicate); err != nil {
			return nil, fmt.Errorf("invalid predicate for claimant %d: %v", i, err)
		}
	}

	// Generate unique ID
	nonce := uint64(time.Now().UnixNano())
	claimableID := GenerateClaimableID(creator, amount, nonce)

	// Check if ID already exists (extremely unlikely but check anyway)
	if _, exists := cs.balances[claimableID]; exists {
		return nil, fmt.Errorf("claimable balance %x already exists", claimableID)
	}

	// Create claimable balance
	balance := &ClaimableBalance{
		ID:         claimableID,
		Amount:     amount,
		Claimants:  claimants,
		CreateTime: createHeight,
		Creator:    creator,
	}

	// Store balance
	cs.balances[claimableID] = balance
	cs.utxos[fundingOutpoint] = balance

	return balance, nil
}

// ClaimBalance attempts to claim a balance by satisfying predicates
func (cs *ClaimableState) ClaimBalance(balanceID ClaimableID, claimer *btcec.PublicKey, proof ClaimProof, currentHeight uint32) (*ClaimableBalance, error) {
	balance, exists := cs.balances[balanceID]
	if !exists {
		return nil, fmt.Errorf("claimable balance %x not found", balanceID)
	}

	// Find valid claimant
	claimantIndex := -1
	for i, claimant := range balance.Claimants {
		if claimant.Destination.IsEqual(claimer) {
			if evaluatePredicate(claimant.Predicate, proof, currentHeight) {
				claimantIndex = i
				break
			}
		}
	}

	if claimantIndex == -1 {
		return nil, errors.New("no valid claim found for this public key")
	}

	// Remove from state
	delete(cs.balances, balanceID)

	// Clean up UTXO mapping
	for outpoint, bal := range cs.utxos {
		if bal.ID == balanceID {
			delete(cs.utxos, outpoint)
			break
		}
	}

	return balance, nil
}

// GetClaimableBalance retrieves a claimable balance by ID
func (cs *ClaimableState) GetClaimableBalance(balanceID ClaimableID) (*ClaimableBalance, error) {
	balance, exists := cs.balances[balanceID]
	if !exists {
		return nil, fmt.Errorf("claimable balance %x not found", balanceID)
	}

	return balance, nil
}

// validatePredicate ensures a predicate is well-formed
func validatePredicate(pred ClaimPredicate) error {
	switch pred.Type {
	case PredicateUnconditional:
		return nil

	case PredicateBeforeTime, PredicateAfterTime:
		if pred.Timestamp == 0 {
			return errors.New("time predicate requires non-zero timestamp")
		}
		return nil

	case PredicateHashPreimage:
		if pred.Hash == [32]byte{} {
			return errors.New("hash predicate requires non-zero hash")
		}
		return nil

	case PredicateAnd, PredicateOr:
		if len(pred.Children) < 2 {
			return fmt.Errorf("%v predicate requires at least 2 children", pred.Type)
		}
		for i, child := range pred.Children {
			if err := validatePredicate(child); err != nil {
				return fmt.Errorf("child %d: %v", i, err)
			}
		}
		return nil

	case PredicateNot:
		if len(pred.Children) != 1 {
			return errors.New("NOT predicate requires exactly 1 child")
		}
		return validatePredicate(pred.Children[0])

	default:
		return fmt.Errorf("unknown predicate type: %v", pred.Type)
	}
}

// evaluatePredicate checks if a predicate is satisfied
func evaluatePredicate(pred ClaimPredicate, proof ClaimProof, currentHeight uint32) bool {
	switch pred.Type {
	case PredicateUnconditional:
		return true

	case PredicateBeforeTime:
		// Convert block height to approximate timestamp (5-minute blocks)
		currentTime := currentHeight * 300 // 5 minutes = 300 seconds
		return currentTime < pred.Timestamp

	case PredicateAfterTime:
		currentTime := currentHeight * 300
		return currentTime >= pred.Timestamp

	case PredicateHashPreimage:
		preimage, exists := proof.Preimages[pred.Hash]
		if !exists {
			return false
		}
		hash := sha256.Sum256(preimage)
		return hash == pred.Hash

	case PredicateAnd:
		for _, child := range pred.Children {
			if !evaluatePredicate(child, proof, currentHeight) {
				return false
			}
		}
		return true

	case PredicateOr:
		for _, child := range pred.Children {
			if evaluatePredicate(child, proof, currentHeight) {
				return true
			}
		}
		return false

	case PredicateNot:
		if len(pred.Children) != 1 {
			return false
		}
		return !evaluatePredicate(pred.Children[0], proof, currentHeight)

	default:
		return false
	}
}

// ClaimableOpType represents the type of claimable balance operation
type ClaimableOpType uint8

const (
	ClaimableOpCreate ClaimableOpType = iota
	ClaimableOpClaim
)

// ValidateClaimableOperation validates claimable balance operations for consensus
func ValidateClaimableOperation(op ClaimableOpType, state *ClaimableState, params []interface{}) error {
	switch op {
	case ClaimableOpCreate:
		if len(params) < 4 {
			return errors.New("insufficient parameters for claimable create")
		}

		creator, ok := params[0].(*btcec.PublicKey)
		if !ok || creator == nil {
			return errors.New("invalid creator public key")
		}

		amount, ok := params[1].(uint64)
		if !ok {
			return errors.New("invalid amount")
		}

		if amount == 0 {
			return errors.New("claimable amount must be positive")
		}

		claimants, ok := params[2].([]Claimant)
		if !ok {
			return errors.New("invalid claimants")
		}

		if len(claimants) == 0 {
			return errors.New("must have at least one claimant")
		}

		// Validate each claimant
		for i, claimant := range claimants {
			if claimant.Destination == nil {
				return fmt.Errorf("claimant %d has nil destination", i)
			}
			if err := validatePredicate(claimant.Predicate); err != nil {
				return fmt.Errorf("claimant %d has invalid predicate: %v", i, err)
			}
		}

		return nil

	case ClaimableOpClaim:
		if len(params) < 3 {
			return errors.New("insufficient parameters for claimable claim")
		}

		balanceID, ok := params[0].(ClaimableID)
		if !ok {
			return errors.New("invalid balance ID")
		}

		claimer, ok := params[1].(*btcec.PublicKey)
		if !ok || claimer == nil {
			return errors.New("invalid claimer public key")
		}

		_, ok = params[2].(ClaimProof)
		if !ok {
			return errors.New("invalid claim proof")
		}

		// Check if balance exists
		balance, err := state.GetClaimableBalance(balanceID)
		if err != nil {
			return err
		}

		// Verify claimer is in claimant list
		found := false
		for _, claimant := range balance.Claimants {
			if claimant.Destination.IsEqual(claimer) {
				found = true
				break
			}
		}

		if !found {
			return errors.New("claimer not in claimant list")
		}

		return nil

	default:
		return fmt.Errorf("unknown claimable operation: %v", op)
	}
}

// Common predicate constructors for convenience

// UnconditionalPredicate creates a predicate that always evaluates to true
func UnconditionalPredicate() ClaimPredicate {
	return ClaimPredicate{Type: PredicateUnconditional}
}

// BeforeTimePredicate creates a predicate that's valid before a timestamp
func BeforeTimePredicate(timestamp uint32) ClaimPredicate {
	return ClaimPredicate{
		Type:      PredicateBeforeTime,
		Timestamp: timestamp,
	}
}

// AfterTimePredicate creates a predicate that's valid after a timestamp
func AfterTimePredicate(timestamp uint32) ClaimPredicate {
	return ClaimPredicate{
		Type:      PredicateAfterTime,
		Timestamp: timestamp,
	}
}

// HashPreimagePredicate creates a predicate requiring a hash preimage
func HashPreimagePredicate(hash [32]byte) ClaimPredicate {
	return ClaimPredicate{
		Type: PredicateHashPreimage,
		Hash: hash,
	}
}

// AndPredicate creates a predicate requiring all children to be true
func AndPredicate(children ...ClaimPredicate) ClaimPredicate {
	return ClaimPredicate{
		Type:     PredicateAnd,
		Children: children,
	}
}

// OrPredicate creates a predicate requiring at least one child to be true
func OrPredicate(children ...ClaimPredicate) ClaimPredicate {
	return ClaimPredicate{
		Type:     PredicateOr,
		Children: children,
	}
}
