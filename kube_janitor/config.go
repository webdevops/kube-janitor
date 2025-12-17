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
	}

	ConfigLabelSelector struct {
		metav1.LabelSelector `json:",inline"`
		selector             *string
	}
)

func NewConfig() *Config {
	return &Config{
		Ttl: &ConfigTtl{
			Resources: []*ConfigResource{},
		},
		Rules: []*ConfigRule{},
	}
}

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

func (c *ConfigTtl) Validate() error {
	if c.Label != "" {
		if strings.Contains(c.Label, " ") {
			return errors.New("label must not contain spaces")
		}
	}

	return nil
}

func (c *ConfigRule) Validate() error {
	if c.Id == "" {
		return errors.New("rules requires an id")
	}

	if len(c.Resources) == 0 {
		return errors.New("rules requires at least one resource")
	}

	return nil
}

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

func (c *ConfigResource) String() string {
	return fmt.Sprintf("%s/%s/%s", c.Group, c.Version, c.Kind)
}

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

func (selector *ConfigLabelSelector) IsEmpty() bool {
	if selector == nil || (len(selector.MatchLabels) == 0 && len(selector.MatchExpressions) == 0) {
		return true
	}

	return false
}

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
