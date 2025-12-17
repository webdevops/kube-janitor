package kube_janitor

import (
	"context"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
	prometheusCommon "github.com/webdevops/go-common/prometheus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func (j *Janitor) runRules(ctx context.Context) error {
	metricResourceRule := prometheusCommon.NewMetricsList()

	for _, rule := range j.config.Rules {
		ruleLogger := j.logger.With(
			slog.Any("rule", rule),
		)
		ruleLogger.Info("running rule")

		err := j.kubeEachNamespace(ctx, rule.NamespaceSelector, func(namespace corev1.Namespace) error {
			namespaceLogger := ruleLogger

			for _, resourceType := range rule.Resources {
				gvkLogger := namespaceLogger.With(slog.Any("gvk", resourceType))
				err := j.kubeEachResource(ctx, resourceType.AsGVR(), namespace.GetName(), resourceType.Selector, func(resource unstructured.Unstructured) error {
					resourceLogger := gvkLogger.With(
						slog.String("resource", resource.GetName()),
					)
					resourceLogger.Debug("checking resources")

					return j.checkResourceTtlAndTriggerDeleteIfExpired(
						ctx,
						gvkLogger,
						resourceType,
						resource,
						rule.Id,
						rule.Ttl,
						metricResourceRule,
						prometheus.Labels{
							"rule": rule.String(),
						},
					)
				})
				if err != nil {
					return err
				}
			}

			return nil
		})
		if err != nil {
			return err
		}
	}

	metricResourceRule.GaugeSet(j.prometheus.rule)

	return nil
}
