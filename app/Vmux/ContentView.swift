import SwiftUI

struct ContentView: View {
    var body: some View {
        VStack(spacing: 12) {
            Image(systemName: "terminal")
                .font(.system(size: 48, weight: .light))
                .foregroundStyle(.secondary)
            Text("vmux")
                .font(.largeTitle.weight(.semibold))
            Text("AI terminal observatory")
                .font(.callout)
                .foregroundStyle(.secondary)
        }
        .frame(minWidth: 720, minHeight: 480)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

#Preview {
    ContentView()
}
