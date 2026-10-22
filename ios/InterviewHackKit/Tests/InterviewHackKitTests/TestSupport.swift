// DesignDocument/11_TestDesign.md 11.4：API クライアントはモックの通信層で検証する
import Foundation
@testable import InterviewHackKit

struct MockResponse {
    var status: Int
    var body: String
    var headers: [String: String] = [:]

    static func json(_ status: Int, _ body: String, headers: [String: String] = [:]) -> MockResponse {
        MockResponse(status: status, body: body, headers: headers)
    }

    static let noContent = MockResponse(status: 204, body: "")
}

enum MockStep {
    case respond(MockResponse)
    case fail(URLError.Code)
}

/// 記録付きのモック通信層。登録した順に応答を返す
final class MockTransport: HTTPTransport, @unchecked Sendable {
    private let lock = NSLock()
    private var steps: [MockStep]
    private(set) var requests: [URLRequest] = []

    init(_ steps: [MockStep]) {
        self.steps = steps
    }

    var recorded: [URLRequest] { lock.withLock { requests } }

    func send(_ request: URLRequest) async throws -> (Data, HTTPURLResponse) {
        let step: MockStep = lock.withLock {
            requests.append(request)
            return steps.isEmpty ? .respond(.json(500, "{}")) : steps.removeFirst()
        }
        switch step {
        case .fail(let code):
            throw URLError(code)
        case .respond(let response):
            let http = HTTPURLResponse(url: request.url!, statusCode: response.status, httpVersion: "HTTP/1.1",
                                       headerFields: response.headers.merging(["Content-Type": "application/json"]) { a, _ in a })!
            return (Data(response.body.utf8), http)
        }
    }
}

final class SleepRecorder: @unchecked Sendable {
    private let lock = NSLock()
    private var values: [TimeInterval] = []
    var delays: [TimeInterval] { lock.withLock { values } }
    func record(_ value: TimeInterval) { lock.withLock { values.append(value) } }
}

final class Counter: @unchecked Sendable {
    private let lock = NSLock()
    private var value = 0
    var count: Int { lock.withLock { value } }
    func increment() { lock.withLock { value += 1 } }
}

func makeClient(_ transport: MockTransport, auth: AuthTokenProvider = InMemoryAuthService(signedInAs: AuthUser(uid: "u1", email: "a@example.com")),
                sleeps: SleepRecorder = SleepRecorder(), unauthorized: Counter = Counter()) -> APIClient {
    APIClient(baseURL: URL(string: "https://api.example.com")!, transport: transport, tokenProvider: auth,
              sleep: { sleeps.record($0) }, onUnauthorized: { unauthorized.increment() })
}

func temporaryDirectory() -> URL {
    FileManager.default.temporaryDirectory.appendingPathComponent("interviewhack-tests-\(UUID().uuidString)")
}

func bodyJSON(_ request: URLRequest) -> [String: Any] {
    guard let data = request.httpBody, let object = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
        return [:]
    }
    return object
}

let sessionPromptJSON = """
{"sessionId":"s1","templateVersion":"qa-v1","promptText":"あなたは面接官です"}
"""
