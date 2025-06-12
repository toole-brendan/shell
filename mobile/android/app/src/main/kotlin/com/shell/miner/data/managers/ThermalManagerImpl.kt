package com.shell.miner.data.managers

import com.shell.miner.domain.ThermalManager
import com.shell.miner.domain.ThermalState
import kotlinx.coroutines.*
import timber.log.Timber
import java.io.File
import java.security.MessageDigest
import java.util.concurrent.atomic.AtomicReference
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class ThermalManagerImpl @Inject constructor() : ThermalManager {

    private val currentTemperature = AtomicReference(30.0f)
    private val thermalHistory = mutableListOf<ThermalReading>()
    private val maxHistorySize = 60 // Keep last 60 readings (1 minute at 1Hz)
    
    private var monitoringJob: Job? = null
    private val thermalZones = listOf(
        "/sys/class/thermal/thermal_zone0/temp",
        "/sys/class/thermal/thermal_zone1/temp",
        "/sys/class/thermal/thermal_zone2/temp",
        "/sys/class/thermal/thermal_zone3/temp",
        "/sys/class/thermal/thermal_zone4/temp"
    )

    init {
        startThermalMonitoring()
    }

    override fun getCurrentTemperature(): Float {
        return currentTemperature.get()
    }

    override fun getThermalState(): ThermalState {
        val temp = getCurrentTemperature()
        return when {
            temp >= 55.0f -> ThermalState.CRITICAL
            temp >= 48.0f -> ThermalState.SERIOUS
            temp >= 42.0f -> ThermalState.FAIR
            else -> ThermalState.NOMINAL
        }
    }

    override fun shouldThrottle(currentTemp: Float, limit: Float): Boolean {
        return currentTemp > limit
    }

    override fun generateThermalProof(): Long {
        val readings = getThermalHistory()
        if (readings.isEmpty()) {
            return 0L
        }

        // Generate proof based on thermal compliance
        return calculateThermalProof(readings)
    }

    private fun startThermalMonitoring() {
        monitoringJob = CoroutineScope(Dispatchers.IO).launch {
            while (isActive) {
                try {
                    val temperature = readThermalZones()
                    currentTemperature.set(temperature)
                    
                    synchronized(thermalHistory) {
                        thermalHistory.add(ThermalReading(
                            temperature = temperature,
                            timestamp = System.currentTimeMillis()
                        ))
                        
                        // Maintain history size
                        if (thermalHistory.size > maxHistorySize) {
                            thermalHistory.removeAt(0)
                        }
                    }
                    
                    delay(1000) // Read every second
                } catch (e: Exception) {
                    Timber.e(e, "Error monitoring thermal zones")
                    delay(5000) // Retry after 5 seconds on error
                }
            }
        }
    }

    private fun readThermalZones(): Float {
        var maxTemp = 30.0f // Default fallback temperature
        var readingsCount = 0
        var totalTemp = 0.0f

        for (zonePath in thermalZones) {
            try {
                val file = File(zonePath)
                if (file.exists() && file.canRead()) {
                    val tempString = file.readText().trim()
                    val temp = tempString.toInt() / 1000.0f // Convert from millidegrees to degrees
                    
                    totalTemp += temp
                    readingsCount++
                    maxTemp = maxOf(maxTemp, temp)
                    
                    Timber.v("Thermal zone $zonePath: ${temp}°C")
                }
            } catch (e: Exception) {
                Timber.w(e, "Failed to read thermal zone: $zonePath")
            }
        }

        // Return average if we have readings, otherwise return sensible default
        return if (readingsCount > 0) {
            totalTemp / readingsCount
        } else {
            Timber.w("No thermal zones readable, using default temperature")
            35.0f // Conservative default
        }
    }

    private fun getThermalHistory(): List<ThermalReading> {
        return synchronized(thermalHistory) {
            thermalHistory.toList()
        }
    }

    private fun calculateThermalProof(readings: List<ThermalReading>): Long {
        if (readings.isEmpty()) return 0L

        try {
            // Create thermal proof based on recent temperature patterns
            val recentReadings = readings.takeLast(10) // Last 10 seconds
            
            // Calculate statistical properties
            val avgTemp = recentReadings.map { it.temperature }.average()
            val maxTemp = recentReadings.maxOf { it.temperature }
            val minTemp = recentReadings.minOf { it.temperature }
            val variance = calculateVariance(recentReadings.map { it.temperature.toDouble() })
            
            // Check thermal compliance
            val isCompliant = recentReadings.all { it.temperature < 50.0f }
            val stabilityScore = if (variance < 2.0) 1.0 else 0.5
            
            // Create proof data
            val proofData = StringBuilder().apply {
                append("THERMAL_PROOF_V1:")
                append("AVG:${String.format("%.2f", avgTemp)}:")
                append("MAX:${String.format("%.2f", maxTemp)}:")
                append("MIN:${String.format("%.2f", minTemp)}:")
                append("VAR:${String.format("%.4f", variance)}:")
                append("COMPLIANT:$isCompliant:")
                append("STABILITY:${String.format("%.2f", stabilityScore)}:")
                append("COUNT:${recentReadings.size}")
            }.toString()

            // Generate hash-based proof
            val messageDigest = MessageDigest.getInstance("SHA-256")
            val hashBytes = messageDigest.digest(proofData.toByteArray())
            
            // Convert first 8 bytes to long for thermal proof
            var proof = 0L
            for (i in 0 until 8) {
                proof = (proof shl 8) or (hashBytes[i].toLong() and 0xFF)
            }
            
            Timber.d("Generated thermal proof: $proof (compliance: $isCompliant)")
            return proof
            
        } catch (e: Exception) {
            Timber.e(e, "Error generating thermal proof")
            return 0L
        }
    }

    private fun calculateVariance(values: List<Double>): Double {
        if (values.size < 2) return 0.0
        
        val mean = values.average()
        val variance = values.map { (it - mean) * (it - mean) }.average()
        return variance
    }

    /**
     * Get thermal statistics for debugging and optimization
     */
    fun getThermalStats(): ThermalStats {
        val history = getThermalHistory()
        
        if (history.isEmpty()) {
            return ThermalStats(
                currentTemp = getCurrentTemperature(),
                avgTemp = getCurrentTemperature(),
                maxTemp = getCurrentTemperature(),
                minTemp = getCurrentTemperature(),
                readingCount = 0,
                thermalState = getThermalState(),
                isStable = true
            )
        }

        val temperatures = history.map { it.temperature }
        val avgTemp = temperatures.average().toFloat()
        val maxTemp = temperatures.maxOrNull() ?: getCurrentTemperature()
        val minTemp = temperatures.minOrNull() ?: getCurrentTemperature()
        
        // Determine stability (variance < 2 degrees)
        val variance = calculateVariance(temperatures.map { it.toDouble() })
        val isStable = variance < 4.0 // 2 degrees squared
        
        return ThermalStats(
            currentTemp = getCurrentTemperature(),
            avgTemp = avgTemp,
            maxTemp = maxTemp,
            minTemp = minTemp,
            readingCount = history.size,
            thermalState = getThermalState(),
            isStable = isStable
        )
    }

    /**
     * Force a thermal reading update (for testing)
     */
    fun updateThermalReading() {
        val temperature = readThermalZones()
        currentTemperature.set(temperature)
    }

    private data class ThermalReading(
        val temperature: Float,
        val timestamp: Long
    )

    data class ThermalStats(
        val currentTemp: Float,
        val avgTemp: Float,
        val maxTemp: Float,
        val minTemp: Float,
        val readingCount: Int,
        val thermalState: ThermalState,
        val isStable: Boolean
    ) {
        fun getTemperatureRange(): Float = maxTemp - minTemp
        
        fun isSafeForMining(): Boolean = 
            thermalState in listOf(ThermalState.NOMINAL, ThermalState.FAIR) && isStable
    }
} 