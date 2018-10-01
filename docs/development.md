# Development

## Prerequisites

* A Kubernetes Cluster running on Kubernetes 1.7.0+.
* The mysql-operator repository checked out locally.

## Build the project and push the Docker image to a registry

The following will delete the existing built binaries, build
the project and the new binaries, create the agent and operator containers with
those binaries inside and then push them to the destination registry.

```bash
$ make push
```

The resulting tag for the container image will be named as the agent version
in the format of `$USER-TIMESTAMP`. This will need to be remembered as this is
needed for a latter step or can be exported as the `$MYSQL_AGENT_VERSION`
envrionment variable.

```bash
$ export MYSQL_AGENT_VERSION=$(cat dist/version.txt)
```

## Create a namespace

Create the namespace that the operator will reside in. By default this is
`mysql-operator` however for development this must match the `$USER` environment
variable.


```bash
$ kubectl create ns $USER
```

## Install Custom Resource Definitions, ServiceAccounts, ClusterRoles, and ClusterRoleBindings

The following will install the required Custom Resource Definitions,
ServiceAccounts, ClusterRoles, and ClusterRoleBindings for the operator to
function.

```bash
$ kubectl -n $USER apply -f contrib/manifests/custom-resource-definitions.yaml 
```
```bash
$ sed -e "s/<NAMESPACE>/${USER}/g" contrib/manifests/rbac.yaml | kubectl -n $USER apply -f -
```
```bash
$ sed -e "s/<NAMESPACE>/${USER}/g" contrib/manifests/role-binding-template.yaml | kubectl -n $USER apply -f -
```

### Run the MySQL Operator

The following will allow you to run the MySQL Operator out of cluster for
development purposes.

```bash
$ make run-dev
```

## Creating an InnoDB cluster

For the purpose of this document, we will create a cluster with 3 members with
the example yaml.

```bash
$ kubectl apply -n $USER -f examples/cluster/cluster-with-3-members.yaml
```
