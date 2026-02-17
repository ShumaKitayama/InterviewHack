# 10. 分析・計測（プロダクト改善 / ビジネス指標）

## 10.1 方針

- 本プロダクトは「継続利用されて改善ループが回っているか」「LLMコストに対して収益が健全か」を常に可視化する
- 計測は以下の3レイヤーで行う
  1. プロダクト利用状況（UX/KPI）
  2. 収益・クレジット消費（ビジネス）
  3. AIコスト・品質（運用）
- 個人情報・面接内容の本文は分析基盤に送らず、メタデータのみを送信する

---

## 10.2 計測対象の大分類

### 10.2.1 プロダクトKPI

- アクティブ指標
  - DAU / MAU
  - DAU / MAU 比（定着度）
- 行動ファネル
  - 新規登録 → 自己資産登録 → 企業登録 → 初回AI生成 → Q&A保存 → 復習実行
- 継続指標
  - D1 / D7 / D30 リテンション
  - 週次・月次の「面接ログ登録ユーザー率」
- 価値体験指標
  - 「1週間以内にAI生成を1回以上実行したユーザー割合」
  - 「AI生成→Q&A保存」まで到達した割合
  - 「復習UIを1回以上使った割合」

### 10.2.2 ビジネスKPI（クレジット方式）

- 収益
  - 売上（月次・週次）
  - ARPU / ARPPU
- 課金行動
  - クレジット購入ユーザー率
  - 初回購入までの平均日数
  - 追加購入率（リピート率）
- クレジット利用
  - ユーザーあたり月間消費クレジット
  - 機能別消費割合（qa_generate / answer_review / summary_generate / research_summarize / es_generate / es_refine）
  - 残高ゼロ到達ユーザー数（購入導線のトリガー）

### 10.2.3 AIコスト・運用KPI

- OpenAIコスト
  - 月間総トークン / 推定コスト
  - ユーザーあたり平均コスト
- 品質・安定性
  - 生成成功率 / 失敗率
  - リトライ発生率
  - 出力スキーマ不正率
- 機能別
  - 事前生成 / 添削 / 要約 / 企業研究要約 / ES生成 / ES推敲 の呼び出し回数
  - 各機能の平均レイテンシ

---

## 10.3 イベント設計（Event Tracking）

### 10.3.1 認証・オンボーディング

- user_signup
- user_login
- onboarding_completed
- asset_created (count)
- company_created

### 10.3.2 コア機能

- qa_generate_requested
- qa_generate_succeeded
- qa_generate_failed
- answer_review_requested
- answer_review_succeeded
- answer_review_failed
- summary_generate_requested
- summary_generate_succeeded
- research_summarize_requested
- research_summarize_succeeded
- research_summarize_failed
- es_generate_requested
- es_generate_succeeded
- es_generate_failed
- es_refine_requested
- es_refine_succeeded
- es_refine_failed
- qa_saved
- interview_log_created
- es_project_created
- es_section_saved
- review_viewed
- review_status_changed

### 10.3.3 課金・クレジット

- credit_package_viewed
- checkout_started
- checkout_completed
- credit_purchased (package_id, credits)
- credit_consumed (feature, amount)
- credit_reversed (feature, amount)
- insufficient_credits_error

### 10.3.4 システム・エラー

- openai_error (type)
- stripe_webhook_error
- api_error (endpoint, code)

※ 送信するのは user_id（または匿名化ID）とイベント名・数値メタデータのみ。  
※ 質問文・回答文・自己資産本文などの内容は送らない。

---

## 10.4 データソース設計

### 10.4.1 DB由来（一次ソース）

- credit_transactions
- credit_balances
- generation_jobs
- interviews / interview_logs / qa
- assets / companies / research_materials / es_projects / es_sections

これらから以下を集計可能：

- 機能別利用回数
- クレジット消費構造
- AI成功率・失敗率
- ユーザー行動ファネル

### 10.4.2 イベントログ（分析基盤）

- フロントエンド：UI操作・遷移・CTAクリック
- BFF：AI実行・課金・エラー
- 目的：
  - UX改善（どこで離脱しているか）
  - 課金導線の最適化
  - 機能優先度の意思決定

---

## 10.5 ダッシュボード設計（運営者向け）

### 10.5.1 概要ダッシュボード

- MAU / DAU
- 売上（月次）
- クレジット購入ユーザー数
- 月間AIコスト / 利益概算（売上 - AIコスト）

### 10.5.2 利用状況ダッシュボード

- 機能別利用回数（事前生成 / 添削 / 要約 / 企業研究要約 / ES生成 / ES推敲）
- 1ユーザーあたり平均利用回数
- Q&A保存率 / 復習実行率
- ES完了率（作成開始→completed / submitted）

### 10.5.3 課金・クレジットダッシュボード

- パッケージ別販売数
- 初回購入率 / リピート購入率
- クレジット消費スピード分布

### 10.5.4 AI運用ダッシュボード

- 成功率 / 失敗率
- 平均レイテンシ
- モデル別コスト比率
- 異常検知（急激な失敗率増加・コスト急増）

---

## 10.6 A/Bテスト候補

- ホーム画面のCTA文言
  - 「面接対策を始める」 vs 「想定Q&Aを作る」
- オンボーディング
  - 自己資産3件入力必須 vs スキップ可
- 課金導線
  - 残高ゼロ時のみ表示 vs 残り少ない時も表示
- 生成結果UI
  - 一覧表示 vs 1問ずつカード表示

評価指標：

- 初回AI生成到達率
- Q&A保存率
- 購入率
- 継続率（D7/D30）

---

## 10.7 プライバシー配慮（分析データ）

- 送信しないもの
  - 面接質問・回答の本文
  - 自己資産の本文
  - 志望動機メモの本文
- 送信するもの
  - user_id（またはハッシュ化ID）
  - イベント種別
  - 時刻
  - 数値データ（回数、クレジット量など）
- ログにはPIIを残さないポリシーを徹底する

---

## 10.8 この章の凍結事項

- 分析は「プロダクトKPI / ビジネスKPI / AI運用KPI」の3系統で行う
- 内容データ（質問・回答本文）は分析基盤に送らない
- クレジット消費・生成ジョブは必ず集計対象にする
- ダッシュボードで「売上・AIコスト・利用状況」を常時可視化する
