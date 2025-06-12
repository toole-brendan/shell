import SwiftUI

// Shell Reserve Brand Colors
extension Color {
    static let shellPrimary = Color(red: 0.15, green: 0.20, blue: 0.35)      // Deep Navy
    static let shellSecondary = Color(red: 0.85, green: 0.75, blue: 0.25)    // Gold Accent
    static let shellBackground = Color(red: 0.08, green: 0.08, blue: 0.12)   // Dark Background
    static let shellSurface = Color(red: 0.12, green: 0.12, blue: 0.18)      // Dark Surface
    static let shellSuccess = Color(red: 0.25, green: 0.65, blue: 0.35)      // Success Green
    static let shellWarning = Color(red: 0.95, green: 0.65, blue: 0.15)      // Warning Orange
    static let shellError = Color(red: 0.85, green: 0.25, blue: 0.25)        // Error Red
    
    // Thermal state colors
    static let thermalNormal = Color.shellSuccess
    static let thermalModerate = Color.shellWarning
    static let thermalHot = Color.shellError
    
    // Battery colors
    static let batteryCharging = Color.shellSuccess
    static let batteryGood = Color.shellSecondary
    static let batteryLow = Color.shellWarning
    static let batteryCritical = Color.shellError
}

// Typography
struct ShellTypography {
    static let headline = Font.system(size: 24, weight: .bold, design: .default)
    static let title = Font.system(size: 20, weight: .semibold, design: .default)
    static let body = Font.system(size: 16, weight: .regular, design: .default)
    static let caption = Font.system(size: 14, weight: .medium, design: .default)
    static let small = Font.system(size: 12, weight: .regular, design: .default)
}

// Card styling
struct ShellCardStyle: ViewModifier {
    func body(content: Content) -> some View {
        content
            .background(Color.shellSurface)
            .cornerRadius(12)
            .shadow(color: Color.black.opacity(0.3), radius: 4, x: 0, y: 2)
    }
}

extension View {
    func shellCard() -> some View {
        modifier(ShellCardStyle())
    }
} 