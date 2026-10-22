// DesignDocument/7_BackendDesign.md 7.5: リクエスト / レスポンスは JSON、フィールド名は camelCase
import Foundation

public enum JSONCoding {
    /// BFF の JSON を読むデコーダ。日時は RFC3339（小数秒あり / なしの両方）を受け付ける。
    public static func makeDecoder() -> JSONDecoder {
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .custom { decoder in
            let container = try decoder.singleValueContainer()
            let raw = try container.decode(String.self)
            if let date = parseDate(raw) {
                return date
            }
            throw DecodingError.dataCorruptedError(in: container, debugDescription: "日時の形式が不正です: \(raw)")
        }
        return decoder
    }

    /// BFF に送る JSON を作るエンコーダ。日時は RFC3339 で送る。
    public static func makeEncoder() -> JSONEncoder {
        let encoder = JSONEncoder()
        encoder.dateEncodingStrategy = .custom { date, encoder in
            var container = encoder.singleValueContainer()
            try container.encode(formatDate(date))
        }
        encoder.outputFormatting = [.sortedKeys]
        return encoder
    }

    public static func parseDate(_ raw: String) -> Date? {
        let withFraction = ISO8601DateFormatter()
        withFraction.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let date = withFraction.date(from: raw) {
            return date
        }
        let plain = ISO8601DateFormatter()
        plain.formatOptions = [.withInternetDateTime]
        return plain.date(from: raw)
    }

    public static func formatDate(_ date: Date) -> String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime]
        return formatter.string(from: date)
    }
}

/// `YYYY-MM-DD` 形式の日付（ES の提出期限、面接日）。DB の `date` 型に対応する。
public struct CalendarDate: Codable, Hashable, Comparable, Sendable, CustomStringConvertible {
    public let year: Int
    public let month: Int
    public let day: Int

    public init(year: Int, month: Int, day: Int) {
        self.year = year
        self.month = month
        self.day = day
    }

    public init?(_ raw: String) {
        let parts = raw.prefix(10).split(separator: "-")
        guard parts.count == 3,
              let year = Int(parts[0]), let month = Int(parts[1]), let day = Int(parts[2]),
              (1...12).contains(month), (1...31).contains(day) else {
            return nil
        }
        self.init(year: year, month: month, day: day)
    }

    public init(date: Date, calendar: Calendar = .current) {
        let components = calendar.dateComponents([.year, .month, .day], from: date)
        self.init(year: components.year ?? 1970, month: components.month ?? 1, day: components.day ?? 1)
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        let raw = try container.decode(String.self)
        guard let value = CalendarDate(raw) else {
            throw DecodingError.dataCorruptedError(in: container, debugDescription: "日付の形式が不正です: \(raw)")
        }
        self = value
    }

    public func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        try container.encode(description)
    }

    public var description: String {
        String(format: "%04d-%02d-%02d", year, month, day)
    }

    public func date(calendar: Calendar = .current) -> Date? {
        calendar.date(from: DateComponents(year: year, month: month, day: day))
    }

    public static func < (lhs: CalendarDate, rhs: CalendarDate) -> Bool {
        (lhs.year, lhs.month, lhs.day) < (rhs.year, rhs.month, rhs.day)
    }
}
