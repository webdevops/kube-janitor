package kube_janitor

import (
	"context"
	"encoding/json"
	"log/slog"

	jmespath "github.com/jmespath-community/go-jmespath"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/webdevops/go-common/log/slogger"
	prometheusCommon "github.com/webdevops/go-common/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func (j *Janitor) checkResourceIsSkippedByJmesPath(resource unstructured.Unstructured, jmesPath jmespath.JMESPath) (bool, error) {

	resourceRaw, err := resource.MarshalJSON()
	if err != nil {
		return true, err
	}
	var data any
	err = json.Unmarshal(resourceRaw, &data)
	if err != nil {
		return true, err
	}

	// check if resource is valid by JMES path
	result, err := jmesPath.Search(data)
	if err != nil {
		return true, err
	}

	switch v := result.(type) {
	case string:
		// skip if string is empty
		if len(v) == 0 {
			return true, nil
		}
	case bool:
		// skip if false (not selected)
		return !v, nil
	case nil:
		// nil? jmes path didn't find anything? better skip the resource
		return true, nil
	}

	return false, nil
}

func (j *Janitor) checkResourceTtlAndTriggerDeleteIfExpired(ctx context.Context, logger *slogger.Logger, resourceConfig *ConfigResource, resource unstructured.Unstructured, ttlValue string, metricResourceTtl *prometheusCommon.MetricList, labels prometheus.Labels) error {
	resourceLogger := logger.With(
		slog.String("namespace", resource.GetNamespace()),
		slog.String("resource", resource.GetName()),
		slog.String("ttl", ttlValue),
	)

	if resourceConfig.JmesPath != "" {
		skipped, err := j.checkResourceIsSkippedByJmesPath(resource, resourceConfig.CompiledJmesPath())
		if err != nil {
			return err
		}

		if skipped {
			resourceLogger.Debug("resource skipped by JMES path")
			return nil
		}
	}

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
			err := j.dynClient.Resource(resourceConfig.AsGVR()).Namespace(resource.GetNamespace()).Delete(ctx, resource.GetName(), metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}

	return nil
}
