package util

import (
	"fmt"
	"math/rand"
	"os/exec"
	"strings"
	"time"

	api "github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
	"github.com/oracle/mysql-operator/test/e2e/framework"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TestDB struct {
	cluster             *api.MySQLCluster
	t                   *T
	destroyAfterFailure bool
}

func (db *TestDB) Cluster() *api.MySQLCluster {
	return db.cluster
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func CreateTestDB(t *T, prefix string, replicas int32, destroyAfterFailure bool) *TestDB {
	f := framework.Global

	res, err := f.MySQLOpClient.MysqlV1().MySQLClusters(f.Namespace).Create(NewMySQLCluster(prefix, replicas))
	if err != nil {
		t.Fatalf("Failed to create cluster: %v", err)
	}

	t.Logf("Created MySQLCluster named '%s'", res.Name)

	// Do we have a valid cluster?
	cl, err := WaitForClusterPhase(t, res, api.MySQLClusterRunning, DefaultRetry, f.MySQLOpClient)
	if err != nil {
		t.Fatalf("Cluster failed to reach phase %q: %v", api.MySQLClusterRunning, err)
	}
	t.Logf("Using cluster:%s", cl.Name)

	return &TestDB{
		cluster:             cl,
		t:                   t,
		destroyAfterFailure: destroyAfterFailure,
	}
}

func GetTestDB(t *T, name string, destroyAfterFailure bool) *TestDB {
	f := framework.Global

	res, err := f.MySQLOpClient.MysqlV1().
		MySQLClusters(f.Namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to find cluster: %v", err)
	}
	t.Logf("Found MySQLCluster named '%s'", res.Name)

	// Do we have a valid cluster?
	cl, err := WaitForClusterPhase(t, res, api.MySQLClusterRunning, DefaultRetry, f.MySQLOpClient)
	if err != nil {
		t.Fatalf("Cluster failed to reach phase %q: %v", api.MySQLClusterRunning, err)
	}
	t.Logf("Using cluster:%s", cl.Name)

	return &TestDB{
		cluster:             cl,
		t:                   t,
		destroyAfterFailure: destroyAfterFailure,
	}
}

func (testDB *TestDB) install() {
	clusterName := testDB.cluster.Name
	podname := string(clusterName + "-0")
	username := "root"
	password := GetMySQLPassword(testDB.t, podname, testDB.cluster.Namespace)
	executor := NewKubectlSimpleSQLExecutor(testDB.t, podname, username, password, testDB.cluster.Namespace)

	testDB.t.Logf("Installing git")
	output, err := executor.ExecuteCMD("yum install -y git")
	if err != nil {
		testDB.t.Fatalf("Failed to install git:%s", output)
	}

	testDB.t.Logf("Cloning testdb")
	err = Retry(NewDefaultRetyWithDuration(25*time.Second), func() (bool, error) {
		output, err = executor.ExecuteCMD("git clone https://github.com/datacharmer/test_db.git")
		if err != nil {
			testDB.t.Logf("failed to clone test db, retrying ...")
			testDB.t.Logf("    output: %s", output)
			testDB.t.Logf("    err: %v", err)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		testDB.t.Fatalf("Failed to clone test db")
	}
}

func (testDB *TestDB) Populate() {
	clusterName := testDB.cluster.Name
	podname := string(clusterName + "-0")
	testDB.t.Logf("Populating test db on cluster: %s from pod: %s", clusterName, podname)
	username := "root"
	password := GetMySQLPassword(testDB.t, podname, testDB.cluster.Namespace)
	executor := NewKubectlSimpleSQLExecutor(testDB.t, podname, username, password, testDB.cluster.Namespace)

	testDB.install()

	testDB.t.Logf("Loading test db data")
	output, err := executor.ExecuteCMD(fmt.Sprintf("cd test_db && mysql -uroot -p%s < employees.sql", password))
	if err != nil {
		testDB.t.Log(output)
		testDB.t.Fatalf("Failed load db data, err: %v", err)
	}
}

func (testDB *TestDB) GetClusterName() string {
	return testDB.cluster.Name
}

func (testDB *TestDB) GetPassword() (string, error) {
	clusterName := testDB.cluster.Name
	podname := string(clusterName + "-0")

	cmd := exec.Command(
		"kubectl",
		"-n", testDB.cluster.Namespace,
		"exec", podname, "--",
		"bash", "-c", "env | grep MYSQL_ROOT_PASSWORD",
	)
	output, err := executeCmd(testDB.t, cmd)
	if err != nil {
		return output, err
	}
	return strings.TrimSpace(strings.SplitN(output, "=", 2)[1]), nil
}

func (testDB *TestDB) Test() {
	clusterName := testDB.cluster.Name
	podname := string(clusterName + "-0")
	testDB.t.Logf("Validating test db on cluster: %s from pod: %s", clusterName, podname)
	username := "root"
	password := GetMySQLPassword(testDB.t, podname, testDB.cluster.Namespace)
	executor := NewKubectlSimpleSQLExecutor(testDB.t, podname, username, password, testDB.cluster.Namespace)
	output, err := executor.ExecuteCMD("cd test_db")
	if err != nil {
		testDB.install()
	}

	cmd := exec.Command(
		"kubectl",
		"-n", testDB.cluster.Namespace,
		"cp",
		"sql_test.sh",
		fmt.Sprintf("%s:/sql_test.sh", podname),
	)
	output, err = executeCmd(testDB.t, cmd)
	if err != nil {
		testDB.t.Fatalf("Copy db test script failed:%s", output)
	}

	testDB.t.Logf("Testing db data")
	output, err = executor.ExecuteCMD(fmt.Sprintf("/sql_test.sh 'mysql -uroot -p%s'", password))
	if err != nil {
		testDB.t.Fatalf("Test db md5 failed\n%s", output)
	}
	if !testOK(output, "employees") {
		testDB.t.Error("'employees' database integrity checksum failed.")
	}
}

func testOK(output string, target string) bool {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, target) {
			elements := strings.Fields(line)
			if len(elements) > 3 {
				ok1 := strings.Trim(elements[3], " ")
				ok2 := strings.Trim(elements[4], " ")
				if ok1 == "(OK" && ok2 == "OK" {
					return true
				}
			}
		}
	}
	return false
}

// If err is an error and
func (testDB *TestDB) Delete() {
	f := framework.Global

	if testDB.t.Failed() && !testDB.destroyAfterFailure {
		testDB.t.Logf("Not deleting DB cause %v", testDB.destroyAfterFailure)
		return
	}
	testDB.t.Logf("Deleting Cluster:%#v", testDB.Cluster())
	err := f.MySQLOpClient.MysqlV1().
		MySQLClusters(f.Namespace).
		Delete(testDB.cluster.Name, &metav1.DeleteOptions{})
	if err != nil {
		testDB.t.Fatalf("Failed clean up cluster: %v", err)
	}
	testDB.t.Log("Delete db finished")
}
