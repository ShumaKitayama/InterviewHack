// DesignDocument/3_UIUXDesign.md 3.4.5 共有リンクフロー / 3.5.14 共有リンク管理（S-ES-04）/ D12
import InterviewHackKit
import SwiftUI

struct ShareLinksView: View {
    @Environment(AppModel.self) private var model
    let project: ESProject
    let sections: [ESSection]

    @ViewState private var links: [ESShareLink] = []
    @ViewState private var selectedSections: Set<ID> = []
    @ViewState private var expiry: ShareLinkExpiry = .default
    @ViewState private var showCompanyName = false
    @ViewState private var created: CreatedShareLink?
    @ViewState private var revoking: ESShareLink?
    @ViewState private var error: Error?

    var body: some View {
        Form {
            if let created {
                Section("発行した URL") {
                    Text(created.url).font(.callout).textSelection(.enabled)
                    Text("この URL はこの画面でしか表示されません。発行時点の内容が固定され、\(created.expiresAt.formatted(date: .abbreviated, time: .shortened)) まで閲覧できます。")
                        .font(.footnote).foregroundStyle(.secondary)
                    ShareLink(item: created.url) { Label("LINE などで送る", systemImage: "square.and.arrow.up") }
                    Button("コピー") { Clipboard.copy(created.url) }
                }
            }
            Section("新しく発行する") {
                ForEach(sections) { section in
                    Toggle(section.title, isOn: Binding(get: { selectedSections.contains(section.id) }, set: { on in
                        if on { selectedSections.insert(section.id) } else { selectedSections.remove(section.id) }
                    }))
                }
                Picker("有効期限", selection: $expiry) {
                    ForEach(ShareLinkExpiry.allCases, id: \.self) { Text($0.label).tag($0) }
                }
                Toggle("企業名を表示する", isOn: $showCompanyName)
                AsyncButton("発行") {
                    do {
                        created = try await model.api.createShareLink(projectId: project.id, ShareLinkRequest(
                            sectionIds: Array(selectedSections), expiresInDays: expiry, showCompanyName: showCompanyName))
                        error = nil
                        await load()
                    } catch {
                        self.error = error
                    }
                }
                .disabled(selectedSections.isEmpty || !model.connectivity.isOnline)
            }
            if let error { ErrorBanner(error: error) }
            Section("発行済みのリンク") {
                if links.isEmpty { Text("まだありません").foregroundStyle(.secondary) }
                ForEach(links) { link in
                    VStack(alignment: .leading, spacing: 4) {
                        HStack {
                            Text(link.state().label).font(.headline)
                                .foregroundStyle(link.state() == .active ? .green : .secondary)
                            Spacer()
                            if link.state() == .active {
                                Button("無効にする", role: .destructive) { revoking = link }
                                    .buttonStyle(.borderless)
                            }
                        }
                        Text(link.sectionTitles.joined(separator: "・")).font(.callout)
                        Text("期限 \(link.expiresAt.formatted(date: .abbreviated, time: .shortened))")
                            .font(.caption).foregroundStyle(.secondary)
                    }
                }
            }
        }
        .navigationTitle("共有リンク")
        .task { await load() }
        .confirmationDialog("このリンクを無効にしますか？", isPresented: Binding(get: { revoking != nil }, set: { if !$0 { revoking = nil } }),
                            titleVisibility: .visible) {
            Button("無効にする", role: .destructive) {
                guard let link = revoking else { return }
                Task {
                    do {
                        try await model.api.revokeShareLink(link.id)
                        await load()
                    } catch {
                        self.error = error
                    }
                }
            }
        }
    }

    private func load() async {
        do {
            links = try await model.api.shareLinks(projectId: project.id)
        } catch {
            self.error = error
        }
    }
}
