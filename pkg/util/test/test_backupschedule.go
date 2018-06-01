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

type TestMySQLBackupSchedule struct {
	*api.MySQLBackupSchedule
}

func NewTestMySQLBackupSchedule(namespace, name string) *TestMySQLBackupSchedule {
	return &TestMySQLBackupSchedule{
		MySQLBackupSchedule: &api.MySQLBackupSchedule{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      name,
				Labels:    make(map[string]string),
			},
			Spec: api.BackupScheduleSpec{
				BackupTemplate: api.BackupSpec{
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
		},
	}
}

func (s *TestMySQLBackupSchedule) WithPhase(phase api.BackupSchedulePhase) *TestMySQLBackupSchedule {
	s.Status.Phase = phase
	return s
}

func (s *TestMySQLBackupSchedule) WithCronSchedule(cronExpression string) *TestMySQLBackupSchedule {
	s.Spec.Schedule = cronExpression
	return s
}

func (s *TestMySQLBackupSchedule) WithLastBackupTime(timeString string) *TestMySQLBackupSchedule {
	t, _ := time.Parse("2006-01-02 15:04:05", timeString)
	s.Status.LastBackup = metav1.Time{Time: t}
	return s
}

func (s *TestMySQLBackupSchedule) WithLabel(key, value string) *TestMySQLBackupSchedule {
	s.Labels[key] = value
	return s
}
