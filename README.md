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

| 変数                    | 説明                            |
| ----------------------- | ------------------------------- |
| `DATABASE_URL`          | Supabase PostgreSQL 接続文字列  |
| `SUPABASE_JWT_SECRET`   | JWT 検証用シークレット          |
| `STRIPE_SECRET_KEY`     | Stripe シークレットキー         |
| `STRIPE_WEBHOOK_SECRET` | Stripe Webhook 署名シークレット |
| `OPENAI_API_KEY`        | OpenAI API キー                 |
| `ALLOW_ORIGIN`          | CORS 許可オリジン               |

**Frontend** (`frontend/.env`)

| 変数                     | 説明                      |
| ------------------------ | ------------------------- |
| `VITE_SUPABASE_URL`      | Supabase プロジェクト URL |
| `VITE_SUPABASE_ANON_KEY` | Supabase anon キー        |
| `VITE_API_BASE_URL`      | BFF の URL                |
| `VITE_APP_ORIGIN`        | フロントエンドのオリジン  |

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
