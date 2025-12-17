package kube_janitor

import (
	"context"
	"log/slog"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	prometheusCommon "github.com/webdevops/go-common/prometheus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func (j *Janitor) runTtlResources(ctx context.Context) error {
	metricResourceTtl := prometheusCommon.NewMetricsList()

	resourceList, err := j.kubeLookupGvrs(j.config.Ttl.Resources)
	if err != nil {
		return err
	}

	for _, resourceType := range resourceList {
		gvkLogger := j.logger.With(slog.Any("gvk", resourceType))
		gvkLogger.Info("checking resources")

		err := j.kubeEachResource(ctx, resourceType.AsGVR(), KubeNoNamespace, resourceType.Selector, func(resource unstructured.Unstructured) error {
			var ttlValue string

			if j.config.Ttl.Annotation != "" {
				// get from meta.annotations
				if val, exists := resource.GetAnnotations()[j.config.Ttl.Annotation]; exists {
					ttlValue = strings.TrimSpace(val)
				}
			} else if j.config.Ttl.Label != "" {
				// get from meta.labels
				if val, exists := resource.GetLabels()[j.config.Ttl.Label]; exists {
					ttlValue = strings.TrimSpace(val)
				}
			}

			// check if we got a valid ttl value
			if ttlValue == "" {
				return nil
			}

			return j.checkResourceTtlAndTriggerDeleteIfExpired(
				ctx,
				gvkLogger,
				resourceType,
				resource,
				RuleIdInternalTTL,
				ttlValue,
				metricResourceTtl,
				prometheus.Labels{},
			)
		})
		if err != nil {
			return err
		}
	}

	metricResourceTtl.GaugeSet(j.prometheus.ttl)

	return nil
}
