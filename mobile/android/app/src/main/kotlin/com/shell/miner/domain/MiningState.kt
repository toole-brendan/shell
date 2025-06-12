package com.shell.miner.domain

import kotlinx.coroutines.flow.Flow

/**
 * Core mining state for the application
 */
data class MiningState(
    val isMining: Boolean = false,
    val hashRate: Double = 0.0,
    val randomXHashRate: Double = 0.0,
    val mobileXHashRate: Double = 0.0,
    val sharesSubmitted: Long = 0L,
    val blocksFound: Int = 0,
    val temperature: Float = 30.0f,
    val batteryLevel: Int = 100,
    val isCharging: Boolean = false,
    val estimatedEarnings: Double = 0.0,
    val projectedDailyEarnings: Double = 0.0,
    val intensity: MiningIntensity = MiningIntensity.MEDIUM,
    val algorithm: MiningAlgorithm = MiningAlgorithm.DUAL,
    val npuUtilization: Float = 0.0f,
    val thermalThrottling: Boolean = false,
    val error: String? = null
)

/**
 * Mining intensity levels optimized for mobile devices
 */
enum class MiningIntensity(val value: Int, val displayName: String, val description: String) {
    DISABLED(0, "Disabled", "No mining activity"),
    LIGHT(1, "Light", "2 cores, battery-friendly"),
    MEDIUM(2, "Medium", "4 cores, balanced performance"),
    FULL(3, "Full", "All cores, maximum performance");

    fun getCoreCount(deviceClass: DeviceClass): Int = when (this) {
        DISABLED -> 0
        LIGHT -> 2
        MEDIUM -> when (deviceClass) {
            DeviceClass.BUDGET -> 2
            DeviceClass.MIDRANGE -> 3
            DeviceClass.FLAGSHIP -> 4
        }
        FULL -> when (deviceClass) {
            DeviceClass.BUDGET -> 4
            DeviceClass.MIDRANGE -> 6
            DeviceClass.FLAGSHIP -> 8
        }
    }
}

/**
 * Mining algorithm selection
 */
enum class MiningAlgorithm(val displayName: String) {
    RANDOMX("RandomX Only"),
    MOBILEX("MobileX Only"),
    DUAL("RandomX + MobileX")
}

/**
 * Device classification for mining optimization
 */
enum class DeviceClass(val displayName: String) {
    BUDGET("Budget Device"),
    MIDRANGE("Mid-range Device"),
    FLAGSHIP("Flagship Device");

    fun getExpectedHashRate(intensity: MiningIntensity): Double = when (this) {
        BUDGET -> when (intensity) {
            MiningIntensity.DISABLED -> 0.0
            MiningIntensity.LIGHT -> 15.0
            MiningIntensity.MEDIUM -> 25.0
            MiningIntensity.FULL -> 40.0
        }
        MIDRANGE -> when (intensity) {
            MiningIntensity.DISABLED -> 0.0
            MiningIntensity.LIGHT -> 30.0
            MiningIntensity.MEDIUM -> 60.0
            MiningIntensity.FULL -> 90.0
        }
        FLAGSHIP -> when (intensity) {
            MiningIntensity.DISABLED -> 0.0
            MiningIntensity.LIGHT -> 50.0
            MiningIntensity.MEDIUM -> 120.0
            MiningIntensity.FULL -> 170.0
        }
    }

    fun getThermalLimit(): Float = when (this) {
        BUDGET -> 50.0f
        MIDRANGE -> 45.0f
        FLAGSHIP -> 40.0f
    }
}

/**
 * Mining configuration settings
 */
data class MiningConfig(
    val intensity: MiningIntensity = MiningIntensity.MEDIUM,
    val algorithm: MiningAlgorithm = MiningAlgorithm.DUAL,
    val deviceClass: DeviceClass = DeviceClass.MIDRANGE,
    val npuEnabled: Boolean = true,
    val thermalLimit: Float = 45.0f,
    val chargeOnlyMode: Boolean = true,
    val minimumBatteryLevel: Int = 80,
    val poolUrl: String = "stratum+tcp://pool.shell.reserve:4333",
    val walletAddress: String = ""
)

/**
 * Power management state
 */
data class PowerState(
    val batteryLevel: Int,
    val isCharging: Boolean,
    val thermalState: ThermalState,
    val canMine: Boolean
)

/**
 * Thermal state categories
 */
enum class ThermalState {
    NOMINAL,     // Normal temperature, full performance
    FAIR,        // Slightly warm, reduce intensity
    SERIOUS,     // Hot, significant throttling
    CRITICAL     // Too hot, stop mining
}

/**
 * Mining share submitted to pool
 */
data class MiningShare(
    val nonce: Long,
    val hash: String,
    val difficulty: Double,
    val timestamp: Long,
    val thermalProof: Long? = null  // Mobile-specific thermal proof
)

/**
 * Repository interface for mining operations
 */
interface MiningRepository {
    suspend fun startMining(config: MiningConfig): Result<Unit>
    suspend fun stopMining(): Result<Unit>
    suspend fun updateIntensity(intensity: MiningIntensity): Result<Unit>
    suspend fun submitShare(share: MiningShare): Result<Boolean>
    fun getMiningState(): Flow<MiningState>
    fun getPowerState(): Flow<PowerState>
}

/**
 * Power management interface
 */
interface PowerManager {
    fun getCurrentPowerState(): PowerState
    fun determineOptimalIntensity(currentConfig: MiningConfig): MiningIntensity
    fun shouldStartMining(config: MiningConfig): Boolean
    fun shouldStopMining(currentState: PowerState): Boolean
}

/**
 * Thermal management interface
 */
interface ThermalManager {
    fun getCurrentTemperature(): Float
    fun getThermalState(): ThermalState
    fun shouldThrottle(currentTemp: Float, limit: Float): Boolean
    fun generateThermalProof(): Long
}

/**
 * Pool client interface for network communication
 */
interface PoolClient {
    suspend fun connect(poolUrl: String): Result<Unit>
    suspend fun disconnect(): Result<Unit>
    suspend fun getWork(): Result<MiningWork>
    suspend fun submitWork(share: MiningShare): Result<Boolean>
    fun isConnected(): Boolean
}

/**
 * Mining work from pool
 */
data class MiningWork(
    val jobId: String,
    val blockHeader: String,
    val target: String,
    val difficulty: Double,
    val timestamp: Long
) 