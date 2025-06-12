import SwiftUI

struct MiningDashboardView: View {
    @EnvironmentObject var coordinator: MiningCoordinator
    @State private var showingSettings = false
    
    var body: some View {
        NavigationView {
            ScrollView {
                LazyVStack(spacing: 16) {
                    // Header with mining status
                    MiningHeaderCard()
                    
                    // Error card if needed
                    if let error = coordinator.miningState.error {
                        ErrorCard(message: error)
                    }
                    
                    // Main mining statistics
                    MiningStatsCard()
                    
                    // Power and thermal status
                    PowerThermalCard()
                    
                    // Mining controls
                    MiningControlsCard()
                    
                    // Performance details
                    PerformanceDetailsCard()
                    
                    // Earnings card
                    EarningsCard()
                }
                .padding(.horizontal, 16)
                .padding(.bottom, 20)
            }
            .background(Color.shellBackground.ignoresSafeArea())
            .navigationTitle("Shell Mining")
            .navigationBarTitleDisplayMode(.large)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Settings") {
                        showingSettings = true
                    }
                    .foregroundColor(.shellSecondary)
                }
            }
            .sheet(isPresented: $showingSettings) {
                SettingsView()
                    .environmentObject(coordinator)
            }
        }
    }
}

// MARK: - Header Card
struct MiningHeaderCard: View {
    @EnvironmentObject var coordinator: MiningCoordinator
    
    var body: some View {
        VStack(spacing: 16) {
            HStack {
                VStack(alignment: .leading, spacing: 4) {
                    Text("Shell Reserve Mining")
                        .font(ShellTypography.headline)
                        .fontWeight(.bold)
                        .foregroundColor(.white)
                    
                    HStack(spacing: 8) {
                        StatusIndicator(isActive: coordinator.miningState.isMining)
                        Text(coordinator.miningState.isMining ? "Mining Active" : "Mining Stopped")
                            .font(ShellTypography.body)
                            .foregroundColor(.secondary)
                    }
                }
                
                Spacer()
                
                Image(systemName: coordinator.miningState.isMining ? "play.fill" : "stop.fill")
                    .font(.title2)
                    .foregroundColor(.shellPrimary)
            }
        }
        .padding(20)
        .shellCard()
        .background(
            coordinator.miningState.isMining 
                ? Color.shellPrimary.opacity(0.1)
                : Color.shellSurface
        )
    }
}

// MARK: - Status Indicator
struct StatusIndicator: View {
    let isActive: Bool
    
    var body: some View {
        Circle()
            .fill(isActive ? Color.shellSuccess : Color.gray)
            .frame(width: 12, height: 12)
    }
}

// MARK: - Error Card
struct ErrorCard: View {
    let message: String
    
    var body: some View {
        HStack(spacing: 12) {
            Image(systemName: "exclamationmark.triangle.fill")
                .foregroundColor(.shellError)
                .font(.title3)
            
            Text(message)
                .font(ShellTypography.body)
                .foregroundColor(.white)
                .multilineTextAlignment(.leading)
            
            Spacer()
        }
        .padding(16)
        .background(Color.shellError.opacity(0.2))
        .cornerRadius(12)
        .overlay(
            RoundedRectangle(cornerRadius: 12)
                .stroke(Color.shellError.opacity(0.5), lineWidth: 1)
        )
    }
}

// MARK: - Mining Stats Card
struct MiningStatsCard: View {
    @EnvironmentObject var coordinator: MiningCoordinator
    
    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text("Mining Statistics")
                .font(ShellTypography.title)
                .fontWeight(.semibold)
                .foregroundColor(.white)
            
            // Hash rate display
            HashRateDisplay()
            
            // Share and block stats
            HStack {
                Spacer()
                StatItem(
                    label: "Shares",
                    value: "\(coordinator.miningState.sharesSubmitted)",
                    icon: "square.and.arrow.up"
                )
                Spacer()
                StatItem(
                    label: "Blocks",
                    value: "\(coordinator.miningState.blocksFound)",
                    icon: "cube"
                )
                Spacer()
                StatItem(
                    label: "NPU",
                    value: "\(Int(coordinator.miningState.npuUtilization * 100))%",
                    icon: "brain"
                )
                Spacer()
            }
        }
        .padding(20)
        .shellCard()
    }
}

// MARK: - Hash Rate Display
struct HashRateDisplay: View {
    @EnvironmentObject var coordinator: MiningCoordinator
    
    var body: some View {
        VStack(spacing: 8) {
            // Total hash rate
            HStack {
                Text("Total Hash Rate")
                    .font(ShellTypography.body)
                    .foregroundColor(.secondary)
                
                Spacer()
                
                Text(coordinator.miningState.hashRate.formatHashRate())
                    .font(ShellTypography.headline)
                    .fontWeight(.bold)
                    .foregroundColor(.shellPrimary)
            }
            
            // Algorithm breakdown
            if coordinator.miningState.randomXHashRate > 0 || coordinator.miningState.mobileXHashRate > 0 {
                HStack {
                    Text("RandomX: \(coordinator.miningState.randomXHashRate.formatHashRate())")
                        .font(ShellTypography.small)
                        .foregroundColor(.secondary)
                    
                    Spacer()
                    
                    Text("MobileX: \(coordinator.miningState.mobileXHashRate.formatHashRate())")
                        .font(ShellTypography.small)
                        .foregroundColor(.secondary)
                }
            }
        }
    }
}

// MARK: - Stat Item
struct StatItem: View {
    let label: String
    let value: String
    let icon: String
    
    var body: some View {
        VStack(spacing: 4) {
            Image(systemName: icon)
                .font(.title3)
                .foregroundColor(.shellPrimary)
            
            Text(value)
                .font(ShellTypography.title)
                .fontWeight(.bold)
                .foregroundColor(.white)
            
            Text(label)
                .font(ShellTypography.small)
                .foregroundColor(.secondary)
        }
    }
}

// MARK: - Power & Thermal Card
struct PowerThermalCard: View {
    @EnvironmentObject var coordinator: MiningCoordinator
    
    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text("Device Status")
                .font(ShellTypography.title)
                .fontWeight(.semibold)
                .foregroundColor(.white)
            
            HStack {
                Spacer()
                
                // Battery status
                DeviceStatusItem(
                    icon: coordinator.miningState.isCharging ? "battery.100.bolt" : "battery.75",
                    label: "Battery",
                    value: "\(coordinator.miningState.batteryLevel)%",
                    color: getBatteryColor()
                )
                
                Spacer()
                
                // Temperature status
                DeviceStatusItem(
                    icon: "thermometer",
                    label: "Temperature",
                    value: coordinator.miningState.temperature.formatTemperature(),
                    color: coordinator.miningState.thermalState.color
                )
                
                Spacer()
                
                // Mining intensity
                DeviceStatusItem(
                    icon: "speedometer",
                    label: "Intensity",
                    value: coordinator.miningState.intensity.displayName,
                    color: .shellPrimary
                )
                
                Spacer()
            }
        }
        .padding(20)
        .shellCard()
    }
    
    private func getBatteryColor() -> Color {
        let level = coordinator.miningState.batteryLevel
        if coordinator.miningState.isCharging {
            return .batteryCharging
        } else if level > 50 {
            return .batteryGood
        } else if level > 20 {
            return .batteryLow
        } else {
            return .batteryCritical
        }
    }
}

// MARK: - Device Status Item
struct DeviceStatusItem: View {
    let icon: String
    let label: String
    let value: String
    let color: Color
    
    var body: some View {
        VStack(spacing: 4) {
            Image(systemName: icon)
                .font(.title2)
                .foregroundColor(color)
            
            Text(value)
                .font(ShellTypography.title)
                .fontWeight(.bold)
                .foregroundColor(color)
            
            Text(label)
                .font(ShellTypography.small)
                .foregroundColor(.secondary)
        }
    }
}

// MARK: - Mining Controls Card
struct MiningControlsCard: View {
    @EnvironmentObject var coordinator: MiningCoordinator
    
    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text("Mining Controls")
                .font(ShellTypography.title)
                .fontWeight(.semibold)
                .foregroundColor(.white)
            
            // Main toggle button
            Button(action: {
                coordinator.toggleMining()
            }) {
                HStack {
                    Image(systemName: coordinator.miningState.isMining ? "stop.fill" : "play.fill")
                        .font(.title3)
                    
                    Text(coordinator.miningState.isMining ? "Stop Mining" : "Start Mining")
                        .font(ShellTypography.title)
                        .fontWeight(.semibold)
                }
                .frame(maxWidth: .infinity)
                .frame(height: 56)
                .background(
                    coordinator.miningState.isMining 
                        ? Color.shellError 
                        : Color.shellPrimary
                )
                .foregroundColor(.white)
                .cornerRadius(12)
            }
            
            // Intensity selection
            Text("Mining Intensity")
                .font(ShellTypography.body)
                .foregroundColor(.secondary)
            
            HStack(spacing: 8) {
                ForEach(MiningIntensity.allCases.filter { $0 != .disabled }, id: \.id) { intensity in
                    IntensityChip(
                        intensity: intensity,
                        isSelected: coordinator.miningState.intensity == intensity
                    ) {
                        coordinator.adjustIntensity(intensity)
                    }
                }
            }
        }
        .padding(20)
        .shellCard()
    }
}

// MARK: - Intensity Chip
struct IntensityChip: View {
    let intensity: MiningIntensity
    let isSelected: Bool
    let action: () -> Void
    
    var body: some View {
        Button(action: action) {
            Text(intensity.displayName)
                .font(ShellTypography.caption)
                .fontWeight(.medium)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 8)
                .background(
                    isSelected 
                        ? Color.shellPrimary 
                        : Color.shellSurface.opacity(0.5)
                )
                .foregroundColor(isSelected ? .white : .secondary)
                .cornerRadius(8)
                .overlay(
                    RoundedRectangle(cornerRadius: 8)
                        .stroke(
                            isSelected ? Color.shellPrimary : Color.gray.opacity(0.3),
                            lineWidth: 1
                        )
                )
        }
    }
}

// MARK: - Performance Details Card
struct PerformanceDetailsCard: View {
    @EnvironmentObject var coordinator: MiningCoordinator
    
    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Performance Details")
                .font(ShellTypography.title)
                .fontWeight(.semibold)
                .foregroundColor(.white)
            
            PerformanceRow(label: "Algorithm", value: coordinator.miningState.algorithm.displayName)
            PerformanceRow(label: "NPU Utilization", value: "\(Int(coordinator.miningState.npuUtilization * 100))%")
            PerformanceRow(label: "Thermal Throttling", value: coordinator.miningState.thermalThrottling ? "Active" : "None")
            PerformanceRow(label: "Power State", value: coordinator.miningState.isCharging ? "Charging" : "Battery")
        }
        .padding(20)
        .shellCard()
    }
}

// MARK: - Performance Row
struct PerformanceRow: View {
    let label: String
    let value: String
    
    var body: some View {
        HStack {
            Text(label)
                .font(ShellTypography.body)
                .foregroundColor(.secondary)
            
            Spacer()
            
            Text(value)
                .font(ShellTypography.body)
                .fontWeight(.medium)
                .foregroundColor(.white)
        }
    }
}

// MARK: - Earnings Card
struct EarningsCard: View {
    @EnvironmentObject var coordinator: MiningCoordinator
    
    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Earnings")
                .font(ShellTypography.title)
                .fontWeight(.semibold)
                .foregroundColor(.white)
            
            HStack {
                VStack(alignment: .leading, spacing: 4) {
                    Text("Current Session")
                        .font(ShellTypography.caption)
                        .foregroundColor(.secondary)
                    
                    Text(String(format: "%.6f XSL", coordinator.miningState.estimatedEarnings))
                        .font(ShellTypography.title)
                        .fontWeight(.bold)
                        .foregroundColor(.shellSecondary)
                }
                
                Spacer()
                
                VStack(alignment: .trailing, spacing: 4) {
                    Text("Daily Projected")
                        .font(ShellTypography.caption)
                        .foregroundColor(.secondary)
                    
                    Text(String(format: "%.6f XSL", coordinator.miningState.projectedDailyEarnings))
                        .font(ShellTypography.title)
                        .fontWeight(.bold)
                        .foregroundColor(.shellSecondary)
                }
            }
        }
        .padding(20)
        .shellCard()
    }
}

#Preview {
    MiningDashboardView()
        .environmentObject(MiningCoordinator())
        .preferredColorScheme(.dark)
} 