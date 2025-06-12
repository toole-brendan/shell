package com.shell.miner.testing

import android.content.Context
import android.util.Log
import com.shell.miner.data.managers.PowerManagerImpl
import com.shell.miner.data.managers.ThermalManagerImpl
import com.shell.miner.domain.*
import com.shell.miner.nativecode.MiningEngine
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.*
import kotlin.time.Duration
import kotlin.time.Duration.Companion.seconds
import kotlin.time.Duration.Companion.minutes

/**
 * Benchmark runner for mobile mining performance testing and optimization validation.
 * Used during integration testing to verify performance across device classes.
 */
class BenchmarkRunner(
    private val context: Context,
    private val miningEngine: MiningEngine,
    private val powerManager: PowerManagerImpl,
    private val thermalManager: ThermalManagerImpl
) {
    companion object {
        private const val TAG = "BenchmarkRunner"
    }
    
    private val benchmarkScope = CoroutineScope(Dispatchers.Default + SupervisorJob())
    
    /**
     * Run quick performance validation test (suitable for CI/CD)
     */
    suspend fun runQuickValidation(): ValidationResult {
        Log.i(TAG, "Starting quick performance validation")
        
        return try {
            // Initialize mining engine
            if (!miningEngine.initializeEngine()) {
                return ValidationResult(
                    success = false,
                    message = "Failed to initialize mining engine",
                    metrics = emptyMap()
                )
            }
            
            // Quick hash rate test (10 seconds)
            val hashRateResult = testHashRate(MiningIntensity.LIGHT, 10.seconds)
            
            // Quick thermal check (5 seconds)
            val thermalResult = testThermalResponse(5.seconds)
            
            // NPU availability check
            val npuResult = testNPUAvailability()
            
            val allPassed = hashRateResult.success && thermalResult.success && npuResult.success
            
            ValidationResult(
                success = allPassed,
                message = if (allPassed) "All quick validation tests passed" else "Some validation tests failed",
                metrics = mapOf(
                    "hash_rate" to hashRateResult.value.toString(),
                    "temperature_stable" to thermalResult.success.toString(),
                    "npu_available" to npuResult.success.toString()
                )
            )
            
        } catch (e: Exception) {
            Log.e(TAG, "Quick validation failed", e)
            ValidationResult(
                success = false,
                message = "Validation failed: ${e.message}",
                metrics = emptyMap()
            )
        } finally {
            miningEngine.stopMining()
        }
    }
    
    /**
     * Run device classification and optimization test
     */
    suspend fun runDeviceOptimizationTest(): OptimizationResult {
        Log.i(TAG, "Starting device optimization test")
        
        val deviceClass = classifyDevice()
        Log.i(TAG, "Device classified as: $deviceClass")
        
        // Test optimal intensity for device class
        val optimalIntensity = when (deviceClass) {
            DeviceClass.FLAGSHIP -> MiningIntensity.FULL
            DeviceClass.MIDRANGE -> MiningIntensity.MEDIUM
            DeviceClass.BUDGET -> MiningIntensity.LIGHT
        }
        
        // Run optimization test
        val optimizedResult = testHashRate(optimalIntensity, 30.seconds)
        val baselineResult = testHashRate(MiningIntensity.LIGHT, 30.seconds)
        
        val improvementRatio = optimizedResult.value / baselineResult.value
        
        return OptimizationResult(
            deviceClass = deviceClass,
            optimalIntensity = optimalIntensity,
            baselineHashRate = baselineResult.value,
            optimizedHashRate = optimizedResult.value,
            improvementRatio = improvementRatio,
            recommendation = generateOptimizationRecommendation(deviceClass, improvementRatio)
        )
    }
    
    /**
     * Test hash rate performance at specified intensity
     */
    private suspend fun testHashRate(intensity: MiningIntensity, duration: Duration): TestResult {
        Log.i(TAG, "Testing hash rate at $intensity intensity for ${duration.inWholeSeconds}s")
        
        val samples = mutableListOf<Double>()
        val startTime = System.currentTimeMillis()
        
        return try {
            miningEngine.startMining(intensity)
            
            val monitoringJob = launch {
                while (System.currentTimeMillis() - startTime < duration.inWholeMilliseconds) {
                    delay(1000)
                    
                    val metrics = miningEngine.getPerformanceMetrics()
                    samples.add(metrics.hashRate)
                    
                    Log.d(TAG, "Hash rate sample: ${metrics.hashRate} H/s")
                }
            }
            
            monitoringJob.join()
            
            val averageHashRate = samples.average()
            val hashRateStable = samples.size >= 5 && samples.takeLast(5).all { 
                kotlin.math.abs(it - averageHashRate) / averageHashRate < 0.2 // Within 20%
            }
            
            TestResult(
                success = hashRateStable && averageHashRate > 0,
                value = averageHashRate,
                message = if (hashRateStable) "Hash rate stable at ${String.format("%.1f", averageHashRate)} H/s" 
                         else "Hash rate unstable"
            )
            
        } catch (e: Exception) {
            Log.e(TAG, "Hash rate test failed", e)
            TestResult(success = false, value = 0.0, message = "Test failed: ${e.message}")
        } finally {
            miningEngine.stopMining()
            delay(2.seconds) // Cool down
        }
    }
    
    /**
     * Test thermal response and stability
     */
    private suspend fun testThermalResponse(duration: Duration): TestResult {
        Log.i(TAG, "Testing thermal response for ${duration.inWholeSeconds}s")
        
        val initialTemp = thermalManager.getCurrentTemperature()
        var maxTemp = initialTemp
        val startTime = System.currentTimeMillis()
        
        return try {
            miningEngine.startMining(MiningIntensity.MEDIUM)
            
            val monitoringJob = launch {
                while (System.currentTimeMillis() - startTime < duration.inWholeMilliseconds) {
                    delay(1000)
                    
                    val currentTemp = thermalManager.getCurrentTemperature()
                    maxTemp = kotlin.math.max(maxTemp, currentTemp)
                    
                    Log.d(TAG, "Temperature: ${currentTemp}°C")
                }
            }
            
            monitoringJob.join()
            
            val tempRise = maxTemp - initialTemp
            val thermalStable = tempRise < 10.0f // Less than 10°C rise in test period
            
            TestResult(
                success = thermalStable && maxTemp < 50.0f,
                value = maxTemp.toDouble(),
                message = "Max temp: ${String.format("%.1f", maxTemp)}°C, rise: ${String.format("%.1f", tempRise)}°C"
            )
            
        } catch (e: Exception) {
            Log.e(TAG, "Thermal test failed", e)
            TestResult(success = false, value = maxTemp.toDouble(), message = "Test failed: ${e.message}")
        } finally {
            miningEngine.stopMining()
        }
    }
    
    /**
     * Test NPU availability and basic functionality
     */
    private suspend fun testNPUAvailability(): TestResult {
        Log.i(TAG, "Testing NPU availability")
        
        return try {
            val npuAvailable = miningEngine.isNPUAvailable()
            
            if (npuAvailable) {
                val npuInitialized = miningEngine.initializeNPU()
                if (npuInitialized) {
                    // Quick NPU functionality test
                    miningEngine.startMining(MiningIntensity.LIGHT)
                    delay(5.seconds)
                    
                    val metrics = miningEngine.getPerformanceMetrics()
                    val npuUtilization = metrics.npuUtilization
                    
                    miningEngine.stopMining()
                    
                    TestResult(
                        success = npuUtilization > 0,
                        value = npuUtilization.toDouble(),
                        message = "NPU utilization: ${String.format("%.1f", npuUtilization)}%"
                    )
                } else {
                    TestResult(success = false, value = 0.0, message = "NPU initialization failed")
                }
            } else {
                TestResult(success = true, value = 0.0, message = "NPU not available (fallback to CPU)")
            }
            
        } catch (e: Exception) {
            Log.e(TAG, "NPU test failed", e)
            TestResult(success = false, value = 0.0, message = "NPU test failed: ${e.message}")
        }
    }
    
    /**
     * Classify device based on hardware specifications
     */
    private fun classifyDevice(): DeviceClass {
        val coreCount = Runtime.getRuntime().availableProcessors()
        val activityManager = context.getSystemService(Context.ACTIVITY_SERVICE) as android.app.ActivityManager
        val memInfo = android.app.ActivityManager.MemoryInfo()
        activityManager.getMemoryInfo(memInfo)
        val totalMemoryGB = (memInfo.totalMem / (1024 * 1024 * 1024)).toInt()
        
        return when {
            coreCount >= 8 && totalMemoryGB >= 6 -> DeviceClass.FLAGSHIP
            coreCount >= 6 && totalMemoryGB >= 4 -> DeviceClass.MIDRANGE
            else -> DeviceClass.BUDGET
        }
    }
    
    /**
     * Generate optimization recommendation based on test results
     */
    private fun generateOptimizationRecommendation(
        deviceClass: DeviceClass,
        improvementRatio: Double
    ): String {
        return when {
            improvementRatio > 2.0 -> "Excellent optimization potential - use higher intensity settings"
            improvementRatio > 1.5 -> "Good optimization potential - current settings are effective"
            improvementRatio > 1.2 -> "Moderate optimization - consider thermal management"
            else -> "Limited optimization potential - use conservative settings"
        }
    }
    
    /**
     * Run integration test to verify all components work together
     */
    suspend fun runIntegrationTest(): IntegrationTestResult {
        Log.i(TAG, "Running integration test")
        
        val results = mutableListOf<String>()
        var overallSuccess = true
        
        try {
            // Test 1: Mining engine initialization
            if (miningEngine.initializeEngine()) {
                results.add("✅ Mining engine initialization: PASS")
            } else {
                results.add("❌ Mining engine initialization: FAIL")
                overallSuccess = false
            }
            
            // Test 2: Power management integration
            val powerCheck = powerManager.shouldStartMining(
                MiningConfig(intensity = MiningIntensity.LIGHT)
            )
            results.add("✅ Power management check: ${if (powerCheck) "PASS" else "CONDITIONAL"}")
            
            // Test 3: Thermal management integration
            val thermalCheck = thermalManager.canMineAtIntensity(MiningIntensity.LIGHT)
            results.add("✅ Thermal management check: ${if (thermalCheck) "PASS" else "CONDITIONAL"}")
            
            // Test 4: Brief mining test
            val miningResult = testHashRate(MiningIntensity.LIGHT, 10.seconds)
            if (miningResult.success) {
                results.add("✅ Mining functionality: PASS (${String.format("%.1f", miningResult.value)} H/s)")
            } else {
                results.add("❌ Mining functionality: FAIL")
                overallSuccess = false
            }
            
            // Test 5: NPU integration (if available)
            val npuResult = testNPUAvailability()
            results.add("✅ NPU integration: ${if (npuResult.success) "PASS" else "OPTIONAL"}")
            
        } catch (e: Exception) {
            Log.e(TAG, "Integration test failed", e)
            results.add("❌ Integration test exception: ${e.message}")
            overallSuccess = false
        }
        
        return IntegrationTestResult(
            success = overallSuccess,
            testResults = results,
            timestamp = System.currentTimeMillis()
        )
    }
}

// Data classes for test results
data class ValidationResult(
    val success: Boolean,
    val message: String,
    val metrics: Map<String, String>
)

data class TestResult(
    val success: Boolean,
    val value: Double,
    val message: String
)

data class OptimizationResult(
    val deviceClass: DeviceClass,
    val optimalIntensity: MiningIntensity,
    val baselineHashRate: Double,
    val optimizedHashRate: Double,
    val improvementRatio: Double,
    val recommendation: String
)

data class IntegrationTestResult(
    val success: Boolean,
    val testResults: List<String>,
    val timestamp: Long
) 