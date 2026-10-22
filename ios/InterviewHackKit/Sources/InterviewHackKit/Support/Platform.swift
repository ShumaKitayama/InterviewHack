// DesignDocument/5_FrontendDesign.md 5.11 外部AIへの受け渡し / 5.14 エラー収集 / 3.6.4 オフライン
import Foundation
import Network
import Observation
import os
#if canImport(UIKit)
import UIKit
#elseif canImport(AppKit)
import AppKit
#endif

/// クリップボード。貼り付けはユーザーがボタンを押したときだけ読む（5.11）
public enum Clipboard {
    public static func copy(_ text: String) {
        #if canImport(UIKit)
        UIPasteboard.general.string = text
        #elseif canImport(AppKit)
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(text, forType: .string)
        #endif
    }

    public static func paste() -> String? {
        #if canImport(UIKit)
        return UIPasteboard.general.string
        #elseif canImport(AppKit)
        return NSPasteboard.general.string(forType: .string)
        #else
        return nil
        #endif
    }

    /// クリップボードに文字列があるか（内容は読まない。3.4.0 の「貼り付け」ボタンの強調に使う）
    public static var hasText: Bool {
        #if canImport(UIKit)
        return UIPasteboard.general.hasStrings
        #elseif canImport(AppKit)
        return NSPasteboard.general.availableType(from: [.string]) != nil
        #else
        return false
        #endif
    }
}

/// オンライン / オフラインの監視（3.6.4 のバナー、ボタンの無効化に使う）
@MainActor
@Observable
public final class ConnectivityMonitor {
    public private(set) var isOnline = true
    private let monitor = NWPathMonitor()

    public init(start: Bool = true) {
        if start { self.start() }
    }

    public func start() {
        monitor.pathUpdateHandler = { [weak self] path in
            let online = path.status == .satisfied
            Task { @MainActor in self?.isOnline = online }
        }
        monitor.start(queue: DispatchQueue(label: "interviewhack.connectivity"))
    }

    /// テスト・プレビュー用
    public func setOnline(_ online: Bool) {
        isOnline = online
    }
}

/// os.Logger の共通設定（subsystem はバンドル ID、category は機能名。本文は出さない。5.14）
public enum AppLog {
    public static let subsystem = Bundle.main.bundleIdentifier ?? "jp.interviewhack.app"

    public static func logger(_ category: String) -> Logger {
        Logger(subsystem: subsystem, category: category)
    }
}
