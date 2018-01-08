package e2e

import (
	"fmt"
	"testing"

	fw "github.com/oracle/mysql-operator/test/e2e/framework"
	e2eutil "github.com/oracle/mysql-operator/test/e2e/util"
)

// TestCrashRecovery spins up a 3-instance cluster then checks then check the
// various pod based failure modes
func TestPodCrashRecovery(t *testing.T) {
	f := fw.Global
	namespace := f.Namespace
	var numInstances int32 = 3
	var testdb *e2eutil.TestDB

	testdb = e2eutil.CreateTestDB(t, "e2e-pr-", numInstances, f.DestroyAfterFailure)
	defer testdb.Delete()

	fmt.Printf("=============== Populating the database ===============\n")
	testdb.Populate()
	fmt.Printf("=============== Validating the database ===============\n")
	testdb.Test()
	fmt.Printf("--------------- Complete ---------------\n")

	clusterName := testdb.GetClusterName()
	var podName string

	fmt.Printf("=============== Testing mysql primary pod crash ===============\n")
	podName = e2eutil.GetPrimaryPodName(t, namespace, clusterName, f.KubeClient)
	e2eutil.TestMySQLPodCrash(t, namespace, podName, f.KubeClient, clusterName, numInstances)
	e2eutil.CheckPrimaryFailover(t, namespace, clusterName, podName, f.KubeClient)
	fmt.Printf("--------------- Test complete ---------------\n")

	fmt.Printf("=============== Testing mysql secondary pod crash ===============\n")
	podName = e2eutil.GetSecondaryPodName(t, namespace, clusterName, f.KubeClient)
	e2eutil.TestMySQLPodCrash(t, namespace, podName, f.KubeClient, clusterName, numInstances)
	fmt.Printf("--------------- Test complete ---------------\n")

	fmt.Printf("=============== Validating the database ===============\n")
	testdb.Test()
	fmt.Printf("--------------- Complete ---------------\n")
}
