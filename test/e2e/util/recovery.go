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
	"os"
	"os/exec"
	"strings"

	"github.com/oracle/mysql-operator/pkg/controllers/cluster/labeler"
	"github.com/oracle/mysql-operator/test/e2e/framework"

	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// testMySQLPodCrash deletes the specified pod by name and then checks it
// recovers.
func TestMySQLPodCrash(t *T, namespace string, podName string,
	kubeClient kubernetes.Interface, clusterName string, numInstances int32) {

	t.Logf("waiting for pod phase: %v", v1.PodRunning)
	pod, err := WaitForPodPhase(t, namespace, podName, v1.PodRunning, NewDefaultRetyWithDuration(20), kubeClient)
	if err != nil {
		t.Fatalf("failed to get pod: %v", err)
	} else {
		t.Logf("pod '%s' running.", pod.Name)
	}

	t.Logf("deleting the pod: %s", podName)
	deletePod(t, namespace, podName, kubeClient)
	if err != nil {
		t.Fatalf("failed to delete pod '%s': %v", pod.Name, err)
	} else {
		t.Logf("pod '%s' deleted.", pod.Name)
	}

	t.Logf("waiting for pod to be unready")
	unreadyBackoff := NewDefaultRetyWithDuration(5)
	unreadyBackoff.Steps = 25
	pod, err = WaitForPodReadyStatus(t, namespace, podName, unreadyBackoff, kubeClient, false)
	if err != nil {
		t.Fatalf("pod '%s' failed to reach 'ready' phase...", podName)
	} else {
		t.Logf("pod '%s' reached 'ready' phase...", podName)
	}

	t.Logf("waiting for pod to be ready")
	readyBackoff := NewDefaultRetyWithDuration(20)
	readyBackoff.Steps = 25
	pod, err = WaitForPodReadyStatus(t, namespace, podName, readyBackoff, kubeClient, true)
	if err != nil {
		t.Fatalf("failed to get pod: %v", err)
	} else {
		t.Logf("pod '%s' running.", pod.Name)
	}

	t.Logf("checking pod is running")
	pod, err = WaitForPodPhase(t, namespace, podName, v1.PodRunning, NewDefaultRetyWithDuration(5), kubeClient)
	if err != nil {
		t.Fatalf("failed to get pod: %v", err)
	} else {
		t.Logf("pod '%s' running.", pod.Name)
	}
}

// testMySQLContainerCrash delete the sql-agent container of specified pod by
// name and then checks it recovers.
func TestMySQLContainerCrash(t *T, namespace string, podName string, containerName string, f *framework.Framework, clusterName string, numInstances int32) {
	podHostInstanceSSHAddress, podHostInstanceSSHKeyPath := getSSHInfo(t, namespace, podName, f)

	t.Logf("waiting for pod phase: %v", v1.PodRunning)
	pod, err := WaitForPodPhase(t, namespace, podName, v1.PodRunning, NewDefaultRetyWithDuration(20), f.KubeClient)
	if err != nil {
		t.Fatalf("failed to get pod: %v", err)
	} else {
		t.Logf("pod '%s' running.", pod.Name)
	}

	t.Logf("deleting the pod container: %s %s", podName, containerName)
	deletedContainerID := deletePodContainer(t, namespace, podName, containerName, podHostInstanceSSHAddress, podHostInstanceSSHKeyPath, f.KubeClient)
	if err != nil {
		t.Fatalf("failed to delete pod '%s' container '%s': %v", pod.Name, containerName, err)
	} else {
		deleteBackoff := NewDefaultRetyWithDuration(20)
		deleteBackoff.Steps = 5
		WaitForPodContainerDeletion(t, namespace, podName, containerName, deletedContainerID, deleteBackoff, f.KubeClient)
		t.Logf("deleted pod '%s' container '%s' with ID: %s", pod.Name, containerName, deletedContainerID)
	}

	t.Logf("waiting for pod to be ready")
	readyBackoff := NewDefaultRetyWithDuration(20)
	readyBackoff.Steps = 25
	pod, err = WaitForPodReadyStatus(t, namespace, podName, readyBackoff, f.KubeClient, true)
	if err != nil {
		t.Fatalf("pod '%s' failed to reach 'ready' phase..., err: %v", podName, err)
	} else {
		t.Logf("pod '%s' reached 'ready' phase...", podName)
	}

	t.Logf("checking pod is running")
	pod, err = WaitForPodPhase(t, namespace, podName, v1.PodRunning, NewDefaultRetyWithDuration(5), f.KubeClient)
	if err != nil {
		t.Fatalf("failed to get pod: %v", err)
	} else {
		t.Logf("pod '%s' running.", pod.Name)
	}
}

func getPodNameFromInstanceName(instanceName string) string {
	return strings.Split(instanceName, ".")[0]
}

// GetPrimaryPodName returns the name of the first primary pod it finds in
// the given cluster.
func GetPrimaryPodName(t *T, namespace string, clusterName string, kubeClient kubernetes.Interface) string {
	var name string
	err := Retry(DefaultRetry, func() (bool, error) {
		pods, err := kubeClient.CoreV1().Pods(namespace).List(meta_v1.ListOptions{
			LabelSelector: labeler.PrimarySelector(clusterName).String(),
		})
		if err != nil {
			return false, err
		}
		if len(pods.Items) == 0 {
			return false, nil
		}
		name = pods.Items[0].Name
		return true, nil
	})
	if err != nil {
		t.Fatalf("failed to get a primary pod name: %v", err)
	}
	return name
}

// GetSecondaryPodName returns the name of the first secondary pod it finds in
// the given cluster.
func GetSecondaryPodName(t *T, namespace string, clusterName string, kubeClient kubernetes.Interface) string {
	var name string
	err := Retry(DefaultRetry, func() (bool, error) {
		pods, err := kubeClient.CoreV1().Pods(namespace).List(meta_v1.ListOptions{
			LabelSelector: labeler.SecondarySelector(clusterName).String(),
		})
		if err != nil {
			return false, err
		}
		if len(pods.Items) == 0 {
			return false, nil
		}
		name = pods.Items[0].Name
		return true, nil
	})
	if err != nil {
		t.Fatalf("failed to get a secondary pod name: %v", err)
	}
	return name
}

// CheckPrimaryFailover exists with an error if the primary has not changed
// from the given one.
func CheckPrimaryFailover(t *T, namespace string, clusterName string, oldPrimaryPod string, kubeClient kubernetes.Interface) {
	newPrimaryPod := GetPrimaryPodName(t, namespace, clusterName, kubeClient)
	if newPrimaryPod == oldPrimaryPod {
		t.Fatalf("failed to failover primary database from pod: %v", oldPrimaryPod)
	}
	t.Logf("primary database is now on pod: %s", newPrimaryPod)
}

func getPod(t *T, namespace string, podName string, kubeClient kubernetes.Interface) (*v1.Pod, error) {
	getOpts := meta_v1.GetOptions{}
	pod, err := kubeClient.CoreV1().Pods(namespace).Get(podName, getOpts)
	if err != nil {
		t.Fatalf("failed to list pods: %v", err)
	}
	return pod, err
}

func deletePod(t *T, namespace string, podName string, kubeClient kubernetes.Interface) {
	deleteOpts := meta_v1.DeleteOptions{}
	err := kubeClient.CoreV1().Pods(namespace).Delete(podName, &deleteOpts)
	if err != nil {
		t.Fatalf("failed to delete pod: %v", err)
	}
}

func deletePodContainer(t *T, namespace string, podName string, containerName string,
	podHostInstance string, podHostInstanceSSHKeyPath string, kubeClient kubernetes.Interface) string {

	mysqlAgentDockerID := getPodContainerDockerID(t, namespace, podName, containerName, kubeClient)
	cmd := exec.Command(
		"ssh", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no",
		"-i", podHostInstanceSSHKeyPath, podHostInstance,
		"sudo", "docker", "rm", "-f", mysqlAgentDockerID)

	t.Logf("cmd: %v", cmd.Args)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to execute delete pod container ssh cmd: %v - %v - %s", cmd.Args, err, string(output))
	}
	t.Logf("output: %s", string(output))

	return mysqlAgentDockerID
}

// WaitForPodContainerDeletion waits for the specified container to have been deleted. It does this by monitoring the
// DockerId of the container and return a success on a change ("" if it is no longer there, or "new_container_uuid" if
// a new container has been spawned).
func WaitForPodContainerDeletion(t *T, namespace string, podName string,
	containerName string, deletedContainerID string, backoff wait.Backoff, kubeClient kubernetes.Interface) string {

	var currentContainerID string
	Retry(backoff, func() (bool, error) {
		currentContainerID := getPodContainerDockerID(t, namespace, podName, containerName, kubeClient)
		isDeleted := deletedContainerID != currentContainerID
		if deletedContainerID == currentContainerID {
			t.Logf("Waiting for pod '%s' container '%s' to be deleted. Deleted containerID: '%s'. Current containerID: '%s'", podName, containerName, deletedContainerID, currentContainerID)
		} else {
			t.Logf("Deleted pod '%s' container '%s'. Deleted containerID: '%s'. Current containerID: '%s'", podName, containerName, deletedContainerID, currentContainerID)
		}
		return isDeleted, nil
	})
	return currentContainerID
}

func getPodContainerDockerID(t *T, namespace string, podName string,
	containerName string, kubeClient kubernetes.Interface) string {

	var pcdID string
	getOpts := meta_v1.GetOptions{}
	pod, err := kubeClient.CoreV1().Pods(namespace).Get(podName, getOpts)
	if err != nil {
		t.Fatalf("failed to get pod '%s': %v", podName, err)
	} else {
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.Name == containerName {
				pcdID = strings.TrimPrefix(containerStatus.ContainerID, "docker://")
				break
			}
		}
	}
	return pcdID
}

func getSSHInfo(t *T, namespace string, podName string, f *framework.Framework) (string, string) {
	instance := getPodExternalIP(t, namespace, podName, f.KubeClient)
	sshAddress := fmt.Sprintf("%s@%s", f.SSHUser, instance)
	sshKeyPath := f.SSHKeyPath
	return sshAddress, sshKeyPath
}

func getPodExternalIP(t *T, namespace string, podName string, kubeClient kubernetes.Interface) string {
	pod, err := getPod(t, namespace, podName, kubeClient)
	hostIP := pod.Status.HostIP
	if err != nil {
		t.Fatalf("failed to find external IP for pod: %s, err: %v", podName, err)
		return ""
	}
	nodeIPs := os.Getenv("NODE_IPS")
	if nodeIPs == "" {
		t.Fatalf("failed to find NODE_IPS in environment")
	}
	for _, entry := range strings.Split(nodeIPs, ",") {
		pair := strings.Split(entry, "=")
		internalIP := pair[0]
		externalIP := pair[1]
		if hostIP == internalIP {
			return externalIP
		}
	}
	t.Fatalf("failed to find external IP for pod: %s", podName)
	return ""
}
