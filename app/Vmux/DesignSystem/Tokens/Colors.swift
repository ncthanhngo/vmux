import SwiftUI
import AppKit

/// Semantic color tokens translated from the locked mockup
/// (`vmux-ui-mockup-apple.html`). This is the ONLY file permitted to contain
/// raw hex / opacity literals — feature code must reference `Theme` instead.
///
/// Every token resolves automatically for light/dark via a dynamic `NSColor`,
/// so switching the macOS appearance live recolors the whole app with no
/// relaunch. `accent` follows the user's System Settings accent color.
enum Theme {
    // Surfaces
    static var window = dyn(dark: hex(0x1E1E20), light: hex(0xECECEE))
    static var content = dyn(dark: hex(0x1C1C1E), light: hex(0xFFFFFF))
    /// Opaque sidebar fallback (used where vibrancy is unavailable).
    static var sidebarSolid = dyn(dark: hex(0x28282B), light: hex(0xF3F3F5))
    static var card = dyn(dark: white(0.05), light: black(0.025))
    static var cardHover = dyn(dark: white(0.08), light: black(0.05))
    static var codeBg = dyn(dark: hex(0x161618), light: hex(0x1C1C1E))

    // Hairlines
    static var separator = dyn(dark: white(0.09), light: black(0.08))
    static var separatorStrong = dyn(dark: white(0.14), light: black(0.13))

    // Text
    static var label = dyn(dark: white(0.92), light: black(0.90))
    static var label2 = dyn(dark: white(0.55), light: black(0.50))
    static var label3 = dyn(dark: white(0.32), light: black(0.28))

    // Accent — follows the system accent color automatically.
    static var accent = Color(nsColor: .controlAccentColor)
    /// Foreground placed on top of an accent fill (always white per mockup).
    static var onAccent = Color.white

    // System semantic colors
    static var green = dyn(dark: hex(0x30D158), light: hex(0x34C759))
    static var red = dyn(dark: hex(0xFF453A), light: hex(0xFF3B30))
    static var orange = dyn(dark: hex(0xFF9F0A), light: hex(0xFF9500))
    static var yellow = dyn(dark: hex(0xFFD60A), light: hex(0xE6B800))
    static var purple = dyn(dark: hex(0xBF5AF2), light: hex(0xAF52DE))
    static var teal = dyn(dark: hex(0x64D2FF), light: hex(0x00C7BE))
    static var pink = dyn(dark: hex(0xFF6482), light: hex(0xFF2D55))

    static var shadow = dyn(dark: NSColor(white: 0, alpha: 0.55), light: NSColor(white: 0, alpha: 0.22))
}

// MARK: - Token construction (hex/opacity literals confined to this file)

private func dyn(dark: NSColor, light: NSColor) -> Color {
    Color(nsColor: NSColor(name: nil) { appearance in
        switch appearance.bestMatch(from: [.aqua, .darkAqua]) {
        case .darkAqua: return dark
        default: return light
        }
    })
}

private func hex(_ rgb: UInt32) -> NSColor {
    NSColor(
        srgbRed: Double((rgb >> 16) & 0xFF) / 255,
        green: Double((rgb >> 8) & 0xFF) / 255,
        blue: Double(rgb & 0xFF) / 255,
        alpha: 1
    )
}

private func white(_ alpha: Double) -> NSColor { NSColor(white: 1, alpha: alpha) }
private func black(_ alpha: Double) -> NSColor { NSColor(white: 0, alpha: alpha) }
