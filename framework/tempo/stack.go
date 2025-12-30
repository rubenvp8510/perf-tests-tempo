package tempo

import (
	"fmt"
	"time"

	"github.com/redhat/perf-tests-tempo/test/framework/wait"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tempoapi "github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

// SetupStack deploys Tempo Stack
func SetupStack(fw FrameworkOperations) error {
	// Build TempoStack CR using typed API
	stackCR := buildTempoStackCR(fw.Namespace())

	// Convert to unstructured for dynamic client
	unstructuredObj, err := toUnstructured(stackCR)
	if err != nil {
		return fmt.Errorf("failed to convert TempoStack to unstructured: %w", err)
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

	_, err = fw.DynamicClient().Resource(TempoStackGVR).Namespace(fw.Namespace()).Create(fw.Context(), unstructuredObj, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create TempoStack: %w", err)
	}

	// Track the created resource
	fw.TrackCR(TempoStackGVR, fw.Namespace(), stackCR.Name)

	// Wait for Tempo to be ready
	return wait.ForTempoPodsReady(fw, 300*time.Second)
}

// buildTempoStackCR builds a TempoStack CR using typed API
func buildTempoStackCR(namespace string) *tempoapi.TempoStack {
	storageSize := resource.MustParse("10Gi")

	return &tempoapi.TempoStack{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tempo.grafana.com/v1alpha1",
			Kind:       "TempoStack",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempostack",
			Namespace: namespace,
		},
		Spec: tempoapi.TempoStackSpec{
			Template: tempoapi.TempoTemplateSpec{
				QueryFrontend: tempoapi.TempoQueryFrontendSpec{
					JaegerQuery: tempoapi.JaegerQuerySpec{
						Enabled: true,
					},
				},
			},
			Storage: tempoapi.ObjectStorageSpec{
				Secret: tempoapi.ObjectStorageSecretSpec{
					Type: tempoapi.ObjectStorageSecretS3,
					Name: "minio",
				},
			},
			StorageSize: storageSize,
			Observability: tempoapi.ObservabilitySpec{
				Metrics: tempoapi.MetricsConfigSpec{
					CreatePrometheusRules: true,
					CreateServiceMonitors: true,
				},
			},
		},
	}
}
