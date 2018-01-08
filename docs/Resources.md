# Mysql Operator Custom Resource Definitions

The Oracle MySQL Operator makes use of CustomResourceDefinitions to extend the Kubernetes API, allowing for a custom interface when working with
MySQL and Kubernetes. We introduce the following CRD's as part of the MySQL Operator:

#### Resource: mysqlcluster(s)

The primary function of the mysql-operator is the ability to quickly create and delete MySQL clusters in Kubernetes.

A MySQLCluster defines how a cluster is created (number of replicas, MySQL version, secrets etc)
as well an optional backup policy for the cluster.

Once created, you can query kubernetes for mysqlcluster resources. For example

```
$ kubectl get mysqlclusters
```

#### Resource: mysqlbackup(s)

MySQLBackup CRD's hold information about backups that have been executed. You can list all backups for a cluster
and also inspect an individual backup (for example: getting information about the location of the backup in Object Storage)

```
$ kubectl get mysqlbackups
```

