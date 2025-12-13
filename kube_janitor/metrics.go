package kube_janitor

import (
	"github.com/prometheus/client_golang/prometheus"
)

type (
	JanitorMetrics struct {
		ttl *prometheus.GaugeVec
	}
)

func (j *Janitor) setupMetrics() {
	j.prometheus.ttl = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kube_janitor_resource_expiry_timestamp_seconds",
			Help: "Expiry unix timestamp for Kubernetes resources",
		},
		[]string{
			"version",
			"kind",
			"namespace",
			"name",
			"ttl",
		},
	)
	prometheus.MustRegister(j.prometheus.ttl)
}
