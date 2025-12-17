package kube_janitor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/jmespath-community/go-jmespath"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type (
	JmesPath struct {
		Path         string
		compiledPath jmespath.JMESPath
	}
)

func (path *JmesPath) IsEmpty() bool {
	return path == nil || path.compiledPath == nil
}

func UmarshallJmesPath(ctx context.Context, path *JmesPath, data []byte) error {
	var valString string

	err := yaml.UnmarshalContext(ctx, data, &valString, yaml.Strict())
	if err != nil {
		return fmt.Errorf(`failed to parse jmespath as string: %w`, err)
	}

	valString = strings.TrimSpace(valString)

	if valString != "" {
		compiledPath, err := jmespath.Compile(valString)
		if err != nil {
			return fmt.Errorf(`failed to compile jmespath "%s": %w`, valString, err)
		}

		path.Path = valString
		path.compiledPath = compiledPath
	}

	return nil
}

//
// func (path *JmesPath) UnmarshalJSON(data []byte) error {
// 	var valString string
//
// 	err := json.Unmarshal(data, &valString)
// 	if err != nil {
// 		return fmt.Errorf(`failed to parse jmespath as string: %w`, err)
// 	}
//
// 	valString = strings.TrimSpace(valString)
//
// 	if valString != "" {
// 		compiledPath, err := jmespath.Compile(valString)
// 		if err != nil {
// 			return fmt.Errorf(`failed to compile jmespath "%s": %w`, valString, err)
// 		}
//
// 		path.Path = valString
// 		path.compiledPath = compiledPath
// 	}
//
// 	return nil
// }

func (j *Janitor) fetchResourceValueByFromJmesPath(resource unstructured.Unstructured, jmesPath *JmesPath) (interface{}, error) {
	resourceRaw, err := resource.MarshalJSON()
	if err != nil {
		return true, err
	}

	var data any
	err = json.Unmarshal(resourceRaw, &data)
	if err != nil {
		return true, err
	}

	// check if resource is valid by JMES path
	result, err := jmesPath.compiledPath.Search(data)
	if err != nil {
		return true, err
	}

	return result, nil
}

func (j *Janitor) checkResourceIsSkippedFromJmesPath(resource unstructured.Unstructured, jmesPath *JmesPath) (bool, error) {
	result, err := j.fetchResourceValueByFromJmesPath(resource, jmesPath)
	if err != nil {
		return true, err
	}

	switch v := result.(type) {
	case string:
		// skip if string is empty
		if len(v) == 0 {
			return true, nil
		}
	case bool:
		// skip if false (not selected)
		return !v, nil
	case nil:
		// nil? jmes path didn't find anything? better skip the resource
		return true, nil
	}

	return false, nil
}

func (j *Janitor) parseResourceTimestampFromJmesPath(resource unstructured.Unstructured, jmesPath *JmesPath) (*time.Time, error) {
	result, err := j.fetchResourceValueByFromJmesPath(resource, jmesPath)
	fmt.Println(jmesPath.Path, result, err)
	if err != nil {
		return nil, err
	}

	switch v := result.(type) {
	case string:
		// skip if string is empty
		if timestamp := j.parseTimestamp(v); timestamp != nil {
			return timestamp, nil
		}
	}

	return nil, nil
}
