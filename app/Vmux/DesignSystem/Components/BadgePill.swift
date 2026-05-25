import SwiftUI

/// Small rounded badge from the mockup: attention (accent), warning (orange),
/// or a count badge (red, used on panel tabs).
struct BadgePill: View {
    enum Kind {
        case attention      // accent tint — "needs input"
        case warning        // orange tint — "2 pending"
        case count          // solid red — panel switcher counts
        case custom(Color)  // tinted with a given color
    }

    let text: String
    let kind: Kind

    init(_ text: String, kind: Kind) {
        self.text = text
        self.kind = kind
    }

    var body: some View {
        Text(text)
            .font(kind.isCount ? .system(size: 9, weight: .bold) : Typo.badge)
            .foregroundStyle(foreground)
            .padding(.horizontal, kind.isCount ? 5 : 8)
            .padding(.vertical, kind.isCount ? 0 : 2)
            .background(background)
            .clipShape(Capsule())
    }

    private var foreground: Color {
        switch kind {
        case .attention: return Theme.accent
        case .warning: return Theme.orange
        case .count: return Theme.onAccent
        case .custom(let c): return c
        }
    }

    @ViewBuilder private var background: some View {
        switch kind {
        case .attention: Theme.accent.opacity(0.16)
        case .warning: Theme.orange.opacity(0.16)
        case .count: Theme.red
        case .custom(let c): c.opacity(0.16)
        }
    }
}

private extension BadgePill.Kind {
    var isCount: Bool { if case .count = self { return true }; return false }
}

#Preview("BadgePill") {
    HStack(spacing: 8) {
        BadgePill("needs input", kind: .attention)
        BadgePill("2 pending", kind: .warning)
        BadgePill("3", kind: .count)
    }
    .padding()
}
