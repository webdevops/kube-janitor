package kube_janitor

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	KubeListLimit = 100
)

func (j *Janitor) kubeEachResource(ctx context.Context, gvr schema.GroupVersionResource, labelSelector string, callback func(unstructured unstructured.Unstructured) error) error {
	listOpts := metav1.ListOptions{
		Limit:         KubeListLimit,
		LabelSelector: labelSelector,
	}
	for {
		result, err := j.dynClient.Resource(gvr).List(ctx, listOpts)
		if err != nil {
			return err
		}

		for _, item := range result.Items {
			err := callback(item)
			if err != nil {
				return err
			}
		}

		if result.GetContinue() != "" {
			listOpts.Continue = result.GetContinue()
			continue
		}

		break
	}

	return nil
}
