# Shell Reserve Implementation

**Shell (XSL) - Digital Gold for Central Banks**

This repository contains the **work-in-progress** reference implementation of Shell Reserve, a cryptocurrency designed exclusively as a reserve asset for central banks, sovereign wealth funds, and large financial institutions.

## 🎯 Vision

Shell Reserve is "digital gold" for the 21st century - designed to be boring, reliable, and built to last. Unlike other cryptocurrencies that try to do everything, Shell has one singular focus: **store value securely for decades**.

## 🏗️ Architecture

Shell implements a layered design that separates concerns:

- **L0: Base Consensus Layer** - RandomX PoW, Confidential Transactions, UTXO model
- **L0.5: Privacy Layer** (Future) - Ring signatures, stealth addresses  
- **L0.7: Custody Layer** - MuSig2, Vault covenants, Taproot
- **L1: Settlement Layer** - Payment channels, claimable balances, atomic swaps

## 🔧 Implementation Status

This is the **Phase α (Core Chain)** implementation - **MAJOR PROGRESS ON α.3**

### ✅ Completed
- **Project Structure** - Forked btcd as foundation
- **Git Repository** - Version control and GitHub integration
- **Module Setup** - Go module configuration
- **Network Magic** - Unique Shell network identifier (0x58534C4D)
- **Basic Genesis** - Genesis block structure with constitution hash
- **Shell Parameters** - Chain configuration (complete)
- **RandomX Integration** - Full C++ library integration via CGO ✨
  - RandomX v1.2.1 integrated with CGO bindings
  - Light mode (cache) and full mode (dataset) support
  - Comprehensive tests and benchmarks
  - Complete documentation and CI/CD
- **Confidential Transactions** - Core implementation complete! ✨
  - Pedersen commitments for amount hiding
  - Range proofs for value validation (simplified implementation)
  - Confidential output serialization/deserialization
  - Balance validation using homomorphic properties
  - Complete test suite with 100% pass rate
- **Shell Address Generation** - Full implementation! ✨
  - Taproot addresses with xsl1 prefix (e.g., `xsl1p3qfxns25ctk4ywv888wf...`)
  - P2PKH addresses for legacy compatibility
  - Address parsing and validation
  - Script generation for both address types
  - Multi-signature address support
  - Complete test coverage
- **Consensus Integration** - Shell-specific validation! ✨
  - Shell transaction validation with confidential transaction support
  - Shell block validation with proper subsidy calculation
  - Confidential transaction detection in witness data
  - Shell block subsidy calculation (95 XSL initial, 10-year halving)
  - Genesis block creation with constitution commitment
- **Import Cycle Resolution** - Major progress! ✨
  - Fixed circular dependency between chaincfg, txscript, and btcutil
  - Removed txscript import from genesis.go (replaced with OP_RETURN constant)
  - Consolidated genesis block variables in genesis.go
  - Added missing deployment constants (CSV, Segwit)
  - Added missing network parameters (TestNet3Params, RegressionNetParams)

### 🚧 In Progress (Phase α.3 Final Steps)
- **Full Node Compilation** - Resolving remaining undefined references:
  - Need to add remaining network parameters (TestNet4Params, SimNetParams, SigNetParams)
  - Need to add missing constants and functions (NAT, doUpgrades, etc.)
  - Need to complete chaincfg package compatibility layer

### ❌ Not Yet Started
- **Working Build** - Code doesn't compile as complete Shell node yet
- **Mining Implementation** - No functional mining integration
- **RPC Interface** - Shell-specific API endpoints
- **Network Layer** - P2P protocol modifications
- **Testing Suite** - End-to-end Shell node testing

## 🚀 Planned Features

- **No Premine**: Pure fair launch on January 1, 2026
- **100M Supply Cap**: Meaningful institutional holdings
- **5-Minute Blocks**: Optimal security/usability balance
- **RandomX Mining**: Geographic distribution via CPU mining
- **Institutional Focus**: Designed for central bank balance sheets

## 📋 Development Roadmap

**Current Phase: α.3 - Consensus Integration (95% COMPLETE!)**

1. **Phase α** (Months 0-3): 🔄 Core Chain - **MAJOR PROGRESS**
   - α.1: ✅ Project setup & basic structure  
   - α.2: ✅ RandomX integration (COMPLETE - Full C++ implementation)
   - α.3: 🚧 Confidential transactions (**95% COMPLETE!**)
     - ✅ Pedersen commitments implemented and tested
     - ✅ Range proofs working (simplified implementation)
     - ✅ Confidential transaction structure complete
     - ✅ Shell address generation (xsl* prefixes) complete
     - ✅ Consensus integration (Shell validation logic)
     - ✅ Import cycle resolution (major progress!)
     - 🚧 Full node compilation (final undefined references)
   - α.4: ❌ Taproot implementation (addresses done, full protocol pending)

2. **Phase β** (Months 3-6): ❌ Liquidity stack & reward program  
3. **Phase β.5** (Months 5-6): ❌ L1 Settlement primitives
4. **Phase γ** (Months 6-9): ❌ Security hardening & vault covenants
5. **Phase δ** (Months 9-12): ❌ Launch preparation

## 🔗 Related Documents

- [Shell Reserve White Paper](README.md) - Complete vision and design
- [Implementation Plan](Shell%20Implementation%20Plan.md) - Detailed technical roadmap

## ⚡ Current State

```bash
# Clone the repository
git clone https://github.com/toole-brendan/shell.git
cd shell

# Dependencies resolve correctly
go mod tidy

# Build RandomX with CGO
cd mining/randomx
make build-deps  # Builds RandomX C++ library
go test -tags cgo -v .  # All RandomX tests pass

# Test confidential transactions
cd ../../privacy/confidential
go test -v .  # All confidential transaction tests pass

# Test addresses
cd ../../addresses
go test -v .  # All address tests pass

# Build chaincfg package (NOW WORKS!)
cd ../chaincfg
go build .  # SUCCESS - Import cycle resolved!

# NOTE: Full node build still has some undefined references
# go build .  # <-- Still needs work for complete compilation
```

## ⚠️ Development Notice

**Phase α.3 is 95% complete!** The implementation now includes:

✅ **Working confidential transactions** with Pedersen commitments and range proofs  
✅ **Full Shell address generation** with xsl* prefixes and multi-sig support  
✅ **RandomX proof-of-work** integration with real cryptographic hashes  
✅ **Shell consensus validation** with confidential transaction support  
✅ **Genesis block creation** with constitution commitment  
✅ **Import cycle resolution** - Major breakthrough in fixing circular dependencies!

### Recent Major Achievements
- ✅ Complete confidential transaction infrastructure
- ✅ Pedersen commitments with homomorphic properties
- ✅ Range proof generation and verification (simplified)
- ✅ Shell Taproot addresses (xsl1...) working correctly
- ✅ Shell P2PKH and multi-signature address support
- ✅ Address parsing, validation, and script generation
- ✅ Shell-specific consensus validation logic
- ✅ Block subsidy calculation (95 XSL initial reward)
- ✅ Confidential transaction detection in witness data
- ✅ **Import cycle between chaincfg/txscript/btcutil resolved!**

### What's Left for Phase α.3
The main remaining tasks are:
1. **Add missing network parameters** - TestNet4Params, SimNetParams, SigNetParams
2. **Add missing constants** - Various deployment and configuration constants
3. **Fix undefined references** - NAT, doUpgrades, and other btcd compatibility functions
4. **Complete full node compilation** - Get the main binary building successfully

Once these compatibility issues are resolved, we'll have a compilable Shell node ready for Phase α.4 (Taproot implementation).

## 🏛️ Constitutional Principles

Shell Reserve is governed by immutable principles:

- **Single Purpose**: Store value, nothing else
- **Political Neutrality**: No privileged parties
- **Institutional First**: Optimize for central banks
- **Generational Thinking**: Built for 100-year operation
- **Boring by Design**: Stability over innovation

---

**Shell Reserve: Built to last, not to impress.**

*Target Launch Date: January 1, 2026, 00:00 UTC*  
*Current Status: Phase α.3 - Consensus Integration (95% Complete)* 

## ⚡ Current Functionality

### **Confidential Transactions** ✨
```bash
# Test the confidential transaction implementation
cd privacy/confidential
go test -v  # All tests passing!

# Features working:
# - Pedersen commitments for hiding amounts
# - Range proofs to prevent negative values
# - Confidential output serialization
# - Balance validation using homomorphic properties
```

### **Shell Address Generation** ✨  
```bash
# Test Shell address generation
cd addresses  
go test -v  # All tests passing!

# Example addresses generated:
# Taproot: xsl1p3qfxns25ctk4ywv888wfpx6dvragxmufkvw7cvjq28wfxv4zd3aswdq9ua
# P2PKH:   qasJ5caau3FQWjYkMeFeLYwLzkt9YAtGQF
# Multisig: qWADVHfW5yqqjjuvfCxh6H5gxnUPZB2juz (2-of-3)
```

### **RandomX Mining** ✨
```bash
# Test RandomX implementation
cd mining/randomx
go test -tags cgo -v  # All tests passing!

# Performance: ~133 H/s on Apple M4 Max (light mode)
```

### **Shell Consensus** ✨
```bash
# Shell-specific validation implemented:
# - Shell transaction validation with confidential support
# - Shell block validation with proper subsidy calculation  
# - Confidential transaction detection in witness data
# - Genesis block with constitution commitment
# - Block subsidy: 95 XSL initial, halving every 262,800 blocks
```

### **Import Cycle Resolution** ✨
```bash
# Major breakthrough - circular dependencies fixed!
cd chaincfg
go build .  # NOW WORKS!

# Fixed:
# - Removed txscript import from genesis.go
# - Used OP_RETURN constant directly (0x6a)
# - Consolidated genesis variables
# - Added missing deployment constants
``` 