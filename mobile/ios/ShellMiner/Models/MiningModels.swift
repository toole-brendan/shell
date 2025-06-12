import Foundation
import SwiftUI

// MARK: - Mining State
struct MiningState: Equatable {
    let hashRate: Double
    let randomXHashRate: Double
    let mobileXHashRate: Double
    let sharesSubmitted: Int64
    let blocksFound: Int32
    let temperature: Float
    let batteryLevel: Int32
    let isCharging: Bool
    let estimatedEarnings: Double
    let projectedDailyEarnings: Double
    let npuUtilization: Float
    let isMining: Bool
    let intensity: MiningIntensity
    let algorithm: MiningAlgorithm
    let thermalState: ThermalState
    let thermalThrottling: Bool
    let error: String?
    
    static let initial = MiningState(
        hashRate: 0.0,
        randomXHashRate: 0.0,
        mobileXHashRate: 0.0,
        sharesSubmitted: 0,
        blocksFound: 0,
        temperature: 25.0,
        batteryLevel: 100,
        isCharging: false,
        estimatedEarnings: 0.0,
        projectedDailyEarnings: 0.0,
        npuUtilization: 0.0,
        isMining: false,
        intensity: .disabled,
        algorithm: .mobileX,
        thermalState: .normal,
        thermalThrottling: false,
        error: nil
    )
}

// MARK: - Mining Intensity
enum MiningIntensity: Int, CaseIterable, Identifiable {
    case disabled = 0
    case light = 1
    case medium = 2
    case full = 3
    
    var id: Int { rawValue }
    
    var displayName: String {
        switch self {
        case .disabled: return "Disabled"
        case .light: return "Light"
        case .medium: return "Medium" 
        case .full: return "Full"
        }
    }
    
    var description: String {
        switch self {
        case .disabled: return "Mining disabled"
        case .light: return "1-2 CPU cores, minimal power"
        case .medium: return "3-4 CPU cores, moderate power"
        case .full: return "All cores + NPU, maximum performance"
        }
    }
}

// MARK: - Mining Algorithm
enum MiningAlgorithm: String, CaseIterable {
    case randomX = "RandomX"
    case mobileX = "MobileX"
    case dual = "Dual"
    
    var displayName: String {
        return rawValue
    }
    
    var description: String {
        switch self {
        case .randomX: return "CPU-optimized RandomX algorithm"
        case .mobileX: return "Mobile-optimized with NPU support"
        case .dual: return "Combined RandomX + MobileX mining"
        }
    }
}

// MARK: - Thermal State
enum ThermalState: String, CaseIterable {
    case normal = "normal"
    case moderate = "moderate"
    case hot = "hot"
    case critical = "critical"
    
    var displayName: String {
        switch self {
        case .normal: return "Normal"
        case .moderate: return "Moderate"
        case .hot: return "Hot"
        case .critical: return "Critical"
        }
    }
    
    var color: Color {
        switch self {
        case .normal: return .thermalNormal
        case .moderate: return .thermalModerate
        case .hot, .critical: return .thermalHot
        }
    }
}

// MARK: - Mining Configuration
struct MiningConfig: Codable, Equatable {
    let intensity: MiningIntensity
    let algorithm: MiningAlgorithm
    let enableNPU: Bool
    let thermalLimit: Float
    let minBatteryLevel: Int32
    let chargingOnlyMode: Bool
    let poolURL: String
    let walletAddress: String
    
    static let `default` = MiningConfig(
        intensity: .medium,
        algorithm: .mobileX,
        enableNPU: true,
        thermalLimit: 45.0,
        minBatteryLevel: 80,
        chargingOnlyMode: true,
        poolURL: "stratum+tcp://pool.shellreserve.org:4444",
        walletAddress: ""
    )
}

// MARK: - Device Information
struct DeviceInfo: Equatable {
    let model: String
    let soc: String
    let npuSupported: Bool
    let coreCount: Int32
    let performanceCores: Int32
    let efficiencyCores: Int32
    let maxThermalLimit: Float
    let deviceClass: DeviceClass
    
    enum DeviceClass: String, CaseIterable {
        case budget = "budget"
        case midrange = "midrange"
        case flagship = "flagship"
        
        var displayName: String {
            switch self {
            case .budget: return "Budget Device"
            case .midrange: return "Mid-Range Device" 
            case .flagship: return "Flagship Device"
            }
        }
    }
}

// MARK: - Pool Statistics
struct PoolStats: Equatable {
    let connectedMiners: Int32
    let networkHashRate: Double
    let blockHeight: Int64
    let lastBlockTime: Date
    let difficulty: Double
    let estimatedTimeToBlock: TimeInterval
}

// MARK: - Formatting Helpers
extension Double {
    func formatHashRate() -> String {
        if self >= 1000000 {
            return String(format: "%.1f MH/s", self / 1000000)
        } else if self >= 1000 {
            return String(format: "%.1f kH/s", self / 1000)
        } else {
            return String(format: "%.1f H/s", self)
        }
    }
}

extension Float {
    func formatTemperature() -> String {
        return String(format: "%.1f°C", self)
    }
} 