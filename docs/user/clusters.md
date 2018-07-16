# Clusters

MySQL cluster examples.

### Create a cluster with 3 members

The following example will create a MySQL Cluster with 3 members, one primary and 2 secondaries:

```yaml
apiVersion: mysql.oracle.com/v1alpha1
kind: Cluster
metadata:
  name: mysql-test-cluster
spec:
  members: 3
```

### Create a cluster with 3 members in multi-master mode

The following example will create a MySQL Cluster with 3 primary (read/write) members:

```yaml
apiVersion: mysql.oracle.com/v1alpha1
kind: Cluster
metadata:
  name: mysql-multimaster-cluster
spec:
  multiMaster: true
  members: 3
```

### Create a cluster with a custom "MYSQL_ROOT_PASSWORD"

Create your own secret with a password field

```
$ kubectl create secret generic mysql-root-user-secret --from-literal=password=foobar
```

Create your cluster and reference it

```yaml
apiVersion: mysql.oracle.com/v1alpha1
kind: Cluster
metadata:
  name: mysql-cluster-custom-secret
spec:
  members: 1
  rootPasswordSecret:
    name: mysql-root-user-secret
```

### Create a cluster with a persistent volume

The following example will create a MySQL Cluster with a persistent local volume.

```yaml
---
apiVersion: v1
kind: PersistentVolume
metadata:
  labels:
    type: local
  name: mysql-local-volume
spec:
  accessModes:
  - ReadWriteMany
  capacity:
    storage: 10Gi
  hostPath:
    path: /tmp/data
  persistentVolumeReclaimPolicy: Recycle
  storageClassName: manual
---
apiVersion: mysql.oracle.com/v1alpha1
kind: Cluster
metadata:
  name: mysql-cluster-with-volume
spec:
  members: 1
  volumeClaimTemplate:
    metadata:
      name: data
    spec:
      storageClassName: manual
      accessModes:
        - ReadWriteMany
      resources:
        requests:
          storage: 1Gi
```

### Create a cluster with a persistent data volume and a persistent volume to use for backups/restore

The following example will create a MySQL Cluster with a persistent local data volume
and a persistent local backup/restore volume.

```yaml
---
apiVersion: v1
kind: PersistentVolume
metadata:
  labels:
    type: local
  name: mysql-local-volume
spec:
  accessModes:
  - ReadWriteMany
  capacity:
    storage: 10Gi
  hostPath:
    path: /tmp/data1
  persistentVolumeReclaimPolicy: Recycle
  storageClassName: manual
---
apiVersion: v1
kind: PersistentVolume
metadata:
  labels:
    type: local
  name: mysql-local-backup-volume
spec:
  accessModes:
  - ReadWriteMany
  capacity:
    storage: 10Gi
  hostPath:
    path: /tmp/data2
  persistentVolumeReclaimPolicy: Recycle
  storageClassName: manual
---
apiVersion: mysql.oracle.com/v1alpha1
kind: Cluster
metadata:
  name: mysql-cluster-with-volume
spec:
  members: 1
  rootPasswordSecret:
    name: mysql-root-user-secret
  volumeClaimTemplate:
    metadata:
      name: data
    spec:
      storageClassName: manual
      accessModes:
        - ReadWriteMany
      resources:
        requests:
          storage: 1Gi
  backupVolumeClaimTemplate:
    metadata:
      name: backup-data
    spec:
      storageClassName: manual
      accessModes:
        - ReadWriteMany
      resources:
        requests:
          storage: 1Gi
```

### Create a cluster with custom my.cnf configuration file

By default, the MySQL Operator starts a cluster with an opinionated set of defaults.

However, you may wish to configure some aspects of your cluster through a my.cnf config file.
This can be achieved by creating a config map and referencing it as part of your cluster spec.

#### Create the config map

First we create a config map containing the configuration file we want to apply to our cluster.

```
kubectl create configmap mycnf --from-file=examples/my.cnf
```

#### Reference it in the cluster spec

Now we can reference our config map in our cluster spec definition. For example:

```yaml
apiVersion: mysql.oracle.com/v1alpha1
kind: Cluster
metadata:
  name: mysql-cluster-with-config
  members: 3
  config:
    name: mycnf
```

### Create a cluster with custom server_id values

By default, the MySQL Operator starts a cluster with `--server_id` set to `1000` and increments it by one for each new cluster member. You can change this behavior by setting the `baseServerId` field on your Cluster. `baseServerId` value can be set to anything in the range from `1` to `4294967286`. `0` is also accepted, but then the default value of `1000` will be used.

The following example will create a MySQL Cluster with following `server_id`'s: 42,43,44
```yaml
apiVersion: mysql.oracle.com/v1alpha1
kind: Cluster
metadata:
  name: mysql-cluster-with-custom-serverid
  members: 3
  baseServerId: 42
```
