# 5. フロントエンド設計（Vite + React + PWA + Cloudflare Pages）

## 5.1 方針（フロントの責務と非責務）

### 5.1.1 フロントの責務

- 画面/UI/状態管理（UXの実現）
- Supabase Auth（Google OAuth + Email/Password）によるログイン/ログアウト
- Supabase DBの通常CRUD（自己資産/企業/面接/ログ/Q&A）
- Cloud Run BFF APIの呼び出し（AI生成・添削、Stripe Checkout開始、残高表示用の補助）
- PWA化（manifest + service worker + オフライン時の最低限挙動）

### 5.1.2 フロントの非責務（禁止）

- Stripe Webhook受信（Cloud Runのみ）
- OpenAI/Gemini APIの直接呼び出し（Cloud Runのみ）
- クレジット残高の更新/付与/消費（Cloud Runのみ）
- user_id の自己申告（JWT由来の auth.uid() を前提にする）

---

## 5.2 採用技術スタック（確定）

- Build: Vite
- UI: React + TypeScript
- Router: React Router
- Data Fetch/Cache: TanStack Query（React Query）
- Form: React Hook Form + Zod
- Styling: Tailwind CSS（MVP最速）
- Auth/DB: supabase-js
- PWA: vite-plugin-pwa
- Hosting: Cloudflare Pages（静的配信 + CDN）
- Monitoring（任意/推奨）：Sentry（フロントエラー収集）

---

## 5.3 アプリ構造（ディレクトリ設計）

```
src/
  app/                # アプリの骨格（Router, Providers, Layout）
  pages/              # 画面単位（Routeと1:1）
  components/         # 再利用UI
  features/           # 機能ドメイン単位（assets, companies, qa, interviews, billing, ai）
  lib/
    supabase/         # supabase client, auth helper
    api/              # BFF API client（fetch wrapper）
    schema/           # zod schema
    utils/            # 日付、文字数、エラー整形など
  hooks/              # 共通hooks
  styles/             # tailwind, global css
public/
  manifest.webmanifest
  icons/
```

---

## 5.4 ルーティング設計（React Router）

### 5.4.1 ルート一覧（MVP）

- `/` : Home（ダッシュボード）
- `/login` : ログイン（Google OAuth / Email/Password）
- `/reset-password` : パスワード再設定（Email/Password 利用時）
- `/auth/callback` : OAuthコールバック処理
- `/assets` : 自己資産一覧
- `/assets/new` : 自己資産作成
- `/assets/:id` : 自己資産詳細/編集
- `/companies` : 企業一覧
- `/companies/new`
- `/companies/:id`
- `/es` : ESプロジェクト一覧
- `/es/new` : ES作成（企業選択）
- `/es/:id` : ESエディタ（セクション構成・執筆）
- `/prep` : 面接対策ルーム（企業選択 → 生成）
- `/prep/result/:generationId` : 生成結果
- `/interviews` : 面接一覧
- `/interviews/new`
- `/interviews/:id` : 面接詳細
- `/interviews/:id/logs/new` : 面接ログ入力
- `/qa` : Q&A一覧
- `/qa/:id` : Q&A詳細
- `/review` : 復習UI（カード）
- `/billing` : クレジット購入
- `/settings` : 設定

### 5.4.2 ガード（認証必須ページ）

- `/login` と `/auth/callback` 以外は原則ログイン必須
- 未ログイン時は `/login` にリダイレクト

---

## 5.5 認証設計（Supabase Auth：Google OAuth + Email/Password）

### 5.5.1 ログインフロー

1. `/login` で「Googleで続行」
2. `supabase.auth.signInWithOAuth({ provider: "google", options: { redirectTo } })`
3. `redirectTo = https://app.example.com/auth/callback`（環境で切替）
4. `/auth/callback` でセッション確定
5. 成功後 `/` へ遷移

### 5.5.1.1 Email/Password（追加ログイン手段）

目的：

- Googleアカウントを持たない/使いたくないユーザーもオンボードできるようにする
- OAuth障害時のフォールバック手段を用意する

フロー（サインアップ）：

1. `/login` で Email/Password を入力
2. `supabase.auth.signUp({ email, password })`
3. 必要に応じてメール確認（Email Confirm）を有効化する
4. 確認完了後にセッション確定 → `/` へ遷移

フロー（ログイン）：

1. `/login` で Email/Password を入力
2. `supabase.auth.signInWithPassword({ email, password })`
3. 成功後 `/` へ遷移

補足：

- MVPでは「Google OAuth を主導線、Email/Password はサブ導線」とする
- パスワードリセット導線（Forgot Password）はMVP後でも良いが、少なくとも設計としては用意する

### 5.5.2 リダイレクトURL（Supabase側設定）

- 本番：
  - `https://app.example.com/auth/callback`
- ローカル：
  - `http://localhost:5173/auth/callback`
- Email/Password 利用時も同一オリジン（app.example.com / localhost）で完結する

### 5.5.3 セッション管理

- supabase-js が refresh を含むセッション管理を担当
- 起動時に `supabase.auth.getSession()` を呼び、ログイン状態を復元
- `onAuthStateChange` で状態更新

---

## 5.6 データアクセス設計（Supabase直 / BFF経由の境界）

### 5.6.1 Supabase直アクセス（CRUD）

- assets, asset_tags, asset_tag_relations
- companies
- research_materials
- es_projects, es_sections
- es_project_asset_relations, es_project_research_material_relations
- interviews, interview_logs
- qa, qa_asset_relations
- profiles（読み取り/一部更新）
- credit_balances（読み取りのみ）
- credit_transactions（読み取りのみ）
- credit_packages / credit_costs（読み取りのみ）

### 5.6.2 BFF経由（必須）

- Stripe Checkout開始
  - `POST /billing/checkout-session`
- AI生成・添削・要約（クレジット消費込み）
  - `POST /ai/qa-generate`
  - `POST /ai/answer-review`
  - `POST /ai/summary-generate`
  - `POST /ai/research-summarize`
  - `POST /ai/es-generate`
  - `POST /ai/es-refine`

理由：

- クレジット付与/消費は改ざん対象なのでサーバー側のみで確定する

BFF専用（クライアント直書き/直読禁止のテーブル）：

- generation_jobs
- credit_balances（書き込み禁止 / 読み取りは可）
- credit_transactions（書き込み禁止 / 読み取りは可）
- stripe_events

---

## 5.7 APIクライアント設計（BFF呼び出し）

### 5.7.1 fetch wrapper（共通）

- 入力：
  - method, path, body
- 自動付与：
  - `Authorization: Bearer <supabase_access_token>`
  - `X-Request-Id` が未指定の場合はクライアント側でUUIDを生成して付与（BFF側で受理/上書き可）
  - `Content-Type: application/json`
  - レスポンスヘッダ `X-Request-Id` が存在する場合は取得し、UIのエラー表示および Sentry タグに付与する
- エラー整形：
  - `{ code, message }` 形式に統一してUI表示
  - 可能であれば `request_id` を含めて表示/記録する（BFFログとの突合用）

### 5.7.2 リトライ方針

- 429/503/ネットワーク断のみ、指数バックオフで最大2回
- 402（INSUFFICIENT_CREDITS）はリトライせず `/billing` へ誘導

---

## 5.8 状態管理（TanStack Query中心）

- サーバーステート（Supabase/BFF結果）は Query に寄せる
- ローカルUIステートのみ useState/useReducer
- Query Key例
  - `["assets"]`
  - `["companies"]`
  - `["es_projects"]`
  - `["es_project", projectId]`
  - `["research_materials", companyId]`
  - `["qa", { companyId, status }]`
  - `["creditBalance"]`
  - `["generation", generationId]`

キャッシュ方針（MVP）：

- assets/companies/qa：staleTime 30s〜2m
- creditBalance：staleTime 10s（購入直後反映が欲しい）
- generation結果：即時保存するならDB起点で再取得でも可

---

## 5.9 フォーム設計（React Hook Form + Zod）

### 5.9.1 バリデーション基本

- 必須項目の明確化（空文字禁止）
- 最大文字数（例：質問300、回答2000など）をZodで固定
- 送信中は二重送信不可（submit disable）

### 5.9.2 フォーム単位

- 自己資産作成/編集
- 企業作成/編集
- ESプロジェクト作成（企業、期限）
- ESセクション執筆（文字数カウント、下書き保存）
- 企業研究資料登録（URL/PDFアップロード）
- 面接作成
- 面接ログ入力（複数追加）
- 生成条件（企業、職種、asset選択）

---

## 5.10 UI設計（MVPの統一ルール）

- グローバルナビ：下部タブ（Home/対策/記録/バンク/データ）
- 主要CTAは画面下固定
- 長文は折りたたみ（回答はデフォルト折りたたみ）
- Empty/Loading/Error を必ず用意（3章の状態設計に従う）
- 課金導線：残高が少ない/不足時にだけ強く出す（常時売り込みしない）

---

## 5.11 PWA設計（vite-plugin-pwa）

### 5.11.1 必須要素

- `manifest.webmanifest`
  - name, short_name, start_url, display=standalone
  - icons（192/512 + maskable推奨）
- service worker
  - 静的アセットのキャッシュ
  - HTMLは network-first（更新取り込み優先）
  - APIレスポンスは基本キャッシュしない（個人データのため）

### 5.11.2 オフライン時の最低限

- オフライン検知 → 「ネット接続が必要」画面を表示
- 下書き（面接ログ入力など）は localStorage に一時保存可（任意、MVP後でも可）。認証トークンや機密情報は保存しない

---

## 5.12 環境変数（Cloudflare Pages）

### 5.12.1 必須

- `VITE_SUPABASE_URL`
- `VITE_SUPABASE_ANON_KEY`
- `VITE_API_BASE_URL`（Cloud Run BFFのURL）
- `VITE_APP_ORIGIN`（例：[https://app.example.com）](https://app.example.com）)

### 5.12.2 環境別

- dev/stg/prod で Pages の環境変数を分ける
- stg は `stg` Supabase と `stg` Cloud Run を向ける

---

## 5.13 Cloudflare Pagesデプロイ設計（最小）

- GitHub連携で自動デプロイ
- Build command：
  - `npm ci && npm run build`
- Output directory：
  - `dist`
- SPAルーティング対応：
  - 404をindex.htmlにフォールバックする設定を有効化（PagesのSPA設定）

---

## 5.14 セキュリティ（フロント観点）

- Supabase anon key は公開前提（RLSで守る）
- BFF の service role key / Stripe / LLM の秘密鍵はフロントに一切置かない（環境変数にも置かない）
- service_role key は絶対にフロントに置かない
- XSS対策
  - ユーザー入力（質問/回答）をHTMLとして解釈しない（dangerouslySetInnerHTML禁止）

追加XSS対策（必須運用ルール）：

- CSP（Content-Security-Policy）を可能な範囲で導入する（少なくとも `script-src` を制限）
- ユーザー入力をHTMLとしてレンダリングしない（Markdown表示を入れる場合はサニタイズ必須）
- 外部リンクは `rel="noopener noreferrer"` を付与（tabnabbing対策）
- 依存パッケージの脆弱性チェック（`npm audit` 等）をCIに組み込む（14章と連携）

- セッション/トークンの保存先は supabase-js に委譲する（独自にlocalStorageへ保存しない）
- XSS耐性をさらに上げたい場合の選択肢として「BFFでHttpOnly Cookieに載せ替える」方式がある（MVP後の検討事項）

---

## 5.15 フロントエンドのエラー収集（Sentry）

### 5.15.1 目的

- Sentry導入は任意だが、課金/AI/認証を含むため本番では推奨する
- フロントの例外・エラーを収集し、問題の早期検知と対応を可能にする
- ユーザー体験の向上と運用コストの削減を目指す
- 収集時は PII/本文を送信しない（`beforeSend` でマスキング/削除）
- タグとして `route`, `feature`, `env`, `request_id`（存在する場合）を付与する
- `release` に Git のコミットSHAを設定し、リリース単位での切り分けを可能にする

---

## 5.16 この章の凍結事項

- フロントは Vite + React + PWA + Cloudflare Pages（静的配信前提）
- CRUDはSupabase直、AI/課金はCloud Run BFF経由
- SSRは使わない（静的配信前提）
- OAuthコールバックは `/auth/callback` 固定
- 環境変数は `VITE`\_ で統一
- 認証は Google OAuth を主導線とし、Email/Password をフォールバックとして提供する
- `/login` 画面は Google OAuth を主導線、Email/Password をサブ導線として提供する
