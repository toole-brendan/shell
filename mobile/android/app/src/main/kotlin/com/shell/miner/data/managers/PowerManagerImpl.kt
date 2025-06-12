package com.shell.miner.data.managers

import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.os.BatteryManager
import com.shell.miner.domain.*
import timber.log.Timber
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class PowerManagerImpl @Inject constructor(
    private val context: Context,
    private val thermalManager: ThermalManager
) : PowerManager {

    override fun getCurrentPowerState(): PowerState {
        val batteryLevel = getBatteryLevel()
        val isCharging = isCharging()
        val thermalState = thermalManager.getThermalState()
        
        val canMine = determineCanMine(batteryLevel, isCharging, thermalState)
        
        return PowerState(
            batteryLevel = batteryLevel,
            isCharging = isCharging,
            thermalState = thermalState,
            canMine = canMine
        )
    }

    override fun determineOptimalIntensity(currentConfig: MiningConfig): MiningIntensity {
        val powerState = getCurrentPowerState()
        
        // Cannot mine at all
        if (!powerState.canMine) {
            return MiningIntensity.DISABLED
        }
        
        // Determine intensity based on power and thermal conditions
        return when {
            // Charging and good thermal conditions
            powerState.isCharging && powerState.thermalState == ThermalState.NOMINAL -> {
                MiningIntensity.FULL
            }
            
            // Charging but thermal throttling
            powerState.isCharging && powerState.thermalState == ThermalState.FAIR -> {
                MiningIntensity.MEDIUM
            }
            
            // Charging but serious thermal issues
            powerState.isCharging && powerState.thermalState == ThermalState.SERIOUS -> {
                MiningIntensity.LIGHT
            }
            
            // High battery on battery power
            powerState.batteryLevel >= 90 && powerState.thermalState == ThermalState.NOMINAL -> {
                MiningIntensity.MEDIUM
            }
            
            // Good battery on battery power
            powerState.batteryLevel >= 80 && powerState.thermalState == ThermalState.NOMINAL -> {
                MiningIntensity.LIGHT
            }
            
            // Default: disabled for safety
            else -> MiningIntensity.DISABLED
        }
    }

    override fun shouldStartMining(config: MiningConfig): Boolean {
        val powerState = getCurrentPowerState()
        
        // Check basic power conditions
        if (!powerState.canMine) {
            Timber.d("Mining not allowed: power conditions not met")
            return false
        }
        
        // Check configuration-specific requirements
        if (config.chargeOnlyMode && !powerState.isCharging) {
            Timber.d("Mining not allowed: charge-only mode enabled and not charging")
            return false
        }
        
        if (powerState.batteryLevel < config.minimumBatteryLevel) {
            Timber.d("Mining not allowed: battery level ${powerState.batteryLevel}% below minimum ${config.minimumBatteryLevel}%")
            return false
        }
        
        if (thermalManager.getCurrentTemperature() > config.thermalLimit) {
            Timber.d("Mining not allowed: temperature exceeds thermal limit")
            return false
        }
        
        Timber.d("Mining allowed: all power conditions met")
        return true
    }

    override fun shouldStopMining(currentState: PowerState): Boolean {
        return when {
            // Critical thermal state
            currentState.thermalState == ThermalState.CRITICAL -> {
                Timber.w("Stopping mining: critical thermal state")
                true
            }
            
            // Very low battery
            !currentState.isCharging && currentState.batteryLevel <= 20 -> {
                Timber.w("Stopping mining: low battery (${currentState.batteryLevel}%)")
                true
            }
            
            // Battery critically low
            currentState.batteryLevel <= 10 -> {
                Timber.w("Stopping mining: critically low battery (${currentState.batteryLevel}%)")
                true
            }
            
            else -> false
        }
    }

    private fun getBatteryLevel(): Int {
        return try {
            val batteryStatus = context.registerReceiver(null, IntentFilter(Intent.ACTION_BATTERY_CHANGED))
            val level = batteryStatus?.getIntExtra(BatteryManager.EXTRA_LEVEL, -1) ?: -1
            val scale = batteryStatus?.getIntExtra(BatteryManager.EXTRA_SCALE, -1) ?: -1
            
            if (level == -1 || scale == -1) {
                Timber.w("Unable to determine battery level")
                50 // Default fallback
            } else {
                ((level.toFloat() / scale.toFloat()) * 100).toInt()
            }
        } catch (e: Exception) {
            Timber.e(e, "Error reading battery level")
            50 // Default fallback
        }
    }

    private fun isCharging(): Boolean {
        return try {
            val batteryStatus = context.registerReceiver(null, IntentFilter(Intent.ACTION_BATTERY_CHANGED))
            val status = batteryStatus?.getIntExtra(BatteryManager.EXTRA_STATUS, -1) ?: -1
            
            status == BatteryManager.BATTERY_STATUS_CHARGING || 
            status == BatteryManager.BATTERY_STATUS_FULL
        } catch (e: Exception) {
            Timber.e(e, "Error reading charging status")
            false
        }
    }

    private fun determineCanMine(
        batteryLevel: Int, 
        isCharging: Boolean, 
        thermalState: ThermalState
    ): Boolean {
        return when {
            // Critical thermal state - never mine
            thermalState == ThermalState.CRITICAL -> false
            
            // Very low battery - never mine
            batteryLevel <= 15 -> false
            
            // Charging - can mine unless thermal issues
            isCharging -> thermalState != ThermalState.CRITICAL
            
            // On battery - need good conditions
            !isCharging -> batteryLevel >= 50 && thermalState == ThermalState.NOMINAL
            
            else -> false
        }
    }

    /**
     * Get detailed battery information for debugging
     */
    fun getBatteryInfo(): BatteryInfo {
        val batteryStatus = context.registerReceiver(null, IntentFilter(Intent.ACTION_BATTERY_CHANGED))
        
        return BatteryInfo(
            level = getBatteryLevel(),
            isCharging = isCharging(),
            health = batteryStatus?.getIntExtra(BatteryManager.EXTRA_HEALTH, BatteryManager.BATTERY_HEALTH_UNKNOWN) ?: BatteryManager.BATTERY_HEALTH_UNKNOWN,
            plugged = batteryStatus?.getIntExtra(BatteryManager.EXTRA_PLUGGED, -1) ?: -1,
            voltage = batteryStatus?.getIntExtra(BatteryManager.EXTRA_VOLTAGE, -1) ?: -1,
            temperature = batteryStatus?.getIntExtra(BatteryManager.EXTRA_TEMPERATURE, -1) ?: -1
        )
    }

    data class BatteryInfo(
        val level: Int,
        val isCharging: Boolean,
        val health: Int,
        val plugged: Int,
        val voltage: Int,
        val temperature: Int // in tenths of degrees Celsius
    ) {
        fun isHealthy(): Boolean = health == BatteryManager.BATTERY_HEALTH_GOOD
        
        fun getPluggedType(): String = when (plugged) {
            BatteryManager.BATTERY_PLUGGED_AC -> "AC"
            BatteryManager.BATTERY_PLUGGED_USB -> "USB"
            BatteryManager.BATTERY_PLUGGED_WIRELESS -> "Wireless"
            else -> "Not plugged"
        }
        
        fun getTemperatureCelsius(): Float = temperature / 10.0f
    }
} 