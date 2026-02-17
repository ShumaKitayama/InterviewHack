# 9. セキュリティ・プライバシー・法務（決済周り含む）

## 9.1 方針（最上位）

- 本プロダクトは「職歴・面接ログ・志望動機」など高感度な個人データを扱うため、最小権限・最小保存・監査可能性を重視する [oai_citation:0‡アプリ開発企画書：Interview Hack.pdf](sediment://file_0000000080a871fa94ccc207e9fe6252)
- 決済はStripeを用い、カード情報は自サーバー/DBに一切触れない（PCI負担を最小化）
- クレジット付与・消費は改ざんされやすい領域なので「サーバー側強制」「冪等性」「監査ログ」を必須要件とする

---

## 9.2 データ分類（取り扱いレベル）

### 9.2.1 最高機密（漏洩時インパクト最大）

- 面接ログ（質問/回答/評価/改善案）
- 自己資産（経歴、成果、強み弱み、エピソード）
- 志望動機メモ、選考状況
- ES下書き/完成文（志望動機・自己PR等）
- 企業研究資料の要約・メモ（ユーザー解釈を含む）

### 9.2.2 機密

- ユーザー識別子（user_id）
- 企業DB（ただし一般公開情報 + ユーザーメモが混在するため注意）
- 企業研究資料の生テキスト（公開情報由来でも、ユーザー選別データとして扱う）

### 9.2.3 決済関連（カード情報は保持しない）

- 自アプリが保持するのは以下のみ
  - Stripeのcheckout_session/payment_intent等の参照ID（external_ref）
  - 購入したクレジットパッケージID
  - クレジット取引履歴（credit_transactions）
- カード番号・CVC等はStripeが保持し、アプリ側は取得しない

---

## 9.3 認証・認可（Supabase + Google OAuth）

- 認証：Supabase Auth（Google OAuth 主導線 + Email/Password フォールバック）
- すべてのユーザーデータは user_id（auth.users.id）に紐付け、RLSで `auth.uid() = user_id` を強制
- BFFは Supabase JWT を検証し、リクエストボディの user_id を信用しない（JWTのsubのみ採用）
- セッション/トークンの保存先は supabase-js に委譲する（独自保存はしない）。より強いXSS耐性が必要な場合は BFF で HttpOnly Cookie に載せ替える方式を将来検討

---

## 9.4 データ保護（通信・保存）

### 9.4.1 通信

- すべて HTTPS/TLS 必須
- BFFはHSTS推奨
- Webhook（Stripe）はTLS + 署名検証で担保

### 9.4.2 保存（DB）

- Supabase(Postgres)に保存するデータはRLSで論理的分離
- クレジット系・生成ジョブ系はBFFのservice roleのみ書き込み可（クライアント直書き禁止）
- 重要ログは改ざん耐性のため「追記型（append-only）」を基本とする（credit_transactions など）

### 9.4.3 秘密情報（Secrets）

- OpenAI API Key / Stripe Secret / Webhook Secret / Supabase Service Role Key はサーバー環境変数にのみ配置
- リポジトリにコミットしない（.env漏洩対策）
- ローテーション可能な運用（漏洩時の再発行手順）を持つ

---

## 9.5 決済・クレジット周りのセキュリティ（必須追記）

### 9.5.1 カード情報非保持（PCI最小化）

- 決済画面はStripe Checkoutを使用し、カード情報はStripeが直接取り扱う
- アプリ・BFF・DBはカード情報を一切保存しない

### 9.5.2 Stripe Webhookの真正性

- Webhook署名検証（Stripe-Signature）を必須化
- 署名検証が失敗したリクエストは即時 400 で破棄
- Webhookエンドポイントは公開される前提なので、IP制限に頼らず「署名検証」を根本対策とする

### 9.5.3 冪等性（idempotency）と二重付与防止

- Stripe の event.id（stripe_event_id）を保存し UNIQUE 制約を付与する
- 併せて external_ref（checkout_session_id / payment_intent_id）とも突合する
- 既に処理済みの stripe_event_id を受信した場合は、何もせず 200 を返す（再送に耐える）
- DB更新は必ずトランザクションで行う
  - credit_transactions(purchase) 追加
  - credit_balances 加算
    を同時に確定

### 9.5.4 クレジット改ざん防止

- クライアントから credit_balances / credit_transactions の INSERT/UPDATE を禁止（RLSで拒否）
- クレジット付与はWebhook処理のみ
- クレジット消費はBFFが残高行をロックして行う（同時実行耐性）
  - `SELECT ... FOR UPDATE` で balance を確保
  - 残高不足は INSUFFICIENT_CREDITS を返す

### 9.5.5 失敗時の整合性（reversed）

- クレジット消費は Reserve / Commit / Reversed の状態遷移で管理する
- AI実行前に type=consume, status=pending で予約
- 成功時は status=committed に更新
- 失敗時は status=reversed に更新し、残高をロールバックする
- これは「内部ロールバック」であり、Stripe への返金（refund API）とは別物である
- Stripe の返金（refund）は、実際に決済を取り消す場合のみ使用する

### 9.5.6 不正購入・乱用対策

- Checkout Session作成APIにレート制限（user_id/IP）
- 同一ユーザーの短時間大量購入・大量消費はアラート対象（generation_jobs / credit_transactionsから検知）
- クーポン等を導入する場合は不正利用に備え、発行/使用履歴を必ず保持

### 9.5.7 リプレイ攻撃・改ざんリクエスト対策

- Checkout Session作成時に metadata に user_id と package_id を入れる
- Webhook処理時に
  - metadata.user_id を信頼しすぎず、DB上の購入パッケージ（stripe_price_id一致）で最終決定する
  - packageが無効/存在しない場合は付与しない

---

## 9.6 LLM利用におけるプライバシー・安全

- 外部AI APIへ送るデータは最小化（必要な自己資産のみ・要約優先）
- 送信前にマスキング可能な項目（電話番号、住所、メール等）を自動マスクする方針を検討
- 出力が「経歴捏造」「詐称助長」にならないよう、プロンプトに禁止事項を明記
- 生成失敗/不正出力は generation_jobs に記録し、再現と改善ができる状態にする

---

## 9.7 ログ・監査（Audit）

- クレジット：
  - credit_transactions は必須（purchase/consume + status: pending/committed/reversed, bonus, adjustment）
  - external_ref を保持し、Stripe側と突合可能にする
- AI：
  - generation_jobs に job_type, model, token/cost, status, error を保存
- アプリログ：
  - 個人情報をログに出さない（PIIマスキング）
  - エラー調査に必要な相関ID（request_id）を採用

---

## 9.8 退会・データ削除（プライバシー対応）

- MVPでは論理削除（アカウント無効化 + データ非表示）を基本
- 将来的に「完全削除」を実装する場合
  - 削除対象テーブル（assets/companies/interviews/logs/qa/credit関連）を定義
  - 削除実行者（本人/管理者）と手順を明確化
- クレジット取引は会計/不正調査の観点から保持期間を設ける（保持方針は法務・運用で確定）

---

## 9.9 規約・法務（最低限）

- 利用規約
  - AI出力の正確性保証なし
  - 採用結果の保証なし
  - 不正行為（経歴詐称助長目的など）の禁止
- プライバシーポリシー
  - 収集するデータ、第三者提供（OpenAI/Stripe/Supabase）を明記
  - 保存期間、削除手順
- 免責
  - AIの出力は参考情報であり、最終判断はユーザー

---

## 9.10 この章の凍結事項

- 決済は Stripe Checkout + Webhook 署名検証（event.id 保存による冪等性）
- external_ref のUNIQUE + 冪等処理で二重付与を防ぐ
- クレジット残高/履歴はサーバー側のみ書き込み可
- RLSで user_id 越境を防止
- クレジットは Reserve/Commit/Reversed の状態遷移で管理し、生成ジョブと相関できる監査ログを必ず残す
