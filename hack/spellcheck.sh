#!/usr/bin/env bash
#
# Performs a spellcheck on golang and markdown files, excluding the vendor directory

set -e

REPO_ROOT=$(git rev-parse --show-toplevel)
REPO_BIN_DIR="${REPO_ROOT}/bin"

cd "${REPO_ROOT}"
GOBIN="${REPO_BIN_DIR}" go install ./vendor/github.com/client9/misspell/cmd/misspell

find . -type f \( -name "*.go" -o -name "*.md" \) -a \( -not -path "./vendor/*" \) | \
  xargs ${REPO_BIN_DIR}/misspell -error
