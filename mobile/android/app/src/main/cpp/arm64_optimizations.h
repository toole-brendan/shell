#pragma once

#include <cstdint>
#include <vector>
#include <string>
#include <mutex>

namespace shell {
namespace mobile {

/**
 * ARM64 CPU feature detection
 */
struct ARM64Features {
    bool hasNEON;        // 128-bit NEON vector support
    bool hasSVE;         // Scalable Vector Extension
    bool hasSVE2;        // SVE2 extensions
    bool hasDotProduct;  // Int8 dot product instructions (SDOT/UDOT)
    bool hasFP16;        // Half-precision floating-point
    bool hasATOMICS;     // LSE atomic instructions
    bool hasAES;         // AES crypto extensions
    bool hasSHA256;      // SHA256 crypto extensions
    
    // Cache information
    int cacheLineSize;   // Typically 64 bytes
    int l1CacheSize;     // L1 data cache size
    int l2CacheSize;     // L2 cache size
    int l3CacheSize;     // L3 cache size (if present)
    
    ARM64Features();
};

/**
 * CPU core topology for heterogeneous scheduling
 */
struct CoreTopology {
    int totalCores;
    int bigCores;        // Performance cores
    int littleCores;     // Efficiency cores
    std::vector<int> bigCoreIds;
    std::vector<int> littleCoreIds;
    
    CoreTopology();
};

/**
 * NEON-optimized cache structure
 */
class NEONCache {
public:
    NEONCache(size_t size, int lineSize, int ways);
    ~NEONCache();

    void initialize();
    void prefetchLine(const void* address);
    void invalidate();
    
    size_t getSize() const { return size_; }
    int getLineSize() const { return lineSize_; }

private:
    size_t size_;
    int lineSize_;
    int ways_;
    int sets_;
    std::vector<uint8_t> data_;
    bool prefetchEnabled_;
};

/**
 * Heterogeneous core scheduler for big.LITTLE architectures
 */
class HeterogeneousScheduler {
public:
    HeterogeneousScheduler();
    ~HeterogeneousScheduler();

    bool initialize();
    void shutdown();

    // Core affinity management
    bool setThreadAffinity(int coreId);
    bool runOnBigCores(std::function<void()> work);
    bool runOnLittleCores(std::function<void()> work);
    
    // Work distribution
    void configureHeterogeneousCores(int bigCoreCount, int littleCoreCount);
    uint32_t getCoreState() const;
    
    // Intensity management
    void reduceIntensity();
    void increaseIntensity();
    int getActiveCores() const;

private:
    CoreTopology topology_;
    std::vector<bool> activeCores_;
    mutable std::mutex coreMutex_;
    int currentIntensity_;
    
    void detectCoreTopology();
    bool setCPUAffinity(const std::vector<int>& coreIds);
};

/**
 * ARM64 Optimizer - Provides ARM64-specific optimizations
 */
class ARM64Optimizer {
public:
    ARM64Optimizer();
    ~ARM64Optimizer();

    // Initialization
    bool initialize();
    void shutdown();

    // Feature detection
    const ARM64Features& getFeatures() const { return features_; }
    bool hasNEON() const { return features_.hasNEON; }
    bool hasSVE() const { return features_.hasSVE; }
    bool hasDotProduct() const { return features_.hasDotProduct; }

    // NEON optimizations
    void enableNEON();
    void enableSVE();
    void enableDotProduct();
    
    std::vector<uint8_t> vectorHash(const std::vector<uint8_t>& data);
    uint32_t dotProductHash(const std::vector<uint8_t>& data, 
                           const std::vector<int8_t>& weights);

    // Memory optimizations
    std::vector<uint8_t> optimizedMemoryAccess(const std::vector<uint8_t>& dataset,
                                               const std::vector<uint32_t>& indices);
    
    // Cache management
    void prefetchCacheLine(const void* address);
    void memoryBarrier();
    int getOptimalWorkingSetSize() const;

    // Heterogeneous core management
    void configureHeterogeneousCores(int bigCores, int littleCores);
    void runOnBigCores(std::function<void()> work);
    void runOnLittleCores(std::function<void()> work);

    // ARM-specific hash operations
    std::vector<uint32_t> armSpecificHash(const std::vector<uint32_t>& state);

    // Thermal optimization
    void configureForThermalEfficiency(float maxTemp);

    // SoC identification
    std::string detectSoCType() const;

private:
    ARM64Features features_;
    std::unique_ptr<NEONCache> cache_;
    std::unique_ptr<HeterogeneousScheduler> scheduler_;
    mutable std::mutex optimizerMutex_;

    // Feature detection methods
    void detectFeatures();
    void detectCacheSizes();
    void detectCoreTopology();

    // Optimization implementations
    std::vector<uint8_t> scalarHash(const std::vector<uint8_t>& data);
    uint32_t scalarDotProduct(const std::vector<uint8_t>& data, 
                             const std::vector<int8_t>& weights);

    // NEON intrinsic wrappers
    void neonVectorAdd(const uint8_t* a, const uint8_t* b, uint8_t* result, size_t size);
    void neonVectorXor(const uint8_t* a, const uint8_t* b, uint8_t* result, size_t size);
    void neonMemoryCopy(const uint8_t* src, uint8_t* dst, size_t size);

    // Platform-specific implementations
    #ifdef __ARM_NEON
    void enableNEONIntrinsics();
    #endif
    
    #ifdef __ARM_FEATURE_SVE
    void enableSVEIntrinsics();
    #endif
};

} // namespace mobile
} // namespace shell 