package com.shell.miner.performance

import android.content.Context
import android.os.Build
import com.shell.miner.data.managers.PowerManagerImpl
import com.shell.miner.data.managers.ThermalManagerImpl
import com.shell.miner.domain.*
import com.shell.miner.nativecode.MiningEngine
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.*
import java.text.SimpleDateFormat
import java.util.*
import kotlin.time.Duration
import kotlin.time.Duration.Companion.seconds
import kotlin.time.Duration.Companion.minutes

/**
 * Comprehensive performance benchmarking tool for mobile mining.
 * Measures and analyzes:
 * - Hash rate performance across intensities
 * - Power consumption and efficiency
 * - Thermal behavior and throttling
 * - NPU utilization and fallback performance
 * - Device-specific optimizations
 */
class PerformanceBenchmark(
    private val context: Context,
    private val miningEngine: MiningEngine,
    private val powerManager: PowerManagerImpl,
    private val thermalManager: ThermalManagerImpl
) {
    
    private val benchmarkScope = CoroutineScope(Dispatchers.Default + SupervisorJob())
    
    /**
     * Run comprehensive performance benchmark suite
     */
    suspend fun runComprehensiveBenchmark(): BenchmarkReport {
        val deviceInfo = collectDeviceInfo()
        val results = mutableListOf<BenchmarkResult>()
        
        println("🚀 Starting Shell Reserve Mobile Mining Benchmark")
        println("================================================")
        println("Device: ${deviceInfo.manufacturer} ${deviceInfo.model}")
        println("Android: ${deviceInfo.androidVersion} (API ${deviceInfo.apiLevel})")
        println("SoC: ${deviceInfo.socModel}")
        println("Cores: ${deviceInfo.coreCount}, RAM: ${deviceInfo.totalMemoryGB}GB")
        println("NPU: ${if (deviceInfo.hasNPU) "Available" else "Not Available"}")
        println()
        
        // Initialize mining engine
        if (!miningEngine.initializeEngine()) {
            throw IllegalStateException("Failed to initialize mining engine")
        }
        
        try {
            // Benchmark 1: Hash rate performance across intensities
            println("📊 Benchmarking hash rate performance...")
            results.addAll(benchmarkHashRatePerformance())
            
            // Benchmark 2: Power efficiency analysis
            println("⚡ Analyzing power efficiency...")
            results.addAll(benchmarkPowerEfficiency())
            
            // Benchmark 3: Thermal behavior and throttling
            println("🌡️ Testing thermal behavior...")
            results.addAll(benchmarkThermalBehavior())
            
            // Benchmark 4: NPU vs CPU performance comparison
            println("🧠 Comparing NPU vs CPU performance...")
            results.addAll(benchmarkNPUPerformance())
            
            // Benchmark 5: ARM64 optimization effectiveness
            println("🏗️ Testing ARM64 optimizations...")
            results.addAll(benchmarkARM64Optimizations())
            
            // Benchmark 6: Heterogeneous core utilization
            println("🔄 Analyzing core utilization...")
            results.addAll(benchmarkCoreUtilization())
            
        } finally {
            miningEngine.stopMining()
        }
        
        val report = BenchmarkReport(
            deviceInfo = deviceInfo,
            timestamp = System.currentTimeMillis(),
            results = results,
            summary = generateSummary(results)
        )
        
        printBenchmarkReport(report)
        return report
    }
    
    private suspend fun benchmarkHashRatePerformance(): List<BenchmarkResult> {
        val results = mutableListOf<BenchmarkResult>()
        val testDuration = 30.seconds
        
        for (intensity in MiningIntensity.values()) {
            if (intensity == MiningIntensity.AUTO) continue
            
            println("  Testing intensity: $intensity")
            
            val metrics = measurePerformance(intensity, testDuration)
            
            results.add(BenchmarkResult(
                testName = "Hash Rate - $intensity",
                category = BenchmarkCategory.HASH_RATE,
                intensity = intensity,
                hashRate = metrics.hashRate,
                powerConsumption = metrics.powerConsumption,
                temperature = metrics.peakTemperature,
                npuUtilization = metrics.npuUtilization,
                coreUtilization = metrics.coreUtilization,
                duration = metrics.duration,
                success = true
            ))
            
            // Cool down between tests
            delay(5.seconds)
        }
        
        return results
    }
    
    private suspend fun benchmarkPowerEfficiency(): List<BenchmarkResult> {
        val results = mutableListOf<BenchmarkResult>()
        val testDuration = 45.seconds
        
        // Test power efficiency at different intensities
        for (intensity in listOf(MiningIntensity.LIGHT, MiningIntensity.MEDIUM, MiningIntensity.FULL)) {
            println("  Measuring power efficiency at $intensity intensity")
            
            val metrics = measurePowerEfficiency(intensity, testDuration)
            
            results.add(BenchmarkResult(
                testName = "Power Efficiency - $intensity",
                category = BenchmarkCategory.POWER_EFFICIENCY,
                intensity = intensity,
                hashRate = metrics.hashRate,
                powerConsumption = metrics.averagePowerConsumption,
                efficiency = metrics.hashRate / metrics.averagePowerConsumption, // H/s per Watt
                temperature = metrics.averageTemperature,
                duration = metrics.duration,
                success = true,
                additionalMetrics = mapOf(
                    "energy_per_hash" to (metrics.averagePowerConsumption / metrics.hashRate).toString(),
                    "thermal_efficiency" to (metrics.hashRate / metrics.peakTemperature).toString()
                )
            ))
            
            delay(10.seconds) // Longer cool down for power tests
        }
        
        return results
    }
    
    private suspend fun benchmarkThermalBehavior(): List<BenchmarkResult> {
        val results = mutableListOf<BenchmarkResult>()
        val testDuration = 90.seconds // Longer test to observe thermal behavior
        
        println("  Running thermal stress test...")
        
        // Start with full intensity and monitor thermal response
        val thermalMetrics = measureThermalBehavior(MiningIntensity.FULL, testDuration)
        
        results.add(BenchmarkResult(
            testName = "Thermal Behavior - Stress Test",
            category = BenchmarkCategory.THERMAL,
            intensity = MiningIntensity.FULL,
            hashRate = thermalMetrics.averageHashRate,
            temperature = thermalMetrics.peakTemperature,
            duration = thermalMetrics.duration,
            success = true,
            additionalMetrics = mapOf(
                "thermal_throttle_events" to thermalMetrics.throttleEvents.toString(),
                "time_to_throttle" to thermalMetrics.timeToThrottle.toString(),
                "temperature_rise_rate" to thermalMetrics.temperatureRiseRate.toString(),
                "steady_state_temp" to thermalMetrics.steadyStateTemperature.toString()
            )
        ))
        
        return results
    }
    
    private suspend fun benchmarkNPUPerformance(): List<BenchmarkResult> {
        val results = mutableListOf<BenchmarkResult>()
        val testDuration = 30.seconds
        
        if (!miningEngine.isNPUAvailable()) {
            println("  NPU not available - skipping NPU benchmark")
            results.add(BenchmarkResult(
                testName = "NPU Performance - Not Available",
                category = BenchmarkCategory.NPU,
                success = false,
                additionalMetrics = mapOf("reason" to "NPU not available on device")
            ))
            return results
        }
        
        // Test NPU performance
        println("  Testing NPU-enabled mining...")
        val npuMetrics = measurePerformance(MiningIntensity.MEDIUM, testDuration, enableNPU = true)
        
        // Test CPU fallback performance
        println("  Testing CPU fallback mining...")
        val cpuMetrics = measurePerformance(MiningIntensity.MEDIUM, testDuration, enableNPU = false)
        
        val npuSpeedup = npuMetrics.hashRate / cpuMetrics.hashRate
        val npuEfficiency = npuMetrics.hashRate / npuMetrics.powerConsumption
        val cpuEfficiency = cpuMetrics.hashRate / cpuMetrics.powerConsumption
        
        results.add(BenchmarkResult(
            testName = "NPU vs CPU Comparison",
            category = BenchmarkCategory.NPU,
            intensity = MiningIntensity.MEDIUM,
            hashRate = npuMetrics.hashRate,
            powerConsumption = npuMetrics.powerConsumption,
            npuUtilization = npuMetrics.npuUtilization,
            duration = npuMetrics.duration,
            success = true,
            additionalMetrics = mapOf(
                "npu_hash_rate" to npuMetrics.hashRate.toString(),
                "cpu_hash_rate" to cpuMetrics.hashRate.toString(),
                "npu_speedup" to String.format("%.2fx", npuSpeedup),
                "npu_efficiency" to String.format("%.1f H/W", npuEfficiency),
                "cpu_efficiency" to String.format("%.1f H/W", cpuEfficiency),
                "npu_power_overhead" to String.format("%.1f%%", ((npuMetrics.powerConsumption - cpuMetrics.powerConsumption) / cpuMetrics.powerConsumption) * 100)
            )
        ))
        
        return results
    }
    
    private suspend fun benchmarkARM64Optimizations(): List<BenchmarkResult> {
        val results = mutableListOf<BenchmarkResult>()
        val testDuration = 30.seconds
        
        if (!miningEngine.hasARM64Support()) {
            println("  ARM64 optimizations not available")
            results.add(BenchmarkResult(
                testName = "ARM64 Optimizations - Not Available",
                category = BenchmarkCategory.OPTIMIZATION,
                success = false
            ))
            return results
        }
        
        // Test with ARM64 optimizations enabled
        println("  Testing ARM64 NEON optimizations...")
        val optimizedMetrics = measurePerformance(MiningIntensity.MEDIUM, testDuration, enableOptimizations = true)
        
        // Test with optimizations disabled (if possible)
        println("  Testing baseline performance...")
        val baselineMetrics = measurePerformance(MiningIntensity.MEDIUM, testDuration, enableOptimizations = false)
        
        val optimizationSpeedup = optimizedMetrics.hashRate / baselineMetrics.hashRate
        
        results.add(BenchmarkResult(
            testName = "ARM64 Optimization Effectiveness",
            category = BenchmarkCategory.OPTIMIZATION,
            intensity = MiningIntensity.MEDIUM,
            hashRate = optimizedMetrics.hashRate,
            duration = optimizedMetrics.duration,
            success = true,
            additionalMetrics = mapOf(
                "optimized_hash_rate" to optimizedMetrics.hashRate.toString(),
                "baseline_hash_rate" to baselineMetrics.hashRate.toString(),
                "optimization_speedup" to String.format("%.2fx", optimizationSpeedup),
                "neon_available" to miningEngine.hasNEONSupport().toString(),
                "sve_available" to miningEngine.hasSVESupport().toString()
            )
        ))
        
        return results
    }
    
    private suspend fun benchmarkCoreUtilization(): List<BenchmarkResult> {
        val results = mutableListOf<BenchmarkResult>()
        val testDuration = 60.seconds
        
        println("  Analyzing heterogeneous core utilization...")
        
        val coreMetrics = measureCoreUtilization(MiningIntensity.FULL, testDuration)
        
        // Analyze big.LITTLE core utilization patterns
        val bigCoreUtilization = coreMetrics.coreUtilization.filterKeys { it < 4 }.values.average()
        val littleCoreUtilization = coreMetrics.coreUtilization.filterKeys { it >= 4 }.values.average()
        
        results.add(BenchmarkResult(
            testName = "Heterogeneous Core Utilization",
            category = BenchmarkCategory.CORE_UTILIZATION,
            intensity = MiningIntensity.FULL,
            hashRate = coreMetrics.hashRate,
            coreUtilization = coreMetrics.coreUtilization,
            duration = coreMetrics.duration,
            success = true,
            additionalMetrics = mapOf(
                "big_core_utilization" to String.format("%.1f%%", bigCoreUtilization),
                "little_core_utilization" to String.format("%.1f%%", littleCoreUtilization),
                "core_balance_ratio" to String.format("%.2f", bigCoreUtilization / littleCoreUtilization),
                "total_cores" to coreMetrics.coreUtilization.size.toString(),
                "effective_parallelism" to coreMetrics.effectiveParallelism.toString()
            )
        ))
        
        return results
    }
    
    private suspend fun measurePerformance(
        intensity: MiningIntensity,
        duration: Duration,
        enableNPU: Boolean = true,
        enableOptimizations: Boolean = true
    ): DetailedPerformanceMetrics {
        val startTime = System.currentTimeMillis()
        val samples = mutableListOf<PerformanceSample>()
        
        // Configure mining engine
        if (enableNPU && miningEngine.isNPUAvailable()) {
            miningEngine.initializeNPU()
        } else {
            miningEngine.fallbackToCPU()
        }
        
        if (enableOptimizations) {
            miningEngine.enableARM64Optimizations()
        } else {
            miningEngine.disableOptimizations()
        }
        
        miningEngine.startMining(intensity)
        
        // Collect samples during mining
        val monitoringJob = launch {
            while (System.currentTimeMillis() - startTime < duration.inWholeMilliseconds) {
                delay(1000) // Sample every second
                
                val metrics = miningEngine.getPerformanceMetrics()
                samples.add(PerformanceSample(
                    timestamp = System.currentTimeMillis(),
                    hashRate = metrics.hashRate,
                    powerConsumption = metrics.powerConsumption,
                    temperature = metrics.temperature,
                    npuUtilization = metrics.npuUtilization,
                    coreUtilization = metrics.coreUtilization.toMap()
                ))
            }
        }
        
        monitoringJob.join()
        miningEngine.stopMining()
        
        return calculateDetailedMetrics(samples)
    }
    
    private suspend fun measurePowerEfficiency(
        intensity: MiningIntensity,
        duration: Duration
    ): PowerEfficiencyMetrics {
        val startTime = System.currentTimeMillis()
        val powerSamples = mutableListOf<Float>()
        val hashRateSamples = mutableListOf<Double>()
        val temperatureSamples = mutableListOf<Float>()
        
        miningEngine.startMining(intensity)
        
        val monitoringJob = launch {
            while (System.currentTimeMillis() - startTime < duration.inWholeMilliseconds) {
                delay(2000) // Sample every 2 seconds for power measurements
                
                val metrics = miningEngine.getPerformanceMetrics()
                powerSamples.add(metrics.powerConsumption)
                hashRateSamples.add(metrics.hashRate)
                temperatureSamples.add(metrics.temperature)
            }
        }
        
        monitoringJob.join()
        miningEngine.stopMining()
        
        return PowerEfficiencyMetrics(
            averagePowerConsumption = powerSamples.average().toFloat(),
            hashRate = hashRateSamples.average(),
            averageTemperature = temperatureSamples.average().toFloat(),
            peakTemperature = temperatureSamples.maxOrNull() ?: 0.0f,
            duration = (System.currentTimeMillis() - startTime) / 1000.0
        )
    }
    
    private suspend fun measureThermalBehavior(
        intensity: MiningIntensity,
        duration: Duration
    ): ThermalBehaviorMetrics {
        val startTime = System.currentTimeMillis()
        val initialTemp = thermalManager.getCurrentTemperature()
        var timeToThrottle = -1.0
        var throttleEvents = 0
        val temperatureSamples = mutableListOf<Float>()
        val hashRateSamples = mutableListOf<Double>()
        
        miningEngine.startMining(intensity)
        
        val monitoringJob = launch {
            var previousTemp = initialTemp
            
            while (System.currentTimeMillis() - startTime < duration.inWholeMilliseconds) {
                delay(1000)
                
                val currentTemp = thermalManager.getCurrentTemperature()
                val metrics = miningEngine.getPerformanceMetrics()
                
                temperatureSamples.add(currentTemp)
                hashRateSamples.add(metrics.hashRate)
                
                // Detect thermal throttling
                if (currentTemp > 45.0f && timeToThrottle < 0) {
                    timeToThrottle = (System.currentTimeMillis() - startTime) / 1000.0
                }
                
                // Count throttle events (temperature spikes followed by reductions)
                if (previousTemp < currentTemp && currentTemp > 47.0f) {
                    throttleEvents++
                }
                
                previousTemp = currentTemp
            }
        }
        
        monitoringJob.join()
        miningEngine.stopMining()
        
        return ThermalBehaviorMetrics(
            peakTemperature = temperatureSamples.maxOrNull() ?: initialTemp,
            averageHashRate = hashRateSamples.average(),
            throttleEvents = throttleEvents,
            timeToThrottle = timeToThrottle,
            temperatureRiseRate = calculateTemperatureRiseRate(temperatureSamples),
            steadyStateTemperature = temperatureSamples.takeLast(10).average().toFloat(),
            duration = (System.currentTimeMillis() - startTime) / 1000.0
        )
    }
    
    private suspend fun measureCoreUtilization(
        intensity: MiningIntensity,
        duration: Duration
    ): CoreUtilizationMetrics {
        val startTime = System.currentTimeMillis()
        val coreUtilizationSamples = mutableMapOf<Int, MutableList<Float>>()
        val hashRateSamples = mutableListOf<Double>()
        
        miningEngine.startMining(intensity)
        
        val monitoringJob = launch {
            while (System.currentTimeMillis() - startTime < duration.inWholeMilliseconds) {
                delay(2000) // Sample every 2 seconds
                
                val metrics = miningEngine.getPerformanceMetrics()
                hashRateSamples.add(metrics.hashRate)
                
                // Collect core utilization data
                metrics.coreUtilization.forEach { (coreId, utilization) ->
                    coreUtilizationSamples.getOrPut(coreId) { mutableListOf() }.add(utilization)
                }
            }
        }
        
        monitoringJob.join()
        miningEngine.stopMining()
        
        // Calculate average utilization per core
        val avgCoreUtilization = coreUtilizationSamples.mapValues { (_, samples) ->
            samples.average().toFloat()
        }
        
        return CoreUtilizationMetrics(
            hashRate = hashRateSamples.average(),
            coreUtilization = avgCoreUtilization,
            effectiveParallelism = calculateEffectiveParallelism(avgCoreUtilization),
            duration = (System.currentTimeMillis() - startTime) / 1000.0
        )
    }
    
    private fun calculateDetailedMetrics(samples: List<PerformanceSample>): DetailedPerformanceMetrics {
        return DetailedPerformanceMetrics(
            hashRate = samples.map { it.hashRate }.average(),
            powerConsumption = samples.map { it.powerConsumption }.average().toFloat(),
            peakTemperature = samples.map { it.temperature }.maxOrNull() ?: 0.0f,
            npuUtilization = samples.map { it.npuUtilization }.average().toFloat(),
            coreUtilization = calculateAverageCoreUtilization(samples),
            duration = samples.size.toDouble()
        )
    }
    
    private fun calculateAverageCoreUtilization(samples: List<PerformanceSample>): Map<Int, Float> {
        val coreUtilization = mutableMapOf<Int, MutableList<Float>>()
        
        samples.forEach { sample ->
            sample.coreUtilization.forEach { (coreId, utilization) ->
                coreUtilization.getOrPut(coreId) { mutableListOf() }.add(utilization)
            }
        }
        
        return coreUtilization.mapValues { (_, values) -> values.average().toFloat() }
    }
    
    private fun calculateTemperatureRiseRate(temperatures: List<Float>): Float {
        if (temperatures.size < 10) return 0.0f
        
        val first10 = temperatures.take(10).average()
        val last10 = temperatures.takeLast(10).average()
        val timeSpan = temperatures.size.toFloat() // seconds
        
        return ((last10 - first10) / timeSpan).toFloat() // °C per second
    }
    
    private fun calculateEffectiveParallelism(coreUtilization: Map<Int, Float>): Float {
        return coreUtilization.values.sum() / 100.0f // Convert percentage to ratio
    }
    
    private fun collectDeviceInfo(): DeviceInfo {
        val activityManager = context.getSystemService(Context.ACTIVITY_SERVICE) as android.app.ActivityManager
        val memInfo = android.app.ActivityManager.MemoryInfo()
        activityManager.getMemoryInfo(memInfo)
        
        return DeviceInfo(
            manufacturer = Build.MANUFACTURER,
            model = Build.MODEL,
            androidVersion = Build.VERSION.RELEASE,
            apiLevel = Build.VERSION.SDK_INT,
            socModel = Build.HARDWARE ?: "Unknown",
            coreCount = Runtime.getRuntime().availableProcessors(),
            totalMemoryGB = (memInfo.totalMem / (1024 * 1024 * 1024)).toInt(),
            hasNPU = miningEngine.isNPUAvailable(),
            abi = Build.SUPPORTED_ABIS.firstOrNull() ?: "unknown"
        )
    }
    
    private fun generateSummary(results: List<BenchmarkResult>): BenchmarkSummary {
        val hashRateResults = results.filter { it.category == BenchmarkCategory.HASH_RATE }
        val bestHashRate = hashRateResults.maxByOrNull { it.hashRate ?: 0.0 }
        val powerResults = results.filter { it.category == BenchmarkCategory.POWER_EFFICIENCY }
        val bestEfficiency = powerResults.maxByOrNull { it.efficiency ?: 0.0 }
        
        return BenchmarkSummary(
            overallScore = calculateOverallScore(results),
            bestHashRate = bestHashRate?.hashRate ?: 0.0,
            bestEfficiency = bestEfficiency?.efficiency ?: 0.0,
            thermalStability = results.find { it.category == BenchmarkCategory.THERMAL }?.let { 
                it.additionalMetrics?.get("thermal_throttle_events")?.toIntOrNull() ?: 0 < 3 
            } ?: false,
            npuAvailable = results.any { it.category == BenchmarkCategory.NPU && it.success },
            recommendedIntensity = determineRecommendedIntensity(results)
        )
    }
    
    private fun calculateOverallScore(results: List<BenchmarkResult>): Float {
        // Simplified scoring algorithm
        val hashRateScore = results.filter { it.category == BenchmarkCategory.HASH_RATE }
            .mapNotNull { it.hashRate }.maxOrNull() ?: 0.0
        
        val efficiencyScore = results.filter { it.category == BenchmarkCategory.POWER_EFFICIENCY }
            .mapNotNull { it.efficiency }.maxOrNull() ?: 0.0
        
        val thermalScore = if (results.any { it.category == BenchmarkCategory.THERMAL && it.success }) 50.0 else 0.0
        
        return ((hashRateScore / 150.0) * 40 + (efficiencyScore / 20.0) * 40 + (thermalScore / 50.0) * 20).toFloat()
    }
    
    private fun determineRecommendedIntensity(results: List<BenchmarkResult>): MiningIntensity {
        val efficiencyResults = results.filter { it.category == BenchmarkCategory.POWER_EFFICIENCY }
        return efficiencyResults.maxByOrNull { it.efficiency ?: 0.0 }?.intensity ?: MiningIntensity.MEDIUM
    }
    
    private fun printBenchmarkReport(report: BenchmarkReport) {
        val dateFormat = SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.getDefault())
        
        println("\n🏆 SHELL RESERVE MOBILE MINING BENCHMARK REPORT")
        println("===============================================")
        println("Device: ${report.deviceInfo.manufacturer} ${report.deviceInfo.model}")
        println("Timestamp: ${dateFormat.format(Date(report.timestamp))}")
        println("Overall Score: ${String.format("%.1f/100", report.summary.overallScore)}")
        println()
        
        println("📈 PERFORMANCE SUMMARY")
        println("----------------------")
        println("Best Hash Rate: ${String.format("%.1f H/s", report.summary.bestHashRate)}")
        println("Best Efficiency: ${String.format("%.1f H/W", report.summary.bestEfficiency)}")
        println("Thermal Stability: ${if (report.summary.thermalStability) "✅ Good" else "⚠️ Throttling detected"}")
        println("NPU Support: ${if (report.summary.npuAvailable) "✅ Available" else "❌ Not available"}")
        println("Recommended Intensity: ${report.summary.recommendedIntensity}")
        println()
        
        println("📊 DETAILED RESULTS")
        println("-------------------")
        report.results.groupBy { it.category }.forEach { (category, categoryResults) ->
            println("$category:")
            categoryResults.forEach { result ->
                println("  ${result.testName}: ${if (result.success) "✅" else "❌"}")
                result.hashRate?.let { println("    Hash Rate: ${String.format("%.1f H/s", it)}") }
                result.efficiency?.let { println("    Efficiency: ${String.format("%.1f H/W", it)}") }
                result.temperature?.let { println("    Temperature: ${String.format("%.1f°C", it)}") }
            }
            println()
        }
    }
}

// Data classes for benchmark metrics and results
data class BenchmarkReport(
    val deviceInfo: DeviceInfo,
    val timestamp: Long,
    val results: List<BenchmarkResult>,
    val summary: BenchmarkSummary
)

data class BenchmarkResult(
    val testName: String,
    val category: BenchmarkCategory,
    val intensity: MiningIntensity? = null,
    val hashRate: Double? = null,
    val powerConsumption: Float? = null,
    val efficiency: Double? = null,
    val temperature: Float? = null,
    val npuUtilization: Float? = null,
    val coreUtilization: Map<Int, Float>? = null,
    val duration: Double? = null,
    val success: Boolean,
    val additionalMetrics: Map<String, String>? = null
)

data class BenchmarkSummary(
    val overallScore: Float,
    val bestHashRate: Double,
    val bestEfficiency: Double,
    val thermalStability: Boolean,
    val npuAvailable: Boolean,
    val recommendedIntensity: MiningIntensity
)

enum class BenchmarkCategory {
    HASH_RATE,
    POWER_EFFICIENCY,
    THERMAL,
    NPU,
    OPTIMIZATION,
    CORE_UTILIZATION
}

data class DeviceInfo(
    val manufacturer: String,
    val model: String,
    val androidVersion: String,
    val apiLevel: Int,
    val socModel: String,
    val coreCount: Int,
    val totalMemoryGB: Int,
    val hasNPU: Boolean,
    val abi: String
)

data class PerformanceSample(
    val timestamp: Long,
    val hashRate: Double,
    val powerConsumption: Float,
    val temperature: Float,
    val npuUtilization: Float,
    val coreUtilization: Map<Int, Float>
)

data class DetailedPerformanceMetrics(
    val hashRate: Double,
    val powerConsumption: Float,
    val peakTemperature: Float,
    val npuUtilization: Float,
    val coreUtilization: Map<Int, Float>,
    val duration: Double
)

data class PowerEfficiencyMetrics(
    val averagePowerConsumption: Float,
    val hashRate: Double,
    val averageTemperature: Float,
    val peakTemperature: Float,
    val duration: Double
)

data class ThermalBehaviorMetrics(
    val peakTemperature: Float,
    val averageHashRate: Double,
    val throttleEvents: Int,
    val timeToThrottle: Double,
    val temperatureRiseRate: Float,
    val steadyStateTemperature: Float,
    val duration: Double
)

data class CoreUtilizationMetrics(
    val hashRate: Double,
    val coreUtilization: Map<Int, Float>,
    val effectiveParallelism: Float,
    val duration: Double
) 