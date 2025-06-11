// Package swaps provides atomic swap functionality for Shell Reserve
// enabling cross-chain exchanges with Bitcoin and Ethereum
package swaps

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/wire"
	"github.com/toole-brendan/shell/txscript"
)

// AtomicSwap represents a cross-chain atomic swap using Hash Time Locked Contracts (HTLCs)
type AtomicSwap struct {
	// Swap identification
	SwapID     [32]byte `json:"swapId"`
	SecretHash [32]byte `json:"secretHash"`

	// Parties
	Initiator   *btcec.PublicKey `json:"initiator"`
	Participant *btcec.PublicKey `json:"participant"`

	// Swap parameters
	Amount  uint64 `json:"amount"`
	Timeout uint32 `json:"timeout"`

	// Cross-chain details
	Chain ChainType `json:"chain"`

	// Contract details
	ContractTx *wire.MsgTx `json:"contractTx,omitempty"`
	RedeemTx   *wire.MsgTx `json:"redeemTx,omitempty"`
	RefundTx   *wire.MsgTx `json:"refundTx,omitempty"`

	// State
	Status    SwapStatus `json:"status"`
	CreatedAt time.Time  `json:"createdAt"`
	ExpiresAt time.Time  `json:"expiresAt"`
}

// ChainType represents supported blockchain types for atomic swaps
type ChainType string

const (
	ChainShell    ChainType = "SHELL"
	ChainBitcoin  ChainType = "BTC"
	ChainEthereum ChainType = "ETH"
)

// SwapStatus represents the current state of an atomic swap
type SwapStatus string

const (
	SwapStatusPending  SwapStatus = "PENDING"
	SwapStatusActive   SwapStatus = "ACTIVE"
	SwapStatusRedeemed SwapStatus = "REDEEMED"
	SwapStatusRefunded SwapStatus = "REFUNDED"
	SwapStatusExpired  SwapStatus = "EXPIRED"
)

// AtomicSwapParams contains parameters for creating an atomic swap
type AtomicSwapParams struct {
	Initiator   *btcec.PublicKey
	Participant *btcec.PublicKey
	Amount      uint64
	Timeout     uint32
	Chain       ChainType
	Secret      []byte // Only known by initiator initially
}

// NewAtomicSwap creates a new atomic swap with the given parameters
func NewAtomicSwap(params *AtomicSwapParams) (*AtomicSwap, error) {
	if params == nil {
		return nil, fmt.Errorf("atomic swap parameters cannot be nil")
	}

	if params.Initiator == nil || params.Participant == nil {
		return nil, fmt.Errorf("initiator and participant keys required")
	}

	if params.Amount == 0 {
		return nil, fmt.Errorf("swap amount must be greater than zero")
	}

	if params.Timeout == 0 {
		return nil, fmt.Errorf("timeout must be specified")
	}

	if len(params.Secret) == 0 {
		return nil, fmt.Errorf("secret required for atomic swap")
	}

	// Generate secret hash
	secretHash := sha256.Sum256(params.Secret)

	// Generate swap ID
	swapID := generateSwapID(params.Initiator, params.Participant, secretHash)

	now := time.Now()
	expiresAt := now.Add(time.Duration(params.Timeout) * time.Second)

	swap := &AtomicSwap{
		SwapID:      swapID,
		SecretHash:  secretHash,
		Initiator:   params.Initiator,
		Participant: params.Participant,
		Amount:      params.Amount,
		Timeout:     params.Timeout,
		Chain:       params.Chain,
		Status:      SwapStatusPending,
		CreatedAt:   now,
		ExpiresAt:   expiresAt,
	}

	return swap, nil
}

// CreateHTLCScript creates the Hash Time Locked Contract script for Shell
func (swap *AtomicSwap) CreateHTLCScript() ([]byte, error) {
	builder := txscript.NewScriptBuilder()

	// IF branch: Participant can redeem with secret
	builder.AddOp(txscript.OP_IF)

	// Check hash preimage
	builder.AddOp(txscript.OP_HASH256)
	builder.AddData(swap.SecretHash[:])
	builder.AddOp(txscript.OP_EQUALVERIFY)

	// Check participant signature
	builder.AddData(swap.Participant.SerializeCompressed())
	builder.AddOp(txscript.OP_CHECKSIG)

	// ELSE branch: Initiator can refund after timeout
	builder.AddOp(txscript.OP_ELSE)

	// Check timeout
	builder.AddInt64(int64(swap.Timeout))
	builder.AddOp(txscript.OP_CHECKLOCKTIMEVERIFY)
	builder.AddOp(txscript.OP_DROP)

	// Check initiator signature
	builder.AddData(swap.Initiator.SerializeCompressed())
	builder.AddOp(txscript.OP_CHECKSIG)

	builder.AddOp(txscript.OP_ENDIF)

	return builder.Script()
}

// CreateContractTransaction creates the contract transaction that locks funds
func (swap *AtomicSwap) CreateContractTransaction(fundingTx *wire.MsgTx, fundingVout uint32) (*wire.MsgTx, error) {
	if fundingTx == nil {
		return nil, fmt.Errorf("funding transaction required")
	}

	// Create HTLC script
	htlcScript, err := swap.CreateHTLCScript()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTLC script: %v", err)
	}

	// Create contract transaction
	contractTx := wire.NewMsgTx(wire.TxVersion)

	// Add input from funding transaction
	fundingHash := fundingTx.TxHash()
	prevOut := wire.NewOutPoint(&fundingHash, fundingVout)
	txIn := wire.NewTxIn(prevOut, nil, nil)
	contractTx.AddTxIn(txIn)

	// Add output with HTLC script
	txOut := wire.NewTxOut(int64(swap.Amount), htlcScript)
	contractTx.AddTxOut(txOut)

	// Set locktime to current time for timeout reference
	contractTx.LockTime = uint32(time.Now().Unix())

	swap.ContractTx = contractTx
	swap.Status = SwapStatusActive

	return contractTx, nil
}

// CreateRedeemTransaction creates the transaction that redeems the swap with secret
func (swap *AtomicSwap) CreateRedeemTransaction(secret []byte, participantAddr []byte) (*wire.MsgTx, error) {
	if swap.ContractTx == nil {
		return nil, fmt.Errorf("contract transaction required")
	}

	// Verify secret
	if sha256.Sum256(secret) != swap.SecretHash {
		return nil, fmt.Errorf("invalid secret provided")
	}

	// Create redeem transaction
	redeemTx := wire.NewMsgTx(wire.TxVersion)

	// Add input from contract transaction
	contractHash := swap.ContractTx.TxHash()
	prevOut := wire.NewOutPoint(&contractHash, 0)
	txIn := wire.NewTxIn(prevOut, nil, nil)
	redeemTx.AddTxIn(txIn)

	// Add output to participant
	txOut := wire.NewTxOut(int64(swap.Amount), participantAddr)
	redeemTx.AddTxOut(txOut)

	// Create witness stack for IF branch
	witness := wire.TxWitness{
		secret,                                 // Secret preimage
		swap.Participant.SerializeCompressed(), // Participant pubkey
		{0x01},                                 // TRUE for IF branch
	}
	redeemTx.TxIn[0].Witness = witness

	swap.RedeemTx = redeemTx
	swap.Status = SwapStatusRedeemed

	return redeemTx, nil
}

// CreateRefundTransaction creates the transaction that refunds the swap after timeout
func (swap *AtomicSwap) CreateRefundTransaction(initiatorAddr []byte) (*wire.MsgTx, error) {
	if swap.ContractTx == nil {
		return nil, fmt.Errorf("contract transaction required")
	}

	// Check if timeout has passed
	if time.Now().Before(swap.ExpiresAt) {
		return nil, fmt.Errorf("swap has not expired yet")
	}

	// Create refund transaction
	refundTx := wire.NewMsgTx(wire.TxVersion)

	// Add input from contract transaction
	contractHash := swap.ContractTx.TxHash()
	prevOut := wire.NewOutPoint(&contractHash, 0)
	txIn := wire.NewTxIn(prevOut, nil, nil)
	refundTx.AddTxIn(txIn)

	// Add output to initiator
	txOut := wire.NewTxOut(int64(swap.Amount), initiatorAddr)
	refundTx.AddTxOut(txOut)

	// Set locktime to enable timeout
	refundTx.LockTime = swap.Timeout

	// Create witness stack for ELSE branch
	witness := wire.TxWitness{
		swap.Initiator.SerializeCompressed(), // Initiator pubkey
		{},                                   // FALSE for ELSE branch
	}
	refundTx.TxIn[0].Witness = witness

	swap.RefundTx = refundTx
	swap.Status = SwapStatusRefunded

	return refundTx, nil
}

// ValidateSwap validates the atomic swap parameters and state
func ValidateSwap(swap *AtomicSwap) error {
	if swap == nil {
		return fmt.Errorf("swap cannot be nil")
	}

	if swap.Initiator == nil || swap.Participant == nil {
		return fmt.Errorf("initiator and participant required")
	}

	if swap.Amount == 0 {
		return fmt.Errorf("amount must be greater than zero")
	}

	if swap.Timeout == 0 {
		return fmt.Errorf("timeout must be specified")
	}

	// Check for expired swaps
	if time.Now().After(swap.ExpiresAt) && swap.Status == SwapStatusActive {
		swap.Status = SwapStatusExpired
	}

	return nil
}

// ExtractSecretFromRedeemTx extracts the secret from a redeem transaction
func ExtractSecretFromRedeemTx(tx *wire.MsgTx) ([]byte, error) {
	if tx == nil {
		return nil, fmt.Errorf("transaction cannot be nil")
	}

	if len(tx.TxIn) == 0 {
		return nil, fmt.Errorf("transaction has no inputs")
	}

	witness := tx.TxIn[0].Witness
	if len(witness) < 3 {
		return nil, fmt.Errorf("insufficient witness items")
	}

	// In HTLC witness: [secret, pubkey, true]
	secret := witness[0]
	if len(secret) == 0 {
		return nil, fmt.Errorf("no secret found in witness")
	}

	return secret, nil
}

// generateSwapID generates a unique swap ID
func generateSwapID(initiator, participant *btcec.PublicKey, secretHash [32]byte) [32]byte {
	data := fmt.Sprintf("%s:%s:%x",
		hex.EncodeToString(initiator.SerializeCompressed()),
		hex.EncodeToString(participant.SerializeCompressed()),
		secretHash)
	return sha256.Sum256([]byte(data))
}

// SwapManager manages multiple atomic swaps
type SwapManager struct {
	swaps map[[32]byte]*AtomicSwap
}

// NewSwapManager creates a new swap manager
func NewSwapManager() *SwapManager {
	return &SwapManager{
		swaps: make(map[[32]byte]*AtomicSwap),
	}
}

// AddSwap adds a swap to the manager
func (sm *SwapManager) AddSwap(swap *AtomicSwap) error {
	if err := ValidateSwap(swap); err != nil {
		return err
	}

	sm.swaps[swap.SwapID] = swap
	return nil
}

// GetSwap retrieves a swap by ID
func (sm *SwapManager) GetSwap(swapID [32]byte) (*AtomicSwap, error) {
	swap, exists := sm.swaps[swapID]
	if !exists {
		return nil, fmt.Errorf("swap not found")
	}

	return swap, nil
}

// ListActiveSwaps returns all active swaps
func (sm *SwapManager) ListActiveSwaps() []*AtomicSwap {
	var active []*AtomicSwap

	for _, swap := range sm.swaps {
		if swap.Status == SwapStatusActive {
			active = append(active, swap)
		}
	}

	return active
}

// CleanupExpiredSwaps removes expired swaps
func (sm *SwapManager) CleanupExpiredSwaps() {
	now := time.Now()

	for id, swap := range sm.swaps {
		if now.After(swap.ExpiresAt) && swap.Status == SwapStatusActive {
			swap.Status = SwapStatusExpired
			delete(sm.swaps, id)
		}
	}
}

// Cross-chain integration interfaces (to be implemented)

// BitcoinAdapter interface for Bitcoin atomic swaps
type BitcoinAdapter interface {
	CreateHTLCScript(secretHash [32]byte, participantPubkey, initiatorPubkey []byte, timeout uint32) ([]byte, error)
	CreateContractTx(htlcScript []byte, amount uint64) (*wire.MsgTx, error)
	CreateRedeemTx(contractTx *wire.MsgTx, secret []byte, participantAddr []byte) (*wire.MsgTx, error)
	CreateRefundTx(contractTx *wire.MsgTx, initiatorAddr []byte, timeout uint32) (*wire.MsgTx, error)
}

// EthereumAdapter interface for Ethereum atomic swaps
type EthereumAdapter interface {
	CreateHTLCContract(secretHash [32]byte, participantAddr, initiatorAddr []byte, timeout uint64, amount *big.Int) ([]byte, error)
	CreateRedeemTx(contractAddr []byte, secret []byte) ([]byte, error)
	CreateRefundTx(contractAddr []byte) ([]byte, error)
}

// CrossChainSwap represents a cross-chain atomic swap
type CrossChainSwap struct {
	ShellSwap   *AtomicSwap
	CounterSwap interface{} // Bitcoin or Ethereum swap
	Chain       ChainType
}

// CreateCrossChainSwap creates a cross-chain atomic swap
func CreateCrossChainSwap(params *AtomicSwapParams, counterChain ChainType) (*CrossChainSwap, error) {
	// Create Shell side swap
	shellSwap, err := NewAtomicSwap(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Shell swap: %v", err)
	}

	swap := &CrossChainSwap{
		ShellSwap: shellSwap,
		Chain:     counterChain,
	}

	// Counter-chain swap creation would be handled by respective adapters
	// This is a placeholder for the cross-chain integration

	return swap, nil
}
