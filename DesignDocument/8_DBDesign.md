# 8. データベース設計（Supabase / PostgreSQL）

## 概要

- **スキーマ定義**: テーブル・カラム・リレーション・インデックスは **DBML** で一元管理しています。
- **DBMLファイル**: [`8_DBDesign.dbml`](./8_DBDesign.dbml)（同ディレクトリ）
- **ER図の見方**: [dbdiagram.io](https://dbdiagram.io) を開き、「Import」→「Import from DBML」で `8_DBDesign.dbml` の内容を貼り付けると、ER図が自動生成されます。

このドキュメントでは、方針・RLS・運用ルール・制約の要点を説明し、詳細なスキーマは DBML を正とします。

---

## 8.1 方針

| 項目                   | 内容                                              |
| ---------------------- | ------------------------------------------------- |
| DB                     | Supabase (PostgreSQL)                             |
| マルチテナント         | 全ユーザーデータは `user_id` で分離               |
| RLS                    | 有効化。原則 `auth.uid() = user_id`               |
| クレジット・生成・課金 | クライアント直書き禁止。**BFF経由のみ**           |
| 論理削除               | 必要なテーブルは `deleted_at` を保有              |
| インデックス           | 検索・一覧で使うカラムに適切に設定（DBML に記載） |

**クライアント直書き禁止テーブル（BFF のみ）**:  
`generation_jobs`, `credit_balances`, `credit_transactions`, `stripe_events`

---

## 8.2 テーブル一覧（Quick Reference）

スキーマの詳細（カラム・型・FK・インデックス）は **`8_DBDesign.dbml`** を参照してください。

| セクション   | テーブル                                 | 説明                                           |
| ------------ | ---------------------------------------- | ---------------------------------------------- |
| 認証         | `auth.users`                             | Supabase Auth 管理（アプリからは参照のみ）     |
| 認証         | `profiles`                               | アプリ用プロフィール（display_name 等）        |
| コアドメイン | `assets`                                 | 自己資産（学歴・職歴・強み等）                 |
| コアドメイン | `asset_tags`                             | タグ名マスタ                                   |
| コアドメイン | `asset_tag_relations`                    | 自己資産 ⇔ タグ（N:M）                         |
| コアドメイン | `companies`                              | 企業・選考情報                                 |
| コアドメイン | `research_materials`                     | 企業研究資料（URL・PDF・メモ）                 |
| コアドメイン | `es_projects`                            | ES作成プロジェクト（ステータス・提出期限）     |
| コアドメイン | `es_sections`                            | ESセクション（志望動機・自己PRごとの内容）     |
| コアドメイン | `es_project_asset_relations`             | ESプロジェクト ⇔ 自己資産（N:M）               |
| コアドメイン | `es_project_research_material_relations` | ESプロジェクト ⇔ 企業研究資料（N:M）           |
| コアドメイン | `interviews`                             | 面接（日付・ phase ・結果）                    |
| コアドメイン | `interview_logs`                         | 面接ログ（質問・回答・AIフィードバック）       |
| コアドメイン | `qa`                                     | Q&Aバンク                                      |
| コアドメイン | `qa_asset_relations`                     | Q&A ⇔ 自己資産（N:M）                          |
| AI/生成      | `generation_jobs`                        | AI ジョブ履歴（BFF のみ書き込み）              |
| 課金         | `credit_balances`                        | クレジット残高（BFF のみ書き込み）             |
| 課金         | `credit_transactions`                    | クレジット履歴（BFF のみ書き込み）             |
| 課金         | `stripe_events`                          | Stripe Webhook 受信済み ID（BFF のみ書き込み） |
| 課金         | `credit_packages`                        | 商品マスタ（管理者変更）                       |
| 課金         | `credit_costs`                           | 機能別消費クレジット（管理者変更）             |

---

## 8.3 認証・プロフィール

- **auth.users**: Supabase Auth が管理（Google OAuth 主導 + Email/Password フォールバック）。アプリからは直接変更しない。
- **profiles**: 定義は DBML 参照。  
  **RLS**: SELECT / UPDATE / INSERT は `auth.uid() = user_id`。

---

## 8.4 コアドメイン（自己資産・企業・ES・面接・Q&A）

- **assets**: 論理削除 `deleted_at`。型は DBML の `type` コメント参照。
- **asset_tags**: 同一ユーザー内でタグ名一意 `(user_id, name)`。
- **asset_tag_relations**: 複合 PK `(asset_id, tag_id)`。RLS は asset / asset_tags の `user_id` に準拠。
- **companies**: 論理削除 `deleted_at`。
- **research_materials**: 企業研究資料（URL/PDF/メモ）の原文と要約を保持。
- **es_projects**: 企業ごとのES作成単位。`status` と `deadline` を管理。
- **es_sections**: ESの各設問（志望動機/自己PR等）の下書き・完成文を管理。
- **es_project_asset_relations**: ESプロジェクトと自己資産の N:M 関係。
- **es_project_research_material_relations**: ESプロジェクトと企業研究資料の N:M 関係。
- **interviews**: `company_id` FK。結果は `result`（pass / fail / pending）。
- **interview_logs**: `interview_id`・`user_id` で紐付け。AI フィードバック・スコアを保持。
- **qa**: `company_id` は null 可。`source_type`・`review_status` の値は DBML の Note 参照。
- **qa_asset_relations**: 複合 PK `(qa_id, asset_id)`。操作は `qa.user_id = auth.uid()` を満たす行のみ。

**RLS**: 上記いずれも、ユーザー所有データは **全操作で `auth.uid() = user_id`**（または qa 経由の制御）。

---

## 8.5 AI / 生成管理

- **generation_jobs**: 定義は DBML 参照。`external_ref` でクレジット取引と紐付け。
- **RLS**:
  - SELECT: `auth.uid() = user_id`
  - INSERT / UPDATE: **BFF の service role のみ**
  - DELETE: 管理者のみ（原則削除しない）

---

## 8.6 クレジット・課金系

- **credit_balances**: 制約 `CHECK (balance >= 0)`。  
  **RLS**: SELECT は `auth.uid() = user_id`。UPDATE / INSERT は BFF の service role のみ。
- **credit_transactions**: `UNIQUE (user_id, external_ref, type)`。状態は `pending` / `committed` / `reversed`。  
  **RLS**: SELECT は `auth.uid() = user_id`。INSERT / UPDATE は BFF の service role のみ。
- **stripe_events**: PK は Stripe の `event.id`。重複処理防止用。  
  **RLS**: SELECT は管理者のみ。INSERT は BFF の service role のみ。
- **credit_packages**: 商品マスタ。SELECT は全ユーザー（`is_active = true` のみ）。変更は管理者 / service role のみ。
- **credit_costs**: 機能別消費量。SELECT は全ユーザー。変更は管理者 / service role のみ。

---

## 8.7 インデックス・パフォーマンス指針

- 一覧で頻繁に使うもの:  
  `assets(user_id, created_at)`, `qa(user_id, created_at)`, `interviews(user_id, interview_date)`, `interview_logs(interview_id, created_at)`
- 検索: `companies(user_id, name)`
- クレジット履歴: `credit_transactions(user_id, created_at)`, `credit_transactions(user_id, external_ref, type)`

※ 実際のインデックス名・複合の有無は **`8_DBDesign.dbml`** の `indexes { }` を参照。

---

## 8.8 参照整合性・削除ポリシー

| 観点                             | 内容                                                                                                            |
| -------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| ユーザー削除時                   | 原則は論理削除（profiles にフラグ or auth 無効化）。物理削除する場合は関連データを CASCADE またはバッチで削除。 |
| assets / companies               | `deleted_at` による論理削除を優先。                                                                             |
| interviews / interview_logs / qa | 履歴性が高いため、物理削除は原則として管理者操作のみ。                                                          |

---

## 8.9 マイグレーション運用

- Supabase Migration でスキーマをバージョン管理する。
- 変更ルール:
  - 既存カラム削除は原則しない（nullable 化または非推奨化）。
  - enum 相当は `text` + マスタテーブル または CHECK 制約で表現。

**スキーマ変更時**: まず **`8_DBDesign.dbml`** を更新し、その内容に合わせてマイグレーション SQL を作成することを推奨します。

---

## 8.10 この章の凍結事項

- DB は **Supabase (PostgreSQL) + RLS** 前提。
- **user_id** による完全なデータ分離。
- クレジット系テーブルは **BFF の service role のみ** 書き込み可。
- AI/課金の監査のため、`generation_jobs` / `credit_transactions` / `stripe_events` を必ず保存し、クレジットは **pending → committed / reversed** の状態遷移で管理する。

dbmlファイルはctrl+shift+dでプレビューできるよ。
