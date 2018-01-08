package e2e

import (
	"fmt"
	"testing"
	"time"

	constants "github.com/oracle/mysql-operator/pkg/constants"
	"github.com/oracle/mysql-operator/test/e2e/framework"
	e2eutil "github.com/oracle/mysql-operator/test/e2e/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestScheduledBackup(t *testing.T) {
	f := framework.Global

	// ---------------------------------------------------------------------- //
	fmt.Println("creating cluster..")
	// ---------------------------------------------------------------------- //
	testdb := e2eutil.CreateTestDB(t, "e2e-br-", 1, f.DestroyAfterFailure)
	defer testdb.Delete()
	clusterName := testdb.Cluster().Name

	// ---------------------------------------------------------------------- //
	fmt.Println("populating database..")
	// ---------------------------------------------------------------------- //
	testDatabaseName := "test"
	podName := clusterName + "-0"
	username := "root"
	password := e2eutil.GetMySQLPassword(t, podName, f.Namespace)
	sqlExecutor := e2eutil.NewKubectlSimpleSQLExecutor(t, podName, username, password, f.Namespace)
	dbHelper := e2eutil.NewMySQLDBTestHelper(t, sqlExecutor)
	dbHelper.EnsureDBTableValue(testDatabaseName, "people", "name", "kris")

	// ---------------------------------------------------------------------- //
	fmt.Printf("creating backup schedule for cluster '%s' that runs every minute...\n", clusterName)
	// ---------------------------------------------------------------------- //
	backupScheduleName := "e2e-test-backup-schedule-"
	s3StorageCredentials := "s3-upload-credentials"
	schedule := "@every 1m"
	backupScheduleSpec := e2eutil.NewMySQLBackupSchedule(clusterName, backupScheduleName, schedule, s3StorageCredentials, []string{testDatabaseName})
	backupSchedule, err := f.MySQLOpClient.MysqlV1().MySQLBackupSchedules(f.Namespace).Create(backupScheduleSpec)
	if err != nil {
		t.Fatalf("Failed to create backup schedule: %v", err)
	}

	// ---------------------------------------------------------------------- //
	fmt.Println("checking that 1 complete backup exists, and is labelled correctly..")
	// ---------------------------------------------------------------------- //
	time.Sleep(5 * time.Second)
	n := numCompletedBackups(t, f, backupSchedule.Name)
	if n != 1 {
		t.Fatalf("Expected 1 completed backups, found: %d", n)
	}

	// ---------------------------------------------------------------------- //
	fmt.Println("checking that 2 complete backups exist, and are labelled correctly..")
	// ---------------------------------------------------------------------- //
	time.Sleep(95 * time.Second)
	n = numCompletedBackups(t, f, backupSchedule.Name)
	if n != 2 {
		t.Fatalf("Expected 2 completed backups, found: %d", n)
	}

	// ---------------------------------------------------------------------- //
	fmt.Println("validating operator version label on the backup schedule..")
	// ---------------------------------------------------------------------- //
	backupSchedule, err = f.MySQLOpClient.MysqlV1().MySQLBackupSchedules(f.Namespace).Get(backupSchedule.Name, metav1.GetOptions{})
	if backupSchedule.Labels[constants.MySQLOperatorVersionLabel] != f.BuildVersion {
		t.Errorf("BackupSchedule MySQLOperatorVersionLabel was incorrect: %s != %s.", backupSchedule.Labels[constants.MySQLOperatorVersionLabel], f.BuildVersion)
	} else {
		t.Logf("BackupSchedule label MySQLOperatorVersionLabel: %s", backupSchedule.Labels[constants.MySQLOperatorVersionLabel])
	}

	// ---------------------------------------------------------------------- //
	fmt.Printf("deleteing backup schedule: %s\n", backupSchedule.Name)
	// ---------------------------------------------------------------------- //
	err = f.MySQLOpClient.MysqlV1().MySQLBackupSchedules(f.Namespace).Delete(backupSchedule.Name, &metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Failed to delete backup schedule: %v", err)
	}
}

func numCompletedBackups(t *testing.T, f *framework.Framework, backupScheduleName string) int {
	labelSelector := fmt.Sprintf("backup-schedule=%s", backupScheduleName)
	listOpts := metav1.ListOptions{LabelSelector: labelSelector}
	backupList, err := f.MySQLOpClient.MysqlV1().MySQLBackups(f.Namespace).List(listOpts)
	if err != nil {
		t.Fatalf("Failed to list backups with label: %s", labelSelector)
		return 0
	} else {
		for _, backup := range backupList.Items {
			fmt.Printf("Found backup, name: %s, phase: %s\n", backup.Name, backup.Status.Phase)
		}
		return len(backupList.Items)
	}
}
