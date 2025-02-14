package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	crStatusMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "custom_resource_status",
			Help: "Tracks the status of the custom resource",
		},
		[]string{"name", "namespace", "status"},
	)
)

func init() {
	metrics.Registry.MustRegister(crStatusMetric)
}
