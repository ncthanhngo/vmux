import SwiftUI

/// Right-sidebar section listing active browser sessions with their latest
/// screenshot thumbnail and Focus/Close actions. Tapping a thumbnail opens the
/// full-size lightbox.
struct BrowserSessionsView: View {
    @ObservedObject var model: BrowserSessionModel
    @State private var lightboxImage: NSImage?

    var body: some View {
        GroupedCard(title: "Browser sessions") {
            if model.sessions.isEmpty {
                Text("No browser activity yet")
                    .font(Typo.caption2)
                    .foregroundStyle(Theme.label3)
            } else {
                VStack(alignment: .leading, spacing: 10) {
                    ForEach(model.sessions) { session in
                        sessionRow(session)
                    }
                }
            }
        }
        .sheet(item: Binding(
            get: { lightboxImage.map { LightboxItem(image: $0) } },
            set: { if $0 == nil { lightboxImage = nil } }
        )) { item in
            ScreenshotLightboxView(
                image: item.image,
                onClose: { lightboxImage = nil },
                onSave: { saveToDesktop(item.image) }
            )
        }
    }

    @ViewBuilder private func sessionRow(_ session: BrowserSession) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 7) {
                Circle().fill(Theme.green).frame(width: 7, height: 7)
                Text(session.id).font(Typo.caption2).foregroundStyle(Theme.label).lineLimit(1)
                Spacer()
                Text("\(session.shotCount) shots").font(.system(size: 10)).foregroundStyle(Theme.label3)
            }
            if let thumb = session.thumbnail {
                Image(nsImage: thumb)
                    .resizable()
                    .aspectRatio(contentMode: .fit)
                    .frame(maxHeight: 120)
                    .clipShape(RoundedRectangle(cornerRadius: 6, style: .continuous))
                    .onTapGesture { openLightbox(session) }
            }
            HStack(spacing: 7) {
                CapsuleButton(title: "Focus", style: .plain) { model.focus() }
                CapsuleButton(title: "Close", style: .tinted(Theme.red)) { model.close(session.id) }
            }
        }
    }

    private func openLightbox(_ session: BrowserSession) {
        Task {
            if let full = await model.loadFullShot(session.id) { lightboxImage = full }
        }
    }

    private func saveToDesktop(_ image: NSImage) {
        guard let tiff = image.tiffRepresentation,
              let rep = NSBitmapImageRep(data: tiff),
              let png = rep.representation(using: .png, properties: [:]) else { return }
        let desktop = FileManager.default.urls(for: .desktopDirectory, in: .userDomainMask)[0]
        let url = desktop.appendingPathComponent("vmux-screenshot-\(Int(Date().timeIntervalSince1970)).png")
        try? png.write(to: url)
    }
}

private struct LightboxItem: Identifiable {
    let id = UUID()
    let image: NSImage
}
