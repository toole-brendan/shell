# Shell v2.5 - Minimal Implementation Plan

## Executive Summary

Project: Shell Reserve
Token: Shell (XSL)

Shell (XSL) is a minimal cryptocurrency designed exclusively as a reserve asset for central banks and sovereign wealth funds. Version 2.5 strikes the perfect balance between simplicity and institutional utility, including only essential features like claimable balances, document hashes, and ISO 20022 compatibility.

### Key Design Principles
- **Minimal Architecture**: L0 (base), L0.7 (basic custody), L1 (institutional settlement)
- **UTXO Model**: Proven Bitcoin architecture for auditability
- **Simple Privacy**: Confidential Transactions only (amounts hidden, flows visible)
- **Institutional Features**: Claimable balances, document hashes, ISO 20022
- **Realistic Blocks**: 500KB normal, 1MB maximum for reliable 5-minute propagation
- **100M Supply Cap**: Simple mining schedule, no pre-allocations

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│ L1: Institutional Settlement Layer                      │
│ • Bilateral Payment Channels (2-party only)            │
│ • Claimable Balances (Stellar-style escrow)           │
│ • Document Hashes (simple commitments)                 │
│ • ISO 20022 message mapping                            │
│ • Atomic Swaps (HTLCs)                                 │
├─────────────────────────────────────────────────────────┤
│ L0.7: Basic Custody Layer                               │
│ • Standard Multisig (2-of-3, 3-of-5, 11-of-15)        │
│ • Time Locks (nLockTime, CLTV)                        │
│ • Taproot for efficiency                               │
├─────────────────────────────────────────────────────────┤
│ L0: Base Consensus Layer                                │
│ • RandomX PoW (5-min blocks)                           │
│ • UTXO Model (Bitcoin-style)                           │
│ • Confidential Transactions                            │
│ • 500KB blocks (1MB emergency max)                     │
└─────────────────────────────────────────────────────────┘
```

## Core Specifications

### Consensus Parameters
| Parameter | Value | Rationale |
|-----------|-------|-----------|
| **Ticker** | XSL | Shell |
| **Supply Cap** | 100,000,000 XSL | Round number for institutions |
| **Block Time** | 5 minutes | Balance between speed and security |
| **Normal Block Size** | 500KB | Reliable global propagation |
| **Max Block Size** | 1MB | Emergency only with 10x fees |
| **Initial Reward** | 95 XSL/block | Simple distribution |
| **Halving** | Every 262,800 blocks (~10 years) | Generational planning |
| **Consensus** | RandomX PoW | CPU-friendly |
| **Minimum Transaction** | 1 XSL | Institutional focus |

### Layer Specifications

#### L0: Base Consensus Layer
- **RandomX PoW**: CPU mining, no ASICs
- **UTXO Model**: Bitcoin-style for simplicity
- **Confidential Transactions**: Hide amounts only
- **Simple Fee Model**: 0.001 XSL/byte burned
- **No Finality Gadget**: Just use confirmations
- **No Complex Scripts**: Basic operations only

#### L0.7: Basic Custody Layer
- **Standard Multisig**: 2-of-3, 3-of-5, etc.
- **Time Locks**: nLockTime and CLTV only
- **Taproot**: For efficiency
- **No Covenants**: Too complex
- **No MuSig2**: Standard multisig is sufficient

#### L1: Institutional Settlement Layer
- **Bilateral Channels**: 2-party only, no routing
- **Claimable Balances**: Stellar-style conditional payments
- **Document Hashes**: Simple OP_HASH256 commitments
- **ISO 20022**: Basic SWIFT message mapping
- **Atomic Swaps**: Basic HTLCs

## Development Phases

### Phase α: Core Chain (Months 0-3)

#### α.1 Base Implementation
```go
// Fork btcd and simplify
git clone https://github.com/btcsuite/btcd shell
cd shell

// Core modifications in chaincfg/params.go
var MainNetParams = Params{
    Name:                   "mainnet",
    Net:                    wire.ShellMainNet,
    DefaultPort:            "8533",
    GenesisBlock:          &genesisBlock,
    TargetTimePerBlock:    time.Minute * 5,
    SubsidyHalvingInterval: 262800,
    MaxSupply:             100000000 * 1e8,
    InitialSubsidy:        95 * 1e8,
    
    // RandomX parameters
    RandomXSeedRotation:   2048,
    RandomXMemory:         2 * 1024 * 1024 * 1024, // 2GB
    
    // Block size limits
    MaxBlockSize:          500 * 1024,  // 500KB normal
    EmergencyBlockSize:    1024 * 1024, // 1MB emergency
}
```

#### α.2 RandomX Integration
```go
// mining/randomx.go
type RandomXMiner struct {
    cache   *randomx.Cache
    dataset *randomx.Dataset
    vm      *randomx.VM
}

func (m *RandomXMiner) Mine(header *wire.BlockHeader, target *big.Int) bool {
    // Simple mining loop
    for nonce := uint32(0); nonce < maxNonce; nonce++ {
        header.Nonce = nonce
        hash := m.computeRandomXHash(header)
        
        if hashBeatsDifficulty(hash, target) {
            return true
        }
    }
    return false
}
```

#### α.3 Confidential Transactions
```go
// privacy/confidential.go
type ConfidentialOutput struct {
    Commitment   [33]byte      // Pedersen commitment
    RangeProof   []byte        // Bulletproof
    ScriptPubKey []byte        // Standard script
}

func CreateConfidentialOutput(amount uint64, script []byte) (*ConfidentialOutput, error) {
    // Generate blinding factor
    blind := generateBlindingFactor()
    
    // Create Pedersen commitment
    commitment := PedersenCommit(amount, blind)
    
    // Create range proof
    proof := BulletproofProve(amount, blind)
    
    return &ConfidentialOutput{
        Commitment:   commitment,
        RangeProof:   proof,
        ScriptPubKey: script,
    }, nil
}
```

### Phase β: Basic Features (Months 3-6)

#### β.1 Standard Multisig
```go
// txscript/multisig.go
func CreateMultisigScript(m int, pubkeys [][]byte) ([]byte, error) {
    builder := NewScriptBuilder()
    
    // Add m
    builder.AddOp(ScriptNum(m))
    
    // Add public keys
    for _, key := range pubkeys {
        builder.AddData(key)
    }
    
    // Add n
    builder.AddOp(ScriptNum(len(pubkeys)))
    
    // Add CHECKMULTISIG
    builder.AddOp(OP_CHECKMULTISIG)
    
    return builder.Script()
}
```

#### β.2 Time Locks
```go
// txscript/timelock.go
func CreateTimeLockScript(lockTime uint32, pubKeyHash []byte) ([]byte, error) {
    builder := NewScriptBuilder()
    
    // Add time lock
    builder.AddInt64(int64(lockTime))
    builder.AddOp(OP_CHECKLOCKTIMEVERIFY)
    builder.AddOp(OP_DROP)
    
    // Standard P2PKH after timelock
    builder.AddOp(OP_DUP)
    builder.AddOp(OP_HASH160)
    builder.AddData(pubKeyHash)
    builder.AddOp(OP_EQUALVERIFY)
    builder.AddOp(OP_CHECKSIG)
    
    return builder.Script()
}
```

#### β.3 Fee Structure
```go
// mempool/fee.go
const (
    MinFeeRate     = 1000      // 0.001 XSL/byte
    MinTransaction = 100000000  // 1 XSL minimum
)

func CalculateFee(txSize int) int64 {
    fee := int64(txSize) * MinFeeRate
    if fee < MinTransaction {
        fee = MinTransaction
    }
    return fee
}
```

### Phase γ: Settlement (Months 6-9)

#### γ.1 Bilateral Channels
```go
// channels/bilateral.go
type BilateralChannel struct {
    ChannelID    [32]byte
    Party1       PublicKey
    Party2       PublicKey
    Capacity     int64
    Balance1     int64
    Balance2     int64
    Nonce        uint64
}

func (c *BilateralChannel) CreateOpenTx() (*wire.MsgTx, error) {
    // Create 2-of-2 multisig
    script, _ := CreateMultisigScript(2, [][]byte{
        c.Party1.SerializeCompressed(),
        c.Party2.SerializeCompressed(),
    })
    
    // Create funding transaction
    tx := wire.NewMsgTx(wire.TxVersion)
    tx.AddTxOut(&wire.TxOut{
        Value:    c.Capacity,
        PkScript: script,
    })
    
    return tx, nil
}

func (c *BilateralChannel) UpdateBalance(newBalance1, newBalance2 int64) error {
    // Verify balance conservation
    if newBalance1 + newBalance2 != c.Capacity {
        return ErrInvalidBalance
    }
    
    // Update state
    c.Balance1 = newBalance1
    c.Balance2 = newBalance2
    c.Nonce++
    
    return nil
}
```

#### γ.2 Claimable Balances (Stellar-Style)
```go
// claimable/balance.go
const (
    OP_CLAIMABLE_CREATE = 0xca
    OP_CLAIMABLE_CLAIM  = 0xcb
)

type ClaimableBalance struct {
    ID         [32]byte
    Amount     int64
    Asset      AssetCode
    Claimants  []Claimant
}

type Claimant struct {
    Destination PublicKey
    Predicate   ClaimPredicate
}

type ClaimPredicate interface {
    Evaluate(ctx *ClaimContext) bool
    Serialize() []byte
}

// Predicate types (inspired by Stellar)
type PredicateUnconditional struct{}

type PredicateAnd struct {
    Left, Right ClaimPredicate
}

type PredicateOr struct {
    Left, Right ClaimPredicate
}

type PredicateNot struct {
    Inner ClaimPredicate
}

type PredicateBeforeAbsoluteTime struct {
    Timestamp int64
}

type PredicateAfterAbsoluteTime struct {
    Timestamp int64
}

// Create claimable balance transaction
func CreateClaimableBalance(amount int64, claimants []Claimant) (*wire.MsgTx, error) {
    // Generate unique ID
    id := generateClaimableID()
    
    // Build script
    builder := NewScriptBuilder()
    builder.AddOp(OP_CLAIMABLE_CREATE)
    builder.AddData(id[:])
    builder.AddInt64(amount)
    builder.AddInt64(int64(len(claimants)))
    
    for _, claimant := range claimants {
        builder.AddData(claimant.Destination.SerializeCompressed())
        builder.AddData(claimant.Predicate.Serialize())
    }
    
    script := builder.Script()
    
    // Create transaction
    tx := wire.NewMsgTx(wire.TxVersion)
    tx.AddTxOut(&wire.TxOut{
        Value:    amount,
        PkScript: script,
    })
    
    return tx, nil
}

// Claim with predicate satisfaction
func ClaimBalance(balanceID [32]byte, destination PublicKey, proof []byte) (*wire.MsgTx, error) {
    builder := NewScriptBuilder()
    builder.AddOp(OP_CLAIMABLE_CLAIM)
    builder.AddData(balanceID[:])
    builder.AddData(destination.SerializeCompressed())
    builder.AddData(proof) // Time proof or hash preimage
    
    return createTxWithScript(builder.Script())
}
```

#### γ.3 Document Hashes
```go
// documents/hash.go
const (
    OP_DOC_HASH = 0xcc
)

type DocumentHash struct {
    Hash      [32]byte
    Timestamp int64
    Reference string // External reference
}

// Create document hash commitment
func CreateDocumentHash(docHash [32]byte, reference string) (*wire.MsgTx, error) {
    builder := NewScriptBuilder()
    builder.AddOp(OP_DOC_HASH)
    builder.AddData(docHash[:])
    builder.AddInt64(time.Now().Unix())
    builder.AddData([]byte(reference))
    
    return createTxWithScript(builder.Script())
}

// Simple escrow with document condition
func CreateDocumentEscrow(amount int64, docHash [32]byte, recipient PublicKey, expiry int64) (*wire.MsgTx, error) {
    // Create hash predicate
    hashPred := &PredicateHashPreimage{
        Hash: docHash,
    }
    
    // Add time limit
    timePred := &PredicateBeforeAbsoluteTime{
        Timestamp: expiry,
    }
    
    // Combine predicates
    finalPred := &PredicateAnd{
        Left:  hashPred,
        Right: timePred,
    }
    
    claimant := Claimant{
        Destination: recipient,
        Predicate:   finalPred,
    }
    
    return CreateClaimableBalance(amount, []Claimant{claimant})
}
```

#### γ.4 ISO 20022 Integration
```go
// iso20022/bridge.go
type ISO20022Message struct {
    Type       MessageType
    Reference  string
    Amount     int64
    Currency   string
    Sender     BankIdentifier
    Receiver   BankIdentifier
    ValueDate  time.Time
}

type MessageType string
const (
    PACS008 MessageType = "pacs.008.001.08" // Credit transfer
    PACS009 MessageType = "pacs.009.001.08" // FI transfer
    CAMT056 MessageType = "camt.056.001.08" // Cancellation
    PAIN001 MessageType = "pain.001.001.09" // Initiation
)

type BankIdentifier struct {
    BIC        string
    Name       string
    Account    string
}

// Map Shell transaction to ISO 20022
func MapToISO20022(tx *wire.MsgTx, msgType MessageType) (*ISO20022Message, error) {
    msg := &ISO20022Message{
        Type:      msgType,
        Reference: tx.TxHash().String()[:16], // First 16 chars
        Currency:  "XSL",
        ValueDate: time.Now(),
    }
    
    // Extract amount (requires view key for CT)
    if viewKey != nil {
        amount, err := DecryptAmount(tx.TxOut[0], viewKey)
        if err != nil {
            return nil, err
        }
        msg.Amount = amount
    }
    
    // Map sender/receiver from metadata
    msg.Sender = extractBankID(tx.TxIn[0])
    msg.Receiver = extractBankID(tx.TxOut[0])
    
    return msg, nil
}

// Generate SWIFT-compatible reference
func GenerateSWIFTReference(tx *wire.MsgTx) string {
    hash := tx.TxHash()
    // Format: XSLYYMMDDHHMMSS + 6 chars from hash
    return fmt.Sprintf("XSL%s%s", 
        time.Now().Format("060102150405"),
        hex.EncodeToString(hash[:3]))
}

// Settlement finality proof
func GenerateSettlementProof(tx *wire.MsgTx, confirmations int) *SettlementProof {
    return &SettlementProof{
        TransactionID:   tx.TxHash().String(),
        BlockHeight:     getBlockHeight(tx),
        Confirmations:   confirmations,
        Timestamp:       time.Now(),
        ISOReference:    GenerateSWIFTReference(tx),
        Irrevocable:     confirmations >= 6,
    }
}
```

#### γ.5 Atomic Swaps
```go
// swaps/atomic.go
type AtomicSwap struct {
    SecretHash   [32]byte
    Sender       PublicKey
    Recipient    PublicKey
    Timeout      uint32
    Amount       int64
}

func (a *AtomicSwap) CreateHTLC() ([]byte, error) {
    builder := NewScriptBuilder()
    
    // If recipient knows secret
    builder.AddOp(OP_IF)
    builder.AddOp(OP_HASH256)
    builder.AddData(a.SecretHash[:])
    builder.AddOp(OP_EQUALVERIFY)
    builder.AddData(a.Recipient.SerializeCompressed())
    builder.AddOp(OP_CHECKSIG)
    
    // Else timeout refund to sender
    builder.AddOp(OP_ELSE)
    builder.AddInt64(int64(a.Timeout))
    builder.AddOp(OP_CHECKLOCKTIMEVERIFY)
    builder.AddOp(OP_DROP)
    builder.AddData(a.Sender.SerializeCompressed())
    builder.AddOp(OP_CHECKSIG)
    
    builder.AddOp(OP_ENDIF)
    
    return builder.Script()
}
```

### Phase δ: Launch Preparation (Months 9-12)

#### δ.1 Genesis Block
```go
// genesis/block.go
var genesisBlock = wire.MsgBlock{
    Header: wire.BlockHeader{
        Version:    1,
        PrevBlock:  chainhash.Hash{},
        MerkleRoot: genesisMerkleRoot,
        Timestamp:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
        Bits:       0x1d00ffff, // Starting difficulty
        Nonce:      0,           // To be mined
    },
    Transactions: []*wire.MsgTx{genesisCoinbaseTx},
}

var genesisCoinbaseTx = &wire.MsgTx{
    Version: 1,
    TxIn: []*wire.TxIn{{
        PreviousOutPoint: wire.OutPoint{Index: 0xffffffff},
        SignatureScript: []byte("Shell Reserve: Digital Gold for the 21st Century"),
    }},
    TxOut: []*wire.TxOut{{
        Value:    0, // No premine
        PkScript: []byte{OP_RETURN},
    }},
}
```

#### δ.2 Network Testing
```go
// test/network.go
func TestBlockPropagation(t *testing.T) {
    // Test 500KB block propagation
    block := createTestBlock(500 * 1024)
    
    start := time.Now()
    propagateBlock(block)
    duration := time.Since(start)
    
    // Should propagate globally in <30 seconds
    if duration > 30*time.Second {
        t.Errorf("Block propagation too slow: %v", duration)
    }
}
```

#### δ.3 Security Audit Checklist
- [ ] RandomX implementation review
- [ ] Confidential Transaction verification
- [ ] Consensus rule validation
- [ ] P2P network security
- [ ] RPC API security
- [ ] Key management
- [ ] Transaction validation
- [ ] Block validation
- [ ] Fee calculation
- [ ] Supply cap enforcement

## Implementation Timeline

```
Month 0-3: Core Chain (α)
├── Week 1-4: Fork btcd, integrate RandomX
├── Week 5-8: UTXO + Confidential Transactions
├── Week 9-10: Basic Taproot
├── Week 11-12: Fee model + testing

Month 3-6: Basic Features (β)
├── Week 13-15: Standard multisig
├── Week 16-18: Time locks
├── Week 19-21: Document hashes
├── Week 22-24: Basic testing

Month 6-9: Settlement (γ)
├── Week 25-26: Bilateral channels
├── Week 27-28: Claimable balances (Stellar-style)
├── Week 29-30: Document hash commitments
├── Week 31-32: ISO 20022 mapping
├── Week 33-34: Atomic swaps
├── Week 35-36: Integration testing

Month 9-12: Launch Prep (δ)
├── Week 37-39: Security audits
├── Week 40-42: Network testing
├── Week 43-45: Documentation
├── Week 46-48: Genesis mining
└── Launch: 2026-01-01 00:00 UTC
```

## Team Structure

### Core Development (8-10 people)
- **Lead Developer**: Architecture and consensus
- **Protocol Engineer**: Core blockchain
- **Cryptography Lead**: CT implementation
- **Settlement Lead**: Channels and claimables
- **ISO 20022 Expert**: SWIFT integration
- **Network Engineer**: P2P and propagation
- **QA Lead**: Testing framework
- **Technical Writer**: Documentation

### External Support
- **Security Auditors**: 2 firms
- **Legal Counsel**: Regulatory compliance

## Key Dependencies

```yaml
# Minimal dependencies
module github.com/shell/shell

go 1.21

require (
    github.com/btcsuite/btcd v0.24.0
    github.com/btcsuite/btcutil v1.1.5
    github.com/nguyenvantuan2391996/go-randomx v1.0.0
    github.com/deroproject/derohe v0.0.0  # Bulletproofs
    github.com/stellar/go v0.0.0          # Claimable balance reference
    github.com/moov-io/iso20022 v0.0.0   # ISO message parsing
    github.com/stretchr/testify v1.8.4
)
```

## Success Metrics

### Technical
- Block propagation <30 seconds globally
- Transaction validation <10ms
- Claimable balance operations <20ms
- ISO 20022 mapping <5ms
- 99.9% uptime

### Adoption (Year 1)
- 20+ institutional nodes
- $1B+ daily volume
- 5+ central banks testing
- 10+ trade finance deals
- 3+ SWIFT member banks

## Risk Mitigation

### Technical Risks
1. **Block Propagation**: Keep blocks small (500KB)
2. **CT Complexity**: Use proven Bulletproofs
3. **Claimable Balance State**: Efficient UTXO indexing
4. **ISO 20022 Compatibility**: Work with SWIFT experts

### Market Risks
1. **Low Adoption**: Focus on institutions only
2. **Regulatory Issues**: Full compliance built-in
3. **Competition**: Unique institutional focus

## Conclusion

Shell v2.5 achieves the perfect balance: maximum simplicity with essential institutional features. Claimable balances enable escrow and conditional payments. Document hashes provide audit trails without trust. ISO 20022 ensures SWIFT compatibility. All implemented with minimal complexity.

This is digital gold built for the institutions that will actually use it. No special privileges, no liquidity rewards, no trusted third parties - just pure proof-of-work and essential features.

**Essential features, eternal reliability.**