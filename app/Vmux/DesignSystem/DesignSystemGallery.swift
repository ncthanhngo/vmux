import SwiftUI

/// Dev-only visual QA harness: renders every design-system component so it can
/// be compared side-by-side with `vmux-ui-mockup-apple.html` in both schemes.
/// Wired as the app's root in phase 4 until the real shell lands in phase 5.
struct DesignSystemGallery: View {
    @State private var titlebarSeg = "Code"
    @State private var panel = PanelTab(title: "AI", badge: 1)

    var body: some View {
        VStack(spacing: 0) {
            titlebar
            ScrollView {
                VStack(alignment: .leading, spacing: 22) {
                    section("Buttons") {
                        HStack(spacing: 8) {
                            CapsuleButton(title: "Approve", style: .filled)
                            CapsuleButton(title: "Deny", style: .tinted(Theme.red))
                            CapsuleButton(title: "Once", style: .plain)
                        }
                    }
                    section("Badges & pills") {
                        HStack(spacing: 8) {
                            BadgePill("needs input", kind: .attention)
                            BadgePill("2 pending", kind: .warning)
                            BadgePill("3", kind: .count)
                            StatusPill(title: "Watch mode")
                            StatusPill(title: "Gate", color: Theme.orange)
                        }
                    }
                    section("Panel switcher") {
                        PanelSwitcher(
                            tabs: [PanelTab(title: "Files"), PanelTab(title: "Git", badge: 3),
                                   PanelTab(title: "AI", badge: 1), PanelTab(title: "MCP")],
                            selection: $panel
                        ).frame(width: 286)
                    }
                    section("Sidebar rows") { sidebarRows }
                    section("Panes") { panes }
                    section("Grouped cards") { groupedCards }
                }
                .padding(20)
            }
            StatusBar {
                StatusItem(glyph: "⎇", value: "main")
                Text("3 modified")
                StatusChip(text: ":3000")
                Text("✦ claude waiting for input").foregroundStyle(Theme.accent)
                Spacer()
                StatusItem(glyph: "🔗", value: "MCP · 2 active")
                Text("v1.0.0")
            }
        }
        .frame(minWidth: 900, minHeight: 680)
        .background(Theme.window)
    }

    private var titlebar: some View {
        Titlebar(title: "myapp", subtitle: "~/dev/myapp · main") {
            StatusPill(title: "Watch mode")
            SegmentedControl(options: ["Code", "Replay"], selection: $titlebarSeg) { $0 }
            ToolbarIconButton(glyph: "⌘K")
            ToolbarIconButton(glyph: "🔔", badge: true)
            ToolbarIconButton(glyph: "⚙︎")
        }
    }

    private var sidebarRows: some View {
        VStack(spacing: 2) {
            SidebarRow(name: "myapp", status: .running, selected: true) {
                Text("⎇ main · 3 mod").foregroundStyle(Theme.onAccent.opacity(0.85))
            }
            SidebarRow(name: "prod-monitor", status: .attention) {
                Text("claude waiting…").foregroundStyle(Theme.orange).italic()
            }
            SidebarRow(name: "blog", status: .idle) {
                Text("⎇ dev").foregroundStyle(Theme.purple)
            }
        }
        .frame(width: 224)
        .padding(8)
        .background(VibrancyView(.sidebar))
        .clipShape(RoundedRectangle.card)
    }

    private var panes: some View {
        HStack(spacing: Space.paneGap) {
            CardPane(attention: true) {
                PaneHeader(icon: "●", title: "claude", badge: BadgePill("needs input", kind: .attention)) {
                    Text("⌘1").font(Typo.caption2)
                }
            } content: {
                VStack(alignment: .leading) {
                    Text("myapp ❯ check production /api/users")
                        .font(.mono(12)).foregroundStyle(Theme.green)
                    Text("✦ claude · opus-4.7").font(.mono(12)).foregroundStyle(Theme.teal)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
                .padding(12)
            }
            CardPane {
                PaneHeader(icon: "±", title: "UserService.ts", badge: BadgePill("2 pending", kind: .warning)) {
                    Text("⌘E Open in Cursor").font(Typo.caption2)
                }
            } content: {
                Text("diff review area").foregroundStyle(Theme.label2).padding(12)
                    .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
            }
        }
        .frame(height: 180)
    }

    private var groupedCards: some View {
        HStack(alignment: .top, spacing: 12) {
            GroupedCard(title: "MCP servers") {
                VStack(alignment: .leading, spacing: 6) {
                    mcpRow("Chrome DevTools MCP", "v0.6.0", Theme.green)
                    mcpRow("vmux builtin", "file·cmd·ws", Theme.green)
                    mcpRow("Playwright MCP", "idle", Theme.label3)
                }
            }.frame(width: 260)
            GroupedCard(title: "Session cost") {
                VStack(alignment: .leading, spacing: 5) {
                    HStack { Text("claude · opus-4.7"); Spacer(); Text("$0.51").font(.mono(11)) }
                    HStack { Text("Total today"); Spacer(); Text("$4.22").font(.mono(11)).foregroundStyle(Theme.green) }
                }
                .font(Typo.caption2).foregroundStyle(Theme.label)
            }.frame(width: 260)
        }
    }

    private func mcpRow(_ name: String, _ ver: String, _ dot: Color) -> some View {
        HStack(spacing: 9) {
            Circle().fill(dot).frame(width: 7, height: 7)
            Text(name).font(Typo.caption2).foregroundStyle(Theme.label)
            Spacer()
            Text(ver).font(.system(size: 10)).foregroundStyle(Theme.label3)
        }
    }

    private func section<Content: View>(_ title: String, @ViewBuilder _ content: () -> Content) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(title.uppercased())
                .font(Typo.sectionHeader).foregroundStyle(Theme.label3)
            content()
        }
    }
}

#Preview("Gallery — dark") {
    DesignSystemGallery().preferredColorScheme(.dark)
}

#Preview("Gallery — light") {
    DesignSystemGallery().preferredColorScheme(.light)
}
