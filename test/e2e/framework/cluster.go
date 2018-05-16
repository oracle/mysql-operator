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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	wait "k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"

	. "github.com/onsi/ginkgo"
	"github.com/pkg/errors"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
	"github.com/oracle/mysql-operator/pkg/controllers/cluster/labeler"
	mysqlclientset "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned"
	"github.com/oracle/mysql-operator/pkg/resources/secrets"
)

// TestDBName is the name of database to use when executing test SQL queries.
const TestDBName = "testdb"

// MySQLClusterTestJig is a jig to help MySQLCluster testing.
type MySQLClusterTestJig struct {
	ID     string
	Name   string
	Labels map[string]string

	MySQLClient mysqlclientset.Interface
	KubeClient  clientset.Interface
}

// NewMySQLClusterTestJig allocates and inits a new MySQLClusterTestJig.
func NewMySQLClusterTestJig(mysqlClient mysqlclientset.Interface, kubeClient clientset.Interface, name string) *MySQLClusterTestJig {
	id := string(uuid.NewUUID())
	return &MySQLClusterTestJig{
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

// newMySQLClusterTemplate returns the default v1.MySQLCluster template for this jig, but
// does not actually create the MySQLCluster.  The default MySQLCluster has the same name
// as the jig and has the given number of replicas.
func (j *MySQLClusterTestJig) newMySQLClusterTemplate(namespace string, replicas int32) *v1.MySQLCluster {
	return &v1.MySQLCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      j.Name,
			Labels:    j.Labels,
		},
		Spec: v1.MySQLClusterSpec{
			Replicas: replicas,
		},
	}
}

// CreateMySQLClusterOrFail creates a new MySQLCluster based on the jig's
// defaults. Callers can provide a function to tweak the MySQLCluster object
// before it is created.
func (j *MySQLClusterTestJig) CreateMySQLClusterOrFail(namespace string, replicas int32, tweak func(cluster *v1.MySQLCluster)) *v1.MySQLCluster {
	cluster := j.newMySQLClusterTemplate(namespace, replicas)
	if tweak != nil {
		tweak(cluster)
	}

	name := types.NamespacedName{Namespace: namespace, Name: j.Name}
	By(fmt.Sprintf("Creating a MySQLCluster %q with .spec.replicas=%d", name, replicas))

	result, err := j.MySQLClient.MysqlV1().MySQLClusters(namespace).Create(cluster)
	if err != nil {
		Failf("Failed to create MySQLCluster %q: %v", name, err)
	}
	return result
}

// CreateAndAwaitMySQLClusterOrFail creates a new MySQLCluster based on the
// jig's defaults, waits for it to become ready, and then sanity checks it and
// its dependant resources. Callers can provide a function to tweak the
// MySQLCluster object before it is created.
func (j *MySQLClusterTestJig) CreateAndAwaitMySQLClusterOrFail(namespace string, replicas int32, tweak func(cluster *v1.MySQLCluster), timeout time.Duration) *v1.MySQLCluster {
	cluster := j.CreateMySQLClusterOrFail(namespace, replicas, tweak)
	cluster = j.WaitForClusterReadyOrFail(namespace, cluster.Name, timeout)
	j.SanityCheckMySQLCluster(cluster)
	return cluster
}

func (j *MySQLClusterTestJig) waitForConditionOrFail(namespace, name string, timeout time.Duration, message string, conditionFn func(*v1.MySQLCluster) bool) *v1.MySQLCluster {
	var cluster *v1.MySQLCluster
	pollFunc := func() (bool, error) {
		c, err := j.MySQLClient.MysqlV1().MySQLClusters(namespace).Get(name, metav1.GetOptions{})
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
		Failf("Timed out waiting for MySQLCluster %q to %s", name, message)
	}
	return cluster
}

// WaitForClusterReadyOrFail waits up to a given timeout for a cluster to be in
// the running phase.
func (j *MySQLClusterTestJig) WaitForClusterReadyOrFail(namespace, name string, timeout time.Duration) *v1.MySQLCluster {
	Logf("Waiting up to %v for MySQLCluster \"%s/%s\" to be ready", timeout, namespace, name)
	cluster := j.waitForConditionOrFail(namespace, name, timeout, "have all nodes ready", func(cluster *v1.MySQLCluster) bool {
		if cluster.Status.Phase == v1.MySQLClusterRunning {
			return true
		}
		return false
	})
	return cluster
}

// SanityCheckMySQLCluster checks basic properties of a given MySQLCluster match
// our expectations.
func (j *MySQLClusterTestJig) SanityCheckMySQLCluster(cluster *v1.MySQLCluster) {
	name := types.NamespacedName{Namespace: cluster.Namespace, Name: cluster.Name}
	ss, err := j.KubeClient.AppsV1beta1().StatefulSets(cluster.Namespace).Get(cluster.Name, metav1.GetOptions{})
	if err != nil {
		Failf("Failed to get StatefulSet %[1]q for MySQLCluster %[1]q: %[2]v", name, err)
	}

	if ss.Status.ReadyReplicas != cluster.Spec.Replicas {
		Failf("StatefulSet %q has %d ready replica(s), want %d", name, ss.Status.ReadyReplicas, cluster.Spec.Replicas)
	}

	// Do we have a service?
	_, err = j.KubeClient.CoreV1().Services(cluster.Namespace).Get(cluster.Name, metav1.GetOptions{})
	if err != nil {
		Failf("Failed to get Servce %[1]q for MySQLCluster %[1]q: %v", name, err)
	}

	// Do we have a root password secret?
	secretName := types.NamespacedName{Namespace: cluster.Namespace, Name: secrets.GetRootPasswordSecretName(cluster)}
	_, err = j.KubeClient.CoreV1().Secrets(cluster.Namespace).Get(secretName.Name, metav1.GetOptions{})
	if err != nil {
		Failf("Error root password secret %q for cluster %q: %v", secretName, name, err)
	}
}

// ExecuteSQLOrDie executes the given SQL statement(s) on a specified MySQLCluster
// member via kubectl exec.
func ExecuteSQL(cluster *v1.MySQLCluster, member, sql string) (string, error) {
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
func ReadSQLTest(cluster *v1.MySQLCluster, member string) (string, error) {
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
func WriteSQLTest(cluster *v1.MySQLCluster, member string) (string, error) {
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
