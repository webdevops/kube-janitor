package kube_janitor

import (
	"errors"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

type (
	Config struct {
		Label     string             `json:"label"`
		Resources []*ConfigResources `json:"resources"`
	}

	ConfigResources struct {
		Group   string `json:"group"`
		Version string `json:"version"`
		Kind    string `json:"kind"`
	}
)

func NewConfig() *Config {
	return &Config{
		Resources: []*ConfigResources{},
	}
}

func (r *ConfigResources) String() string {
	return r.AsGVR().String()
}

func (r *ConfigResources) AsGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    r.Group,
		Version:  r.Version,
		Resource: r.Kind,
	}
}

func (r *Config) Validate() error {
	if r.Label == "" {
		return errors.New("label is required")
	}

	if strings.Contains(r.Label, " ") {
		return errors.New("label must not contain spaces")
	}

	return nil
}
