package kube_janitor

import (
	"errors"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

type (
	Config struct {
		Ttl *ConfigTtl `json:"ttl"`
	}

	ConfigTtl struct {
		Annotation string             `json:"annotation"`
		Label      string             `json:"label"`
		Resources  []*ConfigResources `json:"resources"`
	}

	ConfigResources struct {
		Group   string `json:"group"`
		Version string `json:"version"`
		Kind    string `json:"kind"`
	}
)

func NewConfig() *Config {
	return &Config{
		Ttl: &ConfigTtl{
			Resources: []*ConfigResources{},
		},
	}
}

func (c *Config) Validate() error {
	if err := c.Ttl.Validate(); err != nil {
		return err
	}

	return nil
}

func (c *ConfigTtl) Validate() error {
	if c.Label == "" && c.Annotation == "" {
		return errors.New("label or annotation is required")
	}

	if c.Label != "" {
		if strings.Contains(c.Label, " ") {
			return errors.New("label must not contain spaces")
		}
	}

	return nil
}

func (c *ConfigResources) String() string {
	return c.AsGVR().String()
}

func (c *ConfigResources) AsGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    c.Group,
		Version:  c.Version,
		Resource: c.Kind,
	}
}
