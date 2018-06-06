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
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
)

type TestBackupSchedule struct {
	*api.BackupSchedule
}

func NewTestBackupSchedule(namespace, name string) *TestBackupSchedule {
	return &TestBackupSchedule{
		BackupSchedule: &api.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      name,
				Labels:    make(map[string]string),
			},
			Spec: api.BackupScheduleSpec{
				BackupTemplate: api.BackupSpec{
					Executor: &api.BackupExecutor{
						Name:      "mysqldump",
						Databases: []string{"test"},
					},
					StorageProvider: &api.BackupStorageProvider{
						Name: "s3",
						AuthSecret: &corev1.LocalObjectReference{
							Name: "name",
						},
						Config: map[string]string{
							"endpoint": "endpoint",
							"region":   "region",
							"bucket":   "bucket",
						},
					},
					Cluster: &corev1.LocalObjectReference{},
				},
			},
		},
	}
}

func (s *TestBackupSchedule) WithCronSchedule(cronExpression string) *TestBackupSchedule {
	s.Spec.Schedule = cronExpression
	return s
}

func (s *TestBackupSchedule) WithLastBackupTime(timeString string) *TestBackupSchedule {
	t, _ := time.Parse("2006-01-02 15:04:05", timeString)
	s.Status.LastBackup = metav1.Time{Time: t}
	return s
}

func (s *TestBackupSchedule) WithLabel(key, value string) *TestBackupSchedule {
	s.Labels[key] = value
	return s
}
