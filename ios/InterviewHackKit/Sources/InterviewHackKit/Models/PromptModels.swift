// DesignDocument/7_BackendDesign.md 7.5.5 AI連携（prompt_sessions）/ 方針転換メモ 2.10.1
import Foundation

public enum UseCase: String, Codable, CaseIterable, Sendable {
    case qaGenerate = "qa_generate"
    case answerReview = "answer_review"
    case summaryGenerate = "summary_generate"
    case researchSummarize = "research_summarize"
    case esGenerate = "es_generate"
    case esRefine = "es_refine"

    public var label: String {
        switch self {
        case .qaGenerate: return "想定Q&A"
        case .answerReview: return "面接ログの添削"
        case .summaryGenerate: return "次回のポイント要約"
        case .researchSummarize: return "企業研究資料の要約"
        case .esGenerate: return "ESドラフト"
        case .esRefine: return "ES推敲"
        }
    }

    /// 2.10.1：回答を解析するかどうか（解析しないものは全文を取り込む）
    public var isStructured: Bool {
        switch self {
        case .qaGenerate, .answerReview, .researchSummarize: return true
        case .summaryGenerate, .esGenerate, .esRefine: return false
        }
    }
}

public enum PromptSessionStatus: String, Codable, Sendable {
    case created, pasted, saved, abandoned

    /// 2.10.7：「未完了」は created と pasted
    public var isPending: Bool { self == .created || self == .pasted }
}

public enum ParseResult: String, Codable, Sendable {
    case success, partial, failed

    public var label: String {
        switch self {
        case .success: return "解析できました"
        case .partial: return "一部だけ解析できました"
        case .failed: return "解析できませんでした"
        }
    }
}

/// `POST /prompt-sessions` の Req（use_case ごとに必須の対象 ID が異なる）
public struct PromptSessionRequest: Codable, Sendable, Equatable {
    public var useCase: UseCase
    public var companyId: ID?
    public var interviewId: ID?
    public var interviewLogId: ID?
    public var esProjectId: ID?
    public var esSectionId: ID?
    public var assetIds: [ID]
    public var researchMaterialIds: [ID]
    public var options: PromptOptions?

    public init(useCase: UseCase, companyId: ID? = nil, interviewId: ID? = nil, interviewLogId: ID? = nil,
                esProjectId: ID? = nil, esSectionId: ID? = nil, assetIds: [ID] = [], researchMaterialIds: [ID] = [],
                options: PromptOptions? = nil) {
        self.useCase = useCase
        self.companyId = companyId
        self.interviewId = interviewId
        self.interviewLogId = interviewLogId
        self.esProjectId = esProjectId
        self.esSectionId = esSectionId
        self.assetIds = assetIds
        self.researchMaterialIds = researchMaterialIds
        self.options = options
    }

    public enum ValidationError: Error, Equatable {
        case missingTarget(String)
        case invalidResearchMaterialCount
    }

    /// 7.5.5 の「対象 ID（use_case ごとに必須のもの）」を送信前に検査する（正は BFF の検査）
    public func validate() throws {
        switch useCase {
        case .qaGenerate:
            if companyId == nil { throw ValidationError.missingTarget("companyId") }
        case .answerReview:
            if interviewLogId == nil { throw ValidationError.missingTarget("interviewLogId") }
        case .summaryGenerate:
            break
        case .researchSummarize:
            if companyId == nil { throw ValidationError.missingTarget("companyId") }
            if researchMaterialIds.count != 1 { throw ValidationError.invalidResearchMaterialCount }
        case .esGenerate, .esRefine:
            if esProjectId == nil { throw ValidationError.missingTarget("esProjectId") }
            if esSectionId == nil { throw ValidationError.missingTarget("esSectionId") }
        }
    }
}

public enum ReviewStrictness: String, Codable, Sendable, CaseIterable {
    case normal, hard

    public var label: String { self == .normal ? "標準" : "厳しめ" }
}

public struct PromptOptions: Codable, Sendable, Equatable {
    public var questionCount: Int?
    public var strictness: ReviewStrictness?
    public var instructions: String?

    public init(questionCount: Int? = nil, strictness: ReviewStrictness? = nil, instructions: String? = nil) {
        self.questionCount = questionCount
        self.strictness = strictness
        self.instructions = instructions
    }
}

/// `POST /prompt-sessions`（201）と `GET /prompt-sessions/{id}/prompt`（200）の Res
public struct PromptText: Codable, Sendable, Equatable {
    public var sessionId: ID
    public var templateVersion: String
    public var promptText: String
    /// 4.3.5：プロンプトが推奨上限（8,000 字）を超える場合などの警告
    public var warnings: [String]?
}

/// 未完了一覧（`GET /prompt-sessions`）の項目
public struct PromptSessionSummary: Codable, Sendable, Identifiable, Equatable {
    public var sessionId: ID
    public var useCase: UseCase
    public var templateVersion: String
    public var status: PromptSessionStatus
    /// 「対象の表示名」（例：企業名、ESセクション名）
    public var targetLabel: String?
    public var createdAt: Date?

    public var id: ID { sessionId }
}

// MARK: - parse（7.5.5 の preview の構造）

public struct QAPreviewItem: Codable, Sendable, Equatable, Hashable {
    public var questionText: String
    public var answerText: String
    /// 【意図N】。プレビューに表示するだけで保存しない（4.2.1）
    public var intentText: String?

    public init(questionText: String, answerText: String, intentText: String? = nil) {
        self.questionText = questionText
        self.answerText = answerText
        self.intentText = intentText
    }
}

/// use_case ごとに形が異なる preview。デコード時はすべての候補キーを Optional で受ける。
public struct RawPreview: Codable, Sendable, Equatable {
    public var items: [QAPreviewItem]?
    public var issues: [String]?
    public var improvements: [String]?
    public var improvedAnswerText: String?
    public var highlights: [String]?
    public var idealCandidate: String?
    public var recentTopics: [String]?
    public var text: String?

    public init(items: [QAPreviewItem]? = nil, issues: [String]? = nil, improvements: [String]? = nil,
                improvedAnswerText: String? = nil, highlights: [String]? = nil, idealCandidate: String? = nil,
                recentTopics: [String]? = nil, text: String? = nil) {
        self.items = items
        self.issues = issues
        self.improvements = improvements
        self.improvedAnswerText = improvedAnswerText
        self.highlights = highlights
        self.idealCandidate = idealCandidate
        self.recentTopics = recentTopics
        self.text = text
    }
}

public struct ParseResponse: Codable, Sendable, Equatable {
    public var parseResult: ParseResult
    public var detectedTemplateVersion: String?
    public var preview: RawPreview
    public var warnings: [String]

    public init(parseResult: ParseResult, detectedTemplateVersion: String? = nil, preview: RawPreview, warnings: [String] = []) {
        self.parseResult = parseResult
        self.detectedTemplateVersion = detectedTemplateVersion
        self.preview = preview
        self.warnings = warnings
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        parseResult = try c.decode(ParseResult.self, forKey: .parseResult)
        detectedTemplateVersion = try c.decodeIfPresent(String.self, forKey: .detectedTemplateVersion)
        preview = try c.decodeIfPresent(RawPreview.self, forKey: .preview) ?? RawPreview()
        warnings = try c.decodeIfPresent([String].self, forKey: .warnings) ?? []
    }
}

public struct ParseRequest: Codable, Sendable {
    public var responseText: String
    public init(responseText: String) { self.responseText = responseText }

    /// 7.5.5：最大 100,000 文字
    public static let maxLength = 100_000
}

// MARK: - commit（7.5.5 の use_case ごとの Req）

public struct QACommitItem: Codable, Sendable, Equatable {
    public var questionText: String
    public var answerText: String

    public init(questionText: String, answerText: String) {
        self.questionText = questionText
        self.answerText = answerText
    }
}

public enum CommitPayload: Sendable, Equatable {
    case qa(items: [QACommitItem], assetIds: [ID]?)
    case answerReview(feedbackText: String, improvedAnswerText: String, saveToQa: Bool)
    case researchSummary(summaryText: String)
    case text(String)
}

extension CommitPayload: Encodable {
    private enum CodingKeys: String, CodingKey {
        case items, assetIds, feedbackText, improvedAnswerText, saveToQa, summaryText, text
    }

    public func encode(to encoder: Encoder) throws {
        var c = encoder.container(keyedBy: CodingKeys.self)
        switch self {
        case let .qa(items, assetIds):
            try c.encode(items, forKey: .items)
            try c.encodeIfPresent(assetIds, forKey: .assetIds)
        case let .answerReview(feedbackText, improvedAnswerText, saveToQa):
            try c.encode(feedbackText, forKey: .feedbackText)
            try c.encode(improvedAnswerText, forKey: .improvedAnswerText)
            try c.encode(saveToQa, forKey: .saveToQa)
        case let .researchSummary(summaryText):
            try c.encode(summaryText, forKey: .summaryText)
        case let .text(text):
            try c.encode(text, forKey: .text)
        }
    }
}

public struct CommitResponse: Codable, Sendable, Equatable {
    public var sessionId: ID
    public var status: PromptSessionStatus
    public var savedRefs: [String: [ID]]

    public init(sessionId: ID, status: PromptSessionStatus, savedRefs: [String: [ID]] = [:]) {
        self.sessionId = sessionId
        self.status = status
        self.savedRefs = savedRefs
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        sessionId = try c.decode(ID.self, forKey: .sessionId)
        status = try c.decode(PromptSessionStatus.self, forKey: .status)
        savedRefs = try c.decodeIfPresent([String: [ID]].self, forKey: .savedRefs) ?? [:]
    }
}

public struct SessionStatusResponse: Codable, Sendable, Equatable {
    public var sessionId: ID
    public var status: PromptSessionStatus
}

// MARK: - 共有リンク（7.5.6）

public enum ShareLinkExpiry: Int, Codable, CaseIterable, Sendable {
    case oneDay = 1
    case sevenDays = 7
    case thirtyDays = 30

    /// D12：初期値は 7 日
    public static let `default`: ShareLinkExpiry = .sevenDays

    public var label: String { "\(rawValue)日" }
}

public struct ShareLinkRequest: Codable, Sendable, Equatable {
    public var sectionIds: [ID]
    public var expiresInDays: ShareLinkExpiry
    public var showCompanyName: Bool

    public init(sectionIds: [ID], expiresInDays: ShareLinkExpiry = .default, showCompanyName: Bool = false) {
        self.sectionIds = sectionIds
        self.expiresInDays = expiresInDays
        self.showCompanyName = showCompanyName
    }
}

public struct CreatedShareLink: Codable, Sendable, Equatable {
    public var id: ID
    /// 生トークンを含む URL。返されるのはこの応答だけ（再表示できない）
    public var url: String
    public var expiresAt: Date
}

public struct ESShareLink: Codable, Sendable, Identifiable, Equatable {
    public var id: ID
    public var sectionTitles: [String]
    public var showCompanyName: Bool
    public var expiresAt: Date
    public var revokedAt: Date?
    public var createdAt: Date?
    public var isActive: Bool

    public enum State: Equatable, Sendable {
        case active, expired, revoked

        public var label: String {
            switch self {
            case .active: return "公開中"
            case .expired: return "期限切れ"
            case .revoked: return "無効"
            }
        }
    }

    /// 3.5.14：公開中 / 期限切れ / 無効
    public func state(now: Date = Date()) -> State {
        if revokedAt != nil { return .revoked }
        if now >= expiresAt { return .expired }
        return isActive ? .active : .expired
    }
}
