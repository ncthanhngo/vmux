import SwiftUI

/// Right-sidebar Activity panel: a mode picker, the pending-approval queue, and
/// the live activity stream for the selected workspace's agent sessions.
struct ActivityPanelView: View {
    @ObservedObject var model: ActivityModel
    let workspaceID: String

    private let modes = ["watch", "gate", "sandbox"]

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            modePicker

            if !model.pending.isEmpty {
                GroupedCard(title: "Approval queue · gate") {
                    ApprovalQueueView(pending: model.pending, onDecide: model.decide)
                }
            }

            GroupedCard(title: "Activity") {
                if model.events.isEmpty {
                    Text("No agent activity yet")
                        .font(Typo.caption2).foregroundStyle(Theme.label3)
                } else {
                    LazyVStack(alignment: .leading, spacing: 0) {
                        ForEach(model.events.suffix(200).reversed()) { event in
                            ActivityItemView(event: event)
                        }
                    }
                }
            }
        }
    }

    private var modePicker: some View {
        VStack(alignment: .leading, spacing: 5) {
            Text("AGENT MODE").font(Typo.sectionHeader).foregroundStyle(Theme.label3)
            HStack(spacing: 1) {
                ForEach(modes, id: \.self) { mode in
                    let on = model.mode(for: workspaceID) == mode
                    Text(mode.capitalized)
                        .font(.system(size: 11, weight: .medium))
                        .foregroundStyle(on ? Theme.label : Theme.label2)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 4)
                        .background {
                            if on {
                                RoundedRectangle(cornerRadius: 5, style: .continuous)
                                    .fill(Theme.content)
                                    .shadow(color: Theme.shadow, radius: 1.5, y: 1)
                            }
                        }
                        .contentShape(Rectangle())
                        .onTapGesture { model.setMode(mode, for: workspaceID) }
                }
            }
            .padding(2)
            .background(RoundedRectangle.control.fill(Theme.card))
            .overlay(RoundedRectangle.control.strokeBorder(Theme.separator, lineWidth: 0.5))
        }
    }
}
