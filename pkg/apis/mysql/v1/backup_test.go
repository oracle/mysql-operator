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
package v1

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"

	"github.com/oracle/mysql-operator/pkg/version"
)

func TestEmptyBackupIsInvalid(t *testing.T) {
	backup := MySQLBackup{}
	err := backup.Validate()
	if err == nil {
		t.Error("An empty backup should be invalid")
	}
}

func TestValidateValidBackup(t *testing.T) {
	backup := MySQLBackup{
		Spec: BackupSpec{
			Executor: &Executor{
				Provider:  "mysqldump",
				Databases: []string{"db1", "db2"},
			},
			Storage: &Storage{
				Provider: "s3",
				SecretRef: &corev1.LocalObjectReference{
					Name: "backup-storage-creds",
				},
				Config: map[string]string{
					"endpoint": "endpoint",
					"region":   "region",
					"bucket":   "bucket",
				},
			},
			ClusterRef: &corev1.LocalObjectReference{
				Name: "test-cluster",
			},
		},
	}
	backup.Labels = make(map[string]string)
	SetOperatorVersionLabel(backup.Labels, "v1.0.0")
	err := backup.Validate()
	if err != nil {
		t.Errorf("Expected no validation errors but got %s", err)
	}
}

func TestBackupEnsureDefaultVersionSet(t *testing.T) {
	expected := version.GetBuildVersion()
	backup := &MySQLBackup{}
	backup = backup.EnsureDefaults()

	actual := GetOperatorVersionLabel(backup.Labels)
	if actual != expected {
		t.Errorf("Expected version '%s' but got '%s'", expected, actual)
	}
}

func TestBackupEnsureDefaultVersionNotSetIfExists(t *testing.T) {
	version := "v1.0.0"
	backup := &MySQLBackup{}
	backup.Labels = make(map[string]string)
	SetOperatorVersionLabel(backup.Labels, version)
	backup = backup.EnsureDefaults()

	actual := GetOperatorVersionLabel(backup.Labels)

	if actual != version {
		t.Errorf("Expected version '%s' but got '%s'", version, actual)
	}
}

func TestValidateBackupMissingCluster(t *testing.T) {
	backup := MySQLBackup{
		Spec: BackupSpec{
			Executor: &Executor{
				Provider:  "mysqldump",
				Databases: []string{"db1", "db2"},
			},
			Storage: &Storage{
				Provider: "s3",
				SecretRef: &corev1.LocalObjectReference{
					Name: "backup-storage-creds",
				},
				Config: map[string]string{
					"endpoint": "endpoint",
					"region":   "region",
					"bucket":   "bucket",
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
	backup := MySQLBackup{
		Spec: BackupSpec{
			Executor: &Executor{
				Provider:  "mysqldump",
				Databases: []string{"db1", "db2"},
			},
			Storage: &Storage{
				Provider: "s3",
				Config: map[string]string{
					"endpoint": "endpoint",
					"region":   "region",
					"bucket":   "bucket",
				},
			},
			ClusterRef: &corev1.LocalObjectReference{
				Name: "test-cluster",
			},
		},
	}

	err := backup.Validate()
	if !strings.Contains(err.Error(), "storage.secretRef: Required value") {
		t.Errorf("Expected backup with missing SecretRef to show 'storage.secretRef: Required value' error. Error is: %s", err)
	}
}

// Error is: storage.secretRef: Required value
