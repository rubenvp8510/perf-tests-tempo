package framework

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// ResourceConfig represents optional resource configuration for Tempo components
type ResourceConfig struct {
	// Profile is a preset profile name: "small", "medium", or "large"
	// If set, it will use the corresponding kustomize overlay
	Profile string

	// Custom resources (used when Profile is empty)
	Resources *corev1.ResourceRequirements
}

// Framework is the main deployment framework for performance tests
type Framework struct {
	client    kubernetes.Interface
	config    *rest.Config
	namespace string
	ctx       context.Context
}

// New creates a new Framework instance with a unique namespace
func New() (*Framework, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
		if err != nil {
			return nil, fmt.Errorf("failed to get kubernetes config: %w", err)
		}
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Generate unique namespace
	namespace := generateNamespace()

	return &Framework{
		client:    client,
		config:    config,
		namespace: namespace,
		ctx:       context.Background(),
	}, nil
}

// NewWithNamespace creates a new Framework with a specific namespace
func NewWithNamespace(namespace string) (*Framework, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
		if err != nil {
			return nil, fmt.Errorf("failed to get kubernetes config: %w", err)
		}
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &Framework{
		client:    client,
		config:    config,
		namespace: namespace,
		ctx:       context.Background(),
	}, nil
}

// GetNamespace returns the namespace used by this framework instance
func (f *Framework) GetNamespace() string {
	return f.namespace
}

// GetClient returns the Kubernetes client
func (f *Framework) GetClient() kubernetes.Interface {
	return f.client
}

// GetConfig returns the Kubernetes REST config
func (f *Framework) GetConfig() *rest.Config {
	return f.config
}

// GetContext returns the context
func (f *Framework) GetContext() context.Context {
	return f.ctx
}

// applyResources applies Kubernetes resources from YAML files
// Note: This function is currently unused but kept for potential future use
func (f *Framework) applyResources(resources []runtime.Object) error {
	for _, obj := range resources {
		// Set namespace if the object supports it
		if objMeta, ok := obj.(metav1.Object); ok {
			objMeta.SetNamespace(f.namespace)
		}

		// Apply the resource using the appropriate client
		// This is a simplified version - in practice, you'd use a dynamic client
		// or the specific typed client for each resource type
		_ = obj
	}
	return nil
}

// generateNamespace generates a unique namespace name
func generateNamespace() string {
	return fmt.Sprintf("tempo-perf-test-%d", time.Now().Unix())
}

