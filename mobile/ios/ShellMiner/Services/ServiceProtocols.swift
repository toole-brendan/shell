import Foundation
import Combine

// MARK: - Mining Engine Protocol
protocol MiningEngineProtocol {
    var miningStatsPublisher: AnyPublisher<MiningStats, Never> { get }
    
    func startMining(config: MiningConfig, completion: @escaping (Result<Void, Error>) -> Void)
    func stopMining(completion: @escaping (Result<Void, Error>) -> Void)
    func updateConfig(_ config: MiningConfig, completion: @escaping (Result<Void, Error>) -> Void)
    func configureNPU(enabled: Bool)
    func getCurrentStats() -> MiningStats
}

// MARK: - Power Manager Protocol
protocol PowerManagerProtocol {
    var powerStatePublisher: AnyPublisher<PowerState, Never> { get }
    
    func currentPowerState() -> PowerState
    func canStartMining() -> Bool
    func optimalMiningIntensity() -> MiningIntensity
    func startPowerMonitoring()
    func stopPowerMonitoring()
}

// MARK: - Thermal Manager Protocol
protocol ThermalManagerProtocol {
    var thermalStatePublisher: AnyPublisher<ThermalMonitorState, Never> { get }
    
    func currentThermalState() -> ThermalMonitorState
    func canMineAtIntensity(_ intensity: MiningIntensity) -> Bool
    func startThermalMonitoring()
    func stopThermalMonitoring()
}

// MARK: - Pool Client Protocol
protocol PoolClientProtocol {
    var poolStatsPublisher: AnyPublisher<PoolStats, Never> { get }
    var connectionStatePublisher: AnyPublisher<PoolConnectionState, Never> { get }
    
    func connect(to url: String, completion: @escaping (Result<Void, Error>) -> Void)
    func disconnect()
    func submitShare(_ share: MiningShare, completion: @escaping (Result<Bool, Error>) -> Void)
    func getWork(completion: @escaping (Result<MiningWork, Error>) -> Void)
}

// MARK: - Supporting Data Structures

struct MiningStats: Equatable {
    let totalHashRate: Double
    let randomXHashRate: Double
    let mobileXHashRate: Double
    let sharesSubmitted: Int64
    let blocksFound: Int32
    let npuUtilization: Float
    let currentIntensity: MiningIntensity
    let currentAlgorithm: MiningAlgorithm
    let timestamp: Date
    
    static let empty = MiningStats(
        totalHashRate: 0.0,
        randomXHashRate: 0.0,
        mobileXHashRate: 0.0,
        sharesSubmitted: 0,
        blocksFound: 0,
        npuUtilization: 0.0,
        currentIntensity: .disabled,
        currentAlgorithm: .mobileX,
        timestamp: Date()
    )
}

struct PowerState: Equatable {
    let batteryLevel: Int32
    let isCharging: Bool
    let isPowerSaveMode: Bool
    let thermalState: ProcessInfo.ThermalState
    let timestamp: Date
    
    var canMine: Bool {
        return batteryLevel > 20 && 
               (isCharging || batteryLevel > 80) && 
               thermalState != .critical
    }
}

struct ThermalMonitorState: Equatable {
    let temperature: Float
    let state: ThermalState
    let isThrottling: Bool
    let timestamp: Date
    
    var canMine: Bool {
        return state != .critical && temperature < 50.0
    }
}

enum PoolConnectionState: String, CaseIterable {
    case disconnected = "disconnected"
    case connecting = "connecting"
    case connected = "connected"
    case error = "error"
    
    var displayName: String {
        switch self {
        case .disconnected: return "Disconnected"
        case .connecting: return "Connecting"
        case .connected: return "Connected"
        case .error: return "Connection Error"
        }
    }
}

struct MiningShare {
    let nonce: UInt64
    let extraNonce: UInt32
    let timestamp: Date
    let difficulty: Double
    let hash: Data
    let thermalProof: Data?
}

struct MiningWork {
    let jobId: String
    let blockHeader: Data
    let target: Data
    let difficulty: Double
    let npuChallenge: Data?
    let coreAffinity: [Int32]
}

// MARK: - Error Types
enum MiningError: LocalizedError {
    case engineNotInitialized
    case invalidConfiguration
    case thermalLimitExceeded
    case batteryTooLow
    case npuNotAvailable
    case poolConnectionFailed
    case nativeEngineError(String)
    
    var errorDescription: String? {
        switch self {
        case .engineNotInitialized:
            return "Mining engine not initialized"
        case .invalidConfiguration:
            return "Invalid mining configuration"
        case .thermalLimitExceeded:
            return "Device temperature too high"
        case .batteryTooLow:
            return "Battery level too low for mining"
        case .npuNotAvailable:
            return "Neural Processing Unit not available"
        case .poolConnectionFailed:
            return "Failed to connect to mining pool"
        case .nativeEngineError(let message):
            return "Native engine error: \(message)"
        }
    }
} 