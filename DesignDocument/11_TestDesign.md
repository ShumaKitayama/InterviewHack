# 11. テスト設計（詳細版：具体テストケース定義）

> 改訂：2026-10-22。方針転換メモ（`DirectionChange_20261022.md`）の決定事項 D1〜D16 と 2.10 共通契約を反映した。

※ テスト方針の正は `DesignDocument/11_TestDesign.md` 本書および以下に従う：

- アーキテクチャ：DesignDocument/7_BackendDesign.md（DDD + クリーンアーキテクチャ）
- DB・認可（アプリ層での user_id スコープ）：DesignDocument/8_DBDesign.md
- セキュリティ/冪等性：DesignDocument/9_SecurityDesign.md
- CI/CD：DesignDocument/14_CICDDesign.md

## 11.0 前提（テスト環境・データ）

### 環境

- env: dev / prod を分離する（stg はフェーズ2で追加。2.10.6）
- DB（PostgreSQL）
  - 自動テスト：CI の PostgreSQL サービスコンテナを使う。マイグレーション（`backend/migrations`）を番号順に適用してから実行する
  - 手動テスト：Neon の dev ブランチを使う
- 認証（Firebase Auth）
  - 自動テスト：`port/external.AuthVerifier` をスタブに差し替える。実際の検証ロジック（`infra/firebase`）は、テスト用の鍵で署名したトークンで単体テストする
  - 手動テスト：Firebase の dev プロジェクトのテストユーザーを使う
- 外部AI（ChatGPT・Gemini など）
  - 自動テストでは呼ばない（運営側は API を持たない。D7）
  - 実際の回答はサンプルとして収集し、fixture にする（11.7）
- iOS
  - 自動テスト：Xcode のシミュレータ（`xcodebuild test`）
  - 手動テスト：開発者の実機（11.4、11.5）

### テストユーザー

- U1: Email/Password ユーザー（通常）
- U2: Email/Password ユーザー（別アカウント、越境防止の検証用）
- U3: 不正トークン（署名不正・期限切れ・改ざん・`aud` 不一致）

### 共通テストデータ（固定セット）

- Assetセット（最低6件）
  - A1: project（成果に数値あり）
  - A2: work（チーム/役割あり）
  - A3: strength（抽象→具体の両方）
  - A4: weakness（改善行動あり）
  - A5: skill（技術スタック）
  - A6: episode（STARで語れる）
- Companyセット（最低2件）
  - C1: IT企業（job_title=Backend）
  - C2: コンサル系（job_title=Biz）
- Interview（最低2件）
  - I1: C1 の一次（直近日）
  - I2: C2 の二次
- ResearchMaterial（最低2件）
  - R1: C1 の URL 取り込み（抽出テキストあり、summary なし）
  - R2: C1 の PDF 取り込み（抽出テキスト、summary あり）
- ESProject（最低1件）
  - E1: C1 の ES（セクション：志望動機、自己PR）

### テストの合格条件（共通）

- DB整合性が崩れない（二重保存なし、参照関係が壊れない、他ユーザーのデータに届かない）
- APIは規定のエラーコード/メッセージを返す
- UI操作で「戻れない」「二重送信」「無限ローディング」にならない
- 貼り戻した回答が、解析に失敗しても失われない

---

## 11.1 アーキテクチャ前提（DDD + クリーンアーキテクチャとテスト）

本プロジェクトのバックエンドは **DDD + クリーンアーキテクチャ** に従う（正は `DesignDocument/7_BackendDesign.md`）。
テストはレイヤー責務に従って以下の方針で実施する。

- **Domain / Usecase**：
  - 外部I/F（DB, AuthVerifier, TokenGenerator, Clock 等）は interface に抽象化し、モック/スタブで **純粋ユニットテスト**を行う
  - ビジネスルール（テンプレート生成と解析、prompt_sessions の状態遷移、共有リンクのルール、検証ロジック）はここで網羅する
- **Adapter / Infrastructure**：
  - PostgreSQL サービスコンテナと AuthVerifier スタブを用いた **統合テスト**で結合を検証する
  - HTTP ハンドラは薄く保ち、入出力変換とエラー整形の確認を中心にする
- **E2E**：
  - 主要ユーザーフローを実機で検証する（11.4）

禁止事項：

- Domain にフレームワークや SDK 依存（`gin`、`pgx`、`firebase`、`net/http` など）を持ち込む前提のテストを書かない
- ハンドラや Infra にビジネスロジックがある前提のテストを書かない

## 11.1 テスト分類（何をどこまでやるか）

- Unit（Go BFF）
  - テンプレート生成と解析、prompt_sessions の状態遷移、共有トークンと期限、入力バリデーション
- Integration（Go BFF + PostgreSQL + AuthVerifier スタブ）
  - 認証、user_id 越境防止、commit の冪等性、共有ページ、退会、参照整合性
- Unit（iOS）
  - API クライアント、キャッシュ、ログアウト時のキャッシュ削除、下書き保存
- E2E（実機・手動チェックリスト）
  - 主要ユーザーフロー（登録→資料取り込み→AI連携→保存→復習、ES→共有リンク）
- Manual（実機/運用）
  - UI/UX、ネットワーク断、バックグラウンド復帰、外部AIアプリとの往復

---

## 11.2 Unitテスト（Go BFF）— 具体ケース

テンプレートと解析は `domain/service/prompt` の純粋関数で、**テーブル駆動テスト**で書く。

### 11.2.1 テンプレート生成（use_case × バージョン）

- UT-TPL-001: 全 use_case（`qa_generate` / `answer_review` / `summary_generate` / `research_summarize` / `es_generate` / `es_refine`）× 現行バージョンで、プロンプトが生成できる
- UT-TPL-002: 生成したプロンプトに、回答の 1 行目に出すべき形式行（例：`【形式:qa-v1】`）の指示が含まれる
- UT-TPL-003: 選んだ自己資産・企業研究資料だけがプロンプトに含まれ、選んでいないものは含まれない
- UT-TPL-004: 文字数制限・質問数などのオプションがプロンプトに反映される
- UT-TPL-005: 未知の use_case / バージョンを指定 → VALIDATION_ERROR
- UT-TPL-006: 同じ入力からは同じプロンプトが生成される（`GET /prompt-sessions/{id}/prompt` の再生成の前提）
- UT-TPL-007: 生成結果がゴールデンファイルと一致する（11.7）

### 11.2.2 回答の解析（use_case × バージョン）

- UT-PRS-001: `qa-v1` の正しい回答 → `success`。【Q】【A】【意図】の組が番号どおりに取れる
- UT-PRS-002: `review-v1` の正しい回答 → `success`。【指摘】【改善案】【改善後の回答】が取れる
- UT-PRS-003: `research-v1` の正しい回答 → `success`。【魅力・特徴】【求める人物像】【最近のトピック】が取れる
- UT-PRS-004: 形式行が欠落 → セッションの template_version で解析する
- UT-PRS-005: 形式行のバージョンとセッションの template_version が異なる → 形式行のバージョンを優先する
- UT-PRS-006: 一部の項目だけ取れる（例：【A3】が欠落、【改善後の回答】が欠落）→ `partial`。取れた部分と warnings を返す
- UT-PRS-007: 目印がまったくない自由文 → `failed`。原文全体をプレビューに残し、手動修正できる
- UT-PRS-008: 崩れた入力（全角/半角の括弧の揺れ、余分な空行、前後の挨拶文、Markdown の装飾、番号の飛び）→ 許容範囲は `success` または `partial` で取れる
- UT-PRS-009: 解析しない use_case（`summary_generate` / `es_generate` / `es_refine`）→ 形式行があれば取り除き、全文を 1 つの本文として返す（`success`）
- UT-PRS-010: 空文字・最大長（100,000 文字）超過 → VALIDATION_ERROR
- UT-PRS-011: 未知の形式行（例：`【形式:qa-v9】`）→ `failed` と warnings（原文は保持）

### 11.2.3 prompt_sessions の状態遷移

- UT-PS-001: `created` → `pasted`（parse）→ `saved`（commit）が許可される
- UT-PS-002: `created` → `saved`（parse を経ない commit）は許可しない → SESSION_STATE_CONFLICT
- UT-PS-003: `pasted` → `pasted`（貼り直して再解析）は許可される
- UT-PS-004: `saved` → 任意の遷移は拒否 → SESSION_STATE_CONFLICT
- UT-PS-005: `abandoned` → `pasted` / `saved` は拒否 → SESSION_STATE_CONFLICT
- UT-PS-006: `created` / `pasted` → `abandoned` が許可される
- UT-PS-007: 作成から 7 日を過ぎた `created` / `pasted` は `abandoned` に更新される。7 日以内のものは変わらない（Clock をスタブにして境界値を検証）
- UT-PS-008: `input_refs` に本文が入らない（ID とオプションのみ）

### 11.2.4 共有リンク（トークン・期限）

- UT-SH-001: 生成したトークンが 32 バイト（256 ビット）の暗号学的乱数で、base64url で表される（TokenGenerator の実装を検証）
- UT-SH-002: 保存されるのは SHA-256 のハッシュのみで、ハッシュから生トークンを復元できない
- UT-SH-003: expiresInDays は 1 / 7 / 30 のみ許可。それ以外（0、無期限、2 など）→ VALIDATION_ERROR
- UT-SH-004: 期限の境界（Clock スタブで期限の直前 / ちょうど / 直後）→ 有効 / 無効 / 無効
- UT-SH-005: 失効済み（revoked_at あり）→ 無効
- UT-SH-006: スナップショットに選んだセクションだけが入る。showCompanyName=false なら企業名が入らない
- UT-SH-007: スナップショットにユーザー名が入らない
- UT-SH-008: スナップショットの本文は `content_final`、なければ `content_draft` になる
- UT-SH-009: showCompanyName を省略すると false として扱われる

### 11.2.5 入力バリデーション

- UT-VAL-001: companyId などが不正UUID → 400（VALIDATION_ERROR。7.5）
- UT-VAL-002: 面接ログの questionText 空 → 422
- UT-VAL-003: answerText 空 → 422
- UT-VAL-004: assetIds / researchMaterialIds が上限超過（1000件など）→ 422
- UT-VAL-005: テキストが最大長超過 → 422
- UT-VAL-006: commit の内容が use_case の形と一致しない（例：qa_generate に添削の形）→ 422

---

## 11.3 Integrationテスト（Go BFF + PostgreSQL + AuthVerifier スタブ）— 具体ケース

### 11.3.1 認証（Firebase ID トークン）

- IT-AUTH-001: 正常トークンで保護されたエンドポイントにアクセス → 200
- IT-AUTH-002: トークンなし → 401
- IT-AUTH-003: 期限切れトークン → 401
- IT-AUTH-004: 改ざんトークン / `aud` 不一致 → 401
- IT-AUTH-005: user_id をbodyに入れて別ユーザーを指定しても無視される（トークンの uid 優先）
- IT-AUTH-006: 初回アクセスで `users` が作成され、2回目以降は同じ user_id になる
- IT-AUTH-008: `users.deleted_at` が設定されたユーザーのトークン → 401。`users` の email は同期されない
- IT-AUTH-007（infra/firebase 単体）: テスト用の鍵で署名したトークンを検証できる。鍵 ID 不一致・署名不正は拒否する

### 11.3.2 user_id 越境防止（アプリ層の認可）

- IT-AZ-001: U1 が U2 の assets を一覧・取得できない（一覧に出ない / 単体取得は 404）
- IT-AZ-002: U1 が U2 の qa を更新・削除できない → 404
- IT-AZ-003: U1 が U2 の prompt_session を parse / commit / abandon できない → 404
- IT-AZ-004: U1 が U2 の es_project に共有リンクを作れない → 404
- IT-AZ-005: U1 が U2 の共有リンクを失効できない → 404
- IT-AZ-006: prompt_sessions 作成時に U2 の assetIds / researchMaterialIds / companyId を指定 → 拒否され、プロンプトに U2 のデータが入らない
- IT-AZ-007: 全 Repository のクエリが user_id でスコープされている（Repository ごとに U1/U2 のデータを入れて検証）

### 11.3.3 AI連携（prompt_sessions）

- IT-PS-001: `POST /prompt-sessions` → sessionId / templateVersion / promptText が返り、status=`created`、input_refs に本文が入らない
- IT-PS-002: `GET /prompt-sessions?status=created,pasted` に未完了（`created` / `pasted`）のセッションが出る。作成から 7 日を過ぎたものは `abandoned` に更新され、出ない
- IT-PS-003: `GET /prompt-sessions/{id}/prompt` が作成時と同じプロンプトを返す
- IT-PS-004: `POST /parse` → preview が返り、status=`pasted`。他のテーブルには何も保存されない
- IT-PS-005: `POST /commit`（qa_generate）→ `qa` に保存され、status=`saved`
- IT-PS-006: `POST /commit`（answer_review）→ `interview_logs` の ai_feedback_text / ai_improved_answer_text に保存される
- IT-PS-007: `POST /commit`（research_summarize）→ `research_materials.summary` に保存される
- IT-PS-008: `POST /commit`（summary_generate）→ `dashboard_summaries` に保存され、`GET /dashboard` が最新の1件を返す
- IT-PS-009: `POST /commit`（es_generate / es_refine）→ `es_sections` に保存される
- IT-PS-010: **commit の冪等性**：同じ commit を2回送る → 2回目は SESSION_STATE_CONFLICT（409）、データは1件だけ
- IT-PS-011: **同時 commit**：同じセッションに同時に2本 → 片方だけ成功し、データは1件だけ（条件付き UPDATE）
- IT-PS-012: `abandon` 後の parse / commit → 409
- IT-PS-013: 存在しない sessionId → SESSION_NOT_FOUND（404）
- IT-PS-014: プロンプト全文と貼り戻した回答の生データが、どのテーブルにも保存されない
- IT-PS-015: parse を経ずに `created` のセッションへ commit → SESSION_STATE_CONFLICT（409）。解析しない use_case でも同じ
- IT-PS-016: qa_generate の commit 後、`【意図N】` の内容はどのテーブルにも保存されない
- IT-PS-017: es_generate の commit で、親の ES プロジェクトが `not_started` なら `draft` になる。`in_review` など他の状態は変わらない
- IT-PS-018: ES プロジェクトの作成時の status は `not_started`。PATCH で `submitted` から `draft` に戻せる（遷移の制約がない）

### 11.3.4 共有リンク・共有ページ

- IT-SH-001: `POST /es-projects/{id}/share-links` → url / expiresAt が返る。DB には token_hash だけが保存される
- IT-SH-002: `GET /s/{token}`（有効）→ 200、HTML。`X-Robots-Tag: noindex`、`Cache-Control: no-store`、`Referrer-Policy: no-referrer`、7.5.6 に定める `Content-Security-Policy`（`default-src 'none'; style-src 'unsafe-inline'; img-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'`）が付く（410 / 404 / 503 でも同じ）
- IT-SH-012: `FEATURE_SHARE_VIEW=off` で `GET /s/{token}` → 503 の「一時停止中」HTML。`FEATURE_SHARE_CREATE=off` で共有リンクの発行 → 503 `FEATURE_DISABLED`
- IT-PS-019: `FEATURE_COMMIT_DISABLED_USE_CASES` に含まれる use_case の commit → 503 `FEATURE_DISABLED`。セッションは `pasted` のまま。含まれない use_case は通常どおり保存できる
- IT-SH-003: `GET /s/{token}`（期限切れ）→ 410（SHARE_LINK_GONE）
- IT-SH-004: `GET /s/{token}`（失効済み）→ 410
- IT-SH-005: `GET /s/{token}`（存在しない）→ 404
- IT-SH-006: 共有ページにユーザー名が出ない。showCompanyName=false なら企業名も出ない
- IT-SH-007: リンク作成後に ES を編集しても、共有ページはスナップショットのまま
- IT-SH-008: 共有ページは認証なしで見られる。他のエンドポイントは認証なしでは 401
- IT-SH-009: `/s/{token}` への短時間の大量アクセス → IP 単位のレート制限（429）
- IT-SH-010: 共有ページの本文が HTML エスケープされる（スクリプトを含む ES 本文でも実行されない）
- IT-SH-011: ES プロジェクトを削除すると、紐づく share_links も削除され、`GET /s/{token}` は 404 になる

### 11.3.5 退会（データ削除）

- IT-ACC-001: `DELETE /me` → U1 のデータ（assets、companies、research_materials、es_*、interviews、interview_logs、qa、各関連テーブル、prompt_sessions、share_links、dashboard_summaries）が物理削除される。`users` 行は残り、`deleted_at` が設定され、email と display_name が消去される（6.3.3、9.12）
- IT-ACC-002: 退会後、U1 の共有リンクは 404 になる（行が削除されるため）
- IT-ACC-003: 退会しても U2 のデータは影響を受けない
- IT-ACC-004: 退会後に同じ（古い）ID トークンでアクセス → 401 `UNAUTHORIZED`。新しい `users` 行は作られない
- IT-ACC-005: 退会の途中で失敗した場合、どのテーブルも削除されない（1 トランザクション）

### 11.3.6 DB整合性（参照関係）

- IT-DB-001: interviews.company_id が存在しない場合 insert できない（FK）
- IT-DB-002: interview_logs.interview_id が存在しない場合 insert できない
- IT-DB-003: qa_asset_relations の参照が両方存在しないと作れない
- IT-DB-004: es_sections.es_project_id が存在しない場合 insert できない
- IT-DB-005: es_project_asset_relations の参照が両方存在しないと作れない
- IT-DB-006: es_project_research_material_relations の参照が両方存在しないと作れない
- IT-DB-007: share_links.es_project_id が存在しない場合 insert できない
- IT-DB-008: ESセクションを削除すると、その es_section_versions も削除される

### 11.3.6.1 ESセクションの版（es_section_versions）

- IT-VER-001: es_generate / es_refine の commit で、上書き前の本文が source = use_case の版として残る。本文が空なら版は作られない
- IT-VER-002: PATCH `/es-sections/{id}` では版が作られない
- IT-VER-003: POST `/es-sections/{id}/versions` で source = manual の版が作られる。本文が空なら 422
- IT-VER-004: restore で、戻す直前の本文が manual の版として残り、content_draft が選んだ版の本文になり、version が +1 される（1 トランザクション）
- IT-VER-005: 21件目の版を作ると、最も古い版が削除され、20件に保たれる
- IT-VER-006: 他人のセクションの版の一覧・保存・restore は 404
- IT-DB-008: マイグレーションを2回適用してもエラーにならない（冪等）

---

## 11.4 iOS テスト（XCTest / Swift Testing）

### 11.4.1 自動テスト（シミュレータ）

- IOS-API-001: API クライアントが全リクエストに `Authorization` と `X-Request-Id` を付ける
- IOS-API-002: 7.7.1 のエラー形式を解釈し、`code` で分岐できる（401 は再ログイン導線、409 / 410 は対応する文言）
- IOS-API-003: ID トークンの期限切れ時に、更新してから再送する
- IOS-CACHE-001: 取得した自己資産・企業・ES・Q&A が SwiftData にキャッシュされ、オフラインで読める
- IOS-CACHE-002: オフライン時は書き込み操作が無効になり、理由が表示される（下書きを除く）
- IOS-CACHE-003: **ログアウト時にキャッシュと下書きがすべて削除される**
- IOS-CACHE-004: 退会時にキャッシュと下書きがすべて削除される
- IOS-CACHE-005: キャッシュのファイルに Data Protection の保護クラスが設定されている
- IOS-DRAFT-001: 入力途中の面接ログ・ES が端末に保存され、アプリを再起動しても残る
- IOS-DRAFT-002: オンライン復帰後に下書きがあることが表示され、ユーザーの操作で送信できる（自動送信されない）。送信成功後に端末の下書きが消える

---

### 11.4.2 契約テスト（BFF ↔ iOS）

- IOS-CONTRACT-001: BFF の統合テスト（`TestContractFixtures`）が、主要エンドポイントの実レスポンスを JSON fixture として書き出す（`CONTRACT_OUT` を指定したとき）
- IOS-CONTRACT-002: iOS の `ContractTests` が、すべての fixture を Swift のモデルでデコードできる（キー名・型・日時形式・エラー形式の食い違いを検出する）
- API を変更したら fixture を再生成し、両方のテストを通す

## 11.5 E2Eテスト（実機・手動チェックリスト）

開発者の実機で、リリース前（11.9）に毎回実施する。

### 11.5.1 オンボーディング

- [ ] E2E-ONB-001: メールアドレスで登録 → ログイン → 空のホームが表示される
- [ ] E2E-ONB-002: 途中でアプリを落として復帰 → ログイン状態と入力状態が保たれる
- [ ] E2E-ONB-003: パスワードリセットのメールが届き、再設定できる

### 11.5.2 企業研究資料の取り込み

- [ ] E2E-RES-001: URL を開いて本文を抽出 → プレビューで編集 → 保存できる
- [ ] E2E-RES-002: PDF を選んでテキスト抽出 → 保存できる。ファイル本体はサーバーに送られない
- [ ] E2E-RES-003: 画像（スクリーンショット）を OCR → 保存できる
- [ ] E2E-RES-004: 資料の要約（AI連携）→ summary が保存される

### 11.5.3 事前対策（想定Q&A）

- [ ] E2E-PREP-001: 自己資産3件・企業1件を登録 → プロンプト作成 → 共有シートで ChatGPT に渡す → 回答をコピーして貼り戻す → プレビュー → 保存 → Q&Aバンクに表示
- [ ] E2E-PREP-002: Gemini でも同じ流れが完走する
- [ ] E2E-PREP-003: 自己資産が0件でプロンプト作成 → 入力不足のガイドが出る
- [ ] E2E-PREP-004: 回答の一部が崩れている → `partial` のプレビューで手動修正して保存できる

### 11.5.4 事後対策（添削）

- [ ] E2E-REV-001: 面接作成 → ログ入力 → 添削のプロンプト → 外部AI → 貼り戻し → 改善案を保存 → Q&Aにも保存
- [ ] E2E-REV-002: 貼り戻しの直前にネット断 → 入力が保持され、復帰後に再試行できる

### 11.5.5 ES作成

- [ ] E2E-ES-001: 企業選択 → 資料を選んでドラフトのプロンプト → 外部AI → 貼り戻し → セクションに反映
- [ ] E2E-ES-002: 推敲の指示を加えたプロンプト → 貼り戻し → 本文が更新される
- [ ] E2E-ES-003: 文字数制限の超過時に警告が出る

### 11.5.6 共有リンク

- [ ] E2E-SH-001: セクションを選んで共有リンクを発行 → 別端末（ログインなし）のブラウザで表示できる
- [ ] E2E-SH-002: 失効 → 同じリンクが「無効」ページになる
- [ ] E2E-SH-003: 発行後に ES を編集しても、共有ページは発行時の内容のまま

### 11.5.7 復習（オフライン含む）

- [ ] E2E-REVW-001: Q&Aバンク → 復習UI → review_status の変更が反映される
- [ ] E2E-REVW-002: フィルタ（企業/未復習）で正しく絞り込める
- [ ] E2E-REVW-003: **機内モードで Q&A バンクを開き、復習できる**（閲覧のみ）
- [ ] E2E-REVW-004: お気に入り切替が反映される

### 11.5.8 未完了の AI連携の再開

- [ ] E2E-RES-UME-001: プロンプト作成 → 外部AIアプリに移動 → iOS がアプリを終了 → 再起動すると「未完了」に出る → 同じプロンプトを再表示して貼り戻せる
- [ ] E2E-RES-UME-002: 未完了のセッションを「中止」→ 一覧から消える

---

## 11.6 ネットワーク・端末状態テスト（実機で必須）

- M-NT-001: 低速回線でプロンプト作成・解析 → ローディング表示が崩れない
- M-NT-002: 機内モードでプロンプト作成・保存 → 即エラー、入力保持、再試行可能
- M-NT-003: 外部AIアプリとの往復中にバックグラウンド → 復帰時に状態が破綻しない
- M-NT-004: Neon のコールドスタート時 → キャッシュ済みのデータが先に表示される
- M-NT-005: 端末ロック中にキャッシュを読めない（Data Protection）

---

## 11.7 プロンプト回帰テスト（品質を壊さない）

### 11.7.1 テンプレートのゴールデンファイル

- 固定入力（A1〜A6 + C1(job=Backend) + R1/R2 + E1）から、各 use_case × バージョンのプロンプトを生成し、`testdata/golden/` のファイルと比較する
- テンプレートを意図して変更したときは、ゴールデンファイルを更新し、差分をレビューする
- ケース：
  - PR-001: 全 use_case × 現行バージョンがゴールデンファイルと一致する
  - PR-002: テンプレート変更の PR では、ゴールデンファイルの差分が必ず含まれる

### 11.7.2 実回答サンプルによる解析回帰

- ChatGPT・Gemini（無料版を含む）に実際にプロンプトを渡して得た回答を、`testdata/responses/<use_case>/<template_version>/` に fixture として保存する
  - fixture には実在の個人情報を入れない（共通テストデータから作ったプロンプトの回答のみ）
- 各 fixture に期待する parse_result と取得件数を `testdata/responses/expected.json` に記録し、テーブル駆動テストで検証する（記録のない fixture はテスト失敗）
  - プロンプトは共通テストデータから生成したゴールデンファイル（11.7.1）をそのまま外部AIに渡す。手順は `testdata/responses/README.md`
- ケース：
  - PR-003: 全 fixture で期待どおりの parse_result と件数になる
  - PR-004: 旧バージョン（例：`qa-v1`）の fixture は、新バージョン追加後も解析できる（D10）
  - PR-005: テンプレートを変更したら、新しいバージョンの fixture を ChatGPT と Gemini でそれぞれ最低 3 件追加する
- 実際の解析成功率は、運用データ（10.4.1）でも確認する

---

## 11.8 負荷・耐久（最小でもこれだけ）

- L-001: 1ユーザーが連続で prompt_sessions 作成→parse→commit を20回 → データ整合OK、二重保存0件
- L-002: 同じセッションへの commit を同時に10本 → 保存は1件のみ
- L-003: `/s/{token}` に短時間で大量アクセス → レート制限が効き、他のエンドポイントに影響しない

---

## 11.9 リリース判定（Go/No-Go）— チェック項目を固定

必須（1つでも落ちたらNo-Go）：

- G-001: user_id の越境が起きない（11.3.2 が全件成功）
- G-002: commit の二重保存が発生しない（IT-PS-010、IT-PS-011）
- G-003: プロンプト全文と貼り戻した回答の生データがサーバーに保存されない（IT-PS-014）
- G-004: 共有リンクの期限・失効・ヘッダが仕様どおり（11.3.4 が全件成功）
- G-005: プロンプト回帰テスト（11.7）が全件成功
- G-006: 主要フロー（11.5 のチェックリスト）が実機で完走する
- G-007: ログアウト・退会時に端末のキャッシュが削除される（IOS-CACHE-003、004）
- G-008: 重大クラッシュがない（Cloud Run のログと Xcode で確認）

---

## 11.10 テスト実装メモ（運用ルール）

- 自動テスト（Go の Unit / Integration、iOS の Unit）はCIで必須（mainへのマージ条件）
- 統合テストは PostgreSQL サービスコンテナで回し、Neon には接続しない
- 外部AIはテストから呼ばない。実回答は fixture として手動で収集する
- 失敗したテストは「再現手順・期待値・実際値・ログ（request_id）」を必ず残す
- Domain/Usecase が外部SDK（`gin`、`pgx`、`firebase`、`net/http` など）に依存していないことを CI（`backend/scripts/arch_check.sh`）とレビューで確認する（7_BackendDesign 7.2.1 準拠）

---

## 11.11 この章の凍結事項（合意）

- テンプレート生成と解析、認可（user_id 越境防止）、commit の冪等性、共有リンクは最優先でテストする
- 解析はテーブル駆動テストとし、use_case × バージョン、形式行の欠落、部分成功、崩れた入力を網羅する
- 外部AIの実回答は fixture として固定し、旧バージョンの形式も解析できることを回帰テストで保証する
- 本文データ（面接内容・プロンプト・外部AIの回答）はテストログや分析に出さない。fixture に実在の個人情報を入れない
