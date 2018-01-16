// +build all default

package e2e

import (
	"fmt"
	"testing"

	"github.com/oracle/mysql-operator/test/e2e/framework"
	e2eutil "github.com/oracle/mysql-operator/test/e2e/util"
)

func TestMultiMaster(test *testing.T) {
	t := e2eutil.NewT(test)
	f := framework.Global
	replicas := int32(3)

	// ---------------------------------------------------------------------- //
	t.Log("creating cluster..")
	// ---------------------------------------------------------------------- //
	testdb := e2eutil.CreateTestDB(t, "e2e-mm-", replicas, true, f.DestroyAfterFailure)
	defer testdb.Delete()
	clusterName := testdb.Cluster().Name

	// ---------------------------------------------------------------------- //
	t.Log("test writing to multi-master database..")
	// ---------------------------------------------------------------------- //
	username := "root"
	for i := 0; i < int(replicas); i++ {
		dbName := fmt.Sprintf("test%d", i)
		podName := fmt.Sprintf("%s-%d", clusterName, i)
		password := e2eutil.GetMySQLPassword(t, podName, f.Namespace)
		sqlExecutor := e2eutil.NewKubectlSimpleSQLExecutor(t, podName, username, password, f.Namespace)
		dbHelper := e2eutil.NewMySQLDBTestHelper(t, sqlExecutor)
		t.Logf("writing database: %s from pod: %s", dbName, podName)
		dbHelper.EnsureDBTableValue(dbName, "people", "name", "kris")
	}

	// ---------------------------------------------------------------------- //
	t.Log("test reading from multi-master database..")
	// ---------------------------------------------------------------------- //
	podName := fmt.Sprintf("%s-0", clusterName)
	password := e2eutil.GetMySQLPassword(t, podName, f.Namespace)
	sqlExecutor := e2eutil.NewKubectlSimpleSQLExecutor(t, podName, username, password, f.Namespace)
	dbHelper := e2eutil.NewMySQLDBTestHelper(t, sqlExecutor)
	for i := 0; i < int(replicas); i++ {
		dbName := fmt.Sprintf("test%d", i)
		t.Logf("reading database: %s from pod: %s", dbName, podName)
		dbValueExists := dbHelper.HasDBTableValue(dbName, "people", "name", "kris")
		if !dbValueExists {
			t.Fatalf("Error database table '%s.people.kris' did not contain value 'kris'.", dbName)
		}
	}

	t.Report()
}
