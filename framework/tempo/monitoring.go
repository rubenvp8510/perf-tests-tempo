package tempo

import (
	"fmt"
	"time"

	"github.com/redhat/perf-tests-tempo/test/framework/gvr"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ServiceMonitorStatus represents the status of ServiceMonitor verification
type ServiceMonitorStatus struct {
	Found           bool
	Names           []string
	EndpointsCount  int
	PodMonitorAdded bool
}

// VerifyServiceMonitors checks if ServiceMonitors were created for Tempo
func VerifyServiceMonitors(fw FrameworkOperations) (*ServiceMonitorStatus, error) {
	namespace := fw.Namespace()
	ctx := fw.Context()

	status := &ServiceMonitorStatus{}

	// List ServiceMonitors in the namespace
	list, err := fw.DynamicClient().Resource(gvr.ServiceMonitor).Namespace(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/managed-by=tempo-operator",
	})
	if err != nil {
		if apierrors.IsNotFound(err) {
			fmt.Println("⚠️  ServiceMonitor CRD not found - Prometheus Operator may not be installed")
			return status, nil
		}
		return nil, fmt.Errorf("failed to list ServiceMonitors: %w", err)
	}

	status.Found = len(list.Items) > 0

	for _, item := range list.Items {
		status.Names = append(status.Names, item.GetName())

		// Count endpoints
		endpoints, found, _ := unstructured.NestedSlice(item.Object, "spec", "endpoints")
		if found {
			status.EndpointsCount += len(endpoints)
		}
	}

	if status.Found {
		fmt.Printf("✅ Found %d ServiceMonitor(s) for Tempo: %v\n", len(status.Names), status.Names)
		fmt.Printf("   Total scrape endpoints: %d\n", status.EndpointsCount)
	} else {
		fmt.Println("⚠️  No ServiceMonitors found for Tempo - metrics may not be scraped")
	}

	return status, nil
}

// EnsurePodMonitor creates a PodMonitor for Tempo pods as a fallback
func EnsurePodMonitor(fw FrameworkOperations, variant string) error {
	namespace := fw.Namespace()
	ctx := fw.Context()

	var podMonitorName string
	var matchLabels map[string]interface{}

	if variant == "stack" {
		podMonitorName = "tempo-stack-pods"
		matchLabels = map[string]interface{}{
			"app.kubernetes.io/instance":   "tempostack",
			"app.kubernetes.io/managed-by": "tempo-operator",
		}
	} else {
		podMonitorName = "tempo-monolithic-pods"
		matchLabels = map[string]interface{}{
			"app.kubernetes.io/instance":   "tempo",
			"app.kubernetes.io/managed-by": "tempo-operator",
		}
	}

	// Check if PodMonitor already exists
	_, err := fw.DynamicClient().Resource(gvr.PodMonitor).Namespace(namespace).Get(ctx, podMonitorName, metav1.GetOptions{})
	if err == nil {
		fmt.Printf("✅ PodMonitor %s already exists\n", podMonitorName)
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to check PodMonitor: %w", err)
	}

	// Create PodMonitor
	podMonitor := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "monitoring.coreos.com/v1",
			"kind":       "PodMonitor",
			"metadata": map[string]interface{}{
				"name":      podMonitorName,
				"namespace": namespace,
				"labels": map[string]interface{}{
					"app.kubernetes.io/name":       "tempo",
					"app.kubernetes.io/managed-by": "perf-tests",
				},
			},
			"spec": map[string]interface{}{
				"selector": map[string]interface{}{
					"matchLabels": matchLabels,
				},
				"namespaceSelector": map[string]interface{}{
					"matchNames": []interface{}{namespace},
				},
				"podMetricsEndpoints": []interface{}{
					map[string]interface{}{
						"port":     "http",
						"path":     "/metrics",
						"interval": "30s",
						"relabelings": []interface{}{
							// Add namespace label to all metrics
							map[string]interface{}{
								"sourceLabels": []interface{}{"__meta_kubernetes_namespace"},
								"targetLabel":  "namespace",
							},
							// Add pod label
							map[string]interface{}{
								"sourceLabels": []interface{}{"__meta_kubernetes_pod_name"},
								"targetLabel":  "pod",
							},
							// Add container label
							map[string]interface{}{
								"sourceLabels": []interface{}{"__meta_kubernetes_pod_container_name"},
								"targetLabel":  "container",
							},
						},
					},
					// Also try internal-http port (used by some Tempo components)
					map[string]interface{}{
						"port":     "internal-http",
						"path":     "/metrics",
						"interval": "30s",
						"relabelings": []interface{}{
							map[string]interface{}{
								"sourceLabels": []interface{}{"__meta_kubernetes_namespace"},
								"targetLabel":  "namespace",
							},
							map[string]interface{}{
								"sourceLabels": []interface{}{"__meta_kubernetes_pod_name"},
								"targetLabel":  "pod",
							},
							map[string]interface{}{
								"sourceLabels": []interface{}{"__meta_kubernetes_pod_container_name"},
								"targetLabel":  "container",
							},
						},
					},
				},
			},
		},
	}

	// Add managed labels
	labels := podMonitor.GetLabels()
	for k, v := range fw.GetManagedLabels() {
		labels[k] = v
	}
	podMonitor.SetLabels(labels)

	_, err = fw.DynamicClient().Resource(gvr.PodMonitor).Namespace(namespace).Create(ctx, podMonitor, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create PodMonitor: %w", err)
	}

	// Track for cleanup
	fw.TrackCR(gvr.PodMonitor, namespace, podMonitorName)

	fmt.Printf("✅ Created PodMonitor %s as fallback for Tempo metrics\n", podMonitorName)

	// Give Prometheus time to discover the new PodMonitor
	time.Sleep(5 * time.Second)

	return nil
}

// SetupTempoMonitoring verifies ServiceMonitors and creates PodMonitor fallback if needed
func SetupTempoMonitoring(fw FrameworkOperations, variant string) error {
	fmt.Println("\n📊 Setting up Tempo metrics monitoring...")

	// Verify ServiceMonitors
	status, err := VerifyServiceMonitors(fw)
	if err != nil {
		fmt.Printf("⚠️  Failed to verify ServiceMonitors: %v\n", err)
	}

	// If no ServiceMonitors found, create PodMonitor as fallback
	if !status.Found || status.EndpointsCount == 0 {
		fmt.Println("📦 Creating PodMonitor as fallback for Tempo metrics...")
		if err := EnsurePodMonitor(fw, variant); err != nil {
			return fmt.Errorf("failed to create PodMonitor fallback: %w", err)
		}
		status.PodMonitorAdded = true
	}

	return nil
}
