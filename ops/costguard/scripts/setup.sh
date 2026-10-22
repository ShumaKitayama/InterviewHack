#!/usr/bin/env bash
# DesignDocument/12_Operation.md 12.12（コスト上限と自動停止）/ 14_CICDDesign.md 14.6.4
# 初回だけ実行する。何度実行しても同じ状態になる（既にあるものは作り直さない）。
#
# 使い方：
#   BILLING_ACCOUNT=XXXXXX-XXXXXX-XXXXXX ALERT_EMAIL=you@example.com bash ops/costguard/scripts/setup.sh
#
# 作るもの：
#   1. サービスアカウント 3 つ（interviewhack-api / costguard / costguard-invoker）と最小権限（12.12.5）
#   2. API を専用サービスアカウント・最大インスタンス数 1 に切り替える（12.2.3）
#   3. Artifact Registry のクリーンアップポリシー（12.12.7）
#   4. コストガードのデプロイ（deploy.sh）
#   5. 予算通知用の Pub/Sub トピックと push サブスクリプション、予算（12.12.3）
#   6. 5 分ごとの Cloud Scheduler ジョブ
#   7. 停止・失敗をメールで知らせるログベースのアラート（12.6.3）
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

: "${BILLING_ACCOUNT:?BILLING_ACCOUNT（請求先アカウント ID）を指定してください}"
: "${ALERT_EMAIL:?ALERT_EMAIL（停止を知らせるメールアドレス）を指定してください}"
export PROJECT_ID=${PROJECT_ID:-interviewhack-prod}
export REGION=${REGION:-asia-northeast1}
export AR_REPO=${AR_REPO:-interviewhack}
export API_SERVICE=${API_SERVICE:-interviewhack-api}
export GUARD_SERVICE=${GUARD_SERVICE:-interviewhack-costguard}
DB_SECRET=${DB_SECRET:-interviewhack-database-url}
BUDGET_AMOUNT=${BUDGET_AMOUNT:-300JPY}
BUDGET_NAME="${PROJECT_ID} costguard"
TOPIC=costguard-budget
SUBSCRIPTION=costguard-budget-push
SCHEDULER_JOB=costguard-check
ALERT_POLICY_NAME="costguard: API stopped or check failed"

API_SA="interviewhack-api@${PROJECT_ID}.iam.gserviceaccount.com"
GUARD_SA="costguard@${PROJECT_ID}.iam.gserviceaccount.com"
INVOKER_SA="costguard-invoker@${PROJECT_ID}.iam.gserviceaccount.com"
PROJECT_NUMBER=$(gcloud projects describe "$PROJECT_ID" --format 'value(projectNumber)')
COMPUTE_SA="${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"
g() { gcloud --project "$PROJECT_ID" --quiet "$@"; }

echo "==> enable APIs"
g services enable run.googleapis.com artifactregistry.googleapis.com monitoring.googleapis.com \
  logging.googleapis.com cloudscheduler.googleapis.com pubsub.googleapis.com billingbudgets.googleapis.com

echo "==> service accounts"
for name in interviewhack-api costguard costguard-invoker; do
  g iam service-accounts describe "${name}@${PROJECT_ID}.iam.gserviceaccount.com" >/dev/null 2>&1 ||
    g iam service-accounts create "$name" --display-name "$name"
done

echo "==> API runtime: ${API_SA} (secret only), max instances 1"
g secrets add-iam-policy-binding "$DB_SECRET" --member "serviceAccount:${API_SA}" \
  --role roles/secretmanager.secretAccessor >/dev/null
g run services update "$API_SERVICE" --region "$REGION" --service-account "$API_SA" --max-instances 1
# Secrets are read at every instance start, so the default compute account
# (project Editor) keeps access until no service runs as it any more
# (an empty service account also means the default compute account).
if g run services list --region "$REGION" --format 'value(spec.template.spec.serviceAccountName)' |
  awk -v sa="$COMPUTE_SA" '$0 == sa || $0 == "" { found = 1 } END { exit !found }'; then
  echo "    a service still runs as ${COMPUTE_SA}; keeping its secret access"
else
  g secrets remove-iam-policy-binding "$DB_SECRET" --member "serviceAccount:${COMPUTE_SA}" \
    --role roles/secretmanager.secretAccessor >/dev/null 2>&1 || true
fi

echo "==> cost guard permissions (only the API service can be changed)"
g projects add-iam-policy-binding "$PROJECT_ID" --member "serviceAccount:${GUARD_SA}" \
  --role roles/monitoring.viewer --condition None >/dev/null
g run services add-iam-policy-binding "$API_SERVICE" --region "$REGION" \
  --member "serviceAccount:${GUARD_SA}" --role roles/run.developer >/dev/null
g iam service-accounts add-iam-policy-binding "$API_SA" \
  --member "serviceAccount:${GUARD_SA}" --role roles/iam.serviceAccountUser >/dev/null

echo "==> Artifact Registry cleanup (keep the newest 5 per image)"
POLICY_FILE=$(mktemp)
trap 'rm -f "$POLICY_FILE"' EXIT
cat >"$POLICY_FILE" <<'JSON'
[
  {"name": "keep-newest-5", "action": {"type": "Keep"}, "mostRecentVersions": {"keepCount": 5}},
  {"name": "delete-older", "action": {"type": "Delete"}, "condition": {"tagState": "any", "olderThan": "7d"}}
]
JSON
g artifacts repositories set-cleanup-policies "$AR_REPO" --location "$REGION" \
  --policy "$POLICY_FILE" --no-dry-run >/dev/null

echo "==> deploy cost guard"
bash "${SCRIPT_DIR}/deploy.sh"
GUARD_URL=$(g run services describe "$GUARD_SERVICE" --region "$REGION" --format 'value(status.url)')
g run services add-iam-policy-binding "$GUARD_SERVICE" --region "$REGION" \
  --member "serviceAccount:${INVOKER_SA}" --role roles/run.invoker >/dev/null

echo "==> budget notifications: Pub/Sub ${TOPIC} -> ${GUARD_URL}/budget"
g pubsub topics describe "$TOPIC" >/dev/null 2>&1 || g pubsub topics create "$TOPIC"
g pubsub topics add-iam-policy-binding "$TOPIC" \
  --member serviceAccount:billing-budget-alert@system.gserviceaccount.com --role roles/pubsub.publisher >/dev/null
PUSH_FLAGS=(--push-endpoint "${GUARD_URL}/budget" --push-auth-service-account "$INVOKER_SA"
  --push-auth-token-audience "$GUARD_URL")
if g pubsub subscriptions describe "$SUBSCRIPTION" >/dev/null 2>&1; then
  g pubsub subscriptions update "$SUBSCRIPTION" "${PUSH_FLAGS[@]}"
else
  g pubsub subscriptions create "$SUBSCRIPTION" --topic "$TOPIC" --ack-deadline 60 "${PUSH_FLAGS[@]}"
fi

echo "==> budget ${BUDGET_AMOUNT} (${BUDGET_NAME})"
# The Budget API needs a quota project when called with user credentials.
if ! gcloud billing budgets list --billing-project "$PROJECT_ID" --billing-account "$BILLING_ACCOUNT" \
  --format 'value(displayName)' | grep -Fxq "$BUDGET_NAME"; then
  gcloud billing budgets create --billing-project "$PROJECT_ID" --billing-account "$BILLING_ACCOUNT" \
    --display-name "$BUDGET_NAME" \
    --budget-amount "$BUDGET_AMOUNT" --filter-projects "projects/${PROJECT_ID}" \
    --threshold-rule percent=0.5 --threshold-rule percent=0.9 --threshold-rule percent=1.0 \
    --notifications-rule-pubsub-topic "projects/${PROJECT_ID}/topics/${TOPIC}"
fi

echo "==> scheduler ${SCHEDULER_JOB} (every 5 minutes)"
JOB_FLAGS=(--location "$REGION" --schedule "*/5 * * * *" --time-zone Etc/UTC
  --uri "${GUARD_URL}/check" --http-method POST
  --oidc-service-account-email "$INVOKER_SA" --oidc-token-audience "$GUARD_URL"
  --attempt-deadline 120s --max-retry-attempts 0)
if g scheduler jobs describe "$SCHEDULER_JOB" --location "$REGION" >/dev/null 2>&1; then
  g scheduler jobs update http "$SCHEDULER_JOB" "${JOB_FLAGS[@]}"
else
  g scheduler jobs create http "$SCHEDULER_JOB" "${JOB_FLAGS[@]}"
fi

echo "==> alert e-mail to ${ALERT_EMAIL}"
TOKEN=$(gcloud auth print-access-token)
CHANNELS_URL="https://monitoring.googleapis.com/v3/projects/${PROJECT_ID}/notificationChannels"
CHANNEL=$(curl -sf -H "Authorization: Bearer ${TOKEN}" "$CHANNELS_URL" |
  python3 -c 'import json,sys; email=sys.argv[1]; print(next((c["name"] for c in json.load(sys.stdin).get("notificationChannels", []) if c.get("labels",{}).get("email_address")==email), ""))' "$ALERT_EMAIL")
if [ -z "$CHANNEL" ]; then
  CHANNEL=$(curl -sf -X POST -H "Authorization: Bearer ${TOKEN}" -H "Content-Type: application/json" \
    -d "{\"type\":\"email\",\"displayName\":\"costguard\",\"labels\":{\"email_address\":\"${ALERT_EMAIL}\"}}" \
    "$CHANNELS_URL" | python3 -c 'import json,sys; print(json.load(sys.stdin)["name"])')
fi
if [ -z "$(g monitoring policies list --filter "displayName=\"${ALERT_POLICY_NAME}\"" --format 'value(name)')" ]; then
  cat >"$POLICY_FILE" <<JSON
{
  "displayName": "${ALERT_POLICY_NAME}",
  "combiner": "OR",
  "conditions": [{
    "displayName": "cost guard ERROR log",
    "conditionMatchedLog": {
      "filter": "resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"${GUARD_SERVICE}\" AND severity>=ERROR"
    }
  }],
  "alertStrategy": {"notificationRateLimit": {"period": "300s"}, "autoClose": "1800s"},
  "documentation": {
    "content": "Cost guard stopped ${API_SERVICE} or failed to check usage. See DesignDocument/12_Operation.md 12.12.6.",
    "mimeType": "text/markdown"
  },
  "notificationChannels": ["${CHANNEL}"]
}
JSON
  g monitoring policies create --policy-from-file "$POLICY_FILE" >/dev/null
fi

echo "==> done. Trigger one check now with:"
echo "    gcloud scheduler jobs run ${SCHEDULER_JOB} --location ${REGION} --project ${PROJECT_ID}"
