package k6

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

const (
	// UserWorkloadMonitoringNamespace is the namespace for user workload monitoring
	UserWorkloadMonitoringNamespace = "openshift-user-workload-monitoring"

	// UserWorkloadConfigMapName is the name of the ConfigMap for user workload monitoring config
	UserWorkloadConfigMapName = "user-workload-monitoring-config"

	// OpenShiftMonitoringNamespace is the namespace for cluster monitoring
	OpenShiftMonitoringNamespace = "openshift-monitoring"
)

// GetPrometheusRemoteWriteURL returns the Prometheus remote write URL for user workload monitoring
// In OpenShift, the prometheus-user-workload service uses port 9091
func GetPrometheusRemoteWriteURL() string {
	return fmt.Sprintf("http://prometheus-user-workload.%s.svc:9091/api/v1/write", UserWorkloadMonitoringNamespace)
}

// EnablePrometheusRemoteWriteReceiver enables the remote write receiver in user workload monitoring
// This allows k6 to push metrics directly to Prometheus
func EnablePrometheusRemoteWriteReceiver(ctx context.Context, client kubernetes.Interface) error {
	configMapName := UserWorkloadConfigMapName
	namespace := OpenShiftMonitoringNamespace

	// Get existing ConfigMap or create new one
	cm, err := client.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Create new ConfigMap with remote write receiver enabled
			cm = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      configMapName,
					Namespace: namespace,
				},
				Data: map[string]string{
					"config.yaml": `prometheus:
  remoteWrite:
    - url: "http://localhost:9090/api/v1/write"
  enableRemoteWriteReceiver: true
`,
				},
			}
			_, err = client.CoreV1().ConfigMaps(namespace).Create(ctx, cm, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create user workload monitoring config: %w", err)
			}
			fmt.Println("✅ Created user-workload-monitoring-config with remote write receiver enabled")
			return nil
		}
		return fmt.Errorf("failed to get user workload monitoring config: %w", err)
	}

	// Check if config already has remote write receiver enabled
	configYaml := cm.Data["config.yaml"]
	if configYaml == "" {
		configYaml = "{}"
	}

	// Parse existing config
	var config map[string]interface{}
	if err := yaml.Unmarshal([]byte(configYaml), &config); err != nil {
		return fmt.Errorf("failed to parse existing config: %w", err)
	}

	// Check if prometheus config exists
	prometheusConfig, ok := config["prometheus"].(map[string]interface{})
	if !ok {
		prometheusConfig = make(map[string]interface{})
		config["prometheus"] = prometheusConfig
	}

	// Check if remote write receiver is already enabled
	if enabled, ok := prometheusConfig["enableRemoteWriteReceiver"].(bool); ok && enabled {
		fmt.Println("✅ Prometheus remote write receiver is already enabled")
		return nil
	}

	// Enable remote write receiver
	prometheusConfig["enableRemoteWriteReceiver"] = true

	// Marshal back to YAML
	updatedConfig, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal updated config: %w", err)
	}

	// Update ConfigMap
	cm.Data["config.yaml"] = string(updatedConfig)
	_, err = client.CoreV1().ConfigMaps(namespace).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update user workload monitoring config: %w", err)
	}

	fmt.Println("✅ Enabled Prometheus remote write receiver in user workload monitoring")
	fmt.Println("   Note: Prometheus may take a few minutes to reload the configuration")

	return nil
}

// SetupK6PrometheusMetrics sets up k6 to export metrics to Prometheus
// Returns the remote write URL to use, or empty string if setup fails
func SetupK6PrometheusMetrics(ctx context.Context, client kubernetes.Interface) (string, error) {
	// Enable remote write receiver
	if err := EnablePrometheusRemoteWriteReceiver(ctx, client); err != nil {
		fmt.Printf("⚠️  Failed to enable Prometheus remote write receiver: %v\n", err)
		fmt.Println("   k6 metrics will not be exported to Prometheus")
		return "", nil
	}

	url := GetPrometheusRemoteWriteURL()
	fmt.Printf("📊 k6 metrics will be exported to: %s\n", url)

	return url, nil
}
