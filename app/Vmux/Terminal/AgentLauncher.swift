import Foundation

/// A command to spawn in a PTY tab.
struct AgentCommand: Identifiable, Hashable {
    let id: String
    let title: String
    let cmd: String
    let args: [String]
}

/// vmux spawns a plain login-shell terminal. The user runs whatever AI CLI they
/// have installed (claude, codex, …) inside it — vmux does not bundle or choose
/// an agent. A login shell ensures the user's full PATH is available.
enum AgentLauncher {
    static var loginShell: String {
        ProcessInfo.processInfo.environment["SHELL"] ?? "/bin/zsh"
    }

    static var shell: AgentCommand {
        AgentCommand(id: "terminal", title: "terminal", cmd: loginShell, args: ["-l"])
    }
}
