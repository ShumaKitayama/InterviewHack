// DesignDocument/7_BackendDesign.md 7.5 API一覧（エンドポイント設計）
import Foundation

public struct ListQuery: Sendable, Equatable {
    public var limit: Int?
    public var cursor: String?
    public var filters: [String: String]

    public init(limit: Int? = 50, cursor: String? = nil, filters: [String: String] = [:]) {
        self.limit = limit
        self.cursor = cursor
        self.filters = filters
    }

    var items: [URLQueryItem] {
        var result: [URLQueryItem] = []
        if let limit { result.append(URLQueryItem(name: "limit", value: String(limit))) }
        if let cursor { result.append(URLQueryItem(name: "cursor", value: cursor)) }
        for key in filters.keys.sorted() {
            result.append(URLQueryItem(name: key, value: filters[key]))
        }
        return result
    }
}

public extension APIClient {
    // MARK: 7.5.1 Health

    struct HealthStatus: Codable, Sendable, Equatable { public var status: String }

    func health() async throws -> HealthStatus {
        try await send(Endpoint(method: .get, path: "health", requiresAuth: false))
    }

    // MARK: 7.5.2 Account

    func me() async throws -> Me {
        try await send(Endpoint(method: .get, path: "me"))
    }

    func updateMe(_ input: MeUpdate) async throws -> Me {
        try await send(.json(.patch, "me", body: input))
    }

    /// 退会（6.3.3）。成功後に Firebase のユーザー削除をアプリが行う
    func deleteMe() async throws {
        let _: EmptyResponse = try await send(Endpoint(method: .delete, path: "me"))
    }

    // MARK: 7.5.3 CRUD（共通）

    private func list<T: Codable & Sendable>(_ path: String, _ query: ListQuery) async throws -> Page<T> {
        try await send(Endpoint(method: .get, path: path, query: query.items))
    }

    private func create<T: Codable & Sendable, Body: Encodable>(_ path: String, _ body: Body) async throws -> T {
        try await send(.json(.post, path, body: body))
    }

    private func update<T: Codable & Sendable, Body: Encodable>(_ path: String, _ body: Body) async throws -> T {
        try await send(.json(.patch, path, body: body))
    }

    private func remove(_ path: String) async throws {
        let _: EmptyResponse = try await send(Endpoint(method: .delete, path: path))
    }

    // 自己資産（type / tagId / q / sort）
    func assets(_ query: ListQuery = ListQuery()) async throws -> Page<Asset> { try await list("assets", query) }
    func asset(_ id: ID) async throws -> Asset { try await send(Endpoint(method: .get, path: "assets/\(id)")) }
    func createAsset(_ input: AssetInput) async throws -> Asset { try await create("assets", input) }
    func updateAsset(_ id: ID, _ input: AssetInput) async throws -> Asset { try await update("assets/\(id)", input) }
    func deleteAsset(_ id: ID) async throws { try await remove("assets/\(id)") }

    // タグ
    func assetTags(_ query: ListQuery = ListQuery()) async throws -> Page<AssetTag> { try await list("asset-tags", query) }
    func createAssetTag(_ input: AssetTagInput) async throws -> AssetTag { try await create("asset-tags", input) }
    func updateAssetTag(_ id: ID, _ input: AssetTagInput) async throws -> AssetTag { try await update("asset-tags/\(id)", input) }
    func deleteAssetTag(_ id: ID) async throws { try await remove("asset-tags/\(id)") }

    // 企業（q / selectionPhase）
    func companies(_ query: ListQuery = ListQuery()) async throws -> Page<Company> { try await list("companies", query) }
    func company(_ id: ID) async throws -> Company { try await send(Endpoint(method: .get, path: "companies/\(id)")) }
    func createCompany(_ input: CompanyInput) async throws -> Company { try await create("companies", input) }
    func updateCompany(_ id: ID, _ input: CompanyInput) async throws -> Company { try await update("companies/\(id)", input) }
    func deleteCompany(_ id: ID) async throws { try await remove("companies/\(id)") }

    // 企業研究資料（companyId 必須）
    func researchMaterials(companyId: ID, _ query: ListQuery = ListQuery()) async throws -> Page<ResearchMaterial> {
        var query = query
        query.filters["companyId"] = companyId
        return try await list("research-materials", query)
    }
    func researchMaterial(_ id: ID) async throws -> ResearchMaterial {
        try await send(Endpoint(method: .get, path: "research-materials/\(id)"))
    }
    func createResearchMaterial(_ input: ResearchMaterialInput) async throws -> ResearchMaterial {
        try await create("research-materials", input)
    }
    func updateResearchMaterial(_ id: ID, _ input: ResearchMaterialInput) async throws -> ResearchMaterial {
        try await update("research-materials/\(id)", input)
    }
    func deleteResearchMaterial(_ id: ID) async throws { try await remove("research-materials/\(id)") }

    // ESプロジェクト（companyId / status）
    func esProjects(_ query: ListQuery = ListQuery()) async throws -> Page<ESProject> { try await list("es-projects", query) }
    func esProject(_ id: ID) async throws -> ESProject { try await send(Endpoint(method: .get, path: "es-projects/\(id)")) }
    func createESProject(_ input: ESProjectInput) async throws -> ESProject { try await create("es-projects", input) }
    func updateESProject(_ id: ID, _ input: ESProjectInput) async throws -> ESProject { try await update("es-projects/\(id)", input) }
    func deleteESProject(_ id: ID) async throws { try await remove("es-projects/\(id)") }

    // ESセクション
    func esSections(projectId: ID, _ query: ListQuery = ListQuery()) async throws -> Page<ESSection> {
        try await list("es-projects/\(projectId)/sections", query)
    }
    func esSection(_ id: ID) async throws -> ESSection { try await send(Endpoint(method: .get, path: "es-sections/\(id)")) }
    func createESSection(projectId: ID, _ input: ESSectionInput) async throws -> ESSection {
        try await create("es-projects/\(projectId)/sections", input)
    }
    func updateESSection(_ id: ID, _ input: ESSectionInput) async throws -> ESSection { try await update("es-sections/\(id)", input) }
    func deleteESSection(_ id: ID) async throws { try await remove("es-sections/\(id)") }

    // ESセクションの版（2.2.6.1）
    func esSectionVersions(sectionId: ID) async throws -> [ESSectionVersion] {
        let response: ItemsResponse<ESSectionVersion> = try await send(Endpoint(method: .get, path: "es-sections/\(sectionId)/versions"))
        return response.items
    }
    func saveESSectionVersion(sectionId: ID) async throws -> ESSectionVersion {
        try await send(Endpoint(method: .post, path: "es-sections/\(sectionId)/versions"))
    }
    func restoreESSectionVersion(sectionId: ID, versionId: ID) async throws -> ESSection {
        try await send(Endpoint(method: .post, path: "es-sections/\(sectionId)/versions/\(versionId)/restore"))
    }

    // 面接（companyId / 期間）
    func interviews(_ query: ListQuery = ListQuery()) async throws -> Page<Interview> { try await list("interviews", query) }
    func interview(_ id: ID) async throws -> Interview { try await send(Endpoint(method: .get, path: "interviews/\(id)")) }
    func createInterview(_ input: InterviewInput) async throws -> Interview { try await create("interviews", input) }
    func updateInterview(_ id: ID, _ input: InterviewInput) async throws -> Interview { try await update("interviews/\(id)", input) }
    func deleteInterview(_ id: ID) async throws { try await remove("interviews/\(id)") }

    // 面接ログ（interviewId）
    func interviewLogs(interviewId: ID, _ query: ListQuery = ListQuery()) async throws -> Page<InterviewLog> {
        var query = query
        query.filters["interviewId"] = interviewId
        return try await list("interview-logs", query)
    }
    func interviewLog(_ id: ID) async throws -> InterviewLog { try await send(Endpoint(method: .get, path: "interview-logs/\(id)")) }
    func createInterviewLog(_ input: InterviewLogInput) async throws -> InterviewLog { try await create("interview-logs", input) }
    func updateInterviewLog(_ id: ID, _ input: InterviewLogInput) async throws -> InterviewLog {
        try await update("interview-logs/\(id)", input)
    }
    func deleteInterviewLog(_ id: ID) async throws { try await remove("interview-logs/\(id)") }

    // Q&A（companyId / q / isFavorite / reviewStatus）
    func qaList(_ query: ListQuery = ListQuery()) async throws -> Page<QA> { try await list("qa", query) }
    func qa(_ id: ID) async throws -> QA { try await send(Endpoint(method: .get, path: "qa/\(id)")) }
    func createQA(_ input: QAInput) async throws -> QA { try await create("qa", input) }
    func updateQA(_ id: ID, _ input: QAInput) async throws -> QA { try await update("qa/\(id)", input) }
    func deleteQA(_ id: ID) async throws { try await remove("qa/\(id)") }

    // MARK: 7.5.4 Dashboard

    func dashboard() async throws -> Dashboard {
        try await send(Endpoint(method: .get, path: "dashboard"))
    }

    // MARK: 7.5.5 AI連携

    func createPromptSession(_ request: PromptSessionRequest) async throws -> PromptText {
        try await send(.json(.post, "prompt-sessions", body: request))
    }

    /// 未完了の AI連携（既定は created と pasted）
    func promptSessions(statuses: [PromptSessionStatus] = [.created, .pasted], cursor: String? = nil,
                        limit: Int = 50) async throws -> Page<PromptSessionSummary> {
        var items = [URLQueryItem(name: "status", value: statuses.map(\.rawValue).joined(separator: ","))]
        items.append(URLQueryItem(name: "limit", value: String(limit)))
        if let cursor { items.append(URLQueryItem(name: "cursor", value: cursor)) }
        return try await send(Endpoint(method: .get, path: "prompt-sessions", query: items))
    }

    func promptText(sessionId: ID) async throws -> PromptText {
        try await send(Endpoint(method: .get, path: "prompt-sessions/\(sessionId)/prompt"))
    }

    func parsePromptResponse(sessionId: ID, responseText: String) async throws -> ParseResponse {
        try await send(.json(.post, "prompt-sessions/\(sessionId)/parse", body: ParseRequest(responseText: responseText)))
    }

    /// 7.6.1：commit は条件付き更新で二重保存しないため、再送してよい。
    /// 409 `SESSION_STATE_CONFLICT` を受けたら状態を確認し、`saved` なら保存済みとして扱う。
    func commitPromptSession(sessionId: ID, payload: CommitPayload) async throws -> CommitResponse {
        do {
            return try await send(.json(.post, "prompt-sessions/\(sessionId)/commit", body: payload, retryableWrite: true))
        } catch let error as APIError where error.code == .sessionStateConflict {
            if try await sessionIsSaved(sessionId) {
                return CommitResponse(sessionId: sessionId, status: .saved)
            }
            throw error
        }
    }

    func abandonPromptSession(sessionId: ID) async throws -> SessionStatusResponse {
        try await send(Endpoint(method: .post, path: "prompt-sessions/\(sessionId)/abandon"))
    }

    /// 7.6.1：`GET /prompt-sessions/{id}` でセッションを確認する
    func promptSession(sessionId: ID) async throws -> PromptSessionSummary {
        try await send(Endpoint(method: .get, path: "prompt-sessions/\(sessionId)"))
    }

    private func sessionIsSaved(_ sessionId: ID) async throws -> Bool {
        try await promptSession(sessionId: sessionId).status == .saved
    }

    // MARK: 7.5.6 共有リンク

    func createShareLink(projectId: ID, _ request: ShareLinkRequest) async throws -> CreatedShareLink {
        try await send(.json(.post, "es-projects/\(projectId)/share-links", body: request))
    }

    func shareLinks(projectId: ID) async throws -> [ESShareLink] {
        let response: ItemsResponse<ESShareLink> = try await send(Endpoint(method: .get, path: "es-projects/\(projectId)/share-links"))
        return response.items
    }

    func revokeShareLink(_ id: ID) async throws {
        try await remove("share-links/\(id)")
    }
}
