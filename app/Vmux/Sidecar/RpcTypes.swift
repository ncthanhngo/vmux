import Foundation

/// Codable types mirroring the Go sidecar's JSON-RPC schema (see
/// sidecar/internal/server.go and the pty/workspace packages).

// MARK: - PTY

struct SpawnParams: Encodable {
    let cwd: String
    let cmd: String
    let args: [String]
    let env: [String]
}

struct SpawnResult: Decodable {
    let sessionId: String
}

struct WriteParams: Encodable {
    let sessionId: String
    let data: String // base64
}

struct ResizeParams: Encodable {
    let sessionId: String
    let cols: Int
    let rows: Int
}

struct SessionIdParams: Encodable {
    let sessionId: String
}

/// `pty.data` notification payload (base64 chunk of terminal output).
struct PtyDataNote: Decodable {
    let sessionId: String
    let chunk: String
}

/// `pty.exit` notification payload.
struct PtyExitNote: Decodable {
    let sessionId: String
    let code: Int
}

// MARK: - Workspace

struct OpenParams: Encodable {
    let path: String
}

struct WorkspaceMeta: Codable {
    let name: String
    let createdAt: String?
}

struct GitChange: Codable {
    let status: String
    let path: String
}

struct GitStatus: Codable {
    let isRepo: Bool
    let branch: String?
    let dirty: Bool?
    let changes: [GitChange]?
}

struct WorkspaceDTO: Codable, Identifiable {
    let id: String
    let path: String
    let meta: WorkspaceMeta
    let git: GitStatus
}

struct WorkspaceListResult: Decodable {
    let workspaces: [WorkspaceDTO]
}

/// `workspace.gitChanged` notification payload.
struct GitChangedNote: Decodable {
    let workspaceId: String
    let status: GitStatus
}

// MARK: - Browser sessions

/// `browserSession.shotCaptured` notification (base64 PNG thumbnail).
struct ShotCapturedNote: Decodable {
    let sessionId: String
    let shotId: String
    let thumbnail: String
}

/// `browserSession.portDetected` notification.
struct PortDetectedNote: Decodable {
    let sessionId: String
    let port: Int
}

struct LatestShotParams: Encodable {
    let sessionId: String
}

struct LatestShotResult: Decodable {
    let shotId: String
    let width: Int
    let height: Int
    let png: String // base64
}

struct BrowserSessionParams: Encodable {
    let sessionId: String
}

// MARK: - Activity / Approval

/// One activity event (`activity.event` notification / `activity.recent` result).
struct ActivityEvent: Decodable, Identifiable {
    let seq: Int64
    let ts: String
    let sessionId: String
    let kind: String
    let actor: String?
    let summary: String
    let detail: String?
    let risk: String?
    var id: Int64 { seq }
}

struct ActivityRecentResult: Decodable {
    let events: [ActivityEvent]
}

/// A pending approval item (`approval.changed` notification / `approval.list`).
struct PendingApproval: Decodable, Identifiable {
    let id: String
    let tool: String
    let command: String
    let args: [String]?
    let workspace: String
    let ruleId: String
    let reason: String
}

struct ApprovalChangedNote: Decodable {
    let pending: [PendingApproval]
}

struct ApprovalDecideParams: Encodable {
    let id: String
    let allow: Bool
}

struct SetModeParams: Encodable {
    let workspaceId: String
    let mode: String
}

// MARK: - Replay

struct ReplayTimelineParams: Encodable {
    let sessionId: String
    let bucketSeconds: Int
}

struct ReplayMoment: Decodable, Identifiable {
    let t: String
    let count: Int
    let kind: String
    let maxRisk: String
    var id: String { t }
}

struct ReplayTimelineResult: Decodable {
    let start: String
    let end: String
    let moments: [ReplayMoment]
}

struct ReplayAtParams: Encodable {
    let sessionId: String
    let t: String // RFC3339
    let windowSeconds: Int
}

struct ReplaySnapshot: Decodable {
    let lastToolCall: String?
    let filesTouched: [String]?
    let commandsRun: [String]?
    let consoleErrors: [String]?
}

// MARK: - Chrome import

struct ChromeProfile: Decodable, Identifiable {
    let name: String
    let dir: String
    let path: String
    let hasCookies: Bool
    let hasBookmarks: Bool
    let hasHistory: Bool
    var id: String { dir }
}

struct ChromeScanResult: Decodable {
    let profiles: [ChromeProfile]?
}

struct ImportBookmarksParams: Encodable {
    let workspaceId: String
    let profileDir: String
}

struct ImportBookmarksResult: Decodable {
    let imported: Int
}

// MARK: - Diff review

struct DiffHunk: Decodable, Identifiable {
    let id: Int
    let oldStart: Int
    let oldLines: [String]?
    let newLines: [String]?
}

struct DiffPending: Decodable, Identifiable {
    let path: String
    let hunks: [DiffHunk]
    var id: String { path }
}

struct DiffReviewChangedNote: Decodable {
    let pending: [DiffPending]
}

struct DiffDecideParams: Encodable {
    let path: String
    let hunkId: Int
    let accept: Bool
}

struct DiffPathParams: Encodable {
    let path: String
}

/// Empty params for methods that take none.
struct NoParams: Encodable {}
