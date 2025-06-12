package com.shell.miner.device

import android.content.Context
import android.os.Build
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import com.shell.miner.data.managers.PowerManagerImpl
import com.shell.miner.data.managers.ThermalManagerImpl
import com.shell.miner.domain.*
import com.shell.miner.nativecode.MiningEngine
import kotlinx.coroutines.*
import kotlinx.coroutines.test.*
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import kotlin.test.*
import kotlin.time.Duration.Companion.seconds
import kotlin.time.Duration.Companion.minutes

/**
 * Device validation tests for mobile mining performance across different hardware configurations.
 * These tests run on actual devices to validate:
 * - Performance characteristics by device class
 * - Thermal management effectiveness
 * - NPU utilization and fallback behavior
 * - Power efficiency optimization
 * - Cross-device compatibility
 */
@RunWith(AndroidJUnit4::class)
class DeviceValidationTest {
    
    private lateinit var context: Context
    private lateinit var miningEngine: MiningEngine
    private lateinit var powerManager: PowerManagerImpl
    private lateinit var thermalManager: ThermalManagerImpl
    private lateinit var deviceClassifier: DeviceClassifier
    
    @Before
    fun setup() {
        context = InstrumentationRegistry.getInstrumentation().targetContext
        miningEngine = MiningEngine()
        powerManager = PowerManagerImpl(context)
        thermalManager = ThermalManagerImpl(context)
        deviceClassifier = DeviceClassifier(context)
    }
    
    @Test
    fun `validate device classification accuracy`() {
        // Test device classification based on real hardware specs
        val deviceClass = deviceClassifier.classifyDevice()
        val deviceSpecs = deviceClassifier.getDeviceSpecs()
        
        println("Device Classification Test")
        println("==========================")
        println("Device: ${Build.MODEL}")
        println("SoC: ${deviceSpecs.socModel}")
        println("CPU Cores: ${deviceSpecs.coreCount}")
        println("RAM: ${deviceSpecs.totalMemoryGB} GB")
        println("NPU Available: ${deviceSpecs.hasNPU}")
        println("Classified as: ${deviceClass}")
        
        // Validate classification logic
        when (deviceClass) {
            DeviceClass.FLAGSHIP -> {
                assertTrue(deviceSpecs.coreCount >= 8, "Flagship should have 8+ cores")
                assertTrue(deviceSpecs.totalMemoryGB >= 6, "Flagship should have 6+ GB RAM")
            }
            DeviceClass.MIDRANGE -> {
                assertTrue(deviceSpecs.coreCount >= 6, "Midrange should have 6+ cores")
                assertTrue(deviceSpecs.totalMemoryGB >= 4, "Midrange should have 4+ GB RAM")
            }
            DeviceClass.BUDGET -> {
                assertTrue(deviceSpecs.coreCount >= 4, "Budget should have 4+ cores")
                assertTrue(deviceSpecs.totalMemoryGB >= 2, "Budget should have 2+ GB RAM")
            }
        }
    }
    
    @Test
    fun `benchmark mining performance by device class`() = runTest(timeout = 5.minutes) {
        val deviceClass = deviceClassifier.classifyDevice()
        
        println("\nMining Performance Benchmark")
        println("============================")
        
        // Initialize mining engine
        assertTrue(miningEngine.initializeEngine(), "Mining engine should initialize successfully")
        
        // Test each intensity level
        val intensityResults = mutableMapOf<MiningIntensity, PerformanceResult>()
        
        for (intensity in listOf(MiningIntensity.LIGHT, MiningIntensity.MEDIUM, MiningIntensity.FULL)) {
            // Skip FULL intensity on budget devices to prevent overheating
            if (intensity == MiningIntensity.FULL && deviceClass == DeviceClass.BUDGET) {
                continue
            }
            
            println("\nTesting intensity: $intensity")
            
            val result = benchmarkIntensity(intensity, duration = 30.seconds)
            intensityResults[intensity] = result
            
            println("Hash Rate: ${String.format("%.1f", result.hashRate)} H/s")
            println("Power: ${String.format("%.1f", result.powerConsumption)} W")
            println("Temperature: ${String.format("%.1f", result.peakTemperature)}°C")
            println("NPU Utilization: ${String.format("%.1f", result.npuUtilization)}%")
            
            // Allow cooling between tests
            delay(10.seconds)
        }
        
        // Validate performance meets expectations for device class
        validatePerformanceExpectations(deviceClass, intensityResults)
    }
    
    @Test
    fun `validate thermal management effectiveness`() = runTest(timeout = 3.minutes) {
        println("\nThermal Management Validation")
        println("=============================")
        
        val initialTemp = thermalManager.getCurrentTemperature()
        println("Initial temperature: ${String.format("%.1f", initialTemp)}°C")
        
        // Start mining at full intensity
        assertTrue(miningEngine.startMining(MiningIntensity.FULL))
        
        var thermalThrottleEvents = 0
        var maxTemperature = initialTemp
        
        // Monitor for 2 minutes or until thermal throttling occurs
        val monitoringJob = launch {
            repeat(120) { // 2 minutes with 1-second intervals
                delay(1.seconds)
                
                val currentTemp = thermalManager.getCurrentTemperature()
                maxTemperature = maxOf(maxTemperature, currentTemp)
                
                if (currentTemp > 45.0f) { // Thermal limit
                    thermalThrottleEvents++
                    println("Thermal throttling at ${String.format("%.1f", currentTemp)}°C")
                    
                    // Verify thermal manager responds appropriately
                    val suggestedIntensity = thermalManager.getSuggestedIntensity()
                    assertTrue(
                        suggestedIntensity < MiningIntensity.FULL,
                        "Thermal manager should suggest reduced intensity"
                    )
                    
                    // Test if mining engine responds to thermal throttling
                    miningEngine.adjustIntensity(suggestedIntensity)
                    delay(5.seconds) // Allow time for adjustment
                    
                    val newTemp = thermalManager.getCurrentTemperature()
                    assertTrue(
                        newTemp <= currentTemp + 1.0f, // Should not continue rising significantly
                        "Temperature should stabilize or decrease after throttling"
                    )
                }
                
                // Stop test if temperature exceeds safe limits
                if (currentTemp > 50.0f) {
                    println("SAFETY STOP: Temperature exceeded 50°C")
                    break
                }
            }
        }
        
        monitoringJob.join()
        miningEngine.stopMining()
        
        println("Max temperature reached: ${String.format("%.1f", maxTemperature)}°C")
        println("Thermal throttle events: $thermalThrottleEvents")
        
        // Validate thermal behavior
        assertTrue(maxTemperature < 55.0f, "Temperature should not exceed 55°C")
        
        if (maxTemperature > 45.0f) {
            assertTrue(thermalThrottleEvents > 0, "Thermal throttling should occur above 45°C")
        }
    }
    
    @Test
    fun `validate NPU integration and fallback`() = runTest(timeout = 2.minutes) {
        println("\nNPU Integration Validation")
        println("==========================")
        
        val hasNPU = miningEngine.isNPUAvailable()
        println("NPU Available: $hasNPU")
        
        if (hasNPU) {
            // Test NPU initialization
            val npuInitialized = miningEngine.initializeNPU()
            println("NPU Initialized: $npuInitialized")
            
            if (npuInitialized) {
                // Test mining with NPU enabled
                val npuResult = benchmarkIntensity(MiningIntensity.MEDIUM, duration = 30.seconds, enableNPU = true)
                println("NPU Hash Rate: ${String.format("%.1f", npuResult.hashRate)} H/s")
                println("NPU Utilization: ${String.format("%.1f", npuResult.npuUtilization)}%")
                
                // Test fallback to CPU
                miningEngine.fallbackToCPU()
                val cpuResult = benchmarkIntensity(MiningIntensity.MEDIUM, duration = 30.seconds, enableNPU = false)
                println("CPU Fallback Hash Rate: ${String.format("%.1f", cpuResult.hashRate)} H/s")
                
                // NPU should provide performance benefit
                assertTrue(
                    npuResult.hashRate > cpuResult.hashRate * 1.1f, // At least 10% improvement
                    "NPU should provide performance benefit over CPU fallback"
                )
                
                // NPU utilization should be significant when enabled
                assertTrue(
                    npuResult.npuUtilization > 30.0f,
                    "NPU utilization should be significant when enabled"
                )
            } else {
                println("NPU initialization failed - testing CPU fallback")
                val fallbackResult = miningEngine.fallbackToCPU()
                assertTrue(fallbackResult, "CPU fallback should succeed when NPU fails")
            }
        } else {
            // Test that mining works without NPU
            val cpuOnlyResult = benchmarkIntensity(MiningIntensity.MEDIUM, duration = 30.seconds, enableNPU = false)
            println("CPU-only Hash Rate: ${String.format("%.1f", cpuOnlyResult.hashRate)} H/s")
            
            assertTrue(cpuOnlyResult.hashRate > 0, "CPU-only mining should work on devices without NPU")
        }
    }
    
    @Test
    fun `validate power management policies`() = runTest(timeout = 1.minutes) {
        println("\nPower Management Validation")
        println("===========================")
        
        val batteryLevel = powerManager.getCurrentBatteryLevel()
        val isCharging = powerManager.isCharging()
        val thermalState = thermalManager.getCurrentThermalState()
        
        println("Battery Level: $batteryLevel%")
        println("Charging: $isCharging")
        println("Thermal State: $thermalState")
        
        // Test power management decisions
        val testConfigs = listOf(
            TestPowerScenario(batteryLevel = 95, isCharging = true, shouldMine = true),
            TestPowerScenario(batteryLevel = 85, isCharging = true, shouldMine = true),
            TestPowerScenario(batteryLevel = 75, isCharging = false, shouldMine = false),
            TestPowerScenario(batteryLevel = 50, isCharging = false, shouldMine = false)
        )
        
        for (scenario in testConfigs) {
            // Simulate power conditions (for testing purposes)
            val shouldMine = powerManager.shouldStartMining(
                createMockPowerState(scenario.batteryLevel, scenario.isCharging)
            )
            
            assertEquals(
                scenario.shouldMine, 
                shouldMine,
                "Power management decision incorrect for battery: ${scenario.batteryLevel}%, charging: ${scenario.isCharging}"
            )
            
            if (shouldMine) {
                val optimalIntensity = powerManager.determineOptimalIntensity(
                    createMockPowerState(scenario.batteryLevel, scenario.isCharging)
                )
                
                // Higher battery + charging should allow higher intensity
                when {
                    scenario.batteryLevel >= 90 && scenario.isCharging -> {
                        assertTrue(
                            optimalIntensity >= MiningIntensity.MEDIUM,
                            "High battery + charging should allow medium+ intensity"
                        )
                    }
                    scenario.batteryLevel >= 80 && scenario.isCharging -> {
                        assertTrue(
                            optimalIntensity >= MiningIntensity.LIGHT,
                            "Good battery + charging should allow light+ intensity"
                        )
                    }
                }
            }
        }
    }
    
    @Test
    fun `validate cross-device compatibility`() = runTest(timeout = 1.minutes) {
        println("\nCross-Device Compatibility Validation")
        println("=====================================")
        
        val deviceInfo = DeviceInfo(
            manufacturer = Build.MANUFACTURER,
            model = Build.MODEL,
            androidVersion = Build.VERSION.RELEASE,
            apiLevel = Build.VERSION.SDK_INT,
            abi = Build.SUPPORTED_ABIS.firstOrNull() ?: "unknown"
        )
        
        println("Device: ${deviceInfo.manufacturer} ${deviceInfo.model}")
        println("Android: ${deviceInfo.androidVersion} (API ${deviceInfo.apiLevel})")
        println("ABI: ${deviceInfo.abi}")
        
        // Test basic mining engine compatibility
        assertTrue(miningEngine.initializeEngine(), "Mining engine should initialize on all supported devices")
        
        // Test ARM64 optimization availability
        val hasARM64 = deviceInfo.abi.contains("arm64") || deviceInfo.abi.contains("aarch64")
        if (hasARM64) {
            assertTrue(miningEngine.hasARM64Support(), "ARM64 optimizations should be available")
            println("ARM64 optimizations: Available")
        } else {
            println("ARM64 optimizations: Not available (device not ARM64)")
        }
        
        // Test NEON support
        if (hasARM64) {
            assertTrue(miningEngine.hasNEONSupport(), "NEON support should be available on ARM64 devices")
            println("NEON vector operations: Available")
        }
        
        // Test minimum API level requirements
        assertTrue(
            deviceInfo.apiLevel >= 24, // Android 7.0+
            "Device should meet minimum API level requirement (24+)"
        )
        
        // Perform compatibility mining test
        val compatibilityResult = benchmarkIntensity(MiningIntensity.LIGHT, duration = 10.seconds)
        assertTrue(
            compatibilityResult.hashRate > 0,
            "Basic mining should work on all compatible devices"
        )
        
        println("Compatibility test hash rate: ${String.format("%.1f", compatibilityResult.hashRate)} H/s")
    }
    
    // Helper functions
    
    private suspend fun benchmarkIntensity(
        intensity: MiningIntensity,
        duration: kotlin.time.Duration,
        enableNPU: Boolean = true
    ): PerformanceResult {
        val startTime = System.currentTimeMillis()
        val startTemp = thermalManager.getCurrentTemperature()
        var totalHashes = 0L
        var peakTemperature = startTemp
        var totalPowerConsumption = 0.0f
        var npuUtilization = 0.0f
        var sampleCount = 0
        
        // Configure mining
        if (enableNPU && miningEngine.isNPUAvailable()) {
            miningEngine.initializeNPU()
        } else {
            miningEngine.fallbackToCPU()
        }
        
        assertTrue(miningEngine.startMining(intensity), "Mining should start successfully")
        
        // Collect performance data during benchmark
        val monitoringJob = launch {
            while (System.currentTimeMillis() - startTime < duration.inWholeMilliseconds) {
                delay(1000) // Sample every second
                
                val metrics = miningEngine.getPerformanceMetrics()
                totalHashes += metrics.hashesPerSecond.toLong()
                peakTemperature = maxOf(peakTemperature, metrics.temperature)
                totalPowerConsumption += metrics.powerConsumption
                npuUtilization += metrics.npuUtilization
                sampleCount++
            }
        }
        
        monitoringJob.join()
        miningEngine.stopMining()
        
        val avgHashRate = if (sampleCount > 0) totalHashes.toDouble() / sampleCount else 0.0
        val avgPowerConsumption = if (sampleCount > 0) totalPowerConsumption / sampleCount else 0.0f
        val avgNPUUtilization = if (sampleCount > 0) npuUtilization / sampleCount else 0.0f
        
        return PerformanceResult(
            hashRate = avgHashRate,
            powerConsumption = avgPowerConsumption,
            peakTemperature = peakTemperature,
            npuUtilization = avgNPUUtilization,
            duration = (System.currentTimeMillis() - startTime) / 1000.0
        )
    }
    
    private fun validatePerformanceExpectations(
        deviceClass: DeviceClass,
        results: Map<MiningIntensity, PerformanceResult>
    ) {
        println("\nPerformance Validation")
        println("======================")
        
        // Expected performance ranges by device class
        val expectations = when (deviceClass) {
            DeviceClass.FLAGSHIP -> PerformanceExpectations(
                lightHashRate = 40.0..80.0,
                mediumHashRate = 80.0..120.0,
                fullHashRate = 100.0..150.0,
                maxPowerConsumption = 8.0f,
                maxTemperature = 45.0f
            )
            DeviceClass.MIDRANGE -> PerformanceExpectations(
                lightHashRate = 25.0..50.0,
                mediumHashRate = 50.0..80.0,
                fullHashRate = 60.0..100.0,
                maxPowerConsumption = 6.0f,
                maxTemperature = 50.0f
            )
            DeviceClass.BUDGET -> PerformanceExpectations(
                lightHashRate = 15.0..35.0,
                mediumHashRate = 25.0..50.0,
                fullHashRate = 30.0..60.0,
                maxPowerConsumption = 4.0f,
                maxTemperature = 55.0f
            )
        }
        
        // Validate each intensity result
        results[MiningIntensity.LIGHT]?.let { result ->
            assertTrue(
                result.hashRate in expectations.lightHashRate,
                "Light intensity hash rate ${result.hashRate} not in expected range ${expectations.lightHashRate}"
            )
        }
        
        results[MiningIntensity.MEDIUM]?.let { result ->
            assertTrue(
                result.hashRate in expectations.mediumHashRate,
                "Medium intensity hash rate ${result.hashRate} not in expected range ${expectations.mediumHashRate}"
            )
        }
        
        results[MiningIntensity.FULL]?.let { result ->
            assertTrue(
                result.hashRate in expectations.fullHashRate,
                "Full intensity hash rate ${result.hashRate} not in expected range ${expectations.fullHashRate}"
            )
        }
        
        println("✅ Performance validation passed for $deviceClass device")
    }
    
    private fun createMockPowerState(batteryLevel: Int, isCharging: Boolean): PowerState {
        return PowerState(
            batteryLevel = batteryLevel,
            isCharging = isCharging,
            temperature = 35.0f
        )
    }
}

// Supporting data classes
data class PerformanceResult(
    val hashRate: Double,
    val powerConsumption: Float,
    val peakTemperature: Float,
    val npuUtilization: Float,
    val duration: Double
)

data class PerformanceExpectations(
    val lightHashRate: ClosedFloatingPointRange<Double>,
    val mediumHashRate: ClosedFloatingPointRange<Double>,
    val fullHashRate: ClosedFloatingPointRange<Double>,
    val maxPowerConsumption: Float,
    val maxTemperature: Float
)

data class TestPowerScenario(
    val batteryLevel: Int,
    val isCharging: Boolean,
    val shouldMine: Boolean
)

data class DeviceInfo(
    val manufacturer: String,
    val model: String,
    val androidVersion: String,
    val apiLevel: Int,
    val abi: String
)

data class PowerState(
    val batteryLevel: Int,
    val isCharging: Boolean,
    val temperature: Float
)

/**
 * Device classifier to categorize devices into performance classes
 */
class DeviceClassifier(private val context: Context) {
    
    fun classifyDevice(): DeviceClass {
        val specs = getDeviceSpecs()
        
        return when {
            // Flagship: High-end SoCs with 8+ cores, 6+ GB RAM
            specs.coreCount >= 8 && specs.totalMemoryGB >= 6 && isFlagshipSoC(specs.socModel) -> DeviceClass.FLAGSHIP
            
            // Midrange: Modern SoCs with 6+ cores, 4+ GB RAM
            specs.coreCount >= 6 && specs.totalMemoryGB >= 4 -> DeviceClass.MIDRANGE
            
            // Budget: Basic requirements met
            else -> DeviceClass.BUDGET
        }
    }
    
    fun getDeviceSpecs(): DeviceSpecs {
        val coreCount = Runtime.getRuntime().availableProcessors()
        val totalMemory = getTotalMemoryGB()
        val socModel = getSoCModel()
        val hasNPU = detectNPU()
        
        return DeviceSpecs(
            socModel = socModel,
            coreCount = coreCount,
            totalMemoryGB = totalMemory,
            hasNPU = hasNPU
        )
    }
    
    private fun getTotalMemoryGB(): Int {
        val activityManager = context.getSystemService(Context.ACTIVITY_SERVICE) as android.app.ActivityManager
        val memInfo = android.app.ActivityManager.MemoryInfo()
        activityManager.getMemoryInfo(memInfo)
        return (memInfo.totalMem / (1024 * 1024 * 1024)).toInt()
    }
    
    private fun getSoCModel(): String {
        return Build.HARDWARE ?: "Unknown"
    }
    
    private fun detectNPU(): Boolean {
        return if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            try {
                Build.HARDWARE.contains("qcom") || // Qualcomm Hexagon
                Build.HARDWARE.contains("exynos") || // Samsung NPU
                Build.MODEL.contains("Pixel") // Google Tensor
            } catch (e: Exception) {
                false
            }
        } else {
            false
        }
    }
    
    private fun isFlagshipSoC(socModel: String): Boolean {
        val flagshipKeywords = listOf(
            "8 Gen", "8cx", "A17", "A16", "A15", // High-end Snapdragon and Apple
            "9000", "2200", "2100", // High-end Exynos
            "G3", "G2", "G1" // Google Tensor
        )
        
        return flagshipKeywords.any { socModel.contains(it, ignoreCase = true) }
    }
}

data class DeviceSpecs(
    val socModel: String,
    val coreCount: Int,
    val totalMemoryGB: Int,
    val hasNPU: Boolean
) 