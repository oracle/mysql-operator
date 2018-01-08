package cluster

import (
	"fmt"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/util/retry"

	api "github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
	mysqlop "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned"
	listers "github.com/oracle/mysql-operator/pkg/generated/listers/mysql/v1"
)

type clusterUpdaterInterface interface {
	UpdateClusterStatus(cluster *api.MySQLCluster, status *api.MySQLClusterStatus) error
	UpdateClusterLabels(cluster *api.MySQLCluster, lbls labels.Set) error
}

type clusterUpdater struct {
	client mysqlop.Interface
	lister listers.MySQLClusterLister
}

func newClusterUpdater(client mysqlop.Interface, lister listers.MySQLClusterLister) clusterUpdaterInterface {
	return &clusterUpdater{client: client, lister: lister}
}

func (csu *clusterUpdater) UpdateClusterStatus(cluster *api.MySQLCluster, status *api.MySQLClusterStatus) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cluster.Status = *status
		_, updateErr := csu.client.MysqlV1().MySQLClusters(cluster.Namespace).Update(cluster)
		if updateErr == nil {
			return nil
		}

		updated, err := csu.lister.MySQLClusters(cluster.Namespace).Get(cluster.Name)
		if err != nil {
			glog.Errorf("Error getting updated MySQLCluster %s/%s: %v", cluster.Namespace, cluster.Name, err)
			return err
		}

		// Copy the MySQLCluster so we don't mutate the cache.
		cluster = updated.DeepCopy()
		return updateErr
	})
}

func (csu *clusterUpdater) UpdateClusterLabels(cluster *api.MySQLCluster, lbls labels.Set) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cluster.Labels = labels.Merge(labels.Set(cluster.Labels), lbls)
		_, updateErr := csu.client.MysqlV1().MySQLClusters(cluster.Namespace).Update(cluster)
		if updateErr == nil {
			return nil
		}

		key := fmt.Sprintf("%s/%s", cluster.GetNamespace(), cluster.GetName())
		glog.V(4).Infof("Conflict updating MySQLCluster labels. Getting updated MySQLCluster %s from cache...", key)

		updated, err := csu.lister.MySQLClusters(cluster.GetNamespace()).Get(cluster.GetName())
		if err != nil {
			glog.Errorf("Error getting updated MySQLCluster %s: %v", key, err)
			return err
		}

		// Copy the MySQLCluster so we don't mutate the cache.
		cluster = updated.DeepCopy()
		return updateErr
	})
}
