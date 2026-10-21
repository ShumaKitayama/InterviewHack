# 10. 分析・計測（プロダクト改善）

> 改訂：2026-10-22。方針転換メモ（`DirectionChange_20261022.md`）の決定事項 D1〜D16 を反映した（主に D13・D14・D15）。

## 10.1 方針

- 本プロダクトは「改善ループが回っているか」と「AI連携のテンプレートが機能しているか」を可視化する
- 計測は段階的に行う（D15）
  - フェーズ1（開発者本人のみ）：分析用の SDK は入れない。DB から SQL で集計する。ダッシュボードは作らない
  - フェーズ2（TestFlight で 10 人前後）：ファネルとリテンションを追加する。その時点で SDK の導入を検討する
- 収益化をしない（D14）ため、ビジネス KPI は持たない。LLM の API を呼ばない（D7）ため、AI コスト KPI も持たない
- 個人情報・面接内容の本文は分析に使わず、メタデータ（件数・状態・時刻）だけを集計する

---

## 10.2 計測対象の大分類

### 10.2.1 プロダクトKPI

- フェーズ1
  - AI連携の品質（テンプレートの改善に直結する指標）
    - template_version ごとの回答の解析成功率（`prompt_sessions.parse_result`）
    - template_version ごとの離脱率（`prompt_sessions.status = 'abandoned'` の割合）
  - 改善ループの稼働
    - 週ごとの Q&A 保存数
    - 週ごとの復習の回数（`qa.review_status` の更新件数）
    - 週ごとの面接ログの件数
- フェーズ2で追加
  - 行動ファネル：新規登録 → 自己資産の登録 → 初回プロンプト作成 → 初回 Q&A 保存
  - 継続指標：D7 リテンション

### 10.2.2 ビジネスKPI

- 持たない（D14：MVP では収益化しない）
- 収益化を検討する段階（0.3 のフェーズ3）で、本節を再定義する

### 10.2.3 運用KPI

- AI コスト KPI は持たない（D7：運営側で LLM の API を呼ばない）
- 運用上の確認は、Cloud Run の構造化ログ（7.8.1）でエラー率とレイテンシを見る（ダッシュボードは作らない）

---

## 10.3 イベント設計（Event Tracking）

- フェーズ1ではイベント計測を行わない。DB 由来の集計（10.4.1）だけで判断する
- フェーズ2で導入するときに、イベント名と送信項目を確定する。候補は以下のとおり

### 10.3.1 認証・オンボーディング（候補）

- user_signup
- user_login
- asset_created (count)
- company_created

### 10.3.2 コア機能（候補）

- prompt_session_created (use_case, template_version)
- prompt_shared（共有シート / クリップボードで外部AIに渡した）
- prompt_response_parsed (use_case, template_version, parse_result)
- prompt_session_saved (use_case, template_version)
- prompt_session_resumed（未完了の AI連携を再開した）
- research_material_imported (type: url / pdf / image)
- share_link_created (expires_in_days)
- review_status_changed
- offline_review_opened

### 10.3.3 課金・クレジット

- なし（D14）

### 10.3.4 システム・エラー（候補）

- api_error (endpoint, code)

※ 送信するのは user_id（または匿名化ID）とイベント名・数値メタデータのみ。  
※ 質問文・回答文・自己資産本文・プロンプト・外部AIの回答などの内容は送らない。

---

## 10.4 データソース設計

### 10.4.1 DB由来（一次ソース）

- `prompt_sessions`
- `qa` / `interviews` / `interview_logs`
- `assets` / `companies` / `research_materials` / `es_projects` / `es_sections` / `share_links`

集計は以下の SQL で行う（Neon に対して手動で実行する）。すべてメタデータのみを参照し、本文カラムは読まない。

#### template_version ごとの解析成功率

解析を行うユースケース（`qa_generate` / `answer_review` / `research_summarize`）が対象。

```sql
SELECT
  use_case,
  template_version,
  COUNT(*) FILTER (WHERE parse_result IS NOT NULL)                AS parsed,
  COUNT(*) FILTER (WHERE parse_result = 'success')                AS success,
  COUNT(*) FILTER (WHERE parse_result = 'partial')                AS partial,
  COUNT(*) FILTER (WHERE parse_result = 'failed')                 AS failed,
  ROUND(
    COUNT(*) FILTER (WHERE parse_result = 'success')::numeric
    / NULLIF(COUNT(*) FILTER (WHERE parse_result IS NOT NULL), 0), 3
  )                                                               AS success_rate
FROM prompt_sessions
WHERE use_case IN ('qa_generate', 'answer_review', 'research_summarize')
  AND created_at >= now() - interval '30 days'
GROUP BY use_case, template_version
ORDER BY use_case, template_version;
```

#### template_version ごとの離脱率

作成から 7 日を過ぎた `created` / `pasted` は、一覧の取得時に `abandoned` へ更新される（7.9）。一覧を開いていない期間のセッションは `created` / `pasted` のまま残るため、集計の前にアプリで一覧を開くか、同じ条件で更新しておく。

```sql
SELECT
  use_case,
  template_version,
  COUNT(*)                                              AS sessions,
  COUNT(*) FILTER (WHERE status = 'abandoned')          AS abandoned,
  COUNT(*) FILTER (WHERE status = 'saved')              AS saved,
  ROUND(
    COUNT(*) FILTER (WHERE status = 'abandoned')::numeric
    / NULLIF(COUNT(*), 0), 3
  )                                                     AS abandon_rate
FROM prompt_sessions
WHERE created_at >= now() - interval '30 days'
  AND status IN ('saved', 'abandoned')   -- 進行中（created / pasted）のものは除外する
GROUP BY use_case, template_version
ORDER BY use_case, template_version;
```

#### 週ごとの Q&A 保存数

```sql
SELECT
  date_trunc('week', created_at) AS week,
  source_type,
  COUNT(*)                       AS qa_saved
FROM qa
WHERE created_at >= now() - interval '12 weeks'
GROUP BY week, source_type
ORDER BY week, source_type;
```

#### 週ごとの復習の回数

`qa.review_status` が `new` 以外に更新されたものを、`updated_at` で週ごとに数える（更新履歴テーブルは持たないため、最終更新の週に計上される近似値である）。

```sql
SELECT
  date_trunc('week', updated_at) AS week,
  review_status,
  COUNT(*)                       AS qa_count
FROM qa
WHERE review_status IN ('reviewed', 'needs_review')
  AND updated_at >= now() - interval '12 weeks'
GROUP BY week, review_status
ORDER BY week, review_status;
```

#### 週ごとの面接ログの件数

```sql
SELECT
  date_trunc('week', created_at) AS week,
  COUNT(*)                       AS interview_logs_created
FROM interview_logs
WHERE created_at >= now() - interval '12 weeks'
GROUP BY week
ORDER BY week;
```

### 10.4.2 イベントログ（分析基盤）

- フェーズ1では持たない
- フェーズ2で、Firebase Analytics または Sentry の導入時に検討する
- 目的（フェーズ2）
  - UX改善（どこで離脱しているか。特に外部AIとの往復の前後）
  - 機能優先度の意思決定

---

## 10.5 ダッシュボード設計（運営者向け）

- 作らない（D15）。10.4.1 の SQL を必要なときに実行する
- フェーズ2以降で利用者が増え、手動の集計が負担になった時点で再検討する

---

## 10.6 A/Bテスト候補（テンプレートのバージョン比較）

利用者が少ないため、同時に振り分ける A/B テストは行わない。テンプレートのバージョンを切り替え、前後の期間で比較する（`prompt_sessions.template_version` で区別できる）。

- 想定Q&A：`qa-v1` と、目印の書き方や指示文を変えた `qa-v2` の比較
- 添削：観点の列挙方法を変えた `review-v1` と `review-v2` の比較
- 企業研究要約：出力項目の順序や粒度を変えたバージョンの比較

評価指標：

- 解析成功率（`parse_result = 'success'` の割合）
- 部分成功率（`partial`）。手動修正が必要になった割合
- 離脱率（`abandoned` の割合）
- 定性評価：取り込んだ内容が、そのまま使える品質か（開発者自身の評価）

---

## 10.7 プライバシー配慮（分析データ）

- 分析に使わないもの
  - 面接質問・回答の本文
  - 自己資産の本文
  - 志望動機メモの本文
  - プロンプトの全文、外部AIの回答（そもそもサーバーに保存しない。D13）
  - 共有リンクのスナップショット本文
- 分析に使うもの
  - user_id（またはハッシュ化ID）
  - 種別・状態（use_case、template_version、status、parse_result、review_status など）
  - 時刻
  - 件数
- ログにはPIIを残さないポリシーを徹底する

---

## 10.8 この章の凍結事項

- フェーズ1は DB 集計のみで行い、分析用の SDK とダッシュボードは導入しない
- AI連携の品質は、`prompt_sessions` の template_version ごとの解析成功率と離脱率で評価する
- テンプレートの改善は、バージョンを上げて前後比較する
- 内容データ（質問・回答本文、プロンプト、外部AIの回答）は分析に使わない
- ビジネス KPI・AI コスト KPI は持たない（収益化を検討する段階で再定義する）
