// DesignDocument/3_UIUXDesign.md 3.3.1 認証系（S-Auth-01〜04）/ 3.8 オンボーディング / 5.5.1 サインアップ・ログイン
import InterviewHackKit
import SwiftUI

/// S-Auth-01 ログイン
struct LoginView: View {
    @Environment(AppModel.self) private var model
    @ViewState private var email = ""
    @ViewState private var password = ""
    @ViewState private var error: Error?

    var body: some View {
        Form {
            Section {
                TextField("メールアドレス", text: $email).emailInput()
                SecureField("パスワード", text: $password)
                    #if os(iOS)
                    .textContentType(.password)
                    #endif
            }
            if let error {
                ErrorBanner(error: error)
            }
            Section {
                AsyncButton("ログイン") {
                    do {
                        _ = try await model.auth.signIn(email: email.trimmingCharacters(in: .whitespaces), password: password)
                    } catch {
                        self.error = error
                    }
                }
                .disabled(email.isEmpty || password.isEmpty)
            }
            Section {
                NavigationLink("新規登録") { SignUpView() }
                NavigationLink("パスワードを忘れた場合") { PasswordResetView() }
            }
        }
        .navigationTitle("InterviewHack")
    }
}

/// S-Auth-02 新規登録
struct SignUpView: View {
    @Environment(AppModel.self) private var model
    @ViewState private var email = ""
    @ViewState private var password = ""
    @ViewState private var error: Error?

    var body: some View {
        Form {
            Section(footer: Text("パスワードは 6 文字以上にしてください")) {
                TextField("メールアドレス", text: $email).emailInput()
                SecureField("パスワード", text: $password)
                    #if os(iOS)
                    .textContentType(.newPassword)
                    #endif
            }
            if let error {
                ErrorBanner(error: error)
            }
            AsyncButton("登録する") {
                do {
                    _ = try await model.auth.signUp(email: email.trimmingCharacters(in: .whitespaces), password: password)
                } catch {
                    self.error = error
                }
            }
            .disabled(email.isEmpty || password.count < 6)
        }
        .navigationTitle("新規登録")
    }
}

/// S-Auth-03 パスワードリセット
struct PasswordResetView: View {
    @Environment(AppModel.self) private var model
    @ViewState private var email = ""
    @ViewState private var sent = false
    @ViewState private var error: Error?

    var body: some View {
        Form {
            TextField("メールアドレス", text: $email).emailInput()
            if sent {
                Label("再設定用のメールを送りました", systemImage: "checkmark.circle")
            }
            if let error {
                ErrorBanner(error: error)
            }
            AsyncButton("再設定メールを送る") {
                do {
                    try await model.auth.sendPasswordReset(email: email.trimmingCharacters(in: .whitespaces))
                    sent = true
                } catch {
                    self.error = error
                }
            }
            .disabled(email.isEmpty)
        }
        .navigationTitle("パスワードリセット")
    }
}

/// S-Auth-04 初回オンボーディング（3.8）
struct OnboardingView: View {
    let onFinish: () -> Void
    @ViewState private var step = 0

    private let steps: [(String, String, String)] = [
        ("sparkles", "AI連携のしくみ",
         "このアプリは AI の API を使いません。アプリが作ったプロンプトを、ChatGPT や Gemini などのアプリに渡し、回答を貼り戻して整理します。"),
        ("hand.raised", "外部AIに渡すデータの注意",
         "プロンプトには入力した自己資産や企業情報が含まれます。外部AIでの扱いは各サービスの規約・設定に従います。他人の個人情報や社外秘の資料は含めないでください。"),
        ("person.text.rectangle", "まず自己資産を3件",
         "ガクチカ・強み・エピソードなどを「データ」タブから登録すると、想定Q&AやESの材料になります。"),
        ("building.2", "企業を1件登録",
         "企業名と職種だけで大丈夫です。登録したら「対策」タブから想定Q&Aのプロンプトを作ってみましょう。"),
    ]

    var body: some View {
        VStack(spacing: 24) {
            Spacer()
            Image(systemName: steps[step].0)
                .font(.system(size: 48))
                .foregroundStyle(.tint)
                .accessibilityHidden(true)
            Text(steps[step].1).font(.title2.bold())
            Text(steps[step].2)
                .multilineTextAlignment(.center)
                .padding(.horizontal)
            Spacer()
            Button(step == steps.count - 1 ? "はじめる" : "次へ") {
                if step == steps.count - 1 { onFinish() } else { step += 1 }
            }
            .buttonStyle(.borderedProminent)
            .frame(minHeight: 44)
            .padding(.bottom, 32)
        }
        .navigationTitle("ようこそ")
    }
}
