// DesignDocument/5_FrontendDesign.md 5.4.2 AI連携の画面フロー（共通）
// プロンプト確認 → 共有 / コピー → 貼り戻し → parse → プレビュー編集 → commit
import Foundation
import Observation

/// AI連携で使う API（テストで差し替えられるようにする）
public protocol AILinkAPI: Sendable {
    func createPromptSession(_ request: PromptSessionRequest) async throws -> PromptText
    func promptText(sessionId: ID) async throws -> PromptText
    func parsePromptResponse(sessionId: ID, responseText: String) async throws -> ParseResponse
    func commitPromptSession(sessionId: ID, payload: CommitPayload) async throws -> CommitResponse
    func abandonPromptSession(sessionId: ID) async throws -> SessionStatusResponse
}

extension APIClient: AILinkAPI {}

@MainActor
@Observable
public final class AILinkFlow {
    public enum Step: Equatable, Sendable {
        /// 条件選択中（まだセッションがない）
        case idle
        case preparing
        /// S-AI-01 プロンプト確認
        case prompt
        /// S-AI-02 回答の貼り戻し
        case paste
        case parsing
        /// S-AI-03 取り込みプレビュー
        case preview
        case committing
        case saved(CommitResponse)
        case abandoned
    }

    public let useCase: UseCase
    public private(set) var step: Step = .idle
    public private(set) var sessionId: ID?
    public private(set) var templateVersion: String?
    /// S-AI-01 で編集できる（外部に渡すのは編集後の本文）
    public var promptText: String = ""
    public private(set) var promptWarnings: [String] = []
    /// 貼り戻した回答。入力のたびに端末へ下書き保存する（5.8.3）
    public var responseText: String = "" {
        didSet { persistResponseDraft() }
    }
    public private(set) var parseResult: ParseResult?
    public private(set) var parseWarnings: [String] = []
    public var preview: EditablePreview?
    /// `failed` のとき、全文を 1 つの編集欄として表示する（4.3.7）
    public private(set) var fallbackText: String?
    public private(set) var error: Error?
    /// 一時停止中（FEATURE_DISABLED）などの案内
    public private(set) var notice: String?
    /// ES 推敲の差分表示に使う元の本文（S-AI-03）
    public var originalText: String?
    /// qa_generate の保存時に紐付ける自己資産
    public var assetIds: [ID]?

    private let api: AILinkAPI
    private let drafts: DraftStore?

    public init(useCase: UseCase, api: AILinkAPI, drafts: DraftStore?) {
        self.useCase = useCase
        self.api = api
        self.drafts = drafts
    }

    public var isBusy: Bool {
        step == .preparing || step == .parsing || step == .committing
    }

    /// プロンプトの長さ（S-AI-01 に表示する）
    public var promptLength: Int { promptText.count }

    // MARK: - 開始・再開

    public func start(_ request: PromptSessionRequest) async {
        do {
            try request.validate()
        } catch {
            self.error = error
            return
        }
        step = .preparing
        error = nil
        assetIds = request.assetIds.isEmpty ? nil : request.assetIds
        do {
            let created = try await api.createPromptSession(request)
            apply(created)
            step = .prompt
        } catch {
            self.error = error
            step = .idle
        }
    }

    /// 未完了の AI連携から再開する（S-AI-04 / Home）。プロンプトは input_refs から再生成される
    public func resume(_ summary: PromptSessionSummary) async {
        sessionId = summary.sessionId
        templateVersion = summary.templateVersion
        step = .preparing
        error = nil
        if let draft = try? drafts?.load(TextDraft.self, for: .aiResponse(sessionId: summary.sessionId)) {
            responseText = draft.text
        }
        do {
            apply(try await api.promptText(sessionId: summary.sessionId))
            step = .paste
        } catch {
            self.error = error
            step = .paste
        }
    }

    private func apply(_ prompt: PromptText) {
        sessionId = prompt.sessionId
        templateVersion = prompt.templateVersion
        promptText = prompt.promptText
        promptWarnings = prompt.warnings ?? []
    }

    // MARK: - 受け渡し・貼り戻し

    /// 共有シート / コピーのあと（外部AIアプリへ移動する）
    public func didHandOffPrompt() {
        if step == .prompt { step = .paste }
    }

    public func showPrompt() {
        if step == .paste || step == .preview { step = .prompt }
    }

    public func showPaste() {
        if step == .prompt || step == .preview { step = .paste }
    }

    public func parse() async {
        guard let sessionId else { return }
        let text = responseText.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !text.isEmpty else {
            error = EditablePreview.ValidationError.emptyField("AIの回答")
            return
        }
        guard text.count <= ParseRequest.maxLength else {
            error = APIError.server(status: 422, code: .validationError,
                                    message: "回答が長すぎます（\(ParseRequest.maxLength) 文字まで）", requestId: nil)
            return
        }
        step = .parsing
        error = nil
        notice = nil
        do {
            let result = try await api.parsePromptResponse(sessionId: sessionId, responseText: text)
            parseResult = result.parseResult
            parseWarnings = result.warnings
            preview = EditablePreview.make(useCase: useCase, preview: result.preview)
            fallbackText = (result.parseResult == .failed && useCase.isStructured) ? text : nil
            step = .preview
        } catch {
            self.error = error
            step = .paste
        }
    }

    // MARK: - 保存・中止

    public func commit() async {
        guard let sessionId, let preview else { return }
        let payload: CommitPayload
        do {
            payload = try preview.commitPayload(assetIds: assetIds)
        } catch {
            self.error = error
            return
        }
        step = .committing
        error = nil
        notice = nil
        do {
            let response = try await api.commitPromptSession(sessionId: sessionId, payload: payload)
            try? drafts?.remove(.aiResponse(sessionId: sessionId))
            step = .saved(response)
        } catch let apiError as APIError where apiError.code == .featureDisabled {
            // 5.7.1：一時停止中。貼り戻した内容とプレビューを残し、あとで再度保存できるようにする
            notice = "この機能は一時停止中です。内容は残っているので、あとで保存してください"
            step = .preview
        } catch {
            self.error = error
            step = .preview
        }
    }

    public func abandon() async {
        guard let sessionId else {
            step = .abandoned
            return
        }
        do {
            _ = try await api.abandonPromptSession(sessionId: sessionId)
            try? drafts?.remove(.aiResponse(sessionId: sessionId))
            step = .abandoned
        } catch {
            self.error = error
        }
    }

    public func clearError() {
        error = nil
    }

    /// 5.7.1：SESSION_NOT_FOUND / SESSION_STATE_CONFLICT は最初からやり直すよう案内する
    public var needsRestart: Bool {
        guard let apiError = error as? APIError else { return false }
        return apiError.code == .sessionNotFound || apiError.code == .sessionStateConflict
    }

    private func persistResponseDraft() {
        guard let sessionId, let drafts else { return }
        if responseText.isEmpty {
            try? drafts.remove(.aiResponse(sessionId: sessionId))
        } else {
            try? drafts.save(TextDraft(text: responseText), for: .aiResponse(sessionId: sessionId))
        }
    }
}
