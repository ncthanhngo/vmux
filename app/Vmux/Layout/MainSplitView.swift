import SwiftUI

/// The v0.1 three-region shell: workspace sidebar | tabs+terminal | file tree,
/// with a toolbar and bottom status bar. Assembled from the design system.
struct MainSplitView: View {
    @EnvironmentObject var app: AppState
    @State private var replaySession: String?
    @State private var showChromeImport = false
    @State private var showDiffReview = false
    @State private var markdownFile: URL?
    @State private var codeFile: URL?

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
        .sheet(isPresented: $showChromeImport) {
            if let ws = app.selectedWorkspace {
                ChromeImportWizardView(client: app.client, workspaceID: ws.id) { showChromeImport = false }
            }
        }
    }

    private func rightSidebar(_ ws: WorkspaceDTO) -> some View {
        VStack(spacing: 0) {
            // Files first (primary browsing surface), observability panels below.
            VStack(alignment: .leading, spacing: 4) {
                Text("FILES").font(Typo.sectionHeader).foregroundStyle(Theme.label3)
                    .padding(.horizontal, 14).padding(.top, 10)
                FileTreeView(workspace: ws, onOpenFile: openFile, onOpenInEditor: { openInEditor($0, ws: ws) })
            }
            .frame(maxHeight: .infinity)
            Divider()
            ScrollView { rightPanels(ws) }
                .frame(maxHeight: 320)
        }
        .sheet(isPresented: $showDiffReview) {
            if let pending = app.diffReview.pending.first {
                DiffReviewView(model: app.diffReview, pending: pending).frame(minWidth: 560, minHeight: 420)
            }
        }
        .sheet(item: replayBinding) { item in
            ReplayView(model: ReplayModel(client: app.client, sessionID: item.id), onClose: { replaySession = nil })
        }
        .sheet(item: markdownBinding) { item in
            MarkdownPaneView(fileURL: item.url).frame(minWidth: 560, minHeight: 480)
        }
        .sheet(item: codeBinding) { item in
            CodeViewerPaneView(fileURL: item.url, onOpenInEditor: { openInEditor(item.url, ws: ws) })
                .frame(minWidth: 640, minHeight: 480)
        }
    }

    private var replayBinding: Binding<ReplaySessionItem?> {
        Binding(get: { replaySession.map { ReplaySessionItem(id: $0) } },
                set: { if $0 == nil { replaySession = nil } })
    }

    private var markdownBinding: Binding<MarkdownItem?> {
        Binding(get: { markdownFile.map { MarkdownItem(url: $0) } },
                set: { if $0 == nil { markdownFile = nil } })
    }

    private var codeBinding: Binding<MarkdownItem?> {
        Binding(get: { codeFile.map { MarkdownItem(url: $0) } },
                set: { if $0 == nil { codeFile = nil } })
    }

    @ViewBuilder private func rightPanels(_ ws: WorkspaceDTO) -> some View {
        VStack(alignment: .leading, spacing: 14) {
            ActivityPanelView(model: app.activity, workspaceID: ws.id)
            if !app.diffReview.pending.isEmpty {
                CapsuleButton(title: "Review \(app.diffReview.pending.count) file edit(s)", style: .filled) {
                    showDiffReview = true
                }
            }
            if let session = app.activity.events.last?.sessionId {
                CapsuleButton(title: "Open Session Replay", style: .plain) {
                    replaySession = session
                }
            }
            BrowserSessionsView(model: app.browser)
        }
        .padding(8)
    }

    private func openFile(_ url: URL) {
        if url.pathExtension.lowercased() == "md" {
            markdownFile = url
        } else {
            codeFile = url // in-app read-only code viewer
        }
    }

    private func openInEditor(_ url: URL, ws: WorkspaceDTO) {
        let root = URL(fileURLWithPath: ws.path).standardizedFileURL.path
        let full = url.standardizedFileURL.path
        let rel = full.hasPrefix(root + "/") ? String(full.dropFirst(root.count + 1)) : url.lastPathComponent
        app.editor.open(workspaceID: ws.id, path: rel)
    }

    @ToolbarContentBuilder private var toolbarContent: some ToolbarContent {
        ToolbarItem(placement: .principal) {
            if let ws = app.selectedWorkspace {
                Text(ws.path).font(Typo.caption).foregroundStyle(Theme.label2)
            }
        }
        ToolbarItemGroup(placement: .primaryAction) {
            StatusPill(title: "Watch mode")
            if app.selectedWorkspace != nil {
                Button("Import Chrome") { showChromeImport = true }
            }
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

private struct MarkdownItem: Identifiable {
    let id = UUID()
    let url: URL
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
