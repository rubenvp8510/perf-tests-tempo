package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/redhat/perf-tests-tempo/test/framework"
	"github.com/redhat/perf-tests-tempo/test/framework/k6"
	"github.com/redhat/perf-tests-tempo/test/framework/profile"
)

func main() {
	var (
		profilesFlag      = flag.String("profiles", "", "Comma-separated list of profiles to run (e.g., small,medium)")
		profilesDir       = flag.String("profiles-dir", "profiles", "Directory containing profile YAML files")
		outputDir         = flag.String("output", "results", "Output directory for metrics")
		testType          = flag.String("test-type", "combined", "Test type: ingestion, query, combined")
		dryRun            = flag.Bool("dry-run", false, "Print what would be executed without running")
		skipCleanup       = flag.Bool("skip-cleanup", false, "Skip cleanup after tests (useful for debugging)")
		checkMetrics      = flag.Bool("check-metrics", false, "Check and report metric availability after collection")
		generateDashboard = flag.Bool("generate-dashboard", true, "Generate HTML dashboard after metrics collection")
	)
	flag.Parse()

	// Validate test type
	tt := k6.TestType(*testType)
	switch tt {
	case k6.TestIngestion, k6.TestQuery, k6.TestCombined:
		// Valid
	default:
		fmt.Fprintf(os.Stderr, "Error: invalid test type %q. Must be ingestion, query, or combined\n", *testType)
		os.Exit(1)
	}

	// Load profiles
	var profiles []*profile.Profile
	var err error

	if *profilesFlag != "" {
		names := strings.Split(*profilesFlag, ",")
		profiles, err = profile.LoadByNames(*profilesDir, names)
	} else {
		profiles, err = profile.LoadAll(*profilesDir)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading profiles: %v\n", err)
		os.Exit(1)
	}

	if len(profiles) == 0 {
		fmt.Fprintf(os.Stderr, "Error: no profiles found in %s\n", *profilesDir)
		os.Exit(1)
	}

	// Print summary
	fmt.Printf("Loaded %d profile(s):\n", len(profiles))
	for _, p := range profiles {
		fmt.Printf("  - %s: %s\n", p.Name, p.Description)
	}
	fmt.Println()

	if *dryRun {
		fmt.Println("Dry run mode - would execute the following:")
		for _, p := range profiles {
			printProfileSummary(p, tt)
		}
		return
	}

	// Setup context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived interrupt signal, cleaning up...")
		cancel()
		// Second interrupt force-exits
		<-sigCh
		fmt.Println("\nForce exit requested, terminating immediately...")
		os.Exit(130) // 128 + SIGINT(2)
	}()

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Run profiles sequentially
	results := make(map[string]*RunResult)
	for _, p := range profiles {
		select {
		case <-ctx.Done():
			fmt.Println("Aborted by user")
			printSummary(results)
			os.Exit(1)
		default:
		}

		result := runProfile(ctx, p, tt, *outputDir, *skipCleanup, *checkMetrics, *generateDashboard)
		results[p.Name] = result

		if result.Error != nil {
			fmt.Printf("Profile %s failed: %v\n", p.Name, result.Error)
		}
	}

	// Print summary
	printSummary(results)

	// Exit with error if any profile failed
	for _, r := range results {
		if r.Error != nil {
			os.Exit(1)
		}
	}
}

// RunResult holds the result of running a profile
type RunResult struct {
	Profile  string
	Success  bool
	Duration time.Duration
	Error    error
}

func runProfile(ctx context.Context, p *profile.Profile, testType k6.TestType, outputDir string, skipCleanup, checkMetrics, generateDashboard bool) *RunResult {
	startTime := time.Now()
	result := &RunResult{Profile: p.Name}

	namespace := fmt.Sprintf("tempo-perf-%s", p.Name)
	fmt.Printf("\n========================================\n")
	fmt.Printf("Running profile: %s\n", p.Name)
	fmt.Printf("Namespace: %s\n", namespace)
	fmt.Printf("========================================\n\n")

	// Create framework
	fw, err := framework.New(ctx, namespace)
	if err != nil {
		result.Error = fmt.Errorf("failed to create framework: %w", err)
		result.Duration = time.Since(startTime)
		return result
	}

	// Clean up any leftover resources from previous runs
	fmt.Println("Cleaning up previous resources...")
	if cleanupErr := fw.Cleanup(); cleanupErr != nil {
		fmt.Printf("Warning: pre-cleanup failed (may be expected if namespace doesn't exist): %v\n", cleanupErr)
	}

	// Re-create framework after cleanup (namespace was deleted)
	fw, err = framework.New(ctx, namespace)
	if err != nil {
		result.Error = fmt.Errorf("failed to re-create framework after cleanup: %w", err)
		result.Duration = time.Since(startTime)
		return result
	}

	// Cleanup after test unless skipped
	if !skipCleanup {
		defer func() {
			fmt.Printf("\nCleaning up namespace %s...\n", namespace)
			if cleanupErr := fw.Cleanup(); cleanupErr != nil {
				fmt.Printf("Warning: cleanup failed: %v\n", cleanupErr)
			}
		}()
	}

	// Check prerequisites
	fmt.Println("Checking prerequisites...")
	prereqs, err := fw.CheckPrerequisites()
	if err != nil {
		result.Error = fmt.Errorf("failed to check prerequisites: %w", err)
		result.Duration = time.Since(startTime)
		return result
	}
	if !prereqs.AllMet {
		result.Error = fmt.Errorf("prerequisites not met: Tempo=%v, OTel=%v",
			prereqs.TempoOperator.Installed, prereqs.OpenTelemetryOperator.Installed)
		result.Duration = time.Since(startTime)
		return result
	}

	// Enable user workload monitoring for Tempo metrics collection
	fmt.Println("Enabling user workload monitoring...")
	if err := fw.EnableUserWorkloadMonitoring(); err != nil {
		fmt.Printf("Warning: failed to enable user workload monitoring: %v\n", err)
		fmt.Println("Tempo metrics may not be available. Continuing anyway...")
	}

	// Setup MinIO
	fmt.Println("Setting up MinIO...")
	if err := fw.SetupMinIO(); err != nil {
		result.Error = fmt.Errorf("failed to setup MinIO: %w", err)
		result.Duration = time.Since(startTime)
		return result
	}

	// Setup Tempo with profile resources
	fmt.Printf("Setting up Tempo (%s)...\n", p.Tempo.Variant)
	resourceConfig := profileToResourceConfig(p)
	if err := fw.SetupTempo(p.Tempo.Variant, resourceConfig); err != nil {
		result.Error = fmt.Errorf("failed to setup Tempo: %w", err)
		result.Duration = time.Since(startTime)
		return result
	}

	// Setup OTel Collector (pass Tempo variant for correct gateway endpoint)
	fmt.Println("Setting up OTel Collector...")
	if err := fw.SetupOTelCollector(p.Tempo.Variant); err != nil {
		result.Error = fmt.Errorf("failed to setup OTel Collector: %w", err)
		result.Duration = time.Since(startTime)
		return result
	}

	// Setup Tempo monitoring (ServiceMonitor verification and PodMonitor fallback)
	fmt.Println("Setting up Tempo monitoring...")
	if err := fw.SetupTempoMonitoring(p.Tempo.Variant); err != nil {
		fmt.Printf("Warning: failed to setup Tempo monitoring: %v\n", err)
		// Continue anyway - metrics may still work
	}

	// Setup k6 Prometheus metrics export
	fmt.Println("Setting up k6 Prometheus metrics...")
	prometheusRWURL, err := fw.SetupK6PrometheusMetrics()
	if err != nil {
		fmt.Printf("Warning: failed to setup k6 Prometheus metrics: %v\n", err)
		// Continue anyway - k6 will just not export to Prometheus
	}

	// Run k6 test(s)
	testStartTime := time.Now()
	k6Config := profileToK6Config(p)
	k6Config.PrometheusRWURL = prometheusRWURL

	var testSuccess bool
	var k6Metrics *k6.K6Metrics
	if testType == k6.TestCombined {
		// Run ingestion and query as separate parallel jobs
		fmt.Println("Running parallel k6 tests (ingestion + query as separate jobs)...")
		parallelResult, err := fw.RunK6ParallelTests(k6Config)
		if err != nil {
			result.Error = fmt.Errorf("parallel k6 tests failed: %w", err)
			result.Duration = time.Since(startTime)
			return result
		}
		testSuccess = parallelResult.Success()

		// Save k6 logs to files and collect metrics
		if parallelResult.Ingestion != nil && parallelResult.Ingestion.Output != "" {
			logFile := fmt.Sprintf("%s/%s-k6-ingestion.log", outputDir, p.Name)
			if err := os.WriteFile(logFile, []byte(parallelResult.Ingestion.Output), 0644); err != nil {
				fmt.Printf("Warning: failed to save ingestion logs: %v\n", err)
			} else {
				fmt.Printf("Saved ingestion logs to %s\n", logFile)
			}
			// Export ingestion k6 metrics
			if parallelResult.Ingestion.Metrics != nil {
				metricsFile := fmt.Sprintf("%s/%s-k6-ingestion-metrics.json", outputDir, p.Name)
				if err := fw.ExportK6Metrics(parallelResult.Ingestion.Metrics, metricsFile, "ingestion"); err != nil {
					fmt.Printf("Warning: failed to export ingestion k6 metrics: %v\n", err)
				}
			}
		}
		if parallelResult.Query != nil && parallelResult.Query.Output != "" {
			logFile := fmt.Sprintf("%s/%s-k6-query.log", outputDir, p.Name)
			if err := os.WriteFile(logFile, []byte(parallelResult.Query.Output), 0644); err != nil {
				fmt.Printf("Warning: failed to save query logs: %v\n", err)
			} else {
				fmt.Printf("Saved query logs to %s\n", logFile)
			}
			// Export query k6 metrics
			if parallelResult.Query.Metrics != nil {
				k6Metrics = parallelResult.Query.Metrics // Keep for dashboard
				metricsFile := fmt.Sprintf("%s/%s-k6-query-metrics.json", outputDir, p.Name)
				if err := fw.ExportK6Metrics(parallelResult.Query.Metrics, metricsFile, "query"); err != nil {
					fmt.Printf("Warning: failed to export query k6 metrics: %v\n", err)
				}
			}
		}
	} else {
		// Run single test type
		fmt.Printf("Running k6 %s test...\n", testType)
		k6Result, err := fw.RunK6Test(testType, k6Config)
		if err != nil {
			result.Error = fmt.Errorf("k6 test failed: %w", err)
			result.Duration = time.Since(startTime)
			return result
		}
		testSuccess = k6Result.Success
		k6Metrics = k6Result.Metrics

		// Save k6 logs to file
		if k6Result.Output != "" {
			logFile := fmt.Sprintf("%s/%s-k6-%s.log", outputDir, p.Name, testType)
			if err := os.WriteFile(logFile, []byte(k6Result.Output), 0644); err != nil {
				fmt.Printf("Warning: failed to save k6 logs: %v\n", err)
			} else {
				fmt.Printf("Saved k6 logs to %s\n", logFile)
			}
		}

		// Export k6 metrics to JSON
		if k6Metrics != nil {
			metricsFile := fmt.Sprintf("%s/%s-k6-%s-metrics.json", outputDir, p.Name, testType)
			if err := fw.ExportK6Metrics(k6Metrics, metricsFile, string(testType)); err != nil {
				fmt.Printf("Warning: failed to export k6 metrics: %v\n", err)
			}
		}
	}

	// Log k6 metrics availability
	if k6Metrics != nil {
		fmt.Println("✅ k6 metrics parsed from JSON summary")
	}

	if !testSuccess {
		result.Error = fmt.Errorf("k6 test did not succeed")
		result.Duration = time.Since(startTime)
		return result
	}

	// Collect metrics
	metricsFile := fmt.Sprintf("%s/%s-metrics.csv", outputDir, p.Name)
	fmt.Printf("Collecting metrics to %s...\n", metricsFile)
	if err := fw.CollectMetrics(testStartTime, metricsFile); err != nil {
		fmt.Printf("Warning: failed to collect metrics: %v\n", err)
	}

	// Check metric availability if requested
	if checkMetrics {
		fmt.Println("\nChecking metric availability...")
		testDuration := time.Since(testStartTime)
		report, err := fw.CheckMetricAvailability(testDuration)
		if err != nil {
			fmt.Printf("Warning: failed to check metric availability: %v\n", err)
		} else {
			fw.PrintMetricAvailabilityReport(report)

			// Print diagnostic hints if there are missing metrics
			if report.MissingMetrics > 0 {
				issues := fw.DiagnoseMetricIssues(report)
				if len(issues) > 0 {
					fmt.Println("\nDiagnostic hints:")
					for _, issue := range issues {
						fmt.Printf("  ⚠️  %s\n", issue)
					}
				}
			}
		}
	}

	// Generate dashboard if requested
	if generateDashboard {
		dashboardFile := fmt.Sprintf("%s/%s-dashboard.html", outputDir, p.Name)
		fmt.Printf("Generating dashboard to %s...\n", dashboardFile)
		if err := fw.GenerateDashboard(metricsFile, dashboardFile, p.Name); err != nil {
			fmt.Printf("Warning: failed to generate dashboard: %v\n", err)
		} else {
			fmt.Printf("Dashboard generated: %s\n", dashboardFile)
		}
	}

	result.Success = true
	result.Duration = time.Since(startTime)
	fmt.Printf("\nProfile %s completed successfully in %s\n", p.Name, result.Duration.Round(time.Second))

	return result
}

func profileToResourceConfig(p *profile.Profile) *framework.ResourceConfig {
	config := &framework.ResourceConfig{}
	hasConfig := false

	// Add resources if specified
	if p.Tempo.HasResources() {
		config.Resources = &corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse(p.Tempo.Resources.Memory),
				corev1.ResourceCPU:    resource.MustParse(p.Tempo.Resources.CPU),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse(p.Tempo.Resources.Memory),
				corev1.ResourceCPU:    resource.MustParse(p.Tempo.Resources.CPU),
			},
		}
		hasConfig = true
	}

	// Add replication factor if specified (only applies to TempoStack)
	if p.Tempo.ReplicationFactor != nil {
		config.ReplicationFactor = p.Tempo.ReplicationFactor
		hasConfig = true
	}

	// Get max traces per user from env var (takes precedence) or profile
	maxTracesPerUser := getMaxTracesPerUser(p)
	if maxTracesPerUser != nil {
		config.Overrides = &framework.TempoOverrides{
			MaxTracesPerUser: maxTracesPerUser,
		}
		hasConfig = true
	}

	if !hasConfig {
		return nil // Use operator defaults
	}
	return config
}

// getMaxTracesPerUser returns the max traces per user setting from env var or profile
func getMaxTracesPerUser(p *profile.Profile) *int {
	// Environment variable takes precedence
	if envVal := os.Getenv("MAX_TRACES_PER_USER"); envVal != "" {
		var val int
		if _, err := fmt.Sscanf(envVal, "%d", &val); err == nil {
			return &val
		}
	}

	// Fall back to profile setting
	if p.Tempo.Overrides != nil && p.Tempo.Overrides.MaxTracesPerUser != nil {
		return p.Tempo.Overrides.MaxTracesPerUser
	}

	return nil
}

func profileToK6Config(p *profile.Profile) *k6.Config {
	// Get duration from DURATION env var, default to 5m
	duration := os.Getenv("DURATION")
	if duration == "" {
		duration = "5m"
	}

	return &k6.Config{
		TempoVariant:     k6.TempoVariant(p.Tempo.Variant),
		MBPerSecond:      p.K6.Ingestion.MBPerSecond,
		QueriesPerSecond: p.K6.Query.QueriesPerSecond,
		Duration:         duration,
		VUsMin:           p.K6.VUs.Min,
		VUsMax:           p.K6.VUs.Max,
		TraceProfile:     p.K6.Ingestion.TraceProfile,
	}
}

func printProfileSummary(p *profile.Profile, testType k6.TestType) {
	// Get effective duration
	duration := os.Getenv("DURATION")
	if duration == "" {
		duration = "5m"
	}

	fmt.Printf("\nProfile: %s\n", p.Name)
	fmt.Printf("  Description: %s\n", p.Description)
	fmt.Printf("  Tempo:\n")
	fmt.Printf("    Variant: %s\n", p.Tempo.Variant)
	if p.Tempo.ReplicationFactor != nil {
		fmt.Printf("    ReplicationFactor: %d\n", *p.Tempo.ReplicationFactor)
	}
	if p.Tempo.HasResources() {
		fmt.Printf("    Resources: %s memory, %s CPU\n", p.Tempo.Resources.Memory, p.Tempo.Resources.CPU)
	} else {
		fmt.Printf("    Resources: (operator defaults)\n")
	}

	// Show max traces per user setting
	maxTraces := getMaxTracesPerUser(p)
	if maxTraces != nil {
		if *maxTraces == 0 {
			fmt.Printf("    MaxTracesPerUser: 0 (unlimited)\n")
		} else {
			fmt.Printf("    MaxTracesPerUser: %d\n", *maxTraces)
		}
	} else {
		fmt.Printf("    MaxTracesPerUser: (Tempo default)\n")
	}

	fmt.Printf("  K6 (%s test):\n", testType)
	fmt.Printf("    Duration: %s\n", duration)
	fmt.Printf("    VUs: %d-%d\n", p.K6.VUs.Min, p.K6.VUs.Max)
	fmt.Printf("    Ingestion: %.1f MB/s\n", p.K6.Ingestion.MBPerSecond)
	fmt.Printf("    Queries/sec: %d\n", p.K6.Query.QueriesPerSecond)
	fmt.Printf("    Trace profile: %s\n", p.K6.Ingestion.TraceProfile)
}

func printSummary(results map[string]*RunResult) {
	fmt.Printf("\n========================================\n")
	fmt.Printf("SUMMARY\n")
	fmt.Printf("========================================\n")

	var passed, failed int
	for name, r := range results {
		status := "PASS"
		if r.Error != nil {
			status = "FAIL"
			failed++
		} else {
			passed++
		}
		fmt.Printf("  %s: %s (%s)\n", name, status, r.Duration.Round(time.Second))
	}

	fmt.Printf("\nTotal: %d passed, %d failed\n", passed, failed)
}
