import SwiftUI
import Combine

/// Observes `activity.event` + `approval.changed` notifications and exposes the
/// live activity stream, the pending-approval queue, and per-workspace mode.
@MainActor
final class ActivityModel: ObservableObject {
    @Published private(set) var events: [ActivityEvent] = []
    @Published private(set) var pending: [PendingApproval] = []
    /// Approval mode per workspace id (persisted in UserDefaults).
    @Published var modeByWorkspace: [String: String] = [:]

    private let client: SidecarClient
    private var cancellable: AnyCancellable?
    private let maxEvents = 1000
    private let modeKey = "vmux.approvalModes"

    init(client: SidecarClient) {
        self.client = client
        modeByWorkspace = (UserDefaults.standard.dictionary(forKey: modeKey) as? [String: String]) ?? [:]
    }

    func subscribe() {
        cancellable = client.notifications
            .receive(on: DispatchQueue.main)
            .sink { [weak self] note in self?.handle(note) }
        Task { await reapplyModes() }
    }

    private func handle(_ note: RpcNotification) {
        switch note.method {
        case "activity.event":
            if let e = try? JSONDecoder().decode(ActivityEvent.self, from: note.params) {
                events.append(e)
                if events.count > maxEvents { events.removeFirst(events.count - maxEvents) }
            }
        case "approval.changed":
            if let n = try? JSONDecoder().decode(ApprovalChangedNote.self, from: note.params) {
                pending = n.pending
            }
        default:
            break
        }
    }

    func events(for sessionID: String) -> [ActivityEvent] {
        events.filter { $0.sessionId == sessionID }
    }

    func decide(_ id: String, allow: Bool) {
        Task { try? await client.send("approval.decide", ApprovalDecideParams(id: id, allow: allow)) }
    }

    func mode(for workspaceID: String) -> String {
        modeByWorkspace[workspaceID] ?? "watch"
    }

    func setMode(_ mode: String, for workspaceID: String) {
        modeByWorkspace[workspaceID] = mode
        UserDefaults.standard.set(modeByWorkspace, forKey: modeKey)
        Task { try? await client.send("approval.setMode", SetModeParams(workspaceId: workspaceID, mode: mode)) }
    }

    /// Re-push persisted modes to the sidecar after a (re)connect.
    private func reapplyModes() async {
        for (ws, mode) in modeByWorkspace {
            try? await client.send("approval.setMode", SetModeParams(workspaceId: ws, mode: mode))
        }
    }
}
