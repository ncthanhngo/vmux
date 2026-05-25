---
phase: 4
title: "Design system foundation (Apple HIG, SwiftUI)"
status: pending
priority: P1
effort: "1w"
dependencies: [1]
---

# Phase 4: Design system foundation (Apple HIG, SwiftUI)

> **Visual source of truth:** `visuals/vmux-ui-mockup-apple.html` (approved Apple-style mockup). This phase translates that mockup into a reusable SwiftUI design system. All downstream UI phases (5, 6, 7, 8, 9) consume these tokens + components and must NOT re-style.

## Overview

Build a reusable SwiftUI design system codifying the approved Apple HIG aesthetic: macOS semantic color tokens (light+dark, system accent), `NSVisualEffectView` vibrancy for sidebars, SF Pro / SF Mono typography scale, and a component library (titlebar, segmented control, capsule buttons, floating card panes, sidebar rows, status pill, badge pills, grouped inset cards). Foundation for the entire native UI — built before the shell so phases 5+ assemble from finished parts.

## Requirements

- Functional:
  - Semantic color tokens resolving automatically for light/dark via SwiftUI environment (`@Environment(\.colorScheme)`).
  - Accent color follows user's macOS system accent (`NSColor.controlAccentColor`), default system blue.
  - Vibrancy material wrapper (`NSVisualEffectView`, `.sidebar`/`.headerView` materials) usable from SwiftUI.
  - Component library: each component a standalone, previewable SwiftUI view matching the mockup.
  - Automatic theme switching with system appearance — **no manual toggle in product** (mockup toggle is preview-only).
- Non-functional: all components render in SwiftUI Previews (light+dark) for visual regression; zero hard-coded hex in feature code (only in token layer); 60fps with vibrancy on.

## Architecture

```
app/Vmux/DesignSystem/
├── Tokens/
│   ├── Colors.swift            # semantic Color extensions: .label/.label2/.label3,
│   │                           #   .separator, .accent, .systemGreen/Red/Orange/Purple/Yellow,
│   │                           #   .windowBg/.contentBg/.cardBg/.cardHover
│   ├── Typography.swift        # Font scale: titleBar, body, caption, mono(size, weight)
│   ├── Spacing.swift           # spacing scale (4/8/12/14) + radii (r8, r11, capsule=980)
│   └── Materials.swift         # vibrancy material enum mapping
├── Bridge/
│   └── VibrancyView.swift      # NSViewRepresentable wrapping NSVisualEffectView
├── Components/
│   ├── Titlebar.swift          # 40px unified, single-row title + path, toolbar slot
│   ├── SegmentedControl.swift  # toolbar + panel switcher segmented control
│   ├── CapsuleButton.swift     # filled / tinted / plain variants (Approve/Deny/Accept...)
│   ├── CardPane.swift          # floating pane: radius 11, hairline border, soft shadow, header
│   ├── SidebarRow.swift        # full-width accent-fill selection row (workspace item)
│   ├── StatusBar.swift         # 28px translucent bottom bar with items + chips
│   ├── StatusPill.swift        # mode indicator (Watch/Gate/Sandbox) with colored dot
│   ├── BadgePill.swift         # attention / pending / count badges
│   ├── GroupedCard.swift       # inset grouped card for sidebar panels
│   └── PanelSwitcher.swift     # segmented panel header (Files/Git/AI/MCP...)
└── ColorAssets.xcassets        # any asset-catalog colors needed for system integration
```

Token values (from mockup, dark / light):
- accent `#0A84FF` / `#007AFF` (override by `controlAccentColor`)
- label `.92` / `.9`, label2 `.55` / `.5`, label3 `.32` / `.28` opacity on primary
- separator `.5px` hairline at `.09`/`.08` opacity
- green `#30D158`/`#34C759`, red `#FF453A`/`#FF3B30`, orange `#FF9F0A`/`#FF9500`, purple `#BF5AF2`/`#AF52DE`, yellow `#FFD60A`/`#E6B800`
- radii: 8 (controls), 11 (panes/cards), capsule 980 (buttons/pills)
- pane gap 8, content padding 12-14

## Related Code Files

- Create: all files under `app/Vmux/DesignSystem/` (Swift PascalCase).
- Create: `app/Vmux/DesignSystem/DesignSystemGallery.swift` — a dev-only screen rendering every component in both color schemes (visual QA harness).
- Modify: `app/Vmux/App/VmuxApp.swift` (phase 1 skeleton) — inject design system environment.

## Implementation Steps

1. `Colors.swift`: semantic `Color` static accessors resolving per `colorScheme`; accent reads `NSColor.controlAccentColor` bridged to `Color`. No hex outside this file.
2. `Typography.swift`: SF Pro Text/Display for UI (system font), SF Mono helper `Font.mono(_ size:, weight:)`. Size scale: titlebar 13/600, body 13, caption 11/12, mono 12.
3. `Spacing.swift`: `CGFloat` constants + `RoundedRectangle` radius helpers (`.r8`, `.r11`, `.capsule`).
4. `VibrancyView.swift`: `NSViewRepresentable` over `NSVisualEffectView` with material + blendingMode params; expose `.sidebar`, `.headerView`, `.underWindowBackground`.
5. Build components one-by-one against the mockup; each ships a `#Preview` in light + dark.
   - `Titlebar`: 40px, leading title+path (baseline aligned), trailing toolbar content slot.
   - `SegmentedControl<T>`: generic, pill highlight on selection.
   - `CapsuleButton`: style enum `.filled(accent)`, `.tinted(color)`, `.plain`.
   - `CardPane`: header (title + badge slot + trailing action slot) + content; radius 11, hairline, shadow.
   - `SidebarRow`: selected → accent fill + white content; supports leading status dot + metadata rows.
   - `StatusBar` / `StatusPill` / `BadgePill` / `GroupedCard` / `PanelSwitcher` per mockup.
6. `DesignSystemGallery`: dev menu item rendering all components for visual diffing vs mockup.
7. Wire automatic appearance: ensure no view hard-codes scheme; verify by toggling macOS appearance live.
8. Document token → mockup mapping in `app/Vmux/DesignSystem/README.md` (dev reference, not product docs).

## Todo List

- [ ] Color tokens resolve light/dark + follow system accent
- [ ] VibrancyView renders sidebar material correctly (blur visible over content)
- [ ] All components have light+dark SwiftUI previews matching mockup
- [ ] DesignSystemGallery screen assembles everything
- [ ] Live macOS appearance switch updates entire UI with no relaunch
- [ ] No hex literals outside Tokens/Colors.swift

## Success Criteria

- [ ] Side-by-side: DesignSystemGallery in dark mode is visually indistinguishable from `vmux-ui-mockup-apple.html` dark; same for light
- [ ] Phase 5 shell assembles its UI purely from these components (no new styling)
- [ ] Switching system accent color (System Settings) recolors the app accent live

## Risk Assessment

- **Vibrancy correctness**: `NSVisualEffectView` in SwiftUI can mis-clip/round. Mitigation: wrap in container with explicit corner mask; test over scrolling content.
- **System accent edge cases**: some accents (graphite) need contrast tuning for fills. Mitigation: derive on-accent foreground via luminance check.
- **Token drift from mockup**: hand-translation errors. Mitigation: Gallery screen + mockup compared during review; lock values in one file.
- **Over-engineering**: building components not yet needed. Mitigation: build only the components present in the mockup; add later if a phase needs one.

## Security Considerations

- None (pure presentation layer). No data, no network, no file access.
