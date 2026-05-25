import SwiftUI
import AppKit

/// SwiftUI wrapper over `NSVisualEffectView` for the blurred vibrancy used by
/// sidebars, the titlebar, and the status bar. Use it as a background:
///
///     content.background(VibrancyView(.sidebar))
struct VibrancyView: NSViewRepresentable {
    let material: VibrancyMaterial
    var blendingMode: NSVisualEffectView.BlendingMode = .behindWindow

    init(_ material: VibrancyMaterial) { self.material = material }

    func makeNSView(context: Context) -> NSVisualEffectView {
        let view = NSVisualEffectView()
        view.material = material.nsMaterial
        view.blendingMode = blendingMode
        view.state = .followsWindowActiveState
        return view
    }

    func updateNSView(_ view: NSVisualEffectView, context: Context) {
        view.material = material.nsMaterial
        view.blendingMode = blendingMode
    }
}
