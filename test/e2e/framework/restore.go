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

// MySQLRestoreTestJig is a jig to help MySQLRestore testing.
type MySQLRestoreTestJig struct {
	ID     string
	Name   string
	Labels map[string]string

	MySQLClient mysqlclientset.Interface
	KubeClient  clientset.Interface
}

// NewMySQLRestoreTestJig allocates and inits a new MySQLRestoreTestJig.
func NewMySQLRestoreTestJig(mysqlClient mysqlclientset.Interface, kubeClient clientset.Interface, name string) *MySQLRestoreTestJig {
	id := string(uuid.NewUUID())
	return &MySQLRestoreTestJig{
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

// newMySQLRestoreTemplate returns the default v1.MySQLRestore template for this jig, but
// does not actually create the MySQLRestore. The default MySQLRestore has the
// same name as the jig.
func (j *MySQLRestoreTestJig) newMySQLRestoreTemplate(namespace, clusterName, backupName string) *v1.MySQLRestore {
	return &v1.MySQLRestore{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: j.Name,
			Namespace:    namespace,
			Labels:       j.Labels,
		},
		Spec: v1.RestoreSpec{
			ClusterRef: &corev1.LocalObjectReference{
				Name: clusterName,
			},
			BackupRef: &corev1.LocalObjectReference{
				Name: backupName,
			},
		},
	}
}

// CreateMySQLRestoreOrFail creates a new MySQLRestore based on the jig's
// defaults. Callers can provide a function to tweak the MySQLRestore object
// before it is created.
func (j *MySQLRestoreTestJig) CreateMySQLRestoreOrFail(namespace, clusterName, backupName string, tweak func(restore *v1.MySQLRestore)) *v1.MySQLRestore {
	restore := j.newMySQLRestoreTemplate(namespace, clusterName, backupName)
	if tweak != nil {
		tweak(restore)
	}

	name := types.NamespacedName{Namespace: namespace, Name: j.Name}
	By(fmt.Sprintf("Creating a MySQLRestore %q", name))

	result, err := j.MySQLClient.MysqlV1().MySQLRestores(namespace).Create(restore)
	if err != nil {
		Failf("Failed to create MySQLRestore %q: %v", name, err)
	}
	return result
}

// CreateAndAwaitMySQLRestoreOrFail creates a new MySQLRestore based on the
// jig's defaults, waits for it to become ready. Callers can provide a function
// to tweak the MySQLRestore object before it is created.
func (j *MySQLRestoreTestJig) CreateAndAwaitMySQLRestoreOrFail(namespace, clusterName, backupName string, tweak func(restore *v1.MySQLRestore), timeout time.Duration) *v1.MySQLRestore {
	restore := j.CreateMySQLRestoreOrFail(namespace, clusterName, backupName, tweak)
	restore = j.WaitForRestoreCompleteOrFail(namespace, restore.Name, timeout)
	return restore
}

// CreateS3AuthSecret creates a secret containing the S3 (compat.) credentials
// for storing backups.
func (j *MySQLRestoreTestJig) CreateS3AuthSecret(namespace, name string) (*corev1.Secret, error) {
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

func (j *MySQLRestoreTestJig) waitForConditionOrFail(namespace, name string, timeout time.Duration, message string, conditionFn func(*v1.MySQLRestore) bool) *v1.MySQLRestore {
	var restore *v1.MySQLRestore
	pollFunc := func() (bool, error) {
		r, err := j.MySQLClient.MysqlV1().MySQLRestores(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if conditionFn(r) {
			restore = r
			return true, nil
		}
		return false, nil
	}
	if err := wait.PollImmediate(Poll, timeout, pollFunc); err != nil {
		Failf("Timed out waiting for MySQLRestore %q to %s", name, message)
	}
	return restore
}

// WaitForRestoreCompleteOrFail waits up to a given timeout for a MySQLRestore
// to enter the complete phase.
func (j *MySQLRestoreTestJig) WaitForRestoreCompleteOrFail(namespace, name string, timeout time.Duration) *v1.MySQLRestore {
	Logf("Waiting up to %v for MySQLRestore \"%s/%s\" to be complete executing", timeout, namespace, name)
	restore := j.waitForConditionOrFail(namespace, name, timeout, "to complete executing", func(restore *v1.MySQLRestore) bool {
		phase := restore.Status.Phase
		if phase == v1.RestorePhaseComplete {
			return true
		}

		if phase == v1.RestorePhaseFailed {
			ns := restore.Namespace
			events, err := j.KubeClient.CoreV1().Events(ns).List(metav1.ListOptions{})
			if err != nil {
				Failf("Failed to list Events in %q: %v", ns, err)
			}
			for _, e := range events.Items {
				if e.InvolvedObject.Kind != restore.Kind || e.InvolvedObject.Name != restore.Name {
					continue
				}
				Logf(e.String())
			}
			Failf("MySQLRestore entered state %q", v1.RestorePhaseFailed)
		}
		return false
	})
	return restore
}
