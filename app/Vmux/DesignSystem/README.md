# vmux Design System

Dev reference (not product docs). Codifies the locked mockup
`plans/260524-2153-vmux-ai-dev-cockpit/visuals/vmux-ui-mockup-apple.html` as a
reusable SwiftUI design system. **All downstream UI (phases 5+) must assemble
from these components and tokens — no re-styling, no new hex literals.**

## Rules

- The ONLY file allowed to contain raw hex / opacity literals is
  `Tokens/Colors.swift`. Everything else references `Theme`.
- Theme switches automatically with the macOS appearance (dynamic `NSColor`);
  there is no in-product theme toggle (the mockup toggle is preview-only).
- `Theme.accent` follows the user's System Settings accent color.

## Token → mockup mapping

| Token | Dark | Light |
|---|---|---|
| `Theme.window` | `#1E1E20` | `#ECECEE` |
| `Theme.content` | `#1C1C1E` | `#FFFFFF` |
| `Theme.sidebarSolid` | `#28282B` | `#F3F3F5` (vibrancy preferred via `VibrancyView`) |
| `Theme.card` / `cardHover` | white .05 / .08 | black .025 / .05 |
| `Theme.codeBg` | `#161618` | `#1C1C1E` |
| `Theme.separator` / `separatorStrong` | white .09 / .14 | black .08 / .13 |
| `Theme.label` / `label2` / `label3` | white .92 / .55 / .32 | black .90 / .50 / .28 |
| `Theme.accent` | `controlAccentColor` (≈`#0A84FF`) | `controlAccentColor` (≈`#007AFF`) |
| `Theme.green/red/orange/yellow/purple/teal/pink` | per mockup dark | per mockup light |

Radii: `Radius.control` 8, `Radius.card` 11, `Radius.capsule` (pill). Spacing
scale `Space.xs/sm/md/lg` = 4/8/12/14, `Space.paneGap` 8.

## Components

| Component | Mockup element |
|---|---|
| `Titlebar` + `ToolbarIconButton` | 40px unified titlebar/toolbar |
| `SegmentedControl<T>` | Code/Replay toolbar switch |
| `CapsuleButton` | Approve / Deny / Once |
| `CardPane` + `PaneHeader` | floating content panes |
| `SidebarRow` + `StatusDot` | workspace list rows (accent-fill selection) |
| `PanelSwitcher` + `PanelTab` | right-sidebar Files/Git/AI/MCP switcher |
| `GroupedCard` | inset sidebar panel cards |
| `StatusBar` + `StatusItem` + `StatusChip` | 28px bottom bar |
| `StatusPill` | titlebar mode indicator (Watch/Gate) |
| `BadgePill` | attention / warning / count badges |
| `VibrancyView` | `NSVisualEffectView` bridge for sidebars/header |

## Visual QA

`DesignSystemGallery` renders every component and is the app root until the
phase-5 shell lands. Compare it side-by-side with the mockup in both schemes.
Every component also ships a `#Preview` (light + dark) for Xcode canvas review.
