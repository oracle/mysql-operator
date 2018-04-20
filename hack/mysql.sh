#!/usr/bin/env bash
#
# Gets an interactive mysql prompt on the given instance.

if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <namespace/podname>"
    exit 1
fi

NAMESPACE=${1%/*}
POD=${1#*/}
CLUSTER_NAME=${POD%-*}  # statefulset and service name
HOST="${POD}.${CLUSTER_NAME}"

kubectl exec \
    -n ${NAMESPACE} \
    -it \
    -c mysql-agent \
    ${POD} -- /bin/sh \
    -c "mysql -uroot -p\$MYSQL_ROOT_PASSWORD -h ${HOST}"
