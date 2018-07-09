# Backups

## Introduction

The MySQL Operator allows for on-demand and scheduled backups to be created.
On-demand backups can be created by submitting a Backup custom resource.
Scheduled backups can be created by submitting a BackupSchedule custom resource.

Whilst we plan to offer different options for backups, we currently only support
the mysqldump tool, and storage in S3 compatible object storage providers.

### Types of backup

We only currently support full snapshot backups, although we plan to support
incremental backups in the future.

### Credentials

All backups require object storage credentials in order for the mysql-agent to
persist backups.

Create an oci S3 credential as explained here: [Working with Amazon S3 Compatibility API Keys](https://docs.us-phoenix-1.oraclecloud.com/Content/Identity/Tasks/managingcredentials.htm#To4).

An example for Oracle Cloud Infrastructure S3 is given below.

```yaml
accessKey: accessKey
secretKey: secretKey
```

Now create a secret with the contents of the above yaml file.

```bash
$ kubectl create secret generic s3-credentials --from-literal=accessKey=${S3_ACCESS_KEY} --from-literal=secretKey=${S3_SECRET_KEY}
```

## On-demand backups

You can request a backup at any time by submitting a Backup custom resource to the
operator. The credentialsSecret is the name of a secret that contains your Object
Storage credentials. Note: The databases field is mandatory.

```yaml
apiVersion: mysql.oracle.com/v1alpha1
kind: Backup
metadata:
  name: mysql-backup
spec:
  executor:
    mysqldump:
      databases:
        - name: test
        - name: wordpress
  storageProvider:
    s3:
      endpoint: ocitenancy.compat.objectstorage.ociregion.oraclecloud.com
      region:   ociregion
      bucket:   mybucket
      forcePathStyle: true
      credentialsSecret:
          name: s3-credentials
  cluster:
    name: mysql-cluster
```

### On-demand backups - executor configuration

A backup spec requires an 'executor' to support the backup and restore of
database content.

Currently, the 'mysqldump' utility is provided, although further executors may
be added in the future.

You should additionally configure the list of databases to include in the
backup.

### On-demand backups - storage configuration

A backup spec requires a 'storage' mechanism to save the backed up
content of a database.

Currently, 'S3' based object storage is provided, although further providers
may be added in the future.

#### On-demand backups - OCI S3 storage configuration

When configuring an S3 endpoint you should ensure that it is correct for your
backing provider and that you have pre-created the desired bucket.

For example, An Oracle OCI backend for the tenancy 'mytenancy', region
'us-phoenix-1', and bucket 'mybucket', should have the storage element
configured as follows:

```yaml
  ...
   s3:
     endpoint: mytenancy.compat.objectstorage.us-phoenix-1.oraclecloud.com
     region:   us-phoenix-1
     bucket:   mybucket
  ...
```

The bucket should also be valid for the secret credentials specified previously.

#### On-demand backups - Amazon S3 storage configuration

An AWS storage endpoint can also be configured. For example:

```yaml
  ...
  s3:
    endpoint: s3.eu-west-2.amazonaws.com
    region:   eu-west-2
    bucket:   mybucket
  ...
```

Remember to also configure the valid S3 credentials secret as outlined above.

#### On-demand backups - Google GCE storage configuration

To use a GCE storage bucket, you need to ensure S3 compatibility has been enabled:

1. Log into the GCE console and navigate to 'storage'.
2. Enable S3 compatibility in your GCE storage config.
3. Generate a new S3 'secretKey' and 'accessKey'.

A GCE storage endpoints can then be configured as follows:

```yaml
  ...
  s3:
    endpoint: storage.googleapis.com
    region:   europe-west1
    bucket:   mybucket
  ...
```

Remember to also configure the S3 credentials secret as outlined above.

## Scheduled backups

You can request a backup to be performed on a given schedule by submitting a
BackupSchedule custom resource to the operator. This will create Backup
resources based on the given cron format string. The credentialsSecret is the
name of a Secret that contains your Object Storage credentials. Note: The
databases field is mandatory. For example, the following will create a backup of
the employees database every 30 minutes:

```yaml
apiVersion: mysql.oracle.com/v1alpha1
kind: BackupSchedule
metadata:
  name: mysql-backup-schedule
spec:
  schedule: '*/30 * * * *'
  backupTemplate:
    executor:
      provider: mysqldump
      databases:
        - test
    storageProvider:
      s3:
        endpoint: ocitenancy.compat.objectstorage.ociregion.oraclecloud.com
        region: ociregion
        bucket: mybucket
      credentialsSecret:
        name: s3-credentials
    cluster:
      name: mysql-cluster
```
