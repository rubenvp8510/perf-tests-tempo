package framework

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/redhat/perf-tests-tempo/test/framework/config"

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
	restConfig    *rest.Config
	namespace     string
	ctx           context.Context
	logger        *slog.Logger
	config        *config.Config

	// Resource tracking
	mu                      sync.Mutex
	trackedCRs              []TrackedResource
	trackedClusterResources []TrackedResource
}

// Option is a function that configures the Framework
type Option func(*Framework)

// WithLogger sets a custom logger for the framework
func WithLogger(logger *slog.Logger) Option {
	return func(f *Framework) {
		f.logger = logger
	}
}

// WithConfig sets a custom configuration for the framework
func WithConfig(cfg *config.Config) Option {
	return func(f *Framework) {
		f.config = cfg
	}
}

// New creates a new Framework instance with the specified namespace.
// The context is used for all Kubernetes operations and should be cancelled
// to stop any in-progress operations.
func New(ctx context.Context, namespace string, opts ...Option) (*Framework, error) {
	if namespace == "" {
		return nil, ErrNamespaceRequired
	}

	if ctx == nil {
		ctx = context.Background()
	}

	restConfig, err := rest.InClusterConfig()
	if err != nil {
		restConfig, err = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrClusterConnection, err)
		}
	}

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create kubernetes client: %v", ErrClusterConnection, err)
	}

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create dynamic client: %v", ErrClusterConnection, err)
	}

	f := &Framework{
		client:                  client,
		dynamicClient:           dynamicClient,
		restConfig:              restConfig,
		namespace:               namespace,
		ctx:                     ctx,
		logger:                  slog.Default(),
		config:                  config.FromEnv(),
		trackedCRs:              make([]TrackedResource, 0),
		trackedClusterResources: make([]TrackedResource, 0),
	}

	// Apply options
	for _, opt := range opts {
		opt(f)
	}

	return f, nil
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
	return f.restConfig
}

// FrameworkConfig returns the framework configuration
func (f *Framework) FrameworkConfig() *config.Config {
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
