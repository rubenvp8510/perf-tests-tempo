package framework

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// SetupTempoMonolithic deploys Tempo Monolithic with optional resource configuration
func (f *Framework) SetupTempoMonolithic(resources *ResourceConfig) error {
	// Build TempoMonolithic CR programmatically
	tempoObj := f.buildTempoMonolithicCR(resources)

	// Apply using dynamic client
	dynamicClient, err := dynamic.NewForConfig(f.config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	gvr := schema.GroupVersionResource{
		Group:    "tempo.grafana.com",
		Version:  "v1alpha1",
		Resource: "tempomonolithics",
	}

	_, err = dynamicClient.Resource(gvr).Namespace(f.namespace).Create(f.ctx, tempoObj, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create TempoMonolithic: %w", err)
	}

	// Wait for Tempo to be ready
	return f.WaitForTempoPodsReady(300 * time.Second)
}

// applyTempoResources applies resource configuration to Tempo CR
func (f *Framework) applyTempoResources(tempoObj *unstructured.Unstructured, resources *ResourceConfig) error {
	spec, found, err := unstructured.NestedMap(tempoObj.Object, "spec")
	if !found || err != nil {
		return fmt.Errorf("failed to get spec: %w", err)
	}

	var resourceReqs *corev1.ResourceRequirements

	if resources.Profile != "" {
		// Use preset profile
		resourceReqs = getProfileResources(resources.Profile)
	} else if resources.Resources != nil {
		// Use custom resources
		resourceReqs = resources.Resources
	}

	if resourceReqs != nil {
		resourceMap := map[string]interface{}{}

		// Set limits
		if resourceReqs.Limits != nil {
			limits := map[string]interface{}{}
			if cpu := resourceReqs.Limits[corev1.ResourceCPU]; !cpu.IsZero() {
				limits["cpu"] = cpu.String()
			}
			if memory := resourceReqs.Limits[corev1.ResourceMemory]; !memory.IsZero() {
				limits["memory"] = memory.String()
			}
			if len(limits) > 0 {
				resourceMap["limits"] = limits
			}
		}

		// Set requests
		if resourceReqs.Requests != nil {
			requests := map[string]interface{}{}
			if cpu := resourceReqs.Requests[corev1.ResourceCPU]; !cpu.IsZero() {
				requests["cpu"] = cpu.String()
			}
			if memory := resourceReqs.Requests[corev1.ResourceMemory]; !memory.IsZero() {
				requests["memory"] = memory.String()
			}
			if len(requests) > 0 {
				resourceMap["requests"] = requests
			}
		}

		if len(resourceMap) > 0 {
			spec["resources"] = resourceMap
			tempoObj.Object["spec"] = spec
		}
	}

	return nil
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

// buildTempoMonolithicCR builds a TempoMonolithic CR programmatically
func (f *Framework) buildTempoMonolithicCR(resources *ResourceConfig) *unstructured.Unstructured {
	tempoObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "tempo.grafana.com/v1alpha1",
			"kind":       "TempoMonolithic",
			"metadata": map[string]interface{}{
				"name":      "simplest",
				"namespace": f.namespace,
			},
			"spec": map[string]interface{}{
				"storage": map[string]interface{}{
					"traces": map[string]interface{}{
						"backend": "s3",
						"s3": map[string]interface{}{
							"secret": "minio",
						},
					},
				},
				"multitenancy": map[string]interface{}{
					"enabled": true,
					"mode":    "openshift",
					"authentication": []interface{}{
						map[string]interface{}{
							"tenantName": "tenant-1",
							"tenantId":   "tenant-1",
						},
					},
				},
				"jaegerui": map[string]interface{}{
					"enabled": true,
					"route": map[string]interface{}{
						"enabled": true,
					},
				},
				"observability": map[string]interface{}{
					"metrics": map[string]interface{}{
						"serviceMonitors": map[string]interface{}{
							"enabled": true,
						},
					},
				},
				"extraConfig": map[string]interface{}{
					"tempo": map[string]interface{}{
						"ingester": map[string]interface{}{
							"max_block_duration": "10m",
						},
						"overrides": map[string]interface{}{
							"defaults": map[string]interface{}{
								"ingestion": map[string]interface{}{
									"max_traces_per_user": 0,
								},
							},
						},
					},
				},
			},
		},
	}

	// Apply resource configuration if provided
	if resources != nil {
		f.applyTempoResources(tempoObj, resources)
	}

	return tempoObj
}
