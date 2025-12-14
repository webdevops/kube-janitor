package kube_janitor

import (
	"errors"
	"fmt"
	"strings"

	jmespath "github.com/jmespath-community/go-jmespath"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type (
	Config struct {
		Ttl   *ConfigTtl    `json:"ttl"`
		Rules []*ConfigRule `json:"rules"`
	}

	ConfigTtl struct {
		Annotation string            `json:"annotation"`
		Label      string            `json:"label"`
		Resources  []*ConfigResource `json:"resources"`
	}

	ConfigResource struct {
		Group            string                `json:"group"`
		Version          string                `json:"version"`
		Kind             string                `json:"kind"`
		Selector         *metav1.LabelSelector `json:"selector"`
		JmesPath         string                `json:"jmespath"`
		jmesPathcompiled jmespath.JMESPath
	}

	ConfigRule struct {
		Id                string                `json:"id"`
		Resources         []*ConfigResource     `json:"resources"`
		NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector"`
		Ttl               string                `json:"ttl"`
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

func (c *ConfigResource) CompiledJmesPath() jmespath.JMESPath {
	if c.jmesPathcompiled == nil {
		compiledPath, err := jmespath.Compile(c.JmesPath)
		if err != nil {
			panic(fmt.Errorf(`failed to compile jmespath "%s": %w`, c.JmesPath, err))
		}

		c.jmesPathcompiled = compiledPath
	}

	return c.jmesPathcompiled
}

func (c *ConfigResource) String() string {
	return c.AsGVR().String()
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
