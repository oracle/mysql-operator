#!/usr/bin/env bash
#
# Gets an interactive Python based mysqlsh on the given instance.

if [ "$#" -lt 1 ]; then
    echo "Usage: $0 <namespace/podname> [args]"
    echo "example: $0 default/my-cluster-0 --py"
    exit 1
fi

NAMESPACE=${1%/*}
POD=${1#*/}
CLUSTER_NAME=${POD%-*}  # statefulset and service name
URI="root:\$MYSQL_ROOT_PASSWORD@${POD}.${CLUSTER_NAME}:3306"

kubectl exec \
    -n ${NAMESPACE} \
    -it \
    -c mysql-agent \
    ${POD} -- /bin/sh \
    -c "PS1='\u@\h:\w\$ ' mysqlsh --no-wizard --uri ${URI} ${@:2}"
