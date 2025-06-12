#pragma once

#include <vector>
#include <memory>
#include <atomic>
#include <thread>
#include <cstdint>

namespace shell {
namespace mobile {
namespace ios {

// Mining configuration for iOS
struct IOSMiningConfig {
    int intensity;
    int algorithm;
    bool npuEnabled;
    float maxTemperature;
    float throttleTemperature;
    uint32_t coreCount;
};

// Mining statistics
struct IOSMiningStats {
    double totalHashRate;
    double randomXHashRate;
    double mobileXHashRate;
    int64_t sharesSubmitted;
    int32_t blocksFound;
    float npuUtilization;
    int currentIntensity;
    int currentAlgorithm;
};

// Forward declarations
class IOSThermalManager;
class CoreMLNPUProvider;

/**
 * iOS MobileX Mining Engine
 * Optimized for Apple Silicon (M1/M2/M3/A-series)
 */
class IOSMobileXMiner {
public:
    IOSMobileXMiner();
    ~IOSMobileXMiner();

    // Lifecycle
    bool initialize(const IOSMiningConfig& config);
    bool startMining();
    bool stopMining();
    void shutdown();

    // Configuration
    bool updateConfig(const IOSMiningConfig& config);

    // Status
    IOSMiningStats getCurrentStats() const;
    bool isMining() const;
    uint64_t getTotalHashes() const;

    // Hash computation
    std::vector<uint8_t> computeHash(const std::vector<uint8_t>& headerBytes, int algorithm);

private:
    // Configuration
    IOSMiningConfig config_;
    
    // State
    std::atomic<bool> mining_;
    std::atomic<bool> initialized_;
    std::atomic<uint64_t> totalHashes_;
    std::atomic<uint64_t> randomXHashes_;
    std::atomic<uint64_t> mobileXHashes_;
    std::atomic<uint64_t> startTime_;
    
    // Mining threads
    std::vector<std::thread> miningThreads_;
    std::atomic<bool> shouldStop_;
    
    // RandomX integration
    void* randomxCache_;
    void* randomxDataset_;
    std::vector<void*> randomxVMs_;
    
    // Apple Silicon optimizations
    bool hasAppleNPU_;
    bool hasAMXUnit_;
    uint32_t pCoreCount_;
    uint32_t eCoreCount_;
    
    // Components
    std::unique_ptr<IOSThermalManager> thermalManager_;
    std::unique_ptr<CoreMLNPUProvider> npuProvider_;
    
    // Internal methods
    bool initializeRandomX();
    void cleanupRandomX();
    bool detectAppleSiliconFeatures();
    void configureCorePriorities();
    
    void miningThreadMain(int threadId);
    std::vector<uint8_t> computeRandomXHash(const std::vector<uint8_t>& input, int vmIndex);
    std::vector<uint8_t> computeMobileXHash(const std::vector<uint8_t>& input, int threadId);
    std::vector<uint8_t> applyAppleSiliconOptimizations(const std::vector<uint8_t>& hash);
    std::vector<uint8_t> applyNPUMixing(const std::vector<uint8_t>& hash);
    
    bool shouldRunNPU() const;
    void updateHashRates();
    
    // Apple Silicon specific optimizations
    void enableAppleMatrixUnit();
    void configureMemoryPrefetching();
    void setThreadAffinityToPCore(int threadId);
    void setThreadAffinityToECore(int threadId);
    
    // Thermal management
    bool checkThermalState();
    void adjustIntensityForThermal();
};

} // namespace ios
} // namespace mobile
} // namespace shell 