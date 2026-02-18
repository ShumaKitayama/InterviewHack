#!/bin/bash
# DesignDocument/7_BackendDesign.md 7.2.1 / 14_CICDDesign.md 14.5.2
# Verifies that Domain layer does not import forbidden external packages.

set -euo pipefail

DOMAIN_DIR="internal/domain"
FORBIDDEN_IMPORTS=(
  "github.com/gin-gonic"
  "github.com/stripe"
  "github.com/sashabaranov/go-openai"
  "github.com/jackc/pgx"
  "supabase"
  "google.golang.org/api"
)

VIOLATIONS=0

for pkg in "${FORBIDDEN_IMPORTS[@]}"; do
  if grep -r "\"${pkg}" "${DOMAIN_DIR}" 2>/dev/null; then
    echo "ERROR: Domain layer imports forbidden package: ${pkg}"
    VIOLATIONS=$((VIOLATIONS + 1))
  fi
done

if [ $VIOLATIONS -gt 0 ]; then
  echo "Architecture check FAILED: ${VIOLATIONS} violation(s) found"
  exit 1
fi

echo "Architecture check PASSED: No forbidden imports in Domain layer"
