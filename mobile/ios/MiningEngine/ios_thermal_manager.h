#pragma once

#include <vector>
#include <atomic>
#include <memory>
#include <thread>
#include <cstdint>

namespace shell {
namespace mobile {
namespace ios {

// Thermal state enumeration
enum class IOSThermalState {
    Normal = 0,
    Fair = 1,
    Serious = 2,
    Critical = 3
};

// Thermal monitoring state
struct IOSThermalMonitorState {
    float temperature;
    int state;
    bool isThrottling;
};

/**
 * iOS Thermal Manager
 * Provides native access to iOS thermal sensors and thermal verification
 * Uses IOKit for direct hardware access where possible
 */
class IOSThermalManager {
public:
    IOSThermalManager();
    ~IOSThermalManager();

    // Lifecycle
    bool initialize();
    void shutdown();

    // Thermal monitoring
    IOSThermalMonitorState getCurrentState();
    float getCurrentTemperature();
    IOSThermalState getCurrentThermalState();
    bool isThrottling();

    // Thermal verification for mining
    std::vector<uint8_t> generateThermalProof(float temperature, uint64_t cycleCount);
    bool validateProof(const std::vector<uint8_t>& proof, float reportedTemperature);

    // Configuration
    void setTemperatureThresholds(float warning, float critical);
    void setMonitoringInterval(double interval);
    void enableContinuousMonitoring(bool enable);

    // Callbacks
    void setThermalWarningCallback(std::function<void(float)> callback);
    void setThermalCriticalCallback(std::function<void(float)> callback);

private:
    // Monitoring state
    std::atomic<bool> initialized_;
    std::atomic<bool> monitoring_;
    std::atomic<float> currentTemperature_;
    std::atomic<int> currentState_;
    std::atomic<bool> isThrottling_;
    
    // Configuration
    float warningThreshold_;
    float criticalThreshold_;
    double monitoringInterval_;
    bool continuousMonitoring_;
    
    // Monitoring thread
    std::thread monitoringThread_;
    std::atomic<bool> shouldStop_;
    
    // Callbacks
    std::function<void(float)> thermalWarningCallback_;
    std::function<void(float)> thermalCriticalCallback_;
    
    // IOKit integration
    void* thermalService_;  // io_service_t
    void* thermalConnection_; // io_connect_t
    
    // Internal methods
    bool initializeIOKit();
    void cleanupIOKit();
    bool connectToThermalService();
    void disconnectFromThermalService();
    
    float readTemperatureFromIOKit();
    float readTemperatureFromProcessInfo();
    IOSThermalState getThermalStateFromProcessInfo();
    
    void monitoringThreadMain();
    void checkThermalState();
    void handleThermalWarning(float temperature);
    void handleThermalCritical(float temperature);
    
    // Apple-specific thermal sensor access
    bool readThermalSensors();
    float readCPUTemperature();
    float readGPUTemperature();
    float readSoCTemperature();
    
    // Thermal proof generation
    std::vector<uint8_t> computeThermalHash(float temperature, uint64_t cycleCount, uint64_t timestamp);
    uint64_t getCPUCycleCount();
    uint64_t getTimestamp();
    
    // Device-specific implementations
    bool isAppleSilicon();
    std::string getDeviceModel();
    float getDeviceSpecificThermalLimit();
    
    // Validation helpers
    bool isTemperatureReasonable(float temperature);
    bool isCycleCountValid(uint64_t cycleCount, float temperature);
    bool isProofTimestampValid(uint64_t timestamp);
    
    // Utility methods
    std::vector<uint8_t> serializeThermalData(float temperature, uint64_t cycleCount, uint64_t timestamp);
    bool deserializeThermalData(const std::vector<uint8_t>& data, float& temperature, 
                               uint64_t& cycleCount, uint64_t& timestamp);
    
    // Error handling
    void logIOKitError(const std::string& operation, int errorCode);
    void logThermalError(const std::string& message);
};

// Utility functions for thermal management
namespace thermal_utils {
    // Temperature conversion utilities
    float celsiusToFahrenheit(float celsius);
    float fahrenheitToCelsius(float fahrenheit);
    
    // Thermal state utilities
    const char* thermalStateToString(IOSThermalState state);
    IOSThermalState intToThermalState(int state);
    
    // Device-specific thermal limits
    float getRecommendedThermalLimit(const std::string& deviceModel);
    bool shouldThrottleAtTemperature(float temperature, const std::string& deviceModel);
    
    // Thermal proof validation
    bool isValidThermalProof(const std::vector<uint8_t>& proof);
    double calculateThermalEfficiency(float temperature, uint64_t cycleCount);
}

} // namespace ios
} // namespace mobile
} // namespace shell 