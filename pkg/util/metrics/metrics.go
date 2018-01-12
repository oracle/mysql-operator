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
