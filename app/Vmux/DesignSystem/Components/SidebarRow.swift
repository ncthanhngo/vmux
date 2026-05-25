import SwiftUI

/// Workspace status, shown as a colored dot in the sidebar row.
enum WorkspaceStatus {
    case running, idle, attention

    var color: Color {
        switch self {
        case .running: return Theme.green
        case .idle: return Theme.label3
        case .attention: return Theme.red
        }
    }
}

/// Full-width sidebar row from the mockup. When selected it fills with the
/// accent color and forces white content; otherwise a hover background applies.
struct SidebarRow<Meta: View>: View {
    let name: String
    let status: WorkspaceStatus
    var selected: Bool = false
    @ViewBuilder var meta: () -> Meta

    var body: some View {
        VStack(alignment: .leading, spacing: 3) {
            HStack(spacing: 9) {
                StatusDot(status: status, onAccent: selected)
                Text(name).font(Typo.bodyEmphasis)
            }
            meta()
                .font(Typo.caption2)
                .padding(.leading, 18)
        }
        .foregroundStyle(selected ? Theme.onAccent : Theme.label)
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 10)
        .padding(.vertical, 7)
        .background {
            RoundedRectangle(cornerRadius: 8, style: .continuous)
                .fill(selected ? AnyShapeStyle(Theme.accent) : AnyShapeStyle(Color.clear))
        }
        .contentShape(Rectangle())
    }
}

/// The leading status dot; pulses for attention, ringed when on an accent fill.
struct StatusDot: View {
    let status: WorkspaceStatus
    var onAccent: Bool = false
    @State private var pulse = false

    var body: some View {
        Circle()
            .fill(onAccent ? Theme.onAccent : status.color)
            .frame(width: 9, height: 9)
            .overlay {
                if onAccent {
                    Circle().strokeBorder(Theme.onAccent.opacity(0.35), lineWidth: 2)
                }
            }
            .opacity(status == .attention && pulse ? 0.35 : 1)
            .animation(status == .attention ? .easeInOut(duration: 0.8).repeatForever(autoreverses: true) : .default, value: pulse)
            // Drive from status (not just onAppear) so a row reused in a live
            // list still starts/stops pulsing when its status changes.
            .onAppear { pulse = (status == .attention) }
            .onChange(of: status) { _, newStatus in pulse = (newStatus == .attention) }
    }
}

#Preview("SidebarRow") {
    VStack(spacing: 2) {
        SidebarRow(name: "myapp", status: .running, selected: true) {
            VStack(alignment: .leading, spacing: 1) {
                Text("⎇ main · 3 mod").foregroundStyle(Theme.onAccent.opacity(0.85))
            }
        }
        SidebarRow(name: "prod-monitor", status: .attention) {
            Text("claude waiting…").foregroundStyle(Theme.orange).italic()
        }
        SidebarRow(name: "blog", status: .idle) {
            Text("⎇ dev").foregroundStyle(Theme.purple)
        }
    }
    .frame(width: 224)
    .padding()
    .background(Theme.sidebarSolid)
}
