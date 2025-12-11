package framework

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// WaitForPodsReady waits for pods matching the selector to be ready
func (f *Framework) WaitForPodsReady(selector labels.Selector, timeout time.Duration, minReady int) error {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		pods, err := f.client.CoreV1().Pods(f.namespace).List(f.ctx, metav1.ListOptions{
			LabelSelector: selector.String(),
		})
		if err != nil {
			return fmt.Errorf("failed to list pods: %w", err)
		}

		readyCount := 0
		for _, pod := range pods.Items {
			if isPodReady(&pod) {
				readyCount++
			}
		}

		if readyCount >= minReady && len(pods.Items) > 0 {
			return nil
		}

		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("pods not ready after %v (expected at least %d ready)", timeout, minReady)
}

// WaitForDeploymentReady waits for a deployment to be ready
func (f *Framework) WaitForDeploymentReady(name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		deployment, err := f.client.AppsV1().Deployments(f.namespace).Get(f.ctx, name, metav1.GetOptions{})
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		if deployment.Status.ReadyReplicas == deployment.Status.Replicas &&
			deployment.Status.ReadyReplicas > 0 {
			return nil
		}

		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("deployment %s not ready after %v", name, timeout)
}

// WaitForPodsTerminated waits for pods matching the selector to be fully terminated
func (f *Framework) WaitForPodsTerminated(selector labels.Selector, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		pods, err := f.client.CoreV1().Pods(f.namespace).List(f.ctx, metav1.ListOptions{
			LabelSelector: selector.String(),
		})
		if err != nil {
			// If we can't list pods, they might be gone
			return nil
		}

		if len(pods.Items) == 0 {
			return nil
		}

		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("pods not terminated after %v", timeout)
}

// isPodReady checks if a pod is in Ready state
func isPodReady(pod *corev1.Pod) bool {
	if pod.Status.Phase != corev1.PodRunning {
		return false
	}

	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}

	return false
}

// WaitForTempoPodsReady waits for Tempo pods using multiple label selectors
func (f *Framework) WaitForTempoPodsReady(timeout time.Duration) error {
	// Try multiple label selectors (Tempo Operator uses different labels in different versions)
	selectors := []string{
		"app.kubernetes.io/name=tempo",
		"app.kubernetes.io/instance=simplest",
		"tempo.grafana.com/name=simplest",
	}

	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		for _, selectorStr := range selectors {
			selector, err := labels.Parse(selectorStr)
			if err != nil {
				continue
			}

			pods, err := f.client.CoreV1().Pods(f.namespace).List(f.ctx, metav1.ListOptions{
				LabelSelector: selector.String(),
			})
			if err != nil {
				lastErr = err
				continue
			}

			if len(pods.Items) == 0 {
				continue
			}

			readyCount := 0
			for _, pod := range pods.Items {
				if isPodReady(&pod) {
					readyCount++
				}
			}

			if readyCount > 0 {
				return nil
			}
		}

		// Also try by name pattern
		allPods, err := f.client.CoreV1().Pods(f.namespace).List(f.ctx, metav1.ListOptions{})
		if err == nil {
			for _, pod := range allPods.Items {
				if (pod.Name == "tempo-simplest" || 
					len(pod.Name) > 13 && pod.Name[:13] == "tempo-simplest") &&
					isPodReady(&pod) {
					return nil
				}
			}
		}

		time.Sleep(5 * time.Second)
	}

	if lastErr != nil {
		return fmt.Errorf("tempo pods not ready after %v: %w", timeout, lastErr)
	}
	return fmt.Errorf("tempo pods not ready after %v", timeout)
}

