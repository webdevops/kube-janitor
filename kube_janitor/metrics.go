package kube_janitor

import (
	"github.com/prometheus/client_golang/prometheus"
)

type (
	JanitorMetrics struct {
		deleted *prometheus.CounterVec
		ttl     *prometheus.GaugeVec
		rule    *prometheus.GaugeVec
	}
)

// setupMetrics setups all Prometheus metrics with name, help and corresponding labels
func (j *Janitor) setupMetrics() {
	ttlLabels := []string{
		"rule",
		"groupVersionKind",
		"namespace",
		"name",
		"ttl",
	}

	j.prometheus.deleted = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kube_janitor_resource_deleted_total",
			Help: "Total count of deleted Kubernetes resources",
		},
		[]string{
			"rule",
			"groupVersionKind",
			"namespace",
		},
	)
	prometheus.MustRegister(j.prometheus.deleted)

	j.prometheus.ttl = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kube_janitor_resource_ttl_expiry_timestamp_seconds",
			Help: "Expiry unix timestamp for Kubernetes resources by ttl",
		},
		ttlLabels,
	)
	prometheus.MustRegister(j.prometheus.ttl)

	j.prometheus.rule = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kube_janitor_resource_rule_expiry_timestamp_seconds",
			Help: "Expiry unix timestamp for Kubernetes resources by rule",
		},
		ttlLabels,
	)
	prometheus.MustRegister(j.prometheus.rule)
}
