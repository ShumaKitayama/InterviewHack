# 8. データベース設計（Neon / PostgreSQL）

> 改訂：2026-10-22。方針転換メモ（`DirectionChange_20261022.md`）の決定事項 D1〜D16 と 2.10 共通契約を反映した。

## 概要

- **スキーマ定義**: テーブル・カラム・リレーション・インデックスは **DBML** で一元管理しています。
- **DBMLファイル**: [`8_DBDesign.dbml`](./8_DBDesign.dbml)（同ディレクトリ）
- **ER図の見方**: [dbdiagram.io](https://dbdiagram.io) を開き、「Import」→「Import from DBML」で `8_DBDesign.dbml` の内容を貼り付けると、ER図が自動生成されます。

このドキュメントでは、方針・認可・運用ルール・制約の要点を説明し、詳細なスキーマは DBML を正とします。

---

## 8.1 方針

| 項目           | 内容                                                                                         |
| -------------- | -------------------------------------------------------------------------------------------- |
| DB             | Neon（PostgreSQL）。無料枠で運用する（D4）                                                   |
| アクセス経路   | **Go BFF のみ**。iOS アプリから DB へ直接接続しない                                          |
| マルチテナント | 全ユーザーデータは `user_id`（`users.id`）で分離                                             |
| 認可           | RLS は使わない。BFF の Repository で、全クエリに `user_id = $1` を付けて行う（D6）           |
| ユーザー       | Firebase Auth の `firebase_uid` を `users` に保持し、BFF が取得または作成する（D5）          |
| 論理削除       | 必要なテーブルは `deleted_at` を保有                                                         |
| 保存しないもの | プロンプトの全文、貼り戻した回答の生データ、PDF・画像のファイル本体、共有トークンの生の値     |
| インデックス   | 検索・一覧で使うカラムに適切に設定（DBML に記載）                                            |

---

## 8.2 テーブル一覧（Quick Reference）

スキーマの詳細（カラム・型・FK・インデックス）は **`8_DBDesign.dbml`** を参照してください。

| セクション   | テーブル                                 | 説明                                                      |
| ------------ | ---------------------------------------- | --------------------------------------------------------- |
| ユーザー     | `users`                                  | アプリのユーザー（`firebase_uid` で Firebase Auth と対応） |
| コアドメイン | `assets`                                 | 自己資産（学歴・職歴・強み等）                            |
| コアドメイン | `asset_tags`                             | タグ名マスタ                                              |
| コアドメイン | `asset_tag_relations`                    | 自己資産 ⇔ タグ（N:M）                                    |
| コアドメイン | `companies`                              | 企業・選考情報                                            |
| コアドメイン | `research_materials`                     | 企業研究資料（端末で抽出したテキスト・要約）              |
| コアドメイン | `es_projects`                            | ES作成プロジェクト（ステータス・提出期限）                |
| コアドメイン | `es_sections`                            | ESセクション（志望動機・自己PRごとの内容）                |
| コアドメイン | `es_section_versions`                    | ESセクションの版（最新20件まで）                          |
| コアドメイン | `es_project_asset_relations`             | ESプロジェクト ⇔ 自己資産（N:M）                          |
| コアドメイン | `es_project_research_material_relations` | ESプロジェクト ⇔ 企業研究資料（N:M）                      |
| コアドメイン | `interviews`                             | 面接（日付・ phase ・結果）                               |
| コアドメイン | `interview_logs`                         | 面接ログ（質問・回答・AI連携で取り込んだ添削結果）        |
| コアドメイン | `qa`                                     | Q&Aバンク                                                 |
| コアドメイン | `qa_asset_relations`                     | Q&A ⇔ 自己資産（N:M）                                     |
| AI連携       | `prompt_sessions`                        | AI連携 1 回分の進行状態（D13）                            |
| AI連携       | `dashboard_summaries`                    | ダッシュボード要約（`summary_generate` の取り込み先）     |
| 共有         | `share_links`                            | ES 共有リンク（スナップショット・期限・失効。D12）        |

廃止したテーブル：`auth_users`（`auth.users` 参照）, `profiles`（`users` に統合）, `generation_jobs`（`prompt_sessions` に置き換え）, `credit_balances`, `credit_transactions`, `stripe_events`, `credit_packages`, `credit_costs`

---

## 8.3 ユーザー

- **users**: 定義は DBML 参照。旧 `auth.users` と `profiles` を統合した。
  - `firebase_uid` は一意。BFF は Firebase ID トークンを検証した後、`firebase_uid` で検索し、存在しなければ作成する。
  - 全テーブルの `user_id` は `users.id` を参照する。
  - パスワード等の認証情報は保持しない（Firebase Auth が管理）。

---

## 8.4 コアドメイン（自己資産・企業・ES・面接・Q&A）

- **assets**: 論理削除 `deleted_at`。型は DBML の `type` コメント参照。
- **asset_tags**: 同一ユーザー内でタグ名一意 `(user_id, name)`。
- **asset_tag_relations**: 複合 PK `(asset_id, tag_id)`。BFF は、asset と tag が同じユーザーのものであることを確認してから書き込む。
- **companies**: 論理削除 `deleted_at`。
- **research_materials**: 端末で抽出したテキスト（`content_raw`）と、AI連携で取り込んだ要約（`summary`）を保持。ファイル本体は保存しない（D11）。
- **es_projects**: 企業ごとのES作成単位。`status` と `deadline` を管理。`status` は `not_started` / `draft` / `in_review` / `completed` / `submitted`（初期値 `not_started`、CHECK 制約で値を限定）。
- **es_sections**: ESの各設問（志望動機/自己PR等）の下書き・完成文を管理。AI連携の取り込み先は `content_draft`。
- **es_section_versions**: `content_draft` の過去の版。作るのは AI連携の commit で上書きする直前、手動保存、「この版に戻す」の直前だけ。セクションごとに最新20件まで保持し、超えた分は版を作るトランザクションで古い順に削除する。ESセクションの削除で一緒に削除する（ON DELETE CASCADE）。
- **es_project_asset_relations**: ESプロジェクトと自己資産の N:M 関係。
- **es_project_research_material_relations**: ESプロジェクトと企業研究資料の N:M 関係。
- **interviews**: `company_id` FK。結果は `result`（pass / fail / pending）。
- **interview_logs**: `interview_id`・`user_id` で紐付け。`ai_feedback_text` / `ai_improved_answer_text` は、カラム名を変えずに「AI連携で取り込んだ結果」とする。
- **qa**: `company_id` は null 可。`source_type`・`review_status` の値は DBML の Note 参照。`qa_generate` の `【意図N】` は保存しない（意図のカラムは持たない）。
- **qa_asset_relations**: 複合 PK `(qa_id, asset_id)`。

**認可**: 上記いずれも、BFF の Repository が **全操作で `user_id = $1`** を条件に含める。中間テーブル（`*_relations`）は親テーブルの `user_id` を JOIN または事前確認で検証する。

---

## 8.5 AI連携（prompt_sessions / dashboard_summaries）

- **prompt_sessions**: 定義は DBML 参照（D13）。
  - `use_case` と `template_version` は 2.10.1 の値を使う。
  - `input_refs`（jsonb）には、選んだ自己資産・企業研究資料の ID とオプションのみを入れる。**本文は入れない**。
  - プロンプトの全文と、貼り戻した回答の生データは保存しない。
  - `status` の遷移は `created` → `pasted`（parse）、`pasted` → `pasted`（再解析）、`pasted` → `saved`（commit）、`created` / `pasted` → `abandoned`。`created` → `saved` の直接の commit は許可しない。`saved` / `abandoned` は終端（7.6.1）。
  - `commit` は `UPDATE ... WHERE id = $1 AND user_id = $2 AND status = 'pasted'` の条件付き更新で行い、影響行数 0 なら `SESSION_NOT_FOUND` または `SESSION_STATE_CONFLICT` とする（二重保存の防止）。
  - `parse_result` は未解析のとき null。解析しないユースケースは parse 時に `success` とする。
  - 作成から 7 日を過ぎても `created` / `pasted` のままの行は `abandoned` に更新する。定期ジョブは持たず、一覧の取得時（`GET /prompt-sessions`、`GET /dashboard`）に条件付き UPDATE で行う（7.9）。
- **dashboard_summaries**: `summary_generate` の取り込み先。ダッシュボードには `(user_id, created_at)` の最新 1 件を表示する。

---

## 8.6 共有リンク（share_links）

- **share_links**: 定義は DBML 参照（D12）。
  - トークンは 32 バイトの暗号学的乱数。`token_hash` にはその SHA-256 ハッシュを保存し、一意とする。生トークンは保存せず、作成時の応答でのみ返す。
  - `snapshot`（jsonb）には、公開するセクションの title と本文（`content_final`、なければ `content_draft`）、および `show_company_name` が真の場合の企業名を入れる。ユーザー名は入れない。`show_company_name` の既定値は false。
  - 閲覧可否は `revoked_at IS NULL AND expires_at > now()` で判定する。
  - `expires_at` は作成時刻 + 1日 / 7日 / 30日。null は許可しない（無期限不可）。
  - ESプロジェクトを削除すると、紐づく行も削除する。
  - 期限切れ・失効した行の物理削除は、運用で定期的に行ってよい（12 章）。

---

## 8.7 インデックス・パフォーマンス指針

- 一覧で頻繁に使うもの:
  `assets(user_id, created_at)`, `qa(user_id, created_at)`, `interviews(user_id, interview_date)`, `interview_logs(interview_id, created_at)`
- 検索: `companies(user_id, name)`
- AI連携: `prompt_sessions(user_id, status, created_at)`（未完了一覧）, `prompt_sessions(use_case, template_version)`（D15 の集計）
- ダッシュボード: `dashboard_summaries(user_id, created_at)`
- 共有リンク: `share_links(token_hash)`（unique）, `share_links(es_project_id)`, `share_links(expires_at)`
- ユーザー: `users(firebase_uid)`（unique）

※ 実際のインデックス名・複合の有無は **`8_DBDesign.dbml`** の `indexes { }` を参照。

---

## 8.8 参照整合性・削除ポリシー

| 観点                             | 内容                                                                                                                       |
| -------------------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| ユーザー削除時（退会）           | 1 トランザクションで、関連データ（`share_links` を含む）を即時に物理削除する。`users` 行は物理削除せず、`deleted_at` を設定し、`email` と `display_name` を消して残す。Firebase Auth 側のユーザーは iOS アプリが削除する（6.3.3）。 |
| assets / companies               | `deleted_at` による論理削除を優先。                                                                                        |
| interviews / interview_logs / qa | ユーザーの DELETE（7.5.3）で物理削除する（管理者ロールは設けない。6.4）。                                                  |
| prompt_sessions                  | 対象（company / es_section 等）が削除されても、集計用に行は残す（参照は null 可）。                                        |
| share_links                      | ESプロジェクトの削除時は、関連する共有リンクも削除する（閲覧できなくなる）。                                               |
| es_section_versions              | ESセクションの削除時に一緒に削除する。セクションごとに最新20件を超えた分は、版の作成時に削除する。                         |

---

## 8.9 マイグレーション運用

- `backend/migrations` に、**冪等な SQL** を番号順に置き、`psql` で順番に適用する（`CREATE TABLE IF NOT EXISTS` など）。
- 新しい構成用に、マイグレーションを 001 から作り直す（Supabase 用の旧マイグレーションは廃止）。
- 環境ごとに Neon のブランチを分ける（`dev` ブランチ / `main` ブランチ）。
- 変更ルール:
  - 既存カラム削除は原則しない（nullable 化または非推奨化）。
  - enum 相当は `text` + CHECK 制約で表現。
  - マイグレーションは後方互換を保ち、バックエンドより先に適用する。

**スキーマ変更時**: まず **`8_DBDesign.dbml`** を更新し、その内容に合わせてマイグレーション SQL を作成することを推奨します。

---

## 8.10 この章の凍結事項

- DB は **Neon（PostgreSQL）**。アクセスは **Go BFF のみ**。
- **RLS は使わない**。認可は BFF の Repository で `user_id` による完全なデータ分離を行う。
- ユーザーは `users.firebase_uid` で Firebase Auth と対応付ける。
- プロンプトの全文・貼り戻した回答の生データ・ファイル本体・共有トークンの生の値は **保存しない**。
- AI連携の進行は `prompt_sessions` の **created → pasted → saved / abandoned** で管理し、保存は条件付き更新で 1 回だけ成功させる。
- 共有リンクは **期限付き（無期限不可）** で、ハッシュ化したトークンで管理する。

dbmlファイルはctrl+shift+dでプレビューできるよ。
