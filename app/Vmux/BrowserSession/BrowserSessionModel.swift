import SwiftUI
import Combine

/// One tracked browser session and its latest screenshot thumbnail.
struct BrowserSession: Identifiable {
    let id: String          // sessionId (upstream MCP server id for v0.2)
    var latestShotID: String?
    var thumbnail: NSImage?
    var shotCount: Int
}

/// Observes `browserSession.*` notifications and exposes the live set of browser
/// sessions plus any detected dev-server ports for banners.
@MainActor
final class BrowserSessionModel: ObservableObject {
    @Published private(set) var sessions: [BrowserSession] = []
    /// Detected ports keyed by PTY session id (for the in-terminal banner).
    @Published private(set) var detectedPorts: [String: [Int]] = [:]

    private let client: SidecarClient
    private var cancellable: AnyCancellable?

    init(client: SidecarClient) {
        self.client = client
    }

    func subscribe() {
        cancellable = client.notifications
            .receive(on: DispatchQueue.main)
            .sink { [weak self] note in self?.handle(note) }
    }

    private func handle(_ note: RpcNotification) {
        switch note.method {
        case "browserSession.shotCaptured":
            guard let n = try? JSONDecoder().decode(ShotCapturedNote.self, from: note.params) else { return }
            applyShot(n)
        case "browserSession.portDetected":
            guard let n = try? JSONDecoder().decode(PortDetectedNote.self, from: note.params) else { return }
            var ports = detectedPorts[n.sessionId] ?? []
            if !ports.contains(n.port) { ports.append(n.port) }
            detectedPorts[n.sessionId] = ports
        default:
            break
        }
    }

    private func applyShot(_ n: ShotCapturedNote) {
        let image = NSImage(base64PNG: n.thumbnail)
        if let idx = sessions.firstIndex(where: { $0.id == n.sessionId }) {
            sessions[idx].latestShotID = n.shotId
            sessions[idx].thumbnail = image
            sessions[idx].shotCount += 1
        } else {
            sessions.append(BrowserSession(id: n.sessionId, latestShotID: n.shotId, thumbnail: image, shotCount: 1))
        }
    }

    /// Dismiss a detected port (after the user acts on or ignores the banner).
    func dismissPort(sessionID: String, port: Int) {
        detectedPorts[sessionID]?.removeAll { $0 == port }
    }

    func focus() {
        Task { try? await client.send("browserSession.focus", NoParams()) }
    }

    func close(_ sessionID: String) {
        Task { try? await client.send("browserSession.close", BrowserSessionParams(sessionId: sessionID)) }
        sessions.removeAll { $0.id == sessionID }
    }

    /// Fetch the full-resolution latest screenshot for the lightbox.
    func loadFullShot(_ sessionID: String) async -> NSImage? {
        guard let res: LatestShotResult = try? await client.call("browserSession.latestShot", LatestShotParams(sessionId: sessionID)) else { return nil }
        return NSImage(base64PNG: res.png)
    }
}

extension NSImage {
    convenience init?(base64PNG: String) {
        guard let data = Data(base64Encoded: base64PNG) else { return nil }
        self.init(data: data)
    }
}
