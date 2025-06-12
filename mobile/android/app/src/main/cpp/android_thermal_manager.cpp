#include "android_thermal_manager.h"
#include <android/log.h>
#include <fstream>
#include <sstream>
#include <algorithm>
#include <thread>
#include <chrono>

#define TAG "AndroidThermalManager"
#define LOGD(...) __android_log_print(ANDROID_LOG_DEBUG, TAG, __VA_ARGS__)
#define LOGE(...) __android_log_print(ANDROID_LOG_ERROR, TAG, __VA_ARGS__)
#define LOGI(...) __android_log_print(ANDROID_LOG_INFO, TAG, __VA_ARGS__)

namespace shell {
namespace mobile {

AndroidThermalManager::AndroidThermalManager()
    : currentTemp_(35.0f)
    , maxTemp_(45.0f)
    , throttleTemp_(40.0f)
    , thermalState_(ThermalState::NORMAL)
    , monitoring_(false) {
}

AndroidThermalManager::~AndroidThermalManager() {
    stopMonitoring();
}

bool AndroidThermalManager::initialize() {
    LOGD("Initializing Android Thermal Manager");
    
    // Detect available thermal zones
    detectThermalZones();
    
    // Read initial temperature
    updateTemperature();
    
    return true;
}

void AndroidThermalManager::startMonitoring() {
    if (monitoring_.load()) {
        return; // Already monitoring
    }
    
    monitoring_.store(true);
    
    // Start monitoring thread
    monitoringThread_ = std::thread([this]() {
        monitoringLoop();
    });
    
    LOGI("Thermal monitoring started");
}

void AndroidThermalManager::stopMonitoring() {
    if (!monitoring_.load()) {
        return; // Not monitoring
    }
    
    monitoring_.store(false);
    
    if (monitoringThread_.joinable()) {
        monitoringThread_.join();
    }
    
    LOGI("Thermal monitoring stopped");
}

float AndroidThermalManager::getCurrentTemperature() const {
    std::lock_guard<std::mutex> lock(tempMutex_);
    return currentTemp_;
}

ThermalState AndroidThermalManager::getThermalState() const {
    std::lock_guard<std::mutex> lock(tempMutex_);
    return thermalState_;
}

bool AndroidThermalManager::shouldThrottle() const {
    std::lock_guard<std::mutex> lock(tempMutex_);
    return thermalState_ >= ThermalState::THROTTLE;
}

bool AndroidThermalManager::shouldStop() const {
    std::lock_guard<std::mutex> lock(tempMutex_);
    return thermalState_ >= ThermalState::CRITICAL;
}

void AndroidThermalManager::setTemperatureLimits(float throttleTemp, float maxTemp) {
    std::lock_guard<std::mutex> lock(tempMutex_);
    throttleTemp_ = throttleTemp;
    maxTemp_ = maxTemp;
    
    // Re-evaluate thermal state with new limits
    updateThermalState();
    
    LOGI("Thermal limits updated: throttle=%.1f°C, max=%.1f°C", throttleTemp, maxTemp);
}

std::vector<float> AndroidThermalManager::getTemperatureHistory() const {
    std::lock_guard<std::mutex> lock(historyMutex_);
    return temperatureHistory_;
}

// Private methods
void AndroidThermalManager::detectThermalZones() {
    thermalZones_.clear();
    
    // Common Android thermal zone paths
    std::vector<std::string> possibleZones = {
        "/sys/class/thermal/thermal_zone0/temp",
        "/sys/class/thermal/thermal_zone1/temp",
        "/sys/class/thermal/thermal_zone2/temp",
        "/sys/class/thermal/thermal_zone3/temp",
        "/sys/devices/virtual/thermal/thermal_zone0/temp",
        "/sys/devices/virtual/thermal/thermal_zone1/temp",
        "/sys/class/power_supply/battery/temp"
    };
    
    for (const auto& path : possibleZones) {
        std::ifstream file(path);
        if (file.is_open()) {
            int temp;
            if (file >> temp) {
                thermalZones_.push_back(path);
                LOGD("Found thermal zone: %s", path.c_str());
            }
        }
    }
    
    if (thermalZones_.empty()) {
        LOGD("No thermal zones found, using fallback temperature monitoring");
    } else {
        LOGI("Detected %zu thermal zones", thermalZones_.size());
    }
}

void AndroidThermalManager::updateTemperature() {
    float newTemp = readTemperatureFromSensors();
    
    {
        std::lock_guard<std::mutex> lock(tempMutex_);
        currentTemp_ = newTemp;
        updateThermalState();
    }
    
    // Add to history
    addToHistory(newTemp);
}

float AndroidThermalManager::readTemperatureFromSensors() const {
    std::vector<float> temperatures;
    
    // Read from all available thermal zones
    for (const auto& zonePath : thermalZones_) {
        std::ifstream file(zonePath);
        if (file.is_open()) {
            int tempValue;
            if (file >> tempValue) {
                float temp;
                
                // Convert based on typical Android thermal zone formats
                if (zonePath.find("battery") != std::string::npos) {
                    // Battery temperature is usually in tenths of degrees
                    temp = static_cast<float>(tempValue) / 10.0f;
                } else {
                    // Thermal zones are usually in milli-Celsius
                    temp = static_cast<float>(tempValue) / 1000.0f;
                }
                
                // Sanity check: reasonable temperature range
                if (temp >= 10.0f && temp <= 100.0f) {
                    temperatures.push_back(temp);
                }
            }
        }
    }
    
    if (temperatures.empty()) {
        // Fallback: simulate temperature based on time and load
        return simulateTemperature();
    }
    
    // Return the maximum temperature (most conservative)
    return *std::max_element(temperatures.begin(), temperatures.end());
}

float AndroidThermalManager::simulateTemperature() const {
    // Simple temperature simulation for testing
    auto now = std::chrono::steady_clock::now();
    auto millis = std::chrono::duration_cast<std::chrono::milliseconds>(now.time_since_epoch()).count();
    
    // Base temperature + some variation
    float baseTemp = 35.0f;
    float variation = 5.0f * std::sin(millis / 10000.0);
    
    return baseTemp + variation;
}

void AndroidThermalManager::updateThermalState() {
    ThermalState newState;
    
    if (currentTemp_ >= maxTemp_) {
        newState = ThermalState::CRITICAL;
    } else if (currentTemp_ >= throttleTemp_) {
        newState = ThermalState::THROTTLE;
    } else {
        newState = ThermalState::NORMAL;
    }
    
    if (newState != thermalState_) {
        ThermalState oldState = thermalState_;
        thermalState_ = newState;
        
        LOGI("Thermal state changed: %s -> %s (%.1f°C)",
             thermalStateToString(oldState).c_str(),
             thermalStateToString(newState).c_str(),
             currentTemp_);
    }
}

void AndroidThermalManager::addToHistory(float temperature) {
    std::lock_guard<std::mutex> lock(historyMutex_);
    
    temperatureHistory_.push_back(temperature);
    
    // Maintain maximum history size
    const size_t maxHistorySize = 1000;
    if (temperatureHistory_.size() > maxHistorySize) {
        temperatureHistory_.erase(temperatureHistory_.begin(),
                                 temperatureHistory_.begin() + (temperatureHistory_.size() - maxHistorySize));
    }
}

void AndroidThermalManager::monitoringLoop() {
    LOGD("Thermal monitoring loop started");
    
    while (monitoring_.load()) {
        updateTemperature();
        
        // Sleep for monitoring interval
        std::this_thread::sleep_for(std::chrono::seconds(1));
    }
    
    LOGD("Thermal monitoring loop ended");
}

std::string AndroidThermalManager::thermalStateToString(ThermalState state) const {
    switch (state) {
        case ThermalState::NORMAL:
            return "NORMAL";
        case ThermalState::THROTTLE:
            return "THROTTLE";
        case ThermalState::CRITICAL:
            return "CRITICAL";
        default:
            return "UNKNOWN";
    }
}

} // namespace mobile
} // namespace shell 