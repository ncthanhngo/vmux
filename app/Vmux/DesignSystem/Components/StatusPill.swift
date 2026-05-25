import SwiftUI

/// Mode indicator pill with a colored dot (Watch / Gate / Sandbox) from the
/// titlebar in the mockup.
struct StatusPill: View {
    let title: String
    var color: Color = Theme.green

    var body: some View {
        HStack(spacing: 5) {
            Circle().fill(color).frame(width: 6, height: 6)
            Text(title).font(.system(size: 11, weight: .semibold))
        }
        .foregroundStyle(color)
        .padding(.horizontal, 10)
        .padding(.vertical, 4)
        .background(color.opacity(0.16))
        .clipShape(Capsule())
    }
}

#Preview("StatusPill") {
    HStack(spacing: 8) {
        StatusPill(title: "Watch mode", color: Theme.green)
        StatusPill(title: "Gate", color: Theme.orange)
        StatusPill(title: "Sandbox", color: Theme.purple)
    }
    .padding()
}
