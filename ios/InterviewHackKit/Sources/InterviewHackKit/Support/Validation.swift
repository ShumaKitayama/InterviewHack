// DesignDocument/5_FrontendDesign.md 5.9 フォームとバリデーション（正は BFF の検査）
import Foundation

public enum FieldRule: Sendable {
    /// 面接ログの質問（5.9.1 の例：300 字）
    public static let questionMaxLength = 300
    /// 面接ログの回答（5.9.1 の例：2000 字）
    public static let answerMaxLength = 2_000
}

public enum FieldValidationError: Error, Equatable, Sendable, LocalizedError {
    case required(String)
    case tooLong(String, max: Int)
    case invalidURL

    public var errorDescription: String? {
        switch self {
        case .required(let name): return "\(name)を入力してください"
        case let .tooLong(name, max): return "\(name)は \(max) 文字以内で入力してください"
        case .invalidURL: return "URL の形式が正しくありません"
        }
    }
}

public enum Validator {
    /// 空文字や空白だけの入力は不可
    public static func required(_ value: String, name: String) throws {
        if value.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            throw FieldValidationError.required(name)
        }
    }

    public static func maxLength(_ value: String, _ max: Int, name: String) throws {
        if value.count > max { throw FieldValidationError.tooLong(name, max: max) }
    }

    public static func interviewLog(_ input: InterviewLogInput) throws {
        try required(input.questionText ?? "", name: "質問")
        try required(input.answerText ?? "", name: "回答")
        try maxLength(input.questionText ?? "", FieldRule.questionMaxLength, name: "質問")
        try maxLength(input.answerText ?? "", FieldRule.answerMaxLength, name: "回答")
    }

    /// 5.10.1：取り込む URL は http / https のみ
    public static func webURL(_ raw: String) throws -> URL {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        let candidate = trimmed.contains("://") ? trimmed : "https://" + trimmed
        guard let url = URL(string: candidate), let scheme = url.scheme?.lowercased(),
              ["http", "https"].contains(scheme), let host = url.host, host.contains(".") else {
            throw FieldValidationError.invalidURL
        }
        return url
    }
}

/// ES セクションの文字数（5.9.1：文字数制限に対する現在の文字数を常に表示する）
public struct CharacterCount: Equatable, Sendable {
    public var count: Int
    public var limit: Int?

    public init(text: String, limit: Int?) {
        count = text.count
        self.limit = limit
    }

    /// 4.2.5：指定文字数の ±10% 以内か
    public var isWithinTolerance: Bool {
        guard let limit, limit > 0 else { return true }
        let lower = Int((Double(limit) * 0.9).rounded(.up))
        let upper = Int((Double(limit) * 1.1).rounded(.down))
        return (lower...upper).contains(count)
    }

    public var isOverLimit: Bool {
        guard let limit else { return false }
        return count > limit
    }

    public var label: String {
        if let limit { return "\(count) / \(limit) 字" }
        return "\(count) 字"
    }
}
