// DesignDocument/3_UIUXDesign.md 3.6 画面状態設計（Empty / Loading / Error / Offline）/ 3.9 UI規約
import InterviewHackKit
import SwiftUI

/// エラー表示。`request_id` を小さく表示する（5.7.1）
struct ErrorBanner: View {
    let error: Error
    var retry: (() -> Void)?

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Label(message, systemImage: "exclamationmark.triangle.fill")
                .foregroundStyle(.red)
                .font(.callout)
            if let requestId = (error as? APIError)?.requestId {
                Text("request_id: \(requestId)")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .textSelection(.enabled)
            }
            if let retry {
                Button("再試行", action: retry)
                    .font(.callout)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(12)
        .background(Color.red.opacity(0.08), in: RoundedRectangle(cornerRadius: 10))
        .accessibilityElement(children: .combine)
    }

    private var message: String {
        (error as? LocalizedError)?.errorDescription ?? error.localizedDescription
    }
}

/// 3.6.4：オフライン時のバナー
struct OfflineBanner: View {
    var lastSyncedAt: Date?

    var body: some View {
        HStack(spacing: 6) {
            Image(systemName: "wifi.slash")
            Text("オフライン：閲覧と下書きのみ")
            if let lastSyncedAt {
                Text("（最終同期 \(lastSyncedAt.formatted(date: .abbreviated, time: .shortened))）")
                    .foregroundStyle(.secondary)
            }
        }
        .font(.footnote)
        .frame(maxWidth: .infinity)
        .padding(.vertical, 6)
        .background(Color.orange.opacity(0.15))
        .accessibilityElement(children: .combine)
    }
}

struct EmptyStateView: View {
    let title: String
    let message: String
    var actionTitle: String?
    var action: (() -> Void)?

    var body: some View {
        VStack(spacing: 12) {
            Text(title).font(.headline)
            Text(message)
                .font(.callout)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
            if let actionTitle, let action {
                Button(actionTitle, action: action)
                    .buttonStyle(.borderedProminent)
                    .frame(minHeight: 44)
            }
        }
        .padding(24)
        .frame(maxWidth: .infinity)
    }
}

/// 送信中はボタンを無効にし、二重送信を防ぐ（5.9.1）
struct AsyncButton<Label: View>: View {
    var role: ButtonRole?
    let action: () async -> Void
    @ViewBuilder let label: () -> Label
    @ViewState private var isRunning = false

    var body: some View {
        Button(role: role) {
            guard !isRunning else { return }
            isRunning = true
            Task {
                await action()
                isRunning = false
            }
        } label: {
            HStack(spacing: 8) {
                if isRunning { ProgressView().controlSize(.small) }
                label()
            }
            .frame(minHeight: 44)
        }
        .disabled(isRunning)
    }
}

extension AsyncButton where Label == Text {
    init(_ title: String, role: ButtonRole? = nil, action: @escaping () async -> Void) {
        self.role = role
        self.action = action
        self.label = { Text(title) }
    }
}

/// 5.9.1：文字数制限に対する現在の文字数を常に表示する
struct CharacterCounterView: View {
    let text: String
    let limit: Int?

    var body: some View {
        let count = CharacterCount(text: text, limit: limit)
        Text(count.label)
            .font(.caption.monospacedDigit())
            .foregroundStyle(count.isOverLimit ? .red : (count.isWithinTolerance ? .secondary : .orange))
            .accessibilityLabel("文字数 \(count.label)")
    }
}

/// 3.6.2：更新中であることを控えめに示す
struct RefreshingIndicator: View {
    let isRefreshing: Bool

    var body: some View {
        if isRefreshing {
            HStack(spacing: 6) {
                ProgressView().controlSize(.small)
                Text("更新中").font(.caption).foregroundStyle(.secondary)
            }
        }
    }
}

/// 差分の表示（S-ES-05 / S-AI-03 の ES 推敲）
struct DiffTextView: View {
    let old: String
    let new: String

    var body: some View {
        let segments = TextDiff.diff(old: old, new: new)
        segments.reduce(Text("")) { result, segment in
            switch segment {
            case .equal(let text):
                return result + Text(text)
            case .inserted(let text):
                return result + Text(text).foregroundColor(.green).underline()
            case .deleted(let text):
                return result + Text(text).foregroundColor(.red).strikethrough()
            }
        }
        .font(.body)
        .textSelection(.enabled)
        .accessibilityLabel("変更点 \(TextDiff.changeCount(segments)) 文字")
    }
}

/// iOS 専用の修飾子を macOS でもコンパイルできるようにまとめる
extension View {
    func inlineNavigationTitle() -> some View {
        #if os(iOS)
        navigationBarTitleDisplayMode(.inline)
        #else
        self
        #endif
    }

    func emailInput() -> some View {
        #if os(iOS)
        textInputAutocapitalization(.never)
            .keyboardType(.emailAddress)
            .textContentType(.emailAddress)
            .autocorrectionDisabled()
        #else
        self
        #endif
    }

    func urlInput() -> some View {
        #if os(iOS)
        textInputAutocapitalization(.never)
            .keyboardType(.URL)
            .autocorrectionDisabled()
        #else
        self
        #endif
    }
}

enum ToolbarPlacement {
    static var trailing: ToolbarItemPlacement {
        #if os(iOS)
        .topBarTrailing
        #else
        .automatic
        #endif
    }
}

/// SwiftUI の状態プロパティ（プロパティラッパー版の `State`）。
/// 新しい SDK では `@ViewState` が SwiftUIMacros のマクロとして解決されるが、Xcode を入れていない環境
/// （Command Line Tools だけの CI・開発機）にはマクロのプラグインがなく、ビルドできない。
/// 動作は同じなので、どちらの環境でもビルドできるプロパティラッパー版を使う。
typealias ViewState = SwiftUI.State
