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
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

	// Set default endpoints based on Tempo variant (using gateway for multitenancy)
	if config.TempoEndpoint == "" || config.TempoQueryEndpoint == "" {
		ingestion, query := getDefaultEndpoints(config.TempoVariant, namespace)
		if config.TempoEndpoint == "" {
			config.TempoEndpoint = ingestion
		}
		if config.TempoQueryEndpoint == "" {
			config.TempoQueryEndpoint = query
		}
	}
	// Default tenant for multitenancy mode
	if config.TempoTenant == "" {
		config.TempoTenant = DefaultTenant
	}

	fmt.Printf("\n🚀 Deploying k6 %s test (size: %s)\n", testType, config.Size)
	fmt.Printf("   Namespace: %s\n", namespace)
	fmt.Printf("   Tempo Variant: %s\n", config.TempoVariant)
	fmt.Printf("   Image: %s\n", config.Image)
	fmt.Printf("   Ingestion Endpoint: %s\n", config.TempoEndpoint)
	fmt.Printf("   Query Endpoint: %s\n", config.TempoQueryEndpoint)
	fmt.Printf("   Tenant: %s\n\n", config.TempoTenant)

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

	// Parse k6 metrics from the JSON summary in the logs
	k6Metrics := ParseK6Metrics(logs)

	result := &Result{
		Success:  success,
		Output:   logs,
		Duration: duration,
		Metrics:  k6Metrics,
	}

	if !success {
		result.Error = fmt.Errorf("k6 test failed")
		return result, result.Error
	}

	// Print k6 metrics summary if available
	if k6Metrics != nil {
		fmt.Println("\n📊 k6 Metrics Summary:")
		if k6Metrics.QueryRequestsTotal > 0 {
			fmt.Printf("   Query Requests: %.0f (failures: %.0f)\n", k6Metrics.QueryRequestsTotal, k6Metrics.QueryFailuresTotal)
			fmt.Printf("   Query Latency P99: %.3fs\n", k6Metrics.QueryDurationSeconds.P99)
		}
		if k6Metrics.IngestionTracesTotal > 0 {
			fmt.Printf("   Traces Ingested: %.0f\n", k6Metrics.IngestionTracesTotal)
			fmt.Printf("   Ingestion Rate: %.2f MB/s\n", k6Metrics.IngestionRateBPS/1024/1024)
		}
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

// ParallelResult holds results from parallel ingestion and query tests
type ParallelResult struct {
	Ingestion *Result
	Query     *Result
	Duration  time.Duration
}

// Success returns true if both tests succeeded
func (p *ParallelResult) Success() bool {
	return p.Ingestion != nil && p.Query != nil &&
		p.Ingestion.Success && p.Query.Success
}

// ServiceCAConfigMap is the name of the ConfigMap for OpenShift service CA
const ServiceCAConfigMap = "k6-service-ca"

// K6ServiceAccount is the name of the ServiceAccount for k6 pods
const K6ServiceAccount = "k6-query-sa"

// setupK6RBAC creates ServiceAccount and RBAC for k6 query pods to access Tempo
func setupK6RBAC(c Clients) error {
	namespace := c.Namespace()
	client := c.Client()
	ctx := c.Context()

	// Create ServiceAccount
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      K6ServiceAccount,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "k6-perf-test",
			},
		},
	}
	_, err := client.CoreV1().ServiceAccounts(namespace).Create(ctx, sa, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create ServiceAccount: %w", err)
	}

	// Create ClusterRole for reading traces from tenant-1
	clusterRoleName := fmt.Sprintf("allow-read-traces-%s", namespace)
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleName,
			Labels: map[string]string{
				"app": "k6-perf-test",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{"tempo.grafana.com"},
				Resources:     []string{DefaultTenant}, // tenant-1
				ResourceNames: []string{"traces"},
				Verbs:         []string{"get"},
			},
		},
	}
	_, err = client.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create ClusterRole: %w", err)
	}

	// Create ClusterRoleBinding
	clusterRoleBindingName := fmt.Sprintf("allow-read-traces-%s", namespace)
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleBindingName,
			Labels: map[string]string{
				"app": "k6-perf-test",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      K6ServiceAccount,
				Namespace: namespace,
			},
		},
	}
	_, err = client.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create ClusterRoleBinding: %w", err)
	}

	fmt.Printf("🔐 Created RBAC for k6 query (ServiceAccount: %s)\n", K6ServiceAccount)
	return nil
}

// RunParallelTests runs ingestion and query tests as separate parallel Kubernetes Jobs
func RunParallelTests(c Clients, config *Config) (*ParallelResult, error) {
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

	// Set default endpoints based on Tempo variant (using gateway for multitenancy)
	if config.TempoEndpoint == "" || config.TempoQueryEndpoint == "" {
		ingestion, query := getDefaultEndpoints(config.TempoVariant, namespace)
		if config.TempoEndpoint == "" {
			config.TempoEndpoint = ingestion
		}
		if config.TempoQueryEndpoint == "" {
			config.TempoQueryEndpoint = query
		}
	}
	// Default tenant for multitenancy mode
	if config.TempoTenant == "" {
		config.TempoTenant = DefaultTenant
	}

	fmt.Printf("\n🚀 Deploying parallel k6 tests (ingestion + query)\n")
	fmt.Printf("   Namespace: %s\n", namespace)
	fmt.Printf("   Tempo Variant: %s\n", config.TempoVariant)
	fmt.Printf("   Image: %s\n", config.Image)
	fmt.Printf("   Ingestion Endpoint: %s\n", config.TempoEndpoint)
	fmt.Printf("   Query Endpoint: %s\n", config.TempoQueryEndpoint)
	fmt.Printf("   Tenant: %s\n\n", config.TempoTenant)

	// Create ConfigMap with k6 scripts
	if err := createScriptsConfigMap(c); err != nil {
		return nil, fmt.Errorf("failed to create k6 scripts ConfigMap: %w", err)
	}

	// Create ConfigMap for OpenShift service CA (for TLS)
	if err := createServiceCAConfigMap(c); err != nil {
		return nil, fmt.Errorf("failed to create service CA ConfigMap: %w", err)
	}

	// Setup RBAC for k6 query pods
	if err := setupK6RBAC(c); err != nil {
		return nil, fmt.Errorf("failed to setup k6 RBAC: %w", err)
	}

	// Create both jobs
	ingestionJobName := fmt.Sprintf("k6-ingestion-%s", config.Size)
	queryJobName := fmt.Sprintf("k6-query-%s", config.Size)

	if err := createJob(c, ingestionJobName, TestIngestion, config); err != nil {
		return nil, fmt.Errorf("failed to create ingestion Job: %w", err)
	}

	if err := createJob(c, queryJobName, TestQuery, config); err != nil {
		return nil, fmt.Errorf("failed to create query Job: %w", err)
	}

	// Wait for both jobs to complete in parallel
	fmt.Printf("⏳ Waiting for both k6 Jobs to complete (timeout: %s)...\n", JobTimeout)

	type jobResult struct {
		name    string
		success bool
		logs    string
		err     error
	}

	results := make(chan jobResult, 2)

	// Wait for ingestion job
	go func() {
		success, err := waitForJob(c, ingestionJobName)
		logs, _ := getJobLogs(c, ingestionJobName)
		results <- jobResult{name: "ingestion", success: success, logs: logs, err: err}
	}()

	// Wait for query job
	go func() {
		success, err := waitForJob(c, queryJobName)
		logs, _ := getJobLogs(c, queryJobName)
		results <- jobResult{name: "query", success: success, logs: logs, err: err}
	}()

	// Collect results
	parallelResult := &ParallelResult{}
	for i := 0; i < 2; i++ {
		r := <-results
		result := &Result{
			Success: r.success,
			Output:  r.logs,
		}
		if r.err != nil {
			result.Error = r.err
		} else if !r.success {
			result.Error = fmt.Errorf("k6 %s test failed", r.name)
		}

		if r.name == "ingestion" {
			parallelResult.Ingestion = result
			if r.success {
				fmt.Printf("✅ Ingestion test completed\n")
			} else {
				fmt.Printf("❌ Ingestion test failed\n")
			}
		} else {
			parallelResult.Query = result
			if r.success {
				fmt.Printf("✅ Query test completed\n")
			} else {
				fmt.Printf("❌ Query test failed\n")
			}
		}
	}

	parallelResult.Duration = time.Since(startTime)

	if parallelResult.Success() {
		fmt.Printf("\n✅ Both tests completed successfully in %s\n", parallelResult.Duration.Round(time.Second))
	} else {
		fmt.Printf("\n❌ One or more tests failed (duration: %s)\n", parallelResult.Duration.Round(time.Second))
	}

	return parallelResult, nil
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

// createServiceCAConfigMap creates a ConfigMap that OpenShift will inject with the service CA
func createServiceCAConfigMap(c Clients) error {
	namespace := c.Namespace()
	client := c.Client()
	ctx := c.Context()

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceCAConfigMap,
			Namespace: namespace,
			Labels: map[string]string{
				"app":       "k6-perf-test",
				"component": "service-ca",
			},
			Annotations: map[string]string{
				// This annotation tells OpenShift to inject the service-serving CA bundle
				"service.beta.openshift.io/inject-cabundle": "true",
			},
		},
		// Data will be populated by OpenShift service-ca operator
		Data: map[string]string{},
	}

	// Delete existing ConfigMap if it exists
	_ = client.CoreV1().ConfigMaps(namespace).Delete(ctx, ServiceCAConfigMap, metav1.DeleteOptions{})
	time.Sleep(1 * time.Second)

	// Create new ConfigMap
	_, err := client.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create service CA ConfigMap: %w", err)
	}

	// Wait a bit for the CA bundle to be injected
	time.Sleep(2 * time.Second)

	fmt.Printf("📦 Created ConfigMap %s for service CA\n", ServiceCAConfigMap)
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
	// The service CA is mounted from the ConfigMap at /etc/ssl/certs/service-ca.crt
	serviceCAMountPath := "/etc/ssl/certs/service-ca.crt"
	env := []corev1.EnvVar{
		{Name: "SIZE", Value: string(config.Size)},
		{Name: "TEMPO_ENDPOINT", Value: config.TempoEndpoint},
		{Name: "TEMPO_QUERY_ENDPOINT", Value: config.TempoQueryEndpoint},
		// TLS configuration for query (gateway) - ingestion goes through OTel Collector (no TLS)
		{Name: "TEMPO_QUERY_TLS_ENABLED", Value: "true"},
		{Name: "TEMPO_TLS_CA_FILE", Value: serviceCAMountPath},
		{Name: "TEMPO_TOKEN_FILE", Value: ServiceAccountTokenPath},
	}

	if config.TempoTenant != "" {
		env = append(env, corev1.EnvVar{Name: "TEMPO_TENANT", Value: config.TempoTenant})
	}
	if config.TempoToken != "" {
		env = append(env, corev1.EnvVar{Name: "TEMPO_TOKEN", Value: config.TempoToken})
	}
	if config.MBPerSecond > 0 {
		env = append(env, corev1.EnvVar{Name: "MB_PER_SECOND", Value: fmt.Sprintf("%f", config.MBPerSecond)})
	}
	if config.QueriesPerSecond > 0 {
		env = append(env, corev1.EnvVar{Name: "QUERIES_PER_SECOND", Value: fmt.Sprintf("%d", config.QueriesPerSecond)})
	}
	if config.Duration != "" {
		env = append(env, corev1.EnvVar{Name: "DURATION", Value: config.Duration})
	}
	if config.VUsMin > 0 {
		env = append(env, corev1.EnvVar{Name: "VUS_MIN", Value: fmt.Sprintf("%d", config.VUsMin)})
	}
	if config.VUsMax > 0 {
		env = append(env, corev1.EnvVar{Name: "VUS_MAX", Value: fmt.Sprintf("%d", config.VUsMax)})
	}
	if config.TraceProfile != "" {
		env = append(env, corev1.EnvVar{Name: "TRACE_PROFILE", Value: config.TraceProfile})
	}

	// Prometheus remote write configuration for exporting k6 metrics
	if config.PrometheusRWURL != "" {
		env = append(env,
			corev1.EnvVar{Name: "K6_PROMETHEUS_RW_SERVER_URL", Value: config.PrometheusRWURL},
			corev1.EnvVar{Name: "K6_PROMETHEUS_RW_TREND_AS_NATIVE_HISTOGRAM", Value: "true"},
			corev1.EnvVar{Name: "K6_PROMETHEUS_RW_STALE_MARKERS", Value: "true"},
		)
	}

	// Build the script path inside the container
	scriptName := fmt.Sprintf("%s-test.js", testType)

	// Build k6 run command with JSON summary export
	// Always export summary to JSON for metrics parsing
	k6RunCmd := fmt.Sprintf("k6 run --summary-export=/tmp/summary.json %s", scriptName)
	if config.PrometheusRWURL != "" {
		k6RunCmd = fmt.Sprintf("k6 run -o experimental-prometheus-rw --summary-export=/tmp/summary.json %s", scriptName)
	}

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
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: K6ServiceAccount,
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
									%s
									exit_code=$?
									echo "===K6_SUMMARY_JSON_START==="
									cat /tmp/summary.json 2>/dev/null || echo "{}"
									echo "===K6_SUMMARY_JSON_END==="
									exit $exit_code
								`, scriptName, scriptName, k6RunCmd),
							},
							Env: env,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "k6-scripts",
									MountPath: "/k6-scripts",
									ReadOnly:  true,
								},
								{
									Name:      "scripts",
									MountPath: "/scripts",
								},
								{
									Name:      "service-ca",
									MountPath: "/etc/ssl/certs",
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
						{
							Name: "scripts",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "service-ca",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: ServiceCAConfigMap,
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

// getDefaultEndpoints returns the default ingestion and query endpoints
// based on the Tempo deployment variant.
//
// Ingestion goes through the OpenTelemetry Collector (no TLS needed in-cluster)
// Queries go directly to the Tempo gateway (with TLS/auth and multitenancy path)
func getDefaultEndpoints(variant TempoVariant, namespace string) (ingestion, query string) {
	var crName string
	switch variant {
	case TempoStack:
		crName = StackCRName
	case TempoMonolithic:
		crName = MonolithicCRName
	default:
		crName = MonolithicCRName
	}

	// Ingestion through OpenTelemetry Collector (handles auth to Tempo)
	otelCollectorHost := fmt.Sprintf("otel-collector-collector.%s.svc.cluster.local", namespace)
	ingestion = fmt.Sprintf("%s:4317", otelCollectorHost)

	// Query through Tempo gateway (with TLS/auth)
	// For multitenancy, the Observatorium API routes are:
	// /api/traces/v1/{tenant}/tempo/api/... for Tempo native API
	gatewayHost := fmt.Sprintf("tempo-%s-gateway.%s.svc.cluster.local", crName, namespace)
	query = fmt.Sprintf("https://%s:8080/api/traces/v1/%s/tempo", gatewayHost, DefaultTenant)

	return ingestion, query
}
