// Copyright 2018 Oracle and/or its affiliates. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package manager

import (
	"github.com/oracle/mysql-operator/pkg/util/metrics"
)

var (
	clusterCreateCount       = metrics.NewAgentEventCounter("cluster_created", "Total number of times an innodb cluster is successfully created")
	clusterCreateErrorCount  = metrics.NewAgentEventCounter("cluster_create_error", "Total number of times and innodb cluster fails to create")
	clusterNoQuorumCount     = metrics.NewAgentEventCounter("cluster_no_quorum", "Total number of times the cluster has been seen in a NO_QUORUM state from an instance")
	instanceAddCount         = metrics.NewAgentEventCounter("instance_added", "Total number of times an instance is successfully added to the innodb cluster")
	instanceAddErrorCount    = metrics.NewAgentEventCounter("instance_add_error", "Total number of times an instance failed to add to the innodb cluster")
	instanceRejoinCount      = metrics.NewAgentEventCounter("instance_rejoined", "Total number of times an instance successfully rejoins the innodb cluster")
	instanceRejoinErrorCount = metrics.NewAgentEventCounter("instance_rejoin_error", "Total number of times an instance failed to rejoin the innodb cluster")
	instanceStatusCount      = metrics.NewAgentStatusCounter("instance_status", "Total number of times the operator detects an instance with a specific innodb status")
)

// RegisterMetrics registers the cluster managemnent metrics.
func RegisterMetrics() {
	metrics.RegisterAgentMetric(clusterCreateCount)
	metrics.RegisterAgentMetric(clusterCreateErrorCount)
	metrics.RegisterAgentMetric(clusterNoQuorumCount)
	metrics.RegisterAgentMetric(instanceAddCount)
	metrics.RegisterAgentMetric(instanceAddErrorCount)
	metrics.RegisterAgentMetric(instanceRejoinCount)
	metrics.RegisterAgentMetric(instanceRejoinErrorCount)
	metrics.RegisterAgentMetric(instanceStatusCount)
}
