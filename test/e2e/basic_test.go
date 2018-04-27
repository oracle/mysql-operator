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

// +build all default

package e2e

import (
	"testing"

	"github.com/oracle/mysql-operator/pkg/constants"
	"github.com/oracle/mysql-operator/pkg/resources/secrets"
	"github.com/oracle/mysql-operator/test/e2e/framework"
	e2eutil "github.com/oracle/mysql-operator/test/e2e/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateCluster(test *testing.T) {
	t := e2eutil.NewT(test)
	f := framework.Global
	replicas := int32(3)

	testdb := e2eutil.CreateTestDB(t, "e2e-create-cluster-longn-", replicas, false, f.DestroyAfterFailure)
	defer testdb.Delete()

	testdb.Populate()
	testdb.Test()

	cluster := testdb.Cluster()

	if cluster.Labels[constants.MySQLOperatorVersionLabel] != f.BuildVersion {
		t.Errorf("Cluster label %q incorrect: %q != %q.", constants.MySQLOperatorVersionLabel, cluster.Labels[constants.MySQLOperatorVersionLabel], f.BuildVersion)
	} else {
		t.Logf("Cluster label %q: %q", constants.MySQLOperatorVersionLabel, cluster.Labels[constants.MySQLOperatorVersionLabel])
	}

	if cluster.Spec.Replicas != replicas {
		t.Errorf("Got cluster with %d replica(s), want %d", cluster.Spec.Replicas, replicas)
	}

	// Do we have a valid statefulset?
	ss, err := f.KubeClient.AppsV1beta1().StatefulSets(cluster.Namespace).Get(cluster.Name, metav1.GetOptions{})
	if err != nil {
		t.Errorf("Error getting statefulset for cluster %s: %v", cluster.Name, err)
	} else {
		if ss.Status.ReadyReplicas != replicas {
			t.Logf("%#v", ss.Status)
			t.Errorf("Got statefulset with %d ready replica(s), want %d", ss.Status.ReadyReplicas, replicas)
		}
		if ss.Labels[constants.MySQLOperatorVersionLabel] != f.BuildVersion {
			t.Errorf("StatefulSet label %q incorrect: %q != %q.", constants.MySQLOperatorVersionLabel, ss.Labels[constants.MySQLOperatorVersionLabel], f.BuildVersion)
		} else {
			t.Logf("StatefulSet label %q: %s", constants.MySQLOperatorVersionLabel, ss.Labels[constants.MySQLOperatorVersionLabel])
		}
	}

	// Do we have a service?
	_, err = f.KubeClient.CoreV1().Services(cluster.Namespace).Get(cluster.Name, metav1.GetOptions{})
	if err != nil {
		t.Errorf("Error getting service for cluster %s: %v", cluster.Name, err)
	}

	// Do we have a root password secret?
	f.KubeClient.CoreV1().Secrets(cluster.Namespace).Get(secrets.GetRootPasswordSecretName(cluster), metav1.GetOptions{})
	if err != nil {
		t.Errorf("Error getting root password secret for cluster %s: %v", cluster.Name, err)
	}

	t.Report()
}
