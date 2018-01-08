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
