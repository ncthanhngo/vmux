import SwiftUI

@main
struct VmuxApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate
    @StateObject private var app = AppState()

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(app)
                .task { await app.bootstrap() }
        }
        .windowStyle(.titleBar)
        .windowToolbarStyle(.unified)
        .commands {
            CommandGroup(replacing: .newItem) {}
            CommandGroup(after: .newItem) {
                Button("New Workspace") { app.addWorkspace() }
                    .keyboardShortcut("n", modifiers: .command)
                Button("New Shell Tab") { app.newTab(AgentLauncher.shell) }
                    .keyboardShortcut("t", modifiers: .command)
                    .disabled(app.selectedWorkspace == nil)
            }
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
