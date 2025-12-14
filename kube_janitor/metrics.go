package kube_janitor

import (
	"github.com/prometheus/client_golang/prometheus"
)

type (
	JanitorMetrics struct {
		ttl  *prometheus.GaugeVec
		rule *prometheus.GaugeVec
	}
)

func (j *Janitor) setupMetrics() {
	j.prometheus.ttl = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kube_janitor_resource_ttl_expiry_timestamp_seconds",
			Help: "Expiry unix timestamp for Kubernetes resources by ttl",
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

	j.prometheus.rule = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kube_janitor_resource_rule_expiry_timestamp_seconds",
			Help: "Expiry unix timestamp for Kubernetes resources by rule",
		},
		[]string{
			"rule",
			"version",
			"kind",
			"namespace",
			"name",
			"ttl",
		},
	)
	prometheus.MustRegister(j.prometheus.rule)
}
