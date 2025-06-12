import Foundation
import UIKit
import Combine
import OSLog

class PowerManager: PowerManagerProtocol {
    private let powerStateSubject = CurrentValueSubject<PowerState, Never>(
        PowerState(
            batteryLevel: 100,
            isCharging: false,
            isPowerSaveMode: false,
            thermalState: .nominal,
            timestamp: Date()
        )
    )
    private let logger = Logger(subsystem: "com.shell.miner", category: "PowerManager")
    private var monitoringTimer: Timer?
    
    var powerStatePublisher: AnyPublisher<PowerState, Never> {
        powerStateSubject.eraseToAnyPublisher()
    }
    
    init() {
        setupBatteryMonitoring()
        startPowerMonitoring()
    }
    
    deinit {
        stopPowerMonitoring()
    }
    
    // MARK: - PowerManagerProtocol
    
    func currentPowerState() -> PowerState {
        return getCurrentPowerState()
    }
    
    func canStartMining() -> Bool {
        let state = currentPowerState()
        return state.canMine
    }
    
    func optimalMiningIntensity() -> MiningIntensity {
        let state = currentPowerState()
        
        // Determine optimal intensity based on power state
        switch (state.batteryLevel, state.isCharging, state.thermalState) {
        case (95..., true, .nominal):
            return .full
        case (85..., true, .nominal):
            return .medium
        case (80..., true, .nominal):
            return .light
        case (_, true, .fair):
            return .light
        case (_, true, .serious...):
            return .disabled
        case (_, false, _):
            return .disabled  // Don't mine on battery
        default:
            return .disabled
        }
    }
    
    func startPowerMonitoring() {
        guard monitoringTimer == nil else { return }
        
        logger.info("Starting power monitoring")
        
        // Monitor power state every 5 seconds
        monitoringTimer = Timer.scheduledTimer(withTimeInterval: 5.0, repeats: true) { _ in
            self.updatePowerState()
        }
        
        // Initial update
        updatePowerState()
    }
    
    func stopPowerMonitoring() {
        logger.info("Stopping power monitoring")
        monitoringTimer?.invalidate()
        monitoringTimer = nil
    }
    
    // MARK: - Private Methods
    
    private func setupBatteryMonitoring() {
        // Enable battery monitoring
        UIDevice.current.isBatteryMonitoringEnabled = true
        
        // Listen for battery state changes
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(batteryStateDidChange),
            name: UIDevice.batteryStateDidChangeNotification,
            object: nil
        )
        
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(batteryLevelDidChange),
            name: UIDevice.batteryLevelDidChangeNotification,
            object: nil
        )
        
        // Listen for power save mode changes
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(powerSaveModeDidChange),
            name: .NSProcessInfoPowerStateDidChange,
            object: nil
        )
    }
    
    @objc private func batteryStateDidChange() {
        updatePowerState()
    }
    
    @objc private func batteryLevelDidChange() {
        updatePowerState()
    }
    
    @objc private func powerSaveModeDidChange() {
        updatePowerState()
    }
    
    private func updatePowerState() {
        let newState = getCurrentPowerState()
        
        // Only publish if state actually changed
        let currentState = powerStateSubject.value
        if newState != currentState {
            logger.debug("Power state changed: battery=\(newState.batteryLevel)%, charging=\(newState.isCharging), thermal=\(newState.thermalState.rawValue)")
            powerStateSubject.send(newState)
        }
    }
    
    private func getCurrentPowerState() -> PowerState {
        let device = UIDevice.current
        let processInfo = ProcessInfo.processInfo
        
        // Get battery level (0.0 to 1.0, or -1.0 if unknown)
        let batteryLevel = device.batteryLevel >= 0 ? Int32(device.batteryLevel * 100) : 100
        
        // Check charging state
        let isCharging = device.batteryState == .charging || device.batteryState == .full
        
        // Check power save mode
        let isPowerSaveMode = processInfo.isLowPowerModeEnabled
        
        // Get thermal state
        let thermalState = processInfo.thermalState
        
        return PowerState(
            batteryLevel: batteryLevel,
            isCharging: isCharging,
            isPowerSaveMode: isPowerSaveMode,
            thermalState: thermalState,
            timestamp: Date()
        )
    }
}

// MARK: - ProcessInfo.ThermalState Extensions
extension ProcessInfo.ThermalState {
    var displayName: String {
        switch self {
        case .nominal: return "Normal"
        case .fair: return "Fair"
        case .serious: return "Serious"
        case .critical: return "Critical"
        @unknown default: return "Unknown"
        }
    }
}

// MARK: - Future Native Integration Points
// TODO: Integrate with native thermal monitoring
/*
 This implementation uses standard iOS APIs. For enhanced functionality, we can integrate with:
 
 1. IOKit thermal APIs for more detailed temperature readings
 2. Native C++ thermal monitoring (similar to Android implementation)
 3. Custom thermal management for mining-specific scenarios
 4. Apple Silicon specific power management features
 
 The interface will remain the same, but the implementation may call into:
 - ios_thermal_manager.cpp (native thermal monitoring)
 - Apple Silicon specific APIs for P-core/E-core management
 */ 