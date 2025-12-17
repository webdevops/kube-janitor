package kube_janitor

import (
	"context"
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

	KubeVerbGet    = "get"
	KubeVerbList   = "list"
	KubeVerbDelete = "delete"
)

type (
	KubeServerGroupVersionKindList []KubeServerGroupVersionKind

	KubeServerGroupVersionKind struct {
		metav1.GroupVersionKind
		Namespaced bool
	}
)

func (j *Janitor) kubeDiscoverGVKs() (KubeServerGroupVersionKindList, error) {
	cacheKey := "kube.servergroups"

	// from cache
	if val, ok := j.cache.Get(cacheKey); ok {
		if v, ok := val.(KubeServerGroupVersionKindList); ok {
			return v, nil
		}
	}

	ret := KubeServerGroupVersionKindList{}

	j.logger.Info("discovering Kubernetes api groups and resources (GroupVersionKind)")

	apiGroupsResult, apiResourcesResult, err := j.kubeClient.Discovery().ServerGroupsAndResources()
	if err != nil {
		return nil, err
	}

	// build GVK list
	// loop though all available api groups
	for _, apiGroup := range apiGroupsResult {
		// go though all the api resources (by api group)
		for _, apiResourceGroup := range apiResourcesResult {
			// only use preferred version, we don't care about the others (old) versions
			if apiResourceGroup.GroupVersion == apiGroup.PreferredVersion.GroupVersion {
				for _, resource := range apiResourceGroup.APIResources {
					// only select resources if we can get, list and delete it
					// (otherwise it doesn't make sense)
					if slices.Contains([]string(resource.Verbs), KubeVerbGet) &&
						slices.Contains([]string(resource.Verbs), KubeVerbList) &&
						slices.Contains([]string(resource.Verbs), KubeVerbDelete) {
						ret = append(ret, KubeServerGroupVersionKind{
							GroupVersionKind: metav1.GroupVersionKind{
								Group:   apiGroup.Name,
								Version: apiGroup.PreferredVersion.Version,
								Kind:    resource.Name,
							},
							Namespaced: resource.Namespaced,
						})
					}
				}
			}
		}
	}

	j.cache.SetDefault(cacheKey, ret)

	return ret, nil
}

func (j *Janitor) kubeLookupGvrs(list ConfigResourceList, namespaced bool) (ConfigResourceList, error) {
	var (
		gvrList KubeServerGroupVersionKindList
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

			for _, serverGroupVersionKind := range gvrList {
				if namespaced && !serverGroupVersionKind.Namespaced {
					continue
				}

				if resource.Group != "*" && !strings.EqualFold(resource.Group, serverGroupVersionKind.Group) {
					continue
				}
				if resource.Version != "*" && !strings.EqualFold(resource.Version, serverGroupVersionKind.Version) {
					continue
				}
				if resource.Kind != "*" && !strings.EqualFold(resource.Kind, serverGroupVersionKind.Kind) {
					continue
				}

				clone := resource.Clone()
				clone.Group = serverGroupVersionKind.Group
				clone.Version = serverGroupVersionKind.Version
				clone.Kind = serverGroupVersionKind.Kind

				ret = append(ret, clone)
			}
		} else {
			// no lookup needed
			ret = append(ret, resource)
		}
	}

	return ret, nil
}

func (j *Janitor) kubeEachNamespace(ctx context.Context, selector ConfigLabelSelector, callback func(namespace corev1.Namespace) error) error {
	labelSelector, err := selector.Compile()
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

func (j *Janitor) kubeEachResource(ctx context.Context, gvr schema.GroupVersionResource, namespace string, selector ConfigLabelSelector, callback func(unstructured unstructured.Unstructured) error) error {
	labelSelector, err := selector.Compile()
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
