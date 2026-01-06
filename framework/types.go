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

// BuildNodeAntiAffinity creates a NodeAffinity that prevents scheduling on nodes
// matching the given selector. This is used to ensure generator pods (k6, MinIO, OTel)
// don't run on the same nodes as Tempo.
//
// For labels with empty values (e.g., "node-role.kubernetes.io/infra": ""),
// it uses DoesNotExist operator.
// For labels with non-empty values, it uses NotIn operator.
func BuildNodeAntiAffinity(nodeSelector map[string]string) *corev1.NodeAffinity {
	if len(nodeSelector) == 0 {
		return nil
	}

	var matchExpressions []corev1.NodeSelectorRequirement

	for key, value := range nodeSelector {
		var req corev1.NodeSelectorRequirement
		if value == "" {
			// For empty value selectors (like node roles), use DoesNotExist
			req = corev1.NodeSelectorRequirement{
				Key:      key,
				Operator: corev1.NodeSelectorOpDoesNotExist,
			}
		} else {
			// For non-empty values, use NotIn
			req = corev1.NodeSelectorRequirement{
				Key:      key,
				Operator: corev1.NodeSelectorOpNotIn,
				Values:   []string{value},
			}
		}
		matchExpressions = append(matchExpressions, req)
	}

	return &corev1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
			NodeSelectorTerms: []corev1.NodeSelectorTerm{
				{
					MatchExpressions: matchExpressions,
				},
			},
		},
	}
}

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

	// NodeSelector is a selector which must match a node's labels for pods to be scheduled.
	// Example: {"node-role.kubernetes.io/infra": ""}
	NodeSelector map[string]string

	// Storage configures S3-compatible storage for Tempo.
	// If nil, uses default MinIO setup (requires calling SetupMinIO first).
	Storage *StorageConfig
}

// StorageConfig defines S3-compatible storage configuration
type StorageConfig struct {
	// Type is the storage type: "minio" (default, in-cluster) or "s3" (external AWS S3)
	Type string

	// SecretName is the name of the secret containing S3 credentials.
	// If empty, defaults to "minio" for minio type or "tempo-s3" for s3 type.
	SecretName string

	// Endpoint is the S3 endpoint URL (required for minio, optional for AWS S3)
	// For AWS S3, leave empty to use the default AWS endpoint.
	// Example: "http://minio.namespace.svc.cluster.local:9000" or "https://s3.us-east-2.amazonaws.com"
	Endpoint string

	// Bucket is the S3 bucket name (required)
	Bucket string

	// Region is the AWS region (required for AWS S3, ignored for minio)
	Region string

	// AccessKeyID is the AWS access key ID (required)
	AccessKeyID string

	// SecretAccessKey is the AWS secret access key (required)
	SecretAccessKey string

	// Insecure allows insecure (non-TLS) connections to the S3 endpoint
	Insecure bool
}

// TempoOverrides defines Tempo limits and overrides
type TempoOverrides struct {
	// MaxTracesPerUser limits the number of active traces per user.
	// Set to 0 for unlimited (prevents "max live traces reached" errors).
	// If nil/not set, uses Tempo's default.
	MaxTracesPerUser *int

	// Ingester contains ingester-specific tuning parameters
	Ingester *IngesterConfig
}

// IngesterConfig defines ingester tuning parameters for performance testing
type IngesterConfig struct {
	// FlushCheckPeriod is the interval for checking flush readiness (e.g., "10s")
	FlushCheckPeriod string

	// TraceIdlePeriod is the time before flushing an idle trace to WAL (e.g., "5s")
	TraceIdlePeriod string

	// MaxBlockDuration is the maximum time before cutting a block (e.g., "30m")
	MaxBlockDuration string

	// ConcurrentFlushes is the number of parallel flush operations
	ConcurrentFlushes *int
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

// NodeScheduling provides access to node scheduling configuration
type NodeScheduling interface {
	// GetTempoNodeSelector returns the node selector used for Tempo pods.
	// Used to create anti-affinity for generator pods (k6, MinIO, OTel).
	GetTempoNodeSelector() map[string]string
}

// FrameworkOperations combines all capabilities needed by subpackages
type FrameworkOperations interface {
	Clients
	Tracker
	NodeScheduling
}
