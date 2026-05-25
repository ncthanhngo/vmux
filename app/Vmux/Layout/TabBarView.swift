import SwiftUI

/// Center-area tab strip: one chip per open terminal tab plus a "+" button that
/// opens a new terminal. (vmux doesn't pick an AI — run claude/codex/etc.
/// yourself inside the terminal.)
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
            Button {
                app.newTab(AgentLauncher.shell)
            } label: {
                Image(systemName: "plus").font(.system(size: 11))
            }
            .buttonStyle(.plain)
            .frame(width: 22, height: 20)
            .foregroundStyle(Theme.label3)
            .disabled(app.selectedWorkspace == nil)
            .help("New terminal")

            Spacer()
        }
        .padding(.horizontal, 8)
        .frame(height: 28)
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
        .padding(.horizontal, 11)
        .frame(height: 22)
        .background(
            RoundedRectangle(cornerRadius: 6, style: .continuous)
                .fill(selected ? Theme.content : (hovering ? Theme.card : Color.clear))
        )
        .contentShape(Rectangle())
        .onTapGesture(perform: onSelect)
        .onHover { hovering = $0 }
    }
}
