package cluster

import "github.com/prometheus/client_golang/prometheus"

var (
	clustersTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mysql_operator",
		Subsystem: "cluster",
		Name:      "clusters",
		Help:      "Total number of clusters managed",
	})

	clustersCreated = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mysql_operator",
		Subsystem: "cluster",
		Name:      "clusters_created",
		Help:      "Total number of clusters created",
	})

	clustersDeleted = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mysql_operator",
		Subsystem: "cluster",
		Name:      "clusters_deleted",
		Help:      "Total number of clusters deleted",
	})
)

// RegisterMetrics will register Prometheus metrics for the cluster controller
func RegisterMetrics() {
	prometheus.MustRegister(clustersTotal)
	prometheus.MustRegister(clustersCreated)
	prometheus.MustRegister(clustersDeleted)
}
