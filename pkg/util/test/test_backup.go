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

	api "github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
)

type TestMySQLBackup struct {
	*api.MySQLBackup
}

func NewTestMySQLBackup() *TestMySQLBackup {
	return &TestMySQLBackup{
		MySQLBackup: &api.MySQLBackup{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: metav1.NamespaceDefault,
			},
			Spec: api.BackupSpec{
				Executor: &api.Executor{
					Provider:  "mysqldump",
					Databases: []string{"test"},
				},
				Storage: &api.Storage{
					Provider: "s3",
					SecretRef: &corev1.LocalObjectReference{
						Name: "name",
					},
					Config: map[string]string{
						"endpoint": "endpoint",
						"region":   "region",
						"bucket":   "bucket",
					},
				},
				ClusterRef: &corev1.LocalObjectReference{},
			},
		},
	}
}

func (b *TestMySQLBackup) WithNamespace(namespace string) *TestMySQLBackup {
	b.Namespace = namespace
	return b
}

func (b *TestMySQLBackup) WithName(name string) *TestMySQLBackup {
	b.Name = name
	return b
}

func (b *TestMySQLBackup) WithLabel(key, value string) *TestMySQLBackup {
	if b.Labels == nil {
		b.Labels = make(map[string]string)
	}
	b.Labels[key] = value

	return b
}
