package kube_janitor

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	KubeDefaultListLimit = 100

	KubeNoNamespace = ""

	KubeSelectorError = "<error>"
	KubeSelectorNone  = "<none>"
)

type (
	KubeServerGvrList []metav1.GroupVersionKind
)

func (j *Janitor) kubeDiscoverGVKs() (KubeServerGvrList, error) {
	cacheKey := "kube.servergroups"

	// from cache
	if val, ok := j.cache.Get(cacheKey); ok {
		if v, ok := val.(KubeServerGvrList); ok {
			return v, nil
		}
	}

	ret := KubeServerGvrList{}

	groupsResult, resourcesResult, err := j.kubeClient.Discovery().ServerGroupsAndResources()
	if err != nil {
		return nil, err
	}

	// build GVK list
	for _, serverGroup := range groupsResult {
		for _, resourceGroup := range resourcesResult {
			if resourceGroup.GroupVersion == serverGroup.PreferredVersion.GroupVersion {
				for _, resource := range resourceGroup.APIResources {
					if slices.Contains([]string(resource.Verbs), "list") && slices.Contains([]string(resource.Verbs), "delete") {
						ret = append(ret, metav1.GroupVersionKind{
							Group:   serverGroup.Name,
							Version: serverGroup.PreferredVersion.Version,
							Kind:    resource.Name,
						})
					}
				}
			}
		}
	}

	j.cache.SetDefault(cacheKey, ret)

	return ret, nil
}

func (j *Janitor) kubeLookupGvrs(list ConfigResourceList) (ConfigResourceList, error) {
	var (
		gvrList KubeServerGvrList
		err     error
	)
	ret := []*ConfigResource{}

	for _, resource := range list {
		if resource.Group == "*" || resource.Version == "*" || resource.Kind == "*" {
			// lookup possible types
			if gvrList == nil {
				gvrList, err = j.kubeDiscoverGVKs()
				if err != nil {
					return nil, err
				}
			}

			for _, row := range gvrList {
				if resource.Group != "*" && !strings.EqualFold(resource.Group, row.Group) {
					continue
				}
				if resource.Version != "*" && !strings.EqualFold(resource.Version, row.Version) {
					continue
				}
				if resource.Kind != "*" && !strings.EqualFold(resource.Kind, row.Kind) {
					continue
				}

				clone := resource.Clone()
				clone.Group = row.Group
				clone.Version = row.Version
				clone.Kind = row.Kind

				ret = append(ret, clone)
			}
		} else {
			// no lookup needed
			ret = append(ret, resource)
		}
	}

	return ret, nil
}

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
		Limit:         j.kubePageLimit,
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
		Limit:         j.kubePageLimit,
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
