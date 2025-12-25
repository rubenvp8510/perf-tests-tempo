package tempo

import (
	"context"
	"fmt"
	"log/slog"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// GVRs for Tempo custom resources
var (
	TempoMonolithicGVR = schema.GroupVersionResource{
		Group:    "tempo.grafana.com",
		Version:  "v1alpha1",
		Resource: "tempomonolithics",
	}
	TempoStackGVR = schema.GroupVersionResource{
		Group:    "tempo.grafana.com",
		Version:  "v1alpha1",
		Resource: "tempostacks",
	}
)

// ResourceConfig represents optional resource configuration for Tempo components
type ResourceConfig struct {
	// Profile is a preset profile name: "small", "medium", or "large"
	Profile string

	// Custom resources (used when Profile is empty)
	Resources *corev1.ResourceRequirements
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
// resources: optional resource configuration (only applies to monolithic)
func Setup(fw FrameworkOperations, variant string, resources *ResourceConfig) error {
	switch variant {
	case "monolithic":
		return SetupMonolithic(fw, resources)
	case "stack":
		if resources != nil {
			fw.Logger().Warn("resources configuration is not supported for stack variant")
		}
		return SetupStack(fw)
	default:
		return fmt.Errorf("invalid tempo variant: %s (must be 'monolithic' or 'stack')", variant)
	}
}
