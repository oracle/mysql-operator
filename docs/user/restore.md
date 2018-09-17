# Restore

The MySQL Operator supports the notion of restoring a cluster from an existing backup image.

## On-demand restores

You can request a restore from a previous backup at any time by submitting a
Restore custom resource to the operator. The backup is the name of the
backup that you wish to restore, and the cluster is the name of the
destination cluster of the restore operation.

```yaml
apiVersion: mysql.oracle.com/v1alpha1
kind: Restore
metadata:
  name: example-restore
spec:
  cluster:
    name: mycluster
  backup:
    name: mysql-backup
```
