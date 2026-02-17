# 6. 認証・アカウント・課金（Supabase + Google OAuth + クレジット方式）

## 6.1 方針（固定）

- 認証基盤：Supabase Auth を採用
- 認証方式：Google OAuth を主導線として提供（フォールバックとして Email/Password もサポート）
- 課金方式：サブスクではなくクレジット購入方式
- クレジット残高・消費は Supabase(Postgres) で管理し、サーバー側で強制する
- 決済基盤：Stripe（Checkout + Webhook）を採用

---

## 6.2 認証（Supabase Auth）

### 6.2.1 サポートするログイン手段

- Google OAuth（必須）
- Email/Password（任意：MVPで入れるなら入れる。入れないなら設計だけ残す）

### 6.2.2 Google OAuth（要件）

- ログイン/新規登録を「Googleで続行」1ボタンで完結
- 初回ログイン時に user プロフィール行を自動作成（後述の profiles）
- 既存ユーザーは同じGoogleアカウントで再ログイン可能
- ログイン状態はアプリ再起動後も維持される（Supabaseセッション）

### 6.2.3 セッション管理

- Supabaseセッション（access_token + refresh_token）を利用
- クライアントはセッションを安全に保持（OSの安全領域/Keychain相当）
- 失効時は自動で再認証（refresh）→失敗時のみログイン画面へ

### 6.2.4 アカウント紐付け方針（GoogleとEmailの共存）

- 原則：Supabaseのユーザー識別子（auth.users.id）を唯一の user_id とする
- 同一メールアドレスで
  - 先にEmail/Password登録 → 後からGoogleでログイン
  - 先にGoogle → 後からEmail/Passwordで登録
    の“アカウント重複”が起きうる
- MVP方針：
  - Google OAuth を主導線とする
  - Email/Password はフォールバック手段として提供する
  - 「同一メールのアカウント統合」はサポートしない（別アカウント扱い）
  - 上記制約はヘルプ/FAQに明記する
- 将来拡張：
  - Supabase の identity linking を使った統合機能を検討可能（MVP外）

### 6.2.5 パスワードリセット（Email/Passwordを入れる場合のみ）

- Supabaseの標準フローを利用
- エラー文言はユーザー向けに整形

---

## 6.3 アカウント（DB設計：Supabase Postgres）

### 6.3.1 profiles（ユーザープロフィール）

- 目的：auth.users はSupabase管理領域なので、アプリ独自情報は profiles に持つ
- テーブル：profiles
  - user_id (PK, FK -> auth.users.id)
  - display_name (nullable)
  - created_at
  - updated_at

### 6.3.2 マルチテナント（データ分離）

- 全ドメインデータ（assets / companies / interviews / qa / etc）は user_id を持つ
- RLS（Row Level Security）で `auth.uid() = user_id` を強制
- クライアントはuser_idを信用せず、サーバー/DB側で強制する

### 6.3.3 アカウント削除

- MVP方針（推奨）
  - アプリ側：退会リクエストを受け付ける
  - 実データ：論理削除 or 完全削除（どちらかを先に固定）
- Supabase Auth のユーザー削除は管理権限が必要になるため
  - 「退会＝利用停止（論理削除）」を先に実装し、完全削除は後回しでも良い

---

## 6.4 課金モデル（クレジット方式）

### 6.4.1 基本コンセプト

- ユーザーはクレジットを購入し、AI機能利用時に消費する
- クレジット残高が足りない場合はAI機能を実行できない
- 価格/消費量は後から調整可能な設計にする（DBで管理）

### 6.4.2 クレジット商品の例（初期案）

- 100 credits
- 500 credits
- 1000 credits

※ 価格は後で決める。Stripe側は Product/Price で管理。

### 6.4.3 クレジット消費ルール（初期案）

- 想定Q&A生成：10 credits
- 面接ログ添削：5 credits
- 要約生成：2 credits
- 再生成：対象機能と同額

※ 実際のトークン・APIコストに応じて調整する。

---

## 6.5 Stripe連携（購入処理）

### 6.5.1 使用コンポーネント

- Stripe Checkout：購入画面
- Stripe Webhook：支払い成功通知（サーバーで受信）
- 自前DB（Supabase Postgres）：クレジット残高・履歴の管理

### 6.5.1.1 環境分離（Test / Live）

- stg 環境では Stripe Test Mode を使用する
- prod 環境では Stripe Live Mode を使用する
- Webhook secret / API Key / Price ID は環境ごとに完全に分離する

### 6.5.2 購入フロー（シーケンス）

1. クライアント：購入する商品（例：100 credits）を選択
2. サーバー：Checkout Session を作成
   - metadata に `user_id(auth.users.id)` と `credit_package_id` を入れる
3. クライアント：Stripe Checkout に遷移して支払い
4. Stripe：支払い成功イベントをWebhook送信
5. サーバー：Webhook署名検証 → イベント重複防止 → クレジット付与
6. DB更新：credit_balance を加算、取引履歴を保存
7. クライアント：残高を再取得して反映

### 6.5.3 Webhookで必須の安全要件

- 署名検証（Webhook secret）
- イベントの重複処理防止：
  - Stripe の `event.id` を必ず保存し、一意制約で二重処理を防止する
  - `external_ref`（checkout_session / payment_intent）とも突合する
- DB更新はトランザクションで実施（残高 + 履歴を同時に確定）

---

## 6.6 クレジット管理（DB）

### 6.6.1 balances（残高）

- テーブル：credit_balances
  - user_id (PK, FK -> auth.users.id)
  - balance (int, >= 0)
  - updated_at
- 制約：balance >= 0 の CHECK 制約を付与する

### 6.6.2 transactions（履歴：監査）

- テーブル：credit_transactions
  - id (PK)
  - user_id (FK)
  - type (enum: purchase, consume, bonus, adjustment)
  - amount (int: +100 / -5)
  - status (enum: pending, committed, reversed)
  - reason (text: "qa_generate", "interview_review", "stripe_session:xxx" 等)
  - external_ref (text: Stripeのpayment_intentやcheckout_session等 / 生成ジョブID等)
  - created_at

制約・ルール：

- (user_id, external_ref, type) に一意制約を付け、二重処理を防止する（Stripeの場合は external_ref に加えて event.id も別テーブルまたは同テーブルで一意管理する）
- status=pending は「予約」、committed は「確定」、reversed は「取消（ロールバック）」を表す
- 残高（credit_balances）は committed の合計のみを反映させる
- (user_id, created_at) にインデックスを張り、履歴表示を高速化する

### 6.6.3 packages（商品マスタ：アプリ側の柔軟性）

- テーブル：credit_packages
  - id (PK)
  - name (text: "100 credits")
  - credits (int)
  - stripe_price_id (text)
  - is_active (bool)

### 6.6.4 costs（機能ごとの消費マスタ）

- テーブル：credit_costs
  - feature (enum: qa_generate, answer_review, summary_generate, regenerate, research_summarize, es_generate, es_refine)
  - cost (int)
  - is_active (bool)

---

## 6.7 クレジット消費（実行時ルール）

### 6.7.1 原則（標準パターン：Reserve / Commit）

- クレジット判定・更新は必ずサーバー側で行う（クライアントの表示は信用しない）
- 「予約（reserve）」→「確定（commit）」の2段階で整合性を保つ
- 失敗時は「取消（reverse）」で元に戻す（差分返金ではなく状態遷移で管理）
- 冪等性のため、external_ref（例：generation_job_id）を必ず付与する

### 6.7.2 消費手順（標準フロー）

1. 残高チェック（balance >= cost）
2. トランザクション開始
3. credit_transactions に以下を INSERT
   - type=consume
   - amount = -cost
   - status = pending
   - external_ref = <generation_job_id 等>
4. credit_balances.balance を cost 分だけ減算（同一トランザクション内）
5. トランザクション commit（＝予約完了）
6. AI処理を実行
7. 成功時：
   - credit_transactions.status を committed に更新
8. 失敗時：
   - credit_transactions.status を reversed に更新
   - credit_balances.balance を cost 分だけ加算して元に戻す

補足：

- external_ref + type で重複 INSERT を防ぎ、同一リクエストの二重消費を防止する
- purchase（Stripe購入）は type=purchase, status=committed として直接確定して良い（予約フェーズ不要）

---

## 6.8 無料クレジット（オンボーディング）

- 初回登録時にボーナスクレジット付与（例：20 credits）
- 目的：クレカなしで価値体験させ、購入につなげる

実装:

- 初回ログイン検知（profiles作成時など）で
  - credit_balances が未作成なら作成 + bonus transaction を追加

---

## 6.9 RLS（Row Level Security）方針

- profiles / credit_balances / credit_transactions / assets / companies / interviews / qa 等は RLS を有効化
- 基本ポリシー
  - SELECT/INSERT/UPDATE/DELETE は `auth.uid() = user_id` のみ許可
- 例外
  - credit_packages / credit_costs は読み取りのみ公開（is_active=true のみ）

---

## 6.10 この章で凍結する仕様（合意事項）

- 認証：Supabase Auth（Google OAuth 主導線 + Email/Password フォールバック）
- 課金：Stripe + クレジット方式
- クレジット管理：Supabase(Postgres)で残高 + 履歴を保持
- クレジット消費：Reserve/Commit の2段階方式で管理し、失敗時は reversed に遷移させてロールバックする
- 初回ボーナスクレジットを付与する
