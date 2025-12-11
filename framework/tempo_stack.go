package framework

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// SetupTempoStack deploys Tempo Stack
func (f *Framework) SetupTempoStack() error {
	// Build TempoStack CR programmatically
	stackObj := f.buildTempoStackCR()

	// Apply using dynamic client
	dynamicClient, err := dynamic.NewForConfig(f.config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	gvr := schema.GroupVersionResource{
		Group:    "tempo.grafana.com",
		Version:  "v1alpha1",
		Resource: "tempostacks",
	}

	_, err = dynamicClient.Resource(gvr).Namespace(f.namespace).Create(f.ctx, stackObj, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create TempoStack: %w", err)
	}

	// Wait for Tempo to be ready
	return f.WaitForTempoPodsReady(300 * time.Second)
}

// buildTempoStackCR builds a TempoStack CR programmatically
func (f *Framework) buildTempoStackCR() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "tempo.grafana.com/v1alpha1",
			"kind":       "TempoStack",
			"metadata": map[string]interface{}{
				"name":      "tempostack",
				"namespace": f.namespace,
			},
			"spec": map[string]interface{}{
				"nodeSelector": map[string]interface{}{
					"node-role.kubernetes.io/infra": "",
				},
				"observability": map[string]interface{}{
					"metrics": map[string]interface{}{
						"createPrometheusRules": true,
						"createServiceMonitors": true,
					},
				},
				"template": map[string]interface{}{
					"jaegerQuery": map[string]interface{}{
						"enabled": true,
					},
				},
				"storage": map[string]interface{}{
					"secret": map[string]interface{}{
						"type": "s3",
						"name": "s3-secret",
					},
				},
				"storageSize": "10Gi",
			},
		},
	}
}

