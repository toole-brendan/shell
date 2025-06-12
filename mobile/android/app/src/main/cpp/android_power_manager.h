#pragma once

#include "mobile_randomx.h"
#include <string>
#include <vector>

namespace shell {
namespace mobile {

/**
 * Android-specific power management for mining operations
 * Monitors battery level, charging state, and temperature
 */
class AndroidPowerManager {
public:
    AndroidPowerManager();
    ~AndroidPowerManager();

    // Initialization
    bool initialize();

    // Power state monitoring
    void updatePowerState();
    
    // Mining permission logic
    bool canStartMining() const;
    bool shouldStopMining() const;
    MiningIntensity determineOptimalIntensity() const;

    // Power state queries
    int getBatteryLevel() const;
    bool isCharging() const;
    float getTemperature() const;

    // Manual control
    void setMiningAllowed(bool allowed);

private:
    // Current power state
    int batteryLevel_;      // 0-100%
    bool isCharging_;       // Charging state
    float currentTemp_;     // Device temperature in Celsius
    bool canMine_;          // Mining permission flag

    // Internal monitoring methods
    int readBatteryLevel() const;
    bool readChargingState() const;
    float readTemperature() const;
    void updateMiningPermission();
};

} // namespace mobile
} // namespace shell 