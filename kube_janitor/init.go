package kube_janitor

import (
	"github.com/goccy/go-yaml"
)

// init registers all yaml Unmarshaler
func init() {
	yaml.RegisterCustomUnmarshalerContext(UnmarshallJmesPath)
}
