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

// setupMetrics setups all Prometheus metrics with name, help and corresponding labels
func (j *Janitor) setupMetrics() {
	commonLabels := []string{
		"rule",
		"version",
		"kind",
		"namespace",
		"name",
		"ttl",
	}

	j.prometheus.ttl = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kube_janitor_resource_ttl_expiry_timestamp_seconds",
			Help: "Expiry unix timestamp for Kubernetes resources by ttl",
		},
		commonLabels,
	)
	prometheus.MustRegister(j.prometheus.ttl)

	j.prometheus.rule = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kube_janitor_resource_rule_expiry_timestamp_seconds",
			Help: "Expiry unix timestamp for Kubernetes resources by rule",
		},
		commonLabels,
	)
	prometheus.MustRegister(j.prometheus.rule)
}
