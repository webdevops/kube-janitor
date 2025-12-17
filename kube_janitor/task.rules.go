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

		namespaced := false
		if !rule.NamespaceSelector.IsEmpty() {
			namespaced = true
		}

		resourceList, err := j.kubeLookupGvrs(rule.Resources, namespaced)
		if err != nil {
			return err
		}

		// build namespace list
		var namespaceList []string
		if namespaced {
			err = j.kubeEachNamespace(ctx, rule.NamespaceSelector, func(namespace corev1.Namespace) error {
				namespaceList = append(namespaceList, namespace.Name)
				return nil
			})
			if err != nil {
				return err
			}
		} else {
			// we fake an empty namespace (=get resources from the cluster view)
			namespaceList = append(namespaceList, KubeNoNamespace)
		}

		// find resources, check and process them
		for _, namespace := range namespaceList {
			for _, resourceType := range resourceList {
				gvkLogger := ruleLogger.With(slog.Any("groupVersionKind", resourceType))
				if namespace != KubeNoNamespace {
					gvkLogger = gvkLogger.With(slog.String("namespace", namespace))
				}

				gvkLogger.Info("checking resources")
				err := j.kubeEachResource(ctx, resourceType.AsGVR(), namespace, resourceType.Selector, func(resource unstructured.Unstructured) error {
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
					gvkLogger.Error("failed to list resources", slog.Any("error", err))
				}
			}
		}
	}

	metricResourceRule.GaugeSet(j.prometheus.rule)

	return nil
}
