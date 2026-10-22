// DesignDocument/5_FrontendDesign.md 5.8 状態管理とキャッシュ（D16）
// アプリ本体では SwiftData の実装（ios/App の SwiftDataCacheStore）を使う。
// ここではプロトコルと、テスト・プレビューでも使えるファイル実装を置く。
import Foundation

/// キャッシュの対象（5.8.2）。未完了の AI連携と共有リンクの一覧はキャッシュしない。
public enum CacheKey: Hashable, Sendable {
    case dashboard
    case assets
    case assetTags
    case companies
    case researchMaterials(companyId: ID)
    case esProjects
    case esSections(projectId: ID)
    case interviews
    case interviewLogs(interviewId: ID)
    case qa

    public var storageName: String {
        switch self {
        case .dashboard: return "dashboard"
        case .assets: return "assets"
        case .assetTags: return "asset-tags"
        case .companies: return "companies"
        case .researchMaterials(let id): return "research-materials-\(id)"
        case .esProjects: return "es-projects"
        case .esSections(let id): return "es-sections-\(id)"
        case .interviews: return "interviews"
        case .interviewLogs(let id): return "interview-logs-\(id)"
        case .qa: return "qa"
        }
    }
}

public struct CachedValue<Value: Sendable>: Sendable {
    public var value: Value
    /// 最終同期日時（オフライン表示で使う。3.6.4）
    public var savedAt: Date
}

public protocol CacheStore: Sendable {
    func save<Value: Codable & Sendable>(_ value: Value, for key: CacheKey) throws
    func load<Value: Codable & Sendable>(_ type: Value.Type, for key: CacheKey) throws -> CachedValue<Value>?
    /// 5.8.4：ログアウト・退会・別アカウントでのログイン時にすべて消す
    func removeAll() throws
}

/// JSON ファイルによる実装。iOS では Data Protection（completeUnlessOpen）を付ける（5.8.4）
public final class FileCacheStore: CacheStore, @unchecked Sendable {
    private let directory: URL
    private let fileManager: FileManager
    private let now: @Sendable () -> Date
    private let lock = NSLock()

    public init(directory: URL, fileManager: FileManager = .default, now: @escaping @Sendable () -> Date = { Date() }) {
        self.directory = directory
        self.fileManager = fileManager
        self.now = now
    }

    private struct Envelope<Value: Codable>: Codable {
        var savedAt: Date
        var value: Value
    }

    public func save<Value: Codable & Sendable>(_ value: Value, for key: CacheKey) throws {
        try lock.withLock {
            try ensureDirectory()
            let data = try JSONCoding.makeEncoder().encode(Envelope(savedAt: now(), value: value))
            try FileProtection.write(data, to: url(for: key))
        }
    }

    public func load<Value: Codable & Sendable>(_ type: Value.Type, for key: CacheKey) throws -> CachedValue<Value>? {
        try lock.withLock {
            let fileURL = url(for: key)
            guard fileManager.fileExists(atPath: fileURL.path) else { return nil }
            let data = try Data(contentsOf: fileURL)
            let envelope = try JSONCoding.makeDecoder().decode(Envelope<Value>.self, from: data)
            return CachedValue(value: envelope.value, savedAt: envelope.savedAt)
        }
    }

    public func removeAll() throws {
        try lock.withLock {
            guard fileManager.fileExists(atPath: directory.path) else { return }
            try fileManager.removeItem(at: directory)
        }
    }

    private func url(for key: CacheKey) -> URL {
        directory.appendingPathComponent(key.storageName).appendingPathExtension("json")
    }

    private func ensureDirectory() throws {
        if !fileManager.fileExists(atPath: directory.path) {
            try fileManager.createDirectory(at: directory, withIntermediateDirectories: true)
        }
    }
}

enum FileProtection {
    /// 5.8.4：端末のロック中は読めない保護クラス（completeUnlessOpen）で書き込む
    static func write(_ data: Data, to url: URL) throws {
        #if os(iOS)
        try data.write(to: url, options: [.atomic, .completeFileProtectionUnlessOpen])
        #else
        try data.write(to: url, options: [.atomic])
        #endif
    }
}
