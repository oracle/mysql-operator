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

package v1alpha1

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"

	"github.com/oracle/mysql-operator/pkg/version"
)

func TestEmptyBackupIsInvalid(t *testing.T) {
	backup := Backup{}
	err := backup.Validate()
	if err == nil {
		t.Error("An empty backup should be invalid")
	}
}

func TestValidateValidBackup(t *testing.T) {
	backup := Backup{
		Spec: BackupSpec{
			Executor: BackupExecutor{
				MySQLDump: &MySQLDumpBackupExecutor{
					Databases: []Database{{Name: "db1"}, {Name: "db2"}},
				},
			},
			StorageProvider: StorageProvider{
				S3: &S3StorageProvider{
					Endpoint: "endpoint",
					Region:   "region",
					Bucket:   "bucket",
					CredentialsSecret: &corev1.LocalObjectReference{
						Name: "backup-storage-creds",
					},
				},
			},
			Cluster: &corev1.LocalObjectReference{
				Name: "test-cluster",
			},
		},
	}
	backup.Labels = make(map[string]string)
	setOperatorVersionLabel(backup.Labels, "v1.0.0")
	err := backup.Validate()
	if err != nil {
		t.Errorf("Expected no validation errors but got %s", err)
	}
}

func TestBackupEnsureDefaultVersionSet(t *testing.T) {
	expected := version.GetBuildVersion()
	backup := &Backup{}
	backup = backup.EnsureDefaults()

	actual := getOperatorVersionLabel(backup.Labels)
	if actual != expected {
		t.Errorf("Expected version '%s' but got '%s'", expected, actual)
	}
}

func TestBackupEnsureDefaultVersionNotSetIfExists(t *testing.T) {
	version := "v1.0.0"
	backup := &Backup{}
	backup.Labels = make(map[string]string)
	setOperatorVersionLabel(backup.Labels, version)
	backup = backup.EnsureDefaults()

	actual := getOperatorVersionLabel(backup.Labels)

	if actual != version {
		t.Errorf("Expected version '%s' but got '%s'", version, actual)
	}
}

func TestValidateBackupMissingCluster(t *testing.T) {
	backup := Backup{
		Spec: BackupSpec{
			Executor: BackupExecutor{
				MySQLDump: &MySQLDumpBackupExecutor{
					Databases: []Database{{Name: "db1"}, {Name: "db2"}},
				},
			},
			StorageProvider: StorageProvider{
				S3: &S3StorageProvider{
					Endpoint: "endpoint",
					Region:   "region",
					Bucket:   "bucket",
					CredentialsSecret: &corev1.LocalObjectReference{
						Name: "backup-storage-creds",
					},
				},
			},
		},
	}

	err := backup.Validate()
	if !strings.Contains(err.Error(), "missing cluster") {
		t.Errorf("Expected backup with missing Cluster to show 'missing cluster' error. Error is: %s", err)
	}
}

func TestValidateBackupMissingSecretRef(t *testing.T) {
	backup := Backup{
		Spec: BackupSpec{
			Executor: BackupExecutor{
				MySQLDump: &MySQLDumpBackupExecutor{
					Databases: []Database{{Name: "db1"}, {Name: "db2"}},
				},
			},
			StorageProvider: StorageProvider{
				S3: &S3StorageProvider{
					Endpoint: "endpoint",
					Region:   "region",
					Bucket:   "bucket",
				},
			},
			Cluster: &corev1.LocalObjectReference{
				Name: "test-cluster",
			},
		},
	}

	err := backup.Validate()
	if !strings.Contains(err.Error(), "storageProvider.s3.credentialsSecret: Required value") {
		t.Errorf("Expected backup with missing Secret to show 'storageProvider.s3.credentialsSecret: Required value' error. Error is: %s", err)
	}
}

func TestEmptyBackupScheduleIsInvalid(t *testing.T) {
	bs := BackupSchedule{}
	err := bs.Validate()
	if err == nil {
		t.Error("An empty backup schedule should be invalid")
	}
}

func TestValidateValidBackupSchedule(t *testing.T) {
	bs := BackupSchedule{
		Spec: BackupScheduleSpec{
			Schedule: "* * * * * *",
			BackupTemplate: BackupSpec{
				Executor: BackupExecutor{
					MySQLDump: &MySQLDumpBackupExecutor{
						Databases: []Database{{Name: "db1"}, {Name: "db2"}},
					},
				},
				StorageProvider: StorageProvider{
					S3: &S3StorageProvider{
						Endpoint: "endpoint",
						Region:   "region",
						Bucket:   "bucket",
						CredentialsSecret: &corev1.LocalObjectReference{
							Name: "backup-storage-creds",
						},
					},
				},
				Cluster: &corev1.LocalObjectReference{
					Name: "test-cluster",
				},
			},
		},
	}
	bs.Labels = make(map[string]string)
	setOperatorVersionLabel(bs.Labels, "v1.0.0")
	err := bs.Validate()
	if err != nil {
		t.Errorf("Expected no validation errors but got %s", err)
	}
}

func TestBackupScheduleEnsureDefaultVersionSet(t *testing.T) {
	expected := version.GetBuildVersion()
	bs := &BackupSchedule{}
	bs = bs.EnsureDefaults()

	actual := getOperatorVersionLabel(bs.Labels)
	if actual != expected {
		t.Errorf("Expected version '%s' but got '%s'", expected, actual)
	}
}

func TestBackupScheduleEnsureDefaultVersionNotSetIfExists(t *testing.T) {
	version := "v1.0.0"
	bs := &BackupSchedule{}
	bs.Labels = make(map[string]string)
	setOperatorVersionLabel(bs.Labels, version)
	bs = bs.EnsureDefaults()

	actual := getOperatorVersionLabel(bs.Labels)

	if actual != version {
		t.Errorf("Expected version '%s' but got '%s'", version, actual)
	}
}

func TestValidateBackupScheduleMissingCluster(t *testing.T) {
	bs := BackupSchedule{
		Spec: BackupScheduleSpec{
			Schedule: "* * * * * *",
			BackupTemplate: BackupSpec{
				Executor: BackupExecutor{
					MySQLDump: &MySQLDumpBackupExecutor{
						Databases: []Database{{Name: "db1"}, {Name: "db2"}},
					},
				},
				StorageProvider: StorageProvider{
					S3: &S3StorageProvider{
						Endpoint: "endpoint",
						Region:   "region",
						Bucket:   "bucket",
						CredentialsSecret: &corev1.LocalObjectReference{
							Name: "backup-storage-creds",
						},
					},
				},
				ScheduledMember: "hostname-1",
			},
		},
	}

	err := bs.Validate()
	if !strings.Contains(err.Error(), "missing cluster") {
		t.Errorf("Expected backup schedule with missing Cluster to show 'missing cluster' error. Error is: %s", err)
	}
}

func TestValidateBackupScheduleMissingSecretRef(t *testing.T) {
	bs := BackupSchedule{
		Spec: BackupScheduleSpec{
			Schedule: "* * * * * *",
			BackupTemplate: BackupSpec{
				Executor: BackupExecutor{
					MySQLDump: &MySQLDumpBackupExecutor{
						Databases: []Database{{Name: "db1"}, {Name: "db2"}},
					},
				},
				StorageProvider: StorageProvider{
					S3: &S3StorageProvider{
						Endpoint: "endpoint",
						Region:   "region",
						Bucket:   "bucket",
					},
				},
				Cluster: &corev1.LocalObjectReference{
					Name: "test-cluster",
				},
				ScheduledMember: "hostname-1",
			},
		},
	}

	err := bs.Validate()
	if !strings.Contains(err.Error(), "storageProvider.s3.credentialsSecret: Required value") {
		t.Errorf("Expected backup schedule with missing authSecret to show 'storageProvider.s3.credentialsSecret: Required value' error. Error is: %s", err)
	}
}

func TestRestoreEnsureDefaults(t *testing.T) {
	// test a version is set if one does not exist.
	bv1 := version.GetBuildVersion()
	r := Restore{
		Spec: RestoreSpec{
			Cluster: &corev1.LocalObjectReference{
				Name: "foo",
			},
			Backup: &corev1.LocalObjectReference{
				Name: "foo",
			},
		},
	}
	dr := *r.EnsureDefaults()
	if getOperatorVersionLabel(dr.Labels) != bv1 {
		t.Errorf("Expected restore version label: '%s'", bv1)
	}
	// test a version is not set if one already exists.
	bv2 := "test-existing-build-version"
	r2 := Restore{}
	r2.Labels = make(map[string]string)
	setOperatorVersionLabel(r2.Labels, bv2)
	dr2 := *r2.EnsureDefaults()
	if getOperatorVersionLabel(dr2.Labels) != bv2 {
		t.Errorf("Expected restore version label: '%s'", bv2)
	}
}

func TestRestoreValidate(t *testing.T) {
	// Test a malformed restore returns errors.
	r := Restore{
		Spec: RestoreSpec{
			Cluster: &corev1.LocalObjectReference{
				Name: "foo",
			},
			Backup: &corev1.LocalObjectReference{
				Name: "foo",
			},
		},
	}
	rErr := r.Validate()
	if rErr == nil {
		t.Error("Restore should have had a validation error.")
	}
	// Test a valid restore returns no errors.
	r.Labels = make(map[string]string)
	setOperatorVersionLabel(r.Labels, "some-build-version")
	rErr = r.Validate()
	if rErr != nil {
		t.Errorf("Restore should have had no validation errors: %v", rErr)
	}
}
