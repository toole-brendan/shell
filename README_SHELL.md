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

This is the **Phase α (Core Chain)** implementation - **EARLY DEVELOPMENT**

### ✅ Completed (Project Setup)
- **Project Structure** - Forked btcd as foundation
- **Git Repository** - Version control and GitHub integration
- **Module Setup** - Go module configuration
- **Network Magic** - Unique Shell network identifier (0x58534C4D)
- **Basic Genesis** - Genesis block structure with constitution hash

### 🚧 In Progress (Core Features)
- **Shell Parameters** - Chain configuration (partially done)
- **RandomX Integration** - CPU-friendly mining algorithm
- **Confidential Transactions** - Amount hiding via Pedersen commitments
- **Address Generation** - xsl* prefixed addresses
- **Consensus Rules** - Shell-specific validation logic

### ❌ Not Yet Started
- **Working Build** - Code doesn't compile as Shell node yet
- **Mining Implementation** - No functional mining
- **RPC Interface** - Shell-specific API endpoints
- **Network Layer** - P2P protocol modifications
- **Testing Suite** - Shell-specific test coverage

## 🚀 Planned Features

- **No Premine**: Pure fair launch on January 1, 2026
- **100M Supply Cap**: Meaningful institutional holdings
- **5-Minute Blocks**: Optimal security/usability balance
- **RandomX Mining**: Geographic distribution via CPU mining
- **Institutional Focus**: Designed for central bank balance sheets

## 📋 Development Roadmap

**Current Phase: α.1 - Basic Implementation (25% complete)**

1. **Phase α** (Months 0-3): 🔄 Core Chain - IN PROGRESS
   - α.1: ✅ Project setup & basic structure  
   - α.2: 🚧 RandomX integration
   - α.3: ❌ Confidential transactions
   - α.4: ❌ Taproot implementation

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

# NOTE: Build currently fails - Shell features not yet implemented
# go build  # <-- This doesn't work yet

# Dependencies resolve correctly
go mod tidy

# Basic structure inspection
ls -la  # See forked btcd structure with Shell modifications
```

## ⚠️ Development Notice

**This is early-stage development code.** The implementation is not functional yet and cannot:
- Mine Shell blocks
- Process Shell transactions  
- Connect to Shell network
- Generate Shell addresses

This repository currently serves as the foundation for implementing Shell Reserve features on top of the proven btcd codebase.

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
*Current Status: Early Development (Phase α.1)* 