# E2E Tests

## Create a cluster

Test create, delete, defaults, validation, volumes, custom secret for MySQL root password

#### Creating a MySQL cluster

As a developer,
Given I have the a valid yaml spec
When I submit the mysqlcluster spec using kubectl create -f
Then a single instance MySQL EE cluster is created

#### Deleting a MySQL cluster

As a developer,
Given I have created a mysqlcluster
When I run kubectl delete -f
Then the cluster is deleted

#### Creating a MySQL cluster with defaults

As a developer,
Given I have the a valid yaml spec
And no spec is provided
When I submit the mysqlcluster spec using kubectl create -f
Then a single instance MySQL EE cluster is created using the default MySQL EE version

```yaml
apiVersion: "mysql.oracle.com/v1"
kind: MySQLCluster
metadata:
  name: example-mysql-cluster-with-defaults
```

#### Creating a MySQL cluster with a volume claim template

Create a Kubernetes PV. Reference that PV using the mysqlcluster spec

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  labels:
    type: local
  name: data-volume
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
  version: 5.7.19-1.1.0
  replicas: 2
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

#### Creating a MySQL cluster with a custom MySQL root password secret

Create a custom secret

```
kubectl create secret generic mysql-root-user-secret --from-literal=password=foobar
```

Reference that secret in the mysqlcluster spec

```yaml
apiVersion: "mysql.oracle.com/v1"
kind: MySQLCluster
metadata:
  name: example-mysql-cluster-with-custom-secret
spec:
  version: 5.7.19-1.1.0
  replicas: 2
  secretRef:
    name: mysql-root-user-secret
```
