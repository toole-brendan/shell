# Shell Reserve Implementation

**Shell (XSL) - Digital Gold for Central Banks**

This repository contains the reference implementation of Shell Reserve, a cryptocurrency designed exclusively as a reserve asset for central banks, sovereign wealth funds, and large financial institutions.

## 🎯 Vision

Shell Reserve is "digital gold" for the 21st century - designed to be boring, reliable, and built to last. Unlike other cryptocurrencies that try to do everything, Shell has one singular focus: **store value securely for decades**.

## 🏗️ Architecture

Shell implements a layered design that separates concerns:

- **L0: Base Consensus Layer** - RandomX PoW, Confidential Transactions, UTXO model
- **L0.5: Privacy Layer** (Future) - Ring signatures, stealth addresses  
- **L0.7: Custody Layer** - MuSig2, Vault covenants, Taproot
- **L1: Settlement Layer** - Payment channels, claimable balances, atomic swaps

## 🔧 Implementation Status

This is the **Phase α (Core Chain)** implementation, featuring:

✅ **Forked from btcd** - Proven Bitcoin codebase as foundation  
✅ **Shell-specific parameters** - 100M XSL supply, 5-minute blocks  
✅ **RandomX PoW** - CPU-friendly mining  
✅ **Shell network magic** - Unique network identifier  
✅ **Genesis block** - Fair launch with constitution hash  
✅ **Address prefixes** - xsl* addresses for Shell network  

## 🚀 Key Features

- **No Premine**: Pure fair launch on January 1, 2026
- **100M Supply Cap**: Meaningful institutional holdings
- **5-Minute Blocks**: Optimal security/usability balance
- **RandomX Mining**: Geographic distribution via CPU mining
- **Institutional Focus**: Designed for central bank balance sheets

## 📋 Next Steps

The implementation roadmap follows these phases:

1. **Phase α** (Months 0-3): ✅ Core Chain - DONE
2. **Phase β** (Months 3-6): Liquidity stack & reward program  
3. **Phase β.5** (Months 5-6): L1 Settlement primitives
4. **Phase γ** (Months 6-9): Security hardening & vault covenants
5. **Phase δ** (Months 9-12): Launch preparation

## 🔗 Related Documents

- [Shell Reserve White Paper](README.md) - Complete vision and design
- [Implementation Plan](Shell%20Implementation%20Plan.md) - Detailed technical roadmap

## ⚡ Quick Start

```bash
# Clone the repository
git clone https://github.com/toole-brendan/shell.git
cd shell

# Build Shell Reserve
go build

# Run tests
go test ./...
```

## 🏛️ Constitutional Principles

Shell Reserve is governed by immutable principles:

- **Single Purpose**: Store value, nothing else
- **Political Neutrality**: No privileged parties
- **Institutional First**: Optimize for central banks
- **Generational Thinking**: Built for 100-year operation
- **Boring by Design**: Stability over innovation

---

**Shell Reserve: Built to last, not to impress.**

*Launch Date: January 1, 2026, 00:00 UTC* 