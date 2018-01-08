#!/usr/bin/env bash
#
# Gets an interactive mysql prompt on the given instance.

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
    -c 'mysql -uroot -p$MYSQL_ROOT_PASSWORD'
