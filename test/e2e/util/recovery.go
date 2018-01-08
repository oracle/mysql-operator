package util

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/oracle/mysql-operator/pkg/controllers/cluster/labeler"
	fw "github.com/oracle/mysql-operator/test/e2e/framework"
)

// testMySQLPodCrash deletes the specified pod by name and then checks it
// recovers.
func TestMySQLPodCrash(t *testing.T, namespace string, podName string,
	kubeClient kubernetes.Interface, clusterName string, numInstances int32) {

	fmt.Printf("waiting for pod phase: %v\n", v1.PodRunning)
	pod, err := WaitForPodPhase(t, namespace, podName, v1.PodRunning, NewDefaultRetyWithDuration(20), kubeClient)
	if err != nil {
		t.Fatalf("failed to get pod: %v\n", err)
	} else {
		fmt.Printf("pod '%s' running.\n", pod.Name)
	}

	fmt.Printf("deleting the pod: %s\n", podName)
	deletePod(t, namespace, podName, kubeClient)
	if err != nil {
		t.Fatalf("failed to delete pod '%s': %v\n", pod.Name, err)
	} else {
		fmt.Printf("pod '%s' deleted.\n", pod.Name)
	}

	fmt.Printf("waiting for pod to be unready\n")
	unreadyBackoff := NewDefaultRetyWithDuration(5)
	unreadyBackoff.Steps = 25
	pod, err = WaitForPodReadyStatus(t, namespace, podName, unreadyBackoff, kubeClient, false)
	if err != nil {
		t.Fatalf("pod '%s' failed to reach 'ready' phase...\n", podName)
	} else {
		fmt.Printf("pod '%s' reached 'ready' phase...\n", podName)
	}

	fmt.Printf("waiting for pod to be ready\n")
	readyBackoff := NewDefaultRetyWithDuration(20)
	readyBackoff.Steps = 25
	pod, err = WaitForPodReadyStatus(t, namespace, podName, readyBackoff, kubeClient, true)
	if err != nil {
		t.Fatalf("failed to get pod: %v\n", err)
	} else {
		fmt.Printf("pod '%s' running.\n", pod.Name)
	}

	fmt.Printf("checking pod is running\n")
	pod, err = WaitForPodPhase(t, namespace, podName, v1.PodRunning, NewDefaultRetyWithDuration(5), kubeClient)
	if err != nil {
		t.Fatalf("failed to get pod: %v\n", err)
	} else {
		fmt.Printf("pod '%s' running.\n", pod.Name)
	}
}

// testMySQLContainerCrash delete the sql-agent container of specified pod by
// name and then checks it recovers.
func TestMySQLContainerCrash(t *testing.T, namespace string, podName string, containerName string, f *fw.Framework, clusterName string, numInstances int32) {

	podHostInstanceSSHAddress, podHostInstanceSSHKeyPath := getSSHInfo(t, namespace, podName, f)

	fmt.Printf("waiting for pod phase: %v\n", v1.PodRunning)
	pod, err := WaitForPodPhase(t, namespace, podName, v1.PodRunning, NewDefaultRetyWithDuration(20), f.KubeClient)
	if err != nil {
		t.Fatalf("failed to get pod: %v\n", err)
	} else {
		fmt.Printf("pod '%s' running.\n", pod.Name)
	}

	fmt.Printf("deleting the pod container: %s %s\n", podName, containerName)
	deletedContainerID := deletePodContainer(t, namespace, podName, containerName, podHostInstanceSSHAddress, podHostInstanceSSHKeyPath, f.KubeClient)
	if err != nil {
		t.Fatalf("failed to delete pod '%s' container '%s': %v\n", pod.Name, containerName, err)
	} else {
		deleteBackoff := NewDefaultRetyWithDuration(20)
		deleteBackoff.Steps = 5
		WaitForPodContainerDeletion(t, namespace, podName, containerName, deletedContainerID, deleteBackoff, f.KubeClient)
		fmt.Printf("deleted pod '%s' container '%s' with ID: %s\n", pod.Name, containerName, deletedContainerID)
	}

	fmt.Printf("waiting for pod to be ready\n")
	readyBackoff := NewDefaultRetyWithDuration(20)
	readyBackoff.Steps = 25
	pod, err = WaitForPodReadyStatus(t, namespace, podName, readyBackoff, f.KubeClient, true)
	if err != nil {
		t.Fatalf("pod '%s' failed to reach 'ready' phase..., err: %v\n", podName, err)
	} else {
		fmt.Printf("pod '%s' reached 'ready' phase...\n", podName)
	}

	fmt.Printf("checking pod is running\n")
	pod, err = WaitForPodPhase(t, namespace, podName, v1.PodRunning, NewDefaultRetyWithDuration(5), f.KubeClient)
	if err != nil {
		t.Fatalf("failed to get pod: %v\n", err)
	} else {
		fmt.Printf("pod '%s' running.\n", pod.Name)
	}
}

func getPodNameFromInstanceName(instanceName string) string {
	return strings.Split(instanceName, ".")[0]
}

// GetPrimaryPodName returns the name of the first primary pod it finds in
// the given cluster.
func GetPrimaryPodName(t *testing.T, namespace string, clusterName string, kubeClient kubernetes.Interface) string {
	pods, err := kubeClient.CoreV1().Pods(namespace).List(meta_v1.ListOptions{
		LabelSelector: labeler.PrimarySelector(clusterName).String(),
	})
	if err != nil {
		t.Fatalf("failed to get a primary pod name: err: %v", err)
	}
	for _, pod := range pods.Items {
		return pod.Name
	}
	t.Fatalf("failed to get a primary pod name")
	return ""
}

// GetSecondaryPodName returns the name of the first secondary pod it finds in
// the given cluster.
func GetSecondaryPodName(t *testing.T, namespace string, clusterName string, kubeClient kubernetes.Interface) string {
	pods, err := kubeClient.CoreV1().Pods(namespace).List(meta_v1.ListOptions{
		LabelSelector: labeler.SecondarySelector(clusterName).String(),
	})
	if err != nil {
		t.Fatalf("failed to get a secondary pod name: err: %v", err)
	}
	for _, pod := range pods.Items {
		return pod.Name
	}
	t.Fatalf("failed to get a secondary pod name")
	return ""
}

// CheckPrimaryFailover exists with an error if the primary has not changed
// from the given one.
func CheckPrimaryFailover(t *testing.T, namespace string, clusterName string, oldPrimaryPod string, kubeClient kubernetes.Interface) {
	newPrimaryPod := GetPrimaryPodName(t, namespace, clusterName, kubeClient)
	if newPrimaryPod == oldPrimaryPod {
		t.Fatalf("failed to failover primary database from pod: %v\n", oldPrimaryPod)
	}
	fmt.Printf("primary database is now on pod: %s\n", newPrimaryPod)
}

func getPod(t *testing.T, namespace string, podName string, kubeClient kubernetes.Interface) (*v1.Pod, error) {
	getOpts := meta_v1.GetOptions{}
	pod, err := kubeClient.CoreV1().Pods(namespace).Get(podName, getOpts)
	if err != nil {
		t.Fatalf("failed to list pods: %v", err)
	}
	return pod, err
}

func deletePod(t *testing.T, namespace string, podName string, kubeClient kubernetes.Interface) {
	deleteOpts := meta_v1.DeleteOptions{}
	err := kubeClient.CoreV1().Pods(namespace).Delete(podName, &deleteOpts)
	if err != nil {
		t.Fatalf("failed to delete pod: %v", err)
	}
}

func deletePodContainer(t *testing.T, namespace string, podName string, containerName string,
	podHostInstance string, podHostInstanceSSHKeyPath string, kubeClient kubernetes.Interface) string {

	mysqlAgentDockerID := getPodContainerDockerID(t, namespace, podName, containerName, kubeClient)
	cmd := exec.Command(
		"ssh", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no",
		"-i", podHostInstanceSSHKeyPath, podHostInstance,
		"sudo", "docker", "rm", "-f", mysqlAgentDockerID)

	fmt.Printf("cmd: %v\n", cmd.Args)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to execute delete pod container ssh cmd: %v - %v - %s", cmd.Args, err, string(output))
	}
	fmt.Printf("output: %s\n", string(output))

	return mysqlAgentDockerID
}

// WaitForPodContainerDeletion waits for the specified container to have been deleted. It does this by monitoring the
// DockerId of the container and return a success on a change ("" if it is no longer there, or "new_container_uuid" if
// a new container has been spawned).
func WaitForPodContainerDeletion(
	t *testing.T,
	namespace string,
	podName string,
	containerName string,
	deletedContainerID string,
	backoff wait.Backoff,
	kubeClient kubernetes.Interface,
) string {
	var currentContainerID string
	Retry(backoff, func() (bool, error) {
		currentContainerID := getPodContainerDockerID(t, namespace, podName, containerName, kubeClient)
		isDeleted := deletedContainerID != currentContainerID
		if deletedContainerID == currentContainerID {
			fmt.Printf("Waiting for pod '%s' container '%s' to be deleted. Deleted containerID: '%s'. Current containerID: '%s'\n", podName, containerName, deletedContainerID, currentContainerID)
		} else {
			fmt.Printf("Deleted pod '%s' container '%s'. Deleted containerID: '%s'. Current containerID: '%s'\n", podName, containerName, deletedContainerID, currentContainerID)
		}
		return isDeleted, nil
	})
	return currentContainerID
}

func getPodContainerDockerID(t *testing.T, namespace string, podName string,
	containerName string, kubeClient kubernetes.Interface) string {

	var pcdID string
	getOpts := meta_v1.GetOptions{}
	pod, err := kubeClient.CoreV1().Pods(namespace).Get(podName, getOpts)
	if err != nil {
		t.Fatalf("failed to get pod '%s': %v\n", podName, err)
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

func getSSHInfo(t *testing.T, namespace string, podName string, f *fw.Framework) (string, string) {
	instance := getPodExternalIP(t, namespace, podName, f.KubeClient)
	sshAddress := fmt.Sprintf("%s@%s", f.SSHUser, instance)
	sshKeyPath := f.SSHKeyPath
	return sshAddress, sshKeyPath
}

func getPodExternalIP(t *testing.T, namespace string, podName string, kubeClient kubernetes.Interface) string {
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
