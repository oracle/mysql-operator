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

package test

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
)

type TestBackup struct {
	*v1alpha1.Backup
}

func NewTestBackup() *TestBackup {
	return &TestBackup{
		Backup: &v1alpha1.Backup{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: metav1.NamespaceDefault,
			},
			Spec: v1alpha1.BackupSpec{
				Executor: v1alpha1.BackupExecutor{
					MySQLDump: &v1alpha1.MySQLDumpBackupExecutor{
						Databases: []v1alpha1.Database{{Name: "test"}},
					},
				},
				StorageProvider: v1alpha1.StorageProvider{
					S3: &v1alpha1.S3StorageProvider{
						Endpoint: "endpoint",
						Region:   "region",
						Bucket:   "bucket",
						CredentialsSecret: &corev1.LocalObjectReference{
							Name: "name",
						},
					},
				},
				Cluster: &corev1.LocalObjectReference{},
			},
		},
	}
}

func (b *TestBackup) WithNamespace(namespace string) *TestBackup {
	b.Namespace = namespace
	return b
}

func (b *TestBackup) WithName(name string) *TestBackup {
	b.Name = name
	return b
}

func (b *TestBackup) WithLabel(key, value string) *TestBackup {
	if b.Labels == nil {
		b.Labels = make(map[string]string)
	}
	b.Labels[key] = value

	return b
}
