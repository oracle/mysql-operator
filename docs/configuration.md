# Customise the MySQL configuration for your cluster

By default, the MySQL Operator starts a cluster with an opinionated set of defaults.

However, you may wish to configure some aspects of your cluster through a my.cnf config file.
This can be achieved by creating a config map and referencing it as part of your cluster spec.

## Create the config map

First we create a config map containing the configuration file we want to apply to our cluster.

```
kubectl create configmap mycnf --from-file=examples/my.cnf
```

## Reference it in the cluster spec

Now we can reference our config map in our cluster spec definition. For example:

```yaml
apiVersion: "mysql.oracle.com/v1"
kind: MySQLCluster
metadata:
  name: mysql-cluster-with-config
  replicas: 3
  configRef:
    name: mycnf
```
