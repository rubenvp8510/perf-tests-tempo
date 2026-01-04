package framework

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/yaml"
)

const (
	monitoringNamespace            = "openshift-monitoring"
	userWorkloadMonitoringNS       = "openshift-user-workload-monitoring"
	clusterMonitoringConfigMap     = "cluster-monitoring-config"
	userWorkloadMonitoringTimeout  = 2 * time.Minute
	userWorkloadMonitoringInterval = 5 * time.Second
)

// MonitoringConfig represents the OpenShift cluster monitoring configuration
type MonitoringConfig struct {
	EnableUserWorkload bool `json:"enableUserWorkload,omitempty"`
}

// EnableUserWorkloadMonitoring enables user workload monitoring in OpenShift
// by creating or updating the cluster-monitoring-config ConfigMap
func (f *Framework) EnableUserWorkloadMonitoring() error {
	ctx := f.ctx
	client := f.client.CoreV1().ConfigMaps(monitoringNamespace)

	// Check if ConfigMap exists
	existingCM, err := client.Get(ctx, clusterMonitoringConfigMap, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Create new ConfigMap
			if err := f.createMonitoringConfigMap(ctx); err != nil {
				return err
			}
			// Wait for user workload monitoring pods to be ready
			return f.waitForUserWorkloadMonitoring()
		}
		return fmt.Errorf("failed to get cluster-monitoring-config: %w", err)
	}

	// Update existing ConfigMap if needed
	if err := f.updateMonitoringConfigMap(ctx, existingCM); err != nil {
		return err
	}

	// Wait for user workload monitoring pods to be ready
	return f.waitForUserWorkloadMonitoring()
}

// createMonitoringConfigMap creates the cluster-monitoring-config ConfigMap
func (f *Framework) createMonitoringConfigMap(ctx context.Context) error {
	config := MonitoringConfig{
		EnableUserWorkload: true,
	}

	configYAML, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal monitoring config: %w", err)
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterMonitoringConfigMap,
			Namespace: monitoringNamespace,
		},
		Data: map[string]string{
			"config.yaml": string(configYAML),
		},
	}

	_, err = f.client.CoreV1().ConfigMaps(monitoringNamespace).Create(ctx, cm, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create cluster-monitoring-config: %w", err)
	}

	f.logger.Info("Created cluster-monitoring-config with user workload monitoring enabled")
	return nil
}

// updateMonitoringConfigMap updates the existing ConfigMap to enable user workload monitoring
func (f *Framework) updateMonitoringConfigMap(ctx context.Context, cm *corev1.ConfigMap) error {
	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}

	// Parse existing config
	var config MonitoringConfig
	if existingConfig, ok := cm.Data["config.yaml"]; ok && existingConfig != "" {
		if err := yaml.Unmarshal([]byte(existingConfig), &config); err != nil {
			// If we can't parse, we'll create a new config
			f.logger.Warn("Could not parse existing monitoring config, will update with new config")
			config = MonitoringConfig{}
		}
	}

	// Check if already enabled
	if config.EnableUserWorkload {
		f.logger.Info("User workload monitoring is already enabled")
		return nil
	}

	// Enable user workload monitoring
	config.EnableUserWorkload = true

	configYAML, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal monitoring config: %w", err)
	}

	cm.Data["config.yaml"] = string(configYAML)

	_, err = f.client.CoreV1().ConfigMaps(monitoringNamespace).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update cluster-monitoring-config: %w", err)
	}

	f.logger.Info("Updated cluster-monitoring-config to enable user workload monitoring")
	return nil
}

// IsUserWorkloadMonitoringEnabled checks if user workload monitoring is enabled
func (f *Framework) IsUserWorkloadMonitoringEnabled() (bool, error) {
	ctx := f.ctx
	client := f.client.CoreV1().ConfigMaps(monitoringNamespace)

	cm, err := client.Get(ctx, clusterMonitoringConfigMap, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get cluster-monitoring-config: %w", err)
	}

	if cm.Data == nil {
		return false, nil
	}

	existingConfig, ok := cm.Data["config.yaml"]
	if !ok || existingConfig == "" {
		return false, nil
	}

	var config MonitoringConfig
	if err := yaml.Unmarshal([]byte(existingConfig), &config); err != nil {
		return false, fmt.Errorf("failed to parse monitoring config: %w", err)
	}

	return config.EnableUserWorkload, nil
}

// waitForUserWorkloadMonitoring waits for the user workload monitoring Prometheus to be ready
func (f *Framework) waitForUserWorkloadMonitoring() error {
	f.logger.Info("Waiting for user workload monitoring to be ready...")

	ctx, cancel := context.WithTimeout(f.ctx, userWorkloadMonitoringTimeout)
	defer cancel()

	return wait.PollUntilContextCancel(ctx, userWorkloadMonitoringInterval, true, func(ctx context.Context) (bool, error) {
		// Check if the prometheus-user-workload pods are running
		pods, err := f.client.CoreV1().Pods(userWorkloadMonitoringNS).List(ctx, metav1.ListOptions{
			LabelSelector: "app.kubernetes.io/name=prometheus",
		})
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil // Namespace doesn't exist yet
			}
			return false, nil // Keep polling on other errors
		}

		if len(pods.Items) == 0 {
			return false, nil // No pods yet
		}

		// Check if at least one Prometheus pod is ready
		for _, pod := range pods.Items {
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
					f.logger.Info("User workload monitoring is ready")
					return true, nil
				}
			}
		}

		return false, nil
	})
}
