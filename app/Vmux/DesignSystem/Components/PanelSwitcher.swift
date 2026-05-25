import SwiftUI

/// One tab in the right-sidebar panel switcher, optionally carrying a count badge.
struct PanelTab: Hashable {
    let title: String
    var badge: Int? = nil
}

/// Segmented panel header (Files / Git / AI / MCP …) from the mockup's right
/// sidebar. Wider than `SegmentedControl` and supports per-tab count badges.
struct PanelSwitcher: View {
    let tabs: [PanelTab]
    @Binding var selection: PanelTab

    var body: some View {
        HStack(spacing: 2) {
            ForEach(tabs, id: \.self) { tab in
                Tab(tab: tab, isOn: tab == selection, onTap: { selection = tab })
            }
        }
        .padding(2)
        .background(RoundedRectangle.control.fill(Theme.card))
        .overlay(RoundedRectangle.control.strokeBorder(Theme.separator, lineWidth: 0.5))
    }

    private struct Tab: View {
        let tab: PanelTab
        let isOn: Bool
        let onTap: () -> Void

        var body: some View {
            HStack(spacing: 3) {
                Text(tab.title)
                if let badge = tab.badge {
                    BadgePill("\(badge)", kind: .count)
                }
            }
            .font(.system(size: 11, weight: .medium))
            .foregroundStyle(isOn ? Theme.label : Theme.label2)
            .frame(maxWidth: .infinity)
            .padding(.vertical, 5)
            .background { highlight }
            .contentShape(Rectangle())
            .onTapGesture(perform: onTap)
        }

        @ViewBuilder private var highlight: some View {
            if isOn {
                RoundedRectangle(cornerRadius: 6, style: .continuous)
                    .fill(Theme.content)
                    .shadow(color: Theme.shadow, radius: 1.5, y: 1)
            }
        }
    }
}

private struct PanelSwitcherPreview: View {
    @State private var sel = PanelTab(title: "AI", badge: 1)
    var body: some View {
        PanelSwitcher(
            tabs: [PanelTab(title: "Files"), PanelTab(title: "Git", badge: 3),
                   PanelTab(title: "AI", badge: 1), PanelTab(title: "MCP")],
            selection: $sel
        )
        .frame(width: 286)
        .padding()
        .background(Theme.sidebarSolid)
    }
}

#Preview("PanelSwitcher") { PanelSwitcherPreview() }
