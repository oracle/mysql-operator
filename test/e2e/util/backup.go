package util

import (
	api "github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
	"github.com/oracle/mysql-operator/pkg/constants"
	"github.com/oracle/mysql-operator/test/e2e/framework"
)

// Backup creates backup of the given database in the given cluster.
func Backup(t *T, f *framework.Framework, clusterName string, backupNamePrefix string, databaseName string) string {
	s3StorageCredentials := "s3-upload-credentials"
	backupSpec := NewMySQLBackup(clusterName, backupNamePrefix, s3StorageCredentials, []string{databaseName})
	backup, err := f.MySQLOpClient.MysqlV1().MySQLBackups(f.Namespace).Create(backupSpec)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}
	backupBackoff := NewDefaultRetyWithDuration(10)
	backupBackoff.Steps = 10
	backup, err = WaitForBackupPhase(t, backup, api.BackupPhaseComplete, backupBackoff, f.MySQLOpClient)
	if err != nil {
		t.Fatalf("Backup failed to reach phase %q: %v", api.BackupPhaseComplete, err)
	}
	if backup.Labels[constants.MySQLOperatorVersionLabel] != f.BuildVersion {
		t.Errorf("Backup MySQLOperatorVersionLabel was incorrect: %s != %s.", backup.Labels[constants.MySQLOperatorVersionLabel], f.BuildVersion)
	} else {
		t.Logf("Backup label MySQLOperatorVersionLabel: %s", backup.Labels[constants.MySQLOperatorVersionLabel])
	}
	if backup.Spec.AgentScheduled != clusterName+"-0" {
		t.Fatalf("Backup was not scheduled on cluster master node.")
	} else {
		t.Logf("Backup AgentScheduled: %s", backup.Spec.AgentScheduled)
	}
	if backup.Status.Outcome.Location == "" {
		t.Fatalf("Backup failed to specify a location for the image archive.")
	}
	t.Logf("created backup at location: %s", backup.Status.Outcome.Location)
	return backup.Name
}

// DeleteDatabase deletes the given database from a cluster.
func DeleteDatabase(t *T, f *framework.Framework, clusterName string, databaseName string) {
	podName := clusterName + "-0"
	username := "root"
	password := GetMySQLPassword(t, podName, f.Namespace)
	sqlExecutor := NewKubectlSimpleSQLExecutor(t, podName, username, password, f.Namespace)
	dbHelper := NewMySQLDBTestHelper(t, sqlExecutor)
	dbHelper.DeleteDB(databaseName)
	if dbHelper.HasDB(databaseName) {
		t.Fatalf("Failed to delete %s database", databaseName)
	}
}

// Restore restores a backup into the given cluster.
func Restore(t *T, f *framework.Framework, clusterName string, backupName string) {
	restoreName := backupName + "-restore-"
	restoreSpec := NewMySQLRestore(clusterName, backupName, restoreName)
	restore, err := f.MySQLOpClient.MysqlV1().MySQLRestores(f.Namespace).Create(restoreSpec)
	if err != nil {
		t.Fatalf("Failed to create restore '%s': %v", backupName, err)
	}
	restoreBackoff := NewDefaultRetyWithDuration(10)
	restoreBackoff.Steps = 24
	restore, err = WaitForRestorePhase(t, restore, api.RestorePhaseComplete, restoreBackoff, f.MySQLOpClient)
	if err != nil {
		t.Fatalf("Restore failed to reach phase %q: %v", api.RestorePhaseComplete, err)
	}
	if restore.Labels[constants.MySQLOperatorVersionLabel] != f.BuildVersion {
		t.Errorf("Restore MySQLOperatorVersionLabel was incorrect: %s != %s.", restore.Labels[constants.MySQLOperatorVersionLabel], f.BuildVersion)
	} else {
		t.Logf("Restore label MySQLOperatorVersionLabel: %s", restore.Labels[constants.MySQLOperatorVersionLabel])
	}
}
