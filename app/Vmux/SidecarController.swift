import Foundation

/// Launches the embedded `vmux-sidecar` Go binary and tears it down on quit.
/// Stdout/stderr are appended to ~/Library/Logs/vmux/sidecar.log.
final class SidecarController {
    private var process: Process?
    private var logHandle: FileHandle?

    /// Locate, launch, and wire up the embedded sidecar. Idempotent.
    func start() {
        guard process == nil else { return }

        guard let binaryURL = Bundle.main.url(forResource: "vmux-sidecar", withExtension: nil) else {
            NSLog("[vmux] sidecar binary not found in bundle Resources; skipping launch")
            return
        }

        let logURL = ensureLogFile()
        let proc = Process()
        proc.executableURL = binaryURL
        if let logURL {
            let handle = try? FileHandle(forWritingTo: logURL)
            handle?.seekToEndOfFile()
            logHandle = handle
            proc.standardOutput = handle
            proc.standardError = handle
        }

        do {
            try proc.run()
            process = proc
            NSLog("[vmux] sidecar launched (pid=\(proc.processIdentifier))")
        } catch {
            NSLog("[vmux] failed to launch sidecar: \(error.localizedDescription)")
        }
    }

    /// Terminate the sidecar on quit. Sends SIGTERM and waits briefly for the
    /// clean shutdown path. This runs from applicationWillTerminate, so it must
    /// be synchronous — an async escalation would never fire before the app
    /// exits. If the sidecar wedges past the deadline, the sidecar's own
    /// parent-PID watchdog is the real backstop: once this app exits the sidecar
    /// is reparented to launchd and self-terminates within ~1s. We never raw
    /// kill(pid) here because Foundation may have reaped and the OS recycled it.
    func stop() {
        guard let proc = process else { return }
        defer {
            process = nil
            try? logHandle?.close()
            logHandle = nil
        }
        guard proc.isRunning else { return }
        proc.terminate() // SIGTERM → sidecar cancels its context and exits
        let deadline = Date().addingTimeInterval(1.5)
        while proc.isRunning && Date() < deadline {
            usleep(50_000)
        }
    }

    /// Create ~/Library/Logs/vmux/sidecar.log if needed; return its URL.
    private func ensureLogFile() -> URL? {
        let fm = FileManager.default
        guard let library = fm.urls(for: .libraryDirectory, in: .userDomainMask).first else { return nil }
        let dir = library.appendingPathComponent("Logs/vmux", isDirectory: true)
        try? fm.createDirectory(at: dir, withIntermediateDirectories: true)
        let logURL = dir.appendingPathComponent("sidecar.log")
        if !fm.fileExists(atPath: logURL.path) {
            fm.createFile(atPath: logURL.path, contents: nil)
        }
        return logURL
    }
}
