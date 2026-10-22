// DesignDocument/5_FrontendDesign.md 5.3 App（ルートの画面切り替え、DI）/ 5.5.2 ログイン状態の変化
import Foundation
import InterviewHackKit
import Observation

@MainActor
@Observable
public final class AppModel {
    public let api: APIClient
    public let auth: AuthService
    public let cache: CacheStore
    public let drafts: DraftStore
    public let session: SessionManager
    public let connectivity: ConnectivityMonitor

    public private(set) var currentUser: AuthUser?
    public private(set) var me: Me?
    public private(set) var isRestoring = true
    public var signInError: Error?

    public init(api: APIClient, auth: AuthService, cache: CacheStore, drafts: DraftStore,
                lastUser: LastUserStore, connectivity: ConnectivityMonitor) {
        self.api = api
        self.auth = auth
        self.cache = cache
        self.drafts = drafts
        self.connectivity = connectivity
        session = SessionManager(auth: auth, api: api, cache: cache, drafts: drafts, lastUser: lastUser)
    }

    /// `addStateDidChangeListener` 相当の変化を受けてルート画面を切り替える
    public func observeAuthState() async {
        for await user in auth.authStateChanges() {
            currentUser = user
            if let user {
                do {
                    me = try await session.didSignIn(user)
                    signInError = nil
                } catch {
                    // オフラインでもキャッシュで閲覧できるようにする（D16）
                    signInError = error
                }
            } else {
                me = nil
            }
            isRestoring = false
        }
    }

    public func signOut() {
        try? session.signOut()
    }

    public func updateDisplayName(_ name: String) async throws {
        me = try await api.updateMe(MeUpdate(displayName: name))
    }

    public func makeFlow(_ useCase: UseCase) -> AILinkFlow {
        AILinkFlow(useCase: useCase, api: api, drafts: drafts)
    }

    public func resource<Value: Codable & Sendable>(_ key: CacheKey?, fetch: @escaping @Sendable () async throws -> Value) -> CachedResource<Value> {
        CachedResource(key: key, cache: cache, fetch: fetch)
    }

    /// プレビュー用（ネットワークに出ない）
    public static func preview() -> AppModel {
        let auth = InMemoryAuthService(signedInAs: AuthUser(uid: "preview", email: "preview@example.com"))
        let base = FileManager.default.temporaryDirectory.appendingPathComponent("interviewhack-preview")
        return AppModel(api: APIClient(baseURL: URL(string: "http://localhost:8080")!, tokenProvider: auth),
                        auth: auth,
                        cache: FileCacheStore(directory: base.appendingPathComponent("cache")),
                        drafts: FileDraftStore(directory: base.appendingPathComponent("drafts")),
                        lastUser: InMemoryLastUserStore(), connectivity: ConnectivityMonitor(start: false))
    }
}
