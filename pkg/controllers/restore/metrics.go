package restore

import (
	"github.com/oracle/mysql-operator/pkg/util/metrics"
)

var (
	clusterRestoreCount = metrics.NewAgentEventCounter("cluster_restores", "Total number of times the cluster has been restored")
)

func RegisterMetrics() {
	metrics.RegisterAgentMetric(clusterRestoreCount)
}
