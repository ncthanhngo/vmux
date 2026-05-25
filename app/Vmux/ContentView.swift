import SwiftUI

struct ContentView: View {
    var body: some View {
        // Until the phase-5 shell lands, the app root renders the design-system
        // gallery so the components can be visually verified against the mockup.
        DesignSystemGallery()
    }
}

#Preview {
    ContentView()
}
