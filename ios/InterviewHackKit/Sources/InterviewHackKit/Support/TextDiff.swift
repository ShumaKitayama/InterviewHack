// DesignDocument/2_DomainDesign.md 2.2.6.1：差分は iOS 側で計算する（版の履歴 S-ES-05 / ES推敲のプレビュー S-AI-03）
import Foundation

public enum DiffSegment: Equatable, Sendable {
    case equal(String)
    case inserted(String)
    case deleted(String)
}

public enum TextDiff {
    /// 文字単位の差分（日本語の文章は空白で区切れないため、文字単位で比べる）。
    /// 長すぎる文章は計算量を抑えるため、行単位の比較に切り替える。
    public static func diff(old: String, new: String, characterLimit: Int = 4_000) -> [DiffSegment] {
        if old.count <= characterLimit && new.count <= characterLimit {
            return merge(lcsDiff(Array(old).map(String.init), Array(new).map(String.init)))
        }
        let oldLines = old.components(separatedBy: "\n").map { $0 + "\n" }
        let newLines = new.components(separatedBy: "\n").map { $0 + "\n" }
        return merge(lcsDiff(oldLines, newLines))
    }

    /// 変更された文字数（挿入＋削除）
    public static func changeCount(_ segments: [DiffSegment]) -> Int {
        segments.reduce(0) { total, segment in
            switch segment {
            case .equal: return total
            case .inserted(let text), .deleted(let text): return total + text.count
            }
        }
    }

    private static func lcsDiff(_ a: [String], _ b: [String]) -> [DiffSegment] {
        let n = a.count
        let m = b.count
        if n == 0 { return b.map { .inserted($0) } }
        if m == 0 { return a.map { .deleted($0) } }
        // lengths[i][j]：a[i...] と b[j...] の最長共通部分列の長さ
        var lengths = Array(repeating: Array(repeating: 0, count: m + 1), count: n + 1)
        for i in stride(from: n - 1, through: 0, by: -1) {
            for j in stride(from: m - 1, through: 0, by: -1) {
                lengths[i][j] = a[i] == b[j] ? lengths[i + 1][j + 1] + 1 : max(lengths[i + 1][j], lengths[i][j + 1])
            }
        }
        var result: [DiffSegment] = []
        var i = 0
        var j = 0
        while i < n && j < m {
            if a[i] == b[j] {
                result.append(.equal(a[i]))
                i += 1
                j += 1
            } else if lengths[i + 1][j] >= lengths[i][j + 1] {
                result.append(.deleted(a[i]))
                i += 1
            } else {
                result.append(.inserted(b[j]))
                j += 1
            }
        }
        while i < n { result.append(.deleted(a[i])); i += 1 }
        while j < m { result.append(.inserted(b[j])); j += 1 }
        return result
    }

    /// 同じ種類が続く部分をまとめる
    private static func merge(_ segments: [DiffSegment]) -> [DiffSegment] {
        var merged: [DiffSegment] = []
        for segment in segments {
            guard let last = merged.last else {
                merged.append(segment)
                continue
            }
            switch (last, segment) {
            case let (.equal(x), .equal(y)): merged[merged.count - 1] = .equal(x + y)
            case let (.inserted(x), .inserted(y)): merged[merged.count - 1] = .inserted(x + y)
            case let (.deleted(x), .deleted(y)): merged[merged.count - 1] = .deleted(x + y)
            default: merged.append(segment)
            }
        }
        return merged
    }
}
