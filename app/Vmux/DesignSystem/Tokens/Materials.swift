import AppKit

/// The vibrancy materials vmux uses, mapped to `NSVisualEffectView.Material`.
/// Sidebars and the titlebar/status bar use blurred materials over content.
enum VibrancyMaterial {
    /// Left/right sidebars.
    case sidebar
    /// Titlebar and status bar.
    case headerView
    /// Under-window translucency.
    case underWindow

    var nsMaterial: NSVisualEffectView.Material {
        switch self {
        case .sidebar: return .sidebar
        case .headerView: return .headerView
        case .underWindow: return .underWindowBackground
        }
    }
}
