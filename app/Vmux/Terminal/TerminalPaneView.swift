import SwiftUI
import AppKit
import SwiftTerm

/// SwiftUI host for a SwiftTerm `TerminalView`, bound to a `PtySession`.
struct TerminalPaneView: NSViewRepresentable {
    @ObservedObject var session: PtySession

    func makeCoordinator() -> Coordinator { Coordinator(session: session) }

    func makeNSView(context: Context) -> TerminalView {
        let tv = TerminalView(frame: CGRect(x: 0, y: 0, width: 800, height: 480))
        tv.terminalDelegate = context.coordinator
        tv.nativeBackgroundColor = NSColor(Theme.codeBg)
        tv.nativeForegroundColor = NSColor.white
        if let mono = NSFont(name: "SF Mono", size: 12) ?? NSFont(name: "Menlo", size: 12) {
            tv.font = mono
        }
        session.terminalView = tv

        let term = tv.getTerminal()
        session.start(cols: term.cols, rows: term.rows)
        return tv
    }

    func updateNSView(_ nsView: TerminalView, context: Context) {}

    /// Bridges SwiftTerm delegate callbacks to the PtySession.
    final class Coordinator: NSObject, TerminalViewDelegate {
        let session: PtySession
        init(session: PtySession) { self.session = session }

        func send(source: TerminalView, data: ArraySlice<UInt8>) {
            session.sendInput(data)
        }

        func sizeChanged(source: TerminalView, newCols: Int, newRows: Int) {
            session.resize(cols: newCols, rows: newRows)
        }

        func setTerminalTitle(source: TerminalView, title: String) {}
        func hostCurrentDirectoryUpdate(source: TerminalView, directory: String?) {}
        func scrolled(source: TerminalView, position: Double) {}
        func requestOpenLink(source: TerminalView, link: String, params: [String: String]) {
            if let url = URL(string: link) { NSWorkspace.shared.open(url) }
        }
        func bell(source: TerminalView) {}
        func clipboardCopy(source: TerminalView, content: Data) {
            if let s = String(data: content, encoding: .utf8) {
                NSPasteboard.general.clearContents()
                NSPasteboard.general.setString(s, forType: .string)
            }
        }
        func iTermContent(source: TerminalView, content: ArraySlice<UInt8>) {}
        func rangeChanged(source: TerminalView, startY: Int, endY: Int) {}
    }
}
