// DesignDocument/3_UIUXDesign.md 3.3.3 / 3.4.0 / 3.5.10〜3.5.13 AI連携（S-AI-01〜04）/ 5_FrontendDesign.md 5.4.2・5.11
import InterviewHackKit
import SwiftUI

/// AI連携の共通フロー。`flow.step` に応じて S-AI-01〜03 を切り替える
struct AILinkFlowView: View {
    @Bindable var flow: AILinkFlow
    var title: String
    var onSaved: (CommitResponse) -> Void
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            content
                .navigationTitle(title)
                .inlineNavigationTitle()
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("閉じる") { dismiss() }
                    }
                }
        }
        .onChange(of: flow.step) { _, step in
            switch step {
            case .saved(let response):
                onSaved(response)
                dismiss()
            case .abandoned:
                dismiss()
            default:
                break
            }
        }
    }

    @ViewBuilder private var content: some View {
        switch flow.step {
        case .idle, .preparing:
            VStack(spacing: 16) {
                if let error = flow.error {
                    ErrorBanner(error: error)
                } else {
                    ProgressView("プロンプトを作成中")
                }
            }
            .padding()
        case .prompt:
            PromptReviewView(flow: flow)
        case .paste, .parsing:
            ResponsePasteView(flow: flow)
        case .preview, .committing:
            ImportPreviewView(flow: flow)
        case .saved, .abandoned:
            ProgressView()
        }
    }
}

/// S-AI-01 プロンプト確認
struct PromptReviewView: View {
    @Bindable var flow: AILinkFlow
    @AppStorage("interviewhack.aiLinkNoticeShown") private var noticeShown = false
    @ViewState private var showNotice = true

    /// 4.3.5：プロンプト全体の推奨上限
    private let recommendedLength = 8_000

    var body: some View {
        Form {
            Section {
                DisclosureGroup("外部AIに渡すデータの注意", isExpanded: $showNotice) {
                    Text("""
                    ・プロンプトには、入力した自己資産や企業情報などが含まれます
                    ・外部AIサービスでの扱いは、各サービスの規約・設定（学習への利用など）に従います
                    ・他人の個人情報や、社外秘の資料は含めないでください
                    """)
                    .font(.callout)
                }
            }
            if flow.promptLength > recommendedLength || !flow.promptWarnings.isEmpty {
                Section("警告") {
                    if flow.promptLength > recommendedLength {
                        Text("プロンプトが長すぎます（\(flow.promptLength) 字）。資料を減らすか、先に資料を要約してください")
                    }
                    ForEach(flow.promptWarnings, id: \.self) { Text($0) }
                }
                .foregroundStyle(.orange)
            }
            Section {
                TextEditor(text: $flow.promptText)
                    .frame(minHeight: 240)
                    .font(.callout)
            } header: {
                HStack {
                    Text("プロンプト（編集できます）")
                    Spacer()
                    Text("\(flow.promptLength) 字").monospacedDigit()
                }
            }
            Section {
                ShareLink(item: flow.promptText) {
                    Label("ChatGPT などに送る", systemImage: "square.and.arrow.up")
                        .frame(minHeight: 44)
                }
                .simultaneousGesture(TapGesture().onEnded { handOff() })
                Button {
                    Clipboard.copy(flow.promptText)
                    handOff()
                } label: {
                    Label("コピー", systemImage: "doc.on.doc").frame(minHeight: 44)
                }
                Button("回答を貼り付ける") { flow.showPaste() }
                    .frame(minHeight: 44)
            }
            Section {
                AsyncButton("中止", role: .destructive) { await flow.abandon() }
            }
        }
        .onAppear { showNotice = !noticeShown }
    }

    private func handOff() {
        noticeShown = true
        flow.didHandOffPrompt()
    }
}

/// S-AI-02 回答の貼り戻し
struct ResponsePasteView: View {
    @Bindable var flow: AILinkFlow
    @Environment(AppModel.self) private var model

    var body: some View {
        Form {
            Section(flow.useCase.label) {
                TextEditor(text: $flow.responseText)
                    .frame(minHeight: 260)
                    .font(.callout)
                Button {
                    // 5.11：ユーザーがボタンを押したときだけクリップボードを読む
                    if let text = Clipboard.paste() { flow.responseText = text }
                } label: {
                    Label("クリップボードから貼り付け", systemImage: "doc.on.clipboard")
                        .frame(minHeight: 44)
                }
                .buttonStyle(.borderedProminent)
                .tint(Clipboard.hasText ? .accentColor : .gray)
            }
            if let error = flow.error {
                Section {
                    ErrorBanner(error: error)
                    if flow.needsRestart {
                        Text("このAI連携は続けられません。最初からやり直してください")
                    }
                }
            }
            Section {
                AsyncButton("取り込む") { await flow.parse() }
                    .disabled(flow.responseText.isEmpty || !model.connectivity.isOnline)
                Button("プロンプトをもう一度見る") { flow.showPrompt() }
            }
        }
        .overlay {
            if flow.step == .parsing { ProgressView("解析中") }
        }
    }
}

/// S-AI-03 取り込みプレビュー
struct ImportPreviewView: View {
    @Bindable var flow: AILinkFlow
    @Environment(AppModel.self) private var model

    var body: some View {
        Form {
            if let result = flow.parseResult {
                Section {
                    Label(result.label, systemImage: result == .success ? "checkmark.circle" : "exclamationmark.circle")
                        .foregroundStyle(result == .success ? .green : .orange)
                    ForEach(flow.parseWarnings, id: \.self) { warning in
                        Text(warning).font(.callout).foregroundStyle(.orange)
                    }
                    if result != .success {
                        Button("形式どおりに出力し直してもらう依頼文をコピー") {
                            Clipboard.copy("先ほどの回答を、指定した出力形式（【形式:…】から始まる目印付きの形式）だけで出力し直してください。")
                        }
                        .font(.callout)
                    }
                }
            }
            if let fallback = flow.fallbackText {
                Section("貼り付けた全文（ここから項目へ写して保存できます）") {
                    Text(fallback).font(.callout).textSelection(.enabled)
                }
            }
            if flow.preview != nil {
                PreviewEditor(preview: Binding(get: { flow.preview! }, set: { flow.preview = $0 }),
                              originalText: flow.originalText)
            }
            if let notice = flow.notice {
                Section { Text(notice).foregroundStyle(.orange) }
            }
            if let error = flow.error {
                Section { ErrorBanner(error: error) }
            }
            Section {
                AsyncButton("保存") { await flow.commit() }
                    .disabled(!model.connectivity.isOnline)
                Button("貼り直す") { flow.showPaste() }
            }
        }
        .overlay {
            if flow.step == .committing { ProgressView("保存中") }
        }
    }
}

/// use_case ごとのプレビュー編集
struct PreviewEditor: View {
    @Binding var preview: EditablePreview
    var originalText: String?

    var body: some View {
        switch preview {
        case .qa(let items):
            QAPreviewEditor(items: Binding(get: { items }, set: { preview = .qa(items: $0) }))
        case let .review(issues, improvements, improved, saveToQa):
            Section("指摘") {
                LineListEditor(lines: Binding(get: { issues }, set: {
                    preview = .review(issues: $0, improvements: improvements, improvedAnswerText: improved, saveToQa: saveToQa)
                }))
            }
            Section("改善案") {
                LineListEditor(lines: Binding(get: { improvements }, set: {
                    preview = .review(issues: issues, improvements: $0, improvedAnswerText: improved, saveToQa: saveToQa)
                }))
            }
            Section("改善後の回答") {
                TextEditor(text: Binding(get: { improved }, set: {
                    preview = .review(issues: issues, improvements: improvements, improvedAnswerText: $0, saveToQa: saveToQa)
                }))
                .frame(minHeight: 140)
                Toggle("改善後の回答をQ&Aバンクにも登録する", isOn: Binding(get: { saveToQa }, set: {
                    preview = .review(issues: issues, improvements: improvements, improvedAnswerText: improved, saveToQa: $0)
                }))
            }
        case let .research(highlights, ideal, topics):
            Section("魅力・特徴") {
                LineListEditor(lines: Binding(get: { highlights }, set: {
                    preview = .research(highlights: $0, idealCandidate: ideal, recentTopics: topics)
                }))
            }
            Section("求める人物像") {
                TextEditor(text: Binding(get: { ideal }, set: {
                    preview = .research(highlights: highlights, idealCandidate: $0, recentTopics: topics)
                }))
                .frame(minHeight: 100)
            }
            Section("最近のトピック") {
                LineListEditor(lines: Binding(get: { topics }, set: {
                    preview = .research(highlights: highlights, idealCandidate: ideal, recentTopics: $0)
                }))
            }
        case .text(let text):
            if let originalText, !originalText.isEmpty {
                Section("元の文章との差分") {
                    DiffTextView(old: originalText, new: text)
                }
            }
            Section {
                TextEditor(text: Binding(get: { text }, set: { preview = .text($0) }))
                    .frame(minHeight: 240)
            } header: {
                HStack {
                    Text("取り込む本文")
                    Spacer()
                    Text("\(text.count) 字").monospacedDigit()
                }
            }
        }
    }
}

struct QAPreviewEditor: View {
    @Binding var items: [EditableQAItem]

    var body: some View {
        Section("想定Q&A（\(items.count) 件）") {
            ForEach($items) { $item in
                VStack(alignment: .leading, spacing: 8) {
                    TextField("質問", text: $item.questionText, axis: .vertical).font(.headline)
                    TextField("模範解答", text: $item.answerText, axis: .vertical)
                    if let intent = item.intentText, !intent.isEmpty {
                        // 意図はプレビューでの確認用。保存しない（4.2.1）
                        Text("意図：\(intent)").font(.caption).foregroundStyle(.secondary)
                    }
                }
                .padding(.vertical, 4)
            }
            .onDelete { items.remove(atOffsets: $0) }
            Button {
                items.append(EditableQAItem())
            } label: {
                Label("Q&Aを手で追加", systemImage: "plus")
            }
        }
    }
}

struct LineListEditor: View {
    @Binding var lines: [String]

    var body: some View {
        ForEach(lines.indices, id: \.self) { index in
            TextField("項目", text: Binding(get: { lines[index] }, set: { lines[index] = $0 }), axis: .vertical)
        }
        .onDelete { lines.remove(atOffsets: $0) }
        Button {
            lines.append("")
        } label: {
            Label("追加", systemImage: "plus")
        }
    }
}

/// S-AI-04 未完了の AI連携一覧（キャッシュしない。5.8.2）
struct PendingSessionsView: View {
    @Environment(AppModel.self) private var model
    @ViewState private var sessions: [PromptSessionSummary] = []
    @ViewState private var error: Error?
    @ViewState private var activeFlow: AILinkFlow?

    var body: some View {
        List {
            if let error {
                ErrorBanner(error: error) { Task { await load() } }
            }
            if sessions.isEmpty && error == nil {
                EmptyStateView(title: "未完了の AI連携はありません", message: "プロンプトを作ると、回答を貼り戻すまでここに表示されます")
            }
            ForEach(sessions) { session in
                Button {
                    let flow = model.makeFlow(session.useCase)
                    activeFlow = flow
                    Task { await flow.resume(session) }
                } label: {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(session.useCase.label).font(.headline)
                        if let target = session.targetLabel { Text(target).font(.subheadline) }
                        if let createdAt = session.createdAt {
                            Text(createdAt.formatted(date: .abbreviated, time: .shortened))
                                .font(.caption).foregroundStyle(.secondary)
                        }
                    }
                    .frame(minHeight: 44, alignment: .leading)
                }
                .swipeActions {
                    Button("中止", role: .destructive) {
                        Task {
                            _ = try? await model.api.abandonPromptSession(sessionId: session.sessionId)
                            await load()
                        }
                    }
                }
            }
        }
        .navigationTitle("未完了の AI連携")
        .task { await load() }
        .refreshable { await load() }
        .sheet(item: Binding(get: { activeFlow.map(FlowBox.init) }, set: { activeFlow = $0?.flow })) { box in
            AILinkFlowView(flow: box.flow, title: box.flow.useCase.label) { _ in
                Task { await load() }
            }
        }
    }

    private func load() async {
        do {
            sessions = try await model.api.promptSessions().items
            error = nil
        } catch {
            self.error = error
        }
    }
}

/// `sheet(item:)` 用の識別子付きラッパー
struct FlowBox: Identifiable {
    let flow: AILinkFlow
    var id: ObjectIdentifier { ObjectIdentifier(flow) }
}

extension View {
    /// AI連携のシートを出す共通の修飾子
    func aiLinkSheet(_ flow: Binding<AILinkFlow?>, title: String, onSaved: @escaping (CommitResponse) -> Void) -> some View {
        sheet(item: Binding(get: { flow.wrappedValue.map(FlowBox.init) }, set: { flow.wrappedValue = $0?.flow })) { box in
            AILinkFlowView(flow: box.flow, title: title, onSaved: onSaved)
        }
    }
}
