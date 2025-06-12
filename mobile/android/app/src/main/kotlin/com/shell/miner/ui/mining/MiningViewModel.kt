package com.shell.miner.ui.mining

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.shell.miner.domain.*
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.launch
import timber.log.Timber
import javax.inject.Inject

@HiltViewModel
class MiningViewModel @Inject constructor(
    private val miningRepository: MiningRepository,
    private val powerManager: PowerManager,
    private val thermalManager: ThermalManager
) : ViewModel() {

    private val _uiState = MutableStateFlow(MiningUiState())
    val uiState: StateFlow<MiningUiState> = _uiState.asStateFlow()

    private val _config = MutableStateFlow(MiningConfig())
    val config: StateFlow<MiningConfig> = _config.asStateFlow()

    init {
        // Start observing mining state
        observeMiningState()
        observePowerState()
    }

    fun toggleMining() {
        viewModelScope.launch {
            try {
                val currentState = _uiState.value
                if (currentState.isMining) {
                    stopMining()
                } else {
                    startMining()
                }
            } catch (e: Exception) {
                Timber.e(e, "Error toggling mining")
                updateError("Failed to toggle mining: ${e.message}")
            }
        }
    }

    fun adjustIntensity(intensity: MiningIntensity) {
        viewModelScope.launch {
            try {
                val currentConfig = _config.value
                val newConfig = currentConfig.copy(intensity = intensity)
                _config.value = newConfig

                // If mining is active, update the running intensity
                if (_uiState.value.isMining) {
                    val result = miningRepository.updateIntensity(intensity)
                    if (result.isFailure) {
                        Timber.e("Failed to update mining intensity")
                        updateError("Failed to adjust intensity: ${result.exceptionOrNull()?.message}")
                    } else {
                        Timber.i("Mining intensity updated to: $intensity")
                    }
                }
            } catch (e: Exception) {
                Timber.e(e, "Error adjusting intensity")
                updateError("Failed to adjust intensity: ${e.message}")
            }
        }
    }

    fun updateSettings(newConfig: MiningConfig) {
        viewModelScope.launch {
            try {
                _config.value = newConfig
                Timber.i("Mining configuration updated")
                
                // If mining is active and critical settings changed, restart mining
                if (_uiState.value.isMining) {
                    val currentConfig = _config.value
                    if (shouldRestartMining(currentConfig, newConfig)) {
                        Timber.i("Restarting mining due to configuration changes")
                        stopMining()
                        startMining()
                    }
                }
            } catch (e: Exception) {
                Timber.e(e, "Error updating settings")
                updateError("Failed to update settings: ${e.message}")
            }
        }
    }

    private suspend fun startMining() {
        try {
            val config = _config.value
            
            // Check if mining is allowed
            if (!powerManager.shouldStartMining(config)) {
                updateError("Mining not allowed: check power and thermal conditions")
                return
            }

            // Determine optimal intensity
            val optimalIntensity = powerManager.determineOptimalIntensity(config)
            if (optimalIntensity == MiningIntensity.DISABLED) {
                updateError("Mining disabled: power conditions not suitable")
                return
            }

            // Update config with optimal intensity if different
            val miningConfig = if (config.intensity != optimalIntensity) {
                config.copy(intensity = optimalIntensity).also {
                    _config.value = it
                }
            } else {
                config
            }

            // Start mining
            val result = miningRepository.startMining(miningConfig)
            if (result.isFailure) {
                updateError("Failed to start mining: ${result.exceptionOrNull()?.message}")
            } else {
                Timber.i("Mining started successfully")
                clearError()
            }
        } catch (e: Exception) {
            Timber.e(e, "Error starting mining")
            updateError("Failed to start mining: ${e.message}")
        }
    }

    private suspend fun stopMining() {
        try {
            val result = miningRepository.stopMining()
            if (result.isFailure) {
                updateError("Failed to stop mining: ${result.exceptionOrNull()?.message}")
            } else {
                Timber.i("Mining stopped successfully")
                clearError()
            }
        } catch (e: Exception) {
            Timber.e(e, "Error stopping mining")
            updateError("Failed to stop mining: ${e.message}")
        }
    }

    private fun observeMiningState() {
        viewModelScope.launch {
            miningRepository.getMiningState().collect { miningState ->
                _uiState.value = _uiState.value.copy(
                    isMining = miningState.isMining,
                    hashRate = miningState.hashRate,
                    randomXHashRate = miningState.randomXHashRate,
                    mobileXHashRate = miningState.mobileXHashRate,
                    sharesSubmitted = miningState.sharesSubmitted,
                    blocksFound = miningState.blocksFound,
                    temperature = miningState.temperature,
                    batteryLevel = miningState.batteryLevel,
                    isCharging = miningState.isCharging,
                    estimatedEarnings = miningState.estimatedEarnings,
                    projectedDailyEarnings = miningState.projectedDailyEarnings,
                    intensity = miningState.intensity,
                    algorithm = miningState.algorithm,
                    npuUtilization = miningState.npuUtilization,
                    thermalThrottling = miningState.thermalThrottling
                )
            }
        }
    }

    private fun observePowerState() {
        viewModelScope.launch {
            miningRepository.getPowerState().collect { powerState ->
                // Auto-stop mining if power conditions become unsuitable
                if (_uiState.value.isMining && powerManager.shouldStopMining(powerState)) {
                    Timber.w("Auto-stopping mining due to power conditions")
                    stopMining()
                }

                // Update power-related UI state
                _uiState.value = _uiState.value.copy(
                    canMine = powerState.canMine,
                    thermalState = powerState.thermalState
                )
            }
        }
    }

    private fun updateError(message: String) {
        _uiState.value = _uiState.value.copy(error = message)
    }

    private fun clearError() {
        _uiState.value = _uiState.value.copy(error = null)
    }

    private fun shouldRestartMining(oldConfig: MiningConfig, newConfig: MiningConfig): Boolean {
        return oldConfig.algorithm != newConfig.algorithm ||
               oldConfig.poolUrl != newConfig.poolUrl ||
               oldConfig.npuEnabled != newConfig.npuEnabled ||
               oldConfig.thermalLimit != newConfig.thermalLimit
    }

    // Additional helper functions for UI
    fun getFormattedHashRate(): String {
        val hashRate = _uiState.value.hashRate
        return when {
            hashRate >= 1_000_000 -> String.format("%.2f MH/s", hashRate / 1_000_000)
            hashRate >= 1_000 -> String.format("%.2f KH/s", hashRate / 1_000)
            else -> String.format("%.2f H/s", hashRate)
        }
    }

    fun getFormattedEarnings(): String {
        val earnings = _uiState.value.estimatedEarnings
        return String.format("%.6f SHELL", earnings)
    }

    fun getFormattedDailyEarnings(): String {
        val dailyEarnings = _uiState.value.projectedDailyEarnings
        return String.format("%.6f SHELL/day", dailyEarnings)
    }

    fun getThermalStatusText(): String {
        return when (_uiState.value.thermalState) {
            ThermalState.NOMINAL -> "Normal"
            ThermalState.FAIR -> "Warm"
            ThermalState.SERIOUS -> "Hot"
            ThermalState.CRITICAL -> "Critical"
        }
    }

    fun getPowerStatusText(): String {
        val state = _uiState.value
        return when {
            state.isCharging -> "Charging (${state.batteryLevel}%)"
            state.batteryLevel > 50 -> "Battery (${state.batteryLevel}%)"
            state.batteryLevel > 20 -> "Low Battery (${state.batteryLevel}%)"
            else -> "Critical Battery (${state.batteryLevel}%)"
        }
    }

    fun canStartMining(): Boolean {
        val state = _uiState.value
        return !state.isMining && state.canMine && state.error == null
    }

    fun shouldShowWarning(): Boolean {
        val state = _uiState.value
        return state.thermalThrottling || 
               state.thermalState == ThermalState.SERIOUS ||
               (!state.isCharging && state.batteryLevel < 30)
    }

    fun getWarningMessage(): String? {
        val state = _uiState.value
        return when {
            state.thermalState == ThermalState.CRITICAL -> "Device too hot - mining stopped"
            state.thermalThrottling -> "Thermal throttling active"
            state.thermalState == ThermalState.SERIOUS -> "Device running hot"
            !state.isCharging && state.batteryLevel < 20 -> "Low battery - mining may stop"
            !state.isCharging && state.batteryLevel < 30 -> "Consider charging device"
            else -> null
        }
    }
}

/**
 * UI state data class for the mining dashboard
 */
data class MiningUiState(
    val isMining: Boolean = false,
    val canMine: Boolean = false,
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
    val thermalState: ThermalState = ThermalState.NOMINAL,
    val error: String? = null
) 