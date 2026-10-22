// DesignDocument/3_UIUXDesign.md S-Data-01 / S-Data-02 自己資産 / 1_Requirementdefinition.md 1.1.2
import InterviewHackKit
import SwiftUI

/// S-Data-01 自己資産DB一覧
struct AssetListView: View {
    @Environment(AppModel.self) private var model
    @ViewState private var assets: CachedResource<Page<Asset>>?
    @ViewState private var tags: CachedResource<Page<AssetTag>>?
    @ViewState private var typeFilter: AssetType?
    @ViewState private var tagFilter: ID?
    @ViewState private var keyword = ""
    @ViewState private var editing: AssetEditTarget?

    var body: some View {
        List {
            if let error = assets?.error, assets?.value == nil {
                ErrorBanner(error: error) { Task { await assets?.refresh() } }
            }
            Section {
                Picker("種別", selection: $typeFilter) {
                    Text("すべて").tag(AssetType?.none)
                    ForEach(AssetType.allCases, id: \.self) { Text($0.label).tag(AssetType?.some($0)) }
                }
                Picker("タグ", selection: $tagFilter) {
                    Text("すべて").tag(ID?.none)
                    ForEach(tags?.value?.items ?? []) { Text($0.name).tag(ID?.some($0.id)) }
                }
                NavigationLink("タグを管理") { TagManagerView(onChange: { Task { await tags?.refresh() } }) }
            }
            let items = filtered
            if items.isEmpty && assets?.isRefreshing == false {
                EmptyStateView(title: "自己資産がありません",
                               message: "ガクチカ・強み・エピソードなどを最低3件登録すると、想定Q&AやESの材料になります",
                               actionTitle: "登録する") { editing = .new }
            }
            ForEach(items) { asset in
                Button {
                    editing = .existing(asset)
                } label: {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(asset.title).font(.headline)
                        Text(asset.type.label).font(.caption).foregroundStyle(.secondary)
                        Text(asset.content).font(.callout).lineLimit(2)
                    }
                    .frame(minHeight: 44, alignment: .leading)
                }
                .foregroundStyle(.primary)
            }
            .onDelete { offsets in
                let targets = offsets.map { items[$0] }
                Task {
                    for asset in targets { try? await model.api.deleteAsset(asset.id) }
                    await assets?.refresh()
                }
            }
        }
        .searchable(text: $keyword)
        .navigationTitle("自己資産")
        .toolbar {
            ToolbarItem(placement: ToolbarPlacement.trailing) {
                Button { editing = .new } label: { Image(systemName: "plus").accessibilityLabel("追加") }
                    .disabled(!model.connectivity.isOnline)
            }
        }
        .task {
            let api = model.api
            if assets == nil { assets = model.resource(.assets) { try await api.assets() } }
            if tags == nil { tags = model.resource(.assetTags) { try await api.assetTags() } }
            await assets?.refresh()
            await tags?.refresh()
        }
        .refreshable { await assets?.refresh() }
        .sheet(item: $editing) { target in
            NavigationStack {
                AssetEditView(target: target, tags: tags?.value?.items ?? []) {
                    Task { await assets?.refresh() }
                }
            }
        }
    }

    private var filtered: [Asset] {
        (assets?.value?.items ?? []).filter { asset in
            (typeFilter == nil || asset.type == typeFilter)
                && (tagFilter == nil || (asset.tagIds ?? []).contains(tagFilter!))
                && (keyword.isEmpty || asset.title.localizedCaseInsensitiveContains(keyword)
                    || asset.content.localizedCaseInsensitiveContains(keyword))
        }
    }
}

enum AssetEditTarget: Identifiable {
    case new
    case existing(Asset)

    var id: String {
        switch self {
        case .new: return "new"
        case .existing(let asset): return asset.id
        }
    }
}

/// S-Data-02 自己資産作成/編集
struct AssetEditView: View {
    @Environment(AppModel.self) private var model
    @Environment(\.dismiss) private var dismiss
    let target: AssetEditTarget
    let tags: [AssetTag]
    let onSaved: () -> Void

    @ViewState private var type: AssetType = .episode
    @ViewState private var title = ""
    @ViewState private var content = ""
    @ViewState private var selectedTags: Set<ID> = []
    @ViewState private var error: Error?

    var body: some View {
        Form {
            Picker("種別", selection: $type) {
                ForEach(AssetType.allCases, id: \.self) { Text($0.label).tag($0) }
            }
            TextField("タイトル（例：学園祭の実行委員長）", text: $title)
            Section("本文（結論→理由→具体例→学び）") {
                TextEditor(text: $content).frame(minHeight: 180)
            }
            if !tags.isEmpty {
                Section("タグ") {
                    ForEach(tags) { tag in
                        Toggle(tag.name, isOn: Binding(get: { selectedTags.contains(tag.id) }, set: { on in
                            if on { selectedTags.insert(tag.id) } else { selectedTags.remove(tag.id) }
                        }))
                    }
                }
            }
            if let error { ErrorBanner(error: error) }
        }
        .navigationTitle(isNew ? "自己資産の登録" : "自己資産の編集")
        .toolbar {
            ToolbarItem(placement: .cancellationAction) { Button("キャンセル") { dismiss() } }
            ToolbarItem(placement: .confirmationAction) {
                AsyncButton("保存") { await save() }
                    .disabled(title.trimmingCharacters(in: .whitespaces).isEmpty || content.trimmingCharacters(in: .whitespaces).isEmpty)
            }
        }
        .onAppear {
            if case .existing(let asset) = target {
                type = asset.type
                title = asset.title
                content = asset.content
                selectedTags = Set(asset.tagIds ?? [])
            }
        }
    }

    private var isNew: Bool {
        if case .new = target { return true }
        return false
    }

    private func save() async {
        let input = AssetInput(type: type, title: title, content: content, tagIds: Array(selectedTags))
        do {
            switch target {
            case .new: _ = try await model.api.createAsset(input)
            case .existing(let asset): _ = try await model.api.updateAsset(asset.id, input)
            }
            onSaved()
            dismiss()
        } catch {
            self.error = error
        }
    }
}

struct TagManagerView: View {
    @Environment(AppModel.self) private var model
    let onChange: () -> Void
    @ViewState private var tags: [AssetTag] = []
    @ViewState private var newName = ""
    @ViewState private var error: Error?

    var body: some View {
        List {
            Section {
                HStack {
                    TextField("新しいタグ", text: $newName)
                    AsyncButton("追加") {
                        do {
                            _ = try await model.api.createAssetTag(AssetTagInput(name: newName))
                            newName = ""
                            await load()
                            onChange()
                        } catch {
                            self.error = error
                        }
                    }
                    .disabled(newName.trimmingCharacters(in: .whitespaces).isEmpty)
                }
            }
            if let error { ErrorBanner(error: error) }
            ForEach(tags) { Text($0.name) }
                .onDelete { offsets in
                    let targets = offsets.map { tags[$0] }
                    Task {
                        for tag in targets { try? await model.api.deleteAssetTag(tag.id) }
                        await load()
                        onChange()
                    }
                }
        }
        .navigationTitle("タグ")
        .task { await load() }
    }

    private func load() async {
        do {
            tags = try await model.api.assetTags().items
        } catch {
            self.error = error
        }
    }
}

/// AI連携で使う自己資産の選択（タグで絞り込み、前回の選択を初期値にする。3.7）
struct AssetSelectionView: View {
    @Environment(AppModel.self) private var model
    @Binding var selection: Set<ID>
    @ViewState private var assets: [Asset] = []
    @ViewState private var tags: [AssetTag] = []
    @ViewState private var tagFilter: ID?
    @AppStorage("interviewhack.lastAssetSelection") private var lastSelection = ""

    var body: some View {
        List {
            Picker("タグで絞り込む", selection: $tagFilter) {
                Text("すべて").tag(ID?.none)
                ForEach(tags) { Text($0.name).tag(ID?.some($0.id)) }
            }
            ForEach(assets.filter { tagFilter == nil || ($0.tagIds ?? []).contains(tagFilter!) }) { asset in
                Toggle(isOn: Binding(get: { selection.contains(asset.id) }, set: { on in
                    if on { selection.insert(asset.id) } else { selection.remove(asset.id) }
                    lastSelection = selection.sorted().joined(separator: ",")
                })) {
                    VStack(alignment: .leading) {
                        Text(asset.title)
                        Text(asset.type.label).font(.caption).foregroundStyle(.secondary)
                    }
                }
            }
        }
        .navigationTitle("使う自己資産（\(selection.count)）")
        .task {
            if let cached = try? model.cache.load(Page<Asset>.self, for: .assets) { assets = cached.value.items }
            if let fresh = try? await model.api.assets() {
                assets = fresh.items
                try? model.cache.save(fresh, for: .assets)
            }
            tags = (try? await model.api.assetTags().items) ?? []
            if selection.isEmpty, !lastSelection.isEmpty {
                let ids = Set(lastSelection.split(separator: ",").map(String.init))
                selection = ids.intersection(assets.map(\.id))
            }
        }
    }
}
