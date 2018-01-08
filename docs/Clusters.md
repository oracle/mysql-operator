# Clusters

MySQL cluster examples.

### Create a cluster with 3 replicas

The following example will create a MySQL Cluster with 3 replicas, one primary and 2 secondaries:

```yaml
apiVersion: "mysql.oracle.com/v1"
kind: MySQLCluster
metadata:
  name: mysql-test-cluster
spec:
  replicas: 3
```

### Create a cluster with a custom "MYSQL_ROOT_PASSWORD"

Create your own secret with a password field

```
$ kubectl create secret generic mysql-root-user-secret --from-literal=password=foobar
```

Create your cluster and reference it

```yaml
apiVersion: "mysql.oracle.com/v1"
kind: MySQLCluster
metadata:
  name: example-mysql-cluster-custom-secret
spec:
  replicas: 1
  secretRef:
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
apiVersion: "mysql.oracle.com/v1"
kind: MySQLCluster
metadata:
  name: example-mysql-cluster-with-volume
spec:
  replicas: 1
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
apiVersion: "mysql.oracle.com/v1"
kind: MySQLCluster
metadata:
  name: example-mysql-cluster-with-volume
spec:
  replicas: 1
  secretRef:
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
