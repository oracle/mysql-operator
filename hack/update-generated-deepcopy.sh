#!/bin/bash -e
#
# Updates the generated deep copy methods for the MySQL Operator API.
#
# NOTE: Requires coreutils.

ABS_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/$(basename "${BASH_SOURCE[0]}")"
OPERATOR_ROOT=$(realpath $(dirname ${ABS_PATH})/..)
BIN=${OPERATOR_ROOT}/bin

mkdir -p ${BIN}
go build -o ${BIN}/deepcopy-gen ./vendor/k8s.io/code-generator/cmd/deepcopy-gen

OUTPUT_BASE=""
if [[ -z "${GOPATH}" ]]; then
  OUTPUT_BASE="${HOME}/go/src"
else
  OUTPUT_BASE="${GOPATH}/src"
fi

${BIN}/deepcopy-gen \
  -v 2 \
  --logtostderr \
  --go-header-file /dev/null \
  --output-base "${OUTPUT_BASE}" \
  --input-dirs "github.com/oracle/mysql-operator/pkg/apis/mysql/v1" \
  --output-file-base "zz_generated.deepcopy" \
  --bounding-dirs "github.com/oracle/mysql-operator/pkg/api" \
  $@
