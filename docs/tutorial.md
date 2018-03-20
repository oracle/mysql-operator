# Tutorial

This guide provides a quick-start guide for users of the Oracle MySQL Operator.

## Prerequisites

* Kubernetes
* Password for ODX docker registry
* The mysql-operator repo checked out locally

#### Create a namespace and Docker secret for the registry username/password

First create the namespace that the operator will reside in:

```
kubectl create ns mysql-operator
```

## Deploy a version of the MySQL Operator using Helm

The MySQL Operator is installed into your cluster via a Helm chart

### Ensure you have Helm installed and working.

Install the helm tool locally by following [these instructions](https://docs.helm.sh/using_helm/#installing-helm)

If you  have not already installed tiller to your cluster set it up with:

```bash
helm init
```

Verify helm is installed :
```bash
helm version

Client: &version.Version{SemVer:"v2.5.0", GitCommit:"012cb0ac1a1b2f888144ef5a67b8dab6c2d45be6", GitTreeState:"clean"}
Server: &version.Version{SemVer:"v2.5.0", GitCommit:"012cb0ac1a1b2f888144ef5a67b8dab6c2d45be6", GitTreeState:"clean"}
```

### Installing the Chart

The helm chart for the  operator is [included in this git repo](../mysql-operator), run the following in the root of the checked out `mysql-operator` repo.

To install the chart with the release name `my-release`:

```console
$ helm install --name my-release mysql-operator
```

The command deploys the MySQL Operator on the Kubernetes cluster in the default configuration. The [configuration](#configuration) section lists the parameters that can be configured during installation.

> **Tip**: List all releases using `helm list`

### Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```console
$ helm delete my-release
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

### Configuration

The following tables lists the configurable parameters of the MySQL-operator chart and their default values.

Parameter | Description | Default
--------- | ----------- | -------
`rbac.enabled` | If true, enables RBAC | `false`
`operator.namespace` | Controls the namespace in which the operator is deployed | `mysql-operator`

## Create a simple MySQL cluster

Now let's create a new MySQL cluster. Create a cluster.yaml file with the following contents

```yaml
apiVersion: mysql.oracle.com/v1
kind: MySQLCluster
metadata:
  name: myappdb
```

And create it with **kubectl**

```
$ kubectl apply -f cluster.yaml
mysqlcluster "myappdb" created
```

You should now have a cluster in the default namespace

```
$ kubectl get mysqlclusters
NAME      KIND
myappdb   MySQLCluster.v1.mysql.oracle.com
```

To find out how to create larger clusters, and configure storage see [Clusters](Clusters.md).

#### Verify that you can connect to MySQL

The first thing you need to do is fetch the MySQL root password which is auto-generated for us by default and stored ia secret named `<dbname>-root-password`

```
$ kubectl get secret myappdb-root-password -o jsonpath="{.data.password}" | base64 -D
ETdmMKh2UuDq9m7y
```

You can use a MySQL client container to verify  that you can connect to MySQL inside the Kubernetes cluster.

```
$ kubectl run mysql-client --image=mysql:5.7 -i -t --rm --restart=Never \
    -- mysql -h myappdb -uroot -pETdmMKh2UuDq9m7y -e 'SELECT 1'
Waiting for pod default/mysql-client to be running, status is Pending, pod ready: false
mysql: [Warning] Using a password on the command line interface can be insecure.
+---+
| 1 |
+---+
| 1 |
+---+
```

You can then execute any further commands via 'kubectl exec' against the 'mysql'
container:

```
$ kubectl exec -it -c mysql  \
    -- mysql -h myappdb -uroot -pETdmMKh2UuDq9m7y -e 'SELECT 1'
+---+
| 1 |
+---+
| 1 |
+---+
```

# Troubleshooting

## cannot list configmaps in the namspace "kube-system"


Note: If `helm list` gives the following error

```console
Error: User "system:serviceaccount:kube-system:default" cannot list configmaps in the namespace "kube-system". (get configmaps)
```

then it could be because the cluster you are targeting has role-based-authentication (RBAC) enabled. To fix this, issue the following commands:

```console
kubectl create serviceaccount --namespace kube-system tiller
kubectl create clusterrolebinding \
      tiller-cluster-rule \
      --clusterrole=cluster-admin \
      --serviceaccount=kube-system:tiller
kubectl patch deploy --namespace kube-system \
          tiller-deploy \
          -p '{"spec":{"template":{"spec":{"serviceAccount":"tiller"}}}}'
```

