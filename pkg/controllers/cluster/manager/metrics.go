package manager

import (
	"github.com/oracle/mysql-operator/pkg/util/metrics"
)

var (
	clusterCreateCount       = metrics.NewAgentEventCounter("cluster_created", "Total number of times an innodb cluster is successfully created")
	clusterCreateErrorCount  = metrics.NewAgentEventCounter("cluster_create_error", "Total number of times and innodb cluster fails to create")
	instanceAddCount         = metrics.NewAgentEventCounter("instance_added", "Total number of times an instance is successfully added to the innodb cluster")
	instanceAddErrorCount    = metrics.NewAgentEventCounter("instance_add_error", "Total number of times an instance failed to add to the innodb cluster")
	instanceRejoinCount      = metrics.NewAgentEventCounter("instance_rejoined", "Total number of times an instance successfully rejoins the innodb cluster")
	instanceRejoinErrorCount = metrics.NewAgentEventCounter("instance_rejoin_error", "Total number of times an instance failed to rejoin the innodb cluster")
	instanceStatusCount      = metrics.NewAgentStatusCounter("instance_status", "Total number of times the operator detects an instance with a specific innodb status")
)

func RegisterMetrics() {
	metrics.RegisterAgentMetric(clusterCreateCount)
	metrics.RegisterAgentMetric(clusterCreateErrorCount)
	metrics.RegisterAgentMetric(instanceAddCount)
	metrics.RegisterAgentMetric(instanceAddErrorCount)
	metrics.RegisterAgentMetric(instanceRejoinCount)
	metrics.RegisterAgentMetric(instanceRejoinErrorCount)
	metrics.RegisterAgentMetric(instanceStatusCount)
}
