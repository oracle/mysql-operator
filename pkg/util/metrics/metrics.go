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

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/oracle/mysql-operator/pkg/cluster/innodb"
)

var podName string
var clusterName string

func RegisterPodName(name string) {
	podName = name
}

func RegisterClusterName(name string) {
	clusterName = name
}

func RegisterOperatorMetric(metric prometheus.Collector) {
	assertPodName()
	prometheus.MustRegister(metric)
}

func RegisterAgentMetric(metric prometheus.Collector) {
	assertPodName()
	assertClusterName()
	prometheus.MustRegister(metric)
}

func NewOperatorEventCounter(name string, help string) *prometheus.CounterVec {
	return newCounter("mysql_operator", "cluster", name, help, []string{"podName"})
}

func NewOperatorEventGauge(name string, help string) *prometheus.GaugeVec {
	return newGauge("mysql_operator", "cluster", name, help, []string{"podName"})
}

func NewAgentEventCounter(name string, help string) *prometheus.CounterVec {
	return newCounter("mysql", "innodb", name, help, []string{"podName", "clusterName"})
}

func NewAgentStatusCounter(name string, help string) *prometheus.CounterVec {
	return newCounter("mysql", "innodb", name, help, []string{"podName", "clusterName", "instanceStatus"})
}

func IncEventCounter(counter *prometheus.CounterVec) {
	counter.With(eventLabels()).Inc()
}

func IncEventGauge(gauge *prometheus.GaugeVec) {
	gauge.With(eventLabels()).Inc()
}

func DecEventGauge(gauge *prometheus.GaugeVec) {
	gauge.With(eventLabels()).Dec()
}

func IncStatusCounter(counter *prometheus.CounterVec, status innodb.InstanceStatus) {
	counter.With(statusLabels(status)).Inc()
}

func newCounter(namespace string, subsystem string, name string, help string, labels []string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
		},
		labels,
	)
}

func newGauge(namespace string, subsystem string, name string, help string, labels []string) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
		},
		labels,
	)
}

func assertPodName() {
	if podName == "" {
		panic("Metrics package requires podName. Unable to register metrics")
	}
}

func assertClusterName() {
	if clusterName == "" {
		panic("Metrics package requires clusterName. Unable to register metrics")
	}
}

func eventLabels() prometheus.Labels {
	labels := prometheus.Labels{
		"podName": podName,
	}
	if clusterName != "" {
		labels["clusterName"] = clusterName
	}
	return labels
}

func statusLabels(status innodb.InstanceStatus) prometheus.Labels {
	return prometheus.Labels{
		"podName":        podName,
		"clusterName":    clusterName,
		"instanceStatus": string(status),
	}
}
