package framework

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

// Cleanup removes all resources including PVs/PVCs
func (f *Framework) Cleanup() error {
	// Delete trace generator jobs
	if err := f.cleanupTraceGenerators(); err != nil {
		return fmt.Errorf("failed to cleanup trace generators: %w", err)
	}

	// Delete query generator deployment
	if err := f.cleanupQueryGenerator(); err != nil {
		return fmt.Errorf("failed to cleanup query generator: %w", err)
	}

	// Delete Tempo resources
	if err := f.cleanupTempo(); err != nil {
		return fmt.Errorf("failed to cleanup Tempo: %w", err)
	}

	// Delete OpenTelemetry Collector
	if err := f.cleanupOTelCollector(); err != nil {
		return fmt.Errorf("failed to cleanup OTel Collector: %w", err)
	}

	// Delete MinIO
	if err := f.cleanupMinIO(); err != nil {
		return fmt.Errorf("failed to cleanup MinIO: %w", err)
	}

	// Wait for all pods to terminate
	if err := f.waitForPodsTerminated(); err != nil {
		return fmt.Errorf("failed to wait for pods to terminate: %w", err)
	}

	// Delete PVCs and PVs
	if err := f.cleanupPVCsAndPVs(); err != nil {
		return fmt.Errorf("failed to cleanup PVCs/PVs: %w", err)
	}

	// Delete namespace (this will clean up any remaining resources)
	if err := f.DeleteNamespace(); err != nil {
		return fmt.Errorf("failed to delete namespace: %w", err)
	}

	return nil
}

// cleanupTraceGenerators deletes trace generator jobs
func (f *Framework) cleanupTraceGenerators() error {
	selector, err := labels.Parse("app=trace-generator")
	if err != nil {
		return err
	}

	jobs, err := f.client.BatchV1().Jobs(f.namespace).List(f.ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return err
	}

	for _, job := range jobs.Items {
		err := f.client.BatchV1().Jobs(f.namespace).Delete(f.ctx, job.Name, metav1.DeleteOptions{})
		if err != nil {
			// Continue on error
		}
	}

	return nil
}

// cleanupQueryGenerator deletes query generator deployment
func (f *Framework) cleanupQueryGenerator() error {
	err := f.client.AppsV1().Deployments(f.namespace).Delete(f.ctx, "query-load-generator", metav1.DeleteOptions{})
	if err != nil {
		// Ignore if not found
	}
	return nil
}

// cleanupTempo deletes Tempo resources
func (f *Framework) cleanupTempo() error {
	dynamicClient, err := dynamic.NewForConfig(f.config)
	if err != nil {
		return err
	}

	// Delete TempoMonolithic
	gvr := schema.GroupVersionResource{
		Group:    "tempo.grafana.com",
		Version:  "v1alpha1",
		Resource: "tempomonolithics",
	}

	err = dynamicClient.Resource(gvr).Namespace(f.namespace).Delete(f.ctx, "simplest", metav1.DeleteOptions{})
	if err != nil {
		// Ignore if not found
	}

	// Delete TempoStack
	gvrStack := schema.GroupVersionResource{
		Group:    "tempo.grafana.com",
		Version:  "v1alpha1",
		Resource: "tempostacks",
	}

	err = dynamicClient.Resource(gvrStack).Namespace(f.namespace).Delete(f.ctx, "tempostack", metav1.DeleteOptions{})
	if err != nil {
		// Ignore if not found
	}

	return nil
}

// cleanupOTelCollector deletes OpenTelemetry Collector resources
func (f *Framework) cleanupOTelCollector() error {
	dynamicClient, err := dynamic.NewForConfig(f.config)
	if err != nil {
		return err
	}

	// Delete OpenTelemetryCollector CR
	gvr := schema.GroupVersionResource{
		Group:    "opentelemetry.io",
		Version:  "v1beta1",
		Resource: "opentelemetrycollectors",
	}

	err = dynamicClient.Resource(gvr).Namespace(f.namespace).Delete(f.ctx, "otel-collector", metav1.DeleteOptions{})
	if err != nil {
		// Ignore if not found
	}

	// Delete deployments
	for _, name := range []string{"otel-collector-collector", "otel-collector"} {
		err = f.client.AppsV1().Deployments(f.namespace).Delete(f.ctx, name, metav1.DeleteOptions{})
		if err != nil {
			// Ignore if not found
		}
	}

	// Delete ServiceAccount
	err = f.client.CoreV1().ServiceAccounts(f.namespace).Delete(f.ctx, "otel-collector-sa", metav1.DeleteOptions{})
	if err != nil {
		// Ignore if not found
	}

	// Delete Role
	err = f.client.RbacV1().Roles(f.namespace).Delete(f.ctx, "otel-collector-role", metav1.DeleteOptions{})
	if err != nil {
		// Ignore if not found
	}

	// Delete RoleBinding
	err = f.client.RbacV1().RoleBindings(f.namespace).Delete(f.ctx, "otel-collector-rolebinding", metav1.DeleteOptions{})
	if err != nil {
		// Ignore if not found
	}

	// Delete ClusterRole
	err = f.client.RbacV1().ClusterRoles().Delete(f.ctx, "allow-write-traces-tenant-1", metav1.DeleteOptions{})
	if err != nil {
		// Ignore if not found
	}

	// Delete ClusterRoleBinding
	err = f.client.RbacV1().ClusterRoleBindings().Delete(f.ctx, "allow-write-traces-tenant-1", metav1.DeleteOptions{})
	if err != nil {
		// Ignore if not found
	}

	return nil
}

// cleanupMinIO deletes MinIO resources
func (f *Framework) cleanupMinIO() error {
	// Delete deployment
	err := f.client.AppsV1().Deployments(f.namespace).Delete(f.ctx, "minio", metav1.DeleteOptions{})
	if err != nil {
		// Ignore if not found
	}

	// Delete service
	err = f.client.CoreV1().Services(f.namespace).Delete(f.ctx, "minio", metav1.DeleteOptions{})
	if err != nil {
		// Ignore if not found
	}

	// Delete secret
	err = f.client.CoreV1().Secrets(f.namespace).Delete(f.ctx, "minio", metav1.DeleteOptions{})
	if err != nil {
		// Ignore if not found
	}

	return nil
}

// waitForPodsTerminated waits for all pods to be fully terminated
func (f *Framework) waitForPodsTerminated() error {
	selectors := []string{
		"app=trace-generator",
		"app.kubernetes.io/name=tempo",
		"app.kubernetes.io/name=opentelemetry-collector",
		"app.kubernetes.io/name=minio",
	}

	timeout := 120 * time.Second
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		allTerminated := true

		for _, selectorStr := range selectors {
			selector, err := labels.Parse(selectorStr)
			if err != nil {
				continue
			}

			pods, err := f.client.CoreV1().Pods(f.namespace).List(f.ctx, metav1.ListOptions{
				LabelSelector: selector.String(),
			})
			if err != nil {
				continue
			}

			if len(pods.Items) > 0 {
				allTerminated = false
				break
			}
		}

		if allTerminated {
			return nil
		}

		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("pods not terminated after %v", timeout)
}

// cleanupPVCsAndPVs deletes PVCs and associated PVs
func (f *Framework) cleanupPVCsAndPVs() error {
	// Delete MinIO PVC
	pvc, err := f.client.CoreV1().PersistentVolumeClaims(f.namespace).Get(f.ctx, "minio", metav1.GetOptions{})
	if err != nil {
		// PVC doesn't exist, that's fine
		return nil
	}

	// Get PV name before deleting PVC
	pvName := pvc.Spec.VolumeName

	// Delete PVC
	err = f.client.CoreV1().PersistentVolumeClaims(f.namespace).Delete(f.ctx, "minio", metav1.DeleteOptions{})
	if err != nil {
		// Ignore if not found
	}

	// Wait for PVC to be deleted with timeout
	timeout := 120 * time.Second
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		_, err := f.client.CoreV1().PersistentVolumeClaims(f.namespace).Get(f.ctx, "minio", metav1.GetOptions{})
		if err != nil {
			// PVC is gone
			break
		}
		time.Sleep(5 * time.Second)
	}

	// If PVC still exists, try to remove finalizers
	_, err = f.client.CoreV1().PersistentVolumeClaims(f.namespace).Get(f.ctx, "minio", metav1.GetOptions{})
	if err == nil {
		// PVC still exists, remove finalizers
		patch := []byte(`{"metadata":{"finalizers":[]}}`)
		_, err = f.client.CoreV1().PersistentVolumeClaims(f.namespace).Patch(
			f.ctx,
			"minio",
			types.MergePatchType,
			patch,
			metav1.PatchOptions{},
		)
		if err == nil {
			// Try to delete again
			f.client.CoreV1().PersistentVolumeClaims(f.namespace).Delete(f.ctx, "minio", metav1.DeleteOptions{
				GracePeriodSeconds: func() *int64 { zero := int64(0); return &zero }(),
			})
		}

		// Wait again
		deadline = time.Now().Add(60 * time.Second)
		for time.Now().Before(deadline) {
			_, err := f.client.CoreV1().PersistentVolumeClaims(f.namespace).Get(f.ctx, "minio", metav1.GetOptions{})
			if err != nil {
				break
			}
			time.Sleep(5 * time.Second)
		}
	}

	// Clean up associated PV
	if pvName != "" {
		if err := f.cleanupPV(pvName); err != nil {
			// Log but don't fail
		}
	}

	// Check for orphaned PVs
	if err := f.cleanupOrphanedPVs(); err != nil {
		// Log but don't fail
	}

	return nil
}

// cleanupPV deletes a PV if it's in Released or Available state
func (f *Framework) cleanupPV(pvName string) error {
	pv, err := f.client.CoreV1().PersistentVolumes().Get(f.ctx, pvName, metav1.GetOptions{})
	if err != nil {
		return nil // PV doesn't exist
	}

	if pv.Status.Phase == corev1.VolumeReleased || pv.Status.Phase == corev1.VolumeAvailable {
		err = f.client.CoreV1().PersistentVolumes().Delete(f.ctx, pvName, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

// cleanupOrphanedPVs finds and deletes orphaned PVs related to this namespace
func (f *Framework) cleanupOrphanedPVs() error {
	pvs, err := f.client.CoreV1().PersistentVolumes().List(f.ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, pv := range pvs.Items {
		// Check if PV is related to this namespace or minio
		related := false
		if pv.Spec.ClaimRef != nil && pv.Spec.ClaimRef.Namespace == f.namespace {
			related = true
		}
		if !related {
			// Check labels or annotations
			for key, value := range pv.Labels {
				if key == "app.kubernetes.io/name" && value == "minio" {
					related = true
					break
				}
			}
		}

		if related && (pv.Status.Phase == corev1.VolumeReleased || pv.Status.Phase == corev1.VolumeAvailable) {
			err = f.client.CoreV1().PersistentVolumes().Delete(f.ctx, pv.Name, metav1.DeleteOptions{})
			if err != nil {
				// Continue on error
			}
		}
	}

	return nil
}
