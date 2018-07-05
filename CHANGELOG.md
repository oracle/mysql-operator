# CHANGELOG

## 0.2.0

**MAJOR BACKWARDS INCOMPATIBLE CHANGES**. For an overview of the changes and
upgrade information please see [`docs/0.1-upgrade.md`][1].

 - Enforce 8.0.11 as the minimum supported MySQL server version. [#169]
 - Numerous changes to the MySQL Operator Custom Resources. [#123]
 - Downgrade the MySQL Operator Custom Resource API version from v1 to v1alpha1
   to enable future iteration on the API and better reflect its stability [#122]
 - Group communication connections as are now secured using SSL with support for
   specifying your own certificate [#115].

## 0.1.1

 - Allow any version string to be used in Cluster spec. [#120]

## 0.1.0

Initial release of Oracle MySQL Operator.

 - Create and manage MySQL clusters.
 - Implement full cluster backup to S3 using mysqldump.
 - Implement ability to restore cluster from backup.
 - Expose cluster metrics with Prometheus.
 - Helm chart for deploying the operator.

[1]: https://github.com/oracle/mysql-operator/blob/master/docs/0.1-upgrade.md
