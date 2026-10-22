// DesignDocument/3_UIUXDesign.md 3.4.1 / 3.5.2 面接対策ルーム（S-Prep-01）/ 3.5.3 取り込み結果（S-Prep-02）
import InterviewHackKit
import SwiftUI

struct PrepRoomView: View {
    @Environment(AppModel.self) private var model
    @ViewState private var companies: [Company] = []
    @ViewState private var companyId: ID?
    @ViewState private var assetIds: Set<ID> = []
    @ViewState private var materialIds: Set<ID> = []
    @ViewState private var questionCount = 10
    @ViewState private var flow: AILinkFlow?
    @ViewState private var savedQAIds: [ID]?
    @ViewState private var creatingCompany = false

    var body: some View {
        Form {
            Section("企業") {
                Picker("企業", selection: $companyId) {
                    Text("選択してください").tag(ID?.none)
                    ForEach(companies) { Text($0.name).tag(ID?.some($0.id)) }
                }
                if let company = companies.first(where: { $0.id == companyId }) {
                    LabeledContent("職種", value: company.jobTitle ?? "未設定")
                }
                Button("企業を新しく登録") { creatingCompany = true }
            }
            Section("材料") {
                NavigationLink {
                    AssetSelectionView(selection: $assetIds)
                } label: {
                    LabeledContent("使う自己資産", value: "\(assetIds.count) 件")
                }
                if let companyId {
                    NavigationLink {
                        MaterialSelectionView(companyId: companyId, selection: $materialIds)
                    } label: {
                        LabeledContent("使う企業研究資料", value: "\(materialIds.count) 件")
                    }
                }
                Stepper("質問数：\(questionCount)", value: $questionCount, in: 3...20)
            }
            Section {
                Button("プロンプトを作る") { start() }
                    .frame(minHeight: 44)
                    .disabled(companyId == nil || assetIds.isEmpty || !model.connectivity.isOnline)
                if companyId == nil || assetIds.isEmpty {
                    Text(companyId == nil ? "企業を選んでください" : "自己資産を 1 件以上選んでください")
                        .font(.footnote).foregroundStyle(.orange)
                }
            }
        }
        .navigationTitle("面接対策ルーム")
        .task { companies = (try? await model.api.companies().items) ?? [] }
        .sheet(isPresented: $creatingCompany) {
            NavigationStack {
                CompanyEditView(company: nil) { company in
                    companies.append(company)
                    companyId = company.id
                }
            }
        }
        .aiLinkSheet($flow, title: UseCase.qaGenerate.label) { response in
            savedQAIds = response.savedRefs["qa"] ?? []
        }
        .navigationDestination(item: Binding(get: { savedQAIds.map(SavedIDs.init) }, set: { savedQAIds = $0?.ids })) { saved in
            QAImportResultView(ids: saved.ids, assetIds: Array(assetIds))
        }
    }

    private func start() {
        guard let companyId else { return }
        let flow = model.makeFlow(.qaGenerate)
        self.flow = flow
        Task {
            await flow.start(PromptSessionRequest(useCase: .qaGenerate, companyId: companyId, assetIds: Array(assetIds),
                                                  researchMaterialIds: Array(materialIds),
                                                  options: PromptOptions(questionCount: questionCount)))
        }
    }
}

struct SavedIDs: Identifiable, Hashable {
    let ids: [ID]
    var id: String { ids.joined(separator: ",") }
}

/// 使う企業研究資料（要約済みのものを優先表示する）
struct MaterialSelectionView: View {
    @Environment(AppModel.self) private var model
    let companyId: ID
    @Binding var selection: Set<ID>
    @ViewState private var materials: [ResearchMaterial] = []

    var body: some View {
        List {
            if materials.isEmpty {
                Text("資料がありません。「データ」タブの企業から取り込めます").foregroundStyle(.secondary)
            }
            ForEach(materials.sorted { ($0.summary != nil ? 0 : 1) < ($1.summary != nil ? 0 : 1) }) { material in
                Toggle(isOn: Binding(get: { selection.contains(material.id) }, set: { on in
                    if on { selection.insert(material.id) } else { selection.remove(material.id) }
                })) {
                    VStack(alignment: .leading) {
                        Text(material.title)
                        Text(material.summary != nil ? "要約済み" : "要約なし（先頭 \(ResearchLengthAdvice.promptExcerptLimit) 字を使います）")
                            .font(.caption).foregroundStyle(.secondary)
                    }
                }
            }
        }
        .navigationTitle("使う企業研究資料")
        .task { materials = (try? await model.api.researchMaterials(companyId: companyId).items) ?? [] }
    }
}

/// S-Prep-02 想定Q&Aの取り込み結果
struct QAImportResultView: View {
    @Environment(AppModel.self) private var model
    let ids: [ID]
    let assetIds: [ID]
    @ViewState private var items: [QA] = []
    @ViewState private var assets: [Asset] = []

    var body: some View {
        List {
            Section("保存した Q&A（\(items.count) 件）") {
                ForEach($items) { $qa in
                    DisclosureGroup {
                        Text(qa.answerText).font(.callout)
                    } label: {
                        HStack {
                            Text(qa.questionText).font(.headline)
                            Spacer()
                            Button {
                                Task {
                                    if let updated = try? await model.api.updateQA(qa.id, QAInput(isFavorite: !qa.isFavorite)) {
                                        qa = updated
                                    }
                                }
                            } label: {
                                Image(systemName: qa.isFavorite ? "star.fill" : "star")
                                    .accessibilityLabel(qa.isFavorite ? "お気に入りを外す" : "お気に入り")
                            }
                            .buttonStyle(.borderless)
                        }
                    }
                }
            }
            if !assets.isEmpty {
                Section("使った自己資産") {
                    ForEach(assets.filter { assetIds.contains($0.id) }) { asset in
                        Text("\(asset.title)（\(asset.type.label)）").font(.callout)
                    }
                }
            }
            Section {
                NavigationLink("Q&Aバンクを開く") { QAListView() }
                NavigationLink("条件を変えて作り直す") { PrepRoomView() }
            }
        }
        .navigationTitle("取り込み結果")
        .task {
            var loaded: [QA] = []
            for id in ids {
                if let qa = try? await model.api.qa(id) { loaded.append(qa) }
            }
            items = loaded
            assets = (try? model.cache.load(Page<Asset>.self, for: .assets))?.value.items ?? []
        }
    }
}
