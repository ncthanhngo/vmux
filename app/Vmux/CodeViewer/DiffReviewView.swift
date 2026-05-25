import SwiftUI
import Combine

/// Tracks pending diff reviews (agent file edits staged in gate mode).
@MainActor
final class DiffReviewModel: ObservableObject {
    @Published private(set) var pending: [DiffPending] = []

    private let client: SidecarClient
    private var cancellable: AnyCancellable?

    init(client: SidecarClient) { self.client = client }

    func subscribe() {
        cancellable = client.notifications
            .receive(on: DispatchQueue.main)
            .sink { [weak self] note in
                guard note.method == "diffReview.changed",
                      let n = try? JSONDecoder().decode(DiffReviewChangedNote.self, from: note.params) else { return }
                self?.pending = n.pending
            }
    }

    func decide(path: String, hunkID: Int, accept: Bool) {
        Task { try? await client.send("diffReview.decide", DiffDecideParams(path: path, hunkId: hunkID, accept: accept)) }
    }

    func acceptAll(_ path: String) {
        Task { try? await client.send("diffReview.acceptAll", DiffPathParams(path: path)) }
    }

    func rejectAll(_ path: String) {
        Task { try? await client.send("diffReview.rejectAll", DiffPathParams(path: path)) }
    }
}

/// Inline diff review for a staged file change: red/green hunks with per-hunk
/// Accept/Reject plus Accept all / Reject all.
struct DiffReviewView: View {
    @ObservedObject var model: DiffReviewModel
    let pending: DiffPending

    var body: some View {
        VStack(spacing: 0) {
            HStack {
                Text(URL(fileURLWithPath: pending.path).lastPathComponent)
                    .font(Typo.caption).foregroundStyle(Theme.label)
                BadgePill("\(pending.hunks.count) pending", kind: .warning)
                Spacer()
                CapsuleButton(title: "Reject all", style: .plain) { model.rejectAll(pending.path) }
                CapsuleButton(title: "Accept all", style: .filled) { model.acceptAll(pending.path) }
            }
            .padding(.horizontal, 12)
            .frame(height: 32)
            .overlay(alignment: .bottom) { Rectangle().fill(Theme.separator).frame(height: 0.5) }

            ScrollView {
                LazyVStack(alignment: .leading, spacing: 0) {
                    ForEach(pending.hunks) { hunk in
                        hunkView(hunk)
                    }
                }
            }
        }
        .background(Theme.codeBg)
    }

    @ViewBuilder private func hunkView(_ hunk: DiffHunk) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            ForEach(Array((hunk.oldLines ?? []).enumerated()), id: \.offset) { _, line in
                codeLine(line, sign: "-", bg: Theme.red.opacity(0.12), fg: Theme.red)
            }
            ForEach(Array((hunk.newLines ?? []).enumerated()), id: \.offset) { _, line in
                codeLine(line, sign: "+", bg: Theme.green.opacity(0.12), fg: Theme.green)
            }
            HStack(spacing: 6) {
                Text("hunk \(hunk.id + 1)").font(.system(size: 10)).foregroundStyle(Theme.label2)
                Spacer()
                CapsuleButton(title: "Reject", style: .tinted(Theme.red)) {
                    model.decide(path: pending.path, hunkID: hunk.id, accept: false)
                }
                CapsuleButton(title: "Accept ⌘⏎", style: .tinted(Theme.green)) {
                    model.decide(path: pending.path, hunkID: hunk.id, accept: true)
                }
            }
            .padding(.horizontal, 12).padding(.vertical, 5)
            .background(Theme.card)
        }
    }

    private func codeLine(_ text: String, sign: String, bg: Color, fg: Color) -> some View {
        HStack(spacing: 0) {
            Text(sign).font(.mono(11)).foregroundStyle(fg).frame(width: 16)
            Text(text).font(.mono(11)).foregroundStyle(fg)
            Spacer(minLength: 0)
        }
        .padding(.horizontal, 8).padding(.vertical, 1)
        .background(bg)
    }
}
