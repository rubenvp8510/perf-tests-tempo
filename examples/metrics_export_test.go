package examples

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/redhat/perf-tests-tempo/test/framework"
)

var _ = Describe("Performance Test with Metrics Collection", func() {
	var (
		fw        *framework.Framework
		testStart time.Time
	)

	BeforeEach(func() {
		var err error
		fw, err = framework.New("tempo-perf-test")
		Expect(err).NotTo(HaveOccurred())

		GinkgoWriter.Printf("Test namespace: %s\n", fw.Namespace())

		// Check prerequisites
		prereqs, err := fw.CheckPrerequisites()
		Expect(err).NotTo(HaveOccurred())
		Expect(prereqs.AllMet).To(BeTrue(), prereqs.String())

		// Enable user workload monitoring for metrics collection
		Expect(fw.EnableUserWorkloadMonitoring()).To(Succeed())

		// Deploy MinIO
		err = fw.SetupMinIO()
		Expect(err).NotTo(HaveOccurred())

		// Deploy Tempo with medium resources
		resourceConfig := &framework.ResourceConfig{
			Profile: "medium",
		}
		err = fw.SetupTempo("monolithic", resourceConfig)
		Expect(err).NotTo(HaveOccurred())

		// Deploy OpenTelemetry Collector
		err = fw.SetupOTelCollector()
		Expect(err).NotTo(HaveOccurred())

		// Record when the actual test starts
		testStart = time.Now()
		GinkgoWriter.Printf("Test started at: %s\n", testStart.Format(time.RFC3339))
	})

	It("should collect metrics after performance test", func() {
		// Your performance test here
		// Example:
		// - Deploy trace generators
		// - Run load test
		// - Deploy query generators
		// - Wait for test duration

		// Simulate some test duration
		GinkgoWriter.Printf("Running performance test...\n")
		time.Sleep(5 * time.Second) // Replace with actual test

		// Collect metrics at the end of the test
		// Metrics will be collected from testStart to now for this namespace only
		outputFile := fmt.Sprintf("results/%s-metrics.csv", fw.Namespace())
		err := fw.CollectMetrics(testStart, outputFile)
		Expect(err).NotTo(HaveOccurred())

		GinkgoWriter.Printf("Metrics collected and exported to: %s\n", outputFile)
	})

	AfterEach(func() {
		if fw != nil {
			// Collect metrics before cleanup
			if testStart.IsZero() {
				testStart = time.Now().Add(-5 * time.Minute) // Fallback
			}

			outputFile := fmt.Sprintf("results/%s-metrics.csv", fw.Namespace())
			_ = fw.CollectMetrics(testStart, outputFile)
			// Ignore errors in AfterEach to ensure cleanup runs

			// Cleanup resources
			err := fw.Cleanup()
			Expect(err).NotTo(HaveOccurred())
		}
	})
})

var _ = Describe("Performance Test with Duration-based Collection", func() {
	var fw *framework.Framework

	BeforeEach(func() {
		var err error
		fw, err = framework.New("tempo-perf-test")
		Expect(err).NotTo(HaveOccurred())

		// Check prerequisites
		prereqs, err := fw.CheckPrerequisites()
		Expect(err).NotTo(HaveOccurred())
		Expect(prereqs.AllMet).To(BeTrue(), prereqs.String())

		// Enable user workload monitoring for metrics collection
		Expect(fw.EnableUserWorkloadMonitoring()).To(Succeed())

		// Deploy stack
		err = fw.SetupMinIO()
		Expect(err).NotTo(HaveOccurred())

		err = fw.SetupTempo("monolithic", &framework.ResourceConfig{Profile: "large"})
		Expect(err).NotTo(HaveOccurred())

		err = fw.SetupOTelCollector()
		Expect(err).NotTo(HaveOccurred())
	})

	It("should collect metrics for last 30 minutes", func() {
		// If you don't track the exact start time, you can collect
		// metrics for a specific duration (e.g., last 30 minutes)

		// Run your test...
		time.Sleep(5 * time.Second)

		// Collect metrics for the last 30 minutes
		outputFile := fmt.Sprintf("results/%s-metrics.csv", fw.Namespace())
		err := fw.CollectMetricsWithDuration(30*time.Minute, outputFile)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if fw != nil {
			err := fw.Cleanup()
			Expect(err).NotTo(HaveOccurred())
		}
	})
})
