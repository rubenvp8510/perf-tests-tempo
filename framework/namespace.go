package framework

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EnsureNamespace creates the namespace if it doesn't exist
func (f *Framework) EnsureNamespace() error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: f.namespace,
		},
	}

	_, err := f.client.CoreV1().Namespaces().Create(f.ctx, ns, metav1.CreateOptions{})
	if err != nil {
		// Check if namespace already exists
		_, getErr := f.client.CoreV1().Namespaces().Get(f.ctx, f.namespace, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("failed to create namespace: %w", err)
		}
		// Namespace exists, that's fine
	}

	// Wait a moment for namespace to be ready
	time.Sleep(2 * time.Second)
	return nil
}

// DeleteNamespace deletes the namespace
func (f *Framework) DeleteNamespace() error {
	err := f.client.CoreV1().Namespaces().Delete(f.ctx, f.namespace, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete namespace: %w", err)
	}

	// Wait for namespace deletion with timeout
	timeout := 120 * time.Second
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		_, err := f.client.CoreV1().Namespaces().Get(f.ctx, f.namespace, metav1.GetOptions{})
		if err != nil {
			// Namespace is gone
			return nil
		}
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("namespace deletion timed out after %v", timeout)
}
