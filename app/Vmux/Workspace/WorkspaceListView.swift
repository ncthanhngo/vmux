import SwiftUI

/// Left-sidebar workspace list, assembled from the design system's `SidebarRow`.
struct WorkspaceListView: View {
    @EnvironmentObject var app: AppState

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            Text("Workspaces")
                .font(Typo.sectionHeader)
                .foregroundStyle(Theme.label3)
                .padding(.horizontal, 18)
                .padding(.top, 14)
                .padding(.bottom, 6)

            ScrollView {
                LazyVStack(spacing: 2) {
                    ForEach(app.workspaces) { ws in
                        SidebarRow(
                            name: ws.meta.name,
                            status: status(for: ws),
                            selected: ws.id == app.selectedWorkspaceID
                        ) {
                            metaRows(for: ws)
                        }
                        .onTapGesture { app.selectWorkspace(ws.id) }
                    }
                }
                .padding(.horizontal, 10)
            }

            Button(action: app.addWorkspace) {
                Text("+ New workspace")
                    .font(Typo.caption)
                    .frame(maxWidth: .infinity)
                    .padding(8)
                    .background(RoundedRectangle(cornerRadius: 8, style: .continuous).fill(Theme.card))
                    .overlay(RoundedRectangle(cornerRadius: 8, style: .continuous).strokeBorder(Theme.separator, lineWidth: 0.5))
                    .foregroundStyle(Theme.label2)
            }
            .buttonStyle(.plain)
            .padding(.horizontal, 14)
            .padding(.vertical, 14)
        }
    }

    private func status(for ws: WorkspaceDTO) -> WorkspaceStatus {
        !(app.tabsByWorkspace[ws.id]?.isEmpty ?? true) ? .running : .idle
    }

    @ViewBuilder private func metaRows(for ws: WorkspaceDTO) -> some View {
        let selected = ws.id == app.selectedWorkspaceID
        if ws.git.isRepo, let branch = ws.git.branch {
            HStack(spacing: 6) {
                Text("⎇ \(branch)")
                    .foregroundStyle(selected ? Theme.onAccent.opacity(0.85) : Theme.purple)
                if let count = ws.git.changes?.count, count > 0 {
                    Text("· \(count) mod").foregroundStyle(selected ? Theme.onAccent.opacity(0.7) : Theme.label3)
                }
            }
        }
    }
}
