#!/bin/bash -e
#
# Updates the generated client set for the MySQL Operator.
#
# NOTE: Requires coreutils and gnu sed

ABS_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/$(basename "${BASH_SOURCE[0]}")"
OPERATOR_ROOT=$(realpath $(dirname ${ABS_PATH})/..)
BIN=${OPERATOR_ROOT}/bin

mkdir -p ${BIN}
go build -o ${BIN}/client-gen ./vendor/k8s.io/code-generator/cmd/client-gen

OUTPUT_BASE=""
if [[ -z "${GOPATH}" ]]; then
  OUTPUT_BASE="${HOME}/go/src"
else
  OUTPUT_BASE="${GOPATH}/src"
fi

verify=""
for i in "$@"; do
  if [[ $i == "--verify-only" ]]; then
    verify=1
    break
  fi
done

if [[ -z ${verify} ]]; then
  find ${OPERATOR_ROOT}/pkg/generated/clientset \
    \( \
      -name '*.go' -and \
      \( \
        ! -name '*_expansion.go' \
        -or \
        -name generated_expansion.go \
      \) \
    \) -exec rm {} \;
fi

${BIN}/client-gen \
  --go-header-file /dev/null \
  --output-base ${OUTPUT_BASE} \
  --input-base github.com/oracle/mysql-operator/pkg/api \
  --clientset-path github.com/oracle/mysql-operator/pkg/generated \
  --input mysqlop/v1 \
  --clientset-name clientset \
  $@
