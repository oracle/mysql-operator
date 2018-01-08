package util

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
	mysqlop "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned"
)

func CreateMySQLCluster(t *testing.T, mysqlopClient mysqlop.Interface, ns string, cluster *api.MySQLCluster) (*api.MySQLCluster, error) {
	cluster.Namespace = ns
	res, err := mysqlopClient.MysqlV1().MySQLClusters(ns).Create(cluster)
	if err != nil {
		return nil, err
	}
	t.Logf("Creating mysql cluster: %s", res.Name)
	return res, nil
}

// TODO(apryde): Wait for deletion of underlying resources.
func DeleteMySQLCluster(t *testing.T, mysqlopClient mysqlop.Interface, cluster *api.MySQLCluster) error {
	t.Logf("Deleting mysql cluster: %s", cluster.Name)
	err := mysqlopClient.MysqlV1().MySQLClusters(cluster.Namespace).Delete(cluster.Name, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}
