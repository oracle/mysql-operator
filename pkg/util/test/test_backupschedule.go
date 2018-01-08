package test

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
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
