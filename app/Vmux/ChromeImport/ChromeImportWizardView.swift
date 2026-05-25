import SwiftUI

/// Sheet that imports a chosen Chrome profile's bookmarks into the current
/// workspace's browser profile. (Cookies/history are a later pass.)
struct ChromeImportWizardView: View {
    let workspaceID: String
    let onClose: () -> Void
    @StateObject private var importer: ChromeImporter
    @State private var importing = false

    init(client: SidecarClient, workspaceID: String, onClose: @escaping () -> Void) {
        self.workspaceID = workspaceID
        self.onClose = onClose
        _importer = StateObject(wrappedValue: ChromeImporter(client: client))
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            Text("Import from Chrome").font(Typo.titleBar).foregroundStyle(Theme.label)
            Text("Bring your Chrome bookmarks into this workspace's browser profile.")
                .font(Typo.caption).foregroundStyle(Theme.label2)

            if importer.profiles.isEmpty {
                Text("No Chrome profiles found").font(Typo.caption2).foregroundStyle(Theme.label3)
            } else {
                ForEach(importer.profiles) { profile in
                    profileRow(profile)
                }
            }

            if !importer.status.isEmpty {
                Text(importer.status).font(Typo.caption2).foregroundStyle(Theme.accent)
            }

            Spacer()
            HStack {
                Spacer()
                CapsuleButton(title: "Done", style: .filled, action: onClose)
            }
        }
        .padding(16)
        .frame(width: 440, height: 360)
        .background(Theme.content)
        .task { await importer.scan() }
    }

    private func profileRow(_ profile: ChromeProfile) -> some View {
        HStack(spacing: 9) {
            Image(systemName: "person.crop.circle").foregroundStyle(Theme.label2)
            VStack(alignment: .leading, spacing: 1) {
                Text(profile.name).font(Typo.caption).foregroundStyle(Theme.label)
                Text(dataSummary(profile)).font(.system(size: 10)).foregroundStyle(Theme.label3)
            }
            Spacer()
            CapsuleButton(title: importing ? "…" : "Import bookmarks", style: .plain) {
                importing = true
                Task {
                    await importer.importBookmarks(profile: profile, workspaceID: workspaceID)
                    importing = false
                }
            }
            .disabled(importing || !profile.hasBookmarks)
        }
        .padding(10)
        .background(RoundedRectangle.control.fill(Theme.card))
    }

    private func dataSummary(_ p: ChromeProfile) -> String {
        var parts: [String] = []
        if p.hasBookmarks { parts.append("bookmarks") }
        if p.hasCookies { parts.append("cookies") }
        if p.hasHistory { parts.append("history") }
        return parts.isEmpty ? "no data" : parts.joined(separator: " · ")
    }
}
