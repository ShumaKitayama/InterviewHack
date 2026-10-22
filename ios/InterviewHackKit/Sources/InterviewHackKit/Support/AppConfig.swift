// DesignDocument/5_FrontendDesign.md 5.12.1: xcconfig → Info.plist 経由で API_BASE_URL / APP_ENV を読む
import Foundation

public enum AppEnvironment: String, Sendable {
    case dev
    case prod
}

public struct AppConfig: Sendable, Equatable {
    public let apiBaseURL: URL
    public let environment: AppEnvironment

    public init(apiBaseURL: URL, environment: AppEnvironment) {
        self.apiBaseURL = apiBaseURL
        self.environment = environment
    }

    public enum LoadError: Error, Equatable {
        case missing(String)
        case invalid(String)
    }

    /// Info.plist の辞書から読む。キーは `API_BASE_URL` と `APP_ENV`。
    public static func load(from info: [String: Any]) throws -> AppConfig {
        guard let rawURL = info["API_BASE_URL"] as? String, !rawURL.isEmpty else {
            throw LoadError.missing("API_BASE_URL")
        }
        guard let url = URL(string: rawURL), let scheme = url.scheme, ["http", "https"].contains(scheme) else {
            throw LoadError.invalid("API_BASE_URL")
        }
        guard let rawEnv = info["APP_ENV"] as? String else {
            throw LoadError.missing("APP_ENV")
        }
        guard let env = AppEnvironment(rawValue: rawEnv) else {
            throw LoadError.invalid("APP_ENV")
        }
        // DesignDocument/5_FrontendDesign.md 5.13: HTTP は dev のローカル BFF のみ許可する
        if scheme == "http" {
            let host = url.host ?? ""
            guard env == .dev, ["localhost", "127.0.0.1"].contains(host) else {
                throw LoadError.invalid("API_BASE_URL")
            }
        }
        return AppConfig(apiBaseURL: url, environment: env)
    }
}
