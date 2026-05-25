import SwiftUI
import Combine

/// A center-area tab. v0.1 hosts only terminal sessions.
@MainActor
final class TabModel: ObservableObject, Identifiable {
    let id = UUID()
    let agentID: String
    let session: PtySession
    var title: String { session.title }

    init(session: PtySession) {
        self.session = session
        self.agentID = session.command.id
    }
}

enum ConnectionState: Equatable {
    case connecting
    case connected
    case failed(String)
}

/// Top-level observable app state: sidecar connection, workspace registry, and
/// per-workspace tabs. Owns the single `SidecarClient`.
@MainActor
final class AppState: ObservableObject {
    @Published var connection: ConnectionState = .connecting
    @Published var workspaces: [WorkspaceDTO] = []
    @Published var selectedWorkspaceID: String?
    @Published var tabsByWorkspace: [String: [TabModel]] = [:]
    @Published var selectedTabID: [String: UUID] = [:]

    let client = SidecarClient()
    lazy var browser = BrowserSessionModel(client: client)
    lazy var activity = ActivityModel(client: client)
    lazy var diffReview = DiffReviewModel(client: client)
    private var gitCancellable: AnyCancellable?

    private let persistedPathsKey = "vmux.workspacePaths"

    var selectedWorkspace: WorkspaceDTO? {
        workspaces.first { $0.id == selectedWorkspaceID }
    }

    var currentTabs: [TabModel] {
        guard let id = selectedWorkspaceID else { return [] }
        return tabsByWorkspace[id] ?? []
    }

    // MARK: Lifecycle

    func bootstrap() async {
        do {
            try await client.connect()
            connection = .connected
            subscribeGitChanges()
            browser.subscribe()
            activity.subscribe()
            diffReview.subscribe()
            NotificationCenterBridge.shared.requestAuthorization()
            await restoreWorkspaces()
        } catch {
            connection = .failed("\(error)")
        }
    }

    private func subscribeGitChanges() {
        gitCancellable = client.notifications
            .receive(on: DispatchQueue.main)
            .sink { [weak self] note in
                guard note.method == "workspace.gitChanged",
                      let n = try? JSONDecoder().decode(GitChangedNote.self, from: note.params) else { return }
                self?.applyGitStatus(workspaceID: n.workspaceId, status: n.status)
            }
    }

    private func applyGitStatus(workspaceID: String, status: GitStatus) {
        guard let idx = workspaces.firstIndex(where: { $0.id == workspaceID }) else { return }
        let w = workspaces[idx]
        workspaces[idx] = WorkspaceDTO(id: w.id, path: w.path, meta: w.meta, git: status)
    }

    // MARK: Workspaces

    func addWorkspace() {
        let panel = NSOpenPanel()
        panel.canChooseDirectories = true
        panel.canChooseFiles = false
        panel.allowsMultipleSelection = false
        guard panel.runModal() == .OK, let url = panel.url else { return }
        Task { await openWorkspace(path: url.path) }
    }

    func openWorkspace(path: String) async {
        do {
            let ws: WorkspaceDTO = try await client.call("workspace.open", OpenParams(path: path))
            if !workspaces.contains(where: { $0.id == ws.id }) {
                workspaces.append(ws)
            }
            selectedWorkspaceID = ws.id
            persistWorkspaces()
        } catch {
            // Surface via connection banner only on hard failures; ignore dup opens.
        }
    }

    func selectWorkspace(_ id: String) { selectedWorkspaceID = id }

    private func restoreWorkspaces() async {
        let paths = UserDefaults.standard.stringArray(forKey: persistedPathsKey) ?? []
        for path in paths where FileManager.default.fileExists(atPath: path) {
            await openWorkspace(path: path)
        }
        selectedWorkspaceID = workspaces.first?.id
    }

    private func persistWorkspaces() {
        UserDefaults.standard.set(workspaces.map(\.path), forKey: persistedPathsKey)
    }

    // MARK: Tabs

    func newTab(_ agent: AgentCommand) {
        guard let ws = selectedWorkspace else { return }
        let session = PtySession(client: client, command: agent, cwd: ws.path)
        let tab = TabModel(session: session)
        tabsByWorkspace[ws.id, default: []].append(tab)
        selectedTabID[ws.id] = tab.id
    }

    func closeTab(_ id: UUID) {
        guard let wsID = selectedWorkspaceID, var tabs = tabsByWorkspace[wsID] else { return }
        if let tab = tabs.first(where: { $0.id == id }) { tab.session.kill() }
        tabs.removeAll { $0.id == id }
        tabsByWorkspace[wsID] = tabs
        if selectedTabID[wsID] == id { selectedTabID[wsID] = tabs.last?.id }
    }

    func selectTab(_ id: UUID) {
        guard let wsID = selectedWorkspaceID else { return }
        selectedTabID[wsID] = id
    }

    func shutdown() {
        for tabs in tabsByWorkspace.values { for t in tabs { t.session.kill() } }
        client.close()
    }
}
