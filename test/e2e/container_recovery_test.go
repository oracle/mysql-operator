// +build all default

package e2e

import (
	"testing"

	"github.com/oracle/mysql-operator/test/e2e/framework"
	e2eutil "github.com/oracle/mysql-operator/test/e2e/util"
)

// TestCrashRecovery spins up a 3-instance cluster then checks then check the
// various container based failure modes
func TestContainerCrashRecovery(test *testing.T) {
	t := e2eutil.NewT(test)
	f := framework.Global
	namespace := f.Namespace
	var numInstances int32 = 3
	var testdb *e2eutil.TestDB

	testdb = e2eutil.CreateTestDB(t, "e2e-cr-", numInstances, f.DestroyAfterFailure)
	defer testdb.Delete()

	t.Log("=============== Populating the database ===============")
	testdb.Populate()
	t.Log("=============== Validating the database ===============")
	testdb.Test()
	t.Log("--------------- Complete ---------------")

	clusterName := testdb.GetClusterName()
	var podName string

	t.Log("=============== Testing mysql-agent primary container crash ===============")
	podName = e2eutil.GetPrimaryPodName(t, namespace, clusterName, f.KubeClient)
	e2eutil.TestMySQLContainerCrash(t, namespace, podName, "mysql-agent", f, clusterName, numInstances)
	t.Log("--------------- Test complete ---------------")

	t.Log("=============== Testing mysql-agent secondary container crash ===============")
	podName = e2eutil.GetSecondaryPodName(t, namespace, clusterName, f.KubeClient)
	e2eutil.TestMySQLContainerCrash(t, namespace, podName, "mysql-agent", f, clusterName, numInstances)
	t.Log("--------------- Test complete ---------------")

	t.Log("=============== Testing mysql primary container crash ===============")
	podName = e2eutil.GetPrimaryPodName(t, namespace, clusterName, f.KubeClient)
	e2eutil.TestMySQLContainerCrash(t, namespace, podName, "mysql", f, clusterName, numInstances)
	e2eutil.CheckPrimaryFailover(t, namespace, clusterName, podName, f.KubeClient)
	t.Log("--------------- Test complete ---------------")

	t.Log("=============== Testing mysql secondary container crash ===============")
	podName = e2eutil.GetSecondaryPodName(t, namespace, clusterName, f.KubeClient)
	e2eutil.TestMySQLContainerCrash(t, namespace, podName, "mysql", f, clusterName, numInstances)
	t.Log("--------------- Test complete ---------------")

	t.Log("=============== Validating the database ===============")
	testdb.Test()
	t.Log("--------------- Complete ---------------")

	t.Report()
}
