import SwiftUI

/// Typography scale from the mockup. UI text uses the system font (SF Pro);
/// terminal/code/monospace numerics use SF Mono via `Font.mono`.
enum Typo {
    /// Titlebar / pane title — 13pt semibold.
    static let titleBar = Font.system(size: 13, weight: .semibold)
    /// Default body — 13pt.
    static let body = Font.system(size: 13)
    /// Slightly heavier body used for sidebar workspace names (590 ≈ semibold).
    static let bodyEmphasis = Font.system(size: 13, weight: .semibold)
    /// Secondary caption — 12pt.
    static let caption = Font.system(size: 12)
    /// Small caption / metadata — 11pt.
    static let caption2 = Font.system(size: 11)
    /// Section headers in sidebars — 11pt semibold.
    static let sectionHeader = Font.system(size: 11, weight: .semibold)
    /// Tiny badge / pill text — 10pt semibold.
    static let badge = Font.system(size: 10, weight: .semibold)
}

extension Font {
    /// SF Mono at the given size/weight, falling back to the system monospaced face.
    static func mono(_ size: CGFloat, weight: Font.Weight = .regular) -> Font {
        .system(size: size, weight: weight, design: .monospaced)
    }
}
