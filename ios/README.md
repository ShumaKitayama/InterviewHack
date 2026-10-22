# InterviewHack iOS アプリ

DesignDocument/5_FrontendDesign.md（iOS クライアント設計）と 3_UIUXDesign.md（画面）に従った SwiftUI アプリです。
最小 iOS は 17。データの読み書きはすべて Go BFF（`backend/`）の API を経由します。

## 構成

```
ios/
  project.yml              # XcodeGen の定義（InterviewHack.xcodeproj はここから生成し、コミットしない）
  InterviewHackKit/        # Swift Package（Xcode なしでビルド・テストできる）
    Sources/InterviewHackKit/
      API/                 # APIClient（Authorization / X-Request-Id / 401 再試行 / リトライ）、全エンドポイント
      Models/              # API の DTO（camelCase）
      Auth/                # AuthService / AuthTokenProvider（Firebase 実装はアプリ本体）
      Cache/               # CacheStore・DraftStore（プロトコルとファイル実装）、CachedResource
      Session/             # ログアウト・退会・別アカウントでのキャッシュ消去
      AILink/              # AI連携の状態機械（プロンプト → 貼り戻し → 解析 → プレビュー → 保存）
      Research/            # 資料取り込み（本文抽出の後処理、PDFKit、Vision OCR、長さの案内）
      Support/             # 設定、JSON、差分、入力検査、クリップボード、オンライン監視
    Sources/InterviewHackUI/  # SwiftUI の画面（3 章の画面 ID ごと）
    Tests/InterviewHackKitTests/
  App/                     # アプリ本体（Xcode でのみビルドできる）
    Sources/               # @main、FirebaseAuthService、SwiftData のキャッシュ実装
    Config/                # Dev.xcconfig / Prod.xcconfig、Info.plist（Dev は localhost の HTTP を許可）
    Resources/             # GoogleService-Info-Dev / Prod.plist を置く（コミットしない）
    Tests/                 # SwiftData ストアの XCTest
```

## Xcode なしでのビルドとテスト（ロジックと画面）

```bash
cd ios/InterviewHackKit
swift build
swift test
```

Command Line Tools だけの環境で `swift test` が「plugin for module 'TestingMacros' not found」で失敗する場合は、プラグインのパスを指定してください。

```bash
swift test -Xswiftc -plugin-path -Xswiftc /Library/Developer/CommandLineTools/usr/lib/swift/host/plugins/testing
```

画面では `@State` の代わりに `@ViewState`（`SwiftUI.State` の型エイリアス）を使っています。新しい SDK では `@State` がマクロとして解決されますが、Command Line Tools にはそのマクロのプラグインがなく、ビルドできないためです。動作は同じです。

## アプリのビルド（Xcode が必要）

1. Xcode と XcodeGen を入れる（`brew install xcodegen`）
2. Firebase の dev / prod プロジェクトで iOS アプリを登録し、`GoogleService-Info.plist` を `App/Resources/GoogleService-Info-Dev.plist` と `GoogleService-Info-Prod.plist` として置く（App/Resources/README.md）
3. `App/Config/Prod.xcconfig` の `API_BASE_URL` を Cloud Run の URL にする
4. プロジェクトを生成して開く

   ```bash
   cd ios
   xcodegen generate
   open InterviewHack.xcodeproj
   ```

5. スキーム `InterviewHack`、構成 `Debug-Dev` で実機またはシミュレータを選んで実行する
   - dev はローカルの Go BFF（`http://localhost:8080`）につなぐ。実機から使う場合は、Mac の IP アドレスに置き換えるか、Cloud Run の dev 環境を使う
   - 無料の Apple ID では、アプリは 7 日ごとに再インストールが必要（5.12.3）

テストは Xcode の Product > Test（`InterviewHackTests` と `InterviewHackKitTests`）、またはコマンドで実行します。

```bash
xcodebuild test -project InterviewHack.xcodeproj -scheme InterviewHack -configuration Debug-Dev -destination 'platform=iOS Simulator,name=iPhone 16'
```
