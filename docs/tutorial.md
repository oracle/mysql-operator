# Tutorial

This guide provides a quick-start guide for users of the Oracle MySQL Operator.

## Prerequisites

* A Kubernetes v1.8.0+ cluster.
* The mysql-operator Git repository checked out locally.
* [Helm](https://github.com/kubernetes/helm) installed and configured in your cluster.

### Configuring Helm and Tiller

Before deploying the mysql-operator, you must ensure [Tiller](https://github.com/kubernetes/helm)
is installed in your cluster. Tiller is the server side component to Helm.

Your cluster administrator may have already setup and configured Helm for you,
in which case you can skip this step.

Full documentation on installing Helm can be found in the [Installing helm docs](https://github.com/kubernetes/helm/blob/master/docs/install.md).

If your cluster has RBAC (Role Based Access Control) enabled, you will need to
take special care when deploying Tiller, to ensure Tiller has permission to
create resources as a cluster administrator. More information on deploying Helm
with RBAC can be found in the [Helm RBAC docs](https://github.com/kubernetes/helm/blob/master/docs/rbac.md).

## Installation

### Create a namespace

First create a namespace for the mysql-operator. By default this is
`mysql-operator` unless you specify `--set operator.namespace=` when installing
the mysql-operator Helm chart.

```console
$ kubectl create ns mysql-operator
```

### Installing the Chart

The helm chart for the operator is [included in this Git repository](../mysql-operator),
run the following in the root of the checked out `mysql-operator` repository.

To install the chart in a cluster without RBAC with the release name `mysql-operator`:

```console
$ helm install \
    --name mysql-operator \
    mysql-operator
```

If your cluster does not use RBAC (Role Based Access Control), you will need to
disable creation of RBAC resources by adding `--set rbac.enabled=false` to your
`helm install` command above.

The above command deploys the MySQL Operator on the Kubernetes cluster in the
default configuration. The [configuration](#configuration) section lists the
parameters that can be configured during installation.

> **Tip**: List all releases using `helm list`

### Uninstalling the Chart

To uninstall/delete the `mysql-operator` deployment:

```console
$ helm delete mysql-operator
```

### Configuration

The following tables lists the configurable parameters of the MySQL-operator
chart and their default values.

Parameter | Description | Default
--------- | ----------- | -------
`rbac.enabled` | If true, enables RBAC | `true`
`operator.namespace` | Controls the namespace in which the operator is deployed | `mysql-operator`
`operator.global` | Controls whether the `mysql-operator` is installed in cluster-wide mode or in a single namespace | `true`
`image.tag` | The version of the mysql-operator to install | `0.2.0`

## Create a simple MySQL cluster

The first time you create a MySQL Cluster in a namespace (other than in the
namespace into which you installed the mysql-operator) you need to create the
`mysql-agent` ServiceAccount and RoleBinding in that namespace:

```console
$ cat <<EOF | kubectl create -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: mysql-agent
  namespace: my-namespace
---
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: mysql-agent
  namespace: my-namespace
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: mysql-agent
subjects:
- kind: ServiceAccount
  name: mysql-agent
  namespace: my-namespace
EOF
```

Now let's create a new MySQL cluster. Create a `cluster.yaml` file with the following contents:

```yaml
apiVersion: mysql.oracle.com/v1alpha1
kind: Cluster
metadata:
  name: my-app-db
  namespace: my-namespace
```

And create it with **kubectl**

```console
$ kubectl apply -f cluster.yaml
mysqlcluster "my-app-db" created
```

You should now have a cluster in the default namespace

```console
$ kubectl -n my-namespace get mysqlclusters
NAME      KIND
myappdb   Cluster.v1alpha1.mysql.oracle.com
```

To find out how to create larger clusters, and configure storage see [Clusters](user/clusters.md#clusters).

#### Verify that you can connect to MySQL

The first thing you need to do is fetch the MySQL root password which is
auto-generated for us by default and stored in a Secret named `<dbname>-root-password`

```console
$ kubectl -n my-namespace get secret my-app-db-root-password -o jsonpath="{.data.password}" | base64 --decode
ETdmMKh2UuDq9m7y
```

You can use a MySQL client container to verify that you can connect to MySQL
from within the Kubernetes cluster.

```console
$ kubectl run mysql-client --image=mysql:5.7 -it --rm --restart=Never \
    -- mysql -h my-app-db -uroot -pETdmMKh2UuDq9m7y -e 'SELECT 1'
Waiting for pod default/mysql-client to be running, status is Pending, pod ready: false
mysql: [Warning] Using a password on the command line interface can be insecure.
+---+
| 1 |
+---+
| 1 |
+---+
```
