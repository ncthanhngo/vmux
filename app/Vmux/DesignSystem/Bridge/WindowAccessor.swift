import SwiftUI
import AppKit

/// Rounds the host window's content corners with a continuous curve for the
/// Apple-style look (matches the mockup's rounded app frame). The radius is
/// small enough not to clip the inset traffic-light controls.
struct RoundedWindow: NSViewRepresentable {
    var radius: CGFloat = 12

    func makeNSView(context: Context) -> NSView {
        let v = NSView()
        DispatchQueue.main.async { [weak v] in apply(from: v) }
        return v
    }

    func updateNSView(_ nsView: NSView, context: Context) {
        DispatchQueue.main.async { [weak nsView] in apply(from: nsView) }
    }

    private func apply(from view: NSView?) {
        guard let content = view?.window?.contentView else { return }
        content.wantsLayer = true
        content.layer?.cornerRadius = radius
        content.layer?.cornerCurve = .continuous
        content.layer?.masksToBounds = true
    }
}

extension View {
    /// Applies rounded corners to the hosting window's content.
    func roundedWindowCorners(_ radius: CGFloat = 12) -> some View {
        background(RoundedWindow(radius: radius))
    }
}
