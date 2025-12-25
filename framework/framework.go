package framework

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Framework is the main deployment framework for performance tests
type Framework struct {
	client        kubernetes.Interface
	dynamicClient dynamic.Interface
	config        *rest.Config
	namespace     string
	ctx           context.Context
	logger        *slog.Logger

	// Resource tracking
	mu                      sync.Mutex
	trackedCRs              []TrackedResource
	trackedClusterResources []TrackedResource
}

// New creates a new Framework instance with the specified namespace
func New(namespace string) (*Framework, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

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

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &Framework{
		client:                  client,
		dynamicClient:           dynamicClient,
		config:                  config,
		namespace:               namespace,
		ctx:                     context.Background(),
		logger:                  slog.Default(),
		trackedCRs:              make([]TrackedResource, 0),
		trackedClusterResources: make([]TrackedResource, 0),
	}, nil
}

// Namespace returns the namespace used by this framework instance
func (f *Framework) Namespace() string {
	return f.namespace
}

// Client returns the Kubernetes client
func (f *Framework) Client() kubernetes.Interface {
	return f.client
}

// DynamicClient returns the dynamic Kubernetes client
func (f *Framework) DynamicClient() dynamic.Interface {
	return f.dynamicClient
}

// Config returns the Kubernetes REST config
func (f *Framework) Config() *rest.Config {
	return f.config
}

// Context returns the context
func (f *Framework) Context() context.Context {
	return f.ctx
}

// Logger returns the logger
func (f *Framework) Logger() *slog.Logger {
	return f.logger
}

// GetManagedLabels returns the labels that should be applied to all resources created by this framework
func (f *Framework) GetManagedLabels() map[string]string {
	return map[string]string{
		LabelManagedBy: LabelManagedByValue,
		LabelInstance:  f.namespace,
	}
}

// TrackCR adds a custom resource to the tracked resources list
func (f *Framework) TrackCR(gvr schema.GroupVersionResource, namespace, name string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.trackedCRs = append(f.trackedCRs, TrackedResource{
		GVR:       gvr,
		Namespace: namespace,
		Name:      name,
	})
}

// TrackClusterResource adds a cluster-scoped resource to the tracked resources list
func (f *Framework) TrackClusterResource(gvr schema.GroupVersionResource, name string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.trackedClusterResources = append(f.trackedClusterResources, TrackedResource{
		GVR:  gvr,
		Name: name,
	})
}

// GetTrackedCRs returns a copy of the tracked custom resources
func (f *Framework) GetTrackedCRs() []TrackedResource {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]TrackedResource, len(f.trackedCRs))
	copy(result, f.trackedCRs)
	return result
}

// GetTrackedClusterResources returns a copy of the tracked cluster-scoped resources
func (f *Framework) GetTrackedClusterResources() []TrackedResource {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]TrackedResource, len(f.trackedClusterResources))
	copy(result, f.trackedClusterResources)
	return result
}
