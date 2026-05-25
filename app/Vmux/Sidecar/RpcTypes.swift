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

/// Empty params for methods that take none.
struct NoParams: Encodable {}
