import Foundation
import Security

/// Scans Chrome profiles and imports their data into a workspace's browser
/// profile via the sidecar. Cookie import (which needs the Keychain Safe Storage
/// key) is stubbed pending the sidecar's cookie write-back path.
@MainActor
final class ChromeImporter: ObservableObject {
    @Published private(set) var profiles: [ChromeProfile] = []
    @Published var status: String = ""

    private let client: SidecarClient
    init(client: SidecarClient) { self.client = client }

    func scan() async {
        guard let res: ChromeScanResult = try? await client.call("chromeImport.scan", NoParams()) else {
            status = "Could not read Chrome profiles"
            return
        }
        profiles = res.profiles ?? []
    }

    func importBookmarks(profile: ChromeProfile, workspaceID: String) async {
        do {
            let res: ImportBookmarksResult = try await client.call(
                "chromeImport.importBookmarks",
                ImportBookmarksParams(workspaceId: workspaceID, profileDir: profile.dir)
            )
            status = "Imported \(res.imported) bookmarks from \(profile.name)"
        } catch {
            status = "Import failed: \(error)"
        }
    }

    /// Fetches the "Chrome Safe Storage" Keychain password (shows the system
    /// prompt on first access). Used for cookie import once write-back lands.
    static func safeStoragePassword() -> String? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: "Chrome Safe Storage",
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne,
        ]
        var item: CFTypeRef?
        guard SecItemCopyMatching(query as CFDictionary, &item) == errSecSuccess,
              let data = item as? Data else { return nil }
        return String(data: data, encoding: .utf8)
    }
}
