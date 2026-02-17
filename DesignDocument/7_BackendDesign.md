# 7. バックエンド / API設計（Supabase + クレジット課金 + OpenAI）

## 7.1 方針（アーキテクチャの前提）

- DB：Supabase Postgres（RLSで user_id 分離を強制）
- 認証：Supabase Auth（Google OAuth 主導線 + Email/Password フォールバック）
- バックエンド：BFF（Backend For Frontend）を用意する
  - 役割：Stripe Checkout作成、Stripe Webhook受信、OpenAI呼び出し、クレジット消費のトランザクション管理
- フロントは原則 Supabase の CRUD を直接使って良い
  - ただし「クレジットを消費する処理」「Stripe」「OpenAI」は必ずBFF経由
- 生成結果はDBに保存し、同条件の再生成を抑制できる設計にする [oai_citation:0‡アプリ開発企画書：Interview Hack.pdf](sediment://file_00000000fc1071faae3e4aa768ae8b73)

---

## 7.2 バックエンド設計方針（DDD + クリーンアーキテクチャ）

本プロジェクトのBFF（Go + Gin）は、**DDD（ドメイン駆動設計）を意識したクリーンアーキテクチャ**を採用する。
目的は「ビジネスルールをフレームワーク/外部SDKから分離し、変更容易性とテスト容易性を最大化する」こと。

### 7.2.1 依存関係ルール（必須）

依存方向は内向き（外側→内側）に限定する：

- Domain（最内核）：ビジネスルール（エンティティ/値オブジェクト/ドメインサービス）
- Application / Usecase：ユースケース（フロー制御、トランザクション境界、外部I/Fの呼び出し順序）
- Interface / Adapter：HTTPハンドラ（Gin）、DTO変換、入力バリデーション、エラー整形
- Infrastructure：DB（Supabase/Postgres）、Stripe/OpenAI/Gemini等の外部SDK、ログ、設定

**禁止事項（重要）**

- Domain は `gin` / `supabase` / `stripe` / `openai` / `gemini` 等の外部ライブラリを import してはならない
- Application/Usecase はフレームワーク依存（GinのContext等）を持たない
- Infrastructure 以外の層から「ネットワーク呼び出し」「DB直接操作」をしない

### 7.2.2 境界（DTO / Entity / Persistence Model）

- HTTP入力/出力は DTO として Interface/Adapter 層で定義する
- Domain は「永続化都合のフィールド」を持たない（必要なら別モデルに分離する）
- Repository/External Client の interface は Application（またはDomain）側に定義し、実装は Infrastructure に置く
- 変換（DTO↔Usecase input↔Domain↔Persistence）は Adapter/Infra で行い、Domain を汚さない

### 7.2.3 Goの推奨パッケージ構成（例）

```
backend/
  cmd/
    api/                 # main（Gin起動、DI）
  internal/
    domain/
      entity/
      vo/                # value objects
      service/
      error/             # domain error definitions (optional)
    usecase/             # application layer
      credit/
      billing/
      ai/
    port/                # interfaces (inbound/outbound)
      repository/
      external/
    adapter/
      http/
        handler/
        middleware/
        presenter/       # error/response shaping
        dto/
    infra/
      supabase/
      stripe/
      llm/               # openai/gemini clients
      logging/
      config/
```

備考：

- `internal/port` は「依存性逆転」のためのinterface置き場（Repository/External client）
- `adapter/http` はGinに依存してよいが、`domain` と `usecase` は依存しない
- `infra/*` はSDK/DBに依存してよい（最外層）

### 7.2.4 この設計で特に重要なユースケース

- クレジット消費（Reserve/Commit/Reversed）：Usecase層でトランザクション境界を持つ（7.6参照）
- Stripe Webhook：冪等性（event.id）と残高/履歴の同時確定を Usecase で統括（7.5.2参照）
- AI実行：クレジット予約→AI→確定/取消の一連を Usecase として実装し、Infra は「呼び出し手段」に徹する

## 7.3 コンポーネント構成

- Client（iOS/Android）
  - Supabase Auth でログイン
  - Supabase DB へ通常CRUD（自己資産/企業/面接/Q&A）
  - 生成・課金はBFF APIを叩く
- Supabase
  - Auth（Google OAuth）
  - Postgres（RLS）
- BFF（HTTP API）
  - /billing/（Stripe）
  - /ai/（OpenAI + credit consume）
- External
  - Stripe
  - OpenAI API（モデルはコスト優先で軽量を基本） [oai_citation:1‡アプリ開発企画書：Interview Hack.pdf](sediment://file_00000000fc1071faae3e4aa768ae8b73)

---

## 7.3 データアクセス方針（RLSと責務分離）

### 7.3.1 クライアントが直接CRUDしてよい領域

- assets（自己資産）
- companies（企業）
- research_materials（企業研究資料）
- es_projects（ESプロジェクト）
- es_sections（ESセクション）
- es_project_asset_relations / es_project_research_material_relations（ES参照関連）
- interviews（面接）
- interview_logs（面接ログ）
- qa（Q&A）
- profiles（ユーザープロフィールの読み取り・一部更新）
- credit_balances（読み取りのみ）
- credit_transactions（読み取りのみ）
- credit_packages / credit_costs（読み取りのみ）

### 7.3.2 BFFのみが書き込みする領域

- credit_balances（残高更新）
- credit_transactions（購入/消費/ロールバックの記録）
- generation_jobs（OpenAI呼び出し履歴、コスト、失敗理由など）

理由：

- クレジットと生成処理は不正改ざん対象なので、クライアント書き込みを禁止する

---

## 7.4 認証・認可（APIセキュリティ）

- クライアントは Supabase の JWT を取得して BFF に付与
  - Header: Authorization: Bearer <supabase_access_token>
- BFF は supabase JWT を検証し、user_id を確定する
- BFF はリクエストボディの user_id を信用しない（必ずJWTのsubから採用）
- RLSにより、DB直アクセスでも user_id 越境を防ぐ

---

## 7.5 API一覧（エンドポイント設計）

### 7.5.1 Health / Meta

- GET /health
  - 200: { "status": "ok" }

### 7.5.2 Billing（Stripe）

- GET /billing/packages
  - 目的：購入可能なクレジット商品一覧
  - 200: [{ id, name, credits, priceLabel, stripePriceId }]
  - 備考：priceLabelはUI表示用（Stripeから取得orDB保持）
- POST /billing/checkout-session
  - 目的：Stripe Checkout Session 作成
  - Auth：必須
  - Req:
    - packageId: string（credit_packages.id）
  - Res:
    - checkoutUrl: string
  - 失敗:
    - 400 INVALID_PACKAGE
    - 401 UNAUTHORIZED
    - 500 STRIPE_ERROR
- POST /billing/webhook
  - 目的：Stripeイベント受信（署名検証必須）
  - Auth：不要（Stripe署名で担保）
  - 受信イベント例：
    - checkout.session.completed（支払い成功）
    - payment_intent.succeeded（用途次第）
  - 成功時処理：
    - Stripe の event.id（stripe_event_id）を保存し一意制約で重複排除し、併せて external_ref（session/payment_intent）とも突合する（idempotency）
    - credit_transactions(purchase) 追加
    - credit_balances に credits 加算
  - Res:
    - 200 OK（処理済み/重複でもOK）
    - 400（署名不正）

### 7.5.3 Credit（残高・履歴）

- GET /credits/balance
  - Auth：必須
  - 200: { balance: number }
- GET /credits/transactions?limit=50&cursor=...
  - Auth：必須
  - 200: { items: [...], nextCursor: string|null }

※ クライアントはSupabase直読みでもよいが、将来の集計/整形のためBFF経由も用意可能

### 7.5.4 AI（OpenAI + クレジット消費）

#### 共通仕様

- Auth：必須
- 失敗時はクレジット取引を status=reversed に遷移させ、残高をロールバックする（7.6参照）
- OpenAI出力はJSONスキーマで検証し、壊れていたらリトライ（最大回数を制限）
- POST /ai/qa-generate
  - 目的：事前対策の想定Q&A生成 [oai_citation:2‡アプリ開発企画書：Interview Hack.pdf](sediment://file_00000000fc1071faae3e4aa768ae8b73)
  - 消費クレジット：credit_costs.feature="qa_generate" を参照
  - Req:
    - companyId: string
    - jobTitle: string（省略可：companyにあればそれを使う）
    - assetIds: string[]（省略可：空ならサーバーが自動選抜）
    - options:
      - questionCount: number（default 10）
      - regenerateMode: "all"|"questions_only"|"answers_only"（任意）
  - Res:
    - generationId: string
    - items: [{ questionText, answerText, referencedAssetIds: string[] }]
  - 副作用：
    - qa テーブルに保存（保存する/しないは options で選択しても良い）
    - generation_jobs に記録
- POST /ai/answer-review
  - 目的：面接ログ添削（事後対策） [oai_citation:3‡アプリ開発企画書：Interview Hack.pdf](sediment://file_00000000fc1071faae3e4aa768ae8b73)
  - 消費クレジット：credit_costs.feature="answer_review"
  - Req:
    - questionText: string
    - answerText: string
    - assetIds: string[]（省略可：自動選抜）
    - options:
      - strictness: "normal"|"hard"（任意）
  - Res:
    - generationId: string
    - issues: string[]
    - improvements: string[]
    - improvedAnswerText: string
    - referencedAssetIds: string[]
  - 副作用：
    - 必要なら interview_logs にAI結果を書き戻す（フロントが保存でも良い）
    - generation_jobs に記録
- POST /ai/summary-generate
  - 目的：ダッシュボード用「次回直すべきポイント要約」 [oai_citation:4‡アプリ開発企画書：Interview Hack.pdf](sediment://file_00000000fc1071faae3e4aa768ae8b73)
  - 消費クレジット：credit_costs.feature="summary_generate"
  - Req:
    - interviewId?: string（直近の面接単位）
    - interviewLogIds?: string[]（複数ログ指定）
  - Res:
    - generationId: string
    - summaryBullets: string[]（3〜5個）
  - 副作用：
    - 保存先は後で決める（home用のlatest_summaryテーブルなど）

- POST /ai/research-summarize
  - 目的：企業研究資料の要約・構造化（セット2作成支援）
  - 消費クレジット：credit_costs.feature="research_summarize"
  - Req:
    - type: "url" | "text" | "pdf"
    - content: string（URL または テキスト または Base64/S3Key）
  - Res:
    - summary: string（要約テキスト）
    - keyPoints: string[]（重要なポイント箇条書き）
    - usageSuggestion: string（「志望動機に使えそう」などの提案）

- POST /ai/es-generate
  - 目的：ESドラフト生成（セット1＋セット2統合）
  - 消費クレジット：credit_costs.feature="es_generate"
  - Req:
    - esProjectId: string
    - sectionTitle: string（例：「志望動機」）
    - assetIds: string[]（使用する自己資産）
    - researchMaterialIds: string[]（参照する企業研究資料）
    - characterLimit: number
  - Res:
    - generationId: string
    - draftText: string
    - rationale: string（構成の意図・解説）

- POST /ai/es-refine
  - 目的：ES推敲・ブラッシュアップ
  - 消費クレジット：credit_costs.feature="es_refine"
  - Req:
    - currentText: string
    - instructions: string（修正指示）
  - Res:
    - generationId: string
    - refinedText: string
    - changes: string[]（変更点リスト）

---

## 7.6 クレジット消費のトランザクション設計（重要）

### 7.6.1 基本戦略（標準：Reserve / Commit）

- クレジットは「予約（pending）」→「確定（committed）」の2段階で管理する
- 失敗時は「取消（reversed）」へ遷移させ、残高をロールバックする
- 冪等性のため、external_ref（例：generation_job_id）を必ず付与する
- クライアントの表示残高は信用せず、必ずサーバー側で判定・更新する

### 7.6.2 DB更新の原則（同時実行耐性）

- balance 更新と transactions 追加/更新は必ず同一トランザクションで行う
- 残高行は `SELECT ... FOR UPDATE` 等でロックし、同時消費競合を防ぐ
- 手順（要約）：
  1. 残高チェック（balance >= cost）
  2. credit_transactions に type=consume, amount=-cost, status=pending, external_ref を INSERT
  3. credit_balances.balance を cost 分だけ減算
  4. コミット（予約完了）
  5. AI処理を実行
  6. 成功時：credit_transactions.status = committed
  7. 失敗時：credit_transactions.status = reversed に更新し、balance を cost 分戻す
- 残高不足時は 402（PAYMENT_REQUIRED 相当）で返す
  - エラーコード：INSUFFICIENT_CREDITS

補足：

- purchase（Stripe購入）は type=purchase, status=committed として直接確定して良い（予約フェーズ不要）
- external_ref + type の一意性で二重消費を防止する

---

## 7.7 エラー設計（API共通）

### 7.7.1 エラーレスポンス形式（統一）

- status: "error"
- code: string（機械可読）
- message: string（ユーザー向け文言）
- detail?: object（開発者向け）

例：

- 401 UNAUTHORIZED
- 403 FORBIDDEN
- 404 NOT_FOUND
- 409 CONFLICT（重複、整合性）
- 422 VALIDATION_ERROR
- 429 RATE_LIMITED
- 500 INTERNAL_ERROR
- 502 OPENAI_ERROR / STRIPE_ERROR

### 7.7.2 HTTPステータスとエラーコード対応表

方針：

- HTTPステータスは「通信/権限/リソース状態」を表す
- `code` はアプリケーション固有の機械可読エラー種別
- クライアントは `status` と `code` の両方で分岐して良い

対応表（例）：

- 400 BAD_REQUEST
  - VALIDATION_ERROR
  - INVALID_PACKAGE
- 401 UNAUTHORIZED
  - UNAUTHORIZED
- 403 FORBIDDEN
  - FORBIDDEN
- 404 NOT_FOUND
  - NOT_FOUND
- 409 CONFLICT
  - CONFLICT
  - DUPLICATE_EVENT
- 422 UNPROCESSABLE_ENTITY
  - VALIDATION_ERROR
- 429 TOO_MANY_REQUESTS
  - RATE_LIMITED
- 402 PAYMENT_REQUIRED
  - INSUFFICIENT_CREDITS
- 500 INTERNAL_SERVER_ERROR
  - INTERNAL_ERROR
- 502 BAD_GATEWAY
  - OPENAI_ERROR
  - STRIPE_ERROR

運用ルール：

- Webhookの重複受信は 200 を返しつつ、`code=DUPLICATE_EVENT` をログに記録する
- クライアントに表示する文言は `message`、分岐は `code` を使用する

### 7.7.3 リクエストID（相関ID）

- BFFは全リクエストに対して `request_id`（UUID推奨）を生成し、以後のログ/レスポンスに一貫して付与する
- 受信ヘッダに `X-Request-Id` があればそれを採用し、なければサーバー側で生成する
- すべてのレスポンスヘッダに `X-Request-Id: <request_id>` を付与する
- 例外/エラー時も必ず同じ `request_id` を返す（クライアントとログの突合のため）

---

補足（テスト方針）：

- Domain / Usecase は外部I/Fをinterface化し、モックで純粋ユニットテスト可能にする
- Adapter/Infra は統合テストで担保する
- 最終的なテスト戦略は `DesignDocument/11_TestDesign.md` を正とする

## 7.8 レート制限（コスト防御）

- user_id 単位でAIエンドポイントにレート制限
  - 例：/ai/ は 1分あたりN回
- Stripe Checkout作成も乱発防止（1分あたり数回）
- 429時はリトライ可能時間を返す（Retry-After）

## 7.8.1 構造化ログ（Cloud Run向け）

目的：

- 決済（Webhook）/クレジット/AI失敗を「一意なID」で追跡し、再現なしで原因特定できる状態にする

方針：

- Cloud Runの標準出力（stdout/stderr）にJSON形式でログを出力し、Cloud Loggingで収集する
- ログは「機械可読（JSON）」「PIIを含めない」「相関可能（request_id）」を必須要件とする

ログ共通フィールド（全リクエスト）：

- `severity`: "INFO" | "WARN" | "ERROR"
- `timestamp`: RFC3339
- `request_id`: 7.7.3 の相関ID
- `method`: HTTPメソッド
- `path`: ルート（例："/ai/qa-generate"）
- `status`: HTTPステータス
- `latency_ms`: レスポンスまでの処理時間
- `user_id`: JWT検証後に取得できる場合のみ（未認証/検証失敗時は出さない）
- `error_code`: 7.7 の `code` と一致（成功時はnull）

決済（Stripe）ログ追加フィールド：

- `stripe_event_id`: StripeのイベントID
- `external_ref`: checkout_session_id / payment_intent_id
- `package_id`: credit_packages.id（分かる場合）
- `credits_delta`: 付与したクレジット量（purchase時）

AIログ追加フィールド：

- `job_id`: generation_jobs.id
- `provider`: "openai" | "gemini"
- `model`: 使用モデル名
- `retry_count`: 出力検証/再試行の回数
- `token_used`: 取得できる場合のみ
- `cost_estimate`: 取得できる場合のみ

禁止（ログに絶対出さない）：

- 面接ログ/自己資産/志望動機などの本文
- Authorizationヘッダ、JWT、生のStripe署名ヘッダ、APIキー
- メールアドレス等の個人情報（必要ならハッシュ化/マスキング）

運用：

- 重大障害時の一次調査は `request_id` を起点に「BFFログ → generation_jobs / credit_transactions → Stripeイベント」へ辿る
- Webhook処理はリトライされるため、`stripe_event_id` と `external_ref` を必ずログとDBに残す

---

## 7.9 生成ジョブ管理（generation_jobs）

- 記録する項目
  - user_id
  - job_type（qa_generate / answer_review / summary_generate / research_summarize / es_generate / es_refine）
  - input_snapshot（要約JSON）
  - model_name
  - token_used（可能なら）
  - cost_estimate（可能なら）
  - status（success/failed）
  - error_code / error_message
  - created_at
- 目的
  - 失敗解析
  - コスト分析
  - 不正検知（短時間大量生成）

---

## 7.10 データフロー（要約）

### 7.10.1 事前対策（想定Q&A）

- Client: companyId選択 → BFF /ai/qa-generate
- BFF: asset/company取得 → クレジット消費 → OpenAI → スキーマ検証 → 保存 → 응答

### 7.10.2 事後対策（添削）

- Client: question/answer入力 → BFF /ai/answer-review
- BFF: asset取得 → クレジット消費 → OpenAI → スキーマ検証 → 응答
- Client: interview_logs に保存（またはBFFが保存）

### 7.10.3 購入（Stripe）

- Client: package選択 → BFF /billing/checkout-session → Checkout
- Stripe: webhook → BFF /billing/webhook → credit付与

---

## 7.11 凍結ポイント（この章の合意事項）

- AI/課金はBFF経由（クライアント直叩き禁止）
- Supabase JWTでBFF認証、user_idはJWT由来のみ信用（Authは Google 主導線 + Email/Password フォールバック）
- クレジット消費は Reserve/Commit の2段階方式（pending → committed / reversed）で管理する
- StripeはWebhookでのみクレジット付与（クライアントから付与不可）
- BFFはDDD + クリーンアーキテクチャの依存方向（Domainに外部依存を入れない）を遵守する
