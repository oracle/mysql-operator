package util

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
)

func NewMySQLCluster(genName string, replicas int32) *api.MySQLCluster {
	return &api.MySQLCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       api.MySQLClusterCRDResourceKind,
			APIVersion: api.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: genName,
		},
		Spec: api.MySQLClusterSpec{
			Replicas: replicas,
		},
	}
}

// NewMySQLBackup creates a valid mock MySQLBackup for e2e testing.
func NewMySQLBackup(clusterName string, backupName string, ossCredsSecretRef string, databases []string) *api.MySQLBackup {
	return &api.MySQLBackup{
		TypeMeta: metav1.TypeMeta{
			Kind:       api.MySQLBackupCRDResourceKind,
			APIVersion: api.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: backupName,
		},
		Spec: api.BackupSpec{
			Executor: &api.Executor{
				Provider:  "mysqldump",
				Databases: databases,
			},
			Storage: &api.Storage{
				Provider: "s3",
				SecretRef: &v1.LocalObjectReference{
					Name: ossCredsSecretRef,
				},
				Config: map[string]string{
					"endpoint": "bristoldev.compat.objectstorage.us-phoenix-1.oraclecloud.com",
					"region":   "us-phoenix-1",
					"bucket":   "trjl-test",
				},
			},
			ClusterRef: &v1.LocalObjectReference{
				Name: clusterName,
			},
		},
	}
}

func NewMySQLRestore(clusterName string, backupName string, restoreName string) *api.MySQLRestore {
	return &api.MySQLRestore{
		TypeMeta: metav1.TypeMeta{
			Kind:       api.MySQLRestoreCRDResourceKind,
			APIVersion: api.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: restoreName,
		},
		Spec: api.RestoreSpec{
			ClusterRef: &v1.LocalObjectReference{
				Name: clusterName,
			},
			BackupRef: &v1.LocalObjectReference{
				Name: backupName,
			},
		},
	}
}

func NewMySQLBackupSchedule(clusterName string, backupScheduleName string, schedule string, ossCredsSecretRef string, databases []string) *api.MySQLBackupSchedule {
	return &api.MySQLBackupSchedule{
		TypeMeta: metav1.TypeMeta{
			Kind:       api.MySQLBackupScheduleCRDResourceKind,
			APIVersion: api.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: backupScheduleName,
		},
		Spec: api.BackupScheduleSpec{
			Schedule: schedule,
			// trjl
			// BackupTemplate: api.BackupSpec{
			// 	Cluster: &v1.LocalObjectReference{
			// 		Name: clusterName,
			// 	},
			// 	SecretRef: &v1.LocalObjectReference{
			// 		Name: ossCredsSecretRef,
			// 	},
			// 	Databases: databases,
			// },
			BackupTemplate: api.BackupSpec{
				Executor: &api.Executor{
					Provider:  "mysqldump",
					Databases: databases,
				},
				Storage: &api.Storage{
					Provider: "s3",
					SecretRef: &v1.LocalObjectReference{
						Name: ossCredsSecretRef,
					},
					Config: map[string]string{
						"endpoint": "bristoldev.compat.objectstorage.us-phoenix-1.oraclecloud.com",
						"region":   "us-phoenix-1",
						"bucket":   "trjl-test",
					},
				},
				ClusterRef: &v1.LocalObjectReference{
					Name: clusterName,
				},
			},
		},
	}
}
