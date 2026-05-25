import SwiftUI

/// Generic segmented control with the mockup's pill-highlight-on-selection look
/// (used in the titlebar Code/Replay switch and elsewhere).
struct SegmentedControl<T: Hashable>: View {
    let options: [T]
    @Binding var selection: T
    var label: (T) -> String

    var body: some View {
        HStack(spacing: 1) {
            ForEach(options, id: \.self) { option in
                Segment(
                    title: label(option),
                    isOn: option == selection,
                    onTap: { selection = option }
                )
            }
        }
        .padding(2)
        .background(RoundedRectangle(cornerRadius: 7, style: .continuous).fill(Theme.card))
        .overlay(
            RoundedRectangle(cornerRadius: 7, style: .continuous)
                .strokeBorder(Theme.separator, lineWidth: 0.5)
        )
    }

    private struct Segment: View {
        let title: String
        let isOn: Bool
        let onTap: () -> Void

        var body: some View {
            Text(title)
                .font(.system(size: 12, weight: .medium))
                .foregroundStyle(isOn ? Theme.label : Theme.label2)
                .padding(.horizontal, 11)
                .padding(.vertical, 3)
                .background { highlight }
                .contentShape(Rectangle())
                .onTapGesture(perform: onTap)
        }

        @ViewBuilder private var highlight: some View {
            if isOn {
                RoundedRectangle(cornerRadius: 5, style: .continuous)
                    .fill(Theme.content)
                    .shadow(color: Theme.shadow, radius: 1.5, y: 1)
            }
        }
    }
}

private struct SegmentedControlPreview: View {
    @State private var sel = "Code"
    var body: some View {
        SegmentedControl(options: ["Code", "Replay"], selection: $sel) { $0 }
            .padding()
    }
}

#Preview("SegmentedControl") { SegmentedControlPreview() }
