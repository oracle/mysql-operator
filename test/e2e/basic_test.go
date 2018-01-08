package e2e

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/oracle/mysql-operator/pkg/constants"
	"github.com/oracle/mysql-operator/pkg/resources/secrets"
	"github.com/oracle/mysql-operator/test/e2e/framework"
	e2eutil "github.com/oracle/mysql-operator/test/e2e/util"
)

func TestCreateCluster(t *testing.T) {
	f := framework.Global
	replicas := int32(3)

	var err error

	testdb := e2eutil.CreateTestDB(t, "e2e-cc-biglongnametocheckitstillworks", replicas, f.DestroyAfterFailure)
	defer testdb.Delete()

	testdb.Populate()
	testdb.Test()

	cluster := testdb.Cluster()

	if cluster.Labels[constants.MySQLOperatorVersionLabel] != f.BuildVersion {
		t.Errorf("Cluster MySQLOperatorVersionLabel was incorrect: %s != %s.", cluster.Labels[constants.MySQLOperatorVersionLabel], f.BuildVersion)
	} else {
		t.Logf("Cluster label MySQLOperatorVersionLabel: %s", cluster.Labels[constants.MySQLOperatorVersionLabel])
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
			t.Errorf("StatefulSet MySQLOperatorVersionLabel was incorrect: %s != %s.", ss.Labels[constants.MySQLOperatorVersionLabel], f.BuildVersion)
		} else {
			t.Logf("StatefulSet label MySQLOperatorVersionLabel: %s", ss.Labels[constants.MySQLOperatorVersionLabel])
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
}
