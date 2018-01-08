#!/usr/bin/env bash
#
# Gets an interactive Python based mysqlsh on the given instance.

if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <namespace/podname>"
    exit 1
fi

NAMESPACE=${1%/*}
POD=${1#*/}

kubectl exec \
    -n ${NAMESPACE} \
    -it \
    -c mysql-agent \
    ${POD} -- /bin/sh \
    -c 'mysqlsh --uri "root:$MYSQL_ROOT_PASSWORD@localhost:3306" --py'
