// DesignDocument/3_UIUXDesign.md S-Data-03 / 04 / 05 企業・企業研究資料 / 1_Requirementdefinition.md 1.1.3
import InterviewHackKit
import SwiftUI

/// S-Data-03 企業DB一覧
struct CompanyListView: View {
    @Environment(AppModel.self) private var model
    @ViewState private var companies: CachedResource<Page<Company>>?
    @ViewState private var keyword = ""
    @ViewState private var creating = false

    var body: some View {
        List {
            if let error = companies?.error, companies?.value == nil {
                ErrorBanner(error: error) { Task { await companies?.refresh() } }
            }
            let items = (companies?.value?.items ?? []).filter {
                keyword.isEmpty || $0.name.localizedCaseInsensitiveContains(keyword)
            }
            if items.isEmpty && companies?.isRefreshing == false {
                EmptyStateView(title: "企業がありません", message: "企業名と職種だけで登録できます。登録したらすぐ想定Q&Aを作れます",
                               actionTitle: "登録する") { creating = true }
            }
            ForEach(items) { company in
                NavigationLink {
                    CompanyDetailView(company: company) { Task { await companies?.refresh() } }
                } label: {
                    VStack(alignment: .leading) {
                        Text(company.name).font(.headline)
                        Text([company.jobTitle, company.selectionPhase].compactMap { $0 }.joined(separator: " / "))
                            .font(.caption).foregroundStyle(.secondary)
                    }
                }
            }
            .onDelete { offsets in
                let targets = offsets.map { items[$0] }
                Task {
                    for company in targets { try? await model.api.deleteCompany(company.id) }
                    await companies?.refresh()
                }
            }
        }
        .searchable(text: $keyword)
        .navigationTitle("企業")
        .toolbar {
            ToolbarItem(placement: ToolbarPlacement.trailing) {
                Button { creating = true } label: { Image(systemName: "plus").accessibilityLabel("追加") }
                    .disabled(!model.connectivity.isOnline)
            }
        }
        .task {
            let api = model.api
            if companies == nil { companies = model.resource(.companies) { try await api.companies() } }
            await companies?.refresh()
        }
        .refreshable { await companies?.refresh() }
        .sheet(isPresented: $creating) {
            NavigationStack { CompanyEditView(company: nil) { _ in Task { await companies?.refresh() } } }
        }
    }
}

/// S-Data-04 企業作成/編集
struct CompanyEditView: View {
    @Environment(AppModel.self) private var model
    @Environment(\.dismiss) private var dismiss
    let company: Company?
    let onSaved: (Company) -> Void

    @ViewState private var name = ""
    @ViewState private var jobTitle = ""
    @ViewState private var selectionPhase = ""
    @ViewState private var motivationMemo = ""
    @ViewState private var referenceUrl = ""
    @ViewState private var error: Error?

    var body: some View {
        Form {
            TextField("企業名", text: $name)
            TextField("職種", text: $jobTitle)
            TextField("選考フェーズ（例：一次面接）", text: $selectionPhase)
            TextField("URL", text: $referenceUrl).urlInput()
            Section("志望動機メモ") {
                TextEditor(text: $motivationMemo).frame(minHeight: 120)
            }
            if let error { ErrorBanner(error: error) }
        }
        .navigationTitle(company == nil ? "企業の登録" : "企業の編集")
        .toolbar {
            ToolbarItem(placement: .cancellationAction) { Button("キャンセル") { dismiss() } }
            ToolbarItem(placement: .confirmationAction) {
                AsyncButton("保存") { await save() }
                    .disabled(name.trimmingCharacters(in: .whitespaces).isEmpty)
            }
        }
        .onAppear {
            guard let company else { return }
            name = company.name
            jobTitle = company.jobTitle ?? ""
            selectionPhase = company.selectionPhase ?? ""
            motivationMemo = company.motivationMemo ?? ""
            referenceUrl = company.referenceUrl ?? ""
        }
    }

    private func save() async {
        let input = CompanyInput(name: name, jobTitle: jobTitle.nilIfBlank, selectionPhase: selectionPhase.nilIfBlank,
                                 motivationMemo: motivationMemo.nilIfBlank, referenceUrl: referenceUrl.nilIfBlank)
        do {
            let saved: Company
            if let company {
                saved = try await model.api.updateCompany(company.id, input)
            } else {
                saved = try await model.api.createCompany(input)
            }
            onSaved(saved)
            dismiss()
        } catch {
            self.error = error
        }
    }
}

struct CompanyDetailView: View {
    @ViewState var company: Company
    let onChange: () -> Void
    @ViewState private var editing = false

    var body: some View {
        List {
            Section {
                LabeledContent("職種", value: company.jobTitle ?? "-")
                LabeledContent("選考フェーズ", value: company.selectionPhase ?? "-")
                if let memo = company.motivationMemo, !memo.isEmpty {
                    Text(memo).font(.callout)
                }
            }
            Section {
                NavigationLink {
                    ResearchMaterialsView(company: company)
                } label: {
                    Label("企業研究資料", systemImage: "doc.richtext")
                }
                NavigationLink {
                    QAListView(filter: QAFilter(companyId: company.id))
                } label: {
                    Label("この企業のQ&A", systemImage: "rectangle.stack")
                }
                NavigationLink {
                    InterviewListView(companyId: company.id)
                } label: {
                    Label("面接履歴", systemImage: "clock")
                }
            }
        }
        .navigationTitle(company.name)
        .toolbar {
            ToolbarItem(placement: ToolbarPlacement.trailing) { Button("編集") { editing = true } }
        }
        .sheet(isPresented: $editing) {
            NavigationStack {
                CompanyEditView(company: company) { saved in
                    company = saved
                    onChange()
                }
            }
        }
    }
}

/// S-Data-05 企業研究資料管理（資料一覧・要約確認）
struct ResearchMaterialsView: View {
    @Environment(AppModel.self) private var model
    let company: Company
    @ViewState private var materials: CachedResource<Page<ResearchMaterial>>?
    @ViewState private var importing: ImportKind?
    @ViewState private var summarizeFlow: AILinkFlow?

    var body: some View {
        List {
            Section {
                Button { importing = .url } label: { Label("Webページを取り込む", systemImage: "safari") }
                Button { importing = .file } label: { Label("PDF・画像を取り込む", systemImage: "doc.viewfinder") }
                Button { importing = .note } label: { Label("メモを書く", systemImage: "note.text") }
            }
            .disabled(!model.connectivity.isOnline)
            if let error = materials?.error, materials?.value == nil {
                ErrorBanner(error: error) { Task { await materials?.refresh() } }
            }
            ForEach(materials?.value?.items ?? []) { material in
                NavigationLink {
                    ResearchMaterialDetailView(material: material) { summarize(material) }
                } label: {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(material.title).font(.headline)
                        HStack {
                            Text(material.type.label)
                            if material.summary != nil { Text("要約済み").foregroundStyle(.green) }
                        }
                        .font(.caption)
                    }
                }
            }
            .onDelete { offsets in
                let items = materials?.value?.items ?? []
                let targets = offsets.map { items[$0] }
                Task {
                    for material in targets { try? await model.api.deleteResearchMaterial(material.id) }
                    await materials?.refresh()
                }
            }
        }
        .navigationTitle("企業研究資料")
        .task {
            let api = model.api
            let companyId = company.id
            if materials == nil {
                materials = model.resource(.researchMaterials(companyId: companyId)) { try await api.researchMaterials(companyId: companyId) }
            }
            await materials?.refresh()
        }
        .sheet(item: $importing) { kind in
            NavigationStack {
                switch kind {
                case .url: URLImportView(companyId: company.id) { Task { await materials?.refresh() } }
                case .file: FileImportView(companyId: company.id) { Task { await materials?.refresh() } }
                case .note:
                    MaterialTextEditorView(companyId: company.id, type: .note, sourceUrl: nil, initialText: "") {
                        Task { await materials?.refresh() }
                    }
                }
            }
        }
        .aiLinkSheet($summarizeFlow, title: UseCase.researchSummarize.label) { _ in Task { await materials?.refresh() } }
    }

    private func summarize(_ material: ResearchMaterial) {
        let flow = model.makeFlow(.researchSummarize)
        summarizeFlow = flow
        Task {
            await flow.start(PromptSessionRequest(useCase: .researchSummarize, companyId: company.id,
                                                  researchMaterialIds: [material.id]))
        }
    }
}

enum ImportKind: String, Identifiable {
    case url, file, note
    var id: String { rawValue }
}

struct ResearchMaterialDetailView: View {
    let material: ResearchMaterial
    let onSummarize: () -> Void

    var body: some View {
        List {
            if let source = material.sourceUrl {
                Section("取り込み元") { Text(source).font(.caption).textSelection(.enabled) }
            }
            Section("要約") {
                if let summary = material.summary {
                    Text(summary).font(.callout).textSelection(.enabled)
                } else {
                    Text("まだ要約していません").foregroundStyle(.secondary)
                }
                Button("要約する（AI連携）", action: onSummarize)
            }
            Section("本文") {
                Text(material.contentRaw ?? "").font(.callout).textSelection(.enabled)
            }
        }
        .navigationTitle(material.title)
    }
}

extension String {
    var nilIfBlank: String? {
        trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? nil : self
    }
}
