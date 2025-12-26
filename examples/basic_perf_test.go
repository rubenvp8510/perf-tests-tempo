package examples

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/redhat/perf-tests-tempo/test/framework"
	"github.com/redhat/perf-tests-tempo/test/framework/k6"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var _ = Describe("Tempo Performance Tests", func() {
	var (
		fw        *framework.Framework
		testStart time.Time
		ctx       context.Context
	)

	Context("with medium profile", func() {
		BeforeEach(func() {
			ctx = context.Background()
			var err error
			fw, err = framework.New(ctx, "tempo-perf-medium")
			Expect(err).NotTo(HaveOccurred())

			// Check prerequisites
			prereqs, err := fw.CheckPrerequisites()
			Expect(err).NotTo(HaveOccurred())
			Expect(prereqs.AllMet).To(BeTrue(), prereqs.String())

			// Enable user workload monitoring for metrics collection
			Expect(fw.EnableUserWorkloadMonitoring()).To(Succeed())

			Expect(fw.SetupMinIO()).To(Succeed())
			Expect(fw.SetupTempo("monolithic", &framework.ResourceConfig{
				Profile: "medium",
			})).To(Succeed())
			Expect(fw.SetupOTelCollector()).To(Succeed())

			testStart = time.Now()
		})

		AfterEach(func() {
			if fw != nil {
				// Collect metrics before cleanup
				outputFile := fmt.Sprintf("results/%s-metrics.csv", fw.Namespace())
				_ = fw.CollectMetrics(testStart, outputFile)

				Expect(fw.Cleanup()).To(Succeed())
			}
		})

		It("should handle ingestion load", func() {
			result, err := fw.RunK6IngestionTest(k6.SizeMedium)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeTrue())

			GinkgoWriter.Printf("Test completed in %s\n", result.Duration)
		})

		It("should handle query load", func() {
			result, err := fw.RunK6QueryTest(k6.SizeMedium)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeTrue())
		})

		It("should handle combined load", func() {
			result, err := fw.RunK6CombinedTest(k6.SizeMedium)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeTrue())
		})
	})

	Context("with custom resources", func() {
		BeforeEach(func() {
			ctx = context.Background()
			var err error
			fw, err = framework.New(ctx, "tempo-perf-custom")
			Expect(err).NotTo(HaveOccurred())

			// Check prerequisites
			prereqs, err := fw.CheckPrerequisites()
			Expect(err).NotTo(HaveOccurred())
			Expect(prereqs.AllMet).To(BeTrue(), prereqs.String())

			// Enable user workload monitoring for metrics collection
			Expect(fw.EnableUserWorkloadMonitoring()).To(Succeed())

			Expect(fw.SetupMinIO()).To(Succeed())
			Expect(fw.SetupTempo("monolithic", &framework.ResourceConfig{
				Resources: &corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("6Gi"),
						corev1.ResourceCPU:    resource.MustParse("750m"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("6Gi"),
						corev1.ResourceCPU:    resource.MustParse("750m"),
					},
				},
			})).To(Succeed())
			Expect(fw.SetupOTelCollector()).To(Succeed())

			testStart = time.Now()
		})

		AfterEach(func() {
			if fw != nil {
				Expect(fw.Cleanup()).To(Succeed())
			}
		})

		It("should handle load with custom resources", func() {
			result, err := fw.RunK6IngestionTest(k6.SizeSmall)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeTrue())
		})
	})

	Context("with Tempo Stack", func() {
		BeforeEach(func() {
			ctx = context.Background()
			var err error
			fw, err = framework.New(ctx, "tempo-perf-stack")
			Expect(err).NotTo(HaveOccurred())

			// Check prerequisites
			prereqs, err := fw.CheckPrerequisites()
			Expect(err).NotTo(HaveOccurred())
			Expect(prereqs.AllMet).To(BeTrue(), prereqs.String())

			// Enable user workload monitoring for metrics collection
			Expect(fw.EnableUserWorkloadMonitoring()).To(Succeed())

			Expect(fw.SetupMinIO()).To(Succeed())
			Expect(fw.SetupTempo("stack", nil)).To(Succeed())
			Expect(fw.SetupOTelCollector()).To(Succeed())

			testStart = time.Now()
		})

		AfterEach(func() {
			if fw != nil {
				Expect(fw.Cleanup()).To(Succeed())
			}
		})

		It("should handle stack deployment", func() {
			result, err := fw.RunK6IngestionTest(k6.SizeSmall)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeTrue())
		})
	})
})
