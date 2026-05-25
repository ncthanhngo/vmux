import Foundation
import Combine
import SwiftTerm

/// Binds a SwiftTerm `TerminalView` to a sidecar PTY session: user keystrokes
/// become `pty.write`, `pty.data` notifications feed the terminal, and view
/// resizes become `pty.resize`.
final class PtySession: ObservableObject, Identifiable {
    let id = UUID()
    let command: AgentCommand
    let cwd: String
    var title: String { command.title }

    @Published private(set) var sessionId: String?
    @Published private(set) var exited = false

    private let client: SidecarClient
    weak var terminalView: TerminalView?
    private var cancellable: AnyCancellable?
    private var started = false

    init(client: SidecarClient, command: AgentCommand, cwd: String) {
        self.client = client
        self.command = command
        self.cwd = cwd
    }

    /// Spawn the PTY (idempotent). `cols`/`rows` seed the initial window size.
    func start(cols: Int, rows: Int) {
        guard !started else { return }
        started = true

        cancellable = client.notifications
            .receive(on: DispatchQueue.main)
            .sink { [weak self] note in self?.handle(note) }

        let env = ProcessInfo.processInfo.environment.map { "\($0.key)=\($0.value)" }
        Task {
            do {
                let res: SpawnResult = try await client.call(
                    "pty.spawn",
                    SpawnParams(cwd: cwd, cmd: command.cmd, args: command.args, env: env)
                )
                try? await client.send("pty.resize", ResizeParams(sessionId: res.sessionId, cols: cols, rows: rows))
                await MainActor.run { self.sessionId = res.sessionId }
            } catch {
                await MainActor.run { self.exited = true }
            }
        }
    }

    private func handle(_ note: RpcNotification) {
        switch note.method {
        case "pty.data":
            guard let d = try? JSONDecoder().decode(PtyDataNote.self, from: note.params),
                  d.sessionId == sessionId,
                  let bytes = Data(base64Encoded: d.chunk) else { return }
            terminalView?.feed(byteArray: [UInt8](bytes)[...])
        case "pty.exit":
            guard let e = try? JSONDecoder().decode(PtyExitNote.self, from: note.params),
                  e.sessionId == sessionId else { return }
            exited = true
        default:
            break
        }
    }

    func sendInput(_ data: ArraySlice<UInt8>) {
        guard let sid = sessionId else { return }
        // Fire-and-forget on the client's serial write queue so keystrokes
        // (and pastes) reach the PTY in the exact order they were typed —
        // detached Tasks would have no ordering guarantee.
        let b64 = Data(data).base64EncodedString()
        client.notify("pty.write", WriteParams(sessionId: sid, data: b64))
    }

    func resize(cols: Int, rows: Int) {
        guard let sid = sessionId else { return }
        client.notify("pty.resize", ResizeParams(sessionId: sid, cols: cols, rows: rows))
    }

    func kill() {
        cancellable?.cancel()
        guard let sid = sessionId else { return }
        Task { try? await client.send("pty.kill", SessionIdParams(sessionId: sid)) }
    }
}
