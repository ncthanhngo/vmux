import SwiftUI

/// Full-size modal screenshot viewer, opened from a session thumbnail.
struct ScreenshotLightboxView: View {
    let image: NSImage
    let onClose: () -> Void
    let onSave: () -> Void

    var body: some View {
        VStack(spacing: 0) {
            HStack {
                Text("Screenshot").font(Typo.titleBar).foregroundStyle(Theme.label)
                Spacer()
                CapsuleButton(title: "Save to Desktop", style: .plain, action: onSave)
                CapsuleButton(title: "Close", style: .filled, action: onClose)
            }
            .padding(12)

            Image(nsImage: image)
                .resizable()
                .aspectRatio(contentMode: .fit)
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .padding(12)
        }
        .frame(minWidth: 640, minHeight: 480)
        .background(Theme.content)
    }
}
