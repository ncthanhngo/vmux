import SwiftUI
import Combine

/// Loads a session's replay timeline and reconstructs state at a scrubbed time.
@MainActor
final class ReplayModel: ObservableObject {
    @Published private(set) var moments: [ReplayMoment] = []
    @Published private(set) var snapshot: ReplaySnapshot?
    @Published var index: Double = 0

    private let client: SidecarClient
    let sessionID: String

    init(client: SidecarClient, sessionID: String) {
        self.client = client
        self.sessionID = sessionID
    }

    func load() async {
        guard let res: ReplayTimelineResult = try? await client.call(
            "replay.timeline", ReplayTimelineParams(sessionId: sessionID, bucketSeconds: 5)
        ) else { return }
        moments = res.moments
        index = Double(max(0, res.moments.count - 1))
        await loadSnapshot()
    }

    func loadSnapshot() async {
        let i = Int(index.rounded())
        guard moments.indices.contains(i) else { return }
        snapshot = try? await client.call(
            "replay.at", ReplayAtParams(sessionId: sessionID, t: moments[i].t, windowSeconds: 30)
        )
    }
}

/// Session Replay: a scrubber over activity density + a reconstructed snapshot.
struct ReplayView: View {
    @StateObject var model: ReplayModel
    let onClose: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack {
                Text("Session Replay").font(Typo.titleBar).foregroundStyle(Theme.label)
                Spacer()
                CapsuleButton(title: "Close", style: .filled, action: onClose)
            }
            .padding(12)

            detail
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
                .padding(12)

            scrubber.padding(12)
        }
        .frame(minWidth: 560, minHeight: 420)
        .background(Theme.content)
        .task { await model.load() }
    }

    @ViewBuilder private var detail: some View {
        if let s = model.snapshot {
            VStack(alignment: .leading, spacing: 8) {
                if let call = s.lastToolCall { row("Last tool call", call) }
                if let cmds = s.commandsRun, !cmds.isEmpty { listRow("Commands", cmds) }
                if let files = s.filesTouched, !files.isEmpty { listRow("Files touched", files) }
                if let errs = s.consoleErrors, !errs.isEmpty { listRow("Console", errs) }
            }
        } else {
            Text("No events recorded for this session yet")
                .font(Typo.caption).foregroundStyle(Theme.label3)
        }
    }

    private var scrubber: some View {
        VStack(alignment: .leading, spacing: 6) {
            // Density heatmap.
            GeometryReader { geo in
                HStack(spacing: 1) {
                    ForEach(model.moments) { m in
                        Rectangle().fill(color(for: m.maxRisk))
                            .frame(height: 18)
                            .opacity(0.4 + min(0.6, Double(m.count) / 10))
                    }
                }
            }
            .frame(height: 18)
            .clipShape(RoundedRectangle(cornerRadius: 6, style: .continuous))

            if model.moments.count > 1 {
                Slider(value: $model.index, in: 0...Double(model.moments.count - 1), step: 1)
                    .onChange(of: model.index) { _, _ in Task { await model.loadSnapshot() } }
            }
        }
    }

    private func row(_ label: String, _ value: String) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(label.uppercased()).font(Typo.sectionHeader).foregroundStyle(Theme.label3)
            Text(value).font(.mono(11)).foregroundStyle(Theme.label)
        }
    }

    private func listRow(_ label: String, _ values: [String]) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(label.uppercased()).font(Typo.sectionHeader).foregroundStyle(Theme.label3)
            ForEach(values, id: \.self) { v in
                Text("• \(v)").font(.mono(11)).foregroundStyle(Theme.label2)
            }
        }
    }

    private func color(for risk: String) -> Color {
        switch risk {
        case "high": return Theme.red
        case "medium": return Theme.orange
        default: return Theme.accent
        }
    }
}
