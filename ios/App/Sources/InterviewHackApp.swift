// DesignDocument/5_FrontendDesign.md 5.3 App（@main、DI）/ 5.12 環境設定
import FirebaseCore
import Foundation
import InterviewHackKit
import InterviewHackUI
import SwiftUI

@main
struct InterviewHackApp: App {
    private let model: AppModel

    init() {
        // 5.12.2：ビルド構成ごとの GoogleService-Info.plist が Run Script でバンドルに入る
        FirebaseApp.configure()

        let config: AppConfig
        do {
            config = try AppConfig.load(from: Bundle.main.infoDictionary ?? [:])
        } catch {
            fatalError("API_BASE_URL / APP_ENV の設定が不正です（xcconfig を確認してください）: \(error)")
        }

        let auth = FirebaseAuthService()
        let storeDirectory = Self.storeDirectory()
        let store: SwiftDataLocalStore
        do {
            store = try SwiftDataLocalStore(directory: storeDirectory)
        } catch {
            fatalError("端末キャッシュを開けませんでした: \(error)")
        }
        let unauthorized = UnauthorizedHandler()
        let api = APIClient(baseURL: config.apiBaseURL, tokenProvider: auth, onUnauthorized: {
            await unauthorized.handle()
        })
        model = AppModel(api: api, auth: auth, cache: store, drafts: store, lastUser: UserDefaultsLastUserStore(),
                         connectivity: ConnectivityMonitor())
        unauthorized.model = model
    }

    var body: some Scene {
        WindowGroup {
            RootView(model: model)
        }
    }

    private static func storeDirectory() -> URL {
        let base = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask)[0]
        return base.appendingPathComponent("InterviewHackStore", isDirectory: true)
    }
}

/// 5.5.2：トークンを再取得しても 401 なら、ログイン画面へ戻す
final class UnauthorizedHandler: @unchecked Sendable {
    @MainActor weak var model: AppModel?

    @MainActor
    func handle() {
        model?.signOut()
    }
}
