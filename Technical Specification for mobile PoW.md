## Technical Specification v1.0

### Executive Summary

This document outlines a novel Proof-of-Work (PoW) cryptocurrency designed specifically to be most efficiently mined on mobile phone System-on-Chips (SoCs). By leveraging the unique architectural advantages of mobile processors—including unified memory, heterogeneous compute cores, neural processing units, and thermal constraints—we create a mining algorithm that is naturally ASIC and GPU resistant while enabling billions of smartphone users to participate in network security.

### Core Design Principles

1. **Economic ASIC Resistance**: Rather than making ASICs impossible, we make them economically equivalent to mobile SoCs
2. **Heterogeneous Compute Requirements**: Leverage all aspects of modern mobile chips (CPU, GPU, NPU, cache hierarchy)
3. **Thermal Compliance**: Enforce mobile-like power envelopes through protocol-level thermal verification
4. **Progressive Decentralization**: Enable participation from both high-end and budget smartphones
5. **Scheduled Evolution**: Regular updates aligned with mobile hardware generations

### Technical Architecture

#### Base Algorithm: Modified RandomX

We build upon RandomX, the proven CPU-friendly PoW algorithm, with mobile-specific optimizations:

- **Foundation**: RandomX's virtual machine and random program generation
- **Memory Configuration**: 
  - Fast mode: 2 GiB dataset (down from 2.25 GiB)
  - Light mode: 256 MiB dataset (for older/budget phones)
- **Instruction Set**: ARMv9-A64 with mandatory NEON/SVE vector extensions

#### Mobile-Specific Optimizations

##### 1. ARM Vector Unit Exploitation
```
- Force frequent 128-bit NEON vector operations
- Require floating-point rounding mode changes (as in original RandomX)
- Utilize ARM-specific instructions (SDOT, UDOT for int8 dot products)
- Implement SVE2 predicated operations where available
```

**ASIC Impact**: Requires implementing complex vector and FP units, increasing die size and power consumption

##### 2. Cache-Optimized Memory Access
```
- Working set: 1-3 MB (fits in L2/L3 cache of mobile SoCs)
- Access pattern: Pseudo-random pointer chasing optimized for ARM cache predictors
- Cache line alignment: 64-byte boundaries (ARM standard)
- Memory ordering: Exploit ARM's relaxed memory model
```

**ASIC Impact**: Must implement comparable SRAM sizes, approaching mobile SoC costs

##### 3. NPU Integration ("Neural Mining")
```
Every N iterations (randomized between 100-200):
1. Hash current VM state → small tensor (e.g., 32x32x3)
2. Run depthwise separable convolution on NPU
3. Feed result back into VM registers
4. Missing NPU path → 50-60% performance penalty
```

**NPU Operations**:
- Simple enough for cross-platform compatibility (NNAPI, Core ML, SNPE)
- Complex enough to require actual ML accelerators
- Progressively updated through hard forks

**ASIC Impact**: Must embed programmable neural processing units

##### 4. Heterogeneous Core Cooperation
```
Algorithm splits work between big.LITTLE cores:
- Performance cores: Main hash computation, vector operations
- Efficiency cores: Memory scheduling, pointer chasing, NPU coordination
- Synchronization required every 50-100 operations
```

**ASIC Impact**: Homogeneous designs suffer 30-40% performance penalty

##### 5. Thermal Budget Verification
```
Protocol-level thermal compliance:
1. Miners include "thermal proof" with solutions
2. Validation: Re-run 10% of work at 50% clock speed
3. Compare cycle counters - drift indicates non-compliance
4. Blocks from "overclocked" miners are invalid
```

**Implementation**:
- Use ARM's PMU (Performance Monitoring Unit) counters
- Account for legitimate variance (±5%)
- Ensures mining stays within mobile thermal envelopes (35-40°C optimal)

### Mining Application Design

#### Power Management
- **Charge-Only Mode**: Mine only when plugged in and battery >80%
- **Thermal Throttling**: Automatic reduction when approaching limits
- **Intensity Settings**: Light (2 cores), Medium (4 cores), Full (all cores)

#### User Experience
- One-click mining activation
- Real-time earnings display
- Integrated light wallet
- Background operation with notifications

#### Network Participation
- SPV client by default (bandwidth conscious)
- Optional full node mode on WiFi
- Automatic peer discovery via DHT

### Implementation Roadmap

#### Phase 1: Core Development (Months 1-6)
1. Port RandomX to ARM64 (interpreted mode)
2. Implement basic NEON optimizations
3. Develop mobile mining app prototype
4. Launch testnet with limited features

#### Phase 2: Advanced Features (Months 7-12)
1. Complete JIT compiler for ARM
2. Integrate NPU operations via abstraction layer
3. Implement thermal budget verification
4. Security audit by established firm

#### Phase 3: Mainnet Launch (Month 13)
1. Genesis block with CPU-only mining
2. Gradual activation of mobile features
3. Exchange listings and ecosystem development

### Evolution Strategy

#### 18-Month Hard Fork Cycle
Aligned with mobile SoC generations (Snapdragon, Apple Silicon, MediaTek)

**Update Scope**:
- NPU model weights and operations
- Memory access patterns
- New ARM instructions as they become standard
- Thermal budget parameters

**Governance**:
- Changes proposed 6 months in advance
- Community testing on permanent testnet
- Miner signaling for activation

### Challenges and Mitigations

#### Hardware Fragmentation
- **Challenge**: Different NPU implementations across vendors
- **Mitigation**: Abstraction layer with fallback, performance tiers

#### Battery and Device Wear
- **Challenge**: Continuous high load damages phones
- **Mitigation**: Thermal guards, charge-only mode, protocol incentives for sustainable mining

#### Network Stability
- **Challenge**: Mobile devices have intermittent connectivity
- **Mitigation**: Graceful disconnection handling, reputation system for reliable nodes

#### Vendor Centralization
- **Challenge**: Apple/Qualcomm could optimize chips for mining
- **Mitigation**: Regular algorithm updates, diverse hardware requirements

### Economic Model

#### Mining Rewards
- Block reward: Decreasing on schedule
- Bonus multiplier for consistent miners (encourages stable participation)
- Reduced rewards for "hot" mining (thermal non-compliance)

#### Fee Structure
- Dynamic fees based on network congestion
- Priority fees for faster confirmation
- Portion of fees burned to control inflation

### Security Considerations

#### 51% Attack Resistance
- Diverse hardware base (billions of potential miners)
- Geographic distribution (follows smartphone adoption)
- High cost to acquire sufficient mobile hardware

#### Selfish Mining Prevention
- Fast block propagation optimized for mobile networks
- Uncle block rewards to reduce orphan rates

### Conclusion

This mobile-optimized PoW design creates a unique mining ecosystem where:
1. ASICs must essentially recreate mobile SoCs to compete
2. Billions of existing devices can participate
3. Geographic and economic decentralization is maximized
4. Energy efficiency is incentivized through thermal constraints

By aligning mining hardware with the most widely distributed computing platform on Earth—smartphones—we create a truly democratic and decentralized cryptocurrency network.

### Technical Appendices

#### A. RandomX ARM64 Modifications
- Detailed instruction mapping table
- NEON optimization strategies
- Memory controller tuning parameters

#### B. NPU Abstraction Layer
- Supported operations across platforms
- Fallback CPU implementations
- Performance benchmarks by device

#### C. Thermal Verification Protocol
- Mathematical proof of thermal bounds
- Cycle counter methodology
- Statistical variance allowances

#### D. Reference Implementation
- Open-source mobile mining client
- Pool protocol extensions
- Stratum modifications for mobile miners