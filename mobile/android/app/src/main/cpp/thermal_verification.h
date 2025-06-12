#pragma once

#include <cstdint>
#include <vector>
#include <mutex>
#include <chrono>

namespace shell {
namespace mobile {

/**
 * ARM Performance Monitoring Unit interface
 * Provides access to CPU cycle counters and performance metrics
 */
class ARMPMUCounters {
public:
    ARMPMUCounters();
    ~ARMPMUCounters();

    bool initialize();
    bool isSupported() const;

    // Performance counter access
    uint64_t readCycleCount() const;
    uint64_t readInstructionCount() const;
    uint64_t readCacheAccessCount() const;
    uint64_t readCacheMissCount() const;

private:
    bool cycleCounterSupported_;
    bool instructionCounterSupported_;
    bool cacheCountersSupported_;
};

/**
 * Thermal proof data structure
 * Contains verification data for thermal compliance
 */
struct ThermalProof {
    uint64_t cycleCount;        // Actual cycles used
    uint64_t expectedCycles;    // Thermal-compliant cycle count
    uint64_t frequency;         // Operating frequency in MHz
    float temperature;          // SoC temperature in Celsius
    int64_t timestamp;          // Proof generation time
    uint8_t workHash[32];       // Hash of work being validated
    
    ThermalProof();
};

/**
 * Thermal verification statistics
 */
struct ThermalStatistics {
    float averageTemperature;
    float minTemperature;
    float maxTemperature;
    float stdDevTemperature;
    double averageFrequency;
    size_t sampleCount;
    
    ThermalStatistics();
};

/**
 * Thermal Verification System
 * Generates and validates thermal proofs for mobile mining
 */
class ThermalVerification {
public:
    ThermalVerification(uint64_t baseFreq = 2000, double tolerance = 5.0);
    ~ThermalVerification();

    // Lifecycle
    bool initialize();
    void shutdown();

    // Temperature monitoring
    void updateTemperature(float temperature);
    float getCurrentTemperature() const;

    // Thermal proof generation and validation
    uint64_t generateThermalProof(const std::vector<uint8_t>& headerBytes);
    bool validateThermalProof(uint64_t thermalProof, const std::vector<uint8_t>& headerBytes);

    // Statistics and analysis
    ThermalStatistics getThermalStatistics() const;
    std::vector<int> detectThermalCheating(const std::vector<ThermalProof>& proofs, 
                                           double threshold = 2.0) const;

    // Configuration
    void setTolerancePercent(double tolerance);
    void setBaseFrequency(uint64_t freqMHz);

private:
    // Configuration
    uint64_t baseFrequency_;    // Expected CPU frequency in MHz
    double tolerancePercent_;   // Allowed variance (e.g., 5%)
    
    // Current state
    mutable std::mutex tempMutex_;
    float currentTemperature_;
    
    // Performance monitoring
    std::unique_ptr<ARMPMUCounters> pmcCounters_;
    
    // Thermal history for statistical analysis
    mutable std::mutex historyMutex_;
    std::vector<ThermalProof> thermalHistory_;
    static const size_t MAX_HISTORY_SIZE = 1000;

    // Internal methods
    void runHalfSpeedHash(const std::vector<uint8_t>& workload);
    uint64_t calculateExpectedCycles(size_t workloadSize) const;
    uint64_t encodeProof(const ThermalProof& proof) const;
    ThermalProof decodeProof(uint64_t encodedProof, const std::vector<uint8_t>& workHash) const;
    void addToHistory(const ThermalProof& proof);
    
    // Temperature reading (platform-specific)
    float readDeviceTemperature() const;
    
    // Validation helpers
    bool validateProofWithRecomputation(const ThermalProof& proof, 
                                       const std::vector<uint8_t>& headerBytes, 
                                       double clockSpeed = 0.5) const;
    
    // Static utilities
    static std::vector<uint8_t> serializeHeaderForThermalValidation(
        const std::vector<uint8_t>& headerBytes);
    static std::vector<uint8_t> sha256Hash(const std::vector<uint8_t>& data);
    static uint64_t getCurrentTimeMs();
};

} // namespace mobile  
} // namespace shell 