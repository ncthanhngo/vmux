import SwiftUI

/// 28px translucent bottom status bar from the mockup. Hosts left items, a
/// flexible spacer, and right items; a monospace chip variant is provided.
struct StatusBar<Content: View>: View {
    @ViewBuilder var content: () -> Content

    var body: some View {
        HStack(spacing: 18) {
            content()
        }
        .font(Typo.caption2)
        .foregroundStyle(Theme.label2)
        .frame(height: 28)
        .padding(.horizontal, 14)
        .frame(maxWidth: .infinity)
        .background(VibrancyView(.headerView))
        .overlay(alignment: .top) {
            Rectangle().fill(Theme.separator).frame(height: 0.5)
        }
    }
}

/// A single status item: glyph + value, e.g. "⎇ main".
struct StatusItem: View {
    let glyph: String
    let value: String
    var accent: Bool = false

    var body: some View {
        HStack(spacing: 5) {
            Text(glyph).foregroundStyle(Theme.label3)
            Text(value).foregroundStyle(accent ? Theme.accent : Theme.label)
        }
        .font(Typo.caption2)
    }
}

/// A monospace chip used for ports (e.g. ":3000").
struct StatusChip: View {
    let text: String
    var color: Color = Theme.green

    var body: some View {
        Text(text)
            .font(.mono(10))
            .foregroundStyle(color)
            .padding(.horizontal, 7)
            .padding(.vertical, 1)
            .background(color.opacity(0.14))
            .clipShape(Capsule())
    }
}

#Preview("StatusBar") {
    StatusBar {
        StatusItem(glyph: "⎇", value: "main")
        Text("3 modified")
        StatusChip(text: ":3000")
        Text("✦ claude waiting for input").foregroundStyle(Theme.accent)
        Spacer()
        Text("v1.0.0")
    }
    .frame(width: 700)
    .background(Theme.window)
}
