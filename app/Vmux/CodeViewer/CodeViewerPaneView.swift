import SwiftUI

/// Read-only in-app code viewer: monospaced, line-numbered, selectable. Native
/// (no syntax highlighting yet — that's the deferred CodeMirror/tree-sitter
/// runtime). Use "Open in Editor" to edit in your external editor.
struct CodeViewerPaneView: View {
    let fileURL: URL
    var onOpenInEditor: (() -> Void)? = nil

    @State private var lines: [String] = []
    @State private var tooLarge = false

    private let maxLines = 20_000

    var body: some View {
        VStack(spacing: 0) {
            HStack(spacing: 8) {
                Text(fileURL.lastPathComponent).font(Typo.caption).foregroundStyle(Theme.label)
                Spacer()
                if let onOpenInEditor {
                    CapsuleButton(title: "Open in Editor", style: .plain, action: onOpenInEditor)
                }
            }
            .padding(.horizontal, 12)
            .frame(height: 32)
            .overlay(alignment: .bottom) { Rectangle().fill(Theme.separator).frame(height: 0.5) }

            if tooLarge {
                emptyNote("File too large to preview — open in your editor")
            } else if lines.isEmpty {
                emptyNote("Empty file")
            } else {
                ScrollView([.vertical, .horizontal]) {
                    LazyVStack(alignment: .leading, spacing: 0) {
                        ForEach(Array(lines.enumerated()), id: \.offset) { idx, line in
                            HStack(alignment: .top, spacing: 0) {
                                Text("\(idx + 1)")
                                    .font(.mono(11))
                                    .foregroundStyle(Theme.label3)
                                    .frame(width: 44, alignment: .trailing)
                                    .padding(.trailing, 10)
                                Text(line.isEmpty ? " " : line)
                                    .font(.mono(11))
                                    .foregroundStyle(Theme.label)
                                    .textSelection(.enabled)
                            }
                            .padding(.vertical, 0.5)
                        }
                    }
                    .padding(.vertical, 8)
                }
            }
        }
        .background(Theme.codeBg)
        .task { load() }
    }

    private func emptyNote(_ text: String) -> some View {
        Text(text).font(Typo.caption).foregroundStyle(Theme.label3)
            .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private func load() {
        guard let content = try? String(contentsOf: fileURL, encoding: .utf8) else {
            tooLarge = false
            lines = []
            return
        }
        let all = content.components(separatedBy: "\n")
        if all.count > maxLines {
            tooLarge = true
        } else {
            lines = all
        }
    }
}
