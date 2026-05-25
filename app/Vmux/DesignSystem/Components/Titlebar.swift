import SwiftUI

/// 40px unified titlebar/toolbar from the mockup: a leading title + path
/// (baseline-aligned), a flexible spacer, then a trailing toolbar content slot.
/// Vibrancy-backed with a hairline bottom separator.
struct Titlebar<Trailing: View>: View {
    let title: String
    let subtitle: String
    @ViewBuilder var trailing: () -> Trailing

    var body: some View {
        HStack(spacing: 12) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text(title).font(Typo.titleBar).foregroundStyle(Theme.label)
                Text(subtitle).font(Typo.caption).foregroundStyle(Theme.label2)
            }
            Spacer()
            trailing()
        }
        .padding(.horizontal, 14)
        .frame(height: 40)
        .frame(maxWidth: .infinity)
        .background(VibrancyView(.headerView))
        .overlay(alignment: .bottom) {
            Rectangle().fill(Theme.separator).frame(height: 0.5)
        }
    }
}

/// 28×28 toolbar icon button with hover highlight and an optional red dot badge.
struct ToolbarIconButton: View {
    let glyph: String
    var badge: Bool = false
    var action: () -> Void = {}
    @State private var hovering = false

    var body: some View {
        Button(action: action) {
            Text(glyph)
                .font(.system(size: 14))
                .foregroundStyle(hovering ? Theme.label : Theme.label2)
                .frame(width: 28, height: 28)
                .background(RoundedRectangle(cornerRadius: 7, style: .continuous)
                    .fill(hovering ? Theme.cardHover : Color.clear))
                .overlay(alignment: .topTrailing) {
                    if badge {
                        Circle().fill(Theme.red).frame(width: 7, height: 7).offset(x: -3, y: 2)
                    }
                }
        }
        .buttonStyle(.plain)
        .onHover { hovering = $0 }
    }
}

private struct TitlebarPreview: View {
    @State private var seg = "Code"
    var body: some View {
        Titlebar(title: "myapp", subtitle: "~/dev/myapp · main") {
            StatusPill(title: "Watch mode")
            SegmentedControl(options: ["Code", "Replay"], selection: $seg) { $0 }
            ToolbarIconButton(glyph: "⌘K")
            ToolbarIconButton(glyph: "🔔", badge: true)
            ToolbarIconButton(glyph: "⚙︎")
        }
        .frame(width: 900)
        .background(Theme.window)
    }
}

#Preview("Titlebar") { TitlebarPreview() }
