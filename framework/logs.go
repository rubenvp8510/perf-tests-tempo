package framework

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/redhat/perf-tests-tempo/test/framework/gvr"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// LogCollectionConfig configures log collection behavior
type LogCollectionConfig struct {
	// OutputDir is the directory to write logs to
	OutputDir string
	// IncludePrevious includes logs from previous container instances
	IncludePrevious bool
	// SinceTime only returns logs after this time
	SinceTime *time.Time
	// TailLines limits the number of lines to return (0 = all)
	TailLines int64
}

// ComponentLogs holds logs for a single component
type ComponentLogs struct {
	Component string
	Pod       string
	Container string
	Logs      string
	Error     error
}

// LogCollectionResult holds the result of collecting logs from all components
type LogCollectionResult struct {
	Namespace string
	Timestamp time.Time
	Logs      []ComponentLogs
	OutputDir string
}

// CollectLogs collects logs from all test components (Tempo, MinIO, OTel, k6)
func (f *Framework) CollectLogs(config *LogCollectionConfig) (*LogCollectionResult, error) {
	if config == nil {
		config = &LogCollectionConfig{}
	}
	if config.OutputDir == "" {
		config.OutputDir = "logs"
	}

	result := &LogCollectionResult{
		Namespace: f.namespace,
		Timestamp: time.Now(),
		OutputDir: config.OutputDir,
		Logs:      make([]ComponentLogs, 0),
	}

	// Create output directory
	logDir := filepath.Join(config.OutputDir, f.namespace)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	fmt.Printf("\n📋 Collecting logs from namespace %s...\n", f.namespace)

	// Define component selectors
	components := []struct {
		name     string
		selector string
	}{
		{"tempo", "app.kubernetes.io/name=tempo"},
		{"tempo-monolithic", "app.kubernetes.io/component=tempo"},
		{"tempo-distributor", "app.kubernetes.io/component=distributor"},
		{"tempo-ingester", "app.kubernetes.io/component=ingester"},
		{"tempo-querier", "app.kubernetes.io/component=querier"},
		{"tempo-compactor", "app.kubernetes.io/component=compactor"},
		{"tempo-query-frontend", "app.kubernetes.io/component=query-frontend"},
		{"tempo-gateway", "app.kubernetes.io/component=gateway"},
		{"minio", "app.kubernetes.io/name=minio"},
		{"otel-collector", "app.kubernetes.io/name=opentelemetry-collector"},
		{"k6", "app=k6-perf-test"},
	}

	for _, comp := range components {
		logs := f.collectPodsLogs(comp.name, comp.selector, config)
		result.Logs = append(result.Logs, logs...)
	}

	// Write logs to files
	for _, log := range result.Logs {
		if log.Error != nil {
			continue
		}
		if log.Logs == "" {
			continue
		}

		filename := fmt.Sprintf("%s-%s.log", log.Component, log.Pod)
		if log.Container != "" && log.Container != log.Component {
			filename = fmt.Sprintf("%s-%s-%s.log", log.Component, log.Pod, log.Container)
		}
		// Sanitize filename
		filename = strings.ReplaceAll(filename, "/", "-")
		filepath := filepath.Join(logDir, filename)

		if err := os.WriteFile(filepath, []byte(log.Logs), 0644); err != nil {
			fmt.Printf("   Warning: failed to write %s: %v\n", filename, err)
		} else {
			fmt.Printf("   ✓ %s (%d bytes)\n", filename, len(log.Logs))
		}
	}

	// Count collected logs
	collected := 0
	for _, log := range result.Logs {
		if log.Error == nil && log.Logs != "" {
			collected++
		}
	}

	fmt.Printf("📋 Collected %d log files to %s\n", collected, logDir)
	return result, nil
}

// collectPodsLogs collects logs from pods matching the selector
func (f *Framework) collectPodsLogs(component, selector string, config *LogCollectionConfig) []ComponentLogs {
	var results []ComponentLogs

	pods, err := f.client.CoreV1().Pods(f.namespace).List(f.ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return results
	}

	for _, pod := range pods.Items {
		// Skip pods that aren't running or completed
		if pod.Status.Phase != corev1.PodRunning &&
			pod.Status.Phase != corev1.PodSucceeded &&
			pod.Status.Phase != corev1.PodFailed {
			continue
		}

		// Collect logs from each container
		for _, container := range pod.Spec.Containers {
			logs, err := f.getPodContainerLogs(pod.Name, container.Name, config)
			results = append(results, ComponentLogs{
				Component: component,
				Pod:       pod.Name,
				Container: container.Name,
				Logs:      logs,
				Error:     err,
			})
		}
	}

	return results
}

// getPodContainerLogs retrieves logs from a specific container
func (f *Framework) getPodContainerLogs(podName, containerName string, config *LogCollectionConfig) (string, error) {
	opts := &corev1.PodLogOptions{
		Container: containerName,
		Previous:  config.IncludePrevious,
	}

	if config.SinceTime != nil {
		t := metav1.NewTime(*config.SinceTime)
		opts.SinceTime = &t
	}

	if config.TailLines > 0 {
		opts.TailLines = &config.TailLines
	}

	req := f.client.CoreV1().Pods(f.namespace).GetLogs(podName, opts)

	ctx, cancel := context.WithTimeout(f.ctx, 30*time.Second)
	defer cancel()

	stream, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to stream logs: %w", err)
	}
	defer stream.Close()

	var logs strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := stream.Read(buf)
		if n > 0 {
			logs.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}

	return logs.String(), nil
}

// TempoCRDump holds information about a dumped Tempo CR
type TempoCRDump struct {
	Variant   string // "monolithic" or "stack"
	Name      string
	Namespace string
	FilePath  string
}

// DumpTempoCR fetches the Tempo CR from the cluster and writes it to a YAML file
func (f *Framework) DumpTempoCR(variant, outputDir string) (*TempoCRDump, error) {
	if outputDir == "" {
		outputDir = "."
	}

	// Create output directory if needed
	logDir := filepath.Join(outputDir, f.namespace)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	var crName string
	var gvrToUse = gvr.TempoMonolithic

	switch variant {
	case "monolithic":
		crName = "simplest"
		gvrToUse = gvr.TempoMonolithic
	case "stack":
		crName = "tempostack"
		gvrToUse = gvr.TempoStack
	default:
		return nil, fmt.Errorf("invalid tempo variant: %s (must be 'monolithic' or 'stack')", variant)
	}

	fmt.Printf("\n📄 Dumping Tempo CR (%s/%s)...\n", variant, crName)

	// Fetch the CR from the cluster
	cr, err := f.dynamicClient.Resource(gvrToUse).Namespace(f.namespace).Get(f.ctx, crName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get Tempo CR %s/%s: %w", variant, crName, err)
	}

	// Remove managed fields and other metadata that clutters the output
	cr.SetManagedFields(nil)
	unstructuredContent := cr.UnstructuredContent()

	// Remove status if present (we want the spec, not runtime status)
	// Actually, keep status as it shows the actual state after reconciliation

	// Convert to YAML
	yamlData, err := yaml.Marshal(unstructuredContent)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Tempo CR to YAML: %w", err)
	}

	// Write to file
	filename := fmt.Sprintf("tempo-%s-%s.yaml", variant, crName)
	filePath := filepath.Join(logDir, filename)

	if err := os.WriteFile(filePath, yamlData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write Tempo CR to file: %w", err)
	}

	fmt.Printf("   ✓ %s (%d bytes)\n", filename, len(yamlData))

	return &TempoCRDump{
		Variant:   variant,
		Name:      crName,
		Namespace: f.namespace,
		FilePath:  filePath,
	}, nil
}
