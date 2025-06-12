import Foundation
import SwiftUI
import Combine
import OSLog

@MainActor
class MiningCoordinator: ObservableObject {
    // MARK: - Published Properties
    @Published var miningState = MiningState.initial
    @Published var deviceInfo: DeviceInfo?
    @Published var poolStats: PoolStats?
    @Published var config = MiningConfig.default
    
    // MARK: - Private Properties
    private var cancellables = Set<AnyCancellable>()
    private let logger = Logger(subsystem: "com.shell.miner", category: "MiningCoordinator")
    
    // Services
    private let miningEngine: MiningEngineProtocol
    private let powerManager: PowerManagerProtocol
    private let thermalManager: ThermalManagerProtocol
    private let poolClient: PoolClientProtocol
    
    // MARK: - Initialization
    init(
        miningEngine: MiningEngineProtocol = MiningEngine(),
        powerManager: PowerManagerProtocol = PowerManager(),
        thermalManager: ThermalManagerProtocol = ThermalManager(),
        poolClient: PoolClientProtocol = PoolClient()
    ) {
        self.miningEngine = miningEngine
        self.powerManager = powerManager
        self.thermalManager = thermalManager
        self.poolClient = poolClient
        
        setupBindings()
        detectDeviceInfo()
    }
    
    // MARK: - Public Interface
    func toggleMining() {
        if miningState.isMining {
            stopMining()
        } else {
            startMining()
        }
    }
    
    func adjustIntensity(_ intensity: MiningIntensity) {
        config = MiningConfig(
            intensity: intensity,
            algorithm: config.algorithm,
            enableNPU: config.enableNPU,
            thermalLimit: config.thermalLimit,
            minBatteryLevel: config.minBatteryLevel,
            chargingOnlyMode: config.chargingOnlyMode,
            poolURL: config.poolURL,
            walletAddress: config.walletAddress
        )
        
        if miningState.isMining {
            restartMiningWithNewConfig()
        }
    }
    
    func updateSettings(_ newConfig: MiningConfig) {
        config = newConfig
        
        if miningState.isMining {
            restartMiningWithNewConfig()
        }
    }
    
    // MARK: - Private Methods
    private func setupBindings() {
        // Monitor power state changes
        powerManager.powerStatePublisher
            .receive(on: DispatchQueue.main)
            .sink { [weak self] powerState in
                self?.handlePowerStateChange(powerState)
            }
            .store(in: &cancellables)
        
        // Monitor thermal state changes
        thermalManager.thermalStatePublisher
            .receive(on: DispatchQueue.main)
            .sink { [weak self] thermalState in
                self?.handleThermalStateChange(thermalState)
            }
            .store(in: &cancellables)
        
        // Monitor mining stats
        miningEngine.miningStatsPublisher
            .receive(on: DispatchQueue.main)
            .sink { [weak self] stats in
                self?.updateMiningStats(stats)
            }
            .store(in: &cancellables)
        
        // Monitor pool stats
        poolClient.poolStatsPublisher
            .receive(on: DispatchQueue.main)
            .sink { [weak self] stats in
                self?.poolStats = stats
            }
            .store(in: &cancellables)
    }
    
    private func startMining() {
        guard canStartMining() else {
            logger.warning("Cannot start mining - preconditions not met")
            return
        }
        
        logger.info("Starting mining with intensity: \(config.intensity.displayName)")
        
        // Configure Core ML for NPU if enabled
        if config.enableNPU {
            miningEngine.configureNPU(enabled: true)
        }
        
        // Start mining with current configuration
        miningEngine.startMining(config: config) { [weak self] result in
            DispatchQueue.main.async {
                switch result {
                case .success:
                    self?.miningState = self?.miningState.with(isMining: true, error: nil) ?? MiningState.initial
                    self?.startMonitoring()
                case .failure(let error):
                    self?.handleMiningError(error)
                }
            }
        }
    }
    
    private func stopMining() {
        logger.info("Stopping mining")
        
        miningEngine.stopMining { [weak self] result in
            DispatchQueue.main.async {
                switch result {
                case .success:
                    self?.miningState = self?.miningState.with(isMining: false) ?? MiningState.initial
                    self?.stopMonitoring()
                case .failure(let error):
                    self?.logger.error("Failed to stop mining: \(error.localizedDescription)")
                }
            }
        }
    }
    
    private func canStartMining() -> Bool {
        let powerState = powerManager.currentPowerState()
        
        // Check battery level
        if powerState.batteryLevel < config.minBatteryLevel {
            updateError("Battery level too low (\(powerState.batteryLevel)%). Minimum: \(config.minBatteryLevel)%")
            return false
        }
        
        // Check charging requirement
        if config.chargingOnlyMode && !powerState.isCharging {
            updateError("Device must be charging to mine")
            return false
        }
        
        // Check thermal state
        let thermalState = thermalManager.currentThermalState()
        if thermalState.temperature > config.thermalLimit {
            updateError("Device temperature too high (\(thermalState.temperature.formatTemperature()))")
            return false
        }
        
        return true
    }
    
    private func restartMiningWithNewConfig() {
        miningEngine.updateConfig(config) { [weak self] result in
            if case .failure(let error) = result {
                self?.handleMiningError(error)
            }
        }
    }
    
    private func startMonitoring() {
        // Start periodic monitoring of device state
        Timer.publish(every: 1.0, on: .main, in: .common)
            .autoconnect()
            .sink { [weak self] _ in
                self?.updateDeviceState()
            }
            .store(in: &cancellables)
    }
    
    private func stopMonitoring() {
        cancellables.removeAll()
        setupBindings() // Re-establish basic bindings
    }
    
    private func updateDeviceState() {
        let powerState = powerManager.currentPowerState()
        let thermalState = thermalManager.currentThermalState()
        
        miningState = miningState.with(
            batteryLevel: powerState.batteryLevel,
            isCharging: powerState.isCharging,
            temperature: thermalState.temperature,
            thermalState: thermalState.state,
            thermalThrottling: thermalState.isThrottling
        )
        
        // Auto-stop if conditions are no longer met
        if miningState.isMining && !canStartMining() {
            stopMining()
        }
    }
    
    private func handlePowerStateChange(_ powerState: PowerState) {
        miningState = miningState.with(
            batteryLevel: powerState.batteryLevel,
            isCharging: powerState.isCharging
        )
        
        if miningState.isMining && !canStartMining() {
            stopMining()
        }
    }
    
    private func handleThermalStateChange(_ thermalState: ThermalMonitorState) {
        miningState = miningState.with(
            temperature: thermalState.temperature,
            thermalState: thermalState.state,
            thermalThrottling: thermalState.isThrottling
        )
        
        if miningState.isMining && thermalState.temperature > config.thermalLimit {
            logger.warning("Thermal limit exceeded, stopping mining")
            stopMining()
        }
    }
    
    private func updateMiningStats(_ stats: MiningStats) {
        miningState = miningState.with(
            hashRate: stats.totalHashRate,
            randomXHashRate: stats.randomXHashRate,
            mobileXHashRate: stats.mobileXHashRate,
            sharesSubmitted: stats.sharesSubmitted,
            blocksFound: stats.blocksFound,
            npuUtilization: stats.npuUtilization,
            intensity: stats.currentIntensity,
            algorithm: stats.currentAlgorithm
        )
    }
    
    private func handleMiningError(_ error: Error) {
        logger.error("Mining error: \(error.localizedDescription)")
        miningState = miningState.with(
            isMining: false,
            error: error.localizedDescription
        )
    }
    
    private func updateError(_ message: String) {
        miningState = miningState.with(error: message)
    }
    
    private func detectDeviceInfo() {
        // This will be implemented with the native bridge
        // For now, use placeholder data
        deviceInfo = DeviceInfo(
            model: "iPhone",
            soc: "Apple Silicon",
            npuSupported: true,
            coreCount: 8,
            performanceCores: 2,
            efficiencyCores: 6,
            maxThermalLimit: 45.0,
            deviceClass: .flagship
        )
    }
}

// MARK: - MiningState Extensions
extension MiningState {
    func with(
        hashRate: Double? = nil,
        randomXHashRate: Double? = nil,
        mobileXHashRate: Double? = nil,
        sharesSubmitted: Int64? = nil,
        blocksFound: Int32? = nil,
        temperature: Float? = nil,
        batteryLevel: Int32? = nil,
        isCharging: Bool? = nil,
        estimatedEarnings: Double? = nil,
        projectedDailyEarnings: Double? = nil,
        npuUtilization: Float? = nil,
        isMining: Bool? = nil,
        intensity: MiningIntensity? = nil,
        algorithm: MiningAlgorithm? = nil,
        thermalState: ThermalState? = nil,
        thermalThrottling: Bool? = nil,
        error: String? = nil
    ) -> MiningState {
        return MiningState(
            hashRate: hashRate ?? self.hashRate,
            randomXHashRate: randomXHashRate ?? self.randomXHashRate,
            mobileXHashRate: mobileXHashRate ?? self.mobileXHashRate,
            sharesSubmitted: sharesSubmitted ?? self.sharesSubmitted,
            blocksFound: blocksFound ?? self.blocksFound,
            temperature: temperature ?? self.temperature,
            batteryLevel: batteryLevel ?? self.batteryLevel,
            isCharging: isCharging ?? self.isCharging,
            estimatedEarnings: estimatedEarnings ?? self.estimatedEarnings,
            projectedDailyEarnings: projectedDailyEarnings ?? self.projectedDailyEarnings,
            npuUtilization: npuUtilization ?? self.npuUtilization,
            isMining: isMining ?? self.isMining,
            intensity: intensity ?? self.intensity,
            algorithm: algorithm ?? self.algorithm,
            thermalState: thermalState ?? self.thermalState,
            thermalThrottling: thermalThrottling ?? self.thermalThrottling,
            error: error ?? self.error
        )
    }
} 