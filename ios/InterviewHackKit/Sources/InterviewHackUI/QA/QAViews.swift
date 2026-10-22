// DesignDocument/3_UIUXDesign.md 3.4.4 復習フロー / S-QA-01〜03（Q&Aバンク・詳細・復習）
import InterviewHackKit
import SwiftUI

struct QAFilter: Equatable {
    var companyId: ID?
    var reviewStatus: QAReviewStatus?
    var favoritesOnly = false
    var keyword = ""

    func apply(_ items: [QA]) -> [QA] {
        items
            .filter { companyId == nil || $0.companyId == companyId }
            .filter { reviewStatus == nil || $0.reviewStatus == reviewStatus }
            .filter { !favoritesOnly || $0.isFavorite }
            .filter { keyword.isEmpty || $0.questionText.localizedCaseInsensitiveContains(keyword)
                || $0.answerText.localizedCaseInsensitiveContains(keyword) }
            // お気に入りは一覧の先頭に表示する（2.10.8）
            .sorted { lhs, rhs in
                if lhs.isFavorite != rhs.isFavorite { return lhs.isFavorite }
                return (lhs.createdAt ?? .distantPast) > (rhs.createdAt ?? .distantPast)
            }
    }
}

/// S-QA-01 Q&Aバンク一覧（オフラインでも閲覧可）
struct QAListView: View {
    @Environment(AppModel.self) private var model
    @ViewState var filter = QAFilter()
    @ViewState private var qa: CachedResource<Page<QA>>?
    @ViewState private var companies: [Company] = []

    init(filter: QAFilter = QAFilter()) {
        _filter = ViewState(initialValue: filter)
    }

    var body: some View {
        List {
            Section {
                Picker("企業", selection: $filter.companyId) {
                    Text("すべて").tag(ID?.none)
                    ForEach(companies) { Text($0.name).tag(ID?.some($0.id)) }
                }
                Picker("復習", selection: $filter.reviewStatus) {
                    Text("すべて").tag(QAReviewStatus?.none)
                    ForEach(QAReviewStatus.allCases, id: \.self) { Text($0.label).tag(QAReviewStatus?.some($0)) }
                }
                Toggle("お気に入りだけ", isOn: $filter.favoritesOnly)
                NavigationLink {
                    QAReviewView(filter: filter)
                } label: {
                    Label("この条件で復習する", systemImage: "rectangle.on.rectangle.angled")
                }
            }
            if let resource = qa, let lastSynced = resource.lastSyncedAt, !model.connectivity.isOnline {
                Text("最終同期 \(lastSynced.formatted(date: .abbreviated, time: .shortened))").font(.caption)
            }
            let items = filter.apply(qa?.value?.items ?? [])
            if items.isEmpty && qa?.isRefreshing == false {
                EmptyStateView(title: "Q&Aがありません", message: "まず想定Q&Aを作るか、面接を振り返って改善版を保存しましょう")
            }
            ForEach(items) { item in
                NavigationLink {
                    QADetailView(qa: item) { Task { await qa?.refresh() } }
                } label: {
                    HStack {
                        VStack(alignment: .leading, spacing: 4) {
                            Text(item.questionText).font(.headline).lineLimit(2)
                            Text("\(item.sourceType.label)・\(item.reviewStatus.label)").font(.caption).foregroundStyle(.secondary)
                        }
                        if item.isFavorite {
                            Spacer()
                            Image(systemName: "star.fill").foregroundStyle(.yellow).accessibilityLabel("お気に入り")
                        }
                    }
                }
            }
        }
        .searchable(text: $filter.keyword)
        .navigationTitle("Q&Aバンク")
        .task {
            let api = model.api
            if qa == nil { qa = model.resource(.qa) { try await api.qaList(ListQuery(limit: 200)) } }
            companies = (try? model.cache.load(Page<Company>.self, for: .companies))?.value.items ?? []
            await qa?.refresh()
            if let fresh = try? await api.companies() { companies = fresh.items }
        }
        .refreshable { await qa?.refresh() }
    }
}

/// S-QA-02 Q&A詳細
struct QADetailView: View {
    @Environment(AppModel.self) private var model
    @Environment(\.dismiss) private var dismiss
    @ViewState var qa: QA
    var onChange: (() -> Void)?
    @ViewState private var question = ""
    @ViewState private var answer = ""
    @ViewState private var error: Error?

    var body: some View {
        Form {
            Section("質問") { TextField("質問", text: $question, axis: .vertical) }
            Section("回答") { TextEditor(text: $answer).frame(minHeight: 160) }
            Section {
                Toggle("お気に入り", isOn: Binding(get: { qa.isFavorite }, set: { update(QAInput(isFavorite: $0)) }))
                Picker("復習", selection: Binding(get: { qa.reviewStatus }, set: { update(QAInput(reviewStatus: $0)) })) {
                    ForEach(QAReviewStatus.allCases, id: \.self) { Text($0.label).tag($0) }
                }
            }
            .disabled(!model.connectivity.isOnline)
            if let error { ErrorBanner(error: error) }
            AsyncButton("保存") {
                do {
                    qa = try await model.api.updateQA(qa.id, QAInput(questionText: question, answerText: answer))
                    onChange?()
                } catch {
                    self.error = error
                }
            }
            .disabled(!model.connectivity.isOnline)
            AsyncButton("削除", role: .destructive) {
                do {
                    try await model.api.deleteQA(qa.id)
                    onChange?()
                    dismiss()
                } catch {
                    self.error = error
                }
            }
            .disabled(!model.connectivity.isOnline)
        }
        .navigationTitle("Q&A")
        .onAppear {
            question = qa.questionText
            answer = qa.answerText
        }
    }

    private func update(_ input: QAInput) {
        Task {
            do {
                qa = try await model.api.updateQA(qa.id, input)
                onChange?()
            } catch {
                self.error = error
            }
        }
    }
}

/// S-QA-03 復習UI（カード。オフラインでも閲覧できる。ステータス更新はオンライン時のみ）
struct QAReviewView: View {
    @Environment(AppModel.self) private var model
    let filter: QAFilter
    @ViewState private var cards: [QA] = []
    @ViewState private var index = 0
    @ViewState private var revealed = false

    var body: some View {
        VStack(spacing: 16) {
            if cards.isEmpty {
                EmptyStateView(title: "復習する Q&A がありません", message: "条件を変えるか、Q&A を追加してください")
            } else {
                Text("\(index + 1) / \(cards.count)").font(.caption.monospacedDigit()).foregroundStyle(.secondary)
                let card = cards[index]
                VStack(alignment: .leading, spacing: 16) {
                    Text(card.questionText).font(.title3.bold())
                    if revealed {
                        Divider()
                        ScrollView { Text(card.answerText).font(.body).frame(maxWidth: .infinity, alignment: .leading) }
                    } else {
                        Text("タップして回答を表示").foregroundStyle(.secondary)
                    }
                    Spacer(minLength: 0)
                }
                .padding(20)
                .frame(maxWidth: .infinity, minHeight: 320, alignment: .topLeading)
                .background(.thinMaterial, in: RoundedRectangle(cornerRadius: 16))
                .contentShape(Rectangle())
                .onTapGesture { revealed.toggle() }
                .gesture(DragGesture(minimumDistance: 40).onEnded { value in
                    if value.translation.width < 0 { move(1) } else { move(-1) }
                })
                .accessibilityAction(named: "回答を表示") { revealed = true }
                .padding(.horizontal)

                HStack(spacing: 12) {
                    Button("前へ") { move(-1) }.disabled(index == 0)
                    Button(card.isFavorite ? "お気に入り解除" : "お気に入り") {
                        update(card, QAInput(isFavorite: !card.isFavorite))
                    }
                    Button("次へ") { move(1) }.disabled(index >= cards.count - 1)
                }
                .buttonStyle(.bordered)
                HStack(spacing: 12) {
                    Button("要再確認") { update(card, QAInput(reviewStatus: .needsReview)) }
                    Button("復習済み") { update(card, QAInput(reviewStatus: .reviewed)) }
                        .buttonStyle(.borderedProminent)
                }
                .disabled(!model.connectivity.isOnline)
                if !model.connectivity.isOnline {
                    Text("オフラインのため、復習ステータスは更新できません").font(.caption).foregroundStyle(.secondary)
                }
            }
            Spacer()
        }
        .navigationTitle("復習")
        .task {
            // キャッシュを先に表示し、オンラインなら最新に差し替える（オフライン復習。D16）
            if let cached = try? model.cache.load(Page<QA>.self, for: .qa) { cards = filter.apply(cached.value.items) }
            if let fresh = try? await model.api.qaList(ListQuery(limit: 200)) {
                try? model.cache.save(fresh, for: .qa)
                cards = filter.apply(fresh.items)
            }
        }
    }

    private func move(_ delta: Int) {
        let next = index + delta
        guard cards.indices.contains(next) else { return }
        index = next
        revealed = false
    }

    private func update(_ card: QA, _ input: QAInput) {
        Task {
            if let updated = try? await model.api.updateQA(card.id, input), let position = cards.firstIndex(where: { $0.id == card.id }) {
                cards[position] = updated
            }
        }
    }
}
