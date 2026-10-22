// DesignDocument/11_TestDesign.md 11.4 / 5_FrontendDesign.md 5.4.2 AI連携の画面フロー
import Foundation
import Testing
@testable import InterviewHackKit

final class FakeAILinkAPI: AILinkAPI, @unchecked Sendable {
    private let lock = NSLock()
    var parseResponse = ParseResponse(parseResult: .success, preview: RawPreview(items: [
        QAPreviewItem(questionText: "志望理由は？", answerText: "御社の…", intentText: "動機の確認"),
    ]))
    var commitError: Error?
    var createError: Error?
    private(set) var committedPayloads: [CommitPayload] = []
    private(set) var parsedTexts: [String] = []
    private(set) var abandoned: [ID] = []

    func createPromptSession(_ request: PromptSessionRequest) async throws -> PromptText {
        if let createError { throw createError }
        return PromptText(sessionId: "s1", templateVersion: "qa-v1", promptText: "あなたは面接官です", warnings: ["長いです"])
    }

    func promptText(sessionId: ID) async throws -> PromptText {
        PromptText(sessionId: sessionId, templateVersion: "qa-v1", promptText: "再生成したプロンプト", warnings: nil)
    }

    func parsePromptResponse(sessionId: ID, responseText: String) async throws -> ParseResponse {
        lock.withLock { parsedTexts.append(responseText) }
        return parseResponse
    }

    func commitPromptSession(sessionId: ID, payload: CommitPayload) async throws -> CommitResponse {
        lock.withLock { committedPayloads.append(payload) }
        if let commitError { throw commitError }
        return CommitResponse(sessionId: sessionId, status: .saved, savedRefs: ["qa": ["q1"]])
    }

    func abandonPromptSession(sessionId: ID) async throws -> SessionStatusResponse {
        lock.withLock { abandoned.append(sessionId) }
        return SessionStatusResponse(sessionId: sessionId, status: .abandoned)
    }
}

@Suite("AI連携フロー")
@MainActor
struct AILinkFlowTests {
    let request = PromptSessionRequest(useCase: .qaGenerate, companyId: "c1", assetIds: ["a1"])

    @Test("作成 → プロンプト確認 → 受け渡し → 貼り戻し → 解析 → 保存")
    func happyPath() async throws {
        let api = FakeAILinkAPI()
        let drafts = FileDraftStore(directory: temporaryDirectory())
        let flow = AILinkFlow(useCase: .qaGenerate, api: api, drafts: drafts)

        await flow.start(request)
        #expect(flow.step == .prompt)
        #expect(flow.sessionId == "s1")
        #expect(flow.promptWarnings == ["長いです"])

        flow.didHandOffPrompt()
        #expect(flow.step == .paste)

        flow.responseText = "【形式:qa-v1】\n【Q1】志望理由は？\n【A1】御社の…"
        #expect(try drafts.load(TextDraft.self, for: .aiResponse(sessionId: "s1"))?.text == flow.responseText)

        await flow.parse()
        #expect(flow.step == .preview)
        #expect(flow.parseResult == .success)
        guard case .qa(let items) = flow.preview else {
            Issue.record("qa のプレビューになるはず")
            return
        }
        #expect(items.first?.intentText == "動機の確認")

        await flow.commit()
        #expect(flow.step == .saved(CommitResponse(sessionId: "s1", status: .saved, savedRefs: ["qa": ["q1"]])))
        #expect(api.committedPayloads == [.qa(items: [QACommitItem(questionText: "志望理由は？", answerText: "御社の…")], assetIds: ["a1"])])
        // 保存に成功したら下書きを消す
        #expect(try drafts.load(TextDraft.self, for: .aiResponse(sessionId: "s1")) == nil)
    }

    @Test("対象 ID が足りないときはセッションを作らない")
    func validatesTargets() async {
        let flow = AILinkFlow(useCase: .esGenerate, api: FakeAILinkAPI(), drafts: nil)
        await flow.start(PromptSessionRequest(useCase: .esGenerate, esProjectId: "p1"))
        #expect(flow.step == .idle)
        #expect(flow.error as? PromptSessionRequest.ValidationError == .missingTarget("esSectionId"))
    }

    @Test("解析に失敗した場合は全文を手動修正用に残す")
    func failedParseKeepsText() async {
        let api = FakeAILinkAPI()
        api.parseResponse = ParseResponse(parseResult: .failed, preview: RawPreview(items: []), warnings: ["目印が見つかりません"])
        let flow = AILinkFlow(useCase: .qaGenerate, api: api, drafts: nil)
        await flow.start(request)
        flow.responseText = "形式どおりでない回答"
        await flow.parse()
        #expect(flow.step == .preview)
        #expect(flow.fallbackText == "形式どおりでない回答")
        #expect(flow.parseWarnings == ["目印が見つかりません"])
    }

    @Test("保存する Q&A が空なら commit を送らない")
    func emptyQANotCommitted() async {
        let api = FakeAILinkAPI()
        api.parseResponse = ParseResponse(parseResult: .failed, preview: RawPreview(items: []))
        let flow = AILinkFlow(useCase: .qaGenerate, api: api, drafts: nil)
        await flow.start(request)
        flow.responseText = "x"
        await flow.parse()
        await flow.commit()
        #expect(api.committedPayloads.isEmpty)
        #expect(flow.error as? EditablePreview.ValidationError == .noQAItems)
        #expect(flow.step == .preview)
    }

    @Test("FEATURE_DISABLED のときはプレビューと貼り戻した内容を残す")
    func featureDisabledKeepsPreview() async throws {
        let api = FakeAILinkAPI()
        api.commitError = APIError.server(status: 503, code: .featureDisabled, message: "停止中", requestId: nil)
        let drafts = FileDraftStore(directory: temporaryDirectory())
        let flow = AILinkFlow(useCase: .qaGenerate, api: api, drafts: drafts)
        await flow.start(request)
        flow.responseText = "【Q1】a\n【A1】b"
        await flow.parse()
        await flow.commit()
        #expect(flow.step == .preview)
        #expect(flow.notice != nil)
        #expect(flow.preview != nil)
        #expect(try drafts.load(TextDraft.self, for: .aiResponse(sessionId: "s1"))?.text == "【Q1】a\n【A1】b")
    }

    @Test("SESSION_STATE_CONFLICT はやり直しを案内する")
    func conflictNeedsRestart() async {
        let api = FakeAILinkAPI()
        api.commitError = APIError.server(status: 409, code: .sessionStateConflict, message: "中止済み", requestId: nil)
        let flow = AILinkFlow(useCase: .esGenerate, api: api, drafts: nil)
        api.parseResponse = ParseResponse(parseResult: .success, preview: RawPreview(text: "ES本文"))
        await flow.start(PromptSessionRequest(useCase: .esGenerate, esProjectId: "p1", esSectionId: "sec1"))
        flow.responseText = "ES本文"
        await flow.parse()
        await flow.commit()
        #expect(flow.needsRestart)
        #expect(api.committedPayloads == [.text("ES本文")])
    }

    @Test("未完了のセッションは下書きを読み込んで貼り戻しから再開する")
    func resume() async throws {
        let drafts = FileDraftStore(directory: temporaryDirectory())
        try drafts.save(TextDraft(text: "前回の貼り付け"), for: .aiResponse(sessionId: "s9"))
        let flow = AILinkFlow(useCase: .qaGenerate, api: FakeAILinkAPI(), drafts: drafts)
        await flow.resume(PromptSessionSummary(sessionId: "s9", useCase: .qaGenerate, templateVersion: "qa-v1",
                                               status: .created, targetLabel: "A社", createdAt: nil))
        #expect(flow.step == .paste)
        #expect(flow.responseText == "前回の貼り付け")
        #expect(flow.promptText == "再生成したプロンプト")
    }

    @Test("中止すると abandon を呼び、下書きを消す")
    func abandon() async throws {
        let api = FakeAILinkAPI()
        let drafts = FileDraftStore(directory: temporaryDirectory())
        let flow = AILinkFlow(useCase: .qaGenerate, api: api, drafts: drafts)
        await flow.start(request)
        flow.responseText = "途中"
        await flow.abandon()
        #expect(flow.step == .abandoned)
        #expect(api.abandoned == ["s1"])
        #expect(try drafts.pendingDrafts().isEmpty)
    }

    @Test("空の回答は解析に送らない")
    func emptyResponse() async {
        let api = FakeAILinkAPI()
        let flow = AILinkFlow(useCase: .qaGenerate, api: api, drafts: nil)
        await flow.start(request)
        flow.responseText = "   "
        await flow.parse()
        #expect(api.parsedTexts.isEmpty)
        #expect(flow.step == .prompt)
    }
}

@Suite("取り込みプレビューの保存内容")
struct EditablePreviewTests {
    @Test("添削は指摘と改善案をまとめて feedbackText にする")
    func reviewPayload() throws {
        let preview = EditablePreview.make(useCase: .answerReview, preview: RawPreview(
            issues: ["論理性：結論が後ろ", " "], improvements: ["結論を先に"], improvedAnswerText: " 改善後の回答 "))
        guard case .review(let issues, let improvements, let improved, let save) = preview else {
            Issue.record("review になるはず")
            return
        }
        let payload = try EditablePreview.review(issues: issues, improvements: improvements,
                                                 improvedAnswerText: improved, saveToQa: !save).commitPayload(assetIds: nil)
        #expect(payload == .answerReview(feedbackText: "【指摘】\n- 論理性：結論が後ろ\n【改善案】\n- 結論を先に",
                                         improvedAnswerText: "改善後の回答", saveToQa: true))
    }

    @Test("企業研究の要約は 3 つの見出しを付けて保存する")
    func researchPayload() throws {
        let preview = EditablePreview.make(useCase: .researchSummarize, preview: RawPreview(
            highlights: ["若手に裁量", "海外展開"], idealCandidate: "主体的な人", recentTopics: ["新サービス"]))
        let payload = try preview.commitPayload(assetIds: nil)
        #expect(payload == .researchSummary(summaryText:
            "【魅力・特徴】\n- 若手に裁量\n- 海外展開\n【求める人物像】\n主体的な人\n【最近のトピック】\n- 新サービス"))
    }

    @Test("全文の取り込みは空なら保存できない")
    func emptyText() {
        #expect(throws: EditablePreview.ValidationError.emptyField("本文")) {
            _ = try EditablePreview.text("  \n").commitPayload(assetIds: nil)
        }
    }

    @Test("commit の JSON は use_case ごとの形（7.5.5）")
    func commitJSON() throws {
        let encoder = JSONCoding.makeEncoder()
        let qa = try encoder.encode(CommitPayload.qa(items: [QACommitItem(questionText: "Q", answerText: "A")], assetIds: ["a1"]))
        #expect(String(decoding: qa, as: UTF8.self) == #"{"assetIds":["a1"],"items":[{"answerText":"A","questionText":"Q"}]}"#)
        let review = try encoder.encode(CommitPayload.answerReview(feedbackText: "F", improvedAnswerText: "I", saveToQa: false))
        #expect(String(decoding: review, as: UTF8.self) == #"{"feedbackText":"F","improvedAnswerText":"I","saveToQa":false}"#)
        let research = try encoder.encode(CommitPayload.researchSummary(summaryText: "S"))
        #expect(String(decoding: research, as: UTF8.self) == #"{"summaryText":"S"}"#)
        let text = try encoder.encode(CommitPayload.text("T"))
        #expect(String(decoding: text, as: UTF8.self) == #"{"text":"T"}"#)
    }

    @Test("解析しないユースケースは全文プレビューになる")
    func unstructured() {
        for useCase in [UseCase.summaryGenerate, .esGenerate, .esRefine] {
            #expect(!useCase.isStructured)
            #expect(EditablePreview.make(useCase: useCase, preview: RawPreview(text: "全文")) == .text("全文"))
        }
        #expect(UseCase.qaGenerate.isStructured)
    }

    @Test("parse の応答は preview や warnings が欠けても読める")
    func decodeParseResponse() throws {
        let data = Data(#"{"parseResult":"partial","detectedTemplateVersion":"qa-v1","preview":{"items":[{"questionText":"Q","answerText":""}]}}"#.utf8)
        let response = try JSONCoding.makeDecoder().decode(ParseResponse.self, from: data)
        #expect(response.parseResult == .partial)
        #expect(response.warnings.isEmpty)
        #expect(response.preview.items?.first?.intentText == nil)
    }
}
