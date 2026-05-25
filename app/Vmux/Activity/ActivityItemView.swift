import SwiftUI

/// A single activity-stream row: kind glyph, summary, and a risk-tinted accent.
struct ActivityItemView: View {
    let event: ActivityEvent

    var body: some View {
        HStack(alignment: .top, spacing: 9) {
            Text(glyph)
                .font(.system(size: 12))
                .foregroundStyle(tint)
                .frame(width: 17)
            VStack(alignment: .leading, spacing: 1) {
                Text(event.summary)
                    .font(Typo.caption2)
                    .foregroundStyle(Theme.label)
                    .lineLimit(2)
                if let actor = event.actor, !actor.isEmpty {
                    Text(actor).font(.system(size: 10)).foregroundStyle(Theme.label3)
                }
            }
            Spacer(minLength: 0)
        }
        .padding(.vertical, 3)
    }

    private var glyph: String {
        switch event.kind {
        case "tool_call": return "wrench.and.screwdriver"
        case "file_edit": return "pencil"
        case "command_run": return "terminal"
        case "browser_nav": return "globe"
        case "screenshot": return "camera"
        case "console": return "exclamationmark.bubble"
        case "attention": return "bell"
        case "cost": return "dollarsign.circle"
        default: return "circle"
        }
    }

    private var tint: Color {
        switch event.risk {
        case "high": return Theme.red
        case "medium": return Theme.orange
        default: return Theme.accent
        }
    }
}
