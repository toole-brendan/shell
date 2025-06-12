import SwiftUI

struct WalletView: View {
    @State private var balance: Double = 0.0
    @State private var transactions: [Transaction] = []
    
    var body: some View {
        NavigationView {
            ScrollView {
                LazyVStack(spacing: 16) {
                    // Balance Card
                    BalanceCard(balance: balance)
                    
                    // Recent Transactions
                    TransactionListCard(transactions: transactions)
                    
                    // Wallet Actions
                    WalletActionsCard()
                }
                .padding(.horizontal, 16)
                .padding(.bottom, 20)
            }
            .background(Color.shellBackground.ignoresSafeArea())
            .navigationTitle("Wallet")
            .navigationBarTitleDisplayMode(.large)
        }
        .onAppear {
            loadWalletData()
        }
    }
    
    private func loadWalletData() {
        // Placeholder data - will be replaced with actual wallet integration
        balance = 12.345678
        transactions = [
            Transaction(
                id: "1",
                type: .mining,
                amount: 0.123456,
                timestamp: Date().addingTimeInterval(-3600),
                confirmations: 6
            ),
            Transaction(
                id: "2", 
                type: .mining,
                amount: 0.098765,
                timestamp: Date().addingTimeInterval(-7200),
                confirmations: 12
            )
        ]
    }
}

// MARK: - Balance Card
struct BalanceCard: View {
    let balance: Double
    
    var body: some View {
        VStack(spacing: 16) {
            HStack {
                VStack(alignment: .leading, spacing: 4) {
                    Text("XSL Balance")
                        .font(ShellTypography.body)
                        .foregroundColor(.secondary)
                    
                    Text(String(format: "%.6f XSL", balance))
                        .font(ShellTypography.headline)
                        .fontWeight(.bold)
                        .foregroundColor(.shellSecondary)
                }
                
                Spacer()
                
                Image(systemName: "bitcoinsign.circle.fill")
                    .font(.title)
                    .foregroundColor(.shellSecondary)
            }
            
            // USD Value (placeholder)
            HStack {
                Text("≈ $\(String(format: "%.2f", balance * 0.25)) USD")
                    .font(ShellTypography.caption)
                    .foregroundColor(.secondary)
                Spacer()
            }
        }
        .padding(20)
        .shellCard()
    }
}

// MARK: - Transaction List Card
struct TransactionListCard: View {
    let transactions: [Transaction]
    
    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text("Recent Transactions")
                .font(ShellTypography.title)
                .fontWeight(.semibold)
                .foregroundColor(.white)
            
            if transactions.isEmpty {
                VStack(spacing: 8) {
                    Image(systemName: "tray")
                        .font(.title2)
                        .foregroundColor(.secondary)
                    
                    Text("No transactions yet")
                        .font(ShellTypography.body)
                        .foregroundColor(.secondary)
                    
                    Text("Start mining to earn XSL")
                        .font(ShellTypography.caption)
                        .foregroundColor(.secondary)
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 20)
            } else {
                ForEach(transactions) { transaction in
                    TransactionRow(transaction: transaction)
                    
                    if transaction.id != transactions.last?.id {
                        Divider()
                            .background(Color.gray.opacity(0.3))
                    }
                }
            }
        }
        .padding(20)
        .shellCard()
    }
}

// MARK: - Transaction Row
struct TransactionRow: View {
    let transaction: Transaction
    
    var body: some View {
        HStack(spacing: 12) {
            // Transaction icon
            Image(systemName: transaction.type.icon)
                .font(.title3)
                .foregroundColor(transaction.type.color)
                .frame(width: 32, height: 32)
                .background(transaction.type.color.opacity(0.2))
                .clipShape(Circle())
            
            // Transaction details
            VStack(alignment: .leading, spacing: 2) {
                Text(transaction.type.displayName)
                    .font(ShellTypography.body)
                    .fontWeight(.medium)
                    .foregroundColor(.white)
                
                Text(transaction.timestamp.formatted(date: .abbreviated, time: .shortened))
                    .font(ShellTypography.small)
                    .foregroundColor(.secondary)
            }
            
            Spacer()
            
            // Amount and confirmations
            VStack(alignment: .trailing, spacing: 2) {
                Text("+\(String(format: "%.6f", transaction.amount)) XSL")
                    .font(ShellTypography.body)
                    .fontWeight(.semibold)
                    .foregroundColor(.shellSuccess)
                
                Text("\(transaction.confirmations) confirmations")
                    .font(ShellTypography.small)
                    .foregroundColor(.secondary)
            }
        }
    }
}

// MARK: - Wallet Actions Card
struct WalletActionsCard: View {
    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text("Wallet Actions")
                .font(ShellTypography.title)
                .fontWeight(.semibold)
                .foregroundColor(.white)
            
            VStack(spacing: 12) {
                WalletActionButton(
                    title: "Receive XSL",
                    icon: "qrcode",
                    color: .shellPrimary
                ) {
                    // Show receive QR code
                }
                
                WalletActionButton(
                    title: "Send XSL",
                    icon: "paperplane",
                    color: .shellSecondary
                ) {
                    // Show send interface
                }
                
                WalletActionButton(
                    title: "View History",
                    icon: "list.bullet",
                    color: .gray
                ) {
                    // Show full transaction history
                }
            }
        }
        .padding(20)
        .shellCard()
    }
}

// MARK: - Wallet Action Button
struct WalletActionButton: View {
    let title: String
    let icon: String
    let color: Color
    let action: () -> Void
    
    var body: some View {
        Button(action: action) {
            HStack(spacing: 12) {
                Image(systemName: icon)
                    .font(.title3)
                    .foregroundColor(color)
                    .frame(width: 24)
                
                Text(title)
                    .font(ShellTypography.body)
                    .fontWeight(.medium)
                    .foregroundColor(.white)
                
                Spacer()
                
                Image(systemName: "chevron.right")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            .padding(.vertical, 12)
            .padding(.horizontal, 16)
            .background(Color.shellSurface.opacity(0.5))
            .cornerRadius(8)
        }
    }
}

// MARK: - Transaction Model
struct Transaction: Identifiable, Equatable {
    let id: String
    let type: TransactionType
    let amount: Double
    let timestamp: Date
    let confirmations: Int32
}

enum TransactionType {
    case mining
    case received
    case sent
    
    var displayName: String {
        switch self {
        case .mining: return "Mining Reward"
        case .received: return "Received"
        case .sent: return "Sent"
        }
    }
    
    var icon: String {
        switch self {
        case .mining: return "cpu"
        case .received: return "arrow.down.circle"
        case .sent: return "arrow.up.circle"
        }
    }
    
    var color: Color {
        switch self {
        case .mining: return .shellSuccess
        case .received: return .shellPrimary
        case .sent: return .shellSecondary
        }
    }
}

#Preview {
    WalletView()
        .preferredColorScheme(.dark)
} 