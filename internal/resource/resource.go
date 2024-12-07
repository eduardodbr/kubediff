package resource

import (
	"context"
	"fmt"

	k8s "github.com/eduardodbr/kubediff/internal/kubernetes"
	"k8s.io/client-go/kubernetes"
)

// Apply aplies a function to every resource of a given resource type filtered by namespace and labels
func Apply(ctx context.Context, client kubernetes.Interface, resourceType string, labels []string, namespace string, fn func(item any, name string) error) error {
	switch resourceType {
	case "deployment", "deploy":
		resources, err := k8s.ListDeployments(ctx, client, k8s.ListDeploymentsOpts{
			Namespace: namespace,
			Labels:    labels,
		})
		if err != nil {
			return err
		}
		for _, deploy := range resources.Items {
			err := fn(deploy, deploy.Name)
			if err != nil {
				return err
			}
		}
	case "daemonset", "ds":
		resources, err := k8s.ListDaemonSets(ctx, client, k8s.ListDaemonSetsOpts{
			Namespace: namespace,
			Labels:    labels,
		})
		if err != nil {
			return err
		}
		for _, ds := range resources.Items {
			err := fn(ds, ds.Name)
			if err != nil {
				return err
			}
		}
	case "statefulset", "sts":
		resources, err := k8s.ListStatefulSets(ctx, client, k8s.ListStatefulSetsOpts{
			Namespace: namespace,
			Labels:    labels,
		})
		if err != nil {
			return err
		}
		for _, ds := range resources.Items {
			err := fn(ds, ds.Name)
			if err != nil {
				return err
			}
		}
	case "configmap", "cm":
		resources, err := k8s.ListConfigMaps(ctx, client, k8s.ListConfigMapsOpts{
			Namespace: namespace,
			Labels:    labels,
		})
		if err != nil {
			return err
		}
		for _, cm := range resources.Items {
			err := fn(cm, cm.Name)
			if err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("resource %s not supported", resourceType)
	}
	return nil
}
