# コストガード（ops/costguard）

prod の API（Cloud Run `interviewhack-api`）の利用量が無料枠に近づいたら、API だけを自動で止める小さな Go サービスです。
設計は [DesignDocument/12_Operation.md](../../DesignDocument/12_Operation.md) の 12.12 が正です。

## しくみ

```
Cloud Scheduler（5分ごと）──POST /check──┐
                                          ├─▶ interviewhack-costguard ──▶ Cloud Monitoring（当月の使用量）
予算通知 → Pub/Sub ──POST /budget────────┘            │
                                                       └─▶ Cloud Run Admin API：interviewhack-api を手動スケーリング 0 に
```

- 閾値のどれか1つに達したら、API を **手動スケーリング 0**（公式の「サービスの無効化」）にします。課金の無効化はしません
- 既に止まっていれば何もしません。止めたら ERROR ログを出し、メールが届きます
- 自動では再開しません（下の「止まったとき」）

## 閾値（無料枠の 90%）

2026-10-22 に [Cloud Run の料金ページ](https://cloud.google.com/run/pricing) で確認した、リクエスト課金の無料枠（請求先アカウント単位・毎月リセット）から計算しています。無料枠が変わったら `scripts/deploy.sh` の既定値を直して再デプロイしてください。

| 指標 | 無料枠 | 閾値（90%） | 環境変数 |
| --- | --- | --- | --- |
| リクエスト数 | 2,000,000 | 1,800,000 | `LIMIT_REQUESTS` |
| vCPU 秒 | 180,000 | 162,000 | `LIMIT_CPU_SECONDS` |
| GiB 秒 | 360,000 | 324,000 | `LIMIT_MEMORY_GIB_SECONDS` |

予算は prod プロジェクトだけを対象に月 300 円です（50% / 90% / 100% でメール、100% で停止）。

## 初回セットアップ（1回だけ）

`gcloud` にログインし、Docker が起動している状態で、リポジトリのルートから実行します。

```bash
BILLING_ACCOUNT=<請求先アカウントID> ALERT_EMAIL=<通知先メール> bash ops/costguard/scripts/setup.sh
```

作られるもの：サービスアカウント 3 つ（最小権限）、API の実行用アカウントの切り替えと最大インスタンス数 1、Artifact Registry のクリーンアップ、コストガード本体、Pub/Sub、予算、Cloud Scheduler、メールのアラート。何度実行しても同じ状態になります。

## コードを変えたとき

```bash
cd ops/costguard && go test ./...          # テスト
bash ops/costguard/scripts/deploy.sh       # デプロイ（リポジトリのルートから）
```

## 動作確認

```bash
gcloud scheduler jobs run costguard-check --location asia-northeast1 --project interviewhack-prod
gcloud logging read 'resource.labels.service_name="interviewhack-costguard" AND jsonPayload.event:*' \
  --project interviewhack-prod --freshness 10m --limit 5 \
  --format 'value(timestamp,jsonPayload.event,jsonPayload.requests,jsonPayload.cpu_seconds,jsonPayload.memory_gib_seconds)'
```

`costguard_usage` に当月の使用量が出ていれば正常です。

## 止まったとき（12.12.6）

1. メール（「costguard: API stopped or check failed」または予算通知）で気づく
2. Cloud Run のログで原因を確かめる（401 / 404 / 429 の急増なら攻撃）
3. 翌月1日（太平洋時間。日本時間では1日の 16〜17 時）以降に再開する

```bash
gcloud run services update interviewhack-api --region asia-northeast1 --project interviewhack-prod --scaling=auto
```

再開の直後は十数秒ほどエラーになることがあります。止まっている間に CD を実行すると、最初の確認で失敗します（先に再開してください）。
