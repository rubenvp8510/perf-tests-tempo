package framework

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redhat/perf-tests-tempo/test/framework/gvr"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// Cleanup removes all resources created by the framework
func (f *Framework) Cleanup() error {
	f.logger.Info("starting cleanup", "namespace", f.namespace)

	// 1. Delete CRs first (let operators clean up their managed resources)
	if err := f.cleanupCRs(); err != nil {
		return fmt.Errorf("failed to cleanup CRs: %w", err)
	}

	// 2. Wait for CRs to be fully deleted before proceeding
	if err := f.waitForCRsDeletion(); err != nil {
		f.logger.Warn("some CRs may not have been fully deleted", "error", err)
		// Continue with cleanup - the namespace deletion may still work
	}

	// 3. Delete cluster-scoped resources (not deleted with namespace)
	if err := f.cleanupClusterScopedResources(); err != nil {
		return fmt.Errorf("failed to cleanup cluster-scoped resources: %w", err)
	}

	// 4. Delete namespace (cascades to all namespaced resources)
	if err := f.DeleteNamespace(); err != nil {
		return fmt.Errorf("failed to delete namespace: %w", err)
	}

	// 5. Clean up orphaned PVs
	if err := f.cleanupOrphanedPVs(); err != nil {
		f.logger.Warn("failed to cleanup orphaned PVs", "error", err)
		// Non-critical, continue
	}

	f.logger.Info("cleanup completed", "namespace", f.namespace)
	return nil
}

// cleanupCRs deletes all tracked custom resources in parallel
func (f *Framework) cleanupCRs() error {
	trackedCRs := f.GetTrackedCRs()

	// If no tracked CRs, fall back to label-based cleanup
	if len(trackedCRs) == 0 {
		f.logger.Info("no tracked CRs, using label-based cleanup")
		return f.cleanupCRsByLabel()
	}

	f.logger.Info("deleting tracked CRs", "count", len(trackedCRs))

	var wg sync.WaitGroup
	errCh := make(chan error, len(trackedCRs))

	for _, cr := range trackedCRs {
		wg.Add(1)
		go func(res TrackedResource) {
			defer wg.Done()
			f.logger.Debug("deleting CR", "resource", res.GVR.Resource, "name", res.Name)
			err := f.dynamicClient.Resource(res.GVR).Namespace(res.Namespace).Delete(f.ctx, res.Name, metav1.DeleteOptions{})
			if err != nil && !apierrors.IsNotFound(err) {
				errCh <- fmt.Errorf("failed to delete %s/%s: %w", res.GVR.Resource, res.Name, err)
			}
		}(cr)
	}

	wg.Wait()
	close(errCh)

	// Collect errors
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// cleanupCRsByLabel finds and deletes CRs using the managed-by label
func (f *Framework) cleanupCRsByLabel() error {
	labelSelector := fmt.Sprintf("%s=%s,%s=%s", LabelManagedBy, LabelManagedByValue, LabelInstance, f.namespace)

	gvrs := gvr.AllManagedCRs()

	var wg sync.WaitGroup
	errCh := make(chan error, len(gvrs))

	for _, gvr := range gvrs {
		wg.Add(1)
		go func(gvr schema.GroupVersionResource) {
			defer wg.Done()

			list, err := f.dynamicClient.Resource(gvr).Namespace(f.namespace).List(f.ctx, metav1.ListOptions{
				LabelSelector: labelSelector,
			})
			if err != nil {
				if !apierrors.IsNotFound(err) {
					errCh <- fmt.Errorf("failed to list %s: %w", gvr.Resource, err)
				}
				return
			}

			for _, item := range list.Items {
				// Check context before each delete
				if f.ctx.Err() != nil {
					errCh <- fmt.Errorf("context cancelled during %s cleanup: %w", gvr.Resource, f.ctx.Err())
					return
				}
				f.logger.Debug("deleting CR by label", "resource", gvr.Resource, "name", item.GetName())
				err := f.dynamicClient.Resource(gvr).Namespace(f.namespace).Delete(f.ctx, item.GetName(), metav1.DeleteOptions{})
				if err != nil && !apierrors.IsNotFound(err) {
					errCh <- fmt.Errorf("failed to delete %s/%s: %w", gvr.Resource, item.GetName(), err)
				}
			}
		}(gvr)
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// waitForCRsDeletion waits for all tracked CRs to be fully deleted
func (f *Framework) waitForCRsDeletion() error {
	trackedCRs := f.GetTrackedCRs()
	if len(trackedCRs) == 0 {
		return nil
	}

	crDeletionTimeout := f.config.CRDeletionTimeout
	crDeletionPollInterval := f.config.CRDeletionPollInterval

	f.logger.Info("waiting for CRs to be deleted", "count", len(trackedCRs), "timeout", crDeletionTimeout)

	// Track pending CRs to avoid re-checking deleted ones
	pending := make([]TrackedResource, len(trackedCRs))
	copy(pending, trackedCRs)

	ticker := time.NewTicker(crDeletionPollInterval)
	defer ticker.Stop()

	timeout := time.After(crDeletionTimeout)

	for {
		select {
		case <-f.ctx.Done():
			return fmt.Errorf("context cancelled while waiting for CR deletion: %w", f.ctx.Err())
		case <-timeout:
			// Attempt to remove finalizers from stuck resources
			f.logger.Warn("timeout waiting for CR deletion, attempting to remove finalizers", "remaining", len(pending))
			if err := f.removeFinalizersFromCRs(pending); err != nil {
				f.logger.Warn("failed to remove finalizers from some CRs", "error", err)
			}
			remaining := make([]string, len(pending))
			for i, cr := range pending {
				remaining[i] = fmt.Sprintf("%s/%s", cr.GVR.Resource, cr.Name)
			}
			return fmt.Errorf("timeout waiting for CRs to be deleted after %v, remaining: %v", crDeletionTimeout, remaining)
		case <-ticker.C:
			var stillPending []TrackedResource

			for _, cr := range pending {
				_, err := f.dynamicClient.Resource(cr.GVR).Namespace(cr.Namespace).Get(f.ctx, cr.Name, metav1.GetOptions{})
				if err == nil {
					// Resource still exists
					stillPending = append(stillPending, cr)
					continue
				}
				if !apierrors.IsNotFound(err) {
					// Unexpected error - keep tracking this resource
					f.logger.Warn("error checking CR status", "resource", cr.GVR.Resource, "name", cr.Name, "error", err)
					stillPending = append(stillPending, cr)
					continue
				}
				// Resource is gone (NotFound) - don't add to stillPending
				f.logger.Debug("CR deleted", "resource", cr.GVR.Resource, "name", cr.Name)
			}

			if len(stillPending) == 0 {
				f.logger.Info("all CRs deleted successfully")
				return nil
			}

			pending = stillPending
			f.logger.Debug("waiting for CRs to be deleted", "remaining", len(pending))
		}
	}
}

// removeFinalizersFromCRs removes finalizers from stuck CRs to allow deletion
func (f *Framework) removeFinalizersFromCRs(crs []TrackedResource) error {
	var errs []error

	for _, cr := range crs {
		// Check if resource still exists
		obj, err := f.dynamicClient.Resource(cr.GVR).Namespace(cr.Namespace).Get(f.ctx, cr.Name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue // Already deleted
			}
			errs = append(errs, fmt.Errorf("failed to get %s/%s: %w", cr.GVR.Resource, cr.Name, err))
			continue
		}

		// Check if it has finalizers
		finalizers := obj.GetFinalizers()
		if len(finalizers) == 0 {
			continue
		}

		f.logger.Info("removing finalizers from stuck CR", "resource", cr.GVR.Resource, "name", cr.Name, "finalizers", finalizers)

		// Patch to remove all finalizers
		patch := []byte(`{"metadata":{"finalizers":null}}`)
		_, err = f.dynamicClient.Resource(cr.GVR).Namespace(cr.Namespace).Patch(
			f.ctx,
			cr.Name,
			types.MergePatchType,
			patch,
			metav1.PatchOptions{},
		)
		if err != nil && !apierrors.IsNotFound(err) {
			errs = append(errs, fmt.Errorf("failed to remove finalizers from %s/%s: %w", cr.GVR.Resource, cr.Name, err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// cleanupClusterScopedResources deletes cluster-scoped resources created by the framework
func (f *Framework) cleanupClusterScopedResources() error {
	trackedResources := f.GetTrackedClusterResources()

	// If no tracked resources, fall back to label-based cleanup
	if len(trackedResources) == 0 {
		f.logger.Info("no tracked cluster resources, using label-based cleanup")
		return f.cleanupClusterResourcesByLabel()
	}

	f.logger.Info("deleting tracked cluster resources", "count", len(trackedResources))

	var wg sync.WaitGroup
	errCh := make(chan error, len(trackedResources))

	for _, res := range trackedResources {
		wg.Add(1)
		go func(res TrackedResource) {
			defer wg.Done()

			f.logger.Debug("deleting cluster resource", "kind", res.GVR.Resource, "name", res.Name)

			var err error
			switch res.GVR.Resource {
			case "clusterroles":
				err = f.client.RbacV1().ClusterRoles().Delete(f.ctx, res.Name, metav1.DeleteOptions{})
			case "clusterrolebindings":
				err = f.client.RbacV1().ClusterRoleBindings().Delete(f.ctx, res.Name, metav1.DeleteOptions{})
			default:
				err = f.dynamicClient.Resource(res.GVR).Delete(f.ctx, res.Name, metav1.DeleteOptions{})
			}

			if err != nil && !apierrors.IsNotFound(err) {
				f.logger.Warn("failed to delete cluster resource", "kind", res.GVR.Resource, "name", res.Name, "error", err)
				errCh <- fmt.Errorf("failed to delete %s/%s: %w", res.GVR.Resource, res.Name, err)
			}
		}(res)
	}

	wg.Wait()
	close(errCh)

	// Collect errors
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// cleanupClusterResourcesByLabel finds and deletes cluster resources using the managed-by label
func (f *Framework) cleanupClusterResourcesByLabel() error {
	labelSelector := fmt.Sprintf("%s=%s,%s=%s", LabelManagedBy, LabelManagedByValue, LabelInstance, f.namespace)

	var errs []error

	// Delete ClusterRoles
	clusterRoles, err := f.client.RbacV1().ClusterRoles().List(f.ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil && !apierrors.IsNotFound(err) {
		errs = append(errs, fmt.Errorf("failed to list ClusterRoles: %w", err))
	} else if clusterRoles != nil {
		for _, cr := range clusterRoles.Items {
			if f.ctx.Err() != nil {
				return fmt.Errorf("context cancelled during ClusterRole cleanup: %w", f.ctx.Err())
			}
			f.logger.Debug("deleting ClusterRole by label", "name", cr.Name)
			if err := f.client.RbacV1().ClusterRoles().Delete(f.ctx, cr.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
				errs = append(errs, fmt.Errorf("failed to delete ClusterRole %s: %w", cr.Name, err))
			}
		}
	}

	// Delete ClusterRoleBindings
	clusterRoleBindings, err := f.client.RbacV1().ClusterRoleBindings().List(f.ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil && !apierrors.IsNotFound(err) {
		errs = append(errs, fmt.Errorf("failed to list ClusterRoleBindings: %w", err))
	} else if clusterRoleBindings != nil {
		for _, crb := range clusterRoleBindings.Items {
			if f.ctx.Err() != nil {
				return fmt.Errorf("context cancelled during ClusterRoleBinding cleanup: %w", f.ctx.Err())
			}
			f.logger.Debug("deleting ClusterRoleBinding by label", "name", crb.Name)
			if err := f.client.RbacV1().ClusterRoleBindings().Delete(f.ctx, crb.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
				errs = append(errs, fmt.Errorf("failed to delete ClusterRoleBinding %s: %w", crb.Name, err))
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// cleanupOrphanedPVs finds and deletes orphaned PVs related to this namespace
func (f *Framework) cleanupOrphanedPVs() error {
	var deletedCount int
	var errs []error
	deletedPVs := make(map[string]bool)

	// First, efficiently find PVs with our labels
	labelSelector := fmt.Sprintf("%s=%s", LabelInstance, f.namespace)
	labeledPVs, err := f.client.CoreV1().PersistentVolumes().List(f.ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return fmt.Errorf("failed to list labeled PVs: %w", err)
	}

	for _, pv := range labeledPVs.Items {
		if deleted, err := f.deleteOrphanedPV(&pv); err != nil {
			errs = append(errs, err)
		} else if deleted {
			deletedCount++
			deletedPVs[pv.Name] = true
		}
	}

	// Then check for PVs bound to PVCs in our namespace (requires ClaimRef check)
	// Only do this scan if namespace deletion might leave orphaned PVs
	allPVs, err := f.client.CoreV1().PersistentVolumes().List(f.ctx, metav1.ListOptions{})
	if err != nil {
		f.logger.Warn("failed to list all PVs for ClaimRef check", "error", err)
	} else {
		for _, pv := range allPVs.Items {
			// Skip already processed PVs
			if deletedPVs[pv.Name] {
				continue
			}
			// Check if PV was bound to a PVC in this namespace
			if pv.Spec.ClaimRef != nil && pv.Spec.ClaimRef.Namespace == f.namespace {
				if deleted, err := f.deleteOrphanedPV(&pv); err != nil {
					errs = append(errs, err)
				} else if deleted {
					deletedCount++
				}
			}
		}
	}

	if deletedCount > 0 {
		f.logger.Info("deleted orphaned PVs", "count", deletedCount)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// deleteOrphanedPV deletes a PV if it's in Released or Available phase
// Returns true if the PV was deleted, false otherwise
func (f *Framework) deleteOrphanedPV(pv *corev1.PersistentVolume) (bool, error) {
	// Only delete Released or Available PVs
	if pv.Status.Phase != corev1.VolumeReleased && pv.Status.Phase != corev1.VolumeAvailable {
		f.logger.Debug("skipping PV not in Released/Available phase", "pv", pv.Name, "phase", pv.Status.Phase)
		return false, nil
	}

	f.logger.Debug("deleting orphaned PV", "pv", pv.Name)
	err := f.client.CoreV1().PersistentVolumes().Delete(f.ctx, pv.Name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		f.logger.Warn("failed to delete orphaned PV", "pv", pv.Name, "error", err)
		return false, fmt.Errorf("failed to delete PV %s: %w", pv.Name, err)
	}
	return true, nil
}
