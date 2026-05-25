import SwiftUI

/// Pending-approval cards (gate mode). Each shows the command the agent wants to
/// run and Approve/Deny actions, styled like the mockup's red approval card.
struct ApprovalQueueView: View {
    let pending: [PendingApproval]
    let onDecide: (String, Bool) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            ForEach(pending) { item in
                card(item)
            }
        }
    }

    private func card(_ item: PendingApproval) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 6) {
                Image(systemName: "exclamationmark.triangle.fill").foregroundStyle(Theme.red)
                Text("\(item.command) wants approval")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(Theme.red)
            }
            Text(commandLine(item))
                .font(.mono(11))
                .foregroundStyle(Theme.label)
                .padding(8)
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(RoundedRectangle(cornerRadius: 8, style: .continuous).fill(Theme.codeBg))
            Text(item.reason).font(Typo.caption2).foregroundStyle(Theme.label2)
            HStack(spacing: 7) {
                CapsuleButton(title: "Approve", style: .filled) { onDecide(item.id, true) }
                CapsuleButton(title: "Deny", style: .tinted(Theme.red)) { onDecide(item.id, false) }
            }
        }
        .padding(12)
        .background(RoundedRectangle.card.fill(Theme.red.opacity(0.09)))
        .overlay(RoundedRectangle.card.strokeBorder(Theme.red.opacity(0.35), lineWidth: 0.5))
    }

    private func commandLine(_ item: PendingApproval) -> String {
        ([item.command] + (item.args ?? [])).joined(separator: " ")
    }
}
