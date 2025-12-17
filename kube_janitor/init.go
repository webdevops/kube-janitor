package kube_janitor

import (
	"github.com/goccy/go-yaml"
)

func init() {
	yaml.RegisterCustomUnmarshalerContext(UmarshallJmesPath)
}
