#!/bin/bash
# DesignDocument/7_BackendDesign.md 7.2.1 / 14_CICDDesign.md 14.5.2
# Verifies the dependency rule: Domain must not import frameworks, DB/network
# SDKs or net/http; Usecase/Port must not import Gin, pgx, Firebase or Infra.
# Run from the backend directory.

set -euo pipefail

VIOLATIONS=0

check() {
  local dir="$1"; shift
  for pkg in "$@"; do
    if grep -rn --include='*.go' "\"${pkg}" "$dir" 2>/dev/null; then
      echo "ERROR: ${dir} imports forbidden package: ${pkg}"
      VIOLATIONS=$((VIOLATIONS + 1))
    fi
  done
}

DOMAIN_FORBIDDEN=(
  "github.com/gin-gonic"
  "github.com/jackc/pgx"
  "firebase.google.com"
  "google.golang.org/api"
  "net/http"
  "database/sql"
  "github.com/interviewhack/backend/internal/usecase"
  "github.com/interviewhack/backend/internal/adapter"
  "github.com/interviewhack/backend/internal/infra"
  "github.com/interviewhack/backend/internal/port"
)

INNER_FORBIDDEN=(
  "github.com/gin-gonic"
  "github.com/jackc/pgx"
  "firebase.google.com"
  "net/http"
  "database/sql"
  "github.com/interviewhack/backend/internal/adapter"
  "github.com/interviewhack/backend/internal/infra"
)

check internal/domain "${DOMAIN_FORBIDDEN[@]}"
check internal/usecase "${INNER_FORBIDDEN[@]}"
check internal/port "${INNER_FORBIDDEN[@]}"

if [ $VIOLATIONS -gt 0 ]; then
  echo "Architecture check FAILED: ${VIOLATIONS} violation(s) found"
  exit 1
fi

echo "Architecture check PASSED: dependency rule holds for domain / usecase / port"
