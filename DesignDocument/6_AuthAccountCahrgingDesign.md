# 6. 認証・アカウント（Firebase Auth）

> 改訂：2026-10-22。方針転換メモ（`DirectionChange_20261022.md`）の決定事項 D1〜D16 と 2.10 共通契約を反映した。
> 旧版の課金（Stripe + クレジット方式、Reserve / Commit / Reversed）は D8・D14 により廃止した。ファイル名は互換のため据え置く。

## 6.1 方針（固定）

- 認証基盤：Firebase Auth を採用する（D5）
- 認証方式：Email/Password を先行して提供する
  - Google ログインは追加候補
  - Sign in with Apple は Apple Developer Program 加入後（フェーズ2）に追加する
- アプリ内のユーザー識別子は `users.id`（uuid）とし、Firebase の `uid` は `users.firebase_uid` として対応付ける
- 認可は Go BFF のアプリ層で行う（RLS は使わない。D6）
- 課金：MVP では行わない（D14。6.6 参照）

---

## 6.2 認証（Firebase Auth）

### 6.2.1 サポートするログイン手段

| 手段 | 時期 | 備考 |
|------|------|------|
| Email/Password | MVP（フェーズ1） | 必須 |
| Google | 追加候補 | Firebase Auth の Google プロバイダ |
| Sign in with Apple | フェーズ2（Developer Program 加入後） | 無料の Apple ID では使えないため。Google ログインを提供する場合は App Store ガイドライン上ほぼ必須 |

### 6.2.2 Email/Password（要件）

- 新規登録：メールアドレス＋パスワード（Firebase Auth の標準ルールに従う）
- ログイン / ログアウト
- ログイン状態はアプリ再起動後も維持される（Firebase iOS SDK のセッション）
- メールアドレスの確認（verification）は MVP では必須にしない（フェーズ2で再検討）

### 6.2.3 セッション管理

- iOS アプリは Firebase iOS SDK でログインし、ID トークンを取得する
  - ID トークンの有効期限は 1 時間で、SDK が自動で更新する
  - トークンとリフレッシュトークンは SDK が Keychain に保持する（D16）
- BFF 呼び出し時は `Authorization: Bearer <Firebase ID トークン>` を付与する
- BFF が 401 を返した場合、アプリはトークンを強制更新して 1 回だけ再試行する。失敗した場合のみログイン画面へ戻す
- ログアウト時は、Firebase のサインアウトに加えて端末内キャッシュ（SwiftData）と下書きをすべて削除する（D16）

### 6.2.4 アカウント紐付け方針（複数ログイン手段の共存）

- 原則：アプリ内の user_id は `users.id` のみとし、Firebase の `uid` 1 つに `users` 1 行を対応させる
- Firebase Auth の「メールアドレスごとに 1 アカウント」設定を有効にし、同一メールでの重複アカウントを作らない
- Google / Sign in with Apple を追加する際の既存アカウントとの統合（アカウントリンク）は、追加時に設計する（MVP 外）

### 6.2.5 パスワードリセット

- Firebase Auth の標準フロー（パスワード再設定メール）を利用する
- エラー文言はユーザー向けに整形する（存在しないメールアドレスかどうかを推測させない文言にする）

---

## 6.3 アカウント（DB：Neon / PostgreSQL）

### 6.3.1 users（ユーザー）

- 目的：Firebase の `uid` とアプリ内の user_id を対応付け、アプリ独自のプロフィール情報を持つ（旧 `auth_users` と `profiles` を統合）
- テーブル：`users`（詳細は 8 章 / `8_DBDesign.dbml`）
  - id (uuid, PK)
  - firebase_uid (text, unique, not null)
  - email (text)
  - display_name (text, nullable)
  - created_at / updated_at
  - deleted_at (nullable)
- 作成タイミング：BFF の認証ミドルウェアが、検証済みトークンの `uid` に対応する行がなければ作成する（初回アクセス時の upsert。7.4 参照）
  - email はトークンの `email` クレームから同期する（退会済みの行には同期しない）

### 6.3.2 マルチテナント（データ分離）

- 全ドメインデータ（assets / companies / research_materials / es_* / interviews / interview_logs / qa / prompt_sessions / share_links / dashboard_summaries）は user_id を持つ
- クライアントは DB に直接アクセスしない。すべて BFF 経由とする
- BFF の Repository は、全クエリに `user_id = $1` の条件を付ける（D6）
  - user_id は検証済みトークンから確定したものだけを使い、リクエストボディやクエリの user_id は信用しない
  - 他人のリソースの ID を指定された場合は 404（NOT_FOUND）を返す（存在を推測させない）
- 例外：共有ページ `GET /s/{token}` は認証なしで、トークンのハッシュで `share_links` のスナップショットだけを返す（7.5.6）

### 6.3.3 アカウント削除（退会）

- `DELETE /me` で退会する
- BFF の処理（1 トランザクション）
  - 当該ユーザーのドメインデータ（共有リンクを含む）を即時にすべて物理削除する。これにより共有ページも閲覧できなくなる（存在しないリンクとして 404）
  - `users` 行は物理削除せず、`deleted_at` を設定し、email と display_name を消去して残す。後から物理削除する処理は持たない
  - 退会後に古い ID トークン（同じ `firebase_uid`）で来たリクエストは 401 `UNAUTHORIZED` とする（`deleted_at` のある `users` は認証失敗。7.4）
- iOS アプリの処理
  - BFF の退会が成功したら、Firebase iOS SDK でユーザーを削除する（直近のログインが必要な場合は再認証を求める）
  - 端末内キャッシュと下書きをすべて削除する
- BFF は Firebase のユーザー削除を行わない（管理者権限の認証情報を BFF に持たせないため）

---

## 6.4 認可（RLS の代替）

- 認可の責務はすべて Go BFF が持つ（D6）
  - 認証：Adapter 層のミドルウェアで ID トークンを検証し、user_id を確定する（7.4）
  - 所有者チェック：Usecase は user_id を受け取り、Repository は user_id でスコープしたクエリだけを提供する
  - 関連テーブル（`*_relations`）の操作では、親リソースと関連先の両方が同じ user_id のものであることを確認する
- 管理者ロールは MVP では設けない
- 越境防止は統合テストで担保する（11 章）

---

## 6.5 プラン・利用制限

- MVP では Free / Pro のプラン判定や利用回数制限を設けない（D14）
- AI連携は LLM の API を呼ばないため、運営側に生成コストが発生しない（D7）。そのため、コスト防御のための回数制限は不要である
- 乱用防止のレート制限は 7.8 で定義する

---

## 6.6 課金（将来の方針のみ）

- MVP では収益化しない（D14）。Stripe 連携、クレジット、関連テーブルは廃止した
- 将来収益化する場合は、Apple のアプリ内課金（StoreKit）を使う
  - 候補：買い切りの Pro 解放（例：企業登録数の無制限化）
  - 検討を始めるタイミング：利用者が増え、Cloud Run / Neon の無料枠を超えそうになったとき
- 収益化を決めた時点で、本章と 2・7・8・9 章を改訂する

---

## 6.7 この章で凍結する仕様（合意事項）

- 認証：Firebase Auth（Email/Password 先行。Google は追加候補、Sign in with Apple はフェーズ2）
- user_id は `users.id`。Firebase の `uid` とは `users.firebase_uid` で対応付ける
- 認可は Go BFF のアプリ層で行い、全クエリを user_id でスコープする（RLS は使わない）
- 退会は `DELETE /me` で、1 トランザクションでドメインデータを物理削除し、`users` 行は `deleted_at` を設定して残す。Firebase のユーザー削除はクライアントが行う
- MVP では課金・プラン・利用回数制限を設けない
