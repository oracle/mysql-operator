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

// +build all upgrade

package e2e

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	api "github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
	"github.com/oracle/mysql-operator/pkg/constants"
	"github.com/oracle/mysql-operator/test/e2e/framework"
	e2eutil "github.com/oracle/mysql-operator/test/e2e/util"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const mysqlOperatorImageName = "wcr.io/oracle/mysql-operator"
const mysqlAgentContainerName = "mysql-agent"

func TestUpgrade(test *testing.T) {
	t := e2eutil.NewT(test)
	f := framework.Global

	oldVersion := getOperatorVersion(t, f)
	t.Logf("Old version: %s", oldVersion)
	newVersion := f.BuildVersion
	t.Logf("New version: %s", newVersion)

	// Check that we have the old operator installed and running.
	if oldVersion == newVersion {
		t.Fatalf("Error: The old version is the same as the new version")
	}

	// Create a cluster using the old version.
	testdb := e2eutil.CreateTestDB(t, "e2e-up-", int32(3), false, f.DestroyAfterFailure)
	cluster := testdb.Cluster()
	t.Logf("Created cluster: %s", cluster.Name)
	testdb.Populate()
	defer testdb.Delete()

	// Check that the cluster resources all have the old version.
	checkClusterVersion(t, f, cluster, oldVersion)

	// Upgrade to new version or the operator. Note: This should also upgrade the agent in the existing cluster.
	currentVersion := upgradeOperator(t, f, newVersion)

	// Check that we now have the new operator version installed and running.
	if currentVersion != newVersion {
		t.Fatalf("Error: The current version should now be the same as the new version")
	}

	// Check that the existing cluster has been upgraded to the new version.
	testdb = e2eutil.GetTestDB(t, cluster.Name, f.DestroyAfterFailure)
	cluster = testdb.Cluster()
	t.Logf("Got cluster: %s", cluster.Name)
	checkClusterVersion(t, f, cluster, newVersion)

	// Test the database..
	testdb.Test()

	t.Report()
}

func getOperatorVersion(t *e2eutil.T, f *framework.Framework) string {
	listOpts := metav1.ListOptions{LabelSelector: "app=mysql-operator"}
	podList, err := f.KubeClient.CoreV1().Pods(f.Namespace).List(listOpts)
	if err != nil {
		t.Fatalf("Error: Unable to retrieve operator version. Could not list pods")
	}
	for _, pod := range podList.Items {
		if pod.Status.Phase == "Running" {
			return versionFromImage(pod.Spec.Containers[0].Image)
		}
	}
	t.Fatalf("Error: Unable to retrieve operator version. No matching pods found")
	return ""
}

func checkClusterVersion(t *e2eutil.T, f *framework.Framework, cluster *api.MySQLCluster, version string) {
	t.Logf("Checking cluster has the version: %s", version)
	if cluster.Labels[constants.MySQLOperatorVersionLabel] != version {
		t.Fatalf("Error: Cluster MySQLOperatorVersionLabel was incorrect: %s != %s.", cluster.Labels[constants.MySQLOperatorVersionLabel], version)
	}

	t.Logf("Checking statefulset has the version: %s", version)
	ss, err := f.KubeClient.AppsV1beta1().StatefulSets(cluster.Namespace).Get(cluster.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Error: Error getting statefulset for cluster %s: %v", cluster.Name, err)
	}
	if ss.Labels[constants.MySQLOperatorVersionLabel] != version {
		t.Fatalf("Error: StatefulSet MySQLOperatorVersionLabel was incorrect: %s != %s.", ss.Labels[constants.MySQLOperatorVersionLabel], version)
	}

	t.Logf("Checking %s container in the statefulset spec has the version: %s", mysqlAgentContainerName, version)
	containerVersion := getContainerImageVersion(t, ss.Spec.Template.Spec, mysqlAgentContainerName)
	if containerVersion != version {
		t.Fatalf("Error: StatefulSet %s container version was incorrect: %s != %s.", mysqlAgentContainerName, containerVersion, version)
	}

	labelSelector := fmt.Sprintf("%s=%s", constants.MySQLClusterLabel, cluster.Name)
	listOpts := metav1.ListOptions{LabelSelector: labelSelector}
	podList, err := f.KubeClient.CoreV1().Pods(f.Namespace).List(listOpts)
	if err != nil {
		t.Fatalf("Error: Unable to list pods for cluster: %s", cluster.Name)
	}
	for _, pod := range podList.Items {
		t.Logf("Checking pod %s has the version: %s", pod.Name, version)
		if pod.Labels[constants.MySQLOperatorVersionLabel] != version {
			t.Fatalf("Error: Pod MySQLOperatorVersionLabel was incorrect: %s != %s.", ss.Labels[constants.MySQLOperatorVersionLabel], version)
		}

		t.Logf("Checking %s container in the pod %s has the version: %s", mysqlAgentContainerName, pod.Name, version)
		containerVersion := getContainerImageVersion(t, pod.Spec, mysqlAgentContainerName)
		if containerVersion != version {
			t.Fatalf("Error: Pod %s container version was incorrect: %s != %s.", mysqlAgentContainerName, containerVersion, version)
		}

		t.Logf("Checking %s version running in the pod %s has the version: %s", mysqlAgentContainerName, pod.Name, version)
		agentVersion := getRunningAgentVersion(t, f.Namespace, pod.Name, mysqlAgentContainerName)
		if agentVersion != version {
			t.Fatalf("Error: Agent version running in pod %s was incorrect: %s != %s.", pod.Name, containerVersion, version)
		}
	}
}

func getContainerImageVersion(t *e2eutil.T, podSpec v1.PodSpec, containerName string) string {
	for _, container := range podSpec.Containers {
		if container.Name == "mysql-agent" {
			return versionFromImage(container.Image)
		}
	}
	t.Fatalf("Error: Unable to retrieve %s container version from statefulset", containerName)
	return ""
}

func versionFromImage(image string) string {
	return strings.Split(image, ":")[1]
}

func getRunningAgentVersion(t *e2eutil.T, namespace string, podName string, containerName string) string {
	cmd := exec.Command("kubectl", "-n", namespace, "logs", podName, "-c", containerName)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Error: Unable to retrieve logs from %s container", containerName)
	}
	firstLine := strings.Split(string(output), "\n")[0]
	fields := strings.Fields(firstLine)
	version := fields[len(fields)-1]
	return version
}

func upgradeOperator(t *e2eutil.T, f *framework.Framework, version string) string {
	t.Logf("Upgrading operator to version: %s", version)
	patchJSON := fmt.Sprintf(
		`{"spec":{
			"template":{
				"spec":{
					"containers":[{
						"name":"mysql-operator-controller",
						"image":"%s:%s"}]}}}}`,
		mysqlOperatorImageName, version)
	cmd := exec.Command("kubectl", "patch", "-n", f.Namespace, "deployment/mysql-operator", "-p", patchJSON)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Error: Unable to patch operator, output: %s, err: %v", output, err)
	}
	return waitForOperatorUpgrade(t, f, version)
}

func waitForOperatorUpgrade(t *e2eutil.T, f *framework.Framework, desiredVersion string) string {
	var version string
	backoff := e2eutil.NewDefaultRetyWithDuration(5)
	backoff.Steps = 25
	err := e2eutil.Retry(backoff, func() (bool, error) {
		version = getOperatorVersion(t, f)
		t.Logf("waiting for operator upgrade. version: '%s', desired version: '%s'", version, desiredVersion)
		if version == desiredVersion {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("Error: Timeout waiting for operator to upgrade")
	}
	return version
}
