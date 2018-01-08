#!/usr/bin/env bash
#
# Outputs the cluster status using mysqlsh.

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
    -c 'mysqlsh --uri "root:$MYSQL_ROOT_PASSWORD@localhost:3306" \
        --py \
        -e "import json; print json.dumps(json.loads(str(dba.get_cluster().status())), indent=4, sort_keys=True)"'
