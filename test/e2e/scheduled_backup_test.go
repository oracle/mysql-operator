// Copyright 2018 Oracle and/or its affiliates. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build all default

package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/oracle/mysql-operator/pkg/constants"
	"github.com/oracle/mysql-operator/test/e2e/framework"
	e2eutil "github.com/oracle/mysql-operator/test/e2e/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestScheduledBackup(test *testing.T) {
	t := e2eutil.NewT(test)
	f := framework.Global

	// ---------------------------------------------------------------------- //
	t.Log("Creating cluster..")
	// ---------------------------------------------------------------------- //
	testdb := e2eutil.CreateTestDB(t, "e2e-br-", 1, false, f.DestroyAfterFailure)
	defer testdb.Delete()
	clusterName := testdb.Cluster().Name

	// ---------------------------------------------------------------------- //
	t.Log("Populating database..")
	// ---------------------------------------------------------------------- //
	testDatabaseName := "test"
	podName := clusterName + "-0"
	username := "root"
	password := e2eutil.GetMySQLPassword(t, podName, f.Namespace)
	sqlExecutor := e2eutil.NewKubectlSimpleSQLExecutor(t, podName, username, password, f.Namespace)
	dbHelper := e2eutil.NewMySQLDBTestHelper(t, sqlExecutor)
	dbHelper.EnsureDBTableValue(testDatabaseName, "people", "name", "kris")

	// ---------------------------------------------------------------------- //
	t.Logf("Creating backup schedule for cluster '%s' that runs every minute...", clusterName)
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
	t.Log("Checking that 1 complete backup exists, and is labelled correctly..")
	// ---------------------------------------------------------------------- //
	time.Sleep(5 * time.Second)
	n := numCompletedBackups(t, f, backupSchedule.Name)
	if n != 1 {
		t.Fatalf("Expected 1 completed backups, found: %d", n)
	}

	// ---------------------------------------------------------------------- //
	t.Log("Checking that 2 complete backups exist, and are labelled correctly..")
	// ---------------------------------------------------------------------- //
	time.Sleep(95 * time.Second)
	n = numCompletedBackups(t, f, backupSchedule.Name)
	if n != 2 {
		t.Fatalf("Expected 2 completed backups, found: %d", n)
	}

	// ---------------------------------------------------------------------- //
	t.Log("Validating operator version label on the backup schedule..")
	// ---------------------------------------------------------------------- //
	backupSchedule, err = f.MySQLOpClient.MysqlV1().MySQLBackupSchedules(f.Namespace).Get(backupSchedule.Name, metav1.GetOptions{})
	if backupSchedule.Labels[constants.MySQLOperatorVersionLabel] != f.BuildVersion {
		t.Errorf("BackupSchedule MySQLOperatorVersionLabel was incorrect: %s != %s.", backupSchedule.Labels[constants.MySQLOperatorVersionLabel], f.BuildVersion)
	} else {
		t.Logf("BackupSchedule label MySQLOperatorVersionLabel: %s", backupSchedule.Labels[constants.MySQLOperatorVersionLabel])
	}

	// ---------------------------------------------------------------------- //
	t.Logf("Deleteing backup schedule: %s", backupSchedule.Name)
	// ---------------------------------------------------------------------- //
	err = f.MySQLOpClient.MysqlV1().MySQLBackupSchedules(f.Namespace).Delete(backupSchedule.Name, &metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Failed to delete backup schedule: %v", err)
	}

	t.Report()
}

func numCompletedBackups(t *e2eutil.T, f *framework.Framework, backupScheduleName string) int {
	labelSelector := fmt.Sprintf("backup-schedule=%s", backupScheduleName)
	listOpts := metav1.ListOptions{LabelSelector: labelSelector}
	backupList, err := f.MySQLOpClient.MysqlV1().MySQLBackups(f.Namespace).List(listOpts)
	if err != nil {
		t.Fatalf("Failed to list backups with label: %s", labelSelector)
		return 0
	} else {
		for _, backup := range backupList.Items {
			t.Logf("Found backup, name: %s, phase: %s", backup.Name, backup.Status.Phase)
		}
		return len(backupList.Items)
	}
}
