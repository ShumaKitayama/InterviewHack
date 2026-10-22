// DesignDocument/7_BackendDesign.md 7.5.2〜7.5.4 / 8_DBDesign.dbml のテーブルに対応する API の DTO（camelCase）
// サーバーの追加フィールドに強くするため、必須でない項目はすべて Optional にしている。
import Foundation

public typealias ID = String

/// 一覧 API の共通形式（7.5：`{ items, nextCursor }`）
public struct Page<Item: Codable & Sendable>: Codable, Sendable {
    public var items: [Item]
    public var nextCursor: String?

    public init(items: [Item], nextCursor: String? = nil) {
        self.items = items
        self.nextCursor = nextCursor
    }
}

public struct ItemsResponse<Item: Codable & Sendable>: Codable, Sendable {
    public var items: [Item]
}

// MARK: - Account (7.5.2)

public struct Me: Codable, Sendable, Equatable {
    public var id: ID
    public var email: String?
    public var displayName: String?
    public var createdAt: Date?
}

public struct MeUpdate: Codable, Sendable {
    public var displayName: String?
    public init(displayName: String?) { self.displayName = displayName }
}

// MARK: - 自己資産・タグ

public enum AssetType: String, Codable, CaseIterable, Sendable {
    case academic, work, project, skill, strength, weakness, episode, other

    public var label: String {
        switch self {
        case .academic: return "学歴・ガクチカ"
        case .work: return "職務経歴"
        case .project: return "プロジェクト"
        case .skill: return "スキル"
        case .strength: return "強み"
        case .weakness: return "弱み"
        case .episode: return "エピソード"
        case .other: return "その他"
        }
    }
}

public struct Asset: Codable, Sendable, Identifiable, Equatable {
    public var id: ID
    public var type: AssetType
    public var title: String
    public var content: String
    public var tagIds: [ID]?
    public var createdAt: Date?
    public var updatedAt: Date?
}

public struct AssetInput: Codable, Sendable {
    public var type: AssetType?
    public var title: String?
    public var content: String?
    public var tagIds: [ID]?

    public init(type: AssetType? = nil, title: String? = nil, content: String? = nil, tagIds: [ID]? = nil) {
        self.type = type
        self.title = title
        self.content = content
        self.tagIds = tagIds
    }
}

public struct AssetTag: Codable, Sendable, Identifiable, Equatable, Hashable {
    public var id: ID
    public var name: String
}

public struct AssetTagInput: Codable, Sendable {
    public var name: String
    public init(name: String) { self.name = name }
}

// MARK: - 企業・企業研究資料

public struct Company: Codable, Sendable, Identifiable, Equatable {
    public var id: ID
    public var name: String
    public var jobTitle: String?
    public var selectionPhase: String?
    public var motivationMemo: String?
    public var referenceUrl: String?
    public var createdAt: Date?
    public var updatedAt: Date?

    public init(id: ID, name: String, jobTitle: String? = nil, selectionPhase: String? = nil, motivationMemo: String? = nil,
                referenceUrl: String? = nil, createdAt: Date? = nil, updatedAt: Date? = nil) {
        self.id = id
        self.name = name
        self.jobTitle = jobTitle
        self.selectionPhase = selectionPhase
        self.motivationMemo = motivationMemo
        self.referenceUrl = referenceUrl
        self.createdAt = createdAt
        self.updatedAt = updatedAt
    }
}

public struct CompanyInput: Codable, Sendable {
    public var name: String?
    public var jobTitle: String?
    public var selectionPhase: String?
    public var motivationMemo: String?
    public var referenceUrl: String?

    public init(name: String? = nil, jobTitle: String? = nil, selectionPhase: String? = nil,
                motivationMemo: String? = nil, referenceUrl: String? = nil) {
        self.name = name
        self.jobTitle = jobTitle
        self.selectionPhase = selectionPhase
        self.motivationMemo = motivationMemo
        self.referenceUrl = referenceUrl
    }
}

public enum ResearchMaterialType: String, Codable, CaseIterable, Sendable {
    case url, pdf, note, image

    public var label: String {
        switch self {
        case .url: return "Webページ"
        case .pdf: return "PDF"
        case .note: return "メモ"
        case .image: return "画像"
        }
    }
}

public struct ResearchMaterial: Codable, Sendable, Identifiable, Equatable {
    public var id: ID
    public var companyId: ID
    public var title: String
    public var type: ResearchMaterialType
    public var sourceUrl: String?
    public var contentRaw: String?
    public var summary: String?
    public var createdAt: Date?
    public var updatedAt: Date?
}

public struct ResearchMaterialInput: Codable, Sendable {
    public var companyId: ID?
    public var title: String?
    public var type: ResearchMaterialType?
    public var sourceUrl: String?
    public var contentRaw: String?

    public init(companyId: ID? = nil, title: String? = nil, type: ResearchMaterialType? = nil,
                sourceUrl: String? = nil, contentRaw: String? = nil) {
        self.companyId = companyId
        self.title = title
        self.type = type
        self.sourceUrl = sourceUrl
        self.contentRaw = contentRaw
    }
}

// MARK: - ES（DesignDocument/2_DomainDesign.md 2.2.5 / 2.2.6 / 2.2.6.1）

public enum ESStatus: String, Codable, CaseIterable, Sendable {
    case notStarted = "not_started"
    case draft
    case inReview = "in_review"
    case completed
    case submitted

    public var label: String {
        switch self {
        case .notStarted: return "未着手"
        case .draft: return "作成中"
        case .inReview: return "添削中"
        case .completed: return "完了"
        case .submitted: return "提出済"
        }
    }

    /// 3.5.4：「提出済」を選ぶときだけ確認ダイアログを出す
    public var requiresConfirmation: Bool { self == .submitted }
}

public struct ESProject: Codable, Sendable, Identifiable, Equatable {
    public var id: ID
    public var companyId: ID
    public var status: ESStatus
    public var deadline: CalendarDate?
    public var assetIds: [ID]?
    public var researchMaterialIds: [ID]?
    public var createdAt: Date?
    public var updatedAt: Date?
}

public struct ESProjectInput: Codable, Sendable {
    public var companyId: ID?
    public var status: ESStatus?
    public var deadline: CalendarDate?
    public var assetIds: [ID]?
    public var researchMaterialIds: [ID]?

    public init(companyId: ID? = nil, status: ESStatus? = nil, deadline: CalendarDate? = nil,
                assetIds: [ID]? = nil, researchMaterialIds: [ID]? = nil) {
        self.companyId = companyId
        self.status = status
        self.deadline = deadline
        self.assetIds = assetIds
        self.researchMaterialIds = researchMaterialIds
    }
}

public struct ESSection: Codable, Sendable, Identifiable, Equatable {
    public var id: ID
    public var esProjectId: ID
    public var title: String
    public var contentDraft: String?
    public var contentFinal: String?
    public var characterLimit: Int?
    public var version: Int
    public var createdAt: Date?
    public var updatedAt: Date?
}

public struct ESSectionInput: Codable, Sendable {
    public var title: String?
    public var contentDraft: String?
    public var contentFinal: String?
    public var characterLimit: Int?

    public init(title: String? = nil, contentDraft: String? = nil, contentFinal: String? = nil, characterLimit: Int? = nil) {
        self.title = title
        self.contentDraft = contentDraft
        self.contentFinal = contentFinal
        self.characterLimit = characterLimit
    }
}

public enum ESSectionVersionSource: String, Codable, Sendable {
    case manual
    case esGenerate = "es_generate"
    case esRefine = "es_refine"

    public var label: String {
        switch self {
        case .manual: return "手動保存"
        case .esGenerate: return "ドラフト"
        case .esRefine: return "推敲"
        }
    }
}

public struct ESSectionVersion: Codable, Sendable, Identifiable, Equatable {
    public var id: ID
    public var version: Int
    public var source: ESSectionVersionSource
    public var content: String
    public var createdAt: Date?
}

// MARK: - 面接・面接ログ

public enum InterviewResult: String, Codable, CaseIterable, Sendable {
    case pass, fail, pending

    public var label: String {
        switch self {
        case .pass: return "通過"
        case .fail: return "不通過"
        case .pending: return "結果待ち"
        }
    }
}

public struct Interview: Codable, Sendable, Identifiable, Equatable {
    public var id: ID
    public var companyId: ID?
    public var interviewDate: CalendarDate
    public var phase: String?
    public var result: InterviewResult?
    public var memo: String?
    public var createdAt: Date?
    public var updatedAt: Date?
}

public struct InterviewInput: Codable, Sendable {
    public var companyId: ID?
    public var interviewDate: CalendarDate?
    public var phase: String?
    public var result: InterviewResult?
    public var memo: String?

    public init(companyId: ID? = nil, interviewDate: CalendarDate? = nil, phase: String? = nil,
                result: InterviewResult? = nil, memo: String? = nil) {
        self.companyId = companyId
        self.interviewDate = interviewDate
        self.phase = phase
        self.result = result
        self.memo = memo
    }
}

public struct InterviewLog: Codable, Sendable, Identifiable, Equatable {
    public var id: ID
    public var interviewId: ID
    public var questionText: String
    public var answerText: String
    public var selfEvaluation: String?
    public var aiFeedbackText: String?
    public var aiImprovedAnswerText: String?
    public var score: Int?
    public var createdAt: Date?
    public var updatedAt: Date?
}

public struct InterviewLogInput: Codable, Sendable, Equatable {
    public var interviewId: ID?
    public var questionText: String?
    public var answerText: String?
    public var selfEvaluation: String?

    public init(interviewId: ID? = nil, questionText: String? = nil, answerText: String? = nil, selfEvaluation: String? = nil) {
        self.interviewId = interviewId
        self.questionText = questionText
        self.answerText = answerText
        self.selfEvaluation = selfEvaluation
    }
}

// MARK: - Q&A

public enum QASourceType: String, Codable, Sendable {
    case preGenerated = "pre_generated"
    case reviewImproved = "review_improved"

    public var label: String {
        switch self {
        case .preGenerated: return "事前対策"
        case .reviewImproved: return "振り返り改善"
        }
    }
}

public enum QAReviewStatus: String, Codable, CaseIterable, Sendable {
    case new
    case reviewed
    case needsReview = "needs_review"

    public var label: String {
        switch self {
        case .new: return "未復習"
        case .reviewed: return "復習済み"
        case .needsReview: return "要再確認"
        }
    }
}

public struct QA: Codable, Sendable, Identifiable, Equatable {
    public var id: ID
    public var companyId: ID?
    public var questionText: String
    public var answerText: String
    public var sourceType: QASourceType
    public var isFavorite: Bool
    public var reviewStatus: QAReviewStatus
    public var assetIds: [ID]?
    public var createdAt: Date?
    public var updatedAt: Date?
}

public struct QAInput: Codable, Sendable {
    public var companyId: ID?
    public var questionText: String?
    public var answerText: String?
    public var isFavorite: Bool?
    public var reviewStatus: QAReviewStatus?
    public var assetIds: [ID]?

    public init(companyId: ID? = nil, questionText: String? = nil, answerText: String? = nil,
                isFavorite: Bool? = nil, reviewStatus: QAReviewStatus? = nil, assetIds: [ID]? = nil) {
        self.companyId = companyId
        self.questionText = questionText
        self.answerText = answerText
        self.isFavorite = isFavorite
        self.reviewStatus = reviewStatus
        self.assetIds = assetIds
    }
}

// MARK: - Dashboard (7.5.4)

public struct DashboardSummary: Codable, Sendable, Identifiable, Equatable {
    public var id: ID
    public var content: String
    public var sourceInterviewId: ID?
    public var createdAt: Date?
}

public struct PendingPromptSessions: Codable, Sendable, Equatable {
    public var count: Int
    public var items: [PromptSessionSummary]

    public init(count: Int, items: [PromptSessionSummary]) {
        self.count = count
        self.items = items
    }
}

public struct Dashboard: Codable, Sendable, Equatable {
    public var upcomingInterviews: [Interview]
    public var latestSummary: DashboardSummary?
    public var recentQa: [QA]
    public var recentInterviewLogs: [InterviewLog]
    public var pendingPromptSessions: PendingPromptSessions

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        upcomingInterviews = try c.decodeIfPresent([Interview].self, forKey: .upcomingInterviews) ?? []
        latestSummary = try c.decodeIfPresent(DashboardSummary.self, forKey: .latestSummary)
        recentQa = try c.decodeIfPresent([QA].self, forKey: .recentQa) ?? []
        recentInterviewLogs = try c.decodeIfPresent([InterviewLog].self, forKey: .recentInterviewLogs) ?? []
        pendingPromptSessions = try c.decodeIfPresent(PendingPromptSessions.self, forKey: .pendingPromptSessions)
            ?? PendingPromptSessions(count: 0, items: [])
    }

    public init(upcomingInterviews: [Interview] = [], latestSummary: DashboardSummary? = nil, recentQa: [QA] = [],
                recentInterviewLogs: [InterviewLog] = [], pendingPromptSessions: PendingPromptSessions = .init(count: 0, items: [])) {
        self.upcomingInterviews = upcomingInterviews
        self.latestSummary = latestSummary
        self.recentQa = recentQa
        self.recentInterviewLogs = recentInterviewLogs
        self.pendingPromptSessions = pendingPromptSessions
    }
}
