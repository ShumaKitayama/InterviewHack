// DesignDocument/5_FrontendDesign.md 5.5.4 ログアウト・退会 / 5.8.4 キャッシュの保護と消去
import Foundation

/// 端末に残す「最後にログインしたユーザー」。別アカウントでログインしたらキャッシュを消すために使う
public protocol LastUserStore: Sendable {
    func lastUserId() -> String?
    func setLastUserId(_ id: String?)
}

public final class UserDefaultsLastUserStore: LastUserStore, @unchecked Sendable {
    private let defaults: UserDefaults
    private let key = "interviewhack.lastUserUid"

    public init(defaults: UserDefaults = .standard) {
        self.defaults = defaults
    }

    public func lastUserId() -> String? { defaults.string(forKey: key) }
    public func setLastUserId(_ id: String?) { defaults.set(id, forKey: key) }
}

public final class InMemoryLastUserStore: LastUserStore, @unchecked Sendable {
    private let lock = NSLock()
    private var value: String?
    public init(_ value: String? = nil) { self.value = value }
    public func lastUserId() -> String? { lock.withLock { value } }
    public func setLastUserId(_ id: String?) { lock.withLock { value = id } }
}

public final class SessionManager: Sendable {
    private let auth: AuthService
    private let api: APIClient
    private let cache: CacheStore
    private let drafts: DraftStore
    private let lastUser: LastUserStore

    public init(auth: AuthService, api: APIClient, cache: CacheStore, drafts: DraftStore, lastUser: LastUserStore) {
        self.auth = auth
        self.api = api
        self.cache = cache
        self.drafts = drafts
        self.lastUser = lastUser
    }

    public enum DeletionOutcome: Equatable, Sendable {
        case completed
        /// BFF 側の退会は完了したが、Firebase のユーザー削除に再ログインが必要（以後の API は 401 になる）
        case firebaseDeletionNeedsRecentLogin
    }

    /// ログイン成功後：別アカウントならキャッシュを消してから `GET /me` を呼ぶ（BFF が users を作成する。5.5.1）
    @discardableResult
    public func didSignIn(_ user: AuthUser) async throws -> Me {
        if let previous = lastUser.lastUserId(), previous != user.uid {
            try clearLocalData()
        }
        lastUser.setLastUserId(user.uid)
        return try await api.me()
    }

    public func signOut() throws {
        try auth.signOut()
        try clearLocalData()
        lastUser.setLastUserId(nil)
    }

    /// 6.3.3：`DELETE /me` → Firebase のユーザー削除 → ログアウトと同じ消去
    public func deleteAccount() async throws -> DeletionOutcome {
        try await api.deleteMe()
        var outcome = DeletionOutcome.completed
        do {
            try await auth.deleteCurrentUser()
        } catch AuthServiceError.requiresRecentLogin {
            outcome = .firebaseDeletionNeedsRecentLogin
        } catch {
            // BFF 側は退会済み（users.deleted_at 設定済み）なので、以後の API は 401 になる。端末の消去は続ける
            outcome = .firebaseDeletionNeedsRecentLogin
        }
        try? auth.signOut()
        try clearLocalData()
        lastUser.setLastUserId(nil)
        return outcome
    }

    public func clearLocalData() throws {
        try cache.removeAll()
        try drafts.removeAll()
    }
}
