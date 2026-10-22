// DesignDocument/11_TestDesign.md 11.4（iOS）/ 5_FrontendDesign.md 5.7 APIクライアント
import Foundation
import Testing
@testable import InterviewHackKit

@Suite("APIClient")
struct APIClientTests {
    @Test("Authorization と X-Request-Id（リクエストごとに異なる UUID）を付ける")
    func addsHeaders() async throws {
        let transport = MockTransport([
            .respond(.json(200, #"{"id":"me1","email":"a@example.com"}"#)),
            .respond(.json(200, #"{"id":"me1"}"#)),
        ])
        let client = makeClient(transport)
        _ = try await client.me()
        _ = try await client.me()

        let requests = transport.recorded
        #expect(requests.count == 2)
        #expect(requests[0].value(forHTTPHeaderField: "Authorization") == "Bearer token-u1-0")
        let ids = requests.compactMap { $0.value(forHTTPHeaderField: "X-Request-Id") }
        #expect(ids.count == 2)
        #expect(ids[0] != ids[1])
        #expect(UUID(uuidString: ids[0]) != nil)
        #expect(requests[0].url?.absoluteString == "https://api.example.com/me")
    }

    @Test("GET /health は認証ヘッダを付けない")
    func healthWithoutAuth() async throws {
        let transport = MockTransport([.respond(.json(200, #"{"status":"ok"}"#))])
        let client = makeClient(transport, auth: InMemoryAuthService())
        let status = try await client.health()
        #expect(status.status == "ok")
        #expect(transport.recorded[0].value(forHTTPHeaderField: "Authorization") == nil)
        #expect(transport.recorded[0].value(forHTTPHeaderField: "X-Request-Id") != nil)
    }

    @Test("未ログインで認証付き API を呼ぶと notAuthenticated（通信しない）")
    func notSignedIn() async throws {
        let transport = MockTransport([])
        let client = makeClient(transport, auth: InMemoryAuthService())
        await #expect(throws: APIError.notAuthenticated) { _ = try await client.me() }
        #expect(transport.recorded.isEmpty)
    }

    @Test("7.7.1 のエラー形式を APIError に変換する（code / message / request_id）")
    func convertsErrorBody() async throws {
        let transport = MockTransport([
            .respond(.json(404, #"{"status":"error","code":"NOT_FOUND","message":"企業 が見つかりません","request_id":"req-123"}"#)),
        ])
        let client = makeClient(transport)
        do {
            _ = try await client.company("c1")
            Issue.record("エラーになるはず")
        } catch let error as APIError {
            #expect(error == .server(status: 404, code: .notFound, message: "企業 が見つかりません", requestId: "req-123"))
            #expect(error.errorDescription == "企業 が見つかりません")
            #expect(error.requestId == "req-123")
        }
    }

    @Test("本文が JSON でない場合はステータスからコードを推定し、ヘッダの X-Request-Id を使う")
    func fallbackErrorCode() async throws {
        let transport = MockTransport([.respond(MockResponse(status: 410, body: "<html>gone</html>", headers: ["X-Request-Id": "r9"]))])
        let client = makeClient(transport)
        do {
            _ = try await client.qa("q1")
            Issue.record("エラーになるはず")
        } catch let error as APIError {
            #expect(error.code == .shareLinkGone)
            #expect(error.requestId == "r9")
        }
    }

    @Test("401 ならトークンを強制更新して一度だけ再送する")
    func refreshesTokenOnce() async throws {
        let auth = InMemoryAuthService(signedInAs: AuthUser(uid: "u1", email: nil))
        let transport = MockTransport([
            .respond(.json(401, #"{"status":"error","code":"UNAUTHORIZED","message":"認証が必要です"}"#)),
            .respond(.json(200, #"{"id":"me1"}"#)),
        ])
        let client = makeClient(transport, auth: auth)
        let me = try await client.me()
        #expect(me.id == "me1")
        #expect(auth.forcedRefreshCount == 1)
        #expect(transport.recorded[0].value(forHTTPHeaderField: "Authorization") == "Bearer token-u1-0")
        #expect(transport.recorded[1].value(forHTTPHeaderField: "Authorization") == "Bearer token-u1-1")
    }

    @Test("再取得しても 401 なら notAuthenticated にし、ログアウト処理を呼ぶ")
    func unauthorizedTwice() async throws {
        let unauthorized = Counter()
        let transport = MockTransport([
            .respond(.json(401, #"{"status":"error","code":"UNAUTHORIZED","message":""}"#)),
            .respond(.json(401, #"{"status":"error","code":"UNAUTHORIZED","message":""}"#)),
        ])
        let client = makeClient(transport, unauthorized: unauthorized)
        await #expect(throws: APIError.notAuthenticated) { _ = try await client.me() }
        #expect(transport.recorded.count == 2)
        #expect(unauthorized.count == 1)
    }

    @Test("GET はネットワーク断で最大 2 回まで指数バックオフで再送する")
    func retriesGetOnNetworkError() async throws {
        let sleeps = SleepRecorder()
        let transport = MockTransport([.fail(.timedOut), .fail(.networkConnectionLost), .respond(.json(200, #"{"items":[],"nextCursor":null}"#))])
        let client = makeClient(transport, sleeps: sleeps)
        let page = try await client.assets()
        #expect(page.items.isEmpty)
        #expect(transport.recorded.count == 3)
        #expect(sleeps.delays == [0.5, 1.0])
    }

    @Test("GET でも 3 回目の失敗でネットワークエラーにする")
    func givesUpAfterMaxRetries() async throws {
        let transport = MockTransport([.fail(.timedOut), .fail(.timedOut), .fail(.timedOut)])
        let client = makeClient(transport)
        await #expect(throws: APIError.self) { _ = try await client.assets() }
        #expect(transport.recorded.count == 3)
    }

    @Test("書き込み（POST）は処理済みか分からないネットワーク断では再送しない")
    func doesNotRetryWritesOnNetworkError() async throws {
        let transport = MockTransport([.fail(.timedOut), .respond(.json(201, "{}"))])
        let client = makeClient(transport)
        await #expect(throws: APIError.self) { _ = try await client.createCompany(CompanyInput(name: "A社")) }
        #expect(transport.recorded.count == 1)
    }

    @Test("commit はネットワーク断でも再送してよい（条件付き更新で二重保存しない）")
    func retriesCommit() async throws {
        let transport = MockTransport([.fail(.timedOut), .respond(.json(200, #"{"sessionId":"s1","status":"saved","savedRefs":{"qa":["q1"]}}"#))])
        let client = makeClient(transport)
        let response = try await client.commitPromptSession(sessionId: "s1", payload: .text("本文"))
        #expect(response.status == .saved)
        #expect(response.savedRefs["qa"] == ["q1"])
        #expect(transport.recorded.count == 2)
    }

    @Test("429 は Retry-After の秒数を待って再送する")
    func honorsRetryAfter() async throws {
        let sleeps = SleepRecorder()
        let transport = MockTransport([
            .respond(.json(429, #"{"status":"error","code":"RATE_LIMITED","message":"多すぎます"}"#, headers: ["Retry-After": "3"])),
            .respond(.json(201, #"{"id":"c1","name":"A社"}"#)),
        ])
        let client = makeClient(transport, sleeps: sleeps)
        let company = try await client.createCompany(CompanyInput(name: "A社"))
        #expect(company.id == "c1")
        #expect(sleeps.delays == [3])
    }

    @Test("503 FEATURE_DISABLED は再送せずにそのまま返す")
    func featureDisabledIsNotRetried() async throws {
        let transport = MockTransport([
            .respond(.json(503, #"{"status":"error","code":"FEATURE_DISABLED","message":"停止中"}"#)),
        ])
        let client = makeClient(transport)
        do {
            _ = try await client.commitPromptSession(sessionId: "s1", payload: .text("本文"))
            Issue.record("エラーになるはず")
        } catch let error as APIError {
            #expect(error.code == .featureDisabled)
            #expect(error.errorDescription == "この機能は一時停止中です")
        }
        #expect(transport.recorded.count == 1)
    }

    @Test("commit の 409 は保存済みか確認し、saved なら成功として扱う")
    func commitConflictTreatedAsSaved() async throws {
        let transport = MockTransport([
            .respond(.json(409, #"{"status":"error","code":"SESSION_STATE_CONFLICT","message":"保存済みです"}"#)),
            .respond(.json(200, #"{"sessionId":"s1","useCase":"qa_generate","templateVersion":"qa-v1","status":"saved","targetLabel":"株式会社テスト","createdAt":"2026-10-22T04:06:15.672156+09:00"}"#)),
        ])
        let client = makeClient(transport)
        let response = try await client.commitPromptSession(sessionId: "s1", payload: .text("x"))
        #expect(response == CommitResponse(sessionId: "s1", status: .saved))
        let check = transport.recorded[1]
        #expect(check.httpMethod == "GET")
        #expect(check.url?.path.hasSuffix("/prompt-sessions/s1") == true)
    }

    @Test("commit の 409 で保存済みでなければエラーのまま返す")
    func commitConflictNotSaved() async throws {
        let transport = MockTransport([
            .respond(.json(409, #"{"status":"error","code":"SESSION_STATE_CONFLICT","message":"中止済み"}"#)),
            .respond(.json(200, #"{"sessionId":"s1","useCase":"qa_generate","templateVersion":"qa-v1","status":"abandoned"}"#)),
        ])
        let client = makeClient(transport)
        do {
            _ = try await client.commitPromptSession(sessionId: "s1", payload: .text("x"))
            Issue.record("エラーになるはず")
        } catch let error as APIError {
            #expect(error.code == .sessionStateConflict)
        }
    }

    @Test("未完了一覧は status=created,pasted で取得する")
    func pendingSessionsQuery() async throws {
        let transport = MockTransport([.respond(.json(200, #"{"items":[{"sessionId":"s1","useCase":"es_refine","templateVersion":"esrefine-v1","status":"pasted","targetLabel":"志望動機","createdAt":"2026-10-22T10:00:00Z"}],"nextCursor":null}"#))])
        let client = makeClient(transport)
        let page = try await client.promptSessions()
        #expect(page.items.first?.useCase == .esRefine)
        #expect(page.items.first?.status.isPending == true)
        let components = URLComponents(url: transport.recorded[0].url!, resolvingAgainstBaseURL: false)
        #expect(components?.queryItems?.first { $0.name == "status" }?.value == "created,pasted")
    }

    @Test("prompt-sessions の作成は use_case と対象 ID を camelCase で送る")
    func createPromptSessionBody() async throws {
        let transport = MockTransport([.respond(.json(201, sessionPromptJSON))])
        let client = makeClient(transport)
        let request = PromptSessionRequest(useCase: .qaGenerate, companyId: "c1", assetIds: ["a1", "a2"],
                                           options: PromptOptions(questionCount: 10))
        let prompt = try await client.createPromptSession(request)
        #expect(prompt.templateVersion == "qa-v1")
        let body = bodyJSON(transport.recorded[0])
        #expect(body["useCase"] as? String == "qa_generate")
        #expect(body["companyId"] as? String == "c1")
        #expect(body["assetIds"] as? [String] == ["a1", "a2"])
        #expect((body["options"] as? [String: Any])?["questionCount"] as? Int == 10)
        #expect(body["interviewLogId"] == nil)
    }

    @Test("共有リンクの発行は既定で 7 日・企業名を表示しない")
    func shareLinkDefaults() async throws {
        let transport = MockTransport([.respond(.json(201, #"{"id":"l1","url":"https://api.example.com/s/tok","expiresAt":"2026-10-29T00:00:00.123Z"}"#))])
        let client = makeClient(transport)
        let created = try await client.createShareLink(projectId: "p1", ShareLinkRequest(sectionIds: ["sec1"]))
        #expect(created.url.hasSuffix("/s/tok"))
        let body = bodyJSON(transport.recorded[0])
        #expect(body["expiresInDays"] as? Int == 7)
        #expect(body["showCompanyName"] as? Bool == false)
        #expect(transport.recorded[0].url?.path == "/es-projects/p1/share-links")
    }

    @Test("DELETE の 204 は本文なしで成功にする")
    func deleteNoContent() async throws {
        let transport = MockTransport([.respond(.noContent)])
        let client = makeClient(transport)
        try await client.revokeShareLink("l1")
        #expect(transport.recorded[0].httpMethod == "DELETE")
        #expect(transport.recorded[0].url?.path == "/share-links/l1")
    }

    @Test("版の履歴・保存・戻す のパス")
    func versionEndpoints() async throws {
        let transport = MockTransport([
            .respond(.json(200, #"{"items":[{"id":"v1","version":2,"source":"es_refine","content":"旧本文","createdAt":"2026-10-22T10:00:00Z"}]}"#)),
            .respond(.json(201, #"{"id":"v2","version":3,"source":"manual","content":"今の本文"}"#)),
            .respond(.json(200, #"{"id":"sec1","esProjectId":"p1","title":"志望動機","contentDraft":"旧本文","version":4}"#)),
        ])
        let client = makeClient(transport)
        let versions = try await client.esSectionVersions(sectionId: "sec1")
        #expect(versions.first?.source == .esRefine)
        _ = try await client.saveESSectionVersion(sectionId: "sec1")
        let restored = try await client.restoreESSectionVersion(sectionId: "sec1", versionId: "v1")
        #expect(restored.version == 4)
        #expect(transport.recorded.map { $0.url!.path } == [
            "/es-sections/sec1/versions", "/es-sections/sec1/versions", "/es-sections/sec1/versions/v1/restore",
        ])
    }

    @Test("ダッシュボードは欠けた項目を空として読む")
    func dashboardDefaults() async throws {
        let transport = MockTransport([.respond(.json(200, #"{"latestSummary":{"id":"d1","content":"- 結論から話す"}}"#))])
        let client = makeClient(transport)
        let dashboard = try await client.dashboard()
        #expect(dashboard.latestSummary?.content == "- 結論から話す")
        #expect(dashboard.upcomingInterviews.isEmpty)
        #expect(dashboard.pendingPromptSessions.count == 0)
    }

    @Test("ESプロジェクトのステータス（5 値）と日付を読む")
    func decodesESProject() async throws {
        let transport = MockTransport([.respond(.json(200, #"{"items":[{"id":"p1","companyId":"c1","status":"not_started","deadline":"2026-11-01","createdAt":"2026-10-22T10:00:00.5Z"}],"nextCursor":"n1"}"#))])
        let client = makeClient(transport)
        let page = try await client.esProjects()
        #expect(page.items[0].status == .notStarted)
        #expect(page.items[0].deadline == CalendarDate(year: 2026, month: 11, day: 1))
        #expect(page.nextCursor == "n1")
        #expect(ESStatus.submitted.requiresConfirmation)
        #expect(!ESStatus.draft.requiresConfirmation)
    }
}
