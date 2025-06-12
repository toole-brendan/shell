#include "android_power_manager.h"
#include <android/log.h>
#include <unistd.h>
#include <fstream>
#include <sstream>

#define TAG "AndroidPowerManager"
#define LOGD(...) __android_log_print(ANDROID_LOG_DEBUG, TAG, __VA_ARGS__)
#define LOGE(...) __android_log_print(ANDROID_LOG_ERROR, TAG, __VA_ARGS__)
#define LOGI(...) __android_log_print(ANDROID_LOG_INFO, TAG, __VA_ARGS__)

namespace shell {
namespace mobile {

AndroidPowerManager::AndroidPowerManager()
    : batteryLevel_(100)
    , isCharging_(false)
    , currentTemp_(30.0f)
    , canMine_(false) {
}

AndroidPowerManager::~AndroidPowerManager() {
}

bool AndroidPowerManager::initialize() {
    LOGD("Initializing Android Power Manager");
    updatePowerState();
    return true;
}

void AndroidPowerManager::updatePowerState() {
    batteryLevel_ = readBatteryLevel();
    isCharging_ = readChargingState();
    currentTemp_ = readTemperature();
    
    // Update mining permission based on power state
    updateMiningPermission();
    
    LOGD("Power state updated: Battery=%d%%, Charging=%s, Temp=%.1f°C",
         batteryLevel_, isCharging_ ? "yes" : "no", currentTemp_);
}

bool AndroidPowerManager::canStartMining() const {
    return canMine_;
}

bool AndroidPowerManager::shouldStopMining() const {
    return !canMine_ || batteryLevel_ < 20 || currentTemp_ > 50.0f;
}

MiningIntensity AndroidPowerManager::determineOptimalIntensity() const {
    if (!canMine_) {
        return MiningIntensity::DISABLED;
    }
    
    // Determine intensity based on power conditions
    if (!isCharging_) {
        return MiningIntensity::DISABLED; // Never mine on battery
    }
    
    if (batteryLevel_ < 80) {
        return MiningIntensity::DISABLED; // Wait until battery is charged
    }
    
    if (currentTemp_ > 45.0f) {
        return MiningIntensity::LIGHT; // Thermal throttling
    }
    
    if (batteryLevel_ > 95 && currentTemp_ < 40.0f) {
        return MiningIntensity::FULL; // Optimal conditions
    }
    
    if (batteryLevel_ > 85) {
        return MiningIntensity::MEDIUM; // Good conditions
    }
    
    return MiningIntensity::LIGHT; // Conservative default
}

int AndroidPowerManager::getBatteryLevel() const {
    return batteryLevel_;
}

bool AndroidPowerManager::isCharging() const {
    return isCharging_;
}

float AndroidPowerManager::getTemperature() const {
    return currentTemp_;
}

void AndroidPowerManager::setMiningAllowed(bool allowed) {
    canMine_ = allowed;
    LOGI("Mining permission manually set to: %s", allowed ? "allowed" : "denied");
}

// Private methods
int AndroidPowerManager::readBatteryLevel() const {
    // Try to read from Android battery interface
    std::ifstream file("/sys/class/power_supply/battery/capacity");
    if (file.is_open()) {
        int level;
        if (file >> level) {
            return std::clamp(level, 0, 100);
        }
    }
    
    // Fallback: try alternative paths
    std::vector<std::string> batteryPaths = {
        "/sys/class/power_supply/BAT0/capacity",
        "/sys/class/power_supply/BAT1/capacity",
        "/proc/sys/kernel/battery_capacity"
    };
    
    for (const auto& path : batteryPaths) {
        std::ifstream altFile(path);
        if (altFile.is_open()) {
            int level;
            if (altFile >> level) {
                return std::clamp(level, 0, 100);
            }
        }
    }
    
    // Final fallback: assume good battery level
    LOGD("Could not read battery level, assuming 85%");
    return 85;
}

bool AndroidPowerManager::readChargingState() const {
    // Try to read charging status from Android power supply interface
    std::ifstream file("/sys/class/power_supply/battery/status");
    if (file.is_open()) {
        std::string status;
        if (file >> status) {
            return (status == "Charging" || status == "Full");
        }
    }
    
    // Try alternative paths
    std::vector<std::string> statusPaths = {
        "/sys/class/power_supply/ac/online",
        "/sys/class/power_supply/usb/online",
        "/sys/class/power_supply/wireless/online"
    };
    
    for (const auto& path : statusPaths) {
        std::ifstream statusFile(path);
        if (statusFile.is_open()) {
            int online;
            if (statusFile >> online) {
                if (online == 1) {
                    return true; // Any power source is online
                }
            }
        }
    }
    
    // Fallback: assume not charging for safety
    LOGD("Could not read charging state, assuming not charging");
    return false;
}

float AndroidPowerManager::readTemperature() const {
    // Try to read battery temperature
    std::ifstream file("/sys/class/power_supply/battery/temp");
    if (file.is_open()) {
        int tempTenthsCelsius;
        if (file >> tempTenthsCelsius) {
            // Temperature is usually in tenths of degrees Celsius
            return static_cast<float>(tempTenthsCelsius) / 10.0f;
        }
    }
    
    // Try thermal zone interfaces
    std::vector<std::string> thermalPaths = {
        "/sys/class/thermal/thermal_zone0/temp",
        "/sys/class/thermal/thermal_zone1/temp",
        "/sys/devices/virtual/thermal/thermal_zone0/temp"
    };
    
    for (const auto& path : thermalPaths) {
        std::ifstream thermalFile(path);
        if (thermalFile.is_open()) {
            int tempMilliCelsius;
            if (thermalFile >> tempMilliCelsius) {
                // Thermal zone temperatures are usually in milli-Celsius
                return static_cast<float>(tempMilliCelsius) / 1000.0f;
            }
        }
    }
    
    // Fallback: return reasonable temperature
    LOGD("Could not read temperature, assuming 35°C");
    return 35.0f;
}

void AndroidPowerManager::updateMiningPermission() {
    // Basic mining permission logic
    bool wasAllowed = canMine_;
    
    canMine_ = isCharging_ && 
               batteryLevel_ >= 80 && 
               currentTemp_ < 50.0f;
    
    if (canMine_ != wasAllowed) {
        LOGI("Mining permission changed: %s -> %s",
             wasAllowed ? "allowed" : "denied",
             canMine_ ? "allowed" : "denied");
    }
}

} // namespace mobile
} // namespace shell 