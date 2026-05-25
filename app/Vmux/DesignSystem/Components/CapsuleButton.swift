import SwiftUI

/// Pill-shaped button matching the mockup's Approve/Deny/Accept actions.
struct CapsuleButton: View {
    enum Style {
        /// Solid accent fill, white text (primary action — Approve / Accept all).
        case filled
        /// Tinted background of a semantic color, colored text (e.g. Deny in red).
        case tinted(Color)
        /// No background, secondary-label text (e.g. Once).
        case plain
    }

    let title: String
    var style: Style = .filled
    var action: () -> Void = {}

    var body: some View {
        Button(action: action) {
            Text(title)
                .font(.system(size: 11, weight: .semibold))
                .padding(.horizontal, 13)
                .padding(.vertical, 6)
                .foregroundStyle(foreground)
                .background(background)
                .clipShape(Capsule())
                .contentShape(Capsule())
        }
        .buttonStyle(.plain)
    }

    private var foreground: Color {
        switch style {
        case .filled: return Theme.onAccent
        case .tinted(let c): return c
        case .plain: return Theme.label2
        }
    }

    @ViewBuilder private var background: some View {
        switch style {
        case .filled: Theme.accent
        case .tinted(let c): c.opacity(0.16)
        case .plain: Color.clear
        }
    }
}

#Preview("CapsuleButton") {
    HStack(spacing: 8) {
        CapsuleButton(title: "Approve", style: .filled)
        CapsuleButton(title: "Deny", style: .tinted(Theme.red))
        CapsuleButton(title: "Once", style: .plain)
    }
    .padding()
}
