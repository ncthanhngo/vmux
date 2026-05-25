import SwiftUI

/// Spacing scale from the mockup (4 / 8 / 12 / 14) plus a couple of derived steps.
enum Space {
    static let xs: CGFloat = 4
    static let sm: CGFloat = 8
    static let md: CGFloat = 12
    static let lg: CGFloat = 14
    /// Gap between floating panes.
    static let paneGap: CGFloat = 8
}

/// Corner radii from the mockup.
enum Radius {
    /// Controls (segmented control, small buttons): 8.
    static let control: CGFloat = 8
    /// Panes and grouped cards: 11.
    static let card: CGFloat = 11
    /// Capsule (buttons, pills) — large value clamps to a pill shape.
    static let capsule: CGFloat = 980
}

extension RoundedRectangle {
    static var control: RoundedRectangle { RoundedRectangle(cornerRadius: Radius.control, style: .continuous) }
    static var card: RoundedRectangle { RoundedRectangle(cornerRadius: Radius.card, style: .continuous) }
}
