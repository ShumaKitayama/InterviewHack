# 2. 情報設計（ドメイン設計）

## 2.1 ドメイン全体像

本プロダクトは以下の主要ドメインで構成される。

- ユーザー
- 自己資産（ユーザーの経歴・エピソード・スキル等）
- 企業（応募先・職種・選考フェーズ）
- 企業研究資料（URL/PDF/メモの要約・構造化）
- ESプロジェクト（企業ごとの作成単位）
- ESセクション（志望動機・自己PR等の本文）
- 面接
- 面接ログ（質問・回答・AI添削結果）
- Q&A（想定質問・模範解答・復習用データ）
- AI生成ジョブ（生成リクエスト・コスト管理）
- クレジット（残高・消費・購入履歴）

これらは「準備（自己資産・企業研究）→ ES作成/面接実施 → 振り返り → 復習」というループを回すために相互に関連する。

---

## 2.2 エンティティ定義

### 2.2.1 User（ユーザー）

- id
- email
- password_hash / auth_provider
- display_name
- tier（enum: Free / Pro）
- created_at
- updated_at
- deleted_at（論理削除用）

注記: 課金・利用量の管理はクレジットで行い、tier は機能制限や上限ポリシーの切替にのみ使用する（サブスクリプション管理は行わない）

関係:

- User 1 : N 自己資産
- User 1 : N 企業
- User 1 : N 企業研究資料
- User 1 : N ESプロジェクト
- User 1 : N ESセクション
- User 1 : N 面接
- User 1 : N Q&A
- User 1 : N AI生成ジョブ

---

### 2.2.2 Asset（自己資産）

- id
- user_id
- type（enum: 学業, 職務経歴, プロジェクト, スキル, 強み, 弱み, エピソード, その他）
- title
- content（本文）
- tags（配列 or 中間テーブル）
- created_at
- updated_at
- deleted_at

関係:

- Asset N : 1 User
- Asset N : M Q&A（どの資産を参照して作られたかを記録）

制約:

- title は必須
- content は最大文字数制限を設ける

---

### 2.2.3 Company（企業）

- id
- user_id
- name
- job_title
- selection_phase（例: 書類, 一次, 二次, 最終）
- motivation_memo
- reference_url
- created_at
- updated_at
- deleted_at

関係:

- Company N : 1 User
- Company 1 : N ResearchMaterial（企業研究資料）
- Company 1 : N ESProject（この企業へのES）
- Company 1 : N 面接
- Company 1 : N Q&A（この企業向けに生成されたもの）

---

### 2.2.4 ResearchMaterial（企業研究資料 / セット2拡張）

- id
- company_id
- user_id
- title
- type（enum: url, pdf, text_note, image）
- source_url（任意）
- content_raw（スクレイピング結果やPDF抽出テキスト）
- summary（AI要約結果）
- created_at
- updated_at

関係:

- ResearchMaterial N : 1 Company
- ResearchMaterial N : M ESProject（ES作成時に参照）

---

### 2.2.5 ESProject（ES作成プロジェクト）

- id
- user_id
- company_id
- status（enum: draft, in_review, completed, submitted）
- deadline（提出期限）
- created_at
- updated_at

関係:

- ESProject N : 1 User
- ESProject N : 1 Company
- ESProject 1 : N ESSection
- ESProject N : M ResearchMaterial（参照資料）
- ESProject N : M Asset（使用した自己資産）

---

### 2.2.6 ESSection（ES項目 / 志望動機・自己PRなど）

- id
- es_project_id
- title（例: 志望動機, 自己PR, ガクチカ）
- content_draft（生成/編集中の文章）
- content_final（完成版）
- character_limit（文字数制限）
- version（履歴管理用、または別テーブル化）
- created_at
- updated_at

関係:

- ESSection N : 1 ESProject

---

### 2.2.7 Interview（面接）

- id
- user_id
- company_id
- interview_date
- phase
- result（任意: 合格/不合格/未確定）
- memo
- created_at
- updated_at

関係:

- Interview N : 1 User
- Interview N : 1 Company
- Interview 1 : N 面接ログ

---

### 2.2.8 InterviewLog（面接ログ）

- id
- interview_id
- question_text
- answer_text
- self_evaluation（任意: 自己評価メモ）
- ai_feedback_text
- ai_improved_answer_text
- score（任意: 1-5など）
- created_at
- updated_at

関係:

- InterviewLog N : 1 Interview
- InterviewLog 1 : N Q&A（改善後回答がQ&Aに昇格する場合）

---

### 2.2.9 QA（Q&A / 復習用）

- id
- user_id
- company_id（任意）
- question_text
- answer_text
- source_type（enum: 事前生成 / 振り返り改善）
- is_favorite
- review_status（enum: 未復習 / 復習済み / 要再確認）
- created_at
- updated_at

関係:

- QA N : 1 User
- QA N : 0..1 Company
- QA N : M Asset（どの自己資産を根拠にしたか）

---

### 2.2.10 GenerationJob（AI生成ジョブ）

- id
- user_id
- job_type（enum: 想定Q&A生成 / 添削 / 要約 / 企業研究要約 / ES生成 / ES推敲）
- input_snapshot（JSON: 送信したデータの要約）
- model_name
- token_used
- cost_estimate
- status（enum: pending / success / failed）
- error_message
- created_at

関係:

- GenerationJob N : 1 User

目的:

- コスト管理
- 失敗調査
- クレジット消費と紐付けるための外部参照ID（external_ref）の基点

---

### 2.2.11 Credit（クレジット / 課金ドメイン）

このドメインは詳細設計（DB/バックエンド設計）に委譲するが、ドメイン上は以下を担う。

役割：

- ユーザーのクレジット残高の管理
- クレジット購入（Stripe 連携）の履歴管理
- AI 実行などによるクレジット消費の記録
- 冪等性・二重消費防止・ロールバックの基点

ドメイン上の責務：

- 「残高があるか」の判定は常にサーバー側で行われる
- 消費は Reserve / Commit / Reversed の状態遷移モデルで管理される
- GenerationJob は external_ref としてクレジット取引と紐付けられる

---

## 2.3 エンティティ間の関係まとめ

- User をルートとしたマルチテナント構造
- Company, Asset, ResearchMaterial, ESProject, ESSection, Interview, QA, GenerationJob はすべて User に従属
- ResearchMaterial は Company に従属
- ESProject は Company に従属
- ESSection は ESProject に従属
- ESProject と Asset は N:M 関係（中間テーブルで関連付け）
- ESProject と ResearchMaterial は N:M 関係（中間テーブルで関連付け）
- Interview は Company に従属
- InterviewLog は Interview に従属
- QA は Company に紐づく場合と、汎用のものが存在する
- QA と Asset は N:M 関係（中間テーブルで関連付け）
- Credit ドメインは User に従属し、GenerationJob と外部参照（external_ref）で関連付けられる

---

## 2.4 ライフサイクル設計

### 2.4.1 自己資産

- 作成 → 編集 → （参照される）→ アーカイブ or 削除
- 削除時も、過去のQ&Aの履歴としては参照IDを保持するか検討

### 2.4.2 企業・面接

- 企業作成 → 面接追加 → 面接ログ追加 → 結果確定 → アーカイブ

### 2.4.3 ES

- 企業選択 → 企業研究資料登録/要約 → セクション作成 → AIドラフト生成 → 推敲 → 完了/提出
- セクションの版管理は履歴保持（差分比較可能）を前提とする

### 2.4.4 Q&A

- 生成 or 振り返りから作成 → 復習 → ステータス更新 → お気に入り or アーカイブ

### 2.4.5 クレジット

- 購入 → 残高加算
- AI実行要求 → 消費予約（pending）
- 成功 → 消費確定（committed）
- 失敗 → 取消（reversed）して残高復元
