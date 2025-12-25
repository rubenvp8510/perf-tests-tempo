package wait

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// Clients provides access to Kubernetes clients needed for wait operations
type Clients interface {
	Client() kubernetes.Interface
	Context() context.Context
	Namespace() string
	Logger() *slog.Logger
}

// ForPodsReady waits for pods matching the selector to be ready
func ForPodsReady(c Clients, selector labels.Selector, timeout time.Duration, minReady int) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		pods, err := c.Client().CoreV1().Pods(c.Namespace()).List(c.Context(), metav1.ListOptions{
			LabelSelector: selector.String(),
		})
		if err != nil {
			return fmt.Errorf("failed to list pods: %w", err)
		}

		readyCount := 0
		for _, pod := range pods.Items {
			if IsPodReady(&pod) {
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

// ForDeploymentReady waits for a deployment to be ready
func ForDeploymentReady(c Clients, name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		deployment, err := c.Client().AppsV1().Deployments(c.Namespace()).Get(c.Context(), name, metav1.GetOptions{})
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

// ForPodsTerminated waits for pods matching the selector to be fully terminated
func ForPodsTerminated(c Clients, selector labels.Selector, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		pods, err := c.Client().CoreV1().Pods(c.Namespace()).List(c.Context(), metav1.ListOptions{
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

// ForTempoPodsReady waits for Tempo pods using multiple label selectors
func ForTempoPodsReady(c Clients, timeout time.Duration) error {
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

			pods, err := c.Client().CoreV1().Pods(c.Namespace()).List(c.Context(), metav1.ListOptions{
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
				if IsPodReady(&pod) {
					readyCount++
				}
			}

			if readyCount > 0 {
				return nil
			}
		}

		// Also try by name pattern
		allPods, err := c.Client().CoreV1().Pods(c.Namespace()).List(c.Context(), metav1.ListOptions{})
		if err == nil {
			for _, pod := range allPods.Items {
				if (pod.Name == "tempo-simplest" ||
					len(pod.Name) > 13 && pod.Name[:13] == "tempo-simplest") &&
					IsPodReady(&pod) {
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

// IsPodReady checks if a pod is in Ready state
func IsPodReady(pod *corev1.Pod) bool {
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
