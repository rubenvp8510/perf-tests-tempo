package k6

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// Clients provides access to Kubernetes clients needed for k6 operations
type Clients interface {
	Client() kubernetes.Interface
	Context() context.Context
	Namespace() string
	Logger() *slog.Logger
}

// scriptsPath returns the path to k6 test scripts
func scriptsPath() string {
	return "tests/k6"
}

// RunTest deploys and runs a k6 test as a Kubernetes Job
func RunTest(c Clients, testType TestType, config *Config) (*Result, error) {
	startTime := time.Now()

	// Set defaults
	if config == nil {
		config = &Config{Size: SizeMedium}
	}
	if config.Size == "" {
		config.Size = SizeMedium
	}
	if config.Image == "" {
		config.Image = DefaultImage
	}

	namespace := c.Namespace()

	// Set default endpoints (internal cluster DNS)
	if config.TempoEndpoint == "" {
		config.TempoEndpoint = fmt.Sprintf("tempo-distributor.%s.svc.cluster.local:4317", namespace)
	}
	if config.TempoQueryEndpoint == "" {
		config.TempoQueryEndpoint = fmt.Sprintf("http://tempo-query-frontend.%s.svc.cluster.local:3200", namespace)
	}

	fmt.Printf("\n🚀 Deploying k6 %s test (size: %s)\n", testType, config.Size)
	fmt.Printf("   Namespace: %s\n", namespace)
	fmt.Printf("   Image: %s\n", config.Image)
	fmt.Printf("   Ingestion Endpoint: %s\n", config.TempoEndpoint)
	fmt.Printf("   Query Endpoint: %s\n\n", config.TempoQueryEndpoint)

	// Create ConfigMap with k6 scripts
	if err := createScriptsConfigMap(c); err != nil {
		return nil, fmt.Errorf("failed to create k6 scripts ConfigMap: %w", err)
	}

	// Create and run k6 Job
	jobName := fmt.Sprintf("k6-%s-%s", testType, config.Size)
	if err := createJob(c, jobName, testType, config); err != nil {
		return nil, fmt.Errorf("failed to create k6 Job: %w", err)
	}

	// Wait for Job to complete
	fmt.Printf("⏳ Waiting for k6 Job to complete (timeout: %s)...\n", JobTimeout)
	success, err := waitForJob(c, jobName)
	if err != nil {
		return nil, fmt.Errorf("error waiting for k6 Job: %w", err)
	}

	// Get logs from Job pod
	logs, err := getJobLogs(c, jobName)
	if err != nil {
		fmt.Printf("Warning: failed to get Job logs: %v\n", err)
		logs = "(logs unavailable)"
	}

	duration := time.Since(startTime)

	result := &Result{
		Success:  success,
		Output:   logs,
		Duration: duration,
	}

	if !success {
		result.Error = fmt.Errorf("k6 test failed")
		return result, result.Error
	}

	fmt.Printf("\n✅ k6 test completed in %s\n", duration.Round(time.Second))
	return result, nil
}

// RunIngestionTest runs the ingestion performance test
func RunIngestionTest(c Clients, size Size) (*Result, error) {
	return RunTest(c, TestIngestion, &Config{Size: size})
}

// RunQueryTest runs the query performance test
func RunQueryTest(c Clients, size Size) (*Result, error) {
	return RunTest(c, TestQuery, &Config{Size: size})
}

// RunCombinedTest runs the combined ingestion+query performance test
func RunCombinedTest(c Clients, size Size) (*Result, error) {
	return RunTest(c, TestCombined, &Config{Size: size})
}

// createScriptsConfigMap creates a ConfigMap with all k6 test scripts
func createScriptsConfigMap(c Clients) error {
	scriptsDir := scriptsPath()
	namespace := c.Namespace()
	client := c.Client()
	ctx := c.Context()

	data := make(map[string]string)

	// Read all JavaScript files from the k6 scripts directory
	files := []string{
		"lib/config.js",
		"lib/trace-profiles.js",
		"ingestion-test.js",
		"query-test.js",
		"combined-test.js",
	}

	for _, file := range files {
		filePath := filepath.Join(scriptsDir, file)
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", filePath, err)
		}
		// Use flat key names for ConfigMap (replace / with -)
		key := strings.ReplaceAll(file, "/", "-")
		data[key] = string(content)
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ScriptsConfigMap,
			Namespace: namespace,
			Labels: map[string]string{
				"app":       "k6-perf-test",
				"component": "scripts",
			},
		},
		Data: data,
	}

	// Delete existing ConfigMap if it exists
	_ = client.CoreV1().ConfigMaps(namespace).Delete(ctx, ScriptsConfigMap, metav1.DeleteOptions{})

	// Create new ConfigMap
	_, err := client.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create ConfigMap: %w", err)
	}

	fmt.Printf("📦 Created ConfigMap %s with k6 scripts\n", ScriptsConfigMap)
	return nil
}

// createJob creates a Kubernetes Job to run the k6 test
func createJob(c Clients, jobName string, testType TestType, config *Config) error {
	namespace := c.Namespace()
	client := c.Client()
	ctx := c.Context()

	// Delete existing job if it exists
	_ = client.BatchV1().Jobs(namespace).Delete(ctx, jobName, metav1.DeleteOptions{
		PropagationPolicy: func() *metav1.DeletionPropagation {
			p := metav1.DeletePropagationBackground
			return &p
		}(),
	})

	// Wait for job to be deleted
	time.Sleep(2 * time.Second)

	// Build environment variables
	env := []corev1.EnvVar{
		{Name: "SIZE", Value: string(config.Size)},
		{Name: "TEMPO_ENDPOINT", Value: config.TempoEndpoint},
		{Name: "TEMPO_QUERY_ENDPOINT", Value: config.TempoQueryEndpoint},
	}

	if config.TempoTenant != "" {
		env = append(env, corev1.EnvVar{Name: "TEMPO_TENANT", Value: config.TempoTenant})
	}
	if config.TempoToken != "" {
		env = append(env, corev1.EnvVar{Name: "TEMPO_TOKEN", Value: config.TempoToken})
	}
	if config.TracesPerSecond > 0 {
		env = append(env, corev1.EnvVar{Name: "TRACES_PER_SECOND", Value: fmt.Sprintf("%d", config.TracesPerSecond)})
	}
	if config.QueriesPerSecond > 0 {
		env = append(env, corev1.EnvVar{Name: "QUERIES_PER_SECOND", Value: fmt.Sprintf("%d", config.QueriesPerSecond)})
	}
	if config.Duration != "" {
		env = append(env, corev1.EnvVar{Name: "DURATION", Value: config.Duration})
	}

	// Build the script path inside the container
	scriptName := fmt.Sprintf("%s-test.js", testType)

	backoffLimit := int32(0)
	ttlSeconds := int32(3600) // Keep job for 1 hour after completion

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":       "k6-perf-test",
				"test-type": string(testType),
				"size":      string(config.Size),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttlSeconds,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":       "k6-perf-test",
						"test-type": string(testType),
						"size":      string(config.Size),
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "k6",
							Image: config.Image,
							Command: []string{
								"/bin/sh",
								"-c",
								fmt.Sprintf(`
									mkdir -p /scripts/lib
									cp /k6-scripts/lib-config.js /scripts/lib/config.js
									cp /k6-scripts/lib-trace-profiles.js /scripts/lib/trace-profiles.js
									cp /k6-scripts/%s /scripts/%s
									cd /scripts
									k6 run %s
								`, scriptName, scriptName, scriptName),
							},
							Env: env,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "k6-scripts",
									MountPath: "/k6-scripts",
									ReadOnly:  true,
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("2"),
									corev1.ResourceMemory: resource.MustParse("2Gi"),
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "k6-scripts",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: ScriptsConfigMap,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := client.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create Job: %w", err)
	}

	fmt.Printf("📋 Created Job %s\n", jobName)
	return nil
}

// waitForJob waits for the k6 Job to complete
func waitForJob(c Clients, jobName string) (bool, error) {
	ctx, cancel := context.WithTimeout(c.Context(), JobTimeout)
	defer cancel()

	namespace := c.Namespace()
	client := c.Client()

	var success bool

	err := wait.PollUntilContextCancel(ctx, 5*time.Second, true, func(ctx context.Context) (bool, error) {
		job, err := client.BatchV1().Jobs(namespace).Get(ctx, jobName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		// Check if job completed
		if job.Status.Succeeded > 0 {
			success = true
			return true, nil
		}

		// Check if job failed
		if job.Status.Failed > 0 {
			success = false
			return true, nil
		}

		// Still running
		fmt.Printf("   Job %s: active=%d, succeeded=%d, failed=%d\n",
			jobName, job.Status.Active, job.Status.Succeeded, job.Status.Failed)
		return false, nil
	})

	return success, err
}

// getJobLogs retrieves logs from the k6 Job pod
func getJobLogs(c Clients, jobName string) (string, error) {
	namespace := c.Namespace()
	client := c.Client()
	ctx := c.Context()

	// Find the pod created by the job
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", jobName),
	})
	if err != nil {
		return "", fmt.Errorf("failed to list pods: %w", err)
	}

	if len(pods.Items) == 0 {
		return "", fmt.Errorf("no pods found for job %s", jobName)
	}

	podName := pods.Items[0].Name

	// Get logs from the pod
	req := client.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{})
	stream, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get pod logs: %w", err)
	}
	defer stream.Close()

	var logs strings.Builder
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		logs.WriteString(scanner.Text())
		logs.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return logs.String(), fmt.Errorf("error reading logs: %w", err)
	}

	return logs.String(), nil
}
