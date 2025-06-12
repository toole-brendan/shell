import SwiftUI
import CoreML
import OSLog

@main
struct ShellMinerApp: App {
    @StateObject private var miningCoordinator = MiningCoordinator()
    
    private let logger = Logger(subsystem: "com.shell.miner", category: "App")
    
    init() {
        // Initialize native mining library
        initializeNativeMining()
        logger.info("Shell Miner app initialized")
    }
    
    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(miningCoordinator)
                .preferredColorScheme(.dark) // Shell Reserve brand theme
        }
    }
    
    private func initializeNativeMining() {
        // Initialize the native C++ mining engine
        // This will be implemented when we create the Objective-C++ bridge
        logger.info("Initializing native mining engine...")
    }
} 