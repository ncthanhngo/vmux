import SwiftUI

/// In-terminal toast offering to open a detected dev-server port. Never opens
/// automatically — requires an explicit click (avoids exposing internal
/// services unintentionally).
struct PortDetectorBanner: View {
    let port: Int
    let onOpen: () -> Void
    let onDismiss: () -> Void

    var body: some View {
        HStack(spacing: 10) {
            Image(systemName: "globe").foregroundStyle(Theme.accent)
            Text("Open localhost:\(port) in browser?")
                .font(Typo.caption)
                .foregroundStyle(Theme.label)
            Spacer()
            CapsuleButton(title: "Open", style: .filled, action: onOpen)
            CapsuleButton(title: "Dismiss", style: .plain, action: onDismiss)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
        .background(RoundedRectangle.card.fill(Theme.card))
        .overlay(RoundedRectangle.card.strokeBorder(Theme.separator, lineWidth: 0.5))
        .padding(8)
    }
}
