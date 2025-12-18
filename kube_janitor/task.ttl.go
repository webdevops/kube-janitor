package kube_janitor

import (
	"context"
	"strings"

	prometheusCommon "github.com/webdevops/go-common/prometheus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// runTtlResources executes the ttl rule from the configuration file
func (j *Janitor) runTtlResources(ctx context.Context) error {
	metricResourceTtl := prometheusCommon.NewMetricsList()

	filterFunc := func(rule *ConfigRule, resource unstructured.Unstructured) (string, bool) {
		ttlValue := ""
		if j.config.Ttl.Annotation != "" {
			// get from meta.annotations
			if val, exists := resource.GetAnnotations()[j.config.Ttl.Annotation]; exists {
				ttlValue = val
			}
		} else if j.config.Ttl.Label != "" {
			// get from meta.labels
			if val, exists := resource.GetLabels()[j.config.Ttl.Label]; exists {
				ttlValue = val
			}
		}

		ttlValue = strings.TrimSpace(ttlValue)
		if ttlValue != "" {
			return ttlValue, true
		}

		return "", false
	}

	// faked rule for ttl handling
	rule := &ConfigRule{
		Id:        RuleIdInternalTTL,
		Resources: j.config.Ttl.Resources,
	}

	err := j.runRule(ctx, j.logger, rule, metricResourceTtl, filterFunc)
	if err != nil {
		return err
	}

	metricResourceTtl.GaugeSet(j.prometheus.ttl)

	return nil
}
