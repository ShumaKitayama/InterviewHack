# InterviewHack

就活・転職の面接対策アシスタント（iPhone アプリ）。自己資産（ガクチカ・経歴など）と企業研究資料を整理し、ES 作成・想定 Q&A・面接ログの添削・復習までを 1 つのアプリで回します。

AI は **API を呼ばない「AI連携（コピペ方式）」** で使います。アプリがプロンプトを組み立て、ユーザーが ChatGPT・Gemini などに渡し、回答を貼り戻すと BFF が解析して保存します。運営側の LLM 費用と API キーはありません（設計：[DesignDocument/DirectionChange_20261022.md](DesignDocument/DirectionChange_20261022.md)）。

## アーキテクチャ

```
┌──────────────────────────┐   Firebase ID トークン   ┌───────────────────────────┐
│ iOS アプリ (SwiftUI)      │ ───────────────────────▶ │ Go BFF (Gin) on Cloud Run │
│ InterviewHackKit / UI     │   X-Request-Id           │ DDD + Clean Architecture  │
│ SwiftData キャッシュ       │ ◀─────────────────────── │ 共有ページ (/s/{token})    │
└──────────┬───────────────┘                          └────────────┬──────────────┘
           │ 共有シート / クリップボード                              │ pgx
           ▼                                                        ▼
   ChatGPT / Gemini（ユーザー自身）                          Neon (PostgreSQL)
           ▲
           └ Firebase Auth（メールアドレス＋パスワード）
```

| コンポーネント | 技術 |
| --- | --- |
| クライアント | iOS 17+ / SwiftUI / SwiftData / Firebase Auth iOS SDK（`ios/`） |
| BFF | Go 1.27 / Gin / pgx v5 / DDD + Clean Architecture（`backend/`） |
| DB | Neon（PostgreSQL）。RLS は使わず BFF のアプリ層で user_id 認可 |
| 認証 | Firebase Auth（BFF は Google の公開鍵で ID トークンを検証） |
| AI | API なし。プロンプト生成 → 外部AI → 貼り戻し → 解析（目印付きテキスト） |
| CI/CD | GitHub Actions（14 章） |

## プロジェクト構造

```
InterviewHack/
├── DesignDocument/              # 設計書（Source of Truth）
├── backend/
│   ├── cmd/api/                 # main（設定・DI・サーバー起動）
│   ├── internal/
│   │   ├── domain/              # entity / vo / error / service（prompt・share・session・es）
│   │   ├── usecase/             # account / asset / company / research / es / interview / qa / dashboard / prompt / share
│   │   ├── port/                # repository / external の interface
│   │   ├── adapter/http/        # handler / middleware / presenter / dto / sharepage
│   │   ├── infra/               # postgres / firebase / system / logging / config
│   │   └── app/                 # DI（cmd/api と統合テストで共有）＋ 統合テスト
│   ├── migrations/              # 冪等な SQL（psql で番号順に適用）
│   └── scripts/                 # migrate.sh / arch_check.sh
└── ios/
    ├── InterviewHackKit/        # Swift Package（ロジック・画面。swift test で検証）
    ├── App/                     # アプリ本体（Firebase / SwiftData。Xcode でビルド）
    └── project.yml              # XcodeGen 定義
```

## セットアップ

### 前提

- Go 1.27+
- PostgreSQL 15+（ローカル開発・テスト用。Neon の dev ブランチでもよい）
- iOS：Xcode（iOS 17 以上の SDK）と XcodeGen（アプリ本体のビルド）。ロジックのテストだけなら Command Line Tools の Swift 6 で可

### Backend

```bash
cd backend
cp .env.example .env   # DATABASE_URL と FIREBASE_PROJECT_ID を設定
```

マイグレーション（冪等。何度実行してもよい）：

```bash
DATABASE_URL=postgres://postgres@localhost:5432/interviewhack ./backend/scripts/migrate.sh
```

起動（`backend` ディレクトリで実行。`.env` を読み込む）：

```bash
cd backend && go run ./cmd/api
```

`http://localhost:8080/health` が `{"status":"ok"}` を返せば OK です。

### 環境変数（backend）

| 変数 | 内容 | 必須 |
| --- | --- | --- |
| `DATABASE_URL` | Neon / PostgreSQL の接続文字列（秘匿） | はい |
| `FIREBASE_PROJECT_ID` | ID トークン検証の `aud` | はい |
| `SHARE_BASE_URL` | 共有リンクの URL の基点 | いいえ（既定 `http://localhost:8080`） |
| `APP_ENV` | `dev` / `stg` / `prod` | いいえ（既定 `dev`） |
| `LOG_LEVEL` | `debug` / `info` / `warn` / `error` | いいえ |
| `FEATURE_SHARE_VIEW` / `FEATURE_SHARE_CREATE` | 共有ページの閲覧 / 発行（`off` で停止） | いいえ（既定 `on`） |
| `FEATURE_COMMIT_DISABLED_USE_CASES` | commit を止める use_case（カンマ区切り） | いいえ |

### iOS

ロジックと画面のテスト（Xcode 不要）：

```bash
cd ios/InterviewHackKit && swift test
```

Command Line Tools だけの環境で `TestingMacros` のプラグインが見つからない場合は、[ios/README.md](ios/README.md) の回避策を使ってください。アプリ本体のビルド手順（XcodeGen、`GoogleService-Info.plist` の置き方、Debug-Dev / Release-Prod）も [ios/README.md](ios/README.md) にあります。

## テスト

```bash
cd backend && go vet ./... && go test ./...
```

統合テスト（実 PostgreSQL。`TEST_DATABASE_URL` を指定したときだけ実行し、テーブルを TRUNCATE するので専用 DB を使うこと）：

```bash
cd backend && TEST_DATABASE_URL=postgres://postgres@localhost:5432/interviewhack_test go test ./...
```

アーキテクチャ検査（Domain / Usecase / Port の依存ルール）：

```bash
cd backend && bash scripts/arch_check.sh
```

プロンプトのゴールデンファイル更新（テンプレートを意図して変えたときだけ）：

```bash
cd backend && go test ./internal/domain/service/prompt -run TestBuildAllUseCases -update
```

BFF ↔ iOS の契約 fixture の再生成（API を変えたとき）：

```bash
cd backend && TEST_DATABASE_URL=postgres://postgres@localhost:5432/interviewhack_test CONTRACT_OUT=../ios/InterviewHackKit/Tests/InterviewHackKitTests/Fixtures/contract go test ./internal/app -run TestContractFixtures
```

Lint と脆弱性チェック（CI と同じもの）：

```bash
cd backend && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run ./...
cd backend && go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

## CI/CD（14 章）

| ワークフロー | いつ動くか | 内容 |
| --- | --- | --- |
| `ci-backend.yml` | `backend/` を変えた PR | gofmt / vet / golangci-lint / govulncheck / arch_check / マイグレーション 2 回 / ユニット＋統合テスト / Docker build / RELEASE 埋め込み確認 |
| `ci-ios.yml` | `ios/` を変えた PR | XcodeGen → シミュレータで `xcodebuild test`（署名なし、ダミーの plist） |
| `cd-production.yml` | `backend/` の main マージ、または手動 | 承認待ち → マイグレーション → Cloud Run デプロイ → スモークテスト（失敗時は直前のリビジョンへ戻す） |

### 本番環境の準備（初回だけ）

以下はアカウントが必要なので、ブラウザとターミナルで 1 回だけ行います。`<...>` は自分の値に置き換えてください。

1. **Neon**：[console.neon.tech](https://console.neon.tech) でプロジェクトを作り、`main` と `dev` のブランチの接続文字列（`postgres://...?sslmode=require`）を控える
2. **Firebase**：[console.firebase.google.com](https://console.firebase.google.com) で dev 用と prod 用の 2 プロジェクトを作り、Authentication の「メール / パスワード」を有効にする。iOS アプリ（バンドル ID は dev が `jp.interviewhack.app.dev`、prod が `jp.interviewhack.app`）を登録し、plist を `ios/App/Resources/` に置く（[ios/App/Resources/README.md](ios/App/Resources/README.md)）
3. **GCP**（prod の Firebase プロジェクトと同じ GCP プロジェクトでよい）：

```bash
PROJECT_ID=<GCPプロジェクトID>; REGION=asia-northeast1; REPO=ShumaKitayama/InterviewHack
gcloud config set project $PROJECT_ID
gcloud services enable run.googleapis.com artifactregistry.googleapis.com secretmanager.googleapis.com iamcredentials.googleapis.com
gcloud artifacts repositories create interviewhack --repository-format=docker --location=$REGION
printf '%s' '<Neon main の接続文字列>' | gcloud secrets create interviewhack-database-url --data-file=-

# デプロイ用サービスアカウントと Workload Identity Federation（鍵を GitHub に置かない。14.7.1）
gcloud iam service-accounts create github-deployer
SA=github-deployer@$PROJECT_ID.iam.gserviceaccount.com
for role in roles/run.admin roles/artifactregistry.writer roles/iam.serviceAccountUser; do
  gcloud projects add-iam-policy-binding $PROJECT_ID --member=serviceAccount:$SA --role=$role
done
PROJECT_NUMBER=$(gcloud projects describe $PROJECT_ID --format='value(projectNumber)')
gcloud secrets add-iam-policy-binding interviewhack-database-url \
  --member=serviceAccount:$PROJECT_NUMBER-compute@developer.gserviceaccount.com --role=roles/secretmanager.secretAccessor
gcloud iam workload-identity-pools create github --location=global
gcloud iam workload-identity-pools providers create-oidc github --location=global --workload-identity-pool=github \
  --issuer-uri=https://token.actions.githubusercontent.com \
  --attribute-mapping=google.subject=assertion.sub,attribute.repository=assertion.repository \
  --attribute-condition="assertion.repository=='$REPO'"
gcloud iam service-accounts add-iam-policy-binding $SA --role=roles/iam.workloadIdentityUser \
  --member="principalSet://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/github/attribute.repository/$REPO"
echo "GCP_WIF_PROVIDER=projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/github/providers/github"
```

4. **GitHub**：リポジトリの Settings → Environments で `prod` を作り、Required reviewers に自分を入れて、次を登録する

| 種類 | 名前 | 値 |
| --- | --- | --- |
| Secret | `PROD_DATABASE_URL` | Neon main の接続文字列 |
| Variable | `GCP_PROJECT_ID` / `GCP_REGION` | 上の `PROJECT_ID` / `asia-northeast1` |
| Variable | `GCP_WIF_PROVIDER` | 上の echo の出力 |
| Variable | `GCP_DEPLOY_SERVICE_ACCOUNT` | `github-deployer@<PROJECT_ID>.iam.gserviceaccount.com` |
| Variable | `GCP_ARTIFACT_REPOSITORY` | `interviewhack` |
| Variable | `CLOUD_RUN_SERVICE` | `interviewhack-api` |
| Variable | `CLOUD_RUN_DATABASE_URL_SECRET` | `interviewhack-database-url` |
| Variable | `FIREBASE_PROJECT_ID` | prod の Firebase プロジェクト ID |
| Variable | `SHARE_BASE_URL` | Cloud Run の URL（初回デプロイ後に分かる。それまでは仮の値で、確定後に再デプロイ） |

5. Actions タブで「CD Production」を手動実行し、承認する。デプロイ後の URL を `ios/App/Config/Prod.xcconfig` の `API_BASE_URL` に入れる

## 設計原則

- 設計書（`DesignDocument/`）がコードより優先します。変更は設計書から（[AGENTS.md](AGENTS.md)）
- BFF は LLM の API・外部AI・任意の URL に通信しません
- 全クエリを user_id でスコープし、他人のリソースは 404 を返します
- AI連携の保存は条件付き UPDATE で 1 回だけ成功させます（二重保存なし）
- 共有リンクの DB にはトークンのハッシュだけを保存します
- ログに本文・トークン・個人情報を出しません

## ライセンス

Private
