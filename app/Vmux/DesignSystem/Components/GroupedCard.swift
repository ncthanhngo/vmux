import SwiftUI

/// Inset grouped card used in the sidebar panels (active agents, MCP servers,
/// session cost). Optional section title above a hairline-bordered card.
struct GroupedCard<Content: View>: View {
    var title: String? = nil
    @ViewBuilder var content: () -> Content

    var body: some View {
        VStack(alignment: .leading, spacing: 7) {
            if let title {
                Text(title.uppercased())
                    .font(Typo.sectionHeader)
                    .foregroundStyle(Theme.label3)
                    .padding(.horizontal, 4)
            }
            VStack(alignment: .leading, spacing: 0) {
                content()
            }
            .padding(.horizontal, 13)
            .padding(.vertical, 11)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(RoundedRectangle.card.fill(Theme.card))
            .overlay(RoundedRectangle.card.strokeBorder(Theme.separator, lineWidth: 0.5))
        }
    }
}

#Preview("GroupedCard") {
    GroupedCard(title: "MCP servers") {
        VStack(alignment: .leading, spacing: 6) {
            Label("Chrome DevTools MCP", systemImage: "circle.fill")
                .font(Typo.caption2).foregroundStyle(Theme.label)
            Label("vmux builtin", systemImage: "circle.fill")
                .font(Typo.caption2).foregroundStyle(Theme.label)
        }
    }
    .frame(width: 260)
    .padding()
    .background(Theme.sidebarSolid)
}
