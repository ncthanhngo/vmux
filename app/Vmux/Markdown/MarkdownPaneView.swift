import SwiftUI

/// Native markdown viewer. Renders a `.md` file with SwiftUI's built-in
/// AttributedString markdown (no network, no JS bundle). A View/Source toggle
/// shows raw text. The richer markdown-it/Shiki/KaTeX/Mermaid WKWebView pipeline
/// is a later upgrade; this covers everyday README/notes review.
struct MarkdownPaneView: View {
    let fileURL: URL
    @State private var raw: String = ""
    @State private var showSource = false

    var body: some View {
        VStack(spacing: 0) {
            HStack {
                Text(fileURL.lastPathComponent).font(Typo.caption).foregroundStyle(Theme.label)
                Spacer()
                SegmentedControl(options: ["View", "Source"], selection: Binding(
                    get: { showSource ? "Source" : "View" },
                    set: { showSource = ($0 == "Source") }
                )) { $0 }
            }
            .padding(.horizontal, 12)
            .frame(height: 32)
            .overlay(alignment: .bottom) { Rectangle().fill(Theme.separator).frame(height: 0.5) }

            ScrollView {
                Group {
                    if showSource {
                        Text(raw).font(.mono(12)).foregroundStyle(Theme.label)
                            .frame(maxWidth: .infinity, alignment: .leading)
                    } else {
                        Text(rendered).foregroundStyle(Theme.label)
                            .frame(maxWidth: .infinity, alignment: .leading)
                    }
                }
                .textSelection(.enabled)
                .padding(16)
            }
        }
        .background(Theme.content)
        .task { raw = (try? String(contentsOf: fileURL, encoding: .utf8)) ?? "" }
    }

    private var rendered: AttributedString {
        (try? AttributedString(
            markdown: raw,
            options: .init(interpretedSyntax: .inlineOnlyPreservingWhitespace)
        )) ?? AttributedString(raw)
    }
}
