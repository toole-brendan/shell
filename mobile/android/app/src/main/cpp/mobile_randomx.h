#pragma once

#include <memory>
#include <atomic>
#include <vector>
#include <cstdint>
#include <functional>

namespace shell {
namespace mobile {

// Mining intensity levels
enum class MiningIntensity : int {
    DISABLED = 0,
    LIGHT = 1,
    MEDIUM = 2,
    FULL = 3
};

// Forward declarations
class ThermalVerification;
class ARM64Optimizer;
class NPUIntegration;
class HeterogeneousScheduler;

/**
 * MobileX Miner - ARM64 optimized mining implementation
 * Combines RandomX with mobile-specific optimizations
 */
class MobileXMiner {
public:
    MobileXMiner();
    ~MobileXMiner();

    // Lifecycle management
    bool initialize();
    bool startMining(MiningIntensity intensity);
    bool stopMining();
    void close();

    // Hash rate monitoring
    double getHashRate() const;
    double getRandomXHashRate() const;
    double getMobileXHashRate() const;

    // Status queries
    bool isMining() const;
    uint64_t getHashesCompleted() const;

    // Mining configuration
    void setNPUEnabled(bool enabled);
    void setThermalLimits(float maxTemp, float throttleTemp);

    // Hash computation
    std::vector<uint8_t> computeMobileXHash(const std::vector<uint8_t>& headerBytes);

private:
    // Internal state
    std::atomic<bool> mining_;
    std::atomic<uint64_t> hashesCompleted_;
    std::atomic<uint64_t> startTime_;
    
    MiningIntensity currentIntensity_;
    
    // Configuration
    bool npuEnabled_;
    uint64_t npuInterval_;
    float maxTemperature_;
    float throttleTemperature_;

    // Component interfaces
    std::unique_ptr<ThermalVerification> thermal_;
    std::unique_ptr<ARM64Optimizer> arm64_;
    std::unique_ptr<NPUIntegration> npu_;
    std::unique_ptr<HeterogeneousScheduler> scheduler_;

    // RandomX integration
    void* randomxCache_;    // RandomX cache
    void* randomxDataset_;  // RandomX dataset (optional)
    void* randomxVM_;       // RandomX VM

    // Internal methods
    bool initializeRandomX();
    void cleanupRandomX();
    
    std::vector<uint8_t> applyMobileMixing(const std::vector<uint8_t>& randomxHash);
    bool shouldRunNPU() const;
    void runNPUStep();
    void updateHashRate();
    
    // Static helpers
    static std::vector<uint8_t> serializeBlockHeader(const std::vector<uint8_t>& header);
    static std::vector<uint8_t> bytesToUint32s(const std::vector<uint8_t>& bytes);
    static std::vector<uint8_t> uint32sToBytes(const std::vector<uint32_t>& data);
};

} // namespace mobile
} // namespace shell 