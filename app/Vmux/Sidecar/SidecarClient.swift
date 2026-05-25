import Foundation
import Combine
import Darwin

/// A server→client notification (method + raw params JSON for typed decoding).
struct RpcNotification {
    let method: String
    let params: Data
}

enum SidecarError: Error {
    case connectFailed(String)
    case notConnected
    case rpc(String)
    case decode(String)
}

/// JSON-RPC client to the Go sidecar over its Unix socket. Line-delimited
/// framing; async/await request/response matched by id; notifications are
/// published on `notifications` for terminal sessions and the file tree.
final class SidecarClient: @unchecked Sendable {
    let notifications = PassthroughSubject<RpcNotification, Never>()

    private let socketPath: String
    private var fd: Int32 = -1
    private var readSource: DispatchSourceRead?
    private var inbound = Data()

    private let writeQueue = DispatchQueue(label: "vmux.sidecar.write")
    private let stateLock = NSLock()
    private var nextID = 0
    private var pending: [Int: CheckedContinuation<Data, Error>] = [:]

    init(socketPath: String = SidecarClient.defaultSocketPath) {
        self.socketPath = socketPath
    }

    static var defaultSocketPath: String {
        let lib = FileManager.default.urls(for: .libraryDirectory, in: .userDomainMask)[0]
        return lib.appendingPathComponent("Application Support/vmux/sidecar.sock").path
    }

    // MARK: Connection

    /// Connect, retrying with backoff while the sidecar finishes launching.
    func connect(retries: Int = 8) async throws {
        for attempt in 0..<retries {
            if tryConnectOnce() { startReadLoop(); return }
            try? await Task.sleep(nanoseconds: UInt64(150_000_000 * (attempt + 1)))
        }
        throw SidecarError.connectFailed("could not reach sidecar at \(socketPath)")
    }

    private func tryConnectOnce() -> Bool {
        let s = socket(AF_UNIX, SOCK_STREAM, 0)
        guard s >= 0 else { return false }

        var addr = sockaddr_un()
        addr.sun_family = sa_family_t(AF_UNIX)
        let sunPathSize = MemoryLayout.size(ofValue: addr.sun_path)
        let ok = socketPath.withCString { cpath -> Bool in
            guard strlen(cpath) < sunPathSize else { return false }
            withUnsafeMutablePointer(to: &addr.sun_path) { sunPtr in
                sunPtr.withMemoryRebound(to: CChar.self, capacity: sunPathSize) { dst in
                    _ = strcpy(dst, cpath)
                }
            }
            return true
        }
        guard ok else { Darwin.close(s); return false }

        let len = socklen_t(MemoryLayout<sockaddr_un>.size)
        let connected = withUnsafePointer(to: &addr) { ptr in
            ptr.withMemoryRebound(to: sockaddr.self, capacity: 1) { sa in
                Darwin.connect(s, sa, len) == 0
            }
        }
        guard connected else { Darwin.close(s); return false }
        fd = s
        return true
    }

    private func startReadLoop() {
        let source = DispatchSource.makeReadSource(fileDescriptor: fd, queue: writeQueue)
        source.setEventHandler { [weak self] in self?.drainSocket() }
        source.resume()
        readSource = source
    }

    private func drainSocket() {
        var buf = [UInt8](repeating: 0, count: 64 * 1024)
        let n = Darwin.read(fd, &buf, buf.count)
        guard n > 0 else {
            // EOF or error: the sidecar is gone. Fail every in-flight call so
            // awaiting callers don't hang forever.
            handleDisconnect()
            return
        }
        inbound.append(contentsOf: buf[0..<n])
        while let nl = inbound.firstIndex(of: 0x0A) {
            let line = inbound.subdata(in: inbound.startIndex..<nl)
            inbound.removeSubrange(inbound.startIndex...nl)
            if !line.isEmpty { route(line) }
        }
    }

    private func route(_ line: Data) {
        guard let obj = try? JSONSerialization.jsonObject(with: line) as? [String: Any] else { return }
        if let id = obj["id"] as? Int {
            removePending(id)?.resume(returning: line)
        } else if let method = obj["method"] as? String {
            let params = (try? JSONSerialization.data(withJSONObject: obj["params"] ?? [:])) ?? Data()
            notifications.send(RpcNotification(method: method, params: params))
        }
    }

    // MARK: Calls

    @discardableResult
    func call<P: Encodable, R: Decodable>(_ method: String, _ params: P) async throws -> R {
        guard fd >= 0 else { throw SidecarError.notConnected }
        let id = makeID()
        let req = RpcRequestEnvelope(id: id, method: method, params: params)
        let data = try JSONEncoder().encode(req)

        let responseLine: Data = try await withCheckedThrowingContinuation { cont in
            addPending(id, cont)
            writeQueue.async { [weak self] in
                guard let self else { return }
                if !self.writeLine(data) {
                    self.removePending(id)?.resume(throwing: SidecarError.notConnected)
                }
            }
        }

        let resp = try JSONDecoder().decode(RpcResponse<R>.self, from: responseLine)
        if let err = resp.error { throw SidecarError.rpc(err.message) }
        guard let result = resp.result else { throw SidecarError.decode("empty result for \(method)") }
        return result
    }

    // Synchronous lock helpers keep NSLock usage out of async contexts.
    private func makeID() -> Int {
        stateLock.lock(); defer { stateLock.unlock() }
        nextID += 1
        return nextID
    }

    private func addPending(_ id: Int, _ cont: CheckedContinuation<Data, Error>) {
        stateLock.lock(); defer { stateLock.unlock() }
        pending[id] = cont
    }

    @discardableResult
    private func removePending(_ id: Int) -> CheckedContinuation<Data, Error>? {
        stateLock.lock(); defer { stateLock.unlock() }
        return pending.removeValue(forKey: id)
    }

    /// Convenience for calls whose result is unused.
    func send<P: Encodable>(_ method: String, _ params: P) async throws {
        let _: EmptyResult = try await call(method, params)
    }

    /// Fire-and-forget JSON-RPC notification (no id, no response). Enqueued on
    /// the serial write queue so calls made in order arrive in order — used for
    /// PTY input/resize where ordering matters and acknowledgement does not.
    func notify<P: Encodable>(_ method: String, _ params: P) {
        guard fd >= 0, let data = try? JSONEncoder().encode(NotificationEnvelope(method: method, params: params)) else { return }
        writeQueue.async { [weak self] in _ = self?.writeLine(data) }
    }

    /// Tear down the connection and fail all pending calls (sidecar gone).
    private func handleDisconnect() {
        readSource?.cancel()
        readSource = nil
        if fd >= 0 { Darwin.close(fd); fd = -1 }
        stateLock.lock()
        let inflight = pending
        pending.removeAll()
        stateLock.unlock()
        for (_, cont) in inflight { cont.resume(throwing: SidecarError.notConnected) }
    }

    private func writeLine(_ data: Data) -> Bool {
        var framed = data
        framed.append(0x0A)
        return framed.withUnsafeBytes { raw -> Bool in
            var off = 0
            let base = raw.bindMemory(to: UInt8.self).baseAddress!
            while off < framed.count {
                let w = Darwin.write(fd, base + off, framed.count - off)
                if w <= 0 { return false }
                off += w
            }
            return true
        }
    }

    func close() {
        readSource?.cancel()
        if fd >= 0 { Darwin.close(fd); fd = -1 }
    }
}

private struct RpcRequestEnvelope<P: Encodable>: Encodable {
    let jsonrpc = "2.0"
    let id: Int
    let method: String
    let params: P
}

private struct NotificationEnvelope<P: Encodable>: Encodable {
    let jsonrpc = "2.0"
    let method: String
    let params: P
}

private struct RpcResponse<R: Decodable>: Decodable {
    let result: R?
    let error: RpcErrorObject?
}

private struct RpcErrorObject: Decodable {
    let code: Int
    let message: String
}

/// Decodes the sidecar's `{}` results for void-like methods.
struct EmptyResult: Decodable {}
