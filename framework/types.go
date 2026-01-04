package framework

import (
	"context"
	"log/slog"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	// LabelManagedBy is the label key used to identify resources managed by the framework
	LabelManagedBy = "tempo-perf-test.io/managed-by"
	// LabelInstance is the label key used to identify the specific framework instance
	LabelInstance = "tempo-perf-test.io/instance"
	// LabelManagedByValue is the value for the managed-by label
	LabelManagedByValue = "framework"
)

// TrackedResource represents a resource created by the framework
type TrackedResource struct {
	GVR       schema.GroupVersionResource
	Namespace string
	Name      string
}

// ResourceConfig represents optional resource configuration for Tempo components
type ResourceConfig struct {
	// Profile is a preset profile name: "small", "medium", or "large"
	// If set, it will use the corresponding kustomize overlay
	Profile string

	// Custom resources (used when Profile is empty)
	Resources *corev1.ResourceRequirements

	// ReplicationFactor determines how many ingesters must acknowledge data
	// before accepting a span. Only applies to TempoStack (not monolithic).
	ReplicationFactor *int

	// Overrides contains Tempo limits configuration
	Overrides *TempoOverrides
}

// TempoOverrides defines Tempo limits and overrides
type TempoOverrides struct {
	// MaxTracesPerUser limits the number of active traces per user.
	// Set to 0 for unlimited (prevents "max live traces reached" errors).
	// If nil/not set, uses Tempo's default.
	MaxTracesPerUser *int
}

// Clients provides access to Kubernetes clients
type Clients interface {
	Client() kubernetes.Interface
	DynamicClient() dynamic.Interface
	Config() *rest.Config
	Context() context.Context
	Namespace() string
	Logger() *slog.Logger
}

// Tracker provides resource tracking capabilities
type Tracker interface {
	TrackCR(gvr schema.GroupVersionResource, namespace, name string)
	TrackClusterResource(gvr schema.GroupVersionResource, name string)
	GetManagedLabels() map[string]string
}

// FrameworkOperations combines all capabilities needed by subpackages
type FrameworkOperations interface {
	Clients
	Tracker
}
