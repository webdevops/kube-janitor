package kube_janitor

import (
	"context"

	prometheusCommon "github.com/webdevops/go-common/prometheus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// runRules executes the rules from the configuration file
func (j *Janitor) runRules(ctx context.Context) error {
	metricResourceRule := prometheusCommon.NewMetricsList()

	filterFunc := func(rule *ConfigRule, resource unstructured.Unstructured) (string, bool) {
		return rule.Ttl, true
	}

	for _, rule := range j.config.Rules {
		err := j.runRule(ctx, j.logger, rule, metricResourceRule, filterFunc)
		if err != nil {
			return err
		}
	}

	metricResourceRule.GaugeSet(j.prometheus.rule)

	return nil
}
