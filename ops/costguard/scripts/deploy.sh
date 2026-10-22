#!/usr/bin/env bash
# DesignDocument/14_CICDDesign.md 14.6.4 コストガードの手動デプロイ / 14.8.3 環境変数
# 使い方（リポジトリのどこからでも）：bash ops/costguard/scripts/deploy.sh
# 初回は setup.sh から呼ばれる。コードを変えたときは、これだけを実行すればよい。
set -euo pipefail
cd "$(dirname "$0")/.."

PROJECT_ID=${PROJECT_ID:-interviewhack-prod}
REGION=${REGION:-asia-northeast1}
AR_REPO=${AR_REPO:-interviewhack}
API_SERVICE=${API_SERVICE:-interviewhack-api}
GUARD_SERVICE=${GUARD_SERVICE:-interviewhack-costguard}
MONITORED_PROJECTS=${MONITORED_PROJECTS:-$PROJECT_ID}
# 閾値は公式の無料枠（リクエスト課金）の 90%。2026-10-22 に確認した値（README.md）。
LIMIT_REQUESTS=${LIMIT_REQUESTS:-1800000}
LIMIT_CPU_SECONDS=${LIMIT_CPU_SECONDS:-162000}
LIMIT_MEMORY_GIB_SECONDS=${LIMIT_MEMORY_GIB_SECONDS:-324000}
BUDGET_STOP_RATIO=${BUDGET_STOP_RATIO:-1.0}

RELEASE=$(git rev-parse HEAD)
if [ -n "$(git status --porcelain -- .)" ]; then
  RELEASE="${RELEASE}-dirty"
fi
IMAGE="${REGION}-docker.pkg.dev/${PROJECT_ID}/${AR_REPO}/${GUARD_SERVICE}:${RELEASE}"

echo "==> build and push ${IMAGE}"
gcloud auth configure-docker "${REGION}-docker.pkg.dev" --quiet
docker build --build-arg RELEASE="$RELEASE" -t "$IMAGE" .
docker push "$IMAGE"

# 認証必須・内部からのみ（Cloud Scheduler / Pub/Sub は同じプロジェクトの内部扱い）。
# CPU 1 未満は同時実行 1 が条件。1 回の集計は 1 秒未満なので十分（12.12.7）。
echo "==> deploy ${GUARD_SERVICE}"
gcloud run deploy "$GUARD_SERVICE" \
  --project "$PROJECT_ID" --region "$REGION" --image "$IMAGE" \
  --service-account "costguard@${PROJECT_ID}.iam.gserviceaccount.com" \
  --no-allow-unauthenticated --ingress internal \
  --cpu 0.25 --memory 256Mi --concurrency 1 --min-instances 0 --max-instances 1 --timeout 120 \
  --set-env-vars "^@^TARGET_PROJECT=${PROJECT_ID}@TARGET_REGION=${REGION}@TARGET_SERVICE=${API_SERVICE}@MONITORED_PROJECTS=${MONITORED_PROJECTS}@LIMIT_REQUESTS=${LIMIT_REQUESTS}@LIMIT_CPU_SECONDS=${LIMIT_CPU_SECONDS}@LIMIT_MEMORY_GIB_SECONDS=${LIMIT_MEMORY_GIB_SECONDS}@BUDGET_STOP_RATIO=${BUDGET_STOP_RATIO}@LOG_LEVEL=info@RELEASE=${RELEASE}" \
  --quiet

echo "==> done: $(gcloud run services describe "$GUARD_SERVICE" --project "$PROJECT_ID" --region "$REGION" --format 'value(status.url)')"
