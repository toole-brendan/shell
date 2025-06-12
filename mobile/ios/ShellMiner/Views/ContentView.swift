import SwiftUI

struct ContentView: View {
    @EnvironmentObject var miningCoordinator: MiningCoordinator
    @State private var selectedTab = 0
    
    var body: some View {
        TabView(selection: $selectedTab) {
            MiningDashboardView()
                .tabItem {
                    Image(systemName: "cpu")
                    Text("Mining")
                }
                .tag(0)
            
            WalletView()
                .tabItem {
                    Image(systemName: "bitcoinsign.circle")
                    Text("Wallet")
                }
                .tag(1)
            
            SettingsView()
                .tabItem {
                    Image(systemName: "gearshape")
                    Text("Settings")
                }
                .tag(2)
        }
        .accentColor(Color.shellPrimary)
    }
}

#Preview {
    ContentView()
        .environmentObject(MiningCoordinator())
} 