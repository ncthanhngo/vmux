import Foundation

/// A command to spawn in a PTY tab.
struct AgentCommand: Identifiable, Hashable {
    let id: String
    let title: String
    let cmd: String
    let args: [String]
}

/// Presets for the tab "+" menu. Agents launch through the user's login shell
/// so they inherit the full PATH (a Finder-launched app otherwise has only a
/// minimal PATH, hiding tools like `claude` installed in ~/.local/bin).
enum AgentLauncher {
    static var loginShell: String {
        ProcessInfo.processInfo.environment["SHELL"] ?? "/bin/zsh"
    }

    static var shell: AgentCommand {
        AgentCommand(id: "shell", title: "shell", cmd: loginShell, args: ["-l"])
    }

    static func agent(_ name: String) -> AgentCommand {
        // `-lc 'exec <name>'`: login shell resolves PATH, then replaces itself
        // with the agent so the agent owns the PTY directly.
        AgentCommand(id: name, title: name, cmd: loginShell, args: ["-l", "-c", "exec \(name)"])
    }

    static var claude: AgentCommand { agent("claude") }
    static var codex: AgentCommand { agent("codex") }

    static var presets: [AgentCommand] { [shell, claude, codex] }
}
