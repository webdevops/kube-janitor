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
			Name: "kube_janitor_resource_ttl",
			Help: "TTL date for a resource",
		},
		[]string{
			"version",
			"kind",
			"namespace",
			"name",
		},
	)
	prometheus.MustRegister(j.prometheus.ttl)
}
