# 5. クライアント設計（iOS / SwiftUI）

> 改訂：2026-10-22。方針転換メモ（`DirectionChange_20261022.md`）の決定事項 D1〜D16 と、2.10 共通契約を反映した。旧版（Vite + React + PWA + Cloudflare Pages）は廃止した。

## 5.1 方針（クライアントの責務と非責務）

### 5.1.1 クライアントの責務

- 画面・UI・状態管理（UX の実現）
- Firebase Auth（メールアドレス＋パスワード）によるログイン・ログアウト・パスワードリセット（D5）
- Go BFF の API 呼び出し（CRUD・AI連携・共有リンク・退会のすべて）（D6）
- AI連携の UI（D7 / D10）
  - BFF が返したプロンプトを、共有シートやクリップボードで外部AIアプリに渡す
  - 外部AIの回答を貼り戻して BFF に解析を依頼し、プレビューで確認・修正してから保存する
- 企業研究資料の端末側での取り込み（D11）
  - URL：WKWebView で本文を抽出する
  - PDF：PDFKit でテキストを抽出する
  - スキャンPDF・画像：Vision で OCR する
- 読み取りキャッシュと下書きの端末保存（D16）

### 5.1.2 クライアントの非責務（禁止）

- DB への直接アクセス（すべて BFF 経由。Neon の接続情報をアプリに置かない）
- LLM API の直接呼び出し（D7。そもそも API を使わない）
- プロンプトテンプレートの生成、回答の解析（Go BFF の Domain 層が担う。D10）
- user_id の自己申告（BFF が Firebase ID トークンから確定する）
- 共有リンクのトークンの生成（BFF が生成する。クライアントは返された URL を表示するだけ）
- 書き込みを含むオフライン同期（D16）

---

## 5.2 採用技術スタック（確定）

| 区分 | 採用 |
|------|------|
| 言語・UI | Swift / SwiftUI |
| 最小 OS | iOS 17（SwiftData を使うため） |
| 状態管理 | Observation（`@Observable`） |
| 端末キャッシュ・下書き | SwiftData |
| 認証 | Firebase iOS SDK（Auth のみ。Swift Package Manager で導入） |
| 通信 | URLSession（async/await） |
| URL 本文抽出 | WebKit（WKWebView） |
| PDF テキスト抽出 | PDFKit |
| OCR | Vision（`VNRecognizeTextRequest`。日本語を指定する） |
| 秘密情報の保存 | Keychain（Firebase SDK が管理） |
| 外部AIへの受け渡し | `ShareLink` / `UIActivityViewController`、`UIPasteboard` |
| ログ | os.Logger（フェーズ1） |
| テスト | XCTest / Swift Testing（11 章に従う） |

- Firebase 以外のサードパーティ依存は追加しない。追加するときは、設計を先に更新する

---

## 5.3 アプリ構造（ディレクトリ設計）

リポジトリ直下の `ios/` に置く（2.10.5）。

```
ios/
  InterviewHack.xcodeproj
  InterviewHack/
    App/                 # @main, ルートの画面切り替え（ログイン済みかどうか）, DI
    Config/              # xcconfig（Dev / Prod）, 環境値の読み出し
    Core/
      API/               # APIClient（URLSession）, エンドポイント定義, DTO, エラー変換
      Auth/              # Firebase Auth のラッパー（AuthService）
      Cache/             # SwiftData のモデル（キャッシュ・下書き）, キャッシュ消去
      Logging/           # os.Logger の共通設定
      UI/                # 共通コンポーネント（Empty / Loading / Error, 文字数カウンタ）
    Features/
      Home/              # ダッシュボード, 未完了の AI連携一覧
      Assets/            # 自己資産, タグ
      Companies/         # 企業, 企業研究資料
      Research/          # 取り込み（WKWebView 抽出, PDFKit, Vision）
      ES/                # ESプロジェクト, セクション編集, 版の履歴（差分表示）, 共有リンク
      Prep/              # 面接対策ルーム（想定Q&A）
      Interviews/        # 面接, 面接ログ, 添削
      QA/                # Q&Aバンク, 復習
      AILink/            # AI連携の共通 UI（プロンプト表示 → 受け渡し → 貼り戻し → プレビュー）
      Settings/          # アカウント, ログアウト, 退会
    Resources/           # Assets.xcassets, GoogleService-Info（環境別, 5.12 参照）
  InterviewHackTests/
  InterviewHackUITests/
```

- 機能ごとに `Features/<機能>/` の中で View と、その画面の状態を持つ `@Observable` のモデルをまとめる
- 機能をまたいで使うものだけを `Core/` に置く

---

## 5.4 画面ナビゲーション（タブ＋NavigationStack）

### 5.4.1 ルート構成

- 未ログイン：ログイン画面（サインアップ・パスワードリセットへの導線を含む）
- ログイン済み：`TabView`。各タブは独自の `NavigationStack` を持つ

| タブ | ルート画面 | 主な遷移先 |
|------|-----------|-----------|
| Home | ダッシュボード（S-Home-01） | 未完了の AI連携の再開（S-AI-04）、直近の面接、最近の Q&A、設定（S-Set-01） |
| 対策 | 面接対策ルーム（S-Prep-01）/ ESプロジェクト一覧（S-ES-01） | 企業選択 → 自己資産選択 → AI連携 → Q&A 保存 / ESエディタ → 推敲（AI連携）→ 共有リンク |
| 記録 | 面接一覧（S-Log-01） | 面接ログ入力 → 添削（AI連携） |
| バンク | Q&A一覧（S-QA-01） | Q&A詳細、復習（カード） |
| データ | 自己資産DB一覧（S-Data-01）/ 企業DB一覧（S-Data-03） | 自己資産・企業の作成/編集、企業研究資料の管理・取り込み |

- タブ構成は 3.2.1 に従う（タブは 5 つまで）。「対策」タブには面接対策ルームと ES 作成ルームの両方を置く
- 設定は、Home のナビゲーションバー右上のボタンから開く（3.2.1）
- 画面の詳細と ID は 3 章（UI/UX 設計）に従う

### 5.4.2 AI連携の画面フロー（共通）

`Features/AILink` に、全ユースケース共通のフローを置く。

1. **条件選択**：use_case ごとの対象（企業、ES セクション、面接ログなど）と、使う自己資産・資料を選ぶ
2. **プロンプト表示**：`POST /prompt-sessions` の `promptText` を表示する。「ChatGPT などに送る」（共有シート）と「コピー」のボタンを置く
3. **貼り戻し**：テキスト入力欄と「クリップボードから貼り付け」ボタンを置く。`POST /prompt-sessions/{id}/parse` を呼ぶ
4. **プレビュー**：`preview` と `warnings` を表示する。解析に失敗した部分（`parseResult = partial / failed`）は、編集できる形で表示する
5. **保存**：`POST /prompt-sessions/{id}/commit` を呼ぶ。成功したら、取り込み先の画面（Q&A、ES セクション、面接ログなど）へ戻る

- 2 の画面で外部AIアプリに移動し、iOS がアプリを終了させても、Home の「未完了の AI連携」から 2 に戻れる（`GET /prompt-sessions?status=created,pasted` と `GET /prompt-sessions/{id}/prompt`）
- 解析しないユースケース（`summary_generate` / `es_generate` / `es_refine`）でも、同じフローで parse を呼び、「全文のプレビュー → 保存」にする（parse を経ずに commit はできない）
- ユーザーが中止したら `POST /prompt-sessions/{id}/abandon` を呼ぶ

---

## 5.5 認証設計（Firebase Auth）

### 5.5.1 サインアップ・ログイン

- サインアップ：メールアドレス＋パスワードで `Auth.auth().createUser`
- ログイン：`Auth.auth().signIn(withEmail:password:)`
- パスワードリセット：`Auth.auth().sendPasswordReset(withEmail:)`
- ログイン成功後に `GET /me` を呼ぶ。BFF はこの時点で `users` を取得または作成する（2.10.3）
- Google ログインは追加候補とする。Sign in with Apple は Apple Developer Program に加入してから追加する（D5、2.11）

### 5.5.2 トークンの取得と更新

- API を呼ぶたびに `currentUser.getIDToken()` で ID トークンを取得する。期限が近いときは SDK が自動で更新する
- BFF が 401 を返したら、`getIDTokenForcingRefresh(true)` で一度だけ再取得して再送する。それでも 401 なら、ログイン画面へ戻す
- ログイン状態の変化は `addStateDidChangeListener` で受け取り、ルート画面を切り替える

### 5.5.3 トークンの保存先

- トークンとセッションの保存は Firebase SDK に任せる（Keychain に保存される）
- アプリ独自に、トークンを UserDefaults・SwiftData・ファイルへ保存しない

### 5.5.4 ログアウト・退会

- ログアウト：`Auth.auth().signOut()` のあと、SwiftData のキャッシュと下書きをすべて消す（D16）
- 退会：`DELETE /me` が成功したら、`Auth.auth().currentUser?.delete()` で Firebase のユーザーを削除し（直近のログインが必要な場合は再認証を求める）、そのあとログアウトと同じ処理をする（6.3.3）
  - BFF は Firebase のユーザー削除を行わない。Firebase の削除に失敗しても、`users.deleted_at` が設定済みのため、以後の API 呼び出しは 401 になる

---

## 5.6 データアクセス設計

- すべての読み書きは Go BFF の API を経由する（D6）。エンドポイントは 2.10.3 と 7 章に従う
- クライアントが直接触るのは、端末内の SwiftData（キャッシュと下書き）だけ
- 旧設計の「Supabase 直アクセス」と「BFF 専用テーブル」の区別はなくなった

---

## 5.7 APIクライアント設計

### 5.7.1 APIClient（共通）

- 入力：method、path、body（Encodable）
- 自動で付けるヘッダ
  - `Authorization: Bearer <Firebase ID トークン>`（`GET /health` 以外）
  - `X-Request-Id`：クライアントで UUID を生成して付ける。BFF が返した `X-Request-Id` を受け取り、エラー表示とログに使う
  - `Content-Type: application/json`
- ベース URL は環境ごとの設定から読む（5.12）
- エラーの変換
  - BFF のエラー形式（`status` / `code` / `message`）を `APIError` に変換する
  - 画面には `message` を表示し、分岐には `code` を使う
  - エラー画面には `request_id` を小さく表示する（BFF のログと照合するため）
- 主な `code` の扱い
  - `UNAUTHORIZED`：トークンを再取得して一度だけ再送する（5.5.2）
  - `SESSION_NOT_FOUND` / `SESSION_STATE_CONFLICT`：AI連携を最初からやり直すよう案内する。二重送信だった場合は、保存済みの画面へ移る
  - `FEATURE_DISABLED`（503）：「この機能は一時停止中です」と表示する。commit の場合は貼り戻した内容をプレビュー画面に残し、あとで再度保存できるようにする（12.11）
  - `VALIDATION_ERROR`：入力欄にエラーを表示する

### 5.7.2 リトライ方針

- リトライするのは、ネットワーク断・タイムアウト・429・503 のみ。指数バックオフで最大 2 回
  - 429 で `Retry-After` があれば、その秒数を待つ
- Neon のコールドスタートを考え、タイムアウトは 15 秒とする
- 書き込み（POST / PATCH / DELETE）は、サーバーが処理を終えたか分からないとき自動で再送しない
  - 例外：`commit` は status の条件付き更新で二重保存を防いでいるため（2.10.3）、再送してよい

---

## 5.8 状態管理とキャッシュ（D16）

### 5.8.1 状態の持ち方

- 画面の状態は、機能ごとの `@Observable` モデルが持つ
- サーバーから取得したデータは、取得した時点で SwiftData のキャッシュにも書く
- 画面は「キャッシュを先に表示 → API で最新を取得 → 差し替え」の順に描画する（1.2.1：コールドスタート中も待たせない）

### 5.8.2 キャッシュの範囲

| 対象 | キャッシュ | オフライン時 |
|------|-----------|-------------|
| 自己資産・タグ | する | 閲覧のみ |
| 企業・企業研究資料 | する | 閲覧のみ |
| ESプロジェクト・セクション | する | 閲覧のみ（編集中の本文は下書きとして保存） |
| 面接・面接ログ | する | 閲覧のみ（入力中のログは下書きとして保存） |
| Q&A | する | 閲覧・復習（復習済みチェックはオンライン時のみ） |
| ダッシュボード要約 | する | 閲覧のみ |
| 未完了の AI連携 | しない | 利用不可（プロンプト生成にオンラインが必要） |
| 共有リンクの一覧 | しない | 利用不可 |

- キャッシュはサーバーデータの写しであり、正はサーバー側とする。差し替え時はサーバーの内容で上書きする
- 書き込みを含むオフライン同期と、衝突の解決は行わない

### 5.8.3 下書きの端末保存

- 対象：入力中の面接ログ、編集中の ES セクション本文、貼り戻した直後の回答テキスト（保存前）
- 入力のたびに SwiftData へ保存する。送信に成功したら下書きを消す
- オンラインに戻ったときは、下書きがあることを画面に表示し、ユーザーの操作で送信する（自動送信はしない）

### 5.8.4 キャッシュの保護と消去

- SwiftData のストアファイルには、Data Protection の `completeUnlessOpen` 以上の保護クラスを設定する（端末のロック中は読めない）
- ログアウト・退会・別アカウントでのログイン時に、キャッシュと下書きをすべて消す

---

## 5.9 フォームとバリデーション

### 5.9.1 基本ルール

- 必須項目は、空文字や空白だけの入力を不可にする
- 最大文字数（例：質問 300 字、回答 2000 字）はクライアントでも検査する。ただし正は BFF の検査とする（9 章）
- 送信中はボタンを無効にし、二重送信を防ぐ
- ES セクションには、文字数制限に対する現在の文字数を常に表示する

### 5.9.2 フォーム一覧

- 自己資産の作成・編集（種別、タイトル、本文、タグ）
- 企業の作成・編集
- 企業研究資料の取り込み（URL / PDF / 画像 / メモ）と、抽出テキストの編集
- ESプロジェクトの作成（企業、提出期限）
- ES セクションの執筆（文字数カウント、下書き保存）
- 面接の作成
- 面接ログの入力（複数追加）
- AI連携の条件（企業、職種、自己資産・資料の選択、オプション）
- 共有リンクの発行（公開セクション、有効期限 1 / 7 / 30 日、企業名の表示有無）

---

## 5.10 資料の取り込み（D11）

### 5.10.1 URL（WKWebView）

1. ユーザーが URL を入力すると、アプリ内の WKWebView でページを開く
2. ページの読み込み完了後、ユーザーが「本文を取り込む」を押す
3. `evaluateJavaScript` で `document.body.innerText` を取得する（`article` / `main` 要素があれば、そちらを優先する）
4. 抽出したテキストを編集画面に表示する。ユーザーが不要な部分を消してから保存する
5. `POST /research-materials` に、type = `url`、`source_url`、`content_raw` を送る

- 取り込みはユーザーの操作で 1 ページずつ行う。自動クローリングはしない
- WKWebView は非永続のデータストア（`nonPersistent()`）を使い、Cookie などを残さない
- サーバー側では URL を取得しない（9 章：SSRF 対策が不要）

### 5.10.2 PDF（PDFKit）

- ファイル選択（`fileImporter`）で PDF を選び、`PDFDocument` の `string` でテキストを抽出する
- テキストがほとんど取れない場合は、スキャン PDF とみなし、各ページを画像にして 5.10.3 の OCR に回す
- 保存するのは抽出したテキストだけ。PDF ファイルはアップロードも保存もしない

### 5.10.3 画像・スキャン PDF（Vision）

- `VNRecognizeTextRequest` を使い、認識言語を `ja-JP` と `en-US`、精度を `accurate` にする
- 認識結果を編集画面に表示し、ユーザーが修正してから保存する
- 画像ファイルはアップロードも保存もしない

### 5.10.4 長い資料

- 抽出テキストが長い場合（目安：3,000 字を超える。プロンプトでは要約がない資料を 1 件あたり先頭 3,000 字に切り詰めるため。4.3.5）、保存時に「先に要約（AI連携：`research_summarize`）すると、プロンプトで使いやすくなります」と案内する
  - `research_summarize` で使うのは `content_raw` の先頭 8,000 字までであることも併せて表示する（4.3.5）

---

## 5.11 外部AIへの受け渡し（D7）

- 「送る」ボタンは `ShareLink`（または `UIActivityViewController`）で `promptText` を共有する。ChatGPT や Gemini などのアプリは、ユーザーが共有先として選ぶ
- 「コピー」ボタンは `UIPasteboard.general.string` に書き込む
- 貼り戻しの「クリップボードから貼り付け」は、ユーザーがボタンを押したときだけクリップボードを読む（iOS の貼り付け許可の表示に従う）
- 初回の受け渡し時に、外部AIサービスにデータを渡すことの注意を表示する（9 章）
- アプリは外部AIアプリの起動や操作を自動化しない

---

## 5.12 環境設定（dev / prod）

### 5.12.1 構成の切り替え

- Xcode の Build Configuration を `Debug-Dev` と `Release-Prod` に分け、それぞれ xcconfig を割り当てる
- xcconfig に置く値
  - `API_BASE_URL`（dev：ローカルの Go BFF、prod：Cloud Run の URL）
  - `APP_ENV`（`dev` / `prod`）
- Info.plist 経由で値を読み出し、`Config/` の型で扱う

### 5.12.2 GoogleService-Info.plist

- Firebase のプロジェクトを dev と prod で分ける（2.10.6）
- `GoogleService-Info-Dev.plist` と `GoogleService-Info-Prod.plist` を用意し、ビルド時に構成に応じたものを `GoogleService-Info.plist` としてコピーする（Run Script）
- この plist に含まれる API キーは秘密情報ではない（クライアントに埋め込まれる前提の値）。ただし、Firebase Console と Google Cloud Console で、キーの利用を iOS のバンドル ID と必要な API に制限する（9 章）
- プライバシーと運用の観点から、plist はリポジトリにコミットせず、各自がローカルに置く。手順は README に書く

### 5.12.3 配布

- フェーズ1：Xcode から開発者本人の iPhone に直接インストールする（無料 Apple ID。7 日ごとに再インストールが必要）（D2）
- フェーズ2：Apple Developer Program に加入し、TestFlight で配布する

---

## 5.13 セキュリティ（クライアント観点）

- アプリに秘密情報を埋め込まない。`DATABASE_URL` などのサーバーのシークレットは、アプリにも xcconfig にも置かない
- ATS（App Transport Security）を有効のままにする。HTTP 通信の例外は、dev のローカル BFF（`localhost`）だけに限定し、Debug-Dev 構成にだけ設定する
- ユーザー入力や外部AIの回答を HTML や Markdown として実行・解釈しない（プレーンテキストで表示する）
- 取り込み用の WKWebView では、ページ内のリンクで外部アプリを開かせない。ファイルのダウンロードも禁止する
- キャッシュは Data Protection で保護し、ログアウト時に消す（5.8.4）
- os.Logger には本文データやメールアドレスを出さない。出す場合は `privacy: .private` を指定する

---

## 5.14 エラー収集

- フェーズ1：Xcode のデバッグと os.Logger（subsystem はバンドル ID、category は機能名）で確認する
  - ログには `request_id`・画面名・エラーコードを出し、本文は出さない
- フェーズ2：TestFlight のクラッシュレポートを使う。必要になったら Sentry の導入を検討する（D15）

---

## 5.15 この章の凍結事項

- クライアントは iOS ネイティブ（SwiftUI、最小 iOS 17）のみ。Android と Web アプリは作らない
- データアクセスはすべて Go BFF 経由。DB の接続情報はアプリに置かない
- 認証は Firebase Auth（メールアドレス＋パスワード）。トークンの保存は SDK（Keychain）に任せる
- プロンプトの生成と回答の解析はサーバー（Go）で行い、クライアントは表示・受け渡し・貼り戻し・確認だけを担う
- 企業研究資料の取り込みは端末側（WKWebView / PDFKit / Vision）で行い、テキストだけをサーバーに送る
- オフラインは読み取りキャッシュと下書き保存のみ。書き込みの同期はしない
- 全リクエストに `X-Request-Id` を付ける
