package tempo

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/redhat/perf-tests-tempo/test/framework/wait"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	tempoapi "github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// SetupMonolithic deploys Tempo Monolithic with optional resource configuration
func SetupMonolithic(fw FrameworkOperations, resources *ResourceConfig) error {
	// Build TempoMonolithic CR using typed API
	tempoCR := buildTempoMonolithicCR(fw.Namespace(), resources)

	// Convert to unstructured for dynamic client
	unstructuredObj, err := toUnstructured(tempoCR)
	if err != nil {
		return fmt.Errorf("failed to convert TempoMonolithic to unstructured: %w", err)
	}

	// Add managed labels
	labels := unstructuredObj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	for k, v := range fw.GetManagedLabels() {
		labels[k] = v
	}
	unstructuredObj.SetLabels(labels)

	_, err = fw.DynamicClient().Resource(TempoMonolithicGVR).Namespace(fw.Namespace()).Create(fw.Context(), unstructuredObj, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create TempoMonolithic: %w", err)
	}

	// Track the created resource (even if it already exists, for cleanup)
	fw.TrackCR(TempoMonolithicGVR, fw.Namespace(), tempoCR.Name)

	// Wait for Tempo to be ready
	return wait.ForTempoPodsReady(fw, 300*time.Second)
}

// toUnstructured converts a typed object to unstructured
func toUnstructured(obj interface{}) (*unstructured.Unstructured, error) {
	content, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: content}, nil
}

// getProfileResources returns resource requirements for a preset profile
func getProfileResources(profile string) *corev1.ResourceRequirements {
	switch profile {
	case "small":
		return &corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("4Gi"),
				corev1.ResourceCPU:    resource.MustParse("500m"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("4Gi"),
				corev1.ResourceCPU:    resource.MustParse("500m"),
			},
		}
	case "medium":
		return &corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("8Gi"),
				corev1.ResourceCPU:    resource.MustParse("1000m"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("8Gi"),
				corev1.ResourceCPU:    resource.MustParse("1000m"),
			},
		}
	case "large":
		return &corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("12Gi"),
				corev1.ResourceCPU:    resource.MustParse("1500m"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("12Gi"),
				corev1.ResourceCPU:    resource.MustParse("1500m"),
			},
		}
	default:
		return nil
	}
}

// buildTempoMonolithicCR builds a TempoMonolithic CR using typed API
func buildTempoMonolithicCR(namespace string, resources *ResourceConfig) *tempoapi.TempoMonolithic {
	// Determine storage secret name
	secretName := GetStorageSecretName(nil)
	if resources != nil && resources.Storage != nil {
		secretName = GetStorageSecretName(resources.Storage)
	}

	// Build extra config as JSON
	extraConfig := map[string]interface{}{}

	// Build ingester config from ResourceConfig or use defaults
	ingesterConfig := buildIngesterExtraConfig(resources)
	if len(ingesterConfig) > 0 {
		extraConfig["ingester"] = ingesterConfig
	}

	// Add overrides if configured
	if resources != nil && resources.Overrides != nil && resources.Overrides.MaxTracesPerUser != nil {
		extraConfig["overrides"] = map[string]interface{}{
			"defaults": map[string]interface{}{
				"ingestion": map[string]interface{}{
					"max_traces_per_user": *resources.Overrides.MaxTracesPerUser,
				},
			},
		}
	}

	extraConfigJSON, _ := json.Marshal(extraConfig)

	tempoCR := &tempoapi.TempoMonolithic{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tempo.grafana.com/v1alpha1",
			Kind:       "TempoMonolithic",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simplest",
			Namespace: namespace,
		},
		Spec: tempoapi.TempoMonolithicSpec{
			Storage: &tempoapi.MonolithicStorageSpec{
				Traces: tempoapi.MonolithicTracesStorageSpec{
					Backend: tempoapi.MonolithicTracesStorageBackendS3,
					S3: &tempoapi.MonolithicTracesStorageS3Spec{
						MonolithicTracesObjectStorageSpec: tempoapi.MonolithicTracesObjectStorageSpec{
							Secret: secretName,
						},
					},
				},
			},
			Multitenancy: &tempoapi.MonolithicMultitenancySpec{
				Enabled: true,
				TenantsSpec: tempoapi.TenantsSpec{
					Mode: tempoapi.ModeOpenShift,
					Authentication: []tempoapi.AuthenticationSpec{
						{
							TenantName: "tenant-1",
							TenantID:   "tenant-1",
						},
					},
				},
			},
			JaegerUI: &tempoapi.MonolithicJaegerUISpec{
				Enabled: true,
				Route: &tempoapi.MonolithicJaegerUIRouteSpec{
					Enabled: true,
				},
			},
			Observability: &tempoapi.MonolithicObservabilitySpec{
				Metrics: &tempoapi.MonolithicObservabilityMetricsSpec{
					ServiceMonitors: &tempoapi.MonolithicObservabilityMetricsServiceMonitorsSpec{
						Enabled: true,
					},
				},
			},
			ExtraConfig: &tempoapi.ExtraConfigSpec{
				Tempo: apiextensionsv1.JSON{
					Raw: extraConfigJSON,
				},
			},
		},
	}

	// Apply resource configuration if provided
	if resources != nil {
		var resourceReqs *corev1.ResourceRequirements
		if resources.Profile != "" {
			resourceReqs = getProfileResources(resources.Profile)
		} else if resources.Resources != nil {
			resourceReqs = resources.Resources
		}
		if resourceReqs != nil {
			tempoCR.Spec.Resources = resourceReqs
		}

		// Apply node selector if provided
		if len(resources.NodeSelector) > 0 {
			tempoCR.Spec.NodeSelector = resources.NodeSelector
		}
	}

	return tempoCR
}

// buildIngesterExtraConfig builds the ingester portion of extraConfig from ResourceConfig
// If no ingester config is provided, returns a default config with max_block_duration: 10m
func buildIngesterExtraConfig(resources *ResourceConfig) map[string]interface{} {
	config := map[string]interface{}{}

	// Check if we have custom ingester config
	if resources != nil && resources.Overrides != nil && resources.Overrides.Ingester != nil {
		ing := resources.Overrides.Ingester

		if ing.MaxBlockDuration != "" {
			config["max_block_duration"] = ing.MaxBlockDuration
		}
		if ing.TraceIdlePeriod != "" {
			config["trace_idle_period"] = ing.TraceIdlePeriod
		}
		if ing.FlushCheckPeriod != "" {
			config["flush_check_period"] = ing.FlushCheckPeriod
		}
		if ing.ConcurrentFlushes != nil {
			config["concurrent_flushes"] = *ing.ConcurrentFlushes
		}

		// If any config was set, return it
		if len(config) > 0 {
			return config
		}
	}

	// Default: return minimal config with max_block_duration for better performance testing
	return map[string]interface{}{
		"max_block_duration": "10m",
	}
}
