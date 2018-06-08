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

package framework

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"

	. "github.com/onsi/ginkgo"

	backuputil "github.com/oracle/mysql-operator/pkg/api/backup"
	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
	mysqlclientset "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned"
)

// BackupTestJig is a jig to help Backup testing.
type BackupTestJig struct {
	ID     string
	Name   string
	Labels map[string]string

	MySQLClient mysqlclientset.Interface
	KubeClient  clientset.Interface
}

// NewBackupTestJig allocates and inits a new BackupTestJig.
func NewBackupTestJig(mysqlClient mysqlclientset.Interface, kubeClient clientset.Interface, name string) *BackupTestJig {
	id := string(uuid.NewUUID())
	return &BackupTestJig{
		ID:   id,
		Name: name,
		Labels: map[string]string{
			"testID":   id,
			"testName": name,
		},

		MySQLClient: mysqlClient,
		KubeClient:  kubeClient,
	}
}

// newBackupTemplate returns the default v1alpha1.Backup template for
// this jig, but does not actually create the Backup. The default
// Backup has the same name as the jig.
func (j *BackupTestJig) newBackupTemplate(namespace, clusterName string) *v1alpha1.Backup {
	return &v1alpha1.Backup{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha1.BackupCRDResourceKind,
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: j.Name,
			Namespace:    namespace,
			Labels:       j.Labels,
		},
		Spec: v1alpha1.BackupSpec{
			Cluster: &corev1.LocalObjectReference{
				Name: clusterName,
			},
		},
	}
}

// CreateBackupOrFail creates a new Backup based on the jig's
// defaults. Callers can provide a function to tweak the Backup object
// before it is created.
func (j *BackupTestJig) CreateBackupOrFail(namespace, clusterName string, tweak func(backup *v1alpha1.Backup)) *v1alpha1.Backup {
	backup := j.newBackupTemplate(namespace, clusterName)
	if tweak != nil {
		tweak(backup)
	}

	name := types.NamespacedName{Namespace: namespace, Name: j.Name}
	By(fmt.Sprintf("Creating a Backup %q", name))

	result, err := j.MySQLClient.MySQLV1alpha1().Backups(namespace).Create(backup)
	if err != nil {
		Failf("Failed to create Backup %q: %v", name, err)
	}
	return result
}

// CreateAndAwaitBackupOrFail creates a new Backup based on the
// jig's defaults, waits for it to become ready. Callers can provide a function
// to tweak the Backup object before it is created.
func (j *BackupTestJig) CreateAndAwaitBackupOrFail(namespace, clusterName string, tweak func(backup *v1alpha1.Backup), timeout time.Duration) *v1alpha1.Backup {
	backup := j.CreateBackupOrFail(namespace, clusterName, tweak)
	return j.WaitForbackupReadyOrFail(namespace, backup.Name, timeout)
}

// CreateAndAwaitMySQLDumpBackupOrFail creates a new Backup based on the
// jig's defaults, waits for it to become ready. Callers can provide a function
// to tweak the Backup object before it is created.
func (j *BackupTestJig) CreateAndAwaitMySQLDumpBackupOrFail(namespace, clusterName string, databases []v1alpha1.Database, tweak func(backup *v1alpha1.Backup), timeout time.Duration) *v1alpha1.Backup {
	backup := j.CreateBackupOrFail(namespace, clusterName, func(backup *v1alpha1.Backup) {
		backup.Spec.Executor = v1alpha1.BackupExecutor{
			MySQLDump: &v1alpha1.MySQLDumpBackupExecutor{
				Databases: databases,
			},
		}
		tweak(backup)
	})
	backup = j.WaitForbackupReadyOrFail(namespace, backup.Name, timeout)
	return backup
}

// CreateS3AuthSecret creates a secret containing the S3 (compat.) credentials
// for storing backups.
func (j *BackupTestJig) CreateS3AuthSecret(namespace, name string) (*corev1.Secret, error) {
	return j.KubeClient.CoreV1().Secrets(namespace).Create(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"accessKey": []byte(TestContext.S3AccessKey),
			"secretKey": []byte(TestContext.S3SecretKey),
		},
	})
}

func (j *BackupTestJig) waitForConditionOrFail(namespace, name string, timeout time.Duration, message string, conditionFn func(*v1alpha1.Backup) bool) *v1alpha1.Backup {
	var backup *v1alpha1.Backup
	pollFunc := func() (bool, error) {
		b, err := j.MySQLClient.MySQLV1alpha1().Backups(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if conditionFn(b) {
			backup = b
			return true, nil
		}
		return false, nil
	}
	if err := wait.PollImmediate(Poll, timeout, pollFunc); err != nil {
		Failf("Timed out waiting for Backup %q to %s", name, message)
	}
	return backup
}

// WaitForbackupReadyOrFail waits up to a given timeout for a backup to be in
// the running phase.
func (j *BackupTestJig) WaitForbackupReadyOrFail(namespace, name string, timeout time.Duration) *v1alpha1.Backup {
	Logf("Waiting up to %v for Backup \"%s/%s\" to be complete executing", timeout, namespace, name)
	backup := j.waitForConditionOrFail(namespace, name, timeout, "to complete executing", func(backup *v1alpha1.Backup) bool {
		_, cond := backuputil.GetBackupCondition(&backup.Status, v1alpha1.BackupComplete)
		if cond != nil && cond.Status == corev1.ConditionTrue {
			return true
		}
		_, cond = backuputil.GetBackupCondition(&backup.Status, v1alpha1.BackupFailed)
		if cond != nil && cond.Status == corev1.ConditionTrue {
			ns := backup.Namespace
			events, err := j.KubeClient.CoreV1().Events(ns).List(metav1.ListOptions{})
			if err != nil {
				Failf("Failed to list Events in %q: %v", ns, err)
			}
			for _, e := range events.Items {
				if e.InvolvedObject.Kind != backup.Kind || e.InvolvedObject.Name != backup.Name {
					continue
				}
				Logf(e.String())
			}
			Failf("Backup condition failed (%s==%s)", cond.Type, cond.Status)
		}
		return false
	})
	return backup
}
