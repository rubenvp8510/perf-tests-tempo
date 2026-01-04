package tempo

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redhat/perf-tests-tempo/test/framework/gvr"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	// NodeSelector is a selector which must match a node's labels for pods to be scheduled.
	// Example: {"node-role.kubernetes.io/infra": ""}
	NodeSelector map[string]string

	// Storage configures S3-compatible storage for Tempo.
	// If nil, uses default MinIO setup (requires calling SetupMinIO first).
	Storage *StorageConfig
}

// TempoOverrides defines Tempo limits and overrides
type TempoOverrides struct {
	// MaxTracesPerUser limits the number of active traces per user.
	// Set to 0 for unlimited (prevents "max live traces reached" errors).
	// If nil/not set, uses Tempo's default.
	MaxTracesPerUser *int
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
	// Set up external S3 storage secret if configured
	if resources != nil && resources.Storage != nil && resources.Storage.Type == "s3" {
		if err := SetupStorageSecret(fw, resources.Storage); err != nil {
			return fmt.Errorf("failed to setup storage secret: %w", err)
		}
	}

	switch variant {
	case "monolithic":
		return SetupMonolithic(fw, resources)
	case "stack":
		return SetupStack(fw, resources)
	default:
		return fmt.Errorf("invalid tempo variant: %s (must be 'monolithic' or 'stack')", variant)
	}
}

// SetupStorageSecret creates the S3 storage secret for external S3 storage
func SetupStorageSecret(fw FrameworkOperations, storage *StorageConfig) error {
	if storage == nil {
		return fmt.Errorf("storage config is required")
	}

	secretName := storage.SecretName
	if secretName == "" {
		if storage.Type == "s3" {
			secretName = "tempo-s3"
		} else {
			secretName = "minio"
		}
	}

	// Build secret data
	secretData := map[string]string{
		"bucket":            storage.Bucket,
		"access_key_id":     storage.AccessKeyID,
		"access_key_secret": storage.SecretAccessKey,
	}

	// Add endpoint if specified (required for minio, optional for AWS S3)
	if storage.Endpoint != "" {
		secretData["endpoint"] = storage.Endpoint
	}

	// Add region if specified (required for AWS S3)
	if storage.Region != "" {
		secretData["region"] = storage.Region
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: fw.Namespace(),
			Labels:    fw.GetManagedLabels(),
		},
		StringData: secretData,
		Type:       corev1.SecretTypeOpaque,
	}

	_, err := fw.Client().CoreV1().Secrets(fw.Namespace()).Create(fw.Context(), secret, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create S3 secret: %w", err)
	}

	fw.Logger().Info("Created S3 storage secret", "name", secretName, "bucket", storage.Bucket)
	return nil
}

// GetStorageSecretName returns the secret name for the given storage config
func GetStorageSecretName(storage *StorageConfig) string {
	if storage == nil {
		return "minio"
	}
	if storage.SecretName != "" {
		return storage.SecretName
	}
	if storage.Type == "s3" {
		return "tempo-s3"
	}
	return "minio"
}
