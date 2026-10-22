# 14. CI/CD 設計（GitHub Actions / Cloud Run）

> 改訂：2026-10-22。方針転換メモ（`DirectionChange_20261022.md`）の決定事項 D1〜D16 と 2.10 共通契約（特に 2.10.6）を反映した。

## 14.1 方針

- 「人間がやらなくていいこと」はすべて自動化する（テスト / ビルド / バックエンドのデプロイ）
- 個人情報を扱うため、**本番へのデプロイは必ず手動承認を通す**
- 失敗時は「どこで落ちたか」が一目で分かるパイプラインにする
- シークレットはリポジトリに置かない。GitHub Secrets と Cloud Run のシークレット（Secret Manager）で管理する
- すべての成果物は Git SHA でタグ付けし、再現可能なデプロイを保証する
- バックエンドは DDD + クリーンアーキテクチャ（DesignDocument/7_BackendDesign.md）を前提とし、CI で層違反（Domain への外部依存など）を検出・防止する
- 運営費をほぼ0円に保つため、GitHub Actions の無料枠を意識する（特に macOS ランナー。14.5.2）

---

## 14.2 対象リポジトリ構成

- monorepo
  - `backend/`（Go + Gin、Docker、Cloud Run にデプロイ）
  - `ios/`（SwiftUI、Xcode プロジェクト。2.10.5）
  - `ops/costguard/`（コスト上限で API を止めるコストガード。BFF とは別の Go モジュール。12.12）
  - `frontend/`（Vite + React）は廃止済み（2026-10-22 に削除。CI / CD の対象からも外した）

---

## 14.3 環境（dev / prod）

- dev：ローカルの Go＋Neon の dev ブランチ＋Firebase の dev プロジェクト
- prod：Cloud Run＋Neon の main ブランチ＋Firebase の prod プロジェクト
- stg はフェーズ2（TestFlight 配布）で追加する（2.10.6）

CI/CD では以下を分離する：

- Cloud Run：service-prod（stg 追加時に service-stg）
- Neon：ブランチ（`main` / `dev`）ごとに `DATABASE_URL` を分ける
- Firebase：プロジェクト（dev / prod）ごとに `FIREBASE_PROJECT_ID` を分ける
- iOS：ビルド構成（`Debug-Dev` / `Release-Prod`。5.12）ごとに API の URL と `GoogleService-Info.plist` を分ける

---

## 14.4 ブランチ戦略（最小）

- main: 本番
- feature/\*: 機能開発

運用ルール：

- main への直接 push 禁止
- PR 経由でのみ main にマージ
- PR では必ず CI（テスト・ビルド）が通ること

---

## 14.5 CI（Pull Request 時）

### 14.5.1 バックエンド（backend/）

- `backend/` に変更があるときに実行する
- Go setup
- `go vet ./...`
- `golangci-lint`
- `go test ./...`（Domain / Usecase のユニットテスト）
- 統合テスト：GitHub Actions の PostgreSQL サービスコンテナを立て、マイグレーションを適用してから実行する
  - マイグレーションは2回適用し、冪等であることも確認する
  - 認可の越境防止、AI連携の commit の冪等性、共有リンクの期限・失効を必ず含める
- アーキテクチャチェック：`backend/scripts/arch_check.sh`（Domain が `gin` / `pgx` / `firebase` などの外部 SDK と、`net/http` などのネットワーク・DB 関連パッケージを import していないこと。7.2.1）
- Docker build（ビルドが通るかの検証）

### 14.5.2 iOS（ios/）

- `ios/` に変更があるときだけ実行する
  - macOS ランナーは Linux ランナーより無料枠の消費が大きいため（倍率は公式で要確認）
- macOS ランナーで XcodeGen によりプロジェクトを生成し、`xcodebuild test`（シミュレータ）を実行する
  - App ターゲットの単体テストと `InterviewHackKit` のテストを、同じスキームでまとめて実行する
- コード署名は行わない（テストのみ。配布はフェーズ2以降）
- `GoogleService-Info-Dev.plist` はコミットしないため（5.12.2）、CI では形式だけのダミー値でその場に生成する（秘密情報を含まない）

### 14.5.3 コストガード（ops/costguard/）

- `ops/costguard/` に変更があるときだけ実行する（Linux ランナー）
- gofmt、`go vet`、`golangci-lint`、`govulncheck`、`go test ./...`（Cloud Monitoring / Cloud Run Admin API はテスト用の HTTP サーバーで置き換える）、Docker build

共通ゲート：

- 依存関係の脆弱性チェック（backend：`govulncheck`）
- 成果物に `RELEASE=Git SHA` が埋め込まれていることの検証
- 変更内容が DesignDocument/7_BackendDesign.md（アーキテクチャ）および関連設計に準拠していることのレビュー観点チェック（PR テンプレートで宣言）

失敗したら PR はマージ不可

---

## 14.6 CD（main マージ後のデプロイ）

※ CD は CI の全ゲート（テスト・ビルド・アーキテクチャチェック）を通過した成果物のみをデプロイ対象とする。

### 14.6.1 本番（prod）へのデプロイ

- GitHub Actions の `environment: prod` を使用し **手動承認必須**
  - プランが Required reviewers に対応しているときは、そのレビュアー承認を待つ
  - 非公開リポジトリの無料プランでは Required reviewers を作れない（2026-10-22 に確認）。その間は `workflow_dispatch`（Actions 画面からの手動実行）だけを起動条件にし、人が実行したときだけデプロイする
- 承認後、次の順序で実行する（2.10.6）：
  0. 事前確認：API がコストガードで停止中（手動スケーリング 0）なら、ここで失敗させる（停止の解除は人が判断する。12.12.6）
  1. DB マイグレーション（`backend/migrations` を `psql` で番号順に適用。後方互換かつ冪等なもの）
  2. バックエンド：Docker build → Artifact Registry に push → Cloud Run（service-prod）へデプロイ
     - 最大インスタンス数 1、実行用サービスアカウントは `interviewhack-api`（12.2.3、12.12.5）
  3. スモークテスト
     - `GET /health`
     - 認証なしで叩けるチェック（存在しない共有トークンで `GET /s/{token}` が 404 を返す）
     - トークンなしの `GET /me` が 401 を返す（認証ミドルウェアが有効であること）
     - Cloud Run のサービス設定で `APP_ENV=prod` になっていること（14.11。値以外は出さない）
     - 失敗したら以降のステップを止める
- 起動は手動実行（`workflow_dispatch`）。Required reviewers が使えるプランになったら、main への `backend/` のマージでも起動し、`environment: prod` の承認を待つ
- 失敗した場合：
  - 直前のリビジョンへロールバック（Cloud Run のリビジョン機能を使用）

### 14.6.2 ステージング（stg）

- フェーズ2で追加する。追加後は stg へ自動デプロイ → スモークテスト → 承認 → prod の順にする

### 14.6.3 iOS

- フェーズ1は CD を持たない。開発者が Xcode から実機へ直接インストールする
- フェーズ2で、TestFlight へのアップロードを CI/CD に組み込むか検討する

### 14.6.4 コストガード

- 変更がまれなため、フェーズ1は CD を持たない。開発者が `ops/costguard/scripts/deploy.sh` で手動デプロイする（イメージは Git SHA でタグ付けする。14.9）
- 初回だけ `ops/costguard/scripts/setup.sh` で、サービスアカウント・権限・Pub/Sub・予算・Cloud Scheduler・ログのアラート・Artifact Registry のクリーンアップポリシーを作る（何度実行しても同じ状態になるように書く）

---

## 14.7 シークレット管理

### 14.7.1 GitHub に置くもの

- GCP への認証は **Workload Identity Federation を推奨**（サービスアカウントキーを GitHub に置かない）
  - 必要な値：Workload Identity Provider 名、デプロイ用サービスアカウント（秘密情報ではないが、GitHub の Variables / Secrets で管理する）
- マイグレーション用の `DATABASE_URL`（prod）：GitHub Secrets（`environment: prod` に紐付け、承認後のジョブからのみ参照する）

### 14.7.2 Cloud Run に置くもの

- `DATABASE_URL`：Secret Manager に保存し、Cloud Run にシークレットとして渡す
- そのほかの環境変数は秘密情報ではないため、通常の環境変数として設定する（14.8）

### 14.7.3 置いてはいけないもの

- `DATABASE_URL` をリポジトリやワークフローファイルに直書きすること
- GCP のサービスアカウントキー（JSON）
- `GoogleService-Info.plist` の prod 用の値をログに出すこと

---

## 14.8 環境変数の注入方針

### 14.8.1 バックエンド（Cloud Run）

| 変数 | 内容 | 秘密 |
|------|------|------|
| `DATABASE_URL` | Neon の接続文字列（prod は main ブランチ） | はい（Secret Manager） |
| `FIREBASE_PROJECT_ID` | ID トークン検証の `aud` | いいえ |
| `SHARE_BASE_URL` | 共有リンクの URL の基点 | いいえ |
| `APP_ENV` | `dev` / `prod` | いいえ |
| `LOG_LEVEL` | ログレベル | いいえ |
| `FEATURE_SHARE_VIEW` | 共有ページの閲覧（`on` / `off`、既定 `on`。12.11） | いいえ |
| `FEATURE_SHARE_CREATE` | 共有リンクの発行（`on` / `off`、既定 `on`。12.11） | いいえ |
| `FEATURE_COMMIT_DISABLED_USE_CASES` | commit を止めるユースケース（カンマ区切り、既定は空。12.11） | いいえ |

- Firebase ID トークンの検証は Google の公開鍵で行うため、Firebase のサービスアカウントキーは不要

### 14.8.3 コストガード（Cloud Run。12.12）

| 変数 | 内容 | 秘密 |
|------|------|------|
| `TARGET_PROJECT` / `TARGET_REGION` / `TARGET_SERVICE` | 停止する API のサービス | いいえ |
| `MONITORED_PROJECTS` | 使用量を合計するプロジェクト（カンマ区切り。既定は `TARGET_PROJECT`） | いいえ |
| `LIMIT_REQUESTS` | 当月のリクエスト数の閾値（無料枠の 90%） | いいえ |
| `LIMIT_CPU_SECONDS` | 当月の vCPU 秒の閾値（無料枠の 90%） | いいえ |
| `LIMIT_MEMORY_GIB_SECONDS` | 当月の GiB 秒の閾値（無料枠の 90%） | いいえ |
| `BUDGET_STOP_RATIO` | 予算通知で停止する比率（既定 1.0） | いいえ |
| `LOG_LEVEL` / `RELEASE` | バックエンドと同じ | いいえ |

- コストガードは秘密情報を持たない。Google の API へは実行用サービスアカウントの権限で接続する（鍵ファイルは使わない）

### 14.8.2 iOS

- API の URL と Firebase の設定は、ビルド構成（`Debug-Dev` / `Release-Prod`）ごとに切り替える（5.12）
- シークレットをアプリに埋め込まない（`GoogleService-Info.plist` の値は公開されても問題ない識別子のみ）

---

## 14.9 リリース識別子（Versioning）

- バックエンド：
  - Git の commit SHA を `RELEASE` として埋め込む
  - 構造化ログに `release` フィールドとして出力する
- iOS：
  - ビルド番号に Git の commit SHA（短縮形）を対応付ける

これにより：

- どのリリースで不具合が出たか即座に追跡可能

---

## 14.10 ロールバック戦略

### 14.10.1 バックエンド（Cloud Run）

- Cloud Run のリビジョン機能で、直前の安定リビジョンへ即時に切り戻す

### 14.10.2 DB（Neon）

- 原則ロールバックしない（後方互換マイグレーションを徹底する。12.4）
- データ破損時は Neon の復元機能を使う（12.8）

### 14.10.3 iOS

- 直前のコミットを Xcode で再ビルドし、実機に入れ直す

---

## 14.11 よくある事故の予防策

- prod 環境は **必ず手動承認**
- マイグレーションは prod 前に Neon の一時ブランチで適用を確認する（12.4）
- 環境変数の差分はデプロイ前に確認ログを出す（値そのものは出さない）
- dev の `DATABASE_URL` が prod の Cloud Run に入っていないかを確認する（`APP_ENV` と接続先ブランチの対応をスモークテストで検証）
- Domain / Usecase にフレームワーク・SDK 依存を混入させない（アーキテクチャチェックで検出）
- macOS ランナーを不要に動かさない（`ios/` のパスフィルタ）
- コストガードで停止中の API に、CD がデプロイして停止を見えにくくしない（事前確認で失敗させる。14.6.1）

---

## 14.12 MVP 時点の簡略版（現実解）

- PR：CI のみ
- main マージ後：手動承認 → prod デプロイ（stg なし）
- iOS：CI でテストのみ。配布は Xcode から実機へ直接
- フェーズ2（TestFlight 配布）で stg を追加し、stg → 承認 → prod のフローに昇格させる

---

## 14.13 この章の凍結事項

- CI/CD は GitHub Actions を使用する
- PR では必ずテスト + ビルドを通す。iOS の CI は `ios/` 変更時のみ実行する
- 本番デプロイは手動承認必須。順序はマイグレーション → バックエンド → スモークテスト
- シークレットは GitHub に直置きせず、`DATABASE_URL` は Secret Manager で管理する。GCP 認証は Workload Identity Federation を推奨する
- ロールバック可能な構成（Cloud Run リビジョン）を前提にする
- バックエンドは DDD + クリーンアーキテクチャの依存方向を破らない（DesignDocument/7_BackendDesign.md 準拠、CI で検証）
