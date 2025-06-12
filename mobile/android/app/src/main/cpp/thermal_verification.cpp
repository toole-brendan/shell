#include "thermal_verification.h"
#include <chrono>
#include <thread>
#include <fstream>
#include <algorithm>
#include <cmath>
#include <openssl/sha.h>

#ifdef __ANDROID__
#include <unistd.h>
#include <sys/types.h>
#include <sys/stat.h>
#endif

namespace shell {
namespace mobile {

// ARM PMU Counters implementation
ARMPMUCounters::ARMPMUCounters() 
    : cycleCounterSupported_(false)
    , instructionCounterSupported_(false)
    , cacheCountersSupported_(false) {
}

ARMPMUCounters::~ARMPMUCounters() {
}

bool ARMPMUCounters::initialize() {
    // In real implementation, this would:
    // 1. Check for PMU access permissions
    // 2. Enable PMU counters via appropriate system calls
    // 3. Verify counter functionality
    
    // For now, assume basic support is available
    cycleCounterSupported_ = true;
    instructionCounterSupported_ = true;
    cacheCountersSupported_ = false; // Typically requires kernel support
    
    return true;
}

bool ARMPMUCounters::isSupported() const {
    return cycleCounterSupported_;
}

uint64_t ARMPMUCounters::readCycleCount() const {
    if (!cycleCounterSupported_) {
        return 0;
    }
    
    // In real implementation, this would use inline assembly to read PMCCNTR_EL0
    // For now, use high-resolution timer as approximation
    auto now = std::chrono::high_resolution_clock::now();
    auto nanos = now.time_since_epoch().count();
    
    // Approximate cycle count assuming 2GHz CPU
    return static_cast<uint64_t>(nanos * 2.0);
}

uint64_t ARMPMUCounters::readInstructionCount() const {
    if (!instructionCounterSupported_) {
        return 0;
    }
    
    // Approximate instruction count (would use PMINTENSET_EL1 in real implementation)
    return readCycleCount() / 2; // Rough approximation: 2 cycles per instruction
}

uint64_t ARMPMUCounters::readCacheAccessCount() const {
    if (!cacheCountersSupported_) {
        return 0;
    }
    
    // Would access cache performance counters
    return 0;
}

uint64_t ARMPMUCounters::readCacheMissCount() const {
    if (!cacheCountersSupported_) {
        return 0;
    }
    
    // Would access cache miss counters
    return 0;
}

// ThermalProof implementation
ThermalProof::ThermalProof() 
    : cycleCount(0)
    , expectedCycles(0)
    , frequency(0)
    , temperature(0.0f)
    , timestamp(0) {
    std::memset(workHash, 0, sizeof(workHash));
}

// ThermalStatistics implementation
ThermalStatistics::ThermalStatistics()
    : averageTemperature(0.0f)
    , minTemperature(0.0f)
    , maxTemperature(0.0f)
    , stdDevTemperature(0.0f)
    , averageFrequency(0.0)
    , sampleCount(0) {
}

// ThermalVerification implementation
ThermalVerification::ThermalVerification(uint64_t baseFreq, double tolerance)
    : baseFrequency_(baseFreq)
    , tolerancePercent_(tolerance)
    , currentTemperature_(40.0f) // Default optimal temperature
    , pmcCounters_(std::make_unique<ARMPMUCounters>()) {
    
    thermalHistory_.reserve(MAX_HISTORY_SIZE);
}

ThermalVerification::~ThermalVerification() {
    shutdown();
}

bool ThermalVerification::initialize() {
    // Initialize PMU counters
    if (!pmcCounters_->initialize()) {
        return false;
    }
    
    // Read initial temperature
    float initialTemp = readDeviceTemperature();
    updateTemperature(initialTemp);
    
    return true;
}

void ThermalVerification::shutdown() {
    // Cleanup resources
    std::lock_guard<std::mutex> lock(historyMutex_);
    thermalHistory_.clear();
}

void ThermalVerification::updateTemperature(float temperature) {
    std::lock_guard<std::mutex> lock(tempMutex_);
    currentTemperature_ = temperature;
}

float ThermalVerification::getCurrentTemperature() const {
    std::lock_guard<std::mutex> lock(tempMutex_);
    return currentTemperature_;
}

uint64_t ThermalVerification::generateThermalProof(const std::vector<uint8_t>& headerBytes) {
    // Start cycle counting
    uint64_t startCycles = pmcCounters_->readCycleCount();
    auto startTime = std::chrono::high_resolution_clock::now();

    // Run subset of work at half speed to measure thermal compliance
    std::vector<uint8_t> testWorkload(headerBytes.begin(), 
                                     headerBytes.begin() + std::min(headerBytes.size(), size_t(32)));
    runHalfSpeedHash(testWorkload);

    // Measure elapsed cycles and time
    uint64_t endCycles = pmcCounters_->readCycleCount();
    auto endTime = std::chrono::high_resolution_clock::now();
    auto elapsedTime = std::chrono::duration_cast<std::chrono::microseconds>(endTime - startTime);
    
    uint64_t cycleDelta = endCycles - startCycles;

    // Calculate effective frequency
    double elapsedSeconds = elapsedTime.count() / 1e6;
    uint64_t effectiveFreq = static_cast<uint64_t>(cycleDelta / elapsedSeconds / 1e6);

    // Create thermal proof
    ThermalProof proof;
    proof.cycleCount = cycleDelta;
    proof.expectedCycles = calculateExpectedCycles(testWorkload.size());
    proof.frequency = effectiveFreq;
    proof.temperature = getCurrentTemperature();
    proof.timestamp = getCurrentTimeMs();

    // Calculate work hash
    auto workHash = sha256Hash(headerBytes);
    std::memcpy(proof.workHash, workHash.data(), std::min(workHash.size(), sizeof(proof.workHash)));

    // Store in history for statistical analysis
    addToHistory(proof);

    // Generate compact proof value
    return encodeProof(proof);
}

bool ThermalVerification::validateThermalProof(uint64_t thermalProof, 
                                               const std::vector<uint8_t>& headerBytes) {
    // Serialize header for validation (excluding thermal proof itself)
    auto validationBytes = serializeHeaderForThermalValidation(headerBytes);

    // Re-compute thermal proof for verification
    uint64_t expectedProof = generateThermalProof(validationBytes);

    // Allow tolerance for legitimate thermal differences
    double toleranceRange = static_cast<double>(expectedProof) * tolerancePercent_ / 100.0;
    uint64_t minAcceptable = static_cast<uint64_t>(expectedProof - toleranceRange);
    uint64_t maxAcceptable = static_cast<uint64_t>(expectedProof + toleranceRange);

    return (thermalProof >= minAcceptable && thermalProof <= maxAcceptable);
}

ThermalStatistics ThermalVerification::getThermalStatistics() const {
    std::lock_guard<std::mutex> lock(historyMutex_);

    ThermalStatistics stats;
    if (thermalHistory_.empty()) {
        return stats;
    }

    double totalTemp = 0.0;
    double totalFreq = 0.0;
    float minTemp = std::numeric_limits<float>::max();
    float maxTemp = std::numeric_limits<float>::lowest();

    for (const auto& proof : thermalHistory_) {
        totalTemp += proof.temperature;
        totalFreq += proof.frequency;

        minTemp = std::min(minTemp, proof.temperature);
        maxTemp = std::max(maxTemp, proof.temperature);
    }

    size_t count = thermalHistory_.size();
    stats.averageTemperature = static_cast<float>(totalTemp / count);
    stats.minTemperature = minTemp;
    stats.maxTemperature = maxTemp;
    stats.averageFrequency = totalFreq / count;
    stats.sampleCount = count;

    // Calculate standard deviation
    double tempVariance = 0.0;
    for (const auto& proof : thermalHistory_) {
        double diff = proof.temperature - stats.averageTemperature;
        tempVariance += diff * diff;
    }
    stats.stdDevTemperature = static_cast<float>(std::sqrt(tempVariance / count));

    return stats;
}

std::vector<int> ThermalVerification::detectThermalCheating(
    const std::vector<ThermalProof>& proofs, double threshold) const {
    
    if (proofs.size() < 10) {
        return {}; // Not enough data
    }

    // Calculate mean and standard deviation of temperatures
    double sum = 0.0;
    for (const auto& proof : proofs) {
        sum += proof.temperature;
    }
    double mean = sum / proofs.size();

    double variance = 0.0;
    for (const auto& proof : proofs) {
        double diff = proof.temperature - mean;
        variance += diff * diff;
    }
    double stdDev = std::sqrt(variance / proofs.size());

    // Find outliers (Z-score > threshold)
    std::vector<int> outliers;
    for (size_t i = 0; i < proofs.size(); ++i) {
        double zScore = std::abs(proofs[i].temperature - mean) / stdDev;
        if (zScore > threshold) {
            outliers.push_back(static_cast<int>(i));
        }
    }

    return outliers;
}

void ThermalVerification::setTolerancePercent(double tolerance) {
    tolerancePercent_ = tolerance;
}

void ThermalVerification::setBaseFrequency(uint64_t freqMHz) {
    baseFrequency_ = freqMHz;
}

// Private methods
void ThermalVerification::runHalfSpeedHash(const std::vector<uint8_t>& workload) {
    // This simulates running at 50% clock speed for thermal verification
    auto hash = sha256Hash(workload);

    // Artificial delay to simulate half-speed operation
    std::this_thread::sleep_for(std::chrono::microseconds(100));

    // Do some work to ensure compiler doesn't optimize this away
    for (int i = 0; i < 100; ++i) {
        hash = sha256Hash(hash);
    }
}

uint64_t ThermalVerification::calculateExpectedCycles(size_t workloadSize) const {
    // Base cycles for SHA256 operation (rough estimate)
    uint64_t baseCycles = workloadSize * 100;

    // Adjust for temperature
    float temp = getCurrentTemperature();
    double thermalMultiplier = 1.0;

    if (temp > 45.0f) {
        // Higher temperature = slower expected performance
        thermalMultiplier = 1.0 + (temp - 45.0f) * 0.02;
    } else if (temp < 35.0f) {
        // Lower temperature = faster expected performance
        thermalMultiplier = 1.0 - (35.0f - temp) * 0.01;
    }

    return static_cast<uint64_t>(baseCycles * thermalMultiplier);
}

uint64_t ThermalVerification::encodeProof(const ThermalProof& proof) const {
    // Combine various proof elements into a single uint64
    std::vector<uint8_t> data(32);
    
    // Pack proof data
    std::memcpy(data.data() + 0, &proof.cycleCount, 8);
    std::memcpy(data.data() + 8, &proof.expectedCycles, 8);
    std::memcpy(data.data() + 16, &proof.frequency, 8);
    
    uint64_t tempInt = static_cast<uint64_t>(proof.temperature * 100);
    std::memcpy(data.data() + 24, &tempInt, 8);

    auto hash = sha256Hash(data);
    
    // Return first 8 bytes as uint64
    uint64_t result = 0;
    std::memcpy(&result, hash.data(), 8);
    return result;
}

ThermalProof ThermalVerification::decodeProof(uint64_t encodedProof, 
                                              const std::vector<uint8_t>& workHash) const {
    // This is a simplified decoding - real implementation would need more sophisticated encoding/decoding
    ThermalProof proof;
    proof.cycleCount = encodedProof & 0xFFFFFFFF;
    proof.expectedCycles = (encodedProof >> 32) & 0xFFFFFFFF;
    proof.frequency = baseFrequency_; // Use base frequency as estimate
    proof.temperature = getCurrentTemperature();
    proof.timestamp = getCurrentTimeMs();
    
    if (workHash.size() >= sizeof(proof.workHash)) {
        std::memcpy(proof.workHash, workHash.data(), sizeof(proof.workHash));
    }
    
    return proof;
}

void ThermalVerification::addToHistory(const ThermalProof& proof) {
    std::lock_guard<std::mutex> lock(historyMutex_);

    thermalHistory_.push_back(proof);

    // Maintain maximum history size
    if (thermalHistory_.size() > MAX_HISTORY_SIZE) {
        thermalHistory_.erase(thermalHistory_.begin(), 
                             thermalHistory_.begin() + (thermalHistory_.size() - MAX_HISTORY_SIZE));
    }
}

float ThermalVerification::readDeviceTemperature() const {
#ifdef __ANDROID__
    // Try to read from Android thermal zones
    std::vector<std::string> thermalPaths = {
        "/sys/class/thermal/thermal_zone0/temp",
        "/sys/class/thermal/thermal_zone1/temp",
        "/sys/devices/virtual/thermal/thermal_zone0/temp",
        "/sys/devices/virtual/thermal/thermal_zone1/temp"
    };

    for (const auto& path : thermalPaths) {
        std::ifstream file(path);
        if (file.is_open()) {
            int tempMilliC;
            if (file >> tempMilliC) {
                // Convert from milli-Celsius to Celsius
                return static_cast<float>(tempMilliC) / 1000.0f;
            }
        }
    }
#endif

    // Fallback: return simulated temperature
    // In real implementation, this might read from SoC-specific interfaces
    return 40.0f + static_cast<float>(getCurrentTimeMs() % 10000) / 1000.0f;
}

bool ThermalVerification::validateProofWithRecomputation(
    const ThermalProof& proof, 
    const std::vector<uint8_t>& headerBytes, 
    double clockSpeed) const {
    
    auto startTime = std::chrono::high_resolution_clock::now();

    // Simulate reduced speed validation
    std::vector<uint8_t> workload(headerBytes.begin(), 
                                 headerBytes.begin() + std::min(headerBytes.size(), size_t(64)));

    // Run validation workload
    for (int i = 0; i < 1000; ++i) {
        auto hash = sha256Hash(workload);
        workload.assign(hash.begin(), hash.end());

        // Simulate clock speed reduction
        auto sleepDuration = std::chrono::microseconds(
            static_cast<int>(100 * (1.0 - clockSpeed)));
        std::this_thread::sleep_for(sleepDuration);
    }

    auto elapsed = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::high_resolution_clock::now() - startTime);

    // Check if elapsed time is within acceptable range
    auto expectedTime = std::chrono::milliseconds(static_cast<int>(100 / clockSpeed));
    auto tolerance = expectedTime / 10; // 10% tolerance

    return (elapsed >= (expectedTime - tolerance) && 
            elapsed <= (expectedTime + tolerance));
}

// Static utility methods
std::vector<uint8_t> ThermalVerification::serializeHeaderForThermalValidation(
    const std::vector<uint8_t>& headerBytes) {
    
    // Create a copy without the thermal proof field (last 8 bytes)
    if (headerBytes.size() >= 8) {
        return std::vector<uint8_t>(headerBytes.begin(), headerBytes.end() - 8);
    }
    
    return headerBytes;
}

std::vector<uint8_t> ThermalVerification::sha256Hash(const std::vector<uint8_t>& data) {
    std::vector<uint8_t> result(32);
    SHA256(data.data(), data.size(), result.data());
    return result;
}

uint64_t ThermalVerification::getCurrentTimeMs() {
    auto now = std::chrono::system_clock::now();
    auto duration = now.time_since_epoch();
    return std::chrono::duration_cast<std::chrono::milliseconds>(duration).count();
}

} // namespace mobile
} // namespace shell 