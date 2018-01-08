#!/bin/bash -x

USE_GLOBAL_NAMESPACE="${USE_GLOBAL_NAMESPACE:-false}"
if [ "${USE_GLOBAL_NAMESPACE}" = true ]; then
    OPERATOR_NAMESPACE="${OPERATOR_NAMESPACE:-mysql-operator}"
    TEST_NAMESPACE="${TEST_NAMESPACE:-default}"
    REGISTER_CRD=true
else
    NEW_NAMESPACE=${NEW_NAMESPACE:-"e2etest-$(cat /dev/urandom | tr -dc 'a-z0-9' | fold -w 10 | head -n 1)"}
    OPERATOR_NAMESPACE=$(echo ${NEW_NAMESPACE}-${E2E_TEST_RUN} | tr "[:upper:]" "[:lower:]")
    TEST_NAMESPACE=$(echo ${NEW_NAMESPACE}-${E2E_TEST_RUN} | tr "[:upper:]" "[:lower:]")
    REGISTER_CRD=false
fi

echo "OPERATOR_NAMESPACE=${OPERATOR_NAMESPACE}"
echo "TEST_NAMESPACE=${TEST_NAMESPACE}"

export KUBECONFIG_PATH=${KUBECONFIG_PATH:-"/tmp/kubeconf-$(date +'%d%m%y%H%M%S%N').conf"}

function log() {
    echo "$@";
}

function log_err() {
    echo "$@" 1>&2;
}

cmd_or_exit()
{
    "$@"
    ret=$?
    if [[ $ret -eq 0 ]]
    then
        echo "Successfully ran [ $@ ]"
    else
        echo "Error: Command [ $@ ] returned $ret"
        exit $ret
    fi
}

# make sure provided exit status equals 0
function testExitStatus {
  if [ 0 -ne $1 ]
   then
      log_err "FAILED:$2"
      exit 1
  fi
}

# ------------------------------------------------------------------------------
# general functions

function log() {
    echo "$@";
}

function log_err() {
    echo "$@" 1>&2;
}

function fail() {
    # if sourced return; otherwise exit.
    #[[ "${BASH_SOURCE[0]}" != "${0}" ]]  && echo "SOURCED" || echo "RUN"
    [[ "${BASH_SOURCE[0]}" != "${0}" ]]  && return 1 || exit;
}

function __assert_var_set() {
    local env_var=$1
    local env=$(printenv ${env_var})
    if [[ -z "${env}" ]]; then
        log_err "${env_var} - unset"
	fail
    fi
}

function check_env() {
    # So config should be set
    if [[ -z "${KUBECONFIG}" ]]; then
        log_err "KUBECONFIG not set"
	    fail
    fi

    if [[ -z "${S3_ACCESS_KEY}" ]]; then
        log_err "$S3_ACCESS_KEY not set"
        fail
    fi

    if [[ -z "${S3_SECRET_KEY}" ]]; then
        log_err "$S3_SECRET_KEY not set"
        fail
    fi

    if [[ -z "${CLUSTER_INSTANCE_SSH_KEY}" ]]; then
        log_err "CLUSTER_INSTANCE_SSH_KEY not set"
	    fail
    fi

    __assert_var_set MYSQL_ROOT_PASSWORD
    __assert_var_set MYSQL_OPERATOR_VERSION
}

# ------------------------------------------------------------------------------
# k8s init environment functions
function log_cluster_info() {
    kubectl --kubeconfig=${KUBECONFIG} \
        cluster-info | grep "Kubernetes master"
    echo
    kubectl --kubeconfig=${KUBECONFIG} get nodes
    echo
}

# ------------------------------------------------------------------------------
# mysql-operator core init functions

function create_mysql_operator_namespace() {
    kubectl --kubeconfig=${KUBECONFIG} \
        create namespace ${OPERATOR_NAMESPACE}
}

function delete_mysql_operator_namespace() {
    kubectl --kubeconfig=${KUBECONFIG} \
            delete namespace ${OPERATOR_NAMESPACE}

    # Wait for actual operator to be deleted
    while true; do
	kubectl get namespace ${OPERATOR_NAMESPACE}
	if [ $? -ne 0 ]; then
	    break;
	fi
	echo "sleeping till namespace gone.."
	sleep 1
    done
}

function create_odx_docker_pull_secrets() {
    kubectl --kubeconfig=${KUBECONFIG} \
        -n ${OPERATOR_NAMESPACE} \
        create secret docker-registry odx-docker-pull-secret \
        --docker-server="wcr.io" \
        --docker-username=${DOCKER_REGISTRY_USERNAME} \
        --docker-password=${DOCKER_REGISTRY_PASSWORD} \
        --docker-email="k8s@oracle.com"

    if [[ "${TEST_NAMESPACE}" != "${OPERATOR_NAMESPACE}" ]]; then
        kubectl --kubeconfig=${KUBECONFIG} \
            -n ${TEST_NAMESPACE} \
            create secret docker-registry odx-docker-pull-secret \
            --docker-server="wcr.io" \
            --docker-username=${DOCKER_REGISTRY_USERNAME} \
            --docker-password=${DOCKER_REGISTRY_PASSWORD} \
            --docker-email="k8s@oracle.com"
    fi
}

function delete_odx_docker_pull_secrets() {
    kubectl --kubeconfig=${KUBECONFIG} \
        -n ${OPERATOR_NAMESPACE} \
        delete secret odx-docker-pull-secret
    kubectl --kubeconfig=${KUBECONFIG} \
        delete secret odx-docker-pull-secret
}

function create_mysql_root_user_secret() {
    # TODO: if you set the password to anything but this you cannot shell
    export MYSQL_ROOT_PASSWORD=mytestpass
    # if [ -z "${MYSQL_ROOT_PASSWORD}" ] ; then
    #     log "no MYSQL_ROOT_PASSWORD secret - creating..."
    #     export MYSQL_ROOT_PASSWORD=`openssl rand -base64 8`
    #     log "MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD}"
    # fi
    kubectl --kubeconfig=${KUBECONFIG} \
        -n ${OPERATOR_NAMESPACE} \
        create secret generic \
        --from-literal=password="${MYSQL_ROOT_PASSWORD}" \
        mysql-root-user-secret
}

function delete_mysql_root_user_secret() {
    kubectl --kubeconfig=${KUBECONFIG} \
        -n ${OPERATOR_NAMESPACE} \
        delete secret mysql-root-user-secret
}

function create_upload_credentials() {
    kubectl --kubeconfig=${KUBECONFIG} \
        -n ${OPERATOR_NAMESPACE} \
        create secret generic s3-upload-credentials \
        --from-literal=accessKey=${S3_ACCESS_KEY} \
        --from-literal=secretKey=${S3_SECRET_KEY} \
        --from-literal=tenancy="bristoldev"
}

function delete_upload_credentials() {
    kubectl --kubeconfig=${KUBECONFIG} \
        -n ${OPERATOR_NAMESPACE} \
        delete secret s3-upload-credentials
}

function create_mysql_operator() {
    # FIXME: Wait for helm to be ready after init here....
    cmd_or_exit helm init --skip-refresh

    if [ ${USE_GLOBAL_NAMESPACE} = true ]; then
	# Nothing as the CRD will be registerd
	# maybe movee to upgrade here?
	echo ''
    else
	# do our CRD check
	kubectl get customresourcedefinition mysqlclusters.mysql.oracle.com
	if [ $? -ne 0 ]; then
	    # We need to create a CRD and leave it around
	    # FIXME: upgrades? maybe check it this exists and if so upgrade it?
	    cmd_or_exit helm --debug install  --name crd-register --set image.tag=$MYSQL_OPERATOR_VERSION --set rbac.enabled=${USE_RBAC} mysql-operator --set operator.global=false --set operator.namespace=crd-register --set operator.register_crd=true --namespace=crd-register
	fi
    fi

    cmd_or_exit helm --debug install  --name ${OPERATOR_NAMESPACE} --set image.tag=$MYSQL_OPERATOR_VERSION --set rbac.enabled=${USE_RBAC} mysql-operator --set operator.global=${USE_GLOBAL_NAMESPACE} --set operator.namespace=${OPERATOR_NAMESPACE} --set operator.register_crd=${REGISTER_CRD}
}

function delete_mysql_operator() {
    helm init --skip-refresh
    log "deleting mysql operator..."
    helm delete --purge ${OPERATOR_NAMESPACE}
}

# ------------------------------------------------------------------------------
# k8s e2e lifecycle functions

function setup() {
    log_cluster_info
    kubectl create serviceaccount --namespace kube-system tiller || true
    kubectl create clusterrolebinding \
        tiller-cluster-rule \
        --clusterrole=cluster-admin \
        --serviceaccount=kube-system:tiller || true
    kubectl patch deploy \
        --namespace kube-system \
        tiller-deploy -p '{"spec":{"template":{"spec":{"serviceAccount":"tiller"}}}}'

    # core k8s resources
    create_mysql_operator_namespace
    create_odx_docker_pull_secrets
    create_mysql_root_user_secret
    create_upload_credentials
    # mysql operator
    create_mysql_operator
}


function run() {
    log "E2E_TEST_RUN:${E2E_TEST_RUN}"
    cmd_or_exit go test -timeout 45m -v ./test/e2e/ --kubeconfig=${KUBECONFIG} --namespace=${TEST_NAMESPACE} -run ${E2E_TEST_RUN}
}

function teardown() {
    # mysql operator
    delete_mysql_operator
    delete_upload_credentials
    delete_mysql_root_user_secret
    delete_odx_docker_pull_secrets
    delete_mysql_operator_namespace
}


# ------------------------------------------------------------------------------
USE_RBAC=${USE_RBAC:-true}

while [[ $# -gt 1 ]]; do
option_key="$1"
option_value="$2"
    case ${option_key} in
        -r|--use-rbac)
        USE_RBAC=${option_value}
        shift 2
        ;;
        *)
        break
        ;;
    esac
done

log "Use RBAC: ${USE_RBAC}"

MANIFESTS=test/e2e/manifests

log "MYSQL_OPERATOR_VERSION: ${MYSQL_OPERATOR_VERSION}"

export MYSQL_ROOT_PASSWORD="mytestpass"

# the command is the last argument
while (( "$#" )); do
    cmd=$1
    case ${cmd} in
    help)
        echo "usage: init-e2e-resources.sh [-ct|--cluster-type [k8s|oke]] [check-env|setup|run|teardown]"
        ;;
    check-env)
        check_env
        ;;
    setup)
        check_env
        setup
        ;;
    run)
        check_env
        run
        ;;
    teardown)
        check_env
        teardown
        ;;
    "")
        # permit sourcing with no args
        ;;
    *)
        log "error: unknown command: ${cmd}"
        ;;
    esac
    shift
done
