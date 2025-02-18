package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	crStatusMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "challenge_resource_status",
			Help: "Tracks the status of the custom resource",
		},
		[]string{"challeng_id", "challenge_name", "username", "namespace"},
	)

	challengeStatusCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "challenge_status_count",
			Help: "Total count of challenges that are running",
		},
		[]string{"status"},
	)
)

func init() {
	metrics.Registry.MustRegister(crStatusMetric)
	metrics.Registry.MustRegister(challengeStatusCount)

}
