// DesignDocument/5_FrontendDesign.md 5.7 APIクライアント設計 / 7_BackendDesign.md 7.7.3 リクエストID
import Foundation

public enum HTTPMethod: String, Sendable {
    case get = "GET", post = "POST", patch = "PATCH", delete = "DELETE"
}

/// 本文のない応答（204 など）
public struct EmptyResponse: Decodable, Sendable, Equatable {
    public init() {}
}

public struct Endpoint<Response: Decodable & Sendable>: Sendable {
    public var method: HTTPMethod
    public var path: String
    public var query: [URLQueryItem]
    public var body: Data?
    /// `GET /health` 以外は認証が必要（5.7.1）
    public var requiresAuth: Bool
    /// 5.7.2：書き込みは原則自動で再送しない。例外は commit（条件付き更新で二重保存しない）
    public var retryableWrite: Bool

    public init(method: HTTPMethod, path: String, query: [URLQueryItem] = [], body: Data? = nil,
                requiresAuth: Bool = true, retryableWrite: Bool = false) {
        self.method = method
        self.path = path
        self.query = query
        self.body = body
        self.requiresAuth = requiresAuth
        self.retryableWrite = retryableWrite
    }

    public static func json<Body: Encodable>(_ method: HTTPMethod, _ path: String, body: Body,
                                             query: [URLQueryItem] = [], retryableWrite: Bool = false) -> Endpoint {
        // エンコードに失敗するのはプログラムの誤りなので、空の本文で送らずに即座に気付けるようにする
        let data = try! JSONCoding.makeEncoder().encode(body)
        return Endpoint(method: method, path: path, query: query, body: data, retryableWrite: retryableWrite)
    }
}

/// URLSession を差し替えられるようにする（テストではモックを注入する）
public protocol HTTPTransport: Sendable {
    func send(_ request: URLRequest) async throws -> (Data, HTTPURLResponse)
}

public struct URLSessionTransport: HTTPTransport {
    private let session: URLSession

    /// 5.7.2：Neon のコールドスタートを考え、タイムアウトは 15 秒
    public static let timeout: TimeInterval = 15

    public init(session: URLSession? = nil) {
        if let session {
            self.session = session
        } else {
            let configuration = URLSessionConfiguration.ephemeral
            configuration.timeoutIntervalForRequest = Self.timeout
            configuration.requestCachePolicy = .reloadIgnoringLocalCacheData
            configuration.urlCache = nil
            self.session = URLSession(configuration: configuration)
        }
    }

    public func send(_ request: URLRequest) async throws -> (Data, HTTPURLResponse) {
        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse else {
            throw URLError(.badServerResponse)
        }
        return (data, http)
    }
}

public struct RetryPolicy: Sendable {
    /// 5.7.2：指数バックオフで最大 2 回
    public var maxRetries: Int
    public var baseDelay: TimeInterval

    public init(maxRetries: Int = 2, baseDelay: TimeInterval = 0.5) {
        self.maxRetries = maxRetries
        self.baseDelay = baseDelay
    }

    public func delay(forAttempt attempt: Int, retryAfter: TimeInterval?) -> TimeInterval {
        if let retryAfter { return retryAfter }
        return baseDelay * pow(2, Double(attempt))
    }
}

public final class APIClient: Sendable {
    public let baseURL: URL
    private let transport: HTTPTransport
    private let tokenProvider: AuthTokenProvider
    private let requestIDGenerator: @Sendable () -> String
    private let retryPolicy: RetryPolicy
    private let sleep: @Sendable (TimeInterval) async -> Void
    private let onUnauthorized: @Sendable () async -> Void

    public init(baseURL: URL,
                transport: HTTPTransport = URLSessionTransport(),
                tokenProvider: AuthTokenProvider,
                requestIDGenerator: @escaping @Sendable () -> String = { UUID().uuidString.lowercased() },
                retryPolicy: RetryPolicy = RetryPolicy(),
                sleep: @escaping @Sendable (TimeInterval) async -> Void = { seconds in
                    try? await Task.sleep(nanoseconds: UInt64(seconds * 1_000_000_000))
                },
                onUnauthorized: @escaping @Sendable () async -> Void = {}) {
        self.baseURL = baseURL
        self.transport = transport
        self.tokenProvider = tokenProvider
        self.requestIDGenerator = requestIDGenerator
        self.retryPolicy = retryPolicy
        self.sleep = sleep
        self.onUnauthorized = onUnauthorized
    }

    public func send<Response>(_ endpoint: Endpoint<Response>) async throws -> Response {
        let data = try await sendRaw(endpoint)
        if Response.self == EmptyResponse.self {
            return EmptyResponse() as! Response
        }
        do {
            return try JSONCoding.makeDecoder().decode(Response.self, from: data)
        } catch {
            throw APIError.decoding(String(describing: error))
        }
    }

    // MARK: - 送信本体

    private func sendRaw<Response>(_ endpoint: Endpoint<Response>) async throws -> Data {
        var token: String?
        if endpoint.requiresAuth {
            token = try await tokenProvider.idToken(forceRefresh: false)
            guard token != nil else { throw APIError.notAuthenticated }
        }

        var didRefreshToken = false
        var attempt = 0
        while true {
            let request = try makeRequest(endpoint, token: token)
            let canRetryUnknownOutcome = endpoint.method == .get || endpoint.retryableWrite

            let data: Data
            let response: HTTPURLResponse
            do {
                (data, response) = try await transport.send(request)
            } catch {
                // 5.7.2：ネットワーク断・タイムアウトは、処理済みか分からないため GET と commit だけ再送する
                if canRetryUnknownOutcome, attempt < retryPolicy.maxRetries, Self.isRetryable(error) {
                    await sleep(retryPolicy.delay(forAttempt: attempt, retryAfter: nil))
                    attempt += 1
                    continue
                }
                throw APIError.network(String(describing: error))
            }

            if (200..<300).contains(response.statusCode) {
                return data
            }

            let apiError = Self.makeError(status: response.statusCode, data: data, response: response)

            // 5.5.2：401 なら ID トークンを強制更新して一度だけ再送する
            if response.statusCode == 401, endpoint.requiresAuth, !didRefreshToken {
                didRefreshToken = true
                token = try await tokenProvider.idToken(forceRefresh: true)
                if token != nil { continue }
            }
            if response.statusCode == 401, endpoint.requiresAuth {
                await onUnauthorized()
                throw APIError.notAuthenticated
            }

            // 5.7.2：429 はサーバーが処理していないので再送してよい。503 は機能停止（FEATURE_DISABLED）以外を再送する
            if attempt < retryPolicy.maxRetries {
                let retryAfter = Self.retryAfter(response)
                if response.statusCode == 429 {
                    await sleep(retryPolicy.delay(forAttempt: attempt, retryAfter: retryAfter))
                    attempt += 1
                    continue
                }
                if response.statusCode == 503, apiError.code != .featureDisabled, canRetryUnknownOutcome {
                    await sleep(retryPolicy.delay(forAttempt: attempt, retryAfter: retryAfter))
                    attempt += 1
                    continue
                }
            }
            throw apiError
        }
    }

    func makeRequest<Response>(_ endpoint: Endpoint<Response>, token: String?) throws -> URLRequest {
        var components = URLComponents(url: baseURL.appendingPathComponent(endpoint.path), resolvingAgainstBaseURL: false)
        if !endpoint.query.isEmpty {
            components?.queryItems = endpoint.query
        }
        guard let url = components?.url else {
            throw APIError.decoding("URL を組み立てられません: \(endpoint.path)")
        }
        var request = URLRequest(url: url, timeoutInterval: URLSessionTransport.timeout)
        request.httpMethod = endpoint.method.rawValue
        // 7.7.3：リクエストごとに X-Request-Id を生成して付与する
        request.setValue(requestIDGenerator(), forHTTPHeaderField: "X-Request-Id")
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        if let body = endpoint.body {
            request.httpBody = body
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        }
        if let token {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        return request
    }

    static func makeError(status: Int, data: Data, response: HTTPURLResponse) -> APIError {
        let headerRequestId = response.value(forHTTPHeaderField: "X-Request-Id")
        if let body = try? JSONCoding.makeDecoder().decode(ErrorBody.self, from: data), let code = body.code {
            return .server(status: status, code: APIErrorCode(rawValue: code), message: body.message ?? "",
                           requestId: body.requestId ?? headerRequestId)
        }
        return .server(status: status, code: .fallback(for: status), message: "", requestId: headerRequestId)
    }

    static func isRetryable(_ error: Error) -> Bool {
        guard let urlError = error as? URLError else { return false }
        switch urlError.code {
        case .timedOut, .networkConnectionLost, .notConnectedToInternet, .cannotConnectToHost,
             .cannotFindHost, .dnsLookupFailed:
            return true
        default:
            return false
        }
    }

    static func retryAfter(_ response: HTTPURLResponse) -> TimeInterval? {
        guard let raw = response.value(forHTTPHeaderField: "Retry-After"), let seconds = TimeInterval(raw) else {
            return nil
        }
        return max(0, seconds)
    }
}
