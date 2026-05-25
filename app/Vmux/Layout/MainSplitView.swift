import SwiftUI

/// The v0.1 three-region shell: workspace sidebar | tabs+terminal | file tree,
/// with a toolbar and bottom status bar. Assembled from the design system.
struct MainSplitView: View {
    @EnvironmentObject var app: AppState
    @State private var replaySession: String?

    var body: some View {
        VStack(spacing: 0) {
            HSplitView {
                WorkspaceListView()
                    .frame(minWidth: 200, idealWidth: 224, maxWidth: 320)
                    .background(VibrancyView(.sidebar))

                CenterArea(browser: app.browser)
                    .frame(minWidth: 420)
                    .layoutPriority(1)

                if let ws = app.selectedWorkspace {
                    rightSidebar(ws)
                        .frame(minWidth: 220, idealWidth: 286, maxWidth: 380)
                        .background(VibrancyView(.sidebar))
                }
            }
            statusBar
        }
        .frame(minWidth: 1000, minHeight: 640)
        .toolbar { toolbarContent }
        .navigationTitle(app.selectedWorkspace?.meta.name ?? "vmux")
    }

    private func rightSidebar(_ ws: WorkspaceDTO) -> some View {
        VStack(spacing: 0) {
            ScrollView {
                VStack(alignment: .leading, spacing: 14) {
                    ActivityPanelView(model: app.activity, workspaceID: ws.id)
                    if let session = app.activity.events.last?.sessionId {
                        CapsuleButton(title: "Open Session Replay", style: .plain) {
                            replaySession = session
                        }
                    }
                    BrowserSessionsView(model: app.browser)
                }
                .padding(8)
            }
            .frame(maxHeight: .infinity)
            Divider()
            FileTreeView(workspace: ws)
                .frame(maxHeight: 280)
        }
        .sheet(item: Binding(
            get: { replaySession.map { ReplaySessionItem(id: $0) } },
            set: { if $0 == nil { replaySession = nil } }
        )) { item in
            ReplayView(model: ReplayModel(client: app.client, sessionID: item.id), onClose: { replaySession = nil })
        }
    }

    @ToolbarContentBuilder private var toolbarContent: some ToolbarContent {
        ToolbarItem(placement: .principal) {
            if let ws = app.selectedWorkspace {
                Text(ws.path).font(Typo.caption).foregroundStyle(Theme.label2)
            }
        }
        ToolbarItemGroup(placement: .primaryAction) {
            StatusPill(title: "Watch mode")
            ToolbarIconButton(glyph: "⚙︎")
        }
    }

    private var statusBar: some View {
        StatusBar {
            if let ws = app.selectedWorkspace, ws.git.isRepo {
                StatusItem(glyph: "⎇", value: ws.git.branch ?? "—")
                if let n = ws.git.changes?.count, n > 0 { Text("\(n) modified") }
            }
            Spacer()
            switch app.connection {
            case .connecting: Text("connecting…")
            case .connected: StatusItem(glyph: "🔗", value: "sidecar")
            case .failed(let e): Text("sidecar error: \(e)").foregroundStyle(Theme.red)
            }
            Text("v0.1.0")
        }
    }
}

/// The center column: tab bar over the active terminal (or an empty state),
/// with a port-detected banner when the active session prints a localhost port.
private struct CenterArea: View {
    @EnvironmentObject var app: AppState
    @ObservedObject var browser: BrowserSessionModel

    var body: some View {
        VStack(spacing: 0) {
            TabBarView()
            ZStack(alignment: .top) {
                Theme.window
                content
                if let tab = activeTab, let sid = tab.session.sessionId,
                   let port = browser.detectedPorts[sid]?.first {
                    PortDetectorBanner(
                        port: port,
                        onOpen: {
                            if let url = URL(string: "http://localhost:\(port)") { NSWorkspace.shared.open(url) }
                            browser.dismissPort(sessionID: sid, port: port)
                        },
                        onDismiss: { browser.dismissPort(sessionID: sid, port: port) }
                    )
                }
            }
        }
    }

    @ViewBuilder private var content: some View {
        if app.selectedWorkspace == nil {
            EmptyStateView(text: "Open a folder to start", systemImage: "folder.badge.plus")
        } else if let tab = activeTab {
            TerminalPaneView(session: tab.session)
                .id(tab.id)
                .padding(Space.paneGap)
        } else {
            EmptyStateView(text: "Press + to open a shell or agent", systemImage: "terminal")
        }
    }

    private var activeTab: TabModel? {
        guard let ws = app.selectedWorkspaceID, let sel = app.selectedTabID[ws] else { return nil }
        return app.currentTabs.first { $0.id == sel }
    }
}

private struct ReplaySessionItem: Identifiable {
    let id: String
}

private struct EmptyStateView: View {
    let text: String
    let systemImage: String

    var body: some View {
        VStack(spacing: 12) {
            Image(systemName: systemImage)
                .font(.system(size: 42, weight: .light))
                .foregroundStyle(Theme.label3)
            Text(text).font(Typo.body).foregroundStyle(Theme.label2)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}
