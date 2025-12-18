package kube_janitor

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/webdevops/go-common/log/slogger"
	prometheusCommon "github.com/webdevops/go-common/prometheus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	RuleIdInternalTTL = "kube-janitor-ttl"
)

// runRule executes one ConfigRule ttl run
func (j *Janitor) runRule(ctx context.Context, logger *slogger.Logger, rule *ConfigRule, metricList *prometheusCommon.MetricList, filterFunc func(rule *ConfigRule, resource unstructured.Unstructured) (string, bool)) error {
	ruleLogger := logger.With(
		slog.Any("rule", rule),
	)
	ruleLogger.Info("running rule")

	var namespaced bool
	if !rule.NamespaceSelector.IsEmpty() {
		// if we have a namespace selector, we have to lookup matching all namespaces
		// and executes the rule within these namespaces.
		// this automatically excludes cluster resources (non-namespaced) as they are
		// not part of any namespace.
		namespaced = true
	}

	resourceList, err := j.kubeLookupGvkList(rule.Resources, namespaced)
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
				ttl, ok := filterFunc(rule, resource)
				if !ok || ttl == "" {
					return nil
				}

				return j.checkResourceTtlAndTriggerDeleteIfExpired(
					ctx,
					gvkLogger,
					resourceType,
					resource,
					rule.Id,
					ttl,
					metricList,
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

	return nil
}

// checkResourceTtlAndTriggerDeleteIfExpired checks the resource against the defined TTL and deletes if the resource is expired
func (j *Janitor) checkResourceTtlAndTriggerDeleteIfExpired(ctx context.Context, logger *slogger.Logger, resourceConfig *ConfigResource, resource unstructured.Unstructured, ruleId string, ttlValue string, metricResourceTtl *prometheusCommon.MetricList, labels prometheus.Labels) error {
	resourceLogger := logger.WithGroup("resource").With(
		slog.String("namespace", resource.GetNamespace()),
		slog.String("name", resource.GetName()),
		slog.String("ttl", ttlValue),
	)

	// no ttl, no processing
	// better safe than sorry
	if ttlValue == "" {
		return nil
	}

	// check if resource is filtered
	if !resourceConfig.FilterPath.IsEmpty() {
		skipped, err := j.checkResourceIsSkippedFromJmesPath(resource, resourceConfig.FilterPath)
		if err != nil {
			return err
		}

		if skipped {
			resourceLogger.Debug("resource skipped by JMES path")
			return nil
		}
	}

	// use creation timesstamp by default
	// use timestamp from jmespath as alterantive (if configured)
	timestamp := resource.GetCreationTimestamp().Time
	if !resourceConfig.TimestampPath.IsEmpty() {
		val, err := j.parseResourceTimestampFromJmesPath(resource, resourceConfig.TimestampPath)
		if err != nil {
			resourceLogger.Warn("parse resource timestamp from jmesPath failed", slog.Any("error", err))
			return nil
		} else if val == nil {
			resourceLogger.Debug("parse resource timestamp from jmesPath failed")
			return nil
		}

		timestamp = *val
	}

	parsedDate, expired, err := j.checkExpiryDate(timestamp, ttlValue)
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

	resourceLogger.Debug("found resource with valid TTL", slog.Time("expiry", *parsedDate))

	if expired {
		if j.dryRun {
			resourceLogger.Info("resource is expired, would delete resource (DRY-RUN)", slog.Time("expirationDate", *parsedDate))
		} else {
			resourceLogger.Info("deleting expired resource", slog.Time("expirationDate", *parsedDate))
			err := j.dynClient.Resource(resourceConfig.AsGVR()).Namespace(resource.GetNamespace()).Delete(ctx, resource.GetName(), metav1.DeleteOptions{})
			if err != nil {
				return err
			}

			reason := "TimeToLiveExpired"
			message := fmt.Sprintf(`TTL of "%v" is expired and resource is being deleted (%s)`, ttlValue, ruleId)

			err = j.kubeCreateEventFromResource(ctx, resource.GetNamespace(), resource, message, reason)
			if err != nil {
				resourceLogger.Error("unable to create Kubernetes Event", slog.Any("error", err))
			}
		}
	}

	return nil
}
