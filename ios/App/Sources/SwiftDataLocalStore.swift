// DesignDocument/5_FrontendDesign.md 5.8 状態管理とキャッシュ（SwiftData）/ 5.8.4 キャッシュの保護と消去
// 読み取りキャッシュと下書きを SwiftData に保存する。値は JSON（Codable）で持つ。
import Foundation
import InterviewHackKit
import SwiftData

@Model
final class CacheEntry {
    @Attribute(.unique) var key: String
    var payload: Data
    var savedAt: Date

    init(key: String, payload: Data, savedAt: Date) {
        self.key = key
        self.payload = payload
        self.savedAt = savedAt
    }
}

@Model
final class DraftEntry {
    @Attribute(.unique) var key: String
    /// `DraftKey` を JSON にしたもの（下書き一覧の表示に使う）
    var keyPayload: Data
    var payload: Data
    var updatedAt: Date

    init(key: String, keyPayload: Data, payload: Data, updatedAt: Date) {
        self.key = key
        self.keyPayload = keyPayload
        self.payload = payload
        self.updatedAt = updatedAt
    }
}

final class SwiftDataLocalStore: CacheStore, DraftStore, @unchecked Sendable {
    private let container: ModelContainer
    private let lock = NSLock()

    init(directory: URL) throws {
        let fileManager = FileManager.default
        try fileManager.createDirectory(at: directory, withIntermediateDirectories: true)
        // 5.8.4：端末のロック中は読めない保護クラス（completeUnlessOpen）をストアのディレクトリに設定する
        try fileManager.setAttributes([.protectionKey: FileProtectionType.completeUnlessOpen], ofItemAtPath: directory.path)
        let configuration = ModelConfiguration(url: directory.appendingPathComponent("local.store"))
        container = try ModelContainer(for: CacheEntry.self, DraftEntry.self, configurations: configuration)
    }

    private func withContext<T>(_ body: (ModelContext) throws -> T) throws -> T {
        try lock.withLock {
            let context = ModelContext(container)
            let result = try body(context)
            if context.hasChanges { try context.save() }
            return result
        }
    }

    // MARK: CacheStore

    func save<Value: Codable & Sendable>(_ value: Value, for key: CacheKey) throws {
        let payload = try JSONCoding.makeEncoder().encode(value)
        let name = key.storageName
        try withContext { context in
            let existing = try context.fetch(FetchDescriptor<CacheEntry>(predicate: #Predicate { $0.key == name })).first
            if let existing {
                existing.payload = payload
                existing.savedAt = Date()
            } else {
                context.insert(CacheEntry(key: name, payload: payload, savedAt: Date()))
            }
        }
    }

    func load<Value: Codable & Sendable>(_ type: Value.Type, for key: CacheKey) throws -> CachedValue<Value>? {
        let name = key.storageName
        let entry: (Data, Date)? = try withContext { context in
            try context.fetch(FetchDescriptor<CacheEntry>(predicate: #Predicate { $0.key == name })).first.map { ($0.payload, $0.savedAt) }
        }
        guard let (payload, savedAt) = entry else { return nil }
        return CachedValue(value: try JSONCoding.makeDecoder().decode(Value.self, from: payload), savedAt: savedAt)
    }

    /// ログアウト・退会・別アカウントでのログイン時に、キャッシュと下書きをすべて消す
    func removeAll() throws {
        try withContext { context in
            try context.delete(model: CacheEntry.self)
            try context.delete(model: DraftEntry.self)
        }
    }

    // MARK: DraftStore

    func save<Value: Codable & Sendable>(_ value: Value, for key: DraftKey) throws {
        let payload = try JSONCoding.makeEncoder().encode(value)
        let keyPayload = try JSONCoding.makeEncoder().encode(key)
        let name = key.storageName
        try withContext { context in
            let existing = try context.fetch(FetchDescriptor<DraftEntry>(predicate: #Predicate { $0.key == name })).first
            if let existing {
                existing.payload = payload
                existing.updatedAt = Date()
            } else {
                context.insert(DraftEntry(key: name, keyPayload: keyPayload, payload: payload, updatedAt: Date()))
            }
        }
    }

    func load<Value: Codable & Sendable>(_ type: Value.Type, for key: DraftKey) throws -> Value? {
        let name = key.storageName
        let payload: Data? = try withContext { context in
            try context.fetch(FetchDescriptor<DraftEntry>(predicate: #Predicate { $0.key == name })).first?.payload
        }
        return try payload.map { try JSONCoding.makeDecoder().decode(Value.self, from: $0) }
    }

    func remove(_ key: DraftKey) throws {
        let name = key.storageName
        try withContext { context in
            try context.delete(model: DraftEntry.self, where: #Predicate { $0.key == name })
        }
    }

    func pendingDrafts() throws -> [DraftRecord] {
        try withContext { context in
            let entries = try context.fetch(FetchDescriptor<DraftEntry>(sortBy: [SortDescriptor(\.updatedAt, order: .reverse)]))
            return try entries.map { entry in
                DraftRecord(key: try JSONCoding.makeDecoder().decode(DraftKey.self, from: entry.keyPayload), updatedAt: entry.updatedAt)
            }
        }
    }
}
