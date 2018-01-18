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

package util

import (
	"fmt"
	"os/exec"
	"time"

	api "github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
	mysqlop "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// DefaultRetry is the default backoff for e2e tests.
var DefaultRetry = wait.Backoff{
	Steps:    50,
	Duration: 10 * time.Second,
	Factor:   1.0,
	Jitter:   0.1,
}

// NewDefaultRetyWithDuration creates a customized backoff for e2e tests.
func NewDefaultRetyWithDuration(seconds time.Duration) wait.Backoff {
	return wait.Backoff{
		Steps:    3,
		Duration: seconds * time.Second,
		Factor:   1.0,
		Jitter:   0.1,
	}
}

// Retry executes the provided function repeatedly, retrying until the function
// returns done = true, errors, or exceeds the given timeout.
func Retry(backoff wait.Backoff, fn wait.ConditionFunc) error {
	var lastErr error
	err := wait.ExponentialBackoff(backoff, func() (bool, error) {
		done, err := fn()
		if err != nil {
			lastErr = err
		}
		return done, err
	})
	if err == wait.ErrWaitTimeout {
		if lastErr != nil {
			err = lastErr
		}
	}
	return err
}

// WaitForClusterPhase retries until a cluster reaches a given phase or a
// timeout is reached.
func WaitForClusterPhase(
	t *T,
	cluster *api.MySQLCluster,
	phase api.MySQLClusterPhase,
	backoff wait.Backoff,
	mySQLOpClient mysqlop.Interface,
) (*api.MySQLCluster, error) {
	return WaitForNamedClusterPhase(t, cluster.Namespace, cluster.Name, phase, backoff, mySQLOpClient)
}

// WaitForNamedClusterPhase retries until a cluster reaches a given phase or a
// timeout is reached.
func WaitForNamedClusterPhase(
	t *T,
	clusterNameSpace string,
	clusterName string,
	phase api.MySQLClusterPhase,
	backoff wait.Backoff,
	mySQLOpClient mysqlop.Interface,
) (*api.MySQLCluster, error) {
	var cl *api.MySQLCluster
	var err error
	err = Retry(backoff, func() (bool, error) {
		cl, err = mySQLOpClient.MysqlV1().MySQLClusters(clusterNameSpace).Get(clusterName, metav1.GetOptions{})
		if err != nil {
			t.Logf("waiting for cluster '%s' to reach phase: '%v', error: '%v'...", clusterName, phase, err)
			return false, err
		}
		t.Logf("waiting for cluster '%s' to reach phase: '%v', currently: '%v'...", clusterName, phase, cl.Status.Phase)
		return cl.Status.Phase == phase, err
	})
	if err != nil {
		return nil, err
	}
	return cl, nil
}

// WaitForBackupPhase retries until a backup completes or timeout is reached.
func WaitForBackupPhase(
	t *T,
	backup *api.MySQLBackup,
	phase api.BackupPhase,
	backoff wait.Backoff,
	mySQLOpClient mysqlop.Interface,
) (*api.MySQLBackup, error) {
	var latest *api.MySQLBackup
	var err error
	err = Retry(backoff, func() (bool, error) {
		latest, err = mySQLOpClient.MysqlV1().MySQLBackups(backup.Namespace).Get(backup.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		t.Logf("waiting for backup %s to reach phase: '%s', currently: '%s'...", backup.Name, phase, latest.Status.Phase)
		if latest.Status.Phase == api.BackupPhaseFailed {
			return true, fmt.Errorf("Backup '%s' phase reached %s.", backup.Name, api.BackupPhaseFailed)
		}
		return latest.Status.Phase == phase, err
	})
	if err != nil {
		return nil, err
	}
	return latest, nil
}

// WaitForRestorePhase retries until a restore completes or timeout is reached.
func WaitForRestorePhase(
	t *T,
	restore *api.MySQLRestore,
	phase api.RestorePhase,
	backoff wait.Backoff,
	mySQLOpClient mysqlop.Interface,
) (*api.MySQLRestore, error) {
	var latest *api.MySQLRestore
	var err error
	err = Retry(backoff, func() (bool, error) {
		latest, err = mySQLOpClient.MysqlV1().MySQLRestores(restore.Namespace).Get(restore.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		t.Logf("waiting for restore %s to reach phase: '%s', currently: '%s'...", restore.Name, phase, latest.Status.Phase)
		if latest.Status.Phase == api.RestorePhaseFailed {
			return true, fmt.Errorf("Restore '%s' phase reached %s.", restore.Name, api.RestorePhaseFailed)
		}
		return latest.Status.Phase == phase, err
	})
	if err != nil {
		return nil, err
	}
	return latest, nil
}

// WaitForPodPhase retries until the pod phase is reached or timeout is reached.
func WaitForPodPhase(
	t *T,
	namespace string,
	podName string,
	podPhase v1.PodPhase,
	backoff wait.Backoff,
	kubeClient kubernetes.Interface,
) (*v1.Pod, error) {
	var latest *v1.Pod
	var err error
	err = Retry(backoff, func() (bool, error) {
		latest, err = kubeClient.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
		t.Logf("Waiting for pod '%s' to reach phase: '%v', currently: '%v'...", podName, podPhase, latest.Status.Phase)
		if latest.Status.Phase == v1.PodFailed {
			return true, fmt.Errorf("Pod '%s' phase reached %s", podName, v1.PodFailed)
		}
		return latest.Status.Phase == podPhase, err
	})
	if err != nil {
		return nil, err
	}
	return latest, nil
}

// WaitForPodReadyStatus retries until all containers in a pod are in the desired ready state or timeout is reached.
func WaitForPodReadyStatus(
	t *T,
	namespace string,
	podName string,
	backoff wait.Backoff,
	kubeClient kubernetes.Interface,
	desiredReady bool,
) (*v1.Pod, error) {
	var latest *v1.Pod
	var err error
	err = Retry(backoff, func() (bool, error) {
		latest, err = kubeClient.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
		allReady := true
		for _, latestContainerStatus := range latest.Status.ContainerStatuses {
			if !latestContainerStatus.Ready {
				allReady = false
			}
		}
		t.Logf("Waiting for pod '%s' to reach ready: '%v', currently: '%v'...", podName, desiredReady, allReady)
		if allReady == desiredReady {
			return true, err
		}
		return false, err
	})
	if err != nil {
		return nil, err
	}
	return latest, nil
}

func getClusterStatusJSON(
	t *T,
	podName string,
	containerName string,
	namespace string,
) (string, error) {
	cmd := exec.Command("kubectl", "-n", namespace, "exec", podName, "-c", containerName,
		"--", "curl", "localhost:8080/cluster-status")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute command:%v: %v", cmd.Args, err)
	}
	return string(output), nil
}
