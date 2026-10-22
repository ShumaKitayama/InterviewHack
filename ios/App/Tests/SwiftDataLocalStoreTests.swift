// DesignDocument/11_TestDesign.md 11.4：SwiftData のキャッシュと下書き（ログアウト時の削除）
// Xcode（iOS シミュレータ）で実行する。ロジック部分のテストは InterviewHackKit の swift test にある。
@testable import InterviewHack
import InterviewHackKit
import XCTest

final class SwiftDataLocalStoreTests: XCTestCase {
    private func makeStore() throws -> SwiftDataLocalStore {
        let directory = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        return try SwiftDataLocalStore(directory: directory)
    }

    func testCacheRoundTripAndRemoveAll() throws {
        let store = try makeStore()
        // AssetTag は BFF のレスポンスから作るモデルなので、テストでも JSON からデコードする
        let tags = try JSONDecoder().decode([AssetTag].self, from: Data(#"[{"id":"t1","name":"ガクチカ"}]"#.utf8))
        try store.save(Page(items: tags), for: .assetTags)
        XCTAssertEqual(try store.load(Page<AssetTag>.self, for: .assetTags)?.value.items.first?.name, "ガクチカ")

        try store.save(TextDraft(text: "下書き"), for: .esSection(sectionId: "s1"))
        XCTAssertEqual(try store.pendingDrafts().map(\.key), [.esSection(sectionId: "s1")])

        try store.removeAll()
        XCTAssertNil(try store.load(Page<AssetTag>.self, for: .assetTags))
        XCTAssertTrue(try store.pendingDrafts().isEmpty)
    }

    func testDraftOverwriteAndRemove() throws {
        let store = try makeStore()
        try store.save(TextDraft(text: "1"), for: .aiResponse(sessionId: "x"))
        try store.save(TextDraft(text: "2"), for: .aiResponse(sessionId: "x"))
        XCTAssertEqual(try store.load(TextDraft.self, for: .aiResponse(sessionId: "x"))?.text, "2")
        try store.remove(.aiResponse(sessionId: "x"))
        XCTAssertNil(try store.load(TextDraft.self, for: .aiResponse(sessionId: "x")))
    }
}
