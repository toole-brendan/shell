package com.shell.miner.nativecode

import com.shell.miner.domain.MiningIntensity
import timber.log.Timber

/**
 * Kotlin wrapper for the native C++ mobile mining engine.
 * Provides a clean interface to the JNI bridge.
 */
class MiningEngine {
    private var nativePtr: Long = 0
    private var isInitialized: Boolean = false

    companion object {
        init {
            try {
                System.loadLibrary("shellmining")
                Timber.d("Native mining library loaded successfully")
            } catch (e: UnsatisfiedLinkError) {
                Timber.e(e, "Failed to load native mining library")
                throw RuntimeException("Failed to load native mining library", e)
            }
        }

        // Native function declarations
        @JvmStatic
        external fun createMiner(): Long
        
        @JvmStatic
        external fun destroyMiner(minerPtr: Long)
        
        @JvmStatic
        external fun startMining(minerPtr: Long, intensity: Int): Boolean
        
        @JvmStatic
        external fun stopMining(minerPtr: Long): Boolean
        
        @JvmStatic
        external fun getHashRate(minerPtr: Long): Double
        
        @JvmStatic
        external fun getRandomXHashRate(minerPtr: Long): Double
        
        @JvmStatic
        external fun getMobileXHashRate(minerPtr: Long): Double
        
        @JvmStatic
        external fun getCurrentTemperature(minerPtr: Long): Float
        
        @JvmStatic
        external fun getNPUUtilization(minerPtr: Long): Float
        
        @JvmStatic
        external fun isMining(minerPtr: Long): Boolean
        
        @JvmStatic
        external fun generateThermalProof(minerPtr: Long): Long
        
        @JvmStatic
        external fun configureNPU(minerPtr: Long)
    }

    /**
     * Initialize the native mining engine
     */
    fun initialize(): Boolean {
        if (isInitialized) {
            Timber.w("Mining engine already initialized")
            return true
        }

        try {
            nativePtr = createMiner()
            if (nativePtr != 0L) {
                isInitialized = true
                configureNPU(nativePtr) // Initialize NPU if available
                Timber.i("Mining engine initialized successfully")
                return true
            } else {
                Timber.e("Failed to create native miner instance")
                return false
            }
        } catch (e: Exception) {
            Timber.e(e, "Exception during mining engine initialization")
            return false
        }
    }

    /**
     * Start mining with specified intensity
     */
    fun startMining(intensity: MiningIntensity): Boolean {
        if (!isInitialized) {
            Timber.e("Mining engine not initialized")
            return false
        }

        return try {
            val success = startMining(nativePtr, intensity.value)
            if (success) {
                Timber.i("Mining started with intensity: ${intensity.displayName}")
            } else {
                Timber.e("Failed to start mining")
            }
            success
        } catch (e: Exception) {
            Timber.e(e, "Exception while starting mining")
            false
        }
    }

    /**
     * Stop mining
     */
    fun stopMining(): Boolean {
        if (!isInitialized) {
            return true // Already stopped
        }

        return try {
            val success = stopMining(nativePtr)
            if (success) {
                Timber.i("Mining stopped successfully")
            } else {
                Timber.e("Failed to stop mining")
            }
            success
        } catch (e: Exception) {
            Timber.e(e, "Exception while stopping mining")
            false
        }
    }

    /**
     * Get total hash rate (RandomX + MobileX)
     */
    fun getHashRate(): Double {
        return if (isInitialized) {
            try {
                getHashRate(nativePtr)
            } catch (e: Exception) {
                Timber.e(e, "Exception getting hash rate")
                0.0
            }
        } else {
            0.0
        }
    }

    /**
     * Get RandomX-specific hash rate
     */
    fun getRandomXHashRate(): Double {
        return if (isInitialized) {
            try {
                getRandomXHashRate(nativePtr)
            } catch (e: Exception) {
                Timber.e(e, "Exception getting RandomX hash rate")
                0.0
            }
        } else {
            0.0
        }
    }

    /**
     * Get MobileX-specific hash rate
     */
    fun getMobileXHashRate(): Double {
        return if (isInitialized) {
            try {
                getMobileXHashRate(nativePtr)
            } catch (e: Exception) {
                Timber.e(e, "Exception getting MobileX hash rate")
                0.0
            }
        } else {
            0.0
        }
    }

    /**
     * Get current device temperature
     */
    fun getCurrentTemperature(): Float {
        return if (isInitialized) {
            try {
                getCurrentTemperature(nativePtr)
            } catch (e: Exception) {
                Timber.e(e, "Exception getting temperature")
                30.0f // Safe default
            }
        } else {
            30.0f
        }
    }

    /**
     * Get NPU utilization percentage
     */
    fun getNPUUtilization(): Float {
        return if (isInitialized) {
            try {
                getNPUUtilization(nativePtr)
            } catch (e: Exception) {
                Timber.e(e, "Exception getting NPU utilization")
                0.0f
            }
        } else {
            0.0f
        }
    }

    /**
     * Check if mining is currently active
     */
    fun isMining(): Boolean {
        return if (isInitialized) {
            try {
                isMining(nativePtr)
            } catch (e: Exception) {
                Timber.e(e, "Exception checking mining status")
                false
            }
        } else {
            false
        }
    }

    /**
     * Generate thermal proof for current mining state
     */
    fun generateThermalProof(): Long {
        return if (isInitialized) {
            try {
                generateThermalProof(nativePtr)
            } catch (e: Exception) {
                Timber.e(e, "Exception generating thermal proof")
                0L
            }
        } else {
            0L
        }
    }

    /**
     * Get comprehensive mining statistics
     */
    fun getMiningStats(): MiningStats {
        return if (isInitialized) {
            try {
                MiningStats(
                    isMining = isMining(nativePtr),
                    totalHashRate = getHashRate(nativePtr),
                    randomXHashRate = getRandomXHashRate(nativePtr),
                    mobileXHashRate = getMobileXHashRate(nativePtr),
                    temperature = getCurrentTemperature(nativePtr),
                    npuUtilization = getNPUUtilization(nativePtr),
                    thermalProof = generateThermalProof(nativePtr)
                )
            } catch (e: Exception) {
                Timber.e(e, "Exception getting mining stats")
                MiningStats()
            }
        } else {
            MiningStats()
        }
    }

    /**
     * Cleanup native resources
     */
    fun cleanup() {
        if (isInitialized && nativePtr != 0L) {
            try {
                stopMining(nativePtr)
                destroyMiner(nativePtr)
                nativePtr = 0L
                isInitialized = false
                Timber.i("Mining engine cleaned up successfully")
            } catch (e: Exception) {
                Timber.e(e, "Exception during cleanup")
            }
        }
    }

    /**
     * Finalize to ensure cleanup
     */
    protected fun finalize() {
        cleanup()
    }
}

/**
 * Data class containing comprehensive mining statistics
 */
data class MiningStats(
    val isMining: Boolean = false,
    val totalHashRate: Double = 0.0,
    val randomXHashRate: Double = 0.0,
    val mobileXHashRate: Double = 0.0,
    val temperature: Float = 30.0f,
    val npuUtilization: Float = 0.0f,
    val thermalProof: Long = 0L,
    val timestamp: Long = System.currentTimeMillis()
) 