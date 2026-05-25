import SwiftUI

/// Detects editors and opens files in the user's preferred external editor via
/// the sidecar. vmux hands off editing rather than embedding an editor.
@MainActor
final class EditorModel: ObservableObject {
    @Published private(set) var editors: [DetectedEditor] = []
    @Published var preferred: String = ""

    private let client: SidecarClient
    init(client: SidecarClient) { self.client = client }

    func detect() async {
        if let res: EditorDetectedResult = try? await client.call("editor.detected", NoParams()) {
            editors = res.editors ?? []
            if preferred.isEmpty { preferred = editors.first?.id ?? "" }
        }
    }

    func open(workspaceID: String, path: String, line: Int = 1, col: Int = 0) {
        let editorId = preferred.isEmpty ? nil : preferred
        Task {
            try? await client.send("editor.invoke", EditorInvokeParams(
                workspaceId: workspaceID, path: path, line: line, col: col, editorId: editorId))
        }
    }

    func setPreference(workspaceID: String, editorID: String) {
        preferred = editorID
        Task { try? await client.send("editor.preference", EditorPreferenceParams(workspaceId: workspaceID, editorId: editorID)) }
    }
}

/// Compact editor picker for settings (global preference).
struct EditorPreferenceView: View {
    @ObservedObject var model: EditorModel

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("DEFAULT EDITOR").font(Typo.sectionHeader).foregroundStyle(Theme.label3)
            if model.editors.isEmpty {
                Text("No editors detected").font(Typo.caption2).foregroundStyle(Theme.label3)
            } else {
                Picker("", selection: Binding(
                    get: { model.preferred },
                    set: { model.setPreference(workspaceID: "", editorID: $0) }
                )) {
                    ForEach(model.editors) { e in Text(e.name).tag(e.id) }
                }
                .labelsHidden()
            }
        }
        .task { await model.detect() }
    }
}
