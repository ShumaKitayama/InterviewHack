// DesignDocument/7_BackendDesign.md 7.7 エラー設計 / 5_FrontendDesign.md 5.7.1 エラーの変換
import Foundation

/// 7.7.2 のエラーコード。未知のコードは `.other` で受ける。
public enum APIErrorCode: Equatable, Sendable {
    case validationError
    case unauthorized
    case forbidden
    case notFound
    case sessionNotFound
    case conflict
    case sessionStateConflict
    case shareLinkGone
    case rateLimited
    case internalError
    case featureDisabled
    case other(String)

    public init(rawValue: String) {
        switch rawValue {
        case "VALIDATION_ERROR": self = .validationError
        case "UNAUTHORIZED": self = .unauthorized
        case "FORBIDDEN": self = .forbidden
        case "NOT_FOUND": self = .notFound
        case "SESSION_NOT_FOUND": self = .sessionNotFound
        case "CONFLICT": self = .conflict
        case "SESSION_STATE_CONFLICT": self = .sessionStateConflict
        case "SHARE_LINK_GONE": self = .shareLinkGone
        case "RATE_LIMITED": self = .rateLimited
        case "INTERNAL_ERROR": self = .internalError
        case "FEATURE_DISABLED": self = .featureDisabled
        default: self = .other(rawValue)
        }
    }

    public var rawValue: String {
        switch self {
        case .validationError: return "VALIDATION_ERROR"
        case .unauthorized: return "UNAUTHORIZED"
        case .forbidden: return "FORBIDDEN"
        case .notFound: return "NOT_FOUND"
        case .sessionNotFound: return "SESSION_NOT_FOUND"
        case .conflict: return "CONFLICT"
        case .sessionStateConflict: return "SESSION_STATE_CONFLICT"
        case .shareLinkGone: return "SHARE_LINK_GONE"
        case .rateLimited: return "RATE_LIMITED"
        case .internalError: return "INTERNAL_ERROR"
        case .featureDisabled: return "FEATURE_DISABLED"
        case .other(let raw): return raw
        }
    }

    /// HTTP ステータスだけが分かるとき（本文が JSON でない場合など）の推定
    static func fallback(for status: Int) -> APIErrorCode {
        switch status {
        case 400, 422: return .validationError
        case 401: return .unauthorized
        case 403: return .forbidden
        case 404: return .notFound
        case 409: return .conflict
        case 410: return .shareLinkGone
        case 429: return .rateLimited
        case 503: return .featureDisabled
        default: return .internalError
        }
    }
}

/// 7.7.1 のエラーレスポンス形式
struct ErrorBody: Decodable {
    var status: String?
    var code: String?
    var message: String?
    var requestId: String?

    enum CodingKeys: String, CodingKey {
        case status, code, message
        case requestId = "request_id"
        case requestIdCamel = "requestId"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        status = try c.decodeIfPresent(String.self, forKey: .status)
        code = try c.decodeIfPresent(String.self, forKey: .code)
        message = try c.decodeIfPresent(String.self, forKey: .message)
        requestId = try c.decodeIfPresent(String.self, forKey: .requestId)
            ?? c.decodeIfPresent(String.self, forKey: .requestIdCamel)
    }
}

public enum APIError: Error, Equatable, Sendable, LocalizedError {
    /// BFF が返したエラー（status / code / message）。`requestId` はエラー画面に小さく表示する
    case server(status: Int, code: APIErrorCode, message: String, requestId: String?)
    /// ネットワーク断・タイムアウトなど
    case network(String)
    /// レスポンスの形式が想定外
    case decoding(String)
    /// ログインしていない、またはトークンを再取得しても 401
    case notAuthenticated

    public var code: APIErrorCode? {
        if case let .server(_, code, _, _) = self { return code }
        if case .notAuthenticated = self { return .unauthorized }
        return nil
    }

    public var requestId: String? {
        if case let .server(_, _, _, requestId) = self { return requestId }
        return nil
    }

    public var httpStatus: Int? {
        if case let .server(status, _, _, _) = self { return status }
        return nil
    }

    /// 画面に表示する文言（5.7.1：画面には message を表示する）
    public var errorDescription: String? {
        switch self {
        case let .server(_, code, message, _):
            if code == .featureDisabled { return "この機能は一時停止中です" }
            return message.isEmpty ? "エラーが発生しました" : message
        case .network:
            return "通信できませんでした。電波の良い場所で再試行してください"
        case .decoding:
            return "サーバーの応答を読み取れませんでした"
        case .notAuthenticated:
            return "ログインが必要です"
        }
    }
}
