package kube_janitor

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	KubeListLimit = 100

	KubeNoNamespace = ""

	KubeSelectorError = "<error>"
	KubeSelectorNone  = "<none>"
)

func (j *Janitor) kubeBuildLabelSelector(selector *metav1.LabelSelector) (string, error) {
	// no selector
	if selector == nil {
		return "", nil
	}

	compiledSelector := metav1.FormatLabelSelector(selector)
	if strings.EqualFold(compiledSelector, KubeSelectorError) {
		return "", fmt.Errorf(`unable to compile Kubernetes selector for resource: %v`, selector)
	}

	if !strings.EqualFold(compiledSelector, KubeSelectorNone) {
		return compiledSelector, nil
	}

	return "", nil
}

func (j *Janitor) kubeEachNamespace(ctx context.Context, selector *metav1.LabelSelector, callback func(namespace corev1.Namespace) error) error {
	labelSelector, err := j.kubeBuildLabelSelector(selector)
	if err != nil {
		return err
	}

	listOpts := metav1.ListOptions{
		Limit:         KubeListLimit,
		LabelSelector: labelSelector,
	}
	for {
		result, err := j.kubeClient.CoreV1().Namespaces().List(ctx, listOpts)
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

func (j *Janitor) kubeEachResource(ctx context.Context, gvr schema.GroupVersionResource, namespace string, selector *metav1.LabelSelector, callback func(unstructured unstructured.Unstructured) error) error {
	labelSelector, err := j.kubeBuildLabelSelector(selector)
	if err != nil {
		return err
	}

	listOpts := metav1.ListOptions{
		Limit:         KubeListLimit,
		LabelSelector: labelSelector,
	}
	for {
		var (
			result *unstructured.UnstructuredList
			err    error
		)

		if namespace != KubeNoNamespace {
			// get by namespace
			result, err = j.dynClient.Resource(gvr).Namespace(namespace).List(ctx, listOpts)
		} else {
			// get all
			result, err = j.dynClient.Resource(gvr).List(ctx, listOpts)
		}

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

func (j *Janitor) kubeCreateEventFromResource(ctx context.Context, namespace string, resource unstructured.Unstructured, message, reason string) error {
	timestamp := metav1.Time{Time: time.Now()}

	event := corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kube-janitor-",
			Namespace:    resource.GetNamespace(),
		},
		ReportingInstance: "kube-janitor",
		InvolvedObject: corev1.ObjectReference{
			APIVersion: resource.GetAPIVersion(),
			Kind:       resource.GetKind(),
			Namespace:  resource.GetNamespace(),
			Name:       resource.GetName(),
			UID:        resource.GetUID(),
		},
		Reason:  reason,
		Message: message,
		Source: corev1.EventSource{
			Component: "kube-janitor",
		},
		FirstTimestamp:      timestamp,
		LastTimestamp:       timestamp,
		Count:               1,
		Type:                "Normal",
		Series:              nil,
		Action:              "Deleted",
		Related:             nil,
		ReportingController: "kube-janitor",
	}

	_, err := j.kubeClient.CoreV1().Events(namespace).Create(ctx, &event, metav1.CreateOptions{})
	return err
}
