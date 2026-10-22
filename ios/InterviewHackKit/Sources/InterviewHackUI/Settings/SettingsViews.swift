// DesignDocument/3_UIUXDesign.md 3.3.4 設定（S-Set-01 / 03 / 04）/ 5_FrontendDesign.md 5.5.4 ログアウト・退会
import InterviewHackKit
import SwiftUI

/// S-Set-01 設定トップ
struct SettingsView: View {
    @Environment(AppModel.self) private var model
    @ViewState private var displayName = ""
    @ViewState private var error: Error?

    var body: some View {
        Form {
            Section("アカウント") {
                LabeledContent("メールアドレス", value: model.me?.email ?? model.currentUser?.email ?? "-")
                TextField("表示名", text: $displayName)
                AsyncButton("表示名を保存") {
                    do {
                        try await model.updateDisplayName(displayName)
                    } catch {
                        self.error = error
                    }
                }
                .disabled(!model.connectivity.isOnline)
            }
            if let error {
                ErrorBanner(error: error)
            }
            Section {
                NavigationLink("AI連携とデータの扱い") { DataPolicyView() }
            }
            Section {
                Button("ログアウト", role: .destructive) { model.signOut() }
                NavigationLink("アカウントを削除") { AccountDeletionView() }
                    .foregroundStyle(.red)
            }
        }
        .navigationTitle("設定")
        .onAppear { displayName = model.me?.displayName ?? "" }
    }
}

/// S-Set-03 アカウント削除（6.3.3）
struct AccountDeletionView: View {
    @Environment(AppModel.self) private var model
    @ViewState private var confirming = false
    @ViewState private var error: Error?
    @ViewState private var outcome: SessionManager.DeletionOutcome?

    var body: some View {
        Form {
            Section {
                Text("アカウントを削除すると、自己資産・企業・ES・面接ログ・Q&A・共有リンクなど、すべてのデータがすぐに削除されます。元に戻せません。")
            }
            if let error {
                ErrorBanner(error: error)
            }
            if outcome == .firebaseDeletionNeedsRecentLogin {
                Section {
                    Text("データの削除は完了しました。ログイン情報の削除には再ログインが必要なため、再度ログインしてから削除してください。")
                }
            }
            Section {
                Button("すべてのデータを削除して退会する", role: .destructive) { confirming = true }
                    .disabled(!model.connectivity.isOnline)
            }
        }
        .navigationTitle("アカウント削除")
        .confirmationDialog("本当に削除しますか？", isPresented: $confirming, titleVisibility: .visible) {
            Button("削除する", role: .destructive) {
                Task {
                    do {
                        outcome = try await model.session.deleteAccount()
                    } catch {
                        self.error = error
                    }
                }
            }
        }
    }
}

/// S-Set-04 AI連携とデータの扱い
struct DataPolicyView: View {
    @Environment(AppModel.self) private var model
    @ViewState private var cleared = false

    var body: some View {
        Form {
            Section("外部AIに渡すデータ") {
                Text("""
                このアプリは AI の API を使わず、運営のサーバーは外部AIサービスにデータを送りません。
                プロンプトはあなたの操作で ChatGPT・Gemini などに渡されます。外部AIでの扱いは各サービスの規約・設定に従います。
                他人の個人情報や社外秘の資料は含めないでください。
                """)
                .font(.callout)
            }
            Section("端末内のデータ") {
                Text("オフラインで閲覧するため、取得したデータと入力途中の下書きを端末に保存しています。ログアウトすると削除されます。")
                    .font(.callout)
                Button("キャッシュと下書きを削除", role: .destructive) {
                    try? model.session.clearLocalData()
                    cleared = true
                }
                if cleared { Label("削除しました", systemImage: "checkmark") }
            }
        }
        .navigationTitle("AI連携とデータの扱い")
    }
}
