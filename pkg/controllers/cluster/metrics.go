package cluster

import (
	"github.com/oracle/mysql-operator/pkg/util/metrics"
)

var (
	clustersTotalCount   = metrics.NewOperatorEventGauge("clusters", "Total number of clusters managed")
	clustersCreatedCount = metrics.NewOperatorEventCounter("clusters_created", "Total number of clusters created")
	clustersDeletedCount = metrics.NewOperatorEventCounter("clusters_deleted", "Total number of clusters deleted")
)

func RegisterMetrics() {
	metrics.RegisterOperatorMetric(clustersTotalCount)
	metrics.RegisterOperatorMetric(clustersCreatedCount)
	metrics.RegisterOperatorMetric(clustersDeletedCount)
}
