#!/bin/bash -e
#
# Updates the generated shared informers for the MySQL Operator.
#
# NOTE: Requires coreutils.

ABS_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/$(basename "${BASH_SOURCE[0]}")"
OPERATOR_ROOT=$(realpath $(dirname ${ABS_PATH})/..)
BIN=${OPERATOR_ROOT}/bin
mkdir -p ${BIN}
go build -o ${BIN}/informer-gen ./vendor/k8s.io/code-generator/cmd/informer-gen

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
  rm -rf ${OPERATOR_ROOT}/pkg/generated/informers
fi

${BIN}/informer-gen \
  --logtostderr \
  --go-header-file /dev/null \
  --output-base ${OUTPUT_BASE} \
  --input-dirs github.com/oracle/mysql-operator/pkg/apis/mysql/v1 \
  --output-package github.com/oracle/mysql-operator/pkg/generated/informers \
  --listers-package github.com/oracle/mysql-operator/pkg/generated/listers \
  --internal-clientset-package github.com/oracle/mysql-operator/pkg/generated/clientset \
  --versioned-clientset-package github.com/oracle/mysql-operator/pkg/generated/clientset \
  $@
