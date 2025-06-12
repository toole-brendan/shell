#pragma once

#include <string>
#include <vector>
#include <mutex>
#include <atomic>
#include <thread>

namespace shell {
namespace mobile {

/**
 * Thermal states for mining operations
 */
enum class ThermalState {
    NORMAL,     // Normal operating temperature
    THROTTLE,   // Should throttle mining intensity
    CRITICAL    // Should stop mining immediately
};

/**
 * Android-specific thermal management for mining operations
 * Monitors device temperature and provides thermal throttling
 */
class AndroidThermalManager {
public:
    AndroidThermalManager();
    ~AndroidThermalManager();

    // Initialization
    bool initialize();

    // Monitoring control
    void startMonitoring();
    void stopMonitoring();

    // Temperature queries
    float getCurrentTemperature() const;
    ThermalState getThermalState() const;
    
    // Thermal state queries
    bool shouldThrottle() const;
    bool shouldStop() const;

    // Configuration
    void setTemperatureLimits(float throttleTemp, float maxTemp);

    // Historical data
    std::vector<float> getTemperatureHistory() const;

private:
    // Current thermal state
    mutable std::mutex tempMutex_;
    float currentTemp_;         // Current temperature in Celsius
    float maxTemp_;             // Maximum allowed temperature
    float throttleTemp_;        // Temperature at which to start throttling
    ThermalState thermalState_; // Current thermal state

    // Monitoring infrastructure
    std::atomic<bool> monitoring_;
    std::thread monitoringThread_;

    // Thermal zones (Android paths)
    std::vector<std::string> thermalZones_;

    // Temperature history
    mutable std::mutex historyMutex_;
    std::vector<float> temperatureHistory_;

    // Internal methods
    void detectThermalZones();
    void updateTemperature();
    float readTemperatureFromSensors() const;
    float simulateTemperature() const;
    void updateThermalState();
    void addToHistory(float temperature);
    void monitoringLoop();

    // Utility methods
    std::string thermalStateToString(ThermalState state) const;
};

} // namespace mobile
} // namespace shell 