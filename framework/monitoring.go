package framework

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const (
	monitoringNamespace        = "openshift-monitoring"
	clusterMonitoringConfigMap = "cluster-monitoring-config"
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
			return f.createMonitoringConfigMap(ctx)
		}
		return fmt.Errorf("failed to get cluster-monitoring-config: %w", err)
	}

	// Update existing ConfigMap if needed
	return f.updateMonitoringConfigMap(ctx, existingCM)
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
