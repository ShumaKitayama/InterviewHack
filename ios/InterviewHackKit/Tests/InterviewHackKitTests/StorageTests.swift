// DesignDocument/11_TestDesign.md 11.4（キャッシュ・ログアウト / 退会でのキャッシュ削除・下書き）
import Foundation
import Testing
@testable import InterviewHackKit

@Suite("キャッシュと下書き")
struct StorageTests {
    @Test("キャッシュは保存した値と最終同期日時を返す")
    func cacheRoundTrip() throws {
        let fixed = Date(timeIntervalSince1970: 1_800_000_000)
        let cache = FileCacheStore(directory: temporaryDirectory(), now: { fixed })
        let page = Page(items: [AssetTag(id: "t1", name: "ガクチカ")])
        try cache.save(page, for: .assetTags)
        let loaded = try cache.load(Page<AssetTag>.self, for: .assetTags)
        #expect(loaded?.value.items == page.items)
        #expect(loaded?.savedAt == fixed)
        #expect(try cache.load(Page<AssetTag>.self, for: .assets) == nil)
    }

    @Test("removeAll ですべてのキャッシュが消える")
    func cacheRemoveAll() throws {
        let cache = FileCacheStore(directory: temporaryDirectory())
        try cache.save(Page(items: [AssetTag(id: "t1", name: "x")]), for: .assetTags)
        try cache.save(Page<AssetTag>(items: []), for: .researchMaterials(companyId: "c1"))
        try cache.removeAll()
        #expect(try cache.load(Page<AssetTag>.self, for: .assetTags) == nil)
        #expect(try cache.load(Page<AssetTag>.self, for: .researchMaterials(companyId: "c1")) == nil)
    }

    @Test("下書きの保存・読み込み・削除と、下書き一覧")
    func drafts() throws {
        var clock = Date(timeIntervalSince1970: 1_000)
        let store = FileDraftStore(directory: temporaryDirectory(), now: { clock })
        let log = InterviewLogDraft(entries: [InterviewLogInput(interviewId: "i1", questionText: "自己紹介", answerText: "…")])
        try store.save(log, for: .interviewLog(interviewId: "i1"))
        clock = Date(timeIntervalSince1970: 2_000)
        try store.save(TextDraft(text: "志望動機の下書き"), for: .esSection(sectionId: "sec1"))

        #expect(try store.load(InterviewLogDraft.self, for: .interviewLog(interviewId: "i1")) == log)
        let pending = try store.pendingDrafts()
        #expect(pending.map(\.key) == [.esSection(sectionId: "sec1"), .interviewLog(interviewId: "i1")])

        try store.remove(.esSection(sectionId: "sec1"))
        #expect(try store.load(TextDraft.self, for: .esSection(sectionId: "sec1")) == nil)
        #expect(try store.pendingDrafts().count == 1)
    }

    @Test("CachedResource はキャッシュを先に表示し、取得後に差し替えて保存する")
    @MainActor
    func cachedResource() async throws {
        let cache = FileCacheStore(directory: temporaryDirectory())
        try cache.save(Page(items: [AssetTag(id: "old", name: "古い")]), for: .assetTags)
        let resource = CachedResource<Page<AssetTag>>(key: .assetTags, cache: cache) {
            Page(items: [AssetTag(id: "new", name: "新しい")])
        }
        #expect(resource.value?.items.first?.id == "old")
        #expect(resource.lastSyncedAt != nil)
        await resource.refresh()
        #expect(resource.value?.items.first?.id == "new")
        #expect(try cache.load(Page<AssetTag>.self, for: .assetTags)?.value.items.first?.id == "new")
    }

    @Test("CachedResource は取得に失敗してもキャッシュの表示を残す")
    @MainActor
    func cachedResourceOffline() async throws {
        let cache = FileCacheStore(directory: temporaryDirectory())
        try cache.save(Page(items: [AssetTag(id: "old", name: "古い")]), for: .assetTags)
        let resource = CachedResource<Page<AssetTag>>(key: .assetTags, cache: cache) {
            throw APIError.network("offline")
        }
        await resource.refresh()
        #expect(resource.value?.items.first?.id == "old")
        #expect(resource.error == .network("offline"))
    }
}

@Suite("SessionManager（ログアウト・退会）")
struct SessionManagerTests {
    private func makeStores() -> (FileCacheStore, FileDraftStore) {
        (FileCacheStore(directory: temporaryDirectory()), FileDraftStore(directory: temporaryDirectory()))
    }

    @Test("ログアウトでキャッシュと下書きをすべて消す")
    func signOutClears() async throws {
        let (cache, drafts) = makeStores()
        try cache.save(Page<AssetTag>(items: []), for: .assetTags)
        try drafts.save(TextDraft(text: "x"), for: .esSection(sectionId: "s"))
        let auth = InMemoryAuthService(signedInAs: AuthUser(uid: "u1", email: nil))
        let manager = SessionManager(auth: auth, api: makeClient(MockTransport([]), auth: auth), cache: cache,
                                     drafts: drafts, lastUser: InMemoryLastUserStore("u1"))
        try manager.signOut()
        #expect(auth.currentUser == nil)
        #expect(try cache.load(Page<AssetTag>.self, for: .assetTags) == nil)
        #expect(try drafts.pendingDrafts().isEmpty)
    }

    @Test("別アカウントでログインしたらキャッシュを消してから GET /me を呼ぶ")
    func switchingAccountClears() async throws {
        let (cache, drafts) = makeStores()
        try cache.save(Page<AssetTag>(items: []), for: .assetTags)
        let auth = InMemoryAuthService(signedInAs: AuthUser(uid: "u2", email: nil))
        let transport = MockTransport([.respond(.json(200, #"{"id":"me2"}"#))])
        let lastUser = InMemoryLastUserStore("u1")
        let manager = SessionManager(auth: auth, api: makeClient(transport, auth: auth), cache: cache, drafts: drafts, lastUser: lastUser)
        let me = try await manager.didSignIn(AuthUser(uid: "u2", email: nil))
        #expect(me.id == "me2")
        #expect(try cache.load(Page<AssetTag>.self, for: .assetTags) == nil)
        #expect(lastUser.lastUserId() == "u2")
        #expect(transport.recorded[0].url?.path == "/me")
    }

    @Test("同じアカウントで再ログインしてもキャッシュは残す")
    func sameAccountKeepsCache() async throws {
        let (cache, drafts) = makeStores()
        try cache.save(Page<AssetTag>(items: []), for: .assetTags)
        let auth = InMemoryAuthService(signedInAs: AuthUser(uid: "u1", email: nil))
        let manager = SessionManager(auth: auth, api: makeClient(MockTransport([.respond(.json(200, #"{"id":"me1"}"#))]), auth: auth),
                                     cache: cache, drafts: drafts, lastUser: InMemoryLastUserStore("u1"))
        try await manager.didSignIn(AuthUser(uid: "u1", email: nil))
        #expect(try cache.load(Page<AssetTag>.self, for: .assetTags) != nil)
    }

    @Test("退会：DELETE /me → Firebase のユーザー削除 → 端末のデータを消す")
    func deleteAccount() async throws {
        let (cache, drafts) = makeStores()
        try cache.save(Page<AssetTag>(items: []), for: .assetTags)
        try drafts.save(TextDraft(text: "x"), for: .aiResponse(sessionId: "s1"))
        let auth = InMemoryAuthService()
        _ = try await auth.signUp(email: "a@example.com", password: "secret1")
        let transport = MockTransport([.respond(.noContent)])
        let manager = SessionManager(auth: auth, api: makeClient(transport, auth: auth), cache: cache, drafts: drafts,
                                     lastUser: InMemoryLastUserStore("uid-1"))
        let outcome = try await manager.deleteAccount()
        #expect(outcome == .completed)
        #expect(transport.recorded[0].httpMethod == "DELETE")
        #expect(transport.recorded[0].url?.path == "/me")
        #expect(auth.currentUser == nil)
        await #expect(throws: AuthServiceError.invalidCredentials) { _ = try await auth.signIn(email: "a@example.com", password: "secret1") }
        #expect(try cache.load(Page<AssetTag>.self, for: .assetTags) == nil)
        #expect(try drafts.pendingDrafts().isEmpty)
    }

    @Test("DELETE /me が失敗したら端末のデータは消さない")
    func deleteAccountFailure() async throws {
        let (cache, drafts) = makeStores()
        try cache.save(Page<AssetTag>(items: []), for: .assetTags)
        let auth = InMemoryAuthService(signedInAs: AuthUser(uid: "u1", email: nil))
        let transport = MockTransport([.respond(.json(500, #"{"status":"error","code":"INTERNAL_ERROR","message":"失敗"}"#))])
        let manager = SessionManager(auth: auth, api: makeClient(transport, auth: auth), cache: cache, drafts: drafts,
                                     lastUser: InMemoryLastUserStore("u1"))
        await #expect(throws: APIError.self) { _ = try await manager.deleteAccount() }
        #expect(auth.currentUser != nil)
        #expect(try cache.load(Page<AssetTag>.self, for: .assetTags) != nil)
    }
}
