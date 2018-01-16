package backup

import (
	"github.com/oracle/mysql-operator/pkg/util/metrics"
)

var (
	clusterBackupCount = metrics.NewAgentEventCounter("cluster_backups", "Total number of times the cluster has been backed up")
)

func RegisterMetrics() {
	metrics.RegisterAgentMetric(clusterBackupCount)
}
