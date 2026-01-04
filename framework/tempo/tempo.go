package tempo

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redhat/perf-tests-tempo/test/framework/gvr"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// GVR aliases for backward compatibility - use gvr package directly instead
var (
	TempoMonolithicGVR = gvr.TempoMonolithic
	TempoStackGVR      = gvr.TempoStack
)

// ResourceConfig is a type alias for the framework's ResourceConfig.
// Use the framework package's ResourceConfig type for new code.
type ResourceConfig = struct {
	// Profile is a preset profile name: "small", "medium", or "large"
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

// FrameworkOperations provides access to framework capabilities needed by tempo
type FrameworkOperations interface {
	Client() kubernetes.Interface
	DynamicClient() dynamic.Interface
	Context() context.Context
	Namespace() string
	Logger() *slog.Logger
	TrackCR(gvr schema.GroupVersionResource, namespace, name string)
	GetManagedLabels() map[string]string
}

// Setup deploys Tempo (monolithic or stack) with optional resource configuration
// variant: "monolithic" or "stack"
// resources: optional resource configuration
func Setup(fw FrameworkOperations, variant string, resources *ResourceConfig) error {
	switch variant {
	case "monolithic":
		return SetupMonolithic(fw, resources)
	case "stack":
		return SetupStack(fw, resources)
	default:
		return fmt.Errorf("invalid tempo variant: %s (must be 'monolithic' or 'stack')", variant)
	}
}
