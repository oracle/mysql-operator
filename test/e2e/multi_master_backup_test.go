// +build all default

package e2e

import (
	"testing"

	"github.com/oracle/mysql-operator/test/e2e/framework"
	e2eutil "github.com/oracle/mysql-operator/test/e2e/util"
)

func TestMultiMasterBackUpRestore(test *testing.T) {
	t := e2eutil.NewT(test)
	f := framework.Global

	t.Log("Creating mysqlcluster...")
	testdb := e2eutil.CreateTestDB(t, "e2e-mb-", 1, true, f.DestroyAfterFailure)
	defer testdb.Delete()
	clusterName := testdb.Cluster().Name

	testdb.Populate()
	testdb.Test()

	databaseName := "employees"

	t.Logf("Creating mysqlbackup for mysqlcluster '%s'...", clusterName)
	backupName := e2eutil.Backup(t, f, clusterName, "e2e-mb-backup-", databaseName)

	t.Log("Trying connection to container")
	testdb.CheckConnection(t)

	t.Log("Validating database..")
	testdb.Test()

	t.Logf("Deleting the %s database..", databaseName)
	e2eutil.DeleteDatabase(t, f, clusterName, databaseName)

	t.Logf("creating mysqlrestore from mysqlbackup '%s' for mysqlcluster '%s'.", backupName, clusterName)
	e2eutil.Restore(t, f, clusterName, backupName)

	t.Log("trying connection to container")
	testdb.CheckConnection(t)

	t.Log("validating database...")
	testdb.Test()

	t.Report()
}
