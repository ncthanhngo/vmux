import Foundation

/// A lazily-loaded file-tree node. Directory children are loaded on first
/// expansion to keep large trees cheap.
@MainActor
final class FileNode: ObservableObject, Identifiable {
    let url: URL
    let isDirectory: Bool
    let name: String

    @Published var children: [FileNode]?
    @Published var isExpanded = false

    nonisolated var id: URL { url }

    init(url: URL, isDirectory: Bool) {
        self.url = url
        self.isDirectory = isDirectory
        self.name = url.lastPathComponent
    }

    func toggle() {
        isExpanded.toggle()
        if isExpanded && children == nil { loadChildren() }
    }

    func loadChildren() {
        let fm = FileManager.default
        let entries = (try? fm.contentsOfDirectory(
            at: url,
            includingPropertiesForKeys: [.isDirectoryKey],
            options: [.skipsHiddenFiles]
        )) ?? []
        children = entries
            .map { entry -> FileNode in
                let isDir = (try? entry.resourceValues(forKeys: [.isDirectoryKey]))?.isDirectory ?? false
                return FileNode(url: entry, isDirectory: isDir)
            }
            // Directories first, then alphabetical.
            .sorted { a, b in
                if a.isDirectory != b.isDirectory { return a.isDirectory }
                return a.name.localizedCaseInsensitiveCompare(b.name) == .orderedAscending
            }
    }
}

/// Set of workspace-relative paths reported modified by git, for badge display.
struct GitBadges {
    let changedPaths: Set<String>

    init(workspaceRoot: URL, status: GitStatus?) {
        var paths = Set<String>()
        for change in status?.changes ?? [] {
            paths.insert(change.path)
        }
        self.changedPaths = paths
        self.root = workspaceRoot
    }

    private let root: URL

    /// True if the node (or, for a directory, anything under it) is modified.
    func isModified(_ url: URL) -> Bool {
        let rel = relativePath(url)
        guard !rel.isEmpty else { return false }
        return changedPaths.contains(rel) || changedPaths.contains { $0.hasPrefix(rel + "/") }
    }

    private func relativePath(_ url: URL) -> String {
        let rootComponents = root.standardizedFileURL.pathComponents
        let urlComponents = url.standardizedFileURL.pathComponents
        guard urlComponents.count > rootComponents.count,
              Array(urlComponents.prefix(rootComponents.count)) == rootComponents else { return "" }
        return urlComponents.dropFirst(rootComponents.count).joined(separator: "/")
    }
}
