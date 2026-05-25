import SwiftUI

/// Right-sidebar file tree for the selected workspace. Lazy-loads directories,
/// shows a modified-dot badge from git status, opens files in the default app,
/// and offers "Reveal in Finder".
struct FileTreeView: View {
    let workspace: WorkspaceDTO

    @StateObject private var root: FileNode
    private let badges: GitBadges

    init(workspace: WorkspaceDTO) {
        self.workspace = workspace
        let url = URL(fileURLWithPath: workspace.path)
        _root = StateObject(wrappedValue: FileNode(url: url, isDirectory: true))
        self.badges = GitBadges(workspaceRoot: url, status: workspace.git)
    }

    var body: some View {
        ScrollView {
            LazyVStack(alignment: .leading, spacing: 0) {
                FileNodeRows(node: root, depth: 0, badges: badges, isRoot: true)
            }
            .padding(.horizontal, 8)
            .padding(.vertical, 6)
        }
        .onAppear { if root.children == nil { root.loadChildren() } }
    }
}

/// Renders a node row and, when expanded, its children recursively.
private struct FileNodeRows: View {
    @ObservedObject var node: FileNode
    let depth: Int
    let badges: GitBadges
    var isRoot = false

    var body: some View {
        if isRoot {
            ForEach(node.children ?? []) { child in
                FileNodeRows(node: child, depth: depth, badges: badges)
            }
        } else {
            FileRow(node: node, depth: depth, modified: badges.isModified(node.url))
            if node.isExpanded {
                ForEach(node.children ?? []) { child in
                    FileNodeRows(node: child, depth: depth + 1, badges: badges)
                }
            }
        }
    }
}

private struct FileRow: View {
    @ObservedObject var node: FileNode
    let depth: Int
    let modified: Bool

    var body: some View {
        HStack(spacing: 5) {
            Image(systemName: node.isDirectory ? (node.isExpanded ? "chevron.down" : "chevron.right") : "doc")
                .font(.system(size: 9))
                .foregroundStyle(Theme.label3)
                .frame(width: 12)
            Text(node.name)
                .font(Typo.caption)
                .foregroundStyle(Theme.label)
                .lineLimit(1)
            Spacer(minLength: 0)
            if modified {
                Circle().fill(Theme.green).frame(width: 6, height: 6)
            }
        }
        .padding(.leading, CGFloat(depth) * 12)
        .padding(.vertical, 3)
        .contentShape(Rectangle())
        .onTapGesture {
            if node.isDirectory { node.toggle() } else { NSWorkspace.shared.open(node.url) }
        }
        .contextMenu {
            Button("Reveal in Finder") {
                NSWorkspace.shared.activateFileViewerSelecting([node.url])
            }
            if !node.isDirectory {
                Button("Open in Default App") { NSWorkspace.shared.open(node.url) }
            }
        }
    }
}
