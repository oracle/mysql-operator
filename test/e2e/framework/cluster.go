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
	"strings"
	"time"

	apps "k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	wait "k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"

	. "github.com/onsi/ginkgo"
	"github.com/pkg/errors"

	clusterutil "github.com/oracle/mysql-operator/pkg/api/cluster"
	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
	"github.com/oracle/mysql-operator/pkg/controllers/cluster/labeler"
	mysqlclientset "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned"
	"github.com/oracle/mysql-operator/pkg/resources/secrets"
	"github.com/oracle/mysql-operator/pkg/resources/statefulsets"
)

// TestDBName is the name of database to use when executing test SQL queries.
const TestDBName = "testdb"

// ClusterTestJig is a jig to help Cluster testing.
type ClusterTestJig struct {
	ID     string
	Name   string
	Labels map[string]string

	MySQLClient mysqlclientset.Interface
	KubeClient  clientset.Interface
}

// NewClusterTestJig allocates and inits a new ClusterTestJig.
func NewClusterTestJig(mysqlClient mysqlclientset.Interface, kubeClient clientset.Interface, name string) *ClusterTestJig {
	id := string(uuid.NewUUID())
	return &ClusterTestJig{
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

// newClusterTemplate returns the default v1.Cluster template for this jig, but
// does not actually create the Cluster.  The default Cluster has the same name
// as the jig and has the given number of members.
func (j *ClusterTestJig) newClusterTemplate(namespace string, members int32) *v1alpha1.Cluster {
	return &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      j.Name,
			Labels:    j.Labels,
		},
		Spec: v1alpha1.ClusterSpec{
			Members: members,
		},
	}
}

// CreateClusterOrFail creates a new Cluster based on the jig's
// defaults. Callers can provide a function to tweak the Cluster object
// before it is created.
func (j *ClusterTestJig) CreateClusterOrFail(namespace string, members int32, tweak func(cluster *v1alpha1.Cluster)) *v1alpha1.Cluster {
	cluster := j.newClusterTemplate(namespace, members)
	if tweak != nil {
		tweak(cluster)
	}

	name := types.NamespacedName{Namespace: namespace, Name: j.Name}
	By(fmt.Sprintf("Creating a Cluster %q with .spec.members=%d", name, members))

	result, err := j.MySQLClient.MySQLV1alpha1().Clusters(namespace).Create(cluster)
	if err != nil {
		Failf("Failed to create Cluster %q: %v", name, err)
	}
	return result
}

// CreateAndAwaitClusterOrFail creates a new Cluster based on the
// jig's defaults, waits for it to become ready, and then sanity checks it and
// its dependant resources. Callers can provide a function to tweak the
// Cluster object before it is created.
func (j *ClusterTestJig) CreateAndAwaitClusterOrFail(namespace string, members int32, tweak func(cluster *v1alpha1.Cluster), timeout time.Duration) *v1alpha1.Cluster {
	cluster := j.CreateClusterOrFail(namespace, members, tweak)
	cluster = j.WaitForClusterReadyOrFail(namespace, cluster.Name, timeout)
	j.SanityCheckCluster(cluster)
	return cluster
}

func (j *ClusterTestJig) WaitForConditionOrFail(namespace, name string, timeout time.Duration, message string, conditionFn func(*v1alpha1.Cluster) bool) *v1alpha1.Cluster {
	var cluster *v1alpha1.Cluster
	pollFunc := func() (bool, error) {
		c, err := j.MySQLClient.MySQLV1alpha1().Clusters(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if conditionFn(c) {
			cluster = c
			return true, nil
		}
		return false, nil
	}
	if err := wait.PollImmediate(Poll, timeout, pollFunc); err != nil {
		Failf("Timed out waiting for Cluster %q to %s", name, message)
	}
	return cluster
}

// WaitForClusterReadyOrFail waits up to a given timeout for a cluster to be in
// the running phase.
func (j *ClusterTestJig) WaitForClusterReadyOrFail(namespace, name string, timeout time.Duration) *v1alpha1.Cluster {
	Logf("Waiting up to %v for Cluster \"%s/%s\" to be ready", timeout, namespace, name)
	cluster := j.WaitForConditionOrFail(namespace, name, timeout, "have all nodes ready", func(cluster *v1alpha1.Cluster) bool {
		return clusterutil.IsClusterReady(cluster)
	})
	return cluster
}

// WaitForClusterUpgradedOrFail waits for a MySQL cluster to be upgraded to the
// given version or fails.
func (j *ClusterTestJig) WaitForClusterUpgradedOrFail(namespace, name, version string, timeout time.Duration) *v1alpha1.Cluster {
	Logf("Waiting up to %v for Cluster \"%s/%s\" to be upgraded", timeout, namespace, name)

	cluster := j.WaitForConditionOrFail(namespace, name, timeout, "be upgraded ", func(cluster *v1alpha1.Cluster) bool {
		set, err := j.KubeClient.AppsV1beta1().StatefulSets(cluster.Namespace).Get(cluster.Name, metav1.GetOptions{})
		if err != nil {
			Failf("Failed to get StatefulSet %[1]q for Cluster %[1]q: %[2]v", name, err)
		}

		set = j.waitForSSRollingUpdate(set)

		var actualVersion string
		{
			var found bool
			var index int
			for i, c := range set.Spec.Template.Spec.Containers {
				if c.Name == statefulsets.MySQLServerName {
					index = i
					found = true
					break
				}
			}

			if !found {
				Failf("no %q container found for StatefulSet %q", statefulsets.MySQLServerName, set.Name)
			}
			image := set.Spec.Template.Spec.Containers[index].Image
			parts := strings.Split(image, ":")
			if len(parts) < 2 {
				Failf("invalid image %q for StatefulSet %q", image, set.Name)
			}
			actualVersion = parts[len(parts)-1]
		}

		return actualVersion == version
	})
	return cluster
}

// waitForSSState periodically polls for the ss and its pods until the until function returns either true or an error
func (j *ClusterTestJig) waitForSSState(ss *apps.StatefulSet, until func(*apps.StatefulSet, *v1.PodList) (bool, error)) {
	pollErr := wait.PollImmediate(Poll, DefaultTimeout,
		func() (bool, error) {
			ssGet, err := j.KubeClient.AppsV1beta1().StatefulSets(ss.Namespace).Get(ss.Name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}

			selector, err := metav1.LabelSelectorAsSelector(ss.Spec.Selector)
			ExpectNoError(err)
			podList, err := j.KubeClient.CoreV1().Pods(ss.Namespace).List(metav1.ListOptions{LabelSelector: selector.String()})
			ExpectNoError(err)

			return until(ssGet, podList)
		})
	if pollErr != nil {
		Failf("Failed waiting for state update: %v", pollErr)
	}
}

// waitForRollingUpdate waits for all Pods in set to exist and have the correct revision and for the RollingUpdate to
// complete. set must have a RollingUpdateStatefulSetStrategyType.
func (j *ClusterTestJig) waitForSSRollingUpdate(set *apps.StatefulSet) *apps.StatefulSet {
	var pods *v1.PodList
	if set.Spec.UpdateStrategy.Type != apps.RollingUpdateStatefulSetStrategyType {
		Failf("StatefulSet %s/%s attempt to wait for rolling update with updateStrategy %s",
			set.Namespace,
			set.Name,
			set.Spec.UpdateStrategy.Type)
	}
	Logf("Waiting for StatefulSet %s/%s to complete update", set.Namespace, set.Name)
	j.waitForSSState(set, func(set2 *apps.StatefulSet, pods2 *v1.PodList) (bool, error) {
		set = set2
		pods = pods2
		if len(pods.Items) < int(*set.Spec.Replicas) {
			return false, nil
		}
		if set.Status.UpdateRevision != set.Status.CurrentRevision {
			return false, nil
		}
		return true, nil
	})
	return set
}

// SanityCheckCluster checks basic properties of a given Cluster match
// our expectations.
func (j *ClusterTestJig) SanityCheckCluster(cluster *v1alpha1.Cluster) {
	name := types.NamespacedName{Namespace: cluster.Namespace, Name: cluster.Name}
	ss, err := j.KubeClient.AppsV1beta1().StatefulSets(cluster.Namespace).Get(cluster.Name, metav1.GetOptions{})
	if err != nil {
		Failf("Failed to get StatefulSet %[1]q for Cluster %[1]q: %[2]v", name, err)
	}

	if ss.Status.ReadyReplicas != cluster.Spec.Members {
		Failf("StatefulSet %q has %d ready replica(s), want %d", name, ss.Status.ReadyReplicas, cluster.Spec.Members)
	}

	// Do we have a service?
	_, err = j.KubeClient.CoreV1().Services(cluster.Namespace).Get(cluster.Name, metav1.GetOptions{})
	if err != nil {
		Failf("Failed to get Servce %[1]q for Cluster %[1]q: %v", name, err)
	}

	// Do we have a root password secret?
	secretName := types.NamespacedName{Namespace: cluster.Namespace, Name: secrets.GetRootPasswordSecretName(cluster)}
	_, err = j.KubeClient.CoreV1().Secrets(cluster.Namespace).Get(secretName.Name, metav1.GetOptions{})
	if err != nil {
		Failf("Error root password secret %q for cluster %q: %v", secretName, name, err)
	}
}

// ExecuteSQL executes the given SQL statement(s) on a specified Cluster member
// via kubectl exec.
func ExecuteSQL(cluster *v1alpha1.Cluster, member, sql string) (string, error) {
	cmd := fmt.Sprintf("mysql -h %s.%s -u root -p$MYSQL_ROOT_PASSWORD -e '%s'", member, cluster.Name, sql)
	return RunKubectl(fmt.Sprintf("--namespace=%v", cluster.Namespace), "exec", member,
		"-c", "mysql",
		"--", "/bin/sh", "-c", cmd)
}

func lastLine(out string) string {
	outLines := strings.Split(strings.Trim(out, "\n"), "\n")
	return outLines[len(outLines)-1]
}

// ReadSQLTest SELECTs v from testdb.foo where k=foo.
func ReadSQLTest(cluster *v1alpha1.Cluster, member string) (string, error) {
	By("SELECT v FROM foo WHERE k=\"foo\"")
	output, err := ExecuteSQL(cluster, member, strings.Join([]string{
		fmt.Sprintf("use %s;", TestDBName),
		`SELECT v FROM foo WHERE k="foo";`,
	}, " "))
	if err != nil {
		return "", errors.Wrap(err, "executing SQL")
	}

	return lastLine(output), nil
}

// WriteSQLTest creates a test table, inserts a row, and writes a uuid into it.
// It returns the generated UUID.
func WriteSQLTest(cluster *v1alpha1.Cluster, member string) (string, error) {
	By("Creating a database and table, writing to that table, and writing a uuid")
	id := uuid.NewUUID()
	if _, err := ExecuteSQL(cluster, member, strings.Join([]string{
		fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s;", TestDBName),
		fmt.Sprintf("use %s;", TestDBName),
		"CREATE TABLE IF NOT EXISTS foo (k varchar(20) NOT NULL, v varchar(36), PRIMARY KEY (k));",
		fmt.Sprintf("INSERT INTO foo (k, v) VALUES (\"foo\", \"%[1]s\") ON DUPLICATE KEY UPDATE k=\"foo\", v=\"%[1]s\";", id),
	}, " ")); err != nil {
		return "", errors.Wrap(err, "executing SQL")
	}
	return string(id), nil
}

func getReadyClusterMemberMatchingSelector(cs clientset.Interface, namespace string, sel labels.Selector) string {
	Logf("Waiting up to %v for a Pod to match selector %q", DefaultTimeout, sel)

	var name string
	if err := wait.PollImmediate(Poll, DefaultTimeout, func() (bool, error) {
		pods, err := cs.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: sel.String()})
		if err != nil {
			return false, err
		}
		for _, pod := range pods.Items {
			if IsPodReady(&pod) {
				name = pod.Name
				return true, nil
			}
		}
		return false, nil
	}); err != nil {
		Failf("Failed to find a Pod matching %q after %v: %v", sel, DefaultTimeout, err)
	}
	return name
}

// AwaitPodReadyOrDie polls the specified Pod until it is ready of a timeout
func AwaitPodReadyOrDie(cs clientset.Interface, ns, name string, timeout time.Duration) {
	nsName := types.NamespacedName{Namespace: ns, Name: name}
	Logf("Waiting up to %v for a Pod %q to be ready", timeout, nsName)

	if err := wait.PollImmediate(Poll, timeout, func() (bool, error) {
		pod, err := cs.CoreV1().Pods(ns).Get(name, metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return false, err
		}
		if IsPodReady(pod) {
			return true, nil
		}
		return false, nil
	}); err != nil {
		Failf("Pod %q did not become ready after %v: %v", nsName, timeout, err)
	}
}

// GetReadyPrimaryPodName returns the name of the first ready primary Pod it finds in
// the given cluster.
func GetReadyPrimaryPodName(cs clientset.Interface, namespace, clusterName string) string {
	return getReadyClusterMemberMatchingSelector(cs, namespace, labeler.PrimarySelector(clusterName))
}

// GetReadySecondaryPodName returns the name of the first ready secondary pod it
// finds in the given cluster.
func GetReadySecondaryPodName(cs clientset.Interface, namespace, clusterName string) string {
	return getReadyClusterMemberMatchingSelector(cs, namespace, labeler.SecondarySelector(clusterName))
}
