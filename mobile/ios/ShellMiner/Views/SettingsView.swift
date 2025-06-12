import SwiftUI

struct SettingsView: View {
    @EnvironmentObject var coordinator: MiningCoordinator
    @Environment(\.dismiss) private var dismiss
    
    var body: some View {
        NavigationView {
            Form {
                Section("Mining Configuration") {
                    Picker("Mining Intensity", selection: .constant(coordinator.config.intensity)) {
                        ForEach(MiningIntensity.allCases.filter { $0 != .disabled }, id: \.id) { intensity in
                            Text(intensity.displayName).tag(intensity)
                        }
                    }
                    
                    Picker("Algorithm", selection: .constant(coordinator.config.algorithm)) {
                        ForEach(MiningAlgorithm.allCases, id: \.rawValue) { algorithm in
                            Text(algorithm.displayName).tag(algorithm)
                        }
                    }
                    
                    Toggle("Enable NPU", isOn: .constant(coordinator.config.enableNPU))
                }
                
                Section("Device Limits") {
                    HStack {
                        Text("Thermal Limit")
                        Spacer()
                        Text(coordinator.config.thermalLimit.formatTemperature())
                    }
                    
                    HStack {
                        Text("Min Battery Level")
                        Spacer()
                        Text("\(coordinator.config.minBatteryLevel)%")
                    }
                    
                    Toggle("Charging Only Mode", isOn: .constant(coordinator.config.chargingOnlyMode))
                }
                
                Section("Pool Configuration") {
                    HStack {
                        Text("Pool URL")
                        Spacer()
                        Text(coordinator.config.poolURL)
                            .foregroundColor(.secondary)
                            .lineLimit(1)
                    }
                }
                
                Section("Device Information") {
                    if let deviceInfo = coordinator.deviceInfo {
                        HStack {
                            Text("Model")
                            Spacer()
                            Text(deviceInfo.model)
                        }
                        
                        HStack {
                            Text("SoC")
                            Spacer()
                            Text(deviceInfo.soc)
                        }
                        
                        HStack {
                            Text("NPU Support")
                            Spacer()
                            Text(deviceInfo.npuSupported ? "Yes" : "No")
                        }
                        
                        HStack {
                            Text("Core Count")
                            Spacer()
                            Text("\(deviceInfo.coreCount)")
                        }
                        
                        HStack {
                            Text("Device Class")
                            Spacer()
                            Text(deviceInfo.deviceClass.displayName)
                        }
                    }
                }
            }
            .navigationTitle("Settings")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Done") {
                        dismiss()
                    }
                }
            }
        }
        .preferredColorScheme(.dark)
    }
}

#Preview {
    SettingsView()
        .environmentObject(MiningCoordinator())
} 