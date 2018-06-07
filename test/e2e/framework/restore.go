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

	restoreutil "github.com/oracle/mysql-operator/pkg/api/restore"
	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
	mysqlclientset "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned"
)

// RestoreTestJig is a jig to help Restore testing.
type RestoreTestJig struct {
	ID     string
	Name   string
	Labels map[string]string

	MySQLClient mysqlclientset.Interface
	KubeClient  clientset.Interface
}

// NewRestoreTestJig allocates and inits a new RestoreTestJig.
func NewRestoreTestJig(mysqlClient mysqlclientset.Interface, kubeClient clientset.Interface, name string) *RestoreTestJig {
	id := string(uuid.NewUUID())
	return &RestoreTestJig{
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

// newRestoreTemplate returns the default v1alpha1.Restore template for this jig, but
// does not actually create the Restore. The default Restore has the
// same name as the jig.
func (j *RestoreTestJig) newRestoreTemplate(namespace, clusterName, backupName string) *v1alpha1.Restore {
	return &v1alpha1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: j.Name,
			Namespace:    namespace,
			Labels:       j.Labels,
		},
		Spec: v1alpha1.RestoreSpec{
			Cluster: &corev1.LocalObjectReference{
				Name: clusterName,
			},
			Backup: &corev1.LocalObjectReference{
				Name: backupName,
			},
		},
	}
}

// CreateRestoreOrFail creates a new Restore based on the jig's
// defaults. Callers can provide a function to tweak the Restore object
// before it is created.
func (j *RestoreTestJig) CreateRestoreOrFail(namespace, clusterName, backupName string, tweak func(restore *v1alpha1.Restore)) *v1alpha1.Restore {
	restore := j.newRestoreTemplate(namespace, clusterName, backupName)
	if tweak != nil {
		tweak(restore)
	}

	name := types.NamespacedName{Namespace: namespace, Name: j.Name}
	By(fmt.Sprintf("Creating a Restore %q", name))

	result, err := j.MySQLClient.MySQLV1alpha1().Restores(namespace).Create(restore)
	if err != nil {
		Failf("Failed to create Restore %q: %v", name, err)
	}
	return result
}

// CreateAndAwaitRestoreOrFail creates a new Restore based on the
// jig's defaults, waits for it to become ready. Callers can provide a function
// to tweak the Restore object before it is created.
func (j *RestoreTestJig) CreateAndAwaitRestoreOrFail(namespace, clusterName, backupName string, tweak func(restore *v1alpha1.Restore), timeout time.Duration) *v1alpha1.Restore {
	restore := j.CreateRestoreOrFail(namespace, clusterName, backupName, tweak)
	restore = j.WaitForRestoreCompleteOrFail(namespace, restore.Name, timeout)
	return restore
}

// CreateS3AuthSecret creates a secret containing the S3 (compat.) credentials
// for storing backups.
func (j *RestoreTestJig) CreateS3AuthSecret(namespace, name string) (*corev1.Secret, error) {
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

func (j *RestoreTestJig) waitForConditionOrFail(namespace, name string, timeout time.Duration, message string, conditionFn func(*v1alpha1.Restore) bool) *v1alpha1.Restore {
	var restore *v1alpha1.Restore
	pollFunc := func() (bool, error) {
		r, err := j.MySQLClient.MySQLV1alpha1().Restores(namespace).Get(name, metav1.GetOptions{})
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
		Failf("Timed out waiting for Restore %q to %s", name, message)
	}
	return restore
}

// WaitForRestoreCompleteOrFail waits up to a given timeout for a Restore
// to enter the complete phase.
func (j *RestoreTestJig) WaitForRestoreCompleteOrFail(namespace, name string, timeout time.Duration) *v1alpha1.Restore {
	Logf("Waiting up to %v for Restore \"%s/%s\" to be complete executing", timeout, namespace, name)
	restore := j.waitForConditionOrFail(namespace, name, timeout, "to complete executing", func(restore *v1alpha1.Restore) bool {
		_, cond := restoreutil.GetRestoreCondition(&restore.Status, v1alpha1.RestoreComplete)
		if cond != nil && cond.Status == corev1.ConditionTrue {
			return true
		}
		_, cond = restoreutil.GetRestoreCondition(&restore.Status, v1alpha1.RestoreFailed)
		if cond != nil && cond.Status == corev1.ConditionTrue {
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
			Failf("Restore condition failed (%s==%s)", cond.Type, cond.Status)
		}
		return false
	})
	return restore
}
