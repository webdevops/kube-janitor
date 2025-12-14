package kube_janitor

import (
	"context"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/webdevops/go-common/log/slogger"
	prometheusCommon "github.com/webdevops/go-common/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (j *Janitor) checkResourceTtlAndTriggerDeleteIfExpired(ctx context.Context, logger *slogger.Logger, gvr schema.GroupVersionResource, resource unstructured.Unstructured, ttlValue string, metricResourceTtl *prometheusCommon.MetricList, labels prometheus.Labels) error {
	resourceLogger := logger.With(
		slog.String("namespace", resource.GetNamespace()),
		slog.String("resource", resource.GetName()),
		slog.String("ttl", ttlValue),
	)

	parsedDate, expired, err := j.checkExpiryDate(resource.GetCreationTimestamp().Time, ttlValue)
	if err != nil {
		resourceLogger.Error("unable to parse expiration date", slog.String("raw", ttlValue), slog.Any("error", err))
		return nil
	}

	labels["version"] = resource.GetAPIVersion()
	labels["kind"] = resource.GetKind()
	labels["namespace"] = resource.GetNamespace()
	labels["name"] = resource.GetName()
	labels["ttl"] = ttlValue
	metricResourceTtl.AddTime(labels, *parsedDate)

	resourceLogger.Debug("found resource with valid TTL", slog.Time("expirationDate", *parsedDate))

	if expired {
		if j.dryRun {
			resourceLogger.Info("resource is expired, would delete resource (DRY-RUN)", slog.Time("expirationDate", *parsedDate))
		} else {
			resourceLogger.Info("deleting expired resource", slog.Time("expirationDate", *parsedDate))
			err := j.dynClient.Resource(gvr).Namespace(resource.GetNamespace()).Delete(ctx, resource.GetName(), metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}

	return nil
}
