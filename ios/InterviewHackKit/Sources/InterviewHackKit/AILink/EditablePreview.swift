// DesignDocument/3_UIUXDesign.md 3.5.12 取り込みプレビュー（S-AI-03）/ 4_LLMDesign.md 4.2・4.3.7
import Foundation

public struct EditableQAItem: Identifiable, Equatable, Sendable {
    public let id: UUID
    public var questionText: String
    public var answerText: String
    /// 表示のみ。保存しない（4.2.1）
    public var intentText: String?

    public init(id: UUID = UUID(), questionText: String = "", answerText: String = "", intentText: String? = nil) {
        self.id = id
        self.questionText = questionText
        self.answerText = answerText
        self.intentText = intentText
    }
}

/// ユーザーが確認・修正するプレビュー。use_case ごとに形が異なる（7.5.5 の preview の構造）
public enum EditablePreview: Equatable, Sendable {
    case qa(items: [EditableQAItem])
    case review(issues: [String], improvements: [String], improvedAnswerText: String, saveToQa: Bool)
    case research(highlights: [String], idealCandidate: String, recentTopics: [String])
    case text(String)

    /// parse の結果からプレビューを作る。`failed` のときの全文は `AILinkFlow.fallbackText` に残す
    public static func make(useCase: UseCase, preview: RawPreview) -> EditablePreview {
        switch useCase {
        case .qaGenerate:
            let items = (preview.items ?? []).map {
                EditableQAItem(questionText: $0.questionText, answerText: $0.answerText, intentText: $0.intentText)
            }
            return .qa(items: items)
        case .answerReview:
            return .review(issues: preview.issues ?? [], improvements: preview.improvements ?? [],
                           improvedAnswerText: preview.improvedAnswerText ?? "", saveToQa: false)
        case .researchSummarize:
            return .research(highlights: preview.highlights ?? [], idealCandidate: preview.idealCandidate ?? "",
                             recentTopics: preview.recentTopics ?? [])
        case .summaryGenerate, .esGenerate, .esRefine:
            return .text(preview.text ?? "")
        }
    }

    public enum ValidationError: Error, Equatable, Sendable, LocalizedError {
        case noQAItems
        case emptyField(String)

        public var errorDescription: String? {
            switch self {
            case .noQAItems: return "保存する Q&A がありません。質問と回答を入力してください"
            case .emptyField(let name): return "\(name)を入力してください"
            }
        }
    }

    /// commit の本文を作る（7.5.5 の use_case ごとの Req）
    public func commitPayload(assetIds: [ID]?) throws -> CommitPayload {
        switch self {
        case .qa(let items):
            let valid = items
                .map { QACommitItem(questionText: $0.questionText.trimmed, answerText: $0.answerText.trimmed) }
                .filter { !$0.questionText.isEmpty && !$0.answerText.isEmpty }
            guard !valid.isEmpty else { throw ValidationError.noQAItems }
            return .qa(items: valid, assetIds: assetIds)
        case let .review(issues, improvements, improvedAnswerText, saveToQa):
            let improved = improvedAnswerText.trimmed
            guard !improved.isEmpty else { throw ValidationError.emptyField("改善後の回答") }
            return .answerReview(feedbackText: PreviewFormatter.feedbackText(issues: issues, improvements: improvements),
                                 improvedAnswerText: improved, saveToQa: saveToQa)
        case let .research(highlights, idealCandidate, recentTopics):
            let summary = PreviewFormatter.researchSummary(highlights: highlights, idealCandidate: idealCandidate,
                                                           recentTopics: recentTopics)
            guard !summary.isEmpty else { throw ValidationError.emptyField("要約") }
            return .researchSummary(summaryText: summary)
        case .text(let text):
            let body = text.trimmed
            guard !body.isEmpty else { throw ValidationError.emptyField("本文") }
            return .text(body)
        }
    }
}

/// 保存時の整形（4.2.2：指摘と改善案を ai_feedback_text に、4.2.4：3 つの見出しを付けた要約）
public enum PreviewFormatter {
    public static func feedbackText(issues: [String], improvements: [String]) -> String {
        var sections: [String] = []
        let issueLines = issues.map(\.trimmed).filter { !$0.isEmpty }
        let improvementLines = improvements.map(\.trimmed).filter { !$0.isEmpty }
        if !issueLines.isEmpty {
            sections.append("【指摘】\n" + issueLines.map { "- \($0)" }.joined(separator: "\n"))
        }
        if !improvementLines.isEmpty {
            sections.append("【改善案】\n" + improvementLines.map { "- \($0)" }.joined(separator: "\n"))
        }
        return sections.joined(separator: "\n")
    }

    public static func researchSummary(highlights: [String], idealCandidate: String, recentTopics: [String]) -> String {
        var sections: [String] = []
        let highlightLines = highlights.map(\.trimmed).filter { !$0.isEmpty }
        let topicLines = recentTopics.map(\.trimmed).filter { !$0.isEmpty }
        if !highlightLines.isEmpty {
            sections.append("【魅力・特徴】\n" + highlightLines.map { "- \($0)" }.joined(separator: "\n"))
        }
        if !idealCandidate.trimmed.isEmpty {
            sections.append("【求める人物像】\n" + idealCandidate.trimmed)
        }
        if !topicLines.isEmpty {
            sections.append("【最近のトピック】\n" + topicLines.map { "- \($0)" }.joined(separator: "\n"))
        }
        return sections.joined(separator: "\n")
    }
}

extension String {
    var trimmed: String { trimmingCharacters(in: .whitespacesAndNewlines) }
}
