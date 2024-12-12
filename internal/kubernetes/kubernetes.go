// Package kubernetes implements an access layer to kubernetes API
package kubernetes

import (
	"context"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// CreateClients creates a kubernetes clientSet for each context provided in `contexts`. Uses
// the kubeconfig file provided by `kubeconfig`
func CreateClients(kubeconfig string, contexts []string) (map[string]*kubernetes.Clientset, error) {
	clients := make(map[string]*kubernetes.Clientset, len(contexts))
	for _, context := range contexts {
		k8sClient, err := CreateClient(kubeconfig, context)
		if err != nil {
			return nil, err
		}
		clients[context] = k8sClient
	}
	return clients, nil
}

// CreateClient creates a kubernetes clientSet for the context provided in `context`. Uses
// the kubeconfig file provided by `kubeconfig`
func CreateClient(kubeconfig, context string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
		&clientcmd.ConfigOverrides{CurrentContext: context}).
		ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create client config: %v", err)
	}
	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %v", err)
	}
	return clientset, nil
}

type ListDeploymentsOpts struct {
	Namespace string
	Labels    []string
	Timeout   time.Duration
}

func ListDeployments(ctx context.Context, k kubernetes.Interface, opts ListDeploymentsOpts) (*appsv1.DeploymentList, error) {
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}
	deployList, err := k.AppsV1().Deployments(opts.Namespace).List(ctx, v1.ListOptions{
		LabelSelector: joinLabels(opts.Labels),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %v", err)
	}
	return deployList, nil
}

type ListDaemonSetsOpts struct {
	Namespace string
	Labels    []string
	Timeout   time.Duration
}

func ListDaemonSets(ctx context.Context, k kubernetes.Interface, opts ListDaemonSetsOpts) (*appsv1.DaemonSetList, error) {
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}
	daemonSetList, err := k.AppsV1().DaemonSets(opts.Namespace).List(ctx, v1.ListOptions{
		LabelSelector: joinLabels(opts.Labels),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %v", err)
	}
	return daemonSetList, nil
}

type ListStatefulSetsOpts struct {
	Namespace string
	Labels    []string
	Timeout   time.Duration
}

func ListStatefulSets(ctx context.Context, k kubernetes.Interface, opts ListStatefulSetsOpts) (*appsv1.StatefulSetList, error) {
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}
	statefulSetList, err := k.AppsV1().StatefulSets(opts.Namespace).List(ctx, v1.ListOptions{
		LabelSelector: joinLabels(opts.Labels),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %v", err)
	}
	return statefulSetList, nil
}

type ListConfigMapsOpts struct {
	Namespace string
	Labels    []string
	Timeout   time.Duration
}

func ListConfigMaps(ctx context.Context, k kubernetes.Interface, opts ListConfigMapsOpts) (*corev1.ConfigMapList, error) {
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	configmapList, err := k.CoreV1().ConfigMaps(opts.Namespace).List(ctx, v1.ListOptions{
		LabelSelector: joinLabels(opts.Labels),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %v", err)
	}
	return configmapList, nil
}

// joinLabels joins the labels into a single string
func joinLabels(labels []string) string {
	return strings.Join(labels, ",")
}
