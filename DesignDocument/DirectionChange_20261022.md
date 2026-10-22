# 方針転換メモ（2026-10-22）

本メモは、設計書 0〜14 を改訂する前に「何を・なぜ変えるか」を記録するものである。
改訂が完了するまでは、本メモの決定事項が既存設計書より優先する。
各設計書の改訂が完了したら、該当行の「状態」を更新する。

---

## 1. 背景（なぜ変えるか）

- **運営コストを可能な限り 0 円にしたい**
  - LLM API（OpenAI / Gemini）の従量課金が、Free ユーザー制限・クレジット・Stripe 課金という重い設計の根本原因になっていた
- **API キーの設定・管理をしたくない**
  - 運営側に LLM のシークレットを持たないことで、コスト事故・漏洩リスク・運用負荷を同時に消す
- **Supabase をやめたい**
  - 無料枠は一定期間アクセスがないと一時停止し、手動復旧が必要
  - RLS / Supabase Auth / Supabase 直アクセスと BFF 経由の二重境界が設計を複雑にしていた
- **Web（PWA）よりスマホアプリで使いたい**
  - 主利用シーンはスマホ（0.11 / 1.2.7 の前提とも一致）
  - iOS の共有シートで ChatGPT / Gemini アプリに渡せるため、コピペ方式の手間が小さい

---

## 2. 決定事項

| # | 項目 | 旧 | 新 |
|---|------|----|----|
| D1 | クライアント | Vite + React PWA（Cloudflare Pages） | **iOS ネイティブ（SwiftUI）**。Android は対象外 |
| D2 | 配布 | 一般公開（ストア） | **当面は開発者本人の iPhone のみ**（無料 Apple ID / Xcode 直接インストール） |
| D3 | API | Go + Gin BFF（Cloud Run） | **継続**（Go + Gin on Cloud Run） |
| D4 | DB | Supabase Postgres | **Neon（PostgreSQL）** |
| D5 | 認証 | Supabase Auth（Google + Email/Password） | **Firebase Auth**（Email/Password を先に、Google は追加候補） |
| D6 | 認可 | Supabase RLS ＋ BFF | **Go BFF でのアプリ層認可**（全クエリを user_id でスコープ） |
| D7 | LLM | OpenAI / Gemini API をサーバーから呼ぶ | **API を呼ばない。プロンプト生成＋コピペ方式** |
| D8 | 課金 | Stripe Checkout ＋ クレジット（Reserve / Commit / Reversed） | **廃止**。収益化は将来課題（行うなら StoreKit） |
| D9 | 共有リンク | 1.1.4 に記載（1.3.2 では MVP 外） | **MVP に含める**。閲覧ページは Go が HTML を返す |
| D10 | 貼り戻し回答の解析（旧 O1） | — | **目印付きテキスト形式＋プレビューでの手動修正**。解析は Go BFF で行う（2.3 参照） |
| D11 | 企業研究資料の取り込み（旧 O2） | Go でスクレイピング、PDF 保存 | **iOS 側で取り込み、テキストのみ保存**（2.4 参照） |
| D12 | 共有リンクの仕様（旧 O3） | — | **ES プロジェクト単位のスナップショット、ログイン不要の閲覧のみ、期限付き**（2.5 参照） |
| D13 | `generation_jobs`（旧 O4） | AI 生成ジョブ履歴（コスト・クレジット紐付け） | **廃止し、`prompt_sessions` として作り直す**（2.6 参照） |
| D14 | 収益化（旧 O5） | Free / Pro、クレジット購入 | **MVP では収益化しない。プランの概念も作らない**（2.7 参照） |
| D15 | 分析（旧 O6） | プロダクト / ビジネス / AI 運用の 3 系統 KPI、イベント計測 | **フェーズ1は DB 集計のみ。SDK・ダッシュボードなし**（2.8 参照） |
| D16 | オフライン（旧 O7） | 「ネット接続が必要」画面のみ、API はキャッシュしない | **読み取りキャッシュ＋下書きの端末保存**（2.9 参照） |

### 2.1 LLM コピペ方式の概要（D7）

1. アプリが「自己資産（セット1）＋企業情報（セット2）＋ユースケース別テンプレート」からプロンプトを組み立てる
2. ユーザーが iOS の共有シート / クリップボードで ChatGPT・Gemini 等（ユーザー自身のアカウント・無料版可）に渡す
3. 回答をアプリに貼り戻す
4. アプリが回答を解析し、Q&A・ES ドラフト・添削結果として保存する

- 運営側の LLM コスト・API キー・レート制限・クレジット管理は不要になる
- 将来の拡張候補：端末上の LLM（Apple Foundation Models）による軽量タスクのワンタップ化

### 2.3 貼り戻し回答の解析（D10）

- 構造化が必要なユースケースだけを解析する
  - 必要：想定 Q&A 生成（4.2.1）、面接ログ添削（4.2.2）、企業研究要約（4.2.4）
  - 不要（全文をそのまま保存）：ダッシュボード要約（4.2.3）、ES ドラフト（4.2.5）、ES 推敲（4.2.6）
- 出力形式：人が読める目印付きテキスト（例：`【Q1】` `【A1】` `【意図】`）。JSON は使わない
  - 無料版の LLM でも崩れにくく、LLM の画面上でもそのまま読め、手で直しやすいため
- プロンプトに形式バージョン（例：`【形式:qa-v1】`）を埋め込み、テンプレート変更後も古い回答を解析できるようにする
- フロー：貼り戻し → Go BFF が解析してプレビューを返す → ユーザーが確認・手動修正 → 保存
- 解析に失敗した部分は、プレビュー画面で手動修正できる
- テンプレート生成と解析は、Go の Domain 層に純粋関数として置く（テーブル駆動テストの対象）

### 2.4 企業研究資料の取り込み（D11）

- URL：iOS アプリ内の WebView でページを開き、表示された本文を抽出する
  - 抽出した本文はプレビューで編集してから保存する（ヘッダーや広告などのノイズはユーザーが除去できる）
  - サーバー側では URL を取得しない（SSRF 対策が不要になり、Go BFF は CRUD に専念する）
  - 将来の拡張候補：Safari 共有拡張からの取り込み（無料アカウントでの動作は要確認）
- PDF：iOS の PDFKit で端末上でテキストを抽出する
- スキャン PDF / 画像：iOS の Vision で端末上で OCR する
- ファイル本体は保存せず、抽出テキスト（`content_raw`）だけを保存する。オブジェクトストレージは使わない
- スコープ（0.12 の改訂方針）：自動クローリングは行わない。ユーザー操作による 1 ページ単位の取り込みは許可する
- プロンプトに使う資料はユーザーが選ぶ。長い資料は、先に要約（4.2.4）して `summary` を使う流れを推奨する

### 2.5 共有リンク（D12）

- 単位：ES プロジェクト単位。作成時に、公開するセクションを選択できる
- 内容：作成時点のスナップショット（以後の編集は反映しない）。内容を更新したい場合はリンクを作り直す
- 閲覧：リンクを知っていれば、ログインなしで閲覧できる。閲覧のみで、コメント機能は MVP 外（将来の拡張候補）
- トークン：128 ビット以上の暗号学的乱数を使う。DB にはハッシュのみを保存する
- 閲覧ページ：`noindex` を付け、キャッシュさせない。ユーザー名は表示しない。企業名の表示は作成時に選択する
- 有効期限：1日 / 7日 / 30日 から選択（初期値 7日）。無期限は不可。アプリからいつでも失効できる
- 閲覧ページは Go BFF が HTML を返す（D9）

### 2.6 prompt_sessions（D13）

- 目的
  - 「続きから貼り戻す」：ChatGPT 等との往復中にアプリが終了しても、未完了のプロンプトから再開できる
  - テンプレート品質の計測：テンプレートのバージョンごとの解析成功率・離脱率（O6 の材料）
- 項目（案）
  - id, user_id
  - use_case（qa_generate / answer_review / research_summarize / es_generate / es_refine / summary_generate）
  - template_version（例：qa-v1）
  - 対象の参照（company_id / es_section_id / interview_log_id など）
  - status（created → pasted → saved / abandoned）
  - parse_result（success / partial / failed）
  - created_at, completed_at
- 保存しないもの
  - プロンプトの全文：機密データを含むうえ、元データから再生成できるため
  - 貼り戻した回答の生データ：保存した時点で本来のテーブル（qa など）に入るため（後からの再解析はできない）
- 一定期間 `created` のままのものは `abandoned` とみなす
- 旧設計で担っていたコスト分析・クレジット紐付け・不正検知は、役割ごと消える

### 2.7 収益化と事業ゴール（D14）

- MVP では収益化しない。Free / Pro の判定や利用回数制限の仕組みも作らない（使わない仕組みを先に作らない）
- 将来の方針：収益化するなら StoreKit を使う。運営費がほぼかからないため、サブスクより買い切りの Pro 解放（例：企業登録数の無制限化）が候補
- 検討を始めるタイミング：利用者が増え、Cloud Run / Neon の無料枠を超えそうになったとき
- 事業ゴール（0.3 の改訂方針）

| 段階 | ゴール |
|------|--------|
| フェーズ1 | 自分の就活・転職で実際に使い切る（ES と面接対策の両方） |
| フェーズ2 | Apple Developer Program に加入し、TestFlight で 10 人前後に使ってもらう |
| フェーズ3 | App Store で公開し、必要なら収益化を検討する |

### 2.8 分析（D15）

- フェーズ1：分析用の SDK は入れない。DB から集計し、集計用の SQL を設計書に記載する。ダッシュボードは作らない
  - テンプレートのバージョンごとの解析成功率（`prompt_sessions.parse_result`）
  - 離脱率（`prompt_sessions.status = abandoned` の割合）
  - 週ごとの Q&A 保存数、復習の回数、面接ログの件数
  - エラーは Cloud Run の構造化ログと Xcode のデバッグで確認する
- フェーズ2：新規登録 → 自己資産の登録 → 初回プロンプト作成 → 初回 Q&A 保存 のファネルと、D7 リテンションを追加する。Firebase Analytics / Sentry の導入はその時点で検討する
- 全フェーズ共通（10.7 を継続）：本文データは分析に使わない。ログに PII を残さない
- 廃止：ビジネス KPI（クレジット）、AI コスト KPI、運営者向けダッシュボード

### 2.9 オフライン（D16）

- 最後に取得した自分のデータ（自己資産・企業・ES・Q&A）を SwiftData にキャッシュし、オフラインでも閲覧・復習できる
  - 主な利用シーン：面接直前の電車や会場での Q&A 見直し。Neon のコールドスタート時の体感改善も兼ねる
- 書き込みはオンライン時のみ
  - 例外：入力途中の下書き（面接ログ、編集中の ES）は端末に保存し、オンライン復帰後にユーザーの操作で送信する（自動送信はしない）
  - 復習済みチェックは MVP ではオンライン時のみ
- プロンプトの生成・解析は Go BFF で行うため、オンラインが必要
- 書き込みを含むオフライン同期（衝突解決）は行わない
- セキュリティ
  - iOS Data Protection（端末ロック中は読めない保護クラス）を使う
  - ログアウト・退会時にキャッシュをすべて削除する
  - 認証トークンは Keychain に置く（Firebase SDK が管理）

### 2.10 章をまたぐ共通契約（改訂時の基準）

各章の改訂は、以下の名前・形に揃える。変更する場合は、まずこの節を更新する。

#### 2.10.1 ユースケースとテンプレートのバージョン

| use_case | 内容 | 初期 template_version | 回答の解析 |
|----------|------|----------------------|-----------|
| `qa_generate` | 想定Q&A生成（4.2.1） | `qa-v1` | する |
| `answer_review` | 面接ログ添削（4.2.2） | `review-v1` | する |
| `summary_generate` | ダッシュボード要約（4.2.3） | `summary-v1` | しない（全文） |
| `research_summarize` | 企業研究資料の要約（4.2.4） | `research-v1` | する |
| `es_generate` | ESドラフト（4.2.5） | `esdraft-v1` | しない（全文） |
| `es_refine` | ES推敲（4.2.6） | `esrefine-v1` | しない（全文） |

目印付きテキストの形式（AI には、回答の 1 行目に形式行を出すよう指示する）：

```
【形式:qa-v1】
【Q1】質問文
【A1】模範解答
【意図1】質問の意図
【Q2】...
```

```
【形式:review-v1】
【指摘】
- 指摘1
【改善案】
- 改善案1
【改善後の回答】
改善後の回答文
```

```
【形式:research-v1】
【魅力・特徴】
- 3点
【求める人物像】
本文
【最近のトピック】
- 箇条書き
```

- 解析時は、回答内の形式行のバージョンを優先する。形式行がなければ、セッションの template_version を使う
- 解析しないユースケースは、形式行があれば取り除いて全文を取り込む

#### 2.10.2 テーブル（Neon / PostgreSQL）

- 新設
  - `users`：id (uuid, PK), firebase_uid (text, unique), email, display_name, created_at, updated_at, deleted_at。旧 `auth_users` と `profiles` を統合する。全テーブルの user_id はここを参照する
  - `prompt_sessions`：id, user_id, use_case, template_version, company_id / es_project_id / es_section_id / interview_id / interview_log_id（すべて null 可）, input_refs (jsonb：選んだ自己資産や資料の ID とオプションのみ。本文は入れない), status (`created` / `pasted` / `saved` / `abandoned`), parse_result (`success` / `partial` / `failed`, null 可), created_at, pasted_at, completed_at
  - `share_links`：id, user_id, es_project_id, token_hash (unique), snapshot (jsonb：公開するセクションの title と本文、および企業名を表示する場合は企業名), show_company_name (bool), expires_at, revoked_at, created_at
  - `dashboard_summaries`：id, user_id, content, source_interview_id (null 可), created_at（`summary_generate` の取り込み先。ダッシュボードには最新の1件を表示する）
- 継続（参照先を `users.id` に変更し、RLS の注記を削除する）
  - `assets`, `asset_tags`, `asset_tag_relations`, `companies`, `research_materials`, `es_projects`, `es_sections`, `es_project_asset_relations`, `es_project_research_material_relations`, `interviews`, `interview_logs`, `qa`, `qa_asset_relations`
  - `research_materials.content_raw` は「端末で抽出したテキスト」、`summary` は「AI連携で取り込んだ要約」とする
  - `interview_logs.ai_feedback_text` / `ai_improved_answer_text` は、カラム名を変えずに「AI連携で取り込んだ結果」とする
- 廃止：`auth_users`, `profiles`, `generation_jobs`, `credit_balances`, `credit_transactions`, `stripe_events`, `credit_packages`, `credit_costs`
- RLS は使わない。認可は Go BFF の Repository で、全クエリに `user_id = $1` を付けて行う（D6）
- マイグレーションは、冪等な SQL を番号順に `psql` で適用する（`backend/migrations`）。新しい構成用に 001 から作り直す

#### 2.10.3 API（Go BFF）

- 認証：`Authorization: Bearer <Firebase ID トークン>`。ミドルウェアで検証し（Google の公開鍵、`aud` = Firebase プロジェクト ID）、`firebase_uid` から `users` を取得または作成して `user_id` を確定する。リクエストボディの user_id は信用しない
- 相関 ID：`X-Request-Id` を継続する（7.7.3）
- エンドポイント（パスにバージョン接頭辞は付けない）
  - `GET /health`
  - `GET /me`, `DELETE /me`（退会）
  - CRUD（GET 一覧 / POST / GET 単体 / PATCH / DELETE）：`/assets`, `/asset-tags`, `/companies`, `/research-materials`, `/es-projects`, `/es-projects/{id}/sections`, `/es-sections/{id}`, `/interviews`, `/interview-logs`, `/qa`
  - `GET /dashboard`
  - AI連携
    - `POST /prompt-sessions`：{ useCase, 対象 ID, assetIds, researchMaterialIds, options } → { sessionId, templateVersion, promptText }
    - `GET /prompt-sessions?status=created,pasted`：未完了一覧（続きから再開。`status` はカンマ区切りで複数指定できる）
    - `GET /prompt-sessions/{id}/prompt`：`input_refs` からプロンプトを再生成する（全文は保存しない）
    - `POST /prompt-sessions/{id}/parse`：{ responseText } → { parseResult, preview, warnings }。status を `pasted` にする。何も保存しない
    - `POST /prompt-sessions/{id}/commit`：{ ユーザーが確認・修正した内容 } → 本来のテーブルに保存し、status を `saved` にする。status 遷移は条件付き UPDATE で行い、二重保存を防ぐ（冪等性）
    - `POST /prompt-sessions/{id}/abandon`
  - 共有リンク
    - `POST /es-projects/{id}/share-links`：{ sectionIds, expiresInDays (1 / 7 / 30), showCompanyName } → { url, expiresAt }。生トークンを返すのはこの応答だけ
    - `GET /es-projects/{id}/share-links`, `DELETE /share-links/{id}`（失効）
    - `GET /s/{token}`：認証不要。HTML を返す。`X-Robots-Tag: noindex` と `Cache-Control: no-store` を付ける。期限切れ・失効は 410、存在しなければ 404
- エラー形式は 7.7.1 を継続する。廃止するコード：`INVALID_PACKAGE`, `DUPLICATE_EVENT`, `INSUFFICIENT_CREDITS`, `OPENAI_ERROR`, `STRIPE_ERROR`。追加するコード：`SESSION_NOT_FOUND`(404), `SESSION_STATE_CONFLICT`(409), `SHARE_LINK_GONE`(410)
- レート制限：コスト防御が目的ではなくなる。乱用防止として残す（`/s/{token}` は IP 単位、それ以外は user_id 単位）

#### 2.10.4 Go パッケージ構成

```
backend/
  cmd/api/
  internal/
    domain/
      entity/ vo/ error/
      service/
        prompt/        # テンプレート生成と解析（純粋関数、use_case × バージョン）
        share/         # 共有トークンとスナップショットのルール
    usecase/
      account/ asset/ company/ research/ es/ interview/ qa/ dashboard/ prompt/ share/
    port/
      repository/
      external/        # AuthVerifier, TokenGenerator, Clock
    adapter/http/
      handler/ middleware/ presenter/ dto/
      sharepage/       # 共有ページの HTML テンプレート（html/template, embed）
    infra/
      postgres/        # pgx
      firebase/        # ID トークン検証
      logging/ config/
```

- 削除：`infra/supabase`, `infra/stripe`, `infra/llm`, `usecase/credit`, `usecase/billing`, `usecase/ai`

#### 2.10.5 iOS アプリ

- SwiftUI、最小 iOS 17（SwiftData を使うため）、リポジトリ直下の `ios/` に置く
- 依存：Firebase iOS SDK（Auth のみ、Swift Package Manager）。そのほかは標準フレームワーク（SwiftData, WebKit, PDFKit, Vision, Security/Keychain）
- API クライアントは URLSession を使い、`Authorization` と `X-Request-Id` を付与する
- 外部AIへの受け渡しは共有シート（`UIActivityViewController` / `ShareLink`）とクリップボード
- `frontend/`（React PWA）は廃止する（コード移行の段階で削除する）

#### 2.10.6 環境・デプロイ・シークレット

- 環境：`dev`（ローカルの Go＋Neon の dev ブランチ＋Firebase の dev プロジェクト）と `prod`（Cloud Run＋Neon の main ブランチ＋Firebase の prod プロジェクト）。`stg` はフェーズ2で追加する
- Cloud Run の環境変数：`DATABASE_URL`（シークレット）, `FIREBASE_PROJECT_ID`, `SHARE_BASE_URL`, `APP_ENV`, `LOG_LEVEL`、フィーチャーフラグ `FEATURE_SHARE_VIEW` / `FEATURE_SHARE_CREATE` / `FEATURE_COMMIT_DISABLED_USE_CASES`（2.10.8）
- デプロイ順序：マイグレーション（後方互換）→ バックエンド → スモークテスト。prod は手動承認を必須とする
- iOS は Xcode から開発者の実機へ直接インストールする（CI での配布はフェーズ2以降）
- CI：Go（`go vet`、lint、`go test`、PostgreSQL のサービスコンテナでの統合テスト、`scripts/arch_check.sh`）と iOS（`xcodebuild test`、macOS ランナー）

#### 2.10.7 改訂時に確定した追加判断

各章の改訂後に、章をまたいだ食い違いを解消するために確定した判断をまとめる（D 番号は振らない）。各章はこの節に揃えてある。2.5・2.6・2.9・2.10.1〜2.10.6 の記述と異なる場合は、本節を優先する。

- 退会（`DELETE /me`）
  - 1 トランザクションで処理する。ドメインデータ（`share_links` を含む）は即時に物理削除する
  - `users` 行は物理削除せず、`deleted_at` を設定し、`email` と `display_name` を消して残す。「一定期間後に物理削除する」処理は持たない
  - 退会後に古い ID トークンで来たリクエストは 401 `UNAUTHORIZED` にする（`deleted_at` のある `users` は認証失敗。email の同期も行わない）
  - Firebase のユーザー削除は、`DELETE /me` の成功後に iOS 側で行う（BFF は管理者権限の認証情報を持たない）
- `prompt_sessions` の状態遷移
  - `created` → `pasted`（parse）、`pasted` → `pasted`（再解析）、`pasted` → `saved`（commit）、`created` / `pasted` → `abandoned`（明示的な中止、または作成から 7 日経過）
  - `created` → `saved` の直接の commit は禁止する。`saved` / `abandoned` は終端状態
  - commit の条件付き UPDATE は `status = 'pasted'` のときだけ成功させる
  - 7 日経過の判定は定期ジョブを持たず、一覧の取得時（`GET /prompt-sessions`、`GET /dashboard`）に条件付き UPDATE で更新する
  - 解析しないユースケース（`summary_generate` / `es_generate` / `es_refine`）も parse を経由する（形式行を除去して全文をプレビューする）。このときの `parse_result` は `success` とし、`parse_result` が null になるのは未解析のときだけとする
  - 「未完了」は `created` と `pasted` を指す。未完了一覧は `GET /prompt-sessions?status=created,pasted`（`status` はカンマ区切りで複数指定できる）で取得する
- AI連携の解析と保存
  - qa-v1 の `【意図N】` は解析してプレビューに表示するだけで、保存しない（`qa` にカラムを追加しない）
  - `research_summarize` のプレビューの構造は、7 章の `{ highlights, idealCandidate, recentTopics }` とする
- プロンプト長：4.3.5 の値を正とする。プロンプト全体は推奨 8,000 字、`content_raw` は 1 件あたり先頭 3,000 字まで、`research_summarize` は `content_raw` を 8,000 字まで使う。parse の入力上限は 100,000 字とする
- 共有リンク
  - トークンは 32 バイト（256 ビット）の暗号学的乱数とし、DB には SHA-256 のハッシュを保存する
  - スナップショットの本文は `content_final`、なければ `content_draft` とする
  - `GET /s/{token}` の応答ヘッダは `X-Robots-Tag: noindex`、`Cache-Control: no-store`、`Referrer-Policy: no-referrer`、スクリプトを禁止する `Content-Security-Policy` とする
  - `show_company_name` の既定値は false とする（API で省略した場合も false）
  - ES プロジェクトを削除すると、紐づく `share_links` も削除する
  - 期限切れ・失効したリンクは、運用で定期的に物理削除してよい（保持期間は定めない）
- 認可：他人のリソースへのアクセスは 404 `NOT_FOUND`（AI連携セッションは `SESSION_NOT_FOUND`）とし、403 は使わない
- レート制限：認証付き API は user_id 単位で 1 分あたり 120 回、`GET /s/{token}` は IP 単位で 1 分あたり 30 回。Cloud Run のインスタンス内のメモリで数える
- 下書きの送信：オンライン復帰後に自動送信はしない。下書きがあることを表示し、ユーザーの操作で送信する
- 画面構成とタブ：3 章（3.2.1 / 3.3）を正とする。タブは Home / 対策 / 記録 / バンク / データの 5 つで、設定は Home のナビゲーションバーから開く
- Domain 層で import を禁止するもの：`gin`、`pgx`、`firebase` などの外部 SDK と、`net/http` などのネットワーク・DB 関連の標準パッケージ。`scripts/arch_check.sh` で検査する
- ID の形式不正（uuid でない）は 400 `VALIDATION_ERROR`、値の範囲・業務ルール違反は 422 `VALIDATION_ERROR` とする（7.7.2）
- iOS のビルド構成名は `Debug-Dev` / `Release-Prod` とする（5.12.1）
- ユーザーによる DELETE（7.5.3）は、`assets` / `companies` は論理削除、それ以外（`interviews` / `interview_logs` / `qa` など）は物理削除とする。`prompt_sessions` の対象参照は、対象が削除されたら null にして行を残す

#### 2.10.8 改訂前からの食い違いの解消（2026-10-22）

- ES プロジェクトのステータス：`not_started` を追加して5値にする（`not_started` / `draft` / `in_review` / `completed` / `submitted`）。初期値は `not_started`
  - 遷移の制約は設けない。自動遷移は「`not_started` のとき、セクション本文が初めて保存されたら `draft`」の1つだけ
  - UI では「提出済」を選ぶときだけ確認ダイアログを出す
  - 反映先：2.2.5 / 2.4.3、3.5.4、7.5（CRUD 一覧・commit）、8章・dbml、11章（IT-PS-017 / 018）
- ピン留め：お気に入り（`qa.is_favorite`）に統一する。お気に入りは一覧の先頭に表示し、絞り込みにも使う。新しいカラムは追加しない
  - Q&A のアーカイブは持たない（復習ステータスと削除で足りる）。自己資産・企業の「アーカイブ」は論理削除（`deleted_at`）と読み替える
  - 反映先：1.1.7、2.2.9 / 2.4.1 / 2.4.2 / 2.4.4、3.5.3
- ESセクションの版履歴：テーブル `es_section_versions` を新設する（id, es_section_id, user_id, version, content, source (`manual` / `es_generate` / `es_refine`), created_at）
  - 版を作るのは、AI連携の commit で `content_draft` を上書きする直前、「この版を保存」、「この版に戻す」の直前だけ。普段の編集（PATCH）や自動保存では作らない
  - セクションごとに最新20件まで。超えた分は、版を作るトランザクションで古い順に削除する
  - 「この版に戻す」は MVP に含める。差分は iOS 側で計算する。対象は `content_draft` だけ
  - API：`GET` / `POST /es-sections/{id}/versions`、`POST /es-sections/{id}/versions/{versionId}/restore`
  - 画面：S-ES-05（版の履歴）を追加
  - 反映先：1.1.4、2.2.6 / 2.2.6.1 / 2.3 / 2.4.3、3.3 / 3.5.5 / 3.5.6.1、7.5.3 / 7.5.5、8章・dbml・dbdiagram、11章（IT-DB-008、IT-VER-001〜006）
- 実装時に確定した API の補足（2026-10-22、7.5.5 / 7.2.3 に反映済み）
  - `GET /prompt-sessions/{id}` を追加する（7.6.1 の「GET でセッションを確認」に使う）
  - `POST /prompt-sessions` と `GET .../prompt` の応答に `warnings` を含める（4.3.5 の長さ警告）
  - 未完了一覧の項目に `targetLabel`（対象の表示名）を含める
  - `research_summarize` の preview に、保存用に整形した `summaryText` を含める
  - Go のパッケージに `domain/service/session`、`domain/service/es`、`infra/system`、`internal/app`（DI）を追加する
  - 契約テスト：Go の統合テストが実レスポンスを `ios/InterviewHackKit/Tests/InterviewHackKitTests/Fixtures/contract` に書き出し、Swift のテストがデコードできることを検証する（11.4.2）
- 未完了一覧の取得：`GET /prompt-sessions?status=created,pasted`（`status` はカンマ区切りで複数指定できる）。2.10.3 も更新済み
- 解析しないユースケース（`summary_generate` / `es_generate` / `es_refine`）の `parse_result` は `success`。`null` は未解析（`created`）のときだけ
- 共有ページの CSP は `default-src 'none'; style-src 'unsafe-inline'; img-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'` で確定する（7.5.6、9章、11章 IT-SH-002）
- フィーチャーフラグの環境変数を次の3つに確定する（12.11、14.8.1、7.5.6 / 7.7、11章 IT-SH-012 / IT-PS-019）
  - `FEATURE_SHARE_VIEW`（既定 `on`）
  - `FEATURE_SHARE_CREATE`（既定 `on`）
  - `FEATURE_COMMIT_DISABLED_USE_CASES`（既定は空）
  - 停止時は 503。共有ページは「一時停止中」の HTML、それ以外は `FEATURE_DISABLED`（エラーコードを追加）

### 2.11 制約・注意点

- 無料 Apple ID では **Sign in with Apple / iCloud / Push 通知が使えない**。アプリは 7 日ごとに再インストールが必要
- 配布（TestFlight / App Store）を始める時点で Apple Developer Program（年 $99）に加入する
  - その際、Google ログインを提供するなら Sign in with Apple の追加が必要になる見込み（App Store ガイドライン）
- Neon 無料枠はアイドル時にコンピュートが停止し、初回アクセスで自動復帰する（コールドスタートあり）
- Cloud Run と Neon のリージョンは近接させる
- 将来 StoreKit で課金する場合、デジタル機能の課金に Stripe は原則使えない

---

## 3. 廃止されるもの

- Stripe 連携一式（Checkout / Webhook / `stripe_events`）
- クレジット一式（`credit_balances` / `credit_transactions` / `credit_packages` / `credit_costs`、Reserve / Commit / Reversed 状態遷移）
- サーバーからの LLM 呼び出し（`backend/internal/infra/llm`）、AI コスト計測・レート制限（コスト防御目的のもの）
- Supabase 一式（Auth / RLS / 直アクセス境界 / `auth.users` 参照）
- React PWA（`frontend/`）と Cloudflare Pages デプロイ

---

## 4. 未決事項

| # | 内容 | メモ |
|---|------|------|
| O1 | ~~コピペで貼り戻した回答の解析形式~~ | **決定 → D10** |
| O2 | ~~企業研究資料の URL 取得・PDF 抽出の置き場所~~ | **決定 → D11** |
| O3 | ~~共有リンクの仕様~~ | **決定 → D12** |
| O4 | ~~`generation_jobs` の扱い~~ | **決定 → D13** |
| O5 | ~~収益化~~ | **決定 → D14** |
| O6 | ~~分析（10 章）~~ | **決定 → D15** |
| O7 | ~~オフライン~~ | **決定 → D16** |

未決事項はすべて決定済み（2026-10-22）。

---

## 5. 設計書の改訂状況

| 章 | 改訂の大きさ | 状態 |
|----|-------------|------|
| 0_Comparison | 中 | 改訂済み（2026-10-22、章間の整合確認済み・レビュー待ち） |
| 1_Requirementdefinition | 大 | 改訂済み（2026-10-22、章間の整合確認済み・レビュー待ち） |
| 2_DomainDesign | 中 | 改訂済み（2026-10-22、章間の整合確認済み・レビュー待ち） |
| 3_UIUXDesign | 中 | 改訂済み（2026-10-22、章間の整合確認済み・レビュー待ち） |
| 4_LLMDesign | 大（作り直し） | 改訂済み（2026-10-22、章間の整合確認済み・レビュー待ち） |
| 5_FrontendDesign | 大（作り直し：iOS） | 改訂済み（2026-10-22、章間の整合確認済み・レビュー待ち） |
| 6_AuthAccountCahrgingDesign | 大（課金の削除） | 改訂済み（2026-10-22、章間の整合確認済み・レビュー待ち） |
| 7_BackendDesign | 大 | 改訂済み（2026-10-22、章間の整合確認済み・レビュー待ち） |
| 8_DBDesign（.md / .dbml） | 中 | 改訂済み（2026-10-22、章間の整合確認済み・レビュー待ち） |
| 9_SecurityDesign | 中 | 改訂済み（2026-10-22、章間の整合確認済み・レビュー待ち） |
| 10_AnalyzeDesign | 中 | 改訂済み（2026-10-22、章間の整合確認済み・レビュー待ち） |
| 11_TestDesign | 中 | 改訂済み（2026-10-22、章間の整合確認済み・レビュー待ち） |
| 12_Operation | 中 | 改訂済み（2026-10-22、章間の整合確認済み・レビュー待ち） |
| 13_Schedule | 中 | 改訂済み（2026-10-22、章間の整合確認済み・レビュー待ち） |
| 14_CICDDesign | 中 | 改訂済み（2026-10-22、章間の整合確認済み・レビュー待ち） |

改訂後は AGENTS.md / README.md の技術スタック・セットアップ手順も更新する。
