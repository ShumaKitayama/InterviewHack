# 12. リリース・運用設計（Release / Ops）

## 12.1 方針

- 個人開発で最速リリースしつつ、課金・AI・個人情報の取り扱いに耐える運用を最初から持つ [oai_citation:0‡アプリ開発企画書：Interview Hack.pdf](sediment://file_00000000f2b471fa8c73d4b972a1615d)
- env（dev/stg/prod）を分離し、stgで決済/AI/権限（RLS）の最終検証を必須にする
- 監視は「決済」「クレジット整合」「AI失敗率/コスト」「RLS越境兆候」を最優先に設計する
- 障害時の基本戦略：止めるべきは「AI実行とクレジット処理」、閲覧系は可能な限り継続

---

## 12.2 環境構成（dev / stg / prod）

### 12.2.1 Supabase

- Supabaseプロジェクトを環境ごとに分ける（dev / stg / prod）
- 各環境でRLSは常時ON（devでOFFにしない）
- マイグレーションは全環境に同一順序で適用

### 12.2.2 Stripe

- dev/stg: Stripe Test mode（Webhook secretも環境別）
- prod: Stripe Live mode
- Price ID / Webhook Secret / API Key は環境変数で分離

### 12.2.3 OpenAI

- dev: 原則モック（または低頻度で実API）
- stg/prod: 実API
- モデル選定は環境変数で切替可能にする（例：STGは常に低コストモデル）

---

## 12.3 デプロイ設計（CI/CD）

### 12.3.1 リポジトリ/ブランチ戦略（最小）

- main: 本番
- develop: 開発（任意）
- PR必須、CI通過必須

### 12.3.2 CIで必須のチェック

- Lint / Format
- Unitテスト（BFF）
- Integrationテスト（少なくともクレジット整合・Webhook冪等・RLS越境）
- マイグレーション差分検査（破壊的変更の検知）

### 12.3.3 CD（デプロイ順序）

1. DB migration（stg）
2. BFF deploy（stg）
3. スモークテスト（stg）
4. DB migration（prod）
5. BFF deploy（prod）
6. スモークテスト（prod）

---

## 12.4 マイグレーション運用（Supabase）

- 変更はすべて migration で管理（手作業で本番に当てない）
- 破壊的変更（DROP/型変更）は原則禁止
  - 代替：新カラム追加 → 移行 → 旧カラム廃止（残す）
- enumは text + CHECK制約 or マスタテーブルで表現（将来変更に強い）

---

## 12.5 リリース手順（チェックリスト）

### 12.5.1 リリース前（stgで必須）

- 認証（Google OAuth）ログイン成功
- RLS越境テスト（U1/U2）
- クレジット購入（Stripe test）→ Webhook → 残高反映
- Webhook再送（同一external_ref）で二重付与が起きない
- AI実行（成功）で残高が減る
- AI実行（失敗）でreversedとしてロールバックされ残高が戻る
- 主要フロー（生成→保存→復習、面接ログ→添削→保存）が完走
- ESフロー（企業研究資料登録→要約→ESドラフト生成→推敲→保存）が完走
- ES系API（/ai/research-summarize, /ai/es-generate, /ai/es-refine）の失敗時に残高がロールバックされる

### 12.5.2 本番反映（prod）

- 環境変数（Stripe Live / OpenAI / Supabase keys）が正しい
- Webhook endpoint がLive側に正しく設定され、署名検証できる
- 価格（Price ID）がprodのIDに差し替わっている

---

## 12.6 監視・ログ設計（Observability）

### 12.6.1 監視対象（最優先）

- 決済系
  - Webhook 失敗率（署名検証失敗/処理エラー）
  - purchase transaction 記録件数の急変
- クレジット整合
  - balance < 0 が発生していないこと（発生したら即アラート）
- consume と reversed の異常比率（失敗増加の兆候）
- AI運用
  - /ai/\* の成功率/失敗率
  - 出力スキーマ不正率
  - 平均レイテンシ
  - 推定コスト（日次/月次）
- セキュリティ
  - 401/403の急増（攻撃兆候）
  - レート制限（429）の急増

### 12.6.2 ログの種類

- アプリログ（クライアント）
  - 画面遷移・操作・エラー（本文データは送らない）
- BFFログ（サーバー）
  - request_id（相関ID）を必ず付与
  - StripeイベントID / external_ref をログに含める（PIIなし）
- DB監査ログ（アプリDB）
- credit_transactions（purchase/consume + status: pending/committed/reversed）
  - generation_jobs（成功/失敗/コスト）

### 12.6.3 アラート条件（例）

- Stripe webhook error rate > 1%（5分平均）
- /ai/\* failure rate > 5%（10分平均）
- OUTPUT_SCHEMA_INVALID が急増（10分でN件）
- credit_balance < 0 が1件でも発生（即時）
- purchaseが突然0（決済導線死） or 異常増（不正疑い）

---

## 12.7 障害対応（Incident Response）

### 12.7.1 重大度定義（例）

- SEV1: 決済・クレジットが壊れている（付与されない/二重付与/残高負）
- SEV2: AIが広範囲に失敗（reversedロールバックはされるが使えない）
- SEV3: 一部機能の不具合（表示崩れ等）

### 12.7.2 初動（SEV1/2共通）

1. 影響範囲の特定（いつから、どのエンドポイント、何件）
2. 収束策の適用
   - AI実行を一時停止（feature flag で /ai/\* を止める）
   - Checkout作成を止める（/billing/checkout-session を止める）
3. ログ・DBで原因切り分け
4. 復旧後に補償対応（不足クレジット付与、二重付与の調整）

### 12.7.3 事後（Postmortem）

- 原因、再発防止、検知の改善（アラート閾値見直し）
- テストケースの追加（11章へフィードバック）

---

## 12.8 ロールバック戦略

- BFFは前バージョンへ即時ロールバック可能にする（デプロイ基盤で対応）
- DBは原則ロールバックしない前提で、前方互換なマイグレーションを徹底
- 破壊的変更が必要な場合は段階移行（新カラム追加→移行→切替）

---

## 12.9 バックアップ・リストア

- Supabaseのバックアップ機能（スナップショット/point-in-time）が使える場合は有効化
- 最低限の復旧目標
  - RPO（許容データ損失）：24h以内（MVP）
  - RTO（復旧時間）：数時間以内（MVP）
- 重要テーブル（credit\_\* / generation_jobs）は復旧対象最優先

---

## 12.10 サポート・問い合わせ運用

- 問い合わせ導線
  - 設定画面に「問い合わせ」リンク
  - 不具合報告テンプレ（発生日時、画面、操作、request_id）
- 課金問い合わせ（最優先）
  - 購入したのに反映されない
  - 二重に消費された
  - 返金されない
- 対応フロー
  1. credit_transactions と external_ref を確認
  2. Stripe側のイベントと突合
  3. 付与/調整（bonus/adjustment transaction）で対応

---

## 12.11 フィーチャーフラグ（運用の安全弁）

- 以下はフラグでON/OFF可能にする
  - AI実行（機能別に止められる）
  - 特定モデルの利用（高コストモデルを止める）
  - Checkout作成（決済導線を止める）
- フラグはDB（admin only）または環境変数で管理（MVPは環境変数で良い）

---

## 12.12 この章の凍結事項

- env分離（dev/stg/prod）とstgでの決済・RLS・AI検証を必須化 [oai_citation:1‡アプリ開発企画書：Interview Hack.pdf](sediment://file_00000000f2b471fa8c73d4b972a1615d)
- 監視は「決済」「クレジット整合」「AI失敗率/コスト」を最優先
- Webhookは署名検証 + 冪等 + トランザクション
- DBは前方互換マイグレーションを原則とし、ロールバックに頼らない
