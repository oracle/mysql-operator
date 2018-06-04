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

package cluster

import (
	"github.com/oracle/mysql-operator/pkg/util/metrics"
)

var (
	clustersTotalCount   = metrics.NewOperatorEventGauge("clusters", "Total number of clusters managed")
	clustersCreatedCount = metrics.NewOperatorEventCounter("clusters_created", "Total number of clusters created")
	clustersDeletedCount = metrics.NewOperatorEventCounter("clusters_deleted", "Total number of clusters deleted")
)

// RegisterMetrics registers the cluster CRUD metrics.
func RegisterMetrics() {
	metrics.RegisterOperatorMetric(clustersTotalCount)
	metrics.RegisterOperatorMetric(clustersCreatedCount)
	metrics.RegisterOperatorMetric(clustersDeletedCount)
}
