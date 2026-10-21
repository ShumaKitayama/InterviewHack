# 7. バックエンド / API設計（Go BFF + Neon + Firebase Auth）

> 改訂：2026-10-22。方針転換メモ（`DirectionChange_20261022.md`）の決定事項 D1〜D16 と 2.10 共通契約を反映した。
> 旧版の Supabase 直アクセス、Stripe 課金、クレジット消費（Reserve / Commit / Reversed）、OpenAI / Gemini 呼び出しは廃止した。

## 7.1 方針（アーキテクチャの前提）

- API：Go + Gin の BFF（Backend For Frontend）を Cloud Run で動かす（D3）
- DB：Neon（PostgreSQL）。BFF だけが接続する（D4）
- 認証：Firebase Auth。BFF は ID トークンを検証して user_id を確定する（D5）
- 認可：BFF のアプリ層で、全クエリを user_id でスコープする。RLS は使わない（D6）
- クライアント（iOS アプリ）は DB に直接アクセスしない。すべての読み書きは BFF 経由
- BFF の役割
  - ドメインデータの CRUD
  - AI連携のプロンプト生成と回答の解析・保存（D7, D10）。LLM の API は呼ばない
  - ES 共有リンクの発行・失効と、共有ページ（HTML）の配信（D9, D12）
- BFF は外部 AI サービスにも、任意の URL にも通信しない（企業研究資料の取り込みは iOS 側。D11）

---

## 7.2 バックエンド設計方針（DDD + クリーンアーキテクチャ）

本プロジェクトの BFF（Go + Gin）は、**DDD（ドメイン駆動設計）を意識したクリーンアーキテクチャ**を採用する。
目的は「ビジネスルールをフレームワーク/外部SDKから分離し、変更容易性とテスト容易性を最大化する」こと。

### 7.2.1 依存関係ルール（必須）

依存方向は内向き（外側→内側）に限定する：

- Domain（最内核）：ビジネスルール（エンティティ/値オブジェクト/ドメインサービス）
  - AI連携のテンプレート生成と回答の解析、共有トークンとスナップショットのルールも Domain に純粋関数として置く
- Application / Usecase：ユースケース（フロー制御、トランザクション境界、外部I/Fの呼び出し順序）
- Interface / Adapter：HTTPハンドラ（Gin）、DTO変換、入力バリデーション、エラー整形、共有ページの HTML 描画
- Infrastructure：DB（Neon / pgx）、Firebase（ID トークン検証）、ログ、設定

**禁止事項（重要）**

- Domain は `gin` / `pgx` / `firebase` 等の外部ライブラリ、および `net/http` 等のネットワーク・DB 関連パッケージを import してはならない
- Application/Usecase はフレームワーク依存（GinのContext等）を持たない
- Infrastructure 以外の層から「ネットワーク呼び出し」「DB直接操作」をしない
- 上記は `backend/scripts/arch_check.sh` で CI 検査する（14 章）

### 7.2.2 境界（DTO / Entity / Persistence Model）

- HTTP入力/出力は DTO として Interface/Adapter 層で定義する
- Domain は「永続化都合のフィールド」を持たない（必要なら別モデルに分離する）
- Repository/External Client の interface は `port` に定義し、実装は Infrastructure に置く
- 変換（DTO↔Usecase input↔Domain↔Persistence）は Adapter/Infra で行い、Domain を汚さない

### 7.2.3 Goのパッケージ構成

```
backend/
  cmd/api/               # main（Gin起動、DI）
  internal/
    domain/
      entity/
      vo/                # value objects（UseCase, TemplateVersion, SessionStatus など）
      error/             # domain error definitions
      service/
        prompt/          # テンプレート生成と解析（純粋関数、use_case × バージョン）
        share/           # 共有トークンとスナップショットのルール
    usecase/             # application layer
      account/ asset/ company/ research/ es/ interview/ qa/ dashboard/ prompt/ share/
    port/                # interfaces
      repository/
      external/          # AuthVerifier, TokenGenerator, Clock
    adapter/
      http/
        handler/
        middleware/      # 認証、X-Request-Id、構造化ログ、レート制限
        presenter/       # error/response shaping
        dto/
        sharepage/       # 共有ページの HTML テンプレート（html/template, embed）
    infra/
      postgres/          # pgx
      firebase/          # ID トークン検証
      logging/
      config/
```

備考：

- `internal/port` は「依存性逆転」のための interface 置き場（Repository / External client）
- `adapter/http` は Gin に依存してよいが、`domain` と `usecase` は依存しない
- `infra/*` は SDK/DB に依存してよい（最外層）
- 旧構成の `infra/supabase`、`infra/stripe`、`infra/llm`、`usecase/credit`、`usecase/billing`、`usecase/ai` は削除する

### 7.2.4 この設計で特に重要なユースケース

- AI連携の保存（`prompt/commit`）：セッションの状態遷移と本来のテーブルへの保存を、Usecase 層で 1 トランザクションとして扱う（7.6.1）
- 回答の解析（`prompt/parse`）：Domain の解析関数を呼び、何も保存せずプレビューを返す。解析ロジックは use_case × template_version ごとの純粋関数とする
- 共有リンクの発行（`share/create`）：トークン生成（`TokenGenerator`）とスナップショット作成を Usecase が統括し、DB にはハッシュだけを保存する（7.6.2）
- 退会（`account/delete`）：ユーザーのドメインデータ（共有リンクを含む）の物理削除と、`users` 行の `deleted_at` 設定・email / display_name の消去を 1 トランザクションで行う（6.3.3）。`users` 行は物理削除しない

---

## 7.3 コンポーネント構成

- Client（iOS アプリ。D1）
  - Firebase iOS SDK でログインし、ID トークンを取得する
  - すべての読み書きを BFF API で行う
  - 企業研究資料の本文抽出（WebView / PDFKit / Vision）は端末上で行い、抽出テキストを BFF に送る（D11）
  - 外部 AI サービスへの受け渡しは共有シート / クリップボード（D7）
- BFF（Go + Gin on Cloud Run）
  - CRUD API、AI連携 API、共有リンク API、共有ページ（HTML）
- Neon（PostgreSQL）
- Firebase Auth（ID トークンの発行元。BFF は Google の公開鍵で検証する）
- 共有ページの閲覧者（Web ブラウザ。認証なし）

### 7.3.1 データアクセス方針（BFF 経由に一本化）

- クライアントが DB に直接アクセスする経路はない（旧版の「Supabase 直 CRUD」と「BFF のみ書き込み」の区別は廃止）
- すべてのテーブルは BFF の Repository からのみ読み書きされ、全クエリは user_id でスコープされる（6.4）
- 唯一の例外は共有ページで、`share_links.token_hash` による検索だけを認証なしで行う

### 7.3.2 BFF が自ら書き込む領域（クライアントが値を指定できない項目）

- `users`：認証ミドルウェアが作成・email 同期を行う
- `prompt_sessions.status` / `parse_result` / `pasted_at` / `completed_at`：状態遷移は API 経由でのみ変わる（7.6.1）
- `share_links.token_hash` / `snapshot` / `expires_at`：BFF が生成する（7.6.2）

---

## 7.4 認証・認可（APIセキュリティ）

- クライアントは Firebase ID トークンを BFF に付与する
  - Header: `Authorization: Bearer <Firebase ID トークン>`
- BFF の認証ミドルウェア（`adapter/http/middleware`）
  1. ヘッダからトークンを取り出す。なければ 401 `UNAUTHORIZED`
  2. `port/external.AuthVerifier` で検証する（実装は `infra/firebase`。Firebase Admin SDK for Go、または同等の JWT 検証）
     - 署名：Google の公開鍵（キャッシュ可）
     - `aud` = `FIREBASE_PROJECT_ID`、`iss` = `https://securetoken.google.com/<FIREBASE_PROJECT_ID>`
     - `exp` / `iat` / `sub`（= Firebase `uid`）
  3. `firebase_uid` で `users` を取得し、なければ作成する（upsert）
  4. `users.deleted_at` が設定されていれば 401 `UNAUTHORIZED`（退会後の古い ID トークン。email の同期も行わない）
     - 有効なユーザーの場合だけ、email をトークンのクレームから同期する
  5. 確定した user_id を Usecase に渡す（Gin の Context に格納するのは Adapter 層まで）
- BFF はリクエストボディやクエリの user_id を信用しない
- 他人のリソースを指定された場合は 404 `NOT_FOUND` を返す
- 認証不要のエンドポイント：`GET /health`、`GET /s/{token}`

---

## 7.5 API一覧（エンドポイント設計）

共通：

- パスにバージョン接頭辞は付けない
- リクエスト / レスポンスは JSON（共有ページを除く）。フィールド名は camelCase
- 一覧 API はカーソル方式のページングとする：`?limit=50&cursor=...` → `{ items: [...], nextCursor: string|null }`
- ID は uuid。不正な形式は 400 `VALIDATION_ERROR`
- 失敗時のレスポンス形式とコードは 7.7 に従う

### 7.5.1 Health / Meta

- GET /health
  - Auth：不要
  - 200: `{ "status": "ok" }`

### 7.5.2 Account

- GET /me
  - 200: `{ id, email, displayName, createdAt }`
- PATCH /me
  - Req: `{ displayName? }`
  - 200: GET /me と同じ形
- DELETE /me（退会。6.3.3）
  - 204
  - 1 トランザクションで、ドメインデータ（共有リンクを含む）を即時に物理削除し、`users.deleted_at` を設定して email と display_name を消去する（`users` 行は残す）
  - Firebase のユーザー削除は、この API の成功後に iOS アプリが行う（BFF は管理者権限の認証情報を持たない）
  - 以後、同じユーザーの ID トークンでのリクエストは 401 `UNAUTHORIZED`

### 7.5.3 ドメインデータの CRUD

対象と主なフィルタ：

| リソース | パス | 主なフィルタ / 備考 |
|---------|------|--------------------|
| 自己資産 | `/assets` | `type`, `tagId`, `q`（キーワード）, `sort`（created / updated）。論理削除（deleted_at） |
| タグ | `/asset-tags` | ユーザー内で name は一意（重複は 409 `CONFLICT`） |
| 企業 | `/companies` | `q`, `selectionPhase`。論理削除（deleted_at） |
| 企業研究資料 | `/research-materials` | `companyId`（必須）。`contentRaw` は端末で抽出したテキスト |
| ESプロジェクト | `/es-projects` | `companyId`, `status`（`not_started` / `draft` / `in_review` / `completed` / `submitted`。作成時の初期値は `not_started`。遷移の制約はなく、PATCH で任意の値に変更できる。セクション本文の保存（PATCH `/es-sections/{id}` または commit）時に `not_started` なら `draft` にする） |
| ESセクション | `/es-projects/{id}/sections`（一覧・作成）, `/es-sections/{id}`（単体・更新・削除） | 本文の更新時に `version` を +1 する。PATCH では版を作らない |
| ESセクションの版 | `/es-sections/{id}/versions`（一覧・手動保存）, `/es-sections/{id}/versions/{versionId}/restore`（戻す） | 2.2.6.1。下記を参照 |
| 面接 | `/interviews` | `companyId`, 期間 |
| 面接ログ | `/interview-logs` | `interviewId` |
| Q&A | `/qa` | `companyId`, `q`, `isFavorite`, `reviewStatus` |

各リソースの操作：

- GET 一覧 → 200 `{ items, nextCursor }`
- POST → 201 作成したリソース
- GET 単体 → 200
- PATCH（部分更新）→ 200
- DELETE → 204（`assets` / `companies` は論理削除、それ以外は物理削除。ES プロジェクトの削除では、紐づく共有リンクも削除する。削除した対象を参照する `prompt_sessions` は、参照を null にして行を残す。8.8）
- 関連（自己資産のタグ、ES の参照資産・参照資料、Q&A の参照資産）は、親リソースの POST / PATCH で ID 配列（`tagIds`, `assetIds`, `researchMaterialIds`）として受け取り、BFF が関連テーブルを置き換える。指定 ID がすべて同じユーザーのものでなければ 404 `NOT_FOUND`
- 失敗：400 `VALIDATION_ERROR`、401 `UNAUTHORIZED`、404 `NOT_FOUND`、409 `CONFLICT`

ESセクションの版（2.2.6.1）：

- `GET /es-sections/{id}/versions`
  - Res（200）: `{ items: [{ id, version, source, content, createdAt }] }`（新しい順、最大20件）
- `POST /es-sections/{id}/versions`
  - 目的：「この版を保存」。今の `content_draft` を `source = manual` の版として残す
  - Res（201）: 作成した版
  - 失敗：422 `VALIDATION_ERROR`（本文が空）
- `POST /es-sections/{id}/versions/{versionId}/restore`
  - 目的：「この版に戻す」
  - 1 トランザクションで、今の `content_draft` を `manual` の版として残す → `content_draft` を選んだ版の `content` にする → `version` を +1 する
  - Res（200）: 更新後の ESセクション
- 版の作成時（手動保存・restore・AI連携の commit）は、同じトランザクションで、セクションごとに最新20件を超えた分を古い順に削除する
- 失敗：404 `NOT_FOUND`（他人のセクション・版を含む）

### 7.5.4 Dashboard

- GET /dashboard
  - 200:
    - `upcomingInterviews`：今日以降の面接（直近の数件）
    - `latestSummary`：`dashboard_summaries` の最新 1 件（なければ null）
    - `recentQa` / `recentInterviewLogs`：最近の数件
    - `pendingPromptSessions`：未完了の AI連携（`created` / `pasted`）の件数と直近の数件

### 7.5.5 AI連携（prompt_sessions）

use_case と template_version は 2.10.1 に従う。

- POST /prompt-sessions
  - 目的：AI連携のセッションを作成し、プロンプトを生成する
  - Req:
    - `useCase`：`qa_generate` | `answer_review` | `summary_generate` | `research_summarize` | `es_generate` | `es_refine`
    - 対象 ID（use_case ごとに必須のもの）
      - `qa_generate`：`companyId`
      - `answer_review`：`interviewLogId`
      - `summary_generate`：`interviewId`（任意。省略時は直近の面接ログを対象にする）
      - `research_summarize`：`companyId`、`researchMaterialIds`（1 件）
      - `es_generate` / `es_refine`：`esProjectId`、`esSectionId`
    - `assetIds`：string[]（プロンプトに含める自己資産。ユーザーが選ぶ）
    - `researchMaterialIds`：string[]（プロンプトに含める企業研究資料）
    - `options`（use_case ごと。例：`questionCount`（default 10）、`strictness`（`normal` | `hard`）、`instructions`（ES 推敲の指示文））
  - Res（201）:
    - `sessionId`、`templateVersion`、`promptText`
  - 副作用：`prompt_sessions` を `status = created` で作成する。`input_refs` には ID とオプションだけを保存し、プロンプト本文は保存しない
  - 失敗：400 `VALIDATION_ERROR`（必須の対象 ID がない、未知の useCase）、404 `NOT_FOUND`（対象や資産が他人のもの・存在しない）

- GET /prompt-sessions?status=created,pasted
  - 目的：未完了の AI連携の一覧（続きから再開）。「未完了」は `created` と `pasted` を指す
  - `status` は `created` / `pasted` / `saved` / `abandoned` を指定できる（カンマ区切りで複数指定可）
  - 200: `{ items: [{ sessionId, useCase, templateVersion, status, 対象の表示名, createdAt }], nextCursor }`
  - 一覧取得時に、作成から 7 日を過ぎた `created` / `pasted` のセッションを `abandoned` に更新する（7.9）

- GET /prompt-sessions/{id}/prompt
  - 目的：プロンプトの再表示
  - `input_refs` と現在のデータから、セッションの `template_version` でプロンプトを再生成する
  - 200: `{ sessionId, templateVersion, promptText }`
  - 失敗：404 `SESSION_NOT_FOUND`、409 `SESSION_STATE_CONFLICT`（`saved` / `abandoned`）

- POST /prompt-sessions/{id}/parse
  - 目的：貼り戻した回答の解析とプレビュー
  - Req: `{ responseText: string }`（最大 100,000 文字）
  - Res（200）:
    - `parseResult`：`success` | `partial` | `failed`
    - `detectedTemplateVersion`：回答内の形式行から検出したバージョン（なければセッションの値）
    - `preview`：use_case ごとの構造（下表）
    - `warnings`：string[]（例：「【A3】が見つかりません」）
  - 副作用：status を `pasted` に、`parse_result` と `pasted_at` を更新する。`responseText` と解析結果は保存しない
  - `created` と `pasted` のときに呼べる（再解析は可）
  - 解析しないユースケース（`summary_generate` / `es_generate` / `es_refine`）も、commit の前に必ず呼ぶ。形式行を取り除いた全文を返し、`parseResult = success` とする
  - 失敗：404 `SESSION_NOT_FOUND`、409 `SESSION_STATE_CONFLICT`、422 `VALIDATION_ERROR`（空文字、上限超過）
  - `parseResult = failed` でも 200 を返す（プレビュー上で手動修正するため）

| use_case | preview の構造 |
|----------|---------------|
| `qa_generate` | `{ items: [{ questionText, answerText, intentText }] }` |
| `answer_review` | `{ issues: string[], improvements: string[], improvedAnswerText }` |
| `research_summarize` | `{ highlights: string[], idealCandidate, recentTopics: string[] }` |
| `summary_generate` / `es_generate` / `es_refine` | `{ text }`（形式行を取り除いた全文） |

- POST /prompt-sessions/{id}/commit
  - 目的：ユーザーが確認・修正した内容を本来のテーブルに保存する
  - Req（use_case ごと。preview と同じ構造をユーザーが修正したもの）
    - `qa_generate`：`{ items: [{ questionText, answerText }], assetIds? }` → `qa` に `source_type = pre_generated`、`company_id` = セッションの企業で追加し、`qa_asset_relations` を作る。プレビューの `intentText`（`【意図N】`）は保存しない
    - `answer_review`：`{ feedbackText, improvedAnswerText, saveToQa: bool }` → `interview_logs.ai_feedback_text` / `ai_improved_answer_text` を更新する。`saveToQa` が true なら `qa` に `source_type = review_improved` で追加する
    - `research_summarize`：`{ summaryText }` → 対象の `research_materials.summary` を更新する
    - `summary_generate`：`{ text }` → `dashboard_summaries` に追加する（`source_interview_id` = セッションの面接）
    - `es_generate` / `es_refine`：`{ text }` → 同じトランザクションで、上書き前の `content_draft` を `es_section_versions` に残し（`source` = use_case。本文が空なら残さない。最新20件を超えた分は削除）、対象の `es_sections.content_draft` を更新し、`version` を +1 する。親の `es_projects.status` が `not_started` なら、同じトランザクションで `draft` にする（2.2.5）
  - Res（200）: `{ sessionId, status: "saved", savedRefs: { 種別: ID[] } }`
  - 副作用：status を `saved` に、`completed_at` を更新する（7.6.1）
  - `pasted` のときだけ呼べる（`created` からの直接の commit は 409 `SESSION_STATE_CONFLICT`）
  - 失敗：404 `SESSION_NOT_FOUND`、409 `SESSION_STATE_CONFLICT`（`created`、`saved` 済み、`abandoned` を含む）、422 `VALIDATION_ERROR`、503 `FEATURE_DISABLED`（セッションの use_case が `FEATURE_COMMIT_DISABLED_USE_CASES` に含まれる。状態は `pasted` のまま変えない）

- POST /prompt-sessions/{id}/abandon
  - 目的：AI連携を中止する
  - 200: `{ sessionId, status: "abandoned" }`
  - `created` / `pasted` のときだけ呼べる。失敗：404 `SESSION_NOT_FOUND`、409 `SESSION_STATE_CONFLICT`

### 7.5.6 共有リンク

- POST /es-projects/{id}/share-links
  - Req: `{ sectionIds: string[]（1 件以上）, expiresInDays: 1 | 7 | 30, showCompanyName?: bool（省略時 false） }`
  - Res（201）: `{ id, url, expiresAt }`
    - `url` = `SHARE_BASE_URL` + `/s/` + 生トークン。生トークンを返すのはこの応答だけ
  - 副作用：選択されたセクションの title と本文（`content_final`、なければ `content_draft`）、および `showCompanyName` が true なら企業名をスナップショットとして保存する
  - 失敗：404 `NOT_FOUND`（プロジェクトやセクションが他人のもの・存在しない、セクションが別プロジェクトのもの）、422 `VALIDATION_ERROR`（期限が 1 / 7 / 30 以外、セクションが空）、503 `FEATURE_DISABLED`（`FEATURE_SHARE_CREATE` が `off`）

- GET /es-projects/{id}/share-links
  - 200: `{ items: [{ id, sectionTitles, showCompanyName, expiresAt, revokedAt, createdAt, isActive }] }`（URL とトークンは返さない）

- DELETE /share-links/{id}
  - 目的：失効
  - 204。`revoked_at` を設定する（失効済みでも 204）
  - 失敗：404 `NOT_FOUND`

- GET /s/{token}（共有ページ）
  - Auth：不要
  - BFF はトークンの SHA-256 ハッシュで `share_links` を検索し、`adapter/http/sharepage` の `html/template` でスナップショットを描画する
  - レスポンスヘッダ：`Content-Type: text/html; charset=utf-8`、`X-Robots-Tag: noindex`、`Cache-Control: no-store`、`Referrer-Policy: no-referrer`、`Content-Security-Policy: default-src 'none'; style-src 'unsafe-inline'; img-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'`（スクリプト・画像・フォーム・外部リソース・他サイトへの埋め込みをすべて禁止する。CSS はページ内に埋め込む）。410 / 404 / 503 の HTML にも同じヘッダを付ける
  - HTML にも `<meta name="robots" content="noindex">` を入れる。ユーザー名は表示しない
  - 200：有効なリンク
  - 410：期限切れまたは失効（「このリンクは無効です」の HTML。`code = SHARE_LINK_GONE` をログに記録する）
  - 404：存在しない（410 と見分けがつかない汎用の HTML にする）
  - 503：フィーチャーフラグ `FEATURE_SHARE_VIEW` が `off`（「一時停止中」の HTML。`code = FEATURE_DISABLED` をログに記録する。12.11）
  - IP 単位のレート制限の対象（7.8）

---

## 7.6 整合性設計（重要）

### 7.6.1 AI連携セッションの状態遷移と冪等な保存

状態遷移：

```
created ──parse──▶ pasted ──commit──▶ saved
   │                 │ ▲
   │                 └─┘ parse（再解析）
   └──abandon / 7日経過──▶ abandoned ◀──abandon / 7日経過── pasted
```

- 許可する遷移は `created → pasted`（parse）、`pasted → pasted`（再解析）、`pasted → saved`（commit）、`created / pasted → abandoned`（abandon、または作成から 7 日経過）だけとする。`created → saved`（parse を経ない commit）は禁止する
- `saved` と `abandoned` は終端状態とする
- 状態遷移は必ず条件付き UPDATE で行う
  - 例（commit）：`UPDATE prompt_sessions SET status = 'saved', completed_at = now() WHERE id = $1 AND user_id = $2 AND status = 'pasted'`
  - 更新件数が 0 件なら、セッションの有無と現在の status を確認し、404 `SESSION_NOT_FOUND` または 409 `SESSION_STATE_CONFLICT` を返す
- commit では、状態遷移と本来のテーブルへの保存を **同一トランザクション** で行う
  - 同じセッションへの commit が同時に来ても、条件付き UPDATE で 1 件だけが成功し、二重保存は起きない
  - クライアントは、commit の 409 を受けたら GET でセッションを確認し、`saved` であれば保存済みとして扱う（再送で重複しない）
- parse は保存を伴わないため、何度呼んでもよい（`parse_result` と `pasted_at` は最新の値で上書き）

### 7.6.2 共有リンクのトークン

- `TokenGenerator`（`port/external`）が暗号学的乱数で 32 バイト（256 ビット）を生成し、base64url で表現する（D12：128 ビット以上）
- DB には SHA-256 ハッシュだけを `token_hash`（unique）に保存する。生トークンは保存もログ出力もしない
- 有効判定は Domain（`service/share`）の純粋関数：`revoked_at IS NULL かつ now < expires_at`
- スナップショットは発行時点の内容で固定する。以後のセクション編集・削除は共有ページに反映しない
- ES プロジェクトを削除したら、紐づく共有リンクも削除する
- 期限切れ・失効したリンクは、運用で定期的に物理削除してよい（12 章）

---

## 7.7 エラー設計（API共通）

### 7.7.1 エラーレスポンス形式（統一）

- status: "error"
- code: string（機械可読）
- message: string（ユーザー向け文言）
- detail?: object（開発者向け）

### 7.7.2 HTTPステータスとエラーコード対応表

方針：

- HTTPステータスは「通信/権限/リソース状態」を表す
- `code` はアプリケーション固有の機械可読エラー種別
- クライアントは `status` と `code` の両方で分岐して良い

対応表：

| HTTP | code | 主な発生箇所 |
|------|------|-------------|
| 400 | `VALIDATION_ERROR` | 形式不正（JSON、uuid、必須項目の欠落） |
| 401 | `UNAUTHORIZED` | トークンなし・無効・期限切れ、退会済みユーザー |
| 403 | `FORBIDDEN` | 予約（MVP では使わない。他人のリソースは 404） |
| 404 | `NOT_FOUND` | リソースが存在しない、または他人のもの |
| 404 | `SESSION_NOT_FOUND` | AI連携セッションが存在しない、または他人のもの |
| 409 | `CONFLICT` | 一意制約違反（タグ名の重複など） |
| 409 | `SESSION_STATE_CONFLICT` | セッションの状態が操作を許さない（保存済みへの commit など） |
| 410 | `SHARE_LINK_GONE` | 共有リンクの期限切れ・失効（共有ページは HTML で返す） |
| 422 | `VALIDATION_ERROR` | 値の範囲・業務ルール違反（期限の値、文字数上限など） |
| 429 | `RATE_LIMITED` | レート制限（7.8） |
| 500 | `INTERNAL_ERROR` | 想定外のエラー |
| 503 | `FEATURE_DISABLED` | フィーチャーフラグで機能を停止中（12.11）。共有ページは「一時停止中」の HTML で返す |

廃止したコード：`INVALID_PACKAGE`、`DUPLICATE_EVENT`、`INSUFFICIENT_CREDITS`、`OPENAI_ERROR`、`STRIPE_ERROR`

運用ルール：

- クライアントに表示する文言は `message`、分岐は `code` を使用する
- 500 の `message` には内部情報を含めない。詳細はログ（`request_id` で突合）に残す

### 7.7.3 リクエストID（相関ID）

- BFFは全リクエストに対して `request_id`（UUID推奨）を生成し、以後のログ/レスポンスに一貫して付与する
- 受信ヘッダに `X-Request-Id` があればそれを採用し、なければサーバー側で生成する
- すべてのレスポンスヘッダに `X-Request-Id: <request_id>` を付与する（共有ページを含む）
- 例外/エラー時も必ず同じ `request_id` を返す（クライアントとログの突合のため）
- iOS アプリはリクエストごとに `X-Request-Id` を生成して付与する（5 章）

---

補足（テスト方針）：

- Domain / Usecase は外部I/Fをinterface化し、モックで純粋ユニットテスト可能にする
  - テンプレート生成と回答の解析は、use_case × template_version のテーブル駆動テストの対象とする
- Adapter/Infra は統合テスト（PostgreSQL のサービスコンテナ）で担保する
- 最終的なテスト戦略は `DesignDocument/11_TestDesign.md` を正とする

## 7.8 レート制限（乱用防止）

- 目的はコスト防御ではなく、乱用・総当たりの防止とする（LLM の API を呼ばないため。D7）
- 認証付き API：user_id 単位で 1 分あたり 120 回
- 共有ページ `GET /s/{token}`：IP 単位で 1 分あたり 30 回。トークンの総当たりを防ぐ
- 実装：Cloud Run のインスタンスごとのメモリ内カウンタ（フェーズ1 の規模で十分。インスタンス間で共有しない）
- 429 時は `Retry-After` を返す

## 7.8.1 構造化ログ（Cloud Run向け）

目的：

- API の失敗、AI連携の解析失敗、共有ページへのアクセス異常を `request_id` で追跡し、再現なしで原因特定できる状態にする

方針：

- Cloud Runの標準出力（stdout/stderr）にJSON形式でログを出力し、Cloud Loggingで収集する
- ログは「機械可読（JSON）」「PIIを含めない」「相関可能（request_id）」を必須要件とする

ログ共通フィールド（全リクエスト）：

- `severity`: "INFO" | "WARN" | "ERROR"
- `timestamp`: RFC3339
- `request_id`: 7.7.3 の相関ID
- `method`: HTTPメソッド
- `path`: ルートのパターン（例："/prompt-sessions/:id/parse"。実 ID やトークンは含めない）
- `status`: HTTPステータス
- `latency_ms`: レスポンスまでの処理時間
- `user_id`: トークン検証後に取得できる場合のみ（未認証/検証失敗時は出さない）
- `error_code`: 7.7 の `code` と一致（成功時はnull）

AI連携ログ追加フィールド：

- `session_id`: prompt_sessions.id
- `use_case`
- `template_version` / `detected_template_version`
- `parse_result`: success | partial | failed
- `response_length`: 貼り戻した回答の文字数（本文は出さない）

共有リンクログ追加フィールド：

- `share_link_id`: 判明した場合のみ
- `share_result`: ok | gone | not_found

禁止（ログに絶対出さない）：

- 面接ログ/自己資産/志望動機/ES/企業研究資料などの本文、プロンプト本文、貼り戻した回答の本文
- Authorizationヘッダ、ID トークン、共有リンクの生トークン（URL パスを含む）
- メールアドレス等の個人情報（必要ならハッシュ化/マスキング）

運用：

- 障害時の一次調査は `request_id` を起点に「BFFログ → prompt_sessions / 対象テーブル」へ辿る
- テンプレート品質の確認は、DB の `prompt_sessions` の集計で行う（10 章。D15）

---

## 7.9 AI連携セッション管理（prompt_sessions）

- 記録する項目（詳細は 8 章 / 2.10.2）
  - user_id、use_case、template_version
  - 対象の参照（company_id / es_project_id / es_section_id / interview_id / interview_log_id）
  - input_refs（選んだ自己資産・資料の ID とオプションのみ）
  - status、parse_result
  - created_at、pasted_at、completed_at
- 保存しないもの：プロンプトの全文、貼り戻した回答の生データ（D13）
- 放置されたセッション：作成から 7 日を過ぎた `created` / `pasted` は `abandoned` とみなす
  - 定期ジョブは持たず、一覧取得時（`GET /prompt-sessions`、`GET /dashboard`）に条件付き UPDATE で更新する
- 目的
  - 「続きから貼り戻す」再開
  - テンプレートのバージョンごとの解析成功率・離脱率の計測（D15）
- 旧 `generation_jobs` が担っていたコスト分析・クレジット紐付け・不正検知は廃止した

---

## 7.10 データフロー（要約）

### 7.10.1 事前対策（想定Q&A）

- Client: 企業と自己資産を選択 → BFF `POST /prompt-sessions`（`qa_generate`）
- BFF: 企業・自己資産・資料を取得 → Domain でプロンプト生成 → セッション作成 → `promptText` を応答
- Client: 共有シートで外部 AI に渡す → 回答をコピーして戻る → BFF `POST /prompt-sessions/{id}/parse`
- BFF: Domain で解析 → プレビューを応答（保存なし）
- Client: プレビューを確認・修正 → BFF `POST /prompt-sessions/{id}/commit`
- BFF: 1 トランザクションで `qa` 追加と status `saved` → 応答

### 7.10.2 事後対策（添削）

- Client: 面接ログを保存（`POST /interview-logs`）→ BFF `POST /prompt-sessions`（`answer_review`、`interviewLogId`）
- 以降は 7.10.1 と同じ流れ。commit で `interview_logs` の添削結果を更新し、必要なら `qa` に改善後の回答を追加する

### 7.10.3 ES 作成（ドラフト・推敲）

- Client: ES セクションを選択 → BFF `POST /prompt-sessions`（`es_generate` / `es_refine`）
- 外部 AI の回答は解析せず全文をプレビューし、commit で `es_sections.content_draft` を更新する

### 7.10.4 企業研究資料の取り込みと要約

- Client: WebView / PDFKit / Vision で本文を抽出 → プレビューで編集 → BFF `POST /research-materials`（`contentRaw`）
- 要約する場合は `research_summarize` で AI連携を行い、commit で `research_materials.summary` を更新する

### 7.10.5 共有リンク

- Client: セクション・期限・企業名表示を選択 → BFF `POST /es-projects/{id}/share-links` → URL を共有シートで送る
- 閲覧者: ブラウザで `GET /s/{token}` → BFF がスナップショットを HTML で返す

---

## 7.11 凍結ポイント（この章の合意事項）

- クライアントは DB に直接アクセスしない。すべて Go BFF 経由
- Firebase ID トークンで BFF 認証し、user_id は検証済みトークンから確定したものだけを信用する（Auth は Email/Password 先行）
- 認可は BFF のアプリ層で、全クエリを user_id でスコープする（RLS は使わない）
- BFF は LLM の API・外部 AI サービス・任意の URL に通信しない
- AI連携の保存は、条件付き UPDATE による状態遷移と本来のテーブルへの保存を同一トランザクションで行い、二重保存を防ぐ
- 共有リンクの DB にはトークンのハッシュだけを保存し、共有ページは noindex・no-store で配信する
- BFFはDDD + クリーンアーキテクチャの依存方向（Domainに外部依存を入れない）を遵守する
