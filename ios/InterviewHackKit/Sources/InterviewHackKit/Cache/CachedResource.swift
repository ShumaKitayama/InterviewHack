// DesignDocument/5_FrontendDesign.md 5.8.1：キャッシュを先に表示 → API で最新を取得 → 差し替え
import Foundation
import Observation

@MainActor
@Observable
public final class CachedResource<Value: Codable & Sendable> {
    public private(set) var value: Value?
    /// 最終同期日時（オフライン時に表示する。3.6.4）
    public private(set) var lastSyncedAt: Date?
    public private(set) var isRefreshing = false
    public private(set) var error: APIError?

    private let key: CacheKey?
    private let cache: CacheStore
    private let fetch: @Sendable () async throws -> Value

    /// `key` が nil の場合はキャッシュしない（未完了の AI連携・共有リンクの一覧。5.8.2）
    public init(key: CacheKey?, cache: CacheStore, fetch: @escaping @Sendable () async throws -> Value) {
        self.key = key
        self.cache = cache
        self.fetch = fetch
        if let key, let cached = try? cache.load(Value.self, for: key) {
            value = cached.value
            lastSyncedAt = cached.savedAt
        }
    }

    /// サーバーの内容で差し替える（正はサーバー側）。失敗してもキャッシュの表示は残す
    public func refresh() async {
        isRefreshing = true
        defer { isRefreshing = false }
        do {
            let fresh = try await fetch()
            value = fresh
            lastSyncedAt = Date()
            error = nil
            if let key {
                try? cache.save(fresh, for: key)
            }
        } catch let apiError as APIError {
            error = apiError
        } catch {
            self.error = .network(String(describing: error))
        }
    }

    /// 書き込み成功後など、手元で値を更新したいとき
    public func replace(_ newValue: Value) {
        value = newValue
        if let key {
            try? cache.save(newValue, for: key)
        }
    }
}
