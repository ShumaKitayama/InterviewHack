# 11. テスト設計（詳細版：具体テストケース定義）

※ テスト方針の正は `DesignDocument/11_TestDesign.md` 本書および以下に従う：

- アーキテクチャ：DesignDocument/7_BackendDesign.md（DDD + クリーンアーキテクチャ）
- DB境界/RLS：DesignDocument/8_DBDesign.md
- セキュリティ/冪等性：DesignDocument/9_SecurityDesign.md
- CI/CD：DesignDocument/14_CICDDesign.md

## 11.0 前提（テスト環境・データ）

### 環境

- env: dev / stg / prod を分離
- Stripe: test mode（テスト用Webhook secret）
- OpenAI:
  - 原則モック（HTTPスタブ）で自動テスト
  - 手動/E2Eでのみ実APIも実行（コスト管理）
- Supabase:
  - stg用プロジェクトを用意
  - RLSはstgで必ず有効（devで無効化しない）

### テストユーザー

- U1: Google OAuth ユーザー（通常）
- U2: Google OAuth ユーザー（別アカウント、RLS検証用）
- U3: 不正JWT（署名不正・期限切れ・改ざん）

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

### テストの合格条件（共通）

- DB整合性が崩れない（残高が負にならない、二重付与なし、参照関係が壊れない）
- APIは規定のエラーコード/メッセージを返す
- UI操作で「戻れない」「二重送信」「無限ローディング」にならない

---

## 11.1 アーキテクチャ前提（DDD + クリーンアーキテクチャとテスト）

本プロジェクトのバックエンドは **DDD + クリーンアーキテクチャ** に従う（正は `DesignDocument/7_BackendDesign.md`）。
テストはレイヤー責務に従って以下の方針で実施する。

- **Domain / Usecase**：
  - 外部I/F（DB, Stripe, LLM 等）は interface に抽象化し、モック/スタブで **純粋ユニットテスト**を行う
  - ビジネスルール（クレジット状態遷移、検証ロジック、分岐）はここで網羅する
- **Adapter / Infrastructure**：
  - Supabase / Stripe / LLM スタブを用いた **統合テスト**で結合を検証する
  - HTTP ハンドラは薄く保ち、入出力変換とエラー整形の確認を中心にする
- **E2E**：
  - 主要ユーザーフローを本番相当の経路で検証する

禁止事項：

- Domain にフレームワークや SDK 依存を持ち込む前提のテストを書かない
- ハンドラや Infra にビジネスロジックがある前提のテストを書かない

## 11.1 テスト分類（何をどこまでやるか）

- Unit（BFF）
  - クレジット消費・ロールバック、Stripe検証、OpenAIレスポンス検証、入力バリデーション
- Integration（BFF + Supabase + Stripe/OpenAI stub）
  - トランザクション整合性、RLS越境阻止、冪等性
- E2E（アプリ）
  - 主要ユーザーフロー（登録→生成→保存→復習、購入→残高反映→生成）
- Manual（実機/運用）
  - UI/UX、通知（あれば）、ネットワーク断、バックグラウンド復帰、ストア配布前チェック

---

## 11.2 Unitテスト（BFF）— 具体ケース

### 11.2.1 クレジット残高チェック

- UT-CR-001: balance > cost のとき許可
- UT-CR-002: balance = cost のとき許可（消費後0）
- UT-CR-003: balance < cost のとき拒否（INSUFFICIENT_CREDITS）
- UT-CR-004: cost が負/0/極端に大きい場合はサーバー側で拒否（VALIDATION_ERROR）
- UT-CR-005: feature が無効（credit_costs.is_active=false）の場合は拒否（FEATURE_DISABLED）

### 11.2.2 クレジット状態遷移（consume / reversed / committed）

- UT-TX-001: consume で balance が cost 分減る & consume transaction が記録される
- UT-TX-002: OpenAI失敗時に transaction が status=reversed になり、balance が元に戻る
- UT-TX-003: reversed の二重実行が起きても balance が過剰に戻らない（external_ref / status 遷移で防止）
- UT-TX-004: consume と reversed が同一トランザクション境界を壊さない（例外でも整合）
- UT-TX-005: トランザクション記録は status 遷移で管理され、履歴は監査可能である

### 11.2.3 同時実行（競合）テスト

- UT-CONC-001: 同ユーザーで同時にAIリクエスト2本 → balanceが負にならない
- UT-CONC-002: balance=cost の状態で同時2本 → 片方成功、片方はINSUFFICIENT_CREDITS
- UT-CONC-003: DBロック待ちがタイムアウトした場合、適切に 503/429 を返す

### 11.2.4 入力バリデーション

- UT-VAL-001: companyIdが不正UUID → 422
- UT-VAL-002: questionText 空 → 422
- UT-VAL-003: answerText 空 → 422
- UT-VAL-004: assetIds 1000件など異常値 → 422（上限設定）
- UT-VAL-005: textが最大長超過 → 422

### 11.2.5 OpenAIレスポンス検証（JSONスキーマ）

- UT-AI-001: 正常JSON → パース成功
- UT-AI-002: JSONだが必須フィールド欠損 → OUTPUT_SCHEMA_INVALID
- UT-AI-003: JSONパース不可 → OUTPUT_SCHEMA_INVALID
- UT-AI-004: referencedAssetIds に存在しないIDが混入 → サーバー側で除外 or エラー（方針固定）
- UT-AI-005: リトライ上限に達した場合 → OUTPUT_RETRY_EXHAUSTED

---

## 11.3 Integrationテスト（BFF + Supabase + Stripe/OpenAI stub）— 具体ケース

### 11.3.1 認証（Supabase JWT）

- IT-AUTH-001: 正常JWTで /ai/\* にアクセス → 200
- IT-AUTH-002: JWTなし → 401
- IT-AUTH-003: 期限切れJWT → 401
- IT-AUTH-004: 改ざんJWT → 401
- IT-AUTH-005: user_id をbodyに入れて別ユーザーを指定しても無視される（JWT sub優先）

### 11.3.2 RLS（越境防止）

- IT-RLS-001: U1 が U2 の assets を select できない
- IT-RLS-002: U1 が U2 の qa を update できない
- IT-RLS-003: U1 が credit_balances を update できない（クライアント権限拒否）
- IT-RLS-004: U1 が generation_jobs を insert できない（BFFのみ）

### 11.3.3 Stripe Checkout → Webhook → クレジット付与

- IT-STR-001: /billing/checkout-session でcheckoutUrlが返る
- IT-STR-002: checkout.session.completed（署名OK）→ creditが増える
- IT-STR-003: 同じイベントを再送 → credit二重付与されない（冪等）
- IT-STR-004: 署名NG → 400、DB更新なし
- IT-STR-005: packageIdが無効/停止 → checkout作成不可（INVALID_PACKAGE）
- IT-STR-006: Stripe側で支払い失敗 → クレジット付与されない
- IT-STR-007: metadata改ざん相当（別user_id）でも、最終的に付与先が正しい（方針固定）
  - 推奨：checkout session作成時に紐付けた user_id を server-side で保持し、それと照合

### 11.3.4 AI実行（成功/失敗/ロールバック）

- IT-AI-001: 残高十分 → consume(pending) → OpenAI成功 → committed → 残高減少
- IT-AI-002: OpenAIタイムアウト → status=reversed → 残高が元に戻る
- IT-AI-003: OpenAIレート制限 → status=reversed → 429/502で返す
- IT-AI-004: 出力スキーマ不正 → リトライ → 成功なら committed、失敗なら reversed + error
- IT-AI-005: 同一リクエストを短時間に連打 → レート制限（429）発動

### 11.3.5 DB整合性（参照関係）

- IT-DB-001: interviews.company_id が存在しない場合 insert できない（FK）
- IT-DB-002: interview_logs.interview_id が存在しない場合 insert できない
- IT-DB-003: qa_asset_relations の参照が両方存在しないと作れない
- IT-DB-004: es_sections.es_project_id が存在しない場合 insert できない
- IT-DB-005: es_project_asset_relations の参照が両方存在しないと作れない
- IT-DB-006: es_project_research_material_relations の参照が両方存在しないと作れない

### 11.3.6 ES作成支援（新規AIエンドポイント）

- IT-ES-001: /ai/research-summarize 正常時に summary/keyPoints が返る
- IT-ES-002: /ai/es-generate 正常時に draftText/rationale が返る
- IT-ES-003: /ai/es-refine 正常時に refinedText/changes が返る
- IT-ES-004: ES系API失敗時に credit_transactions が reversed となり残高が復元される
- IT-ES-005: 他ユーザーの es_project_id / research_material_id を指定しても越境参照できない

---

## 11.4 E2Eテスト（アプリ）— 主要フローをケース化

### 11.4.1 オンボーディング

- E2E-ONB-001: Googleログイン → 初回ボーナスクレジット付与 → 残高表示OK
- E2E-ONB-002: 途中でアプリを落として復帰 → セッション維持、入力状態保持

### 11.4.2 事前対策（Q&A生成）

- E2E-PREP-001: 自己資産3件登録 → 企業1件登録 → 生成 → Q&A保存 → バンクに表示
- E2E-PREP-002: asset不足（0件）で生成 → 入力不足ガイド（エラー文言が分かる）
- E2E-PREP-003: 生成中にキャンセル → 残高が返る or 消費されない（仕様どちらかで固定）

### 11.4.3 事後対策（添削）

- E2E-REV-001: 面接作成 → ログ入力（複数）→ 添削 → 改善案をQ&Aに保存
- E2E-REV-002: 添削失敗（ネット断）→ 再試行導線 → 入力保持

### 11.4.4 復習

- E2E-REVW-001: Q&Aバンク → 復習UI → review_status 変更が反映
- E2E-REVW-002: フィルタ（企業/未復習）で正しく絞り込める
- E2E-REVW-003: お気に入り切替が反映

### 11.4.5 クレジット不足→購入→再実行

- E2E-BILL-001: 残高不足で生成 → 購入導線 → Checkout → Webhook後に残高反映 → 再実行成功
- E2E-BILL-002: 購入直後にアプリを再起動 → 残高が再取得で反映
- E2E-BILL-003: 購入処理中にネット断 → 結果確認導線（「購入履歴を更新」など）で復帰できる

### 11.4.6 ES作成フロー

- E2E-ES-001: 企業選択 → 研究資料(URL/PDF)登録 → 要約生成 → ESプロジェクト作成が完走
- E2E-ES-002: asset/research material を選択して /ai/es-generate 実行 → draft表示 → 保存が反映
- E2E-ES-003: /ai/es-refine に修正指示を出し、本文差分が反映される
- E2E-ES-004: 文字数制限超過時にUIで警告され、保存可否が仕様通りに制御される

---

## 11.5 ネットワーク・端末状態テスト（実機で必須）

- M-NT-001: 低速回線（3G相当）で生成 → ローディング表示崩れない
- M-NT-002: 機内モードで生成 → 即エラー、入力保持、再試行可能
- M-NT-003: 生成中にバックグラウンド → 復帰時に状態が破綻しない
- M-NT-004: 画面回転（対応する場合）で入力が消えない

---

## 11.6 負荷・耐久（最小でもこれだけ）

- L-001: 1ユーザーが連続でAI生成20回（制限内想定）→ 残高整合OK、失敗時ロールバックOK
- L-002: 50ユーザー同時に /ai/answer-review → 429/timeout時のロールバック整合OK
- L-003: Stripe Webhook を100回連続送信（同一/別イベント混在）→ 二重付与0件

---

## 11.7 プロンプト回帰テスト（品質を壊さない）

### 固定入力セット（例）

- 入力：A1〜A6 + C1(job=Backend)
- 期待チェック（自動/半自動）
  - JSONとしてパースできる
  - questions 数が指定通り（10）
  - 各answerが空でない
  - referencedAssetIds が1件以上含まれる割合が一定以上（例：70%）

ケース：

- PR-001: qa-generate を毎回同入力で実行 → スキーマ崩れ率が閾値以下
- PR-002: answer-review で短文回答/長文回答の両方を検証
- PR-003: 重要プロンプト変更時に必ず実行

---

## 11.8 リリース判定（Go/No-Go）— チェック項目を固定

必須（1つでも落ちたらNo-Go）：

- G-001: Stripe購入→クレジット付与が100%成功（テストケース上）
- G-002: Webhook再送でも二重付与が発生しない
- G-003: AI失敗時は transaction を status=reversed に遷移させ、必ず残高をロールバックする
- G-004: RLS越境が起きない（U1/U2で検証）
- G-005: 主要フロー（生成/添削/保存/復習）が完走する
- G-006: 重大クラッシュがない（ログで確認）

---

## 11.9 テスト実装メモ（運用ルール）

- 自動テストはCIで必須（mainへのマージ条件）
- Stripe/Webhookはテストモードで統合テストを回す
- OpenAIはモックで回し、週1回だけ実APIでスモークテスト（コスト管理）
- 失敗したテストは「再現手順・期待値・実際値・ログ（request_id）」を必ず残す
- Domain/Usecase が外部SDKに依存していないことを CI でレビュー観点として確認する（7_BackendDesign 準拠）

---

## 11.10 この章の凍結事項（合意）

- 課金/クレジット/AI/権限（RLS）は最優先でテストする
- Webhookは署名検証 + 冪等 + トランザクションで整合性を保証する
- AI失敗時は transaction を status=reversed に遷移させ、必ず残高をロールバックする
- 本文データ（面接内容）はテストログや分析基盤に出さない（メタデータのみ）
