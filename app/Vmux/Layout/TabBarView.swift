import SwiftUI

/// Center-area tab strip: one chip per open terminal tab plus a "+" menu to
/// spawn a shell or agent.
struct TabBarView: View {
    @EnvironmentObject var app: AppState

    private var selectedTabID: UUID? {
        guard let ws = app.selectedWorkspaceID else { return nil }
        return app.selectedTabID[ws]
    }

    var body: some View {
        HStack(spacing: 4) {
            ForEach(app.currentTabs) { tab in
                TabChip(
                    title: tab.title,
                    selected: tab.id == selectedTabID,
                    onSelect: { app.selectTab(tab.id) },
                    onClose: { app.closeTab(tab.id) }
                )
            }
            Menu {
                ForEach(AgentLauncher.presets) { agent in
                    Button(agent.title.capitalized) { app.newTab(agent) }
                }
            } label: {
                Image(systemName: "plus").font(.system(size: 12))
            }
            .menuStyle(.borderlessButton)
            .menuIndicator(.hidden)
            .fixedSize()
            .frame(width: 26, height: 26)
            .foregroundStyle(Theme.label3)
            .disabled(app.selectedWorkspace == nil)

            Spacer()
        }
        .padding(.horizontal, 8)
        .frame(height: 38)
        .background(Theme.window)
        .overlay(alignment: .bottom) {
            Rectangle().fill(Theme.separator).frame(height: 0.5)
        }
    }
}

private struct TabChip: View {
    let title: String
    let selected: Bool
    let onSelect: () -> Void
    let onClose: () -> Void
    @State private var hovering = false

    var body: some View {
        HStack(spacing: 6) {
            Text(title).font(Typo.caption).lineLimit(1)
            if hovering || selected {
                Button(action: onClose) {
                    Image(systemName: "xmark").font(.system(size: 8, weight: .bold))
                }
                .buttonStyle(.plain)
            }
        }
        .foregroundStyle(selected ? Theme.label : Theme.label2)
        .padding(.horizontal, 13)
        .frame(height: 30)
        .background(
            RoundedRectangle(cornerRadius: 8, style: .continuous)
                .fill(selected ? Theme.content : (hovering ? Theme.card : Color.clear))
        )
        .contentShape(Rectangle())
        .onTapGesture(perform: onSelect)
        .onHover { hovering = $0 }
    }
}
