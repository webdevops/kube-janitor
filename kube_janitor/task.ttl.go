package kube_janitor

import (
	"context"
	"log/slog"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	prometheusCommon "github.com/webdevops/go-common/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func (j *Janitor) runTtlResources() error {
	j.connect()

	ctx := context.Background()

	metricResourceTtl := prometheusCommon.NewMetricsList()

	labelSelector := ""
	// we can use a label selector if ttl is configured by label
	if j.config.Ttl.Label != "" {
		labelSelector = j.config.Ttl.Label
	}

	for _, resourceType := range j.config.Ttl.Resources {
		gvkLogger := j.logger.With(slog.Any("gvk", resourceType))
		gvkLogger.Debug("checking resources")

		err := j.kubeEachResource(ctx, resourceType.AsGVR(), labelSelector, func(resource unstructured.Unstructured) error {
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

			resourceLogger := gvkLogger.With(
				slog.String("namespace", resource.GetNamespace()),
				slog.String("resource", resource.GetName()),
				slog.String("ttl", ttlValue),
			)

			parsedDate, expired, err := j.checkExpiryDate(resource.GetCreationTimestamp().Time, ttlValue)
			if err != nil {
				resourceLogger.Error("unable to parse expiration date", slog.String("raw", ttlValue), slog.Any("error", err))
				return nil
			}

			metricResourceTtl.AddTime(
				prometheus.Labels{
					"version":   resource.GetAPIVersion(),
					"kind":      resource.GetKind(),
					"namespace": resource.GetNamespace(),
					"name":      resource.GetName(),
					"ttl":       ttlValue,
				},
				*parsedDate,
			)

			resourceLogger.Debug("found resource with valid TTL", slog.Time("expirationDate", *parsedDate))

			if expired {
				if j.dryRun {
					resourceLogger.Info("resource is expired, would delete resource (DRY-RUN)", slog.Time("expirationDate", *parsedDate))
				} else {
					resourceLogger.Info("deleting expired resource", slog.Time("expirationDate", *parsedDate))
					err := j.dynClient.Resource(resourceType.AsGVR()).Namespace(resource.GetNamespace()).Delete(ctx, resource.GetName(), metav1.DeleteOptions{})
					if err != nil {
						return err
					}
				}
			}

			return nil
		})
		if err != nil {
			return err
		}
	}

	metricResourceTtl.GaugeSet(j.prometheus.ttl)

	return nil
}
