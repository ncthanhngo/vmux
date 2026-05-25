import SwiftUI

/// Floating content pane from the mockup: radius-11 rounded rect, hairline
/// border, soft shadow, and a 32px header (title + optional badge + trailing
/// action slot). An `attention` pane gets an accent ring.
struct CardPane<Header: View, Content: View>: View {
    var attention: Bool = false
    @ViewBuilder var header: () -> Header
    @ViewBuilder var content: () -> Content

    var body: some View {
        VStack(spacing: 0) {
            header()
                .frame(height: 32)
                .padding(.horizontal, 12)
                .overlay(alignment: .bottom) {
                    Rectangle().fill(Theme.separator).frame(height: 0.5)
                }
            content()
                .frame(maxWidth: .infinity, maxHeight: .infinity)
        }
        .background(Theme.content)
        .clipShape(RoundedRectangle.card)
        .overlay(
            RoundedRectangle.card.strokeBorder(
                attention ? Theme.accent : Theme.separator,
                lineWidth: attention ? 2 : 0.5
            )
        )
        .shadow(color: Theme.shadow, radius: 2, y: 1)
    }
}

/// Convenience header matching the mockup's pane-header layout.
struct PaneHeader<Trailing: View>: View {
    let icon: String
    let title: String
    var badge: BadgePill? = nil
    @ViewBuilder var trailing: () -> Trailing

    var body: some View {
        HStack(spacing: 8) {
            Text(icon).font(.system(size: 12))
            Text(title).font(Typo.caption).foregroundStyle(Theme.label).fontWeight(.medium)
            if let badge { badge }
            Spacer()
            trailing()
        }
        .foregroundStyle(Theme.label2)
    }
}

#Preview("CardPane") {
    CardPane(attention: true) {
        PaneHeader(icon: "●", title: "claude", badge: BadgePill("needs input", kind: .attention)) {
            Text("⌘1").font(Typo.caption2).foregroundStyle(Theme.label2)
        }
    } content: {
        Text("pane body").foregroundStyle(Theme.label2)
    }
    .frame(width: 360, height: 200)
    .padding()
    .background(Theme.window)
}
