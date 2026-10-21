# 2. 情報設計（ドメイン設計）

> 改訂：2026-10-22。方針転換メモ（`DirectionChange_20261022.md`）の決定事項 D1〜D16 と 2.10 共通契約を反映した。

## 2.1 ドメイン全体像

本プロダクトは以下の主要ドメインで構成される。

- ユーザー
- 自己資産（ユーザーの経歴・エピソード・スキル等）
- 企業（応募先・職種・選考フェーズ）
- 企業研究資料（端末で取り込んだ URL / PDF / 画像のテキストとメモ、およびその要約）
- ESプロジェクト（企業ごとの作成単位）
- ESセクション（志望動機・自己PR等の本文）
- 面接
- 面接ログ（質問・回答・AI連携で取り込んだ添削結果）
- Q&A（想定質問・模範解答・復習用データ）
- プロンプトセッション（AI連携 1 回分の進行状態。旧「AI生成ジョブ」を置き換える。D13）
- 共有リンク（ESプロジェクトのスナップショット公開。D12）
- ダッシュボード要約（AI連携で取り込んだ「次に直すべきポイント」）

これらは「準備（自己資産・企業研究）→ ES作成/面接実施 → 振り返り → 復習」というループを回すために相互に関連する。

AI を使う処理はすべて AI連携（コピペ方式、1.1.0）で行う。ドメインとしては「プロンプトを組み立てる」「貼り戻された回答を解析する」「確認済みの結果を本来のエンティティに保存する」までを扱い、LLM の呼び出しは扱わない（D7）。

旧設計の「クレジット（残高・消費・購入履歴）」ドメインは廃止した（D8 / D14）。

---

## 2.2 エンティティ定義

### 2.2.1 User（ユーザー）

- id
- firebase_uid（Firebase Auth のユーザー ID。一意）
- email
- display_name
- created_at
- updated_at
- deleted_at（退会日時）

注記:

- 認証は Firebase Auth が担う（D5）。パスワードはアプリ側で保持しない
- プラン（Free / Pro）の概念は持たない（D14）
- BFF は Firebase ID トークンの検証後、firebase_uid から User を取得または作成し、以後の user_id とする
- 退会時は、User に従属するデータ（共有リンクを含む）を即時に物理削除する。User 自体は物理削除せず、deleted_at を設定し、email と display_name を消して残す。deleted_at のある User は認証に失敗する（6.3.3）

関係:

- User 1 : N 自己資産
- User 1 : N 企業
- User 1 : N 企業研究資料
- User 1 : N ESプロジェクト
- User 1 : N ESセクション
- User 1 : N 面接
- User 1 : N 面接ログ
- User 1 : N Q&A
- User 1 : N プロンプトセッション
- User 1 : N 共有リンク
- User 1 : N ダッシュボード要約

---

### 2.2.2 Asset（自己資産）

- id
- user_id
- type（enum: 学業, 職務経歴, プロジェクト, スキル, 強み, 弱み, エピソード, その他）
- title
- content（本文）
- tags（中間テーブル）
- created_at
- updated_at
- deleted_at

関係:

- Asset N : 1 User
- Asset N : M Q&A（どの資産を参照して作られたかを記録）
- Asset N : M ESProject（ES作成に使った資産）

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
- Company 1 : N Q&A（この企業向けに作成されたもの）

---

### 2.2.4 ResearchMaterial（企業研究資料 / セット2拡張）

- id
- company_id
- user_id
- title
- type（enum: url, pdf, note, image）
- source_url（任意。type = url のとき）
- content_raw（端末で抽出したテキスト。URL は WebView からの本文抽出、PDF は PDFKit、スキャンPDF・画像は Vision の OCR）
- summary（AI連携 `research_summarize` で取り込んだ要約）
- created_at
- updated_at

関係:

- ResearchMaterial N : 1 Company
- ResearchMaterial N : M ESProject（ES作成時に参照）

制約（D11）:

- ファイル本体（PDF・画像）は保存しない。保存するのは抽出テキストのみ
- テキストの取得・抽出は iOS 側で行い、サーバーは URL を取得しない
- 自動クローリングは行わない（ユーザー操作による 1 ページ単位の取り込みのみ）

---

### 2.2.5 ESProject（ES作成プロジェクト）

- id
- user_id
- company_id
- status（enum: not_started, draft, in_review, completed, submitted。初期値 not_started）
  - 対応：not_started＝未着手、draft＝作成中、in_review＝添削中、completed＝完了、submitted＝提出済（1.1.4）
  - 遷移の制約はない（ユーザーがどの状態からでも変更できる）
  - 自動遷移は1つだけ：not_started のとき、いずれかのセクションの本文が保存されたら（AI連携の取り込み、または手入力）draft にする
- deadline（提出期限）
- created_at
- updated_at

関係:

- ESProject N : 1 User
- ESProject N : 1 Company
- ESProject 1 : N ESSection
- ESProject N : M ResearchMaterial（参照資料）
- ESProject N : M Asset（使用した自己資産）
- ESProject 1 : N ShareLink（共有リンク）

---

### 2.2.6 ESSection（ES項目 / 志望動機・自己PRなど）

- id
- es_project_id
- user_id
- title（例: 志望動機, 自己PR, ガクチカ）
- content_draft（AI連携で取り込んだ文章、または編集中の文章）
- content_final（完成版）
- character_limit（文字数制限）
- version（本文を更新するたびに +1 する。過去の本文は ESSectionVersion に残す）
- created_at
- updated_at

関係:

- ESSection N : 1 ESProject
- ESSection 1 : N ESSectionVersion

---

### 2.2.6.1 ESSectionVersion（ESセクションの版）

- id
- es_section_id
- user_id
- version（保存した時点の ESSection.version）
- content（保存した時点の content_draft）
- source（enum: manual / es_generate / es_refine）
- created_at

ルール:

- 版を作るのは次の2つのときだけ（普段の編集や自動保存では作らない）
  - AI連携（es_generate / es_refine）の commit で content_draft を上書きする直前。source はその use_case
  - ユーザーが「この版を保存」を押したとき。source は manual
- 「この版に戻す」では、戻す直前の content_draft を manual の版として残してから、content_draft を選んだ版の content にする
- 1セクションあたり最新20件まで。版を作るとき、同じトランザクションで超えた分を古い順に削除する
- 差分は iOS 側で計算する（サーバーは本文を返すだけ）
- 対象は content_draft だけとする（content_final は版を持たない）

関係:

- ESSectionVersion N : 1 ESSection（ESSection を削除すると版も削除する）

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
- user_id
- question_text
- answer_text
- self_evaluation（任意: 自己評価メモ）
- ai_feedback_text（AI連携 `answer_review` で取り込んだ指摘・改善案）
- ai_improved_answer_text（AI連携 `answer_review` で取り込んだ改善後の回答）
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
- source_type（enum: 事前作成（`qa_generate` の取り込み） / 振り返り改善（`answer_review` の取り込み））
- 注記：`qa_generate` の回答に含まれる「質問の意図」（`【意図N】`）は、取り込みプレビューに表示するだけで保存しない
- is_favorite（お気に入り。旧要件の「ピン留め」もこれに統一する。お気に入りは一覧の先頭に表示し、絞り込みにも使う）
- review_status（enum: 未復習 / 復習済み / 要再確認）
- created_at
- updated_at

関係:

- QA N : 1 User
- QA N : 0..1 Company
- QA N : M Asset（どの自己資産を根拠にしたか）

---

### 2.2.10 PromptSession（プロンプトセッション / AI連携 1 回分）

旧 GenerationJob を置き換える（D13）。

- id
- user_id
- use_case（enum: qa_generate / answer_review / summary_generate / research_summarize / es_generate / es_refine）
- template_version（例: qa-v1, review-v1）
- 対象の参照（company_id / es_project_id / es_section_id / interview_id / interview_log_id。すべて任意）
- input_refs（JSON: 選んだ自己資産・企業研究資料の ID とオプションのみ。本文は入れない）
- status（enum: created / pasted / saved / abandoned）
- parse_result（enum: success / partial / failed。未解析時は空。解析しないユースケースは success）
- created_at
- pasted_at
- completed_at

関係:

- PromptSession N : 1 User
- PromptSession N : 0..1 Company / ESProject / ESSection / Interview / InterviewLog

目的:

- 「続きから貼り戻す」：外部AIとの往復中にアプリが終了しても、未完了（created / pasted）のセッションから再開できる
- テンプレート品質の計測：template_version ごとの解析成功率・離脱率（D15）

ドメインルール:

- プロンプトの全文は保存しない。必要になったら input_refs と元データから再生成する
- 貼り戻された回答の生データは保存しない。確認・修正後の結果は、本来のエンティティ（QA、InterviewLog、ESSection、ResearchMaterial.summary、DashboardSummary）に保存する
- 解析するユースケース（qa_generate / answer_review / research_summarize）は、目印付きテキスト形式（2.10.1）で解析する。回答内の形式行のバージョンを優先し、なければ template_version を使う
- 解析しないユースケース（summary_generate / es_generate / es_refine）も解析（parse）を経由し、形式行があれば取り除いた全文をプレビューしてから取り込む（parse_result = success）
- テンプレート生成と解析は、Domain の純粋関数として実装する（use_case × バージョン）

各ユースケースの取り込み先:

| use_case | 取り込み先 |
|----------|-----------|
| qa_generate | QA（source_type = 事前作成）、QA ⇔ Asset |
| answer_review | InterviewLog.ai_feedback_text / ai_improved_answer_text（必要に応じて QA に昇格） |
| summary_generate | DashboardSummary |
| research_summarize | ResearchMaterial.summary |
| es_generate | ESSection.content_draft |
| es_refine | ESSection.content_draft |

---

### 2.2.11 ShareLink（共有リンク）

D12 に従う。

- id
- user_id
- es_project_id
- token_hash（共有トークンのハッシュ。一意）
- snapshot（JSON: 公開するセクションの title と本文（content_final、なければ content_draft）。企業名を表示する場合は企業名）
- show_company_name（既定値 false）
- expires_at
- revoked_at
- created_at

関係:

- ShareLink N : 1 User
- ShareLink N : 1 ESProject

ドメインルール:

- 共有単位は ESProject。作成時に公開するセクションを選ぶ
- 内容は作成時点のスナップショットで固定する。以後の ESSection の編集は反映しない
- トークンは 32 バイト（256 ビット）の暗号学的乱数とする（D12 の「128 ビット以上」を満たす）。保存するのは SHA-256 のハッシュのみで、生トークンは作成時に一度だけ返す
- 有効期限は 1日 / 7日 / 30日 から選ぶ（初期値 7日）。無期限は不可
- 失効（revoked_at の設定）はいつでも可能。期限切れ・失効したリンクは閲覧できない
- ユーザー名は snapshot に含めない。企業名は show_company_name が真のときのみ含める
- 閲覧はログイン不要で、閲覧のみ（コメント機能は持たない）
- ESProject を削除すると、紐づく ShareLink も削除する。期限切れ・失効した ShareLink は、運用で定期的に物理削除してよい

---

### 2.2.12 DashboardSummary（ダッシュボード要約）

- id
- user_id
- content（AI連携 `summary_generate` で取り込んだ「次に直すべきポイント」の全文）
- source_interview_id（任意: 要約の元になった面接）
- created_at

関係:

- DashboardSummary N : 1 User
- DashboardSummary N : 0..1 Interview

ドメインルール:

- ダッシュボードには最新の 1 件を表示する

---

## 2.3 エンティティ間の関係まとめ

- User をルートとしたマルチテナント構造
- Company, Asset, ResearchMaterial, ESProject, ESSection, ESSectionVersion, Interview, InterviewLog, QA, PromptSession, ShareLink, DashboardSummary はすべて User に従属
- ResearchMaterial は Company に従属
- ESProject は Company に従属
- ESSection は ESProject に従属
- ESSectionVersion は ESSection に従属
- ShareLink は ESProject に従属
- ESProject と Asset は N:M 関係（中間テーブルで関連付け）
- ESProject と ResearchMaterial は N:M 関係（中間テーブルで関連付け）
- Interview は Company に従属
- InterviewLog は Interview に従属
- QA は Company に紐づく場合と、汎用のものが存在する
- QA と Asset は N:M 関係（中間テーブルで関連付け）
- PromptSession は対象（Company / ESProject / ESSection / Interview / InterviewLog）を任意で参照する。取り込み結果は対象のエンティティに保存される
- 他ユーザーのデータへの参照は一切持たない。認可は BFF で user_id によって行う（D6）

---

## 2.4 ライフサイクル設計

### 2.4.1 自己資産

- 作成 → 編集 → （プロンプトに含めて参照される）→ 削除（論理削除：`deleted_at`）
- 削除時も、過去のQ&Aの履歴としては参照IDを保持するか検討

### 2.4.2 企業・面接

- 企業作成 → 面接追加 → 面接ログ追加 → 添削の AI連携 → 結果確定
- 選考が終わった企業は削除（論理削除：`deleted_at`）で一覧から外す

### 2.4.3 ES

- 企業選択 → 企業研究資料の取り込み（端末で抽出）/ 要約の AI連携 → セクション作成 → ドラフトの AI連携 → 推敲の AI連携 → 完了/提出
- ステータス：作成時は not_started。本文が初めて保存されると自動で draft。以降はユーザーが自由に変更する（2.2.5）
- 必要に応じて共有リンクを発行し、友人・メンターに確認してもらう
- セクションの版管理は ESSectionVersion で行う（2.2.6.1）。AI連携の取り込み直前と手動保存のときに版を残し、最新20件まで保持する。任意の版と差分を比較し、その版に戻せる

### 2.4.4 Q&A

- AI連携の取り込み or 振り返りから作成 → 復習（オフラインでも閲覧可。D16）→ ステータス更新 → お気に入り（必要に応じて）
- アーカイブは持たない。片付いたものは復習ステータス（復習済み）で区別し、不要なものは削除する

### 2.4.5 プロンプトセッション（AI連携）

- 作成（created）：プロンプトを生成し、ユーザーが外部AIに渡す
- 貼り戻し（pasted）：ユーザーが回答を貼り戻し、BFF が解析してプレビューを返す（この時点では何も保存しない）。解析しないユースケースも、形式行を取り除いたプレビューのためにこの段階を経由する
- 保存（saved）：ユーザーが確認・修正した結果を本来のエンティティに保存する
- 離脱（abandoned）：ユーザーが明示的に中止した場合、または作成から 7 日を過ぎても created / pasted のままの場合。定期ジョブは持たず、一覧の取得時に更新する
- 遷移は created → pasted（parse）、pasted → pasted（再解析）、pasted → saved（commit）、created / pasted → abandoned のみ。created → saved の直接の commit は許可しない
- saved / abandoned は終端状態。保存（commit）は条件付き更新で 1 回だけ成功させ、二重保存を防ぐ

### 2.4.6 共有リンク

- 発行（スナップショット作成、期限設定）→ 閲覧（期限内かつ未失効のみ）→ 期限切れ or 失効
- 期限切れ・失効後は閲覧不可。再共有は新しいリンクを発行する
- ESプロジェクトの削除・退会時は物理削除する。期限切れ・失効したリンクも、運用で定期的に物理削除してよい
