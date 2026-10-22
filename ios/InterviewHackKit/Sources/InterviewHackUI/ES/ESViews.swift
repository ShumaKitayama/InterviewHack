// DesignDocument/3_UIUXDesign.md 3.4.2 ES作成フロー / 3.5.4〜3.5.6.1（S-ES-01〜03, 05）
import InterviewHackKit
import SwiftUI

/// S-ES-01 ESプロジェクト一覧
struct ESProjectListView: View {
    @Environment(AppModel.self) private var model
    @ViewState private var projects: CachedResource<Page<ESProject>>?
    @ViewState private var companies: [Company] = []
    @ViewState private var statusFilter: ESStatus?
    @ViewState private var creating = false
    @ViewState private var pendingStatus: (ESProject, ESStatus)?

    var body: some View {
        List {
            Picker("ステータス", selection: $statusFilter) {
                Text("すべて").tag(ESStatus?.none)
                ForEach(ESStatus.allCases, id: \.self) { Text($0.label).tag(ESStatus?.some($0)) }
            }
            if let error = projects?.error, projects?.value == nil {
                ErrorBanner(error: error) { Task { await projects?.refresh() } }
            }
            let items = sortedProjects
            if items.isEmpty && projects?.isRefreshing == false {
                EmptyStateView(title: "ESがありません", message: "企業を選んで ES を作成しましょう",
                               actionTitle: "新しいESを作成") { creating = true }
            }
            ForEach(items) { project in
                NavigationLink {
                    ESEditorView(project: project, companyName: companyName(project.companyId))
                } label: {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(companyName(project.companyId)).font(.headline)
                        HStack {
                            Text(project.status.label)
                            if let deadline = project.deadline { Text("締切 \(deadline.description)") }
                        }
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    }
                }
                .contextMenu {
                    ForEach(ESStatus.allCases, id: \.self) { status in
                        Button(status.label) { changeStatus(project, to: status) }
                    }
                }
            }
            .onDelete { offsets in
                let targets = offsets.map { items[$0] }
                Task {
                    for project in targets { try? await model.api.deleteESProject(project.id) }
                    await projects?.refresh()
                }
            }
        }
        .navigationTitle("ES作成ルーム")
        .toolbar {
            ToolbarItem(placement: ToolbarPlacement.trailing) {
                Button { creating = true } label: { Image(systemName: "plus").accessibilityLabel("新しいESを作成") }
                    .disabled(!model.connectivity.isOnline)
            }
        }
        .task {
            let api = model.api
            if projects == nil { projects = model.resource(.esProjects) { try await api.esProjects() } }
            companies = (try? model.cache.load(Page<Company>.self, for: .companies))?.value.items ?? []
            if let fresh = try? await api.companies() { companies = fresh.items }
            await projects?.refresh()
        }
        .refreshable { await projects?.refresh() }
        .sheet(isPresented: $creating) {
            NavigationStack {
                ESProjectCreateView(companies: companies) { Task { await projects?.refresh() } }
            }
        }
        // 3.5.4：「提出済」を選ぶときだけ確認ダイアログを出す
        .confirmationDialog("提出済にしますか？", isPresented: Binding(get: { pendingStatus != nil }, set: { if !$0 { pendingStatus = nil } }),
                            titleVisibility: .visible) {
            Button("提出済にする") {
                if let (project, status) = pendingStatus { applyStatus(project, status) }
                pendingStatus = nil
            }
        }
    }

    private var sortedProjects: [ESProject] {
        (projects?.value?.items ?? [])
            .filter { statusFilter == nil || $0.status == statusFilter }
            .sorted { ($0.deadline ?? CalendarDate(year: 9999, month: 12, day: 31)) < ($1.deadline ?? CalendarDate(year: 9999, month: 12, day: 31)) }
    }

    private func companyName(_ id: ID) -> String {
        companies.first { $0.id == id }?.name ?? "企業"
    }

    private func changeStatus(_ project: ESProject, to status: ESStatus) {
        if status.requiresConfirmation {
            pendingStatus = (project, status)
        } else {
            applyStatus(project, status)
        }
    }

    private func applyStatus(_ project: ESProject, _ status: ESStatus) {
        Task {
            _ = try? await model.api.updateESProject(project.id, ESProjectInput(status: status))
            await projects?.refresh()
        }
    }
}

struct ESProjectCreateView: View {
    @Environment(AppModel.self) private var model
    @Environment(\.dismiss) private var dismiss
    let companies: [Company]
    let onSaved: () -> Void
    @ViewState private var companyId: ID?
    @ViewState private var hasDeadline = true
    @ViewState private var deadline = Date()
    @ViewState private var error: Error?

    var body: some View {
        Form {
            Picker("企業", selection: $companyId) {
                Text("選択してください").tag(ID?.none)
                ForEach(companies) { Text($0.name).tag(ID?.some($0.id)) }
            }
            Toggle("提出期限を設定する", isOn: $hasDeadline)
            if hasDeadline {
                DatePicker("提出期限", selection: $deadline, displayedComponents: .date)
            }
            if let error { ErrorBanner(error: error) }
        }
        .navigationTitle("新しいES")
        .toolbar {
            ToolbarItem(placement: .cancellationAction) { Button("キャンセル") { dismiss() } }
            ToolbarItem(placement: .confirmationAction) {
                AsyncButton("作成") {
                    guard let companyId else { return }
                    do {
                        _ = try await model.api.createESProject(ESProjectInput(
                            companyId: companyId, deadline: hasDeadline ? CalendarDate(date: deadline) : nil))
                        onSaved()
                        dismiss()
                    } catch {
                        self.error = error
                    }
                }
                .disabled(companyId == nil)
            }
        }
    }
}

/// S-ES-02 ESエディタ
struct ESEditorView: View {
    @Environment(AppModel.self) private var model
    let project: ESProject
    let companyName: String

    enum Segment: String, CaseIterable { case sections = "ESセクション", materials = "企業研究資料" }

    @ViewState private var segment: Segment = .sections
    @ViewState private var sections: CachedResource<Page<ESSection>>?
    @ViewState private var addingSection = false
    @ViewState private var draftFlow: AILinkFlow?
    @ViewState private var draftTarget: ESSection?
    @ViewState private var assetIds: Set<ID> = []
    @ViewState private var materialIds: Set<ID> = []
    @ViewState private var message: String?

    var body: some View {
        List {
            Picker("表示", selection: $segment) {
                ForEach(Segment.allCases, id: \.self) { Text($0.rawValue).tag($0) }
            }
            .pickerStyle(.segmented)

            switch segment {
            case .sections:
                if let message { Text(message).font(.callout).foregroundStyle(.green) }
                Section("ドラフトに使う材料") {
                    NavigationLink { AssetSelectionView(selection: $assetIds) } label: {
                        LabeledContent("自己資産", value: "\(assetIds.count) 件")
                    }
                    NavigationLink { MaterialSelectionView(companyId: project.companyId, selection: $materialIds) } label: {
                        LabeledContent("企業研究資料", value: "\(materialIds.count) 件")
                    }
                }
                ForEach(sections?.value?.items ?? []) { section in
                    Section {
                        NavigationLink {
                            ESSectionEditorView(section: section) { Task { await sections?.refresh() } }
                        } label: {
                            VStack(alignment: .leading, spacing: 4) {
                                HStack {
                                    Text(section.title).font(.headline)
                                    Spacer()
                                    CharacterCounterView(text: section.contentDraft ?? "", limit: section.characterLimit)
                                }
                                Text(section.contentDraft ?? "まだ本文がありません").font(.callout).lineLimit(3)
                                    .foregroundStyle(section.contentDraft == nil ? .secondary : .primary)
                            }
                        }
                        Button("ドラフトを作る（AI連携）") { startDraft(section) }
                            .disabled(!model.connectivity.isOnline)
                        NavigationLink("推敲する") {
                            ESRefineView(project: project, section: section) { Task { await sections?.refresh() } }
                        }
                        Menu("その他") {
                            Button("この版を保存") { saveVersion(section) }
                            NavigationLink("版の履歴") {
                                VersionHistoryView(section: section) { Task { await sections?.refresh() } }
                            }
                        }
                    }
                }
                Button { addingSection = true } label: { Label("セクションを追加", systemImage: "plus") }
                    .disabled(!model.connectivity.isOnline)
            case .materials:
                NavigationLink("企業研究資料を管理・取り込む") {
                    ResearchMaterialsView(company: Company(id: project.companyId, name: companyName))
                }
            }
        }
        .navigationTitle(companyName)
        .toolbar {
            ToolbarItem(placement: ToolbarPlacement.trailing) {
                NavigationLink {
                    ShareLinksView(project: project, sections: sections?.value?.items ?? [])
                } label: {
                    Image(systemName: "link").accessibilityLabel("共有リンク")
                }
            }
        }
        .task {
            let api = model.api
            let projectId = project.id
            if sections == nil { sections = model.resource(.esSections(projectId: projectId)) { try await api.esSections(projectId: projectId) } }
            assetIds = Set(project.assetIds ?? [])
            materialIds = Set(project.researchMaterialIds ?? [])
            await sections?.refresh()
        }
        .sheet(isPresented: $addingSection) {
            NavigationStack { ESSectionCreateView(projectId: project.id) { Task { await sections?.refresh() } } }
        }
        .aiLinkSheet($draftFlow, title: UseCase.esGenerate.label) { _ in
            Task { await sections?.refresh() }
        }
    }

    private func startDraft(_ section: ESSection) {
        let flow = model.makeFlow(.esGenerate)
        flow.originalText = section.contentDraft
        draftFlow = flow
        Task {
            await flow.start(PromptSessionRequest(useCase: .esGenerate, esProjectId: project.id, esSectionId: section.id,
                                                  assetIds: Array(assetIds), researchMaterialIds: Array(materialIds)))
        }
    }

    private func saveVersion(_ section: ESSection) {
        Task {
            do {
                _ = try await model.api.saveESSectionVersion(sectionId: section.id)
                message = "「\(section.title)」の版を保存しました"
            } catch {
                message = (error as? LocalizedError)?.errorDescription
            }
        }
    }
}

struct ESSectionCreateView: View {
    @Environment(AppModel.self) private var model
    @Environment(\.dismiss) private var dismiss
    let projectId: ID
    let onSaved: () -> Void
    @ViewState private var title = "志望動機"
    @ViewState private var limitText = "400"
    @ViewState private var error: Error?

    private let suggestions = ["志望動機", "自己PR", "ガクチカ", "長所・短所"]

    var body: some View {
        Form {
            TextField("項目名", text: $title)
            ScrollView(.horizontal, showsIndicators: false) {
                HStack {
                    ForEach(suggestions, id: \.self) { suggestion in
                        Button(suggestion) { title = suggestion }.buttonStyle(.bordered)
                    }
                }
            }
            TextField("文字数制限（任意）", text: $limitText)
                #if os(iOS)
                .keyboardType(.numberPad)
                #endif
            if let error { ErrorBanner(error: error) }
        }
        .navigationTitle("セクションを追加")
        .toolbar {
            ToolbarItem(placement: .cancellationAction) { Button("キャンセル") { dismiss() } }
            ToolbarItem(placement: .confirmationAction) {
                AsyncButton("追加") {
                    do {
                        _ = try await model.api.createESSection(projectId: projectId,
                                                                ESSectionInput(title: title, characterLimit: Int(limitText)))
                        onSaved()
                        dismiss()
                    } catch {
                        self.error = error
                    }
                }
                .disabled(title.trimmingCharacters(in: .whitespaces).isEmpty)
            }
        }
    }
}

/// ESセクションの執筆（文字数カウント、下書きを端末に保存。5.8.3）
struct ESSectionEditorView: View {
    @Environment(AppModel.self) private var model
    @ViewState var section: ESSection
    let onChange: () -> Void
    @ViewState private var text = ""
    @ViewState private var hasLocalDraft = false
    @ViewState private var error: Error?
    @ViewState private var saved = false

    var body: some View {
        Form {
            if hasLocalDraft {
                Section {
                    Label("端末に送信していない下書きがあります。内容を確認して「保存」で送信してください", systemImage: "tray.and.arrow.up")
                        .font(.callout)
                }
            }
            Section {
                TextEditor(text: $text)
                    .frame(minHeight: 300)
                    .onChange(of: text) { _, newValue in
                        try? model.drafts.save(TextDraft(text: newValue), for: .esSection(sectionId: section.id))
                        hasLocalDraft = newValue != (section.contentDraft ?? "")
                    }
            } header: {
                HStack {
                    Text(section.title)
                    Spacer()
                    CharacterCounterView(text: text, limit: section.characterLimit)
                }
            }
            if let error { ErrorBanner(error: error) }
            if saved { Label("保存しました", systemImage: "checkmark") }
            AsyncButton("保存") {
                do {
                    section = try await model.api.updateESSection(section.id, ESSectionInput(contentDraft: text))
                    try? model.drafts.remove(.esSection(sectionId: section.id))
                    hasLocalDraft = false
                    saved = true
                    onChange()
                } catch {
                    self.error = error
                }
            }
            .disabled(!model.connectivity.isOnline)
            Button("完成版にする") {
                Task {
                    if let updated = try? await model.api.updateESSection(section.id, ESSectionInput(contentDraft: text, contentFinal: text)) {
                        section = updated
                        try? model.drafts.remove(.esSection(sectionId: section.id))
                        hasLocalDraft = false
                        onChange()
                    }
                }
            }
            .disabled(!model.connectivity.isOnline)
            Section("出力") {
                Button("テキストをコピー") { Clipboard.copy(text) }
                ShareLink(item: text) { Label("共有・PDF出力（共有シート）", systemImage: "square.and.arrow.up") }
            }
        }
        .navigationTitle(section.title)
        .onAppear {
            if let draft = try? model.drafts.load(TextDraft.self, for: .esSection(sectionId: section.id)) {
                text = draft.text
                hasLocalDraft = draft.text != (section.contentDraft ?? "")
            } else {
                text = section.contentDraft ?? ""
            }
        }
    }
}

/// S-ES-03 ES推敲
struct ESRefineView: View {
    @Environment(AppModel.self) private var model
    let project: ESProject
    let section: ESSection
    let onSaved: () -> Void
    @ViewState private var instructions = ""
    @ViewState private var flow: AILinkFlow?

    private let chips = ["もっと具体的に", "結論ファーストで", "文字数を減らして", "熱意を強調して"]

    var body: some View {
        Form {
            Section {
                Text(section.contentDraft ?? "本文がありません").font(.callout)
            } header: {
                HStack {
                    Text("現在のドラフト")
                    Spacer()
                    CharacterCounterView(text: section.contentDraft ?? "", limit: section.characterLimit)
                }
            }
            Section("修正指示") {
                TextField("例：もっと具体的に", text: $instructions, axis: .vertical)
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack {
                        ForEach(chips, id: \.self) { chip in
                            Button(chip) {
                                instructions = instructions.isEmpty ? chip : instructions + "、" + chip
                            }
                            .buttonStyle(.bordered)
                        }
                    }
                }
            }
            Button("推敲プロンプトを作る") {
                let flow = model.makeFlow(.esRefine)
                flow.originalText = section.contentDraft
                self.flow = flow
                Task {
                    await flow.start(PromptSessionRequest(useCase: .esRefine, esProjectId: project.id, esSectionId: section.id,
                                                          options: PromptOptions(instructions: instructions)))
                }
            }
            .frame(minHeight: 44)
            .disabled(instructions.trimmingCharacters(in: .whitespaces).isEmpty || (section.contentDraft ?? "").isEmpty
                      || !model.connectivity.isOnline)
        }
        .navigationTitle("ES推敲")
        .aiLinkSheet($flow, title: UseCase.esRefine.label) { _ in onSaved() }
    }
}

/// S-ES-05 版の履歴
struct VersionHistoryView: View {
    @Environment(AppModel.self) private var model
    @Environment(\.dismiss) private var dismiss
    let section: ESSection
    let onRestored: () -> Void
    @ViewState private var versions: [ESSectionVersion] = []
    @ViewState private var selected: ESSectionVersion?
    @ViewState private var confirming: ESSectionVersion?
    @ViewState private var error: Error?

    var body: some View {
        List {
            if let error { ErrorBanner(error: error) }
            if versions.isEmpty {
                Text("まだ版がありません。AI連携の取り込みや「この版を保存」で版が残ります（最大20件）")
                    .foregroundStyle(.secondary)
            }
            ForEach(versions) { version in
                Button {
                    selected = version
                } label: {
                    VStack(alignment: .leading, spacing: 4) {
                        HStack {
                            Text(version.source.label).font(.headline)
                            Spacer()
                            if let createdAt = version.createdAt {
                                Text(createdAt.formatted(date: .abbreviated, time: .shortened)).font(.caption)
                            }
                        }
                        Text(version.content).font(.callout).lineLimit(2)
                    }
                }
                .foregroundStyle(.primary)
            }
            if let selected {
                Section("選んだ版と今の本文との差分") {
                    DiffTextView(old: selected.content, new: section.contentDraft ?? "")
                    Button("この版に戻す") { confirming = selected }
                        .disabled(!model.connectivity.isOnline)
                }
            }
        }
        .navigationTitle("版の履歴")
        .task {
            do {
                versions = try await model.api.esSectionVersions(sectionId: section.id)
            } catch {
                self.error = error
            }
        }
        .confirmationDialog("この版に戻しますか？今の本文は版として残ります", isPresented: Binding(get: { confirming != nil }, set: { if !$0 { confirming = nil } }),
                            titleVisibility: .visible) {
            Button("戻す") {
                guard let target = confirming else { return }
                Task {
                    do {
                        _ = try await model.api.restoreESSectionVersion(sectionId: section.id, versionId: target.id)
                        onRestored()
                        dismiss()
                    } catch {
                        self.error = error
                    }
                }
            }
        }
    }
}
