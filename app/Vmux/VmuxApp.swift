import SwiftUI

@main
struct VmuxApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate

    var body: some Scene {
        WindowGroup {
            ContentView()
        }
        .windowStyle(.titleBar)
        .commands {
            CommandGroup(replacing: .newItem) {}
        }
    }
}

/// Owns the sidecar process lifecycle: launch on app start, terminate on quit.
final class AppDelegate: NSObject, NSApplicationDelegate {
    private let sidecar = SidecarController()

    func applicationDidFinishLaunching(_ notification: Notification) {
        sidecar.start()
    }

    func applicationWillTerminate(_ notification: Notification) {
        sidecar.stop()
    }

    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        true
    }
}
