# Restore

The MySQL Operator supports the notion of restoring a cluster from an existing backup image.

## On-demand restores

You can request a restore from a previous backup at any time by submitting a MySQLrestore CRD to the
operator. The backupRef is the name of the backup that you wish to restore.

```yaml
apiVersion: "mysql.oracle.com/v1"
kind: MySQLRestore
metadata:
  name: example-restore
spec:
  clusterRef:
    name: mycluster
  backupRef:
    name: mysql-example-backup
```
