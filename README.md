# InterviewHack

AI を活用した面接対策アシスタント。自己分析資産の管理、企業研究、ES 作成支援、想定 Q&A 生成、回答添削、面接振り返りまでを一貫してサポートする PWA アプリケーション。

## アーキテクチャ

```
┌─────────────────────┐     ┌──────────────────────────────┐
│  Frontend (PWA)     │     │  Backend (BFF)               │
│  Vite + React + TS  │────▶│  Go + Gin                    │
│  Tailwind CSS       │     │  DDD + Clean Architecture    │
│  TanStack Query     │     │                              │
└────────┬────────────┘     │  Domain ← Port ← Usecase     │
         │                  │  Adapter → Infra              │
         │ Supabase Auth    └──────┬──────────┬────────────┘
         │ (JWT)                   │          │
         ▼                         ▼          ▼
┌─────────────────────┐  ┌────────────┐  ┌────────────┐
│  Supabase           │  │  Stripe    │  │  OpenAI /  │
│  PostgreSQL + RLS   │  │  Checkout  │  │  Gemini    │
│  Auth (Google/Email)│  │  Webhook   │  │  LLM API   │
└─────────────────────┘  └────────────┘  └────────────┘
```

| コンポーネント | 技術スタック                                                                |
| -------------- | --------------------------------------------------------------------------- |
| Frontend       | Vite, React 19, TypeScript, Tailwind CSS, TanStack Query, React Router, PWA |
| Backend (BFF)  | Go 1.24, Gin, pgx/v5, DDD + Clean Architecture                              |
| Database       | Supabase (PostgreSQL) + Row Level Security                                  |
| 認証           | Supabase Auth (Google OAuth + Email/Password)                               |
| 決済           | Stripe Checkout + Webhook                                                   |
| AI             | OpenAI API (GPT-4o-mini), Gemini API                                        |
| CI/CD          | GitHub Actions                                                              |

## プロジェクト構造

```
InterviewHack/
├── DesignDocument/          # 設計書（Source of Truth）
├── backend/
│   ├── cmd/api/             # エントリポイント (main.go, wire.go)
│   ├── internal/
│   │   ├── domain/          # エンティティ, 値オブジェクト, ドメインサービス
│   │   ├── usecase/         # アプリケーションロジック
│   │   ├── port/            # リポジトリ/外部サービスの interface
│   │   ├── adapter/http/    # Gin ハンドラ, ミドルウェア, DTO
│   │   └── infra/           # Supabase, Stripe, OpenAI 実装
│   ├── migrations/          # SQL マイグレーションファイル
│   ├── scripts/             # アーキテクチャチェック等
│   └── Dockerfile
├── frontend/
│   ├── src/
│   │   ├── app/             # App, Router, Providers, Layout
│   │   ├── pages/           # 各画面コンポーネント
│   │   ├── components/      # 共通 UI, AuthGuard
│   │   ├── features/        # 機能別モジュール (auth, billing, ai, ...)
│   │   ├── lib/             # Supabase client, API client, Zod schema
│   │   └── styles/          # Tailwind エントリ
│   └── vite.config.ts
├── .github/
│   ├── workflows/           # CI/CD (frontend, backend, staging, production)
│   └── PULL_REQUEST_TEMPLATE.md
└── Agent.md                 # AI エージェント向けルール
```

## セットアップ

### 前提条件

- Go 1.24+
- Node.js 20+
- Supabase プロジェクト（オンライン）
- Stripe アカウント（テストモード）
- OpenAI API キー

### Backend

```bash
cd backend
cp .env.example .env
# .env を編集して実際の値を設定

go mod download
go build ./cmd/api
./api
# → http://localhost:8080/health で {"status":"ok"}
```

### Frontend

```bash
cd frontend
cp .env.example .env
# .env を編集して実際の値を設定

npm install
npm run dev
# → http://localhost:5173
```

### 環境変数

**Backend** (`backend/.env`)

| 変数                       | 説明                                  |
| -------------------------- | ------------------------------------- |
| `DATABASE_URL`             | Supabase PostgreSQL 接続文字列        |
| `SUPABASE_URL`             | Supabase プロジェクト URL             |
| `SUPABASE_PUBLISHABLE_KEY` | JWT 検証用 API キー（推奨）           |
| `SUPABASE_ANON_KEY`        | JWT 検証用 API キー（フォールバック） |
| `STRIPE_SECRET_KEY`        | Stripe シークレットキー               |
| `STRIPE_WEBHOOK_SECRET`    | Stripe Webhook 署名シークレット       |
| `OPENAI_API_KEY`           | OpenAI API キー                       |
| `ALLOW_ORIGIN`             | CORS 許可オリジン                     |

Supabase の JWT は Signing Keys (非対称鍵) 前提で、BFF は `/auth/v1/user` でトークンを検証します。`SUPABASE_JWT_SECRET` は使用しません。

**Frontend** (`frontend/.env`)

| 変数                     | 説明                      |
| ------------------------ | ------------------------- |
| `VITE_SUPABASE_URL`      | Supabase プロジェクト URL |
| `VITE_SUPABASE_ANON_KEY` | Supabase anon キー        |
| `VITE_API_BASE_URL`      | BFF の URL                |
| `VITE_APP_ORIGIN`        | フロントエンドのオリジン  |

### Google OAuth の設定

「Googleで続行」でログインするには、**Google Cloud Console** と **Supabase Dashboard** で OAuth を有効化する必要があります。手順は [docs/GoogleOAuthSetup.md](docs/GoogleOAuthSetup.md) を参照してください。

## ローカルでの検証

バックエンドとフロントエンドを同時に動かしてローカルで確認する手順です。

### 1. 環境変数を用意する

**Backend** (`backend/.env`)

```bash
cd backend
cp .env.example .env
```

最低限、起動に必須なのは `DATABASE_URL`（Supabase の接続文字列）です。  
その他は未設定でも起動はしますが、課金・AI 機能は動きません。

- ローカル用の例: `ALLOW_ORIGIN=http://localhost:5173`
- Supabase ダッシュボード → Project Settings → Database で「Connection string」をコピーして `DATABASE_URL` に設定

**Frontend** (`frontend/.env`)

```bash
cd frontend
cp .env.example .env
```

ローカルで BFF に繋ぐ場合は次のようにします。

- `VITE_API_BASE_URL=http://localhost:8080` … フロントの API 呼び出し先
- `VITE_APP_ORIGIN=http://localhost:5173`
- `VITE_SUPABASE_URL` / `VITE_SUPABASE_ANON_KEY` … Supabase のプロジェクト URL と anon key

### 2. バックエンドを起動する

```bash
cd backend
go mod download   # 初回のみ
go run ./cmd/api
```

- 成功すると `starting server` のログのあと、**http://localhost:8080** で待ち受けます。
- 止めるときは `Ctrl+C` で終了します。

**確認:**

```bash
curl -s http://localhost:8080/health
# → {"status":"ok"} が返れば OK
```

ビルドしてから実行する場合:

```bash
cd backend
go build -o api ./cmd/api
./api
```

### 3. フロントエンドを起動する（別ターミナル）

バックエンドを起動したまま、**別のターミナル**で:

```bash
cd frontend
npm install   # 初回のみ
npm run dev
```

- **http://localhost:5173** で開発サーバーが立ち上がります。
- ブラウザで開くとログイン画面が出ます。  
  `VITE_API_BASE_URL=http://localhost:8080` にしている場合、ログイン後の API 呼び出しはこのバックエンドに向かいます。

### 4. 起動順序のまとめ

| 順番 | 場所        | コマンド           | 確認先                       |
| ---- | ----------- | ------------------ | ---------------------------- |
| 1    | `backend/`  | `go run ./cmd/api` | http://localhost:8080/health |
| 2    | `frontend/` | `npm run dev`      | http://localhost:5173        |

- 先にバックエンド、そのあとフロントエンドを起動する想定です。
- フロントのみ確認する場合は `npm run dev` だけでも可能ですが、認証や API を使う機能はバックエンドが無いと動きません。

### 5. よく使うコマンド一覧

| 用途   | Backend                     | Frontend                   |
| ------ | --------------------------- | -------------------------- |
| 起動   | `go run ./cmd/api`          | `npm run dev`              |
| ビルド | `go build -o api ./cmd/api` | `npm run build`            |
| テスト | `go test ./...`             | `npm test`（設定済みなら） |
| 停止   | `Ctrl+C`                    | `Ctrl+C`                   |

## 開発コマンド

```bash
# Backend
cd backend
go test ./...                    # テスト実行
go vet ./...                     # 静的解析
bash scripts/arch_check.sh       # Domain 層純粋性チェック
go build ./cmd/api               # ビルド

# Frontend
cd frontend
npm run dev                      # 開発サーバー
npm run build                    # プロダクションビルド
npm run lint                     # ESLint
npx tsc --noEmit                 # 型チェック
```

## 設計原則

本プロジェクトは `DesignDocument/` 配下の設計書を **Source of Truth** とします。

- **Clean Architecture**: Domain → Port → Usecase → Adapter/Infra の依存方向を厳守。Domain 層は外部 SDK を一切 import しない
- **クレジット状態機械**: `pending → committed | reversed` の3状態遷移（Reserve/Commit/Reversed パターン）
- **冪等性**: Stripe Webhook は `event.id` で重複チェック。クレジット付与の二重実行を防止
- **RLS**: 全テーブルで Row Level Security を有効化。課金系テーブルは service_role のみ書き込み可
- **相関 ID**: `X-Request-Id` を Frontend → BFF → ログまで一貫して伝搬

## DB スキーマ

20 テーブル構成（Supabase PostgreSQL）:

- **コアドメイン**: profiles, assets, asset_tags, companies, research_materials, es_projects, es_sections, interviews, interview_logs, qa + 各種リレーションテーブル
- **課金**: credit_balances, credit_transactions, stripe_events, credit_packages, credit_costs
- **AI**: generation_jobs

詳細は `DesignDocument/8_DBDesign.md` および `backend/migrations/` を参照。

## CI/CD

| ワークフロー        | トリガー                 | 内容                                   |
| ------------------- | ------------------------ | -------------------------------------- |
| `ci-frontend.yml`   | PR (frontend/\*\*)       | lint → typecheck → test → build        |
| `ci-backend.yml`    | PR (backend/\*\*)        | vet → test → arch check → docker build |
| `cd-staging.yml`    | main push                | ステージングデプロイ                   |
| `cd-production.yml` | 手動 (workflow_dispatch) | 本番デプロイ（承認必須）               |

## ライセンス

Private
