// DesignDocument/5_FrontendDesign.md 5.8.3 下書きの端末保存
// 入力のたびに保存し、送信に成功したら消す。オンライン復帰後も自動送信はしない（D16 / 2.10.7）。
import Foundation

public enum DraftKey: Hashable, Sendable, Codable {
    /// 入力中の面接ログ（面接ごと）
    case interviewLog(interviewId: ID)
    /// 編集中の ES セクション本文
    case esSection(sectionId: ID)
    /// 貼り戻した直後の回答テキスト（保存前）
    case aiResponse(sessionId: ID)

    public var storageName: String {
        switch self {
        case .interviewLog(let id): return "interview-log-\(id)"
        case .esSection(let id): return "es-section-\(id)"
        case .aiResponse(let id): return "ai-response-\(id)"
        }
    }
}

/// 面接ログの下書き（複数件をまとめて入力できる。3.5.7）
public struct InterviewLogDraft: Codable, Sendable, Equatable {
    public var entries: [InterviewLogInput]
    public init(entries: [InterviewLogInput]) { self.entries = entries }
}

public struct TextDraft: Codable, Sendable, Equatable {
    public var text: String
    public init(text: String) { self.text = text }
}

public struct DraftRecord: Sendable, Equatable {
    public var key: DraftKey
    public var updatedAt: Date
}

public protocol DraftStore: Sendable {
    func save<Value: Codable & Sendable>(_ value: Value, for key: DraftKey) throws
    func load<Value: Codable & Sendable>(_ type: Value.Type, for key: DraftKey) throws -> Value?
    func remove(_ key: DraftKey) throws
    /// 下書きがあることを画面に表示するために使う（自動送信はしない）
    func pendingDrafts() throws -> [DraftRecord]
    func removeAll() throws
}

public final class FileDraftStore: DraftStore, @unchecked Sendable {
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
        var key: DraftKey
        var updatedAt: Date
        var value: Value
    }

    private struct Header: Codable {
        var key: DraftKey
        var updatedAt: Date
    }

    public func save<Value: Codable & Sendable>(_ value: Value, for key: DraftKey) throws {
        try lock.withLock {
            if !fileManager.fileExists(atPath: directory.path) {
                try fileManager.createDirectory(at: directory, withIntermediateDirectories: true)
            }
            let data = try JSONCoding.makeEncoder().encode(Envelope(key: key, updatedAt: now(), value: value))
            try FileProtection.write(data, to: url(for: key))
        }
    }

    public func load<Value: Codable & Sendable>(_ type: Value.Type, for key: DraftKey) throws -> Value? {
        try lock.withLock {
            let fileURL = url(for: key)
            guard fileManager.fileExists(atPath: fileURL.path) else { return nil }
            return try JSONCoding.makeDecoder().decode(Envelope<Value>.self, from: Data(contentsOf: fileURL)).value
        }
    }

    public func remove(_ key: DraftKey) throws {
        try lock.withLock {
            let fileURL = url(for: key)
            if fileManager.fileExists(atPath: fileURL.path) {
                try fileManager.removeItem(at: fileURL)
            }
        }
    }

    public func pendingDrafts() throws -> [DraftRecord] {
        try lock.withLock {
            guard fileManager.fileExists(atPath: directory.path) else { return [] }
            let files = try fileManager.contentsOfDirectory(at: directory, includingPropertiesForKeys: nil)
            return try files.filter { $0.pathExtension == "json" }.map { fileURL in
                let header = try JSONCoding.makeDecoder().decode(Header.self, from: Data(contentsOf: fileURL))
                return DraftRecord(key: header.key, updatedAt: header.updatedAt)
            }.sorted { $0.updatedAt > $1.updatedAt }
        }
    }

    public func removeAll() throws {
        try lock.withLock {
            guard fileManager.fileExists(atPath: directory.path) else { return }
            try fileManager.removeItem(at: directory)
        }
    }

    private func url(for key: DraftKey) -> URL {
        directory.appendingPathComponent(key.storageName).appendingPathExtension("json")
    }
}
