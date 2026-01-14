package kube_janitor

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type (
	Config struct {
		Ttl   *ConfigTtl    `json:"ttl"`
		Rules []*ConfigRule `json:"rules"`
	}

	ConfigTtl struct {
		Annotation string             `json:"annotation"`
		Label      string             `json:"label"`
		Resources  ConfigResourceList `json:"resources"`

		DeleteOptions ConfigRuleDeleteOptions `json:"deleteOptions"`
	}

	ConfigResourceList []*ConfigResource

	ConfigResource struct {
		Group         string              `json:"group"`
		Version       string              `json:"version"`
		Kind          string              `json:"kind"`
		Selector      ConfigLabelSelector `json:"selector"`
		TimestampPath *JmesPath           `json:"timestampPath"`
		FilterPath    *JmesPath           `json:"filterPath"`
	}

	ConfigRule struct {
		Id                string              `json:"id"`
		Resources         ConfigResourceList  `json:"resources"`
		NamespaceSelector ConfigLabelSelector `json:"namespaceSelector"`
		Ttl               string              `json:"ttl"`

		DeleteOptions ConfigRuleDeleteOptions `json:"deleteOptions"`
	}

	ConfigRuleDeleteOptions struct {
		PropagationPolicy  *ConfigRuleDeletePropagationPolicy `json:"propagationPolicy"`
		GracePeriodSeconds *int64                             `json:"gracePeriodSeconds"`
	}

	ConfigRuleDeletePropagationPolicy metav1.DeletionPropagation

	ConfigLabelSelector struct {
		metav1.LabelSelector `json:",inline"`
		selector             *string
	}
)

// NewConfig create a new config instance
func NewConfig() *Config {
	return &Config{
		Ttl: &ConfigTtl{
			Resources: []*ConfigResource{},
		},
		Rules: []*ConfigRule{},
	}
}

// Validate validates the config and all sub objects
func (c *Config) Validate() error {
	if err := c.Ttl.Validate(); err != nil {
		return err
	}

	for _, rule := range c.Rules {
		if err := rule.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate validates the ttl rule
func (c *ConfigTtl) Validate() error {
	if c.Label != "" {
		if strings.Contains(c.Label, " ") {
			return errors.New("label must not contain spaces")
		}
	}

	if err := c.DeleteOptions.PropagationPolicy.validate(); err != nil {
		return err
	}

	return nil
}

// Validate validates the config rule
func (c *ConfigRule) Validate() error {
	if c.Id == "" {
		return errors.New("rules requires an id")
	}

	if len(c.Resources) == 0 {
		return errors.New("rules requires at least one resource")
	}

	if err := c.DeleteOptions.PropagationPolicy.validate(); err != nil {
		return err
	}

	return nil
}

// Clone clones the object
func (c *ConfigResource) Clone() *ConfigResource {
	ret := ConfigResource{}
	buf := bytes.Buffer{}
	err := gob.NewEncoder(&buf).Encode(c)
	if err != nil {
		panic(err)
	}
	err = gob.NewDecoder(&buf).Decode(&ret)
	if err != nil {
		panic(err)
	}
	return &ret
}

// String creates <group>/<version>/<kind> representation
func (c *ConfigResource) String() string {
	return fmt.Sprintf("%s/%s/%s", c.Group, c.Version, c.Kind)
}

// AsGVR converts GVK to GVR
func (c *ConfigResource) AsGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    c.Group,
		Version:  c.Version,
		Resource: c.Kind,
	}
}

func (c *ConfigRule) String() string {
	return c.Id
}

// IsEmpty checks if the selector is empty/defined or not
func (selector *ConfigLabelSelector) IsEmpty() bool {
	if selector == nil || (len(selector.MatchLabels) == 0 && len(selector.MatchExpressions) == 0) {
		return true
	}

	return false
}

// Compile compiles the label selector struct to a string
func (selector *ConfigLabelSelector) Compile() (string, error) {
	// no selector
	if selector.IsEmpty() {
		return "", nil
	}

	if selector.selector == nil {
		labelSelector := metav1.FormatLabelSelector(&selector.LabelSelector)

		switch labelSelector {
		case KubeSelectorError:
			return "", fmt.Errorf(`unable to compile Kubernetes selector for resource: %v`, selector)
		case KubeSelectorNone:
			labelSelector = ""
		}

		selector.selector = &labelSelector
	}

	return *selector.selector, nil
}

// validate validates the deletion PropagationPolicy
func (p *ConfigRuleDeletePropagationPolicy) validate() error {
	if p == nil {
		return nil
	}

	switch metav1.DeletionPropagation(*p) {
	case "", metav1.DeletePropagationForeground, metav1.DeletePropagationBackground, metav1.DeletePropagationOrphan:
		// ok
	default:
		return errors.New("propagation policy must be Foreground, Background, or Orphan")
	}

	return nil
}
