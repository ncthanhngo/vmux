import Foundation
import UserNotifications

/// Posts macOS notifications when an agent needs attention. Authorization is
/// requested once; notifications are deduped per (workspace, reason) within a
/// short window to avoid spam.
@MainActor
final class NotificationCenterBridge {
    static let shared = NotificationCenterBridge()

    private var lastFired: [String: Date] = [:]
    private let debounce: TimeInterval = 30
    private var authorized = false

    func requestAuthorization() {
        UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .sound]) { granted, _ in
            Task { @MainActor in self.authorized = granted }
        }
    }

    /// Notify that a workspace's agent needs attention (e.g. waiting for input).
    func notifyAttention(workspaceName: String, message: String) {
        guard authorized else { return }
        let key = workspaceName + "|" + message
        if let last = lastFired[key], Date().timeIntervalSince(last) < debounce { return }
        lastFired[key] = Date()

        let content = UNMutableNotificationContent()
        content.title = "vmux · \(workspaceName)"
        content.body = message
        content.sound = .default
        let req = UNNotificationRequest(identifier: UUID().uuidString, content: content, trigger: nil)
        UNUserNotificationCenter.current().add(req)
    }
}
