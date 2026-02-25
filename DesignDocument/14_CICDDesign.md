# 14. CI/CD 設計（GitHub Actions / Pages / Cloud Run）

## 14.1 方針

- 「人間がやらなくていいこと」はすべて自動化する（テスト / ビルド / デプロイ）
- 課金・AI・認証が絡むため、**本番直デプロイは原則しない**（stgで検証→prod）
- 失敗時は「どこで落ちたか」が一目で分かるパイプラインにする
- シークレットはGitHub Actionsと各プラットフォーム（Cloud Run / Pages）の環境変数で管理し、リポジトリに置かない
- フロントのエラー収集（Sentry）は任意だが、本番では推奨とする。release（Git SHA）と request_id の相関を維持する
- すべての成果物は Git SHA でタグ付けし、再現可能なデプロイを保証する
- バックエンドは DDD + クリーンアーキテクチャ（DesignDocument/7_BackendDesign.md）を前提とし、CI で層違反（Domain への外部依存など）を検出・防止する

---

## 14.2 対象リポジトリ構成（想定）

- monorepo もしくは 2 repo
  - frontend/（Vite + React）
  - backend/（Go + Gin, Docker）

以下は monorepo 想定で記載（分離しても同じ考え方）

---

## 14.3 環境（dev / stg / prod）

- dev: ローカル開発
- stg: ステージング（本番と同等構成、StripeはTest、Supabaseはstgプロジェクト）
- prod: 本番（Stripe Live、Supabase prod）

CI/CDでは以下を分離する：

- Cloudflare Pages: preview / stg / prod（または stg 用プロジェクト）
- Cloud Run: service-stg / service-prod
- Supabase: project-stg / project-prod
- Stripe: test mode / live mode
- OpenAI / Gemini API Key も環境別に分離する
- Stripe: Test/LIVE で Webhook Secret / Price ID / API Key を完全分離する
- Supabase: project-stg / project-prod の URL / ANON KEY / SERVICE KEY を完全分離する

---

## 14.4 ブランチ戦略（最小）

- main: 本番
- develop（任意）: 開発統合
- feature/\*: 機能開発

運用ルール：

- main への直接 push 禁止
- PR 経由でのみ main にマージ
- PR では必ず CI（テスト・ビルド）が通ること

---

## 14.5 CI（Pull Request 時）

### 14.5.1 フロントエンド（frontend/）

- Node install
- `npm ci`
- `npm run lint`
- `npm run typecheck`（あれば）
- `npm run test`（あれば）
- `npm run build`（Vite build）

### 14.5.2 バックエンド（backend/）

- Go setup
- `go test ./...`
- `go vet ./...`
- `golangci-lint`（任意だが推奨）
- Docker build（ビルドが通るかの検証）
- アーキテクチャチェック（禁止 import / 依存方向の検査。例：Domain が gin/stripe/supabase/openai/gemini 等に依存していないこと）
- パッケージ依存の可視化・循環依存チェック（必要に応じて）

共通ゲート：

- 依存関係の脆弱性チェック（frontend: `npm audit --production` 相当 / backend: 可能であれば `govulncheck`）
- 成果物に `RELEASE=Git SHA` が埋め込まれていることの検証
- 変更内容が DesignDocument/7_BackendDesign.md（アーキテクチャ）および関連設計に準拠していることのレビュー観点チェック（PRテンプレで宣言）

失敗したら PR はマージ不可

---

## 14.6 CD（main マージ後の自動デプロイ）

※ CD は CI の全ゲート（テスト・ビルド・アーキテクチャチェック）を通過した成果物のみをデプロイ対象とする。

### 14.6.1 ステージング（stg）へのデプロイ

1. frontend:
   - Vite build
   - Cloudflare Pages（stg）へデプロイ
2. backend:
   - Docker build
   - Artifact Registry / GCR に push
   - Cloud Run（service-stg）へデプロイ
     2.5. DB マイグレーション（必要な場合のみ、idempotent）
3. スモークテスト:
   - /health エンドポイント
   - 認証なしで叩ける軽量チェックAPI
   - 認証ありの主要AI導線（/ai/qa-generate, /ai/es-generate）の疎通確認（stg用テストデータ）
   - 失敗したら以降のステップを止める

### 14.6.2 本番（prod）へのデプロイ

- GitHub Actions の `environment: prod` を使用し **手動承認必須**
- 承認後：
  1. frontend を Cloudflare Pages（prod）へデプロイ
  2. backend を Cloud Run（service-prod）へデプロイ
     2.5. DB マイグレーション（必要な場合のみ、idempotent）
  3. スモークテスト実行
- 失敗した場合：
  - 直前バージョンへロールバック（Cloud Run のリビジョン機能を使用）

---

## 14.7 シークレット管理

### 14.7.1 GitHub Secrets に置くもの

- GCP_SERVICE_ACCOUNT_KEY（Cloud Run デプロイ用）
- CLOUDFLARE_API_TOKEN（Pages デプロイ用）
- その他 CI が必要とする最小限のトークン

### 14.7.2 置いてはいけないもの

- Stripe API Key（Cloud Run の環境変数で管理）
- Supabase service_role key
- OpenAI / Gemini API Key

これらは **Cloud Run / Pages の環境変数** 側で管理する。

---

## 14.8 環境変数の注入方針

### 14.8.1 フロント（Cloudflare Pages）

- `VITE_SUPABASE_URL`
- `VITE_SUPABASE_ANON_KEY`
- `VITE_API_BASE_URL`
- `VITE_APP_ORIGIN`
- 環境別（stg/prod）に Pages 側で設定

### 14.8.2 バックエンド（Cloud Run）

- `SUPABASE_URL`
- `SUPABASE_PUBLISHABLE_KEY`（推奨）
- `SUPABASE_ANON_KEY`（フォールバック）
- `STRIPE_SECRET_KEY`
- `STRIPE_WEBHOOK_SECRET`
- `OPENAI_API_KEY`
- `GEMINI_API_KEY`
- `ENV=stg|prod` など

認証トークン検証は BFF から `GET {SUPABASE_URL}/auth/v1/user` を呼び、`apikey` に
`SUPABASE_PUBLISHABLE_KEY`（未設定時は `SUPABASE_ANON_KEY`）を付与して行う。

---

## 14.9 リリース識別子（Versioning）

- フロント/バックエンドともに
  - Git の commit SHA を `RELEASE` として埋め込む
- フロント：
  - Sentry の `release` に設定
- バックエンド：
  - 構造化ログに `release` フィールドとして出力

これにより：

- どのリリースで不具合が出たか即座に追跡可能

---

## 14.10 ロールバック戦略

### 14.10.1 バックエンド（Cloud Run）

- Cloud Run のリビジョン機能で
  - 直前の安定リビジョンへ即時切り戻し

### 14.10.2 フロント（Cloudflare Pages）

- 直前のデプロイを再デプロイ（またはロールバック機能）
- 問題があるコミットを revert して再デプロイ

---

## 14.11 よくある事故の予防策

- prod 環境は **必ず手動承認**
- stg を通さずに prod に行けないフローにする
- 環境変数の差分はデプロイ前に確認ログを出す
- Stripe Live Key が stg に入っていないかを CI でチェック（ENV ガード）
- Domain / Usecase にフレームワーク・SDK依存を混入させない（アーキテクチャチェックで検出）

---

## 14.12 MVP時点の簡略版（現実解）

- 最初は：
  - PR: CIのみ
  - main: 直接 prod デプロイ
- ただし：
  - 課金を入れるタイミングで **stg + 承認フロー必須** に昇格させる

---

## 14.13 この章の凍結事項

- CI/CD は GitHub Actions を使用する
- PR では必ずテスト + ビルドを通す
- 本番デプロイは原則 stg → 承認 → prod
- シークレットは GitHub に直置きせず、実行環境の環境変数で管理する
- ロールバック可能な構成（Cloud Run リビジョン / Pages 再デプロイ）を前提にする
- バックエンドは DDD + クリーンアーキテクチャの依存方向を破らない（DesignDocument/7_BackendDesign.md 準拠、CI で検証）
