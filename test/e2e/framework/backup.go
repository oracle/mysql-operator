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

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
	mysqlclientset "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned"
)

// MySQLBackupTestJig is a jig to help MySQLBackup testing.
type MySQLBackupTestJig struct {
	ID     string
	Name   string
	Labels map[string]string

	MySQLClient mysqlclientset.Interface
	KubeClient  clientset.Interface
}

// NewMySQLBackupTestJig allocates and inits a new MySQLBackupTestJig.
func NewMySQLBackupTestJig(mysqlClient mysqlclientset.Interface, kubeClient clientset.Interface, name string) *MySQLBackupTestJig {
	id := string(uuid.NewUUID())
	return &MySQLBackupTestJig{
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

// newMySQLBackupTemplate returns the default v1.MySQLBackup template for this jig, but
// does not actually create the MySQLBackup. The default MySQLBackup has the same name
// as the jig.
func (j *MySQLBackupTestJig) newMySQLBackupTemplate(namespace, clusterName string) *v1.MySQLBackup {
	return &v1.MySQLBackup{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1.MySQLBackupCRDResourceKind,
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: j.Name,
			Namespace:    namespace,
			Labels:       j.Labels,
		},
		Spec: v1.BackupSpec{
			ClusterRef: &corev1.LocalObjectReference{
				Name: clusterName,
			},
		},
	}
}

// CreateMySQLBackupOrFail creates a new MySQLBackup based on the jig's
// defaults. Callers can provide a function to tweak the MySQLBackup object
// before it is created.
func (j *MySQLBackupTestJig) CreateMySQLBackupOrFail(namespace, clusterName string, tweak func(backup *v1.MySQLBackup)) *v1.MySQLBackup {
	backup := j.newMySQLBackupTemplate(namespace, clusterName)
	if tweak != nil {
		tweak(backup)
	}

	name := types.NamespacedName{Namespace: namespace, Name: j.Name}
	By(fmt.Sprintf("Creating a MySQLBackup %q", name))

	result, err := j.MySQLClient.MysqlV1().MySQLBackups(namespace).Create(backup)
	if err != nil {
		Failf("Failed to create MySQLBackup %q: %v", name, err)
	}
	return result
}

// CreateAndAwaitMySQLBackupOrFail creates a new MySQLBackup based on the
// jig's defaults, waits for it to become ready. Callers can provide a function
// to tweak the MySQLBackup object before it is created.
func (j *MySQLBackupTestJig) CreateAndAwaitMySQLBackupOrFail(namespace, clusterName string, tweak func(backup *v1.MySQLBackup), timeout time.Duration) *v1.MySQLBackup {
	backup := j.CreateMySQLBackupOrFail(namespace, clusterName, tweak)
	return j.WaitForbackupReadyOrFail(namespace, backup.Name, timeout)
}

// CreateAndAwaitMySQLDumpBackupOrFail creates a new MySQLBackup based on the
// jig's defaults, waits for it to become ready. Callers can provide a function
// to tweak the MySQLBackup object before it is created.
func (j *MySQLBackupTestJig) CreateAndAwaitMySQLDumpBackupOrFail(namespace, clusterName string, databases []string, tweak func(backup *v1.MySQLBackup), timeout time.Duration) *v1.MySQLBackup {
	backup := j.CreateMySQLBackupOrFail(namespace, clusterName, func(backup *v1.MySQLBackup) {
		backup.Spec.Executor = &v1.Executor{
			Provider:  "mysqldump",
			Databases: databases,
		}
		tweak(backup)
	})
	backup = j.WaitForbackupReadyOrFail(namespace, backup.Name, timeout)
	return backup
}

// CreateS3AuthSecret creates a secret containing the S3 (compat.) credentials
// for storing backups.
func (j *MySQLBackupTestJig) CreateS3AuthSecret(namespace, name string) (*corev1.Secret, error) {
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

func (j *MySQLBackupTestJig) waitForConditionOrFail(namespace, name string, timeout time.Duration, message string, conditionFn func(*v1.MySQLBackup) bool) *v1.MySQLBackup {
	var backup *v1.MySQLBackup
	pollFunc := func() (bool, error) {
		b, err := j.MySQLClient.MysqlV1().MySQLBackups(namespace).Get(name, metav1.GetOptions{})
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
		Failf("Timed out waiting for MySQLBackup %q to %s", name, message)
	}
	return backup
}

// WaitForbackupReadyOrFail waits up to a given timeout for a backup to be in
// the running phase.
func (j *MySQLBackupTestJig) WaitForbackupReadyOrFail(namespace, name string, timeout time.Duration) *v1.MySQLBackup {
	Logf("Waiting up to %v for MySQLBackup \"%s/%s\" to be complete executing", timeout, namespace, name)
	backup := j.waitForConditionOrFail(namespace, name, timeout, "to complete executing", func(backup *v1.MySQLBackup) bool {
		phase := backup.Status.Phase
		if phase == v1.BackupPhaseComplete {
			return true
		}

		if phase == v1.BackupPhaseFailed {
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
			Failf("MySQLBackup entered state %q", v1.BackupPhaseFailed)
		}
		return false
	})
	return backup
}
