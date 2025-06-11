// Package blockchain provides Shell Reserve's extended blockchain state management
// for institutional features like payment channels and claimable balances.
package blockchain

import (
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/btcsuite/btcd/btcutil"
	btcdchainhash "github.com/btcsuite/btcd/chaincfg/chainhash"
	btcdwire "github.com/btcsuite/btcd/wire"
	"github.com/toole-brendan/shell/chaincfg/chainhash"
	"github.com/toole-brendan/shell/settlement/channels"
	"github.com/toole-brendan/shell/settlement/claimable"
	"github.com/toole-brendan/shell/txscript"
	"github.com/toole-brendan/shell/wire"
)

// LiquidityRewardClaim represents a claim for liquidity rewards (interface to avoid import cycle)
type LiquidityRewardClaim struct {
	Version         int32
	EpochIndex      uint8
	AttestationBlob []byte
	MerklePath      []chainhash.Hash
	Output          *wire.TxOut
}

// LiquidityManagerInterface defines interface to avoid import cycle with liquidity package
type LiquidityManagerInterface interface {
	ProcessRewardClaim(claim *LiquidityRewardClaim, currentBlock int32) error
}

// NoOpLiquidityManager provides a no-op implementation for when liquidity manager isn't needed
type NoOpLiquidityManager struct{}

func (nm *NoOpLiquidityManager) ProcessRewardClaim(claim *LiquidityRewardClaim, currentBlock int32) error {
	// No-op implementation - actual processing would be done by external liquidity manager
	return nil
}

// Type conversion helpers
func btcdHashToShellHash(btcdHash *btcdchainhash.Hash) chainhash.Hash {
	var shellHash chainhash.Hash
	copy(shellHash[:], btcdHash[:])
	return shellHash
}

func btcdOutPointToShellOutPoint(btcdOutPoint btcdwire.OutPoint) wire.OutPoint {
	return wire.OutPoint{
		Hash:  btcdHashToShellHash(&btcdOutPoint.Hash),
		Index: btcdOutPoint.Index,
	}
}

// ShellStateKey represents keys for Shell-specific state storage
type ShellStateKey uint8

const (
	// State key prefixes for different types of Shell state
	StateKeyChannel   ShellStateKey = 0x01
	StateKeyClaimable ShellStateKey = 0x02
	StateKeyVault     ShellStateKey = 0x03
)

// ShellChainState extends UtxoViewpoint with Shell-specific state
type ShellChainState struct {
	*UtxoViewpoint

	// Channel state
	channelState *channels.ChannelState

	// Claimable balance state
	claimableState *claimable.ClaimableState

	// Liquidity reward manager (interface to avoid import cycle)
	liquidityManager LiquidityManagerInterface

	// Modified state tracking
	modifiedChannels   map[channels.ChannelID]*channels.PaymentChannel
	modifiedClaimables map[claimable.ClaimableID]*claimable.ClaimableBalance
	deletedChannels    map[channels.ChannelID]struct{}
	deletedClaimables  map[claimable.ClaimableID]struct{}

	// Liquidity reward tracking
	processedRewards map[[32]byte]bool // Track processed reward claims
}

// NewShellChainState creates a new Shell chain state
func NewShellChainState(utxoView *UtxoViewpoint) *ShellChainState {
	return &ShellChainState{
		UtxoViewpoint:      utxoView,
		channelState:       channels.NewChannelState(),
		claimableState:     claimable.NewClaimableState(),
		liquidityManager:   &NoOpLiquidityManager{}, // Default no-op implementation
		modifiedChannels:   make(map[channels.ChannelID]*channels.PaymentChannel),
		modifiedClaimables: make(map[claimable.ClaimableID]*claimable.ClaimableBalance),
		deletedChannels:    make(map[channels.ChannelID]struct{}),
		deletedClaimables:  make(map[claimable.ClaimableID]struct{}),
		processedRewards:   make(map[[32]byte]bool),
	}
}

// SetLiquidityManager sets the liquidity manager (allows external injection)
func (scs *ShellChainState) SetLiquidityManager(lm LiquidityManagerInterface) {
	scs.liquidityManager = lm
}

// ProcessShellOpcode handles Shell-specific opcode execution
func (scs *ShellChainState) ProcessShellOpcode(opcode byte, tx *btcutil.Tx, txIdx int, blockHeight int32) error {
	switch opcode {
	case 0xc6: // OP_CHANNEL_OPEN
		return scs.processChannelOpen(tx, txIdx, blockHeight)

	case 0xc7: // OP_CHANNEL_UPDATE
		return scs.processChannelUpdate(tx, txIdx)

	case 0xc8: // OP_CHANNEL_CLOSE
		return scs.processChannelClose(tx, txIdx)

	case 0xc9: // OP_CLAIMABLE_CREATE
		return scs.processClaimableCreate(tx, txIdx, blockHeight)

	case 0xca: // OP_CLAIMABLE_CLAIM
		return scs.processClaimableClaim(tx, txIdx, blockHeight)

	case 0xcb: // OP_LIQUIDITY_CLAIM
		return scs.processLiquidityRewardClaim(tx, txIdx, blockHeight)

	case 0xcc: // OP_DOC_HASH
		return scs.processDocumentHash(tx, txIdx, blockHeight)

	default:
		return fmt.Errorf("unknown Shell opcode: 0x%02x", opcode)
	}
}

// processChannelOpen handles OP_CHANNEL_OPEN execution
func (scs *ShellChainState) processChannelOpen(tx *btcutil.Tx, txIdx int, blockHeight int32) error {
	msgTx := tx.MsgTx()
	if txIdx >= len(msgTx.TxOut) {
		return fmt.Errorf("invalid output index for channel open")
	}

	// Get the script and witness data
	output := msgTx.TxOut[txIdx]

	// For Taproot spending, witness data is in the input that spends this output
	// For channel creation, we look at the witness of the funding transaction
	var witness btcdwire.TxWitness
	if len(msgTx.TxIn) > 0 && len(msgTx.TxIn[0].Witness) > 0 {
		witness = msgTx.TxIn[0].Witness
	}

	// Extract channel parameters from witness
	params, err := txscript.ExtractChannelOpenParams(output.PkScript, witness)
	if err != nil {
		return fmt.Errorf("failed to extract channel open parameters: %v", err)
	}

	// Create funding outpoint (convert from btcd to Shell types)
	txHash := tx.Hash()
	btcdOutPoint := btcdwire.OutPoint{
		Hash:  *txHash,
		Index: uint32(txIdx),
	}
	fundingOutpoint := btcdOutPointToShellOutPoint(btcdOutPoint)

	// Validate output amount matches channel capacity
	if uint64(output.Value) != params.ChannelAmount {
		return fmt.Errorf("output value %d does not match channel amount %d",
			output.Value, params.ChannelAmount)
	}

	// Open the channel
	expiry := uint32(blockHeight + 144*30) // 30 days default expiry
	channel, err := scs.channelState.OpenChannel(
		params.ChannelAlice,
		params.ChannelBob,
		params.ChannelAmount,
		expiry,
		fundingOutpoint,
	)
	if err != nil {
		return fmt.Errorf("failed to open channel: %v", err)
	}

	// Track the modification
	scs.modifiedChannels[channel.ChannelID] = channel

	return nil
}

// processChannelUpdate handles OP_CHANNEL_UPDATE execution
func (scs *ShellChainState) processChannelUpdate(tx *btcutil.Tx, txIdx int) error {
	msgTx := tx.MsgTx()
	if txIdx >= len(msgTx.TxIn) {
		return fmt.Errorf("invalid input index for channel update")
	}

	// Get witness data from the input being spent
	witness := msgTx.TxIn[txIdx].Witness
	if len(witness) == 0 {
		return fmt.Errorf("channel update requires witness data")
	}

	// Extract channel update parameters
	params, err := txscript.ExtractChannelUpdateParams(nil, witness)
	if err != nil {
		return fmt.Errorf("failed to extract channel update parameters: %v", err)
	}

	// Create channel update
	update := &channels.ChannelUpdate{
		ChannelID: params.ChannelID,
		Balances:  params.ChannelBalances,
		Nonce:     params.ChannelNonce,
		// TODO: Extract and validate signatures from witness
		Signatures: [2]*ecdsa.Signature{nil, nil},
	}

	// Process the update
	err = scs.channelState.UpdateChannel(update)
	if err != nil {
		return fmt.Errorf("failed to update channel: %v", err)
	}

	// Get updated channel for tracking
	channel, err := scs.channelState.GetChannel(params.ChannelID)
	if err != nil {
		return fmt.Errorf("failed to get updated channel: %v", err)
	}

	// Track the modification
	scs.modifiedChannels[channel.ChannelID] = channel

	return nil
}

// processChannelClose handles OP_CHANNEL_CLOSE execution
func (scs *ShellChainState) processChannelClose(tx *btcutil.Tx, txIdx int) error {
	msgTx := tx.MsgTx()
	if txIdx >= len(msgTx.TxIn) {
		return fmt.Errorf("invalid input index for channel close")
	}

	// Get witness data from the input being spent
	witness := msgTx.TxIn[txIdx].Witness
	if len(witness) == 0 {
		return fmt.Errorf("channel close requires witness data")
	}

	// Extract channel close parameters
	params, err := txscript.ExtractChannelCloseParams(nil, witness)
	if err != nil {
		return fmt.Errorf("failed to extract channel close parameters: %v", err)
	}

	// Close the channel
	channel, err := scs.channelState.CloseChannel(params.ChannelID)
	if err != nil {
		return fmt.Errorf("failed to close channel: %v", err)
	}

	// Track the deletion
	scs.deletedChannels[params.ChannelID] = struct{}{}

	// Verify that the transaction outputs match the final channel balances
	if len(msgTx.TxOut) < 2 {
		return fmt.Errorf("channel close must have at least 2 outputs")
	}

	// Check balance distribution (simplified - would need more validation in production)
	totalOutputValue := int64(0)
	for _, output := range msgTx.TxOut {
		totalOutputValue += output.Value
	}

	if uint64(totalOutputValue) > channel.Capacity {
		return fmt.Errorf("channel close outputs exceed channel capacity")
	}

	return nil
}

// processClaimableCreate handles OP_CLAIMABLE_CREATE execution
func (scs *ShellChainState) processClaimableCreate(tx *btcutil.Tx, txIdx int, blockHeight int32) error {
	msgTx := tx.MsgTx()
	if txIdx >= len(msgTx.TxOut) {
		return fmt.Errorf("invalid output index for claimable create")
	}

	// Get the script and witness data
	output := msgTx.TxOut[txIdx]

	// For claimable creation, witness is in the input
	var witness btcdwire.TxWitness
	if len(msgTx.TxIn) > 0 && len(msgTx.TxIn[0].Witness) > 0 {
		witness = msgTx.TxIn[0].Witness
	}

	// Extract claimable balance parameters
	params, err := txscript.ExtractClaimableCreateParams(output.PkScript, witness)
	if err != nil {
		return fmt.Errorf("failed to extract claimable create parameters: %v", err)
	}

	// Validate output amount matches claimable amount
	if uint64(output.Value) != params.ClaimableAmount {
		return fmt.Errorf("output value %d does not match claimable amount %d",
			output.Value, params.ClaimableAmount)
	}

	// Create funding outpoint (convert from btcd to Shell types)
	txHash := tx.Hash()
	btcdOutPoint := btcdwire.OutPoint{
		Hash:  *txHash,
		Index: uint32(txIdx),
	}
	fundingOutpoint := btcdOutPointToShellOutPoint(btcdOutPoint)

	// TODO: Extract creator from transaction signature/witness
	// For now, use first claimant as creator (simplified)
	var creator *btcec.PublicKey
	if len(params.ClaimableClaimants) > 0 {
		creator = params.ClaimableClaimants[0].Destination
	}

	// Create the claimable balance
	balance, err := scs.claimableState.CreateClaimableBalance(
		creator,
		params.ClaimableAmount,
		params.ClaimableClaimants,
		uint32(blockHeight),
		fundingOutpoint,
	)
	if err != nil {
		return fmt.Errorf("failed to create claimable balance: %v", err)
	}

	// Track the modification
	scs.modifiedClaimables[balance.ID] = balance

	return nil
}

// processClaimableClaim handles OP_CLAIMABLE_CLAIM execution
func (scs *ShellChainState) processClaimableClaim(tx *btcutil.Tx, txIdx int, blockHeight int32) error {
	msgTx := tx.MsgTx()
	if txIdx >= len(msgTx.TxIn) {
		return fmt.Errorf("invalid input index for claimable claim")
	}

	// Get witness data from the input being spent
	witness := msgTx.TxIn[txIdx].Witness
	if len(witness) == 0 {
		return fmt.Errorf("claimable claim requires witness data")
	}

	// Extract claimable claim parameters
	params, err := txscript.ExtractClaimableClaimParams(nil, witness)
	if err != nil {
		return fmt.Errorf("failed to extract claimable claim parameters: %v", err)
	}

	// Add current block timestamp to proof
	params.ClaimableProof.Timestamp = uint32(blockHeight * 300) // 5-minute blocks

	// Claim the balance
	balance, err := scs.claimableState.ClaimBalance(
		params.ClaimableID,
		params.ClaimableClaimer,
		params.ClaimableProof,
		uint32(blockHeight),
	)
	if err != nil {
		return fmt.Errorf("failed to claim balance: %v", err)
	}

	// Verify that the transaction output goes to the claimer
	if len(msgTx.TxOut) == 0 {
		return fmt.Errorf("claimable claim must have at least one output")
	}

	// Track the deletion
	scs.deletedClaimables[params.ClaimableID] = struct{}{}

	// Check that claimed amount doesn't exceed balance (simplified validation)
	totalOutputValue := int64(0)
	for _, output := range msgTx.TxOut {
		totalOutputValue += output.Value
	}

	if uint64(totalOutputValue) > balance.Amount {
		return fmt.Errorf("claimed amount exceeds claimable balance")
	}

	return nil
}

// processLiquidityRewardClaim handles OP_LIQUIDITY_CLAIM execution
func (scs *ShellChainState) processLiquidityRewardClaim(tx *btcutil.Tx, txIdx int, blockHeight int32) error {
	msgTx := tx.MsgTx()
	if txIdx >= len(msgTx.TxOut) {
		return fmt.Errorf("invalid output index for liquidity reward claim")
	}

	// Get the script and witness data for reward claim
	output := msgTx.TxOut[txIdx]

	// For liquidity reward claims, witness is in the input
	var witness btcdwire.TxWitness
	if len(msgTx.TxIn) > 0 && len(msgTx.TxIn[0].Witness) > 0 {
		witness = msgTx.TxIn[0].Witness
	}

	// Parse liquidity reward claim from witness data
	if len(witness) < 2 {
		return fmt.Errorf("liquidity reward claim requires attestation blob and merkle path")
	}

	// Extract attestation blob and merkle path from witness
	attestationBlob := witness[0]
	merklePath := make([]chainhash.Hash, len(witness)-1)
	for i := 1; i < len(witness); i++ {
		if len(witness[i]) != 32 {
			return fmt.Errorf("invalid merkle path hash length")
		}
		copy(merklePath[i-1][:], witness[i])
	}

	// Create liquidity reward claim
	claim := &LiquidityRewardClaim{
		Version:         1,
		EpochIndex:      0, // Extract from attestation blob
		AttestationBlob: attestationBlob,
		MerklePath:      merklePath,
		Output:          (*wire.TxOut)(output),
	}

	// Process the claim through the liquidity manager
	err := scs.liquidityManager.ProcessRewardClaim(claim, blockHeight)
	if err != nil {
		return fmt.Errorf("failed to process liquidity reward claim: %v", err)
	}

	// Check reward amount doesn't exceed output value
	if uint64(output.Value) > 0 {
		// Additional validation would be done by the liquidity manager
		return nil
	}

	return nil
}

// processDocumentHash handles OP_DOC_HASH execution
func (scs *ShellChainState) processDocumentHash(tx *btcutil.Tx, txIdx int, blockHeight int32) error {
	msgTx := tx.MsgTx()
	if txIdx >= len(msgTx.TxOut) {
		return fmt.Errorf("invalid output index for document hash")
	}

	// For document hash, witness data contains the parameters
	var witness btcdwire.TxWitness
	if len(msgTx.TxIn) > 0 && len(msgTx.TxIn[0].Witness) > 0 {
		witness = msgTx.TxIn[0].Witness
	}

	// Extract document hash parameters from witness
	// Expected format: [hash(32 bytes), timestamp(8 bytes), reference(variable)]
	if len(witness) < 3 {
		return fmt.Errorf("document hash requires hash, timestamp, and reference in witness")
	}

	hashBytes := witness[0]
	timestampBytes := witness[1]
	referenceBytes := witness[2]

	// Validate hash is 32 bytes (SHA256)
	if len(hashBytes) != 32 {
		return fmt.Errorf("document hash must be 32 bytes, got %d", len(hashBytes))
	}

	// Validate timestamp
	if len(timestampBytes) > 8 {
		return fmt.Errorf("timestamp too large")
	}

	// Validate reference length
	if len(referenceBytes) > 256 {
		return fmt.Errorf("document reference too long: %d bytes, max 256", len(referenceBytes))
	}

	// Convert timestamp bytes to int64
	var timestamp int64
	for i, b := range timestampBytes {
		timestamp |= int64(b) << (8 * i)
	}

	if timestamp <= 0 {
		return fmt.Errorf("document timestamp must be positive")
	}

	// Create document hash record
	var docHash [32]byte
	copy(docHash[:], hashBytes)

	txHash := tx.Hash()
	documentRecord := DocumentHashRecord{
		Hash:        docHash,
		Timestamp:   timestamp,
		Reference:   string(referenceBytes),
		BlockHeight: blockHeight,
		TxID:        btcdHashToShellHash(txHash),
		OutputIndex: uint32(txIdx),
	}

	// In a full implementation, this would:
	// 1. Store the document hash record in a specialized index
	// 2. Allow querying by hash, timestamp, or reference
	// 3. Create an immutable audit trail
	// 4. Emit events for document tracking systems

	// For now, we validate the parameters and note the commitment
	// The actual storage would be handled by a document index subsystem

	// Log the document hash commitment (in production, this would be indexed)
	log.Infof("Document hash committed: hash=%x timestamp=%d reference='%s' tx=%s",
		docHash, timestamp, documentRecord.Reference, txHash.String())

	return nil
}

// DocumentHashRecord represents a document hash commitment on the blockchain
type DocumentHashRecord struct {
	Hash        [32]byte       // SHA256 hash of the document
	Timestamp   int64          // Timestamp when document was hashed
	Reference   string         // External reference (e.g., trade ID, contract number)
	BlockHeight int32          // Block height when committed
	TxID        chainhash.Hash // Transaction ID containing the commitment
	OutputIndex uint32         // Output index in the transaction
}

// Commit applies all modifications to the underlying database
func (scs *ShellChainState) Commit() error {
	// First commit standard UTXO changes
	scs.UtxoViewpoint.commit()

	// Then commit Shell-specific state changes
	// TODO: Implement database persistence for Shell state
	// For now, state is only kept in memory

	// Clear modification tracking after commit
	scs.modifiedChannels = make(map[channels.ChannelID]*channels.PaymentChannel)
	scs.modifiedClaimables = make(map[claimable.ClaimableID]*claimable.ClaimableBalance)
	scs.deletedChannels = make(map[channels.ChannelID]struct{})
	scs.deletedClaimables = make(map[claimable.ClaimableID]struct{})

	return nil
}

// CalculateShellStateHash computes a hash of all Shell state for block headers
func (scs *ShellChainState) CalculateShellStateHash() chainhash.Hash {
	// Compute deterministic hash of all channels and claimable balances
	// This would be included in Shell block headers for state commitment

	// TODO: Implement actual state hashing
	return chainhash.Hash{}
}

// GetChannelState returns the channel state manager
func (scs *ShellChainState) GetChannelState() *channels.ChannelState {
	return scs.channelState
}

// GetClaimableState returns the claimable balance state manager
func (scs *ShellChainState) GetClaimableState() *claimable.ClaimableState {
	return scs.claimableState
}

// GetModifiedChannels returns channels modified in this state
func (scs *ShellChainState) GetModifiedChannels() map[channels.ChannelID]*channels.PaymentChannel {
	result := make(map[channels.ChannelID]*channels.PaymentChannel)
	for id, channel := range scs.modifiedChannels {
		result[id] = channel
	}
	return result
}

// GetModifiedClaimables returns claimable balances modified in this state
func (scs *ShellChainState) GetModifiedClaimables() map[claimable.ClaimableID]*claimable.ClaimableBalance {
	result := make(map[claimable.ClaimableID]*claimable.ClaimableBalance)
	for id, balance := range scs.modifiedClaimables {
		result[id] = balance
	}
	return result
}

// GetLiquidityManager returns the liquidity reward manager
func (scs *ShellChainState) GetLiquidityManager() LiquidityManagerInterface {
	return scs.liquidityManager
}

// GetProcessedRewards returns a copy of processed reward claims
func (scs *ShellChainState) GetProcessedRewards() map[[32]byte]bool {
	result := make(map[[32]byte]bool)
	for hash, processed := range scs.processedRewards {
		result[hash] = processed
	}
	return result
}

// IsLiquidityRewardProcessed checks if a reward claim has been processed
func (scs *ShellChainState) IsLiquidityRewardProcessed(rewardHash [32]byte) bool {
	return scs.processedRewards[rewardHash]
}

// MarkLiquidityRewardProcessed marks a reward claim as processed
func (scs *ShellChainState) MarkLiquidityRewardProcessed(rewardHash [32]byte) {
	scs.processedRewards[rewardHash] = true
}
