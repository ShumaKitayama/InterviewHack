// DesignDocument/3_UIUXDesign.md 3.4.3 事後対策フロー / 3.5.7 面接ログ入力（S-Log-02）/ 3.5.8 添削結果（S-Log-03）
import InterviewHackKit
import SwiftUI

/// S-Log-01 面接一覧
struct InterviewListView: View {
    @Environment(AppModel.self) private var model
    var companyId: ID?
    @ViewState private var interviews: CachedResource<Page<Interview>>?
    @ViewState private var companies: [Company] = []
    @ViewState private var creating = false

    var body: some View {
        List {
            if let error = interviews?.error, interviews?.value == nil {
                ErrorBanner(error: error) { Task { await interviews?.refresh() } }
            }
            let items = (interviews?.value?.items ?? [])
                .filter { companyId == nil || $0.companyId == companyId }
                .sorted { $0.interviewDate > $1.interviewDate }
            if items.isEmpty && interviews?.isRefreshing == false {
                EmptyStateView(title: "面接がありません", message: "面接を登録して、聞かれた質問と回答を記録しましょう",
                               actionTitle: "面接を登録") { creating = true }
            }
            ForEach(items) { interview in
                NavigationLink {
                    InterviewDetailView(interview: interview)
                } label: {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("\(companyName(interview.companyId)) \(interview.phase ?? "")").font(.headline)
                        HStack {
                            Text(interview.interviewDate.description)
                            if let result = interview.result { Text(result.label) }
                        }
                        .font(.caption).foregroundStyle(.secondary)
                    }
                }
            }
            .onDelete { offsets in
                let targets = offsets.map { items[$0] }
                Task {
                    for interview in targets { try? await model.api.deleteInterview(interview.id) }
                    await interviews?.refresh()
                }
            }
        }
        .navigationTitle("記録")
        .toolbar {
            ToolbarItem(placement: ToolbarPlacement.trailing) {
                Button { creating = true } label: { Image(systemName: "plus").accessibilityLabel("面接を登録") }
                    .disabled(!model.connectivity.isOnline)
            }
        }
        .task {
            let api = model.api
            if interviews == nil { interviews = model.resource(.interviews) { try await api.interviews() } }
            companies = (try? model.cache.load(Page<Company>.self, for: .companies))?.value.items ?? []
            if let fresh = try? await api.companies() { companies = fresh.items }
            await interviews?.refresh()
        }
        .refreshable { await interviews?.refresh() }
        .sheet(isPresented: $creating) {
            NavigationStack { InterviewCreateView(companies: companies, companyId: companyId) { Task { await interviews?.refresh() } } }
        }
    }

    private func companyName(_ id: ID?) -> String {
        guard let id else { return "" }
        return companies.first { $0.id == id }?.name ?? ""
    }
}

struct InterviewCreateView: View {
    @Environment(AppModel.self) private var model
    @Environment(\.dismiss) private var dismiss
    let companies: [Company]
    @ViewState var companyId: ID?
    let onSaved: () -> Void
    @ViewState private var date = Date()
    @ViewState private var phase = ""
    @ViewState private var error: Error?

    var body: some View {
        Form {
            Picker("企業", selection: $companyId) {
                Text("選択してください").tag(ID?.none)
                ForEach(companies) { Text($0.name).tag(ID?.some($0.id)) }
            }
            DatePicker("日付", selection: $date, displayedComponents: .date)
            TextField("フェーズ（例：一次面接）", text: $phase)
            if let error { ErrorBanner(error: error) }
        }
        .navigationTitle("面接を登録")
        .toolbar {
            ToolbarItem(placement: .cancellationAction) { Button("キャンセル") { dismiss() } }
            ToolbarItem(placement: .confirmationAction) {
                AsyncButton("登録") {
                    do {
                        _ = try await model.api.createInterview(InterviewInput(
                            companyId: companyId, interviewDate: CalendarDate(date: date), phase: phase.nilIfBlank, result: .pending))
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

/// 面接の詳細：面接ログの一覧と入力
struct InterviewDetailView: View {
    @Environment(AppModel.self) private var model
    @ViewState var interview: Interview
    @ViewState private var logs: CachedResource<Page<InterviewLog>>?
    @ViewState private var reviewFlow: AILinkFlow?
    @ViewState private var reviewedLogId: ID?
    @ViewState private var assetIds: Set<ID> = []
    @ViewState private var strictness: ReviewStrictness = .normal

    var body: some View {
        List {
            Section {
                Picker("結果", selection: Binding(get: { interview.result ?? .pending }, set: { result in
                    Task {
                        if let updated = try? await model.api.updateInterview(interview.id, InterviewInput(result: result)) {
                            interview = updated
                        }
                    }
                })) {
                    ForEach(InterviewResult.allCases, id: \.self) { Text($0.label).tag($0) }
                }
                .disabled(!model.connectivity.isOnline)
                NavigationLink {
                    InterviewLogEntryView(interviewId: interview.id) { Task { await logs?.refresh() } }
                } label: {
                    Label("質問と回答を入力", systemImage: "plus.bubble")
                }
            }
            Section("添削の設定") {
                NavigationLink { AssetSelectionView(selection: $assetIds) } label: {
                    LabeledContent("参照する自己資産", value: "\(assetIds.count) 件")
                }
                Picker("厳しさ", selection: $strictness) {
                    ForEach(ReviewStrictness.allCases, id: \.self) { Text($0.label).tag($0) }
                }
            }
            Section("面接ログ") {
                ForEach(logs?.value?.items ?? []) { log in
                    VStack(alignment: .leading, spacing: 6) {
                        Text(log.questionText).font(.headline)
                        Text(log.answerText).font(.callout).lineLimit(3)
                        HStack {
                            if log.aiImprovedAnswerText != nil {
                                NavigationLink("添削結果") { ReviewResultView(log: log, companyId: interview.companyId) }
                            }
                            Spacer()
                            Button(log.aiFeedbackText == nil ? "添削する" : "もう一度添削") { startReview(log) }
                                .buttonStyle(.bordered)
                                .disabled(!model.connectivity.isOnline)
                        }
                    }
                    .padding(.vertical, 4)
                }
            }
        }
        .navigationTitle(interview.phase ?? "面接")
        .task {
            let api = model.api
            let interviewId = interview.id
            if logs == nil { logs = model.resource(.interviewLogs(interviewId: interviewId)) { try await api.interviewLogs(interviewId: interviewId) } }
            await logs?.refresh()
        }
        .refreshable { await logs?.refresh() }
        .aiLinkSheet($reviewFlow, title: UseCase.answerReview.label) { _ in
            Task { await logs?.refresh() }
        }
        .navigationDestination(item: Binding(get: { reviewedLog.map { $0 } }, set: { _ in reviewedLogId = nil })) { log in
            ReviewResultView(log: log, companyId: interview.companyId)
        }
    }

    private var reviewedLog: InterviewLog? {
        guard let reviewedLogId else { return nil }
        return logs?.value?.items.first { $0.id == reviewedLogId && $0.aiImprovedAnswerText != nil }
    }

    private func startReview(_ log: InterviewLog) {
        let flow = model.makeFlow(.answerReview)
        reviewFlow = flow
        reviewedLogId = log.id
        Task {
            await flow.start(PromptSessionRequest(useCase: .answerReview, interviewLogId: log.id, assetIds: Array(assetIds),
                                                  options: PromptOptions(strictness: strictness)))
        }
    }
}

extension InterviewLog: Hashable {
    public func hash(into hasher: inout Hasher) { hasher.combine(id) }
}

/// S-Log-02 面接ログ入力（複数件を連続登録。入力途中は端末に下書き保存。D16）
struct InterviewLogEntryView: View {
    @Environment(AppModel.self) private var model
    @Environment(\.dismiss) private var dismiss
    let interviewId: ID
    let onSaved: () -> Void
    @ViewState private var entries: [InterviewLogInput] = [InterviewLogInput()]
    @ViewState private var restoredDraft = false
    @ViewState private var error: Error?

    var body: some View {
        Form {
            if restoredDraft {
                Section {
                    Label("送信していない下書きを読み込みました。確認して「保存」で送信してください", systemImage: "tray.and.arrow.up")
                        .font(.callout)
                }
            }
            ForEach(entries.indices, id: \.self) { index in
                Section("質問 \(index + 1)") {
                    TextField("聞かれた質問", text: binding(index, \.questionText), axis: .vertical)
                    TextField("自分の回答", text: binding(index, \.answerText), axis: .vertical)
                        .lineLimit(3...10)
                    TextField("自己評価・補足メモ（任意）", text: binding(index, \.selfEvaluation), axis: .vertical)
                }
            }
            .onDelete { offsets in
                entries.remove(atOffsets: offsets)
                persist()
            }
            Button {
                entries.append(InterviewLogInput())
            } label: {
                Label("質問を追加", systemImage: "plus")
            }
            if let error { ErrorBanner(error: error) }
            AsyncButton("保存") { await save() }
                .disabled(!model.connectivity.isOnline)
        }
        .navigationTitle("面接ログの入力")
        .onAppear {
            if let draft = try? model.drafts.load(InterviewLogDraft.self, for: .interviewLog(interviewId: interviewId)),
               !draft.entries.isEmpty {
                entries = draft.entries
                restoredDraft = true
            }
        }
    }

    private func binding(_ index: Int, _ keyPath: WritableKeyPath<InterviewLogInput, String?>) -> Binding<String> {
        Binding(get: { entries[index][keyPath: keyPath] ?? "" }, set: { value in
            entries[index][keyPath: keyPath] = value
            persist()
        })
    }

    /// 入力のたびに端末へ保存する（5.8.3）
    private func persist() {
        try? model.drafts.save(InterviewLogDraft(entries: entries), for: .interviewLog(interviewId: interviewId))
    }

    private func save() async {
        let filled = entries.filter { !($0.questionText ?? "").isEmpty || !($0.answerText ?? "").isEmpty }
        do {
            for entry in filled { try Validator.interviewLog(entry) }
            var remaining = filled
            for entry in filled {
                var input = entry
                input.interviewId = interviewId
                _ = try await model.api.createInterviewLog(input)
                remaining.removeFirst()
                // 途中で失敗しても、送信済みの分を下書きから外す
                try? model.drafts.save(InterviewLogDraft(entries: remaining), for: .interviewLog(interviewId: interviewId))
            }
            try? model.drafts.remove(.interviewLog(interviewId: interviewId))
            onSaved()
            dismiss()
        } catch {
            self.error = error
        }
    }
}

/// S-Log-03 添削結果
struct ReviewResultView: View {
    @Environment(AppModel.self) private var model
    let log: InterviewLog
    let companyId: ID?
    @ViewState private var savedQA: QA?
    @ViewState private var error: Error?

    var body: some View {
        List {
            Section("質問") { Text(log.questionText) }
            Section("指摘・改善案") {
                Text(log.aiFeedbackText ?? "まだ添削していません").font(.callout).textSelection(.enabled)
            }
            Section("改善版の回答") {
                Text(log.aiImprovedAnswerText ?? "").font(.callout).textSelection(.enabled)
            }
            if let error { ErrorBanner(error: error) }
            Section {
                if let savedQA {
                    NavigationLink("保存した Q&A を開く") { QADetailView(qa: savedQA) }
                } else {
                    AsyncButton("改善版をQ&Aとして保存") {
                        do {
                            savedQA = try await model.api.createQA(QAInput(companyId: companyId, questionText: log.questionText,
                                                                           answerText: log.aiImprovedAnswerText ?? ""))
                        } catch {
                            self.error = error
                        }
                    }
                    .disabled(log.aiImprovedAnswerText == nil || !model.connectivity.isOnline)
                }
            }
        }
        .navigationTitle("添削結果")
    }
}
