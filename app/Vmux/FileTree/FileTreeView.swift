import SwiftUI

/// Right-sidebar file tree for the selected workspace. Lazy-loads directories,
/// shows a modified-dot badge from git status, opens files in the default app,
/// and offers "Reveal in Finder".
struct FileTreeView: View {
    let workspace: WorkspaceDTO
    var onOpenFile: (URL) -> Void = { NSWorkspace.shared.open($0) }
    var onOpenInEditor: (URL) -> Void = { NSWorkspace.shared.open($0) }

    @StateObject private var root: FileNode
    private let badges: GitBadges

    init(workspace: WorkspaceDTO,
         onOpenFile: @escaping (URL) -> Void = { NSWorkspace.shared.open($0) },
         onOpenInEditor: @escaping (URL) -> Void = { NSWorkspace.shared.open($0) }) {
        self.workspace = workspace
        self.onOpenFile = onOpenFile
        self.onOpenInEditor = onOpenInEditor
        let url = URL(fileURLWithPath: workspace.path)
        _root = StateObject(wrappedValue: FileNode(url: url, isDirectory: true))
        self.badges = GitBadges(workspaceRoot: url, status: workspace.git)
    }

    var body: some View {
        ScrollView {
            LazyVStack(alignment: .leading, spacing: 0) {
                FileNodeRows(node: root, depth: 0, badges: badges, isRoot: true,
                             onOpenFile: onOpenFile, onOpenInEditor: onOpenInEditor)
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
    let onOpenFile: (URL) -> Void
    let onOpenInEditor: (URL) -> Void

    var body: some View {
        if isRoot {
            ForEach(node.children ?? []) { child in
                FileNodeRows(node: child, depth: depth, badges: badges, onOpenFile: onOpenFile, onOpenInEditor: onOpenInEditor)
            }
        } else {
            FileRow(node: node, depth: depth, modified: badges.isModified(node.url), onOpenFile: onOpenFile, onOpenInEditor: onOpenInEditor)
            if node.isExpanded {
                ForEach(node.children ?? []) { child in
                    FileNodeRows(node: child, depth: depth + 1, badges: badges, onOpenFile: onOpenFile, onOpenInEditor: onOpenInEditor)
                }
            }
        }
    }
}

private struct FileRow: View {
    @ObservedObject var node: FileNode
    let depth: Int
    let modified: Bool
    let onOpenFile: (URL) -> Void
    let onOpenInEditor: (URL) -> Void

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
            if node.isDirectory { node.toggle() } else { onOpenFile(node.url) }
        }
        .contextMenu {
            if !node.isDirectory {
                Button("Open in Editor") { onOpenInEditor(node.url) }
            }
            Button("Reveal in Finder") {
                NSWorkspace.shared.activateFileViewerSelecting([node.url])
            }
            if !node.isDirectory {
                Button("Open in Default App") { NSWorkspace.shared.open(node.url) }
            }
        }
    }
}
